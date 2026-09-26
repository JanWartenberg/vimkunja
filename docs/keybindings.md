# Vimkunja Keybinding Contract

This is the durable, context-sorted contract for Vimkunja keybindings. It is an exchange format for discussing and comparing alternative interaction designs.

`internal/tui/help.go` lists implemented bindings and marks deferred View entries explicitly. This contract remains the complete source for agreed and provisional bindings; its status column makes the distinction explicit.

This document is maintained manually for now. Generating or deriving `help.go` from it is deferred until the rich-text binding design has stabilized.

## Status

- **Implemented** — available in the current build and eligible for `:help`.
- **Agreed** — designed and accepted, but not yet implemented.
- **Provisional** — intended for prototype review before it becomes agreed.
- **Deferred** — explicitly outside the current scope.

## Global

| Binding | Action | Status |
| --- | --- | --- |
| `Ctrl+W h/j/k/l` | Move Pane focus | Implemented |
| `Ctrl+W Ctrl+W` / `Ctrl+W w` | Cycle open Panes | Implemented |
| `Space+o` | Toggle the Projects Pane, including while editing a Description | Implemented / agreed rich-text scope |
| `:h` / `:help` | Open in-application help | Implemented |
| `:errors` | Open the session error log | Implemented |
| `:labels` | Open the global Label Overview from any normal context | Implemented |
| `u` / `Ctrl+R` | Undo / redo the latest persisted Task action in List, Overview, Kanban, or Task Detail Normal Mode. | Implemented |
| `:q`, `:qa`, and force variants | Close a Pane or quit Vimkunja | Implemented |

## Task context and View navigation

A Task context consists of a Scope (`Overview` or the current Project) and a View (`List`, `Kanban`, later `Table` or `Gantt`). Overview is a Scope, not a Project View. Vimkunja remembers one last non-Overview View globally, plus the last Project and Task selection needed to return there.

| Binding | Action | Status |
| --- | --- | --- |
| `-` | Open Overview from a Project context; this is always one-way and never toggles. | Implemented |
| `Space+a` | Toggle between Overview and the remembered Project context, including its selected Task. | Implemented |
| `Space+o` | Toggle the Projects Pane. Overview starts with it hidden; opening a Task Detail from Overview temporarily reveals and selects that Task's Project. | Implemented |
| `s`, then `d/u/p/t` | Choose Overview sort: Due ↓ / Due ↑ / Priority ↓ / Title. | Implemented |
| `f`, then `o/a/v/t` | Choose Overview filter: open / all / overdue / due today. | Implemented |
| `Space+v`, then `l` | Select List View for the current Project. | Implemented |
| `Space+v`, then `k` | Select or provision the single manual Kanban View for the current Project. | Implemented |
| `Space+v`, then `t` | Select Table View for the current Project. | Agreed; View deferred |
| `Space+v`, then `g` | Select Gantt View for the current Project. | Agreed; View deferred |
| `:overview` | Open Overview. | Implemented |
| `:list` | Select List View for the current Project context; from Overview, restore the last Project context first. | Implemented |
| `:kanban` | Select or provision the single manual Kanban View for the current Project context; from Overview, restore the last Project context first. | Implemented |
| `:table` | Select Table View for the current Project context. | Agreed; View deferred |
| `:gantt` | Select Gantt View for the current Project context. | Agreed; View deferred |

The Project View switcher is visible above the Tasks Pane only outside Overview and lists only Views available to the current Project. If the remembered View is unavailable, restoring a Project uses List View.

## Search

| Binding | Action | Status |
| --- | --- | --- |
| `Space+f` / `:search` | Open the global Search overlay above the current context in Insert Mode. | Implemented |
| Insert Mode: Arrows / `Ctrl+N` / `Ctrl+P` | Select the preceding/following global Search result while editing the query. | Implemented |
| Insert Mode: `Esc` | Leave query input and enter Search Normal Mode. | Implemented |
| Search Normal Mode: `j/k`, `gg/G`, `Ctrl+D/U` | Navigate the result list. | Implemented |
| Search Normal Mode: `i` / `a` | Return to query input. | Implemented |
| `Enter` | Activate the selected global result: open a Task Detail, enter a Project List View, or open a Label-filtered Overview. | Implemented |
| Search Normal Mode: `Esc` / `:q` | Close global Search and restore its pre-search context. | Implemented |
| Search Normal Mode: `:qa` | Quit Vimkunja completely. | Implemented |
| `Ctrl+O` | Return from an activated global result to the latest session-local Search snapshot. | Implemented |
| `/` | Search within the focused Pane; local-search implementation is deferred. | Agreed |
| `n` / `N` | Repeat the next/previous local-search match; deferred with local search. | Agreed |

