# Research — Vikunja default Kanban provisioning

**Question:** Does upstream Vikunja create a default Kanban View and default Buckets, and what compatibility behavior must Vimkunja retain when it provisions one?

**Source revision:** [`go-vikunja/vikunja` `1f23add`](https://github.com/go-vikunja/vikunja/tree/1f23add5743071bb3c28ba56299979ae31f14823), inspected 2026-07-28. These are upstream source facts, not assumptions inferred from Vimkunja's database.

## Findings

1. **Upstream creates default Views when it creates a Project.** `Project.Create` invokes `CreateDefaultViewsForProject` when default Views are requested. The helper creates `List`, `Gantt`, `Table`, and a `Kanban` View; the Kanban View has `bucket_configuration_mode=manual` and position 400.
   - Sources: [`project.go` L1141–1146](https://github.com/go-vikunja/vikunja/blob/1f23add5743071bb3c28ba56299979ae31f14823/pkg/models/project.go#L1141-L1146), [`project_view.go` L523–570](https://github.com/go-vikunja/vikunja/blob/1f23add5743071bb3c28ba56299979ae31f14823/pkg/models/project_view.go#L523-L570).

2. **“Manual” is a real upstream View property.** A View's bucket configuration is `none`, `manual`, or `filter`. Only `manual` means ordinary persisted Buckets to which users can move Tasks; `filter` derives Buckets from filters.
   - Source: [`project_view.go` L92–101, L167–174](https://github.com/go-vikunja/vikunja/blob/1f23add5743071bb3c28ba56299979ae31f14823/pkg/models/project_view.go#L92-L101).

3. **Creating a manual Kanban View with the upstream default path creates exactly three Buckets:** `To-Do` (position 100), `Doing` (200), and `Done` (300). Upstream stores `To-Do` as `default_bucket_id` and `Done` as `done_bucket_id`.
   - Source: [`project_view.go` L346–395](https://github.com/go-vikunja/vikunja/blob/1f23add5743071bb3c28ba56299979ae31f14823/pkg/models/project_view.go#L346-L395).

4. **When the View is added to an existing Project, upstream assigns every existing Task to the new View's `To-Do` Bucket, including Tasks whose global Done flag is set, and recalculates their per-View positions.** The assignment loop does not inspect Done. That operation belongs to the same creation path; it must be transactional in a direct-SQL adapter so no half-created View is exposed.
   - Sources: [`project_view.go` L395–402](https://github.com/go-vikunja/vikunja/blob/1f23add5743071bb3c28ba56299979ae31f14823/pkg/models/project_view.go#L395-L402), [`project_view.go` L404–430](https://github.com/go-vikunja/vikunja/blob/1f23add5743071bb3c28ba56299979ae31f14823/pkg/models/project_view.go#L404-L430).

5. **Bucket semantics are View-scoped.** `default_bucket_id` is the destination for new Tasks and defaults to the leftmost Bucket; moving a Task to `done_bucket_id` marks it done, and marking it done moves it there.
   - Source: [`project_view.go` L167–174](https://github.com/go-vikunja/vikunja/blob/1f23add5743071bb3c28ba56299979ae31f14823/pkg/models/project_view.go#L167-L174).

## Implications for Vimkunja

- Correct the current shorthand: selecting only `view_kind=kanban` is insufficient. The initial board must require **exactly one View with both `view_kind=kanban` and `bucket_configuration_mode=manual`**.
- The proposed provisioning module should reproduce upstream’s creation transaction: View `Kanban` at position 400; `To-Do`/`Doing`/`Done` at 100/200/300; set default/done Bucket IDs; assign existing Tasks to `To-Do`; establish per-View positions.
- Vimkunja should not invent a conflicting `Todo` spelling: upstream’s literal is **`To-Do`**.
- Upstream’s post-hoc provisioning leaves already completed Tasks in `To-Do`. Vimkunja must explicitly decide whether strict source fidelity or preserving its global Done/Bucket invariant takes precedence for that edge case.
- Creation, rename/reorder, and deletion of additional Buckets remain distinct scopes. Upstream supports Bucket CRUD, but implementing that is not required to provision the first compatible manual Kanban View.

## Local compatibility observation

`internal/db/vikunja.db` currently shows the upstream-shaped configuration for inspected Projects: List/Gantt/Table/Kanban Views, and each Kanban View has `bucket_configuration_mode=1` (`manual`) with populated default and done Bucket IDs. Vimkunja's current `ProjectView` type does not expose the configuration mode, so the Kanban work must add it before it can reject filtered Kanban Views correctly.
