package tui

import (
	"fmt"
	"html"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestFocusedDescriptionKeepsLiteralPunctuationAcrossModes(t *testing.T) {
	for _, text := range []string{"TUI-Rendering", "single_under_score", "'", `"`, "`", "*", "~", "#", "+", "[", "]", `C:\temp`} {
		for _, kind := range []string{"plain", "html", "paste"} {
			for _, height := range []int{24, 40} {
				t.Run(fmt.Sprintf("%s/%d/%s", kind, height, text), func(t *testing.T) {
					source := text
					if kind == "html" {
						source = "<p>" + html.EscapeString(text) + "</p>"
					} else if kind == "paste" {
						source = ""
					}
					m := nativeEditingModel(source)
					m.Update(tea.WindowSizeMsg{Width: 100, Height: height})
					if kind == "paste" {
						nativeKeys(m, "i")
						m.Update(tea.PasteMsg{Content: text})
						nativeKeys(m, "esc")
					}
					before := m.editor.draft.Description
					dirty := m.editor.dirty()
					for _, mode := range []string{"normal", "insert", "visual"} {
						nativeKeys(m, "g", "g")
						if mode == "insert" {
							nativeKeys(m, "i")
						} else if mode == "visual" {
							nativeKeys(m, "v")
						}
						view := m.View()
						screen := ansi.Strip(view.Content)
						if !strings.Contains(screen, text) || strings.Count(screen, "\\") != strings.Count(text, "\\") {
							t.Fatalf("%s changed literal punctuation:\n%s", mode, ansi.Strip(view.Content))
						}
						if mode == "insert" && (view.Cursor == nil || view.Cursor.Shape != tea.CursorBar) {
							t.Fatal("literal punctuation lost the insertion cursor")
						}
						nativeKeys(m, "esc")
					}
					if m.editor.draft.Description != before || m.editor.dirty() != dirty {
						t.Fatal("mode switches changed the draft")
					}
				})
			}
		}
	}
}
