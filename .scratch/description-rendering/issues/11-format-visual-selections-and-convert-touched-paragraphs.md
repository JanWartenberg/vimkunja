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
