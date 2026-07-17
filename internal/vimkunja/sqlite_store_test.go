package vimkunja

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"
	"time"
)

// setupTestDB creates a temporary SQLite DB with minimal schema + test data
func setupTestDB(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "vimkunja-test-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	// Minimal schema (projects, tasks, project_views, task_buckets, task_positions)
	schema := `
		CREATE TABLE projects (
			id INTEGER PRIMARY KEY,
			identifier TEXT,
			title TEXT,
			description TEXT,
			is_archived INTEGER DEFAULT 0,
			owner_id INTEGER DEFAULT 1
		);

		CREATE TABLE project_views (
			id INTEGER PRIMARY KEY,
			project_id INTEGER,
			title TEXT,
			view_kind INTEGER,
			position INTEGER,
			done_bucket_id INTEGER,
			default_bucket_id INTEGER
		);

		CREATE TABLE tasks (
			id INTEGER PRIMARY KEY,
			project_id INTEGER,
			title TEXT,
			description TEXT,
			done INTEGER DEFAULT 0,
			done_at DATETIME,
			priority INTEGER DEFAULT 0,
			"index" INTEGER,
			due_date DATETIME,
			created DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_by_id INTEGER DEFAULT 1
		);

		CREATE TABLE buckets (
			id INTEGER PRIMARY KEY,
			title TEXT,
			project_view_id INTEGER,
			position REAL
		);

		CREATE TABLE task_buckets (
			id INTEGER PRIMARY KEY,
			task_id INTEGER,
			bucket_id INTEGER,
			project_view_id INTEGER
		);

		CREATE TABLE task_positions (
			task_id INTEGER,
			project_view_id INTEGER,
			position REAL,
			PRIMARY KEY (task_id, project_view_id)
		);
	`

	if _, err := store.db.Exec(schema); err != nil {
		store.db.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("create schema: %v", err)
	}

	cleanup := func() {
		store.db.Close()
		os.Remove(tmpFile.Name())
	}

	return store, cleanup
}

