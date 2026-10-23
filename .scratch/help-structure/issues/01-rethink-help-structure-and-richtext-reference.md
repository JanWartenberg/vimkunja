# 01 — Restructure help and separate the rich-text reference

**Phase:** M3 — built-in help usability

**Category:** discovery / UX

**Priority:** Medium

**Status:** needs-triage

**Recorded:** 2026-10-19

## Request and problem

Jan reports from a screenshot that help indentation is inconsistent and hard to read. Restructure help; at least rich-text editing needs a separate help page. Brainstorm and critically challenge the new grouping before implementation. This ticket authorizes design work, not product implementation yet.

The `HELP` screenshot shows dense rich-text lines: text objects, paragraph/Code behavior, Visual modes, Links, local Undo/Redo, recovery commands, lists, quotes, and inline syntax. Descriptions start at inconsistent columns; multiple rules/exceptions share a line. Some entries omit the mode in which a key applies. A cleaner column grid alone will not resolve this information density.

## Findings on main before rich-text integration

Historical starting point: rich text is now integrated. Preserve this inventory for comparison; inventory the merged state before designing.

- `internal/tui/help.go` contains one long list of manually space-aligned strings. Existing `count+motion`, `diw / daw`, and neighboring entries already start descriptions at different columns.
- `helpCommand` in `internal/tui/commands.go` always opens the same list. `infoPaneRows` in `internal/tui/panes.go` passes it to generic wrapping; no semantic key/mode/explanation layout exists there.
- The screenshot's additional rich-text lines are not yet in this historical checkout's `help.go`. Check their implementation source and post-integration state before implementing. Do not infer the screenshot's full cause from this checkout.
- `docs/keybindings.md` separates implemented/agreed bindings; automatic help derivation remains deferred. The [rich-text spec](../../../.archive/2026-10-19/description-rendering/spec.md) also requires durable markup rules and a tutorial section. Help, complete reference, and tutorial serve different purposes.

## Requirements and open decisions

**Fixed:** Aligned, readable layout and at least a separate rich-text help page. Moving the same wall of text is insufficient. Help remains keyboard-operated and documents the behavior of the current build.

**Open:** Additional pages, entry point, context, page navigation, exact grouping, and layout. Variants/page names below are discussion material, not agreed commands/architecture.

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

## Design process and completion

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

- 2026-10-19 — Jan: “help indentation is inconsistent,” “hard to read”; requests a scratch ticket for brainstorming, critical review, and new grouping, with at least separate rich-text help.
