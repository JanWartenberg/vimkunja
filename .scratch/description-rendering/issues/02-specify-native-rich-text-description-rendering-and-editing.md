# 02 — Specify native Rich-Text Description rendering and editing

**Phase:** M3 — structured reader and editor

**What to decide:** Define the complete M3 scope for a native, keyboard-first terminal projection and editor for Vikunja HTML Task Descriptions.

**Implementation prerequisites:** M2's read-only HTML-to-plain-text reader is complete; see its [archived ticket](../../../.archive/2026-08-05/description-rendering/issues/01-render-vikunja-html-descriptions.md). The decisions in this ticket govern the M3 implementation; the existing [Rich-Text Description Contract](../spec.md) is the starting point and must be updated only when a decision here changes it.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

**Current handoff:** The compatibility-corpus plan and initial risk-based fixture inventory are established. Basic blocks, inline marks, lists/Checkbox lists, Code blocks, Block quotes, hard breaks, Links, opaque structures, exact opaque-subtree preservation, unknown-node fallback, and malformed-HTML recovery are specified. Serialization now has a hybrid Native/Opaque policy, semantic-only Native canonicalization, compact canonical escaping, UUID v4 assignment for missing Checkbox IDs, VOA-synchronized Link targets and safe attribute preservation, structured semantic equivalence, standards-aligned mark semantics, and empty-structure rules. No implementation has started. Serialization is settled: duplicate Checkbox IDs use a confirmed preserve-first/regenerate-later repair flow; valid top-level inline fragments become paragraphs; ordinary prose whitespace follows HTML collapsing; duplicate Native attributes and document-level markup block the Description; styled elements and HTML comments become Opaque atoms. Fixture acceptance now requires a traceable corpus of focused VOA-generated, synthetic, and regression snippets with manually reviewed golden expectations. No implementation has started. Questions 1–23 have now been discussed. Acceptance evidence policy and editor grammar are agreed. Cursor anchoring, disclosure/reflow, scrolling, narrow layouts, and the Space formatting palette/bindings are provisionally accepted pending practical examples. Opaque subtrees are single word-motion units with configurable visible atomic boundaries. Mark toggling is uniform across the selection, replacing segment-by-segment inversion. Technical direction is accepted for evaluation, not final selection. Next: create concrete fixtures and traceability with reviewed expectations, validate provisional interactions in practical examples, then prove the parser/serializer direction. No native implementation is authorized.

**Scope boundary:** This is phase 2 of the Description plan. It replaces neither Vikunja's HTML persistence format nor its SQLite schema. It aims to natively render and edit every HTML structure that is useful and feasible in a terminal. Content that is not suitably editable must remain as readable as practical and must be preserved without destructive rewriting.

## Questions for dedicated grilling

The contract already establishes Source HTML as the persistence format, a structured document model, safe preservation of Opaque subtrees, and an initial native set of headings, inline marks, lists, Checkbox lists, code blocks, quotes, rules, and links. Resolve these remaining product decisions one at a time:

1. Which Vikunja/Tiptap structures beyond the contract's initial native set are useful enough to render and edit natively in M3, especially images, attachments, mentions, embeds, and complex tables?
2. For every unsupported structure, which support tier applies: natively editable, readable but opaque/read-only, neutral fallback while preserved, or intentionally not rendered?
3. What exact preservation guarantee is required when an edit changes surrounding content containing Opaque subtrees, attributes, or unknown extensions?
4. How should malformed, partially parseable, or unsafe HTML be presented, and when must the entire Description become read-only to prevent a destructive save?
5. What terminal projection conventions make headings, emphasis, nested lists, quotes, code, tables, links, and overflow readable at narrow widths without conflating visual layout with document meaning?
6. Does prototype testing validate the proposed `Space`-led formatting palette and Visual-Mode bindings, or should their interaction model change before implementation?
7. What are the final link rules: display convention, URL validation, editing an existing link, empty-URL behavior, and feedback for invalid input?
8. Which serialization normalizations are acceptable while remaining Vikunja-compatible, including whitespace, attribute order, Checkbox-list IDs, and link attributes?
9. What compatibility corpus and acceptance evidence are required to establish that existing Vikunja-created HTML is rendered usefully and survives unrelated Vimkunja edits?

