# 01 — Preserve Description access in short terminals

**What to decide:** A Task Detail Pane on a short terminal must retain a usable Description area instead of letting fixed metadata consume all visible rows.

**Implementation prerequisites:** Needs a compact-layout and navigation contract.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

- [ ] Define the height threshold and which metadata rows collapse, elide, or move before Description becomes inaccessible.
- [ ] Preserve Title, dirty state, focused-field feedback, save/discard, and keyboard navigation in the compact layout.
- [ ] Define whether the Description keeps a minimum visible-row budget and how its scrollbar bounds behave at that size.
- [ ] Cover resize transitions and short-terminal rendering without changing Vikunja persistence or schema.

## Further context

The current Task Detail Pane can display only fixed fields on short terminals, leaving no visible Description rows. The scrollbar work makes the missing Description area more apparent but does not define the compact-layout behavior.
