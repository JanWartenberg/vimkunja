# 01 — Delete a Task with explicit confirmation

**What to build:** A user can delete a selected Task only after an explicit confirmation, with Vikunja-compatible persistence, error feedback, and refresh.

**Blocked by:** Needs a deletion interaction and confirmation spec.

**Category:** enhancement

**Status:** needs-triage

- [ ] Define command/keybinding, confirmation wording, and cancellation behavior.
- [ ] Preserve existing SQLite/Vikunja persistence semantics without schema changes.
- [ ] Keep failures visible in the footer and session error log; refresh only after success.
- [ ] Cover Store persistence and TUI confirmation/failure/refresh behavior.

## Further context

Archived source: `SPEC_FUNDAMENT.md`, “Task deletion (future feature)”.