## Compatibility corpus and test harness

Before implementation, assemble a representative corpus of valid Vikunja/Tiptap HTML and document the expected behavior for each fixture. The corpus should be created through both greenfield example brainstorming and the grilling decisions above. It must cover combinations that are especially vulnerable to destructive parsing, editing, or serialization, including:

- nested ordered, unordered, and Checkbox lists
- headings and paragraphs combined with inline marks and links
- adjacent or overlapping marks, formatted link text, and hard breaks
- Unicode, entities, significant whitespace, empty blocks, and HTML-looking code text
- quotes containing lists or formatted content, code blocks, and horizontal rules
- tables mixed with editable content
- unknown or opaque subtrees, unusual attributes, and historical Checkbox entries without IDs
- malformed, partially parseable, unsafe, and structurally empty content

For every fixture, specify the expected terminal projection, editable/opaque/read-only status, and preservation guarantee. The test harness should establish at least these invariants:

- parsing and rendering do not panic or accidentally expose raw markup
- supported content round-trips to equivalent Vikunja-compatible HTML semantics
- Opaque subtrees, attributes, and unknown content survive unrelated edits
- unsafe or ambiguous content becomes read-only rather than being destructively rewritten
- list, Checkbox-list, link, formatting, and block boundaries retain their meaning
- Description edits never change the enclosing Task's Done state
- external-save conflicts are rejected rather than silently overwritten

The corpus is a specification deliverable and must exist before native editing implementation begins. It should support fixture-based regression tests, semantic round-trip tests, and property/fuzz tests for parser and serializer boundaries.

## Working plan

- [x] Establish the corpus plan and initial risk-based fixture inventory.
- [ ] Grill the inventory and decide the expected projection, editability, preservation guarantee, and safety behavior for every fixture.
- [ ] Add concrete HTML fixtures and expected terminal projections to the repository.
- [ ] Define semantic round-trip and opaque-content preservation assertions for the fixtures.
- [ ] Add parser/serializer boundary property or fuzz cases.
- [ ] Split implementation tickets only after the corpus and preservation contract are accepted.

### Initial fixture inventory

The first corpus pass should include at least these cases. IDs are stable names for later fixture files and test cases.

#### Editable structure and nesting

- `basic-blocks`: paragraphs, empty paragraphs, headings `h1`–`h3`, and horizontal rules.
- `inline-marks`: bold, italic, underline, and strikethrough individually and in combinations.
- `inline-boundaries`: adjacent marks, nested marks, marks crossing text boundaries, and Unicode text.
- `hard-breaks`: paragraphs containing `<br>` and whitespace around breaks.
- `nested-lists`: ordered and unordered lists with nested lists and mixed list depths.
- `checkbox-lists`: checked and unchecked entries, nested entries, IDs, and historical entries without IDs.
- `list-editing`: empty list entries, list splitting, list exit, and deletion at list boundaries.
- `code-and-quotes`: multiline code blocks, HTML-looking code text, and quotes containing formatted content or lists.
- `links`: plain and formatted link text, adjacent links, empty URLs, URL attributes, and links beside marks.

#### Opaque and preservation-sensitive content

- `table-mixed-content`: readable tables before, between, and after editable blocks.
- `opaque-node`: known unsupported node with nested text and non-default attributes.
- `unknown-node`: unknown Tiptap/HTML element with attributes and nested known content.
- `opaque-edit-neighbors`: editable text changed on both sides of an Opaque subtree.
- `opaque-nesting`: opaque content nested inside or around otherwise editable structures.
- `attribute-preservation`: unusual, ordering-sensitive, or data attributes on opaque content.

#### Parsing and safety boundaries

- `entities-and-whitespace`: entities, non-breaking spaces, significant whitespace, and Unicode edge cases.
- `empty-structures`: empty links, empty formatting nodes, empty lists, empty tables, and empty unsupported nodes.
- `malformed-html`: mismatched tags, incomplete elements, invalid nesting, and partially parseable content.
- `unsafe-html`: scripts, event attributes, unsafe URLs, and other content requiring sanitization or read-only fallback.
- `serialization-normalization`: equivalent source forms differing only in whitespace, attribute order, or Checkbox IDs.