## Projects Pane

The Projects Pane renders Vikunja's persisted parent/child hierarchy. Navigation operates on visible rows; folded descendants are skipped.

| Binding | Action | Status |
| --- | --- | --- |
| `j/k`, arrows | Select the next/previous visible Project. | Implemented |
| `gg/G`, `Home/End` | Select the first/last visible Project. | Implemented |
| `Ctrl+D` / `Ctrl+U` | Move by approximately half the visible Pane height. | Implemented |
| `h` | Close an open branch; on a closed branch or leaf, select its parent. | Implemented |
| `l` | Open a closed branch; on an open branch, select its first child. | Implemented |
| `zo` / `zc` / `za` | Open, close, or toggle the selected branch. | Implemented |
| `zM` / `zR` | Close or open every branch. | Implemented |
| `Shift+J` / `Shift+K` | Move the selected Project down/up among its persisted siblings. | Implemented |
| `r` | Reload the complete active Project hierarchy. | Implemented |

## Tasks Pane

| Binding | Action | Status |
| --- | --- | --- |
| `j/k`, arrows | Move Task selection | Implemented |
| `gg/G`, Home/End | First/last Task | Implemented |
| `Ctrl+D` / `Ctrl+U` | Page Task selection | Implemented |
| `Enter` | Open the selected Task in the Task Detail Pane | Implemented |
| List / Overview: `v` | Preview the selected Task in the Task Detail Pane without moving Pane focus; use `Enter` to enter Detail. | Agreed |
| List / Overview: `i` / `Shift+I` | Edit the selected Task title inline at its end / beginning. In edit: arrows, Home/End, Backspace, Delete; `Enter` saves; `Esc` discards; `Ctrl+W` saves before changing Pane focus. | Implemented |
| `o` | Create a Task in the selected Project | Implemented |
| `x` | Toggle the selected Task's Done state | Implemented |
| `dd`, then `y` | Permanently delete the selected Task after explicit confirmation; `n` or `Esc` cancel. | Implemented |
| `r` | Reload the active View | Implemented |

## Kanban Board

| Binding | Action | Status |
| --- | --- | --- |
| `h` / `l` | Select the adjacent Bucket and reveal it horizontally when necessary. | Implemented |
| `j` / `k` | Select the previous / next Task in the active Bucket; empty Buckets remain active. | Implemented |
| `Shift+H` / `Shift+L` | Move the selected Task to the adjacent Bucket. | Implemented |
| `m`, then `j/k`, `Enter` | Select and confirm any target Bucket; `Esc` cancels. | Implemented |
| `Enter` | Open the selected Task in the Task Detail Pane. | Implemented |

## Task Detail Pane

| Binding | Action | Status |
| --- | --- | --- |
| `j/k`, arrows | Navigate editable fields | Implemented |
| `i` | Enter Insert Mode or open the focused picker; Due Date keeps its `YYYY-MM-DD` mask and shows relative-date helpers below it. | Implemented |
| Detail pickers: `j/k`, arrows, `Ctrl+D/U`, `gg/G` | Move, page by visible choices, or jump to first/last; all motions clamp at the boundary. | Implemented |
| Project picker: printable keys / Backspace | Selection-only; typing does not filter Projects. | Implemented |
| `Esc` | Leave Insert Mode or the Due Date picker | Implemented |
| Due Date picker: `j/k`, arrows; digits; `Enter` | Select masked entry or `n` days/weeks/months; confirm the selected value. | Implemented |
| Due Date / Priority: `x` or `dd` | Clear the field in the Task draft; Due Date becomes empty and Priority becomes None. | Implemented |
| `gt` / `gd` / `gp` / `gl` | Jump directly to and edit Title / Due Date / Priority / Labels. | Implemented |
| `gc` / `gn` | Open Task Comments / focus Description. | Implemented |
| Comments: `j/k`, arrows, `gg/G`, `Ctrl+D/U` | Navigate the chronological Comment thread. | Implemented |
| Comments: `a`, then `:w` / `:wq` / `:q` / `:q!` | Append and save / save and close / retain / discard a Comment draft together with Task changes. | Implemented |
| `:w`, `:wq`, `:wqa` | Save the Task | Implemented |
| `:e!` / `:edit!` | Discard the complete existing Task draft, reload its last saved state, and keep Task Detail open. New unsaved Tasks cannot be reloaded. | Implemented |
| `:q`, `:q!` | Close or discard the editor | Implemented |
| `j/k` reaches Done after Workflow Status and before Description | Include Done in the field focus order | Agreed |
| `x` while Done is focused | Toggle the enclosing Task's Done state | Agreed |
| Labels field after Due Date | Show assigned Labels or `none` | Implemented |
| `Space+l` | Open the Task's Label picker | Implemented |
| Label picker Insert: text, arrows / `Ctrl+N` / `Ctrl+P`, `Enter` | Filter Labels, choose a matching Label or create the shown new Label | Implemented |
| Label picker: `Esc`, then `Esc` / `q` | Leave filter input; close the picker | Implemented |
| Label picker Normal: `o` | Select the shown new-Label action | Implemented |
| Labels field: `h` / `l`, then `x` | Select an assigned Label and stage removal from this Task | Implemented |

