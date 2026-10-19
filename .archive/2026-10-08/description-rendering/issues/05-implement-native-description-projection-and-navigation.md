# 05 — Native terminal projection and document navigation

**What to build:** Render the structured Description and navigate document targets without exposing raw HTML or changing Source HTML.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** completed

**Completed:** 2026-10-08

**Blocked by:** None

**Dependency satisfied:** [03 — document model](../../../2026-10-07/description-rendering/issues/03-implement-source-preserving-description-model.md), completed 2026-10-07.

**Contract:** [spec.md](../../../2026-10-19/description-rendering/spec.md), [decision/handoff ticket 02](../../../2026-10-19/description-rendering/issues/02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../../../../.scratch/description-rendering/corpus.md), [83 synthetic fixtures](../../../../.scratch/description-rendering/corpus.json), [keybinding contract](../../../../docs/keybindings.md).

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

### 2026-10-08 — implementation and verification

Implemented on `feature/description-model`: document-owned immutable projection/navigation (`internal/description/view*.go`) and Task Detail integration (`internal/tui/native_description.go`). Semantic styles, logical blocks/Code lines/quote children, inline syntax disclosure, protected frames, safe Opaque previews, tables/captions/fallbacks, Mention/media labels, checklist progress and static native cursor now render. Grapheme/word/logical/visual/paragraph motions, counts, one-Pane-height paging, anchor-preserving resize, viewport-only scrolling and per-Code/Table horizontal offsets execute without enabling native editing. Help, README and keybinding implementation statuses match this boundary.

Focused fixture mapping: `TestViewCorpusProjectionSafetyAndFocusedContent` covers all nine listed records' semantic content independently from terminal coordinates; all 83 source records receive projection safety/width/height/source-preservation checks. Additional `TestView*` tests cover source-line blank/trailing Code lines, whole quote/Code paragraph units, graphemes, variants, hard/soft breaks, semantic word continuity across marks, block boundaries, local narrow panes, captions/styled tables, summary preservation and horizontal scroll limits. `TestNativeDescriptionResizeMotionsDoNotPersist` exercises actual TUI keys, wide/narrow/short terminal resize and SQLite data-version/source checks. Existing Title/Comment/plain-text and HTML viewport regressions pass.

Checks executed: `go test ./...`; `go test -race ./internal/description ./internal/tui`; `go vet ./internal/description ./internal/tui`; `go build -buildvcs=false ./cmd/vimkunja ./cmd/vimkunja-tutorial`; `go test ./internal/description -run '^$' -fuzz FuzzViewProjectionPreservesSource -fuzztime=5s -parallel=2` (initial 71,206 inputs; final rerun 50,079 inputs). `-buildvcs=false` avoids unrelated Git ownership stamping in the root-owned worktree. No dependencies, SQLite schema or local user database changed.

Independent review rounds and fixes are recorded at the final verification checkpoint below. No practical Windows/German keyboard session, versioned VOA capture, edit-action execution or manual acceptance is claimed. Native editing/history/recovery is ticket 06; later structure/Code/Link actions remain 07–09 and integrated acceptance is 10. Projection options/theme provide the configuration seam; user-facing Description TOML settings remain follow-up integration.

### 2026-10-08 — final verification checkpoint

Five independent review rounds each used fresh read-only bugs/risks and consistency/validation reviewers. Confirmed findings were fixed with focused regressions: hard-break logical targets/visible markers; semantic word continuity and block limits; ge/_ and counted gg/G; one-Pane-height visual paging; local narrow behavior and actual-width hints; source-column retention, horizontal limits, wide-grapheme visibility and resize reveal in Code; empty heading targets; safe Opaque inline extents, checklist text/state/summaries and ordered-entry values; styled-table descendants/captions and ASCII cues. Final round findings are closed; the last hard-break/Opaque-adjacency correction was separately rechecked by both final-round reviewers. No confirmed findings remain.

Final `go test ./...`, affected race tests, affected vet, both entrypoint builds and the 5-second bounded fuzz rerun pass. Changed Go/Markdown files use LF; local Markdown links and `git diff --check` pass. The SQLite observer connection is pinned and checks cross-handle data-version plus same-handle total changes and exact persisted source. Committed on 2026-10-08 in `/workspace/vimkunja-richtext-03` on `feature/description-model`; ticket 06 is next. Practical Windows/German-terminal and VOA acceptance remain outstanding.
