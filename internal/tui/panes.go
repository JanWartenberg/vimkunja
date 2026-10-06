package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"

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

func truncateFieldValue(value string, width int) string {
	if lipgloss.Width(value) <= width {
		return value
	}
	return truncatePlain(value, max(1, width-lipgloss.Width("…"))) + "…"
}

// editorFieldViewport keeps an active one-line field's cursor and a little
// following context visible. Once it reaches the right edge, omitted leading
// text is marked with an ellipsis instead of overflowing the Detail Pane.
func (m *model) editorFieldViewport(field editorField, value string, width int) string {
	width = max(1, width)
	if m.editor == nil || m.editor.field != field || (m.editor.mode != editorInsert && !(m.editor.mode == editorNormal && field == fieldTitle)) {
		return truncateFieldValue(value, width)
	}
	runes := []rune(value)
	cursor := clamp(m.editor.cursor, 0, len(runes))
	start := 0
	if lipgloss.Width(value) > width {
		end := min(len(runes), cursor+3)
		for start < cursor && lipgloss.Width("…"+string(runes[start:end])) > width {
			start++
		}
		value = string(runes[start:end])
	}
	m.editor.fieldViewportStart = start
	if start > 0 {
		value = "…" + value
	}
	// Bubble Tea owns the sole native cursor. Render a textual fallback only
	// when native cursor shapes are explicitly disabled.
	if m.nativeEditorCursorActive() {
		return value
	}
	visibleCursor := cursor - start + boolToInt(start > 0)
	if m.editor.mode == editorNormal {
		return normalCursorText(value, visibleCursor)
	}
	return cursorMarkedText(value, visibleCursor)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// truncateTitle keeps the identifying start of a Task title. The caller
// renders the synthetic ellipsis separately so it remains distinguishable from
// title content.
func truncateTitle(s string, maxW int) (string, bool) {
	if lipgloss.Width(s) <= maxW {
		return s, false
	}
	return truncatePlain(s, max(0, maxW-lipgloss.Width("…"))), true
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
	if !m.inOverview() && m.taskFilter != overviewAll {
		headerText = fmt.Sprintf("Project Tasks · %s", m.projectFilterLabel())
	}
	if m.inOverview() {
		headerText = fmt.Sprintf("Overview · %s · %s", m.overviewSortLabel(), m.overviewFilterLabel())
		if m.overviewLabel.ID != 0 {
			headerText = fmt.Sprintf("Overview · Label: %s · %s · %s", m.overviewLabel.Title, m.overviewSortLabel(), m.overviewFilterLabel())
		}
	}
	headerText = truncateFieldValue(headerText, max(1, textW-2))
	header := " " + renderHeading(headerText) + " "
	focus := m.focus == paneTasks

	lines := []string{header}

	if m.taskLoading {
		lines = append(lines, "  loading...")
		return fitBlock(lines, textW, textH)
	}
	if len(m.tasks) == 0 {
		if (!m.inOverview() && m.taskFilter != overviewAll) || (m.inOverview() && m.overviewFilter != overviewAll) {
			lines = append(lines, "  No matching tasks")
		} else {
			lines = append(lines, "  (no tasks)")
		}
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
	// Layout is based on the complete result set, never only the viewport, so
	// scrolling past a long title cannot make metadata columns appear or vanish.
	layoutTasks := append([]vimkunja.Task(nil), m.tasks...)
	if m.inlineTitle != nil {
		for i := range layoutTasks {
			if layoutTasks[i].ID == m.inlineTitle.task.ID {
				layoutTasks[i].Title = m.inlineTitle.title
				break
			}
		}
	}
	layout := taskMetadataLayoutForWidth(layoutTasks, contentWidth, m.inOverview(), m.currentTime())
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
			s := completedTaskStyle()
			segStyle = &s
		}
		rend := func(s string) string {
			if segStyle != nil {
				return segStyle.Render(s)
			}
			return s
		}

		titleContent, titleTruncated := truncateTitle(titleText, layout.titleWidth)
		title := rend(titleContent)
		ellipsisWidth := 0
		if titleTruncated {
			title += lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Faint(true).Render("…")
			ellipsisWidth = lipgloss.Width("…")
		}
		title += rend(strings.Repeat(" ", max(0, layout.titleWidth-lipgloss.Width(titleContent)-ellipsisWidth)))

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
		segments = append(segments, title)
		if layout.showDue {
			segments = append(segments, dueSeg)
		}
		line := strings.Join(segments, " ")
		if t.Done {
			line = completedTaskStyle().Render(ansi.Strip(line))
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

const minimumTaskTitleWidth = 18

// taskMetadataLayoutForWidth reserves an 18-cell title budget before showing
// optional metadata in narrow Panes. Its shared layout covers the complete
// result set, keeping columns stable while the viewport scrolls.
func taskMetadataLayoutForWidth(tasks []vimkunja.Task, textW int, overview bool, now time.Time) taskMetadataLayout {
	layout := taskMetadataLayout{showPriority: true, showProject: overview, showDue: true}
	dueWidth := 1
	for _, task := range tasks {
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
	prefixWidth := lipgloss.Width("[ ] ")
	for textW < prefixWidth+minimumTaskTitleWidth+metadataWidth() {
		switch {
		case layout.showProject:
			layout.showProject = false
		case layout.showDue:
			layout.showDue = false
		case layout.showPriority:
			layout.showPriority = false
		default:
			layout.titleWidth = max(1, textW-prefixWidth)
			return layout
		}
	}
	layout.titleWidth = max(1, textW-prefixWidth-metadataWidth())
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
	if m.editor.isPickerMode() {
		return renderDetailPickerOverlay(m, textW, textH)
	}
	if textH < 12 {
		return renderCompactTaskDetail(m, textW, textH)
	}
	e := m.editor
	header := "TASK EDITOR"
	if e.id == 0 {
		header = "NEW TASK"
	}
	// At 12–13 rows preserve the regular metadata layout and the pinned
	// Comments entry by sacrificing decorative blank rows and Description body
	// rows first.
	constrainedRegular := e.id != 0 && textH < 14
	lines := []string{" " + renderHeading(header) + " "}
	if !constrainedRegular {
		lines = append(lines, "")
	}
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
	} else if !e.dueDatePickerActive() && !(e.mode == editorInsert && e.field == fieldDueDate) {
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
		if field == fieldDueDate && e.dueDatePickerActive() && e.dueDatePickerSelected != 0 {
			active = false
		}
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
		plainTitles := make([]string, 0, len(e.labels))
		styledTitles := make([]string, 0, len(e.labels))
		for i, label := range e.labels {
			plainTitles = append(plainTitles, label.Title)
			title := label.Title
			if detailFocused && e.field == fieldLabels && i == clamp(e.labelSelected, 0, len(e.labels)-1) {
				title = lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Bold(true).Render(title)
			}
			styledTitles = append(styledTitles, title)
		}
		// Reserve the Detail scrollbar rail when labels are long enough to need
		// truncation, so the ellipsis remains visible beside that rail.
		available := textW - lipgloss.Width("Labels: ") - 2
		plain := strings.Join(plainTitles, ", ")
		if lipgloss.Width(plain) > available {
			labels = truncateFieldValue(plain, available)
		} else {
			labels = strings.Join(styledTitles, ", ")
		}
	}
	renderMetadata := func(field editorField, value string) string {
		if e.draft.Done && !(detailFocused && e.field == field) {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Faint(true).Render(ansi.Strip(value))
		}
		return value
	}
	titleWidth := textW - lipgloss.Width("Title:    ")
	if detailFocused && e.field == fieldTitle {
		titleWidth -= lipgloss.Width("▸ ")
	}
	title := m.editorFieldViewport(fieldTitle, e.draft.Title, titleWidth)
	lines = append(lines,
		renderField(fieldProject, "Project:  ", projectName),
		renderField(fieldTitle, "Title:    ", renderMetadata(fieldTitle, title)),
		renderField(fieldPriority, "Priority: ", renderMetadata(fieldPriority, priority)),
		renderField(fieldDueDate, "Due Date: ", renderMetadata(fieldDueDate, m.editorText(fieldDueDate, due))),
	)
	if e.dueDatePickerActive() {
		lines = append(lines, renderDueDatePickerChoices(m)...)
	}
	lines = append(lines,
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
	// A short Detail Pane gives its active picker the description rows; at
	// normal heights retain the usual Description anchor above the picker.
	if !e.isPickerMode() || textH >= len(lines)+10 {
		if !constrainedRegular {
			lines = append(lines, "")
		}
		lines = append(lines, renderField(fieldDescription, "Description:", ""))
	}
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
		lines = append(lines, renderDetailPickerChoices(choices, selected, tuiTheme.selectionAccent, textW, textH, len(lines))...)
	} else if e.mode == editorPriorityPicker {
		choices := make([]string, len(tuiTheme.priority))
		for priority := range tuiTheme.priority {
			choices[priority] = priorityName(int64(priority))
		}
		lines = append(lines, renderDetailPickerChoices(choices, e.priorityIndex, priorityColor(e.priorityIndex), textW, textH, len(lines))...)
	} else if e.mode == editorWorkflowStatusPicker {
		choices := make([]string, 0, len(e.workflowStatus.Buckets))
		for _, bucket := range e.workflowStatus.Buckets {
			choices = append(choices, bucket.Title)
		}
		lines = append(lines, renderDetailPickerChoices(choices, e.workflowStatusIndex, tuiTheme.selectionAccent, textW, textH, len(lines))...)
	} else if description := projectDescription(e.draft.Description, map[int]bool{}); description == "" {
		if availableDetailDescriptionRows(textH, len(lines), e.id != 0) > 0 {
			lines = append(lines, placeholderText("  (no description)"))
		}
	} else {
		layout := m.taskDetailDescriptionLayout(textW, textH)
		if layout.rows == 0 {
			// The Description label remains an entry point; its body yields to
			// metadata and Comments in the smallest regular layout.
			return renderPinnedCommentsDetail(m, lines, textW, textH)
		}
		descriptionScrollable = layout.scrollable
		descriptionScrollTop = layout.start
		descriptionRows = len(layout.lines)
		descriptionViewportStart = layout.descriptionRow
		descriptionViewportRows = layout.rows
		end := min(len(layout.lines), layout.start+layout.rows)
		for _, line := range layout.lines[layout.start:end] {
			lines = append(lines, m.editorDescriptionText(line))
		}
	}
	if descriptionScrollable {
		lines = appendPinnedCommentsDetail(m, lines, textW, textH)
		return renderVerticalScrollbar(lines, textW, textH, descriptionScrollTop, descriptionRows, descriptionViewportStart, descriptionViewportRows, m.focus == paneDetail)
	}
	return renderPinnedCommentsDetail(m, lines, textW, textH)
}

func availableDetailDescriptionRows(textH, usedRows int, hasComments bool) int {
	reservedCommentsRows := 0
	if hasComments {
		reservedCommentsRows = 2
	}
	return max(0, textH-usedRows-reservedCommentsRows)
}

func renderPinnedCommentsDetail(m *model, lines []string, textW, textH int) string {
	return fitBlock(appendPinnedCommentsDetail(m, lines, textW, textH), textW, textH)
}

func appendPinnedCommentsDetail(m *model, lines []string, textW, textH int) []string {
	e := m.editor
	if e == nil || e.id == 0 || e.isPickerMode() {
		return lines
	}
	for len(lines) < textH-2 {
		lines = append(lines, "")
	}
	divider := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Render(strings.Repeat("─", max(1, textW)))
	commentCount := fmt.Sprintf("Comments (%d)", len(e.comments))
	if e.commentsLoading {
		commentCount = "Comments (…)"
	}
	return append(lines, divider, focusedField(commentCount, m.focus == paneDetail && e.field == fieldComments))
}

// renderDetailPickerOverlay layers a field-local picker over the existing
// Detail content. Pickers must not consume Description rows or become part of
// the Detail Pane's scrollable document.
func renderDetailPickerOverlay(m *model, textW, textH int) string {
	e := m.editor
	mode := e.mode
	e.mode = editorNormal
	background := renderTaskDetail(m, textW, textH)
	e.mode = mode

	rows, selected, choices := detailPickerOverlayContent(m)
	contentWidth := detailPickerContentWidth(rows, choices, textW)
	_, viewportHeight := m.viewportSize()
	if textW < modalFallbackWidth || viewportHeight < modalFallbackHeight {
		for index, row := range rows {
			rows[index] = padPlainRight(ansi.Truncate(row, contentWidth, ""), contentWidth)
		}
		return fitBlock(append(rows, renderDetailPickerChoiceRows(choices, selected, e.mode, contentWidth)...), textW, textH)
	}
	maxChoiceRows := max(1, textH-2-len(rows))
	if len(choices) > maxChoiceRows {
		// The scrollbar owns the final inner column without shortening choice
		// labels merely because the picker happens to overflow.
		contentWidth = min(contentWidth+1, max(1, min(36, textW-4)))
	}
	for index, row := range rows {
		rows[index] = padPlainRight(ansi.Truncate(row, contentWidth, ""), contentWidth)
	}
	start := 0
	if len(choices) > maxChoiceRows && selected >= 0 {
		start = clamp(selected-maxChoiceRows+1, 0, len(choices)-maxChoiceRows)
	}
	end := min(len(choices), start+maxChoiceRows)
	rows = append(rows, renderDetailPickerChoiceViewport(choices, selected, start, end, e.mode, contentWidth)...)
	popover := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(paneBorderColor(true))).
		Padding(0, 1).
		Render(strings.Join(rows, "\n"))

	popoverLines := strings.Split(popover, "\n")
	backgroundLines := strings.Split(background, "\n")
	anchor := detailPickerAnchor(backgroundLines, e.field)
	top := anchor + 1
	if below := len(backgroundLines) - top; below < len(popoverLines) {
		top = max(0, anchor-len(popoverLines))
	}
	if top+len(popoverLines) > len(backgroundLines) {
		top = max(0, len(backgroundLines)-len(popoverLines))
	}
	left := min(max(0, detailPickerAnchorColumn(e.field)), max(0, textW-lipgloss.Width(popoverLines[0])))
	for i, line := range popoverLines {
		row := top + i
		if row >= len(backgroundLines) {
			break
		}
		base := backgroundLines[row]
		if padding := textW - lipgloss.Width(base); padding > 0 {
			base += strings.Repeat(" ", padding)
		}
		backgroundLines[row] = ansi.Cut(base, 0, left) + line + ansi.Cut(base, left+lipgloss.Width(line), textW)
	}
	return strings.Join(backgroundLines, "\n")
}

func detailPickerOverlayContent(m *model) (rows []string, selected int, choices []string) {
	e := m.editor
	switch e.mode {
	case editorProjectPicker:
		for _, project := range m.projects {
			choices = append(choices, project.Title)
		}
		if len(choices) == 0 {
			return nil, -1, []string{"No Projects"}
		}
		return nil, e.pickerIndex, choices
	case editorPriorityPicker:
		for priority := range tuiTheme.priority {
			choices = append(choices, priorityName(int64(priority)))
		}
		return nil, e.priorityIndex, choices
	case editorWorkflowStatusPicker:
		for _, bucket := range e.workflowStatus.Buckets {
			choices = append(choices, bucket.Title)
		}
		if len(choices) == 0 {
			return nil, -1, []string{"No workflow statuses"}
		}
		return nil, e.workflowStatusIndex, choices
	default:
		return nil, -1, nil
	}
}

func detailPickerContentWidth(rows, choices []string, textW int) int {
	width := 0
	for _, row := range rows {
		width = max(width, lipgloss.Width(row))
	}
	for _, choice := range choices {
		width = max(width, lipgloss.Width("▸ "+choice))
	}
	return min(max(1, width), max(1, min(36, textW-4)))
}

func renderDetailPickerChoiceRows(choices []string, selected int, mode editorMode, width int) []string {
	width = max(1, width)
	rows := make([]string, 0, len(choices))
	for index, choice := range choices {
		prefix := "  "
		if index == selected {
			prefix = "▸ "
		}
		choice = ansi.Truncate(choice, max(1, width-lipgloss.Width(prefix)), "")
		row := padPlainRight(prefix+choice, width)
		if index == selected {
			color := tuiTheme.selectionAccent
			if mode == editorPriorityPicker {
				color = priorityColor(selected)
			}
			row = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render(row)
		}
		rows = append(rows, row)
	}
	return rows
}

// renderDetailPickerChoiceViewport reserves its inner-right column for the
// picker rail. Unlike the Detail Pane scrollbar, this rail belongs to the
// popover and must stay visible even when only a few choices fit.
func renderDetailPickerChoiceViewport(choices []string, selected, start, end int, mode editorMode, width int) []string {
	visible := choices[start:end]
	if len(choices) <= len(visible) || width < 2 {
		return renderDetailPickerChoiceRows(visible, selected-start, mode, width)
	}

	rows := renderDetailPickerChoiceRows(visible, selected-start, mode, width-1)
	thumbStart, thumbEnd := scrollbarThumb(start, len(choices), len(visible))
	railStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.scrollbarRail))
	thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.scrollbarThumbFocused))
	for index, row := range rows {
		glyph := railStyle.Render("│")
		if index >= thumbStart && index < thumbEnd {
			glyph = thumbStyle.Render("█")
		}
		rows[index] = row + glyph
	}
	return rows
}

