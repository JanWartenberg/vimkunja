package vimkunja

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestIntegrityAuditRecurringCompletion(t *testing.T) {
	for _, config := range []struct{ after, mode int }{{86400, 0}, {0, 1}, {86400, 2}, {0, 9}} {
		for _, path := range []string{"toggle", "workflow", "draft", "comment", "undo"} {
			t.Run(fmt.Sprintf("%v/%s", config, path), func(t *testing.T) {
				store := setupUndoStore(t)
				ctx := context.Background()
				bucketExec(t, store, `ALTER TABLE tasks ADD COLUMN repeat_after INTEGER; ALTER TABLE tasks ADD COLUMN repeat_mode INTEGER; UPDATE project_views SET done_bucket_id=101 WHERE id=10;`)
				bucketExec(t, store, `UPDATE tasks SET repeat_after=?,repeat_mode=? WHERE id=1`, config.after, config.mode)
				before, err := store.TaskUndoState(ctx, 1)
				if err != nil {
					t.Fatal(err)
				}
				switch path {
				case "toggle":
					err = store.ToggleTaskDone(ctx, 1, true)
				case "workflow":
					err = store.SetTaskWorkflowStatus(ctx, 1, 101)
				case "draft":
					_, err = store.UpdateTask(ctx, TaskDraft{ID: 1, ProjectID: 1, Title: "Rejected", Done: true, DoneSet: true})
				case "comment":
					bucketExec(t, store, `CREATE TABLE users(id INTEGER PRIMARY KEY,name TEXT,username TEXT); CREATE TABLE task_comments(id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME); INSERT INTO users VALUES(1,'Local','local');`)
					_, _, err = store.SaveTaskWithComment(ctx, TaskDraft{ID: 1, ProjectID: 1, Title: "Rejected", Done: true, DoneSet: true}, TaskCommentDraft{Text: "Must not persist"})
					if bucketInt(t, store, `SELECT COUNT(*) FROM task_comments`) != 0 {
						t.Fatal("rejected save leaked Comment")
					}
				case "undo":
					target := before
					target.Done = true
					err = store.RestoreTaskUndoState(ctx, before, target)
				}
				if err == nil || !strings.Contains(err.Error(), "recurring Task") {
					t.Fatalf("unsafe completion: %v", err)
				}
				after, err := store.TaskUndoState(ctx, 1)
				if err != nil || !reflect.DeepEqual(before, after) {
					t.Fatalf("rejected completion changed state: %#v %v", after, err)
				}
				if _, err := store.UpdateTask(ctx, TaskDraft{ID: 1, ProjectID: 1, Title: "Allowed", DueDate: before.DueDate}); err != nil {
					t.Fatal(err)
				}
				bucketExec(t, store, `UPDATE tasks SET done=1 WHERE id=1`)
				if err := store.ToggleTaskDone(ctx, 1, false); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestRecurringHistoricalColumnBoundaries(t *testing.T) {
	for _, column := range []string{"repeat_after", "repeat_mode"} {
		t.Run(column, func(t *testing.T) {
			store := setupUndoStore(t)
			bucketExec(t, store, "ALTER TABLE tasks ADD COLUMN "+column+" INTEGER")
			bucketExec(t, store, "UPDATE tasks SET "+column+"=1 WHERE id=1")
			if err := store.ToggleTaskDone(context.Background(), 1, true); err == nil {
				t.Fatal("partial recurrence schema bypassed guard")
			}
			bucketExec(t, store, "UPDATE tasks SET "+column+"=NULL WHERE id=1")
			if err := store.ToggleTaskDone(context.Background(), 1, true); err != nil {
				t.Fatal(err)
			}
		})
	}
}
