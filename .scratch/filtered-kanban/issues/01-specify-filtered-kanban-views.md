# 01 — Specify filtered Kanban Views

**What to decide:** Extend Kanban support with filters after manual Board management. Distinguish a filter applied to a whole manual Board from filter-generated Buckets whose membership follows queries.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

**Milestone:** Unassigned — later backlog

**Prerequisites:** Follow the [manual View/Bucket management specification](../../../.archive/2026-11-22/kanban-bucket-management/issues/01-manage-manual-kanban-buckets.md). Existing rendering does not apply View filters; filter-generated Buckets require query support and a separate interaction contract. This ticket does not authorize implementation.

- [ ] Specify View-level filtering separately from filter-generated Bucket configuration; split implementation tickets if appropriate.
- [ ] Establish VOA 2.7.0 query semantics, filter validation and Task membership using official sources and disposable round trips.
- [ ] Define how users recognize manual versus automatic Boards and which actions are available in each mode; automatic Buckets do not support manual card moves.
- [ ] Specify editing, empty/error states, overlapping query results and unsupported imported filters without silently changing stored configuration.
- [ ] Preserve Vikunja schema, Task identity and global Done semantics.

## Comments

2026-11-21: Jan accepted manual, unfiltered Boards for the current first version and requested filtered Kanban support be retained in the later backlog. Neither filter configuration nor implementation is accepted by this deferral.
