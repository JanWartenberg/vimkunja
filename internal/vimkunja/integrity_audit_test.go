//go:build integrity_audit

package vimkunja

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"
)

// These opt-in tests assert preservation/compatibility contracts. Known defects
// deliberately fail until repaired; ordinary go test ./... remains usable.
func auditExec(t *testing.T, store *SQLiteStore, query string, args ...any) {
	t.Helper()
	if _, err := store.db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func auditInt(t *testing.T, store *SQLiteStore, query string, args ...any) int64 {
	t.Helper()
	var value int64
	if err := store.db.QueryRow(query, args...).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func auditState(t *testing.T, store *SQLiteStore) TaskUndoState {
	t.Helper()
	state, err := store.TaskUndoState(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func auditTitleChange(t *testing.T, store *SQLiteStore) (TaskUndoState, TaskUndoState) {
	t.Helper()
	before := auditState(t, store)
	if err := store.UpdateTaskTitle(context.Background(), 1, "After"); err != nil {
		t.Fatal(err)
	}
	return before, auditState(t, store)
}

func TestIntegrityAuditCompletedProjectMove(t *testing.T) {
	store := setupUndoStore(t)
	auditExec(t, store, `UPDATE tasks SET done = 1, done_at = '2026-09-01 10:00:00' WHERE id = 1;
		INSERT INTO buckets (id, title, project_view_id) VALUES (201, 'Done Two', 20);
		UPDATE project_views SET done_bucket_id = 201 WHERE id = 20;`)
	task, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 2, Title: "Completed"})
	if err != nil {
		t.Fatal(err)
	}
	if !task.Done {
		t.Fatal("Project move cleared Done")
	}
	if bucket := auditInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = 20`); bucket != 201 {
		t.Errorf("completed Task moved into Bucket %d, want target Done Bucket 201", bucket)
	}
}

func TestIntegrityAuditZeroBucketConfiguration(t *testing.T) {
	t.Run("creation_uses_first_bucket_without_configured_default", func(t *testing.T) {
		store := setupUndoStore(t)
		auditExec(t, store, `UPDATE project_views SET default_bucket_id = 0 WHERE id = 10;
			UPDATE buckets SET position = id;`)
		task, err := store.CreateTask(context.Background(), TaskDraft{ProjectID: 1, Title: "New"})
		if err != nil {
			t.Fatal(err)
		}
		if bucket := auditInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id = ? AND project_view_id = 10`, task.ID); bucket != 100 {
			t.Errorf("new Task assigned to Bucket %d, want first Bucket 100", bucket)
		}
		board, err := store.LoadManualKanbanBoard(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, bucket := range board.Buckets {
			for _, card := range bucket.Tasks {
				found = found || card.ID == task.ID
			}
		}
		if !found {
			t.Error("new Task disappeared from the manual Kanban board")
		}
	})
	t.Run("no_done_bucket_preserves_current_assignment", func(t *testing.T) {
		store := setupUndoStore(t)
		auditExec(t, store, `UPDATE project_views SET done_bucket_id = 0 WHERE id = 10;`)
		if err := store.ToggleTaskDone(context.Background(), 1, true); err != nil {
			t.Fatal(err)
		}
		if bucket := auditInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = 10`); bucket != 100 {
			t.Errorf("no Done Bucket configured: current Bucket changed to %d, want 100", bucket)
		}
	})
}

func TestIntegrityAuditCompletionTimestampPreservation(t *testing.T) {
	for _, action := range []string{"detail_title_save", "inline_title_undo"} {
		t.Run(action, func(t *testing.T) {
			store := setupUndoStore(t)
			auditExec(t, store, `UPDATE project_views SET done_bucket_id = 101 WHERE id = 10;
				UPDATE tasks SET done = 1, done_at = '2026-09-01 10:00:00' WHERE id = 1;
				UPDATE task_buckets SET bucket_id = 101 WHERE task_id = 1;`)
			if action == "detail_title_save" {
				_, err := store.UpdateTask(context.Background(), TaskDraft{
					ID: 1, ProjectID: 1, Title: "Renamed", Description: "old", Priority: 2,
					DueDate: "2026-09-01", WorkflowBucketID: 101, Done: true,
				})
				if err != nil {
					t.Fatal(err)
				}
			} else {
				before, after := auditTitleChange(t, store)
				if err := store.RestoreTaskUndoState(context.Background(), after, before); err != nil {
					t.Fatal(err)
				}
			}
			var doneAt time.Time
			if err := store.db.QueryRow(`SELECT done_at FROM tasks WHERE id = 1`).Scan(&doneAt); err != nil {
				t.Fatal(err)
			}
			if got := doneAt.Format(databaseTimestampLayout); got != "2026-09-01 10:00:00" {
				t.Errorf("%s rewrote unchanged completion timestamp to %s", action, got)
			}
		})
	}
}

func TestIntegrityAuditRecurringCompletion(t *testing.T) {
	store := setupUndoStore(t)
	auditExec(t, store, `ALTER TABLE tasks ADD COLUMN repeat_after INTEGER DEFAULT 0;
		ALTER TABLE tasks ADD COLUMN repeat_mode INTEGER DEFAULT 0;
		UPDATE tasks SET repeat_after = 86400, due_date = '2099-09-01 00:00:00' WHERE id = 1;
		UPDATE project_views SET done_bucket_id = 101 WHERE id = 10;`)
	if err := store.ToggleTaskDone(context.Background(), 1, true); err != nil {
		t.Fatal(err)
	}
	if done := auditInt(t, store, `SELECT done FROM tasks WHERE id = 1`); done != 0 {
		t.Error("recurring Task stayed Done instead of reopening for its next occurrence")
	}
	var due time.Time
	if err := store.db.QueryRow(`SELECT due_date FROM tasks WHERE id = 1`).Scan(&due); err != nil {
		t.Fatal(err)
	}
	if due.Format("2006-01-02") != "2099-09-02" {
		t.Errorf("recurring due date did not advance: %s", due.Format("2006-01-02"))
	}
}

func TestIntegrityAuditCalDAVIdentity(t *testing.T) {
	store := setupUndoStore(t)
	auditExec(t, store, `ALTER TABLE tasks ADD COLUMN uid TEXT;`)
	task, err := store.CreateTask(context.Background(), TaskDraft{ProjectID: 1, Title: "Calendar Task"})
	if err != nil {
		t.Fatal(err)
	}
	var uid sql.NullString
	if err := store.db.QueryRow(`SELECT uid FROM tasks WHERE id = ?`, task.ID).Scan(&uid); err != nil {
		t.Fatal(err)
	}
	if !uid.Valid || uid.String == "" {
		t.Error("created Task has no CalDAV UID")
	}
}

func TestIntegrityAuditTaskDeletionCleansPolymorphicRelations(t *testing.T) {
	store := setupUndoStore(t)
	// Enum values mirror the local Vikunja models: Task Favorite/Subscription
	// are 1/3; Task/Comment Reactions are 0/1. Keep unrelated entities as controls.
	auditExec(t, store, `CREATE TABLE favorites (entity_id INTEGER, kind INTEGER);
		CREATE TABLE subscriptions (entity_id INTEGER, entity_type INTEGER);
		CREATE TABLE reactions (entity_id INTEGER, entity_kind INTEGER);
		CREATE TABLE task_comments (id INTEGER PRIMARY KEY, task_id INTEGER);
		INSERT INTO task_comments VALUES (55, 1), (56, 2);
		INSERT INTO favorites VALUES (1, 1), (2, 1);
		INSERT INTO subscriptions VALUES (1, 3), (2, 3);
		INSERT INTO reactions VALUES (1, 0), (55, 1), (2, 0), (56, 1);`)
	if err := store.DeleteTask(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"favorites", "subscriptions"} {
		if count := auditInt(t, store, "SELECT COUNT(*) FROM "+table+" WHERE entity_id = 1"); count != 0 {
			t.Errorf("Task deletion left %d dangling %s rows", count, table)
		}
		if count := auditInt(t, store, "SELECT COUNT(*) FROM "+table+" WHERE entity_id = 2"); count != 1 {
			t.Errorf("Task deletion damaged unrelated %s rows", table)
		}
	}
	if count := auditInt(t, store, `SELECT COUNT(*) FROM reactions WHERE (entity_id = 1 AND entity_kind = 0) OR (entity_id = 55 AND entity_kind = 1)`); count != 0 {
		t.Errorf("Task deletion left %d dangling Task/Comment reactions", count)
	}
	if count := auditInt(t, store, `SELECT COUNT(*) FROM reactions WHERE entity_id IN (2, 56)`); count != 2 {
		t.Error("Task deletion damaged unrelated Reactions")
	}
}

func TestIntegrityAuditLateFailuresRollBack(t *testing.T) {
	t.Run("creation", func(t *testing.T) {
		store := setupUndoStore(t)
		_, err := store.CreateTask(context.Background(), TaskDraft{
			ProjectID: 1, Title: "Must roll back", LabelIDsSet: true, LabelIDs: []int64{999},
		})
		if err == nil {
			t.Fatal("missing Label was accepted")
		}
		if auditInt(t, store, `SELECT COUNT(*) FROM tasks`) != 1 ||
			auditInt(t, store, `SELECT COUNT(*) FROM task_positions`) != 1 ||
			auditInt(t, store, `SELECT COUNT(*) FROM task_buckets`) != 1 {
			t.Fatal("failed creation left Task or View assignments behind")
		}
	})
	t.Run("labels", func(t *testing.T) {
		store := setupUndoStore(t)
		before := auditState(t, store)
		auditExec(t, store, `CREATE TRIGGER reject_label_insert BEFORE INSERT ON label_tasks BEGIN SELECT RAISE(ABORT, 'injected Label failure'); END;`)
		if err := store.ReplaceTaskLabels(context.Background(), 1, []int64{8}); err == nil {
			t.Fatal("injected Label failure was ignored")
		}
		if !reflect.DeepEqual(auditState(t, store), before) {
			t.Fatal("failed replacement lost original Labels")
		}
	})
	t.Run("kanban_provisioning", func(t *testing.T) {
		store := setupUndoStore(t)
		auditExec(t, store, `INSERT INTO projects (id, title) VALUES (3, 'Empty');
			CREATE TRIGGER reject_second_bucket BEFORE INSERT ON buckets WHEN NEW.title = 'Doing' BEGIN SELECT RAISE(ABORT, 'injected provision failure'); END;`)
		if _, err := store.ProvisionStandardManualKanbanView(context.Background(), 3); err == nil {
			t.Fatal("injected provisioning failure was ignored")
		}
		if auditInt(t, store, `SELECT COUNT(*) FROM project_views WHERE project_id = 3`) != 0 ||
			auditInt(t, store, `SELECT COUNT(*) FROM buckets`) != 3 {
			t.Fatal("failed provisioning left a partial View or Bucket")
		}
	})
	t.Run("task_move", func(t *testing.T) {
		store := setupUndoStore(t)
		before := auditState(t, store)
		auditExec(t, store, `CREATE TRIGGER reject_new_position BEFORE INSERT ON task_positions BEGIN SELECT RAISE(ABORT, 'injected position failure'); END;`)
		_, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 2, Title: "After"})
		if err == nil {
			t.Fatal("injected late move failure was ignored")
		}
		if !reflect.DeepEqual(auditState(t, store), before) || auditInt(t, store, `SELECT COUNT(*) FROM task_positions WHERE task_id = 1 AND project_view_id = 10`) != 1 {
			t.Fatal("failed move partially changed Task or assignments")
		}
	})
	t.Run("undo", func(t *testing.T) {
		store := setupUndoStore(t)
		before, after := auditTitleChange(t, store)
		auditExec(t, store, `CREATE TRIGGER reject_restore_bucket BEFORE INSERT ON task_buckets BEGIN SELECT RAISE(ABORT, 'injected bucket failure'); END;`)
		if err := store.RestoreTaskUndoState(context.Background(), after, before); err == nil {
			t.Fatal("injected late undo failure was ignored")
		}
		if !reflect.DeepEqual(auditState(t, store), after) {
			t.Fatal("failed undo partially changed Task/Labels/Buckets")
		}
	})
	t.Run("deletion", func(t *testing.T) {
		store := setupUndoStore(t)
		before := auditState(t, store)
		auditExec(t, store, `CREATE TRIGGER reject_task_delete BEFORE DELETE ON tasks BEGIN SELECT RAISE(ABORT, 'injected Task deletion failure'); END;`)
		if err := store.DeleteTask(context.Background(), 1); err == nil {
			t.Fatal("injected late deletion failure was ignored")
		}
		if !reflect.DeepEqual(auditState(t, store), before) || auditInt(t, store, `SELECT COUNT(*) FROM task_positions WHERE task_id = 1`) != 1 {
			t.Fatal("failed deletion partially removed Task relations")
		}
	})
	t.Run("workflow", func(t *testing.T) {
		store := setupUndoStore(t)
		before := auditState(t, store)
		auditExec(t, store, `UPDATE project_views SET done_bucket_id = 101 WHERE id = 10;
			CREATE TRIGGER reject_bucket_update BEFORE UPDATE ON task_buckets BEGIN SELECT RAISE(ABORT, 'injected Bucket update failure'); END;`)
		if err := store.SetTaskWorkflowStatus(context.Background(), 1, 101); err == nil {
			t.Fatal("injected workflow failure was ignored")
		}
		if !reflect.DeepEqual(auditState(t, store), before) {
			t.Fatal("failed workflow write partially changed Done")
		}
	})
}

func TestIntegrityAuditSafeBoundaries(t *testing.T) {
	t.Run("invalid_due_date_preserves_task", func(t *testing.T) {
		store := setupUndoStore(t)
		before := auditState(t, store)
		if _, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Bad", DueDate: "2026-02-30"}); err == nil {
			t.Fatal("invalid date was accepted")
		}
		if !reflect.DeepEqual(auditState(t, store), before) {
			t.Fatal("invalid date changed Task")
		}
	})
	t.Run("cancelled_write_preserves_task", func(t *testing.T) {
		store := setupUndoStore(t)
		before := auditState(t, store)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := store.UpdateTask(ctx, TaskDraft{ID: 1, ProjectID: 2, Title: "Cancelled"}); err == nil {
			t.Fatal("cancelled write was accepted")
		}
		if !reflect.DeepEqual(auditState(t, store), before) {
			t.Fatal("cancelled write changed Task")
		}
	})
	t.Run("foreign_bucket_preserves_task", func(t *testing.T) {
		store := setupUndoStore(t)
		before := auditState(t, store)
		if err := store.SetTaskWorkflowStatus(context.Background(), 1, 200, 10); err == nil {
			t.Fatal("Bucket from another Project was accepted")
		}
		if !reflect.DeepEqual(auditState(t, store), before) {
			t.Fatal("foreign Bucket changed Task")
		}
	})
	t.Run("deleted_undo_project_aborts", func(t *testing.T) {
		store := setupUndoStore(t)
		before := auditState(t, store)
		if _, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 2, Title: "Moved"}); err != nil {
			t.Fatal(err)
		}
		after := auditState(t, store)
		auditExec(t, store, `DELETE FROM projects WHERE id = 1`)
		if err := store.RestoreTaskUndoState(context.Background(), after, before); err == nil {
			t.Fatal("undo into a deleted Project was accepted")
		}
		if !reflect.DeepEqual(auditState(t, store), after) {
			t.Fatal("failed undo changed Task")
		}
	})
	t.Run("scoped_done_preserves_other_view", func(t *testing.T) {
		store := setupUndoStore(t)
		auditExec(t, store, `UPDATE project_views SET done_bucket_id = 101 WHERE id = 10;
			INSERT INTO project_views (id, project_id, view_kind, default_bucket_id, done_bucket_id) VALUES (11, 1, 3, 110, 111);
			INSERT INTO buckets (id, title, project_view_id) VALUES (110, 'Todo Other', 11), (111, 'Done Other', 11);
			INSERT INTO task_buckets (task_id, project_view_id, bucket_id) VALUES (1, 11, 110);
			INSERT INTO task_positions (task_id, project_view_id, position) VALUES (1, 11, 42);`)
		if err := store.ToggleTaskDone(context.Background(), 1, true, 10); err != nil {
			t.Fatal(err)
		}
		if auditInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = 11`) != 110 ||
			auditInt(t, store, `SELECT position FROM task_positions WHERE task_id = 1 AND project_view_id = 11`) != 42 {
			t.Fatal("View-scoped toggle changed the other View")
		}
	})
}
