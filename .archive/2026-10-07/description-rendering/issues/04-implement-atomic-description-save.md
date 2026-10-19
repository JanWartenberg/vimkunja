# 04 — Atomic Description saves and Checkbox identities

**What to build:** Compare the opened Description baseline and write the saved candidate in one SQLite transaction, repairing only missing/duplicate Native Checkbox IDs on success.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** completed

**Completed:** 2026-10-07

**Blocked by:** None

**Dependency satisfied:** [03 — document model](03-implement-source-preserving-description-model.md), completed 2026-10-07.

**Contract:** [spec.md](../../../2026-10-19/description-rendering/spec.md), [decision/handoff ticket 02](../../../2026-10-19/description-rendering/issues/02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../../../../.scratch/description-rendering/corpus.md), [83 synthetic fixtures](../../../../.scratch/description-rendering/corpus.json), [keybinding contract](../../../../docs/keybindings.md).

## Scope

- Add the narrow Store interface needed for baseline-aware Description persistence. Cover existing Task save and combined Task/Comment save flows; do not implement a UI-only preflight check with a race between check and UPDATE.
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
- No editor write path may activate before this safe save seam is available.

**Focused coverage:** `external-save-conflict`, `checkbox-lists`, `checkbox-duplicate-ids`, `checkbox-id-reservation-atomicity`, `checkbox-middle-split`; existing Store description-preservation tests.

Run affected tests during implementation; before a commit run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The existing artifact probes are supplemental, not Native execution evidence. Required test commands and fixture mapping must be recorded in the completion comment.

## Boundaries

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. This ticket preparation does not start production implementation. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.


### 2026-10-07 — implementation and verification

Implemented on `feature/description-model` in `/workspace/vimkunja-richtext-03`; committed on 2026-10-08. `description.Document.PrepareSave` produces an immutable, validated candidate without changing the source or snapshot. It reserves imported/draft/Opaque IDs, preserves first nonempty Native IDs, fills only missing/later duplicates with unique lowercase UUID v4, retries collisions and bounds a broken generator. The Store consumes candidates through optional `TaskDraft.DescriptionCandidate` and `DescriptionBaseline` fields, preserving its existing raw API; existing Detail Task and Task/Comment saves always supply their opened baseline. Explicit persistence baselines supersede the candidate's original preservation-document provenance; no external merge occurs.

Existing saves compare the exact source inside the same transaction as all Task/Comment writes. `DescriptionConflictError` carries opened/current HTML for explicit reload or return-to-draft. New Tasks validate candidates before their transactional insert. Task-only saves preserve exact source bytes and Done. Success returns committed HTML; failures return no adoptable Task and leave draft/candidate inputs untouched. Editor completion adopts committed HTML and the refreshed baseline, preserving late changes. Native editor dispatch remains disabled; ticket 06 must adopt committed repaired identities into its live nodes and rebuild future candidates before dispatch, rather than reuse independently prepared candidates with unadopted UUIDs or reconcile IDs by text/position.

Review round 1 confirmed and fixed three issues: failed SQLite COMMIT leaving a live transaction in the pool; a post-commit history read being reported as persistence failure; and a late candidate conflicting with the editor's own successful save. Save transactions now pin their connection, explicitly roll back failed COMMIT and discard the connection if cleanup fails. Undo-read failures warn separately while committed data is adopted. A deferred SQLite transaction may fail with SQLITE_BUSY under concurrent writers; such failures leave the draft available for retry.

Coverage maps `external-save-conflict` to two independent handles and actual Detail/combined conflict retention; `checkbox-lists`, `checkbox-duplicate-ids` and `checkbox-id-reservation-atomicity` to independent corpus candidate goldens, minimal repair, Opaque reservation, generator collision/failure and repeat-save stability. `checkbox-middle-split` executes the specified Alpha/Beta/child snapshot through the candidate seam; key dispatch/splitting remains ticket 07. Disposable Store tests exercise creation/update/combined success, missing Tasks, empty baselines, invalid Comment/candidate/workflow, Comment SQL failure, concurrent writers and deterministic failed COMMIT under a held external reader for all three save paths. Editor tests exercise committed adoption, late changes, unchanged HTML, conflict retention and post-commit history failure.

Checks passed: `go test ./... ./.scratch/description-rendering/verification`; `go test -race ./internal/description ./internal/vimkunja ./internal/tui`; `go vet ./internal/description ./internal/vimkunja ./internal/tui`; `go build -buildvcs=false ./cmd/vimkunja ./cmd/vimkunja-tutorial`; repeated concurrency/rollback tests, gofmt and diff checks. Build stamping is disabled only for checkout Git ownership. Review round 2 used two fresh independent read-only reviewers (bugs/risks and consistency/validation); both report no actionable findings. No confirmed findings remain within ticket 04. No actual user DB, schema/dependency change, Native terminal acceptance or real VOA capture is claimed.
