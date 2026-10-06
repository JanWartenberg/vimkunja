package tui

import (
	"fmt"
	"strings"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

// inlineTitleEdit is deliberately limited to List and Overview task titles.
// Detail remains the editor for every other Task field and for Kanban cards.
type inlineTitleEdit struct {
	task         vimkunja.Task
	title        string
	cursor       int // rune offset
	saving       bool
	pendingFocus *paneID
}

type inlineTitleSavedMsg struct {
	taskID int64
	title  string
	undo   *undoEntry
	err    error
}

func (m *model) openInlineTitleEdit(atStart bool) {
	if m.kanbanActive || m.taskSelected < 0 || m.taskSelected >= len(m.tasks) {
		return
	}
	task := m.tasks[m.taskSelected]
	cursor := len([]rune(task.Title))
	if atStart {
		cursor = 0
	}
	m.inlineTitle = &inlineTitleEdit{task: task, title: task.Title, cursor: cursor}
}

func (m *model) updateInlineTitleEdit(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.inlineTitle
	if e == nil || e.saving {
		return m, nil
	}
	switch k.String() {
	case "esc":
		m.inlineTitle = nil
	case "enter":
		return m, m.saveInlineTitle()
	case "backspace":
		e.title = deleteTextBeforeCursor(e.title, &e.cursor)
	case "delete":
		e.title = deleteTextAtCursor(e.title, &e.cursor)
	case "left":
		moveTextCursor(&e.cursor, e.title, -1)
	case "right":
		moveTextCursor(&e.cursor, e.title, 1)
	case "home":
		e.cursor = 0
	case "end":
		e.cursor = len([]rune(e.title))
	default:
		if text := k.Key().Text; text != "" {
			e.title = insertTextAtCursor(e.title, &e.cursor, text)
		}
	}
	return m, nil
}

// saveInlineTitle changes only the title. Validation and failed writes retain
// the inline draft so the user can correct it or explicitly discard it with Esc.
func (m *model) saveInlineTitle() tea.Cmd {
	e := m.inlineTitle
	if e == nil || e.saving {
		return nil
	}
	if strings.TrimSpace(e.title) == "" {
		m.recordError(fmt.Errorf("project and title are required"))
		return nil
	}
	e.saving = true
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		undo, err := m.trackedTaskMutation(ctx, e.task.ID, func() error {
			return m.store.UpdateTaskTitle(ctx, e.task.ID, e.title)
		})
		return inlineTitleSavedMsg{taskID: e.task.ID, title: e.title, undo: undo, err: err}
	}
}

// saveInlineTitleBeforeFocus starts a write only when Ctrl+W would actually
// change Panes. The focus transition occurs after a successful write, so a
// failed save leaves its editable draft in the Tasks Pane.
func (m *model) saveInlineTitleBeforeFocus(keyLower, key string) tea.Cmd {
	var target paneID
	if keyLower == "ctrl+w" || keyLower == "w" {
		target = m.nextFocus()
	} else {
		switch key {
		case "h":
			target = m.focusNeighborLeftOrStay()
		case "l":
			target = m.focusNeighborRightOrStay()
		case "j":
			target = m.focusNeighborDownOrStay()
		case "k":
			target = m.focusNeighborUpOrStay()
		default:
			return nil
		}
	}
	if target == m.focus {
		return nil
	}
	m.inlineTitle.pendingFocus = nil
	cmd := m.saveInlineTitle()
	if cmd != nil {
		m.inlineTitle.pendingFocus = &target
	}
	return cmd
}

func (m *model) applyInlineTitleSave(msg inlineTitleSavedMsg) {
	e := m.inlineTitle
	if e == nil || e.task.ID != msg.taskID {
		return
	}
	if msg.err != nil {
		e.saving = false
		e.pendingFocus = nil
		m.recordError(msg.err)
		return
	}
	m.rememberUndo(msg.undo)
	pendingFocus := e.pendingFocus
	if m.inOverview() {
		for i := range m.overviewTasks {
			if m.overviewTasks[i].ID == msg.taskID {
				m.overviewTasks[i].Title = msg.title
				break
			}
		}
		m.pendingTaskID = msg.taskID
		m.applyOverview()
	} else {
		for i := range m.tasks {
			if m.tasks[i].ID == msg.taskID {
				m.tasks[i].Title = msg.title
				break
			}
		}
	}
	m.inlineTitle = nil
	if pendingFocus != nil {
		m.focus = *pendingFocus
	}
}

func moveTextCursor(cursor *int, value string, delta int) {
	*cursor = clamp(*cursor+delta, 0, len([]rune(value)))
}

// updateSingleLineTextInput applies cursor-aware edits shared by prompts and
// single-line pickers. It reports whether it consumed the key and changed text.
func updateSingleLineTextInput(value *string, cursor *int, key tea.KeyMsg) (handled, changed bool) {
	*cursor = clamp(*cursor, 0, len([]rune(*value)))
	switch key.String() {
	case "home":
		*cursor = 0
		return true, false
	case "end":
		*cursor = len([]rune(*value))
		return true, false
	case "left":
		moveTextCursor(cursor, *value, -1)
		return true, false
	case "right":
		moveTextCursor(cursor, *value, 1)
		return true, false
	case "backspace":
		updated := deleteTextBeforeCursor(*value, cursor)
		changed := updated != *value
		*value = updated
		return true, changed
	case "delete":
		updated := deleteTextAtCursor(*value, cursor)
		changed := updated != *value
		*value = updated
		return true, changed
	}
	if text := key.Key().Text; text != "" {
		*value = insertTextAtCursor(*value, cursor, text)
		return true, true
	}
	return false, false
}

func insertTextAtCursor(value string, cursor *int, text string) string {
	r := []rune(value)
	add := []rune(text)
	*cursor = clamp(*cursor, 0, len(r))
	r = append(r[:*cursor], append(add, r[*cursor:]...)...)
	*cursor += len(add)
	return string(r)
}

func deleteTextBeforeCursor(value string, cursor *int) string {
	r := []rune(value)
	if *cursor > 0 {
		r = append(r[:*cursor-1], r[*cursor:]...)
		(*cursor)--
	}
	return string(r)
}

func deleteTextAtCursor(value string, cursor *int) string {
	r := []rune(value)
	if *cursor < len(r) {
		r = append(r[:*cursor], r[*cursor+1:]...)
	}
	return string(r)
}
