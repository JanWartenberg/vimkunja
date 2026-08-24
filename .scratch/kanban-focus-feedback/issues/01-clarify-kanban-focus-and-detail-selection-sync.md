# 01 — Clarify Kanban focus and Task Detail selection synchronization

**What to build:** In a Kanban View, focused and unfocused regions are visually distinguishable at a glance, and the Task Detail Pane follows the selected Task while navigating Bucket cards.

**Implementation prerequisites:** None.

**Category:** enhancement

**Priority:** Medium

**Status:** completed

## Decisions confirmed

- Focus is a general styling rule: use a highlighted pane/bucket border and, where useful, a `> task` selection marker. Inactive content retains its normal color; never dim it.
- Kanban selection updates an already-open, clean Detail Pane automatically. Selection navigation never opens Detail; `Enter` opens it and moves focus there.
- List and Overview remain deliberate: `v` previews the selected Task in Detail without moving Pane focus; later `j`/`k` leaves that preview unchanged until another `v` or `Enter`.
- A dirty Detail draft is never silently replaced by selection navigation. Move, Done-toggle, and delete actions are blocked until it is saved or discarded. Autosave and close-time decisions are deferred configuration work.
- When an already-open, clean Detail Pane reaches an empty Bucket, render a horizontally and vertically centered `No task selected` state. The open empty pane remains focusable through `Ctrl+W`; it has no editable fields.
- In that empty, focused Detail state, `i` and `o` start a new Task editor for the active Project and Bucket. On save, the Task appears in that Bucket, becomes selected, and remains shown in Detail. Discard returns to the same empty Bucket and empty Detail state.
- After deletion, select the preceding card when available, otherwise the following card; deleting the last card clears selection and Detail.
- Explicit card moves and Done-triggered workflow moves preserve the moved Task as the selected card and keep it in Detail.
- An Overview preview may update the internal Project context required to load the Task, but does not alter Projects Pane visibility or Pane focus.

- [x] Preserve existing Kanban navigation, Task opening, scrolling, and narrow-terminal rendering behavior.
- [x] Add model-level rendering and selection-synchronization regression coverage without changing Vikunja persistence or schema.

## Further context

Captured from the prior temporary `TODO`: "highlight either TaskDetail Or Bucket" and update the selected Task Detail while selecting cards in Buckets.
