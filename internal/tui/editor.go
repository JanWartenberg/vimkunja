package tui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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
	fieldComments
	fieldMoreFields
)

type editorMode int

const (
	editorNormal editorMode = iota
	editorInsert
	editorProjectPicker
	editorPriorityPicker
	editorWorkflowStatusPicker
	editorDueDatePicker
)

type taskEditor struct {
	id        int64
	taskIndex int64
	original  vimkunja.TaskDraft
	draft     vimkunja.TaskDraft
	field     editorField
	mode      editorMode
	cursor    int // rune offset in the active text field

	projectFilter           string
	fieldViewportStart      int
	pickerIndex             int
	priorityIndex           int
	workflowStatusIndex     int
	dueDatePickerSelected   int
	dueDatePickerDateCursor int
	dueDatePickerInputs     [3]string // days, weeks, months
	descScroll              int
	descViewportScroll      bool // explicit reading scroll, independent of the source cursor
	detailScroll            int
	moreFieldsExpanded      bool
	descriptionActive       bool
	descriptionNavigated    bool
	labelSelected           int
	labels                  []vimkunja.Label
	labelsLoading           bool
	comments                []vimkunja.TaskComment
	commentsLoading         bool
	commentsRequest         uint64
	commentsErr             error
	gArmed                  bool
	dArmed                  bool
	workflowStatus          vimkunja.WorkflowStatus
	workflowStatusLoading   bool
}

func (m *model) openNewEditor() tea.Cmd {
	draft := vimkunja.TaskDraft{ProjectID: m.currentProjectID}
	field := fieldTitle
	if draft.ProjectID == 0 {
		field = fieldProject
	}
	m.editor = &taskEditor{draft: draft, original: draft, field: field, workflowStatusLoading: draft.ProjectID != 0}
	m.showDetail = true
	m.focus = paneDetail
	if draft.ProjectID != 0 {
		return m.loadWorkflowStatus(draft.ProjectID, 0)
	}
	return nil
}

func (m *model) openNewKanbanEditor(bucketID int64) tea.Cmd {
	cmd := m.openNewEditor()
	m.editor.draft.WorkflowBucketID = bucketID
	m.editor.original.WorkflowBucketID = bucketID
	return cmd
}

