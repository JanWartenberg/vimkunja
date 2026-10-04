# 07 — Handle zero and NULL Bucket configuration consistently

**Finding:** F07

**Category:** bug

**Priority:** Medium

**Status:** ready-for-agent

## Problem

Zero default Bucket IDs create invisible Kanban Tasks; zero Done Bucket IDs replace valid current assignments with zero.

## Acceptance

Treat zero and NULL as unconfigured. Resolve an unconfigured default to the first valid Bucket using Vikunja ordering; do not move a Task when no Done Bucket exists. Cover creation, Project moves, completion and reopening.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../report.md) and `TestIntegrityAuditZeroBucketConfiguration` in the audit tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.
