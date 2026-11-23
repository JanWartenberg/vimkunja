# Pane-local Vim search

Accepted by Jan on 2026-11-23. Implement without database/schema or dependency changes.

## Scope and content

- Search navigates and highlights occurrences; it never filters content or writes data.
- Projects: Project names in tree order, including folded descendants; reveal ancestors of matches.
- Tasks: titles and displayed metadata in the current filtered List/Overview or manual Kanban Board, including off-screen entries. Kanban order is left-to-right, then top-to-bottom.
- Task Detail: displayed fields and the complete readable Description, including unsaved draft text. Exclude HTML tags, invisible Link URLs and separately opened Comments.
- Each occurrence is a match. Fields are independent; anchors refer to logical lines, never wrapped screen rows.
- Initially exclude Help, Information, Comments, dialogs and pickers. Searchable Help is a required follow-up ticket.

## Interaction

- Normal Mode `/` searches forward, `?` backward, with live preview like `incsearch`.
- Enter accepts; Esc restores the starting context, selection, fields, cursors, scroll positions, Project folds and previous search pattern/highlight state.
- Task matches select and preview Task Detail; Project matches preview their Tasks. Focus remains in the searching Pane.
- Accepted searches keep opened branches expanded. Navigation wraps both ways with boundary feedback.
- `n` follows the last search direction; `N` reverses it. Counts such as `3n` repeat navigation.
- One session-local last pattern and history across Panes, always applied only to the focused Pane. Empty Enter repeats the last pattern; report its absence.
- Up/Down recall a separate search history, capped at 100 accepted entries, with draft restoration.
- Case matching follows Vim `ignorecase smartcase`, overridable with `\c`/`\C`.
- Explicitly supported Vim pattern subset: `.`, `*`, `^`, `$`, character classes, `\<`/`\>`, `\(`/`\)`, `\|`, `\+`, `\?`/`\=`, bounded `\{n,m}`, common character escapes and `\c`/`\C`. Reject unsupported constructs, including backreferences and lookarounds; never silently interpret another regex dialect as Vim.
- In Task Detail, a match selects its field and places the text/Description cursor at the match start, in Normal Mode.
- Insert Mode treats search keys as text; initially exclude Visual Mode. Search in the current dirty draft is allowed; cross-Task/Project search navigation is blocked until unsaved changes are resolved.
- During input, require Enter/Esc before changing Pane/View. Search input uses the shared Insert cursor.

## Feedback and state changes

- Highlight all matches and distinguish the current match; show occurrence index/total.
- `:nohlsearch` hides highlights; `Space h` is its default synonym if free. `n`/`N` re-enable highlights.
- No matches or invalid patterns restore the preview origin with editable input and clear feedback. Enter accepts a valid no-match pattern, but invalid patterns stay open.
- Recompute matches when context/content changes. Preview load failures cancel and restore as much of the origin as possible. Invalidate pending preview responses so late data cannot overwrite restored context.
- No changes to Global Search or its return snapshot.

## Verification

Focused pattern, projection and TUI regressions cover Unicode, boundaries, multiple occurrences, folds, filtered/off-screen content, Overview/Kanban order, draft protection, cursor/scroll restoration, asynchronous cancellation/failures, input routing and narrow rendering. Run affected tests, full tests and both builds. Human terminal acceptance remains separate.
