# 01 — Decide and enforce Task Title length

**What to decide:** Define whether Vimkunja limits Task Titles and, if so, enforce a Vikunja-compatible limit without silently truncating user input.

**Implementation prerequisites:** Product decision on the limit and compatibility behavior.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Decide whether a Title limit is needed and its exact maximum.
- [ ] Decide whether the limit counts bytes, runes, or another Vikunja-compatible unit.
- [ ] Verify the chosen rule against Vikunja's SQLite schema and original application behavior without changing the schema.
- [ ] Specify validation feedback and save behavior for an overlong Title; input must never be silently truncated.
- [ ] Add model/store coverage for the adopted rule.

## Further context

The Title editor now supports Vim-like horizontal cursor movement and insertion. A possible Title length limit was explicitly deferred because its value and Vikunja compatibility are not yet decided.
