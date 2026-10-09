package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type commentComposer struct {
	text   string
	cursor int
	mode   editorMode
	normal textNormalState
}

type commentsUI struct {
	items         []vimkunja.TaskComment
	selected      int
	viewport      int
	loading       bool
	saving        bool
	err           error
	gArmed        bool
	composer      *commentComposer
	commandMode   bool
	commandInput  string
	commandCursor int
}

type taskCommentsLoadedMsg struct {
	taskID   int64
	request  uint64
	comments []vimkunja.TaskComment
	err      error
}

type taskCommentSavedMsg struct {
	editor      *taskEditor
	draft       vimkunja.TaskDraft
	task        vimkunja.Task
	comment     vimkunja.TaskComment
	closeAfter  bool
	taskChanged bool
	err         error
}

type taskCommentDeletedMsg struct {
	taskID    int64
	commentID int64
	err       error
}

func (m *model) loadTaskComments(taskID int64) tea.Cmd {
	m.commentRequest++
	request := m.commentRequest
	if m.editor != nil && m.editor.id == taskID {
		m.editor.commentsRequest = request
	}
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		comments, err := m.store.ListTaskComments(ctx, taskID)
		return taskCommentsLoadedMsg{taskID: taskID, request: request, comments: comments, err: err}
	}
}

func (m *model) openComments() {
	if m.editor == nil || m.editor.id == 0 {
		m.recordError(fmt.Errorf("Save Task before adding Comments."))
		return
	}
	m.comments = &commentsUI{
		items:    append([]vimkunja.TaskComment(nil), m.editor.comments...),
		selected: max(0, len(m.editor.comments)-1),
		loading:  m.editor.commentsLoading,
		err:      m.editor.commentsErr,
	}
}

func (m *model) closeComments() {
	m.comments = nil
	m.cursorVisible = true
}

func (m *model) startCommentComposer() {
	if m.comments == nil || m.comments.composer != nil {
		return
	}
	m.comments.composer = &commentComposer{mode: editorInsert}
	m.cursorVisible = true
}

func (m *model) updateComments(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	ui := m.comments
	if ui == nil {
		return m, nil
	}
	if ui.saving {
		return m, nil
	}
	if ui.commandMode {
		return m.updateCommentCommand(key)
	}
	if ui.composer != nil {
		if ui.composer.mode == editorInsert {
			return m.updateCommentInsert(key)
		}
		return m.updateCommentNormal(key)
	}
	s := normalizeArrow(key.String())
	if ui.gArmed {
		ui.gArmed = false
		if s == "g" {
			ui.selected = 0
			return m, nil
		}
		if s == "d" {
			m.beginCommentDeletion()
			return m, nil
		}
		return m, nil
	}
	switch s {
	case "esc":
		m.closeComments()
	case ":":
		ui.commandMode, ui.commandInput, ui.commandCursor = true, "", 0
	case "a", "i", "o":
		m.startCommentComposer()
		return m, cursorBlinkCmd()
	case "d":
		ui.gArmed = true
		return m, nil
	case "g":
		ui.gArmed = true
	case "G":
		ui.selected = max(0, len(ui.items)-1)
	case "j", "down":
		ui.selected = clamp(ui.selected+1, 0, max(0, len(ui.items)-1))
	case "k", "up":
		ui.selected = clamp(ui.selected-1, 0, max(0, len(ui.items)-1))
	case "ctrl+d", "pgdown", "pagedown":
		ui.selected = clamp(ui.selected+m.pageStep(), 0, max(0, len(ui.items)-1))
	case "ctrl+u", "pgup", "pageup":
		ui.selected = clamp(ui.selected-m.pageStep(), 0, max(0, len(ui.items)-1))
	}
	return m, nil
}

