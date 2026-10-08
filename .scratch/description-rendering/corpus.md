# Rich-text compatibility corpus — decision-synchronized examples

**Stand 2026-10-07:** 83 synthetic, independently authored records. Model classification, applicable ordinary serialization/semantic round trips and focused save candidates execute in internal/description; atomic Store/Detail save tests also execute; Native editor/terminal actions and real VOA captures remain unverified. Expected fixture HTML is unchanged.

**Contract:** [spec.md](spec.md). **Ticket:** [02](issues/02-specify-native-rich-text-description-rendering-and-editing.md). **Examples:** [corpus.json](corpus.json). **Checks:** [verification/boundaries_test.go](verification/boundaries_test.go).

## Reading the examples

- `source_html` is exact UTF-8 after JSON decoding. All records are synthetic; input/output expectations come from accepted rules, never from the parser/serializer being tested.
- `classification` is Native, mixed Native/Opaque, or Blocked. The mixed tier may contain zero Native nodes. Opaque protects the smallest reliably bounded region; larger blocking requires unreliable boundaries or the explicit fragment-wrapper restriction.
- `projection.content_lines` is logical visible content, not ANSI/pixel evidence. Hint wording and layout remain illustrative where stated. `null` means assertion-based or not yet an exact display expectation, never an empty result.
- `serialization.html` is ordinary-save output before edits; `opaque_bytes` lists exact outermost source slices in order, including duplicate identical atoms. The artifact check also requires every record to be referenced in the current traceability matrix. Allowed Native attribute ordering/entity/whitespace differences use semantic comparison. UUID outputs use assertions, not random fixed goldens.
- Each edit starts fresh from its record's source unless the action explicitly chains steps. `expected_html` is the eventual saved result, not an automatic write per key. A null output requires assertions. Editor mode/cursor/selection/history checks may have unchanged HTML.
- Product-rule acceptance, agent static expectation review, dependency probes and Native execution are distinct. The traceability matrix records specified examples; it does not turn them into passing Native tests.

## Explicit expectation corrections

`duplicate-native-attributes`, `unsafe-html`, `unsafe-attributes-and-urls`, `unsafe-isolated-and-opaque`, `unsafe-url-only`: change the obsolete blanket Blocked expectation to byte-preserved local Opaque protection. Input strings are unchanged. Ordinary saves retain these bounded atoms; internal edits reject locally; failed preservation checks reject save. Global Clear/Recover scenarios are removed from these local cases, because reliably bounded active content no longer blocks the whole Description. Actual Blocked fragment recovery stays in `malformed-html`/`document-wrappers`.

Other synchronized examples cover C05 final Code-line deletion (leaves empty Code), all-family mark-off and retained selection, literal import/paste/escapes, complete Quote-source retention, tabs, fillable empty lists, automatic minimal ID maintenance and accepted layout defaults. 39 new fixture records are manually authored; no implementation-generated goldens.

## Traceability — accepted decisions and technical checks