The inventory is intentionally a starting set rather than an assertion that every listed behavior has already been decided. The grilling step must attach expected behavior and acceptance evidence to each case before implementation begins.

## Grilling decision log — first pass

### Projection mantra

> Friendly projection by default, source disclosure near the cursor.

The current logical block reveals its editable source syntax in Normal and Insert modes. Cursor motions operate over the disclosed source characters. Away from the cursor, source syntax is concealed and the user-friendly interpretation is rendered. This applies uniformly to headings, inline marks, and horizontal rules.

### Basic blocks

- Paragraphs, headings `h1`–`h3`, and horizontal rules are natively editable.
- Existing empty paragraphs are displayed and preserved.
- User-created blank lines persist as empty paragraphs (`<p></p>`).
- Editable heading markers map structurally: `#`, `##`, and `###` create `h1`, `h2`, and `h3`; removing all markers creates a paragraph.
- Dash-only lines with three or more dashes are horizontal rules. They render as `---`; ordinary character navigation applies. Non-canonical dash counts normalize to canonical `<hr>`/`---` on save. The rule is an atomic block: `dd` deletes it entirely, while `x` removes one dash at a time and turns it into ordinary paragraph text below three dashes. Visual selection editing may be added in a future iteration. No dedicated `Space+r` insertion shortcut is required.

### Inline marks

- Bold, italic, underline, and strikethrough use editable delimiters: `**`, `*`, `__`, and `~~`.
- Combined marks serialize in canonical nesting order: bold → italic → underline → strikethrough. `***text***` means one bold plus one italic layer (`<strong><em>text</em></strong>`), not repeated emphasis. Imported repeated emphasis uses nested delimiter pairs with visible protected, presentation-only separators.
- Unmatched, mismatched, or overlapping delimiters remain literal text; they are not repaired automatically.
- Backslash escaping creates literal delimiters. Escapes are removed when semantic HTML is saved, preserving the literal characters in Vikunja.
- Empty mark pairs are removed on save.
- Markup-looking text inside inline code and code blocks remains literal.
- Markdown-like delimiters are editor syntax only; persistence uses semantic Vikunja-compatible HTML. The Vimkunja markup dialect must have a help entry, durable documentation, and a dedicated section in the Vimkunja markup tutorial; this is documentation scope, not a compatibility concern.

### Lists and Checkbox lists

- Nested ordered and unordered lists use indentation and ordinary list markers in the friendly projection; the current block may reveal source syntax.
- Checkbox entries use Unicode `☐` and `☑` markers, with nested entries indented.
- Toggling a Checkbox changes only the Description entry and never the enclosing Task Done state.
- Historical Checkbox entries without IDs remain editable; an ID is assigned only when needed on save.
- Empty list entries remain editable and persist as empty `<li>` elements.

### Code blocks

- Code blocks use a theme-configurable background, defaulting to dark gray against the default black background. The background highlights the entire block while its syntax is disclosed.
- A thin `│` prefix remains visible as a non-color fallback; both prefix and background are presentation-only.
- Code lines never soft-wrap. Overwide lines scroll horizontally with `zh`/`zl`.
- When horizontal overflow exists, a one-row scrollbar appears at the bottom inside the Code block; it disappears when all content fits. The scrollbar is not editable code content.
- Code blocks do not expose HTML. Away from the cursor they use the friendly projection. When the cursor is anywhere inside a Code block, the entire block discloses canonical triple-backtick opening and closing fences in both Normal and Insert modes.
- Disclosed fences are protected structural markers: they are visible but are not cursor targets or directly editable.
- Typing three or four consecutive backticks in an otherwise empty paragraph creates the same structured Code block, consumes the trigger, discloses canonical triple-backtick fences, and places the cursor on an empty Code line between them. `Space+c` and a discoverable command may provide equivalent creation paths.
- Editing preserves every code character exactly, including indentation, tabs, blank lines, and a final trailing newline. Rendering and serialization do not normalize Code-block text. `Enter` inserts a literal Code line; paste preserves line breaks and whitespace.
- Code-block editing uses ordinary Description Insert and Normal modes, not a separate Code Block Mode. `Esc` only returns to Normal Mode.
- In Normal Mode, `o` and `O` create Code lines below and above the current source line. `dd` deletes the current source line; on an empty Code block, `dd` deletes the entire block.
- `{` and `}` move backward and forward by paragraph-motion units; Normal-mode `ö` and `ä` are German-keyboard aliases. Every navigable Description block is one paragraph-motion unit, and an entire Code block remains one unit regardless of its internal or empty lines. Counts apply. Within a Code block, `j` and `k` continue to move between source lines.

