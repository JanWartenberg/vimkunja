package tui

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"testing"
)

func TestCompletedProjectMoveUsesDestinationDoneBucket(t *testing.T) {
	for _, withComment := range []bool{false, true} {
		t.Run(fmt.Sprint(withComment), func(t *testing.T) {
			m, db := auditEditorModel(t)
			const doneAt = "2026-09-01T10:00:00.123456789+02:00"
			_, err := db.Exec(`ALTER TABLE project_views ADD COLUMN done_bucket_id INTEGER;
                ALTER TABLE project_views ADD COLUMN position REAL;
                CREATE TABLE buckets(id INTEGER PRIMARY KEY,title TEXT,project_view_id INTEGER,position REAL);
                CREATE TABLE users(id INTEGER PRIMARY KEY,name TEXT,username TEXT);
                CREATE TABLE task_comments(id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME);
                INSERT INTO users VALUES(1,'Local','local');
                INSERT INTO projects VALUES(2,'TARGET',1);
                INSERT INTO project_views VALUES(10,1,3,1,100,101,1),(20,2,3,1,200,201,1);
                INSERT INTO buckets VALUES(100,'Source Todo',10,1),(101,'Source Done',10,2),(200,'Target Todo',20,1),(201,'Target Done',20,2);
                INSERT INTO tasks(id,project_id,title,description,priority,done,done_at,"index") VALUES(1,1,'Completed','',0,1,?,1);
                INSERT INTO task_buckets VALUES(1,10,101);
                INSERT INTO task_positions VALUES(1,10,1);`, doneAt)
			if err != nil {
				t.Fatal(err)
			}
			m.projects = []vimkunja.Project{{ID: 1, Title: "Source"}, {ID: 2, Title: "Target"}}
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Completed", Done: true})
			m.Update(m.loadWorkflowStatus(1, 1)())
			m.editor.pickerIndex = 1
			_, load := m.updateProjectPicker(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
			if load == nil {
				t.Fatal("Project picker did not load destination workflow")
			}
			m.Update(load())
			if withComment {
				m.comments = &commentsUI{composer: &commentComposer{text: "Moved"}}
				m.Update(m.saveTaskWithComment(false)())
			} else {
				m.Update(m.saveEditor(false)())
			}
			if m.lastErr != "" || m.editor.dirty() || !m.editor.draft.Done {
				t.Fatalf("move completion: %q %#v", m.lastErr, m.editor)
			}
			var bucket int64
			var gotTime string
			if err := db.QueryRow(`SELECT tb.bucket_id,CAST(t.done_at AS TEXT) FROM tasks t JOIN task_buckets tb ON tb.task_id=t.id WHERE t.id=1 AND tb.project_view_id=20`).Scan(&bucket, &gotTime); err != nil || bucket != 201 || gotTime != doneAt {
				t.Fatalf("destination bucket/time: %d %q %v", bucket, gotTime, err)
			}
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 2, Title: "Completed", Done: true})
			m.Update(m.loadWorkflowStatus(2, 1)())
			if m.lastErr != "" || m.editor.draft.WorkflowBucketID != 201 || m.editor.original.WorkflowBucketID != 201 || m.editor.dirty() {
				t.Fatalf("reopened Detail did not load Done Bucket: %q %#v", m.lastErr, m.editor)
			}
		})
	}
}
