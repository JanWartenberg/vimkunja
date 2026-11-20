# Empty Project deletion

Status: implemented on 2026-11-18 following Jan's authorization. Recursive Task
deletion and trash semantics are explicitly deferred to the separate follow-up.

- `:projectdelete` takes no arguments and targets the Project marked in the
  Projects Pane, from any Pane. The heading is DELETE PROJECT; show its Title
  and ID, then the checked status in a separate paragraph. Before enabling y,
  read the current eligibility from the Store. Show "This has no Tasks or
  Subprojects" for eligible Projects, otherwise show the actual blocker.
  During checking or a blocked/error state, y must not submit deletion. r repeats
  the check; n/Esc can close a checking or blocked dialog. Ignore stale results
  after cancellation or replacement. The final delete still rechecks atomically.
- `y` confirms once; `n` or Esc cancels. Enter does not confirm. Repeated keys
  while saving cannot submit another deletion. Unsaved Task/Comment drafts block
  opening the confirmation.
- Permanently delete only Projects without Tasks, including Tasks in trash, and
  without Subprojects. Reject missing/invalid/archived Projects and inconsistent
  descendant relationships. Projects with Time Entries or a background file
  require VOA, preserving time history and filesystem handling.
- Reserve the SQLite writer and revalidate all guards within one transaction.
  Remove standard Views/Buckets, View assignments/positions, Project ancestors,
  counters/aliases, sharing, favorites/subscriptions, notifications and webhooks;
  clear Users' default-Project references. Use upstream entity discriminators.
  Do not modify the upstream schema, dependencies or local user database.
- Errors retain a cancellable confirmation with visible feedback. Success removes
  the Project, refreshes the hierarchy, focuses Projects and selects a remaining
  entry. The last Project may be deleted; the empty workspace supports creation.
  If refreshing fails after commit, remove the cached entry and report reload
  guidance, without retrying deletion. Discard responses from old contexts.

## Verification

Store tests cover active/done/trashed Tasks, children, metadata cleanup, rollback,
unsafe targets and title reuse. TUI tests cover one explicit confirmation, cancel,
drafts, errors, repeated input, remaining/last Project selection and reload failure.
The opt-in `TestProjectDeletionOriginalAppRoundtrip` uses a disposable database
and official Vikunja 2.7.0 to verify an upstream-created empty child can be deleted
and the surviving root and subsequent upstream writes remain usable.

Full tests, both native application builds, vet, focused TUI/Store race checks,
official creation/deletion round trips and a Windows cross-build pass. A Linux
PTY smoke check confirms root/child creation, deletion cancellation and deleting
child and root with y while retaining Inbox. The creation dialog now uses :w;
a follow-up PTY smoke verifies its Normal/Insert modes and local commands. Native Windows runtime
acceptance of deletion was confirmed by Jan. The subsequent creation-dialog
Normal/Insert changes were also accepted by Jan on 2026-11-20.

## Upstream references

- [Project deletion](https://github.com/go-vikunja/vikunja/blob/v2.7.0/pkg/models/project.go)
- [Favorite kinds](https://github.com/go-vikunja/vikunja/blob/v2.7.0/pkg/models/favorites.go)
- [Subscription entity types](https://github.com/go-vikunja/vikunja/blob/v2.7.0/pkg/models/subscription.go)

## Eligibility feedback follow-up — 2026-11-20

Jan requested that the modal answer whether this specific Project can be deleted,
instead of only stating a general rule. Public Store CheckProjectDeletion and the
transactional delete now share validation. Focused tests cover empty/non-empty
Projects, trash, Subprojects and all safety guards; no-write checks, stale
eligibility, loading/blocked input, read errors, rechecking and late replies.

## Final acceptance — 2026-11-20

Jan confirmed that acceptance passed on Windows and authorized committing the
final implementation, including the latest creation-dialog, Parent navigation
and deletion-eligibility feedback changes.
