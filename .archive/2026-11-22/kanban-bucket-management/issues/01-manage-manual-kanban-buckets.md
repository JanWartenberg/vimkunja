# 01 — Manage manual Kanban Views and Buckets

**Scope:** Create, select, edit and delete manual Kanban Views and manage their Buckets in the terminal. Multiple Views belong to a Project; every management action targets one concrete View. The six accepted chat decisions define first-version scope.

**Implementation:** Authorized by Jan through `implement-and-verify`. Implemented against the accepted [specification](../spec.md); automated verification and live official VOA 2.7.0 round trips pass. Jan accepted the result after manual use on 2026-11-22 and requested closure and commit.

**Category:** enhancement

**Priority:** Low

**Status:** completed

## Accepted scope

Jan accepted questionnaire decision 1 on 2026-11-21: manage manual, unfiltered Kanban Boards first. Defer filter-generated Buckets and View-level filters to the [filtered Kanban follow-up](../../../../.scratch/filtered-kanban/issues/01-specify-filtered-kanban-views.md). Preserve existing VOA filter configurations without editing or clearing them. Entry, switching, creation, configuration editing and deletion were subsequently accepted below. All six fundamental decisions are settled and implemented. Jan subsequently authorized implementation through `implement-and-verify`.

## Accepted entry and switching pattern

Jan accepted this pattern on 2026-11-21, resolving questionnaire decisions 2 and 4. The accepted session-memory behavior supersedes decision 4's original explicit-picker recommendation.

- `:kanban` opens a Board immediately: the last used Board for the current Project in this session, or the first in persisted order when there is no session choice. Management is not an intermediate step. If the remembered Board has disappeared, fall back to the first remaining eligible Board.
- The Leader menu offers separate Board-switching and Board-management actions. Switching shows the active Board in a picker; Enter opens the selected Board and Esc preserves the current context. Management guides creation, Board settings, Buckets and deletion; it does not displace everyday card navigation.
- Without an eligible Board, `:kanban` offers guided creation. Show the active Board name above the columns when the Project has multiple Boards; Jan subsequently requested omitting this redundant line for a single Board. Direct commands are `:kanban switch`, `:kanban manage` and `:kanban new`; Leader actions are `Space v b`, `Space v m` and `Space v k`.
- Preserve unsaved Task/Comment drafts when changing context. Session memory does not introduce persisted workspace preferences.

## Accepted creation flow

Jan accepted questionnaire decision 3 on 2026-11-21: enter a name, create and open a standard Board, then customize if needed. Create To-Do, Doing and Done, with To-Do as Default and Done as Done Bucket. Follow the researched VOA initial placement: existing non-trashed Project Tasks start in To-Do, preserving their Done state and identity. Do not copy Tasks. Detailed Bucket configuration is available afterward rather than required during creation.

## Accepted configuration editing flow

Jan accepted questionnaire decision 5: edit Board configuration as a draft and explicitly save with `:w`; `:q!` discards that configuration draft. This applies to Board and Bucket settings, not ordinary card moves.

- The editing surface must unmistakably identify Board configuration and the target Board, including unsaved-change feedback. The standalone viewport scrolls fields and truncates safely at small sizes; destructive confirmation requires enough room for its minimum preview.
- The Task Detail Pane must never be open while Board configuration is being edited. Do not display Task editor fields or allow the configuration command line to target a Task.
- Entering configuration must protect any unsaved Task/Comment draft before hiding or closing its editing surface. Never silently discard or save Task data. Once configuration is active, `:w` affects only the Board configuration draft.
- Save related configuration atomically after validation; failed saves retain the configuration draft. Destructive deletion uses the separate preview and confirmation flow below.

## Accepted deletion scope

Jan accepted questionnaire decision 6: Board and Bucket deletion are required in the first version. Users must be able to remove Boards and Buckets created for experimentation.

