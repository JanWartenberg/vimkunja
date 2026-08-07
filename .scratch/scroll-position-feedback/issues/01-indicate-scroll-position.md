# 01 — Indicate scroll position in long Panes

**What to build:** A Vimkunja user can tell that a Pane contains more content than fits on screen and can estimate their current position, without sacrificing keyboard-first operation.

**Implementation prerequisites:** None — deliberately deferred until after the Fundament/MVP.

**Category:** enhancement

**Priority:** Medium

**Status:** ready-for-agent

- [ ] A Pane with overflow visibly indicates that it can scroll.
- [ ] The UI shows a percentage of the current scroll position, a scrollbar-like edge indicator, or both.
- [ ] The indicator updates for Help, Errors, Description, and future long Pane content using a consistent convention.
- [ ] Panes without overflow do not show distracting scroll affordances.

## Further context

Decide the exact visual design during specification; the request permits either a percentage indicator, a scrollbar-like hint at the Pane edge, or both.

## Comments

### 2026-08-06 — Design-session checkpoint

Accepted contract:

- Every vertically scrollable content area uses the convention, but only when its content exceeds its visible viewport. This applies regardless of Pane focus. Content without overflow shows neither rail nor percentage.
- Determine overflow using the Pane's full inner text width, before reserving space for a rail. Thus a rail must never create its own reason to appear.
- Once actual overflow exists, reserve the two rightmost inner columns: one blank separator and one rail column. Rewrap the content in the remaining width; the resulting rendered rows determine the scroll range, Thumb, and percentage. The rail never overwrites content or causes terminal wrapping.
- The rail is a plain dark-grey `│`, with no filled background track. Its Thumb is a proportional `█`: medium-grey while unfocused and light-grey while focused. Focus never uses the pink accent.
- For an overflowing viewport of at least two rows, Thumb height is `clamp(floor(viewportRows² / contentRows), 1, viewportRows - 1)`. It is therefore at least one terminal row for very long content and leaves at least one rail row exposed when content only just exceeds the viewport. The Thumb begins at the top at scroll position `0`, ends at the bottom at the final scroll position, and follows intermediate positions proportionally with rounding.
- The global Footer shows the focused scrollable content's percentage, right-aligned as `0%` through `100%`, only when it fits alongside all existing status/key hints. It requires five free columns (a separator plus `100%`); hints are never truncated to make room for it.
- The accepted percentage is the visual-viewport position: the top rendered terminal row relative to the maximum possible top row. It is `0%` at the first viewport position and `100%` when the final rendered row is visible at the bottom. Logical source lines, list entries, and cursor position do not determine it.
- `prototype/scrollbar` established the percentage decision but was rejected as a visual reference and removed. Other prototypes are unaffected.

Before implementation:

- Centralize the semantic dark-, medium-, and light-grey scrollbar colours in the TUI theme.
- Add narrow-width rendering checks, including rail reservation, no rail for content that only overflows after reservation, Thumb bounds, and Footer-space omission.
