# Cursor feedback in the Task editor

## Goal

Make Vimkunja's existing editor states recognizable through Vim-inspired cursor feedback without claiming Vim compatibility or changing editing behavior.

## States

| State | Native terminal cursor | Rendered feedback |
| --- | --- | --- |
| Normal | Block | Existing editor feedback remains stable. |
| Insert | Thin vertical bar (about 20% of a character cell) | Existing text rendering remains stable. |
| Picker | Block | The complete active Picker field uses the shared inverted/background focus treatment. |

A Picker is an editor control that selects a value from a finite set. The Project Picker may accept text to filter its choices; this does not make it a separate editor mode. Current and future Pickers use the same complete-field focus treatment.

## Configuration

`tui.Config.DisableCursorShapes` controls native cursor-shape requests. Its zero value leaves native cursor shapes enabled; when set to `true`, Vimkunja does not request native terminal cursor shapes.

Native cursor-shape control is best effort. A terminal may ignore a request; rendered feedback must still make the active Picker and editor state usable. Cursor feedback must not add printable glyphs, shift text, or corrupt editable content.

## Lifecycle

Vimkunja requests the cursor shape for the active editor state while cursor shapes are enabled. On leaving the editor and on application cleanup, it restores the normal block cursor.

## Boundaries

This feature covers only existing Normal, Insert, and Picker states. It does not add Visual, Select, Replace, Operator-pending, command-line, completion, virtual-replace, or configurable Vim `guicursor` syntax.
