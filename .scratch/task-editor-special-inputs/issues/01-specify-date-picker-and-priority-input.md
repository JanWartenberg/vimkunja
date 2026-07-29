# 01 — Specify Due Date and Priority editor inputs

**What to decide:** Improve the Task editor's specialized Due Date and Priority input beyond the current mask and fixed Picker.

**Implementation prerequisites:** Needs interaction decisions.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

- [ ] Decide whether Due Date uses a keyboard date picker, shortcuts, or an improved text mask.
- [ ] Decide whether Priority uses direct number input, `+`/`-`, or retains a Picker.
- [ ] Preserve predictable validation, dirty state, save/discard, and narrow-terminal rendering.

## Further context

M1 extracts one deliberately small input improvement into the task-metadata ticket: accept exactly eight due-date digits (`YYYYMMDD`) and normalize them to `YYYY-MM-DD`. This ticket retains date pickers, shortcuts, natural-language input (including German/English weekday terms), and richer Priority input.

Archived source: `TUI_IDEA.md`, Edit-Mode special-input TODOs.
