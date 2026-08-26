# 01 — Render Detail pickers as field-local overlays

**What to build:** Project, Priority, and Workflow-status pickers render as bounded overlays anchored to their corresponding Task Detail field, rather than as content inside the scrolling Detail Pane.

**Implementation prerequisites:** None.

**Category:** bug

**Priority:** High

**Status:** triaged

## Decision

Use one bounded, field-local popover component for the Project, Priority, and Workflow-status pickers, independent of choice count. It is local to the Task Detail Pane; it never crosses that Pane's border or becomes a whole-screen modal in normal layouts.

### Bounds and layering

- Anchor the popover below the active field; flip above when that leaves more vertical room.
- Align its left edge with the field value. Its preferred width is 40 cells; shift it left only as needed to remain within the Task Detail Pane, and use the full available Detail content width when narrower.
- Size its choice viewport to the available space. One visible choice remains a valid instance of the same popover; do not introduce a compact or takeover variant based on list length.
- If no field-adjacent space can show a choice, the same framed popover may cover the Task Detail Pane's content area. Never crop its frame or choices.
- Preserve the active field visibly behind the popover, but deactivate all underlying Pane and field-border emphasis. The popover is the sole active visual element.
- Freeze Detail navigation and scrolling while a popover is open. Do not duplicate picker rows in the Detail content and do not let a Detail scrollbar intersect its frame.
- Follow the shared text-only fallback below `32×9`; retain the one-line status footer.

### Interaction

- `i` on a focused picker field opens its popover. `j`/`k` and arrows select immediately; `Enter` accepts; `Esc` cancels without changing the draft.
- `Ctrl+D`/`Ctrl+U` page by the currently visible choice-row count. `gg`/`G` select the first/last choice. Every selection movement keeps the selected choice visible.
- Project accepts ordinary text immediately as its filter while retaining the same navigation. Query changes and Backspace reset selection and viewport scroll to the first match. `Esc` cancels the entire popover and restores the unchanged Project selection.
- An empty Project filter renders a non-selectable `No matching Projects` row without a scrollbar; `Enter` is a no-op and retains the filter and popover. Priority and Workflow Status use the same non-selectable empty state when unavailable.
- Every overflowing choice viewport, including a height-constrained six-choice Priority picker, shows an internal scrollbar in its reserved inner-right column.
- On terminal resize, keep the popover open; recompute bounds and placement, preserve the Project filter and selected choice, and adjust only viewport scroll needed to keep that choice visible.
- Interaction hints belong only in the one-line status footer, for pickers and other modals. Explicit confirmation answer actions remain pinned inside confirmation modals for now.

## Acceptance criteria

- [ ] Render Project, Priority, and Workflow-status controls as the shared field-local popover without changing their draft/save semantics.
- [ ] Cover below/above anchoring, horizontal bounds, one-row viewports, and content-area coverage when no adjacent choice row fits.
- [ ] Cover frozen Detail background, sole-active-popover emphasis, and non-intersection with Detail scrollbars.
- [ ] Cover filtering, empty states, keyboard navigation, page/jump motions, selection visibility, internal overflow scrollbars, and resize preservation.
- [ ] Cover the `<32×9` text-only fallback and footer-only picker hints.

## Further context

The current in-Pane picker renderer can lose its top or bottom frame when fixed Detail fields and picker choices exceed the available height. A prior whole-screen overlay attempt also duplicated picker content and crossed Pane boundaries; the overlay must therefore be scoped to the Task Detail Pane and anchored to the active field.
