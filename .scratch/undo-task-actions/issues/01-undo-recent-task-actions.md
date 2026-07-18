# 01 — Undo recent Task actions

**What to build:** A user can undo a recent destructive or workflow-changing Task action through a Vim-like interaction.

**Blocked by:** Needs an explicit undo boundary and persistence model.

**Category:** enhancement

**Status:** needs-triage

- [ ] Decide which actions are undoable: toggle, workflow move, edit, create, delete, or others.
- [ ] Define scope, lifetime, error behavior, and user feedback for undo history.
- [ ] Preserve Vikunja-compatible writes and avoid silently undoing external changes.

## Further context

Archived source: `TUI_IDEA.md`, Kanban undo TODO.
