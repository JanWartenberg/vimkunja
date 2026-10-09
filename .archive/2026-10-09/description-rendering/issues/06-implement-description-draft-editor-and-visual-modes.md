# 06 — Native draft editing, Visual Mode and local history

**What to build:** Edit ordinary Native text/blocks, apply selection formatting and confirmed Opaque operations, and save/discard through the safe Store seam.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** completed

**Completed:** 2026-10-09 (committed)

**Blocked by:** None

**Dependencies satisfied:** [05 — native projection/navigation](../../../2026-10-08/description-rendering/issues/05-implement-native-description-projection-and-navigation.md), completed 2026-10-08; [04 — atomic saves](../../../2026-10-07/description-rendering/issues/04-implement-atomic-description-save.md), completed 2026-10-07 (committed 2026-10-08).

**Contract:** [spec.md](../../../../.scratch/description-rendering/spec.md), [decision/handoff ticket 02](../../../../.scratch/description-rendering/issues/02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../../../../.scratch/description-rendering/corpus.md), [83 synthetic fixtures](../../../../.scratch/description-rendering/corpus.json), [keybinding contract](../../../../docs/keybindings.md).

## Scope

- Integrate the structured draft into Task Detail Normal/Insert modes, dirty tracking, save/quit protection and existing task-wide explicit discard. Keep Source HTML as persistence data, never as the normal editable text buffer.
- Implement i/a/I/A, o/O, text insertion/deletion, paragraph Enter/Shift+Enter, compatible boundary merge, headings/paragraph conversion and logical dd. Route Code/quote internal scopes through document actions; advanced structure/Code/syntax details are completed in tickets 07–09.
- Implement v/V, document-anchored selections, viw/vaw, selection-end swap, counted motions and retained selection until Esc. Never expand a selection implicitly; Opaque is whole-atom only. V inside Code selects clamped source lines; :selectblock explicitly selects whole blocks.
- Implement uniform bold/italic/underline/strike toggles; off removes all selected layers of that family, including triple strong; other marks persist. Space+0 removes all inline formatting including Links while preserving text/structure.
- Provide mode-specific Space palette dispatch/footer behavior: secondary footer unless fewer than three Description rows remain, then primary replacement; Esc/timeout/action closes, Insert Space stays text, Space+o stays global. Unavailable later-ticket actions must not pretend success.
- Opaque formatting skips atoms; complete removal/replacement and parent conversion use one confirmation plus identity-preserving validation. Cancellation retains draft/pending input. Copy only safe visible content and paste it as new Native text.
- After successful persistence, adopt repaired Checkbox identities into the live document before preparing its next candidate. Rebuild pending candidates from the current live nodes; never reconcile independently generated IDs by text/position. The explicit Store persistence baseline may differ from the preservation document after our own save.
- Implement local u/Ctrl+R separately from persisted Task-action history. Confirmed removal Undo restores exact bytes/identity/location; Redo does not prompt again. Each logical action is atomic.
- Implement Blocked Clear/Recover/Discard flows as confirmed unsaved actions using safe text, excluding active source, hidden metadata and warning placeholders. Surface save conflicts with return-to-draft/hard-reload and no automatic merge.

## Acceptance and validation

- Enter at Alpha|Beta creates separate p elements; Shift+Enter creates br. o/O/dd operate on logical blocks, not wrapped rows. Save writes only after successful Store validation; discard restores the source.
- Mixed mark toggles, all-family off, cross-block selections and Opaque skipping execute the exact corpus edits. Character deletion is grapheme-safe; protected cues are never deleted or copied.
- Confirm/cancel/delete/replace/Undo/Redo of inline and block Opaque content preserve the documented scopes. A failed save/conflict keeps draft, selection/history and unadopted UUID candidates.
- V inside Code cannot accidentally include the following paragraph; :selectblock can select the entire Code. Invalid mixed structure actions reject without partial mutation.
- Actual TUI state/event tests cover palette height threshold, Leader timeout, save/quit/discard and unaffected Title/Comment behavior.