func (m *model) updateCommentInsert(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	composer := m.comments.composer
	s := normalizeArrow(key.String())
	switch s {
	case "ctrl+v":
		return m, m.pasteCommentClipboard(composer)
	case "esc":
		composer.mode = editorNormal
		m.cursorVisible = true
	case "enter":
		composer.text = insertTextAtCursor(composer.text, &composer.cursor, "\n")
	case "backspace":
		composer.text = deleteTextBeforeCursor(composer.text, &composer.cursor)
	case "delete":
		composer.text = deleteTextAtCursor(composer.text, &composer.cursor)
	case "left":
		moveTextCursor(&composer.cursor, composer.text, -1)
	case "right":
		moveTextCursor(&composer.cursor, composer.text, 1)
	case "home":
		composer.cursor = textLineStart(composer.text, composer.cursor)
	case "end":
		composer.cursor = textLineEnd(composer.text, composer.cursor)
	case "up":
		moveCommentCursorLine(composer, -1)
	case "down":
		moveCommentCursorLine(composer, 1)
	default:
		if text := key.Key().Text; text != "" {
			composer.text = insertTextAtCursor(composer.text, &composer.cursor, text)
		}
	}
	return m, nil
}

func (m *model) updateCommentNormal(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	composer := m.comments.composer
	s := normalizeArrow(key.String())
	if result := applyTextNormalKey(&composer.normal, composer.text, composer.cursor, s, true); result.handled {
		composer.text, composer.cursor = result.text, result.cursor
		if result.enterInsert {
			composer.mode = editorInsert
			return m, cursorBlinkCmd()
		}
		return m, nil
	}
	switch s {
	case ":":
		m.comments.commandMode, m.comments.commandInput, m.comments.commandCursor = true, "", 0
	case "i":
		composer.mode = editorInsert
		return m, cursorBlinkCmd()
	case "a":
		moveTextCursor(&composer.cursor, composer.text, 1)
		composer.mode = editorInsert
		return m, cursorBlinkCmd()
	case "I":
		composer.cursor = textLineStart(composer.text, composer.cursor)
		composer.mode = editorInsert
		return m, cursorBlinkCmd()
	case "A":
		composer.cursor = textLineEnd(composer.text, composer.cursor)
		composer.mode = editorInsert
		return m, cursorBlinkCmd()
	case "o":
		composer.cursor = textLineEnd(composer.text, composer.cursor)
		composer.text = insertTextAtCursor(composer.text, &composer.cursor, "\n")
		composer.mode = editorInsert
		return m, cursorBlinkCmd()
	case "O":
		composer.cursor = textLineStart(composer.text, composer.cursor)
		composer.text = insertTextAtCursor(composer.text, &composer.cursor, "\n")
		composer.cursor--
		composer.mode = editorInsert
		return m, cursorBlinkCmd()
	case "h", "left":
		moveTextCursor(&composer.cursor, composer.text, -1)
	case "l", "right":
		moveTextCursor(&composer.cursor, composer.text, 1)
	case "j", "down":
		moveCommentCursorLine(composer, 1)
	case "k", "up":
		moveCommentCursorLine(composer, -1)
	case "0", "home":
		composer.cursor = textLineStart(composer.text, composer.cursor)
	case "$", "end":
		composer.cursor = textLineEnd(composer.text, composer.cursor)
	case "x":
		composer.text = deleteTextAtCursor(composer.text, &composer.cursor)
	case "esc":
		if strings.TrimSpace(composer.text) == "" {
			m.comments.composer = nil
			m.cursorVisible = true
			return m, nil
		}
		m.recordError(fmt.Errorf("unsaved Comment; use :w or :q!"))
	}
	return m, nil
}

func moveCommentCursorLine(composer *commentComposer, delta int) {
	editor := taskEditor{draft: vimkunja.TaskDraft{Description: composer.text}, field: fieldDescription, cursor: composer.cursor}
	editor.moveCursorLine(delta)
	composer.cursor = editor.cursor
}

func textLineStart(text string, cursor int) int {
	runes := []rune(text)
	cursor = clamp(cursor, 0, len(runes))
	for cursor > 0 && runes[cursor-1] != '\n' {
		cursor--
	}
	return cursor
}

func textLineEnd(text string, cursor int) int {
	runes := []rune(text)
	cursor = clamp(cursor, 0, len(runes))
	for cursor < len(runes) && runes[cursor] != '\n' {
		cursor++
	}
	return cursor
}

