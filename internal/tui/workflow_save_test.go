package tui

import (
	"fmt"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"testing"
)

func TestDetailWorkflowSavePreservesDone(t *testing.T) {
	for _, withComment := range []bool{false, true} {
		for _, externalBucket := range []bool{false, true} {
			t.Run(fmt.Sprintf("comment=%t/external=%t", withComment, externalBucket), func(t *testing.T) {
				m, db := auditEditorModel(t)
				const doneAt = "2026-09-01T10:00:00.123456789+02:00"
				_, err := db.Exec(`ALTER TABLE project_views ADD COLUMN done_bucket_id INTEGER;
                    ALTER TABLE project_views ADD COLUMN position REAL;
                    CREATE TABLE buckets (id INTEGER PRIMARY KEY,title TEXT,project_view_id INTEGER,position REAL,"limit" INTEGER DEFAULT 0);
                    CREATE TABLE users (id INTEGER PRIMARY KEY,name TEXT,username TEXT);
                    CREATE TABLE task_comments (id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME);
                    INSERT INTO users VALUES(1,'Local','local');
                    INSERT INTO project_views VALUES(10,1,3,1,100,0,1);
                    INSERT INTO buckets(id,title,project_view_id,position) VALUES(100,'Todo',10,1),(101,'Other',10,2);
                    INSERT INTO tasks(id,project_id,title,description,priority,done,done_at,"index") VALUES(1,1,'Before','',0,1,?,1);
                    INSERT INTO task_buckets VALUES(1,10,100);`, doneAt)
				if err != nil {
					t.Fatal(err)
				}
				m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Before", Done: true})
				m.Update(m.loadWorkflowStatus(1, 1)())
				if m.editor.original.WorkflowBucketID != 100 || m.editor.draft.WorkflowBucketID != 100 {
					t.Fatalf("workflow not loaded: %q %#v", m.lastErr, m.editor)
				}
				wantBucket := int64(100)
				if externalBucket {
					wantBucket = 101
					if _, err := db.Exec(`UPDATE task_buckets SET bucket_id=101 WHERE task_id=1`); err != nil {
						t.Fatal(err)
					}
				}
				m.editor.draft.Title = "After"
				var msg any
				if withComment {
					m.comments = &commentsUI{composer: &commentComposer{text: "Added"}}
					msg = m.saveTaskWithComment(false)()
				} else {
					msg = m.saveEditor(false)()
				}
				m.Update(msg)
				if m.lastErr != "" || !m.editor.draft.Done || m.editor.dirty() {
					t.Fatalf("completion changed Done/draft: %q %#v", m.lastErr, m.editor)
				}
				var done bool
				var gotTime string
				var bucket int64
				if err := db.QueryRow(`SELECT t.done,CAST(t.done_at AS TEXT),tb.bucket_id FROM tasks t JOIN task_buckets tb ON tb.task_id=t.id WHERE t.id=1`).Scan(&done, &gotTime, &bucket); err != nil {
					t.Fatal(err)
				}
				if !done || gotTime != doneAt || bucket != wantBucket {
					t.Fatalf("save changed completion/assignment: %v %q %d", done, gotTime, bucket)
				}
			})
		}
	}
}

func TestSaveDraftOnlyOmitsUnchangedExistingWorkflow(t *testing.T) {
	for _, tc := range []struct {
		name                                      string
		id, project, originalBucket, bucket, want int64
	}{
		{"existing loaded", 1, 1, 100, 100, 0},
		{"new task", 0, 1, 100, 100, 100},
		{"explicit change", 1, 1, 100, 101, 101},
		{"project change", 1, 2, 100, 100, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := taskEditor{original: vimkunja.TaskDraft{ProjectID: 1, WorkflowBucketID: tc.originalBucket}, draft: vimkunja.TaskDraft{ID: tc.id, ProjectID: tc.project, WorkflowBucketID: tc.bucket, LabelIDs: []int64{7}}}
			draft := e.saveDraft()
			if draft.WorkflowBucketID != tc.want || e.draft.WorkflowBucketID != tc.bucket {
				t.Fatalf("save draft mutated editor or lost workflow: %#v", draft)
			}
			draft.LabelIDs[0] = 8
			if e.draft.LabelIDs[0] != 7 {
				t.Fatal("save draft aliases editor labels")
			}
		})
	}
}
