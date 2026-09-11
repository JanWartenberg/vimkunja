package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/vimkunja/vimkunja/internal/vimkunja"
)

func TestDueDatePickerHasBlinkingOverwriteCursor(t *testing.T) {
	for _, size := range [][2]int{{100, 40}, {80, 22}, {40, 18}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			m := testModel()
			m.width, m.height = size[0], size[1]
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", DueDate: "2026-09-14"})
			m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
			m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
			check := func(offset int) {
				t.Helper()
				view := m.View()
				cursor := view.Cursor
				if cursor == nil {
					t.Fatal("date picker has no native cursor")
				}
				if !cursor.Blink || cursor.Shape != tea.CursorBlock {
					t.Fatalf("date cursor = %#v, want blinking overwrite block", cursor)
				}
				rows := strings.Split(ansi.Strip(view.Content), "\n")
				if cursor.Position.Y < 0 || cursor.Position.Y >= len(rows) {
					t.Fatalf("cursor outside view: %#v", cursor)
				}
				row := rows[cursor.Position.Y]
				start := strings.Index(row, m.editor.draft.DueDate)
				if start < 0 {
					t.Fatalf("cursor not on date row: %q", row)
				}
				wantX := ansi.StringWidth(row[:start]) + offset
				if cursor.Position.X != wantX {
					t.Fatalf("cursor x=%d, want %d (digit offset %d)", cursor.Position.X, wantX, offset)
				}
			}
			check(0)
			for i, r := range "2030" {
				m = pressKey(m, tea.Key{Code: r, Text: string(r)})
				offset := i + 1
				if offset == 4 {
					offset = 5
				}
				check(offset)
			}
			m = pressKey(m, tea.Key{Code: tea.KeyLeft}) // separator; next overwrite is still month
			check(5)
			m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
			if m.View().Cursor != nil {
				t.Fatal("relative helper retains date-row cursor")
			}
			m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
			check(5)
			m = pressKey(m, tea.Key{Code: tea.KeyEscape})
			if m.View().Cursor != nil {
				t.Fatal("closed picker retains cursor")
			}
		})
	}
}
