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
| `u` / `Ctrl+R` | Undo / redo the latest persisted Task action in List, Overview, Kanban, or Task Detail Normal Mode. | Implemented (06); synthetic tests, practical terminal acceptance pending |
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


HTML Descriptions use the Native draft editor (06): `gn`, `Enter` activates navigation; `i/a/I/A` enters Insert. Ordinary editing, Visual Mode, local history and the Space palette are implemented. Advanced structure, Code/Tab/retab and syntax/URL actions are implemented in 07–09; practical terminal acceptance remains outstanding. Plain-text editing retains the existing behavior.

Native Normal Mode also supports `ciw`/`caw` and `diw`/`daw`. Change selects the semantic word, deletes it and enters Insert; Esc cancels an unfinished prefix. A completed word change (replacement plus typing until Esc) is one local Undo/Redo step. Opaque word replacements require confirmation and retain adjacent punctuation.

| Binding | Action | Status |
| --- | --- | --- |
| `h/l`, arrows | Move the cursor horizontally | Implemented |
| `j/k`, arrows | Plain text: move between Description lines | Implemented |
| `w/b` | Move by word | Implemented |
| `~` / `count~` | Toggle Unicode-aware character case under the cursor, advance, and stop at the current line boundary. Also applies to Task Title and Comment drafts. | Implemented |
| `gg/G`, Home/End | Start/end of Description | Implemented |
| `Ctrl+D` / `Ctrl+U`, PgDn/PgUp | Scroll by page | Implemented |
| `Ctrl+E` / `Ctrl+Y` | Scroll the Description viewport down/up one rendered row without moving the source cursor; includes read-only checklist summaries. Editing or cursor navigation resumes cursor-following. | Implemented |
| `a` / `Shift+I` / `Shift+A` | Enter Insert Mode after the cursor / at line start / end | Implemented |
| `o` / `Shift+O` | Insert a blank line below / above the current line and enter Insert Mode | Implemented |
| `j/k` | Move between navigable logical blocks:<br>paragraphs, headings, and list/Checkbox-list entries | Implemented for native read/navigation (05) |
| `gj/gk` | Move between wrapped visual terminal rows | Implemented for native read/navigation (05) |
| `j/k` in a multi-line Code block | Move between its unwrapped source lines | Implemented for native read/navigation (05) |
| `j/k` in a Block quote | Move between its child blocks | Implemented for native read/navigation (05) |
| `{` / `}` | Move backward / forward by paragraph-motion units; each navigable Description block is one unit and a complete Code block or Block quote remains one unit | Implemented for native read/navigation (05) |
| `ö` / `ä` | German-keyboard aliases for `{` / `}` in Normal Mode | Implemented for native read/navigation (05) |
| `zh/zl` in an overwide Code block or table | Scroll that block horizontally | Implemented for native read/navigation (05) |
| `x` on a Checkbox-list entry | Toggle only that entry's checked state | Agreed |
| `dd` | Delete the current logical Description block, including Native text and inline Opaque content, even on inline-atom focus; confirm the whole deletion if Opaque content is included. Inside nonempty Code delete only the source line; final nonempty line deletion keeps empty Code. Another dd deletes that empty block. Inside a Block quote delete the current child block and remove the quote after its final child. | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `o` / `O` | Insert an empty paragraph after / before the current logical block;<br>inside a Code block insert a Code line, or inside a Block quote a same-depth quoted paragraph, below / above the current line; then enter Insert Mode | Agreed rich-text behavior |
| Code Normal Mode: `ü` / `Ü` | Create a normal paragraph after / before the entire current Code block, move there and enter Insert Mode; preserve Code contents | Implemented (08); synthetic editor/config tests, real terminal acceptance pending |
| Code Normal Mode: `]`, then `Space` / `[`, then `Space` | Create a normal paragraph after / before the entire current Code block and move there in Normal Mode; `i` starts typing | Implemented (08); synthetic editor/config tests, real terminal acceptance pending |
| `:selectblock` | Explicitly select the entire current logical block in `VISUAL BLOCKS`, including a standalone Code block | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `:retab` / `:retaball` | Convert all Tabs to the next tabstop in the current Code block/selected Code lines, or all editable Description text including prose; preserve Opaque content. One Undo step; no V1 numeric arguments or bang variant. | Implemented (08); synthetic editor/config tests, real terminal acceptance pending |
| `i` / `a` / `I` / `A` | Enter Insert Mode at, after, at start, or at end<br>of the current logical line | Implemented (06/09), including disclosed editable delimiters; synthetic tests, terminal acceptance pending |
| `0` / `^` / `_` / `$` | Move to start / first nonblank / end of the current logical line | Implemented for native read/navigation (05) |
| `w` / `b` / `e` / `ge` | Move by words across inline-format and Link boundaries; stop at blocks.<br>Disclosed editable delimiters are ordinary characters; protected markers consume no motion counts.<br>Each complete Opaque subtree is one word-motion unit for `w/b/e`. | Implemented for native read/navigation (05); word traversal follows semantic text across disclosed marks |
| `gg` / `G` | Jump to start / end of the whole Description document | Implemented for native read/navigation (05) |
| `Ctrl+D` / `Ctrl+U` | Move the cursor about one visible Task Detail Pane height down / up | Implemented for native read/navigation (05) |
| `u` / `Ctrl+R` | Undo / redo an unsaved Description-draft change | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `Space` | Open the formatting palette inside the footer frame; wrap complete shortcuts across rows. If this leaves fewer than three Description rows, use the primary footer instead.<br>`SPACE ACTIVE` means the next shortcut is armed. After Leader timeout, `Space again` keeps the hints readable until the next key; press Space to rearm. | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `Space+1/2/3` | Convert current block to H1/H2/H3 | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `Space+0` | Convert current block to a paragraph | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `Space+u/n/x` | Toggle current block between unordered, numbered, or Checkbox list<br>and a paragraph | Implemented (07); synthetic tests, practical terminal acceptance pending |
| `Space+q` | Toggle the applicable local Quote layer; no implicit expansion to unselected blocks | Implemented (07); synthetic tests, practical terminal acceptance pending |
| `Space+c` | Create Code block | Implemented (08); synthetic editor/config tests, real terminal acceptance pending |
| `Space+o` | Toggle Projects Pane; never a Description formatting command | Agreed |
| `Space+l` inside an existing link | Prompt with its current URL; link text remains ordinary Description text | Implemented (09); synthetic editor/save tests, practical terminal acceptance pending |
| `Space+y` inside a link | Copy its destination URL to the system clipboard without opening it | Implemented (09); synthetic editor/save tests, practical terminal acceptance pending |

