package vimkunja

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func bucketExec(t *testing.T, store *SQLiteStore, query string, args ...any) {
	t.Helper()
	if _, err := store.db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
func bucketInt(t *testing.T, store *SQLiteStore, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := store.db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func TestIntegrityAuditZeroBucketConfiguration(t *testing.T) {
	t.Run("creation_uses_first_bucket_without_configured_default", func(t *testing.T) {
		store := setupUndoStore(t)
		bucketExec(t, store, `UPDATE project_views SET default_bucket_id = 0 WHERE id = 10;
			UPDATE buckets SET position = id;`)
		task, err := store.CreateTask(context.Background(), TaskDraft{ProjectID: 1, Title: "New"})
		if err != nil {
			t.Fatal(err)
		}
		if bucket := bucketInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id = ? AND project_view_id = 10`, task.ID); bucket != 100 {
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
		bucketExec(t, store, `UPDATE project_views SET done_bucket_id = 0 WHERE id = 10;`)
		if err := store.ToggleTaskDone(context.Background(), 1, true); err != nil {
			t.Fatal(err)
		}
		if bucket := bucketInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = 10`); bucket != 100 {
			t.Errorf("no Done Bucket configured: current Bucket changed to %d, want 100", bucket)
		}
	})
}

func TestBucketConfigurationFallbackAcrossSaveAndReadPaths(t *testing.T) {
	for _, configured := range []string{"NULL", "0"} {
		t.Run(configured, func(t *testing.T) {
			store := setupUndoStore(t)
			store.db.SetMaxOpenConns(1)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			bucketExec(t, store, `UPDATE project_views SET default_bucket_id=`+configured+`, done_bucket_id=`+configured+`;
				UPDATE buckets SET position = 20;
				INSERT INTO buckets (id,title,project_view_id,position) VALUES
				(105,'Later ID',10,1), (104,'First',10,1), (0,'Invalid zero',10,0), (201,'First destination',20,1);`)
			// List metadata must resolve the default even without any current assignment.
			bucketExec(t, store, `DELETE FROM task_buckets WHERE task_id=1`)
			tasks, _, err := store.ListTasksInView(ctx, 1, 10, 1, 20)
			if err != nil || len(tasks) != 1 || !tasks[0].Workflow.Available || tasks[0].Workflow.DefaultBucketID != 104 {
				t.Fatalf("list fallback without assignment = %#v, %v", tasks, err)
			}
			workflow, err := store.GetTaskWorkflowStatus(ctx, 1, 0)
			if err != nil || workflow.CurrentBucket.ID != 104 {
				t.Fatalf("new editor default = %#v, %v", workflow, err)
			}
			task, err := store.CreateTask(ctx, TaskDraft{ProjectID: 1, Title: "Fallback"})
			if err != nil {
				t.Fatal(err)
			}
			board, err := store.LoadManualKanbanBoard(ctx, 1)
			if err != nil || board.View.DefaultBucketID != 104 {
				t.Fatalf("board default = %#v, %v", board.View, err)
			}
			if got := bucketInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id=?`, task.ID); got != 104 {
				t.Fatalf("created assignment = %d", got)
			}
			for _, done := range []bool{false, true} {
				// Project moves use the same fallback for both persisted Done states.
				bucketExec(t, store, `UPDATE tasks SET done=? WHERE id=?`, done, task.ID)
				_, err = store.UpdateTask(ctx, TaskDraft{ID: task.ID, ProjectID: 2, Title: "Moved"})
				if err != nil {
					t.Fatal(err)
				}
				if got := bucketInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id=? AND project_view_id=20`, task.ID); got != 201 {
					t.Fatalf("destination assignment = %d, done=%t", got, done)
				}
				_, err = store.UpdateTask(ctx, TaskDraft{ID: task.ID, ProjectID: 1, Title: "Returned"})
				if err != nil {
					t.Fatal(err)
				}
			}
			bucketExec(t, store, `UPDATE task_buckets SET bucket_id=105 WHERE task_id=?`, task.ID)
			if err := store.ToggleTaskDone(ctx, task.ID, true); err != nil {
				t.Fatal(err)
			}
			if got := bucketInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id=?`, task.ID); got != 105 {
				t.Fatalf("completion without Done Bucket moved assignment to %d", got)
			}
			if err := store.ToggleTaskDone(ctx, task.ID, false); err != nil {
				t.Fatal(err)
			}
			if got := bucketInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id=?`, task.ID); got != 104 {
				t.Fatalf("reopening assignment = %d", got)
			}
			if got := bucketInt(t, store, `SELECT COUNT(*) FROM project_views WHERE COALESCE(default_bucket_id,0)=0 AND COALESCE(done_bucket_id,0)=0`); got != 2 {
				t.Fatal("fallback rewrote View configuration")
			}
		})
	}
}

