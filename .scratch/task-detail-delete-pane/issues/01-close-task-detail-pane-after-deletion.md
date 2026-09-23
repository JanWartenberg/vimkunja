# 01 — Close Task Detail Pane after deleting a Task

**What to fix:** In the List View, when a Task is opened in the Task Detail Pane and then deleted, the Detail Pane remains open showing the deleted Task.

**Expected behavior:** After deletion succeeds, clear the deleted Task from the Task Detail Pane and close or reset the pane so it no longer displays stale Task data.

**Category:** bug

**Priority:** High

**Status:** implemented

- [x] Reproduce deletion from List View with Task Detail open.
- [x] Clear the deleted Task's selection and Task Detail state after successful deletion.
- [x] Add regression coverage ensuring deleted Task data is not rendered in the Detail Pane.
