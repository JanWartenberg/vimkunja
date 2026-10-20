# 02 — Specify native Rich-Text Description rendering and editing

**Phase:** M3 — structured reader and editor

**Specification scope:** The M3 product rules for a native, keyboard-first terminal projection and editor are resolved. Maintain the accepted contract and examples and coordinate the existing implementation tickets 03–10.

**Implementation prerequisites:** M2's read-only HTML-to-plain-text reader is complete; see its [archived ticket](../../../2026-08-05/description-rendering/issues/01-render-vikunja-html-descriptions.md). The decisions in this ticket govern the M3 implementation; the existing [Rich-Text Description Contract](../spec.md) is the starting point and must be updated only when a decision here changes it.

**Category:** enhancement

**Priority:** Medium

**Status:** completed

**Technical baseline — 2026-10-04:** product feedback is resolved. Spec, keybindings, glossary and 83 independent synthetic examples are synchronized with accepted V2/V4/default decisions. Artifact/dependency probes and focused Store preservation/rollback tests pass; current native editor/serializer remains unimplemented. Concrete technical gaps: atomic baseline compare-and-save, source-slice Code decoding, Code-specific literal-prefix/Paste dispatch, real Windows/German terminal captures, Native round trips/fuzz/layout and versioned VOA output. The user authorizes this synchronization and technical verification pass; no production Native editor or dependency changes are included.

**Handoff state — 2026-10-05:** The user reviewed the highlighted rule cards and accepted the current product-rule set (“ich denke, das passt alles”). L01/L10 and Opaque/selection/o/O/ü/Ü expectations are confirmed. A generated Checkbox UUID can be shown as an illustrative `<randomUUID>` placeholder; its randomness does not make L10's visible result undetermined. The specification/decision phase and implementation-ticket preparation are complete. This ticket stays as the coordination record for the execution deliverables delegated to 03–10; Native implementation began on 2026-10-06 with ticket 03 in the isolated branch `feature/description-model`; the integrated editor is not claimed complete.

**Scope boundary:** This is phase 2 of the Description plan. It replaces neither Vikunja's HTML persistence format nor its SQLite schema. It aims to natively render and edit every HTML structure that is useful and feasible in a terminal. Content that is not suitably editable must remain as readable as practical and must be preserved without destructive rewriting.

## Questions for dedicated grilling

The contract already establishes Source HTML as the persistence format, a structured document model, safe preservation of Opaque subtrees, and an initial native set of headings, inline marks, lists, Checkbox lists, code blocks, quotes, rules, and links. The original interview topics below are retained for context; product decisions are now resolved. Remaining technical validation is recorded in the current handoff:

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

- parsing and rendering do not panic or accidentally disclose Source HTML markup; literal HTML-looking user text remains visible as text
- supported content round-trips to equivalent Vikunja-compatible HTML semantics
- Opaque subtrees, attributes, and unknown content survive unrelated edits
- unsafe or ambiguous content becomes read-only rather than being destructively rewritten
- list, Checkbox-list, link, formatting, and block boundaries retain their meaning
- Description edits never change the enclosing Task's Done state
- external-save conflicts are rejected rather than silently overwritten

The corpus is a specification deliverable and must exist before native editing implementation begins. It should support fixture-based regression tests, semantic round-trip tests, and property/fuzz tests for parser and serializer boundaries.

## Implementation tickets — 2026-10-05

The rules are accepted; these tickets make the remaining work executable. No new product confirmation round is required. Start with **03**; **10** can begin gathering real VOA examples early. After 03, 04 and 05 can proceed independently; 06 integrates their safe save and projection paths, then 07–09 finish editing features. Ticket 10 completes the integrated verification and user documentation.

| Ticket | Deliverable | Blocked by |
| --- | --- | --- |
| [03 — Source-preserving Description document model](../../../2026-10-07/description-rendering/issues/03-implement-source-preserving-description-model.md) | Document/HTML preservation — completed 2026-10-07 | None |
| [04 — Atomic Description saves and Checkbox identities](../../../2026-10-07/description-rendering/issues/04-implement-atomic-description-save.md) | Completed 2026-10-07; transactional conflict check and UUID adoption, committed 2026-10-08 | — |
| [05 — Native terminal projection and document navigation](../../../2026-10-08/description-rendering/issues/05-implement-native-description-projection-and-navigation.md) | Read/navigation and terminal layout | Completed and committed 2026-10-08 |
| [06 — Native draft editing, Visual Mode and local history](../../../2026-10-09/description-rendering/issues/06-implement-description-draft-editor-and-visual-modes.md) | Draft/Visual/history/recovery completed 2026-10-09; committed 2026-10-09 | — |
| [07 — List, Checkbox and Quote structure editing](../../../2026-10-10/description-rendering/issues/07-implement-list-and-quote-structure-editing.md) | L01–L12 / Q01–Q04 completed 2026-10-10; committed 2026-10-10 | — |
| [08 — Code editing, Tabs and terminal input](../../../2026-10-10/description-rendering/issues/08-implement-native-code-editing-and-terminal-input.md) | Completed 2026-10-10; exact Code input/exits, Tabs/retab/config and synthetic terminal dispatch; committed 2026-10-10 | — |
| [09 — Inline syntax, Inline-Code and Link editing](../../../2026-10-10/description-rendering/issues/09-implement-inline-syntax-code-and-links.md) | Completed 2026-10-10; five review rounds and targeted final fixes verified, committed 2026-10-10 | — |
| [10 — Native Rich-text compatibility, terminal acceptance and documentation](10-verify-native-rich-text-compatibility-and-documentation.md) | VOA/terminal evidence, properties/fuzz, Help/tutorial | Integration completion: 03–09; preparation unblocked |

