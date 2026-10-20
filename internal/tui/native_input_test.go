package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNativeInvalidUTF8PasteKeepsDraftSavable(t *testing.T) {
	for _, clipboard := range []bool{false, true} {
		m := nativeEditingModel(`<p>Keep</p>`)
		m = nativeKeys(m, "i")
		if clipboard {
			m.Update(clipboardPastedMsg{editor: m.editor, text: "prefix\xffsuffix"})
		} else {
			m.Update(tea.PasteMsg{Content: "prefix\xffsuffix"})
		}
		if m.editor.draft.Description != `<p>Keep</p>` || m.editor.dirty() {
			t.Fatalf("clipboard=%v: rejected paste changed draft: %q", clipboard, m.editor.draft.Description)
		}
		if !strings.Contains(m.lastErr, "UTF-8") {
			t.Fatalf("clipboard=%v: missing input error: %q", clipboard, m.lastErr)
		}
		if _, err := m.editor.nativeDraft.PrepareSave(); err != nil {
			t.Fatalf("clipboard=%v: rejected paste left unsavable draft: %v", clipboard, err)
		}
	}
}

func TestNativeTypedDelimiterUnicodeAttachment(t *testing.T) {
	m := nativeEditingModel(`<p>é👩‍💻x</p>`)
	m = nativeKeys(m, "i", "**")
	m = pressKey(m, tea.Key{Code: tea.KeyLeft})
	m = nativeKeys(m, "**x**", "Y")
	if got := m.editor.draft.Description; got != `<p><strong><em>x</em></strong>Yé👩‍💻x</p>` {
		t.Fatalf("typed delimiters misplaced following input: %q; error=%s", got, m.lastErr)
	}
}
