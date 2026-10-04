package vimkunja

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func dueExec(t *testing.T, store *SQLiteStore, query string, args ...any) {
	t.Helper()
	if _, err := store.db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func dueTitleChange(t *testing.T, store *SQLiteStore) (TaskUndoState, TaskUndoState) {
	t.Helper()
	before, err := store.TaskUndoState(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateTaskTitle(context.Background(), 1, "After"); err != nil {
		t.Fatal(err)
	}
	after, err := store.TaskUndoState(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	return before, after
}

func TestIntegrityAuditDueTimePreservation(t *testing.T) {
	const due = "2026-09-01 18:45:00"
	t.Run("detail_title_save", func(t *testing.T) {
		store := setupUndoStore(t)
		dueExec(t, store, `UPDATE tasks SET due_date = ? WHERE id = 1`, due)
		tasks, _, err := store.ListTasksInView(context.Background(), 1, 10, 1, 50)
		if err != nil || len(tasks) != 1 {
			t.Fatalf("read Task: tasks=%v err=%v", tasks, err)
		}
		task := tasks[0]
		_, err = store.UpdateTask(context.Background(), TaskDraft{
			ID: task.ID, ProjectID: task.ProjectID, Title: "Renamed", Description: task.Description,
			Priority: task.Priority, DueDate: task.DueDate,
		})
		if err != nil {
			t.Fatal(err)
		}
		var actual time.Time
		if err := store.db.QueryRow(`SELECT due_date FROM tasks WHERE id = 1`).Scan(&actual); err != nil {
			t.Fatal(err)
		}
		if actual.Hour() != 18 || actual.Minute() != 45 {
			t.Errorf("title-only Detail save changed due time from 18:45 to %s", actual.Format("15:04:05"))
		}
	})
	t.Run("inline_title_undo", func(t *testing.T) {
		store := setupUndoStore(t)
		dueExec(t, store, `UPDATE tasks SET due_date = ? WHERE id = 1`, due)
		before, after := dueTitleChange(t, store)
		if err := store.RestoreTaskUndoState(context.Background(), after, before); err != nil {
			t.Fatal(err)
		}
		var actual time.Time
		if err := store.db.QueryRow(`SELECT due_date FROM tasks WHERE id = 1`).Scan(&actual); err != nil {
			t.Fatal(err)
		}
		if actual.Hour() != 18 || actual.Minute() != 45 {
			t.Errorf("undo of inline title edit changed unrelated due time to %s", actual.Format("15:04:05"))
		}
	})
	t.Run("external_time_change_blocks_undo", func(t *testing.T) {
		store := setupUndoStore(t)
		dueExec(t, store, `UPDATE tasks SET due_date = ? WHERE id = 1`, due)
		before, after := dueTitleChange(t, store)
		dueExec(t, store, `UPDATE tasks SET due_date = '2026-09-01 20:30:00' WHERE id = 1`)
		if err := store.RestoreTaskUndoState(context.Background(), after, before); err == nil {
			t.Error("undo accepted an external due-time change and overwrote it")
		}
	})
}

func TestDueTimestampRoundTripWithTaskAndComment(t *testing.T) {
	for _, withComment := range []bool{false, true} {
		for _, due := range []any{
			nil, "2026-09-01", "2026-09-01 18:45:12.123456789",
			"2026-09-01T18:45:12.123456789+02:00",
			"2026-09-01 00:30:12.123456789+14:00",
			"2026-09-01T23:30:12.123456789-11:00",
			"2026-09-01 18:45:12.123456789 +0200 CEST",
			"2026-09-01 18:45:12.123456789 +0000 UTC m=+123.456",
			"2026-09-01T18:45:12Z", "2026-09-01 18:45:12Z",
			"2026-09-01T18:45:12.123456789", "2026-09-01 18:45", "2026-09-01T18:45",
		} {
			t.Run(fmt.Sprintf("comment=%t/due=%v", withComment, due), func(t *testing.T) {
				store := setupUndoStore(t)
				dueExec(t, store, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, username TEXT);
                    CREATE TABLE task_comments (id INTEGER PRIMARY KEY, comment TEXT, author_id INTEGER, task_id INTEGER, created DATETIME, updated DATETIME);
                    INSERT INTO users VALUES (1, 'Local', 'local');`)
				dueExec(t, store, `UPDATE tasks SET due_date = ? WHERE id = 1`, due)
				before, err := store.TaskUndoState(context.Background(), 1)
				if err != nil {
					t.Fatal(err)
				}
				tasks, _, err := store.ListTasksInView(context.Background(), 1, 10, 1, 50)
				if err != nil || len(tasks) != 1 {
					t.Fatalf("load: %v %v", tasks, err)
				}
				draft := TaskDraft{ID: 1, ProjectID: 1, Title: "Renamed", Description: tasks[0].Description, Priority: tasks[0].Priority, DueDate: tasks[0].DueDate}
				if withComment {
					_, _, err = store.SaveTaskWithComment(context.Background(), draft, TaskCommentDraft{Text: "Comment"})
				} else {
					_, err = store.UpdateTask(context.Background(), draft)
				}
				if err != nil {
					t.Fatal(err)
				}
				assertStoredDue(t, store, due)
				after, err := store.TaskUndoState(context.Background(), 1)
				if err != nil {
					t.Fatal(err)
				}
				if err := store.RestoreTaskUndoState(context.Background(), after, before); err != nil {
					t.Fatal(err)
				}
				assertStoredDue(t, store, due)
				if err := store.RestoreTaskUndoState(context.Background(), before, after); err != nil {
					t.Fatal(err)
				}
				assertStoredDue(t, store, due)
				if withComment {
					var count int
					if err := store.db.QueryRow(`SELECT COUNT(*) FROM task_comments`).Scan(&count); err != nil || count != 1 {
						t.Fatalf("Undo changed comments: %d %v", count, err)
					}
				}
			})
		}
	}
}

func assertStoredDue(t *testing.T, store *SQLiteStore, want any) {
	t.Helper()
	var got sql.NullString
	if err := store.db.QueryRow(`SELECT CAST(due_date AS TEXT) FROM tasks WHERE id = 1`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want == nil {
		if got.Valid {
			t.Fatalf("due = %q, want NULL", got.String)
		}
	} else if !got.Valid || got.String != want.(string) {
		t.Fatalf("due = %q, want exact %q", got.String, want)
	}
}

func TestDueDateEditClearAndUndo(t *testing.T) {
	store := setupUndoStore(t)
	const original = "2026-09-01T18:45:12.123456789+02:00"
	dueExec(t, store, `UPDATE tasks SET due_date = ? WHERE id = 1`, original)
	for _, date := range []string{"2026-10-02", ""} {
		before, err := store.TaskUndoState(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		_, err = store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Edited", DueDate: date})
		if err != nil {
			t.Fatal(err)
		}
		after, err := store.TaskUndoState(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if after.DueDate != date {
			t.Fatalf("edited date = %q, want %q", after.DueDate, date)
		}
		var got sql.NullTime
		if err := store.db.QueryRow(`SELECT due_date FROM tasks WHERE id = 1`).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if date == "" {
			if got.Valid {
				t.Fatal("clear did not store NULL")
			}
		} else if !got.Valid || got.Time.Hour() != 0 || got.Time.Minute() != 0 || got.Time.Second() != 0 || got.Time.Nanosecond() != 0 {
			t.Fatalf("explicit date edit retained a time: %v", got)
		}
		if err := store.RestoreTaskUndoState(context.Background(), after, before); err != nil {
			t.Fatal(err)
		}
		assertStoredDue(t, store, original)
		if err := store.RestoreTaskUndoState(context.Background(), before, after); err != nil {
			t.Fatal(err)
		}
		if err := store.RestoreTaskUndoState(context.Background(), after, before); err != nil {
			t.Fatal(err)
		}
		assertStoredDue(t, store, original)
	}
}

func TestDueTimeConflictAndLateFailureKeepStoredState(t *testing.T) {
	store := setupUndoStore(t)
	const due = "2026-09-01 18:45:12.123456789"
	dueExec(t, store, `UPDATE tasks SET due_date = ? WHERE id = 1`, due)
	before, after := dueTitleChange(t, store)
	const external = "2026-09-01 18:45:12.123456788"
	dueExec(t, store, `UPDATE tasks SET due_date = ? WHERE id = 1`, external)
	if err := store.RestoreTaskUndoState(context.Background(), after, before); err == nil || !strings.Contains(err.Error(), "changed since") {
		t.Fatalf("time conflict: %v", err)
	}
	assertStoredDue(t, store, external)
	current, err := store.TaskUndoState(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.UpdateTask(context.Background(), TaskDraft{ID: 1, ProjectID: 1, Title: "Must rollback", DueDate: "2026-10-02", WorkflowBucketID: 999})
	if err == nil {
		t.Fatal("expected late failure")
	}
	unchanged, err := store.TaskUndoState(context.Background(), 1)
	if err != nil || !reflect.DeepEqual(current, unchanged) {
		t.Fatalf("failed save changed state: %v %#v", err, unchanged)
	}
}

func TestUndoRejectsInvalidStoredDueTimestamp(t *testing.T) {
	for _, tc := range []struct{ name, date, stored string }{
		{"malformed", "2026-09-01", "not-a-timestamp"},
		{"different date", "2026-10-02", "2026-09-01 18:45:00"},
		{"clear with retained timestamp", "", "2026-09-01 18:45:00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := setupUndoStore(t)
			expected, err := store.TaskUndoState(context.Background(), 1)
			if err != nil {
				t.Fatal(err)
			}
			target := expected
			target.Title = "Must not persist"
			target.DueDate, target.DueDateStored = tc.date, tc.stored
			if err := store.RestoreTaskUndoState(context.Background(), expected, target); err == nil {
				t.Fatal("invalid snapshot was accepted")
			}
			actual, err := store.TaskUndoState(context.Background(), 1)
			if err != nil || !reflect.DeepEqual(actual, expected) {
				t.Fatalf("invalid snapshot changed database: %#v %v", actual, err)
			}
		})
	}
}

func TestDueDateCommentInsertFailureRollsBack(t *testing.T) {
	store := setupUndoStore(t)
	const original = "2026-09-01T18:45:12.123456789+02:00"
	dueExec(t, store, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, username TEXT);
        CREATE TABLE task_comments (id INTEGER PRIMARY KEY, comment TEXT, author_id INTEGER, task_id INTEGER, created DATETIME, updated DATETIME);
        INSERT INTO users VALUES (1, 'Local', 'local');
        CREATE TRIGGER reject_comment BEFORE INSERT ON task_comments BEGIN SELECT RAISE(ABORT, 'injected comment failure'); END;`)
	dueExec(t, store, `UPDATE tasks SET due_date = ? WHERE id = 1`, original)
	before, err := store.TaskUndoState(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	draft := TaskDraft{ID: 1, ProjectID: 1, Title: "Must rollback", DueDate: "2026-10-02"}
	_, _, err = store.SaveTaskWithComment(context.Background(), draft, TaskCommentDraft{Text: "Rejected"})
	if err == nil || !strings.Contains(err.Error(), "injected comment failure") {
		t.Fatalf("expected post-update Comment failure, got %v", err)
	}
	after, err := store.TaskUndoState(context.Background(), 1)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("combined save did not roll back: %#v %v", after, err)
	}
	assertStoredDue(t, store, original)
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM task_comments`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed save left Comment: %d %v", count, err)
	}
}
