# 05 — Native terminal projection and document navigation

**What to build:** Render the structured Description and navigate document targets without exposing raw HTML or changing Source HTML.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** ready-for-agent

**Blocked by:** None

**Dependency satisfied:** [03 — document model](../../../.archive/2026-10-07/description-rendering/issues/03-implement-source-preserving-description-model.md), completed 2026-10-07.

**Contract:** [spec.md](../spec.md), [decision/handoff ticket 02](02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../corpus.md), [83 synthetic fixtures](../corpus.json), [keybinding contract](../../../docs/keybindings.md).

## Scope

- Render semantic styles with capability fallbacks, paragraphs/headings, nested list markers/numbering, quotes, Code, Links and display-only Checklist progress. Preserve the existing read-only summary behavior while preparing draft-based recomputation.
- Render Tables as readable grid or irregular row/cell fallback; images alt→title→Image, Mentions @label→@id, embedded/active/unknown/comment atoms safely. Make complete Opaque extents visible, including multiline atoms. Never load media, avatars, source URLs or CSS.
- Implement document-anchored j/k, gj/gk, paragraph motions {/} and ö/ä, counted motions, gg/G, Ctrl+D/U, logical-line/word motions and whole-grapheme character motion. Code source lines and quote children are local j/k units; complete Code/quote remain paragraph units.
- Focus discloses editor syntax for the active logical block; Code discloses its whole protected triple-fence enclosure. Preserve variant/depth labels and protected markers without counting them as editable characters.
- Reflow active paragraphs without moving their editing target; compensate vertical scroll, keep cursor visible, and retain per-Code/Table horizontal scrolling across focus transitions. zh/zl, overflow cues/Code scrollbar and distinct soft↪/hard↵ markers follow the contract.
- Preserve draft/document anchor when the Pane is too narrow; report Pane too narrow rather than dropping characters or creating document content.
- Connect Native read/navigation to Task Detail without enabling its unsupported editing commands yet. Keep unaffected Title/Comment/plain-text editor behavior intact.

## Acceptance and validation

- Fixture logical content, semantic styles and atomic boundaries are checked separately from exact terminal coordinates. No hidden attributes, IDs/URLs (except accepted visible fallback labels) enter rendered/copyable text.
- Wide, narrow and short-pane tests verify cursor anchoring through disclosure/reflow, protected motion counts, Unicode graphemes, soft/hard breaks, overflow and focus restoration.
- j/k navigates Code lines/quote children; {/}/ö/ä traverses their complete blocks. An Opaque projection containing multiple words remains one word-motion unit.
- Opening, navigating, resizing and scrolling write nothing and do not dirty the draft. Existing Description viewport and checklist regressions pass.

**Focused coverage:** `layout-and-palette`, `graphemes-and-delimiter-selection`, `code-and-quotes`, `table-mixed-content`, `table-irregular`, `unknown-inner-block-boundaries`, `image-empty-alt`, `mention-label`, `mention-id-fallback`.

Run affected tests during implementation; before a commit run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The existing artifact probes are supplemental, not Native execution evidence. Required test commands and fixture mapping must be recorded in the completion comment.

## Boundaries

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. This ticket preparation does not start production implementation. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.
