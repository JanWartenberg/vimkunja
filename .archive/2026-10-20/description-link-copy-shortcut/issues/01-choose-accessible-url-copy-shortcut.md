# 01 — Choose an accessible shortcut for Copy Link URL

**What to improve:** Copy a Description Link target without requiring Shift after the Space leader. Jan finds Space+Y inconvenient.

**Phase:** M3 — editor usability

**Category:** enhancement

**Priority:** Medium

**Status:** completed

**Order:** 1 of Jan's agreed next-work sequence. Follow with [subtask reconciliation](../../../../.scratch/release-ticket-cleanup/issues/01-reconcile-open-subtasks-before-publication.md), then GitHub publication.

## Scope and next action

- [x] Recheck the current Normal/Visual Description leader bindings and propose a free lowercase shortcut. Jan selected `Space+e`; `Space+d` was rejected.
- [x] Agree the binding with Jan before implementing it; decide whether Space+Y remains an alias.
- [x] Reuse the existing Copy Link URL action and system clipboard path, preserving URL validation, cursor/selection behavior and unavailable-action feedback.
- [x] Update the footer palette, help, keybinding contract and relevant tutorial hints consistently.
- [x] Add focused key-dispatch regressions for the chosen binding and any retained alias.

## Boundary

This is an optional follow-up. Rich-text acceptance AT-01–10 is complete; the accepted Space+Y action remains available until a replacement is chosen. Do not change Space+y (text copy), Space+p (paste), or existing formatting/list shortcuts.

## Comments

2026-10-19 — Jan requests a separate Scratch ticket and closure of the accepted Rich-text work. Origin: [completed Rich-text follow-ups](../../../2026-10-19/description-rendering/issues/11-format-visual-selections-and-convert-touched-paragraphs.md).

2026-10-19 — Jan prioritizes this follow-up first, ahead of subtask cleanup and GitHub initialization/publication. The lowercase binding and legacy alias policy still needed a decision at that point.


### 2026-10-20 — implemented on merged main

Jan selected `Space+e` for Copy Link URL, keeping `Space+Y` as a compatibility alias. Only `Space+e` appears in the palette, Link hints and integrated help. The Leader timeout remains 500 ms; `Space+y/p` and Insert-only prompts are unchanged. README, keybinding reference and agent guidance match the implementation. No affected tutorial hint exists in the current curriculum.

TDD: the new `e` dispatch first failed with a missing clipboard command; the rendered-hint test failed on the old `Space+Y` hint before the display change. UI regressions cover both bindings in Normal and character/block Visual modes, successful/failed clipboard writes, unchanged cursor/selection/draft, and unavailable-Link feedback. Existing shared text copy/paste regressions pass.

Validation passed: `go test ./... ./.scratch/description-rendering/verification`, `go vet ./...`, and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`. The environment needs `/usr/local/go/bin` for Go and a command-scoped Git safe-directory exception for VCS build metadata. Native Windows keyboard/clipboard acceptance of the new binding has not been repeated. No commit or merge performed.

Independent review round 1 (bugs/risks and consistency/validation) found no code issues. Both reviewers flagged that the archive ignore rule hid the moved ticket from the diff; explicit intent-to-add now includes the destination in the pending change.

Fresh independent review round 2: both bugs/risks and consistency/validation report no findings on the complete diff, including the archived ticket. Both independently reran the focused Link-copy tests successfully; final diff whitespace check passed.
