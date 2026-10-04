package vimkunja

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestIntegrityAuditMissingTargetProject(t *testing.T) {
	for _, withComment := range []bool{false, true} {
		for _, targetID := range []int64{999, 2} {
			t.Run(fmt.Sprintf("comment=%t/target=%d", withComment, targetID), func(t *testing.T) {
				store := setupUndoStore(t)
				_, err := store.db.Exec(`
					CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, username TEXT);
					CREATE TABLE task_comments (id INTEGER PRIMARY KEY, comment TEXT, author_id INTEGER, task_id INTEGER, created DATETIME, updated DATETIME);
					INSERT INTO users VALUES (1, 'Local', 'local');
					INSERT INTO task_comments VALUES (77, '<p>Existing</p>', 1, 1, '2026-09-01 10:00:00', '2026-09-01 10:00:00');
				`)
				if err != nil {
					t.Fatal(err)
				}
				// Target 999 never existed; target 2 was present when the draft
				// was opened but disappeared before saving. Its stale View is
				// deliberately retained: View rows do not prove Project existence.
				if targetID == 2 {
					if _, err := store.db.Exec(`DELETE FROM projects WHERE id = 2`); err != nil {
						t.Fatal(err)
					}
				}
				before, err := store.TaskUndoState(context.Background(), 1)
				if err != nil {
					t.Fatal(err)
				}
				draft := TaskDraft{ID: 1, ProjectID: targetID, Title: "Moved", Description: "Changed",
					Priority: 5, DueDate: "2026-12-31", LabelIDsSet: true, LabelIDs: []int64{8}}
				if withComment {
					_, _, err = store.SaveTaskWithComment(context.Background(), draft, TaskCommentDraft{Text: "Must not append"})
				} else {
					_, err = store.UpdateTask(context.Background(), draft)
				}
				if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("project %d not found", targetID)) {
					t.Errorf("save to missing Project = %v, want explicit destination error", err)
				}
				after, err := store.TaskUndoState(context.Background(), 1)
				if err != nil {
					t.Fatalf("failed save orphaned Task: %v", err)
				}
				if !reflect.DeepEqual(after, before) {
					t.Fatalf("failed save changed Task/Labels/Buckets: before=%#v after=%#v", before, after)
				}
				var viewID int64
				var position float64
				if err := store.db.QueryRow(`SELECT project_view_id, position FROM task_positions WHERE task_id = 1`).Scan(&viewID, &position); err != nil || viewID != 10 || position != 1 {
					t.Fatalf("failed save changed source position: view=%d position=%v err=%v", viewID, position, err)
				}
				var comments int
				if err := store.db.QueryRow(`SELECT COUNT(*) FROM task_comments WHERE task_id = 1`).Scan(&comments); err != nil || comments != 1 {
					t.Fatalf("failed save changed Comments: count=%d err=%v", comments, err)
				}
			})
		}
	}
}
