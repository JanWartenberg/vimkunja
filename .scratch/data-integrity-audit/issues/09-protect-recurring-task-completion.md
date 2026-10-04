# 09 — Protect recurring Tasks from incompatible completion writes

**Finding:** F09

**Category:** bug

**Priority:** Medium

**Status:** needs-triage

## Problem

Completing an existing recurring Task leaves it Done without advancing dates or reopening its next occurrence.

## Acceptance

Decide whether recurrence is implemented or unsafe completion is rejected explicitly. Include repeat modes, reminders, dates and checklist semantics in that decision. Do not silently freeze existing recurrence.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../report.md) and `TestIntegrityAuditRecurringCompletion` in the audit tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.
