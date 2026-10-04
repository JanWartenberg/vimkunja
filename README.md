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
```

Timeouts use Go duration syntax, such as `250ms`, `1s`, or `1.5s`. Values must be positive; zero and negative values are invalid. The built-in defaults are `500ms` for `leader_timeout` and `2s` for `ctrl_w_timeout`.

The command-line equivalents override the TOML values:

```bash
go run ./cmd/vimkunja --leader-timeout 1s --ctrl-w-timeout 750ms
```

The precedence is CLI flags, then environment variables (reserved for a future release), then TOML, then built-in defaults. Configuration is read at startup; edit the file and relaunch Vimkunja to apply changes. `--help` lists the CLI flags.

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

Im Detail-pane können Project, Title, Priority, Due Date, Labels, Description und der Workflow-Status bearbeitet werden. `gc` öffnet den Comments-Workspace; `gca` startet dort einen neuen Comment. Im Comment-Thread starten `a`, `i` und `o` einen neuen Comment; ein statischer Blockcursor markiert den ausgewählten Comment. `Space+l` öffnet den Label-Picker; `:labels` verwaltet globale Labels. Der Status verwendet die konfigurierten Buckets der eindeutigen Kanban View des Projects.

In den Command-Prompts von Comments, Labels und Global Search gelten ebenfalls `:qa`, `:qa!` und `:wqa`. `:qa` schützt ungespeicherte Task-/Comment-Entwürfe; `:wqa` speichert einen vorhandenen Comment-Entwurf zusammen mit dem Task atomar und beendet erst nach erfolgreichem Speichern. Bei Fehlern bleibt der Entwurf offen. Im Insert Mode zuerst `Esc` drücken.

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
