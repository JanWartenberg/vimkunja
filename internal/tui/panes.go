package tui

import (
	"fmt"
	"strings"

	"github.com/vimkunja/vimkunja/internal/vimkunja"

	"charm.land/lipgloss/v2"
)

// fitBlock normalises a set of lines to exactly the given display height and
// at least the given width: every line is right-padded with spaces so short
// lines fill the pane, and the slice is padded/trimmed to exactly h lines.
// It deliberately does NOT re-render already-styled lines (which would corrupt
// embedded ANSI sequences). Truncation of over-long lines is left to the outer
// pane style, whose Width/MaxWidth is ANSI-safe.
func fitBlock(lines []string, w, h int) string {
	if w < 1 {
		w = 1
	}
	out := make([]string, 0, h)
	for _, ln := range lines {
		visW := lipgloss.Width(ln) // strips ANSI, measures visible width
		if pad := w - visW; pad > 0 {
			ln = ln + strings.Repeat(" ", pad)
		}
		out = append(out, ln)
	}
	for len(out) < h {
		out = append(out, strings.Repeat(" ", w))
	}
	if len(out) > h {
		out = out[:h]
	}
	return strings.Join(out, "\n")
}

// truncatePlain shortens a plain (non-ANSI) string so its visible width is at
// most maxW, rune-aware. Used to fit the variable title field before any ANSI
// styling is applied, so later layout never has to cut into escape sequences.
func truncatePlain(s string, maxW int) string {
	if maxW < 1 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > maxW {
		r = r[:len(r)-1]
	}
	return string(r)
}

func renderProjects(m *model, textW, textH int) string {
	header := lipgloss.NewStyle().Bold(true).Render(" Projects ")
	focus := m.focus == paneProjects

	lines := []string{header}

	if m.projectLoading {
		lines = append(lines, "  loading...")
		return fitBlock(lines, textW, textH)
	}
	if len(m.projects) == 0 {
		lines = append(lines, "  (no projects)")
		return fitBlock(lines, textW, textH)
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
		line = truncatePlain(line, textW)
		if sel && focus {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render(line)
		}
		lines = append(lines, line)
	}

	return fitBlock(lines, textW, textH)
}

func renderTasks(m *model, textW, textH int) string {
	header := lipgloss.NewStyle().Bold(true).Render(" Tasks (list) ")
	focus := m.focus == paneTasks

	lines := []string{header}

	if m.taskLoading {
		lines = append(lines, "  loading...")
		return fitBlock(lines, textW, textH)
	}
	if len(m.tasks) == 0 {
		lines = append(lines, "  (no tasks)")
		return fitBlock(lines, textW, textH)
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
		prioStr := fmt.Sprintf("P%d", t.Priority)
		// Priority segment: colored only when not done. Rendered once from a
		// PLAIN string (never nested inside another Render, which corrupts
		// the embedded ANSI codes in lipgloss v2).
		prioSeg := prioStr
		if !t.Done {
			prioSeg = lipgloss.NewStyle().Foreground(lipgloss.Color(priorityColor(int(t.Priority)))).Render(prioStr)
		}
		idx := t.Identifier
		if idx == "" {
			idx = fmt.Sprintf("#%d", t.Index)
		}

		// Compute the visible width consumed by the fixed (non-title) parts,
		// using the PLAIN priority string (prioSeg has the same visible
		// width as prioStr). Truncate the plain title to fit so that no later
		// truncation ever touches the embedded ANSI codes.
		fixedW := lipgloss.Width(done) + 1 +
			lipgloss.Width(prioStr) + 1 +
			lipgloss.Width(idx) + 2 +
			2 +
			lipgloss.Width(due)
		avail := textW - fixedW
		title := truncatePlain(t.Title, avail)

		// Style applied to the non-priority segments only. The priority keeps
		// its own colour even when selected. Each segment is rendered from a
		// plain string (no nesting) to avoid ANSI corruption.
		var segStyle *lipgloss.Style
		switch {
		case sel && focus:
			s := lipgloss.NewStyle().Bold(true).Underline(true)
			if !t.Done {
				s = s.Foreground(lipgloss.Color("205"))
			}
			segStyle = &s
		case sel:
			s := lipgloss.NewStyle().Bold(true)
			if !t.Done {
				s = s.Foreground(lipgloss.Color("229"))
			}
			segStyle = &s
		case t.Done:
			s := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Faint(true)
			segStyle = &s
		}
		rend := func(s string) string {
			if segStyle != nil {
				return segStyle.Render(s)
			}
			return s
		}

		line := rend(done) + " " + prioSeg + " " + rend(idx) + "  " + rend(title) + "  " + rend(due)
		lines = append(lines, line)
	}

	return fitBlock(lines, textW, textH)
}

func renderTaskDetail(m *model, textW, textH int) string {
	lines := []string{}
	if m.taskSelected < 0 || m.taskSelected >= len(m.tasks) {
		return fitBlock(append(lines, "No task selected."), textW, textH)
	}
	t := m.tasks[m.taskSelected]

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	titleStyle := lipgloss.NewStyle().Bold(true).Underline(true)

	lines = append(lines, headerStyle.Render(" TASK DETAIL "))
	lines = append(lines, "")
	lines = append(lines, titleStyle.Render(t.Title))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("ID:       #%d", t.Index))
	lines = append(lines, fmt.Sprintf("Priority: P%d", t.Priority))
	lines = append(lines, fmt.Sprintf("Due:      %s", t.DueDate))
	lines = append(lines, fmt.Sprintf("Done:     %v", t.Done))
	lines = append(lines, "")
	lines = append(lines, "Description:")
	if strings.TrimSpace(t.Description) == "" {
		lines = append(lines, "  (no description)")
	} else {
		for _, ln := range strings.Split(t.Description, "\n") {
			lines = append(lines, ln)
		}
	}

	return fitBlock(lines, textW, textH)
}

// compile-time assertion (in case we evolve Task struct)
var _ vimkunja.Task

// priorityColor maps a Vikunja priority to a terminal colour code string:
// 0 = unset/gray, low = green, climbing through yellow/orange to red for urgent.
func priorityColor(p int) string {
	switch {
	case p == 0:
		return "245" // gray (not specified)
	case p <= 1:
		return "34" // green (low priority)
	case p == 2:
		return "220" // yellow (medium)
	case p == 3:
		return "208" // orange (high)
	default:
		return "196" // red (urgent / 4-5)
	}
}
