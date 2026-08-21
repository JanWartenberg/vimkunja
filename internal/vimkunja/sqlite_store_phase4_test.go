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

func TestCreateTaskPlacesItInRequestedWorkflowBucket(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	if _, err := store.db.Exec(`INSERT INTO projects (id, title, owner_id) VALUES (1, 'Project', 7)`); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	board, err := store.ProvisionStandardManualKanbanView(context.Background(), 1)
	if err != nil {
		t.Fatalf("ProvisionStandardManualKanbanView: %v", err)
	}
	doneBucket := board.Buckets[len(board.Buckets)-1]

	task, err := store.CreateTask(context.Background(), TaskDraft{ProjectID: 1, Title: "Done on create", WorkflowBucketID: doneBucket.ID})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if !task.Done {
		t.Fatalf("created task Done = false, want true for Done bucket")
	}
	board, err = store.LoadManualKanbanBoard(context.Background(), 1)
	if err != nil {
		t.Fatalf("LoadManualKanbanBoard: %v", err)
	}
	for _, bucket := range board.Buckets {
		for _, candidate := range bucket.Tasks {
			if candidate.ID == task.ID && bucket.ID == doneBucket.ID {
				return
			}
		}
	}
	t.Fatalf("created task %d is not in requested bucket %d", task.ID, doneBucket.ID)
}

func TestTaskDraftLabelsPersistAtomicallyWithCreateAndUpdate(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	if _, err := store.db.Exec(`
		CREATE TABLE labels (
			id INTEGER PRIMARY KEY, title TEXT NOT NULL, description TEXT, hex_color TEXT,
			created_by_id INTEGER NOT NULL, created DATETIME NOT NULL, updated DATETIME NOT NULL
		);
		CREATE TABLE label_tasks (
			id INTEGER PRIMARY KEY, label_id INTEGER NOT NULL, task_id INTEGER NOT NULL, created DATETIME NOT NULL
		);
		INSERT INTO projects (id, title, owner_id) VALUES (1, 'Project', 7);
		INSERT INTO labels (id, title, created_by_id, created, updated) VALUES
			(10, 'First', 7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			(11, 'Second', 7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`); err != nil {
		t.Fatalf("seed labels: %v", err)
	}

	task, err := store.CreateTask(context.Background(), TaskDraft{ProjectID: 1, Title: "New", LabelIDs: []int64{10}, LabelIDsSet: true})
	if err != nil {
		t.Fatalf("create Task with Labels: %v", err)
	}
	labels, err := store.ListTaskLabels(context.Background(), task.ID)
	if err != nil || len(labels) != 1 || labels[0].ID != 10 {
		t.Fatalf("created Task Labels = %#v, %v", labels, err)
	}

	if _, err := store.UpdateTask(context.Background(), TaskDraft{ID: task.ID, ProjectID: 1, Title: "Renamed", LabelIDs: []int64{11}, LabelIDsSet: true}); err != nil {
		t.Fatalf("update Task Labels: %v", err)
	}
	labels, err = store.ListTaskLabels(context.Background(), task.ID)
	if err != nil || len(labels) != 1 || labels[0].ID != 11 {
		t.Fatalf("updated Task Labels = %#v, %v", labels, err)
	}

	if _, err := store.UpdateTask(context.Background(), TaskDraft{ID: task.ID, ProjectID: 1, Title: "Must Roll Back", LabelIDs: []int64{999}, LabelIDsSet: true}); err == nil {
		t.Fatal("update accepted a missing Label")
	}
	var title string
	if err := store.db.QueryRow(`SELECT title FROM tasks WHERE id = ?`, task.ID).Scan(&title); err != nil {
		t.Fatal(err)
	}
	labels, err = store.ListTaskLabels(context.Background(), task.ID)
	if title != "Renamed" || err != nil || len(labels) != 1 || labels[0].ID != 11 {
		t.Fatalf("failed Label update changed Task = title:%q Labels:%#v error:%v", title, labels, err)
	}
}

func TestUpdateTask_PersistsWorkflowStatusWithTaskFields(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Project');
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id) VALUES (10, 1, 'Board', 3, 12);
		INSERT INTO buckets (id, title, project_view_id, position) VALUES (10, 'To-Do', 10, 1), (11, 'Doing', 10, 2), (12, 'Done', 10, 3);
		INSERT INTO tasks (id, project_id, title, done, "index", created, updated, created_by_id) VALUES (1, 1, 'Old', 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1);
		INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (1, 10, 10);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	if _, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Renamed", WorkflowBucketID: 12}); err != nil {
		t.Fatalf("update task: %v", err)
	}
	var title string
	var done bool
	var doneAt sql.NullTime
	var bucketID int64
	if err := store.db.QueryRow(`SELECT t.title, t.done, t.done_at, tb.bucket_id FROM tasks t JOIN task_buckets tb ON tb.task_id = t.id WHERE t.id = 1 AND tb.project_view_id = 10`).Scan(&title, &done, &doneAt, &bucketID); err != nil {
		t.Fatalf("read updated task: %v", err)
	}
	if title != "Renamed" || !done || !doneAt.Valid || bucketID != 12 {
		t.Fatalf("updated task = title:%q done:%v doneAt:%#v bucket:%d", title, done, doneAt, bucketID)
	}
}

