# 07 — List, Checkbox and Quote structure editing

**What to build:** Apply the accepted list/quote subtree operations atomically, preserving order, state, known source references and meaningful metadata.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** ready-for-agent

**Blocked by:** None

**Dependency satisfied:** [06 — Draft/Visual/history](../../../.archive/2026-10-09/description-rendering/issues/06-implement-description-draft-editor-and-visual-modes.md), completed 2026-10-09; committed 2026-10-09.

**Contract:** [spec.md](../spec.md), [decision/handoff ticket 02](02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../corpus.md), [83 synthetic fixtures](../corpus.json), [keybinding contract](../../../docs/keybindings.md).

## Scope

- Implement Normal Space+u/n/x conversions and Checkbox x with draft progress recomputation; Task Done is independent. Imported entry-less lists keep nonpersisted fill targets; canceled input creates no li, actual text creates an entry, deleting the last real entry removes its container.
- Implement L01–L12 through V Space+k/l: highest-subtree overlap reduction, contiguous siblings/common parent, unselected previous sibling, existing-child append, no root outdent and preserved list kind. Reject invalid selections before any mutation.
- L01 moves immediately after the former parent; following former siblings become children after existing children, preserving reading order. L03 moves selected siblings as siblings, never a chain.
- Preserve start/reversed/li value semantics and destination implicit numbering. Empty middle entry exits into split lists with a paragraph and explicit continuation.
- Enter at Alpha|Beta makes adjacent entries; children go to Beta. Alpha retains ID/checked state, Beta starts unchecked and gains a save-assigned UUID through ticket 04. Allocate other attributes losslessly or reject only the split.
- Normal Space+q toggles locally; V Space+w wraps siblings in one common container, Space+q removes the selected children’s applicable layer. Q04 rejects mixed reductions atomically.
- Split quotes around B while A/C retain their same cite/data-comment-id references without duplicate-reference confirmation. Do not copy true unique IDs/unknown attributes arbitrarily; reject unclear retained-fragment allocation.
- Apply Q03 metadata-loss confirmation only to actual removed-layer metadata, retain child content, cancel unchanged and Undo restores all. Parent changes containing Opaque obey the general confirmation/validation contract.

## Acceptance and validation

- Execute dedicated list/quote fixtures and all L01–L12/Q01–Q04 outcomes; assert tree structure, reading order, list kind, numbering, IDs and checked states, not just rendered indentation.
- User-reviewed L01 and L10 results are mandatory: following siblings reparent on outdent, and split children attach to unchecked Beta.
- Both retained quote fragments reference original comment 42; genuine unique/unknown allocation failure rejects only the split. Whole-layer metadata loss prompts once and Undo restores it.
- Add focused Checkbox move/overlap regressions if corpus coverage is insufficient; never generate goldens from implementation output. No operation changes Task Done or silently rewrites Opaque.

**Focused coverage:** `list-indent-existing-children`, `list-outdent-reading-order`, `list-selection-atomicity`, `list-kind-preservation`, `ordered-middle-exit`, `checkbox-middle-split`, `empty-imported-ul`, `empty-imported-ol`, `empty-imported-checkbox`, `quote-shared-source-split`, `quote-unique-attribute-split`, `quote-shared-wrap-atomic`.

Run affected tests during implementation; before a commit run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The existing artifact probes are supplemental, not Native execution evidence. Required test commands and fixture mapping must be recorded in the completion comment.

## Boundaries

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. This ticket preparation does not start production implementation. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.
