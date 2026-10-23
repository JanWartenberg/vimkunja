# Vimkunja – Status

**As of:** 2026-10-23 | **Branch:** main | **MVP:** completed and archived | **Current milestone:** Milestone 3

## Current state

The MVP, **Milestone 1**, **Milestone 2**, and the initial GitHub source publication are complete. Everyday task management is practical without switching to the original application. Open M3 work remains in the backlog; publication no longer blocks it, and no new implementation priority has been selected.

- Local Vikunja SQLite database: Projects, List View, All Tasks Overview, Task Detail, Task Editor, and Task deletion
- Hierarchical Projects Pane with indentation, session-local folding, defensive recovery entries, and persistent sibling ordering
- Workflow status and Done toggle in Detail, with Done/Default Bucket synchronization
- Consistent Due Date, Priority, and identifier display in List, Overview, and Detail; stable metadata layout in narrow panes
- [Compact Task Detail](.archive/2026-08-14/task-detail-compact-layout/issues/01-preserve-description-access-in-short-terminals.md#regression-correction--2026-10-09): Description visibility and scroll feedback at low heights repaired; Jan accepted the layout on Windows.
- List, Overview, and Kanban Views with Vim navigation, pane focus, Ex commands, help, error log, and Leader key; Kanban card selection stays synchronized with Task Detail
- Separate English tutorial with a flat start menu: guided or free practice for controls or markup, plus Quit; temporary SQLite database created only after mode selection
- `go test ./...` passes

- [Task-based tutorials](.archive/2026-10-20/tutorial-task-workflow/issues/02-convert-all-guided-tutorials.md) – general course and rich-text workshop on `main`: 26 general and 22 rich-text tasks with free interaction, result checks, F5 to restore the current starting state, F8 after success, and F12 to skip; compact display and dismissible context/help modals. Earlier branch references in archived reports are historical; old worktrees have been retired.

- [Native rich-text Descriptions](.archive/2026-10-19/description-rendering/issues/10-verify-native-rich-text-compatibility-and-documentation.md) – implemented and manually accepted: **AT-01–10 PASS**, tutorial completed. Selection/formatting, Checkboxes, Code/Tabs, system clipboard, and live Vikunja round trip including in-app reload verified; regression tests cover known defects. [Editor follow-up](.archive/2026-10-19/description-rendering/issues/11-format-visual-selections-and-convert-touched-paragraphs.md) completed.

- [Rich-text fuzz recheck](.archive/2026-10-20/description-rendering/fuzz-audit-2026-10-17/report.md) – completed; deterministic regressions and short CI fuzz checks protect the fixes.

## Agreed next steps

**2026-10-23 — Publication complete:** [JanWartenberg/vimkunja](https://github.com/JanWartenberg/vimkunja) is public, with `main` at `4358202d7b72c7c5e9766422dbaa55588e844180`, matching the local source state before this documentation update. Remote heads/tags contain only `main`. [Actions run 39991127576](https://github.com/JanWartenberg/vimkunja/actions/runs/39991127576) succeeds on Windows and Ubuntu, including tests, both builds, and vet. The prior Windows configuration-test failure is corrected; the Store suite passes on the runner in this run. The cause of the earlier timeout is not established.

**Local consolidation accepted:** Jan moved the retired folders outside the workspace. The remaining `/workspace/vimkunja` checkout has the GitHub HTTPS remote and `main` upstream configured; Git lists no additional worktrees. The current database and its sidecars are ignored. Jan confirms `go run ./cmd/vimkunja` starts successfully from this isolated checkout on Windows. Backups were verified before consolidation; this update does not alter user data.

**Ticket reconciliation:** All open issue files reviewed. Publication is completed and archived; other tickets remain `needs-triage`. Completed prerequisites and current clipboard bindings are reflected in their tickets, and the adopted Task-copy spec is included. First-run database initialization remains separate open M3 work; broader platform acceptance, live CalDAV, and binary releases are not claimed by publication closure.

1. Select the next M3 ticket and resolve its open product decisions before implementation.
2. Keep first-run database initialization separate from the accepted startup with an existing database.

## Current tickets

All Tasks Overview, Task deletion, editor cursor rendering, Home/End aliases, [Home/End in Insert Mode](.archive/2026-09-11/insert-mode-home-end/issues/01-support-home-end-in-insert-mode.md), Done toggle in Detail, Task metadata display, Task Labels, Task Comments including individual deletion, and [session-local Undo/Redo](.archive/2026-09-16/undo-task-actions/issues/01-undo-recent-task-actions.md) are complete. Standard Kanban provisioning, single and multiple manual Kanban Views, Project hierarchy, and Global Search are implemented and archived. Open tickets below are grouped by milestone; `needs-triage` still means a ticket lacks a concrete specification. Task-status filter shortcuts are implemented and archived.

### Data integrity audit — 2026-10-04

- [Audit report and ticket index](.archive/2026-10-20/data-integrity-audit/report.md#ticket-index) — eleven findings tracked individually. F01/F02 are repaired, manually accepted and closed. F03 rejects missing target Projects transactionally; F04 preserves full due timestamps and detects same-day undo time conflicts. F05 preserves Done and completion metadata when workflow is unchanged. F06 initializes completed Project moves in destination Done Buckets. F07 resolves zero/NULL defaults to the first valid Bucket and preserves assignments without a Done Bucket. F08 preserves exact completion timestamps whenever Done does not transition, including Undo/Redo. F09 is repaired (unsafe recurring completion rejected). F10 is repaired (polymorphic deletion cleanup). F11 is repaired (stable creation UID). Ordinary SQLite/TUI regressions pass; F03–F11 are archived. All eleven audit findings are repaired and archived. Full race-enabled tests, build and vet pass; no manual acceptance of F03–F11 is claimed.

### Milestone 2 — completed

Read-only HTML Description projection, scroll feedback, [additional keyboard shortcuts](.archive/2026-08-08/more-key-shortcuts/issues/01-keyboard-shortcuts.md), [compact Task Detail](.archive/2026-08-14/task-detail-compact-layout/issues/01-preserve-description-access-in-short-terminals.md), [Ex-command history and completion](.archive/2026-08-12/command-line-enhancements/issues/01-add-project-goto-command-history-and-completion.md), [inline Task title editing in List and Overview](.archive/2026-08-18/inline-task-editing/issues/01-edit-task-titles-inline-in-list-and-overview.md), [field-local Detail picker overlays](.archive/2026-09-03/detail-picker-overlay/issues/01-render-detail-pickers-as-field-local-overlays.md), the Due Date picker with relative helpers, canonical UTC timestamps, single-line status footer, narrow modal layouts, and [user configuration](.archive/2026-10-20/user-configuration/issues/01-add-user-facing-tui-configuration.md) are complete and archived; Kanban title editing is deliberately excluded. The native rich-text editor is also implemented and accepted (Milestone 3).

### Milestone 3 — current

12. [Display Checkbox-list progress](.archive/2026-10-20/checkbox-list-progress-summary/issues/01-show-checkbox-list-progress-summary.md) – completed and archived; Native Checkbox edits update progress live, and empty lists are removed. Imported plain text stays literal in the shared editor.
13. [Advanced Vim editor motions](.archive/2026-10-20/editor-motion-polish/issues/01-add-advanced-vim-editor-motions.md) – completed and archived; Jan accepted the final Comment `~` recheck. The old HTML write-protection test is superseded by rich-text acceptance; the original report is preserved.
14. [Specify Project deletion semantics](.scratch/project-deletion/issues/01-specify-project-deletion-semantics.md)
14a. [Specify Project creation](.scratch/project-creation/issues/01-specify-project-creation-workflow.md) – define entry point, hierarchy, initial View, and Vikunja-compatible persistence before implementation.
15. [Render Table View](.scratch/table-view/issues/01-render-table-project-view.md)
16. [Render Gantt View](.scratch/gantt-view/issues/01-render-gantt-project-view.md)
17. [Evaluate independent Link text/URL editing](.scratch/link-label-and-url-editor/issues/01-evaluate-independent-link-label-url-editing.md)
18. [Manage manual Kanban Buckets](.scratch/kanban-bucket-management/issues/01-manage-manual-kanban-buckets.md)
19. [Local search in the focused pane](.scratch/local-search/issues/01-search-within-focused-pane.md) – `/` and `n`/`N` agreed; pane semantics remain open.
20. [Advanced Global Search queries and ranking](.scratch/global-search-filter/issues/02-add-advanced-global-search-query-and-ranking.md) – phrases, negation, fuzzy matching, and ranking deliberately deferred beyond V1.
21. [Diagnose and repair malformed Project hierarchies](.scratch/project-hierarchy/issues/02-diagnose-and-repair-malformed-project-hierarchies.md) – hierarchy reader completed; diagnostic feedback and safe repair still require specification.
22. [Native rich-text Descriptions](.archive/2026-10-19/description-rendering/issues/10-verify-native-rich-text-compatibility-and-documentation.md) – completed and archived; implementation 03–09 and integration/follow-up 10/11 verified. All ten manual acceptance points passed, including Windows/AltGr, clipboard, resize, and live Vikunja checks.
22a. [Copy the selected Task to the clipboard](.scratch/task-clipboard-copy/issues/01-copy-selected-task-to-clipboard.md) – adopted [spec](.scratch/task-clipboard-copy/spec.md): `:copy` for exactly one Task; rich-text integration is complete, while metadata format and unsaved-change handling remain to specify.
22b. [More accessible URL-copy shortcut](.archive/2026-10-20/description-link-copy-shortcut/issues/01-choose-accessible-url-copy-shortcut.md) – completed and archived: `Space+e`, with `Space+Y` retained as an alias; palette and hints show `Space+e`.
23. [Specify separate histories for contextual command prompts](.scratch/contextual-command-histories/issues/01-specify-independent-command-prompt-histories.md) – follows M2 Ex history; separate session-local history per prompt without unifying command grammars.
23a. [Specify workspace/data preferences](.scratch/workspace-configuration/issues/01-specify-workspace-preferences-configuration.md) – default Project/View, startup filters, and persisted navigation preferences; follows local TUI configuration.
23b. [Reuse multiline layout in Comment Composer](.archive/2026-10-20/comment-composer-multiline-layout/issues/01-reuse-multiline-layout-in-comment-composer.md) – technically complete and archived; shared layout and Insert cursor with regressions for wrapping, blank lines, scrolling, and Unicode.
23c. [Restructure help and separate rich text](.scratch/help-structure/issues/01-rethink-help-structure-and-richtext-reference.md) – inconsistent indentation and poor readability; brainstorm and critically review grouping, navigation, and layout, with at least a separate rich-text help page. Design remains open; implementation has not been requested.
23d. [Fresh database and default path](.scratch/first-run-database/issues/01-support-first-run-database-and-default-path.md) – allow a fresh installation to start without an existing database or `--db`, using compatible initialization or a validated starter template. Define the default path and minimum starter records; preserve existing databases. Specification pending.

### Milestone 4

24. [Add View and Global Search navigation history](.scratch/navigation-history/issues/01-add-view-and-search-navigation-history.md) – `Ctrl+O` back and `Ctrl+I` forward through semantic TUI contexts; specify the restore contract first.
25. [Brainstorm timeboxed focus plans](.scratch/timeboxed-focus-plans/issues/01-brainstorm-timeboxed-focus-plans.md) – possible additive Vimkunja layer for limited, regular attention to ongoing Projects; discuss and prototype first.
26. [Investigate full Vim Normal Mode](.scratch/full-vim-normal-mode/issues/01-investigate-full-vim-normal-mode.md) – assess full Vim compatibility against effort, conflicts, and limited scope; follows Tier 1 motions.

### Backlog — milestone not assigned

- [Delete empty Labels](.scratch/label-deletion/issues/01-add-delete-empty-labels-command.md) – specify command, confirmation, and failure reporting; remains `needs-triage`.

## Publication

- Local startup repaired (2026-10-06): the previous default database had only 116 of 150 migrations. Upgrade with official VOA 2.7.0 was first tested on a separate copy, then applied to the production database at Jan's explicit request after a verified backup. Default path remains `internal/db/vikunja.db`; backup `internal/db/vikunja-pre-2.7.0-20261006.db`. Opt-in upgrade regression checks rejection, upgrade, data preservation, and Store reads; startup errors name the path, revision details, and upgrade guidance.
- Review fixes (2026-10-06): permanent deletion atomically reserves Task/alias numbers even with missing or stale counters. Startup validation requires complete unique indexes including Task/Bucket assignment and a single-column counter primary key. Regressions, full tests, vet, builds, and VOA/upgrade round trips pass.

- [Vikunja database compatibility: 2.7.0 only](.archive/2026-10-05/vikunja-db-compatibility/issues/01-verify-vikunja-2-3-to-2-7-compatibility.md) — **completed; publication compatibility prerequisite met**. Production revision checks, Store fixes, and regressions implemented; the [verification report](.archive/2026-10-05/vikunja-db-compatibility/report.md) documents fresh/baseline round trips with official VOA 2.7.0. Tests, vet, and both builds pass. Only 2.7.0, sequential use; permanent deletion of Tasks with Time Entries is blocked. Native Windows terminal tests remain open.
- [01 — License and documentation for GitHub](.archive/2026-10-07/publish-to-github/issues/01-publish-to-github.md) — **completed and archived**. License inventory (30 modules/128 additional notices), README, examples, and links reviewed; HTML write protection, configuration precedence, binary startup, and release boundaries clarified.
- [02 — Publication after the rich-text merge](.archive/2026-10-23/publish-to-github/issues/02-prepare-publication-after-richtext-merge.md) — **completed and archived**. Cleaned source published on `main`; public repository, matching remote commit, selected heads/tags, and successful Windows/Ubuntu Actions verified. No tags or backup branches published. Automatic binaries, a production starter database, and macOS binary approval remain outside this completed source-publication scope.

## Documentation

- [README.md](README.md) – introduction, controls, and vision
- `.scratch/` – sole source for open specs, bugs, and feature requests
- `.archive/` – completed requirements, tickets, and historical reports
