# 02 — Delete one Task Comment

**What to build:** A user can deliberately delete the currently selected persisted Task Comment.

**Implementation prerequisites:** The read-and-append Task Comments workflow must be implemented first. Confirm Vikunja-compatible authorization semantics for the sole local User before implementing deletion.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

## Decisions confirmed

- Delete only the currently selected Comment; multiple selection and bulk deletion are outside scope.
- In Comments Normal Mode, `dd` proposes deletion.
- Open a central confirmation modal equivalent to Task deletion, with the prompt `Delete comment?`.
- `y` confirms. `n` or `Esc` cancels without changing selection or scroll position.
- After successful deletion, select the preceding Comment when available, otherwise the following Comment; an empty thread retains its empty Comments workspace.
- Description is a Task field and may be cleared, but is never deleted as an independent record.

## Decisions still required

- Determine whether V1 may delete only Comments authored by the sole local User or every Comment visible through an accessible Task.
- Define failure and external-deletion feedback without changing the Vikunja schema.
