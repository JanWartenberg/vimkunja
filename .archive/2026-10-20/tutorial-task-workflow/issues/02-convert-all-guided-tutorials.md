# Convert both guided tutorials to result-based tasks

Status: completed

Implementation, automated verification and manual acceptance are complete. The Rich-text tutorial and its tracked follow-ups are accepted (AT-01–10 PASS). Jan accepted the remaining general tutorial tasks 9–26 in the cleanup checklist on 2026-10-20; the earlier first-third acceptance remains valid. Ticket completed and archived.

## Agreed behavior

Jan accepts the pilot workflow and explicitly requests implementation with implement-and-verify for every tutorial. Replace the general and markup keystroke gates with concrete tasks, free application input, result recognition, F5 reset of the current task and F8 explicit advancement after success. Practice modes remain unrestricted application sessions.

Keep the display to task + controls; identify the target as Project/Task/Description and by its visible name. Use a colored left marker and bold task text without a filled background (color treatment remains adjustable). Put context and optional F1 help into short confirmed modals. Enter/Esc dismisses the modal without reaching the application.

Each task starts from an independent disposable database and real UI state; F5 restores that task, including saved/deleted data, draft, editor history, semantic sub-goals and selection. F8 never advances an incomplete task, nor advances during text entry without the explicit function key. Success is latched to permit further exploration. Final F8 shows completion; Ctrl+C exits. Late results from old tasks/resets cannot affect the new task or terminate it.

The curriculum groups the existing general feature topics into coherent outcome tasks and covers the existing rich-text workflow, including arbitrary valid Link URLs, word changes, local history, Checkboxes, Quotes, Code/literal Tab/retab/exits, protected-object confirmation/Undo, save/discard and blocked-source recovery/clear. Optional key examples remain hints rather than grading rules. Multi-stage goals observe meaningful application states (for example Done → Undo → Redo), not pressed keys.

The common runner owns fresh fixtures and setup; setup events are sent to the real UI only before learner work. Read-only SessionSnapshot describes real drafts, saved baselines, focused/selected Tasks, Views and visible interaction states. Outcome recognition never calls mutation APIs or manufactures editor state. Tests exercise Update/View and disposable Store boundaries already used by the pilot, plus CLI/menu behavior.

## Scope limits

No schema/dependency/user database changes. Original implementation scope excluded merge/push/commit; Jan authorized a separately reviewed feature checkpoint on 2026-10-17. The previous tutorials remain recoverable on `feature/description-model` at `10ff5d2`. `--tasks` remains a compatibility alias for general guided mode; the extra pilot menu entry disappears because both Learn entries now use the shared workflow.

## Validation

Complete real-UI journeys, reset/negative/alternate-path coverage, full Go checks and terminal smoke pass. Three independent review pairs checked bugs/risks and consistency/validation; the third pair reports no actionable findings. Physical Windows/keyboard/clipboard acceptance remains manual.


## Implementation and review evidence — 2026-10-16

The shared runner now serves both Learn entries (26 basics tasks and 22 markup tasks). Each F8 transition and F5 reset rebuilds disposable data and prepares the actual application through normal UI events. Both practice entries remain the real unrestricted application. The pilot is [archived](../../../2026-10-16/tutorial-task-workflow/issues/01-result-based-text-input-pilot.md).

The integration journeys visit every task through actual Update/View dispatch, reject early F8, reset every task, solve it and explicitly advance. Both literal-Tab aliases run through the complete markup course. Additional real-UI cases cover alternative inline Title editing, Paste/Backspace correction, wrong Task identity, modal confirmation/Paste isolation, reset after save/delete and during Insert, old-session Quit rejection, final completion and database cleanup. The public markup constructor's default prefix is also tested by creating/editing real Code.

Independent review R1 found three medium false-goal issues: filtering counted as deletion, renaming counted as creation, and a valid Search query ‘Buy’ was rejected. R2 exposed Project moves also being counted as deletion and independent bold/italic copies of ‘Both’ passing combined formatting. All have failing-before-fix UI regressions and are corrected. Successful creation/deletion identity now comes from completed application writes, not inferred list absence or a typed name. Bold-italic recognition retains both mark ancestries on the same text. Search recognizes the target after actual results without prescribing query words.

Built-program Linux PTY journeys traverse all basics tasks at 120x40 and all markup tasks through the menu at 54x24, opening help, resetting and advancing each task. They also check both Practice modes, menu-without-database, selected Literal-Tab override, normal exits and owned-file cleanup. These are injected terminal events; physical Windows/function keys, host clipboard and live Vikunja remain separate manual acceptance.


Final validation after all fixes: `go test -buildvcs=false ./... ./.scratch/description-rendering/verification`; race-enabled Tutorial/TUI/tutorial-CLI tests; `go vet ./...`; builds of both entrypoints; `git diff --check`. All passed. The final binary repeated the full 26-task wide basics journey and 22-task narrow markup/menu journey successfully after the receipt/formatting fixes. No further production changes followed these checks. Current branch remains `feature/tutorial-task-workflow`; no commit, merge or push performed. The main worktree’s unrelated clipboard planning is untouched.