func TestGetTaskWorkflowStatus_ReturnsCurrentBucketAndOrderedChoices(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO project_views (id, project_id, title, view_kind) VALUES (10, 1, 'Board', 3);
		INSERT INTO buckets (id, title, project_view_id, position) VALUES
			(12, 'Done', 10, 3), (10, 'To-Do', 10, 1), (11, 'Doing', 10, 2);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task');
		INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (1, 11, 10);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	status, err := store.GetTaskWorkflowStatus(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("get task workflow status: %v", err)
	}
	if status.Availability != WorkflowStatusAvailable || status.CurrentBucket.ID != 11 || status.CurrentBucket.Title != "Doing" {
		t.Fatalf("status = %#v, want available Doing bucket", status)
	}
	if got, want := status.Buckets, []WorkflowBucket{{ID: 10, Title: "To-Do"}, {ID: 11, Title: "Doing"}, {ID: 12, Title: "Done"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("buckets = %#v, want %#v", got, want)
	}
}

func TestGetTaskWorkflowStatus_ReportsUnavailableAndAmbiguousViews(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	status, err := store.GetTaskWorkflowStatus(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("get unavailable workflow status: %v", err)
	}
	if status.Availability != WorkflowStatusUnavailable {
		t.Fatalf("unavailable status = %#v", status)
	}

	_, err = store.db.Exec(`
		INSERT INTO project_views (id, project_id, title, view_kind) VALUES
			(10, 1, 'Board A', 3), (11, 1, 'Board B', 3);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}
	status, err = store.GetTaskWorkflowStatus(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("get ambiguous workflow status: %v", err)
	}
	if status.Availability != WorkflowStatusAmbiguous {
		t.Fatalf("ambiguous status = %#v", status)
	}
}

func TestSetTaskWorkflowStatus_UpdatesBucketAndDoneState(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id) VALUES (10, 1, 'Board', 3, 12);
		INSERT INTO buckets (id, title, project_view_id, position) VALUES
			(10, 'To-Do', 10, 1), (11, 'Doing', 10, 2), (12, 'Done', 10, 3);
		INSERT INTO tasks (id, project_id, title, done, done_at, updated) VALUES
			(1, 1, 'Task', 1, CURRENT_TIMESTAMP, '2020-01-01 00:00:00');
		INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (1, 12, 10);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	if err := store.SetTaskWorkflowStatus(context.Background(), 1, 11); err != nil {
		t.Fatalf("set Doing workflow status: %v", err)
	}
	var bucketID int64
	var done bool
	var doneAt sql.NullTime
	var updated time.Time
	if err := store.db.QueryRow(`SELECT tb.bucket_id, t.done, t.done_at, t.updated FROM tasks t JOIN task_buckets tb ON tb.task_id = t.id WHERE t.id = 1 AND tb.project_view_id = 10`).Scan(&bucketID, &done, &doneAt, &updated); err != nil {
		t.Fatalf("read Doing state: %v", err)
	}
	if bucketID != 11 || done || doneAt.Valid || !updated.After(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Doing state = bucket:%d done:%v doneAt:%#v updated:%v", bucketID, done, doneAt, updated)
	}

	if err := store.SetTaskWorkflowStatus(context.Background(), 1, 12); err != nil {
		t.Fatalf("set Done workflow status: %v", err)
	}
	if err := store.db.QueryRow(`SELECT tb.bucket_id, t.done, t.done_at FROM tasks t JOIN task_buckets tb ON tb.task_id = t.id WHERE t.id = 1 AND tb.project_view_id = 10`).Scan(&bucketID, &done, &doneAt); err != nil {
		t.Fatalf("read Done state: %v", err)
	}
	if bucketID != 12 || !done || !doneAt.Valid {
		t.Fatalf("Done state = bucket:%d done:%v doneAt:%#v", bucketID, done, doneAt)
	}
}

func TestToggleTaskDone_UpdatesTimestamp(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert test task
	insertTask := `INSERT INTO tasks (id, project_id, title, updated) VALUES (1, 1, 'Test Task', '2020-01-01 00:00:00')`
	if _, err := store.db.Exec(insertTask); err != nil {
		t.Fatalf("insert test task: %v", err)
	}

	// Toggle done
	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatalf("toggle task done: %v", err)
	}

	// Check updated timestamp is bumped
	var updated time.Time
	q := `SELECT updated FROM tasks WHERE id = 1`
	if err := store.db.QueryRow(q).Scan(&updated); err != nil {
		t.Fatalf("query updated: %v", err)
	}

	// Should be recent (within last 5 seconds)
	if time.Since(updated) > 5*time.Second {
		t.Errorf("updated timestamp not bumped: got %v", updated)
	}
}

func TestToggleTaskDone_MovesBucketWhenDone(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert project, view with done_bucket_id, task, bucket assignment
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id) VALUES (1, 1, 'Kanban', 3, 99);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task 1');
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES (1, 1, 10, 1);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	// Toggle done
	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatalf("toggle task done: %v", err)
	}

	// Check bucket_id is moved to done_bucket_id (99)
	var bucketID int64
	q := `SELECT bucket_id FROM task_buckets WHERE task_id = 1`
	if err := store.db.QueryRow(q).Scan(&bucketID); err != nil {
		t.Fatalf("query bucket_id: %v", err)
	}

	if bucketID != 99 {
		t.Errorf("task not moved to done bucket: got %d, want 99", bucketID)
	}
}

func TestToggleTaskDone_LeavesBucketWithoutDoneBucket(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind) VALUES (1, 1, 'Kanban', 3);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task 1');
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES (1, 1, 10, 1);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatalf("toggle task done: %v", err)
	}

	var bucketID int64
	if err := store.db.QueryRow(`SELECT bucket_id FROM task_buckets WHERE task_id = 1`).Scan(&bucketID); err != nil {
		t.Fatalf("query bucket_id: %v", err)
	}
	if bucketID != 10 {
		t.Errorf("bucket changed without a done bucket: got %d, want 10", bucketID)
	}
}

func TestToggleTaskDone_MovesTaskToEachViewsDoneBucket(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id) VALUES
			(1, 1, 'Team', 3, 99),
			(2, 1, 'Personal', 3, 199);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task 1');
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES
			(1, 1, 10, 1),
			(2, 1, 20, 2);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatalf("toggle task done: %v", err)
	}

	for viewID, wantBucketID := range map[int64]int64{1: 99, 2: 199} {
		var gotBucketID int64
		if err := store.db.QueryRow(`
			SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = ?
		`, viewID).Scan(&gotBucketID); err != nil {
			t.Fatalf("query bucket for view %d: %v", viewID, err)
		}
		if gotBucketID != wantBucketID {
			t.Errorf("view %d bucket = %d, want %d", viewID, gotBucketID, wantBucketID)
		}
	}
}

func TestToggleTaskDone_MovesToDefaultBucketWhenUndone(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Setup: task in its Done Bucket, then reopen it.
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id, default_bucket_id) VALUES (1, 1, 'Kanban', 3, 99, 10);
		INSERT INTO tasks (id, project_id, title, done) VALUES (1, 1, 'Task 1', 1);
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES (1, 1, 99, 1);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	// Toggle undone
	if err := store.ToggleTaskDone(ctx, 1, false); err != nil {
		t.Fatalf("toggle task undone: %v", err)
	}

	// Reopening must move to the configured Default Bucket.
	var bucketID int64
	q := `SELECT bucket_id FROM task_buckets WHERE task_id = 1`
	if err := store.db.QueryRow(q).Scan(&bucketID); err != nil {
		t.Fatalf("query bucket_id: %v", err)
	}

	if bucketID != 10 {
		t.Errorf("bucket = %d, want default bucket 10", bucketID)
	}
}

func TestToggleTaskDone_MovesTaskToEachViewsDefaultBucketWhenUndone(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id, default_bucket_id) VALUES
			(1, 1, 'Team', 3, 99, 10),
			(2, 1, 'Personal', 3, 199, 20);
		INSERT INTO tasks (id, project_id, title, done) VALUES (1, 1, 'Task 1', 1);
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES
			(1, 1, 99, 1),
			(2, 1, 199, 2);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	if err := store.ToggleTaskDone(context.Background(), 1, false); err != nil {
		t.Fatalf("reopen task: %v", err)
	}
	for viewID, wantBucketID := range map[int64]int64{1: 10, 2: 20} {
		var gotBucketID int64
		if err := store.db.QueryRow(`SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = ?`, viewID).Scan(&gotBucketID); err != nil {
			t.Fatalf("query bucket for view %d: %v", viewID, err)
		}
		if gotBucketID != wantBucketID {
			t.Errorf("view %d bucket = %d, want default bucket %d", viewID, gotBucketID, wantBucketID)
		}
	}
}

func TestListProjects_ExcludesArchived(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert 2 projects: 1 active, 1 archived
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title, is_archived) VALUES (1, 'Active', 0);
		INSERT INTO projects (id, title, is_archived) VALUES (2, 'Archived', 1);
	`)
	if err != nil {
		t.Fatalf("insert projects: %v", err)
	}

	projects, _, err := store.ListProjects(ctx, 1, 50)
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}
	if len(projects) > 0 && projects[0].Title != "Active" {
		t.Errorf("expected 'Active', got '%s'", projects[0].Title)
	}
}

