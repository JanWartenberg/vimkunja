# 03 — Validate target Project before moving a Task

**Finding:** F03

**Category:** bug

**Priority:** High

**Status:** completed

## Problem

Moving to nonexistent Project 999 succeeds, loses source View assignments, and makes the next undo snapshot fail.

## Acceptance

Validate the destination inside the write transaction before removing any source assignment. Ordinary and Task-plus-Comment saves must fail without changing the Task or its relations.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../../.scratch/data-integrity-audit/report.md) and `TestIntegrityAuditMissingTargetProject` in ordinary Store regression coverage.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-04 — Started

After accepting and closing the save fixes, Jan requested the next ticket.
F03 is now being implemented; the other audit findings remain outside this change.

### 2026-10-04 — Implemented and verified

The original reproduction failed twice. Four expanded Store scenarios also
failed before repair: ordinary and Task-plus-Comment saves, each targeting a
never-existing Project or a Project deleted after draft selection. The shared
updateTaskInTx boundary now verifies target existence inside the transaction
before changing fields or removing source assignments. No schema change is needed.

Regression tests verify Task fields, Labels, Buckets, source position and existing
Comments remain unchanged. A real SQLite/TUI test verifies :wqa keeps the dirty
draft open, reports the missing Project, does not quit or add an Undo Entry, and
releases its save guard. Existing valid Project moves and Comment rollback tests
still pass. The targeted Store tests passed three repetitions; full
`go test -race -count=1 ./...`, build and vet passed. No commit or user database
access. This is automated closure; no manual acceptance of F03 is claimed.