### Block quotes

- A Block quote is the HTML `<blockquote>` element for an extended quotation; it is distinct from an inline HTML `<q>`. `<q>` has no native M3 semantics and is preserved as unsupported inline content.
- A quote's first visual row uses the presentation-only prefix `│  `; its wrapped continuation rows use `│↪ `. ASCII capability fallbacks are `|  ` and `|> `. Nested quotes add a prefix at each depth. No quote-specific background or color is required.
- Quote prefixes are never editable or cursor targets, in either Normal or Insert mode. Quote text is edited directly; quotes never disclose pseudo-Markdown `>` syntax.
- A quote is a structural container for every native editable Description block: paragraphs, headings, lists and Checkbox lists, Code blocks, rules, and nested quotes. Its child blocks retain their normal semantics.
- `Space+q` wraps the current block in a quote; repeated use adds a nested quote layer. Reapplying it removes the current quote layer while retaining its child content. `Enter` creates a sibling child paragraph at the same quote depth; `Enter` on an empty quoted paragraph exits the quote and creates a following normal paragraph. At a quoted child paragraph's boundaries, `Backspace` and `Delete` merge only an adjacent compatible child at the same quote depth; they never implicitly move ordinary text across a quote boundary.
- Code blocks and quotes are symmetric structural blocks: each complete container is one `{`/`}` paragraph-motion unit; `j`/`k` navigate its internal logical lines (literal source lines for Code, child blocks for quotes); `dd` deletes the current internal unit; and `o`/`O` create an internal unit below/above (a Code line or same-depth quoted paragraph). Deleting the final internal unit removes the empty container.
- Imported empty quotes are preserved and render as one empty quoted editable line. `Space+q` on an empty paragraph creates one. Deleting its sole internal line removes it.
- Existing parseable `cite` and VOA `data-comment-id` attributes are preservation-only: Vimkunja carries them through unrelated edits but offers no M3 input or editing UI or Task-Description reply semantics. This follows the general rule to preserve existing content whenever safe, rather than creating surprising data loss.

### Opaque-subtree preservation

- The outermost unsupported ancestor is one atomic Opaque subtree. Descendants may contribute to its safe projection but are not independently focused, selected, edited, or validated.
- Unrelated edits preserve the subtree's original HTML byte-for-byte, including tag spelling, attributes, whitespace, children, and nesting. Native content around it remains editable.
- Selections may cross only complete opaque atoms. Formatting applies to selected native text and skips opaque atoms unchanged. Deletion and replacement of complete opaque atoms, including direct `x`/`dd` deletion on focus, require confirmation; cancellation changes nothing.
- Copy exposes only the safe visible projection, never hidden source data. Pasting it inserts new Native text nodes; it does not duplicate the opaque structure or its IDs. Undo restores confirmed removals byte-for-byte at their original position and nesting; Redo reapplies them without another confirmation.
- Structural changes to a native parent containing an opaque descendant use confirm-and-validate. They proceed only if serialize-and-reparse validation keeps the subtree byte-for-byte at its intended position.
- Every save containing opaque content performs one linear-time preservation round trip, not per-keystroke validation. It verifies retained opaque subtree bytes, identity, order, nesting, and parent position; failure rejects the save and keeps the draft open. Deliberately removed subtrees are excluded.
- Safe unknown attributes do not make a Native node opaque. Preserve their names and decoded values while permitting order, quoting, and equivalent entity normalization. Carry them through structural conversion when safe; otherwise reject the conversion rather than dropping them.
- Explicit conversion, unwrapping, and raw-HTML editing of opaque structures are deferred follow-up work.

