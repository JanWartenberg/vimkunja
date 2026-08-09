package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

type editorField int

const (
	fieldProject editorField = iota
	fieldTitle
	fieldPriority
	fieldDueDate
	fieldLabels
	fieldWorkflowStatus
	fieldDone
	fieldDescription
)

type editorMode int

const (
	editorNormal editorMode = iota
	editorInsert
	editorProjectPicker
	editorPriorityPicker
	editorWorkflowStatusPicker
)

type taskEditor struct {
	id        int64
	taskIndex int64
	original  vimkunja.TaskDraft
	draft     vimkunja.TaskDraft
	field     editorField
	mode      editorMode
	cursor    int // rune offset in the active text field

	projectFilter         string
	pickerIndex           int
	priorityIndex         int
	workflowStatusIndex   int
	descScroll            int
	labelSelected         int
	labels                []vimkunja.Label
	labelsLoading         bool
	gArmed                bool
	workflowStatus        vimkunja.WorkflowStatus
	workflowStatusLoading bool
}

func (m *model) openNewEditor() {
	draft := vimkunja.TaskDraft{ProjectID: m.currentProjectID}
	field := fieldTitle
	if draft.ProjectID == 0 {
		field = fieldProject
	}
	m.editor = &taskEditor{draft: draft, original: draft, field: field}
	m.showDetail = true
	m.focus = paneDetail
}

func (m *model) openExistingEditor(task vimkunja.Task) {
	draft := vimkunja.TaskDraft{
		ID: task.ID, ProjectID: task.ProjectID, Title: task.Title,
		Description: task.Description, Priority: task.Priority, DueDate: task.DueDate, Done: task.Done,
	}
	m.editor = &taskEditor{id: task.ID, taskIndex: task.Index, draft: draft, original: cloneTaskDraft(draft), field: fieldTitle, workflowStatusLoading: true, labelsLoading: true}
	m.showDetail = true
	m.focus = paneDetail
}

func (e *taskEditor) workflowStatusLabel() string {
	if e.workflowStatusLoading {
		return "loading... (read-only)"
	}
	switch e.workflowStatus.Availability {
	case vimkunja.WorkflowStatusUnavailable:
		return "unavailable (no manual Kanban View; read-only)"
	case vimkunja.WorkflowStatusAmbiguous:
		return "ambiguous (multiple manual Kanban Views; read-only)"
	case vimkunja.WorkflowStatusAvailable:
		for _, bucket := range e.workflowStatus.Buckets {
			if bucket.ID == e.draft.WorkflowBucketID {
				return bucket.Title
			}
		}
		return "unassigned"
	default:
		return "unavailable"
	}
}

func (e *taskEditor) dirty() bool {
	return e.draft.ProjectID != e.original.ProjectID || e.draft.Title != e.original.Title ||
		e.draft.Description != e.original.Description || e.draft.Priority != e.original.Priority ||
		e.draft.DueDate != e.original.DueDate || e.draft.WorkflowBucketID != e.original.WorkflowBucketID ||
		e.draft.Done != e.original.Done || !sameLabelIDs(e.draft.LabelIDs, e.original.LabelIDs)
}

func (m *model) updateDetailKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	if e == nil { // Defensive fallback for a detail pane opened by old state.
		if k.String() == "enter" && m.taskSelected >= 0 && m.taskSelected < len(m.tasks) {
			m.openExistingEditor(m.tasks[m.taskSelected])
		}
		return m, nil
	}
	switch e.mode {
	case editorProjectPicker:
		return m.updateProjectPicker(k)
	case editorPriorityPicker:
		return m.updatePriorityPicker(k)
	case editorWorkflowStatusPicker:
		return m.updateWorkflowStatusPicker(k)
	case editorInsert:
		return m.updateEditorInsert(k)
	default:
		return m.updateEditorNormal(k)
	}
}

