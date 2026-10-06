package vimkunja

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// This schema was extracted from the real VOA 2.7.0 baseline, not designed
// around our SQL. The fixture data remains synthetic and disposable.
func setupVikunja270(t *testing.T) *SQLiteStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vikunja.db")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	schema, err := os.ReadFile("testdata/vikunja-2.7.0-schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	compatExec(t, s, string(schema))
	ids, err := os.ReadFile("testdata/vikunja-2.7.0-migrations.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range strings.Fields(string(ids)) {
		compatExec(t, s, `INSERT INTO migration(id,description) VALUES(?,'')`, id)
	}
	compatExec(t, s, `
		INSERT INTO users(id,username,created,updated) VALUES(1,'test',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
		INSERT INTO projects(id,title,identifier,owner_id,created,updated) VALUES
			(1,'One','ONE',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),(2,'Two','TWO',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
		INSERT INTO project_ancestors(ancestor_id,project_id,depth) VALUES(1,1,0),(2,2,0);
		INSERT INTO project_views(id,project_id,title,view_kind,bucket_configuration_mode,default_bucket_id,done_bucket_id,created,updated) VALUES
			(10,1,'List',0,0,0,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
			(11,1,'Board',3,1,100,101,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
			(20,2,'List',0,0,0,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
			(21,2,'Board',3,1,200,201,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
		INSERT INTO buckets(id,title,project_view_id,position,created_by_id,created,updated) VALUES
			(100,'Todo',11,1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),(101,'Done',11,2,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
			(200,'Todo',21,1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),(201,'Done',21,2,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
		INSERT INTO tasks(id,project_id,title,description,done,priority,"index",uid,created_by_id,created,updated) VALUES
			(1,1,'First','',0,0,1,'first',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
			(2,1,'Second','',0,0,2,'second',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
			(3,2,'Third','',0,0,1,'third',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
		INSERT INTO project_task_counters(project_id,last_index) VALUES(1,2),(2,1);
		INSERT INTO task_buckets(task_id,project_view_id,bucket_id) VALUES(1,11,100),(2,11,100),(3,21,200);
		INSERT INTO task_positions(task_id,project_view_id,position) VALUES(1,10,1),(1,11,1),(2,10,2),(2,11,2),(3,20,1),(3,21,1);
		INSERT INTO labels(id,title,created_by_id,created,updated) VALUES(1,'Label',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
		INSERT INTO label_tasks(task_id,label_id,created) VALUES(1,1,CURRENT_TIMESTAMP);
		INSERT INTO task_comments(id,task_id,author_id,comment,created,updated) VALUES(1,1,1,'<p>Comment</p>',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
	`)
	return s
}

