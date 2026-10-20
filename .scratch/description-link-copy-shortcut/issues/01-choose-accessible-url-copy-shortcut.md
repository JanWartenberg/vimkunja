# 01 — Choose an accessible shortcut for Copy Link URL

**What to improve:** Copy a Description Link target without requiring Shift after the Space leader. Jan finds Space+Y inconvenient.

**Phase:** M3 — editor usability

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

**Order:** 1 of Jan's agreed next-work sequence. Follow with [subtask reconciliation](../../release-ticket-cleanup/issues/01-reconcile-open-subtasks-before-publication.md), then GitHub publication.

## Scope and next action

- [ ] Recheck the current Normal/Visual Description leader bindings and propose a free lowercase shortcut. Previously suggested candidates: Space+d (destination) or Space+e; neither is selected.
- [ ] Agree the binding with Jan before implementing it; decide whether Space+Y remains an alias.
- [ ] Reuse the existing Copy Link URL action and system clipboard path, preserving URL validation, cursor/selection behavior and unavailable-action feedback.
- [ ] Update the footer palette, help, keybinding contract and relevant tutorial hints consistently.
- [ ] Add focused key-dispatch regressions for the chosen binding and any retained alias.

## Boundary

This is an optional follow-up. Rich-text acceptance AT-01–10 is complete; the accepted Space+Y action remains available until a replacement is chosen. Do not change Space+y (text copy), Space+p (paste), or existing formatting/list shortcuts.

## Comments

2026-10-19 — Jan requests a separate Scratch ticket and closure of the accepted Rich-text work. Origin: [completed Rich-text follow-ups](../../../.archive/2026-10-19/description-rendering/issues/11-format-visual-selections-and-convert-touched-paragraphs.md).

2026-10-19 — Jan prioritizes this follow-up first, ahead of subtask cleanup and GitHub initialization/publication. The lowercase binding and legacy alias policy still need a decision.
