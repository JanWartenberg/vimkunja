# 01 — Render and navigate Kanban Views

**What to build:** A Project's `view_kind=kanban` can render as a dynamic bucket board in the Tasks Pane, with keyboard navigation and Vikunja-compatible task moves.

**Blocked by:** Needs a dedicated Kanban spec that reconciles the archived Decisions 12–17 with the current workflow-status abstraction.

**Category:** enhancement

**Status:** needs-triage

- [ ] Define bucket columns, narrow-terminal behavior, horizontal scrolling, and its indicator.
- [ ] Define 2D selection and `hjkl` navigation across buckets and cards.
- [ ] Define direct task moves to adjacent and arbitrary buckets, including focus and feedback.
- [ ] Preserve persisted bucket order and task positions; decide reordering within a bucket separately.
- [ ] Reuse the existing workflow-status Store semantics and define behavior for zero or multiple manual Kanban Views.

## Further context

Archived sources: `TUI_IDEA.md`, Decisions 12–17 and Kanban roadmap; `SPEC_FUNDAMENT.md`, “Feature: Kanban View”. The archived checkmarks document design decisions, not implemented UI.
