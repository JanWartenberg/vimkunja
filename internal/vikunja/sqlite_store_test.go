package vikunja

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"
)

// setupTestDB creates a temporary SQLite DB with minimal schema + test data
func setupTestDB(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "vikunja-test-*.db")
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
			is_archived INTEGER DEFAULT 0
		);

		CREATE TABLE project_views (
			id INTEGER PRIMARY KEY,
			project_id INTEGER,
			title TEXT,
			view_kind INTEGER,
			position INTEGER,
			done_bucket_id INTEGER
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
			updated DATETIME DEFAULT CURRENT_TIMESTAMP
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

func TestToggleTaskDone_LeavesInBucketWhenUndone(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Setup: task in bucket 10, mark done, then undo
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id) VALUES (1, 1, 'Kanban', 3, 99);
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

	// Check bucket_id stays at 99 (we don't auto-move back)
	var bucketID int64
	q := `SELECT bucket_id FROM task_buckets WHERE task_id = 1`
	if err := store.db.QueryRow(q).Scan(&bucketID); err != nil {
		t.Fatalf("query bucket_id: %v", err)
	}

	if bucketID != 99 {
		t.Errorf("bucket changed unexpectedly: got %d, want 99", bucketID)
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
