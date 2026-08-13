# 01 — Clarify Kanban focus and Task Detail selection synchronization

**What to build:** In a Kanban View, focused and unfocused regions are visually distinguishable at a glance, and the Task Detail Pane follows the selected Task while navigating Bucket cards.

**Implementation prerequisites:** None.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

- [ ] Define the visual focus treatment for the Tasks Pane's active Bucket/cards and the Task Detail Pane; the unfocused region must remain legible but visibly subdued.
- [ ] Synchronize the Task Detail Pane's selected Task as card selection changes through a Bucket, including empty Buckets and focus transitions.
- [ ] Preserve existing Kanban navigation, Task opening, scrolling, and narrow-terminal rendering behavior.
- [ ] Add model-level rendering and selection-synchronization regression coverage without changing Vikunja persistence or schema.

## Further context

Captured from the prior temporary `TODO`: "highlight either TaskDetail Or Bucket" and update the selected Task Detail while selecting cards in Buckets.
