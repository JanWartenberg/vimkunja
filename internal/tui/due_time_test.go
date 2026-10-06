package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

func TestDetailSaveAndUndoKeepDueTimestamp(t *testing.T) {
	m, db := auditEditorModel(t)
	if _, err := db.Exec(`ALTER TABLE project_views ADD COLUMN done_bucket_id INTEGER;
		CREATE TABLE buckets (id INTEGER PRIMARY KEY, title TEXT, project_view_id INTEGER,"limit" INTEGER DEFAULT 0);`); err != nil {
		t.Fatal(err)
	}
	const due = "2026-09-01T18:45:12.123456789+02:00"
	if _, err := db.Exec(`INSERT INTO tasks (id, project_id, title, description, priority, due_date, done, "index") VALUES (1, 1, 'Before', '', 0, ?, 0, 1)`, due); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Before", DueDate: "2026-09-01"})
	m.editor.draft.Title = "After"
	m, save := typeCommand(t, m, "w")
	if save == nil {
		t.Fatal("no save command")
	}
	m.Update(save())
	m.Update(m.loadTasks(1, 0)())
	if m.lastErr != "" || m.editor.dirty() || m.editor.draft.DueDate != "2026-09-01" || len(m.undoHistory) != 1 {
		t.Fatalf("completion lost date-only baseline/history: error=%q editor=%#v", m.lastErr, m.editor)
	}
	if !reflect.DeepEqual(m.undoHistory[0].changed, []string{"title"}) {
		t.Fatalf("unexpected changed fields: %v", m.undoHistory[0].changed)
	}
	for _, redo := range []bool{false, true} {
		cmd := m.startUndo(redo)
		if cmd == nil {
			t.Fatalf("no undo/redo command: %s", m.lastErr)
		}
		m.Update(cmd())
		m.Update(m.loadTasks(1, 0)())
		if m.lastErr != "" {
			t.Fatal(m.lastErr)
		}
		var actual string
		if err := db.QueryRow(`SELECT CAST(due_date AS TEXT) FROM tasks WHERE id = 1`).Scan(&actual); err != nil || actual != due {
			t.Fatalf("undo/redo changed due timestamp: %q %v", actual, err)
		}
		if m.editor.draft.DueDate != "2026-09-01" || m.editor.dirty() {
			t.Fatal("undo/redo changed date-only editor contract")
		}
	}
	const external = "2026-09-01T20:30:12.123456789+02:00"
	if _, err := db.Exec(`UPDATE tasks SET due_date = ? WHERE id = 1`, external); err != nil {
		t.Fatal(err)
	}
	cmd := m.startUndo(false)
	if cmd == nil {
		t.Fatal("no conflict-checking Undo command")
	}
	m.Update(cmd())
	if !strings.Contains(m.lastErr, "changed since") || len(m.undoHistory) != 1 || len(m.redoHistory) != 0 || m.editor.draft.Title != "After" {
		t.Fatalf("rejected Undo consumed history or changed draft: error=%q undo=%d redo=%d title=%q", m.lastErr, len(m.undoHistory), len(m.redoHistory), m.editor.draft.Title)
	}
	var actual string
	if err := db.QueryRow(`SELECT CAST(due_date AS TEXT) FROM tasks WHERE id = 1`).Scan(&actual); err != nil || actual != external {
		t.Fatalf("rejected Undo changed timestamp: %q %v", actual, err)
	}

}

func TestUndoDetectsFullDueTimestampChange(t *testing.T) {
	before := vimkunja.TaskUndoState{TaskID: 1, DueDate: "2026-09-01", DueDateStored: "2026-09-01 18:45:00"}
	after := before
	after.DueDateStored = "2026-09-01 20:30:00"
	entry := newUndoEntry(before, after)
	if entry == nil || !reflect.DeepEqual(entry.changed, []string{"due date"}) {
		t.Fatalf("time-only change missing from undo history: %#v", entry)
	}
}