- Show a concrete consequence preview and require explicit confirmation.
- Deleting a Board preserves Project Tasks and other Boards; its own layout and card assignments are removed.
- Deleting a Bucket preserves Tasks and moves their associations to the Default Bucket. The last Bucket cannot be deleted.
- The implemented safeguards require reassignment of Default/Done roles before deleting those Buckets. Bucket removal stays in the configuration draft until :w. Last-Board deletion falls back to List or an explicit empty creation state when no List exists.

## Deferred layout requirement

- [x] Board identity, switching/management access and cancellation are covered by narrow/short resize tests. Small normal layouts retain Ex input and, for multiple Boards, Board identity; picker/configuration use the full viewport and scroll. Native Windows terminal acceptance remains pending.

## Implementation checklist

- [x] Define the keyboard-first entry point for Bucket administration without conflicting with Task navigation.
- [x] Settle the six fundamental decisions from the [questionnaire](../questionnaire.html). Chat decisions above take precedence over its original recommendations and browser-local progress.
- [x] Specify View creation, naming, ordering, selection and deletion, including initial Task placement and post-deletion navigation.
- [x] Create a Bucket with Vikunja-compatible title, position, and optional limit semantics.
- [x] Rename a Bucket and define validation and persistence feedback.
- [x] Define reordering of Buckets and preserve Vikunja's persisted positions.
- [x] Delete a Bucket using Vikunja-compatible behavior: move its Tasks to the View's Default Bucket; reject deletion of the last Bucket; require explicit confirmation.
- [x] Keep Default- and Done-Bucket configuration valid when Buckets are edited or removed.
- [x] Add Store-level SQLite regression coverage and TUI interaction coverage; do not change the Vikunja schema.

## Further context

Upstream Vikunja supports CRUD for Buckets scoped to a Project View. A manual Kanban View is the only initial supported kind because it has persisted, movable Buckets. The upstream default Bucket template and its semantics are documented in [`research-vikunja-default-kanban.md`](../../../2026-07-28/kanban-board/research-vikunja-default-kanban.md).

## Comments

2026-11-21: Jan requested code/VOA research and an HTML questionnaire with a few fundamental choices. The [source review](../research.md) checks official VOA 2.7.0 and local code. The [six-page questionnaire](../questionnaire.html) supports individual confirmation, leaving questions open, local progress and export/import. Its proposals are not accepted yet. No application behavior was changed.

2026-11-21: After clarifying manual versus filter-generated Buckets and separate View-level filters, Jan accepted the manual, unfiltered first-version recommendation. Filtered Kanban work is recorded separately for later. Browser-local questionnaire progress does not automatically receive decisions made in chat.

2026-11-21: Jan accepted immediate Board opening, session-local last-Board selection and separate guided switching/management through the Leader menu. Small-terminal layout is explicitly deferred but must be addressed in the specification and verification.

2026-11-21: Jan explicitly required Board and Bucket deletion in the first version, accepting consequence previews, confirmation, Task preservation and protection of the last Bucket. All six fundamental decisions are now recorded; detailed specification remains.