func (m *model) updateCommentCommand(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	ui := m.comments
	s := key.String()
	switch s {
	case "esc":
		ui.commandMode, ui.commandInput = false, ""
	case "backspace", "delete", "left", "right", "home", "end":
		updateSingleLineTextInput(&ui.commandInput, &ui.commandCursor, key)
		if ui.commandInput == "" && key.String() == "backspace" {
			ui.commandMode = false
		}
	case "enter":
		command := strings.TrimSpace(ui.commandInput)
		ui.commandMode, ui.commandInput = false, ""
		switch command {
		case "w", "write":
			return m, m.saveTaskWithComment(false)
		case "wq", "writequit":

			return m, m.saveTaskWithComment(true)
		case "q", "quit":
			if ui.composer != nil || (m.editor != nil && m.editor.dirty()) {
				m.recordError(fmt.Errorf("unsaved changes; use :w or :q!"))
			} else {
				m.closeComments()
			}
		case "q!", "quit!":
			if m.editor != nil {
				m.editor.draft = cloneTaskDraft(m.editor.original)
				m.editor.labelsLoading = true
				taskID := m.editor.id
				m.closeComments()
				return m, m.loadEditorLabels(taskID)
			}
			m.closeComments()
		default:
			m.recordError(fmt.Errorf("unknown Comments command: %s", command))
		}
	default:
		updateSingleLineTextInput(&ui.commandInput, &ui.commandCursor, key)
	}
	return m, nil
}

func (m *model) saveTaskWithComment(closeAfter bool) tea.Cmd {
	if m.editor == nil || m.comments == nil || m.comments.composer == nil {
		return nil
	}
	text := m.comments.composer.text
	if strings.TrimSpace(text) == "" {
		m.recordError(fmt.Errorf("Comment text is required"))
		return nil
	}
	editor := m.editor
	if !m.prepareNativeSave() {
		return nil
	}
	draft := editor.saveDraft()
	taskChanged := m.editor.dirty()
	m.comments.saving = true
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		task, comment, err := m.store.SaveTaskWithComment(ctx, draft, vimkunja.TaskCommentDraft{Text: text})
		return taskCommentSavedMsg{editor: editor, draft: draft, task: task, comment: comment, closeAfter: closeAfter, taskChanged: taskChanged, err: err}
	}
}

func (m *model) renderCommentsOverlay(background string, width int) string {
	ui := m.comments
	if ui == nil {
		return background
	}
	viewportHeight := len(strings.Split(background, "\n"))
	layout := modalBounds(width, viewportHeight, max(1, width-4), viewportHeight)
	rows := []string{renderHeading("TASK COMMENTS"), ""}
	available := max(1, layout.contentHeight-len(rows))
	composerRows := []string(nil)
	if ui.composer != nil {
		composerHeight := min(max(4, len(strings.Split(ui.composer.text, "\n"))+2), max(4, available/2))
		composerRows = m.renderCommentComposer(layout.contentWidth, composerHeight)
		available = max(1, available-len(composerRows)-1)
	}
	threadRows, selectedStart, selectedEnd := renderCommentThread(ui, layout.contentWidth)
	if ui.loading && len(ui.items) == 0 {
		threadRows = []string{"Loading…"}
	} else if ui.err != nil && len(ui.items) == 0 {
		threadRows = []string{"Unable to load Comments: " + ui.err.Error()}
	} else if len(threadRows) == 0 {
		threadRows = []string{"No comments yet. Press a, i, or o to append one."}
	}
	ui.viewport = clamp(ui.viewport, 0, max(0, len(threadRows)-available))
	if selectedStart < ui.viewport {
		ui.viewport = selectedStart
	} else if selectedEnd > ui.viewport+available {
		if selectedEnd-selectedStart > available {
			ui.viewport = selectedStart
		} else {
			ui.viewport = selectedEnd - available
		}
	}
	end := min(len(threadRows), ui.viewport+available)
	rows = append(rows, threadRows[ui.viewport:end]...)
	for len(rows) < layout.contentHeight-len(composerRows) {
		rows = append(rows, "")
	}
	if len(composerRows) > 0 {
		rows = append(rows, composerRows...)
	}
	modal := renderModalFrame(rows, layout)
	return overlayModalLayout(background, modal, width, layout.fallback)
}

