# Rich-text fuzz audit — 2026-10-17

Status: completed

## Outcome

Six confirmed defect groups fixed with focused regressions. Two new fuzz targets
added; eight targets exercised in total. Independent reviews and final integrated
checks pass. No outstanding product decision. The fixes, tests and this report are delivered together.

Across passing campaigns at successive source revisions, Go recorded 16,559,825
executions (not a count of distinct inputs). Failing discovery runs and false
test-property findings remain saved separately; they are not counted as passes.

## Scope and baseline

Jan authorized parallel extended Rich-text testing, focused regression tests,
concrete fixes and independent review. The initial investigation did not request a commit; Jan subsequently authorized committing the reviewed changes.
Existing tutorial changes are outside this audit and must be preserved.

Baseline patch, status, source snapshot and execution logs are retained in
`/workspace/tmp/richtext-fuzz-20261017/`. The source snapshot covers
`internal/description`, initially unchanged relative to HEAD.

## Confirmed fixes

### F01 — non-idempotent trailing-space normalization

Splitting text following a Link and deleting the Link's final character could
leave adjacent trailing-space leaves. Normalization trimmed only the final leaf;
a later normalization exposed another trailing space, so the accepted edit failed
HTML/PrepareSave with ErrRoundTrip. Existing inline fuzz independently found
`*00** ** *` with the same root cause.

`normalizeFlow` now trims backwards through disappearing trailing text leaves
while retaining whitespace explicitly protected immediately after imported hard
breaks. An initial proposed fix lost that whitespace; independent cross-review
caught it, and exact hard-break whitespace regressions cover the correction.

Evidence: `edit-history-linkspace-red.log`, `edit-history-inline.log`,
`edit-history-regressions-green.log`, `parser-view-normalizer-review.log`.
Deterministic tests: `TestDraftDeleteLinkEndBeforeTrailingSpaceRemainsSavable`,
`TestNormalizeTrailingHardBreakWhitespaceIsPreserved`; fuzz seeds retained.

### F02 — typed LF could panic before existing paragraph text

`Type("\n")` before existing paragraph text split the paragraph and then tried to
mark an inserted text byte at offset -1. Prose typed LF now follows Split before
text-origin marking. Imported Code keeps literal line feeds. Exact output and
single-Apply Undo are tested.

This is a module/grouped-text robustness finding: normal UI Enter already uses
Split and Paste uses Insert; no ordinary physical Enter crash is claimed.

Evidence: `oracle-insertion-red.log`, `edit-history-newline-red.log` and focused
green logs. Test: `TestDraftTypedNewlineSplitsWithoutPanic`.

### F03 — escaped Code selection accepted invalid source ranges

Draft API callers could select Code lines and move one endpoint into a preceding
empty paragraph. Replacement could accept a negative range and leave the cursor
inside a multibyte replacement character. Swapping endpoints then Retab could
panic before the initial lower-level guard ran.

Code edits now validate ranges at both the selected-source boundary (before
slicing) and the lower edit boundary used by replacement. Regression tests cover
replacement plus ten related Code operations, exact unchanged state and history.
The second guard was added following independent review.

The normal TUI additionally constrains Code selection movement. This does not
establish a physical-keyboard crash path.

Evidence: `parser-view-code-selection-red.log`, `code-review-retab-red.log`,
`parser-view-code-line-actions-{red,green}.log`, `final-review-r2.md`.

### F04 — invalid UTF-8 input left an unsavable draft

Insert/Type/replacement accepted invalid bytes. Actual UI Paste dispatch accepted
them too, after which preparing a save failed. Apply now rejects invalid UTF-8
before editing, rather than silently repairing bytes. Existing valid text behavior
is unchanged. Focused tests verify state, a nonempty selection, ID allocator and
exact redo restoration, plus both UI Paste routes and visible session error state.

Evidence: `invalid-utf8-red.log`, `invalid-utf8-green-final.log`,
`root-review-r2.md`. Tests: `TestDraftRejectsInvalidUTF8WithoutChangingStateOrHistory`
and `TestNativeInvalidUTF8PasteKeepsDraftSavable`.

### F05 — closing inline syntax run could misplace a Unicode cursor

Start with `é👩‍💻x`, type `**`, move inside those active markers, then type
`**x**`. A consumed closing marker run crossed the cursor. Counting only complete
tokens before the cursor advanced the mapped position into the following UTF-8
text. Count the overlapping consumed prefix instead.

Regression verifies exact formatting and that following insertion produces
`<p><strong><em>x</em></strong>Yé👩‍💻x</p>`. It fails against the initial source
archive, confirming this was not introduced by concurrent feature work.

Evidence: `edit-history-syntaxcursor-{minimal-red,baseline-red,green}.log`,
`final-review-r3.md`. Test:
`TestTypedClosingDelimiterAroundCursorKeepsFollowingUnicodeBoundary`.

### F06 — formatting removed an unrelated empty editor anchor

Deleting the sole text from a Quote leaves an empty Text node carrying an editor
anchor. Formatting text in a following paragraph rebuilt inline runs and removed
that unrelated node. The empty Quote then acquired a different block extent,
shifting the following selection endpoint from valid byte 3 to invalid byte 2.

Keep empty Text runs during formatting, retaining editor anchors and container
extent classification. The exact six-line `formatInline` change preserves the
concurrent formatting feature. The minimal regression fails against the initial
source archive, verifies unchanged cursor/selection, then inserts Y precisely
between the combining-accent grapheme and emoji inside the formatted text.

