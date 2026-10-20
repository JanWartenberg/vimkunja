package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

// A zero-cell layout anchor travels through wrapping, clipping and overlays.
// Render removes every anchor before any content reaches the terminal.
const insertCursorAnchor = "\x1b]777;vimkunja-insert-cursor\a"

// insertCursorText is the common rendering seam for text entry. Field renderers
// provide only source text and its rune offset; cursor shape/layout stay here.
func insertCursorText(value string, cursor int) string {
	runes := []rune(value)
	cursor = clamp(cursor, 0, len(runes))
	return string(runes[:cursor]) + insertCursorAnchor + string(runes[cursor:])
}

func newInputCursor(x, y int) *tea.Cursor {
	cursor := tea.NewCursor(x, y)
	cursor.Shape = tea.CursorBar
	cursor.Blink = true
	return cursor
}

func resolveInputCursor(content string, disabled, visible bool) (string, *tea.Cursor) {
	var cursor *tea.Cursor
	found := false
	lines := strings.Split(content, "\n")
	for y, line := range lines {
		if at := strings.Index(line, insertCursorAnchor); at >= 0 {
			replacement := ""
			if !found {
				found = true
				if disabled {
					if visible {
						replacement = lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render("│")
					}
				} else {
					cursor = newInputCursor(ansi.StringWidth(line[:at]), y)
				}
			}
			line = strings.Replace(line, insertCursorAnchor, replacement, 1)
			lines[y] = strings.ReplaceAll(line, insertCursorAnchor, "")
		}
	}
	return strings.Join(lines, "\n"), cursor
}

func (m *model) editorInputFocused() bool {
	return m.focus == paneDetail && m.editor != nil && !m.commandMode && !m.searchOpen && m.comments == nil && m.labelUI == nil && m.editor.nativeLink == nil
}

// Keep the caret, rather than only the end of the value, in clipped prompts.
func fitInputCursor(value string, width int) string {
	before, after, ok := strings.Cut(value, insertCursorAnchor)
	if !ok || ansi.StringWidth(value) <= width {
		return value
	}
	width = max(1, width)
	afterCells := min(3, min(ansi.StringWidth(after), width-1))
	beforeCells := width - afterCells
	if ansi.StringWidth(before) > beforeCells {
		before = ansi.TruncateLeft(before, ansi.StringWidth(before)-max(0, beforeCells-1), "…")
	}
	after = ansi.Truncate(after, max(0, width-ansi.StringWidth(before)), "…")
	return before + insertCursorAnchor + after
}
