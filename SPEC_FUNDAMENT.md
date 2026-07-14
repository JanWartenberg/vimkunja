# Spec: Vimkunja — Fundament (MVP Core)

**Status:** Ready for Implementation  
**Related:** [`.archive/TUI_IDEA.md`](.archive/TUI_IDEA.md) (historische Decisions #1-11)  
**Priority:** P0 (Blocks all features)

---

## Problem Statement

Users need a stable, Vim-style foundation for Vimkunja that handles:
- Data persistence (SQLite-only, no REST API complexity)
- Error visibility (writes can fail, DB locks can occur)
- Navigation patterns (Vim-splits, command-line, pane focus)
- Discoverability (help system, clear error messages)
- Extensibility (command infrastructure for future features)

The current PoC has:
- A REST client (`client.go`) that won't be used (Decision #1)
- No command-line interface (`:q`, `:h` missing)
- No error log (errors vanish after 2 seconds)
- No help system (users must memorize keybindings)
- Incomplete write semantics (`updated` timestamp not bumped, no bucket moves)
- No leader-key support (Decision #13 needs it for collapsable panes)

---

## Solution

Build a **production-ready foundation** that:
1. **Removes REST client** — SQLite-only data access (Decision #1)
2. **Adds command-line infrastructure** — `:q`, `:qa`, `:h`, `:errors` as Ex-commands (Decision #6)
3. **Implements robust error handling** — Red footer + persistent error log + auto-refresh (Decision #9)
4. **Provides discoverable help** — Help-pane via `:h` or `?` (Decision #10)
5. **Supports leader-key sequences** — `Space` + 500ms timeout for future features (Decision #13)
6. **Ensures write-correctness** — Bump `updated` timestamp, move done tasks to `done_bucket_id` (Decision #2)
7. **Keeps UI fresh** — Auto-refresh after every successful write (Decision #3)
8. **Solidifies pane model** — Detail-pane as full pane (not modal), proper focus graph (Decision #4)

---

## User Stories

1. As a Vimkunja user, I want to see all my projects in the left pane, so that I can choose which project to work on.
2. As a Vimkunja user, I want to navigate between projects using `j/k`, so that I can quickly browse my project list.
3. As a Vimkunja user, I want to jump to the top of the project list with `gg`, so that I can get back to the first project fast.
4. As a Vimkunja user, I want to jump to the bottom of the project list with `G`, so that I can see my last projects.
5. As a Vimkunja user, I want to scroll half-pages with `Ctrl+D` and `Ctrl+U`, so that I can navigate large lists efficiently.
6. As a Vimkunja user, I want to press `Enter` on a project to see its tasks, so that I can focus on work items.
7. As a Vimkunja user, I want to switch focus between panes with `Ctrl+W`, so that I can navigate like in Vim.
8. As a Vimkunja user, I want to press `Ctrl+W h/l/j/k` to move focus directionally, so that I can jump to specific panes.
9. As a Vimkunja user, I want to see tasks in the right pane when I select a project, so that I can view my work items.
10. As a Vimkunja user, I want to see task priority colored (green=low, red=urgent), so that I can identify important tasks at a glance.
11. As a Vimkunja user, I want to see done tasks grayed out, so that I can distinguish completed work.
12. As a Vimkunja user, I want to press `x` on a task to toggle its done status, so that I can mark tasks complete.
13. As a Vimkunja user, I want done tasks to automatically move to the "Done" bucket in Kanban views, so that my board stays organized.
14. As a Vimkunja user, I want the `updated` timestamp bumped when I mark a task done, so that Vikunja's server features (webhooks, sync) work correctly.
15. As a Vimkunja user, I want the task list to auto-refresh after I toggle done, so that I always see the current state.
16. As a Vimkunja user, I want to press `r` to manually refresh the current view, so that I can reload data if the DB changed externally.
17. As a Vimkunja user, I want to type `:q` to close the current pane, so that I can exit like in Vim.
18. As a Vimkunja user, I want to type `:qa` to quit the entire app, so that I can exit all panes at once.
19. As a Vimkunja user, I want to type `:h` or press `?` to see help, so that I can learn keybindings without leaving the app.
20. As a Vimkunja user, I want to type `:errors` to see a log of all errors in this session, so that I can debug issues.
21. As a Vimkunja user, I want to see errors in a red footer immediately, so that I know when a write failed.
22. As a Vimkunja user, I want trivial errors (DB locks that auto-retry successfully) to show green feedback briefly, so that I'm not alarmed.
23. As a Vimkunja user, I want persistent errors to stay visible until I press a key, so that I don't miss critical failures.
24. As a Vimkunja user, I want errors to be logged even if I dismiss the footer, so that I can review them later via `:errors`.
25. As a Vimkunja user, I want to press `Enter` on a task to open a detail pane, so that I can see full task information.
26. As a Vimkunja user, I want the detail pane to be a full pane (not a modal), so that I can navigate between detail and task list with `Ctrl+W`.
27. As a Vimkunja user, I want to press `Esc` or `:q` in the detail pane to close it, so that I can return to the task list.
28. As a Vimkunja user, I want focus to return to the task list when I close detail, so that I can continue navigating tasks.
29. As a Vimkunja user, I want to press `Space` followed by a command key (within 500ms), so that I can use leader-key shortcuts.
30. As a Vimkunja user, I want to press `Space+o` or `Ctrl+W o` to collapse the projects pane, so that I can maximize space for task/kanban views.
31. As a Vimkunja user, I want the collapsed projects pane to toggle back when I press `Space+o` again, so that I can restore the layout.
32. As a developer, I want a command registry/dispatcher pattern, so that adding new commands (`:goto`, `/search`) is trivial.
33. As a developer, I want a `FakeStore` test implementation, so that I can test TUI logic without a real DB.
34. As a developer, I want Store tests to use real SQLite tempfiles, so that DB side-effects (bucket moves, timestamps) are validated.
35. As a developer, I want TUI model tests to mock the Store, so that pane navigation logic is testable independently.
36. As a Vimkunja user, I want to create a task within the TUI for the current project or in the current view, and optionally set its due date, priority, and description.

---

## Implementation Decisions

### 1. Remove REST Client (Decision #1)
- **Action:** Delete `/internal/vimkunja/client.go` entirely.
- **Rationale:** 1-user setup, always local SQLite access, no need for HTTP abstraction.
- **Impact:** `Store` interface remains, but only `SQLiteStore` implementation exists.

### 2. Write Semantics: Server-Fidelity (Decision #2)
- **Module:** `SQLiteStore.ToggleTaskDone()` (and future write methods)
- **Side-effects to replicate:**
  - Bump `tasks.updated` to `NOW()` on every write
  - When toggling to `done=1`, sync the task into the `done_bucket_id` of every project view that has one.
  - Query: `SELECT pv.id, pv.done_bucket_id FROM project_views pv JOIN tasks t ON t.project_id = pv.project_id WHERE t.id = ? AND pv.done_bucket_id IS NOT NULL`
  - Update each view separately: `UPDATE task_buckets SET bucket_id = ? WHERE task_id = ? AND project_view_id = ?`
- **Rationale:** Core Law #1 (schema compatibility) — DB state must match what the real Vikunja server produces.

### 3. Auto-Refresh After Write (Decision #3)
- **Module:** `model.Update()` — after `ToggleTaskDone` (and future writes)
- **Behavior:** On successful write → automatically call `loadTasks()` to reload current view.
- **Error case:** On write failure → show red footer, do NOT refresh (stale data is better than hiding the error).
- **Rationale:** Pragmatic concurrency model (no optimistic locking), keeps UI in sync.

### 4. Error Handling: Three-Tier System (Decision #9)
- **Tier 1: Error Log**
  - **Module:** `internal/errors/log.go` (new package)
  - **API:** `errors.Log(msg string)` — appends to in-memory slice
  - **Command:** `:errors` opens a pane showing all logged errors (scrollable)
  - **Implementation (MVP):** `:errors` shows "Not implemented" placeholder initially, log infrastructure built first.

- **Tier 2: Footer Feedback**
  - **Module:** `model.renderStatusBar()`
  - **Trivial errors (e.g., DB lock → retry succeeded):** Green "write success" toast for 2-3 seconds
  - **Real errors:** Red footer with error text, stays visible until user presses any key (including `j/k`)
  - **State:** `model.lastErr` (string), `model.toast` (string), `model.toastUntil` (time)

- **Tier 3: Auto-Refresh**
  - See Decision #3 — successful writes auto-refresh, failures do not.

### 5. Command-Line Infrastructure (Decision #6)
- **Module:** `internal/tui/commands.go` (new)
- **Pattern:** Command registry (map[string]CommandHandler)
- **Interface:**
  ```go
  type CommandHandler func(m *model, args []string) (tea.Model, tea.Cmd)
  
  type CommandRegistry struct {
      handlers map[string]CommandHandler
  }
  
  func (r *CommandRegistry) Register(name string, aliases []string, handler CommandHandler)
  func (r *CommandRegistry) Execute(m *model, input string) (tea.Model, tea.Cmd, error)
  ```

- **Commands (MVP):**
  - `:q` → Close current pane (or quit if last pane)
  - `:qa` / `:quitall` → Quit app immediately
  - `:h` / `:help` → Open help-pane
  - `:errors` → Open error-log pane (placeholder initially)

- **UI Flow:**
  - `:` key → model enters "command mode" (footer shows `> :_` prompt)
  - User types command + `Enter` → dispatch to registry
  - `Esc` → cancel command mode

- **Extensibility:** Later commands (`:goto <project>`, `/search`) register into same system.

- **Autocomplete:** Later commands need autocomplete system (with `Tab`, same as in Vim)

### 6. Create Task (Phase 4 — short grilling required)
Before implementation, make these product decisions in a short grilling session:
- **Target:** Create in the current view, the selected project, or require an explicit project/view choice?
- **Keybinding:** Use `o` (Vim-style open below), `%` (netrw-style new item), or another binding?
- **Command:** Add `:new`, and if so, which arguments are accepted?

No create-task UI or Store API is specified until these decisions are recorded.

### 7. Help-Pane (Decision #10)
- **Module:** `internal/tui/help.go` (new)
- **Content:** Static markdown-style text, scrollable with `j/k`
- **Structure:**
  - **Top 10-20 lines (no scroll needed):** Most common commands (`hjkl`, `gg/G`, `Ctrl+W`, `:q`, `x`, `r`, `Enter`)
  - **Below (scrollable):** Full keybinding list, all commands, explanations
- **Rendering:** Re-use pane-rendering logic (border, padding, fit-block)
- **Lifecycle:** Help-pane is a special pane (like Detail-pane). Opens via `:h` or `?`, closes via `:q` or `Esc`.

### 8. Leader-Key Support (Decision #13)
- **Module:** `model.Update()` — add `leaderArmed` state + timeout
- **Config:**
  - Leader key = **Space**
  - Timeout = **500ms** (if no second key within 500ms, leader disarms)
- **Implementation:**
  - When `Space` pressed → `model.leaderArmed = true`, `model.leaderTimeout = time.Now().Add(500ms)`
  - Next key within 500ms → dispatch leader sequence (e.g., `Space+o` → toggle projects pane)
  - After 500ms OR after dispatching → `model.leaderArmed = false`
- **Leader Sequences (MVP):**
  - `Space+o` → Toggle projects pane collapse (see Decision #13)
  - Future: `Space+f` (filter), `Space+g` (goto), etc.

### 9. Projects-Pane Collapse (Decision #13)
- **Keybindings:** `Space+o` OR `Ctrl+W o`
- **State:** `model.projectsPaneCollapsed` (bool)
- **Layout Impact:**
  - Collapsed → Projects pane width = 0, Tasks pane gets full width
  - Expanded → Default 30% / 70% split
- **Focus Impact:** When collapsed, `Ctrl+W h` from Tasks-pane is a no-op (can't focus invisible pane).

### 10. Detail-Pane as Full Pane (Decision #4)
- **Current state:** Already implemented (Decision #4 confirmed it's not a modal)
- **Clarification:** Detail-pane is in `neighbors` graph, can be reached with `Ctrl+W j` from Tasks-pane.
- **Rationale:** Prepares for future edit-mode (Decision #18-19) — user needs to switch focus between Detail fields and Task list.

### 11. Architecture: Packages
- **`internal/vimkunja/`**
  - `store.go` — `Store` interface (unchanged)
  - `sqlite_store.go` — `SQLiteStore` (enhanced: bump `updated`, bucket moves)
  - `types.go` — `Project`, `Task`, `ProjectView` structs (unchanged)
  - **DELETE:** `client.go` (REST client removed)

- **`internal/tui/`**
  - `model.go` — Bubbletea model (enhanced: command mode, leader-key, error states)
  - `panes.go` — Rendering functions (unchanged for now)
  - `commands.go` — Command registry + handlers (new)
  - `help.go` — Help-pane content + rendering (new)
  - `model_test.go` — TUI model tests with `FakeStore` (new)

- **`internal/errors/`** (new package)
  - `log.go` — Error log (in-memory slice, `Log(msg)` + `GetAll()`)

- **`internal/vimkunja/sqlite_store_test.go`** (new)
  - Store tests with real SQLite tempfiles

### 12. Testing Strategy
See "Testing Decisions" section below.

---

## Testing Decisions

### What Makes a Good Test
- **Test external behavior, not implementation details.**
  - ✅ Good: "After ToggleTaskDone(done=true), task is in done_bucket_id"
  - ❌ Bad: "ToggleTaskDone calls sql.Exec with this exact query string"
- **Use the highest seam possible** (prefer Store-level tests over SQL-mocking).
- **Tests should be fast** (in-memory SQLite, no sleep/wait).
- **Tests should be isolated** (each test gets fresh DB, no shared state).

### Test Seam 1: Store Interface (Highest Priority)
- **Module:** `internal/vimkunja/sqlite_store_test.go`
- **Seam:** `Store` interface methods
- **Setup:** Real SQLite with `:memory:` or temp file
- **Fixtures:** Helper functions to seed test data:
  ```go
  func seedProject(t *testing.T, store *SQLiteStore, title string) int64
  func seedTask(t *testing.T, store *SQLiteStore, projectID int64, title string, done bool) int64
  func seedView(t *testing.T, store *SQLiteStore, projectID int64, viewKind string, doneBucketID int64) int64
  ```

- **Test Cases:**
  - `TestToggleTaskDone_SetsUpdatedTimestamp` — Verify `tasks.updated` is bumped
  - `TestToggleTaskDone_MovesToDoneBucket` — Verify task moved in `task_buckets` table
  - `TestToggleTaskDone_NoDoneBucket_NoMove` — If view has no `done_bucket_id`, task stays in current bucket
  - `TestListTasksInView_OrdersByPosition` — Verify `task_positions.position` sorting
  - `TestListProjects_ExcludesArchived` — Verify `is_archived=1` projects hidden

- **Prior Art:** No existing tests in codebase (this is the first test suite).

### Test Seam 2: TUI Model (Medium Priority)
- **Module:** `internal/tui/model_test.go`
- **Seam:** `model.Update()` function
- **Setup:** Mock Store via `FakeStore` struct:
  ```go
  type FakeStore struct {
      projects []vimkunja.Project
      tasks    []vimkunja.Task
      toggleDoneErr error // for testing error paths
  }
  
  func (f *FakeStore) ListProjects(...) ([]vimkunja.Project, int, error) {
      return f.projects, 1, nil
  }
  // ... implement all Store methods
  ```

- **Test Cases:**
  - `TestPaneFocus_CtrlW_Cycles` — Press `Ctrl+W` → focus cycles Projects → Tasks
  - `TestPaneFocus_CtrlWH_MovesLeft` — Press `Ctrl+W h` from Tasks → focus Projects
  - `TestPaneFocus_CtrlWJ_OpensDetail` — Press `Ctrl+W j` from Tasks → focus Detail (if open)
  - `TestVimMotion_gg_JumpsTop` — Press `g`, then `g` → `projectSelected = 0`
  - `TestVimMotion_G_JumpsBottom` — Press `G` → `projectSelected = len(projects)-1`
  - `TestCommandMode_ColonQ_QuitsPaneOrApp` — Press `:`, type `q`, Enter → quit behavior
  - `TestLeaderKey_SpaceO_TogglesProjectsPane` — Press `Space`, then `o` within 500ms → `projectsPaneCollapsed` toggles
  - `TestLeaderKey_Timeout_Disarms` — Press `Space`, wait 600ms, press `o` → no toggle (leader timed out)
  - `TestErrorHandling_WriteFailure_RedFooter` — `FakeStore.toggleDoneErr = someError` → `model.lastErr` set, footer red

- **Prior Art:** No existing tests in codebase.

### Test Seam 3: Rendering (Out of Scope)
- **Decision:** No automated rendering tests (Decision from grilling session).
- **Rationale:** Lipgloss styling is fragile to test, changes frequently, manual testing sufficient.
- **Manual Testing Checklist (documented in `TUI_IDEA.md`):**
  - Projects list renders with `[X]` marker for selected
  - Tasks show colored priority (green=low, red=urgent)
  - Done tasks are grayed out + faint
  - Detail pane shows all task fields
  - Help pane shows keybindings clearly
  - Error footer turns red on failure

---

## Out of Scope

This spec focuses on **Fundament only** (Decisions #1-11). The following are explicitly deferred to later specs:

### Deferred to "Feature: OverviewView" (Prio 1)
- Next to the existing List View of the MVP: another view_kind
- should list all tickets, regardless the project
- done tickets should not be shown (at least per default filtered out)
- tickets should be sorted by DueDate first, then by Priority (at least per default)
- later extension: sorting/filtering could be chosen/selected by user

### Deferred to "Feature: Kanban View" (Prio 2)
- Rendering `view_kind=kanban` (Decisions #12-17)
- Bucket columns (horizontal layout, dynamic width)
- 2D navigation (`h/l` for buckets, `j/k` for tasks within bucket)
- Task movement between buckets (`Shift+H/L`, `m+digit`)
- Kanban-specific toggle-done behavior (auto-move + visual feedback)

**Binding Kanban constraint:** Every manual Kanban view has exactly one configured `done_bucket_id`. Its bucket title remains user-configurable, but the UI must make its done semantics clear (for example `Done` or a visible `(Done)` suffix). When a task is marked done, its bucket assignment is synchronized to each Kanban view's own done bucket; `tasks.done` remains the single global source of truth.

### Deferred to "Feature: Edit Mode" (Prio 3)
- Title edit inline (`i` in List/Kanban-pane) (Decision #18)
- Detail-pane multi-field edit (`hjkl` field navigation, `i` insert mode) (Decision #18-19)
- Auto-commit on pane-switch (Decision #19)
- Date-picker, priority +/- keybindings

### Deferred to Later Features
- Search/Filter (`/` command) (Prio 4)
- Comments (`a` to append comment) (Prio 5)
- Undo (`u` keybinding)
- Table/Gantt views
- Autocomplete for commands with arguments (`:goto <project>`)
- Project View should reflect the hierarchy of projects (via indentation)


---

## Further Notes

### Migration Path from PoC to Fundament
1. **Phase 1 (Cleanup):** Delete `client.go`, verify tests still pass (currently no tests, so just compile).
2. **Phase 2 (Error Infra):** Add `internal/errors/log.go`, integrate into `model.Update()`.
3. **Phase 3 (Commands):** Add `commands.go`, implement `:q/:qa/:h` handlers, wire into `model.Update()`.
4. **Phase 4 (Create Task):** Run the short grilling session, then specify and implement task creation.
5. **Phase 5 (Help):** Add `help.go`, static help content, test with `:h` and `?`.
6. **Phase 6 (Leader):** Add leader-key state machine to `model.Update()`, implement `Space+o` toggle.
7. **Phase 7 (Write Semantics):** Enhance `SQLiteStore.ToggleTaskDone()` to bump `updated` + move buckets.
8. **Phase 8 (Tests):** Write `sqlite_store_test.go` + `model_test.go`, verify all behaviors.

### Domain Glossary Alignment
- **Pane:** A UI section (Projects, Tasks, Detail). Not "panel" or "view" (conflicts with ProjectView).
- **View:** A `project_views` row (Kanban, List, Table, Gantt). Not "pane".
- **Focus:** Which pane is active (receives key input). Not "selection".
- **Selection:** Which item within a pane is highlighted (e.g., `projectSelected`, `taskSelected`).
- **Leader:** A prefix key (`Space`) that arms a sequence. Not "meta" or "modifier".

### ADRs Referenced
- None yet (this is the first major architectural decision document).
- Future ADRs should link back to this spec and `TUI_IDEA.md` decisions.

### Open Questions for Implementation
1. **Command parsing:** Simple split-on-space, or shlex-style quoted args?
   - **Recommendation:** Start simple (split-on-space), upgrade if needed for `:goto "My Project"`.
2. **Error log pane:** Same rendering as Detail-pane, or custom scrollable list?
   - **Recommendation:** Re-use Detail-pane layout (less code).
3. **Leader timeout:** 500ms hardcoded, or config file?
   - **Recommendation:** Hardcode initially, add config later if users complain.

### Success Criteria
This spec is **done** when:
- ✅ `client.go` deleted, codebase compiles
- ✅ `:q`, `:qa`, `:h` commands work
- ✅ `?` opens help-pane
- ✅ `Space+o` collapses projects-pane
- ✅ `x` on a task bumps `updated` + moves to done-bucket
- ✅ Auto-refresh after successful `x`
- ✅ Red footer on write errors, errors logged to `:errors`
- ✅ All Store tests pass (5+ test cases)
- ✅ All TUI model tests pass (8+ test cases)
- ✅ Manual smoke test checklist completed

---

**Next Steps:**
1. Review this spec with team/stakeholders.
2. Create implementation tasks (issues or TODO.md).
3. Start with Phase 1 (cleanup), iterate through Phase 8.
4. After Fundament is stable → write "SPEC_KANBAN.md" for Prio 1 features.
