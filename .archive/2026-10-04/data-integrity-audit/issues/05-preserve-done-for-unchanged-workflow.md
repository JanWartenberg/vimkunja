# 05 — Preserve Done when Workflow Status was not edited

**Finding:** F05

**Category:** bug

**Priority:** Medium

**Status:** completed

## Problem

Saving only a title on a completed Task without a configured Done Bucket clears Done because the loaded current Bucket is reapplied.

## Acceptance

Distinguish loaded Workflow Status from an actual user change. Unrelated edits preserve Done and completion metadata; explicit workflow changes keep accepted semantics.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../../.scratch/data-integrity-audit/report.md) and `TestIntegrityAuditUnchangedWorkflowPreservesDone` in the ordinary Store tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-04 — Implementation, review and validation

Root cause: existing Detail and combined Comment saves submitted the loaded
Bucket as a workflow mutation. Reapplying it derived Done from the configured
Done Bucket, clearing Done for completed Tasks with no such configuration.
The original audit scenario failed twice and was promoted to ordinary coverage.

A shared editor saveDraft snapshot omits an unchanged loaded Bucket for existing
Tasks. New Task creation and explicit Bucket changes retain their IDs; DoneSet
remains independent. Both Detail and combined Comment saves use this snapshot.
The Store also detects a valid unchanged assignment inside the transaction and
preserves Done/done_at instead of reapplying workflow semantics. The query checks
Bucket/View ownership and manual-View configuration, so stale foreign assignments
cannot bypass validation. Explicit workflow APIs retain their existing behavior.

Follow-up review checked creation, explicit Bucket/Done changes, failure paths,
multiple Views and asynchronous save completion. The additional stale-editor
case confirms unrelated saves do not reapply an externally changed Bucket.
Invalid Bucket writes roll back, label snapshots do not alias editor state, and
successful completion keeps the editor baseline clean. No unresolved blocker
was found in the scoped diff.

Focused Store/TUI regressions pass three repetitions. The isolated staged tree
passes full race-enabled tests (including corpus verification), both entrypoint
builds and vet; opt-in audit safety controls pass. No schema/dependency changes
or user database access; no manual or native Windows acceptance is claimed.
F08 still tracks completion-time rewriting during Undo: avoiding unchanged
workflow writes also removes its ordinary Detail-save trigger, but does not
repair that remaining Undo path. Other audit findings remain open.
