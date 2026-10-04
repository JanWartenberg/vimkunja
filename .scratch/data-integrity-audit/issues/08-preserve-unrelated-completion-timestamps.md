# 08 — Preserve completion timestamps during unrelated edits

**Finding:** F08

**Category:** bug

**Priority:** Medium

**Status:** ready-for-agent

## Problem

A title-only Detail save or inline-title undo changes an already completed Task's completion timestamp to now.

## Acceptance

Preserve done_at when Done does not transition. Respect the accepted undo exclusion of timestamps; changing that history contract is outside this ticket. Cover both workflow save and undo paths.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../report.md) and `TestIntegrityAuditCompletionTimestampPreservation` in the audit tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.