Completed specification work is distinct from future Native test execution. The remaining deliverable checkboxes below remain open until their owning tickets provide evidence.

## Working plan

- [x] Establish the corpus plan and initial risk-based fixture inventory.
- [x] Resolve product rules/defaults and synchronize applicable fixture expectations; technical execution and exact VOA output remain outstanding.
- [x] Add a concrete synthetic HTML fixture draft with proposed terminal projections; see [corpus](../../../../.scratch/description-rendering/corpus.md). Expectations are not accepted goldens yet.
- [x] Draft semantic round-trip and opaque-content preservation assertions for those fixtures; artifact/dependency and existing Store checks added; Native M3 execution still outstanding.
- [x] Review synthetic expectations, resolve product ambiguities, correct stale references, and add Mention label/ID cases (83 records).
- [ ] Add representative versioned VOA-generated fixtures; synthetic inputs do not certify VOA output.
- [ ] Execute Native parser/serializer boundary property or fuzz cases during implementation.
- [x] Prepare concrete implementation tickets 03–10 from the accepted rules and current corpus; ticket 03 implementation is now in progress on its isolated branch.

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
- Empty imported semantic mark wrappers are removed; newly typed empty delimiter pairs stay literal.
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
- Exactly three or four directly typed backticks in an otherwise empty paragraph create the same structured Code block only after Enter confirmation; consume the complete trigger and place the cursor on an empty Code line between canonical triple-backtick fences. Before confirmation they are ordinary text. Five backticks, additional characters, literal paste and an escaped first marker do not trigger conversion. `Space+c` is the accepted direct creation path; no additional creation command is required.
- Editing preserves every code character exactly, including indentation, tabs, blank lines, and a final trailing newline. Rendering and serialization do not normalize Code-block text. `Enter` inserts a literal Code line; paste preserves line breaks and whitespace.
- Code-block editing uses ordinary Description Insert and Normal modes, not a separate Code Block Mode. `Esc` only returns to Normal Mode.
- In Normal Mode, `o` and `O` create Code lines below and above the current source line. `dd` deletes the current source line; on an empty Code block, `dd` deletes the entire block.
- `{` and `}` move backward and forward by paragraph-motion units; Normal-mode `ö` and `ä` are German-keyboard aliases. Every navigable Description block is one paragraph-motion unit, and an entire Code block remains one unit regardless of its internal or empty lines. Counts apply. Within a Code block, `j` and `k` continue to move between source lines.

### Block quotes

- A Block quote is the HTML `<blockquote>` element for an extended quotation; it is distinct from an inline HTML `<q>`. `<q>` has no native M3 semantics and is preserved as unsupported inline content.
- A quote's first visual row uses the presentation-only prefix `│  `; its wrapped continuation rows use `│↪ `. ASCII capability fallbacks are `|  ` and `|> `. Nested quotes add a prefix at each depth. No quote-specific background or color is required.
- Quote prefixes are never editable or cursor targets, in either Normal or Insert mode. Quote text is edited directly; quotes never disclose pseudo-Markdown `>` syntax.
- A quote is a structural container for every native editable Description block: paragraphs, headings, lists and Checkbox lists, Code blocks, rules, and nested quotes. Its child blocks retain their normal semantics.
- `Space+q` wraps the current unquoted block in a quote; reapplying it removes the applicable surrounding quote layer while retaining its child content. It does not create deeper nesting. Quote-depth increase/decrease use V, selection motions, Space+q/w (remove/add). Q01/Q02/Q04 define local/shared/atomic scope; Q03 permits explicit confirmed metadata loss on removed-layer unwrapping, while retained metadata during splitting needs focused coverage. The former V >/< proposal is superseded. `Enter` creates a sibling child paragraph at the same quote depth; `Enter` on an empty quoted paragraph exits the quote and creates a following normal paragraph. At a quoted child paragraph's boundaries, `Backspace` and `Delete` merge only an adjacent compatible child at the same quote depth; they never implicitly move ordinary text across a quote boundary.
- Code blocks and quotes are symmetric structural blocks: each complete container is one `{`/`}` paragraph-motion unit; `j`/`k` navigate its internal logical lines (literal source lines for Code, child blocks for quotes); `dd` deletes the current internal unit; and `o`/`O` create an internal unit below/above (a Code line or same-depth quoted paragraph). Deleting the final quote child removes the quote. Deleting the final nonempty Code line leaves empty Code; a subsequent dd deletes the empty Code block (C05).
- Imported empty quotes are preserved and render as one empty quoted editable line. `Space+q` on an empty paragraph creates one. Deleting its sole internal line removes it.
- Existing parseable `cite` and VOA `data-comment-id` attributes are preservation-only: Vimkunja carries them through unrelated edits but offers no M3 input or editing UI or Task-Description reply semantics. This follows the general rule to preserve existing content whenever safe, rather than creating surprising data loss.

