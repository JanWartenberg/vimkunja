package tui

import (
	"fmt"
	"strings"

	"github.com/vikunja-tui/vikunja-tui/internal/vikunja"

	"charm.land/lipgloss/v2"
)

func renderProjects(m *model) string {
	w := 30
	header := lipgloss.NewStyle().Bold(true).Render(" Projects ")
	focus := m.focus == paneProjects

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")

	if m.projectLoading {
		b.WriteString("  loading...\n")
		return lipgloss.NewStyle().Width(w).Render(b.String())
	}
	if len(m.projects) == 0 {
		b.WriteString("  (no projects)\n")
		return lipgloss.NewStyle().Width(w).Render(b.String())
	}

	for i, p := range m.projects {
		sel := i == m.projectSelected
		marker := "  "
		if sel {
			marker = "[X]"
		}
		id := p.Identifier
		if id == "" {
			id = strings.TrimSpace(p.Title)
		}
		if id == "" {
			id = fmt.Sprintf("%d", p.ID)
		}
		line := fmt.Sprintf("%s %s", marker, id)
		if sel && focus {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().Width(w).Render(b.String())
}

func renderTasks(m *model) string {
	w := 0 // auto
	_ = w

	header := lipgloss.NewStyle().Bold(true).Render(" Tasks (list) ")
	focus := m.focus == paneTasks

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")

	if m.taskLoading {
		b.WriteString("  loading...\n")
		return b.String()
	}
	if len(m.tasks) == 0 {
		b.WriteString("  (no tasks)\n")
		return b.String()
	}

	for i, t := range m.tasks {
		sel := i == m.taskSelected
		done := "[ ]"
		if t.Done {
			done = "[x]"
		}
		due := strings.TrimSpace(t.DueDate)
		if due == "" {
			due = "-"
		}
		prio := fmt.Sprintf("P%d", t.Priority)
		idx := t.Identifier
		if idx == "" {
			idx = fmt.Sprintf("#%d", t.Index)
		}
		line := fmt.Sprintf("%s %s %s  %s  %s", done, prio, idx, t.Title, due)
		if sel {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Render(line)
		}
		if sel && focus {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Underline(true).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func renderTaskDetail(m *model) string {
	if m.taskSelected < 0 || m.taskSelected >= len(m.tasks) {
		return "No task selected."
	}
	t := m.tasks[m.taskSelected]

	var b strings.Builder
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	titleStyle := lipgloss.NewStyle().Bold(true).Underline(true)

	b.WriteString(headerStyle.Render(" TASK DETAIL "))
	b.WriteString("\n\n")

	b.WriteString(titleStyle.Render(t.Title))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("ID:       #%d\n", t.Index))
	b.WriteString(fmt.Sprintf("Priority: P%d\n", t.Priority))
	b.WriteString(fmt.Sprintf("Due:      %s\n", t.DueDate))
	b.WriteString(fmt.Sprintf("Done:     %v\n", t.Done))
	b.WriteString("\n")
	b.WriteString("Description:\n")
	if strings.TrimSpace(t.Description) == "" {
		b.WriteString("  (no description)")
	} else {
		b.WriteString(t.Description)
	}

	return b.String()
}

// compile-time assertion (in case we evolve Task struct)
var _ vikunja.Task