### 2026-10-16 — skip chapters with F12

Jan requests F12 to skip tasks, including repeated presses from introductory modals (7 presses reach task 8). Both guided courses now share this unconditional skip; F8 still requires success. Each skipped task disposes its current draft/demo database and starts the next independent fixture. Skips are recorded only after a successful transition and the end panel reports their count instead of claiming the entire tutorial was completed. F5 repeats the last task and clears its skipped status for a successful retry. F12 after finishing does nothing; released keys do not advance. The two-line strip now advertises F12; exit remains visible in context/help/completion.

Focused real-UI tests cover both courses, 7 rapid F12 presses with delayed out-of-order old initialization results, context/help/Insert handling, old database cleanup, incomplete F8 protection, end-of-course skipped totals and passing a previously skipped final task after F5. Jan has manually checked roughly the first third of the general course and reports it works well; its granularity is accepted for now. General tutorial completion is lower priority; the new rich-text task workflow and existing live-environment acceptance remain open.

F12 validation passed: full Tutorial and tutorial-CLI tests (including both complete courses and both literal-Tab aliases), both entrypoint builds, targeted vet and diff whitespace check. Built-program Linux PTY checks at 54x24 confirm seven rapid F12 presses land on task 8 in both courses, repeated skipping reaches the correct skipped-total end panel, and exit removes all owned demo databases. No commit.

### 2026-10-17 — acceptance and separate feature checkpoint

Jan reports that the Rich-text tutorial otherwise fits and accepts the overall journey. The Code-whitespace cursor defect and the unconfirmed physical-key Backtick trigger are explicit open subtasks in [Rich-text follow-ups](../../../2026-10-19/description-rendering/issues/11-format-visual-selections-and-convert-touched-paragraphs.md#manual-acceptance-and-remaining-subtasks--2026-10-17). The general course remains partially manually checked; its remaining chapters are lower priority.

The checkpoint also includes productive v/V formatting and paragraph-wide Checkbox conversion, the framed persistent Space palette with an active/expired cue, and Ex Tab completion to the longest shared prefix while retaining candidates. The separate Fuzz audit and its fixes/tests/corpus are excluded.

Pre-commit review used `review-changes`, inspecting staged feature changes and unstaged audit changes separately. No additional actionable findings beyond the two explicitly retained subtasks. The exact staged source was exported with checkout-index into an isolated directory: full `go test ./... ./.scratch/description-rendering/verification`, race tests for Description/TUI/Tutorial/Store, `go vet ./...`, both entrypoint builds and staged whitespace checks all passed without any Fuzz-audit fixes, tests or corpus. Documentation records the acceptance and open subtasks; no code changed after these checks.

### Modal progress follow-up

Jan requests the current chapter while the tutorial modal covers the task strip. Introductory and F1 help modals now display current / total at the top right, including after skipping chapters; long titles yield space to the counter. Actual runner coverage checks both course totals at 54x24. Rich-text Steps 11 and 13 are manually accepted, including Retab and the visible muted Tab cue tracked in ticket 11. Jan authorizes committing this follow-up separately from the subsequent shared insertion-cursor work.

### 2026-10-19 — current acceptance and cleanup sequence

Historical references above to open Rich-text cursor/Backtick/environment subtasks are superseded by final AT-01–10 acceptance and closure in 0b29408. No repeat Rich-text tutorial run is required. General tutorial manual coverage remains partial. Review whether any actual subtasks remain, and separate or close them with evidence, under [release subtask cleanup](../../release-ticket-cleanup/issues/01-reconcile-open-subtasks-before-publication.md), after the URL-Copy shortcut and before finishing GitHub publication.

### Cleanup handoff — 2026-10-20

All Richtext subtasks are closed under final AT-01–10 acceptance. The only
unrecorded manual scope here is the remainder of the general 26-task course.
The earlier report says roughly the first third rather than an exact last task;
the conservative resume is task 9 (eight F12 skips from task 1). Test tasks 9–26
without further skips; intentional skipped tasks 1–8 are not counted as newly
passed. Each remaining task has a separate entry in the
[cleanup acceptance](../../release-ticket-cleanup/acceptance.html).
F5 retry, F1 hints and F8 advancement use the normal course controls. The
automated runner already traverses every course task through real UI/Store
boundaries; this remaining check is physical terminal usability, not missing
implementation.

## Final acceptance — 2026-10-20

Jan reports all 20 cleanup checks PASS, including AT-01–18 for general course
tasks 9–26. This closes the previously outstanding physical usability scope.
Earlier tasks are covered by the existing first-third acceptance; skipped tasks
are not relabeled as newly passed. Richtext acceptance remains unchanged.
