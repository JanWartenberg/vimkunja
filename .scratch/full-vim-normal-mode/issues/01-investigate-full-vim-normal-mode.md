> *This was generated during motion-scope planning.*

# 01 — Investigate Full Vim Normal mode

**What to investigate:** Determine whether Vimkunja should eventually support a substantially complete Vim/Neovim Normal mode beyond the Tier 1 editor motions.

**Implementation prerequisites:** Tier 1 editor motions should be specified and implemented first so the missing behavior and architectural seams are known.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Inventory the canonical Vim/Neovim Normal-mode surface using Vim help (`motion.txt`, `operator`, `object`, and `normal-index`), distinguishing motions, operators, text objects, and commands.
- [ ] Assess the value and feasibility of search motions, character-find motions, marks and jumps, paragraph/sentence motions, Visual mode, registers, macros, replace mode, and other advanced behavior.
- [ ] Identify conflicts with Vimkunja commands and modes, including application-level undo, `g` prefixes, `d`/`c` operator-pending behavior, and pane navigation.
- [ ] Decide whether full compatibility is desirable, define a bounded scope if not, and document the recommended architecture and migration path.
- [ ] Do not implement this ticket; split approved work into focused follow-up tickets.

## Context

A complete canonical list is not one flat list: Vim documents motions, operators, text objects, counts, and Normal-mode commands separately. Full emulation may be excessive for task-title and Description editing; Tier 1 plus the existing editor behavior is the default practical scope.
