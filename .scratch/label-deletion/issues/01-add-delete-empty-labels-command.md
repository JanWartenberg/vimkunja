# 01 — Add a command to delete empty Labels

**What to build:** The global `:labels` view provides a deliberate command for deleting every Label with no Task associations.

**Implementation prerequisites:** Needs command name, confirmation, and failure-reporting semantics.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Specify the command grammar and whether the operation previews affected Labels.
- [ ] Require explicit confirmation before deleting Labels.
- [ ] Preserve non-empty Labels and report partial failures without changing Vikunja schema.
