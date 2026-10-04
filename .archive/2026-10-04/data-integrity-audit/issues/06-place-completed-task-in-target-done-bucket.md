# 06 — Use destination Done Buckets for completed Task moves

**Finding:** F06

**Category:** bug

**Priority:** Medium

**Status:** completed

## Problem

A completed Task moved to another Project remains Done but lands in its To-Do Bucket.

## Acceptance

Initialize destination View assignments according to persisted Done. Preserve the accepted single/multiple View contracts and transactional rollback.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../../.scratch/data-integrity-audit/report.md) and `TestIntegrityAuditCompletedProjectMove` in the ordinary Store tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-04 — Implementation, review and validation

Root cause: initializeViewAssignments always used destination default Buckets,
ignoring the existing Task's persisted Done state. The original reproduction
failed twice, then was promoted to ordinary regression coverage.

Project moves now pass persisted Done into the shared initializer. Each
completed Task receives the destination View's positive configured Done Bucket;
zero/NULL Done configuration retains that View's default assignment. No default
Bucket still means no assignment, with a position retained. Unfinished Tasks and
creation retain their existing initialization behavior. Done and done_at are
not rewritten by a pure move; an explicit DoneSet change still applies afterward.

Review addressed two concrete risks. Creation/unfinished initialization avoids
requiring Done configuration unnecessarily, preserving existing minimal-schema
save fixtures. Selected destination Done Buckets are validated against their
own View, so missing/foreign Buckets reject the move and roll back all changes.
No open blocker remains in the scoped diff.

Focused Store/TUI checks pass three repetitions: persisted Done versus draft
metadata, multiple Views, NULL/zero fallback, Done-only configuration, unrelated
Tasks, exact completion timestamps, explicit Done overrides, missing/foreign
Done Buckets and injected Bucket/Comment failures. The TUI test drives the
Project picker, normal/combined saves and reopened Detail workflow loading.
The isolated staged tree passes full race tests (including corpus verification),
both builds and vet; opt-in audit safety controls pass. No user database access,
schema/dependency changes, manual acceptance or native Windows verification.
F07's default-zero and Done-toggle issues and F08's Undo timestamps remain open.