func TestListProjects_Pagination(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert 5 projects
	for i := 1; i <= 5; i++ {
		_, err := store.db.Exec(`INSERT INTO projects (id, title) VALUES (?, ?)`, i, "Project "+string(rune('A'+i-1)))
		if err != nil {
			t.Fatalf("insert project %d: %v", i, err)
		}
	}

	// Fetch page 1 (2 per page)
	projects, totalPages, err := store.ListProjects(ctx, 1, 2)
	if err != nil {
		t.Fatalf("list projects page 1: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects on page 1, got %d", len(projects))
	}
	if totalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", totalPages)
	}

	// Fetch page 2
	projects, _, err = store.ListProjects(ctx, 2, 2)
	if err != nil {
		t.Fatalf("list projects page 2: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects on page 2, got %d", len(projects))
	}
}

func TestListViews_OrdersAndMapsKinds(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, position) VALUES
			(1, 1, 'Kanban', 3, 30),
			(2, 1, 'List', 0, 10),
			(3, 1, 'Table', 2, 20);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	views, _, err := store.ListViews(context.Background(), 1, 1, 50)
	if err != nil {
		t.Fatalf("list views: %v", err)
	}

	want := []struct {
		id   int64
		kind string
	}{{2, "list"}, {3, "table"}, {1, "kanban"}}
	if len(views) != len(want) {
		t.Fatalf("got %d views, want %d", len(views), len(want))
	}
	for i, expected := range want {
		if views[i].ID != expected.id || views[i].ViewKind != expected.kind {
			t.Errorf("view %d = {ID: %d, ViewKind: %q}, want {ID: %d, ViewKind: %q}", i, views[i].ID, views[i].ViewKind, expected.id, expected.kind)
		}
	}
}

