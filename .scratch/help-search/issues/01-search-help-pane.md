# 01 — Search the Help Pane

**Status:** needs-triage
**Category:** enhancement
**Priority:** Low
**Milestone:** 3

## Goal

Jan requires searchable Help as a follow-up to [pane-local search](../../../.archive/2026-11-23/local-search/spec.md). The initial implementation deliberately excludes Help and must not search a background Pane when Help is focused.

- [ ] Specify whether search covers the current help topic or all topics, including topic navigation and cancellation restoration.
- [ ] Reuse `/`, `?`, `n`/`N`, shared patterns/history, highlighting and `Space h`/`:nohlsearch` where appropriate.
- [ ] Preserve help-local navigation history, scroll positions and any protected Task/Comment draft.
- [ ] Add focused help navigation/layout regressions and update controls.
