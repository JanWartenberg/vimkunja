package tui

import (
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"strings"
	"testing"
)

func TestNativeStructureKeysAndChecklistProgress(t *testing.T) {
	m := nativeKeys(nativeEditingModel(`<p>Hello</p>`), "space", "x", "x")
	if m.editor.draft.Description != `<ul data-type="taskList"><li data-type="taskItem" data-checked="true"><p>Hello</p></li></ul>` {
		t.Fatal(m.editor.draft.Description)
	}
	if m.editor.draft.Done != m.editor.original.Done {
		t.Fatal("Description checkbox changed Task Done")
	}
	var rendered strings.Builder
	for _, row := range m.editor.nativeDescription.Layout(80, true).Rows {
		for _, span := range row.Spans {
			rendered.WriteString(span.Text)
		}
	}
	if !strings.Contains(rendered.String(), "1/1") {
		t.Fatal("progress not recomputed")
	}
	m = nativeKeys(m, "u", "u")
	if m.editor.draft.Description != `<p>Hello</p>` {
		t.Fatal("Undo did not restore paragraph")
	}
	m = nativeKeys(nativeEditingModel(`<ul><li>A</li><li>B</li></ul>`), "j", "V", "space", "l")
	if m.editor.draft.Description != `<ul><li>A<ul><li>B</li></ul></li></ul>` {
		t.Fatal(m.editor.draft.Description)
	}
	m = nativeKeys(nativeEditingModel(`<p>A</p><p>B</p>`), "V", "j", "space", "w")
	if m.editor.draft.Description != `<blockquote><p>A</p><p>B</p></blockquote>` {
		t.Fatal(m.editor.draft.Description)
	}
	m = nativeKeys(m, "space", "q")
	if m.editor.draft.Description != `<p>A</p><p>B</p>` {
		t.Fatal(m.editor.draft.Description)
	}
}

func TestNativeCheckboxSplitSaveKeepsParentAndChildIdentity(t *testing.T) {
	m, db := auditEditorModel(t)
	const source = `<ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="A"><p>AlphaBeta</p><ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="child"><p>Child</p></li></ul></li></ul>`
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task',?,0,0,1)`, source); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	m = nativeKeys(m, "5", "l", "i", "enter", "esc")
	if m.editor.draft.Description == source {
		t.Fatal("split failed")
	}
	cmd := m.saveEditor(false)
	if cmd == nil {
		t.Fatal(m.lastErr)
	}
	m.Update(cmd())
	if m.lastErr != "" || m.editor.dirty() {
		t.Fatal(m.lastErr)
	}
	ns := m.editor.nativeDraft.Snapshot()[0].Children
	if len(ns) != 2 {
		t.Fatal("did not save two sibling entries")
	}
	id := func(n description.Node) string {
		for _, a := range n.Attributes {
			if a.Name == "data-task-id" {
				return a.Value
			}
		}
		return ""
	}
	if id(ns[0]) != "A" || id(ns[1]) == "" || id(ns[1]) == "A" || id(ns[1].Children[1].Children[0]) != "child" {
		t.Fatal("saved IDs are incorrect")
	}
	betaID := id(ns[1])
	m = nativeKeys(m, "i", "!", "esc")
	cmd = m.saveEditor(false)
	m.Update(cmd())
	if m.lastErr != "" || id(m.editor.nativeDraft.Snapshot()[0].Children[1]) != betaID {
		t.Fatal("later save regenerated Beta ID", m.lastErr)
	}
}

func TestNativeEmptyListAttemptDoesNotPersistEntry(t *testing.T) {
	for _, source := range []string{`<ul data-note="keep"></ul>`, `<ol></ol>`, `<ul data-type="taskList"></ul>`} {
		m := nativeKeys(nativeEditingModel(source), "i", "esc")
		if m.editor.dirty() {
			t.Fatal("i/Esc materialized list entry")
		}
		m = nativeKeys(m, "i", "enter")
		if m.editor.dirty() || m.editor.mode != editorNormal {
			t.Fatal("empty Enter did not end fill attempt")
		}
		m = nativeKeys(m, "i", "Hello", "esc", "d", "d")
		if m.editor.draft.Description != "" {
			t.Fatal("final real entry container was retained")
		}
	}
}