### Opaque-subtree preservation

- The outermost unsupported ancestor is one atomic Opaque subtree. Descendants may contribute to its safe projection but are not independently focused, selected, edited, or validated.
- Unrelated edits preserve the subtree's original HTML byte-for-byte, including tag spelling, attributes, whitespace, children, and nesting. Native content around it remains editable.
- Selections may cross only complete opaque atoms. Formatting applies to selected native text and skips opaque atoms unchanged. Deletion and replacement of complete opaque atoms require confirmation; cancellation changes nothing. `x` on an atom targets the atom. `dd` and `V d` target the complete logical block(s), including surrounding Native text and inline opaque atoms, with one confirmation for the whole deletion and exact whole-operation Undo. Standalone opaque blocks are likewise deleted by `dd` after confirmation; soft-wrapped screen rows are not separate logical blocks.
- Copy exposes only the safe visible projection, never hidden source data. Pasting it inserts new Native text nodes; it does not duplicate the opaque structure or its IDs. Undo restores confirmed removals byte-for-byte at their original position and nesting; Redo reapplies them without another confirmation.
- Structural changes to a native parent containing an opaque descendant use confirm-and-validate. They proceed only if serialize-and-reparse validation keeps the subtree byte-for-byte at its intended position.
- Every save containing opaque content performs one linear-time preservation round trip, not per-keystroke validation. It verifies retained opaque subtree bytes, identity, order, nesting, and parent position; failure rejects the save and keeps the draft open. Deliberately removed subtrees are excluded.
- Safe unknown attributes do not make a Native node opaque. Preserve their names and decoded values while permitting order, quoting, and equivalent entity normalization. Carry them through structural conversion when safe; otherwise reject the conversion rather than dropping them. Q03 is an explicit narrow exception: quote-layer removal may discard only the removed container's metadata after the user's requested confirmation, preserving children and ordinary Undo/cancel semantics; no silent loss or broader conversion exception is authorized.
- Explicit conversion, unwrapping, and raw-HTML editing of opaque structures are deferred follow-up work.

## Decision status and remaining evidence

The policy interview is complete for questions 1–23. Provisional interactions still require practical validation. Do not reopen settled decisions unless a fixture or prototype exposes a contradiction.

### Immediate next chunk

**Historical first-pass summary:** these checkmarks record the original 1–23 interview, not complete approval of every subsequently discovered edge case. For current decisions, exceptions and next actions use the latest V2/V4 checkpoint below; in particular later local protection, Code trigger/exit, automatic ID maintenance and Native Inline-Code decisions supersede the original wording.

- [x] **Code blocks:** source disclosure, creation, line preservation, editing, deletion, and paragraph-motion behavior are settled above.
- [x] **Block quotes:** projection, wrapping prefixes and fallbacks, non-editable markers, nested native content, creation/removal, entry/exit, internal navigation/deletion, empty quotes, and preservation attributes are settled.
- [x] **Hard breaks:** projection, navigation/deletion, whitespace, edge cases, and nested-content behavior are settled.
- [x] **Links:** display, editing, VOA-compatible extensible scheme policy, formatting, unknown-scheme preservation, and clipboard copy are settled; unsafe URLs follow the global safety rule.
- [x] **Opaque structures:** Images, generic attachment scope, Mentions, embedded media, Tables, unknown-node fallback, and safe-attribute preservation are settled.
- [x] **Malformed HTML:** fail-closed safety, read-only projection, and confirmed Clear/Recover-text draft flows are settled.
- [x] **Opaque preservation:** outermost atomic boundaries, byte-for-byte retention, cross-boundary operations, confirmed removal/replacement, safe copying, exact Undo, native-node attributes, structural validation, and save-time round trips are settled.
- [x] **Serialization:** settled and recorded in the contract: canonical semantic serialization for Native nodes, byte-exact Opaque preservation, compact canonical escaping, Checkbox-ID assignment and automatic minimal deduplication (current chat supersedes the original confirmation requirement), canonical VOA Link targets with safe meaningful attribute preservation, structured semantic equivalence, standards-aligned mark semantics, meaningful empty-block preservation, root-fragment normalization, prose-whitespace collapsing, duplicate-attribute safety, and Opaque handling for styles and comments.

### Remaining fixture categories

