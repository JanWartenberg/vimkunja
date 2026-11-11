# Topic-based in-application help

## Acceptance and scope

Jan accepted the revised help design and documentation step on 2026-11-10. Jan explicitly requested implementation using `implement-and-verify` and authorized the commit on 2026-11-11. The prototype establishes structure and presentation, not a complete binding inventory. The implementation defaults below fill routine details without requiring another design round.

Replace the flat help reference with a fixed overview and short topic pages in the existing application. Use one help pane without simulated Task/Project content. Retain application pane navigation. No database, dependency, editor-semantics, tutorial-runtime, or general application-history changes.

## Pages and presentation

- Overview: Navigation & Views; Tasks & Detail; Shared text editing; Rich text; Search & filters; Commands & quitting.
- Rich text has an index: Editing basics; Formatting; Links; Lists & Code; Undo & recovery.
- Entries show action and keys, followed by short, always-visible explanations with mode/context where required. No expandable entries, search, contextual entry, or topic arguments in this version.
- Shared Title/Comment motions and clipboard rules remain under Shared text editing. Refer to Description-specific behavior without implying identical semantics. Distinguish persisted Task undo from Description draft undo.
- Use consistent indentation, natural wrapping, and recognizable continuation lines. Long keys must wrap without shifting every explanation. Keep breadcrumbs and concise navigation hints; compress chrome at low heights. Selection and hierarchy remain understandable without color.
- Use actual help-pane dimensions. Check 120×40 and 80×24 terminals and narrow/short help areas, including 44×24 and 60×12. Keep every entry reachable by scrolling and account for Unicode display width.

## Entry, navigation, and lifetime

- `:h` and `:help` always focus the overview, including when help is already open. Explicit entry starts a fresh help navigation session.
- `j/k` and Up/Down select topics in indexes or scroll reference pages. Enter opens a selected topic. Preserve existing help navigation: Ctrl+D/U, PageDown/PageUp, gg/G, Home/End; on indexes, page/endpoint navigation moves selection and keeps it visible.
- `Ctrl+O` goes back; `Ctrl+I` (including terminal Tab) goes forward. Restore page, selection, and reading position; clamp after resizing. Boundaries do nothing. A new visit after going back drops forward history. Keep history session-local and bounded to 100 locations.
- `-` and Backspace open the structural parent, restoring its previous selection/position; at the overview they do nothing. Parent navigation is a new history visit. It must not trigger the application Overview shortcut.
- History keys act only when help has focus and no command prompt is active. Tab retains Ex completion. Help history must not consume Global Search return outside help.
- `:q` closes focused help and returns to the prior working pane. Preserve selection, scroll position, Task/Comment drafts, and local editor history. Opening help must not save, discard, or reload drafts. If the origin no longer exists, return safely to Tasks. Repeated `:h` while help is open must not overwrite the origin.
- Retain application quit/save/discard contracts. Closing help does not quit the application or close a dirty editor. A fresh help session starts at the overview.

## Content and implementation responsibilities

Inventory current `internal/tui/help.go`, implemented command/key handlers, and `docs/keybindings.md` before replacing the reference. Place every currently documented action on an appropriate topic, or record why a stale/incorrect entry is corrected. Do not copy representative prototype content as the complete reference. Label deferred Views as unavailable.

Use small structured page/entry data and shared rendering/navigation within the TUI. Preserve generic error-log behavior. Avoid a generator or generic documentation framework. Help owns the concise in-app reference; `docs/keybindings.md` owns binding status and the durable contract; README owns the introduction; tutorials own guided practice. Update affected documentation and statuses with implementation.

Current integration hazards: `helpCommand` hides Detail and clears the info pane; `quitCommand` always focuses Tasks when closing help. Replace those assumptions with explicit origin preservation. Check Comment overlays, pane switching, repeated help entry, and error-log transitions during integration.

## Acceptance criteria

- [x] Overview and Rich text index expose the specified pages. Shared basics remain discoverable outside Rich text. Content inventory covers the current build, aliases, and mode restrictions.
- [x] Lookup paths work: Tasks & Detail → save Task; Navigation & Views → move Kanban card; Rich text → Links → edit Link; Rich text → Lists & Code → delete Code line; Rich text → Undo & recovery → blocked HTML. Closing help is documented in navigation hints.
- [x] Keyboard tests cover selection, activation, history boundaries/restoration, forward-history truncation, parent navigation, repeated `:h`, and Ex Tab completion. History stays local to help.
- [x] Layout checks at the specified sizes verify readable wrapping, visible selection/orientation, and access to the final entry. Include long keys and wide Unicode; check visible layout rather than only text presence.
- [x] Integration regressions open and close help from Tasks, Task Detail, Description navigation, and Comment editing where available. Assert unchanged dirty drafts, local undo history, origin focus, selections, and scroll positions. Cover pane switching and a missing-origin fallback.
- [x] Error log, Global Search return, pane focus, and application save/quit retain their contracts. Help performs no Store writes.
- [x] Update affected README/keybinding documentation. Run affected TUI tests; before a later implementation commit, run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`.

## Content inventory and corrections

The former flat reference is mapped in full: navigation/Projects/Overview/Views/Kanban → Navigation & Views; Tasks/Detail/field shortcuts/Comments → Tasks & Detail; clipboard and Title/Comment motions → Shared text editing; Description navigation/editing → Editing basics; rich-text actions → Formatting, Links, Lists & Code, or Undo & recovery; Search → Search & filters; Ex commands → Commands & quitting. The former Help Pane section is replaced by the new navigation contract and footer hints.

Corrected stale statements: Ctrl+W timeout is configurable; the rich-text palette remains readable after timeout; Description viewport keys are not Title/Comment motions; Comment composer save/discard behavior is described independently; references to formatting “above” now name its topic. Added missing Code creation/exit/Tab shortcuts and the URL-copy alias without replacing existing bindings.

Help covers the existing bottom pane rather than destroying its state. While it covers an unsaved Task or Comment, context-changing commands and actions are rejected; close help to resume editing. Explicit application save/discard/quit commands retain their contracts. Ordinary layout tests also cover top-pane width containment so narrow help windows do not overflow the terminal.

Verification details and limitations are recorded in the [completed ticket](issues/01-rethink-help-structure-and-richtext-reference.md#implementation-and-verification--2026-11-10).
