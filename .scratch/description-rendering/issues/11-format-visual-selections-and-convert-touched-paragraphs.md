# Format Visual selections and convert touched paragraphs to Checkboxes

Status: in-progress

## User request — 2026-10-17

The markup task “Turn Check item into a checked Checkbox” rejected V → Space x with the obsolete “belongs to a later ticket” message. Jan requests the fix in the productive TUI: selected text/blocks must support Bold/Italic/Underline/Strike, and a Checkbox conversion must cover each logical paragraph touched by the selection.

## Implemented

- Both characterwise v and blockwise V selections accept Space b/i/u/s and Space 0 for inline formatting/removal.
- Space x on a prose selection converts every touched logical paragraph/list entry to a Checkbox, including partially selected endpoints; existing Checkboxes preserve checked state and identity. New adjacent containers merge locally; imported container metadata is preserved. One Undo restores the whole conversion.
- Code keeps its source semantics: V Space x/c still outdents/indents source lines. Code-containing inline formatting and incompatible structural conversion reject atomically. Protected objects retain existing confirmation and byte-preservation behavior.
- Normal Space x continues toggling the current paragraph/list entry into/out of a Checkbox; plain x checks/unchecks the current item. Selection does not check new items automatically.
- Palette, Help, keybindings and markup-course journey now include the selected-block route.

## Validation

New actual-key TUI regressions were red before implementation for V + all four inline marks and v/V multi-paragraph Checkbox conversion. Focused TUI/Draft tests cover reverse/partial selection, nested Quote order, existing checked IDs, idempotent selected conversion, one Undo, Opaque confirmation/cancellation, Code rejection and saving two new Checkboxes with distinct generated UUIDs and unchanged Task Done. Focused race checks, vet and both entrypoint builds pass.

The same worktree is concurrently receiving an independent fuzz audit. A full run encountered its StructuredHistory seed 2ad3841ca1f8334f (accepted ReplaceSelection leaves a non-text-boundary cursor); replaying with the pre-change draft_selection.go and draft_structure.go from the audit's source snapshot reproduces the same failure. Its newly added InvalidUTF8Paste regression was also failing during the full run. These are not silently claimed green or modified by this task. Final test outcomes and manual acceptance remain to record.

Final scoped validation: all new model/UI/save tests pass, focused race checks pass, updated full markup journeys pass for both literal-Tab aliases, vet and both builds pass. The full run also passes Tutorial, Store and artifact-verification packages, but reported the separately reproduced StructuredHistory seed and the concurrent audit’s InvalidUTF8Paste check. No commit; manual V → Space x acceptance awaits Jan.

## Footer feedback — 2026-10-17

The Space palette is now inside the footer frame. Complete shortcut segments wrap at narrow widths; the existing three-Description-row minimum retains a compact single-row fallback. Display persists after timeout until the next key. SPACE ACTIVE identifies the input window; Space again identifies the expired state. Timeout does not execute shortcuts or change the configured leader duration. Focused rendering/input tests cover 54/100/160 columns, short terminals, timeout and rearming.

## Manual acceptance and remaining subtasks — 2026-10-17

Jan accepts the Rich-text tutorial overall after the selection, footer and command-completion changes. The tutorial journey is accepted with the two explicit follow-ups below; this does not claim physical Windows/clipboard or live Vikunja compatibility acceptance. The feature checkpoint may be committed separately from the concurrent Fuzz audit.

- [x] **Code cursor on whitespace — implemented:** Normal cursor on `n` in the Retab fixture, then `j`, reached the correct Tab line but lost its inverse-video cell. Rendering underline/strike whitespace separately discarded inverse video. The cursor cell now suppresses these decorations while retaining inverse video; neighboring cells retain their styles. Regression checks cover Tab, spaces, empty Code lines, horizontal scroll and decorated prose spaces. Source bytes remain unchanged. Manual recheck on Jan's terminal remains outstanding.
- [x] **Backtick creation/confirmation — verified and feedback corrected:** Jan no longer recalls whether Enter was pressed in the reported attempt. No Code-creation defect was reproduced: three/four directly typed backticks plus Enter work through the real terminal decoder and TUI Update, including grouped text, Shift, release events and representative Win32 sequences. The confirmed defect was misleading footer text: it still offered Enter paragraph when Enter would create Code. It now displays Enter create Code, and tutorial help explicitly says i → three/four backticks → Enter → Esc. Paste/import/escape/extra-text exclusions remain covered. No speculative change to parser semantics; physical German-keyboard behavior remains unverified.

