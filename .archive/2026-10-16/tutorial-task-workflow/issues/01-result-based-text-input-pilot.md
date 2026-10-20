# Result-based tutorial: text-input pilot

Status: complete — expanded in ticket 02

## Scope and acceptance

Implement the first agreed vertical slice before converting both existing tutorials. Preserve the checked rich-text and keystroke tutorials on `feature/description-model` at `10ff5d2`; build this pilot on `feature/tutorial-task-workflow`. No merge into main, public push or final pilot commit is requested.

- Display Topic and Task separately: change the Title of LEARN-1 (initially “Practice the Vimkunja tutorial”) to `foo` and return to Normal mode. Saving is optional.
- Forward all ordinary keyboard input, Paste, navigation, corrections and application commands to the actual application. F1 opens a hint modal; the suggested route is optional.
- Recognize the real draft's Task identity, exact Title and Normal state. Do not judge the sequence of keys. Success is latched until reset; further experimentation is allowed.
- Show congratulations and require F8 to continue. For this one-task pilot, continue reaches the completion panel; full general/markup conversion is follow-up work after user feedback.
- F5 restores the task's complete initial state, including any saved/deleted demo data, editor state and completion. Explain that it resets the disposable session.
- Ignore asynchronous results from the previous session after reset. Exit and reset clean up owned temporary databases; no user database or schema changes.

## Validation seams

Use real tutorial Model Update/View with a disposable SQLite Store, from fresh startup. Observe rendered goal/success/reset/finish and existing Store behavior; test alternate input including Paste and correction. CLI mode parsing and start-menu selection are adjacent public seams. No internal editor mutation or mocked application on the main journey.

## Remaining work

Convert the general and markup curricula only after the pilot's learning flow is accepted. Real Windows/function-key/clipboard acceptance remains manual.

## Verification — 2026-10-16

Implemented on `feature/tutorial-task-workflow` in `/workspace/vimkunja-richtext-03`; pilot changes remain uncommitted. The previous Rich-text/workshop changes are secured on `feature/description-model` at `10ff5d2`. The main worktree and its unrelated clipboard planning remain untouched.

Tests were introduced failing before implementation. Real Update/View journeys cover free exploration, Paste/Backspace correction, exact Task identity, delayed success until Normal, explicit F8, subsequent experimentation, help at 54 columns, reset during Insert, reset after save/delete, completion repeatability, stale pending Quit and database cleanup. An independent review exposed the valid List/Overview inline-title route being ignored. Added a failing inline-entry journey, then broadened the read-only Title snapshot to observe the focused Task in both Detail and browsing contexts. No key sequence is enforced. The hint uses the actual supported `0 C` replacement operation.

Two independent read-only reviewers per round checked bugs/risks and consistency/validation. Round 1: inline-result detection (medium), missing visible full-reset scope (low), README/menu ordering (low), all fixed. Both fresh round-2 reviewers reported no remaining actionable findings.

Passed: full tests including `.scratch/description-rendering/verification`; race tests for Tutorial/TUI/tutorial CLI; `go vet ./...`; both entrypoint builds; `git diff --check`. Tests/builds used `-buildvcs=false` in this root-owned, Windows-hosted checkout.

Built-program Linux PTY journeys pass direct `--tasks` at 120x40 and menu selection at 54x24: F1 hint, free title replacement and correction, success after Esc, explicit F8, F5 repeat/reset, Ctrl+C exit and temporary database cleanup. Keys were injected into a PTY; these are not physical Windows/function-key/host-clipboard acceptance. No production database, Go dependency or existing SQLite schema was changed.


### 2026-10-16 — compact task display

Jan rejects the dense five-line panel and white-on-white emphasis. The persistent strip now contains only the concrete task and one controls line. The task uses a cyan background with dark text. Initial identification/reset context moves to a centered, two-sentence dialog confirmed with Enter or Esc; F1 opens a separate short hint dialog. Modal confirmation and Paste stay out of the application. Reset does not repeat the introduction. Success and completion replace the task line rather than adding more lines.

Regression coverage checks the two-line strip at ordinary width, confirmation isolation, modal Paste isolation and all existing free-input/reset/inline-edit journeys. Narrow terminals may wrap the two logical lines.

Jan clarifies that “set title” leaves the object ambiguous. The task line now explicitly says to rename Task “Practice the Vimkunja tutorial” to “foo”; Esc moves into the controls line. The visible Task name identifies the target without requiring identifier lookup.


### 2026-10-16 — pilot accepted

Jan accepts the task/reset/continue workflow and requests applying it to all tutorials. Full implementation and remaining verification are tracked in [ticket 02](../../../2026-10-20/tutorial-task-workflow/issues/02-convert-all-guided-tutorials.md); this pilot is superseded by the shared course runner.
