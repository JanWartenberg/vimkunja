# 02 — Specify native Rich-Text Description rendering and editing

**Phase:** M3 — structured reader and editor

**What to decide:** Define the complete M3 scope for a native, keyboard-first terminal projection and editor for Vikunja HTML Task Descriptions.

**Implementation prerequisites:** M2's read-only HTML-to-plain-text reader is complete; see its [archived ticket](../../../.archive/2026-08-05/description-rendering/issues/01-render-vikunja-html-descriptions.md). The decisions in this ticket govern the M3 implementation; the existing [Rich-Text Description Contract](../spec.md) is the starting point and must be updated only when a decision here changes it.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

**Scope boundary:** This is phase 2 of the Description plan. It replaces neither Vikunja's HTML persistence format nor its SQLite schema. It aims to natively render and edit every HTML structure that is useful and feasible in a terminal. Content that is not suitably editable must remain as readable as practical and must be preserved without destructive rewriting.

## Questions for dedicated grilling

The contract already establishes Source HTML as the persistence format, a structured document model, safe preservation of opaque nodes, and an initial native set of headings, inline marks, lists, Checkbox lists, code blocks, quotes, rules, and links. Resolve these remaining product decisions one at a time:

1. Which Vikunja/Tiptap structures beyond the contract's initial native set are useful enough to render and edit natively in M3, especially images, attachments, mentions, embeds, and complex tables?
2. For every unsupported structure, which support tier applies: natively editable, readable but opaque/read-only, neutral fallback while preserved, or intentionally not rendered?
3. What exact preservation guarantee is required when an edit changes surrounding content containing opaque nodes, attributes, or unknown extensions?
4. How should malformed, partially parseable, or unsafe HTML be presented, and when must the entire Description become read-only to prevent a destructive save?
5. What terminal projection conventions make headings, emphasis, nested lists, quotes, code, tables, links, and overflow readable at narrow widths without conflating visual layout with document meaning?
6. Does prototype testing validate the proposed `Space`-led formatting palette and Visual-Mode bindings, or should their interaction model change before implementation?
7. What are the final link rules: display convention, URL validation, editing an existing link, empty-URL behavior, and feedback for invalid input?
8. Which serialization normalizations are acceptable while remaining Vikunja-compatible, including whitespace, attribute order, Checkbox-list IDs, and link attributes?
9. What compatibility corpus and acceptance evidence are required to establish that existing Vikunja-created HTML is rendered usefully and survives unrelated Vimkunja edits?

## Out of scope

- The M2 HTML-to-plain-text reader and its nine presentation decisions.
- Changing the Vikunja SQLite schema or introducing a second persistence format.
- Persisted Task-action undo; Description-local draft undo/redo remains part of this M3 scope.