func compatExec(t *testing.T, s *SQLiteStore, query string, args ...any) {
	t.Helper()
	if _, err := s.db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func compatInt(t *testing.T, s *SQLiteStore, query string, args ...any) int64 {
	t.Helper()
	var result int64
	if err := s.db.QueryRow(query, args...).Scan(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestVerifiedSQLiteStoreRejectsUnverifiedRevisions(t *testing.T) {
	for _, test := range []struct{ name, mutate string }{
		{"exact", ""},
		{"older", `DELETE FROM migration WHERE id='20260928140648'`},
		{"legacy default", `DELETE FROM migration WHERE id > '20260226172819' AND id != 'SCHEMA_INIT'`},
		{"newer", `INSERT INTO migration(id) VALUES('20990101000000')`},
		{"duplicate", `INSERT INTO migration(id) VALUES('20260928140648')`},
		{"missing history", `DROP TABLE migration`},
		{"missing column", `ALTER TABLE tasks DROP COLUMN uid`},
		{"missing constraint", `DROP INDEX UQE_tasks_tasks_project_index`},
		{"partial constraint", `DROP INDEX UQE_tasks_tasks_project_index; CREATE UNIQUE INDEX partial_task_index ON tasks(project_id,"index") WHERE deleted_at IS NULL`},
		{"expression constraint", `DROP INDEX UQE_tasks_tasks_project_index; CREATE UNIQUE INDEX expression_task_index ON tasks(project_id,"index",lower(title))`},
		{"missing bucket constraint", `DROP INDEX UQE_task_buckets_task_view`},
		{"composite counter key", `DROP TABLE project_task_counters; CREATE TABLE project_task_counters(project_id INTEGER NOT NULL, last_index INTEGER NOT NULL, PRIMARY KEY(project_id,last_index))`},
		{"missing table", `DROP TABLE time_entries`},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := setupVikunja270(t)
			if test.mutate != "" {
				compatExec(t, s, test.mutate)
			}
			path := sqliteDatabasePath(t, s.db)
			store, err := NewVerifiedSQLiteStore(path)
			if store != nil {
				t.Cleanup(func() { _ = store.Close() })
			}
			if test.mutate == "" {
				if err != nil {
					t.Fatal(err)
				}
				_ = store.Close()
			} else if err == nil || store != nil || !strings.Contains(err.Error(), "only the verified 2.7.0") {
				t.Fatalf("unverified revision accepted: %v, %v", store, err)
			}
			if test.name == "legacy default" && (!strings.Contains(err.Error(), "found 116 migrations, latest 20260226172819") || !strings.Contains(err.Error(), "upgrade an older database copy")) {
				t.Fatalf("missing upgrade diagnosis: %v", err)
			}
			if compatInt(t, s, `SELECT COUNT(*) FROM tasks`) != 3 {
				t.Fatal("schema verification changed business data")
			}
		})
	}
}

func TestVikunja270CounterAliasesAndUndo(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	created, err := s.CreateTask(ctx, TaskDraft{ProjectID: 1, Title: "Created"})
	if err != nil || created.Index != 3 || compatInt(t, s, `SELECT last_index FROM project_task_counters WHERE project_id=1`) != 3 {
		t.Fatalf("create = %#v, %v", created, err)
	}
	before, err := s.TaskUndoState(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpdateTask(ctx, TaskDraft{ID: created.ID, ProjectID: 2, Title: "Moved"})
	if err != nil {
		t.Fatal(err)
	}
	after, err := s.TaskUndoState(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreTaskUndoState(ctx, after, before); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreTaskUndoState(ctx, before, after); err != nil {
		t.Fatal(err)
	}
	for _, address := range [][2]int64{{1, 3}, {2, 2}, {1, 4}} {
		if compatInt(t, s, `SELECT task_id FROM task_index_aliases WHERE project_id=? AND "index"=?`, address[0], address[1]) != created.ID {
			t.Fatalf("retired address %v lost", address)
		}
	}
	if err := s.DeleteTask(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if compatInt(t, s, `SELECT COUNT(*) FROM task_index_aliases WHERE task_id=?`, created.ID) != 0 {
		t.Fatal("deleted Task left aliases")
	}
	next, err := s.CreateTask(ctx, TaskDraft{ProjectID: 2, Title: "After deletion"})
	if err != nil || next.Index != 4 {
		t.Fatalf("reserved numbers reused: %#v, %v", next, err)
	}
}

func TestVikunja270CounterFallbackIncludesTrashAndAliases(t *testing.T) {
	s := setupVikunja270(t)
	compatExec(t, s, `DELETE FROM project_task_counters WHERE project_id=1;
		UPDATE tasks SET "index"=8,deleted_at=CURRENT_TIMESTAMP WHERE id=2;
		INSERT INTO task_index_aliases(project_id,"index",task_id) VALUES(1,12,3)`)
	task, err := s.CreateTask(context.Background(), TaskDraft{ProjectID: 1, Title: "New"})
	if err != nil || task.Index != 13 {
		t.Fatalf("counter fallback = %#v, %v", task, err)
	}
}

func TestVikunja270DeletionPreservesMissingOrStaleCounters(t *testing.T) {
	for _, mutation := range []string{
		`DELETE FROM project_task_counters`,
		`UPDATE project_task_counters SET last_index=0`,
	} {
		t.Run(mutation, func(t *testing.T) {
			s := setupVikunja270(t)
			compatExec(t, s, mutation)
			compatExec(t, s, `INSERT INTO task_index_aliases(project_id,"index",task_id) VALUES(2,20,2)`)
			ctx := context.Background()
			if err := s.DeleteTask(ctx, 2); err != nil {
				t.Fatal(err)
			}
			for projectID, wantIndex := range map[int64]int64{1: 3, 2: 21} {
				task, err := s.CreateTask(ctx, TaskDraft{ProjectID: projectID, Title: "After deletion"})
				if err != nil || task.Index != wantIndex {
					t.Fatalf("Project %d reused a deleted address: index %d, want %d; %v", projectID, task.Index, wantIndex, err)
				}
			}
		})
	}
}

func TestVikunja270FailedDeletionRollsBackCounterRepair(t *testing.T) {
	s := setupVikunja270(t)
	compatExec(t, s, `DELETE FROM project_task_counters;
		INSERT INTO task_index_aliases(project_id,"index",task_id) VALUES(2,20,2);
		CREATE TRIGGER reject_deletion BEFORE DELETE ON tasks BEGIN SELECT RAISE(ABORT,'delete failed'); END`)
	if err := s.DeleteTask(context.Background(), 2); err == nil {
		t.Fatal("expected delete failure")
	}
	for table, count := range map[string]int64{"project_task_counters": 0, "tasks": 3, "task_index_aliases": 1, "task_positions": 6, "task_buckets": 3} {
		if compatInt(t, s, `SELECT COUNT(*) FROM `+table) != count {
			t.Fatalf("failed deletion changed %s", table)
		}
	}
}

func TestVikunja270TrashIsHiddenAndProtected(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	before, err := s.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Retain a stale bucket row as an extra defensive read case.
	compatExec(t, s, `UPDATE tasks SET deleted_at=CURRENT_TIMESTAMP WHERE id=1`)
	for name, read := range map[string]func() ([]Task, error){
		"list":     func() ([]Task, error) { tasks, _, err := s.ListTasksInView(ctx, 1, 10, 1, 100); return tasks, err },
		"overview": func() ([]Task, error) { tasks, _, err := s.ListOverviewTasks(ctx, 1, 100); return tasks, err },
		"label":    func() ([]Task, error) { tasks, _, err := s.ListTasksForLabel(ctx, 1, 1, 100); return tasks, err },
		"search":   func() ([]Task, error) { data, err := s.LoadSearchData(ctx); return data.Tasks, err },
	} {
		tasks, err := read()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, task := range tasks {
			if task.ID == 1 {
				t.Fatalf("%s exposes trash", name)
			}
		}
	}
	board, err := s.LoadManualKanbanBoard(ctx, 1)
	if err != nil || len(board.Buckets) != 2 {
		t.Fatalf("board = %#v, %v", board, err)
	}
	for _, bucket := range board.Buckets {
		for _, task := range bucket.Tasks {
			if task.ID == 1 {
				t.Fatal("Kanban exposes trash")
			}
		}
	}
	labels, err := s.ListLabels(ctx)
	if err != nil || len(labels) != 1 || labels[0].TaskCount != 0 {
		t.Fatalf("Label count includes trash: %#v, %v", labels, err)
	}
	for name, op := range map[string]func() error{
		"title": func() error { return s.UpdateTaskTitle(ctx, 1, "Forbidden") },
		"edit": func() error {
			_, err := s.UpdateTask(ctx, TaskDraft{ID: 1, ProjectID: 1, Title: "Forbidden"})
			return err
		},
		"labels": func() error { return s.ReplaceTaskLabels(ctx, 1, nil) },
		"done":   func() error { return s.ToggleTaskDone(ctx, 1, true) },
		"bucket": func() error { return s.SetTaskWorkflowStatus(ctx, 1, 101) },
		"delete": func() error { return s.DeleteTask(ctx, 1) },
		"comment save": func() error {
			_, _, err := s.SaveTaskWithComment(ctx, TaskDraft{ID: 1, ProjectID: 1, Title: "Forbidden"}, TaskCommentDraft{Text: "Forbidden"})
			return err
		},
		"comment delete": func() error { return s.DeleteTaskComment(ctx, 1, 1) },
		"comments read":  func() error { _, err := s.ListTaskComments(ctx, 1); return err },
		"labels read":    func() error { _, err := s.ListTaskLabels(ctx, 1); return err },
		"workflow read":  func() error { _, err := s.GetTaskWorkflowStatus(ctx, 1, 1); return err },
		"undo read":      func() error { _, err := s.TaskUndoState(ctx, 1); return err },
		"undo":           func() error { return s.RestoreTaskUndoState(ctx, before, before) },
	} {
		if err := op(); err == nil {
			t.Errorf("%s allowed on trash", name)
		}
	}
	if compatInt(t, s, `SELECT COUNT(*) FROM task_comments`) != 1 || compatInt(t, s, `SELECT COUNT(*) FROM label_tasks`) != 1 {
		t.Fatal("rejected operations changed related data")
	}
}

func TestVikunja270ArchivedWritesAreRejected(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	before, err := s.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	compatExec(t, s, `UPDATE projects SET is_archived=1 WHERE id=1`)
	for name, op := range map[string]func() error{
		"create": func() error { _, err := s.CreateTask(ctx, TaskDraft{ProjectID: 1, Title: "Forbidden"}); return err },
		"title":  func() error { return s.UpdateTaskTitle(ctx, 1, "Forbidden") },
		"move source": func() error {
			_, err := s.UpdateTask(ctx, TaskDraft{ID: 1, ProjectID: 2, Title: "Forbidden"})
			return err
		},
		"move target": func() error {
			_, err := s.UpdateTask(ctx, TaskDraft{ID: 3, ProjectID: 1, Title: "Forbidden"})
			return err
		},
		"labels":    func() error { return s.ReplaceTaskLabels(ctx, 1, nil) },
		"done":      func() error { return s.ToggleTaskDone(ctx, 1, true) },
		"bucket":    func() error { return s.SetTaskWorkflowStatus(ctx, 1, 101) },
		"delete":    func() error { return s.DeleteTask(ctx, 1) },
		"comment":   func() error { return s.DeleteTaskComment(ctx, 1, 1) },
		"provision": func() error { _, err := s.ProvisionStandardManualKanbanView(ctx, 1); return err },
		"undo":      func() error { return s.RestoreTaskUndoState(ctx, before, before) },
	} {
		if err := op(); err == nil || !strings.Contains(err.Error(), "archived") {
			t.Errorf("%s = %v, want archive rejection", name, err)
		}
	}
	after, err := s.TaskUndoState(ctx, 1)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("archive rejection changed Task: %#v, %v", after, err)
	}
}

func TestVikunja270BucketCapacityAndRollback(t *testing.T) {
	ctx := context.Background()
	for _, action := range []string{"create", "move", "done", "workflow", "undo", "position failure"} {
		t.Run(action, func(t *testing.T) {
			s := setupVikunja270(t)
			before, err := s.TaskUndoState(ctx, 3)
			if err != nil {
				t.Fatal(err)
			}
			if action == "position failure" {
				compatExec(t, s, `CREATE TRIGGER reject_position BEFORE INSERT ON task_positions BEGIN SELECT RAISE(ABORT,'position failed'); END`)
			} else {
				compatExec(t, s, `UPDATE buckets SET "limit"=1 WHERE id=100;
					UPDATE task_buckets SET bucket_id=101 WHERE task_id=2;
					UPDATE buckets SET "limit"=1 WHERE id=101`)
			}
			switch action {
			case "create", "position failure":
				_, err = s.CreateTask(ctx, TaskDraft{ProjectID: 1, Title: "Rejected"})
			case "move":
				_, err = s.UpdateTask(ctx, TaskDraft{ID: 3, ProjectID: 1, Title: "Rejected"})
			case "done":
				err = s.ToggleTaskDone(ctx, 1, true)
			case "workflow":
				err = s.SetTaskWorkflowStatus(ctx, 1, 101)
			case "undo":
				target := before
				target.ProjectID = 1
				target.BucketAssignments = []TaskBucketAssignment{{ViewID: 11, BucketID: 100}}
				err = s.RestoreTaskUndoState(ctx, before, target)
			}
			if err == nil {
				t.Fatal("expected atomic failure")
			}
			after, readErr := s.TaskUndoState(ctx, 3)
			if readErr != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("failed action changed unrelated Task: %#v, %v", after, readErr)
			}
			if compatInt(t, s, `SELECT COUNT(*) FROM tasks`) != 3 || compatInt(t, s, `SELECT last_index FROM project_task_counters WHERE project_id=1`) != 2 || compatInt(t, s, `SELECT COUNT(*) FROM task_index_aliases`) != 0 {
				t.Fatal("failure left Task/counter/alias writes")
			}
			if compatInt(t, s, `SELECT COUNT(*) FROM task_positions`) != 6 {
				t.Fatal("failure changed positions")
			}
		})
	}
	s := setupVikunja270(t)
	compatExec(t, s, `UPDATE buckets SET "limit"=2 WHERE id=100`)
	if err := s.SetTaskWorkflowStatus(ctx, 1, 100); err != nil {
		t.Fatalf("unchanged full-Bucket assignment rejected: %v", err)
	}
}

func TestVikunja270TimeEntriesPreventPermanentDeletion(t *testing.T) {
	s := setupVikunja270(t)
	compatExec(t, s, `INSERT INTO time_entries(user_id,task_id,start_time,created,updated) VALUES(1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	if err := s.DeleteTask(context.Background(), 1); err == nil || !strings.Contains(err.Error(), "Time Entries") {
		t.Fatalf("time-tracked deletion = %v", err)
	}
	for table, count := range map[string]int64{"tasks": 3, "task_comments": 1, "time_entries": 1, "label_tasks": 1} {
		if compatInt(t, s, fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table)) != count {
			t.Fatalf("rejected delete changed %s", table)
		}
	}
}

func TestVikunja270SiblingSortingPreservesHierarchy(t *testing.T) {
	s := setupVikunja270(t)
	compatExec(t, s, `UPDATE projects SET position=id*65536;
 INSERT INTO projects(id,title,owner_id,parent_project_id,position,created,updated) VALUES
 (3,'Child',1,1,65536,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),(4,'Sibling',1,1,131072,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
 INSERT INTO project_ancestors(ancestor_id,project_id,depth) VALUES (3,3,0),(1,3,1),(4,4,0),(1,4,1)`)
	snapshot := func() string {
		var result string
		err := s.db.QueryRow(`SELECT group_concat(row,';') FROM (SELECT ancestor_id||','||project_id||','||depth AS row FROM project_ancestors ORDER BY ancestor_id,project_id)`).Scan(&result)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	before := snapshot()
	ctx := context.Background()
	if err := s.MoveProjectAmongSiblings(ctx, 1, ProjectMoveNext); err != nil {
		t.Fatal(err)
	}
	if err := s.MoveProjectAmongSiblings(ctx, 3, ProjectMoveNext); err != nil {
		t.Fatal(err)
	}
	if snapshot() != before || compatInt(t, s, `SELECT COUNT(*) FROM projects WHERE parent_project_id IS NULL`) != 2 || compatInt(t, s, `SELECT COUNT(*) FROM projects WHERE id IN (3,4) AND parent_project_id=1`) != 2 {
		t.Fatal("sibling sorting changed hierarchy")
	}
	if compatInt(t, s, `SELECT position FROM projects WHERE id=1`) != 131072 || compatInt(t, s, `SELECT position FROM projects WHERE id=2`) != 65536 || compatInt(t, s, `SELECT position FROM projects WHERE id=3`) != 131072 {
		t.Fatal("sibling order was not updated")
	}
}
