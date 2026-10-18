package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"strings"
	"testing"
)

func TestDescriptionsUseOneEditorWithoutConversion(t *testing.T) {
	for _, source := range []string{"", "Alpha beta\n\nlast", "<p>Alpha beta</p>"} {
		t.Run(source, func(t *testing.T) {
			m := nativeEditingModel(source)
			m.Render()
			if m.editor.nativeDraft == nil || m.editor.dirty() {
				t.Fatal("opening does not provide a clean unified draft")
			}
			if source == "" {
				nativeKeys(m, "i")
				m.Update(tea.PasteMsg{Content: "Alpha beta"})
				nativeKeys(m, "esc")
			}
			nativeKeys(m, "v", "l", "space", "b")
			if m.lastErr != "" || !strings.Contains(m.editor.draft.Description, "<strong>") {
				t.Fatalf("formatting requires a mode switch: %q (%s)", m.editor.draft.Description, m.lastErr)
			}
			nativeKeys(m, "esc", "u")
			if source == "" {
				nativeKeys(m, "u")
			}
			if m.editor.draft.Description != source || m.editor.dirty() {
				t.Fatalf("Undo did not restore baseline: %q", m.editor.draft.Description)
			}
		})
	}
}
func TestPlainDescriptionOpeningAndTitleSaveDoNotConvert(t *testing.T) {
	m, db := auditEditorModel(t)
	source := "literal &amp; text\n\nlast"
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task',?,0,0,1)`, source); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	nativeKeys(m, "g", "n", "enter")
	m.Update(tea.WindowSizeMsg{Width: 54, Height: 24})
	m.Render()
	nativeKeys(m, "v", "l", "esc", "esc", "g", "t", "X", "esc")
	// Only Title changes; merely reading/selecting the Description must not rewrite it.
	if !m.prepareNativeSave() {
		t.Fatal(m.lastErr)
	}
	result := m.saveEditor(false)().(taskSavedMsg)
	if result.err != nil {
		t.Fatal(result.err)
	}
	m.Update(result)
	var saved string
	if err := db.QueryRow("SELECT description FROM tasks WHERE id=1").Scan(&saved); err != nil {
		t.Fatal(err)
	}
	if saved != source {
		t.Fatalf("metadata save converted plain source: %q", saved)
	}
}

// These helpers address document anchors rather than the retired raw-text cursor.
func positionDescriptionCursor(t *testing.T, m *model, offset int) {
	t.Helper()
	e := m.editor
	e.descriptionActive = true
	e.descriptionView()
	nodes := e.nativeDraft.Snapshot()
	for i, n := range nodes {
		text := ""
		if len(n.Children) > 0 && n.Children[0].Kind == description.Text {
			text = n.Children[0].Text
		}
		length := len([]rune(text))
		if offset <= length || i == len(nodes)-1 {
			a := description.Anchor{Node: n.ID}
			if len(n.Children) > 0 {
				a.Node = n.Children[0].ID
				a.Offset = len(string([]rune(text)[:min(offset, length)]))
			}
			if err := e.nativeDraft.SetCursor(a); err != nil {
				t.Fatal(err)
			}
			e.nativeDraft.InsertionView(e.nativeDescription, e.mode == editorInsert)
			return
		}
		offset -= length + 1
	}
}
func descriptionCursorOffset(m *model) int {
	e := m.editor
	cursor := e.nativeDraft.Cursor()
	offset := 0
	var visit func(description.Node) (int, bool)
	visit = func(n description.Node) (int, bool) {
		if n.ID == cursor.Node {
			if n.Kind == description.Text {
				return len([]rune(n.Text[:cursor.Offset])), true
			}
			return 0, true
		}
		total := 0
		for _, c := range n.Children {
			if pos, found := visit(c); found {
				return total + pos, true
			}
			if c.Kind == description.Text {
				total += len([]rune(c.Text))
			}
		}
		return total, false
	}
	for _, n := range e.nativeDraft.Snapshot() {
		if pos, found := visit(n); found {
			return offset + pos
		}
		for _, c := range n.Children {
			if c.Kind == description.Text {
				offset += len([]rune(c.Text))
			}
		}
		offset++
	}
	return offset
}
func descriptionText(m *model) string { return m.editor.descriptionView().ContentText() }