func (m *model) updateEditorNormal(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	s := k.String()
	if e.gArmed {
		e.gArmed = false
		if s == "g" && e.field == fieldDescription {
			e.cursor = 0
			e.descScroll = 0
		}
		return m, nil
	}
	if s == "esc" {
		if e.dirty() {
			m.recordError(errUnsavedNavigation)
			return m, nil
		}
		m.closeTaskDetail()
		return m, nil
	}
	if e.field == fieldDescription && descriptionHasHTML(e.draft.Description) {
		limit := m.readOnlyDescriptionScrollLimit()
		switch s {
		case "G", "end":
			e.descScroll = limit
		case "home":
			e.descScroll = 0
		case "j", "down":
			e.descScroll = clamp(e.descScroll+1, 0, limit)
		case "k", "up":
			if e.descScroll == 0 {
				m.moveField(-1)
			} else {
				e.descScroll--
			}
		case "ctrl+d", "pgdown", "pagedown":
			e.descScroll = clamp(e.descScroll+m.pageStep(), 0, limit)
		case "ctrl+u", "pgup", "pageup":
			e.descScroll = clamp(e.descScroll-m.pageStep(), 0, limit)
		}
		return m, nil
	}
	if e.field == fieldDescription {
		switch s {
		case "a":
			e.moveCursor(1)
			return m.startEditorInsert()
		case "I":
			e.cursor = e.lineStart()
			return m.startEditorInsert()
		case "A":
			e.cursor = e.lineEnd()
			return m.startEditorInsert()
		case "o":
			e.cursor = e.lineEnd()
			e.insertText("\n")
			return m.startEditorInsert()
		case "O":
			e.cursor = e.lineStart()
			e.insertText("\n")
			e.cursor--
			return m.startEditorInsert()
		}
	}
	if s == "x" && e.field == fieldDone {
		e.draft.Done = !e.draft.Done
		e.draft.DoneSet = e.draft.Done != e.original.Done
		return m, nil
	}
	if e.field == fieldLabels {
		switch s {
		case "h", "left":
			e.labelSelected = clamp(e.labelSelected-1, 0, max(0, len(e.labels)-1))
			return m, nil
		case "l", "right":
			e.labelSelected = clamp(e.labelSelected+1, 0, max(0, len(e.labels)-1))
			return m, nil
		case "x":
			m.removeSelectedEditorLabel()
			return m, nil
		}
	}
	if s == "i" {
		switch e.field {
		case fieldProject:
			e.mode = editorProjectPicker
			e.projectFilter = ""
			e.pickerIndex = m.projectPickerIndex(e.draft.ProjectID, "")
		case fieldPriority:
			e.mode = editorPriorityPicker
			e.priorityIndex = int(e.draft.Priority)
		case fieldLabels:
			return m, m.openTaskLabelPicker()
		case fieldWorkflowStatus:
			if e.workflowStatus.Availability == vimkunja.WorkflowStatusAvailable && len(e.workflowStatus.Buckets) > 0 {
				e.mode = editorWorkflowStatusPicker
				e.workflowStatusIndex = e.workflowStatusPickerIndex()
			}
		case fieldDone:
			return m, nil
		default:
			if e.field == fieldDueDate && e.draft.DueDate == "" {
				e.draft.DueDate = "YYYY-MM-DD"
				e.cursor = 0
			} else if e.field != fieldDescription && e.field != fieldTitle {
				// Title and Description preserve their Normal-mode cursor position on `i`.
				e.cursor = e.fieldRuneLen()
			}
			return m.startEditorInsert()
		}
		return m, nil
	}
	if s == "g" && e.field == fieldDescription {
		e.gArmed = true
		return m, nil
	}
	if (s == "G" || s == "end") && e.field == fieldDescription {
		e.cursor = e.fieldRuneLen()
		e.descScroll = max(0, len(strings.Split(e.draft.Description, "\n"))-1)
		return m, nil
	}
	if s == "home" && e.field == fieldDescription {
		e.cursor = 0
		e.descScroll = 0
		return m, nil
	}
	if e.field == fieldTitle {
		switch s {
		case "h", "left":
			e.moveCursor(-1)
			return m, nil
		case "l", "right":
			e.moveCursor(1)
			return m, nil
		}
	}
	if e.field == fieldDescription {
		switch s {
		case "h", "left":
			e.moveCursor(-1)
		case "l", "right":
			e.moveCursor(1)
		case "w":
			e.moveWordForward()
		case "b":
			e.moveWordBackward()
		case "j", "down":
			e.moveCursorLine(1)
		case "k", "up":
			if e.descriptionLine() == 0 {
				m.moveField(-1)
			} else {
				e.moveCursorLine(-1)
			}
		case "ctrl+d", "pgdown", "pagedown":
			e.moveCursorLine(m.pageStep())
		case "ctrl+u", "pgup", "pageup":
			e.moveCursorLine(-m.pageStep())
		}
		return m, nil
	}

	switch s {
	case "j", "down":
		m.moveField(1)
	case "k", "up":
		m.moveField(-1)
	}
	return m, nil
}

