# 02 — Delete one Task Comment

**What to build:** A user can deliberately delete the currently selected persisted Task Comment.

**Implementation prerequisites:** The read-and-append Task Comments workflow is implemented. V1 uses the sole local User to enforce Vikunja-compatible author-only deletion.

**Category:** enhancement

**Priority:** Low

**Status:** completed

## Decisions confirmed

- Delete only the currently selected Comment; multiple selection and bulk deletion are outside scope.
- In Comments Normal Mode, `dd` proposes deletion.
- Open a central confirmation modal equivalent to Task deletion, with the prompt `Delete comment?`.
- `y` confirms. `n` or `Esc` cancels without changing selection or scroll position.
- After successful deletion, select the preceding Comment when available, otherwise the following Comment; an empty thread retains its empty Comments workspace.
- Description is a Task field and may be cleared, but is never deleted as an independent record.

## Decisions confirmed

- V1 deletes only Comments authored by the sole local User; Comments authored by another User are not deletable.
- Store failures and already-missing Comments use normal Vimkunja error feedback without changing the Vikunja schema.

## Implementation checkpoint

Implemented with `dd`, a central confirmation modal, author-only Store enforcement, selection adjustment after deletion, empty-thread handling, and regression coverage. Manual UI evaluation passed; `go test ./...` passes.

## Closure reconciliation — 2026-10-20

The implementation scope is complete and its regression coverage passes in the
current main working tree. Archived during the release subtask cleanup. This
records technical completion and preserves existing manual acceptance evidence;
it does not claim a new physical Windows run or acceptance of unrelated future
features. Remaining release checks are tracked in the active cleanup/publication
tickets.
