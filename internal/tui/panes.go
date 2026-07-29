package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/vimkunja/vimkunja/internal/vimkunja"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	if m.kanbanActive && !m.inOverview() {
		return renderKanbanBoard(m, textW, textH)
	}
	headerText := " Tasks (list) "
	if m.inOverview() {
		headerText = fmt.Sprintf(" Overview · %s · %s ", m.overviewSortLabel(), m.overviewFilterLabel())
	}
	header := lipgloss.NewStyle().Bold(true).Render(headerText)
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

	// The Tasks Pane owns its viewport: its selected Task must remain visible
	// when a Detail Pane reduces the available height, regardless of Scope.
	visibleRows := max(0, textH-1) // header consumes the first row
	selected := clamp(m.taskSelected, 0, len(m.tasks)-1)
	// Keep context below the selection instead of pinning it to the bottom.
	bottomMargin := min(2, max(0, visibleRows-1))
	start := max(0, selected-visibleRows+1+bottomMargin)
	end := min(len(m.tasks), start+visibleRows)
	for i, t := range m.tasks[start:end] {
		sel := start+i == selected
		done := "[ ]"
		if t.Done {
			done = "[x]"
		}
		due := strings.TrimSpace(t.DueDate)
		if due == "" {
			due = "-"
		}
		project := ""
		if m.inOverview() {
			project = t.ProjectIdentifier
			if project == "" {
				project = t.ProjectTitle
			}
			if project == "" {
				project = fmt.Sprintf("%d", t.ProjectID)
			}
		}
		prioStr := prioritySummary(t.Priority)
		// Priority segment: colored only when not done. Rendered once from a
		// PLAIN string (never nested inside another Render, which corrupts
		// the embedded ANSI codes in lipgloss v2).
		prioSeg := prioStr
		if !t.Done {
			prioSeg = lipgloss.NewStyle().Foreground(lipgloss.Color(priorityColor(int(t.Priority)))).Render(prioStr)
		}
		// Compute the visible width consumed by the fixed (non-title) parts,
		// using PLAIN segments so that truncation never touches ANSI codes.
		fixedW := lipgloss.Width(done) + 1 +
			lipgloss.Width(prioStr) + 1 +
			lipgloss.Width(project) + 2 +
			2 + lipgloss.Width(due)
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

		dueSeg := due
		if !t.Done {
			dueSeg = duePresentation(t, due, m.currentTime())
		}
		projectSeg := ""
		if project != "" {
			projectSeg = project + "  "
		}
		line := rend(done) + " " + prioSeg + " " + projectSeg + rend(title) + "  " + dueSeg
		if t.Done {
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Faint(true).Render(ansi.Strip(line))
		}
		lines = append(lines, line)
	}

	return fitBlock(lines, textW, textH)
}

func (m *model) currentTime() time.Time {
	if m.now == nil {
		return time.Now()
	}
	return m.now()
}

func duePresentation(task vimkunja.Task, value string, now time.Time) string {
	due, ok := overviewDueDate(task)
	if !ok {
		return value
	}
	today := now.Local().Truncate(24 * time.Hour)
	days := int(due.Sub(today).Hours() / 24)
	label, color := value, ""
	switch {
	case days < 0:
		label, color = fmt.Sprintf("%d days overdue", -days), "196"
	case days == 0:
		label, color = "today", "208"
	case days == 1:
		label, color = "tomorrow", "208"
	case days <= 7:
		label, color = fmt.Sprintf("in %d days", days), "220"
	case due.Year() == today.Year():
		label = due.Format("02 Jan")
	default:
		label = due.Format("02 Jan 2006")
	}
	if color == "" {
		return label
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(label)
}

type infoPane struct {
	title    string
	lines    []string
	offset   int
	errorLog bool
}

func renderInfoPane(pane *infoPane, textW, textH int) string {
	if pane == nil {
		return fitBlock([]string{"No information available."}, textW, textH)
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render(" " + pane.title + " ")
	offset := clamp(pane.offset, 0, max(0, len(pane.lines)-1))
	return fitBlock(append([]string{title, ""}, pane.lines[offset:]...), textW, textH)
}

func renderTaskDetail(m *model, textW, textH int) string {
	if m.editor == nil {
		return fitBlock([]string{"No task editor available."}, textW, textH)
	}
	e := m.editor
	header := " TASK EDITOR "
	if e.id == 0 {
		header = " NEW TASK "
	}
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(selectionAccent)).Render(header), ""}
	projectName := "(none)"
	for _, p := range m.projects {
		if p.ID == e.draft.ProjectID {
			projectName = p.Title
			break
		}
	}
	due := e.draft.DueDate
	if due == "" {
		due = placeholderText("-")
	} else if !(e.mode == editorInsert && e.field == fieldDueDate) {
		due = ansi.Strip(duePresentation(vimkunja.Task{DueDate: due}, due, m.currentTime()))
	}
	priority := prioritySummary(e.draft.Priority)
	if e.draft.Priority == 0 {
		priority = placeholderText(priority)
	} else {
		priority = lipgloss.NewStyle().Foreground(lipgloss.Color(priorityColor(int(e.draft.Priority)))).Render(priority)
	}
	detailFocused := m.focus == paneDetail
	// A Picker field is styled as one unit. Avoid nested placeholder styling
	// resetting that background before the field value is rendered.
	if detailFocused && e.isPickerMode() && e.field == fieldPriority {
		priority = priorityName(e.draft.Priority)
	}
	renderField := func(field editorField, label, value string) string {
		active := detailFocused && e.field == field
		if e.isPickerMode() {
			return label + pickerField(value, active)
		}
		return focusedField(label+value, active)
	}
	lines = append(lines,
		renderField(fieldProject, "Project:  ", projectName),
		renderField(fieldTitle, "Title:    ", m.editorText(fieldTitle, e.draft.Title)),
		renderField(fieldPriority, "Priority: ", priority),
		renderField(fieldDueDate, "Due Date: ", m.editorText(fieldDueDate, due)),
		renderField(fieldWorkflowStatus, "Status:   ", e.workflowStatusLabel()),
		fmt.Sprintf("Done:     %s (read-only)", doneLabel(m, e.id)),
	)
	if e.id != 0 && textH >= 12 && !e.isPickerMode() {
		lines = append(lines, placeholderText(fmt.Sprintf("(DB-ID: %d, Task-Index: %d)", e.id, e.taskIndex)))
	}
	lines = append(lines, "", renderField(fieldDescription, "Description:", ""))
	if e.mode == editorProjectPicker {
		lines = append(lines, "  Filter: "+e.projectFilter)
		matches := m.filteredProjects(e.projectFilter)
		if len(matches) == 0 {
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  No match found"))
		} else {
			for i, p := range matches {
				prefix := "  "
				if i == e.pickerIndex {
					prefix = "▸ "
				}
				lines = append(lines, prefix+p.Title)
			}
		}
	} else if e.mode == editorPriorityPicker {
		for i := int64(0); i <= 5; i++ {
			prefix := "  "
			if int(i) == e.priorityIndex {
				prefix = "▸ "
			}
			lines = append(lines, prefix+priorityName(i))
		}
	} else if e.mode == editorWorkflowStatusPicker {
		for i, bucket := range e.workflowStatus.Buckets {
			prefix := "  "
			if i == e.workflowStatusIndex {
				prefix = "▸ "
			}
			lines = append(lines, prefix+bucket.Title)
		}
	} else if e.draft.Description == "" {
		lines = append(lines, placeholderText("  (no description)"))
	} else {
		desc := wrapDescription(e.draft.Description, textW)
		start := 0
		available := max(1, textH-len(lines))
		if e.field == fieldDescription && len(desc) > available {
			start = clamp(e.descScroll, 0, len(desc)-available)
		}
		for _, line := range desc[start:] {
			lines = append(lines, m.editorDescriptionText(line))
		}
	}
	return fitBlock(lines, textW, textH)
}

