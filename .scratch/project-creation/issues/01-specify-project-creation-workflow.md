# 01 — Specify Project creation workflow

**What to decide:** Define how users create a Vikunja-compatible Project from Vimkunja.

**Implementation prerequisites:** Requires product and persistence decisions before implementation.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Define the entry point: Ex command, Projects-Pane keybinding, or both.
- [ ] Define required and optional fields, including title validation, parent Project selection, initial View behavior, and cancellation.
- [ ] Decide whether a new Project receives the standard manual Kanban View immediately or only through the existing `:kanban` provisioning flow.
- [ ] Define success behavior: reload hierarchy, select/focus the new Project, and error feedback.
- [ ] Verify Vikunja-compatible SQLite persistence without schema changes, including ownership and Project position.
- [ ] Split Store and TUI implementation work into follow-up tickets after the workflow is fixed.

## Further context

Projects are currently read from the local Vikunja SQLite database only. Creation is needed to make the Kanban-provisioning path reachable for Projects created within Vimkunja.