func detailPickerAnchor(lines []string, field editorField) int {
	labels := map[editorField]string{
		fieldProject: "Project:", fieldPriority: "Priority:", fieldWorkflowStatus: "Status:",
	}
	for index, line := range lines {
		if strings.Contains(ansi.Strip(line), labels[field]) {
			return index
		}
	}
	return len(lines) / 2
}

func detailPickerAnchorColumn(field editorField) int {
	switch field {
	case fieldProject:
		return lipgloss.Width("Project:  ")
	case fieldPriority:
		return lipgloss.Width("Priority: ")
	case fieldWorkflowStatus:
		return lipgloss.Width("Status:   ")
	default:
		return 0
	}
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
	if e.descriptionActive && textH < 12 {
		return renderFocusedCompactDescription(m, textW, textH)
	}
	header := "TASK EDITOR"
	if e.id == 0 {
		header = "NEW TASK"
	}
	lines := []string{" " + renderHeading(header) + " ", ""}
	detailFocused := m.focus == paneDetail
	renderField := func(field editorField, label, value string) string {
		active := detailFocused && e.field == field
		if field == fieldDueDate && e.dueDatePickerActive() && e.dueDatePickerSelected != 0 {
			active = false
		}
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
	} else if !e.dueDatePickerActive() && !(e.mode == editorInsert && e.field == fieldDueDate) {
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
		case fieldComments:
			// The Comments entry is pinned after the Description content below.
		default:
			value := values[field]
			body = append(body, renderField(field, value.label, value.value))
			if field == fieldDueDate && e.dueDatePickerActive() {
				if e.dueDatePickerSelected > 0 {
					pickerSelectedRow = len(body) + e.dueDatePickerSelected - 1
				}
				body = append(body, renderDueDatePickerChoices(m)...)
			}
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
			body = append(body, renderPickerChoicesFitted(choices, e.pickerIndex, tuiTheme.selectionAccent, textW)...)
		case field == fieldPriority && e.mode == editorPriorityPicker:
			pickerSelectedRow = len(body) + 1 + clamp(e.priorityIndex, 0, len(tuiTheme.priority)-1)
			body = append(body, renderPriorityPickerFitted(e, textW)...)
		case field == fieldWorkflowStatus && e.mode == editorWorkflowStatusPicker:
			choices := make([]string, 0, len(e.workflowStatus.Buckets))
			for _, bucket := range e.workflowStatus.Buckets {
				choices = append(choices, bucket.Title)
			}
			if len(choices) > 0 {
				pickerSelectedRow = len(body) + 1 + clamp(e.workflowStatusIndex, 0, len(choices)-1)
			}
			body = append(body, renderPickerChoicesFitted(choices, e.workflowStatusIndex, tuiTheme.selectionAccent, textW)...)
		}
	}
	pinnedCommentRows := compactPinnedCommentRows(textH, e.id != 0)
	staticRows := len(fields)
	if e.id != 0 {
		staticRows-- // Comments is pinned outside the scrollable Detail document.
	}
	bodyRows := max(1, textH-2-pinnedCommentRows)
	descriptionRows := compactDescriptionRows(textH-pinnedCommentRows, staticRows)
	description := e.draft.Description
	desc := wrapTaskDescriptionForEditor(e, description, max(1, textW-2))
	if description == "" {
		desc = []descriptionLine{{text: placeholderText("  (no description)")}}
	}
	descriptionScrollable := len(desc) > descriptionRows
	if descriptionScrollable {
		desc = wrapTaskDescriptionForEditor(e, description, max(1, textW-4))
	}
	start := 0
	if (e.descriptionActive || e.descViewportScroll) && len(desc) > descriptionRows {
		start = clamp(e.descScroll, 0, len(desc)-descriptionRows)
	}
	for row := 0; row < descriptionRows; row++ {
		if line := start + row; line < len(desc) {
			body = append(body, "  "+m.editorDescriptionText(desc[line]))
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
	if e.id != 0 {
		for len(lines) < textH-pinnedCommentRows {
			lines = append(lines, "")
		}
		if pinnedCommentRows == 2 {
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Render(strings.Repeat("─", max(1, textW))))
		}
		commentCount := fmt.Sprintf("(%d)", len(e.comments))
		if e.commentsLoading {
			commentCount = "(…)"
		}
		lines = append(lines, renderField(fieldComments, "Comments ", commentCount))
	}
	if e.descriptionActive && descriptionScrollable && descriptionRow >= e.detailScroll && descriptionRow < e.detailScroll+bodyRows {
		return renderVerticalScrollbar(lines, textW, textH, start, len(desc), 2+descriptionRow-e.detailScroll+1, descriptionRows, true)
	}
	if outerLimit > 0 {
		return renderVerticalScrollbar(lines, textW, textH, e.detailScroll, len(body), 2, bodyRows, detailFocused)
	}
	return fitBlock(lines, textW, textH)
}

