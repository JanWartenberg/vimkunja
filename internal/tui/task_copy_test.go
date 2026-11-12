package tui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

func TestCopyTaskFromList(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.projects = []vimkunja.Project{{ID: 1, Title: "Vimkunja", Identifier: "VIM"}}
	m.tasks = []vimkunja.Task{{ID: 7, ProjectID: 1, Index: 42, Title: "Release", Priority: 3, DueDate: "2026-11-15", Description: "<p>Hello</p>"}}
	m.store = &fakeStore{taskLabels: map[int64][]vimkunja.Label{7: {{Title: "Docs"}}}, workflowStatus: vimkunja.WorkflowStatus{Availability: vimkunja.WorkflowStatusAvailable, CurrentBucket: vimkunja.WorkflowBucket{ID: 2, Title: "Doing"}}}
	var copied string
	m.clipboardWrite = func(s string) error { copied = s; return nil }
	_, cmd := typeCommand(t, m, "copy")
	if cmd == nil {
		t.Fatal("copy did not produce a command")
	}
	m.Update(cmd())
	want := "Release\nTask: VIM-42\nProject: Vimkunja\nStatus: Doing\nDone: No\nPriority: High\nDue: 2026-11-15\nLabels: Docs\n\nHello"
	if copied != want {
		t.Fatalf("got %q, want %q", copied, want)
	}
}

func TestCopyTaskUsesVisibleDraftAndCapturesBeforeAsyncWork(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	task := vimkunja.Task{ID: 7, ProjectID: 1, Index: 42, Title: "Saved", Description: "Old"}
	m.tasks = []vimkunja.Task{task}
	m.projects = []vimkunja.Project{{ID: 1, Title: "Work", Identifier: "WORK"}}
	m.openExistingEditor(task)
	e := m.editor
	e.labelsLoading = false
	e.workflowStatusLoading = false
	e.draft.Title = "Unsaved"
	e.draft.Description = "<p>Draft</p>"
	e.draft.Priority = 2
	e.draft.Done = true
	e.labels = []vimkunja.Label{{ID: 9, Title: "New"}}
	e.workflowStatus = vimkunja.WorkflowStatus{Availability: vimkunja.WorkflowStatusUnavailable}
	var copied string
	m.clipboardWrite = func(s string) error { copied = s; return nil }
	_, cmd := typeCommand(t, m, "copy")
	if cmd == nil {
		t.Fatal("no copy command")
	}
	e.draft.Title = "Later"
	e.labels[0].Title = "Later label"
	m.Update(cmd())
	want := "Unsaved\nTask: WORK-42\nProject: Work\nDone: Yes\nPriority: Medium\nLabels: New\n\nDraft"
	if copied != want {
		t.Fatalf("got %q, want %q", copied, want)
	}
	if e.draft.Title != "Later" || !e.dirty() || m.editor != e {
		t.Fatal("copy changed the draft")
	}
}

func TestCopyTaskContextsAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   func(*model)
		command string
		want    string
		errText string
	}{
		{name: "overview", setup: func(m *model) { m.taskScope = scopeOverview }, want: "Task\nTask: #4\nDone: No"},
		{name: "kanban", setup: func(m *model) {
			m.kanbanActive = true
			m.currentViewID = 10
			m.kanbanBoard = vimkunja.KanbanBoard{Buckets: []vimkunja.KanbanBucket{{ID: 1, Tasks: []vimkunja.Task{{ID: 2, Index: 9, Title: "Card"}}}}}
		}, want: "Card\nTask: #9\nDone: No"},
		{name: "no selection", setup: func(m *model) { m.tasks = nil }, errText: "no Task selected"},
		{name: "projects", setup: func(m *model) { m.focus = paneProjects }, errText: "requires a selected Task"},
		{name: "unsupported arguments", command: "copy -c", errText: "usage: :copy"},
		{name: "labels read fails", setup: func(m *model) { m.store = &fakeStore{labelErr: errors.New("labels unavailable")} }, errText: "labels unavailable"},
		{name: "workflow read fails", setup: func(m *model) { m.store = &fakeStore{workflowStatusErr: errors.New("status unavailable")} }, errText: "status unavailable"},
		{name: "clipboard fails", setup: func(m *model) { m.clipboardWrite = func(string) error { return errors.New("clipboard unavailable") } }, errText: "clipboard unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.focus = paneTasks
			m.tasks = []vimkunja.Task{{ID: 1, Index: 4, Title: "Task"}}
			m.store = &fakeStore{workflowStatus: vimkunja.WorkflowStatus{Availability: vimkunja.WorkflowStatusUnavailable}}
			copied := "unchanged"
			m.clipboardWrite = func(s string) error { copied = s; return nil }
			if tc.setup != nil {
				tc.setup(m)
			}
			command := tc.command
			if command == "" {
				command = "copy"
			}
			m, cmd := typeCommand(t, m, command)
			if cmd != nil {
				updated, _ := m.Update(cmd())
				m = updated.(*model)
			}
			if tc.errText != "" {
				if !strings.Contains(m.lastErr, tc.errText) {
					t.Fatalf("error %q, want %q", m.lastErr, tc.errText)
				}
				if copied != "unchanged" {
					t.Fatalf("failed copy changed clipboard: %q", copied)
				}
			} else if copied != tc.want {
				t.Fatalf("got %q want %q", copied, tc.want)
			}
		})
	}
}

