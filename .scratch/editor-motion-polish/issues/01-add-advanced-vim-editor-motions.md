# 01 — Add advanced Vim editor motions

**What to build:** Extend the existing Vim-like editor behavior with the practical Tier 1 motion set, without weakening explicit save and discard behavior.

**Implementation prerequisites:** None — define the motion semantics below before implementation. Full Vim Normal mode is tracked separately.

**Category:** enhancement

**Priority:** Low

**Status:** implemented

## Scope: Tier 1 plus existing behavior

Keep the existing `h`, `j`, `k`, `l`, `w`, `b`, `gg`, `G`, `Home`, `End`, `Ctrl+D`, `Ctrl+U`, `Ctrl+E`, `Ctrl+Y`, `x`, `X`, `D`, `dd`, `i`, `a`, `I`, `A`, `o`, and `O` behavior. Add:

- `0`, `^`, `$` — line start, first non-blank, and line end.
- `e`, `W`, `B`, `E` — end-of-word and WORD motions.
- Numeric counts for motions and adopted operators, such as `2w`, `3j`, and `d2w`.
- Operator motions: `dw`, `db`, and `de`.
- Word changes and text objects: `cw`, `diw`, `daw`, `ciw`, and `caw`.
- Change and replace: `C`, `r{char}`, and `~` to toggle character case.

Visual mode, search motions, character-find motions (`f`/`t`), marks, registers, macros, and full operator coverage remain out of scope here.

## Adopted semantics

- `word` uses Unicode letters, digits, and `_` as one class, punctuation as another, and whitespace as separators; `WORD` is any run of non-whitespace.
- Word motions cross newlines in multiline text. Horizontal and line motions stay within their line; counted `j`/`k` preserve the starting column and clamp on the target line.
- Operator and motion counts multiply (`2d3w` equals `6dw`). Counts also address Description lines for `gg`/`G`. `dw` deletes up to the next word start, `db` back to the previous word start, `de` through the word end, and `cw` uses Vim's `ce` semantics so following whitespace remains.
- `iw` selects the current word or whitespace run. `aw` additionally consumes following whitespace for a word; on whitespace it includes the following word, and at the end it falls back to preceding whitespace. Change operations delete the same range and enter Insert mode.
- All offsets and counts are rune-aware. `r{char}` and `~` stop at a newline; `~` toggles Unicode-aware case under the cursor and advances, while unavailable range at a boundary is a no-op.

## Implementation plan

- [x] Specify exact word, WORD, whitespace, punctuation, newline, count, and cursor-placement semantics, including behavior at text boundaries.
- [x] Apply the text motions and operators consistently to the editable text fields: Task title, plain-text Description, and Comment composer where the shared editor contract permits it. Do not activate them in pickers, non-text fields, or read-only HTML Description.
- [x] Extract or reuse a rune-aware text-editing layer for motions, counts, word objects, deletion, change, and single-character replacement; preserve multiline Description behavior.
- [x] Integrate Normal/Insert/operator-pending handling without breaking existing `d`/`g` prefixes, literal Insert-mode input, dirty tracking, cursor rendering, or explicit save/discard behavior.
- [x] Update help/tutorial documentation for the adopted bindings.
- [x] Add focused model-level regression coverage for the specified motion semantics and editor invariants.

## Acceptance

Manual Tier 1+ acceptance on 2026-09-20: **50 PASS · 0 FAIL · 1 BLOCKED · 1 OPEN**. The detailed report remains a local, untracked acceptance artifact.

Accepted follow-up fixes include multiline browser Paste, Task-wide `:e!`, `~` in Description and Title, replacement at the displayed line end, and visible Normal-mode block cursors in Title and the Comment composer. The remaining manual checks are `~` in the Comment composer and the lower-priority read-only HTML guard; both paths have automated regression coverage.
