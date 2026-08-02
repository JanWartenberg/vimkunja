# 01 — Brainstorm timeboxed Focus Plans for ongoing work

**Phase:** M4 — Future exploration

**What to decide:** Determine whether Vimkunja should support a first-class model for regular, timeboxed attention to ongoing Projects or Tasks that have no natural Done state.

**Implementation prerequisites:** None. This is deliberately a discovery and prototyping ticket, not an implementation commitment. Discuss and validate the domain model before altering production behavior, the SQLite schema, or Vikunja compatibility.

**Category:** discovery

**Priority:** Medium

**Status:** needs-triage

## Problem

Classic Tasks have a clear outcome and can appropriately use Priority and Due Date: for example, “edit the 300 Spain photos” eventually ends. Software development and other ongoing work do not: continuing to improve a Project such as Tagger or Vimkunja continually creates more possible work.

The user nevertheless wants to stay in touch with such Projects regularly—ideally most days—while also having an explicit stopping point. Spending the intended time should mean “enough for today”, leaving time and mental rest for other work. A missed day should be visible only as subtle history, never as an overdue or red failure state.

## Working vocabulary to evaluate

- **Focus Plan:** a durable intention to give one or more Projects or Tasks regular, bounded attention. It is not a never-ending Task and has no Done state.
- **Focus Budget:** a configurable amount of time available per day or other cadence. The likely first model is one shared budget for all targets in a Plan; later models may support per-target budgets.
- **Focus Block:** a concrete timed period of attention started and stopped by the user.
- **Focus Segment:** the portion of a Block spent on a specific Project or Task, if a Block can switch targets.
- **Check-in:** an optional short status recorded with a completed Block or Segment.

These terms are provisional. Do not add them to the product glossary until the model is accepted.

## Direction so far

- Initial candidate targets are Coding Agent Projects `Tagger` and `Vimkunja`, rather than their individual tickets. The Projects or Agents already track their own concrete tickets and outcomes.
- A first Focus Plan could offer a configurable daily shared budget such as 30 minutes for both Projects. Whether the budget should be shared or per target must remain configurable/open.
- Reaching the Budget should produce a neutral, supportive stop signal: “enough for today”, with an explicit option to extend deliberately. It must not claim the underlying Project is done.
- Starting a Block may default to a simple status such as “continued coding on Project Foo”; writing a Check-in remains optional.
- Regularity should be shown as quiet context, for example last touched time, time spent today, or days used in the last seven days. It must not create red overdue states or punitive streak mechanics.

## Questions to explore

- Is the primary unit a shared Focus Plan, an individual target Routine, or both?
- Can one Focus Block contain Segments for several targets, and is that useful in practice?
- What should happen at Budget exhaustion: notification only, pause/stop the timer, or require an explicit extension?
- How should a user choose the next target: manually, round-robin, or by suggesting the least recently touched target?
- Which cadence statistics are helpful without becoming pressure or gamification?
- Do Focus Blocks need to be synchronized across devices? If so, can existing Vikunja time tracking or comments provide a suitable representation, or is a new compatible storage contract needed?
- Can this begin as a Vimkunja-local sidecar model keyed by Project/Task IDs, preserving the original Vikunja API and SQLite schema, while accepting that it is initially local to one device?
- Is “Focus Plan”, “Focus Routine”, “Practice”, or another term the clearest user-facing name?

## Exploration plan

- [ ] Compare the baseline workaround—recurring “work on Project X” Tasks—with a Focus Plan, documenting the semantic and UX costs of the workaround.
- [ ] Discuss concrete daily scenarios: shared versus per-target budgets, intentionally skipped days, a Block that switches Projects, and intentional Budget extension.
- [ ] Create two or three isolated, clearly throwaway interactive prototypes if discussion suggests they would answer open state/interaction questions. Candidate models: a shared daily Budget with Segments; independent per-Project Routines; and a calendar/timer-oriented presentation.
- [ ] Decide whether a viable model can remain a local Vimkunja extension without changing Vikunja's API or SQLite schema.
- [ ] Only after the model and compatibility boundary are accepted, create separate implementation tickets.

## Compatibility boundary

Do not distort normal Tasks into artificial daily completions merely to represent ongoing attention. The preferred exploration order is a Vimkunja-local additive model that references ordinary Vikunja Projects/Tasks and leaves all existing API and database contracts unchanged. Synchronization and any use of native Vikunja time-tracking/comment facilities are explicitly undecided.

## Further context

This is a known gap between task management, habit tracking, calendars, and time tracking: a Project has no natural completion, yet unbounded work needs both regular contact and a legitimate stopping point. The purpose of this ticket is to keep the conversation and potential prototype work available for a later milestone without prematurely committing Vimkunja to a new domain model.
