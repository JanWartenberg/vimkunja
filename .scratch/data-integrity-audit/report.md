# Vimkunja data integrity and compatibility audit

Initial audit date: 2026-10-04. Scope: the working tree at the audit, based on HEAD
`761774b6367296bee2e5e13da65e875a43a18726`, including existing uncommitted
changes. No runtime fixes, dependency changes, schema changes, commits, or user
database access were performed. Only this report and two opt-in test files were
added. Existing work was preserved.

## Result

Eleven reproducible findings, represented by **15 failing scenarios**, plus
**12 passing safety scenarios**. All 27 scenarios produced the same verdict in
three consecutive runs. The ordinary test suite, including the race detector,
passes; these are gaps in its existing coverage, not existing test failures.

Highest-value follow-up: protect asynchronous Detail saves, validate target
Projects inside the write transaction, and preserve full due timestamps during
unrelated edits and undo. Then address Bucket sentinel/default handling and
unchanged Done state. Runtime remediation had not started at that checkpoint;
the authorized F01/F02 follow-up is recorded below.

## Ticket index

The maintainer requested one Scratch ticket per finding and implementation of
the save errors first. Completed F01/F02 tickets were moved to the archive per
repository policy. F03/F04/F05/F06/F07 were subsequently repaired and archived; four tickets
remain in Scratch.

| Finding | Ticket | Current state |
| --- | --- | --- |
| F01 | [Prevent duplicate Detail saves](../../.archive/2026-10-04/data-integrity-audit/issues/01-prevent-duplicate-detail-saves.md) | completed |
| F02 | [Preserve draft during save completion](../../.archive/2026-10-04/data-integrity-audit/issues/02-preserve-draft-during-save-completion.md) | completed |
| F03 | [Validate target Project](../../.archive/2026-10-04/data-integrity-audit/issues/03-validate-target-project-before-task-move.md) | completed |
| F04 | [Preserve full due timestamp](../../.archive/2026-10-04/data-integrity-audit/issues/04-preserve-due-time-and-undo-conflicts.md) | completed |
| F05 | [Preserve Done for unchanged workflow](../../.archive/2026-10-04/data-integrity-audit/issues/05-preserve-done-for-unchanged-workflow.md) | completed |
| F06 | [Use destination Done Bucket](../../.archive/2026-10-04/data-integrity-audit/issues/06-place-completed-task-in-target-done-bucket.md) | completed |
| F07 | [Handle zero/NULL Bucket configuration](../../.archive/2026-10-04/data-integrity-audit/issues/07-handle-zero-bucket-configuration.md) | completed |
| F08 | [Preserve completion timestamps](issues/08-preserve-unrelated-completion-timestamps.md) | ready-for-agent |
| F09 | [Protect recurring Task completion](issues/09-protect-recurring-task-completion.md) | needs-triage |
| F10 | [Clean polymorphic deletion links](issues/10-clean-task-polymorphic-relations.md) | ready-for-agent |
| F11 | [Create CalDAV identity](issues/11-create-stable-caldav-task-identity.md) | ready-for-agent |

## Save-error repair — 2026-10-04

F01/F02 are repaired. A dedicated in-flight editor guard prevents duplicate
saves and pauses keyboard input until the save completes; the footer shows
`Saving Task…`. Ctrl-C remains the emergency exit. Completion is associated with
the originating editor and the submitted draft. Later asynchronous changes stay
dirty and prevent close/quit; a replaced editor is preserved. Success and failure
both release the guard, and a failed save retains its draft for retry.

The two original TUI audit tests now run in the ordinary suite, with additional
regressions for repeated close/discard/new commands, error/retry, background
loads, editor replacement, late draft/Label changes and saved close/quit intent.
Label loading now updates only its own baseline, so it cannot accidentally mark
unrelated text as saved. The remaining Store defect tests retain their opt-in
build tag and still assert unresolved findings. Original failure counts below
describe the initial audit, not the repaired working tree.

Validation: `go test -race -count=1 ./...`,
`go build -buildvcs=false ./cmd/vimkunja ./cmd/vimkunja-tutorial` and
`go vet -buildvcs=false ./...` all passed after the repair. No schema/dependency
changes, commits or user database access. Native Windows interaction was not run.

Jan subsequently confirmed no duplicate Tasks, preserved edited titles and
successful :wq/:wqa in his normal database workflow. The targeted save-lifecycle
tests passed three repetitions in his environment. F01/F02 are manually accepted
and closed. The restricted tutorial is not suitable for free-form acceptance.

