# Vimkunja

## 1. Was es ist

Vimkunja ist eine tastaturorientierte Terminal-Oberfläche für eine lokale [Vikunja](https://vikunja.io/)-SQLite-Datenbank. Sie richtet sich an Vim-Nutzer und ergänzt Vikunjas Task-Verwaltung um einen schnellen Terminal-Workflow mit Projects, List View, Task-Detail und Workflow-Status.

### Installation und erster Start

Voraussetzungen: Git, ein interaktives Terminal und eine aktuelle, gepflegte Go-Version. Das Modul verwendet Go 1.25 als Sprachminimum; zum Bauen eine aktuelle sicherheitsgepatchte Toolchain verwenden (CI nutzt `stable`). Linux amd64 wurde lokal geprüft; Windows amd64 lässt sich bauen, native Windows-Terminaltests stehen noch aus.

Die vorgesehene Clone-URL lautet:

```bash
git clone https://github.com/JanWartenberg/vimkunja.git vimkunja
```

```bash
cd vimkunja
```

Zum Ausprobieren ohne eigene Datenbank:

```bash
go run ./cmd/vimkunja-tutorial
```

Beide Programme bauen (unter Windows erhalten die Binärdateien die Endung `.exe`):

```bash
go build -o bin/ ./cmd/vimkunja ./cmd/vimkunja-tutorial
```

Die gebauten Programme lassen sich aus dem Repository-Verzeichnis starten:

```bash
./bin/vimkunja-tutorial
```

```bash
./bin/vimkunja --db "/path/to/vikunja.db"
```

Unter Windows PowerShell beispielsweise:

```powershell
.\bin\vimkunja.exe --db "C:\path\to\vikunja.db"
```

Für echte Tasks alternativ direkt aus dem Quellcode eine vorhandene Vikunja-SQLite-Datenbank angeben:

```bash
go run ./cmd/vimkunja --db /path/to/vikunja.db
```

`/path/to/vikunja.db` durch den eigenen Pfad ersetzen und bei Leerzeichen in Anführungszeichen setzen. Der Datenbankpfad muss eine vorhandene, lesbare und beschreibbare Datei sein. Vimkunja greift direkt auf SQLite zu; es gibt keinen REST-Modus.

Ohne `--db` verwendet Vimkunja lokal `internal/db/vikunja.db`. Diese Datei ist nicht Teil des Repositories. Neue Installationen müssen eine eigene 2.7.0-Datenbank mit `--db` angeben. Ältere Datenbanken vor einem Upgrade einschließlich vorhandener WAL-Datei sichern und das Upgrade zuerst auf einer Kopie mit dem offiziellen Vikunja-2.7.0-Programm prüfen; Vimkunja führt keine Migrationen aus.

### Datenbank und Kompatibilität

Eine blanke SQLite-Datei enthält keine Vikunja-Tabellen und reicht für einen produktiven Start nicht aus. Vimkunja erstellt oder migriert kein Vikunja-Schema. Für einen frischen produktiven Datenbestand muss Vikunja selbst die Datenbank initialisieren; User, Project und View müssen für den gewünschten Workflow vorhanden sein. Ein dauerhaftes Start-Template ist noch nicht spezifiziert. Die synthetische Tutorial-Datenbank ist ausschließlich zum Ausprobieren gedacht und wird beim regulären Beenden gelöscht.

**VOA** bezeichnet im Folgenden die Vikunja-Original-App bzw. ihren Server. Einziges geprüftes Kompatibilitätsziel ist **Vikunja 2.7.0**. Echte Server-Roundtrips mit frischer Datenbank und einer Kopie der synthetischen Windows-Baseline sind im [Prüfbericht](.archive/2026-10-05/vikunja-db-compatibility/report.md) dokumentiert. Frühere, spätere und Entwicklungsversionen sind nicht unterstützt. Der produktive Start prüft Migrationshistorie und benötigte Schema-Merkmale und lehnt abweichende oder unvollständige DB-Revisionen vor fachlichen Schreibzugriffen ab. Releases mit identischer DB-Revision sind aus der Datei allein nicht unterscheidbar und erhalten keine zusätzliche Freigabe. Die Tutorial-Datenbank ist kein produktives Start-Template.

VOA vor dem Start von Vimkunja beenden; Vimkunja vor dem Neustart von VOA beenden. Parallele Bearbeitung ist nicht verifiziert. Tasks im Papierkorb bleiben verborgen und können nicht bearbeitet werden. Archivierte Projects sind schreibgeschützt; Bucket-Limits werden berücksichtigt. Task-Nummern bleiben nach Löschungen reserviert; alte Adressen nach Project-Wechseln und Undo/Redo sind über VOA API v2 auflösbar.

Vimkunja löscht Tasks weiterhin **endgültig**, ohne Papierkorb oder Lösch-Undo; Attachment-Dateien bleiben erhalten. Tasks mit Time Entries lassen sich nur in VOA löschen. Wiederkehrende Tasks ebenfalls in VOA abschließen. Das Erstellen von Labels sowie das Schreiben/Löschen von Comments setzt genau einen lokalen User voraus. Direkte SQLite-Schreibzugriffe führen keine Serveraktionen wie Webhooks, Benachrichtigungen oder Reminder-Dienste aus. Native Windows-Terminaltests und ein Live-CalDAV-Test stehen weiterhin aus.

Vor dem Arbeiten mit echten Daten ein Backup erstellen. Native Richtext-Bearbeitung ist implementiert; weitere Views sind geplant; der aktuelle Funktionsstand steht in [STATUS.md](STATUS.md).

### Interactive task tutorial

```bash
go run ./cmd/vimkunja-tutorial
```

The separate executable opens one flat menu with four modes and Quit. Use `j/k` or arrows to select, `Enter` to start, and `q` to leave. Learn the basics is selected by default.

| Menu option | What it does | Direct-start flags |
| --- | --- | --- |
| Learn the basics | Result-based tasks for navigation, editing, Tasks and Views | `--guided` |
| Practice the basics | Explore demo Tasks freely without lessons | `--practice` |
| Learn markup | Result-based tasks in the real rich-text Description editor | `--markup` |
| Practice markup | Explore rich-text examples freely without lessons | `--markup --practice` |
| Quit | Exit without creating a database | — |

Both Learn modes accept free application input: navigate, correct mistakes, Paste, undo or use alternative commands. A short context modal introduces each task; its top-right counter shows the current task and course total (also in F1 help). Enter or Esc dismisses it. The task strip shows only the concrete target and controls, with a colored left marker and no filled background. F1 opens optional help. The tutorial checks actual application results, never a prescribed key sequence. Return to Normal mode after editing; success remains visible while you explore further.

F5 restores the **current task's complete starting state**, including saved edits, deleted demo data and local history. F8 starts the next task only after success; the final F8 shows completion. **F12 skips the current task**, including from context/help modals or during editing; seven presses from startup land on task 8. Skipped tasks are reported separately at the end and do not count as passed. Each task has independent disposable data, so experiments do not affect later tasks. At completion F5 repeats the last task; Ctrl+C exits at any time. `--tasks` remains an alias for the general Learn mode and cannot be combined with other mode flags.

Mode flags skip the menu. Timeout flags and `--literal-tab-alias` alone keep the menu and apply to the selected mode. `--guided` and `--practice` cannot be combined.

All modes run the real Vimkunja UI in English. Starting a mode creates a freshly seeded SQLite database in the operating system's temporary directory; leaving the menu does not. The tutorial accepts no database path, never opens `internal/db/vikunja.db` or another user database, and deletes its disposable database on normal exit.

### Rich-text workshop

```bash
go run ./cmd/vimkunja-tutorial --markup
```

This course uses the real Description editor and independent disposable task fixtures. It covers Native navigation, Visual formatting, Links, typed syntax, Code creation and exits, literal Tabs, retab, Opaque confirmation, Undo/Redo, save/discard and blocked-source recovery. Start the general guided tutorial directly with `--guided`, or select it in the start menu. The workshop sets tabstop/indent to 4 and defaults to the literal-Tab alias Ctrl+Q. Use `--markup --literal-tab-alias ctrl+b` if the terminal intercepts Ctrl+Q; the lesson then teaches Ctrl+B.

For unrestricted testing in the same disposable fixtures, use `go run ./cmd/vimkunja-tutorial --markup --practice`. Open the last Task with Enter, G, Enter, then use gn, Enter to activate Description. This mode accepts arbitrary keys and Paste without task guidance. Both guided courses also accept free input and real Ex-command completion; they recognize results instead of tracking command characters. In the Link task, any valid URL is accepted; Enter applies it, empty input unlinks, and Esc cancels. Invalid URLs stay editable and neither cancel nor unlink satisfies the task to create a Link.

Code creation enters Insert mode. Inside Code, `o`/`O` add source lines; `ü`/`Ü` or `]Space`/`[Space` create paragraphs outside Code. Three or four directly typed backticks in an otherwise empty paragraph create Code only after Enter. With six directly typed backticks, move between the third and fourth and press Enter to create an empty Code block. Pasted markers remain literal. Real Tabs throughout the Description, including ordinary paragraphs and Code, appear as muted `>~~~` markers, sized to the next tabstop; retab replaces them with ordinary spaces. If your terminal intercepts Ctrl+V or Ctrl+Q, configure `literal_tab_alias = "ctrl+b"` for normal use; the workshop also accepts `--literal-tab-alias ctrl+b`.

Existing unsupported HTML and Mentions stay protected locally: Native neighboring text remains editable. Deleting an Opaque atom requires confirmation. Blocked descriptions require explicit `:descriptionrecover` or `:descriptionclear`; both change a draft until `:w`. `:e!` discards edits, and external save conflicts require an explicit `:descriptionreload` before continuing; they are never silently merged.

### Local TUI configuration

Vimkunja optionally reads TOML from the platform's standard user configuration directory:

```text
<UserConfigDir>/vimkunja/config.toml
```

For example, on Linux this is usually `~/.config/vimkunja/config.toml` (or the directory selected by `XDG_CONFIG_HOME`). Vimkunja does not create this file automatically. A complete example is:

```toml
[tui]
# How long Space waits for the second Leader key.
leader_timeout = "500ms"

# How long Ctrl-W waits for a pane direction or cycle key.
ctrl_w_timeout = "2s"
# Code editing: independent tab width and indent step (2 or 4).
tabstop = 4
indent_step = 4
# Literal Tab prefix in Code Insert: ctrl+q, ctrl+b, or "" to disable the alias.
literal_tab_alias = "ctrl+q"
```

Timeouts use Go duration syntax, such as `250ms`, `1s`, or `1.5s`. Values must be positive; zero and negative values are invalid. The built-in defaults are `500ms` for `leader_timeout` and `2s` for `ctrl_w_timeout`.

The command-line equivalents override the TOML values:

```bash
go run ./cmd/vimkunja --db "/path/to/vikunja.db" --leader-timeout 1s --ctrl-w-timeout 750ms
```

The implemented precedence is CLI flags, then TOML, then built-in defaults. Environment-variable overrides are not implemented. Configuration is read at startup; edit the file and relaunch Vimkunja to apply changes. `--help` lists the CLI flags.

A missing file is normal. Invalid TOML, unreadable files, unknown sections/settings, and invalid TOML values are recoverable: supported unambiguous settings still apply, invalid fields use their defaults, and Vimkunja starts with warnings. Warnings are printed to stderr, shown briefly in the red startup footer, and retained in `:errors`. Invalid timeout values supplied explicitly on the command line are fatal and prevent startup.

## 2. Bedienung und Details

- `hjkl`, `gg` / `G`, `Ctrl+D` / `Ctrl+U` – Navigation
- `Ctrl+W h/j/k/l` – Pane-Fokus wechseln
- Im Projects Pane: `h/l` Hierarchie schließen/öffnen bzw. Parent/Child wählen, `zo/zc/za` falten, `zM/zR` alles schließen/öffnen
- Im Projects Pane: `Shift+J/K` Project unter seinen Geschwistern verschieben
- `Enter` – Task-Detail öffnen
- `x` – Task in den Done- bzw. Default-Bucket togglen
- `u` / `Ctrl+R` – letzte persistierte Task-Aktion rückgängig machen bzw. wiederholen
- `o` / `:new` – neuen Task anlegen
- `:w`, `:wq`, `:wqa` – speichern
- `:e!` / `:edit!` – ungespeicherte Änderungen am bestehenden Task verwerfen, Task geöffnet lassen
- `:q`, `:q!`, `:qa`, `:qa!` – Pane bzw. Anwendung schließen
- `:h` – integrierte Hilfe
- `:errors` – Session-Fehlerlog
- `Space+o` – Projects-Pane ein- oder ausblenden
- `:kanban` oder `Space+v`, dann `k` – einzelne manuelle Kanban View öffnen oder nach Bestätigung standardkonform provisionieren
- Im Kanban Board: `h/l` Buckets, `j/k` Karten, `Shift+H/L` Nachbar-Move, `m` Ziel-Bucket wählen

Titel, Description und Comment-Entwürfe unterstützen im Normal Mode `v/V` zur Zeichen-/Zeilenauswahl (Description: logische Blöcke), `Space+y` zum Kopieren der Auswahl bzw. aktuellen Zeile/Absatzes in die Systemzwischenablage und `Space+p` zum Einfügen am Cursor bzw. Ersetzen der Auswahl. Copy erhält die Auswahl; Paste bleibt im Normal Mode. `Esc` hebt die Auswahl auf. `Space+Y` kopiert die Ziel-URL eines Links. **Descriptions haben einen gemeinsamen Editor:** `gn`, `Enter` aktiviert ihn auch für bisher leeren oder unformatierten Text. Auswahl, Formatierung und Tippen verwenden sofort dieselben Regeln; ein Richtext-Moduswechsel ist nicht erforderlich. Öffnen und Navigation ändern keine gespeicherten Bytes. Erst eine echte Bearbeitung erzeugt Richtext im Entwurf; `:w` speichert, `:e!` verwirft. Einfügen aus der Zwischenablage interpretiert keine Markdown-Formatierung.

Texteingaben verwenden einheitlich einen dünnen, blinkenden Terminalcursor: Title, Description einschließlich Richtext/Code, Comments sowie Such-, Command-, URL- und Label-Prompts. Normal Mode verwendet einen Blockcursor. Die Datumsmaske markiert die zu ersetzende Ziffer als Überschreibcursor; relative Datumseingaben verwenden den dünnen Eingabecursor.

Im Detail-Pane können Project, Title, Priority, Due Date, Labels, Description und der Workflow-Status bearbeitet werden. `gc` öffnet den Comments-Workspace; `gca` startet dort einen neuen Comment. Im Comment-Thread starten `a`, `i` und `o` einen neuen Comment; ein statischer Blockcursor markiert den ausgewählten Comment. `Space+l` öffnet den Label-Picker; `:labels` verwaltet globale Labels. Der Status verwendet die konfigurierten Buckets der eindeutigen Kanban View des Projects.

HTML-Descriptions werden strukturiert angezeigt und als Native Draft bearbeitet. `gn`, dann `Enter` aktiviert die Navigation: `j/k` zwischen logischen Blöcken (in Code zwischen Quellzeilen, in Zitaten zwischen Kindblöcken), `gj/gk` zwischen sichtbaren Zeilen, `{`/`}` oder `ö/ä` zwischen ganzen Absatz-Einheiten. `h/l`, `0/^/_/$`, `w/b/e/ge`, Counts und `gg/G` bewegen den Dokumentcursor; `Ctrl+D/U` bewegt ihn ungefähr eine sichtbare Pane-Höhe. `zh/zl` scrollt Code und Tabellen horizontal, `Ctrl+E/Y` nur den Viewport. `Esc` verlässt die aktive Navigation. Der aktive Block zeigt Formatierungszeichen; Code zeigt geschützte Backtick-Ränder. Lesen und Resize ändern das gespeicherte HTML nicht. `i/a/I/A` startet Insert, `Enter` teilt Absätze, `Shift+Enter` setzt einen Hard Break. `o/O` erzeugt einen Absatz, innerhalb von Code eine Quellzeile. `dd` löscht logische Blöcke bzw. Codezeilen. `v/V`, `viw/vaw` und `o` wählen Text/Blöcke bzw. tauschen das aktive Ende; `Esc` beendet die Auswahl. In beiden Visual-Modi (`v` und `V`) formatiert `Space+b/i/u/s`, `Space+0` entfernt Inline-Formatierung einschließlich Links. `Space+x` wandelt jeden von der Auswahl berührten logischen Absatz/Listeneintrag in eine Checkbox um; bestehende Checkboxen behalten Zustand und Identität. Die Auswahl bleibt bis Esc bestehen. Normal `Space+1/2/3/0` wandelt Absatz/Heading um. Die Palette steht innerhalb des Footer-Rahmens und bleibt nach dem Leader-Timeout bis zur nächsten Taste lesbar: `SPACE ACTIVE` erlaubt direkte Kürzel, `Space again` verlangt erneut Space. Schmale Fenster umbrechen die Kürzel; bei sehr wenig Höhe ersetzt eine kompakte Palette die normale Footer-Zeile. `Space+o` bleibt global. `u/Ctrl+R` gilt hier für den lokalen Description-Draft. Opaque-Löschung, Ersatz und Änderungen am Native Parent benötigen eine Bestätigung (`y/Enter`, `n/Esc`); Undo erhält die ursprünglichen Bytes. `y/p` kopiert sichere sichtbare Inhalte in den lokalen Description-Register und fügt Native Text ein.

Native Inline-Syntax entsteht nur durch bewusstes Tippen: `**fett**`, `*kursiv*`, `***beides***`, `__unterstrichen__`, `~~durchgestrichen~~` und einzelne gepaarte Backticks für Inline-Code. Paste bleibt wörtlicher Text; Backslash schützt das nächste Syntaxzeichen. `Space+l` bearbeitet die URL eines Links oder verlinkt ausgewählten Text, eine leere URL entfernt den Link. Ohne Auswahl und außerhalb eines Links fragt `Space+l` zuerst nach der URL, dann nach dem Linktext. Das Textfeld startet leer; direktes Tippen oder Paste füllt es. `Enter` bei leerem Text übernimmt die URL als Text. `Ctrl+U` leert die Eingabe, `Esc` bricht beide Schritte ohne Änderung ab. Einmal `u` macht das Einfügen rückgängig. `Space+Y` kopiert das Ziel in die System-Clipboard. Einfügen am Link-Rand oder außerhalb der sichtbaren Inline-Code-Backticks bleibt außerhalb; zwischen den Backticks wird der Code bearbeitet.

`:selectblock` wählt ausdrücklich den ganzen Codeblock; `V` in Code bleibt auf dessen Quellzeilen begrenzt. Für Blocked HTML bieten `:descriptionrecover` und `:descriptionclear` bestätigte, ungespeicherte Recovery-Aktionen. `:e!` verwirft weiterhin den gesamten Task-Draft; `:descriptionreload` lädt den aktuellen Task aus der Datenbank und verwirft den Draft ausdrücklich. Ein Save-Konflikt behält den Draft offen, ohne Merge. Mit `Space+u/n/x` lassen sich Absätze in Listen/Checkbox-Listen umwandeln; `x` toggelt die Description-Checkbox unabhängig von Task-Done. `V` → `Space+k/l` rückt ausgewählte Einträge aus/ein, `Space+q/w` entfernt/ergänzt die Zitatebene. Normal `Space+q` toggelt lokal. Metadaten und Reihenfolge bleiben erhalten; nicht verlustfrei darstellbare Aktionen werden unverändert abgelehnt. Code wird mit Normal `Space+c` oder genau drei/vier direkt getippten Backticks plus Enter in einem leeren Absatz erstellt; bei sechs direkt getippten Backticks erzeugt Enter zwischen dem dritten und vierten einen leeren Codeblock. `ü/Ü` erzeugt danach/davor einen Absatz in Insert; `]Space/[Space` dasselbe in Normal. Code-`Tab` fügt Spaces bis zum nächsten Tabstop ein, `Ctrl+V Tab` (alternativ `Ctrl+Q Tab`, konfigurierbar) einen echten Tab; Paste erhält Tabs und CRLF. `V` → `Space+x/c` rückt Quellzeilen aus/ein; `:retab` wandelt Tabs im aktuellen Code bzw. ausgewählten Codezeilen um, `:retaball` in allen Native-Texten und überspringt Opaque. Beide ohne Argumente/Bang, jeweils ein Undo-Schritt. Leere, unformatierte und HTML-Descriptions verwenden denselben Dokumenteditor; Title und Comment Composer teilen ihre Textauswahl- und Zwischenablagebedienung.

In den Command-Prompts von Comments, Labels und Global Search gelten ebenfalls `:qa`, `:qa!` und `:wqa`. `:qa` schützt ungespeicherte Task-/Comment-Entwürfe; `:wqa` speichert einen vorhandenen Comment-Entwurf zusammen mit dem Task atomar und beendet erst nach erfolgreichem Speichern. Bei Fehlern bleibt der Entwurf offen. Im Insert Mode zuerst `Esc` drücken.

Ein geladener, unveränderter Workflow-Status wird beim Task- oder Comment-Save nicht erneut angewendet. Dadurch bleiben `Done` und der vorhandene Abschlusszeitstempel bei anderen Änderungen erhalten. Ein tatsächlicher Bucket-Wechsel synchronisiert den Status weiterhin wie bisher. Bei unverändertem `Done` bleibt der gespeicherte Abschlusszeitstempel auch bei Done-/Bucket-Schreibvorgängen und Undo/Redo exakt erhalten. Ein echter Wechsel zu erledigt setzt einen aktuellen Zeitstempel, Wiederöffnen löscht ihn. Abschlusszeitstempel bleiben außerhalb der Undo-Historie.

Bucket-Konfigurationen mit `0` oder NULL gelten als nicht gesetzt. Ohne Default-Bucket wird der erste gültige Bucket nach Position verwendet; bei gleicher Position entscheidet die ID. Das gilt für neue Tasks, Project-Wechsel und das Wiederöffnen. Ohne Done-Bucket bleibt beim Abschließen die aktuelle Zuordnung erhalten. Gibt es keinen gültigen Bucket, wird keine Zuordnung erfunden. Ungültige konfigurierte Bucket-Verweise brechen den Schreibvorgang atomar ab.

Beim Verschieben eines erledigten Tasks in ein anderes Project wird je Ziel-View der konfigurierte Done-Bucket verwendet. Fehlt dieser, bleibt die Default-Zuordnung. Ein reiner Project-Wechsel erhält `Done` und den Abschlusszeitstempel.

Neue Tasks erhalten eine stabile UUID für CalDAV, sofern die Datenbank eine `uid`-Spalte enthält. Änderungen, Project-Wechsel und Undo/Redo erhalten diese Identität. Bestehende Tasks werden nicht nachträglich geändert; ein Live-CalDAV-Test steht weiterhin aus.

Die permanente Task-Löschung entfernt auch zugehörige Favoriten, Subscriptions sowie Task- und Comment-Reaktionen. Andere Objekttypen und Attachment-Dateien bleiben erhalten; die DB-Löschung erfolgt atomar.

Wiederkehrende Tasks können in Vimkunja bearbeitet und wieder geöffnet werden. Der Wechsel zu erledigt wird abgelehnt: Den Abschluss in Vikunja durchführen, damit Termine, Reminder und Checklisten für die nächste Wiederholung aktualisiert werden.

Fälligkeitsdaten werden weiterhin als `YYYY-MM-DD` bearbeitet. Solange das Datum unverändert bleibt, erhalten Task-Saves und Undo/Redo die vorhandene Uhrzeit exakt. Ein geändertes Datum speichert Mitternacht UTC; das Löschen des Datums speichert NULL. Undo/Redo weist externe Änderungen am vollständigen Fälligkeitszeitstempel zurück.

While a Task Detail save is running, keyboard input pauses and the footer shows
`Saving Task…`; `Ctrl+C` remains available for an emergency exit. Input resumes
after success or failure. Failed saves keep the draft for retry. A late
asynchronous draft change stays unsaved and prevents save-and-close or
save-and-quit from discarding it.

Tests:

```bash
go test ./...
```

## 3. Ausblick / Vision

Das Fundament ist abgeschlossen: List View, Editor, Workflow-Status, Fehleranzeige, Vim-artige Pane-Navigation, die einzelne manuelle Kanban View und die hierarchische Projects Pane sind verifiziert. Weitere Views sowie Bucket-Verwaltung bleiben im Backlog.

Der aktuelle, priorisierbare Backlog liegt in `.scratch/`; der Kurzstatus mit Verweisen steht in [STATUS.md](STATUS.md). Abgeschlossene Anforderungen und Implementierungsdetails liegen im [Archiv](.archive/).

## Lizenz

Der eigene Vimkunja-Code steht unter der [MIT-Lizenz](LICENSE). Abhängigkeiten behalten ihre jeweiligen Lizenzen; versionierte Zuordnung, Lizenztexte und ergänzende Hinweise stehen in [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt). Beide Dateien bei Binary-Distributionen mitliefern und die dort dokumentierten komponentenspezifischen Bedingungen erfüllen.

Vikunja ist ein separates [AGPL-3.0-Projekt](https://github.com/go-vikunja/vikunja/blob/v2.7.0/LICENSE); seine Lizenz wird durch Vimkunjas MIT-Lizenz nicht ersetzt. Eine spätere Übernahme von Upstream-Code muss gesondert geprüft werden.

Die generierten Darwin-Dateien der SQLite-Abhängigkeiten enthalten zusätzliche APSL-1.1-/2.0-Hinweise. Die besonderen Bedingungen eines **macOS-Binary-Releases sind noch nicht abschließend erfüllt**; Cross-Kompilation und die Notice-Datei allein sind keine Release-Abnahme. Die betreffenden Lizenztexte und Quellcode-Bezugsquellen sind in der Notice-Datei dokumentiert.