func TestUnifiedDescriptionVimOperators(t *testing.T) {
	for _, tc := range []struct{ name, source, keys, want string }{
		{"word", "one two three", "dw", "two three"},
		{"last word", "one", "dw", ""},
		{"counted word", "one two three four", "2dw", "three four"},
		{"counted lines", "one\ntwo\nthree", "2dd", "three"},
		{"change word", "one two", "cwX", "X two"},
		{"line end", "one two", "C", ""},
		{"styled word", "<p>o<strong>ne</strong> two</p>", "dw", "two"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := nativeEditingModel(tc.source)
			pressMotionKeys(m, tc.keys)
			if got := descriptionText(m); got != tc.want {
				t.Fatalf("%s => %q, want %q (%s)", tc.keys, got, tc.want, m.lastErr)
			}
			if m.editor.mode == editorInsert {
				nativeKeys(m, "esc")
			}
			nativeKeys(m, "u")
			if m.editor.draft.Description != tc.source || m.editor.nativeDraft.Selection() != nil {
				t.Fatalf("one Undo did not restore the pre-operator source/selection: %q", m.editor.draft.Description)
			}
		})
	}
}

func TestUnifiedDescriptionCharacterCommands(t *testing.T) {
	for _, tc := range []struct{ name, source, keys, want string }{
		{"replace", "abcde", "l3rX", "aXXXe"},
		{"delete stops at line", "ab\ncd", "l3x", "a\ncd"},
		{"case stops at line", "aBδ\nNext", "3~", "AbΔ\nNext"},
		{"case keeps marks", "<p><strong>ab</strong>cd</p>", "3~", "ABCd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := nativeEditingModel(tc.source)
			pressMotionKeys(m, tc.keys)
			if got := descriptionText(m); got != tc.want {
				t.Fatalf("%s => %q, want %q (%s)", tc.keys, got, tc.want, m.lastErr)
			}
			if tc.name == "case keeps marks" && !strings.Contains(m.editor.draft.Description, "<strong>AB</strong>") {
				t.Fatal("case edit removed formatting")
			}
			nativeKeys(m, "u")
			if m.editor.draft.Description != tc.source {
				t.Fatal("character edit was not one Undo step")
			}
		})
	}
}

func TestUnifiedDescriptionInvalidOperatorCountDoesNotLeak(t *testing.T) {
	m := nativeEditingModel("one two three")
	pressMotionKeys(m, "d2Qw")
	if m.editor.nativeDraft.Cursor().Offset != 4 || m.editor.dirty() {
		t.Fatalf("invalid prefix leaked into w: %+v", m.editor.nativeDraft.Cursor())
	}
}

func TestImportedEmptyCodeAcceptsInsertAndAppend(t *testing.T) {
	for _, key := range []string{"i", "A"} {
		for _, source := range []string{`<pre><code></code></pre>`, `<blockquote data-comment-id="42"><pre><code></code></pre></blockquote>`} {
			m := nativeEditingModel(source)
			nativeKeys(m, key, "z", "esc")
			want := strings.Replace(source, "<code></code>", "<code>z</code>", 1)
			if m.editor.draft.Description != want || m.lastErr != "" {
				t.Fatalf("%s on %s => %q (%s)", key, source, m.editor.draft.Description, m.lastErr)
			}
			if !m.prepareNativeSave() {
				t.Fatal(m.lastErr)
			}
			nativeKeys(m, "u")
			if m.editor.draft.Description != source {
				t.Fatal("Undo did not restore imported empty Code")
			}
		}
	}
}
