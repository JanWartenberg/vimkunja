# Rich-Text Description Contract (Draft)

> This document defines Vimkunja's durable contract for reading, displaying, editing, and saving Vikunja Task Descriptions. It deliberately outlives the originating ticket: the contract may later inform terminal rich-text work outside Vimkunja, but it is currently specific to Vimkunja and does not define a new markup language.

**Related tickets:** M2 reader: [01 — Render Vikunja HTML descriptions as readable terminal text (archived)](../../.archive/2026-08-05/description-rendering/issues/01-render-vikunja-html-descriptions.md); M3 structured reader/editor: [02 — Specify native Rich-Text Description rendering and editing](issues/02-specify-native-rich-text-description-rendering-and-editing.md)

**Companion artifacts:** [Keybinding Contract](../../docs/keybindings.md), [TUI/Rich-text glossary](../../CONTEXT.md), [Vim Visual-Mode binding review](research-vim-visual-bindings.md), [Task Detail Done toggle](../task-detail-done-toggle/issues/01-toggle-task-done-from-detail-pane.md), [deferred Checkbox-list summary](../checkbox-list-progress-summary/issues/01-show-checkbox-list-progress-summary.md), and [deferred independent Link label/URL editor](../link-label-and-url-editor/issues/01-evaluate-independent-link-label-url-editing.md).

## Purpose

Vikunja stores rich-text Task Descriptions as HTML. Vimkunja must provide a keyboard-first terminal editing experience without exposing HTML source for normal editing, while continuing to store Vikunja-compatible HTML and without changing the database schema.

## Terms

- **Source HTML:** HTML read from and written to Vikunja's existing `tasks.description` field.
- **Description document:** Vimkunja's structured interpretation of Source HTML for rendering and editing.
- **Terminal projection:** The keyboard-navigable, terminal-appropriate rendering of a Description document. It need not be visually identical to the web application.
- **Description block:** A logical block in a Description document, such as a paragraph, heading, list item, code block, quote, or table.
- **Checkbox list:** A checklist embedded in a Description, represented by Vikunja/Tiptap `data-type="taskList"` HTML. This is distinct from Vimkunja's Tasks Pane and its Task records.
- **Supported node:** A document structure Vimkunja can render and edit natively.
- **Opaque node:** A recognized or unrecognized existing HTML structure which Vimkunja may render but cannot edit natively.

## Agreed principles