## Description Normal Mode

| Binding | Action | Status |
| --- | --- | --- |
| `h/l`, arrows | Move the cursor horizontally | Implemented |
| `j/k`, arrows | Move between Description lines | Implemented; replaced by logical paragraph navigation in rich-text work |
| `w/b` | Move by word | Implemented |
| `~` / `count~` | Toggle Unicode-aware character case under the cursor, advance, and stop at the current line boundary. Also applies to Task Title and Comment drafts. | Implemented |
| `gg/G`, Home/End | Start/end of Description | Implemented |
| `Ctrl+D` / `Ctrl+U`, PgDn/PgUp | Scroll by page | Implemented |
| `Ctrl+E` / `Ctrl+Y` | Scroll the Description viewport down/up one rendered row without moving the source cursor; includes read-only checklist summaries. Editing or cursor navigation resumes cursor-following. | Implemented |
| `a` / `Shift+I` / `Shift+A` | Enter Insert Mode after the cursor / at line start / end | Implemented |
| `o` / `Shift+O` | Insert a blank line below / above the current line and enter Insert Mode | Implemented |
| `j/k` | Move between navigable logical blocks:<br>paragraphs, headings, and list/Checkbox-list entries | Agreed |
| `gj/gk` | Move between wrapped visual terminal rows | Agreed |
| `j/k` in a multi-line Code block | Move between its unwrapped source lines | Agreed |
| `j/k` in a Block quote | Move between its child blocks | Agreed |
| `{` / `}` | Move backward / forward by paragraph-motion units; each navigable Description block is one unit and a complete Code block or Block quote remains one unit | Agreed |
| `ö` / `ä` | German-keyboard aliases for `{` / `}` in Normal Mode | Agreed |
| `zh/zl` in an overwide Code block or table | Scroll that block horizontally | Agreed |
| `x` on a Checkbox-list entry | Toggle only that entry's checked state | Agreed |
| `dd` | Delete the current logical Description block; inside a nonempty Code block delete only the current source line, and on an empty Code block delete the block; inside a Block quote delete the current child block and remove the quote after its final child | Agreed |
| `o` / `O` | Insert an empty paragraph after / before the current logical block;<br>inside a Code block insert a Code line, or inside a Block quote a same-depth quoted paragraph, below / above the current line; then enter Insert Mode | Agreed rich-text behavior |
| `i` / `a` / `I` / `A` | Enter Insert Mode at, after, at start, or at end<br>of the current logical line | Agreed rich-text behavior |
| `0` / `^` / `_` / `$` | Move to start / first nonblank / end of the current logical line | Agreed |
| `w` / `b` / `e` / `ge` | Move by rendered semantic words, ignoring inline-format and Link boundaries;<br>stop at blocks | Agreed |
| `gg` / `G` | Jump to start / end of the whole Description document | Agreed |
| `Ctrl+D` / `Ctrl+U` | Move the cursor about one visible Task Detail Pane height down / up | Agreed |
| `u` / `Ctrl+R` | Undo / redo an unsaved Description-draft change | Agreed |
| `Space` | Open the Description formatting palette in a secondary footer.<br>It closes on the second key, `Esc`, or the Leader timeout. | Provisional |
| `Space+1/2/3` | Convert current block to H1/H2/H3 | Provisional |
| `Space+0` | Convert current block to a paragraph | Provisional |
| `Space+u/n/x` | Toggle current block between unordered, numbered, or Checkbox list<br>and a paragraph | Provisional |
| `Space+c/q/r` | Toggle current block between Code block or Block quote and a paragraph;<br>insert a horizontal rule after the current block | Provisional |
| `Space+o` | Toggle Projects Pane; never a Description formatting command | Agreed |
| `Space+l` inside an existing link | Prompt with its current URL; link text remains ordinary Description text | Provisional |
| `Space+y` inside a link | Copy its destination URL to the system clipboard without opening it | Agreed |