func TestListProjectsAndTasks_HandleNullableFields(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, identifier, title, description) VALUES (1, NULL, 'Test Project', NULL);
		INSERT INTO project_views (id, project_id, title, view_kind) VALUES (1, 1, 'List', 0);
		INSERT INTO tasks (id, project_id, title, due_date, "index", description) VALUES (1, 1, 'Task 1', NULL, NULL, NULL);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	projects, _, err := store.ListProjects(context.Background(), 1, 50)
	if err != nil {
		t.Fatalf("list projects with NULL fields: %v", err)
	}
	if len(projects) != 1 || projects[0].Identifier != "" || projects[0].Description != "" {
		t.Errorf("projects = %#v, want one project with empty nullable fields", projects)
	}

	tasks, _, err := store.ListTasksInView(context.Background(), 1, 1, 1, 50)
	if err != nil {
		t.Fatalf("list tasks with NULL fields: %v", err)
	}
	if len(tasks) != 1 || tasks[0].DueDate != "" || tasks[0].Index != 0 || tasks[0].Description != "" {
		t.Errorf("tasks = %#v, want one task with empty nullable fields", tasks)
	}
}

func TestListTasksInView_OrdersByPosition(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert project, view, 3 tasks with positions
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Project A');
		INSERT INTO project_views (id, project_id, title, view_kind) VALUES (1, 1, 'List', 0);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task C');
		INSERT INTO tasks (id, project_id, title) VALUES (2, 1, 'Task A');
		INSERT INTO tasks (id, project_id, title) VALUES (3, 1, 'Task B');
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (1, 1, 3.0);
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (2, 1, 1.0);
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (3, 1, 2.0);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	tasks, _, err := store.ListTasksInView(ctx, 1, 1, 1, 50)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// Check order: Task A (pos 1.0), Task B (2.0), Task C (3.0)
	expected := []string{"Task A", "Task B", "Task C"}
	for i, task := range tasks {
		if task.Title != expected[i] {
			t.Errorf("task %d: expected '%s', got '%s'", i, expected[i], task.Title)
		}
	}
}

func TestToggleTaskDone_SetsDoneAt(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert task
	_, err := store.db.Exec(`INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task')`)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	// Toggle done
	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatalf("toggle done: %v", err)
	}

	// Check done_at is set
	var doneAt sql.NullTime
	q := `SELECT done_at FROM tasks WHERE id = 1`
	if err := store.db.QueryRow(q).Scan(&doneAt); err != nil {
		t.Fatalf("query done_at: %v", err)
	}

	if !doneAt.Valid {
		t.Error("done_at not set")
	}
	if time.Since(doneAt.Time) > 5*time.Second {
		t.Errorf("done_at not recent: %v", doneAt.Time)
	}

	// Toggle undone
	if err := store.ToggleTaskDone(ctx, 1, false); err != nil {
		t.Fatalf("toggle undone: %v", err)
	}

	// Check done_at is cleared
	if err := store.db.QueryRow(q).Scan(&doneAt); err != nil {
		t.Fatalf("query done_at after undone: %v", err)
	}

	if doneAt.Valid {
		t.Error("done_at not cleared when undone")
	}
}
