# 01 — Fix Description editor cursor/insert line offset

**What is broken:** In Description Insert Mode, the rendered cursor is offset by one line from the position where newly typed text is actually inserted.

**Expected behavior:** The visible cursor and the insertion position always refer to the same character and rendered line.

**Category:** bug

**Priority:** High

**Status:** needs-triage

- [ ] Reproduce the offset in single-line, multiline, wrapped, and scrolled Description content.
- [ ] Keep cursor rendering and insertion coordinates aligned in Insert Mode.
- [ ] Verify behavior at line starts/ends, around newlines, and in narrow Detail panes.
- [ ] Preserve existing Normal/Insert navigation, dirty-state, and save/discard behavior.
- [ ] Add model-level regression coverage for the failing cases.
