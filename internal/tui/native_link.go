package tui

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/atotto/clipboard"
)

type nativeLinkPrompt struct {
	input  string
	cursor int
}
type linkCopiedMsg struct{ err error }

// DescriptionLinkPromptOpen reports whether URL editing still needs confirmation.
// The guided tutorial uses this to wait for successful application, including retries.
func (m *model) DescriptionLinkPromptOpen() bool {
	return m.editor != nil && m.editor.nativeLink != nil
}

func (m *model) openNativeLink() {
	e := m.editor
	info, ok := e.nativeDraft.LinkAtCursor()
	if e.nativeDraft.Selection() == nil && !ok {
		m.recordError(fmt.Errorf("Select text or place the cursor in a Link"))
		return
	}
	e.nativeLink = &nativeLinkPrompt{input: info.URL, cursor: len([]rune(info.URL))}
	e.nativePrefix = ""
	e.nativeCount = 0
}
func (m *model) updateNativeLink(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	p := e.nativeLink
	switch key.String() {
	case "esc":
		e.nativeLink = nil
	case "enter":
		if err := e.nativeDraft.Apply(description.Action{Kind: description.EditLink, Text: p.input}, false); err != nil {
			if errors.Is(err, description.ErrConfirmation) {
				e.nativeLink = nil
				m.applyNative(description.Action{Kind: description.EditLink, Text: p.input}, false, false)
			} else {
				m.recordError(err)
			}
			return m, nil
		}
		e.nativeLink = nil
		m.syncNativeDraft()
	case "ctrl+v":
		return m, m.readClipboard(func(msg *clipboardPastedMsg) { msg.editor = e; msg.link = p })
	default:
		updateSingleLineTextInput(&p.input, &p.cursor, key)
	}
	return m, nil
}
func (m *model) copyNativeLink() tea.Cmd {
	info, ok := m.editor.nativeDraft.LinkAtCursor()
	if !ok {
		m.recordError(fmt.Errorf("Place the cursor in a Link to copy its destination"))
		return nil
	}
	write := m.clipboardWrite
	if write == nil {
		write = clipboard.WriteAll
	}
	return func() tea.Msg { return linkCopiedMsg{err: write(info.URL)} }
}
func (m *model) nativeLinkHint() string {
	e := m.editor
	if p := e.nativeLink; p != nil {
		return "LINK URL  " + insertCursorText(sanitizeTerminalText(p.input), p.cursor) + " · Enter apply · empty unlink · Esc cancel"
	}
	info, ok := e.nativeDraft.LinkAtCursor()
	if !ok {
		return ""
	}
	hint := "Link → " + sanitizeTerminalText(info.URL) + " · typing extends link · Space+l edit · Space+y copy"
	if info.Edge {
		hint = "Link edge → " + sanitizeTerminalText(info.URL) + " · typing is outside link · Space+l edit"
	}
	if info.Unsupported {
		hint += " · Unsupported link scheme; preserved"
	}
	return strings.TrimSpace(hint)
}

func sanitizeTerminalText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
