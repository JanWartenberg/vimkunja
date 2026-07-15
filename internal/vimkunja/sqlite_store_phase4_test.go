package vimkunja

import (
	"context"
	"database/sql"
	"testing"
)

func TestCreateTask_InitializesDefaultsAndNullDueDate(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title, owner_id) VALUES (1, 'Project', 7);
		INSERT INTO project_views (id, project_id, title, view_kind, default_bucket_id) VALUES
			(10, 1, 'List', 0, NULL), (11, 1, 'Board', 3, 99);
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (42, 10, 4), (42, 11, 8);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	task, err := store.CreateTask(context.Background(), TaskDraft{ProjectID: 1, Title: "New", Priority: 3})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.ID == 0 || task.Index != 1 || task.ProjectID != 1 {
		t.Fatalf("created task = %#v, want generated ID, index 1, project 1", task)
	}
	var dueDate sql.NullTime
	var author int64
	if err := store.db.QueryRow(`SELECT due_date, created_by_id FROM tasks WHERE id = ?`, task.ID).Scan(&dueDate, &author); err != nil {
		t.Fatalf("read created task: %v", err)
	}
	if dueDate.Valid || author != 7 {
		t.Errorf("due/author = %#v/%d, want NULL/7", dueDate, author)
	}
	for viewID, wantPosition := range map[int64]float64{10: 5, 11: 9} {
		var got float64
		if err := store.db.QueryRow(`SELECT position FROM task_positions WHERE task_id = ? AND project_view_id = ?`, task.ID, viewID).Scan(&got); err != nil {
			t.Fatalf("position for view %d: %v", viewID, err)
		}
		if got != wantPosition {
			t.Errorf("position for view %d = %v, want %v", viewID, got, wantPosition)
		}
	}
	var bucket int64
	if err := store.db.QueryRow(`SELECT bucket_id FROM task_buckets WHERE task_id = ? AND project_view_id = 11`, task.ID).Scan(&bucket); err != nil {
		t.Fatalf("default bucket: %v", err)
	}
	if bucket != 99 {
		t.Errorf("bucket = %d, want 99", bucket)
	}
}

func TestUpdateTask_MovesToTargetProjectDefaults(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Old'), (2, 'New');
		INSERT INTO project_views (id, project_id, title, view_kind, default_bucket_id) VALUES
			(10, 1, 'Old Board', 3, 10), (20, 2, 'New Board', 3, 20);
		INSERT INTO tasks (id, project_id, title, "index", created, updated, created_by_id) VALUES (1, 1, 'Old title', 3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1);
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (1, 10, 3);
		INSERT INTO task_buckets (id, task_id, project_view_id, bucket_id) VALUES (1, 1, 10, 10);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	task, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 2, Title: "Moved", Description: "text", DueDate: "2026-08-01", Priority: 4})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if task.ProjectID != 2 || task.Title != "Moved" || task.Index != 1 {
		t.Errorf("updated task = %#v", task)
	}
	var oldCount, newBucket int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM task_positions WHERE task_id = 1 AND project_view_id = 10`).Scan(&oldCount); err != nil {
		t.Fatal(err)
	}
	if oldCount != 0 {
		t.Errorf("old position count = %d, want 0", oldCount)
	}
	if err := store.db.QueryRow(`SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = 20`).Scan(&newBucket); err != nil {
		t.Fatal(err)
	}
	if newBucket != 20 {
		t.Errorf("new bucket = %d, want 20", newBucket)
	}
}
