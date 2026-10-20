# 09 — Protect recurring Tasks from incompatible completion writes

**Finding:** F09

**Category:** bug

**Priority:** Medium

**Status:** completed

## Problem

Completing an existing recurring Task leaves it Done without advancing dates or reopening its next occurrence.

## Acceptance

Decide whether recurrence is implemented or unsafe completion is rejected explicitly. Include repeat modes, reminders, dates and checklist semantics in that decision. Do not silently freeze existing recurrence.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../2026-10-20/data-integrity-audit/report.md) and `TestIntegrityAuditRecurringCompletion` in ordinary regression coverage.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-05 — Implementation, review and validation

F09 rejects incomplete-to-completed transitions for recurring Tasks, including interval, monthly, from-current-date and unknown nonzero configurations. Vikunja owns date advancement, reminder rescheduling and checklist resets; Vimkunja does not reproduce those semantics. The guard covers Done, workflow, draft/Comment and Undo writes transactionally. Reopening and unrelated edits remain allowed. Historical schemas without recurrence columns remain supported. Completion metadata and the history contract are unchanged.

Focused regressions and the complete race-enabled Go suite pass. Both builds and vet pass. Review found no remaining scoped blocker. No user database, schema or dependency changes; no manual, native Windows or live server acceptance.
