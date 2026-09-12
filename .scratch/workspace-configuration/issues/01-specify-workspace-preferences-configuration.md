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

- Database selection (`--db`), which remains a separate operational concern.
- Local TUI behavior such as key timing, which is covered by the user-configuration ticket.

## Further context

This ticket was split from `.scratch/user-configuration/` during specification. Triage it after the local TUI configuration work is complete.