func TestBucketConfigurationNoBucketsAndSelectedView(t *testing.T) {
	store := setupUndoStore(t)
	ctx := context.Background()
	bucketExec(t, store, `UPDATE project_views SET default_bucket_id=0,done_bucket_id=0;
		DELETE FROM buckets WHERE project_view_id=20;
		UPDATE buckets SET position=id;
		INSERT INTO project_views (id,project_id,title,view_kind,bucket_configuration_mode,default_bucket_id,done_bucket_id)
		VALUES (11,1,'Second',3,1,NULL,999);
		INSERT INTO buckets (id,title,project_view_id,position) VALUES (110,'Second first',11,1);
		INSERT INTO task_buckets (task_id,bucket_id,project_view_id) VALUES (1,110,11);
		UPDATE tasks SET done=1 WHERE id=1;
		UPDATE task_buckets SET bucket_id=101 WHERE task_id=1 AND project_view_id=10;`)
	if err := store.ToggleTaskDone(ctx, 1, false, 10); err != nil {
		t.Fatal(err)
	}
	if got := bucketInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id=1 AND project_view_id=10`); got != 100 {
		t.Fatalf("selected View fallback = %d", got)
	}
	if err := store.ToggleTaskDone(ctx, 1, true, 10); err != nil {
		t.Fatal("unselected invalid Done configuration blocked scoped save:", err)
	}
	if got := bucketInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id=1 AND project_view_id=11`); got != 110 {
		t.Fatalf("unselected View assignment = %d", got)
	}
	task, err := store.CreateTask(ctx, TaskDraft{ProjectID: 2, Title: "No buckets"})
	if err != nil {
		t.Fatal(err)
	}
	if got := bucketInt(t, store, `SELECT COUNT(*) FROM task_buckets WHERE task_id=?`, task.ID); got != 0 {
		t.Fatal("creation invented a Bucket without any valid candidates")
	}
	if _, err := store.UpdateTask(ctx, TaskDraft{ID: 1, ProjectID: 2, Title: "No buckets move"}); err != nil {
		t.Fatal(err)
	}
	if err := store.ToggleTaskDone(ctx, 1, false); err != nil {
		t.Fatal(err)
	}
	if got := bucketInt(t, store, `SELECT COUNT(*) FROM task_buckets WHERE task_id=1`); got != 0 {
		t.Fatal("move/reopening invented a Bucket")
	}
	if got := bucketInt(t, store, `SELECT COUNT(*) FROM task_positions WHERE task_id=1 AND project_view_id=20`); got != 1 {
		t.Fatal("missing Buckets prevented position initialization")
	}
}