func (m *model) updateEditorInsert(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	s := k.String()
	switch s {
	case "ctrl+v":
		return m, m.pasteClipboard(e)
	case "esc":
		e.mode = editorNormal
		return m, nil
	case "enter":
		if e.field != fieldDescription {
			e.mode = editorNormal
			return m, nil
		}
		e.insertText("\n")
		return m, nil
	case "backspace":
		if e.field == fieldDueDate {
			e.deleteDueDateDigit(-1)
		} else {
			e.deleteBeforeCursor()
		}
		return m, nil
	case "delete":
		if e.field == fieldDueDate {
			e.deleteDueDateDigit(0)
		} else {
			e.deleteAtCursor()
		}
		return m, nil
	case "left":
		e.moveCursor(-1)
		return m, nil
	case "right":
		e.moveCursor(1)
		return m, nil
	case "up":
		if e.field == fieldDescription {
			e.moveCursorLine(-1)
		}
		return m, nil
	case "down":
		if e.field == fieldDescription {
			e.moveCursorLine(1)
		}
		return m, nil
	}
	if text := k.Key().Text; text != "" {
		if e.field == fieldDueDate {
			for _, r := range text {
				if r >= '0' && r <= '9' {
					e.insertDueDateDigit(r)
				}
			}
		} else {
			e.insertText(text)
		}
	}
	return m, nil
}

func (m *model) startEditorInsert() (tea.Model, tea.Cmd) {
	m.editor.mode = editorInsert
	m.cursorVisible = true
	return m, cursorBlinkCmd()
}

func (m *model) updateProjectPicker(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	s := k.String()
	matches := m.filteredProjects(e.projectFilter)
	switch s {
	case "esc":
		e.mode = editorNormal
	case "enter":
		if len(matches) > 0 {
			projectID := matches[clamp(e.pickerIndex, 0, len(matches)-1)].ID
			if projectID != e.draft.ProjectID {
				e.draft.ProjectID = projectID
				// Buckets are scoped to a project's Kanban View. A bucket from
				// the previous project must never be saved with this move.
				e.draft.WorkflowBucketID = 0
				e.workflowStatus = vimkunja.WorkflowStatus{}
				e.workflowStatusLoading = false
			}
		}
		e.mode = editorNormal
	case "backspace":
		r := []rune(e.projectFilter)
		if len(r) > 0 {
			e.projectFilter = string(r[:len(r)-1])
		}
		e.pickerIndex = 0
	case "j", "down":
		e.pickerIndex = clamp(e.pickerIndex+1, 0, max(0, len(matches)-1))
	case "k", "up":
		e.pickerIndex = clamp(e.pickerIndex-1, 0, max(0, len(matches)-1))
	default:
		if text := k.Key().Text; text != "" {
			e.projectFilter += text
			e.pickerIndex = 0
		}
	}
	return m, nil
}

func (m *model) updatePriorityPicker(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	s := k.String()
	switch s {
	case "esc":
		e.mode = editorNormal
	case "enter":
		e.draft.Priority = int64(e.priorityIndex)
		e.mode = editorNormal
	case "j", "down":
		e.priorityIndex = clamp(e.priorityIndex+1, 0, 5)
	case "k", "up":
		e.priorityIndex = clamp(e.priorityIndex-1, 0, 5)
	}
	return m, nil
}

func (m *model) updateWorkflowStatusPicker(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	s := k.String()
	buckets := e.workflowStatus.Buckets
	switch s {
	case "esc":
		e.mode = editorNormal
	case "enter":
		if len(buckets) > 0 {
			e.draft.WorkflowBucketID = buckets[clamp(e.workflowStatusIndex, 0, len(buckets)-1)].ID
		}
		e.mode = editorNormal
	case "j", "down":
		e.workflowStatusIndex = clamp(e.workflowStatusIndex+1, 0, max(0, len(buckets)-1))
	case "k", "up":
		e.workflowStatusIndex = clamp(e.workflowStatusIndex-1, 0, max(0, len(buckets)-1))
	}
	return m, nil
}