// renderFocusedCompactDescription gives Description editing the entire tiny
// Detail Pane. The metadata and Comments entry remain available after Esc.
func renderFocusedCompactDescription(m *model, textW, textH int) string {
	e := m.editor
	mode := "NORMAL"
	if e.mode == editorInsert {
		mode = "INSERT"
	}
	lines := []string{" " + renderHeading("DESCRIPTION · "+mode) + " ", ""}
	available := max(1, textH-len(lines))
	description := e.draft.Description
	desc := wrapTaskDescriptionForEditor(e, description, textW)
	if description == "" {
		desc = []descriptionLine{{text: ""}}
	}
	if len(desc) > available {
		desc = wrapTaskDescriptionForEditor(e, description, max(1, textW-2))
	}
	cursorRow := 0
	for row, line := range desc {
		if e.cursor >= line.start && e.cursor <= line.end {
			cursorRow = row
		}
	}
	start := clamp(e.descScroll, 0, max(0, len(desc)-available))
	if !e.descViewportScroll {
		start = descriptionViewportStart(desc, available, e.descScroll, cursorRow, e.mode == editorNormal && e.cursor == e.fieldRuneLen())
	}
	e.descScroll = start
	for row := 0; row < available; row++ {
		if line := start + row; line < len(desc) {
			lines = append(lines, m.editorDescriptionText(desc[line]))
		} else {
			lines = append(lines, "")
		}
	}
	if len(desc) > available {
		return renderVerticalScrollbar(lines, textW, textH, start, len(desc), 2, available, true)
	}
	return fitBlock(lines, textW, textH)
}

