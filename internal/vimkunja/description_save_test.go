package vimkunja

import (
	"context"
	"errors"
	"fmt"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"reflect"
	"strings"
	"testing"
)

func TestDescriptionSaveRejectsExternalChangeAfterPreparation(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	const original = `<p>Original</p>`
	bucketExec(t, store, `INSERT INTO projects(id,title) VALUES(1,'Project'); INSERT INTO tasks(id,project_id,title,description,"index") VALUES(1,1,'Before','<p>Original</p>',1)`)
	doc := description.Parse(original)
	nodes := doc.Snapshot()
	nodes[0].Children[0].Text = "Local"
	candidate, err := doc.PrepareSave(nodes, description.Preservation{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var path string
	if err := store.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	remote, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()
	if _, err := remote.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Remote", Description: "<p>Remote</p>"}); err != nil {
		t.Fatal(err)
	}
	draft := TaskDraft{ID: 1, ProjectID: 1, Title: "Local", DescriptionCandidate: &candidate}
	_, err = store.UpdateTask(context.Background(), draft)
	var conflict *DescriptionConflictError
	if !errors.As(err, &conflict) || conflict.Current != "<p>Remote</p>" || conflict.Baseline != original {
		t.Fatalf("expected reloadable conflict: %v", err)
	}
	tasks, _, err := store.ListOverviewTasks(context.Background(), 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks: %#v", tasks)
	}
	persisted := tasks[0]
	if persisted.Title != "Remote" || persisted.Description != "<p>Remote</p>" || nodes[0].Children[0].Text != "Local" {
		t.Fatalf("conflict lost DB or draft: %#v", persisted)
	}
}

func TestNativeDescriptionSaveCommitAndRollback(t *testing.T) {
	for _, mode := range []string{"update", "create", "comment", "workflow-failure", "comment-sql-failure", "conflict", "missing", "missing-comment", "invalid-comment", "comment-conflict", "invalid-candidate"} {
		t.Run(mode, func(t *testing.T) {
			store, cleanup := setupTestDB(t)
			defer cleanup()
			const source = `<ul data-type="taskList"><li data-type="taskItem" data-task-id="A" data-checked="true"><p>Alpha</p></li><li data-type="taskItem" data-checked="false"><p>Beta</p><ul data-type="taskList"><li data-type="taskItem" data-task-id="child" data-checked="true"><p>Child</p></li></ul></li></ul><custom X='keep'>Keep</custom>`
			bucketExec(t, store, `INSERT INTO projects(id,title) VALUES(1,'Project'); CREATE TABLE users(id INTEGER PRIMARY KEY,name TEXT,username TEXT); INSERT INTO users VALUES(1,'Local','local'); CREATE TABLE task_comments(id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME)`)
			if _, err := store.db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,"index") VALUES(1,1,'Before',?,1,1)`, source); err != nil {
				t.Fatal(err)
			}
			doc := description.Parse(source)
			nodes := doc.Snapshot()
			before := doc.Snapshot()
			candidate, err := doc.PrepareSave(nodes, description.Preservation{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			draft := TaskDraft{ID: 1, ProjectID: 1, Title: "After", DescriptionCandidate: &candidate}
			switch mode {
			case "create":
				draft.ID = 0
			case "workflow-failure":
				draft.WorkflowBucketID = 999
			case "comment-sql-failure":
				bucketExec(t, store, `CREATE TRIGGER reject_comment BEFORE INSERT ON task_comments BEGIN SELECT RAISE(ABORT,'failure'); END`)
			case "conflict", "comment-conflict":
				bucketExec(t, store, `UPDATE tasks SET description='<p>Remote</p>' WHERE id=1`)
			case "missing", "missing-comment":
				bucketExec(t, store, `DELETE FROM tasks WHERE id=1`)
			case "invalid-candidate":
				candidate = description.SaveCandidate{}
			}
			var task Task
			if mode == "create" {
				task, err = store.CreateTask(context.Background(), draft)
			} else if mode == "comment" || mode == "comment-sql-failure" || mode == "missing-comment" || mode == "invalid-comment" || mode == "comment-conflict" {
				commentText := "Comment"
				if mode == "invalid-comment" {
					commentText = ""
				}
				task, _, err = store.SaveTaskWithComment(context.Background(), draft, TaskCommentDraft{Text: commentText})
			} else {
				task, err = store.UpdateTask(context.Background(), draft)
			}
			success := mode == "update" || mode == "create" || mode == "comment"
			if (err == nil) != success {
				t.Fatalf("save result: %v", err)
			}
			if !reflect.DeepEqual(nodes, before) || doc.SourceHTML() != source {
				t.Fatal("draft mutated by save")
			}
			tasks, _, readErr := store.ListOverviewTasks(context.Background(), 1, 20)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if success {
				if task.Description != candidate.HTML() || (mode != "create" && !task.Done) {
					t.Fatalf("committed result: %#v", task)
				}
				found := false
				for _, persisted := range tasks {
					if persisted.ID == task.ID {
						found = true
						if persisted.Description != candidate.HTML() {
							t.Fatal("DB differs from committed candidate")
						}
					}
				}
				if !found {
					t.Fatal("saved Task missing")
				}
				savedDoc := description.Parse(task.Description)
				stable, err := savedDoc.PrepareSave(savedDoc.Snapshot(), description.Preservation{}, nil)
				if err != nil || stable.HTML() != task.Description {
					t.Fatal("IDs unstable")
				}
				draft.ID = task.ID
				draft.DescriptionCandidate = &stable
				again, err := store.UpdateTask(context.Background(), draft)
				if err != nil || again.Description != task.Description {
					t.Fatalf("second save: %v", err)
				}
			} else if mode != "missing" && mode != "missing-comment" {
				want := source
				if mode == "conflict" || mode == "comment-conflict" {
					want = "<p>Remote</p>"
				}
				if len(tasks) != 1 || tasks[0].Title != "Before" || tasks[0].Description != want || !tasks[0].Done {
					t.Fatalf("partial write: %#v", tasks)
				}
			}
			// Query persisted rows directly: the public reader intentionally rejects
			// missing Tasks, but rollback must also leave no orphaned Comment.
			var comments int
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM task_comments WHERE task_id = 1`).Scan(&comments); err != nil {
				t.Fatal(err)
			}
			wantComments := 0
			if mode == "comment" {
				wantComments = 1
			}
			if comments != wantComments {
				t.Fatalf("partial Comment write: %#v", comments)
			}
		})
	}
}

