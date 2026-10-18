package tui

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestChecklistSummaryProjection(t *testing.T) {
	tests := []struct {
		name, source string
		summaries    []string
	}{
		{"plain nested", "[ ] parent\n  [x] child\n\t[ ] deep", []string{"Checklist progress: 1/3"}},
		{"plain boundaries", "[x] one\n\n[ ] two\nparagraph\n[x] three", []string{"Checklist progress: 1/1", "Checklist progress: 0/1", "Checklist progress: 1/1"}},
		{"strict markers", "[X] no\n[  ] no\n[xx] no\ntext [x] no\n- [x] no", nil},
		{"empty marker is open", "[] open\n[x] done", []string{"Checklist progress: 1/2"}},
		{"empty entries", "[ ]\n[x]", []string{"Checklist progress: 1/2"}},
		{"ordinary text", "Checklist progress: 3/5", nil},
		{"html nested", `<ul data-type="taskList"><li data-type="taskItem" data-checked="TRUE"><p>Parent</p><ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>Child</p></li></ul></li></ul>`, []string{"Checklist progress: 1/2"}},
		{"html siblings", `<ul data-type="taskList"><li data-type="taskItem" data-checked="true">One</li></ul><ul data-type="taskList"><li data-type="taskItem">Two</li></ul>`, []string{"Checklist progress: 1/1", "Checklist progress: 0/1"}},
		{"html empty", `<ul data-type="taskList"></ul>`, nil},
		{"html malformed", `<ul data-type="taskList"><li>Not a checkbox</li></ul>`, nil},
		{"html mixed", `<ul data-type="taskList"><li>Not a checkbox</li><li data-type="taskItem" data-checked="invalid"></li></ul>`, []string{"Checklist progress: 0/1"}},
		{"html text not markers", `<p>[x] literal</p>`, nil},
		{"html nested in ordinary list", `<ul><li>Heading<ul data-type="taskList"><li data-type="taskItem">Child</li></ul></li></ul>`, []string{"  Checklist progress: 0/1"}},
		{"html multiline before list", "<pre>first\nsecond</pre><ul data-type=\"taskList\"><li data-type=\"taskItem\">Item</li></ul>", []string{"Checklist progress: 0/1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, line := range wrapTaskDescription(tt.source, 100) {
				if line.start == -1 {
					got = append(got, ansi.Strip(line.text))
					if line.end != -1 {
						t.Fatal("summary has source position")
					}
				}
			}
			if strings.Join(got, "\n") != strings.Join(tt.summaries, "\n") {
				t.Fatalf("summaries = %q, want %q", got, tt.summaries)
			}
		})
	}
}

func TestChecklistSummaryWrappingAndSourceOffsets(t *testing.T) {
	source := "[x] first\n\n[ ] second"
	for _, width := range []int{1, 5, 80} {
		lines := wrapTaskDescription(source, width)
		var summaries strings.Builder
		runes := []rune(source)
		for _, line := range lines {
			if line.start == -1 {
				summaries.WriteString(ansi.Strip(line.text))
				continue
			}
			if got := string(runes[line.start:line.end]); got != line.text {
				t.Fatalf("source offset mismatch: %q != %q", got, line.text)
			}
		}
		if summaries.String() != "Checklist progress: 1/1Checklist progress: 0/1" {
			t.Fatal(summaries.String())
		}
		cursor := len(runes) - 1
		layout := layoutTaskDescription(source, width, 100, 0, cursor)
		line := layout.lines[layout.cursorRow]
		if line.start+layout.cursorColumn != cursor || line.start < 0 {
			t.Fatal("cursor targets synthetic content")
		}
	}
}

func TestPlainChecklistSyntaxRemainsLiteralInUnifiedEditor(t *testing.T) {
	source := "[x] done\n[ ] open"
	for _, mode := range []editorMode{editorNormal, editorInsert} {
		e := &taskEditor{field: fieldDescription, mode: mode, descriptionActive: true}
		e.draft.Description = source
		lines := wrapTaskDescriptionForEditor(e, source, 100)
		if len(lines) != 2 {
			t.Fatalf("unexpected plain rows: %v", lines)
		}
		for _, line := range lines {
			if strings.Contains(line.text, "Checklist progress:") {
				t.Fatal("imported plain syntax became a checklist")
			}
		}
		if e.draft.Description != source {
			t.Fatal("reading changed source")
		}
	}
}

func TestChecklistSummaryUsesDedicatedThemeRole(t *testing.T) {
	old := tuiTheme.checkboxListSummary
	defer func() { tuiTheme.checkboxListSummary = old }()
	tuiTheme.checkboxListSummary = textFormat{bold: true, foreground: "123"}
	lines := wrapTaskDescription("[x] done", 5)
	for _, line := range lines {
		if line.start == -1 {
			if line.text != tuiTheme.checkboxListSummary.render(ansi.Strip(line.text)) {
				t.Fatal("summary style lost during wrapping")
			}
		}
	}
	// The shared reader used by Comments must not gain synthetic content.
	if got := descriptionTerminalText("[x] done"); got != "[x] done" {
		t.Fatal(got)
	}
}

func TestChecklistSummaryScreenAndEditing(t *testing.T) {
	for _, height := range []int{24, 40, 60} {
		t.Run(fmt.Sprintf("height%d", height), func(t *testing.T) {
			m := testModel()
			m.width, m.height = 100, height
			m.currentProjectID = 1
			source := "[x] first\n\nTARGET"
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
			m.editor.field = fieldDescription
			m.editor.descriptionActive = true
			m.editor.mode = editorInsert
			m.editor.cursor = len([]rune("[x] first\n\n"))
			positionDescriptionCursor(t, m, m.editor.cursor)
			m = pressKey(m, tea.Key{Code: 'Z', Text: "Z"})
			rows := strings.Split(ansi.Strip(m.Render()), "\n")
			cursor := m.View().Cursor
			if (cursor == nil && !strings.Contains(strings.Join(rows, "\n"), "Z│TARGET")) || (cursor != nil && (cursor.Position.Y >= len(rows) || !strings.Contains(rows[cursor.Position.Y], "ZTARGET"))) {
				t.Fatalf("cursor=%#v, screen=%s", cursor, strings.Join(rows, "\n"))
			}
			if strings.Contains(strings.Join(rows, "\n"), "Checklist progress:") {
				t.Fatal("summary should be hidden during insert mode")
			}
			if descriptionText(m) != "[x] first\n\nZTARGET" {
				t.Fatal("synthetic text entered draft")
			}
			if m.editor.draft.Done {
				t.Fatal("summary changed Task Done")
			}
		})
	}
}