## Description Insert Mode

| Binding | Action | Status |
| --- | --- | --- |
| Exactly three or four directly typed backticks in an empty paragraph, then `Enter` | Confirm and consume the trigger, create a structured Code block with protected triple-backtick fences, and enter its first empty Code line. Before Enter it remains ordinary text; paste, escaped markers, five backticks or additional characters do not trigger conversion. | Implemented (08); synthetic editor/config tests, real terminal acceptance pending |
| Single-backtick Inline-Code editor syntax | Deliberately typed paired backticks create Native Code; escaped inner backticks are literal. Imported/pasted syntax stays literal; removal of Code formatting preserves text. | Implemented (09); synthetic editor/save tests, practical terminal acceptance pending |
| Typed block-start `# `/`## `/`### `; `- `/`+ `/`* `/`1. ` | Create heading / list entry after the required space; escape the first marker to keep it literal | Implemented (09); synthetic editor/save tests, practical terminal acceptance pending |
| Typed dash-only line of at least three dashes | Create a horizontal rule | Implemented (09); synthetic editor/save tests, practical terminal acceptance pending |
| Paste | Insert literal text, never trigger structural conversion; prose line breaks create paragraphs | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `Enter` | Split the current paragraph into two paragraphs; inside a Code block insert a literal source line; inside a Block quote create a sibling quoted paragraph, or leave the quote from an empty quoted paragraph | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `Shift+Enter` | Insert a hard line break inside the paragraph | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `Enter` at end of a list entry | Add an entry of the same list type | Implemented (07); synthetic tests, practical terminal acceptance pending |
| `Enter` on an empty list entry | Leave the list and create a paragraph | Implemented (07); synthetic tests, practical terminal acceptance pending |
| Code Insert Mode: `Tab` / `Ctrl+V`, then `Tab` | Insert Spaces to the next tabstop / explicitly insert a literal Tab. Paste preserves Code Tabs and line endings. Ctrl+Q alias by default; tui.literal_tab_alias selects ctrl+b or disables it. Dispatch tested; real terminal checks pending. | Implemented (08); synthetic editor/config tests, real terminal acceptance pending |
| `i` on an imported empty list fill target | Enter a nonpersisted input target; actual text creates the first entry. `i` then `Esc` leaves source unchanged. | Implemented (07); synthetic tests, practical terminal acceptance pending |