func (m *model) openExistingEditor(task vimkunja.Task) {
	draft := vimkunja.TaskDraft{
		ID: task.ID, ProjectID: task.ProjectID, Title: task.Title,
		Description: task.Description, Priority: task.Priority, DueDate: task.DueDate, Done: task.Done,
	}
	m.editor = &taskEditor{id: task.ID, taskIndex: task.Index, draft: draft, original: cloneTaskDraft(draft), field: fieldTitle, workflowStatusLoading: true, labelsLoading: true, commentsLoading: true}
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
	if e == nil {
		if m.kanbanActive && !m.inOverview() && (k.String() == "i" || k.String() == "o") {
			if bucket, ok := m.activeKanbanBucket(); ok {
				return m, m.openNewKanbanEditor(bucket.ID)
			}
			return m, nil
		}
		// Defensive fallback for a detail pane opened by old state.
		if k.String() == "enter" && m.taskSelected >= 0 && m.taskSelected < len(m.tasks) {
			task := m.tasks[m.taskSelected]
			m.openExistingEditor(task)
			return m, tea.Batch(m.loadWorkflowStatus(task.ProjectID, task.ID), m.loadEditorLabels(task.ID), m.loadTaskComments(task.ID))
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
	case editorDueDatePicker:
		return m.updateDueDatePicker(k)
	case editorInsert:
		return m.updateEditorInsert(k)
	default:
		return m.updateEditorNormal(k)
	}
}

func (m *model) updateEditorNormal(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	s := k.String()
	if e.field == fieldDescription && (s == "ctrl+e" || s == "ctrl+y") {
		delta := 1
		if s == "ctrl+y" {
			delta = -1
		}
		m.scrollDescriptionViewport(delta)
		return m, nil
	}
	e.descViewportScroll = false
	if e.field == fieldDescription && descriptionHasHTML(e.draft.Description) && (s == "enter" || s == "i") {
		m.recordError(fmt.Errorf("Description is read-only: HTML editing is not supported yet."))
		return m, nil
	}
	if !e.descriptionActive {
		switch s {
		case "ctrl+d", "pgdown", "pagedown":
			m.pageDetail(1)
			return m, nil
		case "ctrl+u", "pgup", "pageup":
			m.pageDetail(-1)
			return m, nil
		}
	}
	if e.gArmed {
		e.gArmed = false
		switch s {
		case "g":
			if e.field == fieldDescription {
				e.cursor = 0
				e.descScroll = 0
			}
		case "t":
			m.focusDetailField(fieldTitle)
			e.cursor = e.fieldRuneLen()
			return m.startEditorInsert()
		case "d":
			m.focusDetailField(fieldDueDate)
			return m.openDueDatePicker()
		case "p":
			m.focusDetailField(fieldPriority)
			e.mode = editorPriorityPicker
			e.priorityIndex = int(e.draft.Priority)
		case "l":
			m.focusDetailField(fieldLabels)
			return m, m.openTaskLabelPicker()
		case "c":
			if e.id == 0 {
				m.recordError(fmt.Errorf("Save Task before adding Comments."))
				return m, nil
			}
			m.focusDetailField(fieldComments)
			m.openComments()
		case "n":
			m.focusDetailField(fieldDescription)
		}
		return m, nil
	}
	if s == "g" {
		e.gArmed = true
		return m, nil
	}
	if e.field == fieldMoreFields {
		switch s {
		case "enter":
			e.moreFieldsExpanded = !e.moreFieldsExpanded
			m.ensureDetailFieldVisible()
		case "j", "down":
			m.moveField(1)
		case "k", "up":
			m.moveField(-1)
		}
		return m, nil
	}
	if e.field == fieldComments {
		switch s {
		case "enter", "i":
			m.openComments()
			return m, nil
		case "a":
			m.openComments()
			m.startCommentComposer()
			return m, cursorBlinkCmd()
		}
	}
	if e.field == fieldDescription && e.descriptionNavigated && !e.descriptionActive {
		switch s {
		case "enter":
			e.descriptionActive = true
			if m.compactDetail() {
				e.moreFieldsExpanded = false
			}
			m.ensureDetailFieldVisible()
		case "i":
			e.descriptionActive = true
			if m.compactDetail() {
				e.moreFieldsExpanded = false
			}
			m.ensureDetailFieldVisible()
			if descriptionHasHTML(e.draft.Description) {
				return m, nil
			}
			return m.startEditorInsert()
		case "j", "down":
			m.moveField(1)
		case "k", "up":
			m.moveField(-1)
		case "esc":
			if e.dirty() {
				m.recordError(errUnsavedNavigation)
				return m, nil
			}
			m.closeTaskDetail()
		}
		return m, nil
	}
	if e.dArmed {
		e.dArmed = false
		if s == "d" {
			if e.field == fieldDescription {
				e.deleteDescriptionLine()
			} else {
				e.clearOptionalField()
			}
		}
		return m, nil
	}
	if s == "esc" {
		if e.field == fieldDescription && e.descriptionActive {
			e.descriptionActive = false
			return m, nil
		}
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
		case "d":
			e.dArmed = true
			return m, nil
		case "x":
			e.deleteAtCursor()
			return m, nil
		case "X":
			e.deleteBeforeCursor()
			return m, nil
		case "D":
			e.deleteToLineEnd()
			return m, nil
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
	if s == "x" {
		if e.clearOptionalField() {
			return m, nil
		}
		if e.field == fieldDone {
			e.draft.Done = !e.draft.Done
			e.draft.DoneSet = e.draft.Done != e.original.Done
			return m, nil
		}
	}
	if s == "d" && e.optionalFieldClearable() {
		e.dArmed = true
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
		case fieldComments:
			m.openComments()
			return m, nil
		case fieldWorkflowStatus:
			if e.workflowStatus.Availability == vimkunja.WorkflowStatusAvailable && len(e.workflowStatus.Buckets) > 0 {
				e.mode = editorWorkflowStatusPicker
				e.workflowStatusIndex = e.workflowStatusPickerIndex()
			}
		case fieldDone:
			return m, nil
		default:
			if e.field == fieldDueDate {
				return m.openDueDatePicker()
			}
			if e.field != fieldDescription && e.field != fieldTitle {
				// Title and Description preserve their Normal-mode cursor position on `i`.
				e.cursor = e.fieldRuneLen()
			}
			return m.startEditorInsert()
		}
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
	case "home":
		if e.field == fieldDescription {
			e.cursor = e.lineStart()
		} else {
			e.cursor = 0
		}
		return m, nil
	case "end":
		if e.field == fieldDescription {
			e.cursor = e.lineEnd()
		} else {
			e.cursor = e.fieldRuneLen()
		}
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
		m.insertEditorText(text)
	}
	return m, nil
}

func (m *model) insertEditorText(text string) {
	e := m.editor
	if e.field == fieldDueDate {
		for _, r := range text {
			if r >= '0' && r <= '9' {
				e.insertDueDateDigit(r)
			}
		}
		return
	}
	e.insertText(text)
}

func (m *model) startEditorInsert() (tea.Model, tea.Cmd) {
	m.editor.mode = editorInsert
	m.cursorVisible = true
	return m, cursorBlinkCmd()
}

func (m *model) updateProjectPicker(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	s := k.String()
	if index, handled := m.updatePickerNavigation(s, e.pickerIndex, len(m.projects)); handled {
		e.pickerIndex = index
		return m, nil
	}
	switch s {
	case "esc":
		e.mode = editorNormal
	case "enter":
		var cmd tea.Cmd
		if len(m.projects) > 0 {
			projectID := m.projects[clamp(e.pickerIndex, 0, len(m.projects)-1)].ID
			if projectID != e.draft.ProjectID {
				e.draft.ProjectID = projectID
				// Buckets are scoped to a project's Kanban View. A bucket from
				// the previous project must never be saved with this move.
				e.draft.WorkflowBucketID = 0
				e.workflowStatus = vimkunja.WorkflowStatus{}
				e.workflowStatusLoading = true
				cmd = m.loadWorkflowStatus(projectID, e.id)
			}
		}
		e.mode = editorNormal
		return m, cmd
	}
	return m, nil
}

func (e *taskEditor) optionalFieldClearable() bool {
	return e.field == fieldDueDate || e.field == fieldPriority
}

func (e *taskEditor) clearOptionalField() bool {
	switch e.field {
	case fieldDueDate:
		e.draft.DueDate = ""
		e.cursor = 0
	case fieldPriority:
		e.draft.Priority = 0
	default:
		return false
	}
	return true
}

func (m *model) focusDetailField(field editorField) {
	e := m.editor
	e.field = field
	e.descriptionActive = false
	e.descriptionNavigated = field == fieldDescription
	if m.compactDetail() && field != fieldTitle && field != fieldDescription {
		e.moreFieldsExpanded = true
	}
	m.ensureDetailFieldVisible()
}

func (m *model) openDueDatePicker() (tea.Model, tea.Cmd) {
	e := m.editor
	if e.draft.DueDate == "" {
		e.draft.DueDate = "YYYY-MM-DD"
	}
	// The shared cursor may be past the date mask after editing another
	// field or completing a date. Start each new picker session at its first
	// digit; preserve the date cursor only when switching rows within it.
	e.cursor = 0
	e.dueDatePickerSelected = 0
	e.dueDatePickerInputs = [3]string{}
	e.dueDatePickerDateCursor = e.cursor
	e.mode = editorDueDatePicker
	m.cursorVisible = true
	return m, cursorBlinkCmd()
}

// updateDueDatePicker combines the existing masked date entry with relative
// helpers. Relative values remain a draft until Enter, so changing a helper
// selection or pressing Esc cannot overwrite a manually entered date.
func (m *model) updateDueDatePicker(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	s := k.String()
	// Home and End are text motions elsewhere, not picker-row shortcuts here.
	// Keep the date picker free of vertical Home/End navigation; j/k select
	// helpers and h/l (or arrows) move across the manual date mask.
	if s != "home" && s != "end" {
		if index, handled := m.updatePickerNavigation(s, e.dueDatePickerSelected, 4); handled {
			if e.dueDatePickerSelected == 0 {
				e.dueDatePickerDateCursor = e.cursor
			}
			e.dueDatePickerSelected = index
			if index > 0 && index < 4 {
				e.cursor = len([]rune(e.dueDatePickerInputs[index-1]))
			} else if index == 0 {
				e.cursor = clamp(e.dueDatePickerDateCursor, 0, e.fieldRuneLen())
			}
			return m, nil
		}
	}
	switch s {
	case "home", "end":
		return m, nil
	case "ctrl+v":
		return m, m.pasteClipboard(e)
	case "esc":
		e.mode = editorNormal
		return m, nil
	case "enter":
		if e.dueDatePickerSelected == 0 {
			e.mode = editorNormal
			return m, nil
		}
		input := e.dueDatePickerInputs[e.dueDatePickerSelected-1]
		n, err := strconv.Atoi(input)
		if err != nil || n < 0 {
			m.recordError(fmt.Errorf("enter a non-negative whole number"))
			return m, nil
		}
		now := m.currentTime()
		switch e.dueDatePickerSelected {
		case 1:
			e.draft.DueDate = now.AddDate(0, 0, n).Format("2006-01-02")
		case 2:
			e.draft.DueDate = now.AddDate(0, 0, 7*n).Format("2006-01-02")
		case 3:
			e.draft.DueDate = now.AddDate(0, n, 0).Format("2006-01-02")
		}
		e.mode = editorNormal
		return m, nil
	case "backspace":
		if e.dueDatePickerSelected == 0 {
			e.deleteDueDateDigit(-1)
		} else if e.dueDatePickerSelected < 4 {
			e.setDueDatePickerInput(deleteTextBeforeCursor(e.dueDatePickerInput(), &e.cursor))
		}
		return m, nil
	case "delete":
		if e.dueDatePickerSelected == 0 {
			e.deleteDueDateDigit(0)
		} else if e.dueDatePickerSelected < 4 {
			e.setDueDatePickerInput(deleteTextAtCursor(e.dueDatePickerInput(), &e.cursor))
		}
		return m, nil
	case "left", "h":
		if e.dueDatePickerSelected == 0 {
			e.moveCursor(-1)
		} else if e.dueDatePickerSelected < 4 {
			moveTextCursor(&e.cursor, e.dueDatePickerInput(), -1)
		}
		return m, nil
	case "right", "l":
		if e.dueDatePickerSelected == 0 {
			e.moveCursor(1)
		} else if e.dueDatePickerSelected < 4 {
			moveTextCursor(&e.cursor, e.dueDatePickerInput(), 1)
		}
		return m, nil
	}
	if text := k.Key().Text; text != "" {
		if e.dueDatePickerSelected == 0 {
			m.insertEditorText(text)
		} else if e.dueDatePickerSelected < 4 {
			for _, r := range text {
				if r >= '0' && r <= '9' {
					e.setDueDatePickerInput(insertTextAtCursor(e.dueDatePickerInput(), &e.cursor, string(r)))
				}
			}
		}
	}
	return m, nil
}

func (e *taskEditor) dueDatePickerInput() string {
	if e.dueDatePickerSelected < 1 || e.dueDatePickerSelected > 3 {
		return ""
	}
	return e.dueDatePickerInputs[e.dueDatePickerSelected-1]
}

func (e *taskEditor) setDueDatePickerInput(value string) {
	if e.dueDatePickerSelected < 1 || e.dueDatePickerSelected > 3 {
		return
	}
	e.dueDatePickerInputs[e.dueDatePickerSelected-1] = value
}

func (m *model) updatePriorityPicker(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	s := k.String()
	if index, handled := m.updatePickerNavigation(s, e.priorityIndex, len(tuiTheme.priority)); handled {
		e.priorityIndex = index
		return m, nil
	}
	switch s {
	case "esc":
		e.mode = editorNormal
	case "enter":
		e.draft.Priority = int64(e.priorityIndex)
		e.mode = editorNormal
	}
	return m, nil
}

func (m *model) updateWorkflowStatusPicker(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	s := k.String()
	buckets := e.workflowStatus.Buckets
	if index, handled := m.updatePickerNavigation(s, e.workflowStatusIndex, len(buckets)); handled {
		e.workflowStatusIndex = index
		return m, nil
	}
	switch s {
	case "esc":
		e.mode = editorNormal
	case "enter":
		if len(buckets) > 0 {
			e.draft.WorkflowBucketID = buckets[clamp(e.workflowStatusIndex, 0, len(buckets)-1)].ID
		}
		e.mode = editorNormal
	}
	return m, nil
}

// updatePickerNavigation keeps every Detail picker on the same motion
// contract. The page size is the rendered choice viewport, not the outer Pane
// height, so a short picker predictably clamps at its boundary.
func (m *model) updatePickerNavigation(key string, selected, choices int) (int, bool) {
	e := m.editor
	last := max(0, choices-1)
	selected = clamp(selected, 0, last)
	if e.gArmed {
		e.gArmed = false
		if key == "g" {
			return 0, true
		}
	}
	switch key {
	case "g":
		e.gArmed = true
		return selected, true
	case "G", "end":
		return last, true
	case "home":
		return 0, true
	case "j", "down":
		return clamp(selected+1, 0, last), true
	case "k", "up":
		return clamp(selected-1, 0, last), true
	case "ctrl+d", "pgdown", "pagedown":
		return clamp(selected+m.detailPickerViewportRows(), 0, last), true
	case "ctrl+u", "pgup", "pageup":
		return clamp(selected-m.detailPickerViewportRows(), 0, last), true
	}
	return selected, false
}

func (m *model) detailPickerViewportRows() int {
	// Two rows belong to the picker frame; its content has no heading rows.
	return max(1, m.detailTextHeight()-2)
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
		var undo *undoEntry
		var err error
		if draft.ID == 0 {
			task, err = m.store.CreateTask(ctx, draft)
		} else {
			before, snapshotErr := m.store.TaskUndoState(ctx, draft.ID)
			if snapshotErr != nil {
				return taskSavedMsg{closeAfter: closeAfter, err: snapshotErr}
			}
			task, err = m.store.UpdateTask(ctx, draft)
			if err == nil {
				var after vimkunja.TaskUndoState
				after, err = m.store.TaskUndoState(ctx, draft.ID)
				if err == nil {
					undo = newUndoEntry(before, after)
				}
			}
		}
		return taskSavedMsg{task: task, closeAfter: closeAfter, undo: undo, err: err}
	}
}

