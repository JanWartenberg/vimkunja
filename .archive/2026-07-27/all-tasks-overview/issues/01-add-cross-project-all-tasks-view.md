# 01 — Add cross-project All Tasks overview

**What to build:** A user can open a flat overview of tasks across Projects, with an explicit default treatment for completed tasks and stable task metadata.

**Implementation prerequisites:** None — implemented in `f5d944e`.

**Category:** enhancement

**Priority:** High

**Status:** resolved

- [x] Overview replaces the project-specific Tasks Pane in the same right-hand slot; Projects stay hidden by default.
- [x] `-` opens Overview; `Space+a` toggles to the remembered Project context; `o` starts with a Project Picker.
- [x] Completed Tasks are hidden by default; `f`, then `a` shows all Tasks.
- [x] Default ordering is Due Date ascending, with undated Tasks afterward by Priority; `s` exposes the agreed alternate sorts.

## Further context

This is the first Daily-driver feature: keep its initial scope to opening, browsing, and selecting Tasks across Projects. Advanced filtering, alternate sort controls, and richer creation behavior may follow once the overview is usable.

Archived source: `SPEC_FUNDAMENT.md`, “Feature: OverviewView”, and `TUI_IDEA.md`, “Gesamtsicht”.

## Comments

### 2026-07-22 — TUI prototype awaiting maintainer evaluation

Throwaway prototype: [`prototype/overview_tui/`](../../../../prototype/overview_tui/)

Run it with:

```bash
go run ./prototype/overview_tui
```

It tests these deliberately provisional decisions:

- Overview Pane and the project-specific Tasks Pane are peers in the same right-hand Pane slot; `-` opens Overview, while `Enter` on a Project opens that Project's Tasks Pane.
- The Task Detail Pane is hidden until `Enter` explicitly opens the selected Task; `Esc` closes it.
- Default ordering is Due Date ascending (nearest date first); Tasks without a Due Date follow and are ordered by Priority descending.
- A netrw-inspired second step is available for evaluation: `s` opens a sort picker (`d` Due ↓, `u` Due ↑, `p` Priority ↓, `t` Title) and `f` opens a filter picker (`o` open, `a` all, `v` overdue, `t` due today).

### 2026-07-27 — Resolved

Implemented in `f5d944e` with a cross-Project Store query, Overview navigation, sorting/filtering, Project Picker creation flow, context restoration, Due Date presentation, and Store/TUI regression coverage. `go test ./...` passed.
