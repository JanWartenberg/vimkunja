package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

type clipboardPastedMsg struct {
	editor *taskEditor
	text   string
	err    error
}

func systemClipboardText() (string, error) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("read system clipboard: %w", err)
	}
	return text, nil
}

func (m *model) pasteClipboard(e *taskEditor) tea.Cmd {
	read := m.clipboardRead
	if read == nil {
		read = systemClipboardText
	}
	return func() tea.Msg {
		text, err := read()
		return clipboardPastedMsg{editor: e, text: text, err: err}
	}
}
