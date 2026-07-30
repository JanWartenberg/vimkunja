# 01 — Toggle Task Done from the Task Detail Pane

**What to build:** A user can navigate to the Done field in the Task Detail Pane with `j`/`k` and toggle the selected Task's Done state with `x`.

**Implementation prerequisites:** None — this supersedes the archived Fundament/MVP decision that Done is read-only outside the field focus graph.

**Category:** enhancement

**Priority:** Medium

**Status:** resolved

- [x] Include Done directly after Workflow Status and before Description in the Task Detail Pane's normal field-navigation order, matching its visual position.
- [x] Render a focused Done field without weakening the existing Project, Priority, and Workflow Status interactions.
- [x] While Done is focused, `x` toggles the enclosing Task's Done state through Vikunja-compatible persistence.
- [x] Refresh visible Task state after success and show persistence failures through the existing footer/session error-log path.
- [x] Add TUI and Store regression coverage.

## Comments

### 2026-07-30 — Resolved

Implemented through `c02f7db` and follow-up TUI fixes. The Done value is an editor draft change, `:w` persists it atomically with all other Detail edits, and the active view refreshes after a successful save. Store and TUI regression coverage pass with `go test ./...`.
