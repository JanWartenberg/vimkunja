# 01 — Add Ex-command abbreviations, history, and completion

**What to build:** Vimkunja's Ex command line supports Vim-like unique command abbreviations, command history, and command-name completion.

**Implementation prerequisites:** None — the interaction and registry contract is accepted.

**Category:** enhancement

**Priority:** Medium

**Status:** implemented

- [x] Drop `:goto <project>` and its argument grammar; global search covers the navigation need.
- [x] Define unique-prefix command resolution and Tab completion.
- [x] Define command-history retention scope, recording, and Up/Down navigation.
- [x] Preserve the command registry as the single command dispatch seam.

## Decisions confirmed

- `:goto <project>` is out of scope. It duplicated the planned global-search navigation and would force a multi-word argument grammar without a demonstrated need. The current whitespace-separated parser remains sufficient.
- On `Enter`, an exact command spelling wins. Otherwise, an unambiguous case-insensitive prefix of a registered command spelling executes its command. Aliases that resolve to the same command count as one target. An ambiguous prefix reports the stable, canonical candidate names; an unknown prefix remains an error. Thus `:k`, `:ka`, …, and `:kanban` all open the Kanban View, while `:q` remains `:quit`.
- `Tab` completes a unique prefix to its canonical full command without executing it. For an ambiguous prefix, it shows stable sorted candidates in the command footer and leaves the input unchanged. The first increment has no cycling or `Shift+Tab` behavior.
- History is session-only, retained in memory, and capped at 100 entries. Persisted history is deferred.

## Interaction contract

### Command history

- History belongs only to the top-level Ex prompt. Global Search and Label-picker prompts remain history-free in M2; their distinct session-local histories are deferred to [M3 ticket 01](../../../../.scratch/contextual-command-histories/issues/01-specify-independent-command-prompt-histories.md).
- On every non-empty `Enter`, record the submitted input after trimming surrounding whitespace and omitting the display colon. Preserve its case, alias choice, and arguments verbatim. Record successful, failed, and unknown commands and repeated entries alike; do not record empty submissions or `Esc` cancellations.
- Retain the 100 most recent entries. On the 101st recorded command, discard the oldest.
- The first `Up` saves the current draft and recalls the newest entry. Further `Up` clamps at the oldest. `Down` moves newer; `Down` after the newest restores that original draft and exits history navigation. `Down` before any `Up` does nothing.
- A recalled entry is normal editable input; edits never mutate stored history. A subsequent `Up` or `Down` replaces and discards unsent recalled edits, while Down-past-newest still restores the original pre-history draft.

### Command completion feedback

- Complete only the first command-name token and only when the cursor is at its end. At any other cursor position, `Tab` is a silent no-op. After whitespace or an argument, leave input unchanged and show `Argument completion is not available.`
- `Tab` on an empty prompt leaves input unchanged and shows `Tab completes command names  •  :help lists commands`.
- For an ambiguous non-empty prefix, leave input unchanged and append `matches: <canonical names>` to the command footer. Clear completion feedback on the next input-editing key, `Tab`, `Enter`, or `Esc`; never record it as an error.
- Candidates are case-insensitively sorted by canonical name; aliases never appear as separate candidates. A unique match completes to its canonical spelling without execution or cycling.

### Registry seam

- The registry owns canonical names, ordered aliases, and handlers. Exact resolution, prefix matching, completion, and ambiguity feedback all query it; no parallel command list is permitted.
- Readable long names are canonical completion targets: for example, `quit`, `write`, `help`, `errors`, `writequit`, and `writequitall`. Existing Vim-style short forms remain aliases, including `q`, `w`, `h`, `err`, `wq`, and `wqa`.
- Reject a canonical name or alias that collides with a spelling owned by another command, including future configuration, with a clear startup/configuration error. Never silently override an earlier registration. Normalize duplicate aliases of the same command away.
- The registry is the future extension seam for user-configured commands and keymaps. Whether some commands are immutable is deliberately deferred with the contextual-history work.

## Implementation

- The registry now retains canonical long names and aliases, rejects spelling collisions, resolves exact names before unique prefixes, and supplies completion candidates.
- The top-level Ex prompt retains its 100-entry session history, restores drafts around Up/Down navigation, and renders non-error Tab feedback in its footer.
- Global Search and the Label picker remain unchanged.

## Further context

Archived sources: `SPEC_FUNDAMENT.md` and `TUI_IDEA.md`, command extensibility and autocomplete notes. The current command parser uses `strings.Fields`, which remains sufficient while command arguments are out of scope.