## Description Visual Mode

| Binding | Action | Status |
| --- | --- | --- |
| `v` | Start characterwise Visual Mode | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `V` (Shift+V) | Start logical-block Visual Mode; footer label `VISUAL BLOCKS`. Normal Visual motions extend the selection; Space opens the structure palette without replacing those motions. | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `V` inside Code | Select source lines in `VISUAL CODE LINES`; clamp at Code boundary with hint, never silently expand to whole blocks. `Esc` cancels. | Implemented (06); synthetic tests, practical terminal acceptance pending |
| Normal-Mode motions | Extend the active Visual selection | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `o` | Swap the active end of the Visual selection | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `Esc` | Cancel Visual Mode | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `viw` / `vaw` | Select useful inner word / word with surrounding whitespace; delimiter-start selects nearest inner word, empty expression gives hint | Implemented (06); synthetic tests, practical terminal acceptance pending |
| `Ctrl+V` | Rectangular selection | Deferred |
| Visual `v` or `V`: `Space+b/i/u/s` | Toggle bold/italic/underline/strikethrough uniformly: apply throughout if any selected Native text lacks the mark; otherwise remove throughout. Explicit Bold-off removes all selected b/strong layers, including double/triple strong; Italic/Underline/Strike-off likewise removes all selected layers of its own family; preserve other marks and selection until Esc. Ignore and preserve Opaque subtrees. | Implemented (06); synthetic tests, practical terminal acceptance pending |
| Characterwise `v`: `Space+l` | Prompt for a URL and apply it to the selection; a blank URL removes the link | Implemented (09); synthetic editor/save tests, practical terminal acceptance pending |
| Visual `v` or `V`: `Space+0` | Remove all inline formatting, including Links, while preserving text and blocks | Implemented (06); synthetic tests, practical terminal acceptance pending |
| Blockwise `V`: `Space+q/w` | Remove / add one quote layer on selected blocks; wrap selected siblings in one shared container, remove locally, and reject incompatible reduction atomically. Both retained Quote fragments keep known shared cite/data-comment-id references without extra confirmation for repeating them. Actual metadata loss on removal still confirms; unique/unknown attributes require lossless allocation. | Implemented (07); synthetic tests, practical terminal acceptance pending |
| Blockwise `V`: `Space+k/l` | Decrease / increase list depth on selected entries, preserving subtrees, list kinds and Checkbox IDs/state. Normalize parent/child overlap; validate one sibling range atomically. If no unselected preceding sibling can receive an indent, reject the whole action. | Implemented (07); synthetic tests, practical terminal acceptance pending |
| Visual `v` or `V`: `Space+x` outside Code | Convert every touched logical paragraph/list entry into a Checkbox; partial character selections include the entire touched paragraph. Existing Checkboxes retain state and identity. One Undo restores the whole conversion. Wrapped screen rows do not create separate Checkboxes. | Implemented; focused model/TUI/save tests |
| Blockwise `V` inside Code: `Space+x/c` | Remove up to one indent step from leading whitespace / prepend the Space step. Locally split an affected Tab when needed; preserve empty lines and all other Code contents. Step two/four Spaces, default four; tui.indent_step independent of tui.tabstop. | Implemented (08); synthetic editor/config tests, real terminal acceptance pending |
| `d` | Delete exactly the selected text; after `V`, delete complete selected logical blocks including inline Opaque content. Confirm the whole operation when Opaque content is included; Undo restores the whole deletion. Wrapped screen rows are not separate blocks. | Implemented (06); synthetic tests, practical terminal acceptance pending |


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

