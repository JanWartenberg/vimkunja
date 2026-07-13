# Vikunja TUI

Terminal User Interface für [Vikunja](https://vikunja.io/) – Todo/Kanban-System für Vim-Nutzer.

---

## 🚀 Quick Start

```bash
cd /workspace/vikunja-tui
go run ./cmd/vikunja-tui --db internal/db/vikunja.db
```

Die Anwendung arbeitet ausschließlich mit einer lokalen Vikunja-SQLite-Datenbank. Der angegebene Pfad muss existieren und lesbar sein.

---

## 📖 Navigation (Vim-Style)

- `hjkl` – Bewegen
- `x` – Task toggle (done/undone)
- `Enter` – Details anzeigen
- `Ctrl+W + h/l` – Pane wechseln
- `q` oder `Ctrl+C` – Anwendung beenden
- Ex-Commands wie `:q` sind geplant

---

## 📚 Dokumentation

**Für Menschen:**
- **README.md** ← Diese Datei (Quick Start, Überblick)
- **STATUS.md** – Aktueller Fortschritt, nächste Schritte
- **SPEC_FUNDAMENT.md** – Vollständige Feature-Spec & Architektur

**Für AI Agents:**
- **AGENTS.md** – Workflows, Design-Prinzipien, Konventionen

---

## 🧪 Tests

```bash
go test -v ./...
```

---

**Status:** MVP (40%) | **Lizenz:** MIT *(tbd)* | **Go:** 1.25+
