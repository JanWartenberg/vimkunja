# Vimkunja

## 1. Was es ist

Vimkunja ist eine tastaturorientierte Terminal-Oberfläche für eine lokale [Vikunja](https://vikunja.io/)-SQLite-Datenbank. Sie richtet sich an Vim-Nutzer und ergänzt Vikunjas Task-Verwaltung um einen schnellen Terminal-Workflow mit Projects, List View, Task-Detail und Workflow-Status.

### Installation und erster Start

Voraussetzungen: Git, Go 1.25 oder neuer und ein interaktives Terminal. Linux amd64 wurde lokal geprüft; Windows amd64 lässt sich bauen, native Windows-Terminaltests stehen noch aus.

Bis die GitHub-Adresse feststeht, steht `<repository-url>` für die tatsächliche Clone-URL:

```bash
git clone <repository-url> vimkunja
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

Für echte Tasks eine vorhandene Vikunja-SQLite-Datenbank angeben:

```bash
go run ./cmd/vimkunja --db /path/to/vikunja.db
```

`/path/to/vikunja.db` durch den eigenen Pfad ersetzen und bei Leerzeichen in Anführungszeichen setzen. Der Datenbankpfad muss eine vorhandene, lesbare und beschreibbare Datei sein. Vimkunja greift direkt auf SQLite zu; es gibt keinen REST-Modus.

### Datenbank und Kompatibilität

Eine blanke SQLite-Datei enthält keine Vikunja-Tabellen und reicht für einen produktiven Start nicht aus. Vimkunja erstellt oder migriert kein Vikunja-Schema. Für einen frischen produktiven Datenbestand muss Vikunja selbst die Datenbank initialisieren; User, Project und View müssen für den gewünschten Workflow vorhanden sein. Ein dauerhaftes Start-Template ist noch nicht spezifiziert. Die synthetische Tutorial-Datenbank ist ausschließlich zum Ausprobieren gedacht und wird beim Beenden gelöscht.

Noch ist keine konkrete Vikunja-Release-Version durch einen vollständigen Integrationstest mit der Original-App freigegeben. Store-Tests prüfen synthetische Schemas; sie ersetzen keine Versionszusage. Das Erstellen von Labels sowie das Schreiben/Löschen von Comments setzt genau einen lokalen User voraus. Direkte SQLite-Schreibzugriffe führen keine Vikunja-Serveraktionen wie Webhooks, Benachrichtigungen oder Reminder-Dienste aus. Gleichzeitige Bearbeitung mit dem Server ist nicht als sicherer Workflow verifiziert.

Vor dem Arbeiten mit echten Daten ein Backup erstellen. Weitere Views und native Richtext-Bearbeitung sind geplant; der aktuelle Funktionsstand steht in [STATUS.md](STATUS.md).

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

Ein geladener, unveränderter Workflow-Status wird beim Task- oder Comment-Save nicht erneut angewendet. Dadurch bleiben `Done` und der vorhandene Abschlusszeitstempel bei anderen Änderungen erhalten. Ein tatsächlicher Bucket-Wechsel synchronisiert den Status weiterhin wie bisher.

Bucket-Konfigurationen mit `0` oder NULL gelten als nicht gesetzt. Ohne Default-Bucket wird der erste gültige Bucket nach Position verwendet; bei gleicher Position entscheidet die ID. Das gilt für neue Tasks, Project-Wechsel und das Wiederöffnen. Ohne Done-Bucket bleibt beim Abschließen die aktuelle Zuordnung erhalten. Gibt es keinen gültigen Bucket, wird keine Zuordnung erfunden. Ungültige konfigurierte Bucket-Verweise brechen den Schreibvorgang atomar ab.

Beim Verschieben eines erledigten Tasks in ein anderes Project wird je Ziel-View der konfigurierte Done-Bucket verwendet. Fehlt dieser, bleibt die Default-Zuordnung. Ein reiner Project-Wechsel erhält `Done` und den Abschlusszeitstempel.

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

Vimkunja steht unter der [MIT-Lizenz](LICENSE). Abhängigkeiten behalten ihre jeweiligen Lizenzen.
