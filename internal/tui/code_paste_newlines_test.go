package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestCodePasteRendersTransportNewlines(t *testing.T) {
	for _, tc := range []struct{ name, newline string }{{"LF", "\n"}, {"CRLF", "\r\n"}, {"CR", "\r"}} {
		for _, route := range []string{"terminal", "system clipboard"} {
			t.Run(tc.name+"/"+route, func(t *testing.T) {
				m := nativeEditingModel("<pre><code>original()</code></pre>")
				text := strings.Join([]string{"\ttab_start äöü ß €", "    vier_spaces **literal**", "ENDE"}, tc.newline)
				if route == "terminal" {
					nativeKeys(m, "i")
					m.Update(tea.PasteMsg{Content: text})
				} else {
					m.clipboardRead = func() (string, error) { return text, nil }
					m.Update(clipboardAction(t, m, "p")())
				}
				want := "<pre><code>" + text + "original()</code></pre>"
				if m.editor.draft.Description != want {
					t.Fatalf("Paste changed source bytes: %q", m.editor.draft.Description)
				}
				screen := ansi.Strip(m.View().Content)
				if strings.Contains(screen, `\r`) {
					t.Fatalf("Code Paste shows carriage-return escapes instead of lines:\n%s", screen)
				}
				lines := strings.Split(screen, "\n")
				previous := -1
				for _, expected := range []string{"tab_start äöü ß €", "vier_spaces **literal**", "ENDEoriginal()"} {
					found := -1
					for i, line := range lines {
						if strings.Contains(line, expected) {
							found = i
							break
						}
					}
					if found <= previous {
						t.Fatalf("expected separate source line %q:\n%s", expected, screen)
					}
					previous = found
				}
				if route == "terminal" {
					nativeKeys(m, "esc")
				}
				nativeKeys(m, "g", "g", "j")
				if got := m.editor.nativeDraft.CopyLine(); got != "    vier_spaces **literal**" {
					t.Fatalf("second-line copy: %q", got)
				}
				if !m.prepareNativeSave() {
					t.Fatal(m.lastErr)
				}
				nativeKeys(m, "u")
				if m.editor.draft.Description != "<pre><code>original()</code></pre>" {
					t.Fatal("Paste Undo lost original source")
				}
				nativeKeys(m, "ctrl+r")
				if m.editor.draft.Description != want {
					t.Fatal("Paste Redo changed source bytes")
				}
			})
		}
	}
}
