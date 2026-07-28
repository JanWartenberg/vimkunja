# 01 — Manage manual Kanban Buckets

**What to build:** A Vimkunja user can manage the Buckets of a Project's single manual Kanban View without leaving the terminal.

**Implementation prerequisites:** Initial Kanban-View provisioning and rendering/navigation of the single manual Kanban View must be specified and implemented first. Multiple Kanban Views remain outside this ticket.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Define the keyboard-first entry point for Bucket administration without conflicting with Task navigation.
- [ ] Create a Bucket with Vikunja-compatible title, position, and optional limit semantics.
- [ ] Rename a Bucket and define validation and persistence feedback.
- [ ] Define reordering of Buckets and preserve Vikunja's persisted positions.
- [ ] Delete a Bucket using Vikunja-compatible behavior: move its Tasks to the View's Default Bucket; reject deletion of the last Bucket; require explicit confirmation.
- [ ] Keep Default- and Done-Bucket configuration valid when Buckets are edited or removed.
- [ ] Add Store-level SQLite regression coverage and TUI interaction coverage; do not change the Vikunja schema.

## Further context

Upstream Vikunja supports CRUD for Buckets scoped to a Project View. A manual Kanban View is the only initial supported kind because it has persisted, movable Buckets. The upstream default Bucket template and its semantics are documented in [`../kanban-board/research-vikunja-default-kanban.md`](../../kanban-board/research-vikunja-default-kanban.md).