## Decision status and remaining evidence

The policy interview is complete for questions 1–23. Provisional interactions still require practical validation. Do not reopen settled decisions unless a fixture or prototype exposes a contradiction.

### Immediate next chunk

- [x] **Code blocks:** source disclosure, creation, line preservation, editing, deletion, and paragraph-motion behavior are settled above.
- [x] **Block quotes:** projection, wrapping prefixes and fallbacks, non-editable markers, nested native content, creation/removal, entry/exit, internal navigation/deletion, empty quotes, and preservation attributes are settled.
- [x] **Hard breaks:** projection, navigation/deletion, whitespace, edge cases, and nested-content behavior are settled.
- [x] **Links:** display, editing, VOA-compatible extensible scheme policy, formatting, unknown-scheme preservation, and clipboard copy are settled; unsafe URLs follow the global safety rule.
- [x] **Opaque structures:** Images, generic attachment scope, Mentions, embedded media, Tables, unknown-node fallback, and safe-attribute preservation are settled.
- [x] **Malformed HTML:** fail-closed safety, read-only projection, and confirmed Clear/Recover-text draft flows are settled.
- [x] **Opaque preservation:** outermost atomic boundaries, byte-for-byte retention, cross-boundary operations, confirmed removal/replacement, safe copying, exact Undo, native-node attributes, structural validation, and save-time round trips are settled.
- [x] **Serialization:** settled and recorded in the contract: canonical semantic serialization for Native nodes, byte-exact Opaque preservation, compact canonical escaping, Checkbox-ID assignment and confirmed deduplication, canonical VOA Link targets with safe meaningful attribute preservation, structured semantic equivalence, standards-aligned mark semantics, meaningful empty-block preservation, root-fragment normalization, prose-whitespace collapsing, duplicate-attribute safety, and Opaque handling for styles and comments.

### Remaining fixture categories

- [x] **Hard breaks:** `<br>` renders as a real terminal row break with a subtle, non-editable `↵` end marker while its paragraph is active (`>` ASCII fallback), hidden when unfocused. It remains inside one paragraph: `j`/`k` skip blocks, `gj`/`gk` traverse its rows, left/right cross it, and adjacent `Backspace`/`Delete` remove it. Preserve adjacent imported whitespace in the model and serialization; newly entered content follows normal HTML-renderer-visible behavior. Leading, trailing, and consecutive breaks are preserved and permitted, including intentional blank rows. The same semantics apply inside headings, list/Checkbox-list entries, and Block quotes; Code-block newlines remain literal Code text. Hard-break behavior is settled.
- [x] **Links:** render the visible label with capability-dependent underline/color and no inline URL; cursor focus reveals the destination in the status/footer and `Space+l` opens its prompt. Match VOA's `http`, `https`, `ftp`, `git`, `obsidian`, `notion`, and `message` `://` scheme allowlist; reject invalid URLs in-place, while an empty URL unlinks. Centralize the policy for a future approved tool/protocol extension; M3's allowlist is fixed and unknown imported schemes do not become newly creatable. Typing inside a Link extends it; typing exactly before or after stays ordinary text, so users select text explicitly to extend a Link. Cursor context exposes this behavior: inside, `Link → URL · typing extends link · Space+l edit · Space+y copy`; at an edge, `Link edge → URL · typing is outside link`. Link labels may combine inline marks and hard breaks; serialize marks inside the Link and replace rather than nest a Link URL. Imported empty-label Links render their URL as an editable styled label; Vimkunja does not create them. Imported non-dangerous schemes outside M3's allowlist render and are preserved with an unsupported-scheme footer hint; they can be replaced/removed but not newly created. Truly unsafe schemes defer to the global unsafe-HTML decision.
- [x] **Opaque structures:** Images, Mentions, embedded media, Tables, and unknown structures use readable safe projections and the atomic Opaque-subtree rules above. Their hidden source data is never disclosed; native formatting skips them, while complete deletion or replacement requires confirmation.
- [x] **Opaque preservation:** unrelated edits retain original subtree bytes; native-node unknown attributes retain semantic values; structural edits and saves use the validation rules above.
- [x] **Malformed and unsafe HTML:** executable/active content (`script`, event attributes, dangerous URLs such as `javascript:`) makes the entire Description read-only with a clear safety hint and safe neutral projection; never sanitize-and-save automatically. Input requiring structural parser repair is read-only, renders a safe best-effort text projection, and reports `Error: Malformed HTML`. Provide two explicit confirmed actions: **Clear Description** replaces it with empty content; **Recover text** replaces it with a new editable plain-text Description from the safe text projection. Both intentionally discard the malformed HTML; neither is automatic. After confirmation, each creates an ordinary unsaved Description draft (empty for Clear, prefilled and in Insert Mode for Recover text); normal save persists it and discard retains the malformed source. Malformed-input behavior is settled.
- [ ] **Fixture acceptance:** use a corpus of many focused Source HTML snippets, grouped as Native, mixed Native/Opaque, or Blocked, plus a smaller set of risky compound cases. A traceability matrix must map every settled decision to at least one fixture. Each fixture records source, classification, projection, serialization/preservation, a representative edit when available, feedback/save behavior, and tested invariants. The corpus combines versioned VOA-generated, synthetic edge, and minimized regression inputs. Golden expectations are manually reviewed and may not be silently regenerated from implementation output. Continue with required regression/property/fuzz evidence.

