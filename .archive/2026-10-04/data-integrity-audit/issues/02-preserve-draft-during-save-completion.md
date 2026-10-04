# 02 — Preserve unsaved edits at Detail save completion

**Finding:** F02

**Category:** bug

**Priority:** High

**Status:** completed

## Problem

A save dispatched with title New completes after the live draft changes to XNew; XNew is marked clean although only New was stored.

## Acceptance

Prevent input from changing the Detail session during its save; associate completion with the originating editor and saved draft. Late asynchronous draft changes must remain dirty, and stale completions must not change a replacement editor. A failed save retains its draft.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../../.scratch/data-integrity-audit/report.md) and `TestIntegrityAuditEditDuringSaveRemainsDirty` in the ordinary TUI tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-04 — Implemented and verified

Confirmed that completion promoted the live draft to the saved baseline. Each
save now carries its originating editor and a cloned submitted draft, including
independent Label IDs. Keyboard input pauses while saving; completion confirms
the submitted state, retains later asynchronous draft changes as dirty, and
refuses close/quit when those changes remain. A replaced editor is never rewritten
or closed by an old completion. Late Label loads update only their own baseline
instead of marking unrelated draft content saved.

Original SQLite reproduction and additional tests for delayed responses,
mutable Label slices, late Label loads, saved close/quit intent and failure/retry
pass in ordinary regression coverage. Full `go test -race -count=1 ./...`, build
and vet passed. No commit or user database access. Native Windows acceptance was
not run.

### 2026-10-04 — Maintainer acceptance and closure

Jan confirmed title changes persist after closing/reopening Detail, repeated
saves create no duplicates, and :wq/:wqa pass. The targeted save-lifecycle tests
passed three times in his environment. The ticket remains completed and archived.
Manual tests used his normal database workflow, not the restricted tutorial.
