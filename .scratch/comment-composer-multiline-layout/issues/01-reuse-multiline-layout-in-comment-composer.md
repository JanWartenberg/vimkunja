# 01 — Reuse multiline layout in the Comment composer

**What to build:** The Comment composer uses the shared multiline text layout introduced for Description rendering, so wrapping, empty lines, viewport scrolling, and cursor placement cannot drift apart.

**Implementation prerequisites:** The private `layoutMultilineText` seam in `internal/tui` is implemented and covered through Description screen-level tests.

**Category:** maintenance

**Priority:** Low

**Status:** needs-triage

- [ ] Add screen-level regression coverage for wrapped and consecutive-empty Comment lines before changing the composer.
- [ ] Replace the Comment composer's independent wrapping and viewport slicing with `layoutMultilineText`.
- [ ] Keep Comment modal sizing, chrome, modes, save/discard behavior, and `_` cursor fallback outside the shared layout.
- [ ] Do not introduce a generic editor framework, public interface, cache, or new package.
