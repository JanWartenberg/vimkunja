package tui

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

const undoHistoryLimit = 20

type undoEntry struct {
	before  vimkunja.TaskUndoState
	after   vimkunja.TaskUndoState
	changed []string
}

type undoRestoredMsg struct {
	entry *undoEntry
	redo  bool
	err   error
}

func newUndoEntry(before, after vimkunja.TaskUndoState) *undoEntry {
	changed := changedUndoFields(before, after)
	if len(changed) == 0 {
		return nil
	}
	return &undoEntry{before: before, after: after, changed: changed}
}

func changedUndoFields(before, after vimkunja.TaskUndoState) []string {
	var fields []string
	if before.ProjectID != after.ProjectID {
		fields = append(fields, "Project")
	}
	if before.Title != after.Title {
		fields = append(fields, "title")
	}
	if before.Description != after.Description {
		fields = append(fields, "Description")
	}
	if before.Priority != after.Priority {
		fields = append(fields, "priority")
	}
	if before.DueDate != after.DueDate || before.DueDateStored != after.DueDateStored {
		fields = append(fields, "due date")
	}
	if !reflect.DeepEqual(before.LabelIDs, after.LabelIDs) {
		fields = append(fields, "Labels")
	}
	if before.Done != after.Done {
		fields = append(fields, "Done")
	}
	if !reflect.DeepEqual(before.BucketAssignments, after.BucketAssignments) {
		fields = append(fields, "workflow")
	}
	return fields
}

func (m *model) trackedTaskMutation(ctx context.Context, taskID int64, mutate func() error) (*undoEntry, error) {
	before, err := m.store.TaskUndoState(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if err := mutate(); err != nil {
		return nil, err
	}
	after, err := m.store.TaskUndoState(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return newUndoEntry(before, after), nil
}

func (m *model) rememberUndo(entry *undoEntry) {
	if entry == nil {
		return
	}
	m.undoHistory = append(m.undoHistory, entry)
	m.redoHistory = nil
	for len(m.undoHistory)+len(m.redoHistory) > undoHistoryLimit {
		m.undoHistory = m.undoHistory[1:]
	}
}

func (m *model) undoKeyAllowed() bool {
	if m.editor != nil && m.editor.field == fieldDescription && m.editor.descriptionActive && m.editor.descriptionView() != nil {
		return false
	}
	if m.focus != paneTasks && m.focus != paneDetail {
		return false
	}
	if m.searchOpen || m.commandMode || m.labelUI != nil || m.comments != nil || m.inlineTitle != nil ||
		m.deleteConfirm || m.kanbanPicker || m.kanbanViewPicker || m.kanbanProvisionConfirm ||
		m.overviewPrompt != overviewPromptNone {
		return false
	}
	return m.editor == nil || m.editor.mode == editorNormal
}

func (m *model) startUndo(redo bool) tea.Cmd {
	if m.taskLoading {
		m.lastErr = "Undo/redo is unavailable while a Task write is in progress."
		return nil
	}
	if m.editor != nil && m.editor.dirty() {
		m.lastErr = "Save or discard the Task draft before undo/redo."
		return nil
	}
	history := m.undoHistory
	if redo {
		history = m.redoHistory
	}
	if len(history) == 0 {
		if redo {
			m.toast = "Nothing to redo."
		} else {
			m.toast = "Nothing to undo."
		}
		return nil
	}
	entry := history[len(history)-1]
	expected, target := entry.after, entry.before
	if redo {
		expected, target = entry.before, entry.after
	}
	m.taskLoading = true
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		err := m.store.RestoreTaskUndoState(ctx, expected, target)
		return undoRestoredMsg{entry: entry, redo: redo, err: err}
	}
}

func (m *model) applyUndoRestore(msg undoRestoredMsg) tea.Cmd {
	m.taskLoading = false
	if msg.err != nil {
		m.recordError(msg.err)
		return nil
	}
	if msg.redo {
		m.redoHistory = m.redoHistory[:len(m.redoHistory)-1]
		m.undoHistory = append(m.undoHistory, msg.entry)
	} else {
		m.undoHistory = m.undoHistory[:len(m.undoHistory)-1]
		m.redoHistory = append(m.redoHistory, msg.entry)
	}
	target := msg.entry.before
	verb := "Undid"
	if msg.redo {
		target = msg.entry.after
		verb = "Redid"
	}
	name := target.Identifier
	if name == "" {
		name = fmt.Sprintf("%s (#%d)", target.Title, target.TaskID)
	}
	m.toast = fmt.Sprintf("%s %s: %s", verb, name, strings.Join(msg.entry.changed, ", "))
	editorAffected := m.editor != nil && m.editor.id == target.TaskID
	m.refreshEditorAfterUndo(target)
	refresh := m.refreshActiveTasks()
	if !editorAffected {
		return refresh
	}
	return tea.Batch(refresh, m.loadEditorLabels(target.TaskID), m.loadUndoWorkflowStatus(target.ProjectID, target.TaskID))
}

func (m *model) refreshEditorAfterUndo(state vimkunja.TaskUndoState) {
	if m.editor == nil || m.editor.id != state.TaskID {
		return
	}
	bucketID := int64(0)
	for _, assignment := range state.BucketAssignments {
		if assignment.ViewID == m.currentViewID {
			bucketID = assignment.BucketID
			break
		}
	}
	draft := vimkunja.TaskDraft{
		ID: state.TaskID, ProjectID: state.ProjectID, Title: state.Title,
		Description: state.Description, Priority: state.Priority, DueDate: state.DueDate,
		WorkflowBucketID: bucketID, Done: state.Done,
		LabelIDs: append([]int64(nil), state.LabelIDs...),
	}
	m.editor.draft = draft
	m.editor.original = cloneTaskDraft(draft)
	m.editor.labelsLoading = true
	m.editor.workflowStatusLoading = true
}

func (m *model) loadUndoWorkflowStatus(projectID, taskID int64) tea.Cmd {
	editor := m.editor
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		status, err := m.store.GetTaskWorkflowStatus(ctx, projectID, taskID)
		return workflowStatusLoadedMsg{taskID: taskID, editor: editor, status: status, err: err}
	}
}
