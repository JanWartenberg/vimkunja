# Vikunja TUI (MVP) — Vim-Motions & Pane Layout

## Ziel
Ein Terminal-Interface für Vikunja, das sich **wie vi/vim** bedienen lässt:
- Navigation primär via **Vim-Motions** (`j/k`, `gg/G`, `enter`, …)
- **Multi-Pane Handling** wie Vim-Splits
- MVP bildet erstmal **nur eine Task-Ansicht** ab: `ProjectViewKind = list`

## Daten-/Feature-Scope (MVP)
- **Links:** Projekte auswählen
- **Rechts:** Tasks für die ausgewählte Projekt-View anzeigen
- View-Auswahl: Für ein Projekt wird automatisch die **erste** View mit `view_kind == "list"` genommen.
- Task-CRUD/Comments/Edit sind später — MVP ist “read + navigation”, plus optional “Refresh”.

## Pane-Konzept (Vim-Splits)
Es gibt 3 Panes:
- `projects` (links oben)
- `tasks` (rechts oben)
- `detail` (unten, schließbar)

**Pane-Lifecycle (Decision #4):**
- Detail ist ein vollwertiger Pane im Fokus-Graph, kein Modal.
- `Enter` in Tasks öffnet Detail; `Ctrl+W` navigiert zwischen allen 3 Panes.
- Detail kann geschlossen werden (Keybinding siehe unten).
- Wenn Detail geschlossen wird, Fokus zurück auf Tasks.

### Fokus-Commands
Wie in Vim:
- `Ctrl+W` (als Leader) + `Ctrl+W`  → Fokus togglen/cycle
- `Ctrl+W` + `h/j/k/l`              → Fokus auf das Nachbar-Pane in diese Richtung
- `Ctrl+W k` von Detail → zurück zu Tasks
- `Ctrl+W j` von Tasks/Projects → zu Detail (falls offen)

### Beispiel-Flow
- `Ctrl+W h` → Fokus auf Projekte
- `j/k` → Projekt wechseln
- `Ctrl+W l` → Fokus auf Tasks
- `j/k` → Tasks scrollen
- `r` → Refresh (projekte+tasks reload)

## Erweiterbarkeit (Design)
Obwohl MVP nur `list`-View rendert, wird die Struktur so gebaut, dass später leicht hinzugefügt werden kann:
- Weitere `ProjectViewKind`-Handler (`kanban`, `table`, …)
- Task-Details-Pane (z.B. “Enter öffnet Detail unter dem aktuellen Pane Layout”)
- Compose von mehreren Panes in einer Art “Pane-Grid” (Nachbarn via Koordinaten oder Graph)

## Technische Entscheidung
- **Sprache:** Go
- **UI Framework:** `bubbletea` + `lipgloss`
- **Backend:** Direkter SQLite-Zugriff (keine REST API)

## Architektur-Decisions (Grilling Session 2026-07-09)

### 1. Store-Abstraktion: SQLite-only, kein REST
**Decision:** REST-Client (`client.go`) wird entfernt. Die App greift ausschließlich direkt auf die `vikunja.db`-Datei zu. Begründung: 1-User-Setup, immer Dateizugriff vorhanden, keine Asymmetrie ("not implemented" im REST-Mode), weniger Komplexität.

### 2. Write-Semantik: Treue zu Server-Logik (B)
**Decision:** Beim Schreiben (z.B. `ToggleTaskDone`) replizieren wir die wichtigsten Side-Effects des Vikunja-Servers:
- `updated`-Timestamp bei jedem Write bumpen
- Done-Tasks in den `done_bucket_id` verschieben (falls vorhanden in der View)
- Später bei Edit: entsprechend alle relevanten Felder + `updated` pflegen

**Begründung:** Core Law #1 (Schema-Kompatibilität) — die DB soll in einem Zustand bleiben, den der echte Server auch produzieren würde. `updated` ist kritisch für Server-Features (Webhooks, Sync, "zuletzt geändert"). Bucket-Move verhindert Drift in Kanban-Views.

### 3. Concurrency: Fire-and-forget + Auto-Refresh (D)
**Decision:** Bei Write-Operationen:
- Kein Optimistic Locking (keine `updated`-Check vor Write)
- SQLite regelt Locking (WAL-Modus aktiv)
- Nach jedem erfolgreichen Write: automatischer Reload der aktuellen View
- Bei Write-Fehler: visuelles Feedback (Flash der UI oder roter Footer)

**Begründung:** 1-User-Setup → Write-Write-Races unwahrscheinlich. Auto-Refresh verhindert veraltete Ansicht. Fehler-Feedback macht Probleme (DB-Lock, Server hat Task gelöscht, etc.) sofort sichtbar. Pragmatisch, einfach zu implementieren, später auf Optimistic Locking aufrüstbar falls nötig.

### 4. Detail-Pane: Gleichwertiger Pane (B)
**Decision:** Detail ist kein Modal/Overlay, sondern ein vollwertiger 3. Pane im Fokus-Graph. `Enter` in Tasks öffnet ihn; `Ctrl+W k` navigiert zurück; er kann geschlossen werden.

**Begründung:** Zukunftssicher für Edit-Features. Sobald Detail editierbar wird, muss der User zwischen Detail-Edit und Task-Auswahl wechseln können. Modal würde das blockieren. Konsistent mit Vim-Split-Philosophie.

### 5. Edit-Mode Commit-Policy (revidiert in Decision #19)
**Überschrieben:** Ursprünglich war geplant, Änderungen zu verwerfen ("einfachste Variante"). Nach Edit-Mode-Grilling (Decision #19) wurde auf **Auto-Commit** geändert. Siehe Decision #19.

### 6. Command-Line: Ins Fundament (A)
**Decision:** Command-Line (`:` öffnet Input-Modus im Footer) wird *jetzt* gebaut, als Teil des Fundaments. 

**Begründung:** Vim-motion-style Tool braucht Ex-Commands als kanonische Schnittstelle. 26 Buchstaben + 10 Ziffern + Ctrl-Leader reichen für ein umfangreiches Kanban/Backlog-Board nicht aus. Commands geben Flexibilität. `:q` / `:qa` sind ikonisch Vim — sie von Anfang an zu haben verhindert spätere Breaking-Changes.

**Minimal-Scope fürs Fundament:**
- `:q` — Pane schließen (oder App beenden, falls letztes Pane)
- `:qa` / `:quitall` — App beenden (alle Panes schließen)
- `:h` / `:help` — Hilfe anzeigen (statisch: Keybindings + Commands)
- `:errors` — Error-Log anzeigen (Platzhalter: "Not implemented" fürs Fundament)

**Erweiterbarkeit:** Command-Dispatcher muss erweiterbar sein (Plugin-Pattern oder Command-Registry). Weitere Commands (`:goto`, `/`-Suche, etc.) kommen in späteren Features.

**Autocomplete:** Erst in späteren Iterationen, wenn dynamische Commands (`:goto <project>`) relevant werden.

**Suche/Filter:** Nicht als Command (`:filter`), sondern via **`/`** wie in Neovim. `/<query>` durchsucht alles (Projekte, Tasks, Labels). Separate Befehle für fokussierte Suche ("nur Tasks", "nur Projekte") sind möglich, aber das Design ist noch offen → Feature-Grilling später.

### 7. Keybindings vs Commands: Strategie
**Decision:** Dopplungen vermeiden. Für jede Aktion entweder Keybinding ODER Command, nicht beides (außer wo Vim historisch auch doppelt).

**Prinzipien:**
- **Normal Mode vs Insert Mode** unterscheiden (wie Vim). Navigation/Toggle/Domain-Aktionen im Normal Mode via Keybindings. Edit-Felder via Insert Mode (Character-Eingabe).
- **Keybindings bevorzugen** für häufige, argumentlose Aktionen (Navigation, Toggle, Refresh). 26 Buchstaben + 10 Zahlen + Leader-Kombinationen (z.B. `Space+X`) ausnutzen.
- **Commands** für seltene, Meta-Aktionen oder solche mit Argumenten (`:qa`, `:h`, später `:goto <name>`).
- **Vim-Konsistenz** wichtiger als strikte Regeln. Wenn Vim Dopplungen hat (z.B. `?` und `:help`), dürfen wir auch.
- **Ergonomie** zählt: deutsche Tastatur, wenig Keystrokes, leicht zu merken, eindeutig.

**Konkrete Zuordnung (Fundament):**
- Navigation (`hjkl`, `gg/G`, `Ctrl+D/U`, `öä` als Alias zu `{}`?) → **Keybindings only**
- Refresh → **`r` Keybinding** (häufig, Domain) — **kein** `:r` Command
- Toggle Done → **`x` Keybinding** (häufig) — kein `:toggle` Command (noch)
- Help → **`:h`/`:help` Command** + **`?` Keybinding** (Vim-Standard-Dopplung)
- Quit → **`:q`/`:qa` Commands only** (kein `q` Keybinding). `Ctrl+C` bleibt als Notaus (App sofort beenden).

**Begründung (Decision #8):** Vim-konsistent — `q` ist in Vim nicht Quit, sondern Makro-Recording. Wir halten `q` frei für spätere Features. `:q` schließt Pane (oder App, falls letztes Pane). `:qa` beendet App sofort. `Ctrl+C` als universal-TUI-Panik-Button. 3 Tasten für `:q` sind okay, weil Pane-Close selten ist.

### 8. Quit-Keybinding: Command-only (D)
**Decision:** `q` wird kein Keybinding. Quit nur via `:q`/`:qa`. `Ctrl+C` bleibt als Notaus.

**Begründung:** Siehe oben (Decision #7 Konkrete Zuordnung).

### 9. Error-Handling: Hybrid (roter Footer + Error-Log + Retry-Intelligence)
**Decision:** Dreistufiges Error-Handling:

1. **Error-Log (`:errors` Command):**
   - Alle Fehler werden in einem Session-Log gespeichert.
   - `:errors` öffnet ein Pane (oder zeigt Liste im Footer), das alle Fehler der Session auflistet.
   - Fehler gehen nie verloren, User kann später nachschlagen.
   - **Implementierung:** Erstmal als Platzhalter (`:errors` zeigt "Not implemented"), Log-Infrastruktur später.

2. **Direktes Feedback mit Fallunterscheidung:**
   - **Triviale Fehler (z.B. DB-Lock, aber Retry erfolgreich):** Footer zeigt rechts unten "write success" (grün) für 2-3 Sekunden, verschwindet dann.
   - **Echte Fehler (nicht behoben):** Footer wird rot, zeigt Fehlertext. Bleibt rot, bis User eine Aktion macht (inkl. Cursor-Bewegung `j/k`).

3. **Auto-Refresh nach erfolgreichem Write:**
   - Nach jedem erfolgreichen Write (inkl. Retry-Success) wird die aktuelle View automatisch neu geladen (siehe Decision #3).

**Begründung:** Balance zwischen sofortigem Feedback (Footer) und Nachschlagbarkeit (Error-Log). Triviale Fehler nerven nicht (auto-dismiss). Echte Fehler sind klar sichtbar, verschwinden erst bei User-Aktion (= User hat es gesehen). Error-Log ermöglicht späteres Debugging.

### 10. Help-Content: Help-Pane (B)
**Decision:** `:h` / `:help` öffnet ein Help-Pane (wie Detail-Pane, schließbar mit `:q` oder `Esc`).

**Content-Struktur:**
- **Erste 10-20 Zeilen:** Häufigste Commands (Movement `hjkl/gg/G`, `:q`, `x`, `Enter`, `Ctrl+W`, etc.) — ohne Scrollen sichtbar (90% der Use-Cases).
- **Danach scrollbar:** Vollständige Keybinding-Liste, alle Commands, Erklärungen.

**Begründung:** Statischer Footer-Text zu kurz. Help-Pane zeigt alles, ist scrollbar. Wichtigste Commands auf ersten Blick → schneller Einstieg.

### 11. Schema-Validation: Nein (B)
**Decision:** Keine Schema-Prüfung beim Start. Wir vertrauen darauf, dass die DB-Datei eine valide Vikunja-DB ist.

**Begründung:** SQL-Fehler bei falscher DB sind klar genug (landen im Error-Log). Schema-Check wäre Mehraufwand und Maintenance-Last (bei Vikunja-Schema-Änderungen). Falls in der Praxis zum Problem wird, später nachrüstbar.

## Feature-Decisions (Grilling Kanban-View)

### 12. Kanban im Pane-Graph: Tasks-Pane rendert view_kind-abhängig (A)
**Decision:** Tasks-Pane (rechts) zeigt entweder List-View oder Kanban-View, je nach `view_kind` des ausgewählten Projekts. Pane-Graph bleibt gleich (Projects ↔ Tasks ↔ Detail).

**Begründung:** Konsistent mit aktuellem Layout. User wählt Projekt links → rechts zeigt die View (egal ob List oder Kanban). Kein separater "Kanban-Modus", nur unterschiedliches Rendering.

### 13. Projects-Pane: Collapsable via Leader+o (C-Idee)
**Decision:** Projects-Pane (links) kann ein/ausgeblendet werden (toggle) via:
- **`Leader+o`** (Space, dann `o` innerhalb 500ms)
- **`Ctrl+W o`** als Synonym (Vim-Standard für "maximize/only")

**Begründung:** Kanban braucht Breite für Buckets side-by-side. Collapsable Projects-Pane gibt maximalen Platz bei Bedarf, ohne Pane-Graph umzubauen. `Leader+o` ist aus Neovim-Plugin "Aerial" vertraut. `Ctrl+W o` ist Vim-konsistent.

**Leader-Key-Config:**
- Leader = **Space**
- Timeout = **500ms** (default, konfigurierbar)
- Leader-Sequenzen: `Leader+o` (collapse), später evtl. weitere (`Leader+f` für Filter, etc.)

**Langfristig:** Projects-Pane könnte durch andere Navigations-Panes ersetzt werden (z.B. nach Tags, Labels, Due-Date gruppiert) — dann ist der collapse-Mechanismus wiederverwendbar.

### 14. Kanban-Buckets: Dynamische Spalten + horizontaler Scroll (A, adaptive)
**Decision:** Buckets werden horizontal side-by-side gerendert. Anzahl sichtbarer Buckets ist dynamisch, basierend auf Screen-Breite.

**Layout-Logik:**
- **Mindest-Bucket-Breite:** 25 cols (damit Task-Titel lesbar bleiben)
- **Sichtbare Buckets:** `min(totalBuckets, floor(availableWidth / 25))`
  - Beispiel: 80 cols Screen, Projects-Pane collapsed → 3 Buckets sichtbar (3×25=75, Rest für Borders)
  - Beispiel: 120 cols Screen → 4-5 Buckets sichtbar
- **Horizontal Scroll:** Falls `totalBuckets > sichtbareBuckets` → User scrollt mit `h/l` (Bucket-Focus shiftet)
- **Visuelle Indikatoren:** Pfeile `<` (links) und `>` (rechts) in der Header-Zeile, wenn weitere Buckets in die jeweilige Richtung existieren.

**Typische Bucket-Anzahl:** 3-6 (Todo, Doing, Done, evtl. Review, Backlog, Blocked). Bei Standard-Terminal (80-100 cols) passen 3 Buckets ohne Scroll.

**Begründung:** Board-Gefühl erfordert mehrere sichtbare Buckets. Feste Mindest-Breite hält Tasks lesbar. Dynamische Anpassung nutzt verfügbare Breite optimal. Scroll nur wenn nötig.

### 15. Kanban-Navigation: 2D-Grid via `hjkl` (A)
**Decision:** Fokus-Modell = **ein** Bucket + **eine** Task darin.
- `h/l` — Bucket-Wechsel (horizontal, Fokus geht zur aktuellen Task-Position im neuen Bucket, oder zur ersten/letzten falls Position nicht existiert)
- `j/k` — Task-Auswahl innerhalb des aktuell fokussierten Buckets (vertikal)
- Exakt **eine** Task ist highlighted (über alle Buckets hinweg).

**Edge-Cases:**
- `h` im ersten Bucket oder `l` im letzten Bucket → **No-op**. Fokus bleibt. Visuelles Feedback: Aktueller Bucket blinkt kurz auf (sanftes Flash, z.B. Border-Highlight 200ms).
- `j` in leerem Bucket oder `k` bei erster Task → No-op (kein Wrap zu anderem Bucket).

**Begründung:** Intuitiv, schnell (kein `Ctrl+W`, keine Modi). 2D-Grid → 2D-Navigation via `hjkl`. Buckets wie Panes zu behandeln wäre zu sperrig (3 Keystrokes). Keine Verwirrung durch Wrap-around oder Modi-Switches.

**"Kanban-global" Keybindings:** Für Meta-Aktionen (z.B. Bucket umbenennen, Bucket-Reihenfolge ändern) später separate Keybinds definieren. Fokus hier: Task-Navigation + Move.

### 16. Move Tasks zwischen Buckets: Shift+H/L + m+Ziffer (C+A)
**Decision:** Zwei Move-Mechanismen:

1. **Shift+H / Shift+L — Move zu Nachbar-Bucket**
   - `Shift+H` → Task einen Bucket nach links verschieben
   - `Shift+L` → Task einen Bucket nach rechts verschieben
   - **Fokus bleibt auf der verschobenen Task** (Task "wandert mit" in den neuen Bucket)
   - **Direkt committed** (kein Confirm-Schritt)
   - **Edge-Case:** `Shift+H` im ersten Bucket oder `Shift+L` im letzten Bucket → No-op. Task blinkt kurz auf (dezentes Visual-Feedback, 200ms).

2. **m + Ziffer — Move zu beliebigem Bucket**
   - User drückt `m` (move)
   - Footer zeigt: "Move to: 1=Todo 2=Doing 3=Done 4=Review" (Bucket-Namen + Nummern)
   - User drückt Ziffer (1-9, evtl. 0 für Bucket 10)
   - Task wird verschoben, Fokus bleibt auf Task im neuen Bucket
   - `Esc` bricht ab (kein Move)

**DB-Update beim Move:**
- `task_buckets`-Tabelle: `DELETE` alten Eintrag (task_id, alte project_view_id, alter bucket_id), `INSERT` neuen Eintrag (task_id, project_view_id, neuer bucket_id)
- `tasks.updated` → bumpen (gemäß Decision #2)
- Auto-Refresh nach erfolgreichem Move (Decision #3)

**Begründung:** Die meisten Moves sind "einen Schritt weiter im Flow" (Todo→Doing→Done) → `Shift+H/L` ist dafür optimal (1 Keystroke, intuitiv). Sprünge oder viele Buckets → `m`+Ziffer (2 Keystrokes, funktioniert immer). Kombination gibt Geschwindigkeit + Flexibilität.

**Vim-Konflikt:** `m` ist in Vim "mark" (Marker setzen). Im Task-Kontext weniger relevant, daher akzeptabel. Keine bessere Vim-Motion gefunden, die semantisch passt.

### 17. Toggle Done (`x`) im Kanban-View: Auto-Move + Feedback (D)
**Decision:** Im Kanban-View funktioniert `x` wie im List-View (konsistent mit Decision #2), aber mit visueller Bestätigung.

**Verhalten:**
1. User drückt `x` auf einer Task im Bucket "Doing" (oder einem anderen Nicht-Done-Bucket).
2. Task wird auf `done=1` gesetzt.
3. Task wird **automatisch in den `done_bucket_id` verschoben** (falls die View einen hat).
4. **Fokus bleibt auf der Task** (die jetzt im "Done"-Bucket ist) — User "wandert mit" der Task.
5. **Visuelles Feedback:**
   - Footer zeigt 2 Sekunden: "Task marked done & moved to 'Done'"
   - Task im "Done"-Bucket blinkt kurz auf (200ms, falls sichtbar)
6. `tasks.updated` wird gebumpt, Auto-Refresh (Decision #3).

**Begründung:** Konsistent mit List-View und Decision #2 (done→done_bucket). Vikunja's Semantik respektiert. Fokus bleibt auf Task → konsequent, User weiß immer, wo er ist. Visuelles Feedback verhindert Verwirrung ("wo ist meine Task?").

**Alternative erwägt:** `x` nur in Listen-View erlauben, im Kanban deaktivieren. Verworfen, weil inkonsistent und umständlich (User müsste erst manuell in Done-Bucket moven).

**TODO (später):** Undo-Feature für letzten Move/Toggle im Kanban (z.B. `u` wie Vim). Hilfreich, falls User versehentlich `x` drückt.

---

## Edit-Mode Grilling (Prio 2)

### 18. Edit-Mode: Hybrid — Title inline + Detail-Pane für Rest (C)
**Decision:** Zwei Edit-Modi, je nach Kontext:

**1. Title-Edit inline (in List/Kanban-Pane):**
- User fokussiert Task in List- oder Kanban-View
- **`i` (insert)** → wechselt in Insert-Mode für Task-Title (inline)
- Textfeld erscheint direkt in der Liste (Task-Zeile wird zu Edit-Widget)
- **`Esc`** → zurück zu Normal-Mode (Title-Edit verwerfen, siehe Decision #5)
- **`Enter`** oder **Fokus-Verlust** → Commit (Title speichern, DB-Update)

**2. Multi-Field-Edit im Detail-Pane:**
- **`Enter`** auf Task in List/Kanban → Detail-Pane öffnet (wie bisher)
- `Ctrl+W j` (oder entsprechender Pane-Wechsel) → Fokus zu Detail-Pane
- Im Detail-Pane:
  - **Normal-Mode (default):** `hjkl` navigiert zwischen Feldern (Title, Description, Priority, Due, Done, etc.)
  - **`i` bei fokussiertem Feld** → wechselt in Insert-Mode für dieses Feld
  - **`Esc` im Insert-Mode** → zurück zu Normal-Mode (Feld-Navigation), Änderungen verwerfen (Decision #5)
  - **Commit:** Bei Feld-Wechsel (`jk` im Normal-Mode) oder Pane-Wechsel (`Ctrl+W`) → geänderte Felder werden gespeichert? (TODO: Commit-Logik klären)

**Sonderbehandlung pro Feld (TODO Prio2):**
- **Due-Date:** Date-Picker oder ISO-Format-Eingabe (`YYYY-MM-DD`)?
- **Priority:** Nummer-Eingabe (0-5) oder `+`/`-` zum Inkrementieren?
- **Done:** Toggle via `x` (wie bisher), oder editierbares Checkbox-Feld?
- **Description:** Multi-Line-Textfeld (Vim-Motion `j/k` zum Scrollen, `i` zum Editieren)

**Begründung:** Title-Ändern ist die häufigste Edit-Aktion → inline schnell (1 Keystroke: `i`). Komplexe Edits (Multi-Field, Description) brauchen mehr Platz → Detail-Pane. Konsistent mit Vim's Normal/Insert-Trennung: Normal-Mode = Navigation (zwischen Tasks, zwischen Feldern), Insert-Mode = Editing.

### 19. Detail-Pane Edit: Auto-Commit bei Pane-Wechsel (C, revidiert Decision #5)
**Decision:** Änderungen werden automatisch committed, wenn User den Kontext verlässt.

**Commit-Trigger:**
- **Pane-Wechsel** (`Ctrl+W` aus Detail-Pane raus) → alle pending changes werden committed (ein DB-Write für alle geänderten Felder)
- **Item-Wechsel** (User geht zurück zu List/Kanban, wählt andere Task mit `Enter`) → Commit
- **Feld-Wechsel** (`Esc` im Insert-Mode, dann `jk` zu anderem Feld) → pending (noch nicht committed), erst bei Pane-Wechsel

**Title-Edit inline (extra):**
- **`Enter`** im Title-Insert-Mode → sofort committed (DB-Write)
- **Fokus-Verlust** (User navigiert weg, z.B. `jk` zu anderer Task) → sofort committed

**DB-Update beim Commit:**
- Alle geänderten Felder in einem `UPDATE tasks SET ... WHERE id=?`
- `updated` → bumpen (Decision #2)
- Auto-Refresh nach erfolgreichem Write (Decision #3)
- Bei Fehler → roter Footer + Error-Log (Decision #9)

**Begründung:** Auto-Commit ist "modern" (wie Jira, Notion), User verliert nie Arbeit. Einfacher als explizites `:w`/`:q!` (weniger Verwirrung: "Wann verwerfen? Wann informieren?"). Revidiert Decision #5 ("verwerfen"), weil Auto-Commit bei gleichem Implementierungsaufwand bessere UX bietet.

**Kein explizites Verwerfen:** Falls User Änderungen rückgängig machen will → späteres Feature (Undo, `u`-Keybinding). Fürs Fundament: Auto-Commit ohne Escape-Hatch.

**Offene Entscheidungen (für Edit-Features später):**
- Welche Keybindings für häufige Task-Attribute? (Prio setzen, Datum setzen, Status ändern)
- `i` für Insert Mode in Edit-Feldern? (naheliegend)
- Leader-Key: `Space` oder etwas anderes?
- `öä` als Block-Scroll-Alias — sinnvoll auf dt. Tastatur? (zu testen)

## Rendering (MVP)
- Linke Spalte:
  - Überschrift: `Projects`
  - Liste: `> <identifier or title>`
- Rechte Spalte:
  - Überschrift: `Tasks (list)`
  - Liste: `[%] <priority> <title> (due)`
  - Done: Checkbox / Marker

## Status

**Aktuell (2026-07-09):** 
- PoC (Proof of Concept) läuft — TUI zeigt DB-Inhalt, erste Struktur sichtbar, Navigation funktioniert, `x` togglet Done.
- **Grilling Session abgeschlossen** — Fundament komplett geklärt (Decisions #1-11), Kanban (Prio 1, #12-17) und Edit-Mode (Prio 2, #18-19) durchdesigned.

**Nächstes Ziel:** Implementierung beginnen
1. **Fundament aufräumen:**
   - REST-Client (`client.go`) entfernen
   - Command-Line-Infrastruktur bauen (`:q`, `:qa`, `:h`, `:errors` als Platzhalter)
   - Error-Handling + Error-Log (roter Footer, Auto-Refresh, Retry-Intelligence)
   - Help-Pane (`?` / `:h`)
   - Leader-Key-Support (`Space` + 500ms Timeout)
   - Projects-Pane collapsable (`Leader+o` / `Ctrl+W o`)

2. **Feature Prio 1: Kanban-View**
   - Tasks-Pane rendert `view_kind=kanban` (Buckets horizontal, dynamische Breite)
   - Navigation `hjkl` (2D-Grid)
   - Move via `Shift+H/L` + `m+Ziffer`
   - Toggle Done (`x`) mit Auto-Move + Feedback

3. **Feature Prio 2: Edit-Mode**
   - Title inline via `i`
   - Detail-Pane Multi-Field-Edit (`hjkl` Feld-Navigation, `i` Insert-Mode)
   - Auto-Commit bei Pane-Wechsel

**Offene TODOs (spätere Increments):**
- Prio 3: Gesamtsicht (`-` Keybinding, alle Tasks, Filter/Sort)
- Prio 4: Suche/Filter (`/`)
- Prio 5: Kommentare
- Undo-Feature (`u`)
- Neue Tasks erstellen (`o`)
- Sonderbehandlung (Date-Picker, Priority-Eingabe)

## Roadmap / Backlog

### Fundament (in diesem Grilling geklärt)
- [x] Store: SQLite-only, kein REST
- [x] Write-Semantik: Treue Side-Effects (`updated` + Bucket-Move)
- [x] Concurrency: Fire-and-forget + Auto-Refresh
- [x] Detail-Pane: Gleichwertiger Pane (3. im Graph)
- [x] Command-Line: `:q`, `:qa`, `:h` ins Fundament
- [x] Command-Line Scope: `:q`, `:qa`, `:h`, `:r` (mit Aliassen)
- [x] Keybindings vs Commands: Strategie geklärt (Decision #7)
- [x] Error-Handling: Strategie geklärt (Decision #9)
- [x] Help-Content: Help-Pane (Decision #10)
- [x] Schema-Validation: Nein, vertrauen (Decision #11)

### Features (Priorisierung geklärt)

**Prio 1: Kanban-View** (`view_kind=kanban`)
- [x] Pane-Graph: Tasks-Pane rendert Kanban je nach `view_kind` (Decision #12)
- [x] Projects-Pane: Collapsable via `Leader+o` / `Ctrl+W o` (Decision #13)
- [x] Buckets: Dynamische Spalten, horizontal scroll (Decision #14)
- [x] Navigation: `h/l` = Bucket-Wechsel, `j/k` = Task innerhalb Bucket (Decision #15)
- [x] Move Tasks: Shift+H/L (Nachbar) + m+Ziffer (Sprünge) (Decision #16)
- [x] Toggle Done (`x`) im Kanban: Auto-Move + Feedback (Decision #17)
- [ ] Move Tasks zwischen Buckets (`m` + Target-Auswahl?)
- [ ] Horizontal Scroll bei vielen Buckets?
- [ ] Detail-Pane bei Kanban-Ansicht? (unverändert: Enter öffnet Detail unten)

**Weitere Kanban-TODOs (Implementierungs-Details, Prio 2):**
- [ ] **Sort-Order innerhalb Bucket:** Nach `task_positions.position` (wie List-View). Tasks vertikal in der Position, die Vikunja speichert.
  - **TODO später:** Vertikales Movement/Reordering ermöglichen (Task innerhalb Bucket via `Ctrl+J/K` verschieben? Vikunja unterstützt das, wäre konsistent).
- [ ] **Leere Buckets:** Bucket-Name als Header zeigen, darunter einfach leer (kein "(no tasks)"-Text).
- [ ] **Bucket-Header:** Nur Bucket-Name zeigen (kein Task-Count). Design: Name zentriert oder linksbündig?
- [ ] **Horizontal Scroll Indikator:** Pfeile `<` (links) und `>` (rechts) im Header-Bereich, wenn weitere Buckets existieren.

**Prio 2: Edit-Mode** für Task-Details
- [x] Wo findet Edit statt? Hybrid: Title inline, Rest im Detail-Pane (Decision #18)
- [x] Title-Edit inline: Commit bei `Enter` oder Fokus-Verlust (Decision #18)
- [x] Detail-Pane Edit: Feldauswahl via `hjkl`, Insert-Mode via `i` (Decision #18)
- [x] Commit-Logik: Auto-Commit bei Pane-Wechsel/Item-Wechsel (Decision #19, revidiert #5)
- [ ] Date-Picker für Due-Date (Sonderbehandlung, TODO Prio2)
- [ ] Priority: Nummer-Eingabe oder `+`/`-`? (Sonderbehandlung, TODO Prio2)
- [ ] Neue Tasks erstellen: Keybinding `o` (wie Vim "open line below")? (später)

**Prio 3: Gesamtsicht** ("All Tasks"-View)
- [ ] Neue Ansicht: ALLE Tasks aus allen Projekten, flache Liste
- [ ] Erreichbar via `-` Keybinding (wie `:Explore` in Vim)
- [ ] Filter: Erledigte Tasks ausblenden (Toggle-Keybinding?)
- [ ] Sortierung: Nach Due-Date, Prio, Project, etc. (Keybindings für schnelles Umschalten?)
- [ ] Pane-Graph: Eigene Pane, oder ersetzt Projects+Tasks?

**Prio 4: Suche/Filter** (`/` wie Neovim)
- [ ] `/` öffnet Search-Input im Footer
- [ ] Client-side Filter (nur aktuelle Page) oder Server-side?
- [ ] Match-Highlighting in Projects/Tasks/Labels
- [ ] `n` / `N` für next/prev Match (wie Vim)

**Prio 5: Kommentare** posten
- [ ] Detail-Pane zeigt Comments-Liste
- [ ] `a` (append) öffnet Insert-Mode für neuen Comment
- [ ] Store-Methode: `CreateComment(ctx, taskID, text)`
- [ ] Comment-Edit/Delete später

**Später:**
- [ ] Table-View (`view_kind=table`)
- [ ] Gantt-View (`view_kind=gantt`)
- [ ] Weitere Ex-Commands (`:goto <project>`)
- [ ] Autocomplete für Commands mit dynamischen Argumenten