func (e *taskEditor) isPickerMode() bool {
	switch e.mode {
	case editorProjectPicker, editorPriorityPicker, editorWorkflowStatusPicker:
		return true
	default:
		return false
	}
}

func (e *taskEditor) workflowStatusPickerIndex() int {
	for i, bucket := range e.workflowStatus.Buckets {
		if bucket.ID == e.draft.WorkflowBucketID {
			return i
		}
	}
	return 0
}

func (m *model) saveEditor(closeAfter bool) tea.Cmd {
	e := m.editor
	if e == nil {
		return nil
	}
	if e.draft.ProjectID == 0 || strings.TrimSpace(e.draft.Title) == "" {
		m.recordError(fmt.Errorf("project and title are required"))
		if e.draft.ProjectID == 0 {
			e.field = fieldProject
		} else {
			e.field = fieldTitle
		}
		return nil
	}
	if e.draft.DueDate != "" {
		date, err := time.Parse("2006-01-02", e.draft.DueDate)
		if err != nil || date.Format("2006-01-02") != e.draft.DueDate {
			m.recordError(fmt.Errorf("due date must use YYYY-MM-DD"))
			e.field, e.mode = fieldDueDate, editorInsert
			m.cursorVisible = true
			return cursorBlinkCmd()
		}
	}
	draft := e.draft
	m.taskLoading = true
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		var task vimkunja.Task
		var err error
		if draft.ID == 0 {
			task, err = m.store.CreateTask(ctx, draft)
		} else {
			task, err = m.store.UpdateTask(ctx, draft)
		}
		return taskSavedMsg{task: task, closeAfter: closeAfter, err: err}
	}
}

func (m *model) moveField(delta int) {
	e := m.editor
	// The rendered form is vertical, so j/k and arrow up/down traverse fields.
	field := editorField(clamp(int(e.field)+delta, int(fieldProject), int(fieldDescription)))
	if e.id == 0 && field == fieldDone {
		field = editorField(clamp(int(field)+delta, int(fieldProject), int(fieldDescription)))
	}
	e.field = field
}

func (m *model) filteredProjects(filter string) []vimkunja.Project {
	needle := strings.ToLower(strings.TrimSpace(filter))
	out := make([]vimkunja.Project, 0, len(m.projects))
	for _, p := range m.projects {
		if needle == "" || strings.Contains(strings.ToLower(p.Title), needle) || strings.Contains(strings.ToLower(p.Identifier), needle) {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title) })
	return out
}

func (m *model) projectPickerIndex(projectID int64, filter string) int {
	for i, p := range m.filteredProjects(filter) {
		if p.ID == projectID {
			return i
		}
	}
	return 0
}

func cloneTaskDraft(draft vimkunja.TaskDraft) vimkunja.TaskDraft {
	draft.LabelIDs = append([]int64(nil), draft.LabelIDs...)
	return draft
}