## Target-Project repair — 2026-10-04

After save-error acceptance, the maintainer requested the next ticket, F03.
updateTaskInTx now checks target Project existence in the same transaction before
writing Task fields or deleting source View assignments. Both UpdateTask and
SaveTaskWithComment share this check. Missing destinations produce an explicit
error and preserve all original Task/Label/Bucket/position/Comment state.

The original F03 audit reproduction was promoted to ordinary regression coverage
and expanded to four Store scenarios, including a deleted Project with stale
View rows. A real SQLite/TUI regression checks error feedback, dirty-draft
retention, save-guard release and prevention of quit/Undo Entry on failure.
Valid moves and Comment rollback tests still pass. Focused Store tests passed
three repetitions; full race-enabled tests, build and vet passed after repair.
F03 is automatically verified and archived; no maintainer manual acceptance is
claimed. No user database access or commit.

Severity means impact, not an estimate of how often the trigger occurs:
**High** = hidden lost edits, orphaned Tasks, or overwritten user data;
**Medium** = incorrect workflow/completion data or incomplete integration.

## Due-timestamp repair — 2026-10-04

F04 is repaired and automatically verified. Unchanged date-only drafts now leave
SQLite's full due_date value untouched inside the update transaction, including
fractional seconds and timezone notation. Changed dates keep their existing
midnight-UTC semantics; clearing still stores NULL. Combined Task-plus-Comment
saves share this preservation behavior.

Undo snapshots retain exact persisted due-date text as well as the date-only
presentation. Comparison therefore detects external same-day time changes,
including nanoseconds, and undo/redo restores the original value exactly.
The date-only UI remains unchanged. The three original failing scenarios now
run in the ordinary suite, with additional NULL/offset/fractional preservation,
Comment-save, explicit date/clear, rollback, and real TUI completion regressions.
Focused checks passed three repetitions; full race-enabled tests, both builds,
and vet passed. The opt-in safety controls passed. No user database was accessed,
no general live-server save-conflict guarantee was added, and no manual or native
Windows acceptance is claimed. Seven findings remain open.

Follow-up review closed an undo-target validation gap: malformed raw timestamps,
inconsistent calendar dates and clear targets retaining a raw timestamp are now
rejected before mutation. Three regression cases failed before this repair.
Timestamp validation accepts the SQLite driver's formats and keeps their encoded
calendar date instead of converting it to UTC. Expanded regressions cover 13
NULL/timestamp representations for ordinary and combined saves, offsets crossing
UTC date boundaries, Comment-insert rollback, and retained TUI Undo history on
conflict. Focused Store/TUI tests pass three repetitions; full race tests, build
and vet pass. Exact raw-value comparison deliberately rejects encoding-only
external changes too. The archived ticket's report link was corrected.

## Unchanged-workflow repair — 2026-10-04

F05 is repaired and reviewed. Existing Detail and combined Comment saves omit an
unchanged loaded Bucket from the write snapshot. New Task creation and explicit
workflow changes still submit their Bucket, and DoneSet remains independent.
Store updates also recognize a valid unchanged assignment transactionally and
preserve Done and the exact completion timestamp. Bucket/View ownership and
manual-View configuration remain validated; unrelated Views are preserved.

The original scenario now runs in ordinary coverage. Focused Store/TUI checks
pass three repetitions for NULL/zero/configured Done Buckets, exact completion
metadata, combined Comment saves, creation snapshots, explicit Bucket/Done
changes, stale external assignments, invalid ownership rollback and multi-View
preservation. Review found no remaining blocker after those adjacent checks.
The isolated staged tree passes full race tests, both builds and vet; opt-in
safety controls pass. No user database or schema/dependency changes, no manual or
native Windows acceptance. F08 remains open for completion timestamp rewriting
in Undo; its unchanged-workflow Detail-save trigger is removed by F05. Six
findings remain open.

## Completed-Task Project-move repair — 2026-10-04

F06 is repaired and reviewed. Destination assignments now use persisted Done:
completed Tasks receive each View's positive configured Done Bucket; otherwise
its existing default assignment is retained. Positions are initialized for all
Views, including those without a Bucket. Pure moves preserve Done and the exact
completion timestamp; explicit DoneSet changes retain their subsequent behavior.
Creation and unfinished initialization retain the prior query requirements.

