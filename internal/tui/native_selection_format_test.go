package tui

import (
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"testing"
)

func TestNativeBlockSelectionCanFormatText(t *testing.T) {
	for _, tc := range []struct{ key, tag string }{{"b", "strong"}, {"i", "em"}, {"u", "u"}, {"s", "s"}} {
		t.Run(tc.tag, func(t *testing.T) {
			m := nativeKeys(nativeEditingModel(`<p>Alpha</p><p>Beta</p><p>Keep</p>`), "V", "j", "space", tc.key)
			want := `<p><` + tc.tag + `>Alpha</` + tc.tag + `></p><p><` + tc.tag + `>Beta</` + tc.tag + `></p><p>Keep</p>`
			if m.lastErr != "" || m.editor.draft.Description != want {
				t.Fatalf("%s: %s", m.lastErr, m.editor.draft.Description)
			}
			m = nativeKeys(m, "space", tc.key)
			if m.editor.draft.Description != `<p>Alpha</p><p>Beta</p><p>Keep</p>` {
				t.Fatal("selected format did not toggle off", m.editor.draft.Description)
			}
		})
	}
}

func TestNativeCheckboxConversionUsesAllTouchedParagraphs(t *testing.T) {
	for _, selection := range [][]string{{"V", "j"}, {"l", "v", "j"}} {
		m := nativeEditingModel(`<p>Alpha</p><p>Beta</p><p>Keep</p>`)
		m = nativeKeys(m, selection...)
		m = nativeKeys(m, "space", "x")
		want := `<ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>Alpha</p></li><li data-type="taskItem" data-checked="false"><p>Beta</p></li></ul><p>Keep</p>`
		if m.lastErr != "" || m.editor.draft.Description != want {
			t.Fatalf("%v: %s: %s", selection, m.lastErr, m.editor.draft.Description)
		}
		m = nativeKeys(m, "esc", "u")
		if m.editor.draft.Description != `<p>Alpha</p><p>Beta</p><p>Keep</p>` {
			t.Fatal("one Undo did not restore entire conversion")
		}
	}
}

func TestNativeBlockFormattingPreservesOpaque(t *testing.T) {
	const source = `<p>Alpha<mention-user data-id="jan" data-label="Jan"></mention-user></p><p>Beta</p>`
	m := nativeKeys(nativeEditingModel(source), "V", "G", "space", "b", "space", "b")
	if m.lastErr != "" || m.editor.draft.Description != source {
		t.Fatalf("toggle or preservation failed: %s %s", m.lastErr, m.editor.draft.Description)
	}
}

func TestNativeBlockFormattingRejectsCodeAtomically(t *testing.T) {
	const source = `<p>Alpha</p><pre><code>original()</code></pre><p>Beta</p>`
	m := nativeKeys(nativeEditingModel(source), "V", "G", "space", "b")
	if m.lastErr == "" || m.editor.draft.Description != source {
		t.Fatalf("Code formatting was not rejected atomically: %s %s", m.lastErr, m.editor.draft.Description)
	}
}

func TestNativeSelectedCheckboxesSaveWithDistinctIDs(t *testing.T) {
	m, db := auditEditorModel(t)
	const source = `<p>Alpha</p><p>Beta</p>`
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task',?,0,0,1)`, source); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	m = nativeKeys(m, "V", "j", "space", "x", "esc")
	command := m.saveEditor(false)
	if command == nil {
		t.Fatal("save did not start", m.lastErr)
	}
	m.Update(command())
	if m.lastErr != "" || m.editor.dirty() {
		t.Fatal("save failed", m.lastErr)
	}
	var saved string
	var done bool
	if err := db.QueryRow(`SELECT description,done FROM tasks WHERE id=1`).Scan(&saved, &done); err != nil {
		t.Fatal(err)
	}
	if done {
		t.Fatal("Description Checkboxes changed Task Done")
	}
	nodes := description.Parse(saved).Snapshot()
	if len(nodes) != 1 || len(nodes[0].Children) != 2 {
		t.Fatalf("saved wrong structure: %s", saved)
	}
	ids := map[string]bool{}
	for _, item := range nodes[0].Children {
		for _, attribute := range item.Attributes {
			if attribute.Name == "data-task-id" {
				if attribute.Value == "" || ids[attribute.Value] {
					t.Fatal("checkbox identity missing/reused")
				}
				ids[attribute.Value] = true
			}
		}
	}
	if len(ids) != 2 {
		t.Fatalf("missing checkbox IDs: %s", saved)
	}
}
