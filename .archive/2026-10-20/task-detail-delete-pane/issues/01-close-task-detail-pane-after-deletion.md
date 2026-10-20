# 01 — Close Task Detail Pane after deleting a Task

**What to fix:** In the List View, when a Task is opened in the Task Detail Pane and then deleted, the Detail Pane remains open showing the deleted Task.

**Expected behavior:** After deletion succeeds, clear the deleted Task from the Task Detail Pane and close or reset the pane so it no longer displays stale Task data.

**Category:** bug

**Priority:** High

**Status:** completed

- [x] Reproduce deletion from List View with Task Detail open.
- [x] Clear the deleted Task's selection and Task Detail state after successful deletion.
- [x] Add regression coverage ensuring deleted Task data is not rendered in the Detail Pane.

## Closure reconciliation — 2026-10-20

The implementation scope is complete and its regression coverage passes in the
current main working tree. Archived during the release subtask cleanup. This
records technical completion and preserves existing manual acceptance evidence;
it does not claim a new physical Windows run or acceptance of unrelated future
features. Remaining release checks are tracked in the active cleanup/publication
tickets.