// renderDetailPickerChoices limits a picker to the space left below the
// Detail's fixed fields. Its window follows selection, rather than allowing
// fitBlock to silently cut off choices below the viewport.
func renderDetailPickerChoices(choices []string, selected int, selectionColor string, textW, textH, precedingRows int) []string {
	// Leave one row of slack below the rounded frame: fitBlock owns the final
	// Detail row and otherwise clips the picker's bottom border.
	choiceRows := max(1, textH-precedingRows-3) // rounded picker frame + slack
	start := 0
	if len(choices) > choiceRows && selected >= 0 {
		start = clamp(selected-choiceRows+1, 0, len(choices)-choiceRows)
	}
	end := min(len(choices), start+choiceRows)
	return renderPickerChoicesFitted(choices[start:end], selected-start, selectionColor, textW)
}

// renderPriorityPicker keeps the picker visually separate from Description and
// colours the active choice with the same semantic mapping used everywhere
// else priorities are displayed.
func renderPriorityPicker(e *taskEditor) []string {
	return renderPriorityPickerFitted(e, 0)
}

func renderPriorityPickerFitted(e *taskEditor, width int) []string {
	choices := make([]string, len(tuiTheme.priority))
	for priority := range tuiTheme.priority {
		choices[priority] = priorityName(int64(priority))
	}
	return renderPickerChoicesFitted(choices, e.priorityIndex, priorityColor(e.priorityIndex), width)
}

