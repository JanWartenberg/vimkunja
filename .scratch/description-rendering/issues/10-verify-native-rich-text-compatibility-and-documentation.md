# 10 — Native Rich-text compatibility, terminal acceptance and documentation

**What to build:** Gather actual Vikunja and terminal evidence, run the integrated native contract suite and document/tutorial the implemented feature before declaring M3 Rich-text complete.

**Phase:** M3 — native Rich-text Description

**Category:** enhancement

**Priority:** Medium

**Status:** ready-for-agent

**Implementation dependencies satisfied:** 03–09 completed on `feature/description-model`; 09 is verified and committed 2026-10-10. Integrated acceptance, capture/provenance and tutorial work can proceed.

**Contract:** [spec.md](../spec.md), [decision/handoff ticket 02](02-specify-native-rich-text-description-rendering-and-editing.md), [corpus and traceability](../corpus.md), [83 synthetic fixtures](../corpus.json), [keybinding contract](../../../docs/keybindings.md).

## Scope

- Start versioned VOA capture preparation early; it does not require waiting for the whole editor. Record actual getHTML output, Vikunja/Tiptap versions, capture steps and provenance, including Checkboxes, nested structure, Links, Code and Mentions. Never relabel synthetic examples as captured output.
- Add captures and minimized regressions with independent expected behavior. Golden changes require an explicit diff/contract explanation; do not bulk-regenerate from the serializer under test.
- Execute all relevant fixture classification/projection/serialization/edit assertions against the implemented modules. Report per-rule coverage and explicitly missing/unexecutable cases, not just an aggregate green test count.
- Add semantic Native round-trip properties, byte/identity/order/parent Opaque preservation properties and a small bounded parser/serializer fuzz run in normal CI. Discovered failures become deterministic regressions; no long scheduled fuzz infrastructure.
- Exercise save/discard/conflict/combined-save/UUID candidate failure flows end to end using disposable databases and confirm unchanged Task Done and unrelated editor surfaces.
- Record wide/narrow/short native TUI evidence: disclosure reflow, cursor anchoring, protected atoms, quote/list nesting, Code/Table scroll, soft/hard wrap, palette threshold and too-narrow draft preservation. HTML documentation screenshots are not native TUI execution evidence.
- Capture real German keyboard/Windows terminal events and interaction outcomes for accepted Code keys, AltGr/dead keys/Paste and literal-Tab alias. Record unavailable environments as outstanding evidence, not PASS or another product questionnaire.
- Update Help only for implemented bindings, docs/keybindings status, README usage and a dedicated markup-tutorial section using the real UI. Keep current tutorial isolated in disposable data and preserve existing lesson flows.

## Acceptance and validation

- Complete integration requires tickets 03–09; captures/case preparation may begin earlier. Attach a concise report distinguishing automated Native results, real VOA outputs, practical terminal results and remaining limitations.
- All 83 current synthetic examples are discoverable in the test matrix; new versioned captures are clearly separate. Random IDs are checked structurally (UUID v4, uniqueness, stability), not as literal strings.
- A documentation/review template may use data-task-id="<randomUUID>" as an explicitly illustrative placeholder. It is not a valid persisted UUID or executable golden. L10 visible Alpha/Beta/child output is deterministic despite this placeholder.
- Native tests, configured bounded fuzz, full go test ./... and both entrypoint builds pass. Real VOA/Windows checks are required for their respective compatibility claims; inability to capture is documented and never waived by synthetic probes.
- Help/README/tutorial teach syntax, selections, Code exits/Tabs, Opaque confirmation, recover/discard and conflict handling without raw-HTML editing. Only then report the supported feature as complete.

**Focused coverage:** Entire corpus and current traceability matrix, versioned captured VOA cases, fixture-derived compound editing journeys and real terminal evidence.

Run affected tests during implementation; before a commit run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The existing artifact probes are supplemental, not Native execution evidence. Required test commands and fixture mapping must be recorded in the completion comment.

## Boundaries

Use the accepted product rules, including the smallest reliable Opaque protection and explicit metadata-loss exceptions. Preserve Vikunja HTML persistence and the existing SQLite schema. Do not change `internal/db/vikunja.db`, dependencies or generic Task Attachments. This ticket preparation does not start production implementation. Keep Title, plain-text editing and Comment composer behavior intact outside the explicitly affected Description paths.

## Comments

### 2026-10-05 — implementation preparation

Created after the user's rule-card review and acceptance of the current product-rule set. Remaining uncertainty is implementation and evidence work, not a new product questionnaire. Dependencies order the safe rollout; synthetic examples never stand in for real terminal/VOA execution.

### 2026-10-10 — Ticket 08 terminal evidence handoff

Synthetic Code tests cover grouped Text, ü/Ü, ]Space/[Space, backtick chunks, Paste/clipboard CRLF and literal-Tab prefix dispatch. No real German-layout or Windows decoder capture has been performed. Record actual AltGr-8/9, dead-key three/four backticks, ü/Ü, Ctrl+V interception and Ctrl+Q/Ctrl+B events and interaction outcomes here before terminal acceptance. Ctrl+Q and Ctrl+B have no existing application binding conflicts in Code Insert; terminal flow-control/interception remains an environment check. `tui.literal_tab_alias` defaults to `ctrl+q`, accepts `ctrl+b` or empty; no OS detection.

### 2026-10-10 — Ticket 09 inline execution handoff

Inline syntax, Inline-Code and Links are implemented with actual TUI dispatch/prepared-save execution for 40 goldens across eleven focused records, scoped Draft regressions and temporary-SQLite save adoption. Help, README and keybindings describe the implemented actions. Gather the remaining integrated per-rule evidence and tutorial here; all existing corpus records remain synthetic. Real clipboard-result handling is tested through an injected writer, not a host clipboard/environment acceptance claim.
