package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/vimkunja/vimkunja/internal/vimkunja"
)

func TestDueDatePickerCanReplaceDateAfterPreviousTextEditing(t *testing.T) {
	for _, previous := range []string{"date", "title"} {
		t.Run(previous, func(t *testing.T) {
			m := testModel()
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", DueDate: "2026-09-14"})
			if previous == "title" {
				m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
				m = pressKey(m, tea.Key{Code: 't', Text: "t"})
				for _, r := range "some longer title" {
					m = pressKey(m, tea.Key{Code: r, Text: string(r)})
				}
				m = pressKey(m, tea.Key{Code: tea.KeyEscape})
			} else {
				m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
				m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
				for _, r := range "20260915" {
					m = pressKey(m, tea.Key{Code: r, Text: string(r)})
				}
				m = pressKey(m, tea.Key{Code: tea.KeyEnter})
			}
			m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
			m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
			for _, r := range "20261020" {
				m = pressKey(m, tea.Key{Code: r, Text: string(r)})
			}
			m = pressKey(m, tea.Key{Code: tea.KeyEnter})
			if got := m.editor.draft.DueDate; got != "2026-10-20" {
				t.Fatalf("date after reopening picker and typing = %q, want 2026-10-20", got)
			}
		})
	}
}