// renderPickerChoices draws the common compact picker frame. Its active border
// uses the shared selection accent; Project and Status use that accent for
// their active choice, while Priority uses its semantic choice colour.
func renderPickerChoices(choices []string, selected int, selectionColor string) []string {
	return renderPickerChoicesFitted(choices, selected, selectionColor, 0)
}

// renderPickerChoicesFitted drops picker chrome below the shared modal width
// threshold and bounds every choice to the available Detail Pane width.
func renderPickerChoicesFitted(choices []string, selected int, selectionColor string, width int) []string {
	rows := make([]string, 0, len(choices))
	bare := width > 0 && width < modalFallbackWidth
	for index, choice := range choices {
		prefix := "  "
		if index == selected {
			prefix = "▸ "
		}
		if width > 0 {
			choice = ansi.Truncate(choice, max(1, width-lipgloss.Width(prefix)), "")
		}
		row := prefix + choice
		if index == selected {
			row = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(selectionColor)).Render(row)
		}
		rows = append(rows, row)
	}
	if bare {
		return rows
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(paneBorderColor(true))).
		Padding(0, 1)
	return strings.Split(style.Render(strings.Join(rows, "\n")), "\n")
}

func (e *taskEditor) dueDatePickerActive() bool {
	return e.mode == editorDueDatePicker && e.field == fieldDueDate
}

