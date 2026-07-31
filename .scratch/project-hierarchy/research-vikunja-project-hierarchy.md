# Research — Vikunja Project hierarchy

**Question:** Can Vimkunja render Vikunja's existing Project parent/child hierarchy without a schema change, and what constraints does the current code impose?

**Researched:** 2026-07-31

## Conclusion

Yes. The local Vikunja SQLite schema already has the required relationship: `projects.parent_project_id`. The current database contains a real multi-level hierarchy, an index on that column, and no invalid active parent reference. Vimkunja currently omits both the parent ID and Project position from its `Project` model and reads Projects in flat `id` order, so a hierarchy feature requires read-model and query changes but no schema migration or write behavior.

## Local database findings

The checked-in database's `projects` DDL includes:

```sql
parent_project_id INTEGER NULL,
position REAL NULL
```

It has `IDX_projects_parent_project_id` on `parent_project_id`. `PRAGMA foreign_key_list(projects)` returned no foreign-key constraint, so direct SQLite writes or legacy data can leave an orphan or cycle even though normal Vikunja writes validate them. [Local schema: `internal/db/vikunja.db`, `projects` DDL and PRAGMAs queried 2026-07-31.]

The active data demonstrates the intended shape:

- Roots: `Meta`, `Birding`, `Coding`, `Zeichnungen`, and `Sonstiges` have `parent_project_id = 0`.
- `Birding` contains `Spanien` and `Grundkurs`.
- `Coding` contains `Zig`, `LCTHW`, `misc`, `Vibe`, `Vimkunja`, and `Tagger`.
- No active row references a nonexistent positive parent ID.

The local rows also have usable `position` values. For example, the root Projects are ordered by positions `65536`, `262144`, `327680`, `655360`, and `720896`; the children of `Coding` likewise have distinct positions. This supports ordering sibling groups by `position`, with a deterministic tie-breaker still required.

> Method note: the normal read-only SQLite open reported a host-filesystem I/O error while the database has WAL sidecars. The schema and sample were read from the immutable main database file; this is sufficient for schema confirmation, but the sample must not be treated as a transactional snapshot of a running application.

## Upstream Vikunja contract

Upstream's `Project` model declares `ParentProjectID *int64` as the `parent_project_id` column and `Position float64` as the ordering position. The field documentation says responses always provide a parent ID; `0` means top-level. [Vikunja `Project`, lines 38–86](https://github.com/go-vikunja/vikunja/blob/506dbd7b0483c626295081fd9dd5beef3df8def7/pkg/models/project.go#L38-L86)

The database column remains nullable for old rows. Upstream normalizes `NULL` to `0` after loading, and persists a newly-created top-level Project with `0`. Therefore Vimkunja should normalize both `NULL` and `0` to a root parent for display. [Normalization, lines 120–142](https://github.com/go-vikunja/vikunja/blob/506dbd7b0483c626295081fd9dd5beef3df8def7/pkg/models/project.go#L120-L142) and [create behavior, lines 1108–1116](https://github.com/go-vikunja/vikunja/blob/506dbd7b0483c626295081fd9dd5beef3df8def7/pkg/models/project.go#L1108-L1116).

Vikunja validates normal create/update paths: a parent cannot be negative, self-referential, missing, or form a cycle. It also propagates archive state to descendants. These checks are not a substitute for defensive TUI traversal because Vimkunja reads SQLite directly and the table has no foreign key. [Validation, lines 1018–1055](https://github.com/go-vikunja/vikunja/blob/506dbd7b0483c626295081fd9dd5beef3df8def7/pkg/models/project.go#L1018-L1055) and [archive propagation, lines 1250–1268](https://github.com/go-vikunja/vikunja/blob/506dbd7b0483c626295081fd9dd5beef3df8def7/pkg/models/project.go#L1250-L1268).

Upstream recalculates positions among Projects with the same `parent_project_id`, ordered by `position ASC`. This is the strongest primary-source basis for using `position ASC` as sibling ordering. [Position recalculation, lines 1326–1332](https://github.com/go-vikunja/vikunja/blob/506dbd7b0483c626295081fd9dd5beef3df8def7/pkg/models/project.go#L1326-L1332).

## Current Vimkunja seam

- `vimkunja.Project` contains `ID`, `Identifier`, `Title`, and `Description`, but not `ParentProjectID` or `Position`. [`internal/vimkunja/types.go`](../../internal/vimkunja/types.go)
- `SQLiteStore.ListProjects` selects only those four fields, filters archived Projects, and orders the flat result by `id`. [`internal/vimkunja/sqlite_store.go`](../../internal/vimkunja/sqlite_store.go)
- `renderProjects` renders that slice directly as a flat list. [`internal/tui/panes.go`](../../internal/tui/panes.go)
- The TUI asks for only page 1 with 200 Projects and discards the total-page result. A hierarchy cannot be complete if a parent or its children are outside that page. [`internal/tui/model.go`](../../internal/tui/model.go)
- On a Project-pane refresh (`r`), the loaded slice replaces `m.projects`; selection is retained by index except for the startup-only `initialProjectID` case. Reordering into a hierarchy can therefore silently select a different Project unless refresh first preserves the selected Project ID. [`internal/tui/model.go`](../../internal/tui/model.go)

## Implementation implications (not yet a UX decision)

1. Extend the read model and `ListProjects` query to carry normalized parent information and `position`; do not change the database schema.
2. Build the visible flat navigation list from an in-memory tree, not by embedding traversal policy into pane rendering. The tree builder should be testable as a pure module.
3. Fetch a complete active Project set before building the hierarchy, or explicitly define paging in a hierarchy. The current first-200 behavior is not safe for general hierarchies.
4. Order roots and each sibling group by `position ASC`, then a deterministic fallback such as `id ASC` for `NULL`/equal positions.
5. Keep `j`/`k`, `gg`, and `G` operating on the visible flattened tree. Stable selection should be based on Project ID, not the previous row index.
6. Detect malformed input during tree construction: `NULL`/`0` is a root; a positive missing parent, negative parent, self-parent, or revisited ID must never recurse indefinitely. The product spec must choose whether such Projects appear in a marked recovery-root section or are shown with a warning elsewhere.

## Decisions still needed for the ticket

- Whether M2 includes collapsible branches or initially renders every Project expanded.
- The collapse/expand bindings and whether `h`/`l` are reserved for parent/child navigation in the Projects Pane.
- Visual indentation, depth limit, and narrow-pane truncation behavior.
- The precise recovery presentation for orphaned/cyclic rows.
- Refresh behavior if the selected Project becomes hidden by a collapsed ancestor.

## Primary sources

1. Local Vikunja SQLite schema and active Project rows: `internal/db/vikunja.db` (read-only inspection, 2026-07-31).
2. Vikunja upstream `Project` model at commit `506dbd7b0483c626295081fd9dd5beef3df8def7`: [project.go](https://github.com/go-vikunja/vikunja/blob/506dbd7b0483c626295081fd9dd5beef3df8def7/pkg/models/project.go).
3. Current Vimkunja read model, SQLite store, and Project Pane: `internal/vimkunja/types.go`, `internal/vimkunja/sqlite_store.go`, `internal/tui/model.go`, and `internal/tui/panes.go`.
