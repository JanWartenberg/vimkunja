# 01 — Add View and Global Search navigation history

**What to decide:** Define browser-like backward and forward navigation across activated Views and Global Search results using `Ctrl+O` and `Ctrl+I`.

**Implementation prerequisites:** Requires a precise history-entry and restoration contract. The existing one-step Global Search return snapshot should be generalized only after that contract is accepted.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

## Problem

Vimkunja currently preserves a one-step return path when Global Search activates a result. As navigation expands across Projects, Views, Label-filtered Overviews, and Search, a user needs a consistent way to return to prior working contexts and move forward again without manually reconstructing them.

## Direction so far

- `Ctrl+O` navigates backward; `Ctrl+I` navigates forward, following familiar Vim semantics.
- History records semantic navigation contexts, not rendered Task lists.
- Candidate contexts include the active Project and View, Overview scope and filters, selected Task, focused Pane, and a Global Search query plus its selected result.
- A new navigation after moving backward discards forward history.
- Transient interaction states—Task Editor drafts, Label Pickers, confirmation dialogs, and other modals—do not become history entries.
- Existing Global Search return behavior is the first case to migrate to the general mechanism.

## Questions to decide

- Which changes are meaningful navigation events: Project selection, View changes, entering Task Detail, Label-filtered Overview activation, and/or focus changes between Panes?
- Which selection and scroll positions must be restored, and which are intentionally excluded?
- When asynchronous loading fails or underlying data changed, what safe partial restoration and feedback should occur?
- Does closing Global Search return to its origin without changing history, or constitute a history traversal?
- How should dirty editors and other unsaved state behave when a history command would leave their context?
- What bounded history size and entry coalescing rules keep the behavior predictable?

## Compatibility boundary

This is TUI-local navigation state only. It must neither change the Vikunja SQLite schema nor write navigation history to the database.
