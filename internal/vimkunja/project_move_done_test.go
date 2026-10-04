package vimkunja

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func moveExec(t *testing.T, store *SQLiteStore, query string, args ...any) {
	t.Helper()
	if _, err := store.db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
func TestIntegrityAuditCompletedProjectMove(t *testing.T) {
	store := setupUndoStore(t)
	moveExec(t, store, `UPDATE tasks SET done = 1, done_at = '2026-09-01 10:00:00' WHERE id = 1;
		INSERT INTO buckets (id, title, project_view_id) VALUES (201, 'Done Two', 20);
		UPDATE project_views SET done_bucket_id = 201 WHERE id = 20;
		UPDATE buckets SET position = id WHERE project_view_id = 20;`)
	task, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 2, Title: "Completed"})
	if err != nil {
		t.Fatal(err)
	}
	if !task.Done {
		t.Fatal("Project move cleared Done")
	}
	var bucket int64
	if err := store.db.QueryRow(`SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = 20`).Scan(&bucket); err != nil {
		t.Fatal(err)
	}
	if bucket != 201 {
		t.Errorf("completed Task moved into Bucket %d, want target Done Bucket 201", bucket)
	}
	board, err := store.LoadManualKanbanBoard(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, column := range board.Buckets {
		for _, card := range column.Tasks {
			if card.ID == task.ID {
				found = true
				if column.ID != 201 || !card.Done {
					t.Fatalf("moved Task appears in wrong board column: %#v %#v", column, card)
				}
			}
		}
	}
	if !found {
		t.Fatal("moved completed Task missing from destination board")
	}

}

func TestProjectMoveInitializesEachViewFromPersistedDone(t *testing.T) {
	for _, done := range []bool{false, true} {
		for _, withComment := range []bool{false, true} {
			t.Run(fmt.Sprintf("done=%t/comment=%t", done, withComment), func(t *testing.T) {
				store := setupUndoStore(t)
				moveExec(t, store, `INSERT INTO buckets(id,title,project_view_id) VALUES(201,'Done Two',20),(210,'Todo Three',21),(211,'Done Three',21),(220,'Zero fallback',22),(230,'NULL fallback',23),(251,'Done only',25);
                    UPDATE project_views SET done_bucket_id=201 WHERE id=20;
                    INSERT INTO project_views(id,project_id,title,view_kind,bucket_configuration_mode,default_bucket_id,done_bucket_id) VALUES
                    (21,2,'Second board',3,1,210,211),(22,2,'Zero Done',3,1,220,0),(23,2,'NULL Done',3,1,230,NULL),(24,2,'List',1,1,NULL,NULL),(25,2,'Done only',3,1,NULL,251);
                    CREATE TABLE users(id INTEGER PRIMARY KEY,name TEXT,username TEXT);
                    CREATE TABLE task_comments(id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME);
                    INSERT INTO users VALUES(1,'Local','local');
                    INSERT INTO tasks(id,project_id,title,done,"index") VALUES(2,2,'Unrelated',0,1);
                    INSERT INTO task_buckets(task_id,bucket_id,project_view_id) VALUES(2,200,20);
                    INSERT INTO task_positions(task_id,project_view_id,position) VALUES(2,20,7);`)
				var doneAt any
				if done {
					doneAt = "2026-09-01T10:00:00.123456789+02:00"
				}
				moveExec(t, store, `UPDATE tasks SET done=?,done_at=? WHERE id=1`, done, doneAt)
				draft := TaskDraft{ID: 1, ProjectID: 2, Title: "Moved", Description: "old", Priority: 2, DueDate: "2026-09-01", Done: !done}
				var task Task
				var err error
				if withComment {
					task, _, err = store.SaveTaskWithComment(context.Background(), draft, TaskCommentDraft{Text: "Moved"})
				} else {
					task, err = store.UpdateTask(context.Background(), draft)
				}
				if err != nil || task.Done != done || task.ProjectID != 2 {
					t.Fatalf("move result: %#v %v", task, err)
				}
				state, err := store.TaskUndoState(context.Background(), 1)
				if err != nil {
					t.Fatal(err)
				}
				want := []TaskBucketAssignment{{ViewID: 20, BucketID: 200}, {ViewID: 21, BucketID: 210}, {ViewID: 22, BucketID: 220}, {ViewID: 23, BucketID: 230}}
				if done {
					want[0].BucketID = 201
					want[1].BucketID = 211
				}
				// An unconfigured default falls back to the first Bucket, including a Done-only View.
				want = append(want, TaskBucketAssignment{ViewID: 25, BucketID: 251})
				if !reflect.DeepEqual(state.BucketAssignments, want) {
					t.Fatalf("destination assignments = %#v, want %#v", state.BucketAssignments, want)
				}
				var gotTime sql.NullString
				if err := store.db.QueryRow(`SELECT CAST(done_at AS TEXT) FROM tasks WHERE id=1`).Scan(&gotTime); err != nil {
					t.Fatal(err)
				}
				if gotTime.Valid != done || (done && gotTime.String != doneAt) {
					t.Fatalf("move changed completion metadata: %v", gotTime)
				}
				var count int
				if err := store.db.QueryRow(`SELECT COUNT(*) FROM task_positions WHERE task_id=1`).Scan(&count); err != nil || count != 6 {
					t.Fatalf("destination positions: %d %v", count, err)
				}
				var position float64
				if err := store.db.QueryRow(`SELECT position FROM task_positions WHERE task_id=2 AND project_view_id=20`).Scan(&position); err != nil || position != 7 {
					t.Fatalf("unrelated position changed: %v %v", position, err)
				}
				var otherBucket int64
				if err := store.db.QueryRow(`SELECT bucket_id FROM task_buckets WHERE task_id=2`).Scan(&otherBucket); err != nil || otherBucket != 200 {
					t.Fatalf("unrelated Bucket changed: %d %v", otherBucket, err)
				}
			})
		}
	}
}

