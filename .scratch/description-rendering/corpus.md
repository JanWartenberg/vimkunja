# Rich-text compatibility corpus — review draft

**Status:** draft, unreviewed; not acceptance evidence and not a completed corpus.

**Contract:** [spec.md](spec.md). **Ticket:** [02](issues/02-specify-native-rich-text-description-rendering-and-editing.md).

[corpus.json](corpus.json) contains focused, hand-authored Source HTML fixtures and independent proposed expectations. Stable fixture IDs below identify records in its `fixtures` array. All records inherit the file's synthetic provenance and unreviewed status. None is claimed to be VOA-generated or a minimized defect regression. The current M2 reader is deliberately lossy; its behavior is not the oracle for M3.

## How to review/use the records

- `source_html` is the exact input string after JSON decoding, including tabs, newlines, case, quotes, and Unicode. No indentation or final newline outside that decoded string is part of the input. Preserve it as UTF-8 when importing into a future harness.
- `classification` groups Native, mixed Native/Opaque, and Blocked cases. An all-Opaque nonempty document can be represented in the mixed tier with zero Native blocks; this draft currently uses mixed examples with Native neighbors.
- `projection.content_lines` proposes logical visible content, not an ANSI/pixel golden. Empty strings denote retained empty blocks/rows. `presentation` adds required semantics without prematurely fixing width, color, bracket style, separator glyph, or grid borders. `null` means no justified exact expectation yet, not empty content.
- `serialization.html` proposes an ordinary-save output before any listed edit. It is a representative compact canonical spelling, not a new requirement for attribute ordering or equivalent entity spelling. Semantic comparisons must allow the contract's permitted representation differences. `opaque_bytes` lists outermost exact source slices in document order; those slices remain byte-exact even within otherwise semantically compared output.
- `edits` describe separate scenarios starting from the original source unless an action explicitly continues the preceding scenario. `expected_html` describes the post-action draft's eventual successful-save serialization, not an automatic write on each keystroke. A `null` output must have explicit assertions and/or a recorded unresolved decision; never fill it by blindly running the implementation.
- `feedback`, `invariants`, and inherited assertions are requirements for the eventual harness, **not assertions that have already run**. Blocked fixtures inherit confirmation, cancel, save, and discard checks for both recovery flows. Recovery serialization is asserted only where the contract justifies its exact safe text.
- Changes to proposed expectations require a diff and the relevant contract decision. Once a reviewer approves a fixture, record its ID, approved fields, remaining exceptions, and reviewer/date in this document; do not silently mark the whole file approved.

No executable harness is added: its public seams are not yet agreed, and the parser/serializer has not been selected. JSON structural checks only establish that this artifact is readable and internally linked; they do not establish editor correctness. Existing Go tests protect the current application, not this future M3 contract.

## Traceability — agreed principles

