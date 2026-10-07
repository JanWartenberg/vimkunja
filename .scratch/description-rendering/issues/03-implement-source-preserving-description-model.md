# 03 — Source-preserving Description document model

**What to build:** Parse existing Description HTML into a UI-independent Native/Opaque/Blocked document and serialize it without losing accepted meaning or retained source bytes.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** in-progress

**Contract:** [spec.md](../spec.md), [decision/handoff ticket 02](02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../corpus.md), [83 synthetic fixtures](../corpus.json), [keybinding contract](../../../docs/keybindings.md).

## Scope

- Own source slices, stable node/atom identities, decoded text, semantic marks/attributes, logical block boundaries and document cursor anchors in one cohesive component. Keep parser/tree/source-index details out of TUI callers; choose the smallest package/interface needed for subsequent tickets.
- Implement the full accepted Native vocabulary for import/serialization, including Inline-Code, nested lists/Checkboxes, quotes and Links; implement all Opaque classifications and safe best-effort text for Blocked recovery. Editing operations and terminal layout belong to later tickets.
- Preserve the smallest reliably bounded Opaque ancestor byte-for-byte. Active content, duplicate attributes, comments, style, media and unknown structures must not execute, sanitize-and-save, or automatically block editable neighbors.
- Validate boundaries explicitly: x/net/html tree-parser success alone is insufficient. Handle unambiguous optional end tags, root inline grouping and forbidden document wrappers under the contract.
- Decode Code from original source slices, preserving leading LF, CRLF, Tabs, empty/final lines. Preserve Native hard-break-adjacent whitespace, meaningful empty blocks, safe attributes and imported mark variants/depth; normalize only permitted prose/source spelling.
- Implement one linear-time save-boundary validation of retained Opaque bytes, identity, order, nesting and intended parent positions, with support for deliberately removed atoms and explicit allowed parent conversions. Do not validate per keystroke.
- Consume the independently authored corpus from tests; do not regenerate expected HTML from this serializer. Missing editing capabilities remain explicitly unexecuted, not skipped as PASS.

## Acceptance and validation

- All 83 current fixtures execute classification and applicable ordinary-save/round-trip assertions. UUID requirements use the later save-candidate seam; no random ID is hardcoded as an HTML golden.
- Exact-code-import retains LF/CRLF/Tabs/final newline; malformed-html and document-wrappers reject ordinary save. Duplicate/unsafe inputs preserve their bounded atoms and allow surrounding Native content.
- Opaque identical atoms retain distinct identities; an injected swap or wrong parent fails validation despite identical bytes. Permitted parent conversion compares with the intended new parent.
- Parser/render-safe-text/serializer do not panic on focused malformed inputs and disclose no hidden active content. Add deterministic regressions for parser repairs and decode boundaries.
- Existing M2 display remains a read-only fallback; no Native mutation/save route is activated by this ticket.

**Focused coverage:** `basic-blocks`, `inline-boundaries`, `entities-and-whitespace`, `exact-code-import`, `root-block-grouping`, `valid-optional-end-tags`, `opaque-nesting`, `opaque-identical-atom-identity`, `mention-label`, `mention-id-fallback`, and all current classification/serialization records.

Run affected tests during implementation; before a commit run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The existing artifact probes are supplemental, not Native execution evidence. Required test commands and fixture mapping must be recorded in the completion comment.

## Boundaries

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. This ticket preparation does not start production implementation. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.

### 2026-10-06 — isolated implementation started

User requests beginning ticket 03 on a separate branch while main is being cleaned up. Worktree: `/workspace/vimkunja-richtext-03`; branch: `feature/description-model`. Only accepted Rich-text specification/ticket preparation is carried over from the uncommitted main working tree; unrelated cleanup stays on main.


### 2026-10-06 — first implementation slice and execution evidence

