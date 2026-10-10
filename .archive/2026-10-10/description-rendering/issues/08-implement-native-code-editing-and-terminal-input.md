# 08 — Code editing, Tabs and terminal input

**What to build:** Edit Native Code with exact source whitespace, explicit creation/exit and Vim-like Tab/retab behavior, including reliable event dispatch.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** completed

**Completed:** 2026-10-10; verified in `feature/description-model`, committed 2026-10-10.

**Blocked by:** None

**Dependency satisfied:** [06 — Draft/Visual/history](../../../2026-10-09/description-rendering/issues/06-implement-description-draft-editor-and-visual-modes.md), completed 2026-10-09; committed 2026-10-09.

**Contract:** [spec.md](../../../../.scratch/description-rendering/spec.md), [decision/handoff ticket 02](../../../../.scratch/description-rendering/issues/02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../../../../.scratch/description-rendering/corpus.md), [83 synthetic fixtures](../../../../.scratch/description-rendering/corpus.json), [keybinding contract](../../../../docs/keybindings.md).

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

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.

### 2026-10-10 — implementation and verification completed

Implemented exact Code editing/line operations, creation by Space+c and Enter-confirmed three/four typed backticks, same-container ü/Ü and bracket-Space exits, Tab/literal-Tab dispatch, selected indentation, retab/retaball and existing TOML configuration (`tabstop=4`, independent `indent_step=4` accepting 2/4, `literal_tab_alias="ctrl+q"` accepting ctrl+b or empty). Empty Description supports explicit Space+c. Multi-wrapper Code remains one source while wrapper metadata/identities stay intact. Paste keeps Code CRLF/Tabs and preserves Comment/prose behavior. Native edits surrounding protected Code children remain available; atom gaps are documented source-column barriers, and destructive source-line operations crossing private atom content reject unchanged while explicit whole-block removal remains confirmable.

TDD used the existing Draft Apply/Snapshot/save and actual TUI/config seams. Four independent read-only bug/consistency review pairs ran. Confirmed issues were fixed with deterministic regressions: multi-leaf Code/V source offsets, CRLF append/line endings, empty creation, Comment paste routing, focus/global-key prefix cancellation, stale bracket prefixes, o insertion cursor and protected Code-line deletion. Both final reviewers report no findings.

Validation passed:

- `/usr/local/go/bin/go test ./...`
- `/usr/local/go/bin/go test -race ./internal/description ./internal/tui`
- `/usr/local/go/bin/go vet ./internal/description ./internal/tui`
- `/usr/local/go/bin/go build -buildvcs=false ./cmd/vimkunja ./cmd/vimkunja-tutorial` (root-owned worktree requires disabled VCS stamping)
- `/usr/local/go/bin/go test ./internal/description -run '^$' -fuzz '^FuzzCodeRetabSourcePreservation$' -fuzztime=10s -parallel=2`
- `/usr/local/go/bin/go test ./.scratch/description-rendering/verification` and `git diff --check`

Coverage: `internal/tui/native_code_test.go` executes all 38 independently authored non-null Code edit goldens across `code-trigger-events`, `code-exit-boundaries`, `code-exit-nested`, `code-exit-list-entry`, `standalone-code-editing`, `exact-code-import`, `code-tabs-retab`, `retaball-native-scope`, `code-indent-outdent`. `code-line-selection` uses 06 origin/clamping tests plus 08 indentation/retab/atom/multi-wrapper selection regressions. Additional focused tests exercise config validation/CLI coexistence, grouped typed chunks, escaped first marker, Unicode width, source/wrapper preservation and next-input cursor/prefix outcomes. README, Help, keybindings and corpus traceability are synchronized. No fixture goldens were regenerated or changed.

No live user database, schema or dependency changed. Committed on 2026-10-10 after explicit user authorization. Real German-layout/Windows event and Vikunja/VOA captures remain explicit Ticket 10 work; synthetic PASS does not certify those environments. Next implementation ticket is 09.
