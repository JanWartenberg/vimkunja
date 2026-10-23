# 01 — Specify independent histories for context-specific command prompts

**Phase:** M3 — command-prompt consistency

**What to decide:** Define session-local, independent command histories for every Vimkunja command-input context without making context-specific commands share a grammar or history.

**Implementation prerequisites:** M2's [top-level Ex-command history](../../../.archive/2026-08-12/command-line-enhancements/issues/01-add-project-goto-command-history-and-completion.md) is implemented and archived. That contract covers non-empty Enter submissions, Up/Down draft restoration, editable recalls, and a 100-entry session-only limit. Context-specific histories still require specification.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

## Context

Vimkunja currently has more than one `:` prompt:

- The top-level Ex command line dispatches through the shared command registry.
- Global Search has a context-specific prompt: today it recognizes `:q` to close Search and restore its prior context.
- The Label picker has context-specific `:q`, `:w`, and `:wq` behavior, then falls back to the shared registry for other commands.

These prompts do not currently share a grammar. Grammar unification would mean giving them common rules for command names, aliases, arguments, completion, dispatch, and context availability. It is not a prerequisite for this work and must not accidentally make their histories shared.

The intended direction is one distinct, session-only history per command-input context. A command entered while handling Labels must not appear when opening the top-level Ex line or Global Search, and vice versa.

## Decisions confirmed

- Follow the completed M2 top-level Ex history; its completion does not specify context-specific histories.
- Maintain distinct, session-only histories per command-input context; do not share entries between top-level Ex, Global Search, or the Label picker.
- Do not require grammar unification to add independent histories. Grammar unification remains a separate future design question.
- Keep future user-configured command/keymap availability and the immutable-command boundary explicit triage questions rather than silently introducing override behavior.

## Questions for triage

- Which present and future command-input contexts receive an independent history, and how are their identities kept stable as UI code evolves?
- Does each history inherit the M2 top-level policy for recording, the 100-entry cap, editable recalls, Up/Down boundaries, and draft restoration, or which context needs an explicit exception?
- How should context-specific history behave if its command grammar or available actions change while the session is running?
- Should context-specific prompts gain command-name completion, and if so, do they use their own available-command set or the shared registry plus explicit contextual commands?
- How do future user-configured commands and keymaps declare availability without bypassing the single registry source of truth or permitting silent name collisions?
- Keep a future decision separate on commands that are immutable and cannot be remapped or overridden.

## Scope boundary

Do not merge or redesign the context-specific command grammars merely to add history. Do not persist any history to disk. Do not change SQLite schema or Vikunja compatibility.
