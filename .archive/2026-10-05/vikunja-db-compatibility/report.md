# Vikunja 2.7.0 compatibility — verification report

**Date:** 2026-10-05
**Result:** The bounded Vimkunja workflow is verified against the official Vikunja 2.7.0 server. Automated roundtrips passed on a fresh server-created database and on a disposable copy of Jan's synthetic Windows baseline. Productive startup rejects differing migration histories and incomplete required schemas. Native Windows terminal behavior remains untested.

## Baseline and provenance

- Jan initialized a new SQLite database with the Windows server reporting `Vikunja version v2.7.0`, seeded it through VOA, stopped the server with Ctrl+C and copied the database plus WAL/SHM to `tmp/vikunja_clean_db/`.
- Release tag: `v2.7.0`; resolved source commit: `a16be96aa454671fdf213b0fbe411dd38a098418` ([release](https://github.com/go-vikunja/vikunja/releases/tag/v2.7.0)). The executable's download checksum was not independently recorded.
- Baseline contains one synthetic user, three Projects (including the initial Project), three Tasks, one Label association and one Comment. It also contains the automatically created saved-filter Views. All three Projects have NULL root parents and self-rows in `project_ancestors`.
- The 4 KB main database is not self-contained: its WAL contains the schema and seeded data. Every SQLite operation used disposable copies of all three files. The baseline was hashed before inspection and is checked again at the end; see [baseline manifest](evidence/baseline-manifest.json).
- SQLite `PRAGMA integrity_check` on the copied database returned `ok`.
- `migration` contains `SCHEMA_INIT` plus 150 timestamp IDs. These exactly match the 150 migration files at the pinned commit; latest ID is `20260928140648`. These are **recorded IDs**, not proof that a fresh installation executed all 150 historical migration bodies.
- [Schema snapshot](evidence/schema.sql) and [migration IDs](evidence/migration-ids.txt) contain schema metadata only. They are evidence, not a schema initialization or migration mechanism.

## Completed implementation and acceptance

- Productive startup uses `NewVerifiedSQLiteStore`: all 150 migration IDs, required Store columns, unique Task/index/position/alias/Task-Bucket constraints and counter primary key are checked before business writes. `SCHEMA_INIT` may be absent on upgraded installations; the supplied legacy database upgrade was verified on 2026-10-06 as recorded below; no exhaustive upgrade matrix is claimed. A server release with the same DB revision cannot be distinguished from the file alone and receives no additional version support.
- Trash is hidden from normal reads, Label counts and Kanban; Task writes, detail reads and history reject trashed Tasks. Source/destination archive flags and nonzero Bucket capacities are enforced transactionally, including undo/redo and provisioning.
- Transactional counters include reserved Task/alias indexes, maintain the high-water mark on deletion and recover stale counters. Every Project transition records its retired address, including undo/redo. Permanent Task deletion removes its aliases.
- Regression assertions use a schema snapshot extracted from VOA, with disposable synthetic rows. They cover older/newer/duplicate/missing histories, incomplete schemas, counters, aliases, trash, archive guards, Bucket capacity, injected write failure/rollback and preservation of NULL roots, parents and all ancestor rows during sibling sorting. Existing deletion-relation tests continue to cover polymorphic cleanup.
- The opt-in integration test starts the official server, registers synthetic data through its API, stops it, executes real Store methods, restarts VOA, reads/edits those changes, creates subsequent Tasks, stops VOA and reads its changes through Vimkunja. Server and Store access never overlap. This verifies the Store/VOA boundary; it does not claim manual TUI acceptance.
- Verified operations: create/edit Tasks and Labels, assign Labels, append/delete Comments, manual Bucket changes, Done, Project moves, permanent deletion and supported undo/redo. Retired addresses resolve through **API v2** (the v1 by-index route does not follow aliases). VOA allocates the next indexes A=4 and B=3 after the Store's moves/undo/redo/deletion; no reserved number is reused.
- The fresh run and copied-baseline run both passed. The source baseline's three SHA256 hashes remain identical to the initial manifest. No original DB, user DB, productive schema or dependency was changed.

Official Linux amd64 full ZIP SHA256: `569861fd690f9ffc812e39e20be3943b882d4549a19e556e47066e71e29ca1b9`, verified against the release checksum asset. The server identifies itself as `v2.7.0`; its source tag resolves to the commit recorded below. The Windows baseline executable checksum was not recorded separately.

Evidence: [fresh roundtrip](evidence/roundtrip-fresh.txt), [copied-baseline roundtrip](evidence/roundtrip-baseline.txt), [full tests](evidence/tests.txt), [validation](evidence/validation.txt). `go test ./...`, `go vet ./...` and both command builds passed with Go 1.25.0 on Linux. The older installed toolchain is a test-environment fact, not a recommendation for release builds.

### Bounded workflow contract

Only **Vikunja 2.7.0** is supported. Stop VOA before opening Vimkunja; stop Vimkunja before restarting VOA. Back up the database with any WAL sidecar before use. Vimkunja never creates or migrates the productive schema.

Vimkunja retains its existing **permanent deletion** workflow rather than VOA's normal trash operation. This deletion has no undo and removes Task relations and aliases; it leaves Attachment files on disk. **Tasks with Time Entries cannot be permanently deleted in Vimkunja**: use VOA. This restriction prevents lost/orphaned time history without implementing Pro time tracking. Recurring completion also remains a VOA operation. Direct writes do not run server webhooks, notifications or reminder services. Label creation and Comment writes/deletion require exactly one local User. No additional hierarchy creation/reparenting/deletion feature was added.

### Reproduce automated verification

From the repository root (the official executable path must name Vikunja 2.7.0):

```bash
go test ./...
```

```bash
VIMKUNJA_VOA_BINARY=/absolute/path/to/vikunja-v2.7.0-linux-amd64 go test -tags=integration ./internal/vimkunja -run '^TestVikunja270OriginalAppRoundtrip$' -v -count=1
```

For an existing **stopped synthetic** seed, additionally set `VIMKUNJA_VOA_BASELINE` to its main DB filename and `VIMKUNJA_VOA_USERNAME` / `VIMKUNJA_VOA_PASSWORD` to its synthetic credentials. The harness copies the main file and any WAL/SHM sidecars before opening SQLite. A fresh database and User are created when the baseline variable is absent.

The test lives in `internal/vimkunja/compatibility_roundtrip_test.go`; it skips when the binary variable is absent. API payloads remain in the test harness; the product still accesses SQLite directly.

## 2026-10-06 — Productive legacy database upgrade

The default local database still had 116 timestamp migrations (latest `20260226172819`) and was correctly rejected by the new 2.7.0 guard. The original acceptance covered fresh 2.7.0 databases, not this legacy productive database. On Jan's explicit request, the official, previously checksum-verified 2.7.0 binary first migrated a disposable copy, then migrated `internal/db/vikunja.db` after a byte-verified backup to `internal/db/vikunja-pre-2.7.0-20261006.db` (including the existing SHM sidecar; no WAL existed at backup time). No Vimkunja migration or relaxed revision check was added. The default path remains unchanged.

Both upgrades succeeded. All 150 timestamp IDs now match the verified revision; `PRAGMA integrity_check` returns `ok`. Compared with the backup, Task IDs/titles/descriptions/Done/completion/due dates/Projects/priorities/indexes/UIDs, Project IDs/titles/descriptions/identifiers, User IDs/usernames/password hashes, Labels, Label associations and Comments are preserved exactly. No private field values were added to the evidence. The default command rendered the real TUI and exited cleanly through `:qa`. Future VOA access must use 2.7.0 as well.

The opt-in `TestVikunja270LegacyDatabaseUpgrade` copies a supplied stopped legacy database and its sidecars, reproduces the startup rejection, invokes official `migrate`, verifies acceptance, selected business-field preservation, integrity and Store reads, and checks source bytes remain unchanged. It passed against the legacy productive backup. Ordinary revision tests now cover the 116-migration legacy history and its actionable diagnostic. Full ordinary tests and both builds passed. This upgrade verifies the supplied legacy database; it does not claim an exhaustive upgrade matrix.

```bash
VIMKUNJA_VOA_BINARY=/absolute/path/to/vikunja-v2.7.0-linux-amd64 VIMKUNJA_VOA_UPGRADE_BASELINE=/absolute/path/to/stopped-legacy.db go test -tags=integration ./internal/vimkunja -run '^TestVikunja270LegacyDatabaseUpgrade$' -v -count=1
```

## 2026-10-06 — Review corrections

Review reproduced two actionable gaps. Permanent deletion could erase the final evidence of a reserved Task/alias number when a Project counter was missing or stale. Deletion now raises the relevant counters before removing the Task and aliases, in the same transaction; regression cases cover both source and retired-address Projects and rollback after an injected deletion failure. The startup guard accepted partial unique indexes, indexes with an extra expression key, a missing Task/Bucket unique constraint and a composite counter primary key. These incomplete schemas are now rejected, with focused failing-then-passing regressions. No productive database or schema was changed in this review.

After the fixes, `go test ./...`, `go vet ./...` and both command builds passed. The official fresh-server roundtrip and the legacy-upgrade test against disposable copies of the productive backup both passed. The publication checklist now marks its already-completed compatibility evidence consistently. No commit or push.

## Historical initial audit

The following records the **pre-fix** observations; its failed behavior is repaired above. Probe `PASS` meant observations completed, not compatibility acceptance.

## Executed probes

The probes invoke the current Go Store directly on separate disposable copies of the real schema. Additional edge-case data is synthetic SQL setup inside those copies. They do not launch the TUI or VOA. [Probe source](audit_probe_test.go) and [captured output](evidence/probe-results.txt) are retained for reproduction.

| Area | Observed result | Required action |
| --- | --- | --- |
| Basic reads | Projects, Overview, Search, List View, manual Kanban and the seeded Comment read successfully. | Retest after fixes, then verify the TUI/VOA handoff. |
| Task counters | Creating Task index 3 in Project 2 leaves `last_index` at 2. Simulating upstream's next allocation produces index 3 and a unique-index insertion failure. | Allocate and update the counter in the same transaction as creation, moves and undo/redo. Preserve the high-water mark across deletion. |
| Task address aliases | Moving Task 2 from Project 2 to Project 3 produces no alias for its old address and leaves the destination counter stale. Undo also creates no aliases. | Record the retired address on every Project transition, including undo/redo; follow upstream collision handling. |
| Soft deletion | After setting `deleted_at` and removing positions/buckets as VOA does, Task 1 remains visible in Overview. Inline title editing and undo-snapshot loading still succeed. | Filter every normal Task read and guard every Task write/history operation against trash. |
| Permanent deletion | Deleting Task 2 removes the Task but leaves its synthetic `task_index_aliases` row. | Delete aliases atomically with the Task. Keep permanent deletion explicitly documented as a deviation from VOA's normal trash operation. |
| Archived Projects | Create and Project move succeed into an archived Project. | Reject writes to archived source/destination Projects, including indirect writes and undo. |
| Bucket capacity | With Bucket 7 limited to one existing Task, creation assigns a second Task successfully. | Enforce nonzero bucket limits on affected assignment flows, or agree and document a restricted workflow. |
| Constraints/rollback | Duplicate `(task_id, project_view_id)` is rejected by the real unique index. Injecting a position-write failure rolls back Task creation. | Preserve this behavior; add assertions for counters, aliases and all affected rows once fixed. |
| Root parents/sibling order | Current sibling sorting runs successfully with NULL roots; no parent or ancestor writes occur in the implementation. Probe confirms three NULL roots and unchanged ancestor-row count. | Add focused preservation assertions; no new hierarchy mutation feature is needed. |
| Schema validation | Removing migration history on a copy does not stop normal Store reads. Constructor checks file existence/openability only. | Add the planned productive-start check before business writes; keep tutorial fixtures separate. |

The allocation failure above is a SQL simulation based on the inspected upstream algorithm, **not an executed VOA task-creation roundtrip**. The probe test's `PASS` means its observations completed, not that compatibility passed.

## Persistence rules and limits

- [Upstream numbering](https://github.com/go-vikunja/vikunja/blob/a16be96aa454671fdf213b0fbe411dd38a098418/pkg/models/task_index.go) uses a persistent counter. If absent, its fallback considers both Tasks (including trash) and aliases. The pre-fix Vimkunja used `MAX(tasks.index)+1` in `sqlite_store.go`; `undo.go` shares that allocator.
- [Upstream Task updates/deletion](https://github.com/go-vikunja/vikunja/blob/a16be96aa454671fdf213b0fbe411dd38a098418/pkg/models/tasks.go) create retired-address aliases on Project moves and remove aliases during hard deletion. Normal deletion is soft deletion. Vimkunja's permanent deletion and retention of attachment files are already documented differences and need an explicit 2.7.0 contract.
- The deletion probe also leaves a synthetic `time_entries` row. The inspected upstream `hardDeleteTask` likewise contains no time-entry deletion. Therefore retention alone is **not classified as a confirmed Vimkunja-specific bug**. Time tracking is license-gated upstream; fixture has no genuine Time Entries. Permanent deletion of Tasks with Time Entries is now explicitly excluded and rejected transactionally. Do not delete historical time data merely by analogy with other relations.
- [Project rules](https://github.com/go-vikunja/vikunja/blob/a16be96aa454671fdf213b0fbe411dd38a098418/pkg/models/project.go) treat archived Projects as read-only and materialize archive flags onto descendants. Vimkunja's Project reader already filters the materialized flag; its Task write paths check existence but not archive state.
- [Bucket assignment rules](https://github.com/go-vikunja/vikunja/blob/a16be96aa454671fdf213b0fbe411dd38a098418/pkg/models/kanban_task_bucket.go) enforce capacity on assignment changes. Pre-fix Store assignments did not inspect `buckets.limit`.
- This inspection is not a full audit of every upstream side effect. Existing limitations remain: sequential server/TUI use, no server notifications/webhooks/reminder execution, and recurring completion performed in VOA. The completed operation matrix above supplies write evidence for the bounded contract.

## Reproduce the Store probes

Run from the repository root with Go and Python 3 available. These probes assume this particular seed's Project/Task IDs. Stop VOA before copying a replacement baseline. Go's overlay attaches the probe file to the Store package without adding production files.

```bash
python3 -c 'import json,pathlib,tempfile; r=pathlib.Path.cwd(); p=pathlib.Path(tempfile.gettempdir())/"vimkunja-270-audit-overlay.json"; p.write_text(json.dumps({"Replace":{str(r/"internal/vimkunja/compatibility_audit_probe_test.go"):str(r/".archive/2026-10-05/vikunja-db-compatibility/audit_probe_test.go")}})); print(p)'
```

Use the printed path for `<overlay-path>`:

```bash
go test -overlay=<overlay-path> ./internal/vimkunja -run '^TestCompatibilityAuditProbe$' -v -count=1
```

The default source directory is `tmp/vikunja_clean_db`; `VIMKUNJA_AUDIT_BASELINE` can select another absolute directory. The probe copies the main file and any WAL/SHM sidecars into `t.TempDir()` before opening SQLite. All mutations affect those copies only.

## Completion

All ticket acceptance items are backed by the fixes, focused regressions and real-server roundtrips above. No commit, push or publication was performed. Future Vikunja upgrades require a separate verification task.
