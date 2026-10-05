# 10 — Clean polymorphic relations on permanent Task deletion

**Finding:** F10

**Category:** bug

**Priority:** Medium

**Status:** completed

## Problem

Permanent deletion leaves Task favorites, subscriptions and Task/Comment reactions dangling.

## Acceptance

Clean supported polymorphic relations transactionally before deleting the Task and its Comments, preserve unrelated entity kinds and IDs, tolerate absent historical tables, and retain attachment files. Do not change the separately accepted hard-deletion contract.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../../.scratch/data-integrity-audit/report.md) and `TestIntegrityAuditTaskDeletionCleansPolymorphicRelations` in ordinary regression coverage.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-05 — Implementation, review and validation

F10 removes Task favorites (kind 1), Task subscriptions (type 3), Task reactions (kind 0) and reactions on the deleted Tasks Comments (kind 1) before deleting Comments. Cleanup shares the existing deletion transaction and preserves matching IDs of other entity kinds, unrelated Tasks and attachment files. Absent historical tables are tolerated; errors in present tables abort and roll back. Ordinary regressions cover original audit evidence, same-ID controls, historical table combinations and a late Task deletion failure.

Focused regressions and the complete race-enabled Go suite pass. Both builds and vet pass. Review found no remaining scoped blocker. No user database, schema or dependency changes; no manual, native Windows or live server acceptance.
