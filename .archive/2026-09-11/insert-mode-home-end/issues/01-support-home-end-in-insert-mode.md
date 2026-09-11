# 01 — Support Home and End in Insert Mode

**What to build:** While editing any Vimkunja text field, a user can jump the cursor to the first or last position without leaving Insert Mode.

**Implementation prerequisites:** Resolved — the Due Date picker deliberately has no Home/End row navigation; `h`/`l` (or arrows) move through its manual date mask. All text prompts own a movable cursor.

**Category:** enhancement

**Milestone:** M2

**Priority:** Highest

**Status:** completed

- [x] `Home` and `Pos1` move the Insert Mode cursor to the first position of the current editable line.
- [x] `End` moves the Insert Mode cursor to the last position of the current editable line.
- [x] The cursor remains in Insert Mode after either motion.
- [x] Apply the behavior consistently in the Task title, Description, Comment composer, and any other editable text field.
- [x] Preserve literal text input and existing Normal Mode motions.
- [x] Add model-level regression coverage for each editable field and both motions.

## Implementation checkpoint

- Implemented: Detail Task title, plain-text Description, and Comment composer; inline List/Overview titles already supported Home/End.
- Multiline motions use newline-delimited lines, not wrapped terminal rows; End is the insertion position after the final character.
- Fixed a separate manual-date editing bug: opening the picker now resets its cursor to the first digit instead of inheriting an out-of-range cursor from a previously edited title or date. Regression tests exercise both actual entry sequences.
- Fixed missing date-picker cursor: the manual date row now exposes a blinking native block cursor on the next overwritten digit (skipping separators). Relative helpers retain an inline blinking marker. Rendered-view tests cover regular, compact, and narrow layouts, typing, row switching, and closing the picker.
- Legacy Due Date Insert Mode supports text motions, but normal `i` entry opens the Due Date picker. There, Home/End deliberately do nothing rather than navigating helper rows; `h`/`l` and arrows move through the manual date mask.
- Label filter/create/rename, Global Search, and all Command prompts support cursor-aware insertion, deletion, Left/Right, Home/Pos1, and End. Their visible cursor marks the insertion position.
- Completed: Due Date picker navigation no longer conflicts with the text-motion contract.

## Further context

This is specifically for use while already editing. The existing Home/End navigation ticket covered scrollable informational panes, not Insert Mode cursor movement. For multi-line text, this ticket assumes beginning/end of the current line; triage should confirm whether the editor's terminal position semantics require any special handling.
