package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

type clipboardPastedMsg struct {
	editor   *taskEditor
	composer *commentComposer
	text     string
	err      error
}

func normalizePastedText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

func systemClipboardText() (string, error) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("read system clipboard: %w", err)
	}
	return text, nil
}

func (m *model) pasteClipboard(e *taskEditor) tea.Cmd {
	return m.readClipboard(func(message *clipboardPastedMsg) { message.editor = e })
}

func (m *model) pasteCommentClipboard(composer *commentComposer) tea.Cmd {
	return m.readClipboard(func(message *clipboardPastedMsg) { message.composer = composer })
}

func (m *model) readClipboard(setTarget func(*clipboardPastedMsg)) tea.Cmd {
	read := m.clipboardRead
	if read == nil {
		read = systemClipboardText
	}
	return func() tea.Msg {
		text, err := read()
		message := clipboardPastedMsg{text: text, err: err}
		setTarget(&message)
		return message
	}
}
