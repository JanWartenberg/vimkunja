# 09 — Inline syntax, Inline-Code and Link editing

**What to build:** Implement deliberate editor syntax and separate Link label/URL editing while preserving imported/pasted literal text and safe existing semantics.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** ready-for-agent

**Blocked by:** 06

**Contract:** [spec.md](../spec.md), [decision/handoff ticket 02](02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../corpus.md), [83 synthetic fixtures](../corpus.json), [keybinding contract](../../../docs/keybindings.md).

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