- [x] **Hard breaks:** `<br>` renders as a real terminal row break with a subtle, non-editable `↵` end marker while its paragraph is active (`>` ASCII fallback), hidden when unfocused. It remains inside one paragraph: `j`/`k` skip blocks, `gj`/`gk` traverse its rows, left/right cross it, and adjacent `Backspace`/`Delete` remove it. Preserve adjacent imported whitespace in the model and serialization; newly entered content follows normal HTML-renderer-visible behavior. Leading, trailing, and consecutive breaks are preserved and permitted, including intentional blank rows. The same semantics apply inside headings, list/Checkbox-list entries, and Block quotes; Code-block newlines remain literal Code text. Hard-break behavior is settled.
- [x] **Links:** render the visible label with capability-dependent underline/color and no inline URL; cursor focus reveals the destination in the status/footer and `Space+l` opens its prompt. Match VOA's `http`, `https`, `ftp`, `git`, `obsidian`, `notion`, and `message` `://` scheme allowlist; reject invalid URLs in-place, while an empty URL unlinks. Centralize the policy for a future approved tool/protocol extension; M3's allowlist is fixed and unknown imported schemes do not become newly creatable. Typing inside a Link extends it; typing exactly before or after stays ordinary text, so users select text explicitly to extend a Link. Cursor context exposes this behavior: inside, `Link → URL · typing extends link · Space+l edit · Space+y copy`; at an edge, `Link edge → URL · typing is outside link`. Link labels may combine inline marks and hard breaks; serialize marks inside the Link and replace rather than nest a Link URL. Imported empty-label Links render their URL as an editable styled label; Vimkunja does not create them. Imported non-dangerous schemes outside M3's allowlist render and are preserved with an unsupported-scheme footer hint; they can be replaced/removed but not newly created. Truly unsafe schemes defer to the global unsafe-HTML decision.
- [x] **Opaque structures:** Images, Mentions, embedded media, Tables, and unknown structures use readable safe projections and the atomic Opaque-subtree rules above. Their hidden source data is never disclosed; native formatting skips them, while complete deletion or replacement requires confirmation.
- [x] **Opaque preservation:** unrelated edits retain original subtree bytes; native-node unknown attributes retain semantic values; structural edits and saves use the validation rules above.
- [x] **Malformed and unsafe HTML:** executable/active content (`script`, event attributes, dangerous URLs such as `javascript:`) protects the smallest reliably bounded subtree as Opaque, with a safe neutral projection and editable/savable Native neighbors; unreliable boundaries may require wider blocking; never sanitize-and-save automatically. Input requiring structural parser repair is read-only, renders a safe best-effort text projection, and reports `Error: Malformed HTML`. Provide two explicit confirmed actions: **Clear Description** replaces it with empty content; **Recover text** replaces it with a new editable plain-text Description from the safe text projection. Both intentionally discard the malformed HTML; neither is automatic. After confirmation, each creates an ordinary unsaved Description draft (empty for Clear, prefilled and in Insert Mode for Recover text); normal save persists it and discard retains the malformed source. Malformed-input behavior is settled.
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
12. Directly typed block-start `# `/`## `/`### ` and `- `/`+ `/`* `/`1. ` require a following space to create headings/lists. Dash-only lines of at least three dashes create rules; exactly three/four directly typed backticks in an otherwise empty paragraph create Code only after Enter confirmation. Escaping the first marker suppresses conversion. Pasting never triggers structure: pasted `# Shopping` and `- Milk` become literal ordinary paragraphs.

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
22. **Agreed semantic change:** Toggle marks uniformly, not per segment. If any selected Native text lacks bold/italic/underline/strikethrough, apply that mark throughout; if all has it, remove it throughout. Explicit Bold-off removes every selected bold-family layer, including repeated strong, while unrelated edits retain imported variants/depth. Accepted E01 removes all layers of the selected Italic/Underline/Strike family too, leaving other marks unchanged. Opaque subtrees are ignored in the decision and preserved. Links retain the separate URL flow. Practical interaction validation remains outstanding.

### Technical selection — evaluation direction accepted

23. Evaluate the existing `golang.org/x/net/html` tokenizer with a source-preserving token layer, explicit structural/safety validation, original source slices for Opaque subtrees, and a dedicated Native serializer. The current tree parser repairs malformed input and is insufficient alone. Final selection requires representative fixture evidence for preservation and malformed-input detection; no implementation or dependency change is authorized.

### Next session — resume here

