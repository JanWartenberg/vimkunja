# 01 — Toggle Task Done from the Task Detail Pane

**What to build:** A user can navigate to the Done field in the Task Detail Pane with `j`/`k` and toggle the selected Task's Done state with `x`.

**Blocked by:** None — this supersedes the archived Fundament/MVP decision that Done is read-only outside the field focus graph.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

- [ ] Include Done directly after Workflow Status and before Description in the Task Detail Pane's normal field-navigation order, matching its visual position.
- [ ] Render a focused Done field without weakening the existing Project, Priority, and Workflow Status interactions.
- [ ] While Done is focused, `x` toggles the enclosing Task's Done state through Vikunja-compatible persistence.
- [ ] Refresh visible Task state after success and show persistence failures through the existing footer/session error-log path.
- [ ] Add TUI and Store regression coverage.
