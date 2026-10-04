# 10 — Clean polymorphic relations on permanent Task deletion

**Finding:** F10

**Category:** bug

**Priority:** Medium

**Status:** ready-for-agent

## Problem

Permanent deletion leaves Task favorites, subscriptions and Task/Comment reactions dangling.

## Acceptance

Clean supported polymorphic relations transactionally before deleting the Task and its Comments, preserve unrelated entity kinds and IDs, tolerate absent historical tables, and retain attachment files. Do not change the separately accepted hard-deletion contract.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../report.md) and `TestIntegrityAuditTaskDeletionCleansPolymorphicRelations` in the audit tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.
