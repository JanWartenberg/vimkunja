# 01 Keyboard shortcuts

**What to build:** More Vim-like shortcuts and paste from clipboard

**Implementation prerequisites:** None

**Category:** enhancement

**Milestone:** M2

**Priority:** High

**Status:** implemented


## What to build

In Insertmode: Ctrl+V -> paste contents of Systemclipboard at cursor position

(Selection for copying is also still missing, but that is a future concern.)

In Descriptionfield:
(and in future for any more-line edit fields)

- o    -> enter new line below cursor and change to insert mode
- Shift+O  -> enter new line above cursor and change to insert mode
- a    -> like "i" change to insert mode, but right of the current character

## Implementation

- `Ctrl+V` reads the system clipboard asynchronously and inserts it at the current Insert Mode cursor. A clipboard error is shown in the session error log without changing the draft.
- Plain-text Description Normal Mode supports `o`, `Shift+O`, and `a`; rich-text block semantics remain deferred to Meilenstein 3.
