# 01 — Provision a standard manual Kanban View

**What to build:** From a Project without a manual Kanban View, a user can explicitly create Vimkunja's upstream-compatible standard Kanban View and then open it.

**Implementation prerequisites:** The Kanban domain specification in [`../kanban-board/spec.md`](../../kanban-board/spec.md). This ticket precedes Board rendering.

**Category:** enhancement

**Priority:** High

**Status:** ready-for-agent

## Acceptance criteria

- [ ] `:kanban` in a Project context finds exactly one `view_kind=kanban` View whose `bucket_configuration_mode=manual` and opens it.
- [ ] With no such View, `:kanban` asks for explicit confirmation: `Create standard Kanban View for "<Project>"? y/n` (`Esc` cancels).
- [ ] On confirmation, one SQLite transaction creates the View `Kanban` at position 400 with manual Bucket configuration; its Buckets are `To-Do` (100), `Doing` (200), and `Done` (300).
- [ ] The View's Default Bucket is `To-Do`; its Done Bucket is `Done`.
- [ ] Existing open Tasks receive a `task_buckets` assignment to `To-Do`; existing completed Tasks receive one to `Done`. Every Task receives a persisted position for the new View using the same deterministic per-View ordering convention as upstream Vikunja.
- [ ] On success Vimkunja reloads Views, selects and opens the new Kanban View, and selects an existing Task when one exists.
- [ ] On failure the transaction leaves no partial View, Bucket, Bucket assignment, or position data. Vimkunja keeps the prior List View, reports the error in the footer/session log, and does not refresh stale state over it.
- [ ] Projects with multiple manual Kanban Views are not provisioned or selected by this ticket; they receive the explicit unsupported/ambiguous response defined by the Board spec.
- [ ] Store tests use a real temporary SQLite database; TUI tests cover confirmation, cancellation, success, ambiguity, and failure.

## Compatibility contract

This reproduces upstream Vikunja's default manual Kanban template: `Kanban`, `To-Do`, `Doing`, `Done`; positions 400 / 100 / 200 / 300; and `To-Do`/`Done` as the default/done Buckets. Vimkunja intentionally differs in exactly one post-hoc provisioning case: it assigns already completed Tasks to `Done` rather than upstream's `To-Do`, preserving the established global Task Done state/Bucket invariant.

The operation must not alter the Vikunja schema and must expose `bucket_configuration_mode` so it can distinguish manual from filter-configured Kanban Views.

## Further context

Primary-source findings and source links: [`research-vikunja-default-kanban.md`](../../kanban-board/research-vikunja-default-kanban.md). Board interaction and Store contracts: [`kanban-board/spec.md`](../../kanban-board/spec.md).
