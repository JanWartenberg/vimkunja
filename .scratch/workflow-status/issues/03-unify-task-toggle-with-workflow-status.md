# 03 — Unify Task Toggle with Workflow Status

**What to build:** A Vimkunja user pressing `x` in the Tasks-pane uses the same coherent Workflow Status semantics as the Detail-pane: an open Task moves to its configured Done Bucket and becomes done; a done Task moves to its configured default Bucket and becomes open. The list refreshes immediately and no reopened Task remains assigned to a Done Bucket.

**Blocked by:** 02 — Edit Workflow Status in Task Detail.

**Status:** ready-for-agent

- [ ] `x` on an open Task persists its configured Done Bucket assignment and global completed state through the shared workflow operation.
- [ ] `x` on a done Task persists its configured default Bucket assignment, clears global completed state and Done timestamp, and bumps the updated timestamp.
- [ ] The existing per-View Done-Bucket synchronization remains Vikunja-compatible where applicable.
- [ ] A successful toggle refreshes the current Task list; a failed toggle retains visible state and reports the existing error feedback.
- [ ] Existing regression coverage that expects a reopened Task to remain in a Done Bucket is replaced with the new Default-Bucket behavior.
- [ ] Store and TUI model tests verify both toggle directions and failed-write behavior without asserting implementation details.
- [ ] A manual smoke check confirms To-Do, Doing, Done, and `x` transitions match the Original App's persisted state.
- [ ] `go test ./...` passes.
