package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestFocusedDescriptionDoesNotEscapeSentencePeriod(t *testing.T) {
	const text = "Move through this description."
	for _, source := range []string{text, "<p>" + text + "</p>"} {
		m := nativeEditingModel(source)
		nativeKeys(m, "g", "g")
		view := ansi.Strip(m.View().Content)
		if !strings.Contains(view, text) || strings.Contains(view, `description\.`) {
			t.Fatalf("focused sentence shows an artificial period escape:\n%s", view)
		}
		if m.editor.dirty() || m.editor.draft.Description != source {
			t.Fatal("opening changed source")
		}
	}
}
