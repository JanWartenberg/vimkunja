# 01 — Delete empty Projects after confirmation

**Status:** completed
**Category:** enhancement

Jan authorized the bounded first step on 2026-11-18: Projects without Tasks can
be deleted after one confirmation. This implementation requires no Subprojects
as well, preserving hierarchy until recursive semantics are decided.

- [x] Implement `:projectdelete` with one explicit confirmation and cancellation.
- [x] Show checked eligibility or the concrete blocker before offering deletion (2026-11-20 feedback).
- [x] Revalidate empty leaf status and safety guards transactionally.
- [x] Remove associated metadata without upstream schema changes.
- [x] Cover errors, rollback, drafts and post-success focus with focused tests.
- [x] Add a disposable official VOA round-trip test.
- [x] Record recursive/trash investigation in the separate follow-up ticket.

See the [specification](../spec.md). Jan confirmed Windows acceptance and authorized the commit on 2026-11-20.
