# 01 — Add advanced Vim editor motions

**What to build:** A Vimkunja user editing Task text can use additional familiar Vim-style editing motions beyond the current Detail editor basics, without weakening explicit save and discard behavior.

**Blocked by:** None — deliberately deferred until after the Fundament/MVP.

**Status:** needs-triage

- [ ] Specify the supported behavior for `o`, `O`, `a`, `A`, `I`, and related text-entry motions in each editable field.
- [ ] Preserve literal text input where a key is not an active motion in the current editor mode.
- [ ] Keep multi-line Description behavior, cursor rendering, dirty state, and save/discard semantics coherent.
- [ ] Add model-level regression coverage for every adopted motion.