| Decision / risk | Focused example records | Status |
| --- | --- | --- |
| HTML persistence, semantic serialization, escaping (1–8) | `basic-blocks`, `entities-and-whitespace`, `serialization-normalization`, `root-inline-fragment`, `root-block-grouping`, `empty-description` | Model import/serialization verified; editor/terminal execution outstanding. |
| Checkbox identity/state, atomic repair (10/27, L07, S12/S13) | `checkbox-lists`, `checkbox-duplicate-ids`, `checkbox-id-reservation-atomicity`, `checkbox-middle-split` | Candidate repair and persistence verified; middle-split snapshot verified at save seam, editor split/terminal outstanding. |
| Marks, syntax variants, all-family off (12/21, E01–E03) | `inline-marks`, `inline-boundaries`, `uniform-toggle`, `triple-strong-off`, `all-mark-families-off` | Model import/serialization verified; editor/terminal execution outstanding. |
| Literal import/paste and escape grammar (E04/E06–E10) | `editor-delimiters`, `typed-versus-pasted-markers`, `literal-import-edit`, `syntax-escape-parity` | Model import/serialization verified; editor/terminal execution outstanding. |
| Native Inline-Code (S08) | `native-inline-code`, `inline-code-typed-literal` | Model import/serialization verified; editor/terminal execution outstanding. |
| Code creation, boundaries, o/O/dd (14/32/33/36, C05–C08) | `code-trigger-events`, `standalone-code-editing`, `code-exit-boundaries`, `code-exit-nested`, `code-exit-list-entry`, `exact-code-import` | Model import/serialization verified; editor/terminal execution outstanding. |
| Tabs/retab/indent, Code selection (C01–C04) | `code-tabs-retab`, `retaball-native-scope`, `code-indent-outdent`, `code-line-selection` | Model import/serialization verified; editor/terminal execution outstanding. |
| List moves, kinds, overlaps, atomicity (13, L01–L09/L12) | `nested-lists`, `list-indent-existing-children`, `list-outdent-reading-order`, `list-selection-atomicity`, `list-kind-preservation` | Model import/serialization verified; editor/terminal execution outstanding. |
| List splits/numbering/empty imports (26, L08/L10/L11, S06) | `list-editing`, `ordered-list-editing`, `ordered-middle-exit`, `checkbox-middle-split`, `empty-imported-ul`, `empty-imported-ol`, `empty-imported-checkbox` | Model import/serialization verified; editor/terminal execution outstanding. |
| Quotes: local/shared/atomic scope and shared references (15/31, Q01–Q04) | `code-and-quotes`, `quote-shared-source-split`, `quote-unique-attribute-split`, `quote-shared-wrap-atomic`, `empty-structures` | Model import/serialization verified; editor/terminal execution outstanding. |
| Links: label/URL independence, edges/schemes/copy (17, K01–K05) | `links`, `link-import-edges`, `empty-link-label-edit`, `cross-paragraph-links` | Model import/serialization verified; editor/terminal execution outstanding. |
| Rules and incremental dash history (16, E11) | `basic-blocks`, `typed-rule-dash-history` | Model import/serialization verified; editor/terminal execution outstanding. |
| Selection/motions/Unicode/history (9/20/22/29/34/35, E05/E13) | `native-selection-and-history`, `graphemes-and-delimiter-selection`, `code-line-selection`, `opaque-node`, `code-and-quotes` | Model import/serialization verified; editor/terminal execution outstanding. |
| Breaks, empty blocks and normal block commands (11/23–25) | `hard-breaks`, `paragraph-creation-and-split`, `empty-structures`, `typed-versus-pasted-markers` | Model import/serialization verified; editor/terminal execution outstanding. |
| Opaque bytes/identities/allowed parent changes (18/20/29, T06) | `opaque-edit-neighbors`, `opaque-nesting`, `attribute-preservation`, `opaque-parent-conversion`, `opaque-identical-atom-identity` | Model import/serialization verified; editor/terminal execution outstanding. |
| Smallest-region safety, duplicate attrs and media (S01–S05) | `duplicate-native-attributes`, `unsafe-html`, `unsafe-attributes-and-urls`, `unsafe-isolated-and-opaque`, `unsafe-url-only`, `embedded-media-neighbors`, `malformed-html`, `document-wrappers`, `valid-optional-end-tags` | Model import/serialization verified; editor/terminal execution outstanding. |
| Tables, images, unknown inner boundaries (19, S07/S09–S11) | `table-mixed-content`, `table-irregular`, `empty-table`, `image-fallbacks`, `image-empty-alt`, `unknown-node`, `unknown-inner-block-boundaries`, `opaque-inline-q` | Model import/serialization verified; editor/terminal execution outstanding. |
| Opaque Mentions: visible label/id, preserved bytes, skip-format, safe copy/paste and confirmed removal (18 / native subset) | `mention-label`, `mention-id-fallback` | Synthetic model bytes/label/id fallback verified; editor and real VOA captures outstanding. |
| Palette/reflow/config design (28, P01–P12/C02) | `layout-and-palette`, `code-line-selection`, `code-tabs-retab`, `opaque-node` | Model import/serialization verified; editor/terminal execution outstanding. |
| Conflict rejection and candidate retention (30, T07) | `external-save-conflict`, `checkbox-id-reservation-atomicity` | Atomic persistence and Detail/combined conflict retention verified on disposable DBs; native editing/terminal outstanding. |

Task-level Attachments remain outside Description scope. `mention-label` and `mention-id-fallback` now specify synthetic @label/@id inputs, neighbor edits, skipped formatting, safe copy/paste, internal-edit rejection and confirmed removal/Undo. Their data attributes are hand-authored input assumptions; local `mention-user`/label-or-id source is a lead, not captured generated-output evidence. Exact VOA/Tiptap serialization still needs versioned captures. HTTP/HTTPS/FTP host syntax and non-web scheme-specific URI checks need runtime validator examples; no DNS checks or new allowlist. Existing matrix coverage does not claim exhaustive URL/Unicode/attribute/fuzz coverage.


## Document-model verification — 2026-10-07

