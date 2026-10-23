# 01 — Copy the selected Task to the system clipboard

**What to build:** Add `:copy` to copy a readable plain-text representation of the single selected Task, including its Description and metadata (for example Title and Priority), to the system clipboard for quick sharing.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Define which Task metadata to include and their stable display order/format.
- [ ] Define the readable plain-text projection of HTML Descriptions, including lists, Checkboxes, Links and protected content; reuse the Description projection instead of copying raw HTML or adding a separate parser.
- [ ] Define what happens when no Task or multiple Tasks are selected; copying is initially supported only for exactly one selected Task.
- [ ] Define whether copy uses persisted values or the current Task Detail draft, including unsaved edits.
- [ ] Define behavior for an empty Description and clipboard write failures; report failures through existing error feedback.
- [ ] Check the keybinding contract and implemented clipboard actions before assigning an optional Leader-key shortcut. The integrated editor uses `Space+y` for text and `Space+e` for Link URLs (`Space+Y` remains an alias). Do not displace these actions.

## Further context

The integrated editor reads the system clipboard for paste and writes text and Link URLs to it. Reuse this clipboard infrastructure for Task-level export. Copying must leave Task data and drafts unchanged. The adopted [spec](../spec.md) records the agreed initial scope; metadata formatting and unsaved-draft behavior remain open.
