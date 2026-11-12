# 01 — Copy the selected Task to the system clipboard

**What to build:** Add `:copy` to copy a readable plain-text representation of the single selected Task, including its Description and metadata (for example Title and Priority), to the system clipboard for quick sharing.

**Category:** enhancement

**Priority:** Low

**Status:** completed

- [x] Define which Task metadata to include and their stable display order/format.
- [x] Define the readable plain-text projection of HTML Descriptions, including lists, Checkboxes, Links and protected content; reuse the Description projection instead of copying raw HTML or adding a separate parser.
- [x] Define what happens when no Task or multiple Tasks are selected; copying is initially supported only for exactly one selected Task.
- [x] Define whether copy uses persisted values or the current Task Detail draft, including unsaved edits.
- [x] Define behavior for an empty Description and clipboard write failures; report failures through existing error feedback.
- [x] Check the keybinding contract and implemented clipboard actions before assigning an optional Leader-key shortcut. The integrated editor uses `Space+y` for text and `Space+e` for Link URLs (`Space+Y` remains an alias). Do not displace these actions.

## Further context

The integrated editor reads the system clipboard for paste and writes text and Link URLs to it. Task export reuses this clipboard infrastructure and the Native Description model. The accepted [spec](../spec.md) records the full contract. Comments remain excluded; a separate command or `:copy -c` is a possible future extension.

## Verification — 2026-11-11

Implementation and focused Ex-command/clipboard and Description projection regressions are in place. Full `go test ./...`, `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`, and `go vet ./...` pass. Three independent review rounds (bugs/risks and consistency/validation) are complete. Confirmed findings about failed Detail workflow reads and protected Code previews were fixed with regressions. The final round had no findings. Empty new-draft Titles are also omitted with regression coverage. Full tests, both builds and vet pass after the fixes. No native Windows/system-clipboard manual acceptance is claimed.

## Manual acceptance — 2026-11-12

Jan copied and pasted an existing Task with its identifier, Project, Status, Done, Priority, Due Date, Labels and Description. Unicode and completed Checkboxes were retained. Jan then copied a new unsaved draft with a Description, discarded it, and confirmed the Task was never created. These scenarios confirm actual clipboard output and copy-without-save behavior; no additional manual edge-case coverage is claimed.

## Commit preparation review — 2026-11-12

Review found that explicit ordered-list entry values were ignored by the sharing projection. Corrected explicit numbering and reversed-list progression, with focused regression tests. The manual acceptance above supplements the earlier automated verification.