1. Source HTML remains the persistence format. Vimkunja does not store an alternative format or change Vikunja's schema.
2. Normal editing must not require the user to read or edit HTML source.
3. M2's completed reader-only phase uses a deliberately lossy HTML-to-plain-text projection for display. It never writes Source HTML. Native rendering, editing, and saving use a structured document model rather than lossy conversion.
4. Existing opaque nodes must not be destroyed by an otherwise valid edit. They are visible but initially non-editable.
5. Unsupported terminal presentation degrades to readable content; it must not display literal markup merely because it is unsupported. Unknown parseable HTML renders its text contents recursively without special formatting; structurally empty unsupported content displays a neutral unsupported-content hint.
6. Native terminal formatting is semantic where possible. Terminal capabilities may affect visual fallback, but not the document's HTML meaning.
7. Paragraphs and other Description blocks are parsed from Source HTML even when that HTML is stored as one long string. The Terminal projection separates logical blocks for navigation; it does not navigate HTML source text.
8. Saving serializes the Description document back to compact HTML. It may be one physical string without formatting newlines, while preserving the document's semantics and opaque structures.
9. In a Description, `j` and `k` jump between navigable logical blocks: paragraphs, headings, and individual list or Checkbox-list entries. `gj` and `gk` move between visual terminal rows created by soft-wrapping a navigable text block.
10. Checkbox lists are part of the first native editable subset. Vimkunja can add and remove their entries and toggle their checked state. This changes only the Description document's `data-checked` state, never the enclosing Task record's Done status. Historical Checkbox-list entries without `data-task-id` are accepted; Vimkunja assigns unique IDs when it next serializes a permissible edit, matching Vikunja compatibility behavior.
11. Headings `h1` through `h3` are part of the first native editable subset.
12. Bold, italic, underline, and strikethrough are part of the first native editable subset. Their terminal projections may use capability-dependent visual fallbacks without changing their HTML meaning.
13. Ordered and unordered lists are part of the first native editable subset. Nested lists are rendered and preserved in V1. Their existing text and Checkbox-list state remain editable, but native indentation, outdent, and other structural nesting edits are deferred.
14. Code blocks are part of the first native editable subset. All Vimkunja text is monospaced, so their Terminal projection must distinguish them through non-font treatment such as indentation, a border, background, or another readable convention. Multi-line code blocks preserve their source lines without soft wrapping; overwide code scrolls horizontally with `zh`/`zl`.
15. Block quotes are part of the first native editable subset. Their Terminal projection uses a readable non-font distinction such as a left `│` marker and indentation.
16. Horizontal rules are part of the first native editable subset. `Space+r` inserts a rule after the current block without changing that block's text. A rule is an atomic, selectable Description block; its deletion commands are defined separately.
17. Links are part of the first native editable subset. Their terminal editing flow operates on visible link text and a separate URL, never on raw `<a>` markup. `Space+l` in Visual Mode applies a prompted URL to the selected text; in Description Normal Mode with a cursor inside an existing link, it prompts with the existing URL. Link text remains ordinary editable Description text. An empty URL removes the link; a nonempty URL is serialized using Vikunja's `target="_blank"` link behavior.
18. An opaque node's HTML structure and attributes must survive saves that change other Description content. Serialization may normalize insignificant whitespace and attribute ordering, provided Vikunja renders the node with the same meaning. If HTML cannot be safely parsed or its structure cannot be preserved, Vimkunja opens the Description read-only, displays a clear error hint, and blocks saving rather than risk destructive writes.
19. Tables are opaque in V1: Vimkunja renders them as readable terminal tables but does not edit their cells, rows, columns, or attributes. A table wider than the Task Detail Pane scrolls horizontally with `zh` and `zl` and exposes a visible overflow indicator. Tables are `j`/`k`-focusable for reading and scrolling, but reject deletion, formatting, and other modification commands with a clear hint.
20. Description editing includes Vim-style Visual Mode. `v` begins a selection, which is the basis for applying inline formatting without exposing pseudo-Markdown or HTML. A characterwise selection may span multiple editable Description blocks; inline formatting applies to every selected text segment. Opaque nodes are selection boundaries. Vimkunja never expands a user selection implicitly to word or block boundaries; users select complete words or structures explicitly through motions such as `w`/`b` and future text objects.
21. **Provisional, pending prototype review:** In Description Visual Mode, Vimkunja applies bold, italic, underline, strikethrough, and links through `Space+b`, `Space+i`, `Space+u`, `Space+s`, and `Space+l` respectively. Each formatting action toggles exactly the selected text segments: already formatted segments lose that mark and unformatted segments gain it. This preserves Vim's direct Visual-Mode motions and operators while avoiding raw markup. `Space+0` removes all inline marks, including Links, while preserving text and all block structure.
22. `V` (Shift+V) begins blockwise Visual Mode, selecting whole logical Description blocks such as paragraphs. `viw` and `vaw` select the inner word and word including surrounding whitespace. Further Visual text objects are deferred beyond V1. `Ctrl+V` rectangular selection is explicitly deferred beyond V1.
23. `Enter` in Insert Mode splits the current paragraph at the cursor into two logical paragraphs, serialized as separate `<p>` blocks. `Shift+Enter` inserts a hard line break (`<br>`) within the current paragraph. `Backspace` at a paragraph's start merges it with the preceding compatible block; `Delete` at its end merges it with the following compatible block. A multi-line Plain-Text paste outside a Code block creates paragraphs at line breaks; inside a Code block it preserves source lines. Pasted text that resembles HTML remains literal text; it never creates document markup.
24. In Description Normal Mode, block-format commands act on the current Description block without requiring a selection. Visual Mode is reserved for inline formatting.
25. **Provisional, pending prototype review:** `Space+1`, `Space+2`, and `Space+3` convert the current block to H1, H2, and H3; `Space+0` converts it to a paragraph.
26. `Enter` at the end of a list or Checkbox-list entry creates a new entry of the same kind. `Enter` on an empty entry exits that list and creates a following paragraph.
27. In Description Normal Mode, `x` toggles the current Checkbox-list entry's checked state only. It never changes the enclosing Task record's Done state.
28. **Provisional, pending prototype review:** In Description Normal Mode, `Space` opens a context-specific formatting palette in a secondary footer. The displayed second-key shortcuts execute the corresponding format action directly, keeping common operations to two keystrokes. It uses the existing Leader timeout and closes on the second key, `Esc`, or timeout. `Space+o` remains the global Projects-Pane toggle, including while editing a Description, so the editor can gain more terminal space. Block conversions use `Space+1/2/3/0` (H1/H2/H3/paragraph), `Space+u/n/x` (unordered/numbered/Checkbox list), and `Space+c/q/r` (code block/quote/rule). An overlay above the Task Detail Pane is a later prototype alternative.
29. The Description editor supports local `u` Undo and `Ctrl+R` Redo for changes to its current, unsaved draft. This is distinct from undoing persisted Task actions.
30. Saving detects whether the Description changed externally since the editor opened. A conflict rejects the save instead of silently overwriting newer content. The local Draft remains open; the user explicitly chooses to return to it or perform a hard reload of current database content. Vimkunja performs no automatic merge.
31. Reapplying an applicable block-format command to a block of the same type toggles that format off and converts the block to a paragraph. Commands without a meaningful conversion or toggle define their own behavior.
32. In Description Normal Mode, `dd` deletes the current logical Description block, including an atomic horizontal rule. A logical block is the rich-text equivalent of a Vim line for this command. In Visual Mode, `d` deletes exactly the selected text or, for a `V` selection, the selected logical blocks.
33. In Description Normal Mode, `o` and `O` insert a new empty paragraph after or before the current logical block and enter Insert Mode.
34. In Description Normal Mode, `i`, `a`, `I`, and `A` enter Insert Mode at the cursor, after the cursor, and at the start or end of the current logical line respectively. For normal text, the logical line is the complete paragraph or List entry; within a Code block, it is the current source line. `0`, `^` or `_`, and `$` move to the start, first nonblank position, and end of that line without entering Insert Mode. `w` and `b` traverse rendered semantic text across Inline formatting and Links, and stop at block boundaries. `e` and `ge` move to word ends forward and backward. `gg` and `G` jump to the start and end of the entire structured Description document. `Ctrl+D` and `Ctrl+U` move the cursor by approximately one visible Task Detail Pane height. Visual Mode extends the selection with these motions; `o` swaps its active end and `Esc` cancels it.