func TestCopyNativeDraftPreservesSelectionAndHistory(t *testing.T) {
	m := nativeEditingModel(`<p>Hello <a href="https://example.com">site</a></p>`)
	m.editor.labelsLoading = false
	m.editor.workflowStatusLoading = false
	m = nativeKeys(m, "i", "X", "esc", "v", "l")
	before := m.editor.draft.Description
	cursor := m.editor.nativeDescription.Cursor()
	selection := m.editor.nativeDraft.Selection()
	if selection != nil {
		clone := *selection
		selection = &clone
	}
	var copied string
	m.clipboardWrite = func(s string) error { copied = s; return nil }
	m, cmd := typeCommand(t, m, "copy")
	if cmd == nil {
		t.Fatal("no copy command")
	}
	m.Update(cmd())
	if !strings.Contains(copied, "XHello site (https://example.com)") {
		t.Fatalf("copy did not include native draft: %q", copied)
	}
	if m.editor.draft.Description != before || m.editor.nativeDescription.Cursor() != cursor || !reflect.DeepEqual(m.editor.nativeDraft.Selection(), selection) {
		t.Fatal("copy changed draft or selection")
	}
	if !m.editor.nativeDraft.Undo() {
		t.Fatal("copy lost undo history")
	}
}

func TestCopyNewTaskAndUnrelatedDraft(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 7, Title: "Selected"}}
	m.openNewEditor()
	m.editor.draft.Title = "New draft"
	var copied string
	m.clipboardWrite = func(s string) error { copied = s; return nil }
	m, cmd := typeCommand(t, m, "copy")
	if cmd == nil {
		t.Fatal("new draft copy unavailable")
	}
	m.Update(cmd())
	if copied != "New draft\nDone: No" {
		t.Fatal(copied)
	}
	m.focus = paneTasks
	m, cmd = typeCommand(t, m, "copy")
	if cmd == nil {
		t.Fatal("selected Task copy unavailable")
	}
	m.Update(cmd())
	if copied != "Selected\nDone: No" || m.editor.draft.Title != "New draft" {
		t.Fatalf("copied %q, draft %q", copied, m.editor.draft.Title)
	}
}

func TestCopyRejectsFailedDetailMetadataAndRecoversAfterReload(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 7, ProjectID: 1, Title: "Task"})
	m.Update(taskLabelsLoadedMsg{taskID: 7})
	m.Update(workflowStatusLoadedMsg{taskID: 7, editor: m.editor, err: errors.New("workflow read failed")})
	copied := "unchanged"
	m.clipboardWrite = func(s string) error { copied = s; return nil }
	m, cmd := typeCommand(t, m, "copy")
	if cmd != nil {
		m.Update(cmd())
	}
	if copied != "unchanged" || !strings.Contains(m.lastErr, "workflow read failed") {
		t.Fatalf("copy after failed load: clipboard %q, error %q", copied, m.lastErr)
	}
	m.Update(workflowStatusLoadedMsg{taskID: 7, editor: m.editor, status: vimkunja.WorkflowStatus{Availability: vimkunja.WorkflowStatusAvailable, CurrentBucket: vimkunja.WorkflowBucket{ID: 2, Title: "Doing"}, Buckets: []vimkunja.WorkflowBucket{{ID: 2, Title: "Doing"}}}})
	m, cmd = typeCommand(t, m, "copy")
	if cmd == nil {
		t.Fatal("copy did not recover after successful load")
	}
	m.Update(cmd())
	if !strings.Contains(copied, "Status: Doing") {
		t.Fatal(copied)
	}
}

func TestCopyEmptyNewDraftOmitsEmptyTitle(t *testing.T) {
	m := testModel()
	m.openNewEditor()
	copied := "unchanged"
	m.clipboardWrite = func(s string) error { copied = s; return nil }
	m, cmd := typeCommand(t, m, "copy")
	if cmd == nil {
		t.Fatal("empty draft copy unavailable")
	}
	m.Update(cmd())
	if copied != "Done: No" {
		t.Fatalf("empty fields left an empty line: %q", copied)
	}
}
