# 01 — Add Task status filter shortcuts

**Phase:** M3 — List and Overview filtering

**What to build:** Add quick Task filters in the List and Overview Views:

- `fo` — filter open Tasks
- `fv` — filter overdue Tasks
- `fi` — filter Tasks currently in work

The `fi` filter must show only Tasks that are currently in work.

**Category:** enhancement

**Priority:** Medium

**Status:** implemented

## Decisions confirmed

- `fo` matches exactly Tasks with `Done == false`, including overdue and in-progress Tasks.
- `fv` matches unfinished Tasks whose due date is strictly before the user's local current day. Tasks without a due date are not overdue.
- `fi` matches only unfinished Tasks with an unambiguous workflow status, where the current bucket is neither the configured default bucket nor the configured done bucket. Tasks without an unambiguous workflow status do not match. Bucket titles such as `Doing` or `TODO` are not interpreted by name.
- Filters apply to every non-Kanban task list: project List Views and all Overview variants, including label-filtered Overview. Kanban is excluded because its bucket columns already represent workflow status.
- The existing `f`-prefix interaction remains in place; the new filter is a small extension, not a new filter model. Selecting a shortcut replaces the current task filter.
- An empty result set is rendered explicitly as `No matching tasks`.
- After a Task save, the active filter is reapplied to refreshed results. Changing Views may reset or disregard the filter.

- [x] Add `fo` for open Tasks in List and Overview Views.
- [x] Add `fv` for overdue Tasks in List and Overview Views.
- [x] Add `fi` for Tasks currently in work only.
- [x] Add regression coverage for shortcut handling and filter results.
