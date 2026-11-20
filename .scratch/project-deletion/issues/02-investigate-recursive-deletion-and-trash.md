# 02 — Investigate recursive Project deletion and trash

**Status:** needs-triage
**Category:** research
**Priority:** Low

Jan deferred this investigation on 2026-11-18. The completed first step only
[deletes empty leaf Projects](../../../.archive/2026-11-18/project-deletion/spec.md)
after confirmation. Do not broaden deletion until these semantics are decided.

- [ ] Establish official Vikunja 2.7.0 behavior for Project/Subproject deletion,
  Tasks already in trash, restoring Tasks and referenced metadata/files.
- [ ] Compare recursively deleting all Tasks/Subprojects with soft deletion.
  Determine whether upstream supports a Project deletion flag and restoration
  without schema changes; do not invent a Vimkunja-only database flag.
- [ ] Decide what happens to Time Entries, background/attachment files, shared
  Projects, subscriptions, notifications and default-Project references.
- [ ] Specify the affected-content preview, confirmation, recovery, errors and
  post-success selection; compare with the existing permanent Task deletion flow.
- [ ] Record the decision, compatibility evidence and implementation test seams.

No recursive deletion, bulk Task deletion, trash or restoration is authorized by
this research ticket.
