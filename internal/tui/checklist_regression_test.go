package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

func TestChecklistTrailingSummaryReachable(t *testing.T) {
	for _, height := range []int{24, 28, 32, 36, 40, 60} {
		t.Run(fmt.Sprint(height), func(t *testing.T) {
			m := testModel()
			m.width, m.height = 100, height
			m.currentProjectID = 1
			source := strings.Repeat("[x] entry\n\n", 10) + "[ ] final"
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
			m.editor.field = fieldDescription
			m.editor.descriptionActive = true
			m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
			screen := ansi.Strip(m.Render())
			if !strings.Contains(screen, "Checklist progress: 0/1") {
				t.Fatalf("final summary inaccessible after G:\n%s", screen)
			}
			if m.editor.cursor != len([]rune(source)) {
				t.Fatal("cursor moved outside source")
			}
			if cursor := m.View().Cursor; cursor != nil {
				row := strings.Split(screen, "\n")[cursor.Position.Y]
				if strings.Contains(row, "Checklist progress:") {
					t.Fatal("native cursor targets summary")
				}
			}
			m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
			m = pressKey(m, tea.Key{Code: 'Z', Text: "Z"})
			if got := m.editor.draft.Description; got != source+"Z" {
				t.Fatalf("edit changed source: %q", got)
			}
			if screen := ansi.Strip(m.Render()); !strings.Contains(screen, "finalZ") {
				t.Fatal("source cursor not restored for insertion")
			}
		})
	}
}

func TestChecklistIntermediateSummaryScroll(t *testing.T) {
	for _, height := range []int{24, 32, 40} {
		t.Run(fmt.Sprint(height), func(t *testing.T) {
			m := testModel()
			m.width, m.height = 100, height
			m.currentProjectID = 1
			source := "[x] first\n\n[ ] last\n" + strings.Repeat("body\n", 20)
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
			m.editor.field = fieldDescription
			m.editor.descriptionActive = true
			m.editor.cursor = 0
			m.Render()
			m = pressKey(m, tea.Key{Code: 'e', Mod: tea.ModCtrl})
			screen := ansi.Strip(m.Render())
			if !strings.Contains(screen, "Checklist progress: 1/1") {
				t.Fatalf("intermediate summary inaccessible:\n%s", screen)
			}
			if m.editor.descScroll == 0 {
				t.Fatal("viewport did not scroll")
			}
			if m.editor.cursor != 0 || m.editor.draft.Description != source {
				t.Fatal("viewport scrolling changed source or cursor")
			}
			if cursor := m.View().Cursor; cursor != nil {
				t.Fatal("offscreen source cursor should be hidden")
			}
			m = pressKey(m, tea.Key{Code: 'y', Mod: tea.ModCtrl})
			if !strings.Contains(ansi.Strip(m.Render()), "[x] first") {
				t.Fatal("reverse scrolling did not restore first entry")
			}
			m = pressKey(m, tea.Key{Code: 'e', Mod: tea.ModCtrl})
			m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
			m = pressKey(m, tea.Key{Code: 'Z', Text: "Z"})
			if m.editor.draft.Description != "Z"+source {
				t.Fatal("editing did not resume at original source cursor")
			}
			if !strings.Contains(strings.ReplaceAll(ansi.Strip(m.Render()), "│", ""), "Z[x] first") {
				t.Fatal("editing did not restore cursor visibility")
			}
		})
	}
}

func TestChecklistRenderingMatchesCount(t *testing.T) {
	source := `<ul data-type="taskList"><li data-checked="true">Not a checkbox</li><li data-type="taskItem" data-checked="true">Done</li><li data-type="taskItem"></li></ul>`
	var lines []string
	for _, line := range wrapTaskDescription(source, 100) {
		lines = append(lines, ansi.Strip(line.text))
	}
	want := "- Not a checkbox\n- [x] Done\n- [ ] \nChecklist progress: 1/2"
	if got := strings.Join(lines, "\n"); got != want {
		t.Fatalf("projection=%q, want %q", got, want)
	}
}

func TestChecklistTableCells(t *testing.T) {
	source := `<p>Before</p><table><tr><th>Checklist</th><th>Notes</th></tr><tr><td><ul data-type="taskList"><li data-type="taskItem" data-checked="true"><p>Done</p><ul data-type="taskList"><li data-type="taskItem">Open</li></ul></li></ul><p>After</p><ul data-type="taskList"><li data-type="taskItem">Next</li></ul></td><td>Keep me</td></tr></table><p>End</p>`
	var text, summaries []string
	for _, line := range wrapTaskDescription(source, 100) {
		text = append(text, ansi.Strip(line.text))
		if line.start == -1 {
			summaries = append(summaries, ansi.Strip(line.text))
			if line.end != -1 {
				t.Fatal("table summary has a source position")
			}
		}
	}
	want := "Before\nChecklist | Notes\n- [x] Done | Keep me\n  - [ ] Open | \nChecklist progress: 1/2 | \nAfter | \n- [ ] Next | \nChecklist progress: 0/1 | \nEnd"
	if got := strings.Join(text, "\n"); got != want {
		t.Fatalf("projection=%q, want %q", got, want)
	}
	if got := strings.Join(summaries, "\n"); got != "Checklist progress: 1/2 | \nChecklist progress: 0/1 | " {
		t.Fatalf("summaries=%q", got)
	}
	if got := descriptionTerminalText(source); strings.Contains(got, "Checklist progress:") {
		t.Fatalf("shared Comment reader gained summaries: %q", got)
	}
}

func TestChecklistTableSummaryDoesNotStyleOtherCellText(t *testing.T) {
	source := `<table><tr><td><ul data-type="taskList"><li data-type="taskItem">One</li></ul></td><td><ul data-type="taskList"><li data-type="taskItem">Two</li><li data-type="taskItem">Three</li></ul></td></tr></table>`
	var text []string
	for _, line := range wrapTaskDescription(source, 100) {
		text = append(text, ansi.Strip(line.text))
		if line.start == -1 && strings.Contains(ansi.Strip(line.text), "Three") {
			t.Fatal("ordinary cell text styled as a summary")
		}
	}
	want := "- [ ] One | - [ ] Two\nChecklist progress: 0/1 | \n | - [ ] Three\n | Checklist progress: 0/2"
	if got := strings.Join(text, "\n"); got != want {
		t.Fatalf("projection=%q, want %q", got, want)
	}
}

func TestChecklistMalformedMarkupFallback(t *testing.T) {
	for _, source := range []string{"<broken", "[x] item\n<broken"} {
		lines := wrapTaskDescription(source, 100)
		want := descriptionTerminalText(source)
		if len(lines) != 1 || lines[0].text != want {
			t.Fatalf("projection=%v, want %q", lines, want)
		}
	}
}
