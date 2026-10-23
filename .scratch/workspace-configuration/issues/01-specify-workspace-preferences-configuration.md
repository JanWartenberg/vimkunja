# 01 — Specify workspace and data preferences configuration

**What to specify:** Define user-configurable workspace and data behavior separately from local TUI behavior.

**Category:** enhancement

**Milestone:** 3

**Priority:** Low

**Status:** needs-triage

## Initial scope

- Default Project
- Default View
- Startup filters or search scope
- Other persisted navigation/workspace preferences

## Explicit exclusions

- Database selection (`--db`), which remains a separate operational concern tracked by the [first-run database ticket](../../first-run-database/issues/01-support-first-run-database-and-default-path.md).
- Local TUI behavior such as key timing, which is covered by the user-configuration ticket.

## Further context

This ticket was split from the [local TUI configuration work](../../../.archive/2026-10-20/user-configuration/issues/01-add-user-facing-tui-configuration.md), which is now complete and archived. Workspace preference decisions remain open.
