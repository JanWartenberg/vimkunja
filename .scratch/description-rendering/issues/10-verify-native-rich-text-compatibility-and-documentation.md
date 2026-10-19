# 10 — Native Rich-text compatibility, terminal acceptance and documentation

**What to build:** Gather actual Vikunja and terminal evidence, run the integrated native contract suite and document/tutorial the implemented feature before declaring M3 Rich-text complete.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** in-progress

**Implementation dependencies satisfied:** 03–09 completed on `feature/description-model`; 09 is verified and committed 2026-10-10. Integrated acceptance, capture/provenance and tutorial work can proceed.

**Contract:** [spec.md](../spec.md), [decision/handoff ticket 02](02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../corpus.md), [83 synthetic fixtures](../corpus.json), [keybinding contract](../../../docs/keybindings.md).

## Scope

- Start versioned VOA capture preparation early; it does not require waiting for the whole editor. Record actual getHTML output, Vikunja/Tiptap versions, capture steps and provenance, including Checkboxes, nested structure, Links, Code and Mentions. Never relabel synthetic examples as captured output.
- Add captures and minimized regressions with independent expected behavior. Golden changes require an explicit diff/contract explanation; do not bulk-regenerate from the serializer under test.
- Execute all relevant fixture classification/projection/serialization/edit assertions against the implemented modules. Report per-rule coverage and explicitly missing/unexecutable cases, not just an aggregate green test count.
- Add semantic Native round-trip properties, byte/identity/order/parent Opaque preservation properties and a small bounded parser/serializer fuzz run in normal CI. Discovered failures become deterministic regressions; no long scheduled fuzz infrastructure.
- Exercise save/discard/conflict/combined-save/UUID candidate failure flows end to end using disposable databases and confirm unchanged Task Done and unrelated editor surfaces.
- Record wide/narrow/short native TUI evidence: disclosure reflow, cursor anchoring, protected atoms, quote/list nesting, Code/Table scroll, soft/hard wrap, palette threshold and too-narrow draft preservation. HTML documentation screenshots are not native TUI execution evidence.
- Capture real German keyboard/Windows terminal events and interaction outcomes for accepted Code keys, AltGr/dead keys/Paste and literal-Tab alias. Record unavailable environments as outstanding evidence, not PASS or another product questionnaire.
- Update Help only for implemented bindings, docs/keybindings status, README usage and a dedicated markup-tutorial section using the real UI. Keep current tutorial isolated in disposable data and preserve existing lesson flows.

## Acceptance and validation

- Complete integration requires tickets 03–09; captures/case preparation may begin earlier. Attach a concise report distinguishing automated Native results, real VOA outputs, practical terminal results and remaining limitations.
- All 83 current synthetic examples are discoverable in the test matrix; new versioned captures are clearly separate. Random IDs are checked structurally (UUID v4, uniqueness, stability), not as literal strings.
- A documentation/review template may use data-task-id="<randomUUID>" as an explicitly illustrative placeholder. It is not a valid persisted UUID or executable golden. L10 visible Alpha/Beta/child output is deterministic despite this placeholder.
- Native tests, configured bounded fuzz, full go test ./... and both entrypoint builds pass. Real VOA/Windows checks are required for their respective compatibility claims; inability to capture is documented and never waived by synthetic probes.
- Help/README/tutorial teach syntax, selections, Code exits/Tabs, Opaque confirmation, recover/discard and conflict handling without raw-HTML editing. Only then report the supported feature as complete.

**Focused coverage:** Entire corpus and current traceability matrix, versioned captured VOA cases, fixture-derived compound editing journeys and real terminal evidence.

Run affected tests during implementation; before a commit run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The existing artifact probes are supplemental, not Native execution evidence. Required test commands and fixture mapping must be recorded in the completion comment.

## Boundaries

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. This ticket integrates and verifies 03–09, including capture-derived compatibility fixes. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.

### 2026-10-10 — Ticket 08 terminal evidence handoff