In the Ex-command prompt, `Tab` completes a unique command or the longest shared
prefix of multiple candidates and displays those candidates. For example,
`:desc` becomes `:description` with `descriptionclear`, `descriptionrecover`
and `descriptionreload` still listed. Repeated `Tab` retains the list until
further typing narrows the matches.

Command and Search input retain their prompt and cursor-side input tail, with
an ellipsis for omitted leading input. Completion feedback yields before input
is clipped. An active error owns the footer budget and shows only `Error:` plus
the useful error tail.

## Notes for future binding proposals

- A **Pane** is focusable; a **View** is a Task representation inside the Tasks Pane. See [`../CONTEXT.md`](../CONTEXT.md).
- A Checkbox list is embedded rich text in a Description; it is distinct from the Task Done state.
- Preserve documented Vim Visual-Mode motions and operators. The rejected direct `b`, `i`, `u`, `s`, and `l` proposal and sources are recorded in [the binding review](../.scratch/description-rendering/research-vim-visual-bindings.md).
- The Space formatting palette's framed, persistent hints and compact-height replacement threshold are implemented; practical validation remains outstanding in [the rich-text contract](../.scratch/description-rendering/spec.md).
- Editable delimiters count as ordinary Vim characters, including in `5l`. Protected quote prefixes, Code fences, and emphasis-layer separators are never cursor targets, copied text, or independently editable content.
- Opaque subtrees require discernible atomic boundaries with configurable bracket style and highlighting color; final styling awaits prototype review. Wrapped continuation rows retain a non-editable arrow, distinct from hard-break markers.

Native Draft (06): `y/p` uses a local safe-visible-text register; `:descriptionrecover` / `:descriptionclear` create confirmed unsaved recovery drafts for Blocked HTML. `:descriptionreload` explicitly reloads current persisted Task data and discards the complete Task draft. `:e!` still restores the opened/successfully saved Task baseline. List/Quote structure (07), Code (08) and inline syntax/URL actions (09) are implemented.

Inline-format selections overlapping Code reject atomically; Code-line indent/outdent is implemented (08). Formatting across selected Opaque atoms preserves them and reports that they were skipped.

Normal `x` on a Description Checkbox toggles its checked state; Task Done is independent. `dd` and blockwise `V` deletion operate on complete entry subtrees, including children. Empty imported lists have nonpersisted fill targets: i/Esc and empty Enter create no entry; actual text fills an entry, deleting the final entry removes its container.

### Native Description inline syntax and Links (09)

Direct typing interprets `**bold**`, `*italic*`, `***both***`, `__underline__`, `~~strike~~` and paired single-backtick Inline-Code. A backslash masks the next agreed syntax character; imported/Paste/local-register text stays literal. At block start, `# `, `## `, `### `, `- `, `+ `, `* ` and `1. ` require the following Space. Three or more typed dashes form a rule; x/Undo retain the local dash count, Save emits canonical `<hr>`.

`Space+l` on a characterwise selection or existing Link opens a separate URL prompt: Enter applies, Esc cancels, empty URL unlinks. Newly entered destinations require an absolute `://` URL with http, https, ftp, git, obsidian, notion or message; invalid input stays in the prompt. Safe imported foreign schemes remain preserved and replaceable. `Space+y` copies a Link destination to the system clipboard and reports the actual result. Typing at Link edges or outside the visible Inline-Code backticks stays outside; text targets between those backticks edit Code content. Inside a Link, typing retains the wrapper. Empty imported Link labels show the current URL without materializing it until a text edit.
