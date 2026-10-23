# 01 — Verify Vikunja database compatibility for 2.7.0 only

**What to deliver:** Make Vimkunja's direct SQLite access safe for Vikunja 2.7.0 only and demonstrate compatibility with a real, synthetically populated Vikunja 2.7.0 database. Reject other database revisions clearly at production startup.

**Category:** compatibility

**Priority:** High — prerequisite for initial GitHub publication.

**Status:** completed

**Former blocker:** [Publish to GitHub](../../../2026-10-07/publish-to-github/issues/01-publish-to-github.md) — compatibility prerequisite met.

## REQUIRED: Publish only after this ticket is complete

**Initial public GitHub publication is blocked until this ticket is complete and Vikunja 2.7.0 compatibility is established with reviewable evidence. This prerequisite has been met since 2026-10-05. Source comparison or passing tests against reduced custom schemas alone do not meet it.**

License, documentation, CI, and publication-copy preparation may continue in parallel. This ticket does not authorize a push.

## Starting point

- Vimkunja started on 2026-07-08. According to the reflog, the local upstream checkout was cloned on 2026-07-07 and is at `82ee780161c6ed250cdb1fc4d2952b72464cb724` (`v2.3.0-963-g82ee78016`). This is a development state after 2.3.0, not the exact release.
- The version that initialized the original user database is unproven. Use only disposable databases for this work.
- Source comparison of release tags 2.3.0, 2.4.0, 2.5.0, 2.6.0, and 2.7.0 on 2026-10-05: core models largely preserved, but new persistence rules matter for direct SQLite writes. No executed version round trip was claimed.
- Jan limits scope to exactly Vikunja 2.7.0. The former 2.3.0–2.7.0 investigation remains historical context, not authorization for a version matrix.

## Required scope

- Only compatibility target: release tag `v2.7.0`; record its resolved commit in the report. Documented regressions and real server round trips meet acceptance.
- Earlier releases, later releases including patch releases, and development versions are unsupported. Do not build multi-version adapters or an upgrade matrix.
- At production startup, check migration history and relevant schema features against verified 2.7.0; reject unknown, older, newer, or incomplete revisions before domain writes. A SQLite file does not necessarily identify the exact server version: releases with identical database revisions cannot be distinguished this way and receive no separate approval.
- A fresh database initialized by Vikunja 2.7.0 is mandatory evidence. Older-release upgrade tests and targeted reproduction of historical repair migrations are outside this ticket.
- Run VOA and Vimkunja sequentially during verification. Do not promise concurrent server editing. Tutorial fixtures remain a synthetic demo workflow only.
- Further version checks require an explicitly requested upgrade. Keep maintenance effort limited for Jan's side project.

## Relevant changes and verification tasks

1. **Since 2.4: soft deletion.** Account for `tasks.deleted_at` and trash semantics. Check all Task reads, search, Labels, Kanban, Detail, writes, and Undo/Redo. At this starting point Vimkunja does not generally filter these Tasks and deletes permanently. Define how accepted permanent deletion remains compatible with newer versions; any deviation needs an explicit contract.
2. **Since 2.4: unique indexes.** Check uniqueness of `tasks(project_id, index)` and `task_positions(task_id, project_view_id)` during creation, moves, and Undo/Redo. Conflicts must abort transactions without partial changes.
3. **Valid 2.7.0 data.** Check Kanban Bucket mode, existing indexes, and inherited Project archiving against 2.7.0 rules. Historical 2.5/2.6 repair migrations need not be reproduced separately.
4. **Since 2.7: Task counters and aliases.** Check `project_task_counters` and `task_index_aliases`. Vimkunja's previous `MAX(index)+1` does not maintain these tables. Creation, Project moves, deletion, and Undo/Redo must preserve counter/alias rules; subsequent Vikunja Task creation must not collide with numbers already assigned by Vimkunja. Old Task addresses must remain resolvable under the approved contract.
5. **Since 2.7: Project hierarchy.** Account for NULL root parents and `project_ancestors`. The reader already supports NULL; distinguish current sibling sorting and future Project creation, parent changes, and deletion under the new rules. Unimplemented Project features need not be built in this ticket.
6. **New deletion relations.** Especially check `time_entries` and `task_index_aliases`: what does the original application preserve/remove, and what cleanup is missing from Vimkunja's permanent deletion? Reuse existing data-integrity audit fixes rather than duplicating them.
7. **Archiving and Bucket limits.** Prevent writes to archived source/target Projects and violations of configured Bucket limits. Include creation, Project moves, workflow/Done changes, and Undo/Redo.

## Verification state — 2026-10-05

[Report with evidence](../report.md): production schema validation and Store fixes implemented. Focused regressions pass for trash, archiving, Bucket limits, counters/aliases, rollback, and hierarchy preservation. Two real VOA 2.7.0 round trips pass: a fresh server database and a disposable copy of Jan's synthetic Windows baseline. VOA reads/edits Store changes, API v2 resolves old addresses, and subsequent Task creation uses new numbers. Permanent deletion remains the existing Vimkunja contract; Tasks with Time Entries are rejected and must be deleted in VOA. All tests, vet, and both builds pass; the original baseline including WAL/SHM is unchanged. Native Windows terminal testing is not claimed.

## Acceptance

