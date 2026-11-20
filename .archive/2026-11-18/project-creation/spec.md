# Project creation

Status: completed on 2026-11-18 through implement-and-verify, following Jan’s
acceptance on 2026-11-17. Test seams:
public Store methods on disposable SQLite, TUI Update/View, and the official VOA API.

## Scope and entry points

- Create both Top-level Projects and Subprojects. A Subproject is the same Project
  entity with a parent assigned. Moving or deleting existing Projects is separate.
- `:projectnew` opens the creation dialog from any Pane with no parent selected.
  The currently selected Project never implicitly becomes its parent.
- `O` (`Shift+O`) opens the same dialog only in the focused Projects Pane.
  Lowercase `o` and `:new` retain their existing Task-creation behavior.
- Prioritize `O new project` in the Projects-Pane footer; secondary hints may be
  omitted at narrow widths using the existing footer layout.
- `:projectnewchild` opens the same dialog with the Project selected in the
  Projects Pane preselected as parent. No selection or an ineligible selected
  Project produces an error; do not silently create a Top-level Project instead.

## Dialog and validation

- Only two input fields: Title (required) and Parent Project (optional).
  Description, color and Project identifier are outside this creation workflow;
  setting them still requires VOA.
- Select Title initially; use the shared cursor renderer when entering Insert.
- Trim outer whitespace from Title, preserving internal whitespace. The trimmed
  Title must contain 1–250 Unicode code points.
- Reject a new Title matching any existing Project Title, including archived
  Projects, globally rather than only among siblings. Compare case-insensitively
  after trimming outer whitespace from both titles.
- Existing duplicate Titles remain unchanged and fully readable. Project identity
  and selection use IDs, never Titles. Show hierarchy paths and Project IDs to
  distinguish equal Titles in the Parent picker.
- This is intentionally stricter creation validation than VOA. It preserves
  Vikunja's data model and read compatibility without claiming identical behavior.
  Official VOA 2.7.0 API checks confirmed duplicate Titles are accepted both at
  the top level and under the same parent (HTTP 201 in each case).
- Parent selection uses a searchable hierarchy picker with a no-parent option.
  Archived Projects, descendants of archived Projects, and malformed hierarchy
  entries are not eligible parents.
- Start in Normal mode with Title selected, following Task metadata editing.
  `j/k`, arrows or `Tab`/`Shift+Tab` select Title/Parent. `i` edits Title or opens
  Parent; Enter also opens Parent. Title shares Task Normal text motions,
  `a/I/A`, `v/V` selection and `Space+y/p` clipboard behavior. Insert uses the
  shared cursor. Enter/Esc leave Insert without persisting; Normal Esc clears
  pending text selection/motions and keeps the dialog open.
- `:w` (and `:wq`) creates and opens the resulting Project's List. `:q!` discards
  directly. `:q` closes an unchanged draft, but asks before discarding changed
  Title or Parent: `y` discards, `n`/Esc keeps editing with all inputs retained.
  The initial preselected Parent in `:projectnewchild` is not itself a change.
  There are no Create/Cancel action rows; mode and controls appear in the dialog.
- In the Parent picker, `j/k`, arrows and `Ctrl+N/P` navigate; `gg`/`G`
  jump to first/last filtered choice and `Ctrl+U/D` move half the visible choice
  rows, clamping at both ends. Paging also works in search input. `/` starts search,
  Enter/Esc finishes search retaining its filter, Enter chooses and Esc returns
  to the form without applying the highlighted choice.
- Validation errors appear at the affected field. Retain the dialog and all input
  for correction. Other save failures also retain the dialog and input.
- Block opening creation while an existing Task draft has unsaved changes; explain
  that the Task must be saved or discarded first. Do not displace the draft.

## Persistence and resulting context

- Require exactly one local User; use that User as the new Project owner. Otherwise
  report a clear error instead of guessing or introducing a User picker.
- Create standard List, Gantt, Table and manual Kanban Views with the upstream
  standard Buckets. Table/Gantt rendering remains outside this feature.
