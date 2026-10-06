package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

func TestCommentThreadAppendAliases(t *testing.T) {
	for _, key := range []rune{'a', 'i', 'o'} {
		t.Run(string(key), func(t *testing.T) {
			m := testModel()
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
			m.editor.commentsLoading = false
			m.editor.comments = []vimkunja.TaskComment{{ID: 1, HTML: "<p>Existing</p>"}}
			m.openComments()
			m = pressKey(m, tea.Key{Code: key, Text: string(key)})
			if m.comments.composer == nil || m.comments.composer.mode != editorInsert || m.comments.composer.text != "" {
				t.Fatalf("%c did not start an empty Insert-mode draft: %#v", key, m.comments.composer)
			}
			m = pressKey(m, tea.Key{Text: "io"})
			if m.comments.composer.text != "io" {
				t.Fatal("composer did not preserve literal input")
			}
			m = pressKey(m, tea.Key{Code: tea.KeyEscape})
			m = pressKey(m, tea.Key{Code: 'o', Text: "o"})
			if m.comments.composer.text != "io\n" || m.comments.composer.mode != editorInsert {
				t.Fatal("composer o no longer opens a line")
			}
			if len(m.comments.items) != 1 || m.comments.items[0].HTML != "<p>Existing</p>" {
				t.Fatal("append alias modified an existing Comment")
			}
		})
	}
}

func TestCommentThreadBlockCursorFollowsSelection(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 22
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.commentsLoading = false
	m.editor.comments = []vimkunja.TaskComment{
		{ID: 1, Author: "Ada", HTML: "<p>First</p>"},
		{ID: 2, Author: "Grace", HTML: "<p>Second</p>"},
	}
	m.openComments()
	block := normalCursorText(">", 0)
	check := func(author string) {
		t.Helper()
		view := m.Render()
		if strings.Count(view, block) != 1 {
			t.Fatalf("want exactly one thread block cursor: %q", view)
		}
		for _, row := range strings.Split(view, "\n") {
			if strings.Contains(row, block) && strings.Contains(row, author) {
				return
			}
		}
		t.Fatalf("cursor not on selected author %s", author)
	}
	check("Grace")
	m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
	check("Ada")
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	check("Grace")
	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	if strings.Contains(m.Render(), block) {
		t.Fatal("thread cursor remains active in command prompt")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	if strings.Contains(m.Render(), block) {
		t.Fatal("thread cursor remains active in composer")
	}
}
