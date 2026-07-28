package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

func (m *model) deleteConfirmationTaskTitle() string {
	for _, task := range m.tasks {
		if task.ID == m.deleteTaskID {
			return truncatePlain(task.Title, 46)
		}
	}
	return "selected task"
}

// renderDeleteConfirmationModal overlays a compact destructive-action dialog
// over the current Pane content while preserving the surrounding context.
func (m *model) renderDeleteConfirmationModal(background string, width int) string {
	// The modal's border and horizontal padding consume six cells. Truncate
	// variable content before styling so narrow terminals never overflow.
	contentWidth := max(1, width-6)
	taskTitle := truncatePlain(m.deleteConfirmationTaskTitle(), contentWidth-lipgloss.Width(`Delete ""?`))
	body := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render(truncatePlain("DELETE TASK", contentWidth)),
		"",
		fmt.Sprintf("Delete %q?", taskTitle),
		"",
		truncatePlain("y delete  •  n / Esc cancel", contentWidth),
	}, "\n")
	modal := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("255")).
		Padding(1, 2).
		Width(width).
		Render(body)
	modalLines := strings.Split(modal, "\n")
	backgroundLines := strings.Split(background, "\n")
	modalWidth := lipgloss.Width(modalLines[0])
	left := max(0, (width-modalWidth)/2)
	top := max(0, (len(backgroundLines)-len(modalLines))/2)

	for i, modalLine := range modalLines {
		row := top + i
		if row >= len(backgroundLines) {
			break
		}
		backgroundLine := backgroundLines[row]
		if padding := width - lipgloss.Width(backgroundLine); padding > 0 {
			backgroundLine += strings.Repeat(" ", padding)
		}
		backgroundLines[row] = ansi.Cut(backgroundLine, 0, left) + modalLine + ansi.Cut(backgroundLine, left+modalWidth, width)
	}
	return strings.Join(backgroundLines, "\n")
}
