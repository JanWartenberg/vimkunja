package tui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
)

// beginTaskDeletion keeps the destructive-action state behind a small model
// seam: callers only arm it from the Tasks Pane, while confirmation, Store
// persistence, error feedback, and refresh remain centralized here.
func (m *model) beginTaskDeletion() {
	if m.taskSelected < 0 || m.taskSelected >= len(m.tasks) {
		return
	}
	m.deleteConfirm = true
	m.deleteTaskID = m.tasks[m.taskSelected].ID
}

func (m *model) updateDeleteConfirmation(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		taskID := m.deleteTaskID
		m.taskLoading = true
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			return taskDeletedMsg{err: m.store.DeleteTask(ctx, taskID)}
		}
	case "n", "esc":
		m.deleteConfirm = false
		m.deleteTaskID = 0
	}
	return m, nil
}

func (m *model) deleteConfirmationLabel() string {
	for _, task := range m.tasks {
		if task.ID == m.deleteTaskID {
			return fmt.Sprintf("Delete %q? y yes / n or Esc cancel", truncatePlain(task.Title, 36))
		}
	}
	return "Delete selected task? y yes / n or Esc cancel"
}
