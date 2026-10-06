package tui

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

// Exercise the real Ex-command -> asynchronous save -> SQLite -> completion
// path, holding completion messages to simulate slow disk without sleeps.
func auditEditorModel(t *testing.T) (*model, *sql.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "audit.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE projects (id INTEGER PRIMARY KEY, identifier TEXT, owner_id INTEGER, is_archived INTEGER DEFAULT 0);
		CREATE TABLE tasks (id INTEGER PRIMARY KEY AUTOINCREMENT, project_id INTEGER, title TEXT,
			description TEXT, priority INTEGER, due_date DATETIME, done INTEGER, done_at DATETIME, deleted_at DATETIME,
			"index" INTEGER, created DATETIME, updated DATETIME, created_by_id INTEGER);
		CREATE TABLE project_views (id INTEGER PRIMARY KEY, project_id INTEGER,
			view_kind INTEGER, bucket_configuration_mode INTEGER, default_bucket_id INTEGER);
		CREATE TABLE task_positions (task_id INTEGER, project_view_id INTEGER, position REAL);
		CREATE TABLE task_buckets (task_id INTEGER, project_view_id INTEGER, bucket_id INTEGER);
		CREATE TABLE label_tasks (task_id INTEGER, label_id INTEGER);
		CREATE TABLE project_task_counters (project_id INTEGER PRIMARY KEY, last_index INTEGER NOT NULL DEFAULT 0);
		CREATE TABLE task_index_aliases (project_id INTEGER NOT NULL, "index" INTEGER NOT NULL, task_id INTEGER NOT NULL, UNIQUE(project_id, "index"));
		INSERT INTO projects(id, identifier, owner_id) VALUES (1, 'AUDIT', 1);
	`)
	if err != nil {
		t.Fatal(err)
	}
	store, err := vimkunja.NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	m := NewModel(Config{Store: store}).(*model)
	m.currentProjectID = 1
	m.projects = []vimkunja.Project{{ID: 1, Title: "Audit"}}
	m.showDetail, m.focus = true, paneDetail
	m.editor = &taskEditor{draft: vimkunja.TaskDraft{ProjectID: 1, Title: "New"}, field: fieldTitle}
	return m, db
}

func TestIntegrityAuditRepeatedWriteDoesNotDuplicateTask(t *testing.T) {
	m, db := auditEditorModel(t)
	m, first := typeCommand(t, m, "w")
	if first == nil || !m.taskLoading {
		t.Fatal("first write was not started")
	}
	_, second := typeCommand(t, m, "w")
	if second != nil {
		t.Error("second :w scheduled another save while the first save was pending")
	}
	if msg := first().(taskSavedMsg); msg.err != nil {
		t.Fatal(msg.err)
	}
	if second != nil {
		if msg := second().(taskSavedMsg); msg.err != nil {
			t.Fatal(msg.err)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("repeated :w created %d Tasks, want one", count)
	}
}

func TestIntegrityAuditEditDuringSaveRemainsDirty(t *testing.T) {
	m, db := auditEditorModel(t)
	m, save := typeCommand(t, m, "w")
	if save == nil {
		t.Fatal("write was not started")
	}
	// Real editor keys between dispatching the save and receiving its result.
	for _, key := range []tea.Key{{Code: 'i', Text: "i"}, {Code: 'X', Text: "X"}, {Code: tea.KeyEscape}} {
		updated, _ := m.Update(tea.KeyPressMsg(key))
		m = updated.(*model)
	}
	msg := save().(taskSavedMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	updated, _ := m.Update(msg)
	m = updated.(*model)
	var title string
	if err := db.QueryRow(`SELECT title FROM tasks WHERE id = ?`, msg.task.ID).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if m.editor.draft.Title != title && !m.editor.dirty() {
		t.Errorf("UI draft %q differs from stored title %q but is marked clean", m.editor.draft.Title, title)
	}
}

func TestDetailSaveBlocksRepeatedCommandsAndDraftInput(t *testing.T) {
	for _, repeat := range []string{"w", "wq", "wqa", "q!", "qa!", "e!", "new"} {
		t.Run(repeat, func(t *testing.T) {
			m, _ := auditEditorModel(t)
			editor := m.editor
			m, save := typeCommand(t, m, "w")
			m, repeated := typeCommand(t, m, repeat)
			if repeated != nil || m.editor != editor || m.quitAfterSave {
				t.Fatal("input changed or rescheduled an in-flight Detail save")
			}
			m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
			m = pressKey(m, tea.Key{Code: 'X', Text: "X"})
			if editor.draft.Title != "New" || editor.mode != editorNormal {
				t.Fatal("draft input was not blocked during save")
			}
			// Dispatch guard must also protect callers that bypass key handling.
			if m.saveEditor(false) != nil {
				t.Fatal("direct save dispatch scheduled a second write")
			}
			if msg := save().(taskSavedMsg); msg.err != nil {
				t.Fatal(msg.err)
			} else {
				m.Update(msg)
			}
			m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
			m = pressKey(m, tea.Key{Code: 'X', Text: "X"})
			if editor.draft.Title != "XNew" || !editor.dirty() {
				t.Fatal("editor was not unlocked after a successful save")
			}
		})
	}
}

func TestDetailSaveFailureKeepsDraftAndAllowsRetry(t *testing.T) {
	m := testModel()
	store := m.store.(*fakeStore)
	store.updateErr = errors.New("injected save failure")
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Before"})
	m.editor.draft.Title = "After"
	m, save := typeCommand(t, m, "wqa")
	m.Update(save())
	if m.editor.draft.Title != "After" || !m.editor.dirty() || m.quitAfterSave || m.lastErr != "injected save failure" {
		t.Fatal("failed save lost the draft or retained quit intent")
	}
	store.updateErr = nil
	m, retry := typeCommand(t, m, "w")
	if retry == nil {
		t.Fatal("failed save did not release its dispatch guard")
	}
	m.Update(retry())
	if m.editor.dirty() || m.editor.original.Title != "After" {
		t.Fatal("retry did not persist the draft")
	}
}

func TestDetailSaveIgnoresCompletionForReplacedEditor(t *testing.T) {
	m, _ := auditEditorModel(t)
	m, save := typeCommand(t, m, "wq")
	// A background refresh can replace the editor without a keyboard action.
	m.openExistingEditor(vimkunja.Task{ID: 42, ProjectID: 1, Title: "Other"})
	replacement := m.editor
	m.Update(save())
	if m.editor != replacement || replacement.id != 42 || replacement.draft.Title != "Other" {
		t.Fatal("old save completion closed or rewrote a replacement editor")
	}
	if m.saveEditor(false) == nil {
		t.Fatal("old save completion failed to release the save guard")
	}
}

func TestDetailSavePreservesLaterAsyncDraftChanges(t *testing.T) {
	m, _ := auditEditorModel(t)
	m, save := typeCommand(t, m, "wqa")
	// A queued asynchronous result must not be promoted to the saved baseline.
	m.editor.draft.Description = "Later clipboard text"
	_, next := m.Update(save())
	if !m.editor.dirty() || m.editor.original.Description != "" {
		t.Fatal("save completion marked a later draft mutation as persisted")
	}
	if next != nil {
		if _, quit := next().(tea.QuitMsg); quit {
			t.Fatal("save-and-quit discarded a later unsaved draft")
		}
	}
}

func TestDetailSaveGuardSurvivesBackgroundTaskLoad(t *testing.T) {
	m, _ := auditEditorModel(t)
	m, save := typeCommand(t, m, "w")
	m.Update(tasksLoadedMsg{})
	if m.saveEditor(false) != nil {
		t.Fatal("background task load released an in-flight save")
	}
	m.Update(save())
	if m.toast == "Saving Task…" {
		t.Fatal("save completion left stale saving feedback")
	}
}

func TestDetailSaveSnapshotDoesNotAliasDraftLabels(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Before"})
	m.editor.draft.LabelIDs, m.editor.draft.LabelIDsSet = []int64{7}, true
	save := m.saveEditor(false)
	m.editor.draft.LabelIDs[0] = 8
	msg := save().(taskSavedMsg)
	if msg.err != nil || msg.draft.LabelIDs[0] != 7 || m.store.(*fakeStore).updatedDraft.LabelIDs[0] != 7 {
		t.Fatal("asynchronous save shared mutable Label IDs with the live draft")
	}
	m.Update(msg)
	if !m.editor.dirty() || m.editor.original.LabelIDs[0] != 7 {
		t.Fatal("save completion marked later Label IDs as persisted")
	}
}

func TestDetailSaveLateLabelLoadPreservesUnsavedDescription(t *testing.T) {
	m, _ := auditEditorModel(t)
	m, save := typeCommand(t, m, "w")
	m.editor.draft.Description = "Later text"
	m.Update(taskLabelsLoadedMsg{taskID: 0, labels: []vimkunja.Label{{ID: 7, Title: "Loaded"}}})
	m.Update(save())
	if !m.editor.dirty() || m.editor.original.Description != "" || len(m.editor.original.LabelIDs) != 1 {
		t.Fatal("late Label metadata marked unrelated text as saved or lost loaded Labels")
	}
}

func TestDetailSaveKeepsInitialCloseAndQuitIntent(t *testing.T) {
	for _, command := range []string{"wq", "wqa"} {
		t.Run(command, func(t *testing.T) {
			m, _ := auditEditorModel(t)
			m, save := typeCommand(t, m, command)
			m, repeated := typeCommand(t, m, "w")
			if repeated != nil {
				t.Fatal("repeat write changed initial save intent")
			}
			_, next := m.Update(save())
			if command == "wq" {
				if m.editor != nil {
					t.Fatal("successful :wq did not close Detail")
				}
			} else if next == nil {
				t.Fatal("successful :wqa did not quit")
			} else if _, quit := next().(tea.QuitMsg); !quit {
				t.Fatal("successful :wqa returned a non-quit command")
			}
		})
	}
}

func TestDetailSaveMissingTargetProjectKeepsDraft(t *testing.T) {
	m, db := auditEditorModel(t)
	task, err := m.store.CreateTask(context.Background(), vimkunja.TaskDraft{ProjectID: 1, Title: "Before"})
	if err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(task)
	m.editor.draft.ProjectID, m.editor.draft.Title = 999, "Moved"
	m, save := typeCommand(t, m, "wqa")
	if save == nil {
		t.Fatal("save was not dispatched")
	}
	_, next := m.Update(save())
	if next != nil || m.quitAfterSave || m.savingEditor != nil ||
		!strings.Contains(m.lastErr, "project 999 not found") ||
		m.editor.draft.ProjectID != 999 || m.editor.draft.Title != "Moved" || !m.editor.dirty() || len(m.undoHistory) != 0 {
		t.Fatal("rejected Project move lost draft, quit, or created an Undo Entry")
	}
	var projectID int64
	var title string
	if err := db.QueryRow(`SELECT project_id, title FROM tasks WHERE id = ?`, task.ID).Scan(&projectID, &title); err != nil || projectID != 1 || title != "Before" {
		t.Fatalf("rejected move changed persisted Task: Project=%d title=%q err=%v", projectID, title, err)
	}
}
