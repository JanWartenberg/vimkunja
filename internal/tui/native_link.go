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
	create bool
	url    string
	label  bool
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
	create := e.nativeDraft.Selection() == nil && !ok
	if create && e.nativeDraft.InCode() {
		m.recordError(fmt.Errorf("Links cannot be inserted in Code blocks"))
		return
	}
	e.nativeLink = &nativeLinkPrompt{input: info.URL, cursor: len([]rune(info.URL)), create: create}
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
		if p.create && !p.label {
			if err := description.ValidateLinkURL(p.input); err != nil {
				m.recordError(err)
				return m, nil
			}
			// A new prompt also rejects late clipboard reads from the URL step.
			e.nativeLink = &nativeLinkPrompt{create: true, url: p.input, label: true}
			return m, nil
		}
		action := description.Action{Kind: description.EditLink, Text: p.input}
		if p.create {
			label := p.input
			if label == "" {
				label = p.url
			}
			action = description.Action{Kind: description.InsertLink, Text: p.url, Label: label}
		}
		if err := e.nativeDraft.Apply(action, false); err != nil {
			if errors.Is(err, description.ErrConfirmation) {
				e.nativeLink = nil
				m.applyNative(action, false, false)
			} else {
				m.recordError(err)
			}
			return m, nil
		}
		e.nativeLink = nil
		m.syncNativeDraft()
	case "ctrl+v":
		return m, m.readClipboard(func(msg *clipboardPastedMsg) { msg.editor = e; msg.link = p })
	case "ctrl+u":
		p.input, p.cursor = "", 0
	default:
		updateSingleLineTextInput(&p.input, &p.cursor, key)
	}
	return m, nil
}
func (p *nativeLinkPrompt) insertText(text string) {
	p.input = insertTextAtCursor(p.input, &p.cursor, text)
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
		field, hint := "LINK URL", " · Enter apply · empty unlink · Esc cancel"
		if p.create {
			hint = " · Enter next: link text · Esc cancel"
		}
		if p.label {
			field, hint = "LINK TEXT", " · Enter insert · empty uses URL · Esc cancel"
		}
		return field + "  " + insertCursorText(sanitizeTerminalText(p.input), p.cursor) + hint
	}
	info, ok := e.nativeDraft.LinkAtCursor()
	if !ok {
		return ""
	}
	hint := "Link → " + sanitizeTerminalText(info.URL) + " · typing extends link · Space+l edit · Space+Y copy URL"
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
