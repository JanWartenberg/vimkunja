# 05 — Preserve Done when Workflow Status was not edited

**Finding:** F05

**Category:** bug

**Priority:** Medium

**Status:** ready-for-agent

## Problem

Saving only a title on a completed Task without a configured Done Bucket clears Done because the loaded current Bucket is reapplied.

## Acceptance

Distinguish loaded Workflow Status from an actual user change. Unrelated edits preserve Done and completion metadata; explicit workflow changes keep accepted semantics.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../report.md) and `TestIntegrityAuditUnchangedWorkflowPreservesDone` in the audit tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.
