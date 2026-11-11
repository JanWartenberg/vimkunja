# 01 — Restructure help and separate the rich-text reference

**Phase:** M3 — built-in help usability

**Category:** enhancement (design accepted)

**Priority:** Medium

**Status:** completed

**Specification:** [Accepted help design and implementation criteria](../spec.md)

**Authorization:** Jan explicitly requested implementation via `implement-and-verify` and authorized the commit on 2026-11-11.

**Recorded:** 2026-10-19

## Request and problem

Jan reports from a screenshot that help indentation is inconsistent and hard to read. Restructure help; at least rich-text editing needs a separate help page. Brainstorm and critically challenge the new grouping before implementation. The original request authorized design work; Jan later explicitly requested implementation through `implement-and-verify`.

The `HELP` screenshot shows dense rich-text lines: text objects, paragraph/Code behavior, Visual modes, Links, local Undo/Redo, recovery commands, lists, quotes, and inline syntax. Descriptions start at inconsistent columns; multiple rules/exceptions share a line. Some entries omit the mode in which a key applies. A cleaner column grid alone will not resolve this information density.

## Findings on main before rich-text integration

Historical starting point: rich text is now integrated. Preserve this inventory for comparison; inventory the merged state before designing.

- `internal/tui/help.go` contains one long list of manually space-aligned strings. Existing `count+motion`, `diw / daw`, and neighboring entries already start descriptions at different columns.
- `helpCommand` in `internal/tui/commands.go` always opens the same list. `infoPaneRows` in `internal/tui/panes.go` passes it to generic wrapping; no semantic key/mode/explanation layout exists there.
- The screenshot's additional rich-text lines are not yet in this historical checkout's `help.go`. Check their implementation source and post-integration state before implementing. Do not infer the screenshot's full cause from this checkout.
- `docs/keybindings.md` separates implemented/agreed bindings; automatic help derivation remains deferred. The [rich-text spec](../../../../.archive/2026-10-19/description-rendering/spec.md) also requires durable markup rules and a tutorial section. Help, complete reference, and tutorial serve different purposes.

## Requirements and open decisions

**Fixed:** Aligned, readable layout and at least a separate rich-text help page. Moving the same wall of text is insufficient. Help remains keyboard-operated and documents the behavior of the current build.

**Accepted:** The revised topic overview and always-visible explanations are accepted. The [specification](../spec.md) supersedes the earlier open questions below. The brainstorm and original checklist remain historical design context, not outstanding demands for more variants.

## Brainstorm: possible groupings

1. **Small introduction plus rich-text page:** General help keeps other topics, adds clear groups and a visible rich-text link. Least navigation; remaining page may still be too long.
2. **Topic index with a few pages:** Start with basics, then navigation/Views, Tasks/Detail/Comments, and rich text. Smaller search spaces, but more switching and difficult placement of shared keys.
3. **Contextual entry with complete index:** Editor opens matching help; index accessible everywhere. Fast for immediate questions, but potentially surprising; must not hide global commands/other topics.

For the separate rich-text page, consider orientation/mode switches, navigation/selection, editing, formatting/Links, lists/quotes/Code, saving, local Undo/Redo, and safe recovery. Prefer short examples to long semicolon-linked rules. Whether these become sections or more pages remains open.

Do not move shared Title/plain-text Description/Comment Composer motions into rich-text-only help without checking. Also distinguish global Task actions from local draft actions.

## Critical review: test decisions with concrete questions

- **What is the user looking for?** Primarily an action (“edit Link”), key (`Space+l`), or mode? Which searches must work without long scrolling? Does V1 need search or is a good index enough?
- **Does separation work?** What disappears from general help? How does a new user discover rich text? Which short hints need to appear in both places without maintaining rules twice?
- **How much navigation is acceptable?** Is variant 1 enough? Justify each extra page with a concrete lookup it simplifies. Do not automatically create a page per pane.
- **How does help start?** Is `:h` stable or focus-dependent? Compare topic arguments such as `:help richtext`, selectable index entries, and contextual links without fixing syntax yet. Include completion, unknown topics, and return paths.
- **How does the user return?** Define index, previous topic, reopening, scroll position, and `:q`. Opening/closing help from Task, rich-text, and Comment drafts must preserve focus and unsaved content; explicitly check existing pane behavior.
- **Which rules need examples?** `dd` in paragraphs versus Code, `Enter` versus `Shift+Enter`, Normal versus Visual/block selection, inline markup versus pasted literal text, local Undo versus persisted Task actions, and protected-HTML recovery. Do not squeeze every exception into one table row.
- **What does aligned mean in limited space?** Account for shared description starts, identifiable continuation lines, long commands, and wide Unicode. Compare two columns when wide with stacked key/explanation when narrow. One long command must not indent the whole page unnecessarily.
- **How does the page stay readable?** Headings, whitespace, and clear mode labels without color; split aliases/actions as needed. Low heights must not hide orientation behind large headers. Keep context/position recognizable after scrolling/resizing.
- **How do we prevent drift?** Assign ownership between `help.go`, `docs/keybindings.md`, README, and tutorial. Weigh shared structured entries against manual maintenance; do not introduce a generator just for this layout defect. Do not present planned bindings as available.

## Original design process (historical)

