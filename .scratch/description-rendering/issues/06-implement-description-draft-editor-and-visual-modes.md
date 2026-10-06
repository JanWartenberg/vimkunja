# 06 — Native draft editing, Visual Mode and local history

**What to build:** Edit ordinary Native text/blocks, apply selection formatting and confirmed Opaque operations, and save/discard through the safe Store seam.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** ready-for-agent

**Blocked by:** 04, 05

**Contract:** [spec.md](../spec.md), [decision/handoff ticket 02](02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../corpus.md), [83 synthetic fixtures](../corpus.json), [keybinding contract](../../../docs/keybindings.md).

## Scope

- Integrate the structured draft into Task Detail Normal/Insert modes, dirty tracking, save/quit protection and existing task-wide explicit discard. Keep Source HTML as persistence data, never as the normal editable text buffer.
- Implement i/a/I/A, o/O, text insertion/deletion, paragraph Enter/Shift+Enter, compatible boundary merge, headings/paragraph conversion and logical dd. Route Code/quote internal scopes through document actions; advanced structure/Code/syntax details are completed in tickets 07–09.
- Implement v/V, document-anchored selections, viw/vaw, selection-end swap, counted motions and retained selection until Esc. Never expand a selection implicitly; Opaque is whole-atom only. V inside Code selects clamped source lines; :selectblock explicitly selects whole blocks.
- Implement uniform bold/italic/underline/strike toggles; off removes all selected layers of that family, including triple strong; other marks persist. Space+0 removes all inline formatting including Links while preserving text/structure.
- Provide mode-specific Space palette dispatch/footer behavior: secondary footer unless fewer than three Description rows remain, then primary replacement; Esc/timeout/action closes, Insert Space stays text, Space+o stays global. Unavailable later-ticket actions must not pretend success.
- Opaque formatting skips atoms; complete removal/replacement and parent conversion use one confirmation plus identity-preserving validation. Cancellation retains draft/pending input. Copy only safe visible content and paste it as new Native text.
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

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. This ticket preparation does not start production implementation. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.
