# 01 — Keep status-footer hints to one line

**What to build:** The status footer never wraps; it shows only the highest-priority complete hint segments that fit the terminal width.

**Implementation prerequisites:** None.

**Category:** enhancement

**Priority:** Medium

**Status:** completed

## Decision

- Treat the footer as a single-line priority budget, not a wrapping sentence.
- Reserve space for the active-pane label and optional scroll percentage.
- Add whole hint segments in priority order while they fit; omit lower-priority segments rather than truncating a segment or wrapping it.
- When the focused Pane scrolls, reserve the bottom-right percentage in preference to further hints as soon as the Pane label and three complete hint segments fit; below that threshold, hints win.
- At widths too narrow for the normal hint set, fall back progressively to a compact `:h help` hint, then `:h`.
- Use one ANSI-width-safe `renderFooterHints(width, segments)` seam for static Pane hints, command input, search input, and error feedback; confirmation questions are central modals, not footer content.
- For active command and Search input, retain the prompt and cursor-visible input tail when space is scarce, prefixing omitted input with `…`; omit completion feedback before clipping input.

## Examples

- Wide: `j/k select • h/l fold/tree • z fold • Shift+J/K reorder • r reload • :h help • o new`
- Medium: `j/k select • h/l fold/tree • z fold • :h help`
- Narrow: `j/k select • :h help`
- Very narrow: `:h`

- [x] Define the segment priority for each Pane and transient footer state.
- [x] Preserve current status prompts and scroll-position feedback when their reserved space fits.
- [x] Add narrow-width rendering regression coverage.