The selected Done Bucket must belong to its destination View. Missing or foreign
references fail transactionally. Regression coverage includes multiple Views,
NULL/zero Done configuration, a Done-only View, unchanged unrelated Task rows,
normal and combined Comment saves, explicit Done changes, and late Bucket/Comment
failures restoring source assignments and positions. Real TUI tests drive the
Project picker and verify the Done Bucket after reopening Detail. Focused checks
passed three repetitions; the isolated staged tree passes full race tests,
build and vet. Opt-in audit safety controls pass. No remaining scoped review
blocker, schema/dependency change, user data access, manual or native Windows
acceptance. F07/F08 remain open; five findings remain in Scratch.

## Zero/NULL Bucket-configuration repair — 2026-10-04

F07 is repaired and reviewed. Zero and NULL defaults resolve to the first
positive Bucket ID in the View, ordered by position and ID. Creation, Project
moves (both Done states) and reopening share the rule. Completion without a
positive Done configuration preserves the existing assignment. No candidates
means no invented Bucket assignment; positions remain initialized. Configured
positive references remain authoritative and must belong to their View before
writes proceed. Invalid references roll back the full operation.

Board metadata, new Detail workflow selection and task-list workflow metadata
now expose the same effective default. Review corrected list availability when
a manual View exists but configuration and current assignment are all NULL.
The fallback reads run after closing result rows, without requiring extra DB
connections. Old synthetic fixtures now include their configured Buckets; the
F06 Done-only View expectation follows the new fallback for unfinished moves.

The original two audit reproductions failed consistently before the fix and now
run in ordinary regression coverage. Focused checks pass three repetitions:
NULL/zero configuration, position/ID ordering, invalid zero candidates, creation,
completed/unfinished Project moves, completion/reopening, selected-View scoping,
no-Bucket Views, missing/foreign configuration rollback and combined Comment-save
late failure rollback. Read and write checks also pass with one DB connection.
The isolated staged tree passes full race tests (including corpus verification),
both builds and vet; opt-in audit safety controls pass. No remaining scoped
review blocker, schema/dependency change, user database access, manual acceptance
or native Windows verification. F08–F11 remain open (four findings).

## Confirmed findings

