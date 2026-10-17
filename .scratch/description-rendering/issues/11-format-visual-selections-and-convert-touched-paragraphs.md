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

- [ ] **Code cursor on whitespace:** On the Retab fixture `<pre><code>print(1)\n\t</code></pre>`, position the Normal cursor on `n`, then press `j`. The document cursor reaches the Tab line, but its visible block disappears. A temporary Go overlay reproduces the missing inverse-video cell. The Code text is also underlined; the renderer's whitespace styling drops inverse video. Suppressing underline/strike on the cursor cell restores visibility in the overlay. No production fix has been applied. Acceptance: a visible cursor on Tab, spaces and empty Code lines after j/k, including horizontal scrolling, without changing source bytes or neighboring styles; retain a rendering regression.
- [ ] **Backtick Code creation on the user's terminal:** Jan could create an empty Code block with Normal `Space c`, but not by typing backticks in Insert. The contract is exactly three/four directly typed backticks in an otherwise empty paragraph, then Enter; Paste stays literal. Existing synthesized-key tests pass, so the reported terminal path remains unconfirmed. Acceptance: reproduce the physical-key event sequence (including a possible German-keyboard dead-key/grouped event), identify whether dispatch or the trigger fails, and fix any confirmed defect with a real-dispatch regression. Do not change the documented contract without a separate decision.

These subtasks remain open after the checkpoint; the Fuzz audit is tracked separately and excluded from this commit.

Checkpoint validation: the isolated staged feature source, excluding all concurrent Fuzz-audit files and fixes, passes the full Go suite and Rich-text artifact verification, race checks for Description/TUI/Tutorial/Store, vet and both entrypoint builds. The review found no additional actionable issues; the two subtasks above intentionally remain open.
