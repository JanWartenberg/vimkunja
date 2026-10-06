# 08 — Code editing, Tabs and terminal input

**What to build:** Edit Native Code with exact source whitespace, explicit creation/exit and Vim-like Tab/retab behavior, including reliable event dispatch.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** ready-for-agent

**Blocked by:** 06

**Contract:** [spec.md](../spec.md), [decision/handoff ticket 02](02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../corpus.md), [83 synthetic fixtures](../corpus.json), [keybinding contract](../../../docs/keybindings.md).

## Scope

- Implement Code Enter/o/O/source-line dd: deleting final nonempty line leaves empty Code, subsequent dd removes the block. Esc only returns to Normal. Preserve original and pasted Tabs/CRLF/leading/trailing blank lines.
- Route Code Paste before generic prose CRLF normalization. Code ordinary Tab inserts Spaces to the next tabstop; Ctrl+V then Tab inserts one literal Tab. Preserve grouped Text events/backticks and non-ASCII character bindings.
- Create Code directly via Normal Space+c or exactly three/four directly typed backticks in an otherwise empty paragraph confirmed by Enter. No timer, dropped fourth marker or Paste trigger; Escape/extra text/five markers stay literal, Backspace edits candidates and conversion is one Undo step.
- ü/Ü creates after/before the complete Code into Insert; ]Space/[Space does the same into Normal. Nested placement is same-container sibling, no automatic unnesting. Esc retains the explicit empty paragraph; Undo removes creation. :after is not added.
- Implement selected Code-line Space+x/c outdent/indent without other normalization. Empty lines unchanged; only a partially removed leading Tab becomes remaining Spaces.
- Implement :retab on current Code/selected Code lines and :retaball on all Native text, including prose under its existing whitespace policy. Convert all scoped Tabs, including strings; skip Opaque exactly, one Undo, no V1 args/bang.
- Add existing tui-TOML settings for positive tabstop (default four) independently of indent step (two/four, default four). Provide configurable literal-Tab alternative; Ctrl+Q is the accepted alias candidate and Ctrl+B configurable. Verify actual dispatch conflicts before enabling public aliases, with no automatic Windows detection or dependency changes.

## Acceptance and validation

- Execute trigger event variants, last-line dd, first/final/nested Code exits, selected-line boundaries and exact Code import/Paste fixtures against the actual editor.
- At tabstop four, Tab at column zero yields four Spaces and after one character three. Retab string Tabs too; partial-Tab outdent preserves remaining width, other Code characters stay exact.
- Simulate grouped event chunks and literal-prefix cancellation/error routing; test new settings validation/precedence within existing configuration conventions. Synthetic decoder PASS does not certify real terminal input.
- Record practical German-layout/Windows events for ü/Ü, bracket AltGr, three/four dead-key backticks, Ctrl+V/Paste and the chosen alias in ticket 10. Terminal-specific limitations are recorded rather than silently changing product keys.

**Focused coverage:** `code-trigger-events`, `code-exit-boundaries`, `code-exit-nested`, `code-exit-list-entry`, `standalone-code-editing`, `exact-code-import`, `code-tabs-retab`, `retaball-native-scope`, `code-indent-outdent`, `code-line-selection`.

Run affected tests during implementation; before a commit run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The existing artifact probes are supplemental, not Native execution evidence. Required test commands and fixture mapping must be recorded in the completion comment.

## Boundaries

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. This ticket preparation does not start production implementation. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.
