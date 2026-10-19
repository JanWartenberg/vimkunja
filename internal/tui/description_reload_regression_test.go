package tui

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

func reloadRegressionModel(t *testing.T, withLabel bool) (*model, *sql.DB) {
	t.Helper()
	m, db := auditEditorModel(t)
	if _, err := db.Exec(`CREATE TABLE labels(id INTEGER PRIMARY KEY, title TEXT); INSERT INTO labels VALUES(7,'label'); INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task','<p>Alpha beta</p>',0,0,1)`); err != nil {
		t.Fatal(err)
	}
	if withLabel {
		if _, err := db.Exec(`INSERT INTO label_tasks VALUES(1,7)`); err != nil {
			t.Fatal(err)
		}
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "<p>Alpha beta</p>"})
	m.Update(m.loadEditorLabels(1)())
	if m.lastErr != "" {
		t.Fatal(m.lastErr)
	}
	nativeKeys(m, "g", "n", "enter")
	m.View()
	return m, db
}

func TestDescriptionReloadAfterLabelsLoadedReadsBrowserEdit(t *testing.T) {
	for _, withLabel := range []bool{false, true} {
		for _, saveFirst := range []bool{false, true} {
			name := "without labels"
			if withLabel {
				name = "with label"
			}
			if saveFirst {
				name += " after TUI save"
			}
			t.Run(name, func(t *testing.T) {
				m, db := reloadRegressionModel(t, withLabel)
				if saveFirst {
					nativeKeys(m, "A", " delta", "esc")
					cmd := enterModalCommand(m, "w")
					if cmd == nil {
						t.Fatal(m.lastErr)
					}
					m.Update(cmd())
					if m.lastErr != "" || m.editor.dirty() {
						t.Fatalf("TUI save failed: %s", m.lastErr)
					}
				}
				if _, err := db.Exec(`UPDATE tasks SET description='<p>Alpha gamma</p>' WHERE id=1`); err != nil {
					t.Fatal(err)
				}
				cmd := enterModalCommand(m, "descriptionreload")
				if cmd == nil {
					t.Fatal("reload did not issue a command: " + m.lastErr)
				}
				m.Update(cmd())
				if m.lastErr != "" || m.editor.draft.Description != "<p>Alpha gamma</p>" || m.editor.dirty() {
					t.Fatalf("unchanged draft falsely rejected browser reload: %s; description=%q; labels=%#v", m.lastErr, m.editor.draft.Description, m.editor.draft.LabelIDs)
				}
				if !strings.Contains(ansi.Strip(m.View().Content), "Alpha gamma") {
					t.Fatal("reloaded browser text not visible")
				}
				var source string
				if err := db.QueryRow(`SELECT description FROM tasks WHERE id=1`).Scan(&source); err != nil || source != "<p>Alpha gamma</p>" {
					t.Fatal("reload changed browser data", err, source)
				}
			})
		}
	}
}

func TestDescriptionReloadStillKeepsEditsMadeAfterDispatch(t *testing.T) {
	for _, field := range []string{"Title", "Description", "Labels"} {
		t.Run(field, func(t *testing.T) {
			m, _ := reloadRegressionModel(t, true)
			cmd := enterModalCommand(m, "descriptionreload")
			if cmd == nil {
				t.Fatal(m.lastErr)
			}
			switch field {
			case "Title":
				m.editor.draft.Title = "Late Title"
			case "Description":
				nativeKeys(m, "A", " late", "esc")
			case "Labels":
				m.editor.draft.LabelIDs[0] = 8
			}
			before := cloneTaskDraft(m.editor.draft)
			m.Update(cmd())
			if !strings.Contains(m.lastErr, "changed while reloading") {
				t.Fatal("late edit was discarded", m.lastErr)
			}
			if m.editor.draft.Title != before.Title || m.editor.draft.Description != before.Description || !sameLabelIDs(m.editor.draft.LabelIDs, before.LabelIDs) {
				t.Fatal("late draft changed")
			}
		})
	}
}