- [ ] Inventory help after rich-text integration: entries, modes, aliases, duplicates, missing/unimplemented rules. Compare screenshot with actual state.
- [ ] Exercise all three groupings with the same lookups: save Task, move Kanban card, edit Link, delete Code line, handle blocked HTML, and leave help.
- [ ] Compare at least two concrete text/layout drafts, including a separate rich-text page. Check sizes such as 120×40 and 80×24 and a much narrower help pane after splitting; consider content area, not just terminal size.
- [ ] Answer critical-review questions; record objections, rejected alternatives, and uncertainty. Justify the simplest sufficient grouping.
- [ ] Record chosen content ownership, entry, navigation/return, wrapping rules, and maintenance responsibility. Resolve open product decisions with Jan before treating the ticket as implementable.
- [ ] Derive focused implementation tickets with verifiable acceptance criteria afterward; do not prematurely mark `ready-for-agent`.

## Expected acceptance of later implementation

- Rich text has a clearly discoverable separate page; shared editor basics remain discoverable for Title/Comments.
- Explanations align in the chosen layout. Wrapping, long key sequences, and narrow/short panes remain understandable without clipping rules.
- Mode, action, and key limitations are clear; rich text uses recognizable groups rather than another unstructured command list.
- Representative lookups work without scrolling all help; document concrete paths/steps in the draft and evaluate manually.
- Opening, topic changes, scrolling, resizing, and closing work by keyboard; targeted regressions protect drafts, focus return, and existing pane navigation. Layout tests verify visible results rather than just text presence.
- Help agrees with the keybinding contract for implemented features. No Vikunja API, SQLite schema, or editor-semantics changes merely to simplify help text.

## Comments

- 2026-11-10 — Jan: “allgemein ist das verständlich und wirkt gut”, followed by “passt für mich” after the proposed decisions and documentation step. Accepted the revised fixed overview, separate rich-text topics, short always-visible explanations, help-local history and parent navigation, overview entry through `:h`, and preservation of prior focus and unsaved content on close. Recorded the implementation contract in [spec.md](../spec.md). No further variants or production implementation requested. Earlier requests to compare all variants are superseded by this acceptance.

- 2026-10-19 — Jan: “help indentation is inconsistent,” “hard to read”; requests a scratch ticket for brainstorming, critical review, and new grouping, with at least separate rich-text help.

- 2026-10-24 — Design discussion: Jan chose a fixed overview linking to separate topic pages, with a return path to the overview (principle A). Start with explanations always visible beneath actions/keys; do not prototype expandable reference entries yet. Built a standalone, database-free local TUI prototype (`.scratch/help-structure/tui-prototype/main.go`, not distributed). Run from the repository root: `go run ./.scratch/help-structure/tui-prototype`. `j/k` selects overview topics or scrolls a reference; `Enter` opens; `h`/Backspace returns with selection retained; F2 compares a simulated split pane with full width; `:q` exits. Content is representative, not a complete inventory. Build/package check, 80×24 and 120×40 snapshots, and interactive terminal navigation/scroll/return/quit checks passed. The prototype is ready for Jan's review; production help and implementation readiness are unchanged.

- 2026-10-25 — Jan found the simulated second pane confusing and the wording too verbose. Reworked the TUI prototype as one help pane; removed fake Project/Task context and F2 layout switching. Kept the fixed overview and always-visible explanations; split rich text into Editing basics, Formatting, Links, Lists & Code, and Undo & recovery. Shortened descriptions and combined mode/context with explanations. Added help-local Ctrl+O / Ctrl+I history (Tab is the terminal alias for Ctrl+I), restoring selection and reading position; `-` / Backspace opens the structural parent. New navigation after going back drops forward history. Breadcrumbs show hierarchy. These controls are scoped to the prototype, not changes to application navigation. Build/package check, 48 page snapshots at 120×40, 80×24, 44×24, and 60×12, and interactive overview/subtopic/back/forward/parent/quit checks passed. Awaiting Jan's review.

## Implementation and verification — 2026-11-10

- Implemented structured topics, a fixed overview, five rich-text pages, readable wrapped entries, help-local bounded history, and parent navigation. `:q` restores the originating pane and keeps Task/Comment drafts and local undo intact.
- Regression coverage includes navigation/forward-history truncation, Ex completion, resize and scroll feedback, all topic endpoints at 120×40, 80×24, 44×24, 60×12, long keys/wide Unicode, clean and dirty Comments, missing origins, delayed Kanban refresh, and save/quit behavior. Tutorial lesson text/test now opens a topic before scrolling; tutorial runtime is unchanged.
- Final project checks after all fixes: `go test ./...`, `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`, `go vet ./...`, `gofmt` verification, and `git diff --check` passed. The full run includes Store and tutorial tests.
- Interactive check in the real practice tutorial at 80×24: overview → Rich text → Links, scroll to end, history back/forward, parent, `:q`, and clean exit. No new native Windows terminal acceptance is claimed.
- Five independent review rounds, each with separate read-only bugs/risks and consistency/validation reviewers, completed. Confirmed findings covered clean-Comments context leakage (including direct Overview shortcuts), delayed Kanban focus, Global Search return, and unfocused-help `:q` draft protection. All were fixed with regression tests; both final reviewers reported no findings. No remaining confirmed findings.
- Implementation was initially left uncommitted; Jan authorized the commit on 2026-11-11. Existing design prototypes remain local in `.scratch/help-structure/` and are excluded from the commit.

## Review follow-up — 2026-11-11

- Fixed restored overview selection being clamped against the previous Rich text index's smaller entry count. Only movement keys apply the current page's movement bounds; page changes restore their saved location and normalize it against the destination layout.
- Added a behavioral regression covering forward navigation with Tab/Ctrl+I and parent navigation with minus/Backspace, including opening the restored topic. All four cases failed before the fix and pass afterward.
- Fix review found no further findings. Full `go test ./...`, both entrypoint builds, `go vet ./...`, formatting, and `git diff --check` pass. Jan subsequently authorized committing the implementation and fix on 2026-11-11.
