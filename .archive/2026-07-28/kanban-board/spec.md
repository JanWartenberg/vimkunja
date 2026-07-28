# Spec — Single manual Kanban View

**Status:** ready for implementation after standard-Kanban provisioning

## Goal

Render and operate a Project's one unambiguous, manually configured Kanban View in the Tasks Pane. The feature is keyboard-first, preserves Vikunja data semantics, and keeps the existing Projects ↔ Tasks ↔ Task Detail Pane graph unchanged.

This spec is intentionally limited to one manual Kanban View per Project. Standard View provisioning is a separate, preceding high-priority ticket; Bucket administration, multiple Kanban Views, filter-configured Boards, and card reordering are separate work.

## Domain model

- A **Kanban View** is a `project_views` row with `view_kind=kanban`.
- A **manual Kanban View** additionally has `bucket_configuration_mode=manual`. Its persisted Buckets accept direct Task moves.
- A **Bucket** belongs to exactly one Project View. Its title and persisted position are user data.
- A **Kanban selection** is an active Bucket and an optional selected Task. It is valid for the active Bucket to be empty.
- The global **Task Done state** and the view-specific Bucket assignment must agree: the configured Done Bucket denotes done; every other Bucket denotes open.

`ProjectView` must expose the Bucket configuration mode. Selection, workflow reads, and writes must filter on both `view_kind=kanban` and manual configuration; filtering on View kind alone is incorrect.

## Eligibility and entry

A Project has one of these states:

1. **Exactly one manual Kanban View:** it is the active Kanban View and can be selected with `:kanban`.
2. **No manual Kanban View:** `:kanban` offers explicit standard-View provisioning. The operation is defined by [`../kanban-provisioning/issues/01-provision-standard-manual-kanban-view.md`](../kanban-provisioning/issues/01-provision-standard-manual-kanban-view.md).
3. **More than one manual Kanban View:** Vimkunja does not guess. It shows an explicit unsupported/ambiguous error and leaves the current View unchanged. View selection is deferred to the multi-Kanban ticket.

From Overview, `:kanban` first restores the remembered Project context, then applies the same three-state rule. A future `Space+v`, then `k` uses this exact same selection/provisioning path. `:list` and `Space+v`, then `l` select the Project's List View.

## Store contract

The Store owns SQL, transactionality, View eligibility, and Task Done/Bucket synchronization. TUI code does not query or mutate `project_views`, `buckets`, `task_buckets`, or `task_positions` directly.

The Kanban implementation needs a board-oriented read model, conceptually:

```go
type KanbanBoardAvailability int // available, unavailable, ambiguous

type KanbanBucket struct {
    ID       int64
    Title    string
    Position float64
    Tasks    []Task
}

type KanbanBoard struct {
    Availability KanbanBoardAvailability
    View         ProjectView
    Buckets      []KanbanBucket // persisted Bucket position ascending, then ID
}
```

`LoadManualKanbanBoard(ctx, projectID)` returns the single eligible Board and its Buckets with cards grouped by Bucket. Cards use their persisted per-View `task_positions.position` order, with Task ID as a stable tie-breaker. It must include empty Buckets.

The existing workflow-status write operation is the sole move seam (it may be renamed without changing its behavior). It validates that the target Bucket belongs to the one manual Kanban View, updates the Task's global Done state and timestamps, changes its Bucket assignment atomically, and returns an error without partial state. The implementation must extend the current query to include `bucket_configuration_mode=manual`.

No Board move modifies `task_positions.position`. Creating the standard View seeds positions as specified by the provisioning ticket.

## Rendering and viewport

- The Tasks Pane renders Bucket columns side by side; Projects Pane collapse continues to maximize available width.
- Each Bucket column has a minimum width of 25 terminal cells. The available width determines how many full columns are visible.
- The Board keeps a horizontal Bucket viewport. If Buckets exist outside it, the header shows `<` for hidden left Buckets and `>` for hidden right Buckets.
- Bucket order is persisted position ascending with ID as tie-breaker. Card order is persisted per-View position ascending with Task ID as tie-breaker.
- An empty Bucket renders its header and an empty card area; it does not render a fake Task or a “no tasks” card.
- Existing title truncation rules apply within a Bucket column. No visual card reordering, drag interaction, or implicit append/prepend is part of this slice.

## Navigation

The Board receives normal Tasks-Pane focus and preserves existing generic Pane commands (`Ctrl+W`, `Enter`, `:q`, and `Space+o`).

