# 01 — Show Checkbox-list progress summary

**What to build:** A Vimkunja user sees a completed/total (`n/m`) summary for each Checkbox list embedded in a Task Description, without changing the enclosing Task's Done status.

**Implementation prerequisites:** The first read-only summary slice supports structured Vikunja Checkbox-list HTML and indented plain-text checkbox blocks in the existing Description projection. Native rich-text editing remains a later prerequisite for live synchronization after checkbox toggles and entry insertion/removal.

**Category:** enhancement

**Priority:** Medium

**Status:** implemented

## Decided scope

- Progress is calculated independently for each Checkbox list.
- There is no aggregate across all checkboxes in the Task Description.
- Users who want one progress indicator should keep those entries in one Checkbox list.
- The summary never changes the enclosing Task's Done status.

## Agreed contract

- The summary is `Checklist progress: n/m`, on its own logical line immediately after each nonempty Checkbox-list block. It soft-wraps at narrow widths without truncating the count; all wrapped rows retain the summary style.
- The summary uses a dedicated `checkboxListSummary` theme role, initially mapped to muted grey; it is visible outside Insert Mode and is informational, non-navigable, and non-editable. Insert Mode hides summaries to avoid transient layout changes while a marker is being edited.
- Description Normal Mode supports `Ctrl+E` / `Ctrl+Y` to scroll down/up one rendered row independently of the source cursor. This exposes intermediate or wrapped summaries even in a one-row viewport; offscreen cursors are hidden, and editing or cursor navigation resumes cursor-following.
- An outer HTML `<ul data-type="taskList">` defines one block. Nested checkbox entries are flattened into that block; HTML whitespace does not split it.
- Count valid descendant `<li data-type="taskItem">` entries. `data-checked="true"` (case-insensitive) is completed; every other value is open. Empty-text entries still count. Plain text `[ ]`, `[]`, and `[x]` markers at the first non-whitespace position are supported as a fallback; `[ ]` and `[]` are open, while `[x]` is completed. Contiguous marker lines form one block, while blank or non-checkbox lines split blocks.
- Empty or malformed blocks with no valid checkbox entries produce no summary; `0/0` is never shown. Plain text that merely resembles a checkbox elsewhere is not recognized. A plain-text summary is not recalculated or displayed during Insert Mode.
- Native editing will update the summary live in the draft, and deleting the last entry removes the entire block. The enclosing Task's Done state is never derived from the summary.

- [x] Define summary placement, narrow-terminal behavior, and distinct theming.
- [x] Implement the initial read-only summary from Vikunja Checkbox-list HTML and strict plain-text checkbox blocks.
- [x] Count flattened valid entries in each block, with no `0/0` summary.
- [ ] Deferred to native Rich-Text editing: live synchronization and removal of empty list structures after the last entry is deleted. Plain-text draft summaries already recompute on render.
- [x] Never derive or mutate the enclosing Task's Done status from the summary.
