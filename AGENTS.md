# AGENTS.md – Vimkunja

**Für:** AI Coding Agents  
**Projekt:** Vimkunja – Terminal UI für Vikunja (OpenSource Todo/Kanban-System)

---

## 🎯 Vision

Es soll ein TUI für eine Todo-Liste (mit Projekten, Tickets, Kanban-Board) werden.
Im Wesentlichen soll das OpenSource-Projekt Vikunja mit seiner bestehenden Funktionalität in ein TUI überführt werden.

**Zielgruppe:** Vim-Nutzer, Tastatur-affin, Kommandozeilen-Workflow

---

## ⌨️ UX-Guidelines

- **Nur Tastatur:** Wer Vim kennt, soll nicht überrascht werden
- **Vim-Navigation:** `hjkl`, `gg/G`, `Ctrl+D/U`, `Ctrl+W` (Panes)
- **Commands:** `:q`, `:qa`, `:h` (analog Vim)
- **Leader-Key:** `Space` + Action

---

## 🚦 Workflow für Agents

### ⚠️ WICHTIG: Änderungen nur auf explizite Anweisung!

**Vikunja API erhalten:** Die API zur OriginalApp aufrechthalten, TUI muss kompatibel zur Original App bleiben
**DB-Schema nicht ändern:** bestehendes SQLite-Schema darf nicht geändert werden
**Windows-Host:** `/workspace` wird auf Windows gehostet. Zeilenenden vor Commits auf LF normalisieren; reine CRLF/LF-Umstellungen nicht als fachliche Änderungen committen.
**Markdown-Hygiene:** Keine Markdown-Dateien inflationär anlegen. `README.md` enthält Einführung, Bedienung und Vision; `STATUS.md` ist der kurze Ticket-Index. Offene Arbeit gehört ausschließlich in `.scratch/<feature>/`; abgeschlossene Specs und Tickets nach `.archive/YYYY-MM-DD/`. Temporäre Notizen nach Übernahme löschen.
**Default-Modus:** Vorschläge machen, **NICHT** umsetzen!  
**Umsetzung nur bei:** "setze um", "implementiere", "mach das", etc.

### Antwortformat

- **Kompakt:** ~1 Bildschirm (kein Scroll-Marathon)
- **Struktur:** Bullets, kurze Absätze
- **Fokus:** Was der User wissen muss, nicht alles

### Arbeitsschritte

1. Lies **README.md** und **STATUS.md**
2. Lies das angeforderte oder aktuelle `.scratch`-Ticket samt zugehöriger Spec
3. Implementiere gemäß Ousterhout
4. Teste Store-Logik
5. Aktualisiere Ticket und STATUS-Index, falls sich die Priorität ändert

---

## 🧭 Design-Philosophie (Ousterhout)

- **Deep Modules:** Einfache Interfaces, komplexe Implementation (z.B. `store`)
- **Information Hiding:** Verstecke HTTP/JSON/State-Details
- **Pull Complexity Downward:** UI bleibt einfach, Logik im Store
- **Define Errors Out of Existence:** Validierung beim Schreiben, nicht beim Lesen
- **General-Purpose:** Wiederverwendbare Komponenten (kein Copy-Paste)
- **Consistency:** Einheitliche Patterns (Keybindings, Datenstrukturen, Error-Handling)

---

## 📚 Referenzen

- **README.md** – Einführung, Bedienung und Vision
- **STATUS.md** – Aktueller Ticket-Index
- `.scratch/` – offene Specs und Tickets
- `.archive/` – abgeschlossene Specs, Tickets und historische Berichte

---

## Agent skills

### Issue tracker

Issues are tracked as local Markdown under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default local triage roles are configured. See `docs/agents/triage-labels.md`.

### Domain docs

This is a single-context repository. See `docs/agents/domain.md`.

---

**Letzte Änderung:** 2026-07-12
