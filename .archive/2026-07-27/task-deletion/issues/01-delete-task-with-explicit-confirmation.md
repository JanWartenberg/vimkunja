# 01 — Delete a Task with explicit confirmation

**What to build:** A user can delete a selected Task only after an explicit confirmation, with Vikunja-compatible persistence, error feedback, and refresh.

**Implementation prerequisites:** None — implemented with explicit confirmation and transactional persistence.

**Category:** enhancement

**Priority:** High

**Status:** resolved

- [x] `dd` opens `Delete “Task title”? y yes / n or Esc cancel`; only `y` persists deletion.
- [x] Delete the Task and Vikunja-owned relations transactionally without changing the schema; attachment files remain retained behind their removed Task link.
- [x] Failures use the existing footer/session error-log path; only success refreshes the active Tasks Pane or Overview.
- [x] Store and TUI tests cover relation cleanup, confirmation, cancellation, failure, and refresh.

## Further context

This is a Daily-driver basic operation and deliberately takes priority over presentation polish and rich-text work.

Archived source: `SPEC_FUNDAMENT.md`, “Task deletion (future feature)”.

## Comments

### 2026-07-27 — Resolved

Implemented with `dd` confirmation, transactionally cleaned Task relations, error feedback, and refresh behavior. `go test ./...` passed.