## First native editing set

V1 natively renders and edits:

- paragraphs and headings (`h1`–`h3`)
- bold, italic, underline, and strikethrough text
- code blocks and block quotes
- unordered and ordered lists
- Checkbox lists
- links
- horizontal rules

Tables are initially opaque and read-only, with a readable terminal-table projection. Future or unknown Tiptap extensions are opaque until their rendering and preservation behavior is specified.

## Prototype review and remaining decisions

- Detailed Terminal styles and narrow-width fallbacks are deferred to the rich-text prototype.
- Validate the provisional Space-led formatting palette, its secondary-footer presentation, and the provisional formatting bindings against the prototype.
- Choose the HTML parser/serializer implementation that fulfills the preservation, normalization, and read-only safety rules above; no schema change is permitted.
- Specify URL validation and feedback in the Link prompt.
- Read-only Checkbox-list progress summaries are implemented under [checkbox-list-progress-summary](../checkbox-list-progress-summary/issues/01-show-checkbox-list-progress-summary.md): one `Checklist progress: n/m` line after each outer Checkbox-list block, aggregating nested entries, with no summary for zero entries. Summaries are display-only and never serialized. Plain-text checkbox blocks support `[ ]`, `[]`, and `[x]` during the transitional reader phase; Insert Mode hides summaries while markers are edited, and normal rendering recomputes them. Native editing must recompute summaries from the draft and remove empty list structures after deletion of the last entry; summaries are not editable or cursor targets.

## Related decisions outside Description scope

- The Task Detail Pane's Done field supersedes the archived read-only rule: it sits after Workflow Status and before Description, and `x` toggles the enclosing Task's Done state. The implementation ticket is linked above.
- `docs/keybindings.md` is the durable, manually maintained exchange contract for all contexts. `internal/tui/help.go` remains restricted to implemented bindings until the design stabilizes.
