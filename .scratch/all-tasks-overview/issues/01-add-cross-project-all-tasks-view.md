# 01 — Add cross-project All Tasks overview

**What to build:** A user can open a flat overview of tasks across Projects, with an explicit default treatment for completed tasks and stable task metadata.

**Blocked by:** Needs specification of data source, pane placement, navigation, and default sorting/filtering.

**Category:** enhancement

**Priority:** High

**Status:** needs-triage

- [ ] Decide whether the overview is a View, a Pane, or replaces the Project/Tasks pair while open.
- [ ] Define its entry point (the archived proposal was `-`) and task creation behavior.
- [ ] Define default visibility for completed tasks.
- [ ] Define initial ordering by due date, priority, and project plus future sorting/filter controls.

## Further context

This is the first Daily-driver feature: keep its initial scope to opening, browsing, and selecting Tasks across Projects. Advanced filtering, alternate sort controls, and richer creation behavior may follow once the overview is usable.

Archived source: `SPEC_FUNDAMENT.md`, “Feature: OverviewView”, and `TUI_IDEA.md`, “Gesamtsicht”.

## Comments

### 2026-07-22 — TUI prototype awaiting maintainer evaluation

Throwaway prototype: [`prototype/overview_tui/`](../../../prototype/overview_tui/)

Run it with:

```bash
go run ./prototype/overview_tui
```

It tests these deliberately provisional decisions:

- Overview Pane and the project-specific Tasks Pane are peers in the same right-hand Pane slot; `-` opens Overview, while `Enter` on a Project opens that Project's Tasks Pane.
- The Task Detail Pane is hidden until `Enter` explicitly opens the selected Task; `Esc` closes it.
- Default ordering is Due Date descending; Tasks without a Due Date follow and are ordered by Priority descending.
- A netrw-inspired second step is available for evaluation: `s` opens a sort picker (`d` Due ↓, `u` Due ↑, `p` Priority ↓, `t` Title) and `f` opens a filter picker (`o` open, `a` all, `v` overdue, `t` due today).

**Next session:** maintainer evaluates the interaction and records feedback here before the Overview specification or implementation begins.
