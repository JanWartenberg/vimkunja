# Manual Kanban View and Bucket management

Status: completed and accepted by Jan on 2026-11-22 after manual use; closure and commit authorized. Jan authorized implementation through `implement-and-verify`.
The accepted chat decisions in issue 01 are authoritative. Routine details below
resolve those decisions without changing the Vikunja 2.7.0 schema.

## Entry and scope

`:kanban` and `Space v k` open the last used eligible Board for this Project in
the session, or the first in persisted order. `:kanban switch` / `Space v b`
open a Board picker; Esc leaves the current context intact. `:kanban manage` /
`Space v m` open a standalone Board manager. The Kanban footer advertises
`Space v m manage` when space permits, after card navigation and help hints. Its actions are Open, New, Edit,
Delete. Unsupported filtered Views are identified, never silently converted.
With no supported Board, offer name-first creation. Loading and failed writes
must never discard user input or be mistaken for success.

## Store boundary and test seams

Test public Store methods against disposable real SQLite fixtures, and TUI
Update/Render against input sequences and visible behavior. Use the official
2.7.0 executable for sequential API/Store round trips on disposable databases.
Never open local user data during tests.

The Store owns creation, configuration snapshots, atomic saves, ownership,
archived-Project checks, capacity, conflict detection and deletion. Configuration
contains Board title/order, ordered Buckets with titles/limits, and Default/Done
references. New Bucket draft IDs are local negative IDs resolved on save.
Configuration revisions exclude Task counts: ordinary card movement does not
invalidate configuration editing. External configuration changes reject save
until explicit reload, preserving the draft. Existing extra View fields remain
untouched. Scope is manual Views without effective View filters.

## Creation and editing

Creation asks for a 1–250-character trimmed name, creates standard To-Do /
Doing / Done with no limits, and immediately opens Bucket configuration (Jan’s
acceptance correction). :q or :wq then opens the Board. Existing non-trashed Tasks
start in To-Do without changing their Done state. Duplicate Board names are
allowed as in VOA; display IDs in management to disambiguate them.

Management and configuration use the usual rounded Pane frame with one cell
of top/left spacing and a horizontal separator above the bottom helper row.
Ex input stays in that bottom row. Project-creation Ex input is likewise pinned
directly above its modal’s bottom border.
Configuration is a standalone screen labelled BOARD CONFIGURATION with target
name/ID, mode and dirty marker. Task Detail is closed; unsaved Task/Comment or
inline-title drafts block entry. In-flight context loads are invalidated.
Normal/Insert inputs reuse shared cursor, selection and clipboard behavior.
`:h` / `:help` opens a local command reference for the current management page,
with one explanation per binding and wrapping/scrolling for small terminals.
`q`, Esc or `:q` returns to the unchanged draft. Commands and explanations align
in columns; narrow layouts stack the explanation below its command. Errors
occupy separate rows and never replace the command hints. Default/Done hints
spell out both Space d and Space x.
`j/k` select fields, `i` edits text, Enter/Esc leave Insert. Bucket ordering and
role selection use documented local keys; `o` inserts a Bucket after the selected one; `O` inserts before it. On
Board-level fields, o appends and O prepends. `:w` saves and
stays; `:wq` saves and closes; `:q` asks before discarding a changed draft;
`:q!` discards. Save errors retain the draft. Commands never write Task fields.
No management Undo is promised; incompatible Task undo history is cleared
after structural management writes.

Board order is a nonnegative persisted position; adjust it in configuration.
Bucket slice order is persisted as increasing positions. Zero limit means
unlimited; negative limits fail. Lowering a limit below occupancy is allowed
like VOA, but subsequent moves in remain blocked. Role changes never bulk-edit
Task Done flags or redistribute existing cards. Default and Done must refer to
surviving Buckets of this Board (Default may use VOA's first-Bucket fallback;
Done may be unset).

## Deletion

