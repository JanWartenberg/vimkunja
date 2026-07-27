# 01 — Show and create Task Comments

**What to build:** A Task Detail can show its Vikunja Comments and append a new Comment through a keyboard workflow.

**Implementation prerequisites:** Needs schema compatibility research and interaction specification.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Define comment ordering, pagination, and Detail-pane space behavior.
- [ ] Define `a` or another append flow, save/discard behavior, and error feedback.
- [ ] Add a Store operation for creating a Comment while preserving Vikunja semantics.
- [ ] Keep editing and deleting existing Comments explicitly out of the first version.

## Further context

Comments are deliberately outside the current Daily-driver milestone and should not delay the All-Tasks overview, Kanban View, or Task deletion.

Archived source: `SPEC_FUNDAMENT.md` and `TUI_IDEA.md`, “Kommentare”.
