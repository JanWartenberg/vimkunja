# 07 — List, Checkbox and Quote structure editing

**What to build:** Apply the accepted list/quote subtree operations atomically, preserving order, state, known source references and meaningful metadata.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** completed

**Completed:** 2026-10-10 (committed)

**Blocked by:** None

**Dependency satisfied:** [06 — Draft/Visual/history](../../../2026-10-09/description-rendering/issues/06-implement-description-draft-editor-and-visual-modes.md), completed 2026-10-09; committed 2026-10-09.

**Contract:** [spec.md](../../../../.scratch/description-rendering/spec.md), [decision/handoff ticket 02](../../../../.scratch/description-rendering/issues/02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../../../../.scratch/description-rendering/corpus.md), [83 synthetic fixtures](../../../../.scratch/description-rendering/corpus.json), [keybinding contract](../../../../docs/keybindings.md).

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

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. Production implementation was explicitly authorized through implement-and-verify. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.


### 2026-10-10 — implementation and verification complete

Implemented on `feature/description-model` in `/workspace/vimkunja-richtext-03`, based on committed 06 (`f61e385`); these 07 changes are committed on 2026-10-10. The interrupted session was resumed from its artifacts and final review findings, without restarting implementation or the five-round review budget.

Native Draft actions implement local list-kind conversion and Checkbox toggling, subtree indent/outdent, existing-child append, highest-subtree overlap reduction, whole-selection validation, root rejection, compatible entry merge, Enter/o/O/dd routing and empty-list fill targets. Entry split preserves Alpha state/ID, starts Beta unchecked, moves existing children to Beta and provides a new own label target even at end-of-text. Empty exits retain original flow children and their metadata. Numbered continuation honors start/reversed/value, including implicit reversed prefix starts. Safe cross-kind outdent splits the outer list around a matching container with explicit numbered continuation. A trailing parent block which cannot remain in reading order rejects the specific outdent unchanged.

Quote actions implement local toggle/removal, one shared wrapper for selected siblings, atomic mixed-depth rejection, shared cite/data-comment-id references on both retained fragments, and Q03 confirmation for actual removed-container metadata. Unique/unknown retained-fragment allocation rejects only the ambiguous split. Nested partial reduction through an intervening list rejects without expanding to unselected entries. Empty quote exit preserves paragraph attributes; empty quotes/lists have distinct semantic selection extents. Native actions respect intervening Quote/Code scopes instead of operating on an enclosing list item.

Selection records Code-line origin independently of endpoint orientation. Swapping a retained subtree selection ending in Code does not change its kind; actual Code-V remains bounded. Whole-entry Visual deletion includes children for canonical taskItem/p HTML. Opaque parent changes confirm once and validate preservation; cancellation is unchanged, Undo restores source and Redo does not prompt. Safe metadata on retained list containers does not unnecessarily block indent. Task Done, dependencies, schema and local user DB remain unchanged.

Five fresh independent bug/risk and consistency/validation review pairs examined the full diff. Confirmed findings were reproduced with failing regressions, fixed and retested: scope expansion/narrowing, reversed continuation, dropped container/child metadata, native Quote/Code routing, empty blocks, end-split cursors, cross-kind outdent and retained selection endpoints. After the interruption, two independent targeted verifiers rechecked the final-round Code-tail/endpoint-swap fix and adjacent copy/deletion/history behavior; no confirmed findings remain. This targeted verification did not restart the full review-round budget.

Checks after the last code changes:

- `go test ./...` — pass.
- `go test -race ./internal/description ./internal/tui` — pass.
- `go vet ./internal/description ./internal/tui` — pass.
- `go build -buildvcs=false ./cmd/vimkunja ./cmd/vimkunja-tutorial` — pass; VCS stamping disabled for the root-owned worktree.
- `go test ./internal/description -run '^$' -fuzz '^FuzzDraftStructureActionsAreAtomic$' -fuzztime=5s -parallel=2` — pass, 36,734 executions in the final bounded run.
- `git diff --check`, LF and changed Markdown local-link validation — pass.

Execution mapping: `TestDraftStructureCorpusEdits` checks ten independently authored fixture goldens: list-indent-existing-children, list-outdent-reading-order, list-selection-atomicity, list-kind-preservation, ordered-middle-exit, empty-imported-ul, empty-imported-ol, quote-shared-source-split, quote-unique-attribute-split and quote-shared-wrap-atomic. Null/random-ID fixtures checkbox-middle-split and empty-imported-checkbox are checked structurally in Draft tests, with repeated successful SQLite saves covering Beta ID adoption and Alpha/child ID stability.

`draft_structure_test.go` covers L01–L12/Q01–Q04 with explicit trees/attributes, Checkbox moves and parent-child overlap, invalid mixed selections, root rejection, numbered splits, compatible/incompatible merges, metadata preservation, Opaque confirmation/history and Code selection boundaries. `native_structure_test.go` covers actual keys/palette dispatch, progress recomputation, Task Done independence, empty fill/cancel/Enter/final deletion and repeated temporary-SQLite saves. README, Help and keybinding status document implemented actions. Corpus expectations were not regenerated from implementation output.

Remaining work: 08 owns Code creation/exits/Tab/retab and exact terminal/CRLF input; 09 owns deliberate syntax/Inline-Code/URL flows. Real Windows/German keyboard, human terminal acceptance and versioned VOA evidence remain 10. Synthetic tests and temporary SQLite data do not constitute those external acceptance checks.