[Completed ticket 03](../../.archive/2026-10-07/description-rendering/issues/03-implement-source-preserving-description-model.md) supplies the model seam without changing the TUI or Store:
- [Model fixture tests](../../internal/description/document_test.go) execute classification for all 83 fixtures, 78 exact ordinary-save HTML goldens, two Blocked rejections and current-ID preservation for the three UUID-seam cases.
- [Independent semantic tests](../../internal/description/meaning_test.go) compare all 81 savable outputs with hand-authored expected HTML using the independent HTML tree parser; mutation-sensitivity cases cover meaningful Native dimensions, and 100 deterministic generated examples exercise combinations. Original-slice Code goldens separately certify exact LF/CRLF/Tabs/final newline.
- [Anchor tests](../../internal/description/anchor_test.go) check stable unique imported IDs, logical blocks/ancestors, UTF-8 boundaries, structural/empty/Opaque boundaries and detached data. [Regression tests](../../internal/description/regression_test.go) execute actual useful recovery text, local protection, preserved Code metadata and terminal-control escaping.
- Opaque tests retain identity, order, bytes and declared positions/parents, including identical atoms and explicit removals. Failed serialization never mutates source or supplied draft.
- Full tests, focused race/vet, builds with VCS stamping disabled and a bounded ten-second fuzz run pass. No fixture edit action, transactional save/UUID adoption, VOA capture or interactive native terminal layout is claimed executed.

## Atomic-save verification — 2026-10-07

[Completed ticket 04](../../.archive/2026-10-07/description-rendering/issues/04-implement-atomic-description-save.md) is implemented and committed on 2026-10-08. [Candidate tests](../../internal/description/save_test.go) execute the four focused Checkbox corpus inputs with independently authored deterministic UUID goldens, Opaque/source-ID reservation, collision/failure and stable repeat saves. The middle split is supplied as the accepted edited snapshot; this does not execute editor keys. [Store tests](../../internal/vimkunja/description_save_test.go) exercise two-handle remote changes after preparation, transactional conflict rejection, create/update/combined success and rollback, unchanged HTML/Done, concurrent writers and failed COMMIT with an external reader. [Detail tests](../../internal/tui/description_save_test.go) cover baseline capture, conflict/draft retention, committed adoption, late edits and post-commit history-read errors. Full tests, affected race/vet and both builds pass; no live DB or real terminal/VOA evidence is claimed. Ticket 06 owns adopting committed node identities into its live editor and rebuilding future candidates.

## Historical technical verification — 2026-10-04

- Artifact checks: 83 unique records; null outputs justified; Blocked records have no ordinary-save golden; retained Opaque slices occur byte-for-byte in source/output in order and multiplicity.
- Existing `x/net/html`: copied Raw tokens reconstruct original UTF-8, mixed quote/case/duplicate attributes and CRLF. Token/Text normalizes CRLF to LF. Native Code therefore needs decoding from original text slices. Tree parsing repairs malformed/unexpectedly nested HTML without an error; strict acceptance cannot rely on ParseFragment success alone.
- Store regression tests: title-only/unrelated Native text changes retain exact supplied HTML bytes, CRLF, comments, unusual attributes and Task Done. Failure after UPDATE rolls title/Description back and leaves the caller draft unchanged. Tests use disposable databases.
- Save conflict gap: current UpdateTask WHERE matches only Task ID; no baseline Description comparison exists. Persisted Undo conflict checks do not protect a stale editor save. Required M3 compare-and-save must read/check/write in one transaction and adopt generated IDs only after success, including combined Task/Comment saves.
- Synthetic terminal decoding and current TUI input: UTF-8 ü/Ü/ö/ä, bracket characters, Ctrl+V/Q/B, Tab and four backticks followed by Enter decode distinctly. Current plain-text Insert also preserves one/two/four-backtick Text-event chunks and German characters in focused regression tests. This does not capture German AltGr, Windows interception or real dead-key events. Current Insert Ctrl+V calls Paste; current Paste normalization changes CRLF before context dispatch. Native Code needs dedicated literal-prefix handling and context-sensitive paste before normalization.

Commands (from repository root):

```bash
go test -v ./.scratch/description-rendering/verification
```

```bash
go test ./internal/vimkunja -run 'TestUpdateTask(PreservesDescriptionBytesAndDone|FailureRollsBackDescriptionAndRetainsDraft)$'
```

Full `go test ./... ./.scratch/description-rendering/verification` passes; both entrypoints build with `go build -buildvcs=false ./cmd/vimkunja ./cmd/vimkunja-tutorial`. VCS stamping is disabled only because this shared checkout triggers Git safe-directory checks; compilation is verified.