func (m *model) moveField(delta int) {
	e := m.editor
	fields := m.detailFields()
	current := 0
	for i, field := range fields {
		if field == e.field {
			current = i
			break
		}
	}
	e.field = fields[clamp(current+delta, 0, len(fields)-1)]
	e.descriptionActive = false
	e.descriptionNavigated = e.field == fieldDescription
	m.ensureDetailFieldVisible()
}

func (m *model) detailFields() []editorField {
	return m.detailFieldsFor(m.compactDetail())
}

func (m *model) detailFieldsFor(compact bool) []editorField {
	if m.editor == nil {
		return nil
	}
	e := m.editor
	if compact {
		fields := []editorField{fieldTitle, fieldMoreFields}
		if e.moreFieldsExpanded {
			fields = append(fields, fieldProject, fieldPriority, fieldDueDate, fieldLabels, fieldWorkflowStatus)
			if e.id != 0 {
				fields = append(fields, fieldDone)
			}
		}
		fields = append(fields, fieldDescription)
		if e.id != 0 {
			fields = append(fields, fieldComments)
		}
		return fields
	}
	fields := []editorField{fieldProject, fieldTitle, fieldPriority, fieldDueDate, fieldLabels, fieldWorkflowStatus}
	if e.id != 0 {
		fields = append(fields, fieldDone)
	}
	fields = append(fields, fieldDescription)
	if e.id != 0 {
		fields = append(fields, fieldComments)
	}
	return fields
}

