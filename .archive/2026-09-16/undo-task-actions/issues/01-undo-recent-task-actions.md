# 01 — Undo recent Task actions

**What to build:** A user can undo a recent destructive or workflow-changing Task action through a Vim-like interaction.

**Implementation prerequisites:** None; the undo boundary and persistence model are specified below.

**Category:** enhancement

**Priority:** Medium

**Status:** completed

## Accepted behavior

- `u` undoes and `Ctrl-R` redoes the latest global, chronological Task action.
- A successful save of an existing Task is one Undo Entry. All changed fields in that save are restored together.
- Direct atomic Task actions are also Undo Entries: Done-/Default-Bucket toggle, workflow/bucket move, inline title save, and Project move.
- No-op saves create no entry. Task creation, Task deletion, and Comments are outside this history.
- `:wqa` still saves Task and Comment atomically; when a Comment is included, the operation creates no Undo Entry.
- History is session-local, in memory only, and has a fixed combined maximum of 20 Undo/Redo Entries. It is empty after restart.
- Undo moves an entry to Redo; Redo moves it back. A new successful Task change after Undo clears Redo. Restores do not create additional entries.
- Each entry identifies the Task and contains its complete before/after undo state plus changed-field metadata. The state covers Project, title, description, priority, due date, labels, Done, and workflow/bucket assignment.
- Undo/Redo is available only in Normal Mode in List, Overview, Kanban, and Task Detail. It is blocked in Projects, prompts, pickers, Search, Comments, Insert Mode, while persistence is in flight, or while Task/Comment drafts are dirty. The footer explains the latter case.
- Selection, focus, and scroll position remain unchanged after a successful restore. Visible affected Tasks refresh in place; the UI does not jump to a globally targeted Task.
- Restores are atomic and first verify that the current undo-relevant state matches the expected before/after state. Conflicts, missing Tasks, Projects, or Buckets abort without writing or changing either history stack. Failures are shown in the footer and retained in `:errors`.
- Successful feedback is compact and names the Task (identifier when available, otherwise title plus ID) and changed fields. Empty-stack feedback stays in the footer; there is no separate status log.

## Implementation plan

1. Add the session-local Undo/Redo domain model and normalized Task snapshots without changing the SQLite schema.
2. Capture entries only after successful Task-only persistence; keep `:wqa` with Comment out of the history.
3. Add atomic, conflict-checked restore support for all snapshot fields, including Labels, Project, and Bucket.
4. Bind `u`/`Ctrl-R` in the approved Normal-Mode contexts and guard dirty/in-flight states.
5. Refresh affected UI state without changing selection/focus and add Store/model regression coverage for success, no-op, conflicts, failures, and stack branching.

## Further context

Unsaved Description-draft undo/redo is defined separately by `.scratch/description-rendering/spec.md`; this ticket concerns persisted Task actions and must not duplicate that local editor history.

Archived source: `TUI_IDEA.md`, Kanban undo TODO.

## Comments

### 2026-09-15 — Grilling decision record

The accepted behavior above was agreed during a grilling session. The design deliberately keeps persisted Task history separate from local Description editing and from the separate Task Comment record. The first implementation favors safe aborts over overwriting external changes and avoids introducing SQLite schema or status-log complexity.

### 2026-09-16 — Implemented

Implemented bounded session-local history, snapshot capture for Task saves and direct actions, atomic conflict-checked SQLite restores, Normal-Mode bindings and guards, in-place refresh, feedback, and Store/model regression coverage.

### 2026-09-16 — Acceptance complete

Manual acceptance covered multi-field saves, Detail Done after explicit save, Project moves, no-op saves, redo branching, dirty-draft and context guards, UI stability, the 20-entry limit, and restart behavior. Kanban moves, inline titles, and Label removal were also verified. External-conflict safety remains covered by automated Store tests.