func TestBucketConfigurationInvalidWritesRollBack(t *testing.T) {
	for _, target := range []int64{200, 999} {
		for _, operation := range []string{"create", "move", "complete", "reopen"} {
			t.Run(fmt.Sprintf("%s/%d", operation, target), func(t *testing.T) {
				store := setupUndoStore(t)
				ctx := context.Background()
				bucketExec(t, store, `UPDATE project_views SET default_bucket_id=?,done_bucket_id=? WHERE id=10`, target, target)
				bucketExec(t, store, `UPDATE tasks SET done=1,done_at='2026-09-01T10:00:00Z' WHERE id=1`)
				if operation == "complete" {
					bucketExec(t, store, `UPDATE tasks SET done=0,done_at=NULL WHERE id=1`)
				}
				before, err := store.TaskUndoState(ctx, 1)
				if err != nil {
					t.Fatal(err)
				}
				switch operation {
				case "create":
					_, err = store.CreateTask(ctx, TaskDraft{ProjectID: 1, Title: "Rejected"})
				case "move":
					bucketExec(t, store, `UPDATE tasks SET project_id=2 WHERE id=1; UPDATE task_buckets SET project_view_id=20,bucket_id=200 WHERE task_id=1; UPDATE task_positions SET project_view_id=20 WHERE task_id=1;`)
					before, err = store.TaskUndoState(ctx, 1)
					if err != nil {
						t.Fatal(err)
					}
					_, err = store.UpdateTask(ctx, TaskDraft{ID: 1, ProjectID: 1, Title: "Rejected"})
				case "complete":
					err = store.ToggleTaskDone(ctx, 1, true)
				case "reopen":
					_, err = store.UpdateTask(ctx, TaskDraft{ID: 1, ProjectID: 1, Title: "Rejected", DoneSet: true, Done: false})
				}
				if err == nil {
					t.Fatal("invalid configured Bucket accepted")
				}
				after, err := store.TaskUndoState(ctx, 1)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("failed save changed state:\nbefore %#v\nafter %#v", before, after)
				}
				if got := bucketInt(t, store, `SELECT COUNT(*) FROM tasks`); got != 1 {
					t.Fatal("failed create leaked Task")
				}
			})
		}
	}
}

func TestBucketConfigurationCombinedSaveAndLateFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("failure=%t", fail), func(t *testing.T) {
			store := setupUndoStore(t)
			ctx := context.Background()
			bucketExec(t, store, `UPDATE project_views SET default_bucket_id=0,done_bucket_id=0;
				UPDATE buckets SET position=id;
				CREATE TABLE users(id INTEGER PRIMARY KEY,name TEXT,username TEXT);
				CREATE TABLE task_comments(id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME);
				INSERT INTO users VALUES(1,'Local','local');
				UPDATE tasks SET done=1,done_at='2026-09-01T10:00:00Z' WHERE id=1;
				UPDATE task_buckets SET bucket_id=101 WHERE task_id=1;`)
			before, err := store.TaskUndoState(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			if fail {
				bucketExec(t, store, `CREATE TRIGGER reject_comment BEFORE INSERT ON task_comments BEGIN SELECT RAISE(ABORT,'late comment failure'); END;`)
			}
			_, _, err = store.SaveTaskWithComment(ctx, TaskDraft{ID: 1, ProjectID: 1, Title: "Reopened", DoneSet: true, Done: false}, TaskCommentDraft{Text: "Reopened"})
			if fail {
				if err == nil {
					t.Fatal("injected failure accepted")
				}
				after, readErr := store.TaskUndoState(ctx, 1)
				if readErr != nil || !reflect.DeepEqual(before, after) {
					t.Fatalf("late failure did not restore state: %#v, %v", after, readErr)
				}
				if got := bucketInt(t, store, `SELECT COUNT(*) FROM task_comments`); got != 0 {
					t.Fatal("failed save leaked Comment")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if got := bucketInt(t, store, `SELECT bucket_id FROM task_buckets WHERE task_id=1`); got != 100 {
					t.Fatalf("combined reopening default = %d", got)
				}
				if got := bucketInt(t, store, `SELECT done FROM tasks WHERE id=1`); got != 0 {
					t.Fatal("combined save did not reopen Task")
				}
			}
		})
	}
}