| ID | Severity | Trigger and observed result | Evidence / responsible boundary |
| --- | --- | --- | --- |
| F01 | High | Start saving a new Task with `:w`, then issue `:w` again before completion. Both commands capture ID zero and create separate Tasks. Real SQLite reproduction creates **two** rows. | `TestIntegrityAuditRepeatedWriteDoesNotDuplicateTask`; [save dispatch](../../internal/tui/editor.go#L863) sets `taskLoading` but does not reject another save. |
| F02 | High | Type another title character while a save is pending. The database receives `New`; the draft becomes `XNew`. On completion the UI marks `XNew` clean, although it was never stored. Closing the draft can lose the edit. | `TestIntegrityAuditEditDuringSaveRemainsDirty`; [save completion](../../internal/tui/model.go#L945) copies the live draft into `original`, rather than the successfully saved draft. |
| F03 | High | Move an existing Task to a Project ID that no longer exists. Update succeeds, persists `project_id=999`, and deletes the original Bucket/position assignments. The Task is orphaned. | `TestIntegrityAuditMissingTargetProject`; [updateTaskInTx](../../internal/vimkunja/sqlite_store.go#L1722) never checks target existence; `nextTaskIndex` on an empty/nonexistent Project returns 1. The next undo snapshot uses an inner Project join and fails after the write has already committed. |
| F04 | High | A Task imported or edited through Vikunja has due time 18:45. Saving only its title in Detail changes that time to midnight. Undoing an inline title edit does the same. Changing its due time externally from 18:45 to 20:30 does **not** block undo; the external time is overwritten. | Three `TestIntegrityAuditDueTimePreservation` scenarios; [Task reads](../../internal/vimkunja/sqlite_store.go#L360) and [undo snapshot](../../internal/vimkunja/undo.go#L42) reduce timestamps to `YYYY-MM-DD`; writes then reconstruct midnight. Date-only UI is accepted; loss of the hidden time during unrelated edits is a compatibility preservation gap. |
| F05 | Medium | A completed Task on a manual View without a Done Bucket is edited only in its title. Saving its unchanged Workflow Bucket clears global Done and `done_at`. | `TestIntegrityAuditUnchangedWorkflowPreservesDone`; [updateTaskInTx](../../internal/vimkunja/sqlite_store.go#L1754) reapplies any nonzero WorkflowBucketID; [setTaskWorkflowStatus](../../internal/vimkunja/sqlite_store.go#L1296) computes Done solely from target Bucket instead of detecting an actual workflow transition. The TUI draft carries the loaded current Bucket even when it was not edited. |
| F06 | Medium | Move an already completed Task to another Project. Done remains true, but it is assigned to the destination's To-Do Bucket instead of its configured Done Bucket. | `TestIntegrityAuditCompletedProjectMove`; [initializeViewAssignments](../../internal/vimkunja/sqlite_store.go#L1806) always uses default Buckets. The local Vikunja Project-move implementation uses Done Buckets for completed Tasks. |
| F07 | Medium | `default_bucket_id=0` with real Buckets: creation assigns Bucket zero, and the new Task disappears from the board. `done_bucket_id=0`: marking Done replaces the current assignment with zero. | Two `TestIntegrityAuditZeroBucketConfiguration` scenarios. [Initialization](../../internal/vimkunja/sqlite_store.go#L1828) treats SQL non-NULL zero as a real Bucket; [Done synchronization](../../internal/vimkunja/sqlite_store.go#L1320) checks only `IS NOT NULL`. Vikunja uses zero as “not configured,” falls back to the first Bucket for defaults, and preserves the assignment when no Done Bucket exists. Existing tests mostly seed NULL and miss this distinction. |
| F08 | Medium | Rename an already completed Task in Detail, or undo an inline title change. Its historical completion timestamp is replaced by the current time despite unchanged Done. | Two `TestIntegrityAuditCompletionTimestampPreservation` scenarios; [workflow write](../../internal/vimkunja/sqlite_store.go#L1297) and [undo restore](../../internal/vimkunja/undo.go#L137) regenerate `done_at`. The accepted undo model intentionally excludes timestamps, so exact timestamp restoration is not a spec violation; rewriting an unrelated completion timestamp remains a compatibility risk. |
| F09 | Medium | Mark a recurring Task Done. It stays permanently Done and its due date does not advance. | `TestIntegrityAuditRecurringCompletion`, seeded with a future daily recurrence to avoid clock-dependent catch-up. Vimkunja never reads repeat fields; local Vikunja `updateDone` advances dates and reopens the Task. Recurrence is not exposed as a Vimkunja feature, but existing recurring records can still be modified without a guard. |
| F10 | Medium | Permanently delete a Task with favorites, subscriptions, Task reactions, and Comment reactions. Those rows remain after the Task and its Comments disappear. Unrelated controls remain intact. | `TestIntegrityAuditTaskDeletionCleansPolymorphicRelations`; [DeleteTask](../../internal/vimkunja/sqlite_store.go#L1412) omits these tables. Local Vikunja hard deletion explicitly cleans them. Enum values were checked against source: Task Favorite 1, Task Subscription 3, Task/Comment Reaction 0/1. |
| F11 | Medium | Create a Task in a schema containing Vikunja's `uid` column. The stored UID is NULL. | `TestIntegrityAuditCalDAVIdentity`; [CreateTask](../../internal/vimkunja/sqlite_store.go#L1564) omits `uid`, while Vikunja creation generates a UUID. Missing CalDAV identity is reproduced; a live CalDAV sync was not run, so its exact client-visible failure is not established. |

## Safety checks that passed

New real-SQLite controls cover:

- Late failures in Task creation, Project moves, Label replacement, manual Kanban
  provisioning, workflow writes, Task deletion, and undo. Trigger-injected
  failures and invalid dependencies roll back previously executed writes.
- Invalid due-date rejection and already-cancelled contexts preserve Task state.
- A Bucket belonging to another Project is rejected without changing Done.
- Undo into a deleted Project aborts without changing the current Task.
- View-scoped Done toggling preserves other Views' assignments and positions,
  as explicitly accepted in the multi-View ticket.

Existing tests also passed for combined Task-plus-Comment atomic rollback,
author-only Comment deletion, missing undo dependencies, external title conflicts,
history limits/branching, dirty-draft guards, HTML Description read-only behavior,
Project sibling movement rollback, canonical timestamps, and busy WAL checkpoint
preservation. These checks are meaningful evidence for those scenarios, not a
proof of all possible failure interleavings.

## Compatibility boundaries and remaining uncertainty

The comparison used **local source** under `/workspace/go-vikunja/pkg/models`,
not a running Vikunja server. That directory is not a separately versioned Git
checkout here: its enclosing workspace repository has no commits. Therefore no
release-wide compatibility claim is made. Important source fingerprints:

| File | SHA-256 |
| --- | --- |
| `tasks.go` | `edad7448826b2f00d61ce3247a8ff9fe029029fdf5c1381ab8543e66af531586` |
| `kanban.go` | `90ecaa26c4641786b559536dd79a48640a73d3b800b9990893c898072fd2bad9` |
| `kanban_task_bucket.go` | `388a73931cd2792e5dde4dfc30ae8e5906fd50ae02e3f722a734d1ddb3f42f83` |

- **Multi-View behavior:** local Vikunja propagates some completion changes across
  Views. Vimkunja's accepted ticket deliberately scopes board writes to the
  selected View. This difference was not classified as a new defect.
- **Soft deletion:** local Vikunja has `deleted_at` and a 30-day trash workflow.
  Vimkunja intentionally hard-deletes according to its archived deletion ticket
  and does not filter soft-deleted Tasks on reads. This is a version-dependent
  compatibility gap, separate from F10. Supporting a database produced by that
  version needs an explicit compatibility contract. No trash migration or new
  deletion behavior was introduced in this audit.
- **Server side effects:** direct SQLite writes do not execute Vikunja event
  dispatch, webhooks, notifications, caches, or recurrence/reminder services.
  Updating timestamps alone is not evidence that those features were triggered.
- **Concurrency:** optimistic protection exists for undo's normalized fields,
  not all normal editor writes. No claim of safe live co-editing with the server
  is made. F01/F02 require only local asynchronous save latency and are
  reproduced deterministically without threads or sleeps.
- **Schemas:** the new tests use synthetic disposable SQLite schemas, extending
  existing fixtures with relevant real columns and sentinel values. They do not
  cover every historical migration or run upstream API tests.
- **Environment:** checks ran on Linux. Native Windows interaction, live
  reminders, CalDAV clients, and a running Vikunja service were not tested.
- **Out of scope:** native rich-text editing is not implemented yet. Its pending
  corpus/design work was preserved and was not treated as runtime behavior.

## Reproduction and validation

Unrepaired Store defect tests use the Go build tag `integrity_audit`. They assert the
desired preservation behavior and **intentionally fail until repaired**. They
do not silently encode the faulty behavior as passing expectations. Normal
development tests remain green.

From the Vimkunja repository root, reproduce all scenarios:

```bash
go test -tags integrity_audit ./internal/vimkunja ./internal/tui -run '^TestIntegrityAudit' -count=3 -v
```

Run only the 12 passing safety scenarios:

```bash
go test -tags integrity_audit ./internal/vimkunja -run '^TestIntegrityAudit(LateFailuresRollBack|SafeBoundaries)$' -count=1 -v
```

Executed checks:

| Check | Result |
| --- | --- |
| `go test ./...` | Pass on the pre-audit working tree. |
| `go test -race -count=1 ./...` | Pass after adding opt-in tests; no reported data race in the ordinary suite. |
| `go build -buildvcs=false ./cmd/vimkunja ./cmd/vimkunja-tutorial` | Pass. VCS stamping disabled because the mounted checkout triggers Git's ownership check; no Git configuration was changed. |
| `go vet -buildvcs=false ./...` | Pass. |
| Opt-in audit, three repetitions | 27 leaf scenarios per run: 15 fail identically, 12 pass identically; expected exit 1. |

The Go executable in this environment is `/usr/local/go/bin/go`; commands above
assume Go is on PATH. Exact audit output was captured temporarily at
`/tmp/vimkunja-integrity-audit-results.jsonl`. No production code or database was
altered to make these results pass.

## Recommended repair boundaries

1. **F01/F02:** one in-flight write per editor; associate completion with its
   editor and saved draft. Block editing during save or preserve later edits as
   dirty. Cover repeated save, quit/discard, editor switching, and late results.
2. **F03:** validate destination existence inside the same transaction before
   removing original assignments. Keep the failure visible without losing the
   draft. Apply the same boundary to combined Task-plus-Comment saves.
3. **F04/F08:** retain full persisted due timestamps for preservation and conflict
   detection, even when rendering only a date. Preserve `done_at` when Done
   did not transition. Keep the accepted date input and session history scope.
4. **F05/F06/F07:** distinguish an edited workflow from its loaded current value;
   normalize zero/NULL configuration; choose destination Buckets from Done;
   validate real assignments. Preserve accepted View scoping.
5. **F09:** implement proven recurrence semantics or explicitly reject unsafe
   completion of unsupported recurring Tasks; do not silently freeze recurrence.
6. **F10/F11:** clean polymorphic links transactionally and allocate a stable UID
   where supported. Retain the explicit attachment-file preservation contract.

F01/F02/F03/F04/F05/F06/F07 were subsequently authorized and repaired as recorded above. The
remaining proposed repair boundaries have not been implemented.
