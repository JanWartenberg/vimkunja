package vimkunja

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
)

func TestIntegrityAuditCalDAVIdentity(t *testing.T) {
	store := setupUndoStore(t)
	bucketExec(t, store, `ALTER TABLE tasks ADD COLUMN uid TEXT;`)
	task, err := store.CreateTask(context.Background(), TaskDraft{ProjectID: 1, Title: "Calendar Task"})
	if err != nil {
		t.Fatal(err)
	}
	var uid sql.NullString
	if err := store.db.QueryRow(`SELECT uid FROM tasks WHERE id = ?`, task.ID).Scan(&uid); err != nil {
		t.Fatal(err)
	}
	if !uid.Valid || uid.String == "" {
		t.Error("created Task has no CalDAV UID")
	}
}

func TestTaskUIDUniqueAndStableAcrossMutations(t *testing.T) {
	store := setupUndoStore(t)
	ctx := context.Background()
	bucketExec(t, store, `ALTER TABLE tasks ADD COLUMN uid TEXT NOT NULL DEFAULT ''; CREATE UNIQUE INDEX nonempty_uid ON tasks(uid) WHERE uid!=''`)
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := map[string]bool{}
	var task Task
	for i := 0; i < 20; i++ {
		var err error
		task, err = store.CreateTask(ctx, TaskDraft{ProjectID: 1, Title: "Calendar"})
		if err != nil {
			t.Fatal(err)
		}
		var uid string
		if err := store.db.QueryRow(`SELECT uid FROM tasks WHERE id=?`, task.ID).Scan(&uid); err != nil {
			t.Fatal(err)
		}
		if !pattern.MatchString(uid) || seen[uid] {
			t.Fatalf("invalid or duplicate UUID: %q", uid)
		}
		seen[uid] = true
	}
	read := func() string {
		t.Helper()
		var uid string
		if err := store.db.QueryRow(`SELECT uid FROM tasks WHERE id=?`, task.ID).Scan(&uid); err != nil {
			t.Fatal(err)
		}
		return uid
	}
	original := read()
	before, err := store.TaskUndoState(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateTask(ctx, TaskDraft{ID: task.ID, ProjectID: 2, Title: "Moved"}); err != nil {
		t.Fatal(err)
	}
	after, err := store.TaskUndoState(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read() != original {
		t.Fatal("move changed UID")
	}
	if err := store.RestoreTaskUndoState(ctx, after, before); err != nil {
		t.Fatal(err)
	}
	if read() != original {
		t.Fatal("undo changed UID")
	}
	if err := store.RestoreTaskUndoState(ctx, before, after); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateTaskTitle(ctx, task.ID, "Renamed"); err != nil {
		t.Fatal(err)
	}
	if err := store.ToggleTaskDone(ctx, task.ID, true); err != nil {
		t.Fatal(err)
	}
	if read() != original {
		t.Fatal("redo/title/completion changed UID")
	}
}

func TestTaskUIDFailedCreationRollsBack(t *testing.T) {
	store := setupUndoStore(t)
	bucketExec(t, store, `ALTER TABLE tasks ADD COLUMN uid TEXT; CREATE TRIGGER reject_new_bucket BEFORE INSERT ON task_buckets BEGIN SELECT RAISE(ABORT,'late creation failure'); END;`)
	if _, err := store.CreateTask(context.Background(), TaskDraft{ProjectID: 1, Title: "Rejected"}); err == nil {
		t.Fatal("injected failure accepted")
	}
	if bucketInt(t, store, `SELECT COUNT(*) FROM tasks`) != 1 || bucketInt(t, store, `SELECT COUNT(*) FROM tasks WHERE uid IS NOT NULL`) != 0 || bucketInt(t, store, `SELECT COUNT(*) FROM task_positions`) != 1 {
		t.Fatal("failed UID creation leaked rows")
	}
}

func TestTaskUIDHistoricalAndCaseInsensitiveSchema(t *testing.T) {
	for _, column := range []string{"", "UID"} {
		t.Run(column, func(t *testing.T) {
			store := setupUndoStore(t)
			if column != "" {
				bucketExec(t, store, "ALTER TABLE tasks ADD COLUMN "+column+" TEXT")
			}
			task, err := store.CreateTask(context.Background(), TaskDraft{ProjectID: 1, Title: "Schema"})
			if err != nil {
				t.Fatal(err)
			}
			if column != "" {
				var uid sql.NullString
				if err := store.db.QueryRow(`SELECT UID FROM tasks WHERE id=?`, task.ID).Scan(&uid); err != nil || !uid.Valid || uid.String == "" {
					t.Fatalf("case-insensitive UID missing: %v %v", uid, err)
				}
			}
		})
	}
}
