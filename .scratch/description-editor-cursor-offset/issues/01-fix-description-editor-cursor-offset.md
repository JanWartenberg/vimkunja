# 01 — Fix Description editor cursor/insert line offset

**What is broken:** In Description Insert Mode, the rendered cursor is offset by one line from the position where newly typed text is actually inserted.

**Expected behavior:** The visible cursor and the insertion position always refer to the same character and rendered line.

**Category:** bug

**Priority:** High

**Status:** implemented

- [x] Reproduce the offset in single-line, multiline, wrapped, and scrolled Description content.
- [x] Keep cursor rendering and insertion coordinates aligned in Insert Mode.
- [x] Verify behavior at line starts/ends, around newlines, and in narrow Detail panes.
- [x] Preserve existing Normal/Insert navigation, dirty-state, and save/discard behavior.
- [x] Add model-level regression coverage for the failing cases.

## Follow-up

The Comment composer can adopt the same private multiline-layout seam in a later session; see [01 — Reuse multiline layout in the Comment composer](../../comment-composer-multiline-layout/issues/01-reuse-multiline-layout-in-comment-composer.md).
