# Vimkunja

## 1. Was es ist

Vimkunja ist eine tastaturorientierte Terminal-Oberfläche für eine lokale [Vikunja](https://vikunja.io/)-SQLite-Datenbank. Sie richtet sich an Vim-Nutzer und ergänzt Vikunjas Task-Verwaltung um einen schnellen Terminal-Workflow mit Projects, List View, Task-Detail und Workflow-Status.

```bash
cd /workspace/vimkunja
go run ./cmd/vimkunja --db internal/db/vikunja.db
```

Der Datenbankpfad muss existieren und lesbar sein. Vimkunja greift direkt auf die lokale SQLite-Datenbank zu; es gibt keinen REST-Modus.

## 2. Bedienung und Details

- `hjkl`, `gg` / `G`, `Ctrl+D` / `Ctrl+U` – Navigation
- `Ctrl+W h/j/k/l` – Pane-Fokus wechseln
- `Enter` – Task-Detail öffnen
- `x` – Task in den Done- bzw. Default-Bucket togglen
- `o` / `:new` – neuen Task anlegen
- `:w`, `:wq`, `:wqa` – speichern
- `:q`, `:q!`, `:qa`, `:qa!` – Pane bzw. Anwendung schließen
- `:h` – integrierte Hilfe
- `:errors` – Session-Fehlerlog
- `Space+o` – Projects-Pane ein- oder ausblenden

Im Detail-pane können Project, Title, Priority, Due Date, Description und der Workflow-Status bearbeitet werden. Der Status verwendet die konfigurierten Buckets der eindeutigen Kanban View des Projects.

Tests:

```bash
go test ./...
```

## 3. Ausblick / Vision

Das Fundament ist abgeschlossen: List View, Editor, Workflow-Status, Fehleranzeige und Vim-artige Pane-Navigation sind verifiziert. Als Nächstes werden UX-Verbesserungen und später eine Kanban View spezifiziert.

Der aktuelle, priorisierbare Backlog liegt in `.scratch/`; der Kurzstatus mit Verweisen steht in [STATUS.md](STATUS.md). Abgeschlossene Anforderungen und Implementierungsdetails liegen im [Archiv](.archive/).
