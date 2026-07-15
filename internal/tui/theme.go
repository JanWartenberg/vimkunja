package tui

import "charm.land/lipgloss/v2"

const selectionAccent = "205"

func focusedField(text string, focused bool) string {
	if !focused {
		return text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(selectionAccent)).Bold(true).Render("▸ " + text)
}
