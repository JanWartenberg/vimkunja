# 01 — Show Checkbox-list progress summary

**What to build:** A Vimkunja user can optionally see a completed/total (`n/m`) summary for each Checkbox list embedded in a Task Description, without changing the enclosing Task's Done status.

**Implementation prerequisites:** The first read-only summary slice needs only checkbox-list recognition in the existing Description projection. Native rich-text editing remains a later prerequisite for synchronization after checkbox toggles and entry insertion/removal.

**Category:** enhancement

**Priority:** Medium

**Status:** needs-triage

- [ ] Define whether the summary appears before or after the Checkbox list and its narrow-terminal behavior.
- [ ] Implement the initial read-only summary from Vikunja Checkbox-list HTML.
- [ ] Count only the list's own checked and total entries.
- [ ] Keep the summary synchronized after checkbox toggles and entry insertion/removal once native editing exists.
- [ ] Never derive or mutate the enclosing Task's Done status from the summary.