## Grilling outcomes — questions 1–23

These are the recorded outcomes, not unanswered questions. Practical validation and final technical selection remain outstanding; see **Next session** below.

### Fixture acceptance and evidence

1. **Resolved:** The corpus collectively verifies terminal projection, classification, save/round-trip behavior, and representative editing behavior. Each fixture covers whichever goals apply; the suite must provide sufficient coverage of every goal.
2. **Resolved:** Aim to add a semantic round-trip check for every contract-listed Native property: structure, text, marks and nesting, block boundaries, list/Checkbox state and identity, Link destinations and meaningful attributes, and exact decoded Code content. A check may be omitted if an exception makes it inapplicable.
3. **Resolved:** For every retained Opaque subtree, tests should verify its original HTML bytes remain exact after unrelated edits—including tag spelling, attributes, whitespace, children, and nesting—and verify its identity, order, and parent position.
4. **Resolved:** Property tests should verify that valid Native content parses and serializes back to equivalent meaning, and that parsing/rendering never panics or exposes raw markup.
5. **Resolved:** Include arbitrary HTML and malformed, partially parseable, and unsafe inputs in fuzzing. Keep fuzzing modest and complementary: prioritize a comprehensive deterministic suite for known issues and contract cases.
6. **Resolved:** Use a small, bounded fuzz run in regular CI. No longer scheduled fuzz runs are required; discovered defects should become deterministic regression cases.
7. **Resolved:** Walk through every agreed decision and known rich-text element—including basic blocks and advanced behavior—and create happy-path and relevant corner-case examples for each. Use the traceability matrix to track coverage. There is no rigid fixture-count definition of done; a practical completion check is that every decision/element has relevant examples and the matrix has no known uncovered entries after review.
8. **Resolved (initial set):** Include compound fixtures for the identified risky interactions: quotes containing lists or Code blocks, formatting inside Links, Opaque content between editable text, and Tables mixed with editable content. Treat this as a starting set; add cases if the inventory walkthrough reveals further risks.

### Editor syntax — resolved

9. `***text***` means one bold plus one italic layer, with canonical `<strong><em>text</em></strong>` output; repeated emphasis is distinct.
10. Repeated semantic emphasis discloses nested delimiter pairs separated by protected presentation-only separators. They are neither cursor targets nor persisted text.
11. Backslash escapes the next syntax character: backslash, `*`, `_`, `~`, backtick, `#`, `-`, `+`, `.`, or square bracket. Before other characters or at a block's end it stays literal. Source HTML stores literal results, not escape syntax.
12. Directly typed block-start `# `/`## `/`### ` and `- `/`+ `/`* `/`1. ` require a following space to create headings/lists. Dash-only lines of at least three dashes create rules; three/four backticks in an otherwise empty paragraph create Code. Escaping the first marker suppresses conversion. Pasting never triggers structure: pasted `# Shopping` and `- Milk` become literal ordinary paragraphs.