- Append the new Project after its siblings, at the top level or under its parent.
- Persist Project, ancestor relationships, Views, Buckets and required supporting
  records in one transaction using Vikunja 2.7.0 semantics. Do not change the
  upstream schema, dependencies or compatibility guards.
- Revalidate Title uniqueness and Parent existence/eligibility immediately before
  writing within the transaction. Concurrent/external changes must cause a clear
  failure rather than stale or partially persisted data.
- On success refresh the hierarchy, expand ancestors as needed, select the new
  Project and focus its List View in the Tasks Pane.

## Verification

- Store regressions: root/child creation; ownership; ancestors; sibling ordering;
  standard Views/Buckets; Title trimming/length/case-insensitive global uniqueness,
  including archived Projects and imported duplicates; parent eligibility;
  external changes; and full rollback after an injected failure.
- TUI regressions: both commands; `O` only in the Projects Pane; unchanged Task
  shortcuts; safe no-parent default; child-command errors; searchable Parent
  selection; keyboard flow/cursor; validation; cancellation; draft protection;
  error retention; success focus/selection and narrow footer behavior.
- Verify created root/child Projects and their Views/Buckets with the official
  VOA 2.7.0 server in disposable databases, alternating access with Vimkunja.
- Update README, help and keybinding reference when implementing. Run affected
  tests; full tests and both builds before any implementation commit.

## Verification result — 2026-11-18

- Full `go test ./...`, both application builds and `go vet ./...` pass.
- Focused Store and TUI Project-creation race tests pass.
- All three opt-in official Vikunja 2.7.0 round trips pass, including the new
  `TestProjectCreationOriginalAppRoundtrip`: root/child/grandchild ownership,
  hierarchy, standard Views/Buckets, and subsequent upstream writes are checked
  through the public Store and official HTTP API in disposable databases.
- A Linux PTY smoke test creates a Top-level Project with `O`, creates a child
  with `:projectnewchild`, returns to List and exits cleanly; parent relationships
  and standard Views persist. Windows cross-build passes. Native Windows runtime
  acceptance was pending at that check; Jan accepted the final dialog on 2026-11-20.
- Two independent review rounds completed. The first round found hidden errors
  in short terminals and stale Kanban responses overriding the new List context;
  both were fixed with regressions. Both fresh second-round reviewers reported
  no findings. Invalid Parent-ID handling is also covered.
- README, help, keybinding reference and the disposable tutorial database are
  updated. Dependencies and the application database schema are unchanged.
- Jan authorized the implementation commit after Windows acceptance passed on 2026-11-20.

## Windows feedback and dialog follow-up — 2026-11-18

Jan confirmed Project creation, Parent selection and Subproject creation in a
Windows terminal, and later confirmed creation and empty-Project deletion. Jan
then requested Task-style Normal/Insert editing, :w persistence, :q! discard and
a confirmation on :q for changed input. This supersedes the intermediate
Enter-to-create workflow. Focused Update/View regressions cover modes, commands,
Parent navigation/search, shared text editing/clipboard, draft protection and cursor.
Jan confirmed Windows acceptance of the final controls on 2026-11-20.

The Normal/Insert follow-up passes full tests, both native builds, vet, focused
TUI race tests and a Windows cross-build. A Linux PTY smoke confirms Enter leaves
Insert, :q cancellation retains the draft, :w creates root/child Projects, :q!
discards without creating, and existing empty-child deletion still works.

## Parent-picker navigation follow-up — 2026-11-20

Jan requested the standard list motions in the Parent search results: gg/G and
Ctrl+U/D. Rendering and navigation share the same visible choice count, so resize
changes the half-page step. Focused Update/View regressions cover full/filtered
choices, narrow heights, boundary clamping, interrupted gg and paging during search.

## Final acceptance — 2026-11-20

Jan confirmed that acceptance passed on Windows and authorized committing the
final implementation, including the latest creation-dialog, Parent navigation
and deletion-eligibility feedback changes.
