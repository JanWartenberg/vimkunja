# 09 — Inline syntax, Inline-Code and Link editing

**What to build:** Implement deliberate editor syntax and separate Link label/URL editing while preserving imported/pasted literal text and safe existing semantics.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** completed

**Completed:** 2026-10-10; verified in `feature/description-model`, committed 2026-10-10.

**Blocked by:** None

**Dependency satisfied:** [06 — Draft/Visual/history](../../../2026-10-09/description-rendering/issues/06-implement-description-draft-editor-and-visual-modes.md), completed 2026-10-09; committed 2026-10-09.

**Contract:** [spec.md](../../../2026-10-19/description-rendering/spec.md), [decision/handoff ticket 02](../../../2026-10-19/description-rendering/issues/02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../../../../.scratch/description-rendering/corpus.md), [83 synthetic fixtures](../../../../.scratch/description-rendering/corpus.json), [keybinding contract](../../../../docs/keybindings.md).

## Scope

- Implement accepted delimiter/escape grammar, heading/list following-Space triggers and dash-only rule conversion. Keep temporary typed dash counts for x/Undo; canonicalize to hr only on save.
- Track imported/pasted literal origin so ordinary word edits do not activate delimiters. Deliberate escape removal reparses only the edited region; incomplete/crossed/mismatched expressions remain literal. Empty imported semantic wrappers disappear; new empty delimiter pairs stay literal.
- Inline-Code is Native in V1: deliberately typed paired single backticks create a span; imported/pasted ones stay literal, escaped inner backtick is text, other markup inside stays literal. Removal preserves text/other marks; edge insertion stays outside.
- Implement Space+l selection/current-Link URL prompt, empty-URL unlink, replacement rather than nested anchors, and per-paragraph anchors across block selections while skipping Opaque. Text inserted inside remains linked; exact-edge input is ordinary text.
- Centralize the seven allowed absolute :// schemes with syntax/scheme-appropriate validation and no DNS. Invalid new URLs keep prompt open/draft unchanged; safe unsupported imports remain replaceable/removable but not newly creatable. Unsafe attributes/destinations follow local Opaque protection.
- Serialize Native links with target=_blank, retain safe meaningful title/rel/data attributes, invent no rel, and keep marks/hard breaks inside anchors.
- Empty imported Link label shows URL; focus/URL-only changes keep label empty, first real text edit materializes current fallback once, unlink retains visible fallback and explicit content deletion removes it.
- Space+y copies destination without opening it and reports actual clipboard success/failure without changing draft. Show accepted inside/edge/unsupported-scheme footer hints.

## Acceptance and validation

- Execute literal import/Paste/escape parity, combined mark nesting, Inline-Code and incremental rule-history fixtures. Unrelated literal regions/marks/Opaque bytes remain unchanged.
- Test allowed/rejected URL families plus focused web-host/non-web syntactic edge cases; central policy is extensible internally but V1 allowlist remains fixed.
- Empty Link URL-only edit leaves empty HTML label; first append ! materializes fallback once. Inside/edge edit and cross-paragraph selection produce the exact scoped anchors.
- Clipboard failure is a footer error, no draft mutation or navigation. Existing label picker/Global Space bindings remain correct in their contexts.

**Focused coverage:** `editor-delimiters`, `typed-versus-pasted-markers`, `literal-import-edit`, `syntax-escape-parity`, `native-inline-code`, `inline-code-typed-literal`, `typed-rule-dash-history`, `links`, `link-import-edges`, `empty-link-label-edit`, `cross-paragraph-links`.

Run affected tests during implementation; before a commit run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The existing artifact probes are supplemental, not Native execution evidence. Required test commands and fixture mapping must be recorded in the completion comment.

## Boundaries

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. This ticket preparation does not start production implementation. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.

### 2026-10-10 — implementation started

Implementing on `feature/description-model` in `/workspace/vimkunja-richtext-03`, following 08 commit `3dd850e`. Existing Draft/save and actual TUI event seams remain the validation boundaries. Independent review and final verification are pending; no commit authorized for 09.

### 2026-10-10 — implementation and verification completed

Implemented deliberate typing versus literal import/Paste origins; editable delimiters and protecting escapes; typed heading/list/rule grammar; Native Inline-Code; URL prompt/validation, scoped Link replacement/unlink and preserved metadata; empty-label materialization and live cursor/selection repair; actual clipboard-result feedback and context hints. Draft-only origin/dash/replacement metadata never persists to HTML. Structural actions clear stale delimiter targets; selected empty fallback labels participate in content formatting, including cross-paragraph endpoints.

Five independent read-only review pairs completed. Confirmed findings were fixed with focused regression tests; final-round structural/empty-Link fixes received targeted independent verification. No confirmed findings remain. README, Help, keybindings and corpus traceability are synchronized. Forty independently authored non-null edit goldens across all eleven focused records execute through actual TUI dispatch and prepared-save HTML; additional tests cover assertion-only behavior, all agreed escapes, URL policy, metadata/identity guards, replacement/edge semantics, scoped preservation, actual clipboard failure/success and temporary-SQLite save adoption/local history. Corpus goldens were not regenerated.

Checks run successfully from the isolated worktree:

- `go test ./... ./.scratch/description-rendering/verification`
- `go test -race ./internal/description ./internal/tui`
- `go vet ./internal/description ./internal/tui`
- `go build -buildvcs=false ./cmd/vimkunja ./cmd/vimkunja-tutorial`
- `go test ./internal/description -run '^$' -fuzz '^FuzzDraftInlineEventGrouping$' -fuzztime=10s -parallel=2` (final bounded run: 5,412 executions; earlier run: 40,896)
- `git diff --check`

Go commands used `/usr/local/go/bin/go`; VCS stamping is disabled because of shared-checkout ownership. Final full/race/vet/build checks were rerun after the final source corrections. The grouping fuzz compares single-event/per-character typing and save output; its final run preceded the last structural/fallback-formatting fixes, which have deterministic regressions and final full/race checks.

No live user database, schema or dependency changed. Committed 2026-10-10 after explicit user authorization. Ticket 10 owns exhaustive integrated per-rule reporting, tutorial, real VOA output and German-layout/Windows terminal acceptance. Synthetic tests and injected clipboard writers do not certify those environments.
