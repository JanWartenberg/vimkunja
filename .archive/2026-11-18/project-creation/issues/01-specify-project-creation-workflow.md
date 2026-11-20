# 01 — Project creation workflow

**What to build:** Create Top-level Projects and Subprojects from Vimkunja according to the accepted [specification](../spec.md).

**Implementation prerequisites:** Product decisions accepted by Jan on 2026-11-17. Implemented and verified on 2026-11-18 through implement-and-verify.

**Category:** enhancement

**Priority:** Low

**Status:** completed

- [x] Define the entry point: Ex command, Projects-Pane keybinding, or both.
- [x] Define required and optional fields, including title validation, parent Project selection, initial View behavior, and cancellation.
- [x] Decide whether a new Project receives the standard manual Kanban View immediately or only through the existing `:kanban` provisioning flow.
- [x] Define success behavior: reload hierarchy, select/focus the new Project, and error feedback.
- [x] Verify Vikunja-compatible SQLite persistence without schema changes, including ownership and Project position.
- [x] Split Store and TUI implementation work into follow-up tickets after the workflow is fixed.

## Further context

The design interview concluded on 2026-11-17 with Jan accepting the complete
[specification](../spec.md). This includes global rejection of new duplicate Titles
while preserving read support for existing duplicates, explicit Parent selection,
standard Views, single-User ownership and transactional persistence. No unresolved
product decisions remain. Store and TUI implementation are completed in tickets 02 and 03.

Projects can now be created from Vimkunja with their standard Views and manual Kanban Buckets, preserving the upstream SQLite schema.

The [first-run database work](../../../2026-11-13/first-run-database/issues/01-support-first-run-database-and-default-path.md) now provisions one upstream-generated Inbox in fresh installations. General Project creation, parent selection and ownership behavior remain outside that completed work and are covered by this specification.

Completed on 2026-11-18. Verification and completed Windows acceptance are recorded in the [specification](../spec.md#verification-result--2026-11-18). Jan confirmed Windows acceptance and authorized the commit on 2026-11-20.
