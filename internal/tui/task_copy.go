package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/atotto/clipboard"
)

// copyTaskCommand captures the visible values before any asynchronous reads.
// Only missing labels/workflow are read; copying never saves or reloads a draft.
func copyTaskCommand(m *model, args []string) (tea.Model, tea.Cmd) {
	if len(args) != 0 {
		m.recordError(fmt.Errorf("usage: :copy (comments are not included)"))
		return m, nil
	}
	if m.helpPane != nil || m.infoPane != nil || m.searchOpen || m.labelUI != nil || m.comments != nil || (m.focus != paneTasks && m.focus != paneDetail) || m.taskLoading {
		m.recordError(fmt.Errorf(":copy requires a selected Task in List, Overview, Kanban or Detail"))
		return m, nil
	}
	task, ok := m.selectedTask()
	e := m.editor
	useDraft := m.showDetail && e != nil && (m.focus == paneDetail || ok && task.ID == e.id)
	var labels []vimkunja.Label
	var workflow vimkunja.WorkflowStatus
	loadLabels, loadWorkflow := true, true
	if useDraft {
		if e.workflowStatusErr != nil {
			m.recordError(fmt.Errorf("cannot copy Task status: %w", e.workflowStatusErr))
			return m, nil
		}
		if e.labelsLoading || e.workflowStatusLoading {
			m.recordError(fmt.Errorf("Task metadata is still loading; retry :copy"))
			return m, nil
		}
		task = vimkunja.Task{ID: e.id, Index: e.taskIndex, ProjectID: e.draft.ProjectID, Title: e.draft.Title, Description: e.draft.Description, Priority: e.draft.Priority, DueDate: e.draft.DueDate, Done: e.draft.Done}
		labels = append([]vimkunja.Label(nil), e.labels...)
		workflow = e.workflowStatus
		workflow.CurrentBucket = vimkunja.WorkflowBucket{}
		for _, b := range workflow.Buckets {
			if b.ID == e.draft.WorkflowBucketID {
				workflow.CurrentBucket = b
				break
			}
		}
		loadLabels, loadWorkflow = false, false
		ok = true
	}
	if !ok {
		m.recordError(fmt.Errorf("no Task selected"))
		return m, nil
	}
	for _, p := range m.projects {
		if p.ID == task.ProjectID {
			task.ProjectTitle = p.Title
			task.ProjectIdentifier = p.Identifier
			break
		}
	}
	if useDraft {
		task.Identifier = ""
	}
	if task.Identifier == "" && task.Index > 0 {
		task.Identifier = fmt.Sprintf("#%d", task.Index)
		if task.ProjectIdentifier != "" {
			task.Identifier = fmt.Sprintf("%s-%d", task.ProjectIdentifier, task.Index)
		}
	}
	text := task.Description
	if useDraft && e.nativeDraft != nil {
		text = e.nativeDraft.PlainText()
	} else {
		d := description.NewTextDraft(text)
		if descriptionHasHTML(text) || descriptionHasUnterminatedMarkup(text) || strings.Contains(text, "<!--") {
			d = description.NewDraft(description.Parse(text))
		}
		text = d.PlainText()
	}
	store, write := m.store, m.clipboardWrite
	if write == nil {
		write = clipboard.WriteAll
	}
	viewID := int64(0)
	if m.kanbanActive && !m.inOverview() {
		viewID = m.currentViewID
	}
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		if loadLabels {
			var err error
			labels, err = store.ListTaskLabels(ctx, task.ID)
			if err != nil {
				return textCopiedMsg{err: fmt.Errorf("load Task labels: %w", err)}
			}
		}
		if loadWorkflow {
			var err error
			workflow, err = store.GetTaskWorkflowStatus(ctx, task.ProjectID, task.ID, viewID)
			if err != nil {
				return textCopiedMsg{err: fmt.Errorf("load Task status: %w", err)}
			}
		}
		return textCopiedMsg{err: write(taskShareText(task, labels, workflow, text))}
	}
}

func taskShareText(task vimkunja.Task, labels []vimkunja.Label, workflow vimkunja.WorkflowStatus, body string) string {
	var lines []string
	if strings.TrimSpace(task.Title) != "" {
		lines = append(lines, task.Title)
	}
	add := func(name, value string) {
		if strings.TrimSpace(value) != "" {
			lines = append(lines, name+": "+value)
		}
	}
	add("Task", task.Identifier)
	add("Project", task.ProjectTitle)
	if workflow.Availability == vimkunja.WorkflowStatusAvailable {
		add("Status", workflow.CurrentBucket.Title)
	}
	done := "No"
	if task.Done {
		done = "Yes"
	}
	add("Done", done)
	if task.Priority != 0 {
		add("Priority", priorityName(task.Priority))
	}
	add("Due", task.DueDate)
	var titles []string
	for _, l := range labels {
		if strings.TrimSpace(l.Title) != "" {
			titles = append(titles, l.Title)
		}
	}
	add("Labels", strings.Join(titles, ", "))
	if strings.TrimSpace(body) != "" {
		lines = append(lines, "", body)
	}
	return strings.Join(lines, "\n")
}
