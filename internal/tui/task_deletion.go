package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// beginTaskDeletion keeps the destructive-action state behind a small model
// seam: callers only arm it from the Tasks Pane, while confirmation, Store
// persistence, error feedback, and refresh remain centralized here.
func (m *model) beginTaskDeletion() {
	task, ok := m.selectedTaskForDeletion()
	if !ok {
		return
	}
	if m.kanbanActive && !m.inOverview() {
		bucket, _ := m.activeKanbanBucket()
		m.pendingKanbanBucketID = bucket.ID
		m.pendingKanbanTaskID = 0
		m.pendingKanbanEmptySelection = false
		if m.kanbanTaskSelected > 0 {
			m.pendingKanbanTaskID = bucket.Tasks[m.kanbanTaskSelected-1].ID
		} else if m.kanbanTaskSelected+1 < len(bucket.Tasks) {
			m.pendingKanbanTaskID = bucket.Tasks[m.kanbanTaskSelected+1].ID
		} else {
			m.pendingKanbanEmptySelection = true
		}
	}
	m.deleteConfirm = true
	m.deleteTaskID = task.ID
	m.confirmationScroll = 0
	m.confirmationGArmed = false
}

func (m *model) selectedTaskForDeletion() (vimkunja.Task, bool) {
	if m.kanbanActive && !m.inOverview() {
		return m.selectedKanbanTask()
	}
	if m.taskSelected < 0 || m.taskSelected >= len(m.tasks) {
		return vimkunja.Task{}, false
	}
	return m.tasks[m.taskSelected], true
}

func (m *model) updateDeleteConfirmation(key string) (tea.Model, tea.Cmd) {
	if m.updateConfirmationScroll(key, m.deleteConfirmationScrollLimit()) {
		return m, nil
	}
	switch key {
	case "y":
		taskID := m.deleteTaskID
		m.taskLoading = true
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			return taskDeletedMsg{taskID: taskID, err: m.store.DeleteTask(ctx, taskID)}
		}
	case "n", "esc":
		m.deleteConfirm = false
		m.deleteTaskID = 0
		m.pendingKanbanTaskID = 0
		m.pendingKanbanBucketID = 0
		m.pendingKanbanEmptySelection = false
		m.confirmationScroll = 0
		m.confirmationGArmed = false
	}
	return m, nil
}

func (m *model) deleteConfirmationTaskTitle() string {
	if m.kanbanActive && !m.inOverview() {
		for _, bucket := range m.kanbanBoard.Buckets {
			for _, task := range bucket.Tasks {
				if task.ID == m.deleteTaskID {
					return task.Title
				}
			}
		}
	}
	for _, task := range m.tasks {
		if task.ID == m.deleteTaskID {
			return task.Title
		}
	}
	return "selected task"
}

const deleteConfirmationModalMinWidth = 50

func deleteConfirmationModalWidth(viewportWidth int, title string) int {
	// Expand after a resize when it reveals more of the confirmation, while
	// retaining a readable cap on very wide terminals.
	preferred := max(deleteConfirmationModalMinWidth, lipgloss.Width(`Delete ""?`)+lipgloss.Width(title)+6)
	return min(min(80, preferred), viewportWidth)
}

// renderDeleteConfirmationModal overlays a wrapped destructive-action dialog
// over the current Pane content. Narrow terminals use the shared text-only
// fallback while keeping the answer row visible.
func (m *model) deleteConfirmationScrollLimit() int {
	width, _ := m.viewportSize()
	layout := modalBounds(width, m.modalContentHeight(), deleteConfirmationModalWidth(width, m.deleteConfirmationTaskTitle()), 10)
	_, limit := confirmationBody("", fmt.Sprintf("Delete %q?", m.deleteConfirmationTaskTitle()), "", layout, 0)
	return limit
}

func (m *model) renderDeleteConfirmationModal(background string, width int) string {
	viewportHeight := len(strings.Split(background, "\n"))
	layout := modalBounds(width, viewportHeight, deleteConfirmationModalWidth(width, m.deleteConfirmationTaskTitle()), 10)
	question := fmt.Sprintf("Delete %q?", m.deleteConfirmationTaskTitle())
	body, _ := confirmationBody(
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.danger)).Render("DELETE TASK"),
		question,
		"y delete  •  n / Esc cancel",
		layout,
		m.confirmationScroll,
	)
	modal := renderModalFrame(body, layout)
	return overlayModalLayout(background, modal, width, layout.fallback)
}
