# 11 — Create stable CalDAV identity for new Tasks

**Finding:** F11

**Category:** bug

**Priority:** Medium

**Status:** ready-for-agent

## Problem

CreateTask leaves the uid column NULL while local Vikunja creates a UUID.

## Acceptance

Generate a stable valid UID for creation where the supported schema includes uid, and preserve it across edits/moves/undo. Test uniqueness and the supported-schema boundary; live CalDAV client behavior is not yet verified.

- Add the focused reproduction to ordinary regression coverage when repaired.
- Preserve existing work, SQLite schema and user databases; use temporary test databases.
- Run affected tests and required Go checks. Do not repair other audit findings implicitly.

## Evidence

See [audit report](../report.md) and `TestIntegrityAuditCalDAVIdentity` in the audit tests.

## Comments

### 2026-10-04 — Created from integrity audit

The maintainer requested one Scratch ticket per finding and implementation of the save errors first. F01/F02 are one coordinated save-lifecycle repair; the remaining runtime changes are not started.
