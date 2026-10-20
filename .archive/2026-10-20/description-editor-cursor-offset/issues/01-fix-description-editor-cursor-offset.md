# 01 — Fix Description editor cursor/insert line offset

**What is broken:** In Description Insert Mode, the rendered cursor is offset by one line from the position where newly typed text is actually inserted.

**Expected behavior:** The visible cursor and the insertion position always refer to the same character and rendered line.

**Category:** bug

**Priority:** High

**Status:** completed

- [x] Reproduce the offset in single-line, multiline, wrapped, and scrolled Description content.
- [x] Keep cursor rendering and insertion coordinates aligned in Insert Mode.
- [x] Verify behavior at line starts/ends, around newlines, and in narrow Detail panes.
- [x] Preserve existing Normal/Insert navigation, dirty-state, and save/discard behavior.
- [x] Add model-level regression coverage for the failing cases.

## Follow-up

The Comment composer has adopted the shared multiline-layout seam; see [01 — Reuse multiline layout in the Comment composer](../../comment-composer-multiline-layout/issues/01-reuse-multiline-layout-in-comment-composer.md).

## Closure reconciliation — 2026-10-20

The implementation scope is complete and its regression coverage passes in the
current main working tree. Archived during the release subtask cleanup. This
records technical completion and preserves existing manual acceptance evidence;
it does not claim a new physical Windows run or acceptance of unrelated future
features. Remaining release checks are tracked in the active cleanup/publication
tickets.