func TestProjectMoveInvalidDoneBucketRollsBack(t *testing.T) {
	for _, bucket := range []int64{999, 101} {
		t.Run(fmt.Sprint(bucket), func(t *testing.T) {
			store := setupUndoStore(t)
			moveExec(t, store, `UPDATE tasks SET done=1,done_at='2026-09-01 10:00:00' WHERE id=1`)
			moveExec(t, store, `UPDATE project_views SET done_bucket_id=? WHERE id=20`, bucket)
			before, err := store.TaskUndoState(context.Background(), 1)
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 2, Title: "Must rollback"})
			if err == nil || !strings.Contains(err.Error(), "does not belong") {
				t.Fatalf("invalid destination Done Bucket: %v", err)
			}
			assertProjectMoveRolledBack(t, store, before)
		})
	}
}

func assertProjectMoveRolledBack(t *testing.T, store *SQLiteStore, before TaskUndoState) {
	t.Helper()
	after, err := store.TaskUndoState(context.Background(), 1)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("failed move changed Task: %#v %v", after, err)
	}
	var positions int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM task_positions WHERE task_id=1`).Scan(&positions); err != nil || positions != 1 {
		t.Fatalf("failed move left destination positions: %d %v", positions, err)
	}
	var viewID int64
	var position float64
	if err := store.db.QueryRow(`SELECT project_view_id,position FROM task_positions WHERE task_id=1`).Scan(&viewID, &position); err != nil || viewID != 10 || position != 1 {
		t.Fatalf("failed move changed positions: %d %v %v", viewID, position, err)
	}
	var gotTime string
	if err := store.db.QueryRow(`SELECT CAST(done_at AS TEXT) FROM tasks WHERE id=1`).Scan(&gotTime); err != nil || gotTime != "2026-09-01 10:00:00" {
		t.Fatalf("failed move changed completion time: %q %v", gotTime, err)
	}
}

func TestProjectMoveLateFailuresRollBack(t *testing.T) {
	for _, withComment := range []bool{false, true} {
		t.Run(fmt.Sprint(withComment), func(t *testing.T) {
			store := setupUndoStore(t)
			moveExec(t, store, `INSERT INTO buckets(id,title,project_view_id) VALUES(201,'Done Two',20);
                UPDATE project_views SET done_bucket_id=201 WHERE id=20;
                UPDATE tasks SET done=1,done_at='2026-09-01 10:00:00' WHERE id=1;
                CREATE TABLE users(id INTEGER PRIMARY KEY,name TEXT,username TEXT);
                CREATE TABLE task_comments(id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME);
                INSERT INTO users VALUES(1,'Local','local');`)
			if withComment {
				moveExec(t, store, `CREATE TRIGGER reject_comment BEFORE INSERT ON task_comments BEGIN SELECT RAISE(ABORT,'injected late failure'); END;`)
			} else {
				moveExec(t, store, `CREATE TRIGGER reject_assignment BEFORE INSERT ON task_buckets WHEN NEW.project_view_id=20 BEGIN SELECT RAISE(ABORT,'injected late failure'); END;`)
			}
			before, err := store.TaskUndoState(context.Background(), 1)
			if err != nil {
				t.Fatal(err)
			}
			draft := TaskDraft{ID: 1, ProjectID: 2, Title: "Must rollback"}
			if withComment {
				_, _, err = store.SaveTaskWithComment(context.Background(), draft, TaskCommentDraft{Text: "Rejected"})
			} else {
				_, err = store.UpdateTask(context.Background(), draft)
			}
			if err == nil || !strings.Contains(err.Error(), "injected late failure") {
				t.Fatalf("expected late failure: %v", err)
			}
			assertProjectMoveRolledBack(t, store, before)
		})
	}
}

func TestProjectMoveExplicitDoneChangeOverridesInitialAssignment(t *testing.T) {
	for _, done := range []bool{false, true} {
		t.Run(fmt.Sprint(done), func(t *testing.T) {
			store := setupUndoStore(t)
			moveExec(t, store, `INSERT INTO buckets(id,title,project_view_id) VALUES(201,'Done Two',20); UPDATE project_views SET done_bucket_id=201 WHERE id=20`)
			moveExec(t, store, `UPDATE tasks SET done=? WHERE id=1`, !done)
			task, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 2, Title: "Moved", Done: done, DoneSet: true})
			if err != nil || task.Done != done {
				t.Fatalf("explicit Done: %#v %v", task, err)
			}
			want := int64(200)
			if done {
				want = 201
			}
			var got int64
			if err := store.db.QueryRow(`SELECT bucket_id FROM task_buckets WHERE task_id=1 AND project_view_id=20`).Scan(&got); err != nil || got != want {
				t.Fatalf("explicit Done assignment: %d %v", got, err)
			}
		})
	}
}
