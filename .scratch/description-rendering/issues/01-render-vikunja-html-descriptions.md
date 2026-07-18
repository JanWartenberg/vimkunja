# 01 — Render Vikunja HTML descriptions as readable terminal text

**What to build:** A Vimkunja user can read a Task Description created in Vikunja's rich-text editor without seeing raw HTML tags, while preserving as much useful structure as the chosen rendering strategy supports.

**Blocked by:** None — deliberately deferred until after the Fundament/MVP.

**Category:** enhancement

**Status:** needs-triage

- [ ] Decide whether the first version strips HTML to plain text, converts HTML to Markdown-like text, or renders a supported HTML subset directly.
- [ ] Paragraphs, emphasis, lists, links, and HTML entities have defined terminal behavior.
- [ ] Unsupported markup degrades to readable text rather than literal tags where practical.
- [ ] Description scrolling and wrapping remain consistent with the Detail-pane.

## Further context

Vikunja stores rich-text descriptions as HTML. The previous UX note recommended a deliberately small first step (plain text with tags removed) before considering HTML-to-Markdown conversion or a custom renderer.