func renderDueDatePickerChoices(m *model) []string {
	e := m.editor
	labels := []string{"In days: ", "In weeks: ", "In months: "}
	rows := make([]string, 0, len(labels))
	for index, label := range labels {
		selected := e.dueDatePickerSelected == index+1
		row := "  " + label
		input := e.dueDatePickerInputs[index]
		if selected && m.cursorVisible && !m.nativeEditorCursorActive() {
			input = cursorMarkedText(input, e.cursor)
		}
		if input == "" {
			input = placeholderText("n")
		}
		row += input
		if selected {
			row = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render("▸ " + strings.TrimPrefix(row, "  "))
		}
		rows = append(rows, row)
	}
	return rows
}

func placeholderText(value string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.muted)).Faint(true).Render(value)
}

// editorText marks the character at the insertion point instead of inserting a
// printable cursor glyph, so the text never shifts while the cursor blinks.
func (m *model) editorText(field editorField, value string) string {
	if m.editor == nil || (m.editor.mode != editorInsert && !(m.editor.dueDatePickerActive() && m.editor.dueDatePickerSelected == 0)) || m.editor.field != field || !m.cursorVisible || m.nativeEditorCursorActive() {
		return value
	}
	return cursorMarkedText(value, m.editor.cursor)
}

type descriptionLine struct {
	text       string
	start, end int
}

