package vimkunja

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestIntegrityAuditCompletionTimestampPreservation(t *testing.T) {
	for _, action := range []string{"detail_title_save", "inline_title_undo"} {
		t.Run(action, func(t *testing.T) {
			store := setupUndoStore(t)
			bucketExec(t, store, `UPDATE project_views SET done_bucket_id = 101 WHERE id = 10;
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
				before, err := store.TaskUndoState(context.Background(), 1)
				if err != nil {
					t.Fatal(err)
				}
				if err := store.UpdateTaskTitle(context.Background(), 1, "Renamed"); err != nil {
					t.Fatal(err)
				}
				after, err := store.TaskUndoState(context.Background(), 1)
				if err != nil {
					t.Fatal(err)
				}
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

func completionTime(t *testing.T, store *SQLiteStore) sql.NullString {
	t.Helper()
	var value sql.NullString
	if err := store.db.QueryRow(`SELECT CAST(done_at AS TEXT) FROM tasks WHERE id=1`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestCompletionTimeUnchangedDonePreservesExactValue(t *testing.T) {
	for _, done := range []bool{false, true} {
		for _, timestamp := range []any{nil, "2026-09-01 10:00:00", "2026-09-01T10:00:00.123456789+02:00"} {
			for _, action := range []string{"undo_redo", "undo_redo_external", "toggle", "workflow", "draft", "comment"} {
				t.Run(fmt.Sprintf("%t/%v/%s", done, timestamp, action), func(t *testing.T) {
					store := setupUndoStore(t)
					ctx := context.Background()
					bucketExec(t, store, `UPDATE project_views SET done_bucket_id=101 WHERE id=10; CREATE TABLE users(id INTEGER PRIMARY KEY,name TEXT,username TEXT); CREATE TABLE task_comments(id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME); INSERT INTO users VALUES(1,'Local','local');`)
					bucketExec(t, store, `UPDATE tasks SET done=?,done_at=? WHERE id=1`, done, timestamp)
					bucket := int64(100)
					if done {
						bucket = 101
					}
					bucketExec(t, store, `UPDATE task_buckets SET bucket_id=? WHERE task_id=1`, bucket)
					original := completionTime(t, store)
					draft := TaskDraft{ID: 1, ProjectID: 1, Title: "Renamed", DueDate: "2026-09-01", Done: done, DoneSet: true}
					var err error
					switch action {
					case "undo_redo", "undo_redo_external":
						before, e := store.TaskUndoState(ctx, 1)
						if e != nil {
							t.Fatal(e)
						}
						if e := store.UpdateTaskTitle(ctx, 1, "Renamed"); e != nil {
							t.Fatal(e)
						}
						after, e := store.TaskUndoState(ctx, 1)
						if e != nil {
							t.Fatal(e)
						}
						if action == "undo_redo_external" {
							// Completion metadata is excluded from history. Preserve external edits.
							bucketExec(t, store, `UPDATE tasks SET done_at=? WHERE id=1`, "2026-09-02T09:00:00.987654321-03:00")
							original = completionTime(t, store)
						}
						if err = store.RestoreTaskUndoState(ctx, after, before); err != nil {
							t.Fatal(err)
						}
						if got := completionTime(t, store); got != original {
							t.Fatalf("undo rewrote timestamp: %v", got)
						}
						err = store.RestoreTaskUndoState(ctx, before, after)
					case "toggle":
						err = store.ToggleTaskDone(ctx, 1, done)
					case "workflow":
						err = store.SetTaskWorkflowStatus(ctx, 1, bucket)
					case "draft":
						_, err = store.UpdateTask(ctx, draft)
					case "comment":
						_, _, err = store.SaveTaskWithComment(ctx, draft, TaskCommentDraft{Text: "Saved"})
					}
					if err != nil {
						t.Fatal(err)
					}
					if got := completionTime(t, store); got != original {
						t.Fatalf("unchanged Done rewrote timestamp: got %#v want %#v", got, original)
					}
				})
			}
		}
	}
}

func TestCompletionTimeTransitionsAndUndo(t *testing.T) {
	store := setupUndoStore(t)
	ctx := context.Background()
	bucketExec(t, store, `UPDATE project_views SET done_bucket_id=101 WHERE id=10`)
	before, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetTaskWorkflowStatus(ctx, 1, 101); err != nil {
		t.Fatal(err)
	}
	if !completionTime(t, store).Valid {
		t.Fatal("completion did not set timestamp")
	}
	after, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreTaskUndoState(ctx, after, before); err != nil {
		t.Fatal(err)
	}
	if got := completionTime(t, store); got.Valid {
		t.Fatal("undo completion did not clear timestamp")
	}
	// Redo must create a current timestamp; completion metadata is not restored.
	bucketExec(t, store, `UPDATE tasks SET done_at='2000-01-01 00:00:00' WHERE id=1`)
	if err := store.RestoreTaskUndoState(ctx, before, after); err != nil {
		t.Fatal(err)
	}
	if got := completionTime(t, store); !got.Valid || got.String == "2000-01-01 00:00:00" {
		t.Fatal("redo completion did not set a new timestamp")
	}
	if err := store.ToggleTaskDone(ctx, 1, false); err != nil {
		t.Fatal(err)
	}
	if got := completionTime(t, store); got.Valid {
		t.Fatal("reopening did not clear timestamp")
	}
	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatal(err)
	}
	if got := completionTime(t, store); !got.Valid {
		t.Fatal("completion did not set timestamp")
	}
}

func TestCompletionTimeUndoLateFailureRollsBack(t *testing.T) {
	store := setupUndoStore(t)
	ctx := context.Background()
	bucketExec(t, store, `UPDATE tasks SET done=1,done_at='2026-09-01T10:00:00.123456789+02:00' WHERE id=1`)
	before, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateTaskTitle(ctx, 1, "After"); err != nil {
		t.Fatal(err)
	}
	after, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	original := completionTime(t, store)
	bucketExec(t, store, `CREATE TRIGGER reject_undo_bucket BEFORE INSERT ON task_buckets BEGIN SELECT RAISE(ABORT,'late undo failure'); END;`)
	if err := store.RestoreTaskUndoState(ctx, after, before); err == nil {
		t.Fatal("injected failure accepted")
	}
	actual, err := store.TaskUndoState(ctx, 1)
	if err != nil || !reflect.DeepEqual(actual, after) || completionTime(t, store) != original {
		t.Fatalf("failed undo changed persisted state: %#v %v", actual, err)
	}
}

func TestCompletionTimeBucketMoveWithSameDone(t *testing.T) {
	for _, done := range []bool{false, true} {
		t.Run(fmt.Sprint(done), func(t *testing.T) {
			store := setupUndoStore(t)
			ctx := context.Background()
			bucketExec(t, store, `UPDATE project_views SET done_bucket_id=101 WHERE id=10; INSERT INTO buckets(id,title,project_view_id) VALUES(102,'Doing',10),(111,'Other Done',11); INSERT INTO project_views(id,project_id,title,view_kind,bucket_configuration_mode,done_bucket_id) VALUES(11,1,'Other',3,1,111);`)
			bucketExec(t, store, `UPDATE tasks SET done=?,done_at='2026-09-01T10:00:00.123456789+02:00' WHERE id=1`, done)
			original := completionTime(t, store)
			target, view := int64(102), int64(10)
			if done {
				target, view = 111, 11
			}
			if err := store.SetTaskWorkflowStatus(ctx, 1, target, view); err != nil {
				t.Fatal(err)
			}
			if got := completionTime(t, store); got != original {
				t.Fatalf("bucket move rewrote unchanged Done metadata: %#v", got)
			}
			if got := bucketInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id=1 AND project_view_id=?`, view); got != target {
				t.Fatal("bucket move was skipped")
			}
		})
	}
}
