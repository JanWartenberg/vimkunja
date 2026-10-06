package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

func TestUndoRedoRestoresEntryAndMovesBetweenStacks(t *testing.T) {
	before := vimkunja.TaskUndoState{TaskID: 1, ProjectID: 1, Identifier: "VIM-1", Title: "Before"}
	after := before
	after.Title = "After"
	store := &fakeStore{
		undoStates:  map[int64]vimkunja.TaskUndoState{1: after},
		tasksByView: map[int64][]vimkunja.Task{10: {{ID: 1, ProjectID: 1, Title: "After"}}},
	}
	m := NewModel(Config{Store: store}).(*model)
	m.focus, m.currentProjectID, m.currentViewID = paneTasks, 1, 10
	m.tasks = append([]vimkunja.Task(nil), store.tasksByView[10]...)
	m.taskSelected = 0
	m.undoHistory = []*undoEntry{newUndoEntry(before, after)}

	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'u', Text: "u"}))
	m = updated.(*model)
	if cmd == nil || !m.taskLoading {
		t.Fatal("u did not start restore")
	}
	updated, refresh := m.Update(cmd())
	m = updated.(*model)
	if got := store.undoState(1).Title; got != "Before" {
		t.Fatalf("undo title = %q", got)
	}
	if len(m.undoHistory) != 0 || len(m.redoHistory) != 1 || refresh == nil {
		t.Fatalf("undo stacks = %d/%d, refresh=%v", len(m.undoHistory), len(m.redoHistory), refresh != nil)
	}
	if m.focus != paneTasks || m.taskSelected != 0 || !strings.Contains(m.toast, "Undid VIM-1: title") {
		t.Fatalf("undo changed UI context or feedback: focus=%v selected=%d toast=%q", m.focus, m.taskSelected, m.toast)
	}
	updated, _ = m.Update(refresh())
	m = updated.(*model)

	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'r', Mod: tea.ModCtrl}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("Ctrl-R did not start redo")
	}
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if got := store.undoState(1).Title; got != "After" || len(m.undoHistory) != 1 || len(m.redoHistory) != 0 {
		t.Fatalf("redo state = title:%q stacks:%d/%d", got, len(m.undoHistory), len(m.redoHistory))
	}
}

func TestUndoFailureKeepsHistoryAndLogsError(t *testing.T) {
	before := vimkunja.TaskUndoState{TaskID: 1, ProjectID: 1, Title: "Before"}
	after := before
	after.Title = "After"
	store := &fakeStore{undoStates: map[int64]vimkunja.TaskUndoState{1: after}, restoreUndoErr: errors.New("database is locked")}
	m := NewModel(Config{Store: store}).(*model)
	m.focus = paneTasks
	m.undoHistory = []*undoEntry{newUndoEntry(before, after)}

	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'u', Text: "u"}))
	m = updated.(*model)
	updated, refresh := m.Update(cmd())
	m = updated.(*model)
	if refresh != nil || len(m.undoHistory) != 1 || len(m.redoHistory) != 0 || m.lastErr != "database is locked" {
		t.Fatalf("failed undo = refresh:%v stacks:%d/%d error:%q", refresh != nil, len(m.undoHistory), len(m.redoHistory), m.lastErr)
	}
}

func TestUndoBlockedByDirtyDraftAndNewActionClearsRedo(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.editor = &taskEditor{id: 1, original: vimkunja.TaskDraft{ID: 1, ProjectID: 1, Title: "Before"}, draft: vimkunja.TaskDraft{ID: 1, ProjectID: 1, Title: "Draft"}}
	m.undoHistory = []*undoEntry{{}}
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'u', Text: "u"}))
	m = updated.(*model)
	if cmd != nil || !strings.Contains(m.lastErr, "Save or discard") {
		t.Fatalf("dirty undo = cmd:%v error:%q", cmd != nil, m.lastErr)
	}

	m.redoHistory = []*undoEntry{{}}
	before := vimkunja.TaskUndoState{TaskID: 2, ProjectID: 1, Title: "A"}
	after := before
	after.Done = true
	m.rememberUndo(newUndoEntry(before, after))
	if len(m.redoHistory) != 0 {
		t.Fatal("new Task action did not clear redo history")
	}
}

func TestUndoHistoryHasCombinedLimitAndIgnoresNoOp(t *testing.T) {
	m := testModel()
	state := vimkunja.TaskUndoState{TaskID: 1, ProjectID: 1, Title: "same"}
	m.rememberUndo(newUndoEntry(state, state))
	if len(m.undoHistory) != 0 {
		t.Fatal("no-op created an Undo Entry")
	}
	for i := 0; i < undoHistoryLimit+3; i++ {
		before := state
		after := state
		before.Title = string(rune('a' + i))
		after.Title = string(rune('A' + i))
		m.rememberUndo(newUndoEntry(before, after))
	}
	if len(m.undoHistory) != undoHistoryLimit {
		t.Fatalf("history length = %d, want %d", len(m.undoHistory), undoHistoryLimit)
	}
}