Remaining execution work: Native model/parser/serializer/editor and identity-aware round-trip validation; UUID candidate/failure/conflict tests at that future seam; bounded property/fuzz runs; native wide/narrow/short-pane reflow; real Windows/German layout input capture. VOA captures remain absent: local upstream commit `82ee780161c6ed250cdb1fc4d2952b72464cb724`, frontend package 2.3.0/Tiptap 3.27.1 is available as source, but no installed frontend runtime/editor dependencies are present. No source-backed synthetic example is labeled a real getHTML capture.

## Next work

No further product card confirmation is required. Continue with ticket 05 native projection/navigation using the completed model/save seams; do not substitute the existing lossy reader for its serializer. Model tests pass, while editor/terminal execution and versioned VOA captures remain prerequisites for claiming full rich-text compatibility.

## Follow-up consistency cleanup — 2026-10-04

Corrected stale references to future text objects and missing list/triple-strong examples. Added two independently authored synthetic Mention fixtures under the already accepted @label/@id and Opaque rules. No existing HTML input/output golden or product decision was changed; only stale presentation/assertion notes were corrected. Native execution and real VOA provenance remain outstanding.

## Historical review log — superseded status snapshots

Earlier question counts, blanket safety classifications and proposed outputs below describe prior checkpoints. Current policy and examples are the synchronized records/matrix above; historical approvals retain their stated scope.

### Non-semantic maintenance pass

Extended the synthetic draft from 34 to 42 fixtures without approving expectations or resolving the flagged decisions:

- Native `viw`, blockwise/characterwise deletion, draft Undo/Redo; paragraph o/O, split/merge and H2 repeated toggle.
- Standalone Code o/O, Enter, literal paste, line deletion and Esc; empty Code/quote deletion; numbered-list Enter/exit.
- Image fallback precedence and restoration among multiple atoms; opaque inline q; isolated dangerous URL and active content inside an Opaque ancestor.
- Typed H2/H3 and preservation-validation failure assertions; Link insertion edges, clipboard success/failure and accepted/rejected scheme scenarios; clarified that literal HTML-looking user text is not prohibited Source HTML disclosure.

All additions remain hand-authored, unreviewed proposals. No VOA samples, harness, parser selection, runtime bindings, or native implementation were added. Original review order and unresolved decisions remain unchanged.

## Agent consistency review — decision compression

All 42 records were read against `spec.md`, the ticket, and `docs/keybindings.md`. This is a static specification review, not execution evidence, user approval, or VOA compatibility certification. The table and recommendations below are the pre-feedback review snapshot; subsequent accepted changes are recorded in the user-feedback log. No fixture output or settled policy was changed during the initial preparation. The three-example walkthrough is `vimkunja-corpus-review.html` (historisches lokales HTML-Artefakt, nicht im Repository enthalten); the original 42-card file remains unchanged.

The user authorized preparing representative decisions and propagating later feedback to equivalent cases. Future acceptance may therefore be recorded by rule/family instead of requiring 42 individual approvals. Record the accepted rule, affected IDs/fields, and exceptions; unresolved safety, provenance, and practical-layout requirements do not inherit approval from unrelated examples.

### Per-fixture disposition

**D — derivable:** no additional product decision found for the shown source/output/actions; ordinary static consistency only. Missing variants in the traceability matrix still apply. **Q — decision-dependent:** an ambiguity affects some fields/actions or their extension. **P — practical validation:** behavior is specified but presentation/interaction needs a real example. **E — external evidence:** capture or technical validation is essential. Multiple labels are intentional.