**Saved checkpoint:** `1a5e5b7` records the accepted interview decisions; `927796d` adds the original 34-fixture synthetic corpus draft. A subsequent [maintenance pass](../../../../.scratch/description-rendering/corpus.md#non-semantic-maintenance-pass) extends it to 42 fixtures with settled editing/fallback/safety scenarios and preservation-failure assertions, and repairs documentation drift without approving goldens or choosing unresolved semantics. Artifact validation passed for the extended draft. The Go tests/build passed at the original checkpoint; they are not M3 evidence. At that original checkpoint no expectations were approved; later user rule/field approvals are recorded in the corpus review log. No M3 behavior tests have run.

#### Latest user checkpoint — V2/V4 feedback and next-session grilling

**Historical user request (superseded by later defaults and verification request):** retain all unresolved topics for the next session and use **classic grilling in chat**. Ask one concrete question at a time, with a small example, alternatives and recommendation; wait for the answer before proceeding. Explain terms first. HTML is only for quick questionnaires, never the discussion venue. Do not reopen accepted rules or infer approval from omitted questions.

**Source trail:** V1 answered L01–L12, Q01–Q04, C01; V2 answered C02–C08 and E01–E09; V4 answered S01, S06, S12, S08, K02, P08. Questionnaire snapshots remain in `explain-vimkunja-open-questions{,-v2,-v3,-v4}.html` (historisches lokales HTML-Artefakt, nicht im Repository enthalten); the revised V3 starts at E10 and V4 has six cards. They are historical recommendations, not the current resume workflow or durable approval evidence. The decision trail records the feedback; the 2026-10-04 synchronization below propagates its accepted rules into current spec/corpus/keybindings.

**Explicitly confirmed in V2; do not re-ask:**
- C02: existing user TOML under `tui`, Description-specific setting, no dependency change. Exact key/schema is not yet approved.
- C04: Shift-V inside Code selects source lines, footer `VISUAL CODE LINES`; `j/k` extends, e.g. A+B or A+B+C. Outside Code, `VISUAL BLOCKS`. Crossing its boundary remains open.
- C05: deleting the last nonempty Code line leaves an empty Code block; another `dd` deletes the empty block.
- C08: preserve decoded Code characters, including initial newline and CRLF; unreliable preservation must not permit editable saving.
- E01: Italic/Underline/Strike-off removes all selected layers of its own family, leaving other marks unchanged; unrelated edits retain variants/depth.
- E02/E03: protected variant labels (`b`/`strong`, analogously `i`/`em`); midpoint between repeated delimiter layers, ASCII `|`, no motion-count consumption. Presentation approval is not practical wrap/layout evidence.
- E04/E09: protect imported literal syntax using editor escapes; all paste is literal, including inline markers. Save retains literal characters. Destination-context inheritance remains separate.
- E05: `viw/vaw` targets useful text, not automatically its formatting delimiters; `h/l` can reach editable delimiters. Delimiter-start word selection remains open.
- E06/E07/E08: incomplete edited syntax is literal; palette acts on selected semantic text, not syntax positions; ordinary `d/x` edits syntax. Formatting retains the document-anchored selection until Esc.

**V4 feedback — preserve exact approval boundaries:**
- **S01 unresolved:** user did not understand “blockieren”; asked whether it hides media or disables Description editing. Assistant clarified that the old recommendation disabled the **whole Description**, whereas Opaque preservation would allow surrounding text edits. User finds Opaque video/iframe preservation broadly understandable, but has not approved either the old blocking rule or a complete safety policy.
- **S06 direction accepted, mechanics open:** retain an imported empty list, but allow filling it; user rejects permanent Opaque/read-only treatment. Do not silently normalize or invent an entry on import.
- **S12 historical warning superseded:** after checking local Vikunja ID usage, the user accepts automatic minimal Checkbox-ID maintenance. Preserve valid unique IDs, fill missing IDs, replace only later duplicates; no whole-list ID clearing or ID-loss prompt. Actual content/meaningful metadata loss still requires confirmation. This does not authorize generic Opaque conversion.
- **S08 historical conditional answer, superseded by current chat:** V4 left Inline-Code staging unclear. The user now explicitly chooses A: Native Inline-Code reading, text editing, creation and formatting removal are included in V1. Neither Inline-Code nor Code blocks are deferred by that earlier answer. Detailed grammar defaults and fixture evidence remain to be prepared.
- **K02 accepted:** explicit unlink of an empty-label Link retains the visible URL fallback as plain text; explicit deletion removes it. User asked about a special shortcut; assistant explained existing `Space+l` → empty URL, no new shortcut proposed. K01 first-label-edit materialization was subsequently accepted in the remaining-default package.
- **P08 accepted as provisional test direction:** second footer only if at least three Description rows remain; otherwise temporarily replace primary footer. No automatic overlay. Real layout/comfort validation still required.

**Current chat continuation — C03 Tab decisions:** ordinary text changes preserve existing Tabs. Add `:retab` as an explicit Tab-to-Space command; conversion goes to the next tabstop as in Vim, default width four and configurable, rather than always inserting four Spaces. Add Code Insert Mode `Ctrl+V` then `Tab` for literal Tab insertion, with provision for a configurable alternative binding if the Windows terminal intercepts Paste. The user reports having used `Ctrl+B` as an alias in Vim; no exact alias or configuration key is approved here. Confirmed command scope: without selection, `:retab` affects only the current Code block; selected Code lines limit it to those lines. The user proposes `:retaball` for the complete Description; all Native Code blocks versus also editable prose Tabs remains an explicit question, with no permission to rewrite Opaque contents. These details are now accepted in remaining-default card 4; only fallback alias/config key details in card 14 and technical evidence remain pending. Only these Tab decisions are synchronized into the spec in this pass; no runtime implementation or fixture expectations change.

**Current chat continuation — C06 accepted keystrokes:** from Normal Mode anywhere inside a Code block, `ü` / `Ü` creates a normal paragraph after / before the entire block and enters Insert Mode there. `]` then `Space` / `[` then `Space` creates and focuses the corresponding paragraph in Normal Mode; `i` starts typing. These also work before the first/after the final Code block and leave Code contents unchanged. `o` / `O` continue to create Code source lines. Bind by character, not physical keyboard position; do not remap English `[` / `{` to `ü` / `Ü`. Existing `{` / `}` and `ö` / `ä` paragraph motions stay intact. The earlier `:after` card proposal is superseded, not accepted as an additional command. Spec, keybinding contract and the current HTML explanation are synchronized for this decision only; no runtime implementation or fixture expectation is changed.

**Current chat continuation — C07 accepted trigger:** the user accepts the complete Backtick-trigger card: exactly three or four directly typed backticks require Enter to create Code; otherwise no block conversion. Five backticks/additional characters remain paragraph text, paste is literal, escaping prevents conversion, Backspace edits the candidate, and Esc only leaves Insert Mode. Confirmation is one Undo step; after it, a new backtick is ordinary Code text. `Space+c` (lowercase c in the existing contract) is the direct Code creation path. No timer, no swallowed fourth character. This is contract approval, not terminal-event or runtime evidence.

**Current chat continuation — S08 Native Inline-Code accepted:** after comparing A (Native in V1) with B (initially Opaque), the user chooses A. V1 includes reading imported Code spans, editing their text, creating spans and removing their formatting. No demonstrated technical blocker justifies the previous staging proposal; it is superseded. Single-backtick editor syntax is the intended form, with other markup-looking characters literal inside Code. Boundary/escape/empty-span/mark-combination details are to be covered by consistent defaults and fixtures, not interpreted as previously reviewed outputs. This is scope approval, not runtime evidence or permission to start implementation.

**Current chat continuation — S12/S13 automatic minimal ID maintenance accepted:** the user authorizes updating the contract after the local Vikunja source check. Checkbox IDs identify editor entries, not database Tasks. Preserve valid unique existing IDs, automatically fill missing IDs and regenerate only later duplicate occurrences on successful save; no ID-loss warning, confirmation or clearing the whole list. Preserve text, order, nesting, checked state and other meaningful metadata. Unknown/retained Opaque content stays protected; no generic conversion permission. Keep the existing UUID v4 generation policy. Repair candidates and persistence remain atomic under failures/conflicts; only successful save adopts generated IDs. Confirmation remains for actual content/meaningful-metadata loss. The `checkbox-duplicate-ids` draft assertions are updated for this explicit rule only, without accepting the full corpus or claiming M3 tests.

**Partial Default acceptance — rest-card numbers 1–7 only:** the user skimmed and explicitly agrees with cards 1–7 of `explain-description-restpunkte-2026-10-03.html` (historisches lokales HTML-Artefakt, nicht im Repository enthalten). Stable IDs: `overview`, `principles`, `protection`, `tabs`, `code-selection`, `empty-lists`, `syntax`. Accepted rules, including collapsed supplementary content, are consolidated under **Accepted remaining-default cards 1–7** in the spec. This includes local active/duplicate-attribute protection, Tab/Retab/indent behavior, clamped Code-line selection and `:selectblock`, nested outside-Code paragraph behavior, fillable empty imported lists, and literal/Inline-Code grammar defaults. Cards 8–16 are explicitly unreviewed: motions/Unicode, Links, import/recovery detail, display fallbacks, metadata splitting, layout, config/fallback alias, evidence work and summary. No approval of those cards, fixture outputs, runtime implementation, harness or tests is inferred. Safety classification/goldens need a targeted synchronization pass rather than automatic regeneration.

**Reduced critical review — remaining defaults adopted:** noncritical remaining defaults are accepted; cards 13/14 remain subject to practical checks and cards 15/16 are technical work. The final Quote-source choice is also accepted on 2026-10-04, as recorded below. The current HTML separates accepted decisions from optional technical work. No runtime implementation, fixture regeneration or blanket evidence approval follows.

**Accepted Quote-source split — 2026-10-04:** splitting a Quote around an unwrapped middle paragraph retains the same known source reference (`data-comment-id` or `cite`) on both retained Quote fragments, without an additional confirmation for repeating that reference. For source comment 42, the result is `<blockquote data-comment-id="42"><p>Absatz A</p></blockquote><p>Absatz B</p><blockquote data-comment-id="42"><p>Absatz C</p></blockquote>`. Text and order remain unchanged. The reference identifies the original comment, not the Quote container. Truly unique IDs and unknown attributes are not copied indiscriminately; reject only the specific split if lossless allocation is unclear. Q03 confirmation still applies to actual metadata loss; Opaque preservation rules remain in force. This resolves the final targeted product question, not the outstanding implementation or evidence work.

#### Accepted topics — remaining execution checks

All former queue topics have product rules. The [current traceability matrix](../../../../.scratch/description-rendering/corpus.md#traceability--accepted-decisions-and-technical-checks) maps every fixture to its decision family. Remaining work is execution and practical evidence:

| Topic | Required checks |
| --- | --- |
| Local protection/import/recovery | Reliably bounded Opaque bytes, optional endtags versus unexpected repair, safe neutral display and confirmed Blocked recovery. |
| Code and Inline-Code | Native trigger dispatch, clamped selection, same-container exit paragraphs, exact decoded CRLF/leading/trailing lines, retab/outdent and literal context handling. |
| Syntax/motions/marks | All-family off, protected variant labels, literal origin/escape parity, retained semantic selection and grapheme-safe edits. |
| Lists/Checkboxes/Quotes | Subtree/reading-order moves, numbering, empty fill targets, Beta child ownership, automatic minimal IDs and shared Quote sources. |
| Links/layout/config | Actual label materialization and URI validation; narrow/short reflow/scroll/palette; alias and settings integration. |
| Saves/preservation | Atom identity/order/parent position, conflict compare-and-write, UUID retry/adoption, complete rollback and draft retention. |

#### Current verification scope and prerequisites

The 2026-10-04 request authorizes Contract/example synchronization and HTML/Store/terminal checks. Completed: 83-record artifact validation, source-preserving tokenizer probes, terminal decoder probes and Store byte-preservation/rollback regressions. Source review identifies the remaining conflict and paste-dispatch gaps. Full details, commands and evidence limits: [corpus verification](../../../../.scratch/description-rendering/corpus.md#technical-verification--2026-10-04).

**Next work:** begin implementation with ticket 03 when separately authorized, then 04/05 and the dependent editor tickets in the implementation index. Ticket 10 capture and test-case preparation can begin independently; its final integration requires 03–09. Gather Native round-trip/property/fuzz, versioned VOA rich-text output and real native layout/Windows input evidence during execution. No further product-card vote is needed. This pass does not implement the Native editor or change dependencies/schema.

### Remaining deliverables after decisions

- [x] Create the initial synthetic fixture corpus and traceability matrix (draft only).
- [ ] Complete known coverage gaps and add VOA-generated provenance.
- [x] Independently author and statically review decision-equivalent examples; product rules accepted, no blanket user golden approval.
- [ ] Validate Native execution and actual VOA serialization/projection expectations.
- [ ] Add regression, semantic round-trip, preservation, property, and fuzz tests.
- [ ] Add the Vimkunja markup dialect to Help, durable documentation, and the dedicated markup tutorial.
- [x] Prepare Native implementation and integrated validation tickets 03–10; outstanding execution remains tracked in those tickets.

## Out of scope

- The M2 HTML-to-plain-text reader and its nine presentation decisions.
- Changing the Vikunja SQLite schema or introducing a second persistence format.
- Persisted Task-action undo; Description-local draft undo/redo remains part of this M3 scope.

## Comments

### 2026-10-04 — decision synchronization and technical verification

User requests continuing specification/example alignment and HTML-preservation, save and terminal checks. Synchronized previously missing V2 E01–E09/C04/C05/C08 and accepted defaults; added 39 focused examples to the existing 42 and explicitly corrected five obsolete blanket safety classifications. Added verification probes, current plain-text grouped-key input tests and Store regression tests without production behavior changes. Native behavior remains unexecuted; library probes expose CRLF normalization and automatic tree repair. Current clipboard/Paste handling and baseline-free UpdateTask require changes during Native implementation, not another policy interview. Real Windows/German input captures and real VOA output remain absent.

### 2026-10-04 — stale references and Mention examples

User requests the pre-implementation consistency cleanup. Replaced obsolete future-text-object and missing-list/triple-strong references with the current accepted rules and fixture links. Added `mention-label` and `mention-id-fallback`: independently authored synthetic Opaque inputs with label/id display, neighbor edits, skipped formatting, safe copy/paste, rejected internal edits and confirmed deletion/Undo. The corpus now contains 83 records. These are specification examples, not VOA captures or Native execution evidence; production implementation and dependencies/schema are unchanged.

### 2026-10-04 — ticket/status handoff review

User requests checking and committing ticket/status. Product triage is complete; ready-for-agent now applies to the remaining specification handoff and implementation-ticket preparation, not automatic production implementation. The current plan separates completed synthetic review/Mention coverage from missing versioned VOA output and Native execution. Spec and corpus are included in the same documentation commit so all 83-record references remain consistent.

### 2026-10-05 — rule acceptance and implementation tickets

User accepts the current rule-card set and requests preparing implementation tickets. Added 03–10 with scope, dependencies, focused corpus coverage and execution acceptance. Random UUIDs may be illustrated as `<randomUUID>` without pretending the visible Alpha/Beta/child result is unknown. Production behavior and existing corpus HTML expectations are unchanged; no implementation or automatic commit is part of this preparation.

### 2026-10-06 — planning consistency review

Replaced stale ticket-preparation instructions with the existing 03–10 handoff and dependencies. Clarified byte-exact transactional baselines, the Native Description save boundary and UUID validation for newly generated/repaired IDs; imported unique non-UUID IDs stay valid. VOA 2.7.0 database checks do not substitute for frontend rich-text captures or Native editor evidence. No product rules, corpus goldens or implementation changed.


### 2026-10-06 — implementation started on isolated branch

User authorizes starting 03 while main cleanup continues. `internal/description` now supplies a UI-independent import/snapshot/serialization seam. Ticket 03 records execution evidence and remains in-progress; no TUI Native editor, transactional save, UUID repair or terminal acceptance is claimed.


### 2026-10-07 — model completion

03 is implemented, re-reviewed and archived after intermediate checkpoint 0890503. Stable imported identities, document anchors and Native semantic round-trip checks complete the model seam; 04/05 are now unblocked. Model/corpus execution evidence is recorded in corpus.md and the archived ticket. Editor operations, UUID persistence and real VOA/terminal acceptance remain in 04–10.


### 2026-10-07 — atomic-save completion

04 implements immutable save candidates, minimal Checkbox UUID repair, atomic baseline comparison and Task/Comment persistence, failed-COMMIT cleanup and success-only editor adoption. It is archived with synthetic execution evidence; committed on 2026-10-08. 05 is next; 06 depends only on 05 and must integrate native live-document identity adoption/candidate rebuilding. Native editing and real VOA/terminal acceptance remain open.

### 2026-10-08 — native projection/navigation (05)

05 is implemented and archived on the isolated branch: safe structured rendering and document-anchored read/navigation now execute in module/TUI/SQLite tests. Nine focused projection fixtures and safety checks over all 83 synthetic sources cover the read seam. Native editing is deliberately disabled; 06 is unblocked and integrates draft/Visual/history/recovery through the 04/05 seams. Real VOA/terminal acceptance remains 10. Committed on 2026-10-08 on `feature/description-model` in the same worktree.


### 2026-10-09 — native Draft/Visual implementation (06)

06 is implemented, verified and archived in the same worktree, committed 2026-10-09. Structured editing, Visual formatting/selections, Opaque confirmations, local history, success-only identity adoption and Blocked recovery now have module/TUI/temporary-SQLite execution evidence. Five review rounds and final targeted rechecks leave no confirmed findings; full tests, affected race/vet, both builds and bounded draft fuzz pass. 07–09 are unblocked; 07 is next. Advanced structures/Code/syntax and real VOA/terminal acceptance remain in their owning tickets; no blanket completion of all 83 editor expectations is claimed.


### 2026-10-10 — List/Checkbox/Quote structure implementation (07)

07 is implemented, verified and archived in the same worktree, committed 2026-10-10. Five full independent review pairs and targeted final fix verification leave no confirmed findings. Dedicated structure/TUI/temporary-SQLite tests, ten independent corpus goldens, full tests, affected race/vet, both builds and bounded structure fuzz pass. Native list/quote operations preserve reading order, identities, checked state and meaningful metadata, or reject the specific unrepresentable operation atomically. Code-line origin survives endpoint swap. 08 is next; 08/09 and real VOA/terminal acceptance remain open in their owning tickets.

### 2026-10-10 — Code editing and terminal dispatch (08)

08 is implemented, verified after four independent bug/consistency review pairs and archived in the existing worktree; committed 2026-10-10. Exact Code whitespace/paste, Enter-confirmed typed marker creation, same-container exits, source-line editing/selection, Tab/literal-Tab, retab and settings now execute in Draft/TUI/config tests. All 38 focused non-null corpus Code edit goldens pass actual editor/prepared-save tests; full tests, affected race/vet, both builds and bounded source-preservation fuzz pass. Protected Code-child source gaps are handled without widening Opaque; ambiguous destructive line operations reject locally, whole-block removal remains confirmable. 09 is next. Real German/Windows/VOA acceptance remains 10.

### 2026-10-10 — Ticket 09 execution handoff

Inline syntax, Native Inline-Code and separate Link URL editing are implemented in the same isolated worktree. Forty independent edit HTML goldens across eleven focused corpus records now run through actual TUI dispatch/prepared-save, with additional origin/escape/history, URL, clipboard and temporary-SQLite adoption tests. Product rules remain unchanged. Ticket 10 owns integrated coverage reporting, tutorial and real VOA/German-layout/Windows acceptance; synthetic results do not certify those environments.


## Final acceptance and closure — 2026-10-19

Jan reports **10 PASS / 0 FAIL / 0 NOT ASSESSED — ACCEPT** for AT-01–AT-10, including Code Paste, actual OS Clipboard, German keyboard/AltGr paths, resize without losing drafts, and the bidirectional live Vikunja round-trip with in-app reload. The tutorial and steps 11/13 were already accepted. This final report supersedes earlier partial verdicts and outstanding manual checks above. The checklist's displayed base commit 059559f is historical; acceptance incorporates the subsequent fixes through 0120fbb and the tested working state, rather than claiming those fixes existed in 059559f.

All confirmed defects have focused regressions; affected full suites, race checks, vet and both entrypoint builds pass. Direct Link insertion, initially empty Link-text input (empty Enter uses URL), specific selection hints and visible prose Tab markers are included in the final reviewed feature slice. Existing Space+Y remains the accepted URL-copy binding; choosing a different lowercase binding is an optional future UX change, not an acceptance blocker. No further Rich-text implementation/acceptance subtask remains in this ticket. Technical fixtures/captures and supplemental verification remain in .scratch/description-rendering because automated tests consume those fixed paths; no fixture/data migration is part of closure.

Jan authorizes committing the remaining changes and closing the accepted feature.
