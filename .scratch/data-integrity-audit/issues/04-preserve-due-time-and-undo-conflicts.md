# 04 — Preserve full due timestamps and detect time conflicts

**Finding:** F04

**Category:** bug

**Priority:** High

**Status:** ready-for-agent

## Problem

Title-only Detail saves and inline-title undo replace due time with midnight; an external same-day time change is not detected by undo.

## Acceptance

Keep date-only input while retaining exact untouched persisted timestamps. Include full due time in optimistic undo comparisons. Explicit date edits retain their existing semantics. Test both save and undo preservation.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../report.md) and `TestIntegrityAuditDueTimePreservation` in the audit tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.
