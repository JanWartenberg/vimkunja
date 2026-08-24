# 01 — Stabilize Task metadata layout in narrow windows

**What to build:** List and Overview Views keep a stable metadata layout while scrolling in narrow terminals.

**Implementation prerequisites:** None.

**Category:** enhancement

**Priority:** High

**Status:** completed

## Decision

- Choose one layout profile per View width and complete current result set; never derive visible columns from only the viewport's Tasks.
- Reserve an 18-cell minimum title budget.
- When metadata fits within that budget, truncate overlong Task titles from the end with a single ellipsis (`Long task title…`), including the selected row, rather than changing visible columns while scrolling.
- When metadata still cannot fit, hide columns consistently for the entire View: Project identifier first (Overview only), then Due Date, then Priority. Priority remains visible longest; overdue Tasks have no exception. Never hide the Task title.
- When even the 18-cell title budget cannot fit beside metadata, hide all metadata and give the title every remaining cell; truncate it from the end when necessary.
- Recompute the layout profile only when the result set changes (reload, status/filter change, Task title edit, create, or delete), never while scrolling.

- [x] Apply one shared layout decision to List and Overview rendering.
- [x] Add scroll regression coverage with a long title outside the initial viewport.
