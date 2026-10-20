package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

func TestNativeDescriptionPageUsesVisibleRowsAcrossDetailLayouts(t *testing.T) {
	var source strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&source, "<p>Row%02d</p>", i)
	}
	rows := regexp.MustCompile(`Row\d\d`)
	for _, width := range []int{55, 100} {
		for height := 18; height <= 46; height++ {
			t.Run(fmt.Sprintf("%dx%d", width, height), func(t *testing.T) {
				m := testModel()
				m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source.String()})
				nativeKeys(m, "g", "n", "enter")
				before := rows.FindAllString(ansi.Strip(m.View().Content), -1)
				if len(before) == 0 || before[0] != "Row00" {
					t.Fatalf("initial Description viewport = %v", before)
				}
				m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
				nativeKeys(m, "i")
				view := m.View()
				if view.Cursor == nil {
					t.Fatal("page target has no insertion cursor")
				}
				line := []rune(strings.Split(ansi.Strip(view.Content), "\n")[view.Cursor.Y])
				want := fmt.Sprintf("Row%02d", len(before))
				if view.Cursor.X >= len(line) || !strings.HasPrefix(string(line[view.Cursor.X:]), want) {
					t.Fatalf("page from %v targets %q at column %d, want %s", before, string(line), view.Cursor.X, want)
				}
				if m.editor.dirty() || m.editor.draft.Description != source.String() {
					t.Fatal("paging changed the Description")
				}
			})
		}
	}
}
