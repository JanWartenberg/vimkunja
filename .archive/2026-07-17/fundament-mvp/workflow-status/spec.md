# Workflow status in Task Detail

**Status:** completed — manual smoke test verified

## Problem Statement

Vimkunja currently shows only a Task's global open/done state in List View. Opening Task Detail does not show or allow editing workflow states such as To-Do, Doing, or Done used by Vikunja. This omits an essential original-application feature even though Kanban Buckets and Task/Bucket assignments already exist in the database.

The binary Done toggle and Bucket assignment can also diverge: completing a Task moves it to configured Done Buckets, but reopening currently leaves it there. A later Kanban board would need to fix this again unless workflow status and Done are modeled together first.

## Solution

Task Detail displays an editable workflow status. Status corresponds to the Task's current Bucket in its Project's manual Kanban View and is selected through a picker of existing Buckets. Like other Detail fields, changes participate in existing draft, dirty, save, and discard behavior.

Status logic keeps Bucket assignment and global Done consistent: the configured Done Bucket means completed; other Buckets mean open. Existing `x` toggle uses the same central workflow operation. This establishes a shared domain foundation before Kanban without implementing board rendering/navigation yet.

Status is fully usable in the usual current configuration of exactly one manual Kanban View per Project. With none or multiple, Vimkunja makes no silent, potentially incorrect choice; it shows clear read-only feedback. Multiple-View semantics will be decided separately with Kanban.

## User Stories

1. As a Vimkunja user, I want to see a task's workflow status in the Detail-pane, so that I know whether it is To-Do, Doing, Done, or in another configured bucket.
2. As a Vimkunja user, I want the displayed status to use my existing Vikunja bucket titles, so that Vimkunja and the Original App use the same vocabulary.
3. As a Vimkunja user, I want to focus the Status field with the same vertical field navigation as the other Detail fields, so that editing remains predictable.
4. As a Vimkunja user, I want to open the Status picker with `i`, so that it follows the existing picker interaction.
5. As a Vimkunja user, I want to navigate available statuses with `j`/`k` and the arrow keys, so that I can select a status without leaving the keyboard.
6. As a Vimkunja user, I want `Enter` to confirm the highlighted status, so that the picker behaves like the existing Project and Priority pickers.
7. As a Vimkunja user, I want `Esc` to cancel the picker and preserve the previous status, so that I can safely back out of a selection.
8. As a Vimkunja user, I want a changed status to mark the Detail editor dirty, so that unsaved workflow changes cannot be lost silently.
9. As a Vimkunja user, I want `:w` to save a changed status and keep the Detail-pane open, so that status editing follows the existing save workflow.
10. As a Vimkunja user, I want `:wq` and `:wqa` to include a changed status, so that all save commands have consistent semantics.
11. As a Vimkunja user, I want `:q` to refuse to close a Detail-pane with an unsaved status change, so that the existing dirty protection also protects workflow state.
12. As a Vimkunja user, I want `:q!` to discard an unsaved status change, so that I can restore the persisted workflow state explicitly.
13. As a Vimkunja user, I want selecting the configured Done status to mark the task globally done, so that list, task detail, Kanban views, and the Original App agree.
14. As a Vimkunja user, I want selecting any non-Done status to mark the task globally open, so that moving a task back into active work reopens it.
15. As a Vimkunja user, I want `x` on an open task to select the configured Done status, so that the shortcut and Status picker cannot produce conflicting state.
16. As a Vimkunja user, I want `x` on a done task to return it to the configured default status, so that reopening does not leave it inside the Done bucket.
17. As a Vimkunja user, I want successful status writes to refresh the current task list and editor state, so that the TUI immediately reflects persisted data.
18. As a Vimkunja user, I want failed status writes to retain my editor state and show the existing error feedback, so that I can retry without re-entering the change.
19. As a Vimkunja user, I want status changes to update Vikunja-compatible timestamps, so that external Vikunja behavior can observe the write correctly.
20. As a Vimkunja user, I want a new task to begin in the configured default bucket, so that its initial displayed status matches the Original App and persisted assignment.
21. As a Vimkunja user, I want moving a task to another project to use that project's default workflow status, so that it never keeps an invalid bucket from the old project.
22. As a Vimkunja user, I want a clear read-only message when a project has no manual Kanban View, so that absence of a workflow status is not mistaken for missing data.
23. As a Vimkunja user, I want a clear read-only message when a project has multiple manual Kanban Views, so that Vimkunja does not silently edit the wrong board.
24. As a Vimkunja user, I want custom bucket names and ordering preserved in the picker, so that the workflow reflects my project configuration rather than hard-coded To-Do/Doing/Done labels.
25. As a developer, I want Detail editing and the future Kanbanboard to use one workflow-status abstraction, so that bucket and Done semantics are implemented only once.
26. As a developer, I want status writes to be transactional, so that bucket assignment, Done state, Done timestamp, and updated timestamp cannot partially diverge.
27. As a developer, I want the existing SQLite schema to remain unchanged, so that Vimkunja stays compatible with the Original App.

## Implementation Decisions

