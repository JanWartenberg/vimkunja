package tui

import (
	"fmt"
	"reflect"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/atotto/clipboard"
)

type textCopiedMsg struct{ err error }

// A read may finish after a cursor move, field switch or draft change. The
// captured target validates the full editing context before applying anything.
type textClipboardTarget struct {
	text  string
	valid func(*model) bool
	paste func(*model, string)
}
type textClipboardReadMsg struct {
	generation uint64
	target     textClipboardTarget
	text       string
	err        error
}

func cloneTextNormal(s textNormalState) textNormalState {
	if s.selection != nil {
		selection := *s.selection
		s.selection = &selection
	}
	return s
}
func (m *model) textClipboardTarget() (textClipboardTarget, bool) {
	if m.comments != nil {
		ui := m.comments
		c := ui.composer
		if ui.saving || ui.commandMode || c == nil || c.mode != editorNormal {
			return textClipboardTarget{}, false
		}
		text, cursor, state := c.text, c.cursor, cloneTextNormal(c.normal)
		return textClipboardTarget{
			text: selectedPlainText(text, cursor, &state),
			valid: func(m *model) bool {
				return m.comments == ui && !ui.saving && !ui.commandMode && ui.composer == c && c.mode == editorNormal && c.text == text && c.cursor == cursor && reflect.DeepEqual(c.normal, state)
			},
			paste: func(m *model, value string) {
				c.text, c.cursor = pastePlainText(c.text, c.cursor, &c.normal, value, true)
			},
		}, true
	}
	e := m.editor
	if !m.editorInputFocused() || e.mode != editorNormal || (e.field != fieldTitle && e.field != fieldDescription) || e.field == fieldDescription && !e.descriptionActive {
		return textClipboardTarget{}, false
	}
	field := e.field
	if field == fieldDescription {
		e.descriptionView()
		d := e.nativeDraft
		if d.Classification() == description.Blocked {
			return textClipboardTarget{}, false
		}
		source, target, selection := e.draft.Description, e.nativeDescription.Cursor(), d.Selection()
		return textClipboardTarget{
			text: d.CopyLine(),
			valid: func(m *model) bool {
				return m.editor == e && m.editorInputFocused() && e.mode == editorNormal && e.field == field && e.descriptionActive && e.nativeDraft == d && e.nativePending == nil && e.draft.Description == source && e.nativeDescription.Cursor() == target && reflect.DeepEqual(d.Selection(), selection)
			},
			paste: func(m *model, value string) {
				if value == "" {
					return
				}
				kind := description.Insert
				if d.Selection() != nil {
					kind = description.ReplaceSelection
				}
				e.nativeTrigger = false
				e.nativeCodeEscape = false
				e.nativeLiteralTab = false
				m.applyNative(description.Action{Kind: kind, Text: m.nativePasteText(value)}, false, false)
			},
		}, true
	}
	text, cursor, state := e.fieldValue(), e.cursor, cloneTextNormal(e.normal)
	return textClipboardTarget{
		text: selectedPlainText(text, cursor, &state),
		valid: func(m *model) bool {
			return m.editor == e && m.editorInputFocused() && e.mode == editorNormal && e.field == field && e.fieldValue() == text && e.cursor == cursor && reflect.DeepEqual(e.normal, state)
		},
		paste: func(m *model, value string) {
			if value == "" {
				return
			}
			text, cursor := pastePlainText(e.fieldValue(), e.cursor, &e.normal, value, false)
			e.setFieldValue(text)
			e.cursor = cursor
		},
	}, true
}
func (m *model) textClipboardKey(key string) (tea.Cmd, bool) {
	if key != "y" && key != "p" && key != "e" && key != "Y" {
		return nil, false
	}
	target, ok := m.textClipboardTarget()
	if !ok {
		return nil, false
	}
	if key == "e" || key == "Y" {
		if m.comments == nil && m.editor.field == fieldDescription && m.editor.nativeDraft != nil {
			return m.copyNativeLink(), true
		}
		return nil, false
	}
	if key == "y" {
		write := m.clipboardWrite
		if write == nil {
			write = clipboard.WriteAll
		}
		return func() tea.Msg { return textCopiedMsg{err: write(target.text)} }, true
	}
	read := m.clipboardRead
	if read == nil {
		read = systemClipboardText
	}
	generation := m.textClipboardGeneration
	return func() tea.Msg {
		text, err := read()
		return textClipboardReadMsg{generation: generation, target: target, text: text, err: err}
	}, true
}
func (m *model) applyTextClipboardRead(msg textClipboardReadMsg) {
	if m.savingEditor != nil || msg.generation != m.textClipboardGeneration || !msg.target.valid(m) {
		return
	}
	if msg.err != nil {
		m.recordError(fmt.Errorf("read system clipboard: %w", msg.err))
		return
	}
	msg.target.paste(m, msg.text)
}
