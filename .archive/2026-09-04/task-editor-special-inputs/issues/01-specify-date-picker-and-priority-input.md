# 01 — Specify Due Date and Priority editor inputs

**What to decide:** Improve the Task editor's specialized Due Date and Priority input beyond the current mask and fixed Picker.

**Implementation prerequisites:** Needs interaction decisions.

**Category:** enhancement

**Priority:** Medium

**Status:** completed

## Decision

- **Due Date:** keep the `YYYY-MM-DD` mask for manual entry.
- Add quick-pick helpers for **in n days**, **in n weeks**, and **in n months**.
- **Priority:** keep the existing picker; no `+`/`-` bindings or richer input.
- `x` and `dd` clear Due Date or Priority in the Task draft without confirmation; Priority resets to None.

## Vim-style shortcuts

- `gt` → edit title
- `gd` → edit due date
- `gp` → edit priority
- `gl` → edit labels

Rationale: `g`-prefixed shortcuts fit Vim conventions better than picker-centric key flows, while the picker UI itself should stay keyboard-driven (`hjkl`/Enter/Esc) if it remains part of the editor.

- [x] Preserve predictable validation, dirty state, save/discard, and narrow-terminal rendering.

## Implementation

`i` on Due Date retains the masked `YYYY-MM-DD` entry and opens a field-local picker below it. `j`/`k` or arrows select masked entry, `n` days, `n` weeks, or `n` months; digits enter `n` and `Enter` commits the selected relative date. `Esc` leaves the picker without applying a relative value.

`gt`, `gd`, `gp`, and `gl` jump directly to and edit the corresponding Detail field. Wide normal-mode footers advertise the shortcuts; Help lists them in a dedicated Detail subsection. On Due Date and Priority, both `x` and `dd` clear the draft value before the next explicit save.

## Further context

The existing editor already accepts exactly eight due-date digits (`YYYYMMDD`) through its `YYYY-MM-DD` mask. M1 records and regression-tests that behavior. Natural-language date input and Priority `+`/`-` bindings are not planned.

Archived source: `TUI_IDEA.md`, Edit-Mode special-input TODOs.
