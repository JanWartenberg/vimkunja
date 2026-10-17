package tui

import (
	tea "charm.land/bubbletea/v2"
	"regexp"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// Inverse video must survive the Code/inline formatting on the cursor cell.
var inverseCell = regexp.MustCompile(`\x1b\[(?:[0-9]+;)*7(?:;[0-9]+)*m`)

func TestNativeCodeCursorVisibleOnWhitespaceLines(t *testing.T) {
	for _, whitespace := range []string{"\t", "    ", ""} {
		t.Run(whitespace, func(t *testing.T) {
			source := "<pre><code>print(1)\n" + whitespace + "</code></pre>"
			m := nativeEditingModel(source)
			m.width, m.height = 100, 40
			m = nativeKeys(m, "l", "l", "l", "j")
			lines := strings.Split(m.Render(), "\n")
			found := false
			for i, line := range lines {
				if strings.Contains(ansi.Strip(line), "print(1)") && i+1 < len(lines) {
					found = true
					if !inverseCell.MatchString(lines[i+1]) {
						t.Fatalf("cursor missing on whitespace row: %q", lines[i+1])
					}
				}
			}
			if !found {
				t.Fatal("Code source row missing")
			}
			m = nativeKeys(m, "k")
			if m.editor.draft.Description != source {
				t.Fatal("cursor movement changed source")
			}
		})
	}
}

func TestNativeCodeCursorVisibleAfterHorizontalScroll(t *testing.T) {
	source := "<pre><code>" + strings.Repeat("x", 100) + "END\n\t</code></pre>"
	m := nativeEditingModel(source)
	m.width, m.height = 54, 40
	m = nativeKeys(m, "$")
	m.Render()
	m = nativeKeys(m, "j")
	// Moving to the short source line must also reveal its cursor horizontally.
	frame := m.editor.descriptionView().Viewport(40, 10, true)
	m.Render()
	lines := renderNativeDescriptionFrame(m.editor, frame, 40)
	if frame.CursorY < 0 || !inverseCell.MatchString(lines[frame.CursorY].text) {
		t.Fatalf("horizontal scroll hid whitespace cursor: %+v", frame)
	}
	if m.editor.draft.Description != source {
		t.Fatal("scroll/navigation changed source")
	}
}

func TestNativeCursorVisibleOnDecoratedSpace(t *testing.T) {
	for _, tag := range []string{"u", "s", "code"} {
		source := "<p><" + tag + ">A B</" + tag + "></p>"
		m := nativeKeys(nativeEditingModel(source), "l")
		found := false
		for _, line := range strings.Split(m.Render(), "\n") {
			if strings.Contains(ansi.Strip(line), "A B") {
				found = true
				if !inverseCell.MatchString(line) {
					t.Fatalf("%s hid cursor: %q", tag, line)
				}
			}
		}
		if !found {
			t.Fatalf("%s text missing", tag)
		}
		if m.editor.draft.Description != source {
			t.Fatal("render changed source")
		}
	}
}

func TestNativeBacktickCreationShowsEnterConfirmation(t *testing.T) {
	for _, chunks := range [][]string{{"`", "`", "`"}, {"``", "``"}, {"```"}} {
		m := nativeKeys(nativeEditingModel("<p></p>"), "i")
		for _, text := range chunks {
			code := rune('`')
			if len(text) > 1 {
				code = tea.KeyExtended
			}
			m = pressKey(m, tea.Key{Code: code, Text: text})
		}
		if strings.Contains(m.editor.draft.Description, "<pre>") {
			t.Fatal("Code created before Enter")
		}
		for _, width := range []int{54, 100} {
			footer := ansi.Strip(m.renderStatusBar(width))
			if !strings.Contains(footer, "Enter create Code") {
				t.Fatalf("chunks=%q width=%d missing confirmation cue: %s", chunks, width, footer)
			}
		}
		markerSource := m.editor.draft.Description
		m = nativeKeys(m, "enter")
		if m.editor.draft.Description != "<pre><code></code></pre>" || m.editor.mode != editorInsert {
			t.Fatalf("Code creation failed: %q (%s)", m.editor.draft.Description, m.lastErr)
		}
		m = nativeKeys(m, "esc", "u")
		if m.editor.draft.Description != markerSource {
			t.Fatal("Undo did not restore literal markers")
		}
	}
}

func TestNativeBacktickCreationFromTerminalSequences(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"legacy", "i```\r"},
		{"paired fences", "i``````\x1b[D\x1b[D\x1b[D\r"},
		{"grouped paired fences", "i\x1b[96;1;96:96:96u\x1b[96;1;96:96:96u\x1b[D\x1b[D\x1b[D\r"},
		{"grouped text", "i\x1b[96;1;96:96u\x1b[96;1;96:96u\r"},
		{"shifted text", "i" + strings.Repeat("\x1b[96;2;96u", 3) + "\r"},
		{"press and release", "i" + strings.Repeat("\x1b[96;1:1u\x1b[96;1:3u", 3) + "\r"},
		{"win32 shifted", "i" + strings.Repeat("\x1b[192;41;96;1;16;1_", 3) + "\x1b[13;28;13;1;0;1_"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := nativeEditingModel("<p></p>")
			var decoder uv.EventDecoder
			input := []byte(tc.input)
			for len(input) > 0 {
				n, event := decoder.Decode(input)
				if n <= 0 {
					t.Fatalf("undecodable input: %q", input)
				}
				switch event := event.(type) {
				case uv.KeyPressEvent:
					m.Update(tea.KeyPressMsg(event))
				case uv.KeyReleaseEvent:
					m.Update(tea.KeyReleaseMsg(event))
				default:
					t.Fatalf("unexpected terminal event: %T (%+v)", event, event)
				}
				input = input[n:]
			}
			if m.editor.draft.Description != "<pre><code></code></pre>" || m.editor.mode != editorInsert {
				t.Fatalf("terminal Code creation: %q (%s)", m.editor.draft.Description, m.lastErr)
			}
		})
	}
}

