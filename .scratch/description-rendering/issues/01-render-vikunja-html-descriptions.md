# 01 — Render Vikunja HTML descriptions as readable terminal text

**Phase:** M2 — reader-only

**What to build:** A Vimkunja user can read a Task Description created in Vikunja's rich-text editor without seeing raw HTML tags.

**Chosen strategy:** Extract readable plain text from Vikunja HTML. This ticket introduces no new Description writing or HTML serialization behavior.

**Scope boundary:** This is phase 1 of the Description plan. Native structured rendering and editing, Visual Mode, formatting commands, and preservation-aware serialization belong to the separate M3 ticket [02 — Specify native Rich-Text Description rendering and editing](02-specify-native-rich-text-description-rendering-and-editing.md) and its companion contract [`../spec.md`](../spec.md).

**Implementation prerequisites:** None — deliberately deferred until after the Fundament/MVP.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

## Questions for dedicated grilling

The plain-text strategy is decided. Resolve these nine presentation decisions one at a time before implementation:

1. Which block elements (`p`, `div`, headings, `br`) create paragraph or line breaks, and how is whitespace normalized?
2. How are ordered, unordered, and nested lists represented in plain terminal text?
3. How are links represented: visible text only, or visible text plus URL when they differ?
4. How are images and other media represented: alt text, URL, neutral placeholder, or a combination?
5. How are tables reduced to readable text without raw markup or misleading column structure?
6. How are Vikunja Checkbox lists displayed read-only, including checked state?
7. What is the fallback for unknown, unsupported, malformed, or structurally empty HTML?
8. Which entities, control characters, and terminal escape sequences require decoding, normalization, or safe suppression?
9. What is the exact empty-state behavior, and how must existing non-HTML plain-text descriptions behave?

## Further context

Rich-text Description editing is deliberately outside the current Daily-driver milestone. If raw HTML prevents basic reading before that work resumes, create a separate, narrowly scoped reader-only ticket (for example readable paragraphs with tags removed); do not use it to begin the structured editor.

Vikunja stores rich-text descriptions as HTML. The previous UX note recommended a deliberately small first step (plain text with tags removed) before considering HTML-to-Markdown conversion or a custom renderer.
