# Vim Visual-Mode binding review

**Purpose:** Check the initially proposed direct Visual-Mode bindings before committing Vimkunja's rich-text command vocabulary.

## Findings

Vim's Visual Mode starts with `v`, then extends the selection with motions, before an operator acts on it. Vim documents this interaction explicitly.[^visual-use]

The proposed direct bindings conflict with established Visual-Mode behavior:

| Proposed Vimkunja key | Vim Visual-Mode meaning | Conflict |
| --- | --- | --- |
| `b` | word-backward motion | Yes |
| `i` | prefix for an inner text object, such as `iw` or `ip` | Yes |
| `u` | lowercase selected text | Yes |
| `s` | substitute selected text | Yes |
| `l` | right-character motion | Yes |

`b` and `l` are documented Vim motions; Visual Mode uses motions to extend the active selection.[^motion-b][^motion-l][^visual-use] The Visual-Mode reference lists the `i` text-object forms and the `s`/`u` operations.[^visual-objects][^visual-operations]

## Recommendation

Do **not** bind formatting directly to `b`, `i`, `u`, `s`, or `l` in Visual Mode. Preserve them for native Vim selection/motion behavior. Use Vimkunja's existing Space leader after a selection instead—for example `Space+b`, `Space+i`, `Space+u`, `Space+s`, and `Space+l` for link editing. This is not a markup syntax and does not conflict with the documented single-key Visual-Mode commands.

The current implementation already owns Space as its leader in editor Normal Mode (`leaderEligible` in `internal/tui/model.go`); a future Description Visual Mode can scope its leader actions separately.

[^visual-use]: Vim, [`runtime/doc/visual.txt`, “Using Visual mode”](https://github.com/vim/vim/blob/master/runtime/doc/visual.txt#L26-L34).
[^motion-b]: Vim, [`runtime/doc/motion.txt`, `b`](https://github.com/vim/vim/blob/master/runtime/doc/motion.txt#L414-L416).
[^motion-l]: Vim, [`runtime/doc/motion.txt`, `l`](https://github.com/vim/vim/blob/master/runtime/doc/motion.txt#L187-L190).
[^visual-objects]: Vim, [`runtime/doc/visual.txt`, Visual text objects](https://github.com/vim/vim/blob/master/runtime/doc/visual.txt#L248-L266).
[^visual-operations]: Vim, [`runtime/doc/visual.txt`, Visual operations](https://github.com/vim/vim/blob/master/runtime/doc/visual.txt#L267-L281).
