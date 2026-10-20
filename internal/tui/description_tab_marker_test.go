package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestDescriptionTabMarkersSurviveResizeOutsideCode(t *testing.T) {
	const source = "x\tcc\n\tcc"
	m := nativeEditingModel(source)
	for _, mode := range []editorMode{editorNormal, editorInsert} {
		m.editor.mode = mode
		for _, size := range []struct{ w, h int }{{120, 45}, {55, 24}, {80, 30}, {120, 45}} {
			m.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
			out := m.Render()
			if !strings.Contains(ansi.Strip(out), "x>~~cc") || !strings.Contains(out, "38;5;245") {
				t.Fatalf("missing gray tab at %dx%d mode=%v: %q", size.w, size.h, mode, out)
			}
			if m.editor.dirty() || m.editor.draft.Description != source {
				t.Fatal("render or resize changed source")
			}
			if got := m.editor.nativeDraft.CopyLine(); got != "x\tcc" {
				t.Fatalf("copied display marker: %q", got)
			}
		}
	}
}