| Fixture | Disposition | Review finding / limit |
| --- | --- | --- |
| `basic-blocks` | D | Empty block and rule x/dd outputs follow explicit rules; x turning a rule into `--` is intentionally surprising, not an error. |
| `inline-marks` | D/P | strike→s and clear marks agree; limited-terminal style/disclosure still needs review. |
| `inline-boundaries` | Q/P | Unrelated β edit preserves semantics; imported variant disclosure and repeated layers remain unresolved. |
| `uniform-toggle` | D/Q | Single-layer on/off agrees; do not extrapolate its output to repeated layers yet. |
| `editor-delimiters` | D/Q | Listed escapes/combined marks agree; literal imported delimiters, overlap grammar, and incremental parse behavior are not covered. |
| `typed-versus-pasted-markers` | D/Q | Listed heading/HTML paste results agree; paste's inline-format interpretation and trigger timing need precision. |
| `hard-breaks` | D/P | Serialized break edit agrees; final-br visible rows must not be claimed HTML-pixel-equivalent without review. |
| `entities-and-whitespace` | D | Collapse/escape/NBSP output follows the contract; grapheme navigation is separate missing coverage. |
| `serialization-normalization` | D | Safe attribute carry and rejection rule agree; attribute-specific safety is not proven. |
| `root-inline-fragment` | D | Explicit paragraph normalization; mixed inline/block root boundaries remain uncovered. |
| `empty-description` | D | No edit creates no persisted placeholder. |
| `empty-structures` | D/Q | Listed empty blocks and explicit deletions agree; imported entry-less containers and final nonempty Code-line deletion are unresolved. |
| `nested-lists` | D/Q/E | Text edit preserves nesting; semantic numbering attributes and VOA paragraph wrappers need coverage. |
| `list-editing` | D/Q | End/empty Enter agrees; middle-entry split and cross-kind merge rules remain missing. |
| `checkbox-lists` | D/E | Assertions are appropriate for random IDs; nested VOA structure, failed save/discard and ID scope remain untested. |
| `checkbox-duplicate-ids` | D/E | Current chat supersedes confirm/cancel: automatic minimal repair, preserving first occurrence; draft assertions updated. Collision, failure/conflict and opaque-ID interactions still need evidence. |
| `code-and-quotes` | D/Q/P | Listed blank Code-line deletion preserves exact text; quote command conflict and compound navigation/layout remain open. |
| `links` | D/Q/P | Listed edit outputs follow link policy; new URL feedback/attribute preservation is not a complete URL validator specification. |
| `link-import-edges` | Q | Untouched output follows policy; editing a displayed URL for an actually empty label lacks an explicit rule. Unknown-scheme safety is not exhaustive. |
| `opaque-node` | D/Q/P | Unrelated text edit is byte-preserving; dd target (atom vs parent) and visible atomic extent need clarification. |
| `opaque-edit-neighbors` | D/Q/P | Assertions fit preservation/confirmation; crossing-selection geometry and delimiter exclusion are not yet specified precisely. |
| `opaque-nesting` | D/Q/P | Source slices and unrelated edit agree; safe child projection whitespace/boundaries remain unreviewed. |
| `attribute-preservation` | D/E | Exact bytes and visible comments agree; style opacity cannot exempt dangerous descendant content. |
| `opaque-parent-conversion` | D/Q | Confirm/reject cases agree; validation must allow the explicitly intended parent change, not require identical parent tags. |
| `table-mixed-content` | D/P/E | Exact table retention agrees; grid/cell text boundaries and navigation/scroll need evidence. |
| `table-irregular` | D/P | Row fallback and canceled deletion agree; empty/styled table variants are missing. |
| `unknown-node` | D | Explicit visible unsupported hint and byte-preserving neighbor edit. |
| `malformed-html` | D/Q/E | Truncated strong makes the shown input malformed; omitted optional end tags are a different, undecided acceptance boundary. |
| `duplicate-native-attributes` | D | Block/recover output follows explicit duplicate-attribute policy. |
| `document-wrappers` | D | Block/recover output follows explicit fragment restriction. |
| `unsafe-html` | Q/E | Blocking agrees; safe hint/recovery text must not be guessed. |
| `unsafe-attributes-and-urls` | Q/E | Blocking agrees; exact neutral label/recovery policy and independent safety coverage pending. |
| `layout-and-palette` | P | Explicitly provisional; static lines cannot validate reflow, cursor anchoring, footer threshold or motion comfort. |
| `native-selection-and-history` | D/Q | Listed semantic deletion/history results agree; viw on disclosed syntax and formatting-command selection lifetime need precise mapping. |
| `paragraph-creation-and-split` | D | Listed outputs agree; no invented whitespace on merge is a deliberate policy. |
| `standalone-code-editing` | D/Q/P | Listed literal text/line outputs agree; final nonempty line deletion and Code entry/exit/trigger timing remain open. |
| `ordered-list-editing` | D | Listed outputs agree; continuation after a middle list exit and start/value/reversed semantics remain uncovered. |
| `image-fallbacks` | D/P | Listed alt/title precedence, removal and restoration agree; empty-alt fallback and adjacent atom readability need examples. |
| `opaque-inline-q` | D/P | Unsupported atomic q follows explicit scope; quotation cues are not a reviewed visual golden. |
| `unsafe-isolated-and-opaque` | Q/E | Handler inside opaque ancestry must block; recovery text is intentionally absent. |
| `unsafe-url-only` | Q/E | Dangerous URL alone blocks; exact neutral label/recovery text remains absent. |
| `external-save-conflict` | D/E | Conflict assertions agree; atomic persistence/concurrency evidence still required. |

### Three representative user decisions

