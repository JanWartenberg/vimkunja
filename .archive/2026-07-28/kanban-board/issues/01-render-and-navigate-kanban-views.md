# 01 — Render and navigate Kanban Views

**What to build:** A Project's `view_kind=kanban` can render as a dynamic bucket board in the Tasks Pane, with keyboard navigation and Vikunja-compatible task moves.

**Implementation prerequisites:** Complete [standard Kanban-View provisioning](../../kanban-provisioning/issues/01-provision-standard-manual-kanban-view.md) first; the complete interaction contract is in [`spec.md`](../spec.md).

**Category:** enhancement

**Priority:** High

**Status:** implemented

- [x] Define bucket columns, narrow-terminal behavior, horizontal scrolling, and its indicator.
- [x] Define 2D selection and `hjkl` navigation across buckets and cards.
- [x] Define direct task moves to adjacent and arbitrary buckets, including focus and feedback.
- [x] Preserve persisted bucket order and task positions; decide reordering within a bucket separately.
- [x] Reuse the existing workflow-status Store semantics. The initial Daily-driver scope supports one active/manual Kanban View; zero or multiple manual Kanban Views are specified by the separate follow-up ticket.

## Further context

Archived sources: `TUI_IDEA.md`, Decisions 12–17 and Kanban roadmap; `SPEC_FUNDAMENT.md`, “Feature: Kanban View”. The archived checkmarks document design decisions, not implemented UI.

## Comments

- **2026-07-28, grilling:** The first board scope accepts exactly one `view_kind=kanban` View with `bucket_configuration_mode=manual` per Project. Zero or multiple manual Kanban Views remain outside this ticket.
- **2026-07-28, grilling:** Initial Kanban-View provisioning is a separate, immediately preceding high-priority ticket. It mirrors upstream Vikunja's `Kanban` / `To-Do` / `Doing` / `Done` template and positions, except that pre-existing completed Tasks are assigned to `Done` rather than upstream's `To-Do`; open Tasks are assigned to `To-Do`.
- **2026-07-28, grilling:** Bucket administration is excluded from this Board ticket and tracked separately in `.scratch/kanban-bucket-management/` at low priority.
- **2026-07-28, grilling:** Board layout retains the historical decision: minimum 25 columns per Bucket; show as many as fit; `h`/`l` shifts the horizontal visible range when needed; header arrows indicate hidden Buckets.
- **2026-07-28, grilling:** A Kanban selection is an active Bucket with an optional selected Task. Empty Buckets can be active; `j`/`k` is then a no-op. On horizontal navigation, retain the vertical card position when possible, otherwise select the first or last card.
- **2026-07-28, grilling:** Every board move delegates to the existing workflow-status operation. Moving into the configured Done Bucket sets the global Task Done state; moving out of it reopens the Task. Both adjacent and arbitrary moves retain the standard refresh and error behavior.
- **2026-07-28, grilling:** `Shift+H/L` remains the fast adjacent move. `m` opens a Bucket Picker navigated with `j`/`k`; `Enter` confirms an arbitrary target and `Esc` cancels. This supports any number of Buckets, including hidden ones.
- **2026-07-28, grilling:** After a successful move, the moved Task remains selected in its target Bucket; the active Bucket and horizontal viewport follow it. A failed move leaves selection and Board state unchanged and reports the existing footer error.
- **2026-07-28, grilling:** Cards render in their persisted per-View `task_positions.position` order. A Bucket move does not alter that position; no implicit append, prepend, or reordering occurs. Reordering is deferred.
- **2026-07-28, grilling:** Navigation and adjacent moves at the first/last Bucket are silent Vim-style no-ops: no footer message and no flash.
- **2026-07-28, implementation:** Implemented by the standard-provisioning prerequisite and the Board Store/TUI slice; see commit pending this archive.
