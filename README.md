# Vimkunja

## 1. Was es ist

Vimkunja ist eine tastaturorientierte Terminal-Oberfläche für eine lokale [Vikunja](https://vikunja.io/)-SQLite-Datenbank. Sie richtet sich an Vim-Nutzer und ergänzt Vikunjas Task-Verwaltung um einen schnellen Terminal-Workflow mit Projects, List View, Task-Detail und Workflow-Status.

```bash
cd /workspace/vimkunja
go run ./cmd/vimkunja --db internal/db/vikunja.db
```

Der Datenbankpfad muss existieren und lesbar sein. Vimkunja greift direkt auf die lokale SQLite-Datenbank zu; es gibt keinen REST-Modus.

### Interactive keystroke tutorial

```bash
go run ./cmd/vimkunja-tutorial
```

The separate tutorial executable runs the real Vimkunja UI and guides through every currently implemented feature group in English. Every run creates a freshly seeded SQLite database in the operating system's temporary directory. The tutorial accepts no database path, never opens `internal/db/vikunja.db` or another user database, and deletes its disposable database on exit.

## 2. Bedienung und Details

- `hjkl`, `gg` / `G`, `Ctrl+D` / `Ctrl+U` – Navigation
- `Ctrl+W h/j/k/l` – Pane-Fokus wechseln
- Im Projects Pane: `h/l` Hierarchie schließen/öffnen bzw. Parent/Child wählen, `zo/zc/za` falten, `zM/zR` alles schließen/öffnen
- Im Projects Pane: `Shift+J/K` Project unter seinen Geschwistern verschieben
- `Enter` – Task-Detail öffnen
- `x` – Task in den Done- bzw. Default-Bucket togglen
- `o` / `:new` – neuen Task anlegen
- `:w`, `:wq`, `:wqa` – speichern
- `:q`, `:q!`, `:qa`, `:qa!` – Pane bzw. Anwendung schließen
- `:h` – integrierte Hilfe
- `:errors` – Session-Fehlerlog
- `Space+o` – Projects-Pane ein- oder ausblenden
- `:kanban` oder `Space+v`, dann `k` – einzelne manuelle Kanban View öffnen oder nach Bestätigung standardkonform provisionieren
- Im Kanban Board: `h/l` Buckets, `j/k` Karten, `Shift+H/L` Nachbar-Move, `m` Ziel-Bucket wählen

Im Detail-pane können Project, Title, Priority, Due Date, Labels, Description und der Workflow-Status bearbeitet werden. `gc` öffnet den Comments-Workspace; `gca` startet dort einen neuen Comment. Im Comment-Thread starten `a`, `i` und `o` einen neuen Comment; ein statischer Blockcursor markiert den ausgewählten Comment. `Space+l` öffnet den Label-Picker; `:labels` verwaltet globale Labels. Der Status verwendet die konfigurierten Buckets der eindeutigen Kanban View des Projects.

In den Command-Prompts von Comments, Labels und Global Search gelten ebenfalls `:qa`, `:qa!` und `:wqa`. `:qa` schützt ungespeicherte Task-/Comment-Entwürfe; `:wqa` speichert einen vorhandenen Comment-Entwurf zusammen mit dem Task atomar und beendet erst nach erfolgreichem Speichern. Bei Fehlern bleibt der Entwurf offen. Im Insert Mode zuerst `Esc` drücken.

Tests:

```bash
go test ./...
```

## 3. Ausblick / Vision

Das Fundament ist abgeschlossen: List View, Editor, Workflow-Status, Fehleranzeige, Vim-artige Pane-Navigation, die einzelne manuelle Kanban View und die hierarchische Projects Pane sind verifiziert. Weitere Views sowie Bucket-Verwaltung bleiben im Backlog.

Der aktuelle, priorisierbare Backlog liegt in `.scratch/`; der Kurzstatus mit Verweisen steht in [STATUS.md](STATUS.md). Abgeschlossene Anforderungen und Implementierungsdetails liegen im [Archiv](.archive/).
