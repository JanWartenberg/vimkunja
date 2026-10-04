package vimkunja

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
)

func workflowExec(t *testing.T, store *SQLiteStore, query string, args ...any) {
	t.Helper()
	if _, err := store.db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
func TestIntegrityAuditUnchangedWorkflowPreservesDone(t *testing.T) {
	store := setupUndoStore(t)
	workflowExec(t, store, `UPDATE project_views SET done_bucket_id = 0 WHERE id = 10;
		UPDATE tasks SET done = 1, done_at = '2026-09-01 10:00:00' WHERE id = 1;`)
	_, err := store.UpdateTask(context.Background(), TaskDraft{
		ID: 1, ProjectID: 1, Title: "Renamed", Description: "old", Priority: 2,
		DueDate: "2026-09-01", WorkflowBucketID: 100, Done: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var done bool
	if err := store.db.QueryRow(`SELECT done FROM tasks WHERE id = 1`).Scan(&done); err != nil {
		t.Fatal(err)
	}
	if !done {
		t.Error("title-only Detail save cleared Done although neither Done nor workflow was edited")
	}
}

func TestWorkflowSaveKeepsCompletionMetadata(t *testing.T) {
	for _, config := range []any{nil, int64(0), int64(101)} {
		for _, withComment := range []bool{false, true} {
			t.Run(fmt.Sprintf("done_bucket=%v/comment=%t", config, withComment), func(t *testing.T) {
				store := setupUndoStore(t)
				workflowExec(t, store, `UPDATE project_views SET done_bucket_id=? WHERE id=10`, config)
				const doneAt = "2026-09-01T10:00:00.123456789+02:00"
				bucket := int64(100)
				if config == int64(101) {
					bucket = 101
				}
				workflowExec(t, store, `UPDATE tasks SET done=1, done_at=? WHERE id=1`, doneAt)
				workflowExec(t, store, `UPDATE task_buckets SET bucket_id=? WHERE task_id=1`, bucket)
				workflowExec(t, store, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, username TEXT);
                    CREATE TABLE task_comments (id INTEGER PRIMARY KEY, comment TEXT, author_id INTEGER, task_id INTEGER, created DATETIME, updated DATETIME);
                    INSERT INTO users VALUES (1,'Local','local');`)
				draft := TaskDraft{ID: 1, ProjectID: 1, Title: "Renamed", DueDate: "2026-09-01", WorkflowBucketID: bucket, Done: true}
				var task Task
				var err error
				if withComment {
					task, _, err = store.SaveTaskWithComment(context.Background(), draft, TaskCommentDraft{Text: "Added"})
				} else {
					task, err = store.UpdateTask(context.Background(), draft)
				}
				if err != nil || !task.Done {
					t.Fatalf("save: task=%#v err=%v", task, err)
				}
				var gotDone bool
				var gotTime string
				var gotBucket int64
				if err := store.db.QueryRow(`SELECT t.done,CAST(t.done_at AS TEXT),tb.bucket_id FROM tasks t JOIN task_buckets tb ON tb.task_id=t.id WHERE t.id=1 AND tb.project_view_id=10`).Scan(&gotDone, &gotTime, &gotBucket); err != nil {
					t.Fatal(err)
				}
				if !gotDone || gotTime != doneAt || gotBucket != bucket {
					t.Fatalf("completion changed: done=%v time=%q bucket=%d", gotDone, gotTime, gotBucket)
				}
			})
		}
	}
}

func TestWorkflowSaveAppliesExplicitChanges(t *testing.T) {
	store := setupUndoStore(t)
	workflowExec(t, store, `UPDATE project_views SET done_bucket_id=101 WHERE id=10`)
	for _, bucket := range []int64{101, 100} {
		task, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Changed", WorkflowBucketID: bucket})
		if err != nil || task.Done != (bucket == 101) {
			t.Fatalf("explicit bucket change: task=%#v err=%v", task, err)
		}
		var gotBucket int64
		var doneAt sql.NullString
		if err := store.db.QueryRow(`SELECT tb.bucket_id,CAST(t.done_at AS TEXT) FROM tasks t JOIN task_buckets tb ON tb.task_id=t.id WHERE t.id=1 AND tb.project_view_id=10`).Scan(&gotBucket, &doneAt); err != nil {
			t.Fatal(err)
		}
		if gotBucket != bucket || doneAt.Valid != (bucket == 101) {
			t.Fatalf("bucket=%d done_at=%v", gotBucket, doneAt)
		}
	}
	workflowExec(t, store, `UPDATE tasks SET done=1,done_at='2026-09-01 10:00:00' WHERE id=1`)
	task, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Reopened", WorkflowBucketID: 100, Done: false, DoneSet: true})
	if err != nil || task.Done {
		t.Fatalf("explicit Done change ignored: task=%#v err=%v", task, err)
	}
}

func TestWorkflowSaveRejectsInvalidBucketAndRollsBack(t *testing.T) {
	for _, bucket := range []int64{999, 200} {
		t.Run(fmt.Sprint(bucket), func(t *testing.T) {
			store := setupUndoStore(t)
			workflowExec(t, store, `UPDATE tasks SET done=1,done_at='2026-09-01 10:00:00' WHERE id=1`)
			// A stale relation must not bypass ownership validation.
			workflowExec(t, store, `INSERT INTO task_buckets (task_id,bucket_id,project_view_id) VALUES (1,?,20)`, bucket)
			before, err := store.TaskUndoState(context.Background(), 1)
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Must rollback", WorkflowBucketID: bucket})
			if err == nil {
				t.Fatal("invalid bucket was accepted")
			}
			after, err := store.TaskUndoState(context.Background(), 1)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("failed workflow save changed state: %#v %v", after, err)
			}
		})
	}
}

func TestWorkflowSavePreservesOtherViews(t *testing.T) {
	store := setupUndoStore(t)
	workflowExec(t, store, `INSERT INTO project_views (id,project_id,title,view_kind,bucket_configuration_mode,done_bucket_id) VALUES(11,1,'Second',3,1,111);
        INSERT INTO buckets (id,title,project_view_id) VALUES(110,'Second Todo',11),(111,'Second Done',11);
        INSERT INTO task_buckets (task_id,bucket_id,project_view_id) VALUES(1,111,11);
        UPDATE tasks SET done=1,done_at='2026-09-01 10:00:00' WHERE id=1;`)
	before, err := store.TaskUndoState(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	task, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Renamed", DueDate: "2026-09-01", WorkflowBucketID: 100})
	if err != nil || !task.Done {
		t.Fatalf("unchanged multi-View save: %#v %v", task, err)
	}
	after, err := store.TaskUndoState(context.Background(), 1)
	if err != nil || !reflect.DeepEqual(before.BucketAssignments, after.BucketAssignments) {
		t.Fatalf("changed other View: %#v %v", after, err)
	}
}
