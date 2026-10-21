# 01 — Discard obsolete asynchronous navigation results

**Category:** bug
**Priority:** High — publication prerequisite
**Status:** completed

Implemented from `d241b0a` in `fix/stale-navigation-results-v2`, independently reviewed and crosschecked. Jan requested integration and a commit on `main` on 2026-10-21; this ticket is archived with that commit. Publication remains unauthorized.

## Contract and reproduction

Start Project A / View 10 loading, navigate to Project B / View 20, finish B, then deliver A. A must not replace B's Tasks, loading/error state, selection or drafts. The same rule applies to obsolete Views, Kanban and Overview success/error responses, including A→B→A and cross-View navigation.

## Implementation

- Separate monotonic View-metadata requests and a shared generation for List, Kanban and Overview loads; reject obsolete responses before any state mutation.
- Invalidate Tasks while replacement Views are pending, and invalidate requests when Projects become empty, the Kanban View picker opens or Search restores its saved origin.
- Capture Overview Label and page size before command execution.
- Entering Overview invalidates earlier View metadata; returning with no View reloads metadata before Tasks or a Kanban Board. Project changes clear old View metadata so commands cannot reuse another Project's Views. Fresh metadata requests made inside Overview preserve its independent Task load.

## Evidence — 2026-10-21

- Original audit reproduction failed before implementation: current Project 2 / View 20 displayed Project 1 Tasks.
- Deterministic regressions in `internal/tui/navigation_load_test.go` reorder real command results without sleeps. They cover four result families, old success/error, current pending/completed, A→B and A→B→A, scope changes, pending Views, empty Projects, View picker, Search restore, current errors and captured Label identity.
- Baseline overlay against `d241b0a` confirms the regressions fail on the original implementation.
- Final pre-commit checks on `main`: `go test ./...` PASS; `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial` PASS; `go vet ./...` PASS; `git diff --check` PASS. Focused navigation regressions and the full TUI suite also pass after the crosscheck corrections. Go executable: `/usr/local/go/bin/go` (not present on the shell PATH).
- Independent review found one missing Views→Overview invalidation; a failing success/error regression preceded the correction.
- The pre-commit crosscheck then reproduced metadata refresh cancelling Overview and commands reusing or losing Project metadata after Overview. New regressions preceded corrections and cover return through `:list` and `:kanban`, including subsequent `:list` navigation. Metadata refresh in Overview now leaves its Task request valid. The final independent review reports no further concrete findings.

## Integration and limitations

- Integration into `main` and archival are included in the commit requested by Jan. The original unrelated Clipboard spec remains untouched. Publication preparation may use the resulting `main` commit; no push was requested.
- In the isolated worktree, Go attempted VCS stamping in the unrelated, untrusted `/workspace` repository; builds there passed with `-buildvcs=false`. The regular build on `main` passes without that workaround. No global trust configuration was changed.
- No manual terminal acceptance claimed. Mutation completion handlers (including Kanban provisioning) have a separate stale-navigation policy risk; persisted mutations need a distinct contract and were not changed by this read-result fix.
- No user database, schema or dependencies changed. Only the reviewed fix, regressions, STATUS and this ticket are included in the requested `main` commit.
