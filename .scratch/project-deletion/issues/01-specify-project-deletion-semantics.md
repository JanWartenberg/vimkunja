# 01 — Specify Project deletion semantics

**What to decide:** Define whether and how Vimkunja may delete a Project, including the fate of its contained Tasks.

**Blocked by:** Requires a product decision before implementation.

**Category:** enhancement

**Status:** needs-triage

- [ ] Decide whether contained Tasks are deleted, archived, or moved.
- [ ] Define confirmation, cancellation, error feedback, and post-success focus behavior.
- [ ] Verify Vikunja-compatible SQLite persistence without schema changes.
- [ ] Split implementation work into follow-up tickets only after the semantics are fixed.

## Further context

Archived source: `SPEC_FUNDAMENT.md`, “Project deletion (separate future feature)”.