Added UI-independent `internal/description`: immutable original HTML; detached Native/Opaque snapshots and document-local identities; strict token/source boundaries; original-slice Code decoding; Native normalization and serialization; private Opaque slices with byte/order/path/parent-identity validation; explicit removal/movement intent; safe atom previews and Blocked recovery text. Duplicate attributes are scanned from original source because the current tokenizer silently deduplicates them. Reliably bounded unsupported Native shapes protect local ancestors instead of disabling neighbors.

`TestCorpusImportAndSerialization` executes classification for all 83 independently authored records, their 78 exact ordinary-serialization HTML goldens, rejection of the two Blocked records, and idempotent serialization for all 81 non-Blocked records. Three Checkbox records with no fixed HTML golden preserve current IDs at this model seam; UUID repair remains ticket 04. No corpus editing action is reported as executed. Focused tests additionally cover identical-atom swaps, wrong parent identity at unchanged paths, declared removals/reparenting, snapshot isolation, active-content recovery, image/comment/Mention/Table previews, original attribute decoding/reserved IDs, strict repair rejection and deep unknown input. The exact-code-import golden independently checks leading LF, CRLF, Tabs and final newline.

Validation: `go test ./internal/description`, `go test -race ./internal/description`, `go vet ./internal/description`, `go test ./...`, `go test ./.scratch/description-rendering/verification`, and a ten-second two-worker `FuzzParseAndSerialize` run pass. Both entrypoints build with `go build -buildvcs=false ./cmd/vimkunja ./cmd/vimkunja-tutorial`; ordinary VCS-stamped build is blocked by Git ownership checks in the root-owned worktree, not a compilation failure. No dependencies, SQLite schema, local database, TUI routes or master cleanup files changed.

Ticket remains **in-progress**: this is the authorized start and first tested model slice. Further structural/semantic edge-case review and the draft-anchor interface remain before declaring ticket 03 complete and handing it to 04/05. Real VOA/terminal evidence and integrated editing are still later-ticket work. No commit or merge was requested for this start.


### 2026-10-07 — review findings B fixed and re-reviewed

User requests fixing every finding in section B, adding missing unit tests, then reviewing again. Fixed all six reproduced findings: document-root recovery now visits children and retains useful visible text; paragraph-level cleanup does not descend into Code blocks and discard empty code wrappers/attributes; boolean reversed normalization applies only to ol; indirect nested Links protect the bounded outer Link while preserving editable neighbors; Opaque list projections separate entries; previews/recovery visibly escape terminal controls, including C0/C1/ESC/OSC sequences. Original HTML, Opaque bytes and decoded Native Code are not sanitized or changed for display. CRLF in safe display becomes one line break; standalone carriage returns are shown as escaped text.

Added six focused regression test groups in `internal/description/regression_test.go`, covering malformed recovery with actual expected text, exclusion of script/style/head/hidden metadata, empty and nonempty Code inside nested list/quote/Checkbox containers, element-specific attributes, nested Links through multiple marks and unknown ancestors, nested list boundaries, and control-bearing text/image/Mention labels with source-preservation checks. The six independent temporary review probes that failed before this change now pass; their coverage also exists permanently in the repository. The 83 independently authored corpus expectations remain unchanged.

Verification passes: `go test ./...`, `go test -race ./internal/description`, `go vet ./internal/description`, `go test ./.scratch/description-rendering/verification`, both entrypoint builds with `-buildvcs=false`, and a ten-second two-worker parser/serializer fuzz run (164110 executions). No test failure or new regression was found in the corrected areas on the renewed review.

Known section-C completion work remains: generated import paragraphs need stable nonzero IDs, cursor anchors need a working interface, and independent Native semantic round-trip validation needs completion. The current reparse checks Opaque preservation; it is not a complete semantic comparator for Native meaning. Ticket remains in-progress; no Native TUI or persistent save route is activated. No commit/merge requested.


### 2026-10-07 — authorized intermediate checkpoint

User requests committing the tested/reviewed implementation and all six fixes as an intermediate checkpoint, then completing the remaining ticket-03 work. This checkpoint deliberately keeps status in-progress. Scope carried into the commit includes the accepted specification and implementation-ticket preparation already present in this isolated worktree; main cleanup is not included.
