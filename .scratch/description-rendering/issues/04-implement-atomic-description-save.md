# 04 — Atomic Description saves and Checkbox identities

**What to build:** Compare the opened Description baseline and write the saved candidate in one SQLite transaction, repairing only missing/duplicate Native Checkbox IDs on success.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** ready-for-agent

**Blocked by:** 03

**Contract:** [spec.md](../spec.md), [decision/handoff ticket 02](02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../corpus.md), [83 synthetic fixtures](../corpus.json), [keybinding contract](../../../docs/keybindings.md).

## Scope

- Add the narrow Store interface needed for baseline-aware Description persistence. Compare the exact raw source HTML baseline with the persisted Description in the same transaction as the write; do not compare normalized or projected text. Cover existing Task save and combined Task/Comment save flows; do not implement a UI-only preflight check with a race between check and UPDATE.
- Build serialization/UUID repair candidates separately from the live draft. Preserve the first occurrence of each existing nonempty Native ID; reserve observable existing IDs, including Opaque values, and retry generator collisions. Replace only missing IDs/later Native duplicates with distinct lowercase UUID v4 values.
- Keep entry text, nesting/order, checked state, safe metadata and Opaque bytes. Do not clear all IDs in an affected list, prompt for pure ID repair or change Task Done.
- A missing Task, external Description conflict, preservation failure, invalid combined save or SQL failure must roll back the complete operation and leave local draft/repair candidates unadopted. Return a conflict/error the editor can render.
- Adopt generated IDs and refreshed baseline only after commit; a second save keeps them stable. Preserve current Task-save/history guarantees and unrelated fields; a Task-only edit must never rewrite Description through the lossy reader.
- Expose explicit reload/return-to-draft integration data; there is no automatic merge. New Task creation has no existing-Task baseline but still needs candidate validation and transactional ID adoption.

## Acceptance and validation

- Two independent database handles simulate open baseline Original, local Local, remote Remote: local save is rejected, DB stays Remote and draft stays Local.
- Test an external change after candidate preparation and before persistence; comparison and write remain atomic. Test combined Task/Comment rollback and missing-Task handling with disposable databases.
- `checkbox-id-reservation-atomicity` retries reserved/duplicate generator results; successful save repairs minimally and stabilizes IDs, failure/conflict adopts none.
- Existing byte-preservation/Task Done/rollback regressions still pass. No actual local user database or schema migration is used.
- No Native Description editor write path may activate before this safe save seam is available.

**Focused coverage:** `external-save-conflict`, `checkbox-lists`, `checkbox-duplicate-ids`, `checkbox-id-reservation-atomicity`, `checkbox-middle-split`; existing Store description-preservation tests.

Run affected tests during implementation; before a commit run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The existing artifact probes are supplemental, not Native execution evidence. Required test commands and fixture mapping must be recorded in the completion comment.

## Boundaries

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. This ticket preparation does not start production implementation. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.
