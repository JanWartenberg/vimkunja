# 01 — Indicate scroll position in long Panes

**What to build:** A Vimkunja user can tell that a Pane contains more content than fits on screen and can estimate their current position, without sacrificing keyboard-first operation.

**Implementation prerequisites:** None — deliberately deferred until after the Fundament/MVP.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] A Pane with overflow visibly indicates that it can scroll.
- [ ] The UI shows a percentage of the current scroll position, a scrollbar-like edge indicator, or both.
- [ ] The indicator updates for Help, Errors, Description, and future long Pane content using a consistent convention.
- [ ] Panes without overflow do not show distracting scroll affordances.

## Further context

Decide the exact visual design during specification; the request permits either a percentage indicator, a scrollbar-like hint at the Pane edge, or both.