- A Workflow Status is a Bucket assignment in one manual Kanban View; it is not a new string or enum stored on a Task.
- The existing Vikunja tables and columns remain the source of truth. No database migration or schema change is introduced.
- A project is editable through this feature only when it has exactly one manual Kanban View. Its Buckets, ordered by their persisted position and stable identifier, form the picker choices.
- Projects without a manual Kanban View show Status as unavailable and read-only.
- Projects with multiple manual Kanban Views show Status as ambiguous and read-only. Choosing a View or editing multiple View-specific statuses is deferred until the Kanban feature defines that interaction.
- The configured Done Bucket determines Done semantics; bucket titles such as `Done` have no special hard-coded meaning.
- Selecting the Done Bucket sets the Task's global Done flag, sets its Done timestamp, updates its modification timestamp, and persists the selected Bucket assignment atomically.
- Selecting a non-Done Bucket clears the Task's global Done flag and Done timestamp, updates its modification timestamp, and persists the selected Bucket assignment atomically.
- The Store exposes workflow-oriented reads and writes that hide Bucket SQL, Done synchronization, and transaction details from the TUI. The future Kanbanboard must reuse this interface rather than issue separate movement writes.
- The Status field is part of the shared Task editor and its dirty comparison. Status selection changes the editor draft but does not write immediately.
- Saving an existing Task persists ordinary editable fields and the selected Workflow Status as one user operation. A failure leaves the editor dirty and uses the existing persistent error log and footer feedback.
- New Tasks continue to receive the target project's configured default Bucket. After creation, the editor reflects that persisted Workflow Status.
- Moving a Task to another Project removes obsolete assignments and initializes the target Project's default Bucket as already established by current Store behavior. A previously selected source-project Status is not carried across Projects.
- The Status field is placed with the other compact metadata fields before Description and participates in normal vertical field navigation without wrap-around.
- The Status picker follows the established Project/Priority picker conventions: `i` opens, `j`/`k` and arrows navigate, `Enter` confirms, and `Esc` cancels.
- `x` remains an immediate Tasks-pane shortcut but delegates to the same workflow transition semantics as the editor. Open Tasks move to each applicable configured Done Bucket; done Tasks move to each applicable configured Default Bucket and become open.
- The reopening behavior intentionally replaces the current behavior that clears Done while leaving the Task in a Done Bucket.
- Successful writes retain the established auto-refresh behavior. Failed writes do not reload stale state over the user's current editor.
- The List View continues to show `[ ]` and `[x]`. Adding a Workflow Status column or badge to every task row is not required by this slice; the status is visible in the opened Detail-pane.

## Testing Decisions

- Good tests assert externally observable behavior and persisted outcomes, not SQL text, private helper calls, picker implementation details, or rendering escape sequences.
- The primary seam is the existing Store tested against a real temporary SQLite database. Store tests verify the persisted Bucket assignment, global Done flag, Done timestamp, updated timestamp, default-bucket reopening behavior, transactionality, custom Bucket names/order, and no schema changes.
- Existing Store tests for Done toggling, Done-Bucket synchronization, Task creation defaults, and Project moves are prior art. Tests whose old expectation leaves a reopened Task in a Done Bucket must be updated to the new coherent workflow behavior.
- The secondary seam is the existing TUI model with its fake Store. Model tests verify field navigation, picker confirmation/cancellation, dirty protection, save/discard behavior, successful refresh, and write-error retention without asserting Lip Gloss output.
- Existing TUI model tests for the Project and Priority pickers, Detail dirty protection, save commands, task toggling, and write failure feedback are prior art.
- Store-level behavior carries most of the semantic coverage. TUI model tests remain limited to interaction orchestration that cannot be proven at the Store seam.
- At minimum, acceptance coverage includes: reading the current Bucket; selecting Done; selecting a non-Done Bucket; reopening with `x` into the Default Bucket; preserving custom Bucket order; handling no Kanban View; rejecting ambiguity from multiple manual Kanban Views; saving and discarding editor changes; and retaining dirty state after a failed write.
- A manual smoke check verifies the Detail Status field against the same task in the Original App for To-Do, Doing, Done, and `x` reopen transitions.

## Out of Scope

- Rendering the Kanban View or adding Bucket columns.
- Two-dimensional Kanban navigation and moving cards directly on a board.
- Creating, renaming, ordering, or deleting Buckets.
- Selecting among multiple manual Kanban Views or editing one status per View.
- Filter-based or generated Bucket configurations whose choices are not represented by ordinary persisted manual Buckets.
- Showing Workflow Status as a column or badge in every List View row.
- Configurable status colors, icons, or title-based status semantics.
- Changes to the Vikunja database schema.
- REST API support.

## Further Notes

- This feature is intentionally prioritized ahead of the currently open Fundament acceptance verification and the previously planned Kanban sequence.
- The implementation is a small Kanban data-semantics foundation, not an early Kanban UI.
- The current database has exactly one Kanban View per inspected Project, with To-Do as Default Bucket and Done as Done Bucket, so the non-ambiguous path covers the present installation.
- The terms Pane, View, Focus, and Selection retain their established meanings: Detail is a Pane; List and Kanban are Views; Focus identifies the active Pane; Selection identifies an item or editor field.