### Source disclosure and cursor mapping — practical validation pending

13. **Provisional:** Anchor to the document character/insertion boundary and recalculate screen coordinates. The user explicitly requests practical examples before final acceptance.
14. **Provisional:** Reflow the whole current logical block; other blocks keep their own layout but may shift vertically. Code does not soft-wrap.
15. **Agreed:** Editable delimiters behave as ordinary Vim characters and count in motions such as `5l`. Protected markers are skipped without character/word counts; selections cross them without copying/deleting/formatting them as text. One complete Opaque subtree is one word-motion unit for `w`/`b`/`e`. Its atomic extent must be discernible through configurable bracket style and highlighting color, with non-color and multiline boundary cues; hidden raw attributes stay hidden. Exact presentation awaits prototype review.
16. **Provisional:** Keep the cursor at its existing screen row where possible by adjusting vertical scroll; clamp at boundaries and keep it visible. Focus changes do not reset Code/Table horizontal scroll.
17. **Provisional:** Wrap prose and editable delimiters without hiding characters; keep protected separators attached to neighboring delimiters. `gj`/`gk` traverse wrapped rows, whose continuation arrow is non-editable and distinct from the hard-break marker. Code/Tables scroll horizontally. If fixed prefixes leave no room for a content character, show `Pane too narrow` and preserve draft/cursor.

### Prototype validation — provisional design accepted, evidence outstanding

18. **Validation required:** Compare wide/narrow layouts with formatted paragraphs, nested lists/quotes, and Opaque subtrees. Include cursor anchoring, disclosure reflow, scrolling, wrap arrows, and atomic boundaries. Readability is not yet validated.
19. **Provisional:** The Space formatting palette opens in Normal/Visual Mode in a secondary footer below the primary footer. If the extra footer would leave fewer than three visible Description rows, temporarily replace the primary footer and restore it on close. Use compact contextual choices at narrow widths. Insert-Mode spaces remain ordinary text. An overlay is a comparison alternative, not a settled fallback.
20. **Provisional:** Retain Visual `Space+b/i/u/s/l` and `Space+0`; validate comfort/conflicts.
21. **Provisional:** Retain Normal `Space+1/2/3/0`, `Space+u/n/x`, and `Space+c/q`; validate discoverability and mode-specific meanings.
22. **Agreed semantic change:** Toggle marks uniformly, not per segment. If any selected Native text lacks bold/italic/underline/strikethrough, apply that mark throughout; if all has it, remove it throughout. Opaque subtrees are ignored in the decision and preserved. Links retain the separate URL flow. Practical interaction validation remains outstanding.

### Technical selection — evaluation direction accepted

23. Evaluate the existing `golang.org/x/net/html` tokenizer with a source-preserving token layer, explicit structural/safety validation, original source slices for Opaque subtrees, and a dedicated Native serializer. The current tree parser repairs malformed input and is insufficient alone. Final selection requires representative fixture evidence for preservation and malformed-input detection; no implementation or dependency change is authorized.

### Next session

Create the concrete fixture corpus and traceability matrix with reviewed projections/goldens. Schedule practical validation of provisional interaction decisions, including the explicitly requested cursor examples. Use the corpus to evaluate the technical direction before final parser/serializer selection. Do not reopen settled policies unless concrete evidence exposes a contradiction.

### Remaining deliverables after decisions

- [ ] Create the concrete fixture corpus and traceability matrix.
- [ ] Add expected projections and manually reviewed golden outputs.
- [ ] Add regression, semantic round-trip, preservation, property, and fuzz tests.
- [ ] Add the Vimkunja markup dialect to Help, durable documentation, and the dedicated markup tutorial.
- [ ] Split implementation into concrete tickets after the contract and corpus are accepted.

## Out of scope

- The M2 HTML-to-plain-text reader and its nine presentation decisions.
- Changing the Vikunja SQLite schema or introducing a second persistence format.
- Persisted Task-action undo; Description-local draft undo/redo remains part of this M3 scope.
