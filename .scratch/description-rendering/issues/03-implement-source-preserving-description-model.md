# 03 — Source-preserving Description document model

**What to build:** Parse existing Description HTML into a UI-independent Native/Opaque/Blocked document and serialize it without losing accepted meaning or retained source bytes.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** ready-for-agent

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