func sameLabelIDs(left, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func (e *taskEditor) fieldValue() string {
	switch e.field {
	case fieldTitle:
		return e.draft.Title
	case fieldDueDate:
		return e.draft.DueDate
	case fieldDescription:
		return e.draft.Description
	}
	return ""
}
func (e *taskEditor) setFieldValue(value string) {
	switch e.field {
	case fieldTitle:
		e.draft.Title = value
	case fieldDueDate:
		e.draft.DueDate = value
	case fieldDescription:
		e.draft.Description = value
	}
}
func (e *taskEditor) fieldRuneLen() int    { return len([]rune(e.fieldValue())) }
func (e *taskEditor) moveCursor(delta int) { e.cursor = clamp(e.cursor+delta, 0, e.fieldRuneLen()) }
func (e *taskEditor) insertText(text string) {
	r := []rune(e.fieldValue())
	add := []rune(text)
	e.cursor = clamp(e.cursor, 0, len(r))
	r = append(r[:e.cursor], append(add, r[e.cursor:]...)...)
	e.setFieldValue(string(r))
	e.cursor += len(add)
}
func (e *taskEditor) deleteBeforeCursor() {
	r := []rune(e.fieldValue())
	if e.cursor > 0 {
		r = append(r[:e.cursor-1], r[e.cursor:]...)
		e.cursor--
		e.setFieldValue(string(r))
	}
}
func (e *taskEditor) deleteAtCursor() {
	r := []rune(e.fieldValue())
	if e.cursor < len(r) {
		r = append(r[:e.cursor], r[e.cursor+1:]...)
		e.setFieldValue(string(r))
	}
}
func (e *taskEditor) lineStart() int {
	r := []rune(e.draft.Description)
	start := clamp(e.cursor, 0, len(r))
	for start > 0 && r[start-1] != '\n' {
		start--
	}
	return start
}

func (e *taskEditor) lineEnd() int {
	r := []rune(e.draft.Description)
	end := clamp(e.cursor, 0, len(r))
	for end < len(r) && r[end] != '\n' {
		end++
	}
	return end
}

func (e *taskEditor) moveWordForward() {
	r := []rune(e.draft.Description)
	end := e.lineEnd()
	pos := clamp(e.cursor, 0, end)
	for pos < end && !isWordSpace(r[pos]) {
		pos++
	}
	for pos < end && isWordSpace(r[pos]) {
		pos++
	}
	e.cursor = pos
}

func (e *taskEditor) moveWordBackward() {
	r := []rune(e.draft.Description)
	start := e.lineStart()
	pos := clamp(e.cursor, start, len(r))
	for pos > start && isWordSpace(r[pos-1]) {
		pos--
	}
	for pos > start && !isWordSpace(r[pos-1]) {
		pos--
	}
	e.cursor = pos
}

func isWordSpace(r rune) bool { return r == ' ' || r == '\t' }

func (e *taskEditor) descriptionLine() int {
	r := []rune(e.draft.Description)
	line := 0
	for i := 0; i < clamp(e.cursor, 0, len(r)); i++ {
		if r[i] == '\n' {
			line++
		}
	}
	return line
}

func (e *taskEditor) moveCursorLine(delta int) {
	r := []rune(e.draft.Description)
	pos := clamp(e.cursor, 0, len(r))
	start := pos
	for start > 0 && r[start-1] != '\n' {
		start--
	}
	col := pos - start
	line := e.descriptionLine()
	lines := strings.Split(e.draft.Description, "\n")
	target := clamp(line+delta, 0, len(lines)-1)
	offset := 0
	for i := 0; i < target; i++ {
		offset += len([]rune(lines[i])) + 1
	}
	e.cursor = offset + min(col, len([]rune(lines[target])))
	e.descScroll = max(0, target)
}

func (e *taskEditor) insertDueDateDigit(digit rune) {
	r := []rune(e.draft.DueDate)
	if len(r) != len("YYYY-MM-DD") {
		r = []rune("YYYY-MM-DD")
		e.cursor = 0
	}
	positions := []int{0, 1, 2, 3, 5, 6, 8, 9}
	posIndex := 0
	for posIndex < len(positions) && positions[posIndex] < e.cursor {
		posIndex++
	}
	if posIndex >= len(positions) {
		return
	}
	r[positions[posIndex]] = digit
	e.draft.DueDate = string(r)
	if posIndex+1 < len(positions) {
		e.cursor = positions[posIndex+1]
	} else {
		e.cursor = len(r)
	}
}

func (e *taskEditor) deleteDueDateDigit(direction int) {
	r := []rune(e.draft.DueDate)
	if len(r) != len("YYYY-MM-DD") {
		return
	}
	positions := []int{0, 1, 2, 3, 5, 6, 8, 9}
	posIndex := 0
	for posIndex < len(positions) && positions[posIndex] < e.cursor {
		posIndex++
	}
	if direction < 0 {
		posIndex--
	}
	if posIndex < 0 || posIndex >= len(positions) {
		return
	}
	placeholders := []rune("YYYY-MM-DD")
	r[positions[posIndex]] = placeholders[positions[posIndex]]
	e.draft.DueDate = string(r)
	e.cursor = positions[posIndex]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func contextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 25*time.Second)
}

func cursorBlinkCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return cursorBlinkMsg{} })
}
