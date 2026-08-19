# 01 — Specify Due Date and Priority editor inputs

**What to decide:** Improve the Task editor's specialized Due Date and Priority input beyond the current mask and fixed Picker.

**Implementation prerequisites:** Needs interaction decisions.

**Category:** enhancement

**Priority:** Medium

**Status:** triaged

## Decision

- **Due Date:** keep the `YYYY-MM-DD` mask for manual entry.
- Add quick-pick helpers for **tomorrow**, **in n days**, and **in n months**.
- **Priority:** keep the existing picker.
- Keep task-detail jump keystrokes as part of the special-input discussion, but make them a separate follow-up unless they are required for the same edit flow.

## Vim-style shortcut proposal

- `gt` → edit title
- `gd` → edit due date
- `gp` → edit priority
- `gl` → edit labels

Rationale: `g`-prefixed shortcuts fit Vim conventions better than picker-centric key flows, while the picker UI itself should stay keyboard-driven (`hjkl`/Enter/Esc) if it remains part of the editor.

- [ ] Preserve predictable validation, dirty state, save/discard, and narrow-terminal rendering.

## Further context

The existing editor already accepts exactly eight due-date digits (`YYYYMMDD`) through its `YYYY-MM-DD` mask. M1 records and regression-tests that behavior. This ticket retains date pickers, shortcuts, natural-language input (including German/English weekday terms), and richer Priority input.

Archived source: `TUI_IDEA.md`, Edit-Mode special-input TODOs.
