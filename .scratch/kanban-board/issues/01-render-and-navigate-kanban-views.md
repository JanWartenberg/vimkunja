# 01 — Render and navigate Kanban Views

**What to build:** A Project's `view_kind=kanban` can render as a dynamic bucket board in the Tasks Pane, with keyboard navigation and Vikunja-compatible task moves.

**Blocked by:** Needs a dedicated Kanban spec that reconciles the archived Decisions 12–17 with the current workflow-status abstraction.

**Category:** enhancement

**Priority:** High

**Status:** needs-triage

- [ ] Define bucket columns, narrow-terminal behavior, horizontal scrolling, and its indicator.
- [ ] Define 2D selection and `hjkl` navigation across buckets and cards.
- [ ] Define direct task moves to adjacent and arbitrary buckets, including focus and feedback.
- [ ] Preserve persisted bucket order and task positions; decide reordering within a bucket separately.
- [ ] Reuse the existing workflow-status Store semantics. The initial Daily-driver scope supports one active/manual Kanban View; zero or multiple manual Kanban Views are specified by the separate follow-up ticket.

## Further context

Archived sources: `TUI_IDEA.md`, Decisions 12–17 and Kanban roadmap; `SPEC_FUNDAMENT.md`, “Feature: Kanban View”. The archived checkmarks document design decisions, not implemented UI.
