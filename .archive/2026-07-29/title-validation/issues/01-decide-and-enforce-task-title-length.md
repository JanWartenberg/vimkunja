# 01 — Decide and enforce Task Title length

**What to decide:** Define whether Vimkunja limits Task Titles and, if so, enforce a Vikunja-compatible limit without silently truncating user input.

**Implementation prerequisites:** Product decision on the limit and compatibility behavior.

**Category:** enhancement

**Priority:** Low

**Status:** resolved

## Resolution

Vikunja-compatible title validation requires only a non-empty Title; neither the inspected SQLite schema nor current upstream Vikunja imposes a maximum length. Vimkunja already enforces that rule, so no implementation work remains. See [research-vikunja-title-limit.md](../research-vikunja-title-limit.md).

- [x] Decide whether a Title limit is needed and its exact maximum: no maximum.
- [x] Decide whether the limit counts bytes, runes, or another Vikunja-compatible unit: not applicable.
- [x] Verify the chosen rule against Vikunja's SQLite schema and original application behavior without changing the schema.
- [x] Specify validation feedback and save behavior for an overlong Title: no overlong state exists.
- [x] Add model/store coverage for the adopted rule: existing non-empty-title validation remains the adopted rule.

## Further context

The Title editor now supports Vim-like horizontal cursor movement and insertion. A possible Title length limit was explicitly deferred because its value and Vikunja compatibility are not yet decided.
