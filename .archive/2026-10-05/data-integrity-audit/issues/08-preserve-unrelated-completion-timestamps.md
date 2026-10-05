# 08 — Preserve completion timestamps during unrelated edits

**Finding:** F08

**Category:** bug

**Priority:** Medium

**Status:** completed

## Problem

A title-only Detail save or inline-title undo changes an already completed Task's completion timestamp to now.

## Acceptance

Preserve done_at when Done does not transition. Respect the accepted undo exclusion of timestamps; changing that history contract is outside this ticket. Cover both workflow save and undo paths.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../../.scratch/data-integrity-audit/report.md) and `TestIntegrityAuditCompletionTimestampPreservation` in ordinary regression coverage.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-05 — Implementation, review and validation

The Detail-save reproduction passes since F05. The inline-title Undo scenario
failed twice because restore always regenerated done_at for a completed Task.
Direct Done and workflow writes also rewrote unchanged completion metadata.

All three writes now use a SQL conditional to preserve done_at exactly when
Done does not transition. This retains NULL and raw encoding without parsing.
Real completion transitions set a current timestamp; reopening clears it.
Completion timestamps remain excluded from history snapshots and conflicts:
external timestamp-only edits survive unrelated Undo/Redo, while redo of a real
completion creates new metadata rather than restoring an old timestamp.

The original reproduction runs in the ordinary suite. Review checked normal
and combined saves, repeated Done requests, actual Bucket moves with unchanged
Done, precision/offset/NULL preservation, Undo/Redo transitions, external edits
and late Undo failure rollback. No scoped finding remains. Focused tests pass
three repetitions; the isolated staged tree passes full race tests, corpus
verification, both builds and vet, plus opt-in audit safety controls. No user
database access, schema/dependency change, manual or native Windows acceptance.
F09–F11 remain open.