## Description Insert Mode

| Binding | Action | Status |
| --- | --- | --- |
| Three or four backticks in an empty paragraph | Create a structured Code block, canonicalize to triple-backtick disclosed fences, and place the cursor inside | Agreed |
| `Enter` | Split the current paragraph into two paragraphs; inside a Code block insert a literal source line; inside a Block quote create a sibling quoted paragraph, or leave the quote from an empty quoted paragraph | Agreed |
| `Shift+Enter` | Insert a hard line break inside the paragraph | Agreed |
| `Enter` at end of a list entry | Add an entry of the same list type | Agreed |
| `Enter` on an empty list entry | Leave the list and create a paragraph | Agreed |

## Description Visual Mode

| Binding | Action | Status |
| --- | --- | --- |
| `v` | Start characterwise Visual Mode | Agreed |
| `V` (Shift+V) | Start logical-block Visual Mode | Agreed |
| Normal-Mode motions | Extend the active Visual selection | Agreed |
| `o` | Swap the active end of the Visual selection | Agreed |
| `Esc` | Cancel Visual Mode | Agreed |
| `viw` / `vaw` | Select inner word / word including surrounding whitespace | Agreed |
| `Ctrl+V` | Rectangular selection | Deferred |
| `Space+b/i/u/s` | Toggle bold/italic/underline/strikethrough on the selection | Provisional |
| `Space+l` | Prompt for a URL and apply it to the selection; a blank URL removes the link | Provisional |
| `Space+0` | Remove all inline formatting, including Links, while preserving text and blocks | Provisional |
| `d` | Delete exactly the selected text; delete selected logical blocks after `V` | Agreed |


## Footer hint priority

The footer is a one-line display budget. It shows complete segments only; it
never wraps or clips a static hint. The active Pane label is retained when it
fits, but yields to usable input or error feedback and, at extreme widths, to
`:h`. If even `:h` cannot fit, no hint text is shown.

Static footers use this order: primary action, navigation, `:h help`, then
secondary actions. `:h help` contracts to `:h` before any lower-priority action
is considered. A scrolling Pane reserves its right-aligned percentage once its
label and three actionable segments fit; otherwise hints win.

| Context | Primary | Navigation | Secondary |
| --- | --- | --- | --- |
| Projects | `j/k select` | `h/l fold/tree` | fold commands, reorder, reload, new Task |
| List / Overview | `j/k select`, `Enter detail` | Task navigation | new Task, sort, filter, return |
| Kanban | move Bucket, choose Bucket | card navigation | Done, Pane cycling, new Task |
| Task Detail normal | edit | field navigation | save, close, Pane switching |
| Task Detail insert | `Esc normal` | — | — |
| Help / Info | close Pane | — | Pane switching where available |
| Leader, Views, Sort, Filter | shown choice actions | — | cancel where available |

Command and Search input retain their prompt and cursor-side input tail, with
an ellipsis for omitted leading input. Completion feedback yields before input
is clipped. An active error owns the footer budget and shows only `Error:` plus
the useful error tail.

## Notes for future binding proposals

- A **Pane** is focusable; a **View** is a Task representation inside the Tasks Pane. See [`../CONTEXT.md`](../CONTEXT.md).
- A Checkbox list is embedded rich text in a Description; it is distinct from the Task Done state.
- Preserve documented Vim Visual-Mode motions and operators. The rejected direct `b`, `i`, `u`, `s`, and `l` proposal and sources are recorded in [the binding review](../.scratch/description-rendering/research-vim-visual-bindings.md).
- The Description format-palette presentation remains open in [the rich-text contract](../.scratch/description-rendering/spec.md).
