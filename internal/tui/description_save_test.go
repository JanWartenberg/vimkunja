package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

func TestEditorSaveCarriesOpenedDescriptionBaseline(t *testing.T) {
	e := taskEditor{original: vimkunja.TaskDraft{ID: 1, Description: "<p>Original</p>"}, draft: vimkunja.TaskDraft{ID: 1, Description: "Local"}}
	draft := e.saveDraft()
	if draft.DescriptionBaseline == nil || *draft.DescriptionBaseline != "<p>Original</p>" {
		t.Fatal("opened baseline missing")
	}
	*draft.DescriptionBaseline = "Changed"
	if e.original.Description != "<p>Original</p>" {
		t.Fatal("save draft aliases opened source")
	}
	e.draft.ID = 0
	if e.saveDraft().DescriptionBaseline != nil {
		t.Fatal("new Task has existing baseline")
	}
}

func TestCombinedSaveAdoptsCommittedCandidateAndKeepsLateDescription(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(fmt.Sprint(late), func(t *testing.T) {
			m, db := auditEditorModel(t)
			if _, err := db.Exec(`CREATE TABLE users(id INTEGER PRIMARY KEY,name TEXT,username TEXT);INSERT INTO users VALUES(1,'Local','local');CREATE TABLE task_comments(id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME);INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task','',0,0,1)`); err != nil {
				t.Fatal(err)
			}
			doc := description.Parse("")
			nodes := []description.Node{{Kind: description.Element, Tag: "ul", Attributes: []description.Attribute{{Name: "data-type", Value: "taskList"}}, Children: []description.Node{{Kind: description.Element, Tag: "li", Attributes: []description.Attribute{{Name: "data-type", Value: "taskItem"}, {Name: "data-checked", Value: "false"}}, Children: []description.Node{{Kind: description.Text, Text: "Entry"}}}}}}
			candidate, err := doc.PrepareSave(nodes, description.Preservation{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			m.editor.id = 1
			m.editor.draft.ID = 1
			m.editor.draft.Title = "Task"
			m.editor.original = cloneTaskDraft(m.editor.draft)
			m.editor.draft.Description = "Local projection"
			m.editor.draft.DescriptionCandidate = &candidate
			m.comments = &commentsUI{composer: &commentComposer{text: "Comment"}}
			save := m.saveTaskWithComment(false)
			if late {
				m.editor.draft.Description = "Later"
				m.editor.draft.DescriptionCandidate = nil
			}
			msg := save().(taskCommentSavedMsg)
			if msg.err != nil {
				t.Fatal(msg.err)
			}
			m.Update(msg)
			if m.editor.original.Description != candidate.HTML() {
				t.Fatalf("baseline not committed candidate: %q", m.editor.original.Description)
			}
			want := candidate.HTML()
			if late {
				want = "Later"
			}
			if m.editor.draft.Description != want || m.editor.dirty() != late {
				t.Fatalf("adoption lost late draft: %#v", m.editor)
			}
		})
	}
}

// A history read is a separate persistence boundary and may fail after the
// Task commit; this adapter injects that boundary failure without simulating SQL.
type historyReadFailureStore struct {
	vimkunja.Store
	committed bool
}

func (s *historyReadFailureStore) UpdateTask(ctx context.Context, draft vimkunja.TaskDraft) (vimkunja.Task, error) {
	task, err := s.Store.UpdateTask(ctx, draft)
	s.committed = err == nil
	return task, err
}
func (s *historyReadFailureStore) TaskUndoState(ctx context.Context, id int64) (vimkunja.TaskUndoState, error) {
	if s.committed {
		return vimkunja.TaskUndoState{}, fmt.Errorf("history unavailable")
	}
	return s.Store.TaskUndoState(ctx, id)
}

func TestCommittedDescriptionAdoptedEvenWhenHistoryReadFails(t *testing.T) {
	m, db := auditEditorModel(t)
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task','<p>Original</p>',0,0,1)`); err != nil {
		t.Fatal(err)
	}
	doc := description.Parse("<p>Original</p>")
	nodes := doc.Snapshot()
	nodes[0].Children[0].Text = "Local"
	candidate, err := doc.PrepareSave(nodes, description.Preservation{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.editor.id = 1
	m.editor.draft.ID = 1
	m.editor.draft.Title = "Task"
	m.editor.draft.Description = doc.SourceHTML()
	m.editor.original = cloneTaskDraft(m.editor.draft)
	m.editor.draft.Description = "Local projection"
	m.editor.draft.DescriptionCandidate = &candidate
	m.store = &historyReadFailureStore{Store: m.store}
	msg := m.saveEditor(false)().(taskSavedMsg)
	m.Update(msg)
	if m.editor.original.Description != candidate.HTML() || m.editor.draft.Description != candidate.HTML() || m.editor.dirty() {
		t.Fatalf("committed candidate not adopted: %#v", m.editor)
	}
	if !strings.Contains(m.lastErr, "history unavailable") {
		t.Fatal("history failure not reported")
	}
}

func TestLateNativeCandidateCanSaveAfterOwnCommit(t *testing.T) {
	m, db := auditEditorModel(t)
	const source = `<ul data-type="taskList"><li data-type="taskItem" data-task-id="stable" data-checked="false">Original</li></ul>`
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task',?,0,0,1)`, source); err != nil {
		t.Fatal(err)
	}
	doc := description.Parse(source)
	nodes := doc.Snapshot()
	nodes[0].Children[0].Children[0].Text = "First"
	first, err := doc.PrepareSave(nodes, description.Preservation{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.editor.id = 1
	m.editor.draft.ID = 1
	m.editor.draft.Title = "Task"
	m.editor.draft.Description = source
	m.editor.original = cloneTaskDraft(m.editor.draft)
	m.editor.draft.Description = "First projection"
	m.editor.draft.DescriptionCandidate = &first
	save := m.saveEditor(false)
	nodes[0].Children[0].Children[0].Text = "Late"
	late, err := doc.PrepareSave(nodes, description.Preservation{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.editor.draft.Description = "Late projection"
	m.editor.draft.DescriptionCandidate = &late
	msg := save().(taskSavedMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	m.Update(msg)
	if !m.editor.dirty() || m.editor.draft.Description != "Late projection" {
		t.Fatal("late draft lost")
	}
	retry := m.saveEditor(false)().(taskSavedMsg)
	if retry.err != nil {
		t.Fatalf("late save conflicts with own commit: %v", retry.err)
	}
	m.Update(retry)
	if m.editor.dirty() || m.editor.draft.Description != late.HTML() {
		t.Fatal("late candidate not adopted")
	}
}

func TestEditorExternalConflictRetainsTaskAndCommentDrafts(t *testing.T) {
	for _, combined := range []bool{false, true} {
		t.Run(fmt.Sprint(combined), func(t *testing.T) {
			m, db := auditEditorModel(t)
			if _, err := db.Exec(`CREATE TABLE users(id INTEGER PRIMARY KEY,name TEXT,username TEXT);INSERT INTO users VALUES(1,'Local','local');CREATE TABLE task_comments(id INTEGER PRIMARY KEY,comment TEXT,author_id INTEGER,task_id INTEGER,created DATETIME,updated DATETIME);INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task','<p>Original</p>',0,0,1)`); err != nil {
				t.Fatal(err)
			}
			m.editor.id = 1
			m.editor.draft.ID = 1
			m.editor.draft.Title = "Task"
			m.editor.draft.Description = "<p>Original</p>"
			m.editor.original = cloneTaskDraft(m.editor.draft)
			m.editor.draft.Title = "Local title"
			m.editor.draft.Description = "Local description"
			m.comments = &commentsUI{composer: &commentComposer{text: "Local comment"}}
			if _, err := db.Exec(`UPDATE tasks SET description='<p>Remote</p>' WHERE id=1`); err != nil {
				t.Fatal(err)
			}
			if combined {
				m.Update(m.saveTaskWithComment(true)())
			} else {
				m.Update(m.saveEditor(true)())
			}
			if m.editor == nil || m.editor.draft.Description != "Local description" || m.editor.original.Description != "<p>Original</p>" || m.comments.composer == nil || m.comments.composer.text != "Local comment" || !strings.Contains(m.lastErr, "changed externally") {
				t.Fatal("conflict discarded drafts or closed editor")
			}
			var title, source string
			if err := db.QueryRow(`SELECT title,description FROM tasks WHERE id=1`).Scan(&title, &source); err != nil {
				t.Fatal(err)
			}
			if title != "Task" || source != "<p>Remote</p>" {
				t.Fatal("partial conflict save")
			}
		})
	}
}
