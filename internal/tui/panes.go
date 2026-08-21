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

func padPlainRight(s string, width int) string {
	return s + strings.Repeat(" ", max(0, width-lipgloss.Width(s)))
}

func renderProjects(m *model, textW, textH int) string {
	header := " " + renderHeading("Projects") + " "
	focus := m.focus == paneProjects

	lines := []string{header}

	if m.projectLoading {
		lines = append(lines, "  loading...")
		return fitBlock(lines, textW, textH)
	}
	rows := m.projectRows()
	if len(rows) == 0 {
		lines = append(lines, "  (no projects)")
		return fitBlock(lines, textW, textH)
	}

	visibleRows := max(0, textH-1)
	overflows := len(rows) > visibleRows
	contentWidth := textW
	if overflows {
		contentWidth -= 2
	}
	selected := clamp(m.projectSelected, 0, len(rows)-1)
	m.projectViewportStart = clamp(m.projectViewportStart, 0, max(0, len(rows)-visibleRows))
	if selected < m.projectViewportStart {
		m.projectViewportStart = selected
	} else if visibleRows > 0 && selected >= m.projectViewportStart+visibleRows {
		m.projectViewportStart = selected - visibleRows + 1
	}
	end := min(len(rows), m.projectViewportStart+visibleRows)
	for i, row := range rows[m.projectViewportStart:end] {
		rowIndex := m.projectViewportStart + i
		sel := rowIndex == selected
		marker := "  "
		switch {
		case row.Recovery:
			marker = "! "
		case row.HasChildren && row.Expanded:
			marker = "▾ "
		case row.HasChildren:
			marker = "▸ "
		}
		label := row.Project.Identifier
		if label == "" {
			label = strings.TrimSpace(row.Project.Title)
		}
		if label == "" {
			label = fmt.Sprintf("%d", row.Project.ID)
		}
		if sel {
			label = "> " + label + " <"
		}
		line := strings.Repeat("  ", row.Depth) + marker + label
		line = truncatePlain(line, contentWidth)
		if sel && focus {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render(line)
		}
		lines = append(lines, line)
	}

	if overflows {
		return renderVerticalScrollbar(lines, textW, textH, m.projectViewportStart, len(rows), 1, visibleRows, focus)
	}
	return fitBlock(lines, textW, textH)
}

