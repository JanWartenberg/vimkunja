# 01 — Show, assign, and manage Task Labels

**What to build:** Vimkunja displays a Task's Labels in the Task Detail, lets the user assign or remove Labels through a keyboard-first picker, and provides a global Label Overview for creating, renaming, deleting, and navigating existing Labels.

**Implementation prerequisites:** Interaction and SQLite compatibility confirmed. Do not change the Vikunja schema.

**Category:** enhancement

**Priority:** Medium

**Status:** ready-for-agent

## Decided interaction

### Task Detail

- Add a `Labels:` field after Due Date and before Workflow Status. It shows the Task's assigned Label titles; an empty field reads `none`.
- `Space+l` opens the Label picker for the open Task. The field is also reachable with normal Task Detail navigation.
- In the focused Labels field, `h`/`l` select an assigned Label and `x` stages removal of that Label from the current Task. This removes only the association, not the global Label. The changed assignment is saved by the Task Detail's existing `:w`, `:wq`, and `:wqa` flows and discarded by `:q!`.

### Unified Label picker

- The picker is the common Label UI. In Task context it shows every global Label with its assignment state for that Task and accepts free-text input with autocomplete.
- `j`/`k`, arrows, `gg`/`G`, and `Ctrl+D`/`Ctrl+U` navigate matches in Normal Mode. `i`/`a` enter the input; `Esc` first leaves input and then closes the picker back to the Task Detail.
- Selecting an existing Label with `Enter` toggles its pending assignment. An already assigned Label is therefore removable in the picker as well as with `x` in the Detail field.
- If the entered title has no exact normalized match, `Enter` creates that Label immediately and stages its assignment to the current Task. The global Label remains if the user later discards the Task Detail, so it is available for reuse.
- A blank query lists Labels alphabetically. Matching is case-insensitive substring matching. Exact normalized titles are reused; creation and rename reject a duplicate normalized title rather than creating or merging a second Label.

### Global Label Overview

- `:labels` is available from every normal context and opens a global Label Overview without changing the current Task, Project, or View context.
- It lists every Label alphabetically, including Labels with no assignments, and shows each Label's total Task-association count. The count and a deletion warning include associations on archived Tasks because global deletion removes all of them.
- `Enter` opens the existing label-filtered Overview for the selected Label. That Overview retains its existing archive visibility and status-filter composition.
- `o` opens the same free-text picker to create a Label. A new global Label is persisted immediately when confirmed.
- `i` renames the selected Label through the picker input and persists immediately when confirmed.
- `dd`, then `y`, deletes the selected Label immediately. If it has one or more Task associations, the confirmation names the Label and the affected Task count; `n` or `Esc` cancels. Deletion removes the Label and all of its associations, never the Tasks. An unassigned Label may be deleted with the same command without an affected-Task warning.
- `Esc` or `:q` closes the Overview and restores the exact prior context. `:qa` quits Vimkunja.

### Persistence and errors

- Global Label creation, rename, and deletion persist immediately after their respective confirmation; they do not wait for a Task `:w`.
- Task-assignment changes remain part of the Task Detail draft. Saving creates/removes `label_tasks` rows atomically with the Task update; creating a new Task writes its selected assignments after the Task has an ID.
- Global writes create Vikunja-compatible `labels` rows using the active actor as `created_by_id`, timestamps, and no schema change. Assignment rows carry their required timestamp.
- Reload Label data after every successful global mutation. On validation or database failure, retain the input or draft, keep the picker/Overview open, and show the existing error feedback. A Task save must reject a selected Label that was deleted externally rather than silently losing an assignment.

## Key interfaces

- Extend the Label read model with assignment counts and Task-specific assignment lookup; keep Global Search's complete Label discovery and label-filtered Overview reusable.
- Add Store operations for listing Labels, creating, renaming, deleting, and replacing a Task's Label assignments. Keep each multi-row mutation transactional and hide SQLite details behind the Store.
- Extend `TaskDraft` and Task Detail state with selected Label IDs and their original assignment set so dirty tracking, `:w`, and `:q!` remain consistent.
- Add a dedicated Label picker/Overview state rather than coupling Label management to the Search overlay's result model.

## Acceptance criteria

- [ ] Task Detail renders the `Labels:` field after Due Date; it displays assigned Labels or `none`, supports selecting an assigned Label, and stages direct `x` removal without deleting the global Label.
- [ ] `Space+l` opens the unified picker for an open Task; it supports the specified input, autocomplete, navigation, assignment toggling, cancellation, and Task `:w`/`:q!` semantics.
- [ ] A nonmatching picker entry creates a global Label immediately and stages its assignment; an exact normalized match reuses the existing Label and duplicate normalized titles are rejected on create and rename.
- [ ] `:labels` opens the global Label Overview from every normal context, lists all Labels with total association counts, and `Enter` opens the existing label-filtered Overview.
- [ ] Global creation (`o`), rename (`i`), and deletion (`dd`, then `y`) use the specified persistence and confirmation behavior; deletion removes associations but never Tasks.
- [ ] Unassigned Labels remain listed and reusable; removing the last Task assignment neither prompts nor deletes the Label.
- [ ] Store-level tests cover all Label reads and mutations, timestamp/actor fields, transactional replacement, duplicate handling, archived-Task counts, deletion effects, and external-deletion save failure.
- [ ] TUI tests cover field order and rendering, picker navigation and draft behavior, global Overview navigation and return, immediate mutation feedback, confirmations, errors, and unchanged unsaved-edit protections.

## Out of scope

- Label colors, descriptions, sharing, ownership changes, or permissions UI.
- Bulk assignment across several Tasks.
- Persistent Label filtering outside the existing label-filtered Overview.
- Fuzzy matching, query operators, or Search ranking; those remain Global Search follow-up work.

## Further context

Global Search owns read-only Label discovery and opening the label-filtered Overview. This ticket owns Task Detail Label display, assignment/removal, and global Label management.