1. **Formatting means on/off, not HTML surgery.** Compare mixed bold/plain → all bold → all plain with an imported double-strong word. Recommendation: explicit Bold-off removes all bold-family layers in the selection; unrelated edits retain imported variants/depth. This is a proposed resolution, not an accepted change. Keep imported literal `**text**` literal by escaping it in disclosed editor syntax; do not normalize it into a mark merely on focus/save. Exact b/strong and i/em disclosure still needs a separate presentation decision.
2. **Protected content is visibly atomic.** Select native text on both sides of an image, apply bold, then delete the image itself. Already agreed: formatting skips the image; complete image deletion confirms; Undo restores bytes. Ask whether these protection/confirmation expectations match the user's mental model. This does not decide whether dd from native text in that paragraph deletes the whole paragraph or how delimiter characters participate in formatting selections.
3. **A repeated quote command must have one meaning.** Plain paragraph → quoted paragraph → second Space+q. Recommendation: second invocation removes the applicable quote layer, preserving children; reserve explicit nesting creation for a later separately chosen action. The existing ticket also promises nesting on repeat, so this requires an explicit decision before any golden or contract fix.

### Additional surprises / blockers not covered by three approvals

- `x` toggles a Checkbox anywhere in its entry rather than deleting the character; elsewhere it deletes, and on a rule it changes structure. Show contextual hints and test the conflict with prose editing; no policy change assumed.
- The same `Space+u`, `Space+0`, `Space+l`, and `u` mean different things in Normal/Visual/detail contexts. The actual mode/focus must be visible; runtime dispatch and palette comfort are not approved by HTML examples.
- Editable delimiters count in character motions, but the contract also describes semantic word motions across marks. Define whether `viw` includes delimiters and whether selecting/deleting only one delimiter changes a mark or creates literal text. Avoid a formatter that formats its own presentation syntax.
- Consuming a Code trigger at the third backtick makes a fourth backtick event ambiguous. Rules similarly convert at three dashes before the user finishes four. Specify incremental event sequences, not only final strings. Paste explicitly avoids block conversion but inline delimiter interpretation is not equally explicit.
- Imported empty-link URL display is not stored label text. Typing into that display, URL edits, and copy must not silently invent or duplicate a label. A trailing br may create an editor row not shown equivalently by a web renderer.
- Native attrs may become unsafe or meaningless after conversion; define rejection by attribute/target, not generic copying. Ordered `start`, `reversed`, and li `value` must affect displayed numbers, not merely survive hidden.
- Outermost opaque preservation must not hide unsafe descendants. Media opacity versus active content, unsafe neutral recovery, inline Code support, valid omitted end tags, and imported empty containers remain independent decisions.
- Hidden IDs in preserved opaque structures may interact with generated Checkbox-ID uniqueness. Specify uniqueness scope/collision handling without rewriting retained atoms. Save failures must not leave half-applied deduplication or persisted changes.
- Exact Code semantics need leading newline/CRLF/NUL examples and a stated pre/code import boundary. The shown Code fixtures do not settle HTML preprocessing versus exact decoded text, or deletion of the final nonempty line.
- A save-time preservation check must distinguish authorized parent conversion, split, move and deletion from accidental displacement. Identity cannot be inferred from identical opaque bytes alone. These are test requirements, not a selected parser design.

### Upstream source cross-check (not a captured VOA fixture)

Read-only cross-check of local upstream checkout `go-vikunja` (external to this repository) at commit `82ee78016`: `frontend/src/components/input/editor/TipTap.vue` configures the documented seven Link schemes, non-inclusive Links, nested Checkbox lists, inline Code via StarterKit/toolbar, and custom `mention-user` parsing/rendering. Its paste handler converts Markdown-like text, unlike Vimkunja's deliberately literal block-paste policy. `taskItemWithId.ts` uses `nanoid(8)` and automatically repairs duplicate IDs; Vimkunja retains its UUID-on-successful-save generation policy; the current chat aligns duplicate repair with automatic minimal maintenance. Existing unique imported IDs remain valid. `blockquoteWithCommentId.ts` preserves positive reply IDs and uses plain quotes outside Comments.

This narrows where to capture real examples; it supplies neither rendered/getHTML output, deployed VOA version, nor executed upstream tests. Mentions now have a source-based lead (`mention-user`), not a captured fixture. No dependency changes or native M3 harness were added.

### User feedback — representative rules

**Reviewer:** user, in this session. **Scope:** rule/family approval, not full corpus acceptance or execution evidence. No reliable calendar date was supplied with the feedback; this session is the provenance.

