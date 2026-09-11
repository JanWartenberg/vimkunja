package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestHomeEndEditAllSingleLineInputs(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*model)
		value func(*model) string
	}{
		{
			name: "command prompt",
			setup: func(m *model) {
				m.openCommandPrompt()
				m.commandInput, m.commandCursor = "middle", len([]rune("middle"))
			},
			value: func(m *model) string { return m.commandInput },
		},
		{
			name: "search query",
			setup: func(m *model) {
				m.searchOpen, m.searchMode = true, searchInsertMode
				m.searchInput, m.searchInputCursor = "middle", len([]rune("middle"))
			},
			value: func(m *model) string { return m.searchInput },
		},
		{
			name: "search command",
			setup: func(m *model) {
				m.searchOpen, m.searchCommandMode = true, true
				m.searchCommandInput, m.searchCommandCursor = "middle", len([]rune("middle"))
			},
			value: func(m *model) string { return m.searchCommandInput },
		},
		{
			name: "label filter",
			setup: func(m *model) {
				m.labelUI = &labelUI{inputMode: true, inputAction: labelInputTaskPicker, input: "middle", inputCursor: len([]rune("middle"))}
			},
			value: func(m *model) string { return m.labelUI.input },
		},
		{
			name: "label command",
			setup: func(m *model) {
				m.labelUI = &labelUI{commandMode: true, commandInput: "middle", commandCursor: len([]rune("middle"))}
			},
			value: func(m *model) string { return m.labelUI.commandInput },
		},
		{
			name: "comment command",
			setup: func(m *model) {
				m.comments = &commentsUI{commandMode: true, commandInput: "middle", commandCursor: len([]rune("middle"))}
			},
			value: func(m *model) string { return m.comments.commandInput },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			tc.setup(m)
			m = pressKey(m, tea.Key{Code: tea.KeyHome})
			m = pressKey(m, tea.Key{Text: "X"})
			m = pressKey(m, tea.Key{Code: tea.KeyEnd})
			m = pressKey(m, tea.Key{Text: "Z"})
			if got := tc.value(m); got != "XmiddleZ" {
				t.Fatalf("Home/End edit = %q, want XmiddleZ", got)
			}
		})
	}
}