func (m *model) compactDetail() bool {
	return m.height >= 10 && m.detailTextHeight() < 14
}

func (m *model) detailTextHeight() int {
	_, h := m.viewportSize()
	contentH := max(4, h-3)
	topH := m.detailPaneTop()
	detailH := max(4, contentH-topH)
	return detailH - 2
}

func (m *model) pageDetail(direction int) {
	if m.editor == nil {
		return
	}
	limit := m.detailScrollLimit()
	m.editor.detailScroll = clamp(m.editor.detailScroll+direction*m.pageStep(), 0, limit)
}

func (m *model) normalizeDetailSelection() {
	if m.editor == nil {
		return
	}
	for _, field := range m.detailFields() {
		if field == m.editor.field {
			m.ensureDetailFieldVisible()
			return
		}
	}
	m.editor.field = fieldMoreFields
	m.editor.descriptionActive = false
	m.editor.descriptionNavigated = false
	m.ensureDetailFieldVisible()
}

func (m *model) ensureDetailFieldVisible() {
	if m.editor == nil || !m.compactDetail() {
		return
	}
	row, ok := m.compactDetailFieldRow(m.editor.field)
	if !ok {
		return
	}
	bodyRows := max(1, m.detailTextHeight()-2-compactPinnedCommentRows(m.detailTextHeight(), m.editor.id != 0))
	if row < m.editor.detailScroll {
		m.editor.detailScroll = row
	}
	if row >= m.editor.detailScroll+bodyRows {
		m.editor.detailScroll = row - bodyRows + 1
	}
	m.editor.detailScroll = clamp(m.editor.detailScroll, 0, m.detailScrollLimit())
}

