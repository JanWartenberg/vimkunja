# 07 — Handle zero and NULL Bucket configuration consistently

**Finding:** F07

**Category:** bug

**Priority:** Medium

**Status:** completed

## Problem

Zero default Bucket IDs create invisible Kanban Tasks; zero Done Bucket IDs replace valid current assignments with zero.

## Acceptance

Treat zero and NULL as unconfigured. Resolve an unconfigured default to the first valid Bucket using Vikunja ordering; do not move a Task when no Done Bucket exists. Cover creation, Project moves, completion and reopening.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../2026-10-20/data-integrity-audit/report.md) and `TestIntegrityAuditZeroBucketConfiguration` in ordinary regression coverage.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-04 — Implementation, review and validation

Root cause: non-NULL zero configuration was treated as a real Bucket ID.
Creation assigned Bucket zero and completion replaced a valid assignment with
zero. Both original scenarios failed twice, then moved into the ordinary suite.

Shared default resolution treats zero/NULL as absent, selecting the first
positive View-owned Bucket ordered by position and ID. Creation, completed and
unfinished Project moves and reopening use this fallback. Completion without a
Done Bucket preserves the current assignment; no candidates create no relation.
Configured references are validated transactionally. View configuration is not
rewritten, and selected-View scoping is preserved.

Review aligned board/new-editor/list metadata with the write fallback and fixed
list availability for an unassigned Task with wholly NULL configuration. Result
rows close before additional reads, tested with one connection. Existing test
fixtures now seed their referenced Buckets; the previous Done-only View move
expectation is updated for this ticket's fallback semantics.

Focused checks pass three repetitions, including ordering, no-Bucket cases,
invalid configuration rollback and combined save with a late Comment failure.
The isolated staged tree passes full race tests, corpus verification, both builds
and vet; opt-in safety controls pass. No remaining scoped blocker, user database
access, schema/dependency change, manual acceptance or native Windows testing.
F08–F11 remain open.