func TestNativeBacktickConfirmationExcludesLiteralMarkers(t *testing.T) {
	for _, kind := range []string{"paste", "import", "escaped", "five", "suffix"} {
		m := nativeKeys(nativeEditingModel("<p></p>"), "i")
		switch kind {
		case "paste":
			m.Update(tea.PasteMsg{Content: "```"})
		case "import":
			m = nativeKeys(nativeEditingModel("<p>```</p>"), "A")
		case "escaped":
			m = nativeKeys(m, "\\", "````")
		case "five":
			m = nativeKeys(m, "`````")
		case "suffix":
			m = nativeKeys(m, "```x")
		}
		if strings.Contains(ansi.Strip(m.renderStatusBar(100)), "Enter create Code") {
			t.Fatalf("%s falsely offered Code creation", kind)
		}
		m = nativeKeys(m, "enter")
		if strings.Contains(m.editor.draft.Description, "<pre>") {
			t.Fatalf("%s markers unexpectedly became Code", kind)
		}
	}
}

func TestNativeCodeBlockHasBackgroundWithoutInlineUnderline(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		underlined   bool
	}{
		{"Code block", "<pre><code>print(1)\n\t</code></pre>", false},
		{"Inline Code", "<p><code>print(1)</code></p>", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := nativeEditingModel(tc.source)
			underline := regexp.MustCompile(`\x1b\[(?:[0-9]+;)*4(?:;[0-9]+)*m`)
			found := false
			for _, line := range strings.Split(m.Render(), "\n") {
				if strings.Contains(ansi.Strip(line), "print(1)") {
					found = true
					if underline.MatchString(line) != tc.underlined {
						t.Fatalf("unexpected underline: %q", line)
					}
					if !tc.underlined && !strings.Contains(line, "48;5;236") {
						t.Fatalf("Code background missing: %q", line)
					}
				}
			}
			if !found {
				t.Fatal("source text missing")
			}
			if m.editor.draft.Description != tc.source {
				t.Fatal("render changed source")
			}
		})
	}
}