Evidence: `edit-history-formatcursor-{minimal-red,baseline-red,green}.log`,
`edit-history-selection-audit.patch`, `final-review-r4.md`. Test:
`TestFormattingPreservesCursorAfterEmptiedQuote`.

## Test improvements and interpretation

- New `FuzzDraftStructuredHistory`: valid document shapes; combining accents,
  ZWJ emoji, flags and CJK; real cursor/selection positions; 19 edit operations;
  failed-edit rollback, accepted serialization, valid cursor and exact Undo/Redo.
- New `FuzzSaveCandidateIdentityAndPreservation`: stable first Checkbox IDs,
  missing/duplicate ID repair, reserved-ID collision, exact protected subtree,
  immutable preparation, unchanged native meaning beyond ID repair and stable
  reopened saves. The deterministic UUID generator avoids nondeterministic tests.
- Existing projection fuzz additionally checks terminal controls in spans/hints.
- Existing corpus/golden tests remain independent expected-output evidence.
  Parse/Serialize idempotence alone cannot prove semantic preservation.
- One new-property failure concerned the valid zero cursor of an empty document;
  corrected the property, not production code. Its input remains a regression seed.
- Speculative imported-URL control leakage was disproved by a passing check;
  existing projection already sanitizes it.

## Review and ownership

Root changes had two independent review rounds. Edit/history changes had separate
cross-review; Code guards had two review rounds; the later syntax-cursor fix had
another independent review. Reports are retained with the logs. Review findings
were fixed and rechecked; no product decision has been identified.

The empty-Text fix had a fresh independent reviewer after a provider content flag;
the task was explicitly framed as improving our own local editor's ordinary
functional correctness. The replacement review completed with no findings.

Audit-owned production files: `draft.go`, `draft_code.go`, `draft_syntax.go`,
`normalize.go` and the localized empty-Text branch in `draft_selection.go` under
`internal/description`. Other audit files are focused tests,
two new fuzz targets, minimized inputs and this report.

Concurrent feature modifications to selection/structure, tutorials, help,
keybindings and related UI/docs were preserved and are excluded from audit change
claims. The frozen verification tree includes their captured state; later changes
from other work are outside that verification snapshot.

No merge, push, dependency change or user database/schema change performed by this audit. Concurrent tutorial/feature work was separately committed as `985a40d`, followed by `0e3286e`. Jan authorized this audit's fixes, regressions and report to be committed together. The HTML walkthrough under `tmp/` and raw logs remain local review artifacts.

## Final verification

Final integrated checks use the frozen `verified-source-r4/` tree and its SHA-256
manifest, excluding local database files. All twelve audit-owned source/test
files match the live worktree at completion (`verified-audit-file-check.json`).

- Full tests, including artifact verification: PASS (`completed-tests.log`).
- Race-enabled Description, TUI, Tutorial, Store and tutorial CLI tests: PASS
  (`completed-race.log`).
- Vet and both entrypoint builds: PASS (`completed-vet.log`,
  `completed-build.log`, `completed-verification.json`).
- Final structured-history campaign after all fixes: PASS, 301.233 seconds,
  380,240 executions (`edit-history-structured-round5.log`).
- Affected inline campaign after delimiter correction: PASS, 61.188 seconds,
  379,979 executions (`edit-history-inline-round3-60s.log`).
- Reviewed save-candidate campaign: PASS, 300.068 seconds, 1,334,014 executions
  (`save-fuzz-reviewed.log`).
- All original targets had five-minute campaigns; rebuilt shorter campaigns,
  replayed minimized inputs and exact-output regressions cover subsequent fixes.
  Per-campaign statuses/counts are in `fuzz-results.json`. Git whitespace check
  passes; no temporary diagnostic logging remains in modified production files.

The F05 Unicode case is additionally reproduced through real model.Update input
dispatch (grouped text and Left navigation), with baseline failure and fixed pass
saved as `ui-syntax-red.log` / `ui-syntax-green.log` and a permanent TUI regression.

Useful reproduction commands, from the final frozen tree with Go on PATH:

`go test -buildvcs=false ./... ./.scratch/description-rendering/verification`

`go test -buildvcs=false -race ./internal/description ./internal/tui ./internal/tutorial ./internal/vimkunja ./cmd/vimkunja-tutorial`

`go test -buildvcs=false ./internal/description -run '^$' -fuzz '^FuzzDraftStructuredHistory$' -fuzztime=5m -parallel=2`

Full intermediate reports, red/green evidence, source archives, minimized inputs,
review reports and command logs are retained under
`/workspace/tmp/richtext-fuzz-20261017/`; `report-checkpoints.md` preserves the
investigation checkpoints. Earlier `final-*` and `snapshot-*` failures are
superseded discovery evidence; the completed verification above is the final
verdict.

Before the requested commit, all reviewed audit source/test files still matched the verified snapshot. Full tests (including artifact verification), vet and both builds were rerun on the current branch; logs are `pre-commit-tests.log`, `pre-commit-vet.log` and `pre-commit-build.log` in the evidence directory.

## Decisions and remaining limitations

No outstanding product decision. Invalid byte input is rejected atomically using
the existing error path, without adding a new recovery or normalization feature.
Physical Windows/German-keyboard behavior, real host clipboard integration and
authenticated Vikunja roundtrips remain outside these tests. Synthesized Paste
messages exercise real UI dispatch but do not prove OS clipboard behavior.

Distinguish test-oracle limitations, reproducible production defects, fixes and
unverified environments. Linux fuzzing does not validate physical Windows
keyboard/clipboard behavior or authenticated Vikunja compatibility.
