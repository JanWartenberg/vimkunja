# 02 — Edit Workflow Status in Task Detail

**What to build:** A Vimkunja user can select and save a Workflow Status through the Task Detail-pane using the established keyboard picker and save/discard flow. The persisted Bucket assignment and the Task's global Done state remain consistent: the configured Done Bucket completes the Task; every other Bucket reopens it. New Tasks and Tasks moved to another Project retain valid default Workflow Statuses.

**Blocked by:** 01 — Show Workflow Status in Task Detail.

**Status:** resolved

- [x] Status participates in the Detail editor's vertical field navigation and opens a keyboard picker with `i`.
- [x] `j`/`k` and arrows select, `Enter` confirms, and `Esc` cancels the Status picker consistently with existing pickers.
- [x] A confirmed Status selection is dirty state and is included in `:w`, `:wq`, `:wqa`, `:q`, and `:q!` behavior.
- [x] Saving the configured Done Bucket atomically persists that Bucket, `done=true`, a Done timestamp, and an updated timestamp.
- [x] Saving any non-Done Bucket atomically persists that Bucket, `done=false`, no Done timestamp, and an updated timestamp.
- [x] Failed writes preserve dirty editor state and use the existing error feedback; successful writes refresh the visible Task state.
- [x] New Tasks and cross-Project Task moves use their target Project's configured default Bucket and do not retain invalid source Statuses.
- [x] Store and TUI model tests cover successful changes, cancellation, dirty protection, save/discard, failure retention, and default initialization.
- [x] `go test ./...` passes.
