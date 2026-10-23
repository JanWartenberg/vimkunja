# Vimkunja

An unofficial Vim-style terminal interface for Vikunja.

Vimkunja is an independent project, not affiliated with or endorsed by Vikunja.

## 1. Overview

Vimkunja is a keyboard-driven terminal interface for a local [Vikunja](https://vikunja.io/) SQLite database. It is designed for Vim users and adds a fast terminal workflow for Projects, List View, Task Detail, and workflow status.

### Installation and first run

Requirements: Git, an interactive terminal, and a currently maintained Go version. The module requires Go 1.25 or later; use a current, security-patched toolchain to build it (CI uses `stable`). Linux amd64 has been checked locally; Windows amd64 builds successfully, but native Windows terminal tests are still pending.

The intended clone URL is:

```bash
git clone https://github.com/JanWartenberg/vimkunja.git vimkunja
```

```bash
cd vimkunja
```

Try it without your own database:

```bash
go run ./cmd/vimkunja-tutorial
```

Build both programs (Windows binaries have the `.exe` extension):

```bash
go build -o bin/ ./cmd/vimkunja ./cmd/vimkunja-tutorial
```

Run the built programs from the repository directory:

```bash
./bin/vimkunja-tutorial
```

```bash
./bin/vimkunja --db "/path/to/vikunja.db"
```

For example, in Windows PowerShell:

```powershell
.\bin\vimkunja.exe --db "C:\path\to\vikunja.db"
```

Alternatively, run directly from source with an existing Vikunja SQLite database for real Tasks:

```bash
go run ./cmd/vimkunja --db /path/to/vikunja.db
```

Replace `/path/to/vikunja.db` with your own path and quote it if it contains spaces. The database must be an existing, readable, writable file. Vimkunja accesses SQLite directly; there is no REST mode.

Without `--db`, Vimkunja uses the local path `internal/db/vikunja.db`. This file is not part of the repository. New installations must supply their own 2.7.0 database with `--db`. Before upgrading an older database, back it up together with any WAL file and test the upgrade on a copy using the official Vikunja 2.7.0 executable; Vimkunja does not run migrations.

### Database and compatibility

A blank SQLite file has no Vikunja tables and cannot be used for a production start. Vimkunja does not create or migrate the Vikunja schema. To start with fresh production data, Vikunja itself must initialize the database; the required User, Project, and View must exist for your workflow. A persistent starter template has not been specified. The synthetic tutorial database is only for trying the application and is deleted on normal exit.

**VOA** below means the original Vikunja application or its server. The only verified compatibility target is **Vikunja 2.7.0**. Real server round trips with a fresh database and a copy of the synthetic Windows baseline are documented in the [verification report](.archive/2026-10-05/vikunja-db-compatibility/report.md). Earlier, later, and development versions are unsupported. Production startup checks migration history and required schema features, rejecting different or incomplete database revisions before domain writes. Releases with identical database revisions cannot be distinguished from the file alone and receive no additional compatibility approval. The tutorial database is not a production starter template.

Stop VOA before starting Vimkunja; stop Vimkunja before restarting VOA. Concurrent editing has not been verified. Tasks in the trash remain hidden and cannot be edited. Archived Projects are read-only; Bucket limits are respected. Task numbers remain reserved after deletion; old addresses after Project moves and Undo/Redo can be resolved through VOA API v2.

Vimkunja still deletes Tasks **permanently**, without trash or deletion undo; attachment files remain intact. Tasks with Time Entries can only be deleted in VOA. Complete recurring Tasks in VOA as well. Creating Labels and writing/deleting Comments requires exactly one local User. Direct SQLite writes do not trigger server actions such as webhooks, notifications, or reminder services. Native Windows terminal tests and a live CalDAV test are still pending.

Back up real data before using it. Native rich-text editing is implemented; additional Views are planned. See [STATUS.md](STATUS.md) for the current feature status.

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

## 2. Controls and details

- `hjkl`, `gg` / `G`, `Ctrl+D` / `Ctrl+U` – navigation
- `Ctrl+W h/j/k/l` – change pane focus
- In the Projects Pane: `h/l` collapse/expand the hierarchy or select the parent/child, `zo/zc/za` fold, `zM/zR` collapse/expand all
- In the Projects Pane: `Shift+J/K` move a Project among its siblings
- `Enter` – open Task Detail
- `x` – toggle a Task into the Done or Default Bucket
- `u` / `Ctrl+R` – undo or redo the last persisted Task action
- `o` / `:new` – create a Task
- `:w`, `:wq`, `:wqa` – save
- `:e!` / `:edit!` – discard unsaved changes to an existing Task and keep it open
- `:q`, `:q!`, `:qa`, `:qa!` – close a pane or the application
- `:h` – built-in help
- `:errors` – session error log
- `Space+o` – show or hide the Projects Pane
- `:kanban` or `Space+v`, then `k` – open a single manual Kanban View or provision a standard one after confirmation
- In the Kanban Board: `h/l` select Buckets, `j/k` select cards, `Shift+H/L` move to a neighboring Bucket, `m` choose a destination Bucket

Title, Description, and Comment drafts support `v/V` in Normal Mode for character/line selection (logical blocks in Description), `Space+y` to copy the selection or current line/paragraph to the system clipboard, and `Space+p` to paste at the cursor or replace the selection. Copy preserves the selection; Paste stays in Normal Mode. `Esc` clears the selection. `Space+e` copies a Link's destination URL. **Descriptions use one shared editor:** `gn`, `Enter` activates it even for empty or unformatted text. Selection, formatting, and typing immediately follow the same rules; no rich-text mode switch is needed. Opening and navigation do not change stored bytes. Only an actual edit creates rich text in the draft; `:w` saves and `:e!` discards. Clipboard paste does not interpret Markdown formatting.

Text input consistently uses a thin, blinking terminal cursor: Title, Description including rich text/Code, Comments, and search, command, URL, and Label prompts. Normal Mode uses a block cursor. The date mask marks the digit to replace with an overwrite cursor; relative date input uses the thin input cursor.

The Detail Pane lets you edit Project, Title, Priority, Due Date, Labels, Description, and workflow status. `gc` opens the Comments workspace; `gca` starts a new Comment there. In the Comment thread, `a`, `i`, and `o` start a new Comment; a static block cursor marks the selected Comment. `Space+l` opens the Label picker; `:labels` manages global Labels. Status uses the configured Buckets of the Project's unique Kanban View.

HTML Descriptions are displayed structurally and edited as a Native Draft. `gn`, then `Enter` activates navigation: `j/k` moves between logical blocks (source lines in Code, child blocks in quotes), `gj/gk` between displayed lines, and `{`/`}` or `ö/ä` between whole paragraph units. `h/l`, `0/^/_/$`, `w/b/e/ge`, counts, and `gg/G` move the document cursor; `Ctrl+D/U` moves it by roughly one visible pane height. `zh/zl` scrolls Code and tables horizontally; `Ctrl+E/Y` scrolls only the viewport. `Esc` leaves active navigation. The active block displays formatting markers; Code displays protected backtick boundaries. Reading and resizing do not change stored HTML. `i/a/I/A` enters Insert, `Enter` splits paragraphs, and `Shift+Enter` inserts a Hard Break. `o/O` creates a paragraph, or a source line inside Code. `dd` deletes logical blocks or Code lines. `v/V`, `viw/vaw`, and `o` select text/blocks or swap the active end; `Esc` ends selection. In both Visual modes (`v` and `V`), `Space+b/i/u/s` applies formatting, while `Space+0` removes inline formatting including Links. `Space+x` converts each logical paragraph/list item touched by the selection to a Checkbox; existing Checkboxes retain their state and identity. Selection remains until Esc. Normal `Space+1/2/3/0` converts a paragraph/Heading. The palette sits inside the footer frame and remains readable after the Leader timeout until the next key: `SPACE ACTIVE` allows direct shortcuts, while `Space again` requires another Space. Narrow windows wrap shortcuts; at very low heights a compact palette replaces the normal footer line. `Space+o` remains global. `u/Ctrl+R` operates on the local Description draft here. Opaque deletion, replacement, and changes to its Native parent require confirmation (`y/Enter`, `n/Esc`); Undo preserves the original bytes. `y/p` copies safe visible content to the local Description register and pastes Native text.

Native inline syntax is created only by deliberate typing: `**bold**`, `*italic*`, `***both***`, `__underlined__`, `~~strikethrough~~`, and paired single backticks for inline Code. Paste remains literal text; a backslash protects the next syntax character. `Space+l` edits a Link's URL or links selected text; an empty URL removes the Link. With no selection and outside a Link, `Space+l` asks for the URL first, then the Link text. The text field starts empty; typing or Paste fills it. `Enter` on empty text uses the URL as its text. `Ctrl+U` clears the input, and `Esc` cancels both steps without changes. One `u` undoes the insertion. `Space+e` copies the destination to the system clipboard. Insertion at a Link boundary or outside visible inline Code backticks stays outside; insertion between the backticks edits the Code.

`:selectblock` explicitly selects the entire Code block; `V` in Code remains limited to its source lines. For Blocked HTML, `:descriptionrecover` and `:descriptionclear` offer confirmed, unsaved recovery actions. `:e!` still discards the entire Task draft; `:descriptionreload` reloads the current Task from the database and explicitly discards the draft. A save conflict keeps the draft open without merging. `Space+u/n/x` converts paragraphs into lists/Checkbox lists; `x` toggles a Description Checkbox independently of Task Done. `V` → `Space+k/l` outdents/indents selected items, while `Space+q/w` removes/adds a quote level. Normal `Space+q` toggles locally. Metadata and ordering are preserved; operations that cannot be lossless are rejected without changes. Create Code with Normal `Space+c` or exactly three/four directly typed backticks plus Enter in an empty paragraph; with six directly typed backticks, Enter between the third and fourth creates an empty Code block. `ü/Ü` creates a paragraph after/before Code in Insert; `]Space/[Space` does the same in Normal. Code `Tab` inserts spaces to the next tabstop; `Ctrl+V Tab` (alternatively the configurable `Ctrl+Q Tab`) inserts a real Tab. Paste preserves Tabs and CRLF. `V` → `Space+x/c` outdents/indents source lines; `:retab` converts Tabs in the current Code or selected Code lines, while `:retaball` does so in all Native text and skips Opaque content. Both take no arguments or bang and each creates one Undo step. Empty, unformatted, and HTML Descriptions use the same document editor; Title and Comment Composer share their text-selection and clipboard behavior.

Comment, Label, and Global Search command prompts also accept `:qa`, `:qa!`, and `:wqa`. `:qa` protects unsaved Task/Comment drafts; `:wqa` atomically saves an existing Comment draft with the Task and exits only after a successful save. On failure the draft stays open. In Insert Mode, press `Esc` first.

A loaded, unchanged workflow status is not reapplied when saving a Task or Comment. This preserves `Done` and the existing completion timestamp during unrelated edits. An actual Bucket change still synchronizes status as before. When `Done` is unchanged, the stored completion timestamp remains exactly intact even during Done/Bucket writes and Undo/Redo. A real transition to done sets the current timestamp; reopening clears it. Completion timestamps remain outside Undo history.

Bucket configurations of `0` or NULL are treated as unset. Without a Default Bucket, the first valid Bucket by position is used, with ID breaking ties. This applies to new Tasks, Project moves, and reopening. Without a Done Bucket, completing a Task preserves its current assignment. If no valid Bucket exists, no assignment is invented. Invalid configured Bucket references abort the write atomically.

Moving a completed Task to another Project uses the configured Done Bucket in each destination View. If none exists, the default assignment remains. A Project-only move preserves `Done` and the completion timestamp.

New Tasks receive a stable CalDAV UUID if the database has a `uid` column. Edits, Project moves, and Undo/Redo preserve this identity. Existing Tasks are not retroactively changed; a live CalDAV test is still pending.

Permanent Task deletion also removes related favorites, subscriptions, and Task/Comment reactions. Other object types and attachment files remain intact; database deletion is atomic.

Recurring Tasks can be edited and reopened in Vimkunja. Completing them is rejected: complete them in Vikunja so dates, reminders, and checklists are updated for the next recurrence.

Due dates are still edited as `YYYY-MM-DD`. As long as the date is unchanged, Task saves and Undo/Redo preserve the existing time exactly. Changing the date stores midnight UTC; clearing it stores NULL. Undo/Redo rejects external changes to the full due timestamp.

While a Task Detail save is running, keyboard input pauses and the footer shows
`Saving Task…`; `Ctrl+C` remains available for an emergency exit. Input resumes
after success or failure. Failed saves keep the draft for retry. A late
asynchronous draft change stays unsaved and prevents save-and-close or
save-and-quit from discarding it.

Tests:

```bash
go test ./...
```

## 3. Roadmap and vision

The foundation is complete: List View, editor, workflow status, error display, Vim-style pane navigation, the single manual Kanban View, and the hierarchical Projects Pane are verified. Additional Views and Bucket management remain in the backlog.

The current backlog is in `.scratch/`; [STATUS.md](STATUS.md) provides a short index with links. Completed requirements and implementation details are in the [archive](.archive/).

## License

Vimkunja's own code is licensed under the [MIT License](LICENSE). Dependencies retain their respective licenses; versioned attribution, license texts, and additional notices are in [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt). Include both files with binary distributions and meet the component-specific conditions documented there.

Vikunja is a separate [AGPL-3.0 project](https://github.com/go-vikunja/vikunja/blob/v2.7.0/LICENSE); Vimkunja's MIT license does not replace its license. Any future inclusion of upstream code requires a separate review.

The SQLite dependencies' generated Darwin files contain additional APSL 1.1/2.0 notices. The specific conditions for a **macOS binary release have not yet been fully met**; cross-compilation and the notices file alone do not constitute release acceptance. The relevant license texts and source-code locations are documented in the notices file.