- **Accepted: explicit Bold-off removes all selected bold-family layers.** Double and triple strong become plain in one toggle; other marks and unselected text remain unchanged. Applied to `spec.md`, ticket 02, `uniform-toggle`'s unresolved note, and a new `inline-boundaries` double-strong removal edit/output. The unrelated β-edit/ordinary-save variants remain unchanged. Triple-strong input coverage is still to be added. Repeated-layer removal for other mark families and imported variant disclosure are not approved by this answer.
- **Accepted: formatting skips opaque content; image deletion confirms; Undo restores bytes/position.** Applies to the relevant protection, confirmation/cancel, and Undo fields in `opaque-edit-neighbors`, `opaque-node`, and `image-fallbacks`, and to equivalent Opaque structures under the existing global rule. This is semantic approval, not approval of final highlighting/layout or every fixture field.
- **Accepted intent: whole-block deletion includes inline images.** `dd` or blockwise `V d` deletes the complete logical block(s), with one confirmation if Opaque content is included, and whole-operation Undo. Added confirm/cancel/Undo outputs for `opaque-edit-neighbors`; its image-only removal now explicitly uses x. User said "whole line"; logical-block scope follows the existing V/dd contract and was explained in the response. Wrapped-screen-row deletion is not assumed. x retains atom-only scope; standalone Opaque blocks are themselves complete blocks.
- **Accepted: Space+q toggles quotation, not nesting.** Removed the contradictory repeated-command nesting promise from the ticket and clarified the contract. Quote-depth increase/decrease require separate commands. The initial `V >/<` proposal was subsequently superseded by the accepted Space-led structure palette below. Multi-child/attribute-preserving unwrapping has no invented output golden.

**Before → after expectation changes:** double-strong removal was previously unspecified and now produces plain selected text. Whole-paragraph image deletion was previously ambiguous on atom focus and now removes/restores the complete paragraph after confirmation. Original source strings, ordinary-save outputs, byte-preserved atoms, and unrelated edit outputs were not regenerated or changed. No other unreviewed expectation is implicitly approved.

- **Accepted V1 workflow: block selection then structure palette.** V starts Blockwise Visual Mode (`VISUAL BLOCKS`); ordinary motions extend the selection and d deletes it. Space then opens the structure palette: q/w remove/add quote depth, k/l decrease/increase list depth, x/c unindent/indent Code. Adjacent key pairs apply only after Space, preserving ordinary Visual motions. Characterwise v keeps the inline palette, including Space+l for Links; blockwise V Space+l increases list depth. The previous list-indent deferral is explicitly superseded for V1 design, but no structural output golden or runtime implementation is accepted. Configurable Code indentation width is intended; default, allowed values, tabs and config location remain undecided. Added to spec/keybinding contract and glossary; no new fixtures invented before parent/sibling/boundary semantics are settled.

- **Accepted: single-entry list indent follows structural convention.** Selecting B in sibling entries A/B/C and increasing list depth makes B a child of A. Without a previous sibling, reject without mutation and provide a clear hint. Do not synthesize an empty parent or substitute CSS-only indentation. The user explicitly preferred this convention over unsupported flexibility without a concrete use case. This approves the rule/example, not missing existing-child, multi-selection, mixed-kind, or outdent outcomes.

- **Accepted: parent outdent preserves the subtree.** Children move with the outdented parent, but remain independently selectable for indent/outdent. This does not settle destination/order, remaining siblings, overlapping parent/child selections, or the outermost-list boundary.

### Questions V2 checkpoint — user feedback

**Answered:** L01–L12, Q01–Q04, C01 (17 of 75 cards). **Next:** C02 in `explain-vimkunja-open-questions-v2.html` (historisches lokales HTML-Artefakt, nicht im Repository enthalten) (58 remaining). V1 stays unchanged; V2 uses stable IDs, filters previously answered cards from display/export, and can carry forward matching unanswered browser drafts without changing the V1 store. One copyable aggregate remains available.

