package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

func TestNativeDescriptionDetailShowsCodeAndNavigatesWithoutEditing(t *testing.T) {
	m := testModel()
	const source = "<p>Before</p><pre><code>print(1)\n\treturn</code></pre><blockquote><p>Quote</p></blockquote><p>After</p>"
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	m.editor.field = fieldDescription
	m.editor.descriptionNavigated = true
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	detail := ansi.Strip(renderTaskDetail(m, 80, 24))
	if !strings.Contains(detail, "```") || !strings.Contains(detail, "│ print(1)") {
		t.Fatalf("native Code missing: %q", detail)
	}
	for _, key := range []string{"i", "o", "O", "x", "d", "d", "a", "v", "u"} {
		m = pressKey(m, tea.Key{Code: []rune(key)[0], Text: key})
	}
	if m.editor.mode != editorNormal || m.editor.draft.Description != source || m.editor.dirty() {
		t.Fatal("native read navigation enabled editing")
	}
}

func TestNativeDescriptionKeepsFieldShortcuts(t *testing.T) {
	for _, key := range []string{"c", "d", "t"} {
		t.Run(key, func(t *testing.T) {
			m := testModel()
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "<p>Content</p>"})
			m.editor.field = fieldDescription
			m.editor.descriptionActive = true
			m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
			m = pressKey(m, tea.Key{Code: []rune(key)[0], Text: key})
			switch key {
			case "c":
				if m.comments == nil {
					t.Fatal("gc blocked by read-only handler")
				}
			case "d":
				if m.editor.field != fieldDueDate {
					t.Fatal("gd blocked by read-only handler")
				}
			case "t":
				if m.editor.field != fieldTitle || m.editor.mode != editorInsert {
					t.Fatal("gt failed")
				}
			}
		})
	}
}

func TestNativeDescriptionProtectsEmptyAndMalformedSources(t *testing.T) {
	for _, source := range []string{"<p></p>", "<!-- secret -->", "<p>Visible</unexpected>", "<unfinished"} {
		t.Run(source, func(t *testing.T) {
			m := testModel()
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
			m.editor.field = fieldDescription
			m = pressKey(m, tea.Key{Code: tea.KeyEnter})
			for _, key := range []string{"~", "d", "d", "x", "i", "o"} {
				m = pressKey(m, tea.Key{Code: []rune(key)[0], Text: key})
			}
			if m.editor.draft.Description != source || m.editor.mode != editorNormal || m.editor.dirty() {
				t.Fatal("protected source became editable")
			}
			rendered := ansi.Strip(renderTaskDetail(m, 40, 20))
			if strings.Contains(rendered, "secret") {
				t.Fatal("comment source leaked")
			}
			if source == "<p></p>" && strings.Contains(rendered, "no description") {
				t.Fatal("empty Native paragraph hidden by old reader")
			}
		})
	}
}

func TestNativeDescriptionResizeMotionsDoNotPersist(t *testing.T) {
	m, db := auditEditorModel(t)
	const source = `<p>one two three</p><pre><code>abcdefghijklmnopqrstuvwxyz</code></pre><p>End</p>`
	if _, err := db.Exec(`INSERT INTO tasks (id,project_id,title,description,done) VALUES (1,1,'Task',?,0)`, source); err != nil {
		t.Fatal(err)
	}
	// This connection observes commits from the Store's separate SQL handle.
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var beforeChanges int
	if err := conn.QueryRowContext(context.Background(), `SELECT total_changes()`).Scan(&beforeChanges); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := conn.QueryRowContext(context.Background(), `PRAGMA data_version`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	m.editor.field = fieldDescription
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	m = pressKey(m, tea.Key{Code: 'w', Text: "w"})
	m = pressKey(m, tea.Key{Code: 'w', Text: "w"})
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'e', Text: "e"})
	if m.editor.descriptionView().Cursor().Anchor.Offset != 6 {
		t.Fatal("ge dispatch")
	}
	anchor := m.editor.descriptionView().Cursor()
	for _, size := range [][2]int{{100, 40}, {54, 12}, {20, 6}, {100, 40}} {
		updated, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = updated.(*model)
		m.Render()
		if m.editor.descriptionView().Cursor() != anchor {
			t.Fatal("terminal resize changed source target")
		}
	}
	for _, key := range []string{"2", "j", "z", "l", "z", "h", "ä", "ö", "g", "j", "g", "k", "_"} {
		m = pressKey(m, tea.Key{Code: []rune(key)[0], Text: key})
	}
	var after int
	if err := conn.QueryRowContext(context.Background(), `PRAGMA data_version`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	var got string
	if err := db.QueryRow(`SELECT description FROM tasks WHERE id=1`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	var afterChanges int
	if err := conn.QueryRowContext(context.Background(), `SELECT total_changes()`).Scan(&afterChanges); err != nil {
		t.Fatal(err)
	}
	if before != after || beforeChanges != afterChanges || got != source || m.editor.dirty() {
		t.Fatal("read/navigation wrote to database or draft")
	}
}

func TestNativeDescriptionCountedGGAndGKeepCodeLine(t *testing.T) {
	for _, keys := range [][]string{{"3", "g", "g"}, {"3", "G"}} {
		m := testModel()
		m.width, m.height = 100, 30
		m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "<pre><code>a\nb\nc</code></pre>"})
		m.editor.field = fieldDescription
		m = pressKey(m, tea.Key{Code: tea.KeyEnter})
		for _, key := range keys {
			m = pressKey(m, tea.Key{Code: []rune(key)[0], Text: key})
		}
		if m.editor.descriptionView().Cursor().Anchor.Offset != 4 {
			t.Fatalf("%v lost Code line", keys)
		}
	}
}
