# 01 — Preserve Description access in short terminals

**What to build:** A Task Detail Pane on a short terminal retains an accessible Description while all Task fields remain reachable through a compact, scrollable layout.

**Implementation prerequisites:** None.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

## Acceptance criteria

- [ ] The Task Detail Pane has an outer scroll viewport containing Title, metadata, and Description. In normal Detail navigation, `j/k` select the next/previous field and scroll that outer viewport as needed; `Ctrl+D/U` page the same outer viewport.
- [ ] At fewer than 12 content rows, Project, Priority, Due Date, Labels, Workflow Status, and Done collapse behind one inline **More fields** row. `Enter` expands or collapses it; expanded fields participate in normal field navigation and remain reachable through the outer viewport.
- [ ] At 8 or more content rows, Description retains at least 3 visible rows. Below 8 content rows, retain Title, More fields, and at least 2 Description rows; all remaining content is reachable by outer scrolling.
- [ ] Selecting Description then pressing `Enter` activates Description Normal Mode. `i` from Detail navigation with Description selected enters Description Insert Mode directly. While Description is active, `j/k` and `Ctrl+D/U` navigate/scroll its inner viewport; the first `Esc` leaves Insert Mode and the next `Esc` returns to Detail navigation.
- [ ] The outer Task Detail viewport always has a distinct focus frame/rail and selected-field cue. An active Description viewport has a separate, stronger inner frame/rail and Description cursor; inactive regions remain legible but subdued. Focus transitions never visually imply that both viewports are active.
- [ ] Resize reflows immediately, preserving the focused field and Description cursor/scroll position where possible. Preserve Title, dirty state, save/discard, existing field editing, and narrow-terminal rendering.
- [ ] Add model-level coverage for thresholds, More-fields expansion, outer/inner navigation and focus cues, resize preservation, and scrollbar bounds. Do not change Vikunja persistence or schema.

## Further context

The current Task Detail Pane can display only fixed fields on short terminals, leaving no visible Description rows. The scrollbar work makes the missing Description area more apparent but does not define the compact-layout behavior.
