package vimkunja

import (
	"context"
	"fmt"
	"testing"
)

func TestIntegrityAuditTaskDeletionCleansPolymorphicRelations(t *testing.T) {
	store := setupUndoStore(t)
	// Enum values mirror the local Vikunja models: Task Favorite/Subscription
	// are 1/3; Task/Comment Reactions are 0/1. Keep unrelated entities as controls.
	bucketExec(t, store, `CREATE TABLE favorites (entity_id INTEGER, kind INTEGER);
		CREATE TABLE subscriptions (entity_id INTEGER, entity_type INTEGER);
		CREATE TABLE reactions (entity_id INTEGER, entity_kind INTEGER);
		CREATE TABLE task_comments (id INTEGER PRIMARY KEY, task_id INTEGER);
		INSERT INTO task_comments VALUES (55, 1), (56, 2);
		INSERT INTO favorites VALUES (1, 1), (2, 1);
		INSERT INTO subscriptions VALUES (1, 3), (2, 3);
		INSERT INTO reactions VALUES (1, 0), (55, 1), (2, 0), (56, 1);`)
	if err := store.DeleteTask(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"favorites", "subscriptions"} {
		if count := bucketInt(t, store, "SELECT COUNT(*) FROM "+table+" WHERE entity_id = 1"); count != 0 {
			t.Errorf("Task deletion left %d dangling %s rows", count, table)
		}
		if count := bucketInt(t, store, "SELECT COUNT(*) FROM "+table+" WHERE entity_id = 2"); count != 1 {
			t.Errorf("Task deletion damaged unrelated %s rows", table)
		}
	}
	if count := bucketInt(t, store, `SELECT COUNT(*) FROM reactions WHERE (entity_id = 1 AND entity_kind = 0) OR (entity_id = 55 AND entity_kind = 1)`); count != 0 {
		t.Errorf("Task deletion left %d dangling Task/Comment reactions", count)
	}
	if count := bucketInt(t, store, `SELECT COUNT(*) FROM reactions WHERE entity_id IN (2, 56)`); count != 2 {
		t.Error("Task deletion damaged unrelated Reactions")
	}
}

func TestPolymorphicDeletionScopeAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			store := setupUndoStore(t)
			ctx := context.Background()
			bucketExec(t, store, `CREATE TABLE favorites(entity_id INTEGER,kind INTEGER); CREATE TABLE subscriptions(entity_id INTEGER,entity_type INTEGER); CREATE TABLE reactions(entity_id INTEGER,entity_kind INTEGER); CREATE TABLE task_comments(id INTEGER PRIMARY KEY,task_id INTEGER); CREATE TABLE files(id INTEGER PRIMARY KEY); CREATE TABLE task_attachments(task_id INTEGER,file_id INTEGER);
    INSERT INTO task_comments VALUES(55,1),(56,2);
    INSERT INTO favorites VALUES(1,1),(1,2),(2,1);
    INSERT INTO subscriptions VALUES(1,3),(1,2),(2,3);
    INSERT INTO reactions VALUES(1,0),(1,1),(55,1),(55,0),(56,1),(2,0);
    INSERT INTO files VALUES(7); INSERT INTO task_attachments VALUES(1,7);`)
			if fail {
				bucketExec(t, store, `CREATE TRIGGER reject_task_delete BEFORE DELETE ON tasks BEGIN SELECT RAISE(ABORT,'late delete failure'); END;`)
			}
			err := store.DeleteTask(ctx, 1)
			if fail {
				if err == nil {
					t.Fatal("injected deletion failure accepted")
				}
				for table, want := range map[string]int64{"tasks": 1, "favorites": 3, "subscriptions": 3, "reactions": 6, "task_comments": 2, "task_attachments": 1, "task_buckets": 1, "label_tasks": 1} {
					if got := bucketInt(t, store, "SELECT COUNT(*) FROM "+table); got != want {
						t.Fatalf("rollback %s=%d want %d", table, got, want)
					}
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				for table, want := range map[string]int64{"favorites": 2, "subscriptions": 2, "reactions": 4, "task_comments": 1, "task_attachments": 0} {
					if got := bucketInt(t, store, "SELECT COUNT(*) FROM "+table); got != want {
						t.Fatalf("scope %s=%d want %d", table, got, want)
					}
				}
				if bucketInt(t, store, `SELECT COUNT(*) FROM favorites WHERE entity_id=1 AND kind=2`) != 1 || bucketInt(t, store, `SELECT COUNT(*) FROM reactions WHERE entity_id=55 AND entity_kind=0`) != 1 {
					t.Fatal("damaged same-ID other kind")
				}
			}
			if bucketInt(t, store, `SELECT COUNT(*) FROM files`) != 1 {
				t.Fatal("attachment file removed")
			}
		})
	}
}

func TestPolymorphicDeletionHistoricalTables(t *testing.T) {
	for _, table := range []string{"favorites", "subscriptions", "reactions"} {
		t.Run(table, func(t *testing.T) {
			store := setupUndoStore(t)
			column := map[string]string{"favorites": "kind", "subscriptions": "entity_type", "reactions": "entity_kind"}[table]
			kind := map[string]int{"favorites": 1, "subscriptions": 3, "reactions": 0}[table]
			bucketExec(t, store, "CREATE TABLE "+table+" (entity_id INTEGER,"+column+" INTEGER)")
			bucketExec(t, store, "INSERT INTO "+table+" VALUES(1,?)", kind)
			if err := store.DeleteTask(context.Background(), 1); err != nil {
				t.Fatal(err)
			}
			if bucketInt(t, store, "SELECT COUNT(*) FROM "+table) != 0 {
				t.Fatal("historical cleanup failed")
			}
		})
	}
}
