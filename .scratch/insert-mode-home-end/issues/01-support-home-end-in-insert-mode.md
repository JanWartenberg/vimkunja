# 01 — Support Home and End in Insert Mode

**What to build:** While editing any Vimkunja text field, a user can jump the cursor to the first or last position without leaving Insert Mode.

**Implementation prerequisites:** None — reuse the existing editor behavior consistently across every editable text field.

**Category:** enhancement

**Milestone:** M2

**Priority:** Highest

**Status:** needs-triage

- [ ] `Home` and `Pos1` move the Insert Mode cursor to the first position of the current editable line.
- [ ] `End` moves the Insert Mode cursor to the last position of the current editable line.
- [ ] The cursor remains in Insert Mode after either motion.
- [ ] Apply the behavior consistently in the Task title, Description, Comment composer, and any other editable text field.
- [ ] Preserve literal text input and existing Normal Mode motions.
- [ ] Add model-level regression coverage for each editable field and both motions.

## Further context

This is specifically for use while already editing. The existing Home/End navigation ticket covered scrollable informational panes, not Insert Mode cursor movement. For multi-line text, this ticket assumes beginning/end of the current line; triage should confirm whether the editor's terminal position semantics require any special handling.
