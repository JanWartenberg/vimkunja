package tui

import "charm.land/lipgloss/v2"

const selectionAccent = "205"

func focusedField(text string, focused bool) string {
	if !focused {
		return text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(selectionAccent)).Bold(true).Render("▸ " + text)
}

// pickerField makes the complete active Picker control visible without adding
// a glyph that could change the form's layout.
func pickerField(text string, active bool) string {
	if !active {
		return text
	}
	return lipgloss.NewStyle().Background(lipgloss.Color(selectionAccent)).Render(text)
}
