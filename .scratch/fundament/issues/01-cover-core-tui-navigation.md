# 01 — Cover core TUI navigation behaviour

**What to build:** Regression coverage for Vim-style list motion and pane focus navigation, so users can navigate projects and tasks predictably as the TUI evolves.

**Blocked by:** None — can start immediately.

**Status:** resolved

- [x] Model tests verify `gg` selects the first available item and `G` selects the last available item.
- [x] Model tests verify focus cycles between available panes and directional `Ctrl+W` navigation reaches the expected pane.
- [x] Tests cover empty or unavailable-pane boundaries without allowing invalid focus or selection state.
- [x] `go test ./...` passes.