- [x] Document only Vikunja 2.7.0 as verified; fix findings or agree a justified workflow limitation with Jan.
- [x] Initialize a reproducible test database with Vikunja 2.7.0 using only synthetic Users, Projects, Views, and Tasks. Schema and migration history come from Vikunja; copy no user data.
- [x] Record release tag/commit, applied migrations, schema features, test steps, and actual results. No evidence required for other releases.
- [x] Check creation, editing, Labels/Comments, Done/Buckets, Project moves, deletion, and supported Undo/Redo through Vimkunja; then read/edit results through Vikunja 2.7.0.
- [x] Add targeted regressions for soft deletion, constraints, counters/aliases, Project hierarchy, and deletion relations. Check errors, rollback, and unchanged unrelated data.
- [x] Production startup clearly rejects untested database revisions before domain writes; add regressions for older, newer, and incomplete schemas. Document unsupported workflows and version-detection limits; do not claim safe concurrent server editing.
- [x] Synchronize Store implementation, affected comments, and README with the evidenced contract; keep HTTP/JSON/schema details in the Store.
- [x] Tests and both builds pass; record the report in the feature directory before closing this ticket.
- [x] Update publication ticket and STATUS: mark this prerequisite met only after acceptance is satisfied.

## Implementation steps

1. Pin `v2.7.0` to a commit, initialize a disposable database with that version, and create synthetic data through VOA. Record reproduction, migrations, and schema.
2. Compare 2.7.0 persistence rules with Store, search, and Undo/Redo; define permanent deletion and Task-address contracts.
3. Implement production schema validation and test rejection of untested revisions; distinguish tutorial/test flows clearly.
4. Correct soft deletion, constraints, counters/aliases, and deletion relations in the Store; check hierarchy/archiving rules. Add focused regressions including rollback and unchanged unrelated data.
5. Perform all acceptance operations through Vimkunja, then read/edit through VOA 2.7.0; especially check subsequent VOA Task creation and old Task addresses.
6. Run tests and both builds; synchronize report and README. Close the ticket and publication blocker only after evidenced acceptance.

## Boundaries

Do not open, change, or delete `internal/db/vikunja.db`. No production schema changes, custom Vikunja migrations, or dependency changes without a separate instruction. A persistent starter template is a later decision based on this verified version, not a substitute for compatibility evidence.

## Sources

- [Soft-delete migration, 2.4](https://github.com/go-vikunja/vikunja/blob/v2.4.0/pkg/migration/20260707094311.go)
- [Unique Task numbers, 2.4](https://github.com/go-vikunja/vikunja/blob/v2.4.0/pkg/migration/20260411013328.go), [unique positions, 2.4](https://github.com/go-vikunja/vikunja/blob/v2.4.0/pkg/migration/20260617153629.go)
- [Kanban repair, 2.5](https://github.com/go-vikunja/vikunja/blob/v2.5.0/pkg/migration/20260802162816.go), [archiving, 2.6](https://github.com/go-vikunja/vikunja/blob/v2.6.0/pkg/migration/20260819212842.go)
- [Task counters/aliases and number allocation, 2.7](https://github.com/go-vikunja/vikunja/blob/v2.7.0/pkg/models/task_index.go)
- [Root parents, 2.7](https://github.com/go-vikunja/vikunja/blob/v2.7.0/pkg/migration/20260903104500.go), [Project ancestors, 2.7](https://github.com/go-vikunja/vikunja/blob/v2.7.0/pkg/migration/20260908202544.go)

## Comments

### 2026-10-06 — Production database upgraded to 2.7.0 after backup

The hardened startup check rejected the existing production default database with 116 instead of 150 migrations. After a successful upgrade of a copy, Jan explicitly requests upgrading production with a prior backup. Official VOA 2.7.0 migrates `internal/db/vikunja.db`; backup `internal/db/vikunja-pre-2.7.0-20261006.db` and existing SHM file verified byte-for-byte. Data preservation, exact revision history, integrity, Store reads, and real TUI startup checked. Default path unchanged and guard still strict. Opt-in upgrade regression and clear revision/upgrade guidance added; details in the report. No commit or push.

### 2026-10-05 — Decision with Jan

Create a separate ticket for database compatibility across 2.3.0–2.7.0 and make GitHub publication explicitly dependent on completion. Ticket and dependency recorded today; no compatibility implementation or publication performed.

### 2026-10-05 — Scope limited to exactly 2.7.0

Jan chooses Vikunja 2.7.0 only to limit side-project maintenance. Multi-version support and upgrade matrix removed. Reject untested database revisions; future upgrades need a separate instruction. This sets verification scope, claims no already evidenced compatibility, and authorizes no push.

### 2026-10-05 — Baseline provided and initial verification requested

Jan initializes/populates a real VOA 2.7.0 database, stops VOA, and places the clean copy including WAL/SHM under `tmp/vikunja_clean_db/`. On “go ahead,” schema/persistence checks run on disposable copies. Findings/reproduction probes are in the linked report; baseline unchanged. Implementation and full acceptance pending.

### 2026-10-05 — Implementation and automated acceptance completed

Jan requests taking over further steps. Documented Store fixes and startup checks implemented; fresh/baseline round trips, regressions, full tests, vet, and builds pass. Report and README contain the limited 2.7.0 contract and boundaries. Publication compatibility blocker resolved. No commit or push performed.
