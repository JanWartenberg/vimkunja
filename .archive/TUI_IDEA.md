# Vikunja TUI (MVP) — Vim motions and pane layout

## Goal
A terminal interface for Vikunja that feels **like vi/vim**:
- Navigate primarily with **Vim motions** (`j/k`, `gg/G`, `enter`, …)
- **Multiple panes** like Vim splits
- MVP initially implements **one Task view only**: `ProjectViewKind = list`

## Data/feature scope (MVP)
- **Left:** Select Projects
- **Right:** Show Tasks for the selected Project View
- View selection: Automatically use the **first** View with `view_kind == "list"` for a Project.
- Task CRUD/Comments/editing come later — MVP is “read + navigation,” with optional Refresh.

## Pane concept (Vim splits)
There are three panes:
- `projects` (top left)
- `tasks` (top right)
- `detail` (bottom, closable)

**Pane lifecycle (Decision #4):**
- Detail is a full pane in the focus graph, not a modal.
- `Enter` in Tasks opens Detail; `Ctrl+W` navigates between all three panes.
- Detail can be closed (keybinding below).
- Closing Detail returns focus to Tasks.

### Focus commands
As in Vim:
- `Ctrl+W` (as Leader) + `Ctrl+W` → toggle/cycle focus
- `Ctrl+W` + `h/j/k/l` → focus the neighboring pane in that direction
- `Ctrl+W k` from Detail → return to Tasks
- `Ctrl+W j` from Tasks/Projects → Detail (if open)

### Example flow
- `Ctrl+W h` → focus Projects
- `j/k` → change Project
- `Ctrl+W l` → focus Tasks
- `j/k` → scroll Tasks
- `r` → Refresh (reload Projects and Tasks)

## Extensibility (design)
Although MVP renders only the `list` View, structure it for later additions:
- More `ProjectViewKind` handlers (`kanban`, `table`, …)
- Task Detail Pane (e.g. “Enter opens Detail below the current pane layout”)
- Compose multiple panes into a pane grid (neighbors by coordinates or graph)

## Technical decision
- **Language:** Go
- **UI framework:** `bubbletea` + `lipgloss`
- **Backend:** Direct SQLite access (no REST API)

## Architecture decisions (review session 2026-07-09)

### 1. Store abstraction: SQLite only, no REST
**Decision:** Remove REST client (`client.go`). The application accesses only the `vikunja.db` file directly. Reasons: single-user setup, file access always available, no asymmetry (“not implemented” in REST mode), less complexity.

### 2. Write semantics: follow server logic (B)
**Decision:** On writes (e.g. `ToggleTaskDone`), replicate the main Vikunja server side effects:
- Update `updated` timestamp on every write
- Move completed Tasks to `done_bucket_id` (if configured in the View)
- Later editing: maintain all relevant fields and `updated`

**Rationale:** Core Law #1 (schema compatibility) — leave the database in a state the real server could produce. `updated` matters to server features (webhooks, sync, “last modified”). Bucket moves prevent Kanban drift.

### 3. Concurrency: fire-and-forget and auto-refresh (D)
**Decision:** For writes:
- No optimistic locking (no `updated` check before write)
- SQLite handles locking (WAL mode enabled)
- Automatically reload the current View after every successful write
- On write error: visual feedback (UI flash or red footer)

**Rationale:** Single-user setup makes write/write races unlikely. Auto-refresh prevents stale views. Error feedback immediately exposes database locks, Tasks deleted by the server, etc. Pragmatic, simple, and upgradeable to optimistic locking later if needed.

### 4. Detail Pane: equal pane (B)
**Decision:** Detail is a full third pane in the focus graph, not a modal/overlay. `Enter` in Tasks opens it; `Ctrl+W k` returns; it can be closed.

**Rationale:** Supports future editing. Once Detail is editable, users need to switch between Detail editing and Task selection. A modal would block this. Consistent with Vim splits.

### 5. Edit-mode commit policy (revised by Decision #19)
**Superseded:** Initially planned to discard changes (“simplest option”). After edit-mode review (Decision #19), changed to **auto-commit**. See Decision #19.

### 6. Command line: part of the foundation (A)
**Decision:** Build the command line (`:` opens footer input mode) *now*, as part of the foundation.

**Rationale:** A Vim-motion tool needs Ex commands as its canonical interface. Twenty-six letters, ten digits, and Ctrl Leaders are insufficient for a substantial Kanban/backlog board. Commands provide flexibility. `:q` / `:qa` are iconic Vim behavior — including them from the start avoids later breaking changes.

**Minimum foundation scope:**
- `:q` — close pane (or application if it is the last pane)
- `:qa` / `:quitall` — quit application (close all panes)
- `:h` / `:help` — show help (static keybindings/commands)
- `:errors` — show error log (foundation placeholder: “Not implemented”)

**Extensibility:** Command dispatcher must be extensible (plugin pattern or command registry). More commands (`:goto`, `/` search, etc.) come in later features.

**Autocomplete:** Later iterations, when dynamic commands (`:goto <project>`) matter.

**Search/filter:** Use **`/`** as in Neovim, not a command such as `:filter`. `/<query>` searches everything (Projects, Tasks, Labels). Separate focused-search commands (“Tasks only,” “Projects only”) are possible, but design remains open → later feature review.

### 7. Keybindings versus commands: strategy
**Decision:** Avoid duplication. Give each action either a keybinding OR a command, except where Vim historically provides both.

**Principles:**
- Distinguish **Normal Mode and Insert Mode**, as in Vim. Navigation/toggle/domain actions use Normal Mode keybindings. Edit fields use Insert Mode for character input.
- **Prefer keybindings** for frequent actions without arguments (navigation, toggle, refresh). Use 26 letters, ten digits, and Leader combinations (e.g. `Space+X`).
- **Commands** for rare/meta actions or actions with arguments (`:qa`, `:h`, later `:goto <name>`).
- **Vim consistency** matters more than rigid rules. Where Vim duplicates (`?` and `:help`), we may too.
- **Ergonomics:** German keyboard layout, few keystrokes, memorable, unambiguous.

**Concrete foundation assignments:**
- Navigation (`hjkl`, `gg/G`, `Ctrl+D/U`, `öä` as aliases for `{}`?) → **keybindings only**
- Refresh → **`r` keybinding** (frequent domain action) — **no** `:r` command
- Toggle Done → **`x` keybinding** (frequent) — no `:toggle` command yet
- Help → **`:h`/`:help` commands** + **`?` keybinding** (standard Vim duplication)
- Quit → **`:q`/`:qa` commands only** (no `q` keybinding). `Ctrl+C` remains emergency exit (quit immediately).

**Rationale (Decision #8):** Vim-consistent — Vim `q` records macros rather than quitting. Reserve `q` for future features. `:q` closes a pane (or application if last pane). `:qa` quits immediately. `Ctrl+C` is the universal TUI panic button. Three keystrokes for `:q` are acceptable because pane closing is infrequent.

### 8. Quit keybinding: command only (D)
**Decision:** No `q` keybinding. Quit only through `:q`/`:qa`. Keep `Ctrl+C` for emergencies.

**Rationale:** See Decision #7 concrete assignments above.

### 9. Error handling: hybrid (red footer, error log, retry awareness)
**Decision:** Three levels:

1. **Error log (`:errors`):**
   - Store every error in a session log.
   - `:errors` opens a pane (or footer list) with all session errors.
   - Errors are never lost; users can consult them later.
   - **Implementation:** Initially a placeholder (`:errors` shows “Not implemented”); log infrastructure later.

2. **Immediate feedback by outcome:**
   - **Transient errors resolved by retry (e.g. database lock):** Bottom-right footer shows green “write success” for 2–3 seconds, then disappears.
   - **Unresolved errors:** Red footer with error text. Remains red until any user action, including `j/k` cursor movement.

3. **Auto-refresh after successful write:**
   - Reload current View after each successful write, including retry success (Decision #3).

**Rationale:** Balance immediate footer feedback with an inspectable log. Transient errors auto-dismiss; real errors remain visible until user action indicates they were seen. Log supports later debugging.

### 10. Help content: Help Pane (B)
**Decision:** `:h` / `:help` opens a Help Pane (like Detail, closable with `:q` or `Esc`).

**Content structure:**
- **First 10–20 lines:** Most frequent commands (`hjkl/gg/G`, `:q`, `x`, `Enter`, `Ctrl+W`, etc.) visible without scrolling (90% of uses).
- **Scrollable remainder:** Complete keybinding list, commands, and explanations.

**Rationale:** Static footer text is too short. A scrollable Help Pane shows everything, with common commands immediately visible for a quick start.

### 11. Schema validation: no (B)
**Decision:** No startup schema check. Trust that the file is a valid Vikunja database.

**Rationale:** SQL errors from the wrong database are clear enough and enter the error log. Schema checks add work/maintenance when Vikunja changes. Add later if practical problems arise.

## Feature decisions (Kanban View review)

### 12. Kanban in the pane graph: Tasks Pane renders by view_kind (A)
**Decision:** Tasks Pane (right) displays List or Kanban according to the selected Project's `view_kind`. Pane graph unchanged (Projects ↔ Tasks ↔ Detail).

**Rationale:** Consistent with current layout. User selects Project on the left; right displays its View, whether List or Kanban. Different rendering, no separate Kanban mode.

### 13. Projects Pane: collapsible with Leader+o (idea C)
**Decision:** Toggle the left Projects Pane with:
- **`Leader+o`** (Space, then `o` within 500ms)
- **`Ctrl+W o`** as a synonym (Vim standard for maximize/only)

**Rationale:** Kanban needs width for adjacent Buckets. Collapsing Projects maximizes space without changing the pane graph. `Leader+o` is familiar from Neovim's Aerial plugin; `Ctrl+W o` follows Vim.

**Leader configuration:**
- Leader = **Space**
- Timeout = **500ms** (default, configurable)
- Sequences: `Leader+o` (collapse), possibly others later (`Leader+f` for filters, etc.)

**Long term:** Projects Pane could be replaced with other navigation panes (grouped by tags, Labels, Due Date, etc.); collapse behavior is reusable.

### 14. Kanban Buckets: dynamic columns and horizontal scrolling (A, adaptive)
**Decision:** Render Buckets side by side. Visible count depends on screen width.

**Layout:**
- **Minimum Bucket width:** 25 columns (readable Task titles)
- **Visible Buckets:** `min(totalBuckets, floor(availableWidth / 25))`
  - Example: 80-column screen, Projects collapsed → three Buckets (3×25=75, remainder for borders)
  - Example: 120-column screen → four or five Buckets
- **Horizontal scrolling:** If `totalBuckets > visibleBuckets`, scroll with `h/l` (Bucket focus shifts).
- **Indicators:** Header arrows `<` (left) and `>` (right) when further Buckets exist in that direction.

**Typical Bucket count:** 3–6 (Todo, Doing, Done, possibly Review, Backlog, Blocked). Three fit without scrolling in a standard 80–100-column terminal.

**Rationale:** A board needs multiple visible Buckets. Fixed minimum width keeps Tasks readable; adaptive layout uses available width and scrolls only when needed.

### 15. Kanban navigation: 2D grid with hjkl (A)
**Decision:** Focus model = **one** Bucket and **one** Task inside it.
- `h/l` — change Bucket horizontally, retaining Task position where possible, otherwise using first/last Task
- `j/k` — select a Task vertically within focused Bucket
- Exactly **one** Task highlighted across all Buckets.

**Edge cases:**
- `h` in first or `l` in last Bucket → **no-op**; focus stays. Brief gentle Bucket flash (e.g. 200ms border highlight).
- `j` in empty Bucket or `k` at first Task → no-op (no wrap to another Bucket).

**Rationale:** Intuitive, fast, no `Ctrl+W` or modes. A 2D grid maps to `hjkl`. Treating Buckets as panes is cumbersome (three keystrokes). No confusing wrap-around or mode switches.

**Kanban-global keybindings:** Define meta actions such as Bucket renaming/reordering later. Focus here is Task navigation and movement.

### 16. Move Tasks between Buckets: Shift+H/L and m+digit (C+A)
**Decision:** Two mechanisms:

1. **Shift+H / Shift+L — neighboring Bucket**
   - `Shift+H` → move Task one Bucket left
   - `Shift+L` → move Task one Bucket right
   - **Focus follows the moved Task** into its new Bucket
   - **Commit immediately**, no confirmation
   - **Edge case:** `Shift+H` in first or `Shift+L` in last Bucket → no-op; brief subtle Task flash (200ms).

2. **m + digit — any Bucket**
   - User presses `m` (move)
   - Footer shows “Move to: 1=Todo 2=Doing 3=Done 4=Review” (Bucket names/numbers)
   - User presses digit (1–9, possibly 0 for Bucket 10)
   - Task moves; focus follows into new Bucket
   - `Esc` cancels without moving

**Database update:**
- `task_buckets`: `DELETE` old entry (task_id, old project_view_id, old bucket_id), `INSERT` new entry (task_id, project_view_id, new bucket_id)
- Update `tasks.updated` (Decision #2)
- Auto-refresh after success (Decision #3)

**Rationale:** Most moves advance one workflow step (Todo→Doing→Done); `Shift+H/L` provides one intuitive keystroke. Jumps/many Buckets use `m`+digit (two keystrokes, always works). Together they provide speed/flexibility.

**Vim conflict:** Vim `m` sets a mark; less relevant for Tasks, so acceptable. No better semantically fitting Vim motion found.

### 17. Toggle Done (x) in Kanban: automatic move and feedback (D)
**Decision:** Kanban `x` works like List `x` (Decision #2), with visual confirmation.

**Behavior:**
1. User presses `x` on a Task in Doing or another non-Done Bucket.
2. Set `done=1`.
3. **Automatically move to `done_bucket_id`** if the View has one.
4. **Focus follows the Task**, now in Done.
5. **Visual feedback:**
   - Footer for two seconds: “Task marked done & moved to 'Done'”
   - Brief Task flash in Done (200ms, if visible)
6. Update `tasks.updated` and auto-refresh (Decision #3).

**Rationale:** Consistent with List and Decision #2 (done→done_bucket), respecting Vikunja semantics. Following the Task maintains orientation; feedback prevents “where did my Task go?” confusion.

**Rejected alternative:** Allow `x` only in List, disable it in Kanban. Inconsistent and cumbersome because users would need to move manually to Done.

**TODO (later):** Undo last Kanban move/toggle (`u` like Vim), useful for accidental `x`.

---

## Edit-mode review (priority 2)

### 18. Edit mode: hybrid — inline Title, Detail Pane for everything else (C)
**Decision:** Two editing modes by context:

**1. Inline Title (List/Kanban Pane):**
- Focus Task in List or Kanban
- **`i` (insert)** → inline Task Title Insert Mode
- Text field appears in the list (Task row becomes editing widget)
- **`Esc`** → Normal Mode, discard Title edit (Decision #5)
- **`Enter`** or **focus loss** → commit Title to database

**2. Multiple fields in Detail Pane:**
- **`Enter`** on List/Kanban Task → open Detail as before
- `Ctrl+W j` or corresponding pane switch → focus Detail
- In Detail:
  - **Normal Mode (default):** `hjkl` navigates Title, Description, Priority, Due, Done, etc.
  - **`i` on focused field** → Insert Mode for that field
  - **`Esc` in Insert** → Normal field navigation, discard changes (Decision #5)
  - **Commit:** Field change (`jk` in Normal) or pane switch (`Ctrl+W`) saves changed fields? (TODO: clarify commit logic)

**Field-specific handling (TODO priority 2):**
- **Due Date:** Date picker or ISO input (`YYYY-MM-DD`)?
- **Priority:** Numeric input (0–5) or `+`/`-` increments?
- **Done:** Existing `x` toggle or editable Checkbox field?
- **Description:** Multiline text field (`j/k` scrolls, `i` edits)

**Rationale:** Title changes are most frequent; inline is fast (`i`). Complex/multifield Description edits need Detail's space. Vim separation: Normal navigates Tasks/fields, Insert edits.

### 19. Detail editing: auto-commit on pane switch (C, revises #5)
**Decision:** Automatically commit changes when users leave the context.

**Triggers:**
- **Pane switch** (`Ctrl+W` out of Detail) → commit all pending changes in one database write
- **Item switch** (return to List/Kanban, choose another Task with `Enter`) → commit
- **Field switch** (`Esc` from Insert, then `jk`) → changes remain pending until pane switch

**Inline Title separately:**
- **`Enter`** in Title Insert Mode → immediate database commit
- **Focus loss** (navigate away, e.g. `jk` to another Task) → immediate commit

**Database commit:**
- All changed fields in one `UPDATE tasks SET ... WHERE id=?`
- Update `updated` (Decision #2)
- Auto-refresh after success (Decision #3)
- Failure → red footer and error log (Decision #9)

**Rationale:** Auto-commit is modern (like Jira/Notion); users never lose work. Simpler than explicit `:w`/`:q!` (“when discard/notify?”). Revises #5 discard policy: same implementation effort, better UX.

**No explicit discard:** Reverting changes is a later Undo/`u` feature. Foundation uses auto-commit without an escape hatch.

**Open later editing decisions:**
- Keybindings for frequent attributes (Priority, date, status)?
- `i` for field Insert Mode? (natural choice)
- Leader: `Space` or another key?
- `öä` as block-scroll aliases on German keyboards? (test)

## Rendering (MVP)
- Left column:
  - Heading: `Projects`
  - List: `> <identifier or title>`
- Right column:
  - Heading: `Tasks (list)`
  - List: `[%] <priority> <title> (due)`
  - Done: Checkbox/marker

## Status

**Current (2026-07-09):**
- Proof of concept runs: database contents visible, initial structure/navigation working, `x` toggles Done.
- **Review session complete:** Foundation settled (#1–11), Kanban (priority 1, #12–17), and edit mode (priority 2, #18–19) designed.

**Next goal:** Start implementation
1. **Clean up foundation:**
   - Remove REST client (`client.go`)
   - Build command-line infrastructure (`:q`, `:qa`, `:h`, placeholder `:errors`)
   - Error handling/log (red footer, auto-refresh, retry awareness)
   - Help Pane (`?` / `:h`)
   - Leader support (`Space`, 500ms timeout)
   - Collapsible Projects (`Leader+o` / `Ctrl+W o`)

2. **Priority 1: Kanban View**
   - Tasks Pane renders `view_kind=kanban` (horizontal Buckets, adaptive width)
   - `hjkl` 2D navigation
   - Move using `Shift+H/L` and `m+digit`
   - Toggle Done (`x`) with automatic move/feedback

3. **Priority 2: edit mode**
   - Inline Title with `i`
   - Multifield Detail editing (`hjkl` field navigation, `i` Insert)
   - Auto-commit on pane switch

**Open TODOs (later increments):**
- Priority 3: Overview (`-`, all Tasks, filters/sorting)
- Priority 4: search/filter (`/`)
- Priority 5: Comments
- Undo (`u`)
- Create Tasks (`o`)
- Field-specific date picker/Priority input

## Roadmap / backlog

### Foundation (settled in this review)
- [x] Store: SQLite only, no REST
- [x] Write semantics: faithful side effects (`updated`, Bucket move)
- [x] Concurrency: fire-and-forget and auto-refresh
- [x] Detail: equal third pane in graph
- [x] Command line: `:q`, `:qa`, `:h` in foundation
- [x] Command scope: `:q`, `:qa`, `:h`, `:r` with aliases
- [x] Keybindings versus commands: strategy settled (#7)
- [x] Error handling strategy settled (#9)
- [x] Help content: Help Pane (#10)
- [x] Schema validation: no, trust the database (#11)

### Features (priorities settled)

**Priority 1: Kanban View** (`view_kind=kanban`)
- [x] Pane graph: Tasks renders Kanban by `view_kind` (#12)
- [x] Projects collapsible with `Leader+o` / `Ctrl+W o` (#13)
- [x] Buckets: dynamic columns, horizontal scroll (#14)
- [x] Navigation: `h/l` Buckets, `j/k` Tasks within Bucket (#15)
- [x] Task movement: Shift+H/L neighbors, m+digit jumps (#16)
- [x] Kanban Done toggle (`x`): automatic move and feedback (#17)
- [ ] Move Tasks between Buckets (`m` + destination selection?)
- [ ] Horizontal scrolling with many Buckets?
- [ ] Detail in Kanban? (unchanged: Enter opens Detail below)

**Further Kanban TODOs (implementation details, priority 2):**
- [ ] **Order within Bucket:** `task_positions.position`, like List; vertical order persisted by Vikunja.
  - **Later:** Vertical reordering (`Ctrl+J/K` within Bucket? Vikunja supports it; consistent).
- [ ] **Empty Buckets:** Header name, empty below; no “(no tasks)” text.
- [ ] **Bucket header:** Name only, no Task count. Centered or left-aligned?
- [ ] **Horizontal scroll indicators:** Header `<` and `>` when more Buckets exist.

**Priority 2: editing Task Detail**
- [x] Location: hybrid inline Title, rest in Detail (#18)
- [x] Inline Title commit on Enter/focus loss (#18)
- [x] Detail field selection `hjkl`, Insert `i` (#18)
- [x] Auto-commit on pane/item switch (#19, revises #5)
- [ ] Due Date picker (field-specific, priority 2)
- [ ] Priority numeric input or `+`/`-`? (field-specific, priority 2)
- [ ] New Tasks with `o` (Vim “open line below”)? (later)

**Priority 3: Overview** (“All Tasks”)
- [ ] Flat list of ALL Tasks from all Projects
- [ ] Access with `-` (like Vim `:Explore`)
- [ ] Hide completed Tasks (toggle binding?)
- [ ] Sort by Due Date, Priority, Project, etc. (quick-switch bindings?)
- [ ] Separate pane or replace Projects+Tasks?

**Priority 4: search/filter** (`/` like Neovim)
- [ ] `/` opens footer search input
- [ ] Client-side current-page filter or server-side?
- [ ] Highlight matches in Projects/Tasks/Labels
- [ ] `n` / `N` next/previous match, like Vim

**Priority 5: post Comments**
- [ ] Detail displays Comment list
- [ ] `a` (append) opens Insert for new Comment
- [ ] Store method: `CreateComment(ctx, taskID, text)`
- [ ] Comment editing/deletion later

**Later:**
- [ ] Table View (`view_kind=table`)
- [ ] Gantt View (`view_kind=gantt`)
- [ ] More Ex commands (`:goto <project>`)
- [ ] Autocomplete for commands with dynamic arguments
