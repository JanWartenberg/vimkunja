# 01 — Specify workflow behavior for multiple manual Kanban Views

**What to decide:** Define how Vimkunja handles a Project with multiple manual Kanban Views when showing or editing workflow status and rendering a board.

**Implementation prerequisites:** Requires Kanban interaction decisions.

**Category:** enhancement

**Priority:** Medium

**Status:** completed

## Guiding constraints

- Preserve compatibility with the original Vikunja interface and data model. Vimkunja must display Projects with multiple manual Kanban Views without silently merging, rewriting, or discarding View-specific data.
- Projects with exactly one manual Kanban View must remain visually and behaviorally unchanged. Do not add permanent View indicators, extra buttons, or other display overhead for the common case.
- Add explicit View selection only when a Project actually has multiple manual Kanban Views. Keep the multi-View interaction minimal and compatible with the existing Kanban entry point and keyboard workflow.
- The expected workflow is normally one Kanban View per Project; multi-View support is primarily a safe compatibility and display requirement, not an elaborate multi-board workflow.
- Do not silently guess, merge, or synchronize multiple Views when their data or workflow meaning differs.

## Decisions confirmed

- [x] Select the active View with a picker only when a Project has multiple manual Kanban Views; the single-View path remains unchanged.
- [x] The selected View determines the Workflow Status shown and edited in Task Detail; the status is View-scoped, not a second global Task state.
- [x] Task Detail does not show every View's bucket as one combined status. When opened from a selected Kanban View it shows that View's status; the active View identity should remain discoverable in the multi-View workflow.
- [x] Selecting a View does not rewrite global Task Done state. Board moves and board-local `x` transitions update Done according to the selected View's configured Done Bucket.
- [x] Board reads, navigation, and writes target the selected View only; assignments and positions in other Views are not modified.

## Implementation

- Added explicit View selection for ambiguous manual Kanban Projects.
- Added optional View-scoped Kanban board/workflow operations while preserving the existing project-scoped single-View calls.
- Scoped Kanban card moves and `x` Done transitions to the selected View.
- Added regression coverage for explicitly selecting a Workflow View.
- `go test ./...` passes.

## Acceptance note

- The implementation is intentionally accepted as a compatibility slice, not as a polished or especially practical multi-Board workflow. The current experience is not considered particularly attractive or convenient; further UX investment is explicitly out of scope for this ticket.

## Further context

Archived source: `workflow-status/spec.md`, ambiguity explicitly deferred to the Kanban feature.