func renderTasks(m *model, textW, textH int) string {
	if m.kanbanActive && !m.inOverview() {
		return renderKanbanBoard(m, textW, textH)
	}
	headerText := "Project Tasks"
	if m.inOverview() {
		headerText = fmt.Sprintf("Overview · %s · %s", m.overviewSortLabel(), m.overviewFilterLabel())
		if m.overviewLabel.ID != 0 {
			headerText = fmt.Sprintf("Overview · Label: %s · %s · %s", m.overviewLabel.Title, m.overviewSortLabel(), m.overviewFilterLabel())
		}
	}
	header := " " + renderHeading(headerText) + " "
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
	overflows := len(m.tasks) > visibleRows
	contentWidth := textW
	if overflows {
		contentWidth -= 2
	}
	selected := clamp(m.taskSelected, 0, len(m.tasks)-1)
	// Keep context below the selection instead of pinning it to the bottom.
	bottomMargin := min(2, max(0, visibleRows-1))
	start := clamp(selected-visibleRows+1+bottomMargin, 0, max(0, len(m.tasks)-visibleRows))
	end := min(len(m.tasks), start+visibleRows)
	visibleTasks := append([]vimkunja.Task(nil), m.tasks[start:end]...)
	if m.inlineTitle != nil {
		for i := range visibleTasks {
			if visibleTasks[i].ID == m.inlineTitle.task.ID {
				visibleTasks[i].Title = m.inlineTitle.title
				break
			}
		}
	}
	layout := taskMetadataLayoutForWidth(visibleTasks, contentWidth, m.inOverview(), m.currentTime())
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
		prioStr := padPlainRight(prioritySummary(t.Priority), 6)
		if m.inOverview() {
			project = fmt.Sprintf("%-8s", truncatePlain(project, 8))
		}
		// Priority segment: colored only when not done. Rendered once from a
		// PLAIN string (never nested inside another Render, which corrupts
		// the embedded ANSI codes in lipgloss v2).
		prioSeg := prioStr
		if !t.Done {
			prioSeg = lipgloss.NewStyle().Foreground(lipgloss.Color(priorityColor(int(t.Priority)))).Render(prioStr)
		}
		dueSeg := due
		if !t.Done {
			dueSeg = duePresentation(t, due, m.currentTime())
		}
		titleText := t.Title
		inlineEditing := m.inlineTitle != nil && m.inlineTitle.task.ID == t.ID
		if inlineEditing {
			titleText = m.inlineTitle.title
		}
		title := padPlainRight(truncatePlain(titleText, layout.titleWidth), layout.titleWidth)

		// Style applied to the non-priority segments only. The priority keeps
		// its own colour even when selected. Each segment is rendered from a
		// plain string (no nesting) to avoid ANSI corruption.
		var segStyle *lipgloss.Style
		switch {
		case sel && focus:
			s := lipgloss.NewStyle().Bold(true)
			if !t.Done {
				s = s.Foreground(lipgloss.Color(tuiTheme.selectionAccent))
				dueSeg = s.Render(ansi.Strip(dueSeg))
			}
			segStyle = &s
		case sel:
			s := lipgloss.NewStyle().Bold(true)
			if !t.Done {
				s = s.Foreground(lipgloss.Color(tuiTheme.inactiveSelection))
			}
			segStyle = &s
		case t.Done:
			s := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Faint(true)
			segStyle = &s
		}
		rend := func(s string) string {
			if segStyle != nil {
				return segStyle.Render(s)
			}
			return s
		}

		segments := []string{rend(done)}
		if layout.showPriority {
			segments = append(segments, prioSeg)
		}
		if layout.showProject && project != "" {
			segments = append(segments, project)
		}
		if inlineEditing {
			cursor := clamp(m.inlineTitle.cursor, 0, len([]rune(titleText)))
			cursorWidth := min(lipgloss.Width(string([]rune(titleText)[:cursor])), layout.titleWidth)
			prefixWidth := lipgloss.Width(strings.Join(segments, " "))
			if len(segments) > 0 {
				prefixWidth++
			}
			m.inlineCursorX = m.tasksPaneTextX() + prefixWidth + cursorWidth
			m.inlineCursorY = 2 + i // pane border, header, then visible Task row
			m.inlineCursorVisible = true
		}
		segments = append(segments, rend(title))
		if layout.showDue {
			segments = append(segments, dueSeg)
		}
		line := strings.Join(segments, " ")
		if t.Done {
			line = lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Faint(true).Render(ansi.Strip(line))
			if sel && focus {
				chevron := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent))
				line = chevron.Render("> ") + line + chevron.Render(" <")
			}
		}
		lines = append(lines, line)
	}

	if overflows {
		return renderVerticalScrollbar(lines, textW, textH, start, len(m.tasks), 1, visibleRows, focus)
	}
	return fitBlock(lines, textW, textH)
}

func (m *model) currentTime() time.Time {
	if m.now == nil {
		return time.Now()
	}
	return m.now()
}

type taskMetadataLayout struct {
	titleWidth                int
	showPriority, showProject bool
	showDue                   bool
}

// taskMetadataLayoutForWidth preserves the title before optional metadata in
// narrow Panes. Its shared layout keeps columns aligned across visible Tasks.
func taskMetadataLayoutForWidth(tasks []vimkunja.Task, textW int, overview bool, now time.Time) taskMetadataLayout {
	layout := taskMetadataLayout{showPriority: true, showProject: overview, showDue: true}
	titleWidth, dueWidth := 1, 1
	for _, task := range tasks {
		titleWidth = max(titleWidth, lipgloss.Width(task.Title))
		due := strings.TrimSpace(task.DueDate)
		if due == "" {
			due = "-"
		}
		if !task.Done {
			due = ansi.Strip(duePresentation(task, due, now))
		}
		dueWidth = max(dueWidth, lipgloss.Width(due))
	}
	metadataWidth := func() int {
		width := 0
		if layout.showPriority {
			width += 7 // six-character priority plus its preceding space
		}
		if layout.showProject {
			width += 9 // eight-character project identifier plus its preceding space
		}
		if layout.showDue {
			width += dueWidth + 1
		}
		return width
	}
	for textW < lipgloss.Width("[ ] ")+titleWidth+metadataWidth() {
		switch {
		case layout.showProject:
			layout.showProject = false
		case layout.showDue:
			layout.showDue = false
		case layout.showPriority:
			layout.showPriority = false
		default:
			layout.titleWidth = max(1, textW-lipgloss.Width("[ ] "))
			return layout
		}
	}
	layout.titleWidth = titleWidth
	return layout
}

