# 01 — Prevent duplicate Detail saves

**Finding:** F01

**Category:** bug

**Priority:** High

**Status:** completed

## Problem

A second :w before the first save completes schedules another creation with ID zero and creates two Tasks.

## Acceptance

Reject additional Detail saves while one is in flight, including :wq/:wqa and direct save dispatch. Preserve the first save's close/quit intent. Release the guard on success and failure so retries work.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../../.scratch/data-integrity-audit/report.md) and `TestIntegrityAuditRepeatedWriteDoesNotDuplicateTask` in the ordinary TUI tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-04 — Implemented and verified

Confirmed the missing in-flight dispatch guard with real SQLite: two queued :w
commands created two Tasks. Detail saves now own a dedicated editor guard,
independent of Task-load state, and additional key input/save dispatch is blocked
until completion. The first save retains its close/quit intent. Success and
failure both release the guard; Ctrl-C remains the explicit emergency exit.

The reproduction is now part of ordinary regression coverage. Tests also cover
:wq/:wqa, discard/new commands, background Task-load completion and error/retry.
Full `go test -race -count=1 ./...`, build and vet passed. No commit or user
database access. Native Windows acceptance was not run.

### 2026-10-04 — Maintainer acceptance and closure

Jan confirmed repeated saves create no duplicate, edited titles survive closing
and reopening Detail, and :wq/:wqa pass in his normal database workflow. The
focused Go regression command passed all three repetitions in his environment.
The ticket remains completed and archived. The guided tutorial is unsuitable
for free-form acceptance because it restricts permitted keystrokes.