func TestNativeEnterBetweenTypedBacktickFencesCreatesCode(t *testing.T) {
	m := nativeKeys(nativeEditingModel("<p></p>"), "i", "``````")
	if m.editor.draft.Description != "<p>``````</p>" {
		t.Fatal("six markers must remain text before Enter")
	}
	m = nativeKeys(m, "left", "left", "left")
	if !strings.Contains(ansi.Strip(m.renderStatusBar(100)), "Enter create Code") {
		t.Fatal("missing Code creation cue at the fence boundary")
	}
	m = nativeKeys(m, "enter")
	if m.editor.draft.Description != "<pre><code></code></pre>" || m.editor.mode != editorInsert {
		t.Fatalf("did not create empty Code: %s (%s)", m.editor.draft.Description, m.lastErr)
	}
	m = nativeKeys(m, "esc", "u")
	if m.editor.draft.Description != "<p>``````</p>" {
		t.Fatal("one Undo must restore both marker groups")
	}
	m = nativeKeys(m, "ctrl+r", "i", "x")
	if m.editor.draft.Description != "<pre><code>x</code></pre>" {
		t.Fatalf("restored Code cursor is not editable: %s", m.editor.draft.Description)
	}
}

func TestNativePairedBacktickTriggerRejectsLiteralAndWrongBoundaries(t *testing.T) {
	for _, kind := range []string{"paste", "import", "escaped", "mixed", "before", "after", "end", "eight"} {
		t.Run(kind, func(t *testing.T) {
			m := nativeKeys(nativeEditingModel("<p></p>"), "i")
			moves := 3
			switch kind {
			case "paste":
				m.Update(tea.PasteMsg{Content: "``````"})
			case "import":
				m = nativeKeys(nativeEditingModel("<p>``````</p>"), "A")
			case "escaped":
				m = nativeKeys(m, "\\", "``````")
			case "mixed":
				m = nativeKeys(m, "```")
				m.Update(tea.PasteMsg{Content: "```"})
			case "before":
				m = nativeKeys(m, "``````")
				moves = 4
			case "after":
				m = nativeKeys(m, "``````")
				moves = 2
			case "end":
				m = nativeKeys(m, "``````")
				moves = 0
			case "eight":
				m = nativeKeys(m, "````````")
				moves = 4
			}
			for range moves {
				m = nativeKeys(m, "left")
			}
			if strings.Contains(ansi.Strip(m.renderStatusBar(100)), "Enter create Code") {
				t.Fatal("literal markers offered Code conversion")
			}
			m = nativeKeys(m, "enter")
			if m.lastErr != "" || strings.Contains(m.editor.draft.Description, "<pre>") {
				t.Fatalf("literal split failed: %s (%s)", m.editor.draft.Description, m.lastErr)
			}
		})
	}
}

func TestNativeCodeTabMarkerIsMutedWithVisibleCursor(t *testing.T) {
	source := "<pre><code>print(1)\n\t</code></pre>"
	m := nativeKeys(nativeEditingModel(source), "j")
	found := false
	for _, line := range strings.Split(m.Render(), "\n") {
		if strings.Contains(ansi.Strip(line), ">~~~") {
			found = true
			if !strings.Contains(line, "38;5;245") {
				t.Fatalf("Tab marker is not muted: %q", line)
			}
			if !inverseCell.MatchString(line) {
				t.Fatal("Tab marker hid cursor")
			}
		}
	}
	if !found {
		t.Fatal("Tab marker missing")
	}
	if m.editor.draft.Description != source {
		t.Fatal("Tab marker changed source")
	}
	m = nativeKeys(m, ":", "r", "e", "t", "a", "b", "enter")
	if m.lastErr != "" || m.editor.draft.Description != "<pre><code>print(1)\n    </code></pre>" {
		t.Fatalf("retab failed: %q (%s)", m.editor.draft.Description, m.lastErr)
	}
	if strings.Contains(ansi.Strip(m.Render()), ">~~~") {
		t.Fatal("Tab marker stayed after retab")
	}
}
