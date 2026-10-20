# 01 — Copy the selected Task to the system clipboard

**What to build:** Add `:copy` to copy a readable plain-text representation of the single selected Task, including its Description and metadata (for example Title and Priority), to the system clipboard for quick sharing.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Define which Task metadata to include and their stable display order/format.
- [ ] Define the readable plain-text projection of HTML Descriptions, including lists, Checkboxes, Links and protected content; reuse the Description projection instead of copying raw HTML or adding a separate parser.
- [ ] Define what happens when no Task or multiple Tasks are selected; copying is initially supported only for exactly one selected Task.
- [ ] Define whether copy uses persisted values or the current Task Detail draft, including unsaved edits.
- [ ] Define behavior for an empty Description and clipboard write failures; report failures through existing error feedback.
- [ ] Check the keybinding contract and implemented clipboard actions after Rich-text integration before assigning an optional Leader-key shortcut. The current `main` planning documents reserve `Space+y` for Link URLs; the Rich-text worktree uses `Space+y` for text and `Space+Y` for Link URLs. Do not assign a shortcut based on the older planning state.

## Further context

On the current `main` baseline, the TUI reads the system clipboard for Insert-mode paste. The Rich-text worktree also implements clipboard writes for text and Link URLs. Schedule this Task-level export after Rich-text integration and reuse the resulting clipboard infrastructure. Copying must leave Task data and drafts unchanged.
