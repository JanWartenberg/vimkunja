# 06 — Use destination Done Buckets for completed Task moves

**Finding:** F06

**Category:** bug

**Priority:** Medium

**Status:** ready-for-agent

## Problem

A completed Task moved to another Project remains Done but lands in its To-Do Bucket.

## Acceptance

Initialize destination View assignments according to persisted Done. Preserve the accepted single/multiple View contracts and transactional rollback.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../report.md) and `TestIntegrityAuditCompletedProjectMove` in the audit tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.