func TestUpdateTaskPersistsDoneWithOtherFields(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Project');
		INSERT INTO project_views (id, project_id, title, view_kind, default_bucket_id, done_bucket_id) VALUES (10, 1, 'Board', 3, 10, 12);
		INSERT INTO buckets (id, title, project_view_id, position) VALUES (10, 'To-Do', 10, 1), (12, 'Done', 10, 2);
		INSERT INTO tasks (id, project_id, title, done, "index", created, updated, created_by_id) VALUES (1, 1, 'Old', 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1);
		INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (1, 10, 10);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	task, err := store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Renamed", Done: true, DoneSet: true})
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	var done bool
	var doneAt sql.NullTime
	var bucketID int64
	if err := store.db.QueryRow(`SELECT t.done, t.done_at, tb.bucket_id FROM tasks t JOIN task_buckets tb ON tb.task_id = t.id WHERE t.id = 1 AND tb.project_view_id = 10`).Scan(&done, &doneAt, &bucketID); err != nil {
		t.Fatalf("read updated task: %v", err)
	}
	if task.Title != "Renamed" || !task.Done || !done || !doneAt.Valid || bucketID != 12 {
		t.Fatalf("updated task = %#v; done:%v doneAt:%#v bucket:%d", task, done, doneAt, bucketID)
	}

	task, err = store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Renamed", Done: false, DoneSet: true})
	if err != nil {
		t.Fatalf("reopen task: %v", err)
	}
	if err := store.db.QueryRow(`SELECT t.done, t.done_at, tb.bucket_id FROM tasks t JOIN task_buckets tb ON tb.task_id = t.id WHERE t.id = 1 AND tb.project_view_id = 10`).Scan(&done, &doneAt, &bucketID); err != nil {
		t.Fatalf("read reopened task: %v", err)
	}
	if task.Done || done || doneAt.Valid || bucketID != 10 {
		t.Fatalf("reopened task = %#v; done:%v doneAt:%#v bucket:%d", task, done, doneAt, bucketID)
	}
}

func TestUpdateTaskRollsBackDoneWithOtherFieldsOnFailure(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Project');
		INSERT INTO project_views (id, project_id, title, view_kind, default_bucket_id, done_bucket_id) VALUES (10, 1, 'Board', 3, 10, 12);
		INSERT INTO buckets (id, title, project_view_id, position) VALUES (10, 'To-Do', 10, 1), (12, 'Done', 10, 2);
		INSERT INTO tasks (id, project_id, title, done, "index", created, updated, created_by_id) VALUES (1, 1, 'Old', 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1);
		INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (1, 10, 10);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	_, err = store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Renamed", WorkflowBucketID: 999, Done: true, DoneSet: true})
	if err == nil {
		t.Fatal("update task succeeded with an invalid workflow bucket")
	}
	var title string
	var done bool
	var bucketID int64
	if err := store.db.QueryRow(`SELECT t.title, t.done, tb.bucket_id FROM tasks t JOIN task_buckets tb ON tb.task_id = t.id WHERE t.id = 1 AND tb.project_view_id = 10`).Scan(&title, &done, &bucketID); err != nil {
		t.Fatalf("read rolled-back task: %v", err)
	}
	if title != "Old" || done || bucketID != 10 {
		t.Fatalf("partially saved task = title:%q done:%v bucket:%d", title, done, bucketID)
	}
}

func TestUpdateTaskTitleChangesOnlyTitle(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Project');
		INSERT INTO tasks (id, project_id, title, description, priority, "index", created, updated, created_by_id)
		VALUES (1, 1, 'Old', 'Keep this', 4, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}
	if err := store.UpdateTaskTitle(context.Background(), 1, "Renamed"); err != nil {
		t.Fatalf("UpdateTaskTitle: %v", err)
	}
	var title, description string
	var priority int64
	if err := store.db.QueryRow(`SELECT title, description, priority FROM tasks WHERE id = 1`).Scan(&title, &description, &priority); err != nil {
		t.Fatalf("read title update: %v", err)
	}
	if title != "Renamed" || description != "Keep this" || priority != 4 {
		t.Fatalf("title update changed unrelated fields: title:%q description:%q priority:%d", title, description, priority)
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
