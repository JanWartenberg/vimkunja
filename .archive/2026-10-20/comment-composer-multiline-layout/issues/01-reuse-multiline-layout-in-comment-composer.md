# 01 — Reuse multiline layout in the Comment composer

**What to build:** The Comment composer uses the shared multiline text layout introduced for Description rendering, so wrapping, empty lines, viewport scrolling, and cursor placement cannot drift apart.

**Implementation prerequisites:** The private `layoutMultilineText` seam in `internal/tui` is implemented and covered through Description screen-level tests.

**Category:** maintenance

**Priority:** Low

**Status:** completed

- [x] Add screen-level regression coverage for wrapped and consecutive-empty Comment lines before changing the composer.
- [x] Replace the Comment composer's independent wrapping and viewport slicing with `layoutMultilineText`.
- [x] Keep Comment modal sizing, chrome, modes, save/discard behavior, outside the shared layout; insertion cursor policy belongs to the common input-cursor module.
- [x] Do not introduce a generic editor framework, public interface, cache, or new package.

## Implementation — shared insertion cursors

Jan explicitly requests identical Insert cursor behavior in all text fields, superseding the earlier local underscore fallback. Comment composition now uses layoutMultilineText for wrapping and source cursor rows, keeps the edited row visible instead of always showing the tail, and uses the common insertion anchor/terminal bar. Modal chrome and save/discard/mode behavior are retained. Screen regressions cover wrapping, consecutive empty lines, editing earlier lines, Unicode, narrow view, terminal bar/fallback and returning to Normal. No generic editor framework or new package was introduced.

## Closure reconciliation — 2026-10-20

The implementation scope is complete and its regression coverage passes in the
current main working tree. Archived during the release subtask cleanup. This
records technical completion and preserves existing manual acceptance evidence;
it does not claim a new physical Windows run or acceptance of unrelated future
features. Remaining release checks are tracked in the active cleanup/publication
tickets.