2026-11-21: Implemented following explicit `implement-and-verify` authorization. Store configuration writes are atomic and revision-checked; management protects Task/Comment drafts and uses a separate configuration surface. Real tutorial TUI/SQLite regression, official VOA round trip and Linux terminal smoke cover creation, configuration, Bucket deletion, switching and Board deletion. See [verification evidence](../research.md#implementation-verification--2026-11-21). Native Windows acceptance is the remaining human check; no commit requested.

2026-11-21: Final full tests, both builds, vet, focused race tests, Windows amd64 cross-build and official VOA 2.7.0 round trip pass. Three independent review rounds completed; all confirmed findings were repaired and both final reviewers report no findings. Ready for native Windows terminal acceptance; changes are uncommitted.

2026-11-21: Jan’s usability feedback: local :h/:help now explains commands per management page and returns to the unchanged draft with q/Esc/:q. The Board title line is omitted when the Project has only one Kanban Board; multiple Boards retain identification. Focused regressions cover both behaviors and help scrolling/resizing.

Usability correction verification: `go test ./internal/tui ./internal/tutorial`, both command builds and `git diff --check` pass. Existing single-Board layout assertions were updated for the removed header row.

2026-11-21: Kanban footer now advertises the existing `Space v m` Board-management shortcut after move/navigation/help hints and before secondary actions. Whole-segment clipping remains unchanged for narrow terminals. The existing footer regression failed before the hint was added and passes afterward; no new single-key binding was introduced.

Footer verification: `go test ./...`, both command builds and `git diff --check` pass. Two independent read-only reviewers report no findings in the footer change. Changes remain uncommitted.

2026-11-22: Jan’s acceptance corrections: conventional Pane frames for management/configuration/help; new Boards proceed to Bucket configuration; errors no longer replace helpers; explicit Default/Done hints; dd stages Bucket deletion after confirmation; help explanations align in columns with narrow stacking; deletion/discard uses shared compact centered confirmation modals. These supersede the earlier direct-open creation and frameless layout details. Native Windows re-acceptance remains pending.

2026-11-22 verification: red/green regressions cover creation-to-configuration, Pane frame, persistent error/helpers, dd without text loss, centered compact confirmation, tiny-terminal confirmation blocking and aligned help. `go test ./...`, both command builds, `go vet ./...`, Windows amd64 cross-build and `git diff --check` pass. Both independent reviewers report no findings in this acceptance-correction delta. Ready for Jan’s Windows recheck; uncommitted.

2026-11-22: Follow-up acceptance corrections: Project-creation Ex input is pinned above the modal bottom border; Board management has one-cell top/left spacing and a helper separator; o/O inserts after/before the selected Bucket (including when its Limit field is selected). Normal text operator/selection behavior remains shared. Clarified View-local placement versus global Done: reopening by leaving one Board’s Done Bucket can leave a different Board’s card in its Done column without ✓, matching the checked VOA code; no semantic change in this follow-up.

2026-11-22 follow-up verification: red/green regressions cover relative insertion, top/left spacing, footer separator and bottom Project-creation command placement; focused shared text and resize/cursor regressions pass. `go test ./...`, both builds, vet, Windows amd64 cross-build and diff whitespace checks pass. Both independent delta reviews returned no findings. Uncommitted, native Windows recheck pending.

2026-11-22: Jan authorized the reported menu-consistency corrections with implement-and-verify and requested unit tests. Added Update/Render regressions for tiny Project-discard safety, visible Board-switch selection/actions, Vim navigation, mode-specific hints, editor command aliases/unavailable actions, Project/Parent help preserving drafts, fixed Project footer, clamped fields and created-Board/Project identity. The initially reported no-scroll Board-switch finding was corrected: the public renderer already scrolls; its frame/navigation are now aligned. Confirmation, navigation, footer hints, command resolution and reference formatting reuse shared helpers. Native Windows re-acceptance remains pending.

2026-11-22 menu-consistency verification: 15 focused regression-test functions (including table cases) added through red/green slices. Full `go test ./...`, both command builds, `go vet ./...`, Windows amd64 cross-build and `git diff --check` pass. Review round 1 found fallback Project-help scroll budgeting and missing contextual COMMAND headers; shared effective layout and regressions repair both. Round 2 found long Project names could hide Board identity at 80 columns; separate bounded identity budgets and a regression repair it. Both fresh round-3 reviewers report no findings. No schema, dependency or user-data changes; uncommitted, pending native Windows acceptance.

## Completion — 2026-11-22

Jan reported no further findings on the next manual inspection, accepted the result and explicitly requested committing, closing this ticket and updating STATUS. Additional usability optimizations will be handled only when needed in separate follow-up work. This records acceptance of the delivered scope, not an exhaustive platform certification. All implementation and menu-consistency review findings are resolved. The filtered Kanban follow-up remains open separately.