Deletion always requires a visible consequence preview and explicit `y`.
`n`/Esc cancels. Delete-Board is available only outside a dirty configuration
draft. Preview includes Board identity and affected assignment count, stating
Tasks are retained. Transactionally revalidate configuration and writability,
remove this View's assignments, positions and owned Buckets, then the View.
Return to another supported Board, otherwise List, otherwise an explicit
empty-Board state with creation available. The last Kanban Board is deletable.

Bucket deletion via dd, Space D or :bucketdelete is staged in the configuration
draft after confirmation. dd on a Bucket row preserves the title; other text
operator motions still use the shared editor. Deletion/discard previews use the
shared compact centered modal frame with adjacent question/action rows. Show
affected Task count and the surviving Default destination. Protect the last
Bucket. Require reassignment before deleting an explicit Default/Done Bucket;
the first-Bucket fallback must resolve to a surviving Bucket. At save, move all
removed-Bucket associations, including trash, to that destination. Preserve
Task Done flags and other Views. Recheck active capacity and fail atomically
if the relocation would exceed the destination limit. Roll back on any error.

## Compatibility and layout

Done is global. Verify source-observed cross-Board transitions against VOA;
ordinary non-Done moves remain View-local. Reopening by leaving a Board’s Done
Bucket changes the global Done flag but leaves other Boards’ column placements
untouched; a column named Done does not alone establish completion (✓ does). Preserve existing safeguards for
repeating Tasks, archived Projects and trash. Reject unsupported filtered
Boards rather than displaying or editing an incomplete Task set.

Show the active Board title when the Project has multiple Kanban Boards. Omit
the redundant title line for a single Board (Jan’s usability correction). Switching and management remain accessible
through Ex commands even when hints are clipped. Configuration and picker use
the whole terminal, scroll the selected field into view, and retain title and
action guidance where space permits. At extreme sizes show a concise resize
message while retaining cancel/quit and all data. No fixed extra chrome line is
assumed to fit. Test narrow/short windows and resizing during interaction.

## Verification

Focused tests cover additional-Board creation, configuration round trip,
invalid roles/IDs, stale saves, capacity rollback, Bucket and View deletion,
Task preservation, archived/filtered rejection, session memory and picker
cancellation, dirty-draft protection, configuration-only saves, clipboard and
cursor integration, stale asynchronous results and tiny terminal bounds.
Run full Go tests, both builds, vet, focused race tests and official VOA round
trips. Use two fresh independent read-only reviewers per round, bounded to five
rounds. No commit unless Jan requests it.

## Menu consistency acceptance corrections — 2026-11-22

The new Project-creation form/Parent picker/help and Board manager/switcher/configuration/help use the established Vim conventions. Lists and references support j/k, gg/G, Home/End, Ctrl+D/U and PageUp/PageDown; j/k clamp at field boundaries while Tab cycles fields. Confirmations share paging and always reserve an answer row; Project discard refuses confirmation when that row cannot be displayed. Tiny terminals retain cancellation and the draft. Discard dialogs are compact, and delete actions identify deletion explicitly even in compact action labels.

Local Ex commands resolve names, aliases and unique prefixes through the main registry, then apply only context-supported actions. :h/:help opens Project help too, preserving the draft and Parent selection. Unsupported commands give feedback instead of silently doing nothing. Insert hints explain leaving Insert and pasting; Command input is labelled COMMAND. Errors remain separate from helpers and clear on the next key, like the main UI. Helpers use whole hint segments. Project forms/pickers use a fixed, separated bottom footer just like Board management; Ex input stays above the bottom border.

Board manager/configuration titles identify the target Project. Configuration reserves separate name/ID budgets for the Board and Project, so long Project titles cannot hide the Board at ordinary terminal widths. After creation, a persistent message distinguishes the already saved Board from unsaved configuration: :q! does not delete the Board. The Board switcher already scrolled through the public Render path; the audit's initial no-scroll finding came from an older alternate renderer and was corrected. Public Update/Render regressions retain the selection/footer guarantee and add consistent frame/navigation.