**Focused coverage:** `paragraph-creation-and-split`, `hard-breaks`, `native-selection-and-history`, `uniform-toggle`, `triple-strong-off`, `all-mark-families-off`, `opaque-edit-neighbors`, `opaque-parent-conversion`, `code-line-selection`, `malformed-html`, `document-wrappers`.

Run affected tests during implementation; before a commit run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The existing artifact probes are supplemental, not Native execution evidence. Required test commands and fixture mapping must be recorded in the completion comment.

## Boundaries

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. Production implementation was explicitly authorized through implement-and-verify. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.


### 2026-10-09 — implementation and verification complete

Implemented on `feature/description-model` in `/workspace/vimkunja-richtext-03`; changes are committed on 2026-10-09. Structured Native draft actions now drive Task Detail editing, grapheme-safe text deletion, paragraph/hard-break edits, compatible merges, logical deletion, Visual selections/text objects, uniform formatting, safe local copy/paste and independent local Undo/Redo. Opaque removal/replacement/parent conversion is confirmed and validated; formatting preserves atoms and reports skipping them. Empty quote exit confirms actual quote metadata loss. Selections overlapping Code reject inline formatting atomically; Code indentation is owned by 08.

Save integration rebuilds immutable candidates from current live identities, adopts successful Checkbox repairs by Native ID into live/history snapshots, and keeps the persisted baseline distinct from the original preservation source. Failed/conflicting saves keep draft/selection/history unchanged. Blocked clear/recovery are confirmed unsaved actions; explicit hard reload discards the complete Task draft, guarded against late edits. The palette threshold uses actual Description geometry; empty documents get nonpersisted cursor targets.

Five independent bug/consistency review rounds found and fixed insertion/end-of-line cursors, nested mark removal, Opaque skipping/highlighting, semantic word selection, empty Code-line deletion, recovery dirty tracking, pending-paste freezing, ID/attribute safety, stale persisted-Undo state, actual palette geometry and deletion cursor anchoring. Final-round findings were reproduced with failing focused tests, fixed, and independently rechecked; no confirmed findings remain. The final consistency reviewer reused an existing reviewer thread because the harness refused another fresh thread at its limit; the two review axes remained independent.

Validation after the last code changes:

- `go test ./...` — pass.
- `go test -race ./internal/description ./internal/tui` — pass.
- `go vet ./internal/description ./internal/tui` — pass.
- `go build -buildvcs=false ./cmd/vimkunja ./cmd/vimkunja-tutorial` — pass (VCS stamping disabled for the root-owned worktree).
- `go test ./internal/description -run '^$' -fuzz '^FuzzDraftActionsAreAtomic$' -fuzztime=5s -parallel=2` — pass, 50,298 executions; bounded synthetic evidence.

Fixture mapping: `draft_corpus_test.go` executes paragraph-creation-and-split, hard-breaks, native-selection-and-history, uniform-toggle, triple-strong-off, all-mark-families-off, opaque-edit-neighbors, opaque-parent-conversion, malformed-html and document-wrappers against independently authored corpus assertions. Code-line-selection uses actual newline module/TUI tests (the synthetic fixture's source contains literal backslash-n). `draft_test.go` adds atomic operations/preservation/history/formatting boundaries; `native_draft_test.go` exercises actual event/mode/palette/save/conflict/recovery flows with temporary SQLite data.

Advanced list/Checkbox/Quote operations remain 07; Code creation/exits/Tab/retab and exact CRLF paste remain 08; deliberate syntax/Inline-Code/URL flows remain 09. Existing basic Code Enter/o/O/dd routing is provided here. No real Windows/dead-key/clipboard capture, VOA serialization evidence or human terminal acceptance is claimed: these remain 10. Dependencies, schema and local user database are unchanged.