func renderCommentThread(ui *commentsUI, width int) ([]string, int, int) {
	rows := make([]string, 0)
	selectedStart, selectedEnd := 0, 0
	for index, comment := range ui.items {
		if index == ui.selected {
			selectedStart = len(rows)
		}
		marker := "  "
		if index == ui.selected {
			marker = "> "
			if ui.composer == nil && !ui.commandMode && !ui.saving {
				marker = normalCursorText(">", 0) + " "
			}
		}
		when := comment.Created.In(time.Local).Format("2006-01-02 15:04")
		header := comment.Author + " · " + when
		if comment.Edited() {
			header += " · edited"
		}
		if index == ui.selected {
			header = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render(header)
		}
		rows = append(rows, marker+header)
		for _, line := range wrapDescription(descriptionTerminalText(comment.HTML), max(1, width-2)) {
			rows = append(rows, "  "+line.text)
		}
		if index == ui.selected {
			selectedEnd = len(rows)
		}
		rows = append(rows, "")
	}
	return rows, selectedStart, selectedEnd
}

func (m *model) beginCommentDeletion() {
	if m.comments == nil || m.editor == nil || m.comments.selected < 0 || m.comments.selected >= len(m.comments.items) {
		return
	}
	comment := m.comments.items[m.comments.selected]
	if !comment.CanDelete {
		m.recordError(fmt.Errorf("only the Comment author may delete it"))
		return
	}
	m.commentDeleteConfirm = true
	m.commentDeleteTaskID, m.commentDeleteID = comment.TaskID, comment.ID
	m.confirmationScroll, m.confirmationGArmed = 0, false
}

func (m *model) updateCommentDeleteConfirmation(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		if m.commentDeleting {
			return m, nil
		}
		m.commentDeleting = true
		taskID, commentID := m.commentDeleteTaskID, m.commentDeleteID
		return m, func() tea.Msg {
			ctx, cancel := contextWithTimeout()
			defer cancel()
			return taskCommentDeletedMsg{taskID: taskID, commentID: commentID, err: m.store.DeleteTaskComment(ctx, taskID, commentID)}
		}
	case "n", "esc":
		m.commentDeleteConfirm = false
		m.commentDeleteTaskID, m.commentDeleteID = 0, 0
	}
	return m, nil
}

func (m *model) renderCommentDeleteConfirmationModal(background string, width int) string {
	viewportHeight := len(strings.Split(background, "\n"))
	layout := modalBounds(width, viewportHeight, min(50, width), 10)
	body, _ := confirmationBody(
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.danger)).Render("DELETE COMMENT"),
		"Delete comment?",
		"y delete  •  n / Esc cancel",
		layout,
		0,
	)
	return overlayModalLayout(background, renderModalFrame(body, layout), width, layout.fallback)
}

func (m *model) renderCommentComposer(width, height int) []string {
	composer := m.comments.composer
	label := "NEW COMMENT"
	if composer.mode == editorInsert {
		label += " · INSERT"
	} else {
		label += " · NORMAL"
	}
	divider := strings.Repeat("─", max(1, width))
	text := composer.text
	if composer.mode == editorInsert {
		cursor := composer.cursor
		text = insertTextAtCursor(text, &cursor, "_")
	}
	lines := wrapDescription(text, max(1, width))
	if composer.mode == editorNormal {
		for index, line := range lines {
			if composer.cursor >= line.start && composer.cursor <= line.end {
				lines[index].text = normalCursorText(line.text, composer.cursor-line.start)
				break
			}
		}
	}
	rows := []string{divider, label}
	if len(lines) == 0 {
		rows = append(rows, "_")
	} else {
		for _, line := range lines {
			rows = append(rows, line.text)
		}
	}
	if len(rows) > height {
		contentRows := max(1, height-2)
		rows = append(rows[:2], rows[len(rows)-contentRows:]...)
	}
	return rows
}
