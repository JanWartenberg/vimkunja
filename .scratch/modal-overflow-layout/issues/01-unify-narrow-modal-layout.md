# 01 — Unify narrow-modal layout and overflow behavior

**What to build:** Interactive overlays and picker frames preserve their functionality without horizontal or vertical overflow in narrow terminals, using small shared layout primitives rather than duplicated sizing logic.

**Implementation prerequisites:** None.

**Category:** enhancement

**Priority:** Medium

**Status:** completed

## Decision

- Cover the task-delete confirmation, Global Search, Labels / Label Picker, and Project, Priority, and Workflow-status pickers.
- Treat the Kanban move picker as an in-Pane interaction: share sizing and truncation helpers where useful, but do not force it into the overlay abstraction.
- Keep modal-specific content, selection models, and key handling local; share only shallow rendering primitives for bounds, frames, ANSI-width-safe text fitting, and row-height budgeting.
- Normal-size overlays use the shared active-border treatment.
- All confirmation questions are central modals. Move the Kanban View provisioning question out of the status footer; Task and Label deletion already use central confirmation UI.
- On terminals under **32 columns** wide or **9 rows** high, drop border and padding in favour of content; replace the entire Pane content area with a full-screen text-only fallback while retaining the one-line status footer.
- Wrap confirmation copy rather than truncate it. Keep answer actions pinned and visible; only when wrapped content exceeds the available message area, make that message region scroll with `j`/`k`, arrow keys, `Ctrl+U`/`Ctrl+D`, `gg`, and `G`.
- Always try the normal wrapped layout before introducing confirmation-message scrolling.
- Bound dynamic input, instructions, and result rows horizontally; bound result regions vertically with their existing scroll/selection behavior.

## Candidate seams

- `modalBounds(viewport, preferredWidth, preferredHeight, chrome)`
- `renderModalFrame(spec)`
- `fitModalText(text, width)` / `fitModalRows(rows, height)`

- [x] Define the minimum usable content width and text-only fallback threshold: under 32 columns or 9 rows.
- [x] Replace duplicated overlay sizing while preserving each interaction's behavior.
- [x] Add narrow-width and height regression coverage for every covered interaction.