func (m *model) compactDetailFieldRow(field editorField) (int, bool) {
	if !m.compactDetail() || m.editor == nil {
		return 0, false
	}
	if field == fieldComments {
		return 0, false // Comments is pinned outside the scrollable Detail document.
	}
	for row, candidate := range m.detailFields() {
		if candidate == field {
			return row, true
		}
	}
	return 0, false
}

func (m *model) detailScrollLimit() int {
	if !m.compactDetail() || m.editor == nil {
		return 0
	}
	bodyRows := max(1, m.detailTextHeight()-2-compactPinnedCommentRows(m.detailTextHeight(), m.editor.id != 0))
	rows := len(m.detailFields()) + m.compactDescriptionRows()
	if m.editor.id != 0 {
		rows-- // Comments is pinned outside the scrollable Detail document.
	}
	return max(0, rows-bodyRows)
}

func (m *model) compactDescriptionRows() int {
	textHeight := m.detailTextHeight()
	staticRows := len(m.detailFields())
	if m.editor != nil && m.editor.id != 0 {
		textHeight -= compactPinnedCommentRows(textHeight, true)
		staticRows-- // Comments is pinned outside the scrollable Detail document.
	}
	return compactDescriptionRows(textHeight, staticRows)
}

func compactPinnedCommentRows(textH int, hasComments bool) int {
	if !hasComments {
		return 0
	}
	if textH <= 8 {
		return 1
	}
	return 2
}

