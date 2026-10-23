# 01 — Evaluate independent Link label and URL editing

**What to decide:** Determine whether Vimkunja should eventually offer a dedicated editor for an existing Link that can edit its visible text and URL independently. The current existing-Link prompt edits only the URL; new-Link insertion already has separate URL and text steps.

**Implementation prerequisites:** Met — the rich-text Description editor and `Space+l` URL editing are implemented and accepted. Existing Links have a URL prompt; inserting a new Link outside a selection/Link already asks for URL and then visible text. This ticket concerns independent editing of an existing Link's text and URL.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Compare the URL-only interaction with a two-field terminal dialog through a focused prototype.
- [ ] Define selection, existing-Link, cancellation, validation, and empty-URL behavior.
- [ ] Retain Vikunja-compatible HTML serialization and preserve existing Link attributes.
- [ ] Do not displace the initial URL-only behavior without a demonstrated usability benefit.
