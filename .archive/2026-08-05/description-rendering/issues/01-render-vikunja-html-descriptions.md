# 01 — Render Vikunja HTML descriptions as readable terminal text

**Phase:** M2 — reader-only

**What to build:** A Vimkunja user can read a Task Description created in Vikunja's rich-text editor without seeing raw HTML tags.

**Chosen strategy:** Extract readable plain text from Vikunja HTML. This ticket introduces no new Description writing or HTML serialization behavior.

**Scope boundary:** This is phase 1 of the Description plan. Native structured rendering and editing, Visual Mode, formatting commands, and preservation-aware serialization belong to the separate M3 ticket [02 — Specify native Rich-Text Description rendering and editing](../../../2026-10-19/description-rendering/issues/02-specify-native-rich-text-description-rendering-and-editing.md) and its companion contract [`../../../../.scratch/description-rendering/spec.md`](../../../2026-10-19/description-rendering/spec.md).

**Implementation prerequisites:** None — deliberately deferred until after the Fundament/MVP.

**Category:** enhancement

**Priority:** Medium

**Status:** implemented

## Accepted reader contract

- `<p>` renders as a paragraph without an added blank line; headings have one following blank line; `<br>` creates an additional line break. Normal HTML-text whitespace is normalized and consecutive empty paragraphs collapse to one empty paragraph. `<div>` has no formatting semantics and renders descendants recursively.
- Unordered entries use `- `; ordered entries use their ordinal and `. `; nesting indents by two spaces per level; entries have no inter-item blank lines.
- A Link with distinct visible text renders as `text (URL)`; an URL-only or textless Link renders once as its URL.
- Images render as `[Bild: Alt-Text]` or `[Bild]`; media URLs are omitted; other unsupported media render as `[Medieninhalt]`.
- Table rows render as one line with cells separated by ` | `, without borders, alignment, or special header treatment.
- Checkbox-list entries render as `- [ ] Text` or `- [x] Text`, use normal list indentation, and remain read-only. Their state is distinct from the enclosing Task's Done state.
- Unknown parseable elements render descendant content recursively. An element without readable content renders as `[Unsupported content]`. Malformed HTML renders safely extractable text, or `[Unreadable description content]` if none is safe. Raw HTML tags never render as fallback.
- HTML entities are decoded and ordinary Unicode is retained. Source newlines, tabs, and repeated HTML whitespace normalize as normal text. Other C0/C1 controls, including terminal escape sequences, are removed.
- Empty, whitespace-only, or structurally unreadable content renders as the existing `(no description)` placeholder. Existing non-HTML Plain Text remains Plain Text with user-authored line breaks intact; only dangerous control characters are removed.

M2 is a display-only, deliberately lossy projection. It never serializes or writes Description HTML. A richer terminal-table projection and all structured Description editing remain M3 work.

## Further context

Rich-text Description editing is deliberately outside the current Daily-driver milestone. If raw HTML prevents basic reading before that work resumes, create a separate, narrowly scoped reader-only ticket (for example readable paragraphs with tags removed); do not use it to begin the structured editor.

Vikunja stores rich-text descriptions as HTML. The previous UX note recommended a deliberately small first step (plain text with tags removed) before considering HTML-to-Markdown conversion or a custom renderer.

## Comments

### 2026-08-05 — Implementation and manual acceptance

Implemented and manually accepted in commit `7c0f591`:

- Vikunja HTML renders as the specified safe, read-only terminal-text projection.
- HTML Descriptions are scrollable in Task Detail and cannot expose raw HTML for editing.
- Renderer and TUI behavior are covered by `go test ./...`.

### 2026-08-04 — Grilling, question 1: block boundaries and whitespace

Decided so far:

- `<p>` renders as a paragraph without an added blank line.
- Headings render as paragraphs with one blank line.
- `<br>` renders as an additional line break.
- Normal text whitespace is normalized.
- Consecutive empty paragraphs collapse to one empty paragraph.
- `<div>` has no formatting semantics; its text content and descendants render recursively without an additional forced break.

### 2026-08-04 — Grilling, question 2: lists

- Unordered list entries render with `- `.
- Ordered list entries render with their ordinal followed by `. `.
- Nested list levels indent by two spaces per level.
- Entries follow directly without blank lines between them.

### 2026-08-04 — Grilling, question 3: links

- A link with distinct visible text renders as `text (URL)`.
- A link whose visible text already is its URL renders once as the URL.
- A link without readable visible text renders as its URL.

### 2026-08-04 — Grilling, question 4: images and media

- Images with alt text render as `[Bild: Alt-Text]`; images without it as `[Bild]`.
- Media URLs are omitted.
- Other unsupported media render as `[Medieninhalt]`.

### 2026-08-04 — Grilling, question 5: tables

- Each table row renders as one line, with cells separated by ` | `.
- M2 adds no ASCII borders, column alignment, or special header treatment.
- A richer terminal-table projection remains a possible later improvement.

### 2026-08-04 — Grilling, question 6: Checkbox lists

- Unchecked Checkbox-list entries render as `- [ ] Text`; checked entries as `- [x] Text`.
- Nested entries retain the two-space-per-level indentation rule.
- M2 is read-only. A Checkbox-list state is distinct from the enclosing Task's Done state.

### 2026-08-04 — Grilling, question 7: unsupported and malformed HTML

- Unknown but parseable elements render descendant content recursively without their own formatting.
- An unknown element without readable content renders as `[Unsupported content]`.
- Malformed HTML is parsed tolerantly and exposes any safely extractable text; otherwise it renders as `[Unreadable description content]`.
- Raw HTML tags are never a rendering fallback.

### 2026-08-04 — Grilling, question 8: entities and terminal safety

- HTML entities are decoded; ordinary Unicode remains unchanged.
- Source newlines, tabs, and repeated whitespace are normalized by the normal-text whitespace rule.
- Other C0/C1 controls, including terminal escape sequences, are removed before rendering.

### 2026-08-04 — Grilling, question 9: empty and existing Plain-Text Descriptions

- Empty content, whitespace-only content, and HTML without readable content render as the existing `(no description)` placeholder.
- Existing content without HTML structure remains Plain Text; it is not forced through HTML semantics or rewritten.
- User-authored Plain-Text line breaks remain intact. Only dangerous control characters are removed.
