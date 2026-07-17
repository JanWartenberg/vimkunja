# 01 — Improve editor cursor shapes

**What to build:** Give Vimkunja's editor a mode-specific cursor representation resembling the user's Vim `guicursor` configuration, so editing state is recognizable without reading the footer.

**Blocked by:** None — deliberately deferred until after the Fundament/MVP.

**Status:** needs-triage

- [ ] Normal, Visual, and Select modes use a block-shaped cursor.
- [ ] Insert, Command-line, Insert-completion, and Virtual-replace modes use a vertical bar cursor at approximately 20% character-cell width.
- [ ] Replace and Operator-pending modes use a horizontal bar cursor at approximately 50% character-cell height.
- [ ] Cursor rendering does not shift or corrupt editable text and remains usable on supported terminal backends.
- [ ] The implementation decides and documents which cursor shapes are native terminal capabilities versus TUI-rendered approximations.

## Further context

Target Vim setting: `vim.opt.guicursor = "n-v-sm:block,i-c-ci-cr-ve:ver20,r-o:hor50"`.

Priority: higher than the ordinary later UX backlog, but not part of the completed Fundament/MVP.
