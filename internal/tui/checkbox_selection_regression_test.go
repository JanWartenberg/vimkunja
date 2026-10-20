package tui

import (
	"strings"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

func TestNativeCheckboxSelectionDeleteSaveUndo(t *testing.T) {
	const source = `<ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="entry"><label><input type="checkbox"><span></span></label><div><blockquote><p>one</p><p>two</p></blockquote></div></li></ul>`
	for _, tc := range []struct {
		name, key, want string
	}{
		{"quoted paragraph", "V", strings.Replace(source, `<p>one</p>`, "", 1)},
		{"first character", "v", strings.Replace(source, `<p>one</p>`, `<p>ne</p>`, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, db := auditEditorModel(t)
			if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task',?,0,0,1)`, source); err != nil {
				t.Fatal(err)
			}
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
			m.editor.field = fieldDescription
			m.editor.descriptionActive = true
			m = nativeKeys(m, tc.key, "d")
			if m.lastErr != "" || m.editor.draft.Description != tc.want {
				t.Fatalf("selection deletion changed checkbox control: %s (%s)", m.editor.draft.Description, m.lastErr)
			}
			save := func(want string) {
				t.Helper()
				cmd := m.saveEditor(false)
				if cmd == nil {
					t.Fatal("save unavailable:", m.lastErr)
				}
				m.Update(cmd())
				var stored string
				var done bool
				if err := db.QueryRow(`SELECT description, done FROM tasks WHERE id=1`).Scan(&stored, &done); err != nil {
					t.Fatal(err)
				}
				if m.lastErr != "" || stored != want || done || m.editor.dirty() {
					t.Fatalf("save: stored=%s done=%v dirty=%v error=%s", stored, done, m.editor.dirty(), m.lastErr)
				}
			}
			save(tc.want)
			m = nativeKeys(m, "u")
			if m.editor.draft.Description != source {
				t.Fatal("local Undo failed to restore checkbox:", m.editor.draft.Description)
			}
			save(source)
		})
	}
}