func compactDescriptionRows(textH, staticRows int) int {
	bodyRows := max(1, textH-2)
	minimum := 2
	if textH >= 8 {
		minimum = 3
	}
	return max(minimum, bodyRows-staticRows)
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
func (e *taskEditor) moveCursor(delta int) { moveTextCursor(&e.cursor, e.fieldValue(), delta) }
func (e *taskEditor) insertText(text string) {
	e.setFieldValue(insertTextAtCursor(e.fieldValue(), &e.cursor, text))
}
func (e *taskEditor) deleteBeforeCursor() {
	e.setFieldValue(deleteTextBeforeCursor(e.fieldValue(), &e.cursor))
}
func (e *taskEditor) deleteAtCursor() {
	e.setFieldValue(deleteTextAtCursor(e.fieldValue(), &e.cursor))
}
func (e *taskEditor) deleteToLineEnd() {
	r := []rune(e.draft.Description)
	cursor := clamp(e.cursor, 0, len(r))
	end := e.lineEnd()
	e.draft.Description = string(append(r[:cursor:cursor], r[end:]...))
	e.cursor = cursor
}

func (e *taskEditor) deleteDescriptionLine() {
	r := []rune(e.draft.Description)
	start, end := e.lineStart(), e.lineEnd()
	if end < len(r) { // Remove this line and its trailing newline.
		e.draft.Description = string(append(r[:start:start], r[end+1:]...))
		e.cursor = start
		return
	}
	if start > 0 { // The final line consumes its preceding newline instead.
		start--
	}
	e.draft.Description = string(append(r[:start:start], r[end:]...))
	e.cursor = start
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

// dueDateOverwriteCursor mirrors the date mask's next writable position.
func (e *taskEditor) dueDateOverwriteCursor() int {
	cursor := clamp(e.cursor, 0, len("YYYY-MM-DD"))
	if cursor == 4 || cursor == 7 {
		cursor++
	}
	return cursor
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
