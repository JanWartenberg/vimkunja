# Kanban View management: evidence for the decision questionnaire

Date: 2026-11-21. Local code: `a9d48b1`. This is a source review, not a live VOA round trip or an accepted implementation specification.

## Sources

Official VOA target: **2.7.0**, tag commit `a16be96aa454671fdf213b0fbe411dd38a098418` (resolved with `git ls-remote`). Sources below are pinned to that commit:

- [project_view.go](https://github.com/go-vikunja/vikunja/blob/a16be96aa454671fdf213b0fbe411dd38a098418/pkg/models/project_view.go): `ProjectView`, `Create`, `createDefaultKanbanBuckets`, `Update`, `Delete`.
- [kanban.go](https://github.com/go-vikunja/vikunja/blob/a16be96aa454671fdf213b0fbe411dd38a098418/pkg/models/kanban.go): `Bucket`, `getDefaultBucketID`, `Create`, `Update`, `Delete`.
- [kanban_task_bucket.go](https://github.com/go-vikunja/vikunja/blob/a16be96aa454671fdf213b0fbe411dd38a098418/pkg/models/kanban_task_bucket.go): `TaskBucket`, `updateTaskBucket`.
- [tasks.go](https://github.com/go-vikunja/vikunja/blob/a16be96aa454671fdf213b0fbe411dd38a098418/pkg/models/tasks.go): `checkBucketLimit`, `moveTaskToDoneBuckets`.

## Established model

A Project has multiple Views, including multiple Kanban Views. Buckets and Task positions belong to a View. Task/Bucket association is unique per Task and View; the Task itself and its Done state are shared. Manual and filter-generated Buckets are separate configurations. A manual View can also carry a View-level filter: “manual” alone does not mean “unfiltered”.

VOA creates a manual Board with To-Do, Doing and Done, configuring To-Do as Default and Done as Done Bucket. Its creation helper assigns existing non-trashed Tasks to To-Do without splitting on Done. It does not reset their Done state. This is a source-established initial placement, not a demand that completed Tasks always remain in Done.

Bucket limits are nonnegative; zero means unlimited. The last Bucket cannot be deleted. Bucket deletion retains Tasks and redirects associations to the Default; special Bucket references are cleared when affected. Without an explicit Default, the first Bucket is the fallback.

View deletion removes the View, its Task/Bucket associations and Task positions, retaining Tasks. The inspected model handler does not delete Bucket rows. Cleanup of those rows must be resolved through disposable VOA round trips before implementing local View deletion; do not present an orphan-cleanup policy as already verified.

## Existing Vimkunja behavior at initial research (before implementation)

- `internal/tui/kanban.go`: `beginKanbanViewPicker`, `updateKanbanViewPicker`; `internal/tui/commands.go`: `kanbanCommand`. `:kanban` offers multiple manual Boards; `j/k`, Enter, Esc already work. Esc currently returns to List rather than restoring the previous Board.
- `internal/vimkunja/sqlite_store.go`: `ListViews`, `LoadManualKanbanBoard` accept an explicit View; `ProvisionStandardManualKanbanView` only creates the first manual Board and splits existing Tasks by Done. General Board CRUD and Bucket CRUD are absent.
- `ListViews` does not expose the View filter; Board loading does not apply it. Imported filtered manual Views need explicit detection and a support boundary before management can safely claim support.
- `setTaskWorkflowStatus` changes global Done and only the selected View's association. `syncTaskBucketsForDoneState` can scope changes to one selected View. VOA's `updateTaskBucket` propagates a transition to Done into other manual Boards' Done Buckets. This is concrete source-level drift; reconcile the exact API paths with focused regressions and VOA round trips before expanding management. Do not ask the user to choose incompatible Done semantics.

## Accepted decision — 2026-11-21

Jan accepted decision 1 in chat: manual, unfiltered Boards first. Filter-generated Buckets and View-level filters are deferred to the [later backlog](../../../.scratch/filtered-kanban/issues/01-specify-filtered-kanban-views.md). Existing imported filter configuration must remain intact.

Jan subsequently accepted entry and switching (questionnaire decisions 2 and 4): `:kanban` immediately opens the last used Board per Project in the session, otherwise the first in persisted order. Separate Leader-menu actions provide switching and guided management. No Board means an offer of guided creation. The active Board name remains visible; exact layout for very small terminals is explicitly deferred and tracked in the [ticket](issues/01-manage-manual-kanban-buckets.md). Session memory supersedes the original explicit-picker recommendation. Jan also accepted decision 3: name first, open the standard To-Do / Doing / Done Board, then customize; existing Tasks initially go to To-Do with Done unchanged. Decision 5 is also accepted: draft configuration with explicit `:w` and `:q!`, in an unmistakable Board-configuration surface with no open Task Detail Pane. Protect unsaved Task/Comment drafts before entering; configuration saves never target Task data. Decision 6 is accepted: Board and Bucket deletion are required in the first version, with consequence previews, explicit confirmation, Task preservation and protection of the last Bucket. All six fundamental decisions are settled; detailed safeguards and compatibility checks remain. Browser-local questionnaire progress is independent of these recorded chat decisions.

## Questionnaire proposals

The [questionnaire](questionnaire.html) groups six decisions: scope, management entry, new-Board setup, Board switching, configuration editing, and deletion safeguards. Choices start as recommendations, not approvals. Confirmation is per decision; skipped questions remain open. Export includes status and notes.

Accepted scope is manual, unfiltered Boards; preserve imported unsupported Views without editing their configuration. Accepted creation follows VOA's standard initial placement. Proposed special-Bucket deletion requires reassignment first, a deliberately stricter TUI guard. The questionnaire proposed requiring a List fallback for last-Board deletion. The final specification instead allows deletion with either a List fallback or an explicit empty creation state.

## Prerequisites identified before implementation (historical)

Resolve Done propagation, filtered-View detection, View-deletion cleanup, deleting a Default Bucket, and capacity edge cases with disposable 2.7.0 API/database checks. Keep schema and dependencies unchanged. Protect Task/Comment drafts, archived Projects and other Boards. Preserve imported attributes on partial edits. Treat management Undo as unsupported unless separately specified. Implementation tickets and exact keys follow accepted questionnaire answers.

## Questionnaire verification

Local Chromium smoke checks passed for all six decisions, confirmation versus skipping, notes reopening a decision, live examples, review, reload persistence, progress import/export, malformed-import rejection, decision download, clipboard fallback and unavailable browser storage. Layout checks passed at 360, 728 and 1100 CSS pixels; no JavaScript exceptions were observed. T3 preview could not start because the host blocks its browser sandbox; a local headless Chromium with sandbox disabled was used for this artifact check. No Go code changed and no Go checks were needed for this documentation-only work.

## Implementation verification — 2026-11-21

The initial source review and questionnaire notes above are historical. Jan authorized implementation through `implement-and-verify`; the accepted [specification](spec.md) now governs implemented behavior.

- Public Store tests use disposable Vikunja 2.7.0 SQLite fixtures for creation, configuration revisions, ownership/roles, capacity rollback, archived Projects, unsupported filters and deletion with Task preservation. The schema, dependencies and local user database are unchanged.
- Official executable `vikunja-v2.7.0-linux-amd64` reports v2.7.0. `TestKanbanManagementOriginalAppRoundtrip` starts it on a disposable database and verifies Store/API interoperability for additional Board creation, initial Task placement, configuration changes, Bucket removal and Board removal. Owned Bucket cleanup after View deletion is verified. Entering a Done Bucket propagates completion across other manual Boards; ordinary non-Done moves preserve global Done. Focused regressions reconcile the pre-existing selected-Board-only drift.
- TUI tests cover immediate entry/session choice, switch cancellation, standalone configuration without Task Detail, draft protection, shared Normal/Insert commands/clipboard, stale responses, failed saves, explicit deletion, last-Board empty state, and resize/cursor bounds. Paginated navigation and management share a full-View loader; regressions cover opening/creating a Board after the first 200 Views.
- `TestKanbanManagementThroughRealTUIAndSQLite` exercises the complete name/create/configure/save/Bucket-delete/switch/Board-delete flow through tutorial input and real SQLite. The guided tutorial was updated and its course tests pass.
- Linux PTY smoke in the disposable tutorial practice database exercised creation, configuration, :q!, cancellation and confirmed Board deletion. Windows cross-build succeeds; native Windows terminal acceptance remains unclaimed.
- Independent review round 1 found stale presentation after deleting the final Board without a List, intercepted shared Normal commands, and an unavailable picker hint. All were corrected, with focused regressions for the behavioral defects. Round 2 confirmed those fixes and found inconsistent View pagination; a failing regression reproduced it and now passes with the shared loader. Ticket/status/research drift was reconciled.

Filtered Views and filter-generated Buckets remain a separate backlog. Configuration Undo/Redo is deliberately unsupported; structural management writes clear Task history to prevent stale inverses. Final automated checks and review results follow.

Final verification: `go test ./...`, `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`, `go vet ./...`, focused race-enabled TUI/Store tests, the Windows amd64 cross-build and `TestKanbanManagementOriginalAppRoundtrip` all pass on the final code. `git diff --check` passes. Both fresh independent round-3 reviewers returned no findings; one independently ran focused TUI/Store tests and the other focused TUI/tutorial tests. The first round-3 launch was interrupted by a provider usage limit before review; fresh reviewers completed after continuation. No confirmed findings remain. Changes remain uncommitted as requested by the workflow; native Windows terminal acceptance is the remaining verification gap.

## Acceptance and closure — 2026-11-22

After the subsequent UI acceptance corrections and menu-consistency implementation, Jan reported no further findings on the next manual inspection and requested commit, ticket closure and STATUS reconciliation. Further small usability improvements are deferred until needed in separate follow-up work. Earlier pending-acceptance and uncommitted statements above describe their respective historical verification stages. This acceptance closes the delivered scope without claiming exhaustive platform certification.
