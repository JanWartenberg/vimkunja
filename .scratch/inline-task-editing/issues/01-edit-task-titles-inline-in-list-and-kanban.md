# 01 — Edit Task Titles inline in List and Kanban Views

**What to build:** A selected Task title can be edited inline from List and future Kanban Views with Vim-like save and discard semantics.

**Implementation prerequisites:** Kanban integration depends on the Kanban View; List behavior still needs a focused specification.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Define entry, cursor motions, commit, discard, and focus-change behavior.
- [ ] Reuse Task Title validation and Store update semantics.
- [ ] Define refresh and error behavior without losing the active Task selection.
- [ ] Specify Kanban behavior as an extension after the board exists.

## Further context

Archived sources: `SPEC_FUNDAMENT.md`, “Edit Mode”, and `TUI_IDEA.md`, Decision 18.