## Implementation follow-up — 2026-10-17

Both follow-up slices used failing-before-fix rendering/input regressions. The Code cursor test failed for Tab and spaces, and the Backtick footer test failed because it showed no Code confirmation. The focused checks now pass, including Undo restoring the typed markers. Cursor rendering and the confirmation hint do not change Source HTML or the existing trigger contract. The earlier terminal report cannot be reconstructed and is not claimed as reproduced; the confirmed feedback defect is corrected. Ticket stays in-progress for manual recheck of these changes; separate environment/compatibility acceptance remains in ticket 10.

Historical checkpoint validation (985a40d, before these follow-ups): the isolated staged feature source, excluding all concurrent Fuzz-audit files and fixes, passes the full Go suite and Rich-text artifact verification, race checks for Description/TUI/Tutorial/Store, vet and both entrypoint builds. At that checkpoint the review found no additional actionable issues and both follow-ups were still open. They are now implemented as recorded above; manual recheck remains outstanding.

### Follow-up verification and reviews

An isolated source copy of checkpoint 985a40d plus only these follow-ups (no Fuzz-audit changes) passes `go test ./... ./.scratch/description-rendering/verification`, `go test -race ./internal/tui ./internal/tutorial`, `go vet ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. Whitespace checks pass. Cursor and misleading-footer regressions failed before their fixes; the decoder/negative/Undo checks pass without changing Code creation semantics.

Two independent read-only reviewers checked bugs/risks and consistency/validation in each of two rounds. R1 found only stale historical ticket wording, corrected before R2; both fresh R2 reviewers reported no findings. No code changes followed validation. Jan subsequently authorized a separate commit of these verified follow-ups, excluding the concurrent Fuzz audit. Remaining gap: Jan's physical terminal/German keyboard has not been rerun; the original Backtick failure itself could not be reproduced.

## Further terminal feedback — paired fences and Code styling

Jan clarified the previously missing Backtick path: type six markers, then insert Enter between markers three and four. The previous exact-three/four trigger did not recognize this path, and navigation also disarmed its UI flag. The accepted extension now recognizes a fully directly typed six-marker paragraph with the cursor at that precise boundary, creates one empty Code block, enters its content, and consumes both groups in one Undo step. Other split positions and pasted/imported/escaped/mixed-origin markers remain literal. This is an explicit rule extension, not a claim that the old exact-three/four route was broken. Spec, keybindings, README and task-11 help describe it.

Jan confirms the Tab-line cursor is now visible and reports unintended Code underlines. Projection incorrectly assigned InlineCode to the code element inside a Code block. The Code projection now retains only the block Code role for that wrapper; Inline-Code outside blocks keeps its underline. No source bytes or saved formatting are changed.

Both regressions failed before fixes: the actual TUI render contained underline attributes throughout print(1), and the paired-marker input showed no Code confirmation. Focused tests now cover un-underlined block rendering with its background, retained Inline-Code underline/cursor, paired Enter through real input decoding including grouped events, exact cursor position, provenance exclusions, Quote metadata, Undo/Redo and savability. The complete markup journey now solves task 11 via the six-marker route. Manual terminal recheck remains outstanding; Jan subsequently authorized committing this reviewed follow-up.

Verification for the paired-fence/style extension: full `go test -buildvcs=false ./... ./.scratch/description-rendering/verification`, race tests for Description/TUI/Tutorial, `go vet ./...`, both entrypoint builds and formatting/whitespace checks pass. After updating the course journey, both literal-Tab aliases passed the complete markup course again, including a focused race run. Two independent read-only reviewers (bugs/risks and consistency/validation) found no actionable issues in round 1; each independently passed Description/TUI/Tutorial tests. Baseline is 3bd234b, which already contains the separate Fuzz-audit commit; that work and unrelated tmp/ remain untouched. The implementation invocation left changes uncommitted; Jan subsequently authorized one combined commit for the paired-fence/style fixes.

## Manual feedback — Steps 11 and 13 accepted, visible Tabs

Jan confirms tutorial Step 11 (Code creation) is fixed. Step 13 confirms the unintended Code underline is gone, but a real Tab must remain distinguishable from spaces. Code projection now shows real source Tabs as a muted > followed by ~ cells to the next configured tabstop. Normal spaces remain blank; the Code background and visible cursor remain. Navigation treats each Tab as its original source character, Copy returns a literal Tab, Retab removes the marker by producing spaces, and Undo restores the Tab. The source/save representation is unchanged. Focused projection and actual TUI rendering/input regressions cover these behaviors. Jan subsequently confirms Retab works and accepts Steps 11 and 13, including the visible Tab cue.

The separate tutorial context/help modals now show the current step and total at the top right (for example 3 / 22), with a truncated title when needed. A real runner regression covers both courses, introductory and F1 help modals, at 54x24. Jan authorizes a separate commit of this accepted follow-up, excluding the subsequent shared insertion-cursor changes.

Validation: Description/TUI/Tutorial package suites pass, including both complete courses; focused Tab navigation/Copy/Retab/rendering regressions and their race checks pass. Affected-package vet and both entrypoint builds pass. Regression expectations were red before implementation for hidden Tabs and missing modal progress. Before this separate commit, the exact staged snapshot was exported without the shared-cursor work; full `go test ./... ./.scratch/description-rendering/verification` and both entrypoint builds pass independently. No source/schema/dependency changes outside the described display behavior; unrelated tmp/ untouched.

## Consistent insertion cursors across all fields

Jan reports Title using a thin terminal cursor, Native Description/Code painting a block, and Comments inserting an underscore. All insertion surfaces now use the same internal input_cursor module: field renderers supply source text and a rune offset (Native frame rendering supplies the source target); a zero-width layout anchor survives wrapping/clipping/overlays and is removed before terminal output. The resolved position uses one blinking terminal-bar policy. New fields use the same rendering seam, documented in AGENTS.md; there is no per-field cursor-shape registry. Normal-mode and explicit masked-date digit replacement retain their block cursors. Shape-disabled inputs share a vertical textual fallback.

Title/inline Title, plain/Native Description, Code, Comment composer, Ex and contextual commands, Search, Link URL, Label and relative-date inputs use the shared rule. The prior Inline-Title coordinate tracking is removed. The Comment composer uses the existing multiline layout, with cursor-following for wrapped/empty/earlier lines. A tiny Description workspace omits its blank header row when necessary to leave space for the editable row. Input windows preserve the caret while clipping long prompt values; no underscore/fake Block is inserted into ordinary Insert text.

Regression tests exercise real View/Update behavior across these inputs, Unicode cell placement, prompt clipping, long URL/Label values, Normal transitions, shape-disabled fallback and Comment scrolling. Earlier tests that expected no prompt terminal cursor or a printed Insert glyph now assert the shared native cursor instead. Source text, saves and typing modes are unchanged. No commit requested.

Shared cursor validation: full `go test -buildvcs=false ./... ./.scratch/description-rendering/verification`, TUI/Tutorial race checks, `go vet ./...`, both entrypoint builds and formatting/whitespace checks pass. The input consistency matrix initially failed for Native/Code/Comments/Ex/Search/Label/contextual-command inputs and now passes. Unicode/clipping, URL and Label prompts, Comment wrap/empty-line/scroll, Normal transitions, disabled-shape fallback, tiny Description and masked/relative date cursor regressions pass. The complete basics/markup courses pass with the shared cursor path. Physical terminal appearance remains for Jan to observe after restarting the app. No commit.


### Insert cursor follow-up — Shift+A before Code newline

Jan reports the bar disappears after Shift+A at the end of print(1), while the final Tab line retains it. A real Update/View regression reproduces a nil terminal cursor. Code layout omitted navigation points for nonempty line-ending newlines even in insertion mode; it now includes that insertion boundary while retaining Normal-mode motion targets. Tests cover appending and subsequent typing before a newline, at the final Tab/text line, on an empty first line, with Unicode and in a paragraph. Description/TUI/Tutorial suites, the focused cursor race checks, affected-package vet and both entrypoint builds pass. Temporary debug instrumentation removed. No commit requested.

Adjacent finding retained for separate work: appending and typing into an imported completely empty `<pre><code></code></pre>` fails with `invalid Native Description structure`; its cursor itself is present. This is separate from the missing line-end cursor.


### User acceptance and cursor commit

Jan confirms the Shift+A line-end cursor now works and authorizes committing the shared insertion-cursor implementation with that fix. Steps 11 and 13 remain accepted; repeating the entire tutorial is not required. The imported completely empty Code typing defect above remains open and belongs to implementation follow-up, not a user acceptance task. Physical clipboard/Windows input edge cases and live Vikunja round-trip remain under ticket 10.


### Shared text selections/clipboard and explicit Richtext conversion

Jan requests implementation of consistent Visual Mode and system Clipboard bindings after v/V did nothing in a Plaintext Description. Title, Plaintext Description and Comment drafts now share one source-offset selection implementation (v/V, motions, viw/vaw, endpoint swap, delete/change, Esc); Native Richtext retains its structured selection engine. Space+y copies selected safe text or the current line (Native prose: logical paragraph). Space+p performs literal Paste at the cursor or replaces the selection, staying Normal. Copy keeps the selection; an empty Clipboard and read/write errors never delete it. Clipboard reads are guarded against changed focus, field, mode, text, cursor and selection. Newlines flatten in Title, normalize in prose/Comments and retain Code source line endings/Tabs. Space+Y now copies Link URLs; the local Native y/p register remains unchanged.

:descriptionrichtext explicitly converts a plain Description draft to escaped Native paragraphs, preserving blank lines without interpreting pasted Markdown. Only :w persists it; :e! restores the original plain baseline. Existing HTML/Blocked descriptions are rejected. Tests include real Update/View selection and Clipboard flows across all four editors, failures/stale reads, Unicode graphemes, current Code-line Copy, line deletion, literal Code Paste with one-step Undo, and conversion/save/reopen/formatting against disposable SQLite. No commit requested. The earlier imported-empty-Code input defect remains separate.


Validation: the initial Update/View tests failed for missing Plaintext/Title/Comment Visual Mode, missing system Clipboard bindings, and the unknown conversion command. Full `go test -buildvcs=false ./... ./.scratch/description-rendering/verification` passes, including both complete tutorial courses and Store tests. Final TUI package tests with the race detector, `go vet ./...`, both entrypoint builds and whitespace checks pass. Follow-up review covers cancelled/incomplete selection prefixes, line-change separators, preservation of Opaque confirmation, empty/failing Clipboard operations, stale reads even after returning the cursor to its old position, and actual conversion/save/reopen with unchanged Task Done. Physical host Clipboard acceptance remains for Jan; no commit, dependency, schema or local user database changes.


### One Description editor — supersedes explicit conversion, 2026-10-18

Jan rejects two Description editing modes. Empty, plain and HTML Descriptions now open the same document editor, with selection and formatting immediately available. The previously proposed `:descriptionrichtext` command is removed. Plain source imports literally as paragraphs, preserving empty lines and whitespace on opening. Reading/selecting/resizing and metadata-only saves retain the exact source baseline. Actual Description edits produce HTML; Undo before save restores the original plain bytes, and `:e!` discards the draft. Title/Comment selections remain in their shared text module; Description uses its structured selection engine. All support Space+y/p for the system Clipboard; Space+Y copies a Link URL. Async Paste also rejects stale results after moving away and back.

Review findings fixed in this slice:

- **Medium:** Plain source passed through HTML parsing lost literal whitespace/anchors. Build literal Text nodes directly; test exact displayed text and unchanged baseline.
- **Medium:** Unification exposed missing ordinary Vim editing behavior: final-word `dw` left a character, counted `dd` deleted only one block, invalid `d2Q` leaked the count into the next motion, and internal operator selections leaked into Undo. Word ranges, sibling cursor placement, count reset and one-command Undo groups are corrected. `C/D`, counted `x`, `r` and `~` preserve ordinary editing behavior and inline marks.
- **Medium:** Blockwise `Vc` could insert into a neighboring paragraph. Retain the first selected paragraph as insertion context.
- **Medium:** Tiny Description geometry disagreed with the renderer, preventing access to checklist summaries; manual scrolling also kept the cursor offscreen when typing. Match layout thresholds and resume following on Insert/edit. Summary rows remain scrollable without gaining a source cursor.
- **Medium:** Imported completely empty `<pre><code></code></pre>` rejected typing. Materialize its first Text leaf inside the existing code wrapper, preserving wrapper/ancestor metadata. Real Update tests cover `i` and `A`, save preparation and Undo for bare and quoted Code.
- **Low:** README still described a separate plain Description path. Active docs/help now describe one editor; earlier conversion notes above are historical and superseded.

Targeted tests reproduced the operator/count/empty-Code failures before fixes and now pass. No additional user acceptance of steps 11/13 is requested. Remaining manual work: a short free-editor check with the actual OS Clipboard/terminal, plus the live Vikunja compatibility round-trip tracked by ticket 10. No commit requested.

Final validation of the unified editor and review fixes: `go test -buildvcs=false ./... ./.scratch/description-rendering/verification` passes, including Store and both complete tutorial courses. `go test -race -buildvcs=false ./internal/description ./internal/tui ./internal/tutorial`, `go vet -buildvcs=false ./...` and both entrypoint builds pass. Whitespace checks pass. No additional confirmed findings remain in the reviewed changes; physical OS Clipboard/terminal and live Vikunja acceptance are still outstanding. Changes remain uncommitted.

Jan authorizes committing this reviewed unified-editor/Clipboard slice. Manual free-editor/OS Clipboard and live Vikunja acceptance remain open as documented; tutorial steps 11/13 stay accepted.


### Short-terminal Description visibility regression — 2026-10-19

Jan reports that selecting Description and pressing Enter leaves only the label visible at a short terminal height. Reproduced through actual WindowSize/KeyPress/Full View at terminal heights 30/31 and 34/35 (Detail text heights 12 and 14), for both legacy plain text and HTML. The regular metadata/identifier/Comments budget leaves zero Description body rows. The earlier focused workspace was reached only below 12 Detail rows; it missed these intermediate layouts.

Why existing regressions missed it: the activation test asserted editing state without examining visible content; compact visibility coverage exercised 22 terminal rows, and the no-cursor test deliberately covers the inactive, hidden body. None asserted the complete activated View across every neighboring height. Prior green suites therefore did not certify this case.

`go test -buildvcs=false ./internal/tui -run '^TestActivatedDescriptionIsVisibleAcrossTerminalHeights$' -count=1` failed before the fix with `activated Description invisible at terminal height 30 (detail=12)` and the matching label-only View. The same test now passes. Active Description uses the existing focused workspace when the actual regular row budget is zero; rendering, native navigation geometry, viewport scrolling and cursor eligibility share this decision. Regular metadata remain accessible after Esc. No source/data/schema changes.

New regressions assert real rendered Description content for every terminal height 18–46, with plain and HTML input. Additional 55/100-column shrink/grow journeys check unsaved text, on-screen insertion bar, one-step Undo and returning to Title. No debug instrumentation remains. No commit requested.

Validation: full `go test -buildvcs=false ./... ./.scratch/description-rendering/verification`, TUI race tests, affected-package vet, both entrypoint builds, formatting and whitespace checks pass. This confirms automated rendered-view behavior; the physical terminal screenshot has not been rerun by Jan.

Jan confirms the short-terminal Description visibility fix works and authorizes a separate commit. The focused-period escaping report is a subsequent, separate investigation.


### Focused sentence punctuation — 2026-10-19

After accepting the separate height fix (committed as b2209c0), Jan reports a synthetic backslash before the trailing period in the focused first practice paragraph. Reproduced with `TestFocusedDescriptionDoesNotEscapeSentencePeriod` through the full TUI View; model regressions also showed a Syntax:1 navigation target on an ordinary period and a hidden Backspace step that left abc. unchanged instead of deleting c.

Cause: the focused inline disclosure escaped every literal period, using the same overly broad character set as adjacent-syntax editing. Ordinary periods are now excluded from synthetic disclosure and its adjacent-Backspace handling via a shared character set. Explicit typed backslash-period input remains supported by the unchanged input grammar. Source HTML and plain baselines are unchanged merely by opening/rendering. Tests cover plain/HTML sentences, decimal/domain punctuation, a literal numbered prefix, focused/unfocused display, cursor motion, Backspace and exact Undo. Existing protecting escapes for stars and deliberate escape/Undo tests remain green. A parallel read-only investigation independently identified the same display/edit coupling. No commit requested for this subsequent fix.

Period-fix validation: the original full-View regression and semantic display/motion/Backspace regressions failed before the fix and now pass. Full `go test -buildvcs=false ./... ./.scratch/description-rendering/verification`, Description/TUI race tests, affected-package vet, both entrypoint builds and formatting/whitespace checks pass. No debug instrumentation remains. The height fix is committed; the period fix remains a separate uncommitted change.

Jan authorizes committing the reviewed period-disclosure fix. AT-03 screenshots subsequently show the intended literal-star protection in the focused paragraph and unescaped literal stars after leaving that paragraph; these are distinct from the corrected ordinary-period escape. Screenshot evidence alone does not certify the external clipboard-copy portion of AT-03.


### AT-06 — CR-only Code Paste, 2026-10-19

Jan's screenshot shows the AT-06 multiline test text with visible carriage-return escapes instead of separate lines. `go test -buildvcs=false ./internal/tui -run '^TestCodePasteRendersTransportNewlines$' -count=1` reproduced this exact full-View failure for CR-only terminal Paste and system Clipboard Paste; LF and CRLF pass. The input path preserves its bytes correctly. The defect is missing standalone-CR newline handling in native Code projection and line operations. Earlier Clipboard tests asserted source retention for CRLF and omitted CR-only rendered output.

Code now treats CR, LF and CRLF as source line endings without normalizing their bytes. Shared line-boundary/newline-length handling supports navigation, CopyLine, V selection, line deletion (including final empty lines), Enter/open-line and multi-line indent. Projection recognizes CR as a Code newline while retaining terminal-control escaping in prose. A CRLF pair split across preserved code wrappers still projects as one break. Regression tests cover both Paste routes with the original tabbed/Unicode/literal-marker snippet, separate rendered source lines, exact source retention, save preparation, one-step Undo/Redo and navigation/CopyLine. Semantic Code tests exercise edits and exact Undo for all three newline forms. No dependency/schema/local-user-DB changes. No commit requested.

CR-only fix validation: full `go test -buildvcs=false ./... ./.scratch/description-rendering/verification`, Description/TUI race tests, affected-package vet, both entrypoint builds and formatting/whitespace checks pass. These checks also pass in an isolated snapshot of HEAD plus only this CR/source-newline change, excluding concurrent InsertLink/Link-prompt work. Logs: /tmp/vimkunja-code-cr-isolated-tests.log and /tmp/vimkunja-code-cr-isolated-race.log. No debug instrumentation remains. AT-01–05 stay manually accepted; AT-06 stays FAIL pending focused terminal recheck. No commit requested.

Jan requests rechecking and committing this CR-only Code Paste fix. Re-review found no additional confirmed findings; the code matches the isolated fully tested snapshot. Concurrent Link insertion and general Tab-marker changes are excluded. Manual AT-06 recheck remains outstanding.

## Visual case toggle — 2026-10-19

Fixed Jan's `viw` → `~` report: the TUI explicitly rejected Visual case changes, while the document transform only handled counted Normal-mode characters. ToggleCase now uses the existing structured selection range, changes only Native text, preserves wrappers/URLs/Opaque content, and retains Visual Mode, selection direction and the active cursor until Esc. Selection anchors track Unicode case mappings that change UTF-8 byte width, so repeated toggles target the same text. One local Undo/Redo restores the entire operation. Regression first reproduced the exact “Leave Visual mode before character case changes” error; model tests now cover word, forward/reverse, block and Code-line selections, Unicode with changing UTF-8 width, plain-source conversion, formatting, protected atoms and save preparation. Description, TUI and Store package tests pass. No manual terminal acceptance claimed.


### AT-10 — false concurrent-edit rejection during Description reload, 2026-10-19

Reproduced Jan's exact error via `go test -buildvcs=false ./internal/tui -run '^TestDescriptionReloadAfterLabelsLoadedReadsBrowserEdit$' -count=1`: a real SQLite Task loads its empty label list, an external writer changes Alpha beta to Alpha gamma, then the actual :descriptionreload dispatch rejects the unchanged draft. Prior reload coverage used an unopened/default nil label slice and therefore missed this loaded-editor state.

Cause: cloneTaskDraft used append-to-nil for LabelIDs. It normalized a loaded empty slice to nil; the asynchronous reload safety check uses reflect.DeepEqual and falsely found a difference. slices.Clone retains the representation and the independent backing array. No weakening of the late-edit guard: focused actual-command tests cover Tasks with/without labels, before/after TUI save, visible browser text and unchanged persisted browser source; additional tests retain Title/Description/Labels edits made after dispatch. The minimal exact error test failed before the fix and now passes. No commit requested.

Reload-fix validation: full `go test -buildvcs=false ./... ./.scratch/description-rendering/verification`, TUI race tests, affected-package vet, both entrypoint builds, gofmt and whitespace checks pass. Re-review confirms the production change is limited to lossless LabelIDs snapshot cloning; the existing reload conflict guard remains intact. Jan confirms the physical in-app reload works; AT-10 is manually accepted. Jan authorizes committing this reload fix and its regression tests separately from concurrent Link/Tab-marker work.
