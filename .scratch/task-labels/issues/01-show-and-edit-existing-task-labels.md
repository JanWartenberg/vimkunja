# 01 — Show and edit existing Task Labels

**What to build:** A Task Detail shows and can assign existing Vikunja Labels without changing the database schema.

**Implementation prerequisites:** Needs a Labels interaction spec and schema/API compatibility check.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

- [ ] Read and display Labels assigned to a Task.
- [ ] Define a keyboard Picker/search flow for assigning and removing existing Labels.
- [ ] Decide whether creating, editing, and deleting Labels belongs to this feature or later work.
- [ ] Reuse the Global Search ticket's Label discovery and label-filtered Overview scope for existing-Label navigation; keep mutation UI here.

## Further context

Archived source: `SPEC_FUNDAMENT.md`, “Labels (später)”. Global Search owns read-only Label discovery and opening the label-filtered Overview; this ticket owns Task Detail Label display, assignment, and removal.