func placeholderText(value string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Faint(true).Render(value)
}

// editorText marks the character at the insertion point instead of inserting a
// printable cursor glyph, so the text never shifts while the cursor blinks.
func (m *model) editorText(field editorField, value string) string {
	if m.editor == nil || m.editor.mode != editorInsert || m.editor.field != field || !m.cursorVisible || m.nativeEditorCursorActive() {
		return value
	}
	return cursorMarkedText(value, m.editor.cursor)
}

type descriptionLine struct {
	text       string
	start, end int
}

func wrapDescription(value string, width int) []descriptionLine {
	width = max(1, width)
	var out []descriptionLine
	start := 0
	for _, sourceLine := range strings.Split(value, "\n") {
		r := []rune(sourceLine)
		if len(r) == 0 {
			out = append(out, descriptionLine{"", start, start})
		}
		for len(r) > 0 {
			n := min(width, len(r))
			out = append(out, descriptionLine{string(r[:n]), start, start + n})
			start += n
			r = r[n:]
		}
		start++ // original newline, including the final segment's boundary
	}
	return out
}

func (m *model) editorDescriptionText(line descriptionLine) string {
	if m.editor == nil || m.focus != paneDetail || m.editor.field != fieldDescription {
		return line.text
	}
	if m.editor.cursor < line.start || m.editor.cursor > line.end {
		return line.text
	}
	if m.nativeEditorCursorActive() {
		return line.text
	}
	cursor := m.editor.cursor - line.start
	if m.editor.mode == editorNormal {
		return normalCursorText(line.text, cursor)
	}
	if m.editor.mode == editorInsert && m.cursorVisible {
		return cursorMarkedText(line.text, cursor)
	}
	return line.text
}

// normalCursorText is Vim's static block cursor: white background and black
// glyph, without adding a printable character to the line.
func normalCursorText(value string, cursor int) string {
	r := []rune(value)
	if len(r) == 0 {
		return value
	}
	cursor = clamp(cursor, 0, len(r))
	index := cursor
	if index == len(r) {
		index--
	}
	block := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("255")).Render(string(r[index]))
	return string(r[:index]) + block + string(r[index+1:])
}

func cursorMarkedText(value string, cursor int) string {
	r := []rune(value)
	if len(r) == 0 {
		return value
	}
	cursor = clamp(cursor, 0, len(r))
	index := cursor
	if index == len(r) {
		index--
	}
	return string(r[:index]) + lipgloss.NewStyle().Reverse(true).Render(string(r[index])) + string(r[index+1:])
}

func priorityName(priority int64) string {
	return []string{"None", "Low", "Medium", "High", "Urgent", "Do now"}[clamp(int(priority), 0, 5)]
}

func prioritySummary(priority int64) string {
	if priority == 0 {
		return "—"
	}
	return priorityName(priority)
}

func doneLabel(m *model, taskID int64) string {
	if m.kanbanActive {
		for _, bucket := range m.kanbanBoard.Buckets {
			for _, task := range bucket.Tasks {
				if task.ID == taskID && task.Done {
					return "done"
				}
			}
		}
	}
	for _, task := range m.tasks {
		if task.ID == taskID && task.Done {
			return "done"
		}
	}
	return "open"
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