type multilineTextLayout struct {
	lines                   []descriptionLine
	start                   int
	cursorRow, cursorColumn int
}

// layoutMultilineText is the shared seam between multiline rendering and
// cursor placement. Callers remain responsible for their surrounding chrome.
func layoutMultilineText(value string, width, rows, scroll, cursor int) multilineTextLayout {
	lines := wrapDescription(value, width)
	start := clamp(scroll, 0, max(0, len(lines)-max(0, rows)))
	cursor = clamp(cursor, 0, len([]rune(value)))
	cursorRow, cursorColumn := 0, 0
	for row, line := range lines {
		if cursor >= line.start && cursor <= line.end {
			cursorRow, cursorColumn = row, cursor-line.start
			break
		}
	}
	return multilineTextLayout{
		lines: lines, start: start,
		cursorRow: cursorRow, cursorColumn: cursorColumn,
	}
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
		lines := wrapTaskDescriptionForEditor(m.editor, m.editor.draft.Description, w-4)
		if len(lines) > descriptionRows {
			lines = wrapTaskDescriptionForEditor(m.editor, m.editor.draft.Description, max(1, w-6))
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
	lines := wrapTaskDescriptionForEditor(m.editor, m.editor.draft.Description, textW)
	if len(lines) > available {
		lines = wrapTaskDescriptionForEditor(m.editor, m.editor.draft.Description, max(1, textW-2))
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
	if m.editor.mode == editorInsert {
		return cursorMarkedText(line.text, cursor)
	}
	return line.text
}

// normalCursorText is Vim's static block cursor: white background and black
// glyph, without adding a printable character to the line.
func normalCursorText(value string, cursor int) string {
	r := []rune(value)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.dark)).Background(lipgloss.Color(tuiTheme.bright))
	if len(r) == 0 {
		return style.Render(" ")
	}
	cursor = clamp(cursor, 0, len(r))
	index := cursor
	if index == len(r) {
		index--
	}
	block := style.Render(string(r[index]))
	return string(r[:index]) + block + string(r[index+1:])
}

func cursorMarkedText(value string, cursor int) string {
	r := []rune(value)
	cursor = clamp(cursor, 0, len(r))
	marker := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render("│")
	return string(r[:cursor]) + marker + string(r[cursor:])
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
