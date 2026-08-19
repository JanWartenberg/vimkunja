# 01 — Edit Task Titles inline in List and Overview

**What to build:** A selected Task title can be edited inline from List and Overview with Vim-like save and discard semantics. Kanban inline editing is deferred; Task Detail remains available there.

**Implementation prerequisites:** None.

**Category:** enhancement

**Priority:** Low

**Status:** implemented

## Resolution

- List and Overview: `i` opens the selected title at its end; `Shift+I` opens it at its beginning.
- The inline field reuses the Detail title input's rune-aware text operations: literal typed text, Left/Right, Home/End, Backspace, and Delete. Its Store operation updates only the title, so stale List/Overview metadata cannot overwrite other Task fields.
- `Enter` saves immediately; `Esc` discards the inline draft. `Ctrl+W` changes Pane focus only after a successful save.
- An empty title rejects the save and retains the draft. A failed write retains the draft and reports the existing error feedback; retry with `Enter` or discard with `Esc`.
- A successful save updates the List or Overview row and preserves the current Task selection.
- Kanban inline editing is deferred; use Task Detail for title changes while browsing a Board.

- [x] Define entry, cursor motions, commit, discard, and focus-change behavior.
- [x] Reuse Task Title validation and Store update semantics.
- [x] Define refresh and error behavior without losing the active Task selection.
- [x] Defer Kanban behavior; Task Detail remains its edit path.

## Further context

Archived sources: `SPEC_FUNDAMENT.md`, “Edit Mode”, and `TUI_IDEA.md`, Decision 18.