func duePresentation(task vimkunja.Task, value string, now time.Time) string {
	due, ok := overviewDueDate(task)
	if !ok {
		return value
	}
	today := localStartOfDay(now.Local())
	days := int(due.Sub(today).Hours() / 24)
	label, color := value, ""
	switch {
	case days < 0:
		label, color = fmt.Sprintf("%d days overdue", -days), tuiTheme.danger
	case days == 0:
		label, color = "today", tuiTheme.dueSoon
	case days == 1:
		label, color = "tomorrow", tuiTheme.dueSoon
	case days <= 7:
		label, color = fmt.Sprintf("in %d days", days), tuiTheme.dueUpcoming
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

func infoPaneRows(lines []string, width int) []string {
	wrapped := wrapDescription(strings.Join(lines, "\n"), max(1, width))
	rows := make([]string, len(wrapped))
	for index, line := range wrapped {
		rows[index] = line.text
	}
	return rows
}

func infoPaneScrollRows(pane *infoPane, textW, viewportRows int) ([]string, bool) {
	rows := infoPaneRows(pane.lines, textW)
	if len(rows) <= viewportRows {
		return rows, false
	}
	// Overflow is assessed at the full text width before reserving the rail.
	return infoPaneRows(pane.lines, max(1, textW-2)), true
}

func renderInfoPane(pane *infoPane, textW, textH int, focused bool) string {
	if pane == nil {
		return fitBlock([]string{"No information available."}, textW, textH)
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render(" " + pane.title + " ")
	viewportRows := max(0, textH-2) // heading and its blank separator
	rows, overflows := infoPaneScrollRows(pane, textW, viewportRows)
	offset := clamp(pane.offset, 0, max(0, len(rows)-viewportRows))
	lines := append([]string{title, ""}, rows[offset:]...)
	if overflows {
		return renderVerticalScrollbar(lines, textW, textH, offset, len(rows), 2, viewportRows, focused)
	}
	return fitBlock(lines, textW, textH)
}

func renderTaskDetail(m *model, textW, textH int) string {
	if m.editor == nil {
		if m.kanbanActive && !m.inOverview() {
			return renderEmptyKanbanDetail(textW, textH)
		}
		return fitBlock([]string{"No task editor available."}, textW, textH)
	}
	if textH < 12 {
		return renderCompactTaskDetail(m, textW, textH)
	}
	e := m.editor
	header := "TASK EDITOR"
	if e.id == 0 {
		header = "NEW TASK"
	}
	lines := []string{" " + renderHeading(header) + " ", ""}
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
		if field == fieldLabels && active {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Bold(true).Render("▸ "+label) + value
		}
		return focusedField(label+value, active)
	}
	labels := "none"
	if e.labelsLoading {
		labels = "loading..."
	} else if len(e.labels) > 0 {
		titles := make([]string, 0, len(e.labels))
		for i, label := range e.labels {
			title := label.Title
			if detailFocused && e.field == fieldLabels && i == clamp(e.labelSelected, 0, len(e.labels)-1) {
				title = lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Bold(true).Render(title)
			}
			titles = append(titles, title)
		}
		labels = strings.Join(titles, ", ")
	}
	renderMetadata := func(field editorField, value string) string {
		if e.draft.Done && !(detailFocused && e.field == field) {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Faint(true).Render(ansi.Strip(value))
		}
		return value
	}
	lines = append(lines,
		renderField(fieldProject, "Project:  ", projectName),
		renderField(fieldTitle, "Title:    ", renderMetadata(fieldTitle, m.editorText(fieldTitle, e.draft.Title))),
		renderField(fieldPriority, "Priority: ", renderMetadata(fieldPriority, priority)),
		renderField(fieldDueDate, "Due Date: ", renderMetadata(fieldDueDate, m.editorText(fieldDueDate, due))),
		renderField(fieldLabels, "Labels:   ", renderMetadata(fieldLabels, labels)),
		renderField(fieldWorkflowStatus, "Status:   ", e.workflowStatusLabel()),
	)
	if e.id != 0 {
		done := "[ ]"
		if e.draft.Done {
			done = "[x]"
		}
		lines = append(lines, renderField(fieldDone, "Done:     ", done))
	}
	if e.id != 0 && textH >= 12 && !e.isPickerMode() {
		lines = append(lines, placeholderText(fmt.Sprintf("(DB-ID: %d, Task-Index: %d)", e.id, e.taskIndex)))
	}
	lines = append(lines, "", renderField(fieldDescription, "Description:", ""))
	descriptionScrollable := false
	descriptionScrollTop, descriptionRows, descriptionViewportStart, descriptionViewportRows := 0, 0, 0, 0
	if e.mode == editorProjectPicker {
		lines = append(lines, "  Filter: "+e.projectFilter)
		matches := m.filteredProjects(e.projectFilter)
		choices := make([]string, 0, len(matches))
		for _, project := range matches {
			choices = append(choices, project.Title)
		}
		selected := e.pickerIndex
		if len(choices) == 0 {
			choices = append(choices, lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Render("No match found"))
			selected = -1
		}
		lines = append(lines, renderPickerChoices(choices, selected, tuiTheme.selectionAccent)...)
	} else if e.mode == editorPriorityPicker {
		lines = append(lines, renderPriorityPicker(e)...)
	} else if e.mode == editorWorkflowStatusPicker {
		choices := make([]string, 0, len(e.workflowStatus.Buckets))
		for _, bucket := range e.workflowStatus.Buckets {
			choices = append(choices, bucket.Title)
		}
		lines = append(lines, renderPickerChoices(choices, e.workflowStatusIndex, tuiTheme.selectionAccent)...)
	} else if description := descriptionTerminalText(e.draft.Description); description == "" {
		lines = append(lines, placeholderText("  (no description)"))
	} else {
		available := max(1, textH-len(lines))
		desc := wrapDescription(description, textW)
		// Detect overflow before reserving the rail. Once it exists, the
		// narrower render width determines the real scroll range.
		if len(desc) > available {
			descriptionScrollable = true
			desc = wrapDescription(description, max(1, textW-2))
		}
		start := 0
		if e.field == fieldDescription && len(desc) > available {
			start = clamp(e.descScroll, 0, len(desc)-available)
		}
		descriptionScrollTop = start
		descriptionRows = len(desc)
		descriptionViewportStart = len(lines)
		descriptionViewportRows = available
		for _, line := range desc[start:] {
			lines = append(lines, m.editorDescriptionText(line))
		}
	}
	if descriptionScrollable {
		return renderVerticalScrollbar(lines, textW, textH, descriptionScrollTop, descriptionRows, descriptionViewportStart, descriptionViewportRows, m.focus == paneDetail)
	}
	return fitBlock(lines, textW, textH)
}

func renderEmptyKanbanDetail(textW, textH int) string {
	lines := make([]string, max(1, textH))
	for i := range lines {
		lines[i] = ""
	}
	message := "No task selected"
	lines[len(lines)/2] = strings.Repeat(" ", max(0, (textW-lipgloss.Width(message))/2)) + message
	return fitBlock(lines, textW, textH)
}

func renderCompactTaskDetail(m *model, textW, textH int) string {
	e := m.editor
	header := "TASK EDITOR"
	if e.id == 0 {
		header = "NEW TASK"
	}
	lines := []string{" " + renderHeading(header) + " ", ""}
	detailFocused := m.focus == paneDetail
	renderField := func(field editorField, label, value string) string {
		active := detailFocused && e.field == field
		if field == fieldLabels && active {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Bold(true).Render("▸ "+label) + value
		}
		return focusedField(label+value, active)
	}
	projectName := "(none)"
	for _, project := range m.projects {
		if project.ID == e.draft.ProjectID {
			projectName = project.Title
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
	labels := "none"
	if e.labelsLoading {
		labels = "loading..."
	} else if len(e.labels) > 0 {
		titles := make([]string, 0, len(e.labels))
		for i, label := range e.labels {
			title := label.Title
			if detailFocused && e.field == fieldLabels && i == clamp(e.labelSelected, 0, len(e.labels)-1) {
				title = lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Bold(true).Render(title)
			}
			titles = append(titles, title)
		}
		labels = strings.Join(titles, ", ")
	}
	renderMetadata := func(field editorField, value string) string {
		if e.draft.Done && !(detailFocused && e.field == field) {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Faint(true).Render(ansi.Strip(value))
		}
		return value
	}
	values := map[editorField]struct{ label, value string }{
		fieldProject:        {"Project:  ", projectName},
		fieldTitle:          {"Title:    ", renderMetadata(fieldTitle, m.editorText(fieldTitle, e.draft.Title))},
		fieldPriority:       {"Priority: ", renderMetadata(fieldPriority, priority)},
		fieldDueDate:        {"Due Date: ", renderMetadata(fieldDueDate, m.editorText(fieldDueDate, due))},
		fieldLabels:         {"Labels:   ", renderMetadata(fieldLabels, labels)},
		fieldWorkflowStatus: {"Status:   ", e.workflowStatusLabel()},
		fieldDone:           {"Done:     ", "[ ]"},
	}
	if e.draft.Done {
		values[fieldDone] = struct{ label, value string }{"Done:     ", "[x]"}
	}
	fields := m.detailFieldsFor(true)
	body := make([]string, 0, len(fields)+3)
	descriptionRow := 0
	pickerSelectedRow := -1
	for _, field := range fields {
		switch field {
		case fieldMoreFields:
			state := "collapsed"
			if e.moreFieldsExpanded {
				state = "expanded"
			}
			body = append(body, renderField(field, "More fields: ", state))
		case fieldDescription:
			descriptionRow = len(body)
			body = append(body, renderField(fieldDescription, "Description:", ""))
		default:
			value := values[field]
			body = append(body, renderField(field, value.label, value.value))
		}
		switch {
		case field == fieldProject && e.mode == editorProjectPicker:
			body = append(body, "  Filter: "+e.projectFilter)
			matches := m.filteredProjects(e.projectFilter)
			choices := make([]string, 0, len(matches))
			for _, project := range matches {
				choices = append(choices, project.Title)
			}
			if len(choices) == 0 {
				choices = append(choices, placeholderText("No match found"))
			}
			pickerSelectedRow = len(body) + 1 + clamp(e.pickerIndex, 0, len(choices)-1)
			body = append(body, renderPickerChoices(choices, e.pickerIndex, tuiTheme.selectionAccent)...)
		case field == fieldPriority && e.mode == editorPriorityPicker:
			pickerSelectedRow = len(body) + 1 + clamp(e.priorityIndex, 0, len(tuiTheme.priority)-1)
			body = append(body, renderPriorityPicker(e)...)
		case field == fieldWorkflowStatus && e.mode == editorWorkflowStatusPicker:
			choices := make([]string, 0, len(e.workflowStatus.Buckets))
			for _, bucket := range e.workflowStatus.Buckets {
				choices = append(choices, bucket.Title)
			}
			if len(choices) > 0 {
				pickerSelectedRow = len(body) + 1 + clamp(e.workflowStatusIndex, 0, len(choices)-1)
			}
			body = append(body, renderPickerChoices(choices, e.workflowStatusIndex, tuiTheme.selectionAccent)...)
		}
	}
	bodyRows := max(1, textH-2)
	descriptionRows := compactDescriptionRows(textH, len(fields))
	description := descriptionTerminalText(e.draft.Description)
	desc := wrapDescription(description, textW)
	if description == "" {
		desc = []descriptionLine{{text: placeholderText("  (no description)")}}
	}
	descriptionScrollable := len(desc) > descriptionRows
	if descriptionScrollable {
		desc = wrapDescription(description, max(1, textW-2))
	}
	start := 0
	if e.descriptionActive && len(desc) > descriptionRows {
		start = clamp(e.descScroll, 0, len(desc)-descriptionRows)
	}
	for row := 0; row < descriptionRows; row++ {
		if line := start + row; line < len(desc) {
			body = append(body, m.editorDescriptionText(desc[line]))
		} else {
			body = append(body, "")
		}
	}
	outerLimit := max(0, len(body)-bodyRows)
	if pickerSelectedRow >= 0 {
		if pickerSelectedRow < e.detailScroll {
			e.detailScroll = pickerSelectedRow
		}
		if pickerSelectedRow >= e.detailScroll+bodyRows {
			e.detailScroll = pickerSelectedRow - bodyRows + 1
		}
	}
	e.detailScroll = clamp(e.detailScroll, 0, outerLimit)
	visible := body[e.detailScroll:min(len(body), e.detailScroll+bodyRows)]
	lines = append(lines, visible...)
	if e.descriptionActive && descriptionScrollable && descriptionRow >= e.detailScroll && descriptionRow < e.detailScroll+bodyRows {
		return renderVerticalScrollbar(lines, textW, textH, start, len(desc), 2+descriptionRow-e.detailScroll+1, descriptionRows, true)
	}
	if outerLimit > 0 {
		return renderVerticalScrollbar(lines, textW, textH, e.detailScroll, len(body), 2, bodyRows, detailFocused)
	}
	return fitBlock(lines, textW, textH)
}

// renderPriorityPicker keeps the picker visually separate from Description and
// colours the active choice with the same semantic mapping used everywhere
// else priorities are displayed.
func renderPriorityPicker(e *taskEditor) []string {
	choices := make([]string, len(tuiTheme.priority))
	for priority := range tuiTheme.priority {
		choices[priority] = priorityName(int64(priority))
	}
	return renderPickerChoices(choices, e.priorityIndex, priorityColor(e.priorityIndex))
}

// renderPickerChoices draws the common compact picker frame. Its active border
// uses the shared selection accent; Project and Status use that accent for
// their active choice, while Priority uses its semantic choice colour.
func renderPickerChoices(choices []string, selected int, selectionColor string) []string {
	rows := make([]string, 0, len(choices))
	for index, choice := range choices {
		row := "  " + choice
		if index == selected {
			row = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(selectionColor)).Render("▸ " + choice)
		}
		rows = append(rows, row)
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(paneBorderColor(true))).
		Padding(0, 1)
	return strings.Split(style.Render(strings.Join(rows, "\n")), "\n")
}

func placeholderText(value string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Faint(true).Render(value)
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

func (m *model) readOnlyDescriptionScrollLimit() int {
	if m.editor == nil {
		return 0
	}
	if m.compactDetail() {
		w, _ := m.viewportSize()
		descriptionRows := m.compactDescriptionRows()
		lines := wrapDescription(descriptionTerminalText(m.editor.draft.Description), w-4)
		if len(lines) > descriptionRows {
			lines = wrapDescription(descriptionTerminalText(m.editor.draft.Description), max(1, w-6))
		}
		return max(0, len(lines)-descriptionRows)
	}
	w, h := m.viewportSize()
	textW := w - 4
	contentH := max(4, h-3)
	topH := m.detailPaneTop()
	textH := max(4, contentH-topH) - 2
	precedingLines := 11
	if m.editor.id == 0 {
		precedingLines--
	} else if textH >= 12 && !m.editor.isPickerMode() {
		precedingLines++
	}
	available := max(1, textH-precedingLines)
	lines := wrapDescription(descriptionTerminalText(m.editor.draft.Description), textW)
	if len(lines) > available {
		lines = wrapDescription(descriptionTerminalText(m.editor.draft.Description), max(1, textW-2))
	}
	return max(0, len(lines)-available)
}

func (m *model) editorDescriptionText(line descriptionLine) string {
	if m.editor == nil || m.focus != paneDetail || m.editor.field != fieldDescription || descriptionHasHTML(m.editor.draft.Description) {
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
	block := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.dark)).Background(lipgloss.Color(tuiTheme.bright)).Render(string(r[index]))
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

// compile-time assertion (in case we evolve Task struct)
var _ vimkunja.Task