func TestDescriptionBaselinePreservesUntouchedHTML(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	const source = "<p>Original</p><custom X='keep'>\r\n猫\t</custom><!-- comment -->"
	bucketExec(t, store, `INSERT INTO projects(id,title) VALUES(1,'Project')`)
	if _, err := store.db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,"index") VALUES(1,1,'Before',?,1,1)`, source); err != nil {
		t.Fatal(err)
	}
	baseline := source
	task, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Renamed", Description: source, DescriptionBaseline: &baseline})
	if err != nil || task.Description != source || !task.Done {
		t.Fatalf("Task-only save lost HTML/Done: %#v %v", task, err)
	}
	baseline = ""
	_, err = store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Wrong", Description: "", DescriptionBaseline: &baseline})
	var conflict *DescriptionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("empty baseline was omitted: %v", err)
	}
}

func TestConcurrentDescriptionSavesCannotBothOverwriteOpenedBaseline(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	bucketExec(t, store, `INSERT INTO projects(id,title) VALUES(1,'Project');INSERT INTO tasks(id,project_id,title,description,"index") VALUES(1,1,'Before','<p>Original</p>',1)`)
	var path string
	if err := store.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	remote, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()
	start := make(chan struct{})
	results := make(chan error, 2)
	for i, s := range []*SQLiteStore{store, remote} {
		go func(i int, s *SQLiteStore) {
			baseline := "<p>Original</p>"
			<-start
			_, err := s.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: fmt.Sprint(i), Description: fmt.Sprintf("<p>Writer %d</p>", i), DescriptionBaseline: &baseline})
			results <- err
		}(i, s)
	}
	close(start)
	success := 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			success++
			continue
		}
		var conflict *DescriptionConflictError
		if !errors.As(err, &conflict) && !strings.Contains(err.Error(), "SQLITE_BUSY") {
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}
	// SQLite may reject both deferred read-to-write upgrades as busy. Such
	// failures must preserve the baseline rather than silently overwrite it.
	if success > 1 {
		t.Fatalf("both writers overwrote the same baseline")
	}

	tasks, _, err := store.ListOverviewTasks(context.Background(), 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	want := "<p>Original</p>"
	if success == 1 {
		want = "<p>Writer " + tasks[0].Title + "</p>"
	}
	if len(tasks) != 1 || tasks[0].Description != want {
		t.Fatalf("partial concurrent save: %#v", tasks)
	}
}

func TestDescriptionCommitFailureRollsBackAndReleasesConnection(t *testing.T) {
	for _, mode := range []string{"update", "create", "comment"} {
		t.Run(mode, func(t *testing.T) {
			store, cleanup := setupTestDB(t)
			defer cleanup()
			bucketExec(t, store, `INSERT INTO projects(id,title) VALUES(1,'Project');INSERT INTO tasks(id,project_id,title,description,"index") VALUES(1,1,'Before','<p>Original</p>',1);CREATE TABLE users(id INTEGER PRIMARY KEY,name TEXT,username TEXT);INSERT INTO users VALUES(1,'Local','local');CREATE TABLE task_comments(id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME)`)
			doc := description.Parse("<p>Original</p>")
			nodes := doc.Snapshot()
			nodes[0].Children[0].Text = "Local"
			candidate, err := doc.PrepareSave(nodes, description.Preservation{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			draft := TaskDraft{ID: 1, ProjectID: 1, Title: "After", DescriptionCandidate: &candidate}
			if mode == "create" {
				draft.ID = 0
			}
			var path string
			if err := store.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
				t.Fatal(err)
			}
			reader, err := NewSQLiteStore(path)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			readTx, err := reader.db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer readTx.Rollback()
			var old string
			if err := readTx.QueryRow(`SELECT description FROM tasks WHERE id=1`).Scan(&old); err != nil {
				t.Fatal(err)
			}
			save := func() error {
				switch mode {
				case "create":
					_, err := store.CreateTask(context.Background(), draft)
					return err
				case "comment":
					_, _, err := store.SaveTaskWithComment(context.Background(), draft, TaskCommentDraft{Text: "Comment"})
					return err
				default:
					_, err := store.UpdateTask(context.Background(), draft)
					return err
				}
			}
			err = save()
			if err == nil || !strings.Contains(err.Error(), "commit") || !strings.Contains(err.Error(), "SQLITE_BUSY") {
				t.Fatalf("expected COMMIT busy: %v", err)
			}
			tasks, _, err := store.ListOverviewTasks(context.Background(), 1, 20)
			if err != nil {
				t.Fatalf("connection not cleaned: %v", err)
			}
			if len(tasks) != 1 || tasks[0].Title != "Before" || tasks[0].Description != old {
				t.Fatalf("failed commit retained write: %#v", tasks)
			}
			comments, err := store.ListTaskComments(context.Background(), 1)
			if err != nil || len(comments) != 0 {
				t.Fatalf("failed commit retained Comment: %#v %v", comments, err)
			}
			if doc.SourceHTML() != old || nodes[0].Children[0].Text != "Local" {
				t.Fatal("failed commit lost draft")
			}
			if err := readTx.Rollback(); err != nil {
				t.Fatal(err)
			}
			if err := save(); err != nil {
				t.Fatalf("retry after busy: %v", err)
			}
		})
	}
}
