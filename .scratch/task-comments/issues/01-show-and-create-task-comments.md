# 01 — Show and create Task Comments

**What to build:** A Task Detail can show its Vikunja Comments and append a new Comment through a keyboard workflow.

**Implementation prerequisites:** Complete the remaining interaction decisions below. The local schema stores Comment HTML, author ID, Task ID, and created/updated timestamps.

**Category:** enhancement

**Priority:** Low

**Status:** implemented

## Decisions confirmed

- A Task's Comments form an append-only chronological thread in V1: oldest first, newest last, with the newest Comment visible when opened.
- Load all Comments for the displayed Task asynchronously together with Task Detail data. Do not impose an application limit or expose pagination; constrain only the rendered viewport.
- Show `Comments (…)` while loading and then `Comments (n)`. Opening the overlay during loading shows its conventional loading state; load failures use normal error feedback and an overlay error state.
- Existing Comment HTML uses the same read-only terminal projection as Description HTML. Editing persisted Comments remains outside V1.
- Each Comment has a compact header containing display name (falling back to username), absolute local creation time as `YYYY-MM-DD HH:mm`, and `edited` only when `updated` is later than `created`. Hide technical IDs.
- Persist timestamps as canonical Vikunja-compatible UTC and convert them to the system timezone only for display. Never show relative times.
- V1 remains single-user: resolve the sole local Vikunja User as author without hard-coding an ID. If no unique local User exists, display an error and perform no write. Keep author resolution replaceable by future user context.
- Comments require a persisted Task ID. `gc` on an unsaved new Task reports `Save Task before adding Comments.`
- `gc` is the direct jump to Comments. `gn` is the direct jump to Description without entering Insert Mode.
- With Comments active, `a` appends a Comment, making `gca` the fast open-and-append sequence.
- The Comment composer shares the Description editor contract and, where practical, its editor core: `Enter` inserts a line, `Esc` returns to Normal Mode while retaining the draft, `:w` saves, `:q` refuses to lose unsaved work, and `:q!` discards it.
- A V1 Comment draft is plain text. On save, escape it and serialize paragraphs as `<p>` and intra-paragraph line breaks as `<br>`. Reject empty or whitespace-only Comments. Keep serialization behind a seam that can later accept the shared structured rich-text model.
- A Comment draft and other open Task changes belong to one Task edit session. `:w` persists all open Task changes and creates the Comment as one all-or-nothing operation; `:q!` discards both. After success, keep the Task open and select the new Comment.
- Comment navigation uses `j`/`k` or arrows for next/previous, `gg` for oldest, `G` for newest, and `Ctrl+D`/`Ctrl+U` for pages. The viewport follows the selected Comment. `Esc` leaves Comments.
- Reserve one distinct row for `Comments (n)` at the bottom of the Task Detail Pane, after the Description. Keep the metadata → Description → Comments reading order even in constrained layouts.
- Visually frame the Description so the bottom Comments row cannot be mistaken for Description content.
- `Comments (n)` is an entry point, not an inline thread. `Enter` or `gc` opens an always-large Comments overlay workspace; `Esc` closes it and restores the previous Detail focus.
- The Comments workspace should follow established Vimkunja overlay conventions wherever applicable. Fine visual and interaction details may be refined after evaluating the first implementation.
- When appending, anchor the Comment composer at the bottom of the overlay and keep the existing thread visible in the remaining scrollable area. Let the composer grow with content up to a sensible share of the height; at extremely low heights it may consume almost the full workspace.
- There is no hard Comment-count limit. The initial expectation of small threads is only an optimization assumption.
- Editing persisted Comments remains outside V1. Single-Comment deletion is implemented by the direct follow-up in `02-delete-single-task-comment.md`; multiple selection remains excluded.

## Implementation defaults to validate

- Use existing overlay conventions for empty, error, narrow-terminal, focus restoration, and footer behavior rather than introducing Comment-specific variants.
- Keep failed Task-and-Comment saves in the editor with both drafts intact and show the Store error through normal Vimkunja feedback.
- Refine non-contractual visual details after evaluating the first implementation.

## Implementation checkpoint

The implementation is complete and the hands-on UI evaluation passed. It includes eager loading/counts, the large shared-style overlay, chronological HTML projection, navigation, the Description-like plain-text composer, atomic Task-plus-Comment persistence, `gc`/`gn`, absolute local timestamps, narrow fallback, and regression coverage. `:wq` saves and closes the overlay; save responses invalidate older in-flight Comment snapshots so a newly saved Comment cannot disappear from the UI.

## Compatibility notes

The observed Vikunja table is `task_comments(id, comment, author_id, task_id, created, updated)`, indexed by Task ID. Comment content is HTML. Do not change the Vikunja schema.

Comments are deliberately outside the original Daily-driver milestone and should not delay completed MVP work.

Archived source: `SPEC_FUNDAMENT.md` and `TUI_IDEA.md`, “Kommentare”.