Synthetic Code tests cover grouped Text, ü/Ü, ]Space/[Space, backtick chunks, Paste/clipboard CRLF and literal-Tab prefix dispatch. No real German-layout or Windows decoder capture has been performed. Record actual AltGr-8/9, dead-key three/four backticks, ü/Ü, Ctrl+V interception and Ctrl+Q/Ctrl+B events and interaction outcomes here before terminal acceptance. Ctrl+Q and Ctrl+B have no existing application binding conflicts in Code Insert; terminal flow-control/interception remains an environment check. `tui.literal_tab_alias` defaults to `ctrl+q`, accepts `ctrl+b` or empty; no OS detection.

### 2026-10-10 — Ticket 09 inline execution handoff

Inline syntax, Inline-Code and Links are implemented with actual TUI dispatch/prepared-save execution for 40 goldens across eleven focused records, scoped Draft regressions and temporary-SQLite save adoption. Help, README and keybindings describe the implemented actions. Gather the remaining integrated per-rule evidence and tutorial here; all existing corpus records remain synthetic. Real clipboard-result handling is tested through an injected writer, not a host clipboard/environment acceptance claim.

### 2026-10-11 — automated integration and documentation started

Authorized to complete all work possible without questions in `/workspace/vimkunja-richtext-03`. Reuse Document/Draft, Store and actual TUI event test seams. No commit requested. Real Windows/German events and any unavailable VOA runtime evidence remain explicitly outstanding.

### 2026-10-11 — autonomous implementation and evidence

Implemented on the existing worktree/branch; **not committed**. Ticket remains **in-progress** pending the real environment/manual acceptance below. No product question is outstanding.

- `vimkunja-tutorial --markup` is a separate 23-step workshop using the actual UI and disposable SQLite data. `--markup --practice` removes lesson key restrictions for arbitrary keyboard/Paste acceptance on the same disposable fixtures. Both Ctrl+Q and `--literal-tab-alias ctrl+b` journeys run through UI events, save/discard, Opaque confirmation/history and blocked-source recover/clear. Saved HTML, unchanged Task Done and discarded recovery are asserted. General tutorial fixtures stay unchanged.
- All 83 synthetic import/serialization/projection records pass; 203 indexed Native edits execute **every non-null HTML golden**. The remaining 32 null-HTML action records use assertion contracts; the [ledger](../captures/native-execution-2026-10-11.json) explicitly lists their indices separately from the passing focused regressions. No claim of 235 individual action-driver executions. UUID candidate/reservation/failure, external conflict, combined-save rollback, clipboard success/failure, Code-line selection, palette geometry, semantic oracle and Opaque identity/parent/order tests are included in the full run. All mixed corpus documents additionally reject injected lost atom identity without mutating their origin or allocating UUIDs.
- Seven [actual upstream component captures](../captures/upstream-editor-2026-10-11.json), commit `82ee780161c6ed250cdb1fc4d2952b72464cb724`, frontend 2.3.0, Tiptap 3.27.1, with original inputs/version/hash/provenance. Unmodified actual TipTap.vue mounted in an isolated browser host; **not authenticated Task/backend evidence**. Native-edited Checkbox HTML was reimported and returned byte-identically.
- Capture-derived Checkbox scaffolding now remains Native where understood: text/nesting editable, controls skipped in navigation, checked state synchronized, split/move/conversion/Undo/save tested. Unsafe/unknown control attributes are protected locally, with the safe adjacent body still editable. Synthetic goldens unchanged.
- Fixed replacement inheriting neighboring marks, focused Mention copying its entire paragraph, and extra paragraph wrappers when splitting an inline plain-list entry. Permanent regressions added.
- [Actual TUI model layouts](../captures/tui-model-layouts-2026-10-11.json) and [Linux PTY smoke](../captures/linux-pty-smoke-2026-10-11.json) plus [unrestricted practice/four-backtick creation](../captures/linux-pty-practice-2026-10-11.json) distinguish synthetic size events/injected keys from physical terminal claims. README documents workshop, syntax-vs-Paste, Code exits/Tabs and recover/conflict flows. Existing Help/keybindings already describe implemented actions and were checked for consistency.
- Full tests including the artifact verification package, affected race tests, vet, both builds and the two 5-second fuzz targets passed. The bounded parser/edit fuzz steps are configured in Linux CI. Commands/reproduction and traceability are in [corpus.md](../corpus.md#integration-evidence--2026-10-11). No local user DB, schema or Go dependency changed.

**Manual/environment acceptance still required:**

1. In your German-layout Windows terminal, from this worktree run `go run -buildvcs=false ./cmd/vimkunja-tutorial --markup`. Complete the workshop. If Ctrl+Q is intercepted, use `go run -buildvcs=false ./cmd/vimkunja-tutorial --markup --literal-tab-alias ctrl+b`. Literal Tab should insert a source Tab, retab should use the next tabstop, and ü should leave Code without modifying its source.
2. Repeat with a tall/wide and a narrow/short terminal. The focused target must remain visible or show the too-narrow notice; widening must restore the draft/target. Check palette visibility, quote/list nesting, Code/Table horizontal scroll and readability. Automated reflow tests and model renders are supplementary, not a physical PASS.
3. Start the unrestricted disposable practice UI with `go run -buildvcs=false ./cmd/vimkunja-tutorial --markup --practice`. Enter, G, Enter opens the last Task; gn, Enter activates Description. Use gg, o to create a new empty paragraph. Directly type three and four backticks in separate runs; before Enter they remain literal, Enter creates one Code block. Record the actual dead-key behavior, AltGr-8/9 for `[Space`/`]Space`, ü/Ü, Ctrl+V and the selected alias. Do not substitute injected KeyPress/Paste messages for physical evidence.
4. In this unrestricted practice mode, check actual clipboard Paste in Code with multiline/tabbed text, plus direct typing of the same syntax markers: pasted markers must remain literal, typed agreed markers may create structure. Host clipboard behavior is still unverified here.
5. For a live Vikunja Task, if available, review a Description saved by Vimkunja in the web app, including Checkbox IDs/state, Code, Quote source and Mention. The component serialization capture proves that component boundary; authenticated Task/backend integration and your visual acceptance are still separate.

Only these environment/user observations justify the corresponding compatibility claims. No new design decisions are requested.

### 2026-10-15 — first manual acceptance blocker: ciw

User reported that AT-01 failed immediately: on Alpha beta, `ciw` did not change a word; `i` entered Insert and the `w` was inserted as text. Reproduced through the full `--markup --practice` entry path and disposable Store, not a pre-opened editor fixture. The Native handler had no change-word operator prefix; previous tests exercised Visual selection followed by c and missed the documented acceptance path.

Implemented Native `ciw`/`caw` and `diw`/`daw` using semantic word selection. A change-word plus the following Insert session is one Undo/Redo group until Esc. Cancellation, invalid prefixes, successive groups, grouped errors, formatting and protected Mentions are covered. Selecting a focused Opaque atom also exposed an insertion-boundary ambiguity and adjacent punctuation absorption; both are fixed without bypassing confirmation. Normal editable text no longer gets the generic read-only reader hint. Help and keybindings updated.

`TestMarkupPracticeChangeInnerWordFromFreshTask` executes fresh app startup → Enter/G/Enter/gn/Enter/gg → ciw → individually typed Hallo → Esc → u → Ctrl+R → :w, checking complete persisted HTML and unchanged Task Done. Focused UI tests check neighboring marks/Opaque bytes and cancellation. [Actual Linux PTY replay](../captures/linux-pty-ciw-2026-10-15.json) confirmed change/Undo/Redo/save/reopen in the built real program with injected keys. Full tests including artifact verification, affected race tests, vet and both builds pass. No physical Windows acceptance is claimed. Existing checklist actions remain unchanged; a running process needs restarting to load the fix. No commit requested.

### 2026-10-15 — broad review after first acceptance blocker

Reviewed the complete Rich-text branch from `8de9ba7` (before 03) through current HEAD plus all working changes. Two independent reviewers checked Spec and Standards; the main review replayed all 16 acceptance topics through fresh real application startup, injected keys/Paste and disposable SQLite. No commit requested.

**Spec — two confirmed findings, fixed:**

- P1: word selection in a parent list entry concatenated child-entry text. `ciw`/`diw` could therefore delete child text. SelectWord now uses contiguous leaf runs owned by the current logical block, including parent text resumed after a nested list. Tests cover both word objects, each entry, deletion and Undo.
- P2: word selection on standalone Opaque Tables/media returned an error. It now selects the whole atom; confirmation, exact-byte Undo and Redo are tested.

**Standards — one production finding and one test gap, fixed:**

- P2: partial block deletion within a Native Checkbox could delete its zero-width presentation controls and leave an unsavable draft. Controls now stay with their owning entry during partial deletion; whole-entry deletion still removes them. Model and actual-key UI tests cover save, adoption, Undo and unchanged Task Done.
- AT-15's unchanged-database assertion alone could pass even if draft discard/clear failed. It now checks each visible Blocked/empty-draft transition as well. The guided workshop must issue the real Quit message, not just consume all lesson keys.

**Instructions — corrected:** recovery already enters Insert. The acceptance Card's extra i was removed; the workshop now explicitly leaves Insert with Esc before :e!. All other acceptance sequences passed unchanged. The checklist remains `/workspace/tmp/acceptance-description-2026-10-12.html`.

`internal/tutorial/practice_acceptance_test.go` plus the AT-01 fresh-entry regression cover all 16 Card topics, with independent stored-HTML expectations and real save/reopen/discard. Link clipboard outcomes remain in injected-writer tests; physical Windows/AltGr/dead keys and host clipboard are still environment acceptance, not a claimed PASS. A shared test runner handles async commands/leader timers and actual Quit.

Final independent re-review found no further confirmed issues. Full tests including artifact verification, affected race tests, vet and both builds passed; bounded parser/edit fuzz passed. Ticket remains in-progress for the already recorded real environment/manual checks.

### 2026-10-16 — flat tutorial start menu

Implemented the requested single menu with four modes and Quit, without nested selection: Learn the basics, Practice the basics, Learn markup, Practice markup. `j/k` and arrow keys select; Enter starts the highlighted mode; q/Esc/Ctrl+C cancel. The general guided mode is selected initially. Database creation happens only after mode selection; Quit creates no database.

Direct starts: `--guided`, `--practice`, `--markup`, `--markup --practice`. General practice now uses the ordinary demo fixtures with unrestricted real UI input. Timeout flags and `--literal-tab-alias` alone retain the menu; selected markup modes retain their workshop settings. Conflicting guidance flags, invalid aliases and unexpected positional arguments are rejected. README, CLI help and STATUS describe the menu.

Validation: menu/CLI behavior tests were introduced failing before implementation; full `go test -buildvcs=false ./...`, both entrypoint builds and targeted vet pass. Ten Linux PTY journeys exercise menu Quit, all four menu selections, all four direct starts, and configuration-only flags followed by markup selection. They verify no database at the menu, correct SQLite fixtures, guidance presence/absence, normal exit and temporary-file cleanup. Practice sessions exit using unrestricted `:qa`; guided sessions exit with Ctrl+C. Two independent read-only reviews (bugs/risks and consistency/validation) found no actionable issues. Physical Windows terminal acceptance remains outstanding. No commit requested; existing Rich-text acceptance work remains open.

### 2026-10-16 — guided Link input and instruction contrast

The Link lesson now accepts arbitrary URL typing, editing and Paste instead of requiring the example character by character. Enter advances only after the real URL prompt closes; invalid URLs remain editable for retry. Empty input unlinks and Esc can cancel. The suggested URL is illustrative. All tutorial action instructions are bright white and bold to distinguish them from explanatory notes.

Regression tests run both complete workshop alias journeys with a different URL and exercise invalid input, Backspace correction, pasted replacement, confirmation and exact saved HTML through the real UI. Affected tutorial, TUI and command tests pass. No commit requested.

The confirmation footer now states the actual consequence instead of “Preserved content would change”: one or more protected objects will be DELETED, a structural change retains the objects, Clear deletes the entire Description, and Recover replaces HTML with readable text. Consequences are derived from the proposed edit before rollback; no hidden HTML/attributes are shown. Existing quote-metadata loss confirmation remains specific. Real UI tests cover deletion/replacement, cancellation, structure retention and both blocked recovery actions.

### 2026-10-16 — tutorial command completion and recovery preview

Guided Ex-command sequences forward Tab to the real application. The lesson skips the completed command characters only when the real prompt resolves to the expected command (including aliases); ambiguous completion retains the current position. Recovery teaches `:descriptionrec`, Tab, Enter, y; fully typed commands still work. Both complete workshop alias journeys now exercise completion, with focused ambiguity/alias regressions.

Recovery confirmation shows a bounded readable-text preview (Broken in the workshop), explicitly says HTML/formatting/objects are replaced on save, and warns when no useful text would yield an empty Description. Hidden script content is not included. Cancellation retains the original source. No commit requested.

### 2026-10-16 — complete pending-change verification

Reviewed all tracked and untracked pending changes against HEAD f7c6eb7 using implement-and-verify. Two independent read-only reviewers per round checked bugs/risks and consistency/validation. Round 1 found one medium issue: generic footer tail truncation hid the consequence (notably EMPTY) at narrow widths. Fixed with a consequence-first compact confirmation summary; full preview is used when it fits. Pending warnings remain visible after unrelated keys. A real-View regression failed before the fix and passes at 80 and 54 columns. Both fresh round-2 reviewers reported no remaining confirmed actionable findings.

Validation passed: `go test -buildvcs=false ./... ./.scratch/description-rendering/verification`, race-enabled Description/TUI/Tutorial/Store tests, `go vet ./...`, both entrypoint builds with `-buildvcs=false`, and `git diff --check`. Bounded parser/edit fuzz passed (5 seconds each, 2 workers); no dependency or local user database changes. Physical Windows/German-keyboard, actual host clipboard and live Task/backend acceptance remain the previously recorded gaps. Ticket stays in-progress; all work remains uncommitted.


### 2026-10-16 — user tutorial feedback and branch checkpoint

Jan reports completing the Rich-text tutorial successfully. This records that tutorial journey only; the separately listed Windows/clipboard/live-Vikunja observations are not implicitly accepted. Jan identifies the keystroke restrictions and unclear task expectations as a general tutorial problem and agrees to trial free input, outcome recognition, reset and explicit continuation.

The previously verified Rich-text/workshop state is committed as `10ff5d2` on `feature/description-model`. The task-based text-input pilot is separate work on `feature/tutorial-task-workflow`; see [task-based tutorial work](../../tutorial-task-workflow/issues/02-convert-all-guided-tutorials.md). Both existing curricula remain available for comparison. Ticket 10 remains open for the recorded environment/compatibility checks.


### 2026-10-17 — extended fuzz audit and reviewed fixes

Completed the separately authorized [Rich-text fuzz audit](../fuzz-audit-2026-10-17/report.md): six confirmed defect groups fixed with focused regressions. The fixes cover trailing-space normalization, typed LF handling, Code edit bounds, invalid UTF-8 input, cursor mapping across typed delimiters, and empty editor-anchor preservation during formatting. Two new fuzz targets cover structured Unicode/history editing and save-candidate identity/preservation; minimized discoveries are retained as normal regression inputs.

Independent reviews and follow-up fixes are complete. Final structured five-minute campaign passed 380,240 executions; passing campaigns across successive source revisions recorded 16,559,825 executions. Full tests including artifact verification, affected race tests, vet and both entrypoint builds passed. Before the requested commit, the reviewed audit source was unchanged; full tests, vet and builds were rerun against the current branch.

Jan subsequently authorized committing the reviewed fixes, tests and report. This entry is delivered with that commit. No product decisions remain from the audit. Ticket 10 stays in-progress for its existing environment/compatibility observations; automated/modelled input tests do not imply physical Windows/clipboard or authenticated Live-Vikunja acceptance. The local HTML walkthrough and raw execution logs are review artifacts and are not part of this commit.


### Current manual acceptance scope after cursor follow-up

Jan accepts the task-based Rich-text tutorial, explicitly accepts Steps 11 and 13 including Retab, and confirms the shared insertion cursor's Shift+A line-end fix. Do not request another full tutorial run. Remaining user/environment observations are: actual host clipboard multiline/tabbed Code Paste and Copy Link; unrecorded German/Windows terminal edge cases (dead-key backticks, AltGr bracket alternatives, ü/Ü, selected literal-Tab alias); one narrow/short resize check; and an actual Vimkunja-save → Vikunja web review → reopen round-trip on a disposable Task. Record only observations actually made; tutorial completion does not prove backend or clipboard compatibility. The imported empty Code typing defect in ticket 11 is agent implementation work and does not require Jan to investigate it.


### 2026-10-19 — Jan's partial acceptance and AT-06 blocker

Jan submitted the checklist summary against its displayed commit 059559f (environment unspecified): AT-01 plain Description formatting, AT-02 empty Description editing, AT-03 Description OS Clipboard, AT-04 Title OS Clipboard and AT-05 Comment OS Clipboard are PASS. AT-06 is explicitly FAIL after screenshot evidence of Code Paste rendering literal carriage-return escapes in a single line. AT-07 Link URL Clipboard, AT-08 remaining physical German-keyboard paths, AT-09 resize acceptance and AT-10 live Vikunja round-trip remain NOT ASSESSED. Current human verdict: **5 PASS / 1 FAIL / 4 NOT ASSESSED — NOT READY**. The earlier tutorial and steps 11/13 remain accepted; no full tutorial rerun is requested.

AT-06 is implementation work, not an unresolved product question. Automated reproduction isolates CR-only source line endings: both terminal PasteMsg and Normal Space+p reproduce the screenshot, while LF/CRLF already pass. The fix recognizes CR alongside LF/CRLF in Code projection, navigation, line bounds, copy/selection, insertion and line edits while preserving exact input bytes and Tabs. CRLF split across preserved wrappers remains one logical newline. See ticket 11 for regression/validation details. These automated results do not turn AT-06 into a manual PASS; only the focused real terminal Paste test needs repeating after the fix.


### 2026-10-19 — updated acceptance and live reload blocker

Jan subsequently accepts AT-07 (Link URL OS Clipboard) and AT-08 (physical German keyboard/Code exits), alongside the existing AT-01–05. A real compatibility.db journey confirms TUI save → Browser refresh and Browser save → restarting Vimkunja. Jan then reports that in-app :descriptionreload rejects an unchanged draft with “Task draft changed while reloading”. This later failure supersedes the initially submitted AT-10 PASS: the live reload portion is open again. AT-06 retains its earlier FAIL pending a repeat after the CR fix; AT-09 remains unassessed. The tutorial/steps 11/13 remain accepted.

The reload failure is reproduced with disposable SQLite and the real Ex-command after loading an empty Task-label list. Snapshot cloning converted a non-nil empty LabelIDs slice to nil; reflect.DeepEqual treated that representation-only difference as an intervening edit. CloneTaskDraft now uses slices.Clone, preserving nil/empty representation and copying nonempty slices independently. Browser changes can reload in place; actual edits made after dispatch still block replacement. No product rule or database behavior changes. Ticket 11 records the regressions. Manual AT-10 in-app reload needs a focused recheck after restarting onto the fix.

Jan confirms the fixed in-app :descriptionreload now works and explicitly requests the commit. AT-10, including Browser → running TUI reload, is manually accepted. AT-01–05/07/08 remain PASS; AT-06 still needs the focused post-fix repeat, and AT-09 remains unassessed.
