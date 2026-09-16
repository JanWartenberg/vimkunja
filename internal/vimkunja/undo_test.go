package vimkunja

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func setupUndoStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	_, err := store.db.Exec(`
		CREATE TABLE labels (id INTEGER PRIMARY KEY, title TEXT);
		CREATE TABLE label_tasks (id INTEGER PRIMARY KEY, task_id INTEGER, label_id INTEGER, created DATETIME);
		INSERT INTO projects (id, identifier, title) VALUES (1, 'ONE', 'One'), (2, 'TWO', 'Two');
		INSERT INTO project_views (id, project_id, title, view_kind, default_bucket_id) VALUES
			(10, 1, 'Board One', 3, 100), (20, 2, 'Board Two', 3, 200);
		INSERT INTO buckets (id, title, project_view_id) VALUES
			(100, 'Todo One', 10), (101, 'Done One', 10), (200, 'Todo Two', 20);
		INSERT INTO labels (id, title) VALUES (7, 'First'), (8, 'Second');
		INSERT INTO tasks (id, project_id, title, description, done, priority, "index", due_date)
			VALUES (1, 1, 'Before', 'old', 0, 2, 3, '2026-09-01');
		INSERT INTO label_tasks (task_id, label_id, created) VALUES (1, 7, CURRENT_TIMESTAMP);
		INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (1, 100, 10);
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (1, 10, 1);
	`)
	if err != nil {
		t.Fatalf("seed undo store: %v", err)
	}
	return store
}

func TestRestoreTaskUndoStateRestoresAllFieldsAtomically(t *testing.T) {
	store := setupUndoStore(t)
	ctx := context.Background()
	before, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if before.Identifier != "ONE-3" {
		t.Fatalf("identifier = %q, want ONE-3", before.Identifier)
	}
	_, err = store.UpdateTask(ctx, TaskDraft{
		ID: 1, ProjectID: 2, Title: "After", Description: "new", Priority: 5,
		DueDate: "2026-10-02", Done: true, DoneSet: true,
		LabelIDs: []int64{8}, LabelIDsSet: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreTaskUndoState(ctx, after, before); err != nil {
		t.Fatalf("undo restore: %v", err)
	}
	restored, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !sameTaskUndoState(restored, before) {
		t.Fatalf("restored state = %#v, want %#v", restored, before)
	}
	if err := store.RestoreTaskUndoState(ctx, before, after); err != nil {
		t.Fatalf("redo restore: %v", err)
	}
	redone, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !sameTaskUndoState(redone, after) {
		t.Fatalf("redone state = %#v, want %#v", redone, after)
	}
}

func TestRestoreTaskUndoStateRejectsConflictWithoutWriting(t *testing.T) {
	store := setupUndoStore(t)
	ctx := context.Background()
	before, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateTaskTitle(ctx, 1, "After"); err != nil {
		t.Fatal(err)
	}
	after, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateTaskTitle(ctx, 1, "External"); err != nil {
		t.Fatal(err)
	}
	err = store.RestoreTaskUndoState(ctx, after, before)
	if err == nil || !strings.Contains(err.Error(), "changed since") {
		t.Fatalf("conflict error = %v", err)
	}
	current, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if current.Title != "External" {
		t.Fatalf("conflict changed title to %q", current.Title)
	}
}

func TestRestoreTaskUndoStatePreservesUnassignedBucketSentinel(t *testing.T) {
	store := setupUndoStore(t)
	ctx := context.Background()
	if _, err := store.db.Exec(`UPDATE task_buckets SET bucket_id = 0 WHERE task_id = 1`); err != nil {
		t.Fatal(err)
	}
	before, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceTaskLabels(ctx, 1, nil); err != nil {
		t.Fatal(err)
	}
	after, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreTaskUndoState(ctx, after, before); err != nil {
		t.Fatalf("undo label removal with unassigned Bucket: %v", err)
	}
	restored, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !sameTaskUndoState(restored, before) {
		t.Fatalf("restored state = %#v, want %#v", restored, before)
	}
}

func TestRestoreTaskUndoStateMissingDependencyRollsBack(t *testing.T) {
	store := setupUndoStore(t)
	ctx := context.Background()
	expected, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	target := expected
	target.Title = "Must not persist"
	target.LabelIDs = []int64{999}
	err = store.RestoreTaskUndoState(ctx, expected, target)
	if err == nil || !strings.Contains(err.Error(), "Label 999") {
		t.Fatalf("missing dependency error = %v", err)
	}
	current, err := store.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(current, expected) {
		t.Fatalf("failed restore changed state: %#v", current)
	}
}
