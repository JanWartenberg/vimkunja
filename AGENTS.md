# AGENTS.md – Vikunja TUI

**Für:** AI Coding Agents  
**Projekt:** Terminal UI für Vikunja (OpenSource Todo/Kanban-System)

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
- **Leader-Key:** `Space` + Action (z.B. `Space+o` = neuer Task)

---

## 🚦 Workflow für Agents

### ⚠️ WICHTIG: Änderungen nur auf explizite Anweisung!

**Vikunja API erhalten:** Die API zur OriginalApp aufrechthalten, TUI muss kompatibel zur Original App bleiben
**DB-Schema nicht ändern:** bestehendes SQLite-Schema darf nicht geändert werden
**Default-Modus:** Vorschläge machen, **NICHT** umsetzen!  
**Umsetzung nur bei:** "setze um", "implementiere", "mach das", etc.

### Antwortformat

- **Kompakt:** ~1 Bildschirm (kein Scroll-Marathon)
- **Struktur:** Bullets, kurze Absätze
- **Fokus:** Was der User wissen muss, nicht alles

### Arbeitsschritte

1. Lies **SPEC_FUNDAMENT.md** (komplette Spec)
2. Prüfe **STATUS.md** (aktueller Stand, nächste Phase)
3. Check `.pi/SESSION_NOTES.md` (Kontext)
4. Implementiere gemäß Ousterhout
5. Teste Store-Logik
6. Update STATUS.md

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

- **SPEC_FUNDAMENT.md** – Komplette Feature-Spec + Architektur
- **STATUS.md** – Aktueller Fortschritt, nächste Schritte
- `.archive/` – Erledigte TODOs, Phase-Reports
- `.pi/SESSION_NOTES.md` – Agent-Kontext

---

**Letzte Änderung:** 2026-07-12