- `h` / `l`: activate the adjacent Bucket. The horizontal viewport shifts as needed to reveal it.
- `j` / `k`: select the next / previous Task within the active Bucket. In an empty Bucket they are no-ops.
- Entering a nonempty neighboring Bucket tries to retain the previous card row; if unavailable it selects the first or last card respectively.
- Entering an empty neighboring Bucket leaves Task selection absent.
- At the first/last Bucket, `h`/`l` are silent Vim-style no-ops. There is no footer warning or flash.
- `Enter` on a selected Task opens the existing Task Detail Pane. It does not expand the Projects Pane merely because the Board is active.

## Moves

All moves require a selected Task.

### Adjacent move

- `Shift+H` moves the selected Task to the left neighboring Bucket.
- `Shift+L` moves it to the right neighboring Bucket.
- At an edge, the command is a silent no-op.

### Arbitrary move

- `m` opens a temporary Bucket Picker for the selected Task.
- The picker lists every Bucket in persisted order, including horizontally hidden Buckets; the current Bucket is excluded or nonselectable.
- `j` / `k` choose a target; `Enter` confirms; `Esc` cancels without a write.

### Result of a successful move

- The move uses the central workflow-status Store operation.
- Moving into the configured Done Bucket marks the Task done; moving out of it reopens the Task.
- Vimkunja reloads the Board. The target Bucket becomes active, the moved Task remains selected, and the horizontal viewport follows it if necessary.
- The Task's persisted card position is unchanged.

A failed move leaves the rendered Board and selection unchanged, records the error, and uses the existing persistent footer/error-log path. No move-specific animation or footer toast is required.

## Provisioning boundary

The Board never silently creates a View or Bucket. Standard provisioning occurs only after the confirmation defined by the provisioning ticket. It creates upstream-compatible names and positions:

| Object | Name | Position | Semantic role |
| --- | --- | ---: | --- |
| View | `Kanban` | 400 | manual Kanban View |
| Bucket | `To-Do` | 100 | Default Bucket |
| Bucket | `Doing` | 200 | ordinary Bucket |
| Bucket | `Done` | 300 | Done Bucket |

For a View added to an existing Project, Vimkunja deliberately assigns open Tasks to `To-Do` and completed Tasks to `Done`. This one choice differs from upstream's current all-`To-Do` import behavior and preserves Vimkunja's Task Done state/Bucket invariant.

## Out of scope

- Create, rename, delete, reorder, set limits on, or otherwise administer Buckets. See [`01-manage-manual-kanban-buckets.md`](../../../.scratch/kanban-bucket-management/issues/01-manage-manual-kanban-buckets.md).
- Multiple manual Kanban Views and a View picker.
- Filter-configured Kanban Views.
- Card reordering or changing `task_positions.position`.
- Inline title editing in a Board.
- Table and Gantt View rendering.

## Verification

### Store tests

- Detect no, one, and multiple manual Kanban Views; ignore non-manual Kanban Views for the single-View contract.
- Return empty Buckets and group cards into the correct Bucket in stable persisted order.
- Reject a Bucket from another View.
- Move into Done and out of Done atomically, including Done timestamps and `updated`.
- Preserve `task_positions.position` across a move.
- Provision the exact standard template transactionally; assign open and done existing Tasks to their respective Buckets; seed deterministic positions; roll back injected failures.

### TUI model tests

- `:kanban` selects the one eligible Board; no Board asks to provision; multiple Boards report ambiguity.
- Horizontal navigation, hidden-Bucket indicators, empty-Bucket selection, and vertical row preservation.
- Adjacent edge moves are silent no-ops.
- Picker confirmation/cancellation and successful/failing moves retain the required focus and selection.
- Opening Task Detail from the Board leaves the Projects Pane state unchanged.

### Manual smoke check

Against the same local database in the original Vikunja UI: compare Bucket titles/order, default/done semantics, a move to Done, a move back to Doing, a hidden Bucket, and a standard View provisioned for an existing Project with both open and completed Tasks.

## Sources

- Upstream compatibility research: [`research-vikunja-default-kanban.md`](research-vikunja-default-kanban.md)
- Historical Decisions 12–17: [`TUI_IDEA.md`](../../TUI_IDEA.md)
- Existing workflow-status decisions: [`workflow-status/spec.md`](../../2026-07-17/fundament-mvp/workflow-status/spec.md)
