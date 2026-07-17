# 01 — Show Workflow Status in Task Detail

**What to build:** A Vimkunja user opening an existing Task in the Detail-pane can see its current Workflow Status and the available Bucket choices from the project's single manual Kanban View. When no such View exists, or when more than one exists, the Detail-pane explains that Status is unavailable or ambiguous without making a silent selection.

**Blocked by:** None — can start immediately.

**Status:** resolved

- [x] An existing Task's current Bucket from its single manual Kanban View is shown as its Workflow Status in the Detail-pane.
- [x] Available Status choices preserve the persisted Bucket titles and ordering.
- [x] Projects without a manual Kanban View show an explicit read-only unavailable state.
- [x] Projects with multiple manual Kanban Views show an explicit read-only ambiguous state.
- [x] Store tests verify persisted Workflow Status reading and the unavailable/ambiguous cases against temporary SQLite databases.
- [x] TUI model tests verify that opening a Task exposes the resulting Status state without rendering-specific assertions.
- [x] `go test ./...` passes.
