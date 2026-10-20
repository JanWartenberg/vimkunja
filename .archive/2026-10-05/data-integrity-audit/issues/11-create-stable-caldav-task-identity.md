# 11 — Create stable CalDAV identity for new Tasks

**Finding:** F11

**Category:** bug

**Priority:** Medium

**Status:** completed

## Problem

CreateTask leaves the uid column NULL while local Vikunja creates a UUID.

## Acceptance

Generate a stable valid UID for creation where the supported schema includes uid, and preserve it across edits/moves/undo. Test uniqueness and the supported-schema boundary; live CalDAV client behavior is not yet verified.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../../../2026-10-20/data-integrity-audit/report.md) and `TestIntegrityAuditCalDAVIdentity` in ordinary regression coverage.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.

### 2026-10-05 — Implementation, review and validation

F11 assigns a cryptographically random RFC 4122 UUID v4 during Task insertion when the uid column exists. UID is supplied in the initial INSERT, including NOT NULL/unique schemas. Historical schemas without uid remain supported. Existing Task identities are not backfilled or rewritten. Ordinary regression coverage verifies UUID format and uniqueness, edits, completion, Project moves, Undo/Redo and late creation failure rollback. Live CalDAV client interoperability remains unverified.

Focused regressions and the complete race-enabled Go suite pass. Both builds and vet pass. Review found no remaining scoped blocker. No user database, schema or dependency changes; no manual, native Windows or live server acceptance.

Review also aligned optional-column detection with SQLite case-insensitive identifiers, covered by an uppercase UID schema regression. No migration or backfill is introduced.
