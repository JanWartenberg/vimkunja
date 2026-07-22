# 01 — Show Checkbox-list progress summary

**What to build:** A Vimkunja user can optionally see a completed/total (`n/m`) summary for each Checkbox list embedded in a Task Description, without changing the enclosing Task's Done status.

**Blocked by:** Rich-text Description rendering and Checkbox-list support.

**Category:** enhancement

**Status:** needs-triage

- [ ] Define whether the summary appears before or after the Checkbox list and its narrow-terminal behavior.
- [ ] Count only the list's own checked and total entries.
- [ ] Keep the summary synchronized after checkbox toggles and entry insertion/removal.
- [ ] Never derive or mutate the enclosing Task's Done status from the summary.
