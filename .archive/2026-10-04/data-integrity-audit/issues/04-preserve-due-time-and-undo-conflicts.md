# 04 — Preserve full due timestamps and detect time conflicts

**Finding:** F04

**Category:** bug

**Priority:** High

**Status:** completed

## Problem

Title-only Detail saves and inline-title undo replace due time with midnight; an external same-day time change is not detected by undo.

## Acceptance

Keep date-only input while retaining exact untouched persisted timestamps. Include full due time in optimistic undo comparisons. Explicit date edits retain their existing semantics. Test both save and undo preservation.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../../.scratch/data-integrity-audit/report.md) and `TestIntegrityAuditDueTimePreservation` in the ordinary Store tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-04 — Repaired and automatically verified

Root cause: the date-only read model was written back as midnight, while undo
snapshots discarded the time before comparison and restoration. The three
original scenarios failed twice before repair, then were promoted to ordinary
regression coverage in `internal/vimkunja/due_time_test.go`.

Existing Task updates compare the submitted date with the transaction's current
displayed date. An unchanged date leaves SQLite's full value untouched; a changed
date still stores midnight UTC, and clearing stores NULL. This shared write path
also covers combined Task-plus-Comment saves. Undo snapshots now retain exact
SQLite due-date text for conflict detection and restoration, while the editor
and read models continue to show YYYY-MM-DD. Undo history recognizes full-value
changes as due-date changes.

Focused tests pass three repetitions: Task and combined Comment saves, exact
fractional/offset/date-only/NULL preservation, undo/redo, date edits/clearing,
nanosecond external conflicts, late write rollback, and the real TUI save and
undo/redo completion flow. Full race-enabled tests (including corpus checks),
both entrypoint builds and vet pass. Opt-in audit safety controls also pass.
No schema/dependency changes or user database access. No manual acceptance,
live-server compatibility or native Windows verification is claimed. Ordinary
save concurrency protection beyond this preservation fix remains out of scope.

### 2026-10-04 — Thorough follow-up review

Fixed a medium-severity validation gap: RestoreTaskUndoState accepted malformed
raw timestamps, a raw timestamp for a different target date, or a retained raw
timestamp alongside an empty target date. Three real SQLite regressions failed
before repair. Restoration now validates the original encoding and requires its
calendar date to match DueDate before any writes. Validation follows the formats
recognized by the current SQLite driver and preserves the encoded local date,
offset and precision. Exact expected-state comparison remains conservative:
changing only the stored timestamp encoding can also reject undo.

Fixed the archived ticket's report link and updated its evidence pointer to the
ordinary tests. Expanded checks cover 13 timestamp/NULL variants for Task and
Task-plus-Comment round trips, offsets crossing UTC date boundaries, driver Go
Time encodings, Comment insertion failure after the Task update, and the real
TUI's retained editor/history after a rejected time-conflict undo. Focused Store
and TUI checks pass three repetitions. Full race tests, build and vet pass.
No other integrity finding was repaired; nothing was committed.
