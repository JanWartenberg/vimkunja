# 01 — Add advanced Vim editor motions

**What to build:** A Vimkunja user editing Task text can use additional familiar Vim-style editing motions beyond the current Detail editor basics, without weakening explicit save and discard behavior.

**Implementation prerequisites:** None — deliberately deferred until after the Fundament/MVP.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Specify word/text-object deletion (`dw`, `db`, `diw`, `daw`) and related text-entry motions in each editable field. Basic Description deletion (`x`, `X`, `D`, `dd`) is implemented separately; Visual Mode remains out of scope.
- [ ] Preserve literal text input where a key is not an active motion in the current editor mode.
- [ ] Keep multi-line Description behavior, cursor rendering, dirty state, and save/discard semantics coherent.
- [ ] Add model-level regression coverage for every adopted motion.
