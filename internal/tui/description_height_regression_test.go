package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

func TestActivatedDescriptionIsVisibleAcrossTerminalHeights(t *testing.T) {
	for height := 18; height <= 46; height++ {
		for _, source := range []string{"VisibleDescription", "<p>VisibleDescription</p>"} {
			t.Run(fmt.Sprintf("height_%d_html_%t", height, strings.HasPrefix(source, "<")), func(t *testing.T) {
				m := testModel()
				m.Update(tea.WindowSizeMsg{Width: 100, Height: height})
				m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
				nativeKeys(m, "g", "n", "enter")
				view := ansi.Strip(m.View().Content)
				if !strings.Contains(view, "VisibleDescription") {
					t.Fatalf("activated Description invisible at terminal height %d (detail=%d):\n%s", height, m.detailTextHeight(), view)
				}
			})
		}
	}
}

func TestActiveDescriptionSurvivesResizeAndKeepsInsertionVisible(t *testing.T) {
	for _, width := range []int{55, 100} {
		for _, source := range []string{"VisibleDescription", "<p>VisibleDescription</p>"} {
			t.Run(fmt.Sprintf("width_%d_html_%t", width, strings.HasPrefix(source, "<")), func(t *testing.T) {
				m := testModel()
				m.Update(tea.WindowSizeMsg{Width: width, Height: 46})
				m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
				nativeKeys(m, "g", "n", "enter", "A", "!", "esc")
				for _, height := range []int{46, 35, 34, 33, 31, 30, 29, 22, 18, 30, 34, 46} {
					m.Update(tea.WindowSizeMsg{Width: width, Height: height})
					nativeKeys(m, "A")
					view := m.View()
					if !strings.Contains(ansi.Strip(view.Content), "VisibleDescription!") {
						t.Fatalf("resize to %dx%d hid the draft:\n%s", width, height, ansi.Strip(view.Content))
					}
					if view.Cursor == nil || view.Cursor.Shape != tea.CursorBar || view.Cursor.X < 0 || view.Cursor.X >= width || view.Cursor.Y < 0 || view.Cursor.Y >= height {
						t.Fatalf("resize to %dx%d lost insertion cursor: %#v", width, height, view.Cursor)
					}
					nativeKeys(m, "esc")
				}
				nativeKeys(m, "u")
				if m.editor.draft.Description != source || m.editor.dirty() {
					t.Fatal("resize changed source or lost Undo")
				}
				m.Update(tea.WindowSizeMsg{Width: width, Height: 30})
				nativeKeys(m, "esc", "g", "t")
				if m.editor == nil || m.editor.field != fieldTitle || !strings.Contains(ansi.Strip(m.View().Content), "Task") {
					t.Fatal("could not return to metadata after focused Description")
				}
			})
		}
	}
}