Numbers refer to [Agreed principles](spec.md#agreed-principles). **Draft** means illustrative expected behavior exists but is unreviewed and unexecuted. **Partial** means additional cases or decisions are explicitly outstanding. Neither is a pass.

| Principle | Fixture IDs | Draft coverage / missing evidence |
| --- | --- | --- |
| 1 — HTML persistence, unchanged schema | `basic-blocks`, `external-save-conflict` | Draft; persistence integration evidence pending. |
| 2 — no raw HTML normal editing | `editor-delimiters`, `typed-versus-pasted-markers`, `opaque-node` | Draft; actual terminal projection pending. |
| 3 — structured M3 vs lossy M2 | `root-inline-fragment`, `opaque-edit-neighbors` | Draft; no use of M2 output as golden oracle. |
| 4 — preserve safe structures/attributes | `serialization-normalization`, `opaque-nesting`, `attribute-preservation` | Partial; safe-attribute carry/rejection variants pending. |
| 5 — readable unsupported/empty fallback | `opaque-nesting`, `unknown-node` | Partial; exact unknown whitespace and media/mention policies below. |
| 6 — semantic vs terminal style | `inline-marks`, `inline-boundaries` | Partial; capability/ASCII fallbacks need practical review. |
| 7 — logical block parsing/navigation | `basic-blocks`, `nested-lists`, `code-and-quotes` | Draft; executable navigation scenarios pending. |
| 8 — hybrid serialization | `serialization-normalization`, `opaque-edit-neighbors`, `table-mixed-content` | Draft; Native semantic/Opaque byte comparisons pending. |
| 9 — j/k vs gj/gk | `hard-breaks`, `layout-and-palette` | Partial; cursor examples and visual rows need review. |
| 10 — Checkbox state/IDs | `checkbox-lists`, `checkbox-duplicate-ids` | Partial; nested entries, missing-ID discard/failure, collision retry, and VOA markup pending. |
| 11 — H1–H3 | `basic-blocks`, `typed-versus-pasted-markers` | Partial; typed H2/H3 and conversion-toggle scenarios pending. |
| 12 — marks, distinct tag semantics/depth | `inline-marks`, `inline-boundaries`, `uniform-toggle` | Partial; repeated-layer toggle/removal ambiguity below. |
| 13 — nested lists, deferred indent/outdent | `nested-lists`, `list-editing` | Partial; reverse/deeper nesting and ordered attributes pending. |
| 14 — exact Code, overflow | `code-and-quotes` | Partial; standalone Code, overflow/scrollbar, paste and empty/final deletion scenarios pending. |
| 15 — quotes, prefixes/attributes | `code-and-quotes`, `empty-structures` | Partial; nested quote, wrapping/ASCII, entry/exit/merge examples pending. |
| 16 — atomic rule, x/dd | `basic-blocks` | Draft; typed noncanonical dash-count normalization pending. |
| 17 — Links and schemes | `links`, `link-import-edges`, `unsafe-attributes-and-urls` | Partial; complete allowlist/rejection variants, editing edges and clipboard failures pending. |
| 18 — Opaque preservation and Blocked recovery | `opaque-edit-neighbors`, `opaque-parent-conversion`, `malformed-html`, `unsafe-html` | Partial; validation failure, pending-input cancel, and all unsafe/malformed variants pending. |
| 19 — Tables | `table-mixed-content`, `table-irregular` | Partial; widths, styled/empty tables and before/after placement variants pending. |
| 20 — cross-block selections, complete atoms | `opaque-edit-neighbors`, `uniform-toggle` | Partial; cross-block selection geometry and multiple atoms pending. |
| 21 — Visual marks, uniform toggling, clear marks | `uniform-toggle`, `links`, `opaque-edit-neighbors` | Partial; other marks, repeated layers, and binding comfort pending. |
| 22 — V, viw/vaw, rectangular deferral | `basic-blocks`, `opaque-node` | **Gap:** dedicated selection/text-object scenarios absent; these inputs are candidates only. |
| 23 — split/merge, breaks and paste | `hard-breaks`, `typed-versus-pasted-markers`, `code-and-quotes` | Partial; compatibility boundaries and split/merge/quoted-break scenarios pending. |
| 24 — block commands without selection | `serialization-normalization`, `opaque-parent-conversion` | Draft; UI invocation evidence pending. |
| 25 — heading/paragraph bindings | `typed-versus-pasted-markers`, `serialization-normalization` | Partial; binding prototype and toggle-to-paragraph scenarios pending. |
| 26 — list Enter/exit | `list-editing` | Partial; numbered/Checkbox equivalents pending. |
| 27 — Checkbox x isolation | `checkbox-lists` | Draft; enclosing Task Done unchanged is an inherited assertion. |
| 28 — Space formatting palette | `layout-and-palette` | Partial; all actions/modes, timeout, compact menu and global Space+o need prototype evidence. |
| 29 — draft Undo/Redo | `opaque-edit-neighbors` | Partial; Native edits/history branching and exact multi-atom restoration pending. |
| 30 — external-save conflict | `external-save-conflict` | Draft; persistence seam must be agreed before executable tests. |
| 31 — repeated block-format toggle | `basic-blocks`, `empty-structures` | **Gap:** dedicated repeated-command scenarios absent; quote behavior conflict below. |
| 32 — dd and Visual d | `basic-blocks`, `code-and-quotes`, `opaque-edit-neighbors` | Partial; V/d and final Code/quote-unit deletion scenarios pending. |
| 33 — o/O in ordinary/Code/quote content | `code-and-quotes` | **Gap:** exact creation scenarios absent. |
| 34 — cursor/motions, editable/protected counts | `inline-boundaries`, `hard-breaks`, `opaque-node`, `layout-and-palette` | Partial; complete i/a/I/A, 0/^/_/$, gg/G, page/count and semantic-word scenarios pending. |
| 35 — paragraph-motion units/counts/aliases | `code-and-quotes`, `table-mixed-content` | Partial; exact `{`/`}` and ö/ä count scenarios pending. |
| 36 — ordinary Code modes/newlines | `code-and-quotes` | Partial; literal paste, Enter, Esc, and o/O scenarios pending. |

## Traceability — serialization, syntax, and support tiers

| Contract area | Fixture IDs | Outstanding work |
| --- | --- | --- |
| Empty source / top-level inline normalization | `empty-description`, `root-inline-fragment` | Review proposed source/output pairs. |
| Empty blocks/wrappers | `empty-structures`, `link-import-edges` | Imported entry-less list, empty Table, and all empty link/wrapper variants. |
| Escaping/entities/Unicode/collapsible whitespace | `entities-and-whitespace`, `serialization-normalization`, `hard-breaks`, `editor-delimiters` | Combining characters/graphemes, other noncollapsing whitespace, attribute escape corners. |
| *** combined marks / repeated layers | `editor-delimiters`, `inline-boundaries` | Separator glyph/layout; imported mark variants in disclosed syntax. |
| Backslash grammar | `editor-delimiters` | Enumerate every escapable character, parity/repeated backslashes, overlapping/mismatched pairs. |
| Typed structures vs literal paste | `typed-versus-pasted-markers`, `basic-blocks` | All list triggers and three/four-backtick timing variants. |
| Safe Native attributes / duplicate Native attributes | `serialization-normalization`, `duplicate-native-attributes` | Attribute carry rejection and unsafe/ambiguous Link attributes. |
| Styles/comments/duplicate Opaque attributes | `attribute-preservation` | Dangerous content inside these structures must still block entire Description. |
| Opaque word-motion unit / configurable boundaries | `opaque-node`, `table-mixed-content` | Colorless/multiline practical examples; config contract not chosen. |
| Unknown outermost atoms | `opaque-nesting`, `unknown-node` | Inline q example and known-child safe projection/whitespace review. |
| Images | `opaque-node` | alt/title/no-metadata fallbacks and active URLs in image source. |
| Mentions | **Gap** | Need VOA capture: exact attributes identifying label/id are not supplied by contract. Do not invent VOA markup. |
| Embeds | **Gap** | Resolve active-content conflict below before assigning Native/Opaque/Blocked expectations. |
| Task-level attachments | No Description fixture | Deliberately outside this corpus; not a new Description block. |
| Source-preserving technical evaluation | `attribute-preservation`, `opaque-nesting`, `malformed-html`, `table-mixed-content` | Byte offsets, strict repair detection, linear save validation; evaluate after expectation review. |
| Property/fuzz evidence | `malformed-html`, `unsafe-html`, `unsafe-attributes-and-urls`, `duplicate-native-attributes`, `document-wrappers` | These are deterministic seed candidates only. Agree public seams and add bounded CI fuzzing later; no property/fuzz run claimed. |
| Help/durable markup tutorial | No runtime fixture | Deliverable remains deferred until the interaction contract is accepted; do not add unimplemented bindings to runtime Help. |

## Risky compound cases

The initial contract-required combinations are drafted: quotes containing lists/Code (`code-and-quotes`), formatting inside Links (`links`), Opaque content between editable text (`opaque-edit-neighbors`), and Tables mixed with editable content (`table-mixed-content`). Additional targeted interactions include styles plus duplicate attributes/comments, nested unknown structures containing marks/images, and recovery plus external-save conflict as separate scenarios. This is not exhaustive compound coverage.

## Decisions/gaps not guessed

1. **Repeated-layer uniform toggling:** removing bold from `<strong><strong>text</strong></strong>` could remove every emphasis layer or only an editor mark layer. Uniform on/off wording and preservation of meaningful semantic depth do not settle that destructive boundary. Likewise, disclosed syntax does not yet distinguish imported b/strong or i/em variants. Review before extending `uniform-toggle`.
2. **Quote conversion:** the ticket says repeated `Space+q` adds a nested layer and also says reapplying removes the current layer; principle 31 says applicable same-type commands toggle to paragraph. No deterministic action sequence resolves those alternatives. Keep the existing policies unchanged and ask for the intended nesting/removal flow before writing a golden.
3. **Embeds vs active HTML:** the contract calls iframe/video/audio/embed/object Opaque but also blocks executable/active content. The safe-versus-active boundary (including iframe) is not defined. Require a decision and real samples; do not assume all media is safe or all media is Blocked.
4. **Unsafe neutral projection:** the exact safe hint/text retained for scripts, handlers, and unsafe URLs is not specified. `unsafe-html` and `unsafe-attributes-and-urls` intentionally have no exact projection/recovery golden.
5. **Empty/imported structure boundaries:** deleting a final list entry removes its container, but that does not settle an imported already-empty list. Strict malformed-fragment handling also needs explicit examples for optional closing tags and repair/relocation. Empty-table projection is unspecified.
6. **Inline Code:** the decision log mentions literal syntax inside inline Code, but the native editing set specifies Code blocks only. A code span needs an explicit support tier before a fixture can classify it.
7. **VOA provenance:** this draft has no captured VOA-generated examples, known VOA version, or minimized defect inputs. Existing hand-authored M2 tests are not proof of VOA output. Obtain versioned, anonymized representative exports (especially Checkbox wrappers/IDs, mentions, media, and tables) before claiming compatibility.
8. **Provisional visuals:** content lines are not reviewed visual goldens. Exact cursor anchoring/reflow, wrap-arrow/separator/bracket design, protected-marker selections, narrow prefixes, and footer threshold still need the user's practical review. Bracket/color configurability is agreed; no configuration schema is invented here.

## Next review slice

Start with `basic-blocks`, `inline-marks`, `inline-boundaries`, and `uniform-toggle`. Approve the straightforward source/semantic outputs, then resolve repeated-layer toggling and imported-variant disclosure. Expand edge cases after that review rather than treating synthetic coverage as acceptance. Review remaining rows in small slices; obtain VOA samples and resolve support-tier conflicts in parallel where possible.

**Review log:** no fixtures approved yet. **M3 execution evidence:** none.

**Draft validation:** JSON parses; all 34 IDs are unique and referenced by the matrix; all declared Opaque slices occur in the source and proposed ordinary-save output; null edit outputs have explicit assertions; files use LF. Existing `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial` pass. These checks do not approve the expectations or exercise any M3 behavior.