| Cards | Accepted rule / user amendment |
| --- | --- |
| L01 | Outdent immediately after former parent; following former siblings become children of the moved entry, preserving reading order. |
| L02 | Indent subtree after receiving parent's existing children; preserve them and their order. |
| L03 | Move selected contiguous siblings together, without chaining one under another. |
| L04 | Highest selected subtrees move once. **User amendment:** if A/B are selected and A has no unselected previous receiving sibling, abort the whole action with footer warning; never move only B or use selected A as the recipient. |
| L05 | After overlap normalization require one common sibling range; unsupported mixed-depth selection rejects entirely. |
| L06 | Preserve ordered/unordered/Checkbox kind, using valid matching containers; no silent conversion. |
| L07 | Preserve moved Checkbox IDs/checked states, recompute summaries, never alter Task Done. |
| L08 | Preserve/render explicit start/reversed/li value; implicit numbers follow destination; do not secretly change destination numbering to hold former displayed numbers. |
| L09 | Root-level outdent rejects unchanged; explicit list→paragraph conversion stays separate. |
| L10 | **User amendment, recommendation rejected:** Enter inside AlphaBeta creates `Alpha`, then `Beta` with the original children under **Beta**. New entries must not copy existing IDs. |
| L11 | Empty middle list exit splits preceding/following lists around an empty paragraph; preserve order and safe explicit numbered continuation. |
| L12 | Automatic merge only same compatible kind/structural position/attributes; otherwise reject unchanged with hint. |
| Q01 | Remove only selected quote children, splitting container as necessary; neighbors remain quoted. |
| Q02 | One shared new quote for selected contiguous siblings. Verified shortcut is **V → motions → Space+w**; reduction is **Space+q** after the same palette, not `V >/<`. |
| Q03 | **User amendment, rejection recommendation superseded:** allow quote-layer removal with metadata loss only after `Zitat-Metadaten würden gelöscht werden. Bitte Löschung bestätigen.` Interpreted in the card's unwrap context: keep children, discard only removed-layer metadata; cancel unchanged and Undo restores it. Ordinary content deletion retains its existing scope. Retained metadata must not be silently duplicated during split. |
| Q04 | If any selected block lacks a reducible quote layer, reject the whole operation unchanged. |
| C01 | Default four spaces; configurable two or four; explicit indent adds one step without other normalization. Setting name/location, tabs and selection semantics remain open. |

Applied to the spec and keybinding contract. Existing source strings and ordinary-save outputs were not regenerated, no new fixture is claimed, and no harness/runtime implementation or M3 evidence was added. The old per-fixture table and recommendation snapshots above remain historical, not an alternative current policy. Accepted structure examples need dedicated fixture coverage before implementation.

**Still open:** safety/support tiers, imported empty structures and valid HTML import boundaries, literal imported syntax/inline paste and trigger timing, selection/motion geometry, structure-operation semantics, VOA provenance, practical layout, and M3 tests. The preparatory HTML is a historical proposal; these confirmed rules supersede its question status. **M3 execution evidence:** none.

**Draft validation:** JSON parses; all 42 IDs are unique and referenced by the matrix; all declared Opaque slices occur in the source and proposed ordinary-save output; null edit outputs have explicit assertions; files use LF. `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial` passed at the original draft checkpoint; this artifact-only pass did not rerun them. These checks do not approve the expectations or exercise any M3 behavior.

### Current chat — automatic Checkbox-ID maintenance

The user explicitly requests the corresponding update after reviewing local `taskItemWithId.ts` and `TipTap.vue`: valid unique IDs stay, missing IDs are filled and only later duplicate occurrences receive fresh IDs. No whole-list reset or pure-ID confirmation. Text, order, nesting, checked state, other meaningful metadata and Opaque bytes remain preserved. Actual content/metadata loss still requires confirmation.

**Affected expectation fields:** `checkbox-duplicate-ids.serialization.assertions`, its automatic-save and failed-save/conflict edit assertions, `feedback`, and preservation invariants. Source HTML, projection and random-ID-independent null outputs remain unchanged; ordinary entry-duplication behavior remains unchanged. This approves the repair rule and its equivalent assertions, not the entire fixture/corpus or any implementation. No M3 tests or new VOA capture ran. UUID collisions, Opaque IDs and retry/stability cases still need coverage.

### Partial remaining-default acceptance — cards 1–7

User explicitly accepts cards 1–7 of `explain-description-restpunkte-2026-10-03.html` (historisches lokales HTML-Artefakt, nicht im Repository enthalten), including their supplementary defaults; cards 8–16 are unreviewed. The spec section **Accepted remaining-default cards 1–7** is the consolidated contract. This approves product rules, not all fixture fields. Native/Opaque/Blocked expectations for active content and duplicate attributes now need targeted synchronization: older blanket-Blocked safety fixtures are historical and must not be treated as current goldens. Prepare independent Tab/outdent, selection, empty-list and literal/Inline-Code examples. No fixture outputs are regenerated or approved in this pass, and no M3 evidence is claimed.

### Reduced critical review — noncritical defaults adopted

The noncritical rest and, on 2026-10-04, the final Quote-source split rule are accepted. Both retained Quote fragments preserve the same known `data-comment-id`/`cite` reference without an extra confirmation for repeating it; genuinely unique IDs/unknown attributes still require lossless allocation, and actual metadata loss retains Q03 confirmation. Upstream `blockquoteWithCommentId.ts` identifies `data-comment-id` as the original comment reference, not a unique Quote identity. Existing fixture outputs remain untouched; a targeted split example and preservation assertions still need corpus synchronization and tests. No blanket golden/evidence approval or runtime authorization is inferred.
