# 01 — Improve Task date, priority, and identifier presentation

**What to build:** A Vimkunja user can scan Task metadata in list and Detail Panes without raw timestamps, unclear priority conventions, or missing project identifiers.

**Implementation prerequisites:** None — planned for M1.

**Category:** enhancement

**Priority:** M1

**Status:** ready-for-agent

- [ ] Implement concise Due Date presentation, including overdue and near-term dates.
- [ ] Implement consistent named and colored Priority presentation in list and Detail Panes.
- [ ] Implement the decided Project/Task identifier placement.
- [ ] Preserve predictable truncation and avoid overflow in narrow Panes.

## Decisions — 2026-07-29

### Due Date

- Store and display dates in English and at date precision only; do not display a time. A non-focused Detail field uses this human presentation; its focused editor remains canonical `YYYY-MM-DD`.
- The existing focused Due Date mask already accepts eight digits (`YYYYMMDD`), renders/stores canonical `YYYY-MM-DD`, and does not require typing separators. M1 adds regression coverage while preserving this behavior.
- Display `today`, `tomorrow`, and `in N days` for due dates in the next seven days. Overdue dates display as red `N days overdue`.
- For later dates, display `04 Aug` in the current year and `04 Aug 2027` in another year.
- An optional ` (Mon, 04 Aug)` suffix may appear only when it fits without reducing the title's available width; the base `in N days` form is always sufficient.
- Done Tasks render their Title, Priority, and Due Date in a faint neutral gray rather than retaining urgency colors.
- Natural-language date terms, including German synonyms and two-letter German weekday abbreviations alongside English three-letter weekday abbreviations, belong to the M2 special-inputs ticket; presentation remains English.

### Priority

- Display priority names plus color, never numeric `P0`–`P5`. Do not add warning symbols. The names are `Low`, `Medium`, `High`, `Urgent`, and `Do now`.
- Priority zero displays as `—` in a summary and as `None` in the picker.

### Project and Task Identifiers

- In the All-Tasks Overview, show the Project Identifier before an aligned Task title. If the Project has no Identifier, use a Project-title fallback truncated to roughly eight characters.
- Do not add a Project Identifier to the normal project Task list or Task Detail pane; the Detail pane retains its editable Project field.
- Do not show a Task Identifier in normal lists. When the Task Detail pane has a free visual row, show `(DB-ID: xxx, Task-Index: yyy)` subtly and read-only for debugging; it must never displace an editable field or Description content.

### Narrow Panes

- Preserve information in this order: Title, Priority, Due Date, Project, then Task Identifier. Lower-priority segments disappear or truncate first.

## Further context

This consolidates the unresolved presentation questions from the former UX TODO. Existing title truncation and empty-description placeholders are already implemented and are not part of this ticket.
