package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestTextInputsShareBlinkingTerminalBar(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*model)
	}{
		{"Title", func(m *model) { m.editor.draft.Title = "abc"; m.editor.cursor = 1; m.editor.mode = editorInsert }},
		{"plain Description", func(m *model) {
			m.editor.field = fieldDescription
			m.editor.draft.Description = "abc"
			m.editor.mode = editorInsert
			positionDescriptionCursor(t, m, 1)
		}},
		{"native Description", func(m *model) {
			m.editor.field = fieldDescription
			m.editor.draft.Description = "<p>abc</p>"
			m.editor.descriptionActive = true
			m.editor.descriptionView()
			m.editor.mode = editorInsert
		}},
		{"Code", func(m *model) {
			m.editor.field = fieldDescription
			m.editor.draft.Description = "<pre><code>abc</code></pre>"
			m.editor.descriptionActive = true
			m.editor.descriptionView()
			m.editor.mode = editorInsert
		}},
		{"Comment", func(m *model) {
			m.comments = &commentsUI{composer: &commentComposer{text: "abc", cursor: 1, mode: editorInsert}}
		}},
		{"Ex command", func(m *model) { m.openCommandPrompt(); m.commandInput = "abc"; m.commandCursor = 1 }},
		{"Search", func(m *model) {
			m.searchOpen = true
			m.searchMode = searchInsertMode
			m.searchInput = "abc"
			m.searchInputCursor = 1
		}},
		{"Label", func(m *model) {
			m.labelUI = &labelUI{kind: labelUIGlobalOverview, inputMode: true, inputAction: labelInputCreate, input: "abc", inputCursor: 1}
		}},
		{"Comment command", func(m *model) { m.comments = &commentsUI{commandMode: true, commandInput: "abc", commandCursor: 1} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.width, m.height = 100, 40
			m.currentProjectID = 1
			m.openNewEditor()
			tc.setup(m)
			view := m.View()
			if view.Cursor == nil || view.Cursor.Shape != tea.CursorBar || !view.Cursor.Blink {
				t.Fatalf("%s cursor=%#v, want shared blinking bar", tc.name, view.Cursor)
			}
			lines := strings.Split(ansi.Strip(view.Content), "\n")
			pos := view.Cursor.Position
			if pos.Y < 0 || pos.Y >= len(lines) || pos.X < 0 || pos.X >= ansi.StringWidth(lines[pos.Y]) {
				t.Fatalf("cursor outside rendered input: %+v", pos)
			}
		})
	}
}

func TestTextInputCaretTracksUnicodeAndDoesNotLeakLayoutMarkers(t *testing.T) {
	for _, field := range []string{"Title", "Description", "Native", "Comment", "Command", "Search", "Label"} {
		t.Run(field, func(t *testing.T) {
			m := testModel()
			m.width, m.height = 54, 24
			m.currentProjectID = 1
			m.openNewEditor()
			text := "界éb"
			switch field {
			case "Title":
				m.editor.draft.Title = text
				m.editor.cursor = 3
				m.editor.mode = editorInsert
			case "Description":
				m.editor.field = fieldDescription
				m.editor.draft.Description = text
				m.editor.mode = editorInsert
				positionDescriptionCursor(t, m, 3)
			case "Native":
				m.editor.field = fieldDescription
				m.editor.draft.Description = "<p>" + text + "</p>"
				m.editor.descriptionActive = true
				m.editor.descriptionView()
				nativeContentCursor(t, m, text, len("界é"))
				m.editor.mode = editorInsert
			case "Comment":
				m.comments = &commentsUI{composer: &commentComposer{text: text, cursor: 3, mode: editorInsert}}
			case "Command":
				m.openCommandPrompt()
				m.commandInput = text
				m.commandCursor = 3
			case "Search":
				m.searchOpen = true
				m.searchMode = searchInsertMode
				m.searchInput = text
				m.searchInputCursor = 3
			case "Label":
				m.labelUI = &labelUI{kind: labelUIGlobalOverview, inputMode: true, inputAction: labelInputCreate, input: text, inputCursor: 3}
			}
			view := m.View()
			if strings.Contains(view.Content, insertCursorAnchor) {
				t.Fatal("layout marker leaked to terminal")
			}
			if view.Cursor == nil {
				t.Fatal("input caret missing")
			}
			rows := strings.Split(view.Content, "\n")
			pos := view.Cursor.Position
			if got := ansi.Strip(ansi.Cut(rows[pos.Y], pos.X, pos.X+1)); got != "b" {
				t.Fatalf("caret points at %q, want b: %+v\n%s", got, pos, ansi.Strip(view.Content))
			}
		})
	}
}

func TestCommentCaretRemainsVisibleWhenEditingAboveLastLines(t *testing.T) {
	m := testModel()
	m.width, m.height = 54, 24
	m.currentProjectID = 1
	m.openNewEditor()
	m.comments = &commentsUI{composer: &commentComposer{text: "alpha\n\n\n" + strings.Repeat("tail\n", 20), cursor: 1, mode: editorInsert}}
	view := m.View()
	if view.Cursor == nil {
		t.Fatal("caret scrolled out of Comment composer")
	}
	rows := strings.Split(ansi.Strip(view.Content), "\n")
	if !strings.Contains(rows[view.Cursor.Position.Y], "alpha") {
		t.Fatalf("caret not on edited line: %q", rows[view.Cursor.Position.Y])
	}
}

func TestPromptCaretSurvivesClippingWithTextAfterCursor(t *testing.T) {
	m := testModel()
	m.width, m.height = 40, 18
	m.openCommandPrompt()
	m.commandInput = strings.Repeat("before", 20) + "TARGET" + strings.Repeat("after", 20)
	m.commandCursor = len(strings.Repeat("before", 20))
	view := m.View()
	if view.Cursor == nil {
		t.Fatal("clipped prompt lost caret")
	}
	pos := view.Cursor.Position
	rows := strings.Split(view.Content, "\n")
	if got := ansi.Strip(ansi.Cut(rows[pos.Y], pos.X, pos.X+1)); got != "T" {
		t.Fatalf("clipped caret points at %q", got)
	}
}

func TestURLLabelAndCommandInputsUseTheSameCaretPolicy(t *testing.T) {
	for _, kind := range []string{"URL", "Label command", "Search command", "Long Label"} {
		t.Run(kind, func(t *testing.T) {
			m := nativeEditingModel(`<p><a href="https://example.org/">Link</a></p>`)
			m.width, m.height = 54, 24
			switch kind {
			case "URL":
				m = nativeKeys(m, "space", "l")
				m.editor.nativeLink.input = "https://example.org/" + strings.Repeat("path", 20)
				m.editor.nativeLink.cursor = 22
			case "Label command":
				m.labelUI = &labelUI{commandMode: true, commandInput: "abc", commandCursor: 1}
			case "Search command":
				m.searchOpen = true
				m.searchCommandMode = true
				m.searchCommandInput = "abc"
				m.searchCommandCursor = 1
			case "Long Label":
				m.labelUI = &labelUI{inputMode: true, inputAction: labelInputCreate, input: strings.Repeat("abcd", 30), inputCursor: 80}
			}
			view := m.View()
			if view.Cursor == nil || view.Cursor.Shape != tea.CursorBar || !view.Cursor.Blink {
				t.Fatalf("%s cursor=%#v", kind, view.Cursor)
			}
			if strings.Contains(view.Content, insertCursorAnchor) {
				t.Fatal("internal anchor leaked")
			}
		})
	}
}

func TestInsertCursorFallsBackConsistentlyWhenShapesAreDisabled(t *testing.T) {
	for _, kind := range []string{"Title", "Native", "Comment", "Command"} {
		m := testModel()
		m.disableCursorShapes = true
		m.currentProjectID = 1
		m.openNewEditor()
		switch kind {
		case "Title":
			m.editor.draft.Title = "abc"
			m.editor.cursor = 1
			m.editor.mode = editorInsert
		case "Native":
			m.editor.field = fieldDescription
			m.editor.draft.Description = "<p>abc</p>"
			m.editor.descriptionActive = true
			m.editor.descriptionView()
			nativeContentCursor(t, m, "abc", 1)
			m.editor.mode = editorInsert
		case "Comment":
			m.comments = &commentsUI{composer: &commentComposer{text: "abc", cursor: 1, mode: editorInsert}}
		case "Command":
			m.openCommandPrompt()
			m.commandInput = "abc"
			m.commandCursor = 1
		}
		view := m.View()
		if view.Cursor != nil || !strings.Contains(ansi.Strip(view.Content), "a│bc") {
			t.Fatalf("%s fallback cursor inconsistent: %#v", kind, view.Cursor)
		}
	}
}

func TestCommentCaretFollowsWrappedAndEmptyLines(t *testing.T) {
	m := testModel()
	m.width, m.height = 54, 24
	m.currentProjectID = 1
	m.openNewEditor()
	m.comments = &commentsUI{composer: &commentComposer{text: strings.Repeat("A", 90) + "B", cursor: 90, mode: editorInsert}}
	view := m.View()
	if view.Cursor == nil {
		t.Fatal("wrapped Comment lost caret")
	}
	pos := view.Cursor.Position
	rows := strings.Split(view.Content, "\n")
	if got := ansi.Strip(ansi.Cut(rows[pos.Y], pos.X, pos.X+1)); got != "B" {
		t.Fatalf("wrapped caret points at %q", got)
	}
	m.comments.composer.text = "first\n\nlast"
	m.comments.composer.cursor = 6
	empty := m.View()
	if empty.Cursor == nil {
		t.Fatal("empty Comment line lost caret")
	}
	m.comments.composer.cursor = 7
	next := m.View()
	if next.Cursor == nil || next.Cursor.Position.Y != empty.Cursor.Position.Y+1 {
		t.Fatal("consecutive empty Comment line changes cursor geometry")
	}
}

func TestEscRestoresNormalCursorAndClearsInsertionAnchor(t *testing.T) {
	for _, source := range []string{"<p>abc</p>", "<pre><code>abc</code></pre>"} {
		m := nativeKeys(nativeEditingModel(source), "i")
		if m.View().Cursor == nil {
			t.Fatal("Insert cursor missing")
		}
		m = nativeKeys(m, "esc")
		view := m.View()
		if view.Cursor != nil || !inverseCell.MatchString(view.Content) {
			t.Fatal("Esc did not restore Normal block cursor")
		}
	}
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.comments = &commentsUI{composer: &commentComposer{text: "abc", cursor: 1, mode: editorInsert}}
	if m.View().Cursor == nil {
		t.Fatal("Comment Insert cursor missing")
	}
	m = nativeKeys(m, "esc")
	if view := m.View(); view.Cursor != nil || !strings.Contains(ansi.Strip(view.Content), "NEW COMMENT · NORMAL") {
		t.Fatal("Comment Esc did not restore Normal block")
	}
}

func TestNativeAppendAtLineEndKeepsTerminalCursor(t *testing.T) {
	for _, tc := range []struct {
		name, source, before, want string
		down                       bool
	}{
		{"Code before newline", "<pre><code>print(1)\n\t</code></pre>", "print(1)", "<pre><code>print(1)z\n\t</code></pre>", false},
		{"Code final Tab", "<pre><code>print(1)\n\t</code></pre>", ">~~~", "<pre><code>print(1)\n\tz</code></pre>", true},
		{"Code final text", "<pre><code>abc</code></pre>", "abc", "<pre><code>abcz</code></pre>", false},
		{"Code Unicode", "<pre><code>界é\nnext</code></pre>", "界é", "<pre><code>界éz\nnext</code></pre>", false},
		{"Code empty first line", "<pre><code>\nnext</code></pre>", "│ ", "<pre><code>z\nnext</code></pre>", false},
		{"paragraph", "<p>abc</p>", "abc", "<p>abcz</p>", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := nativeEditingModel(tc.source)
			if tc.down {
				m = nativeKeys(m, "j")
			}
			m = nativeKeys(m, "A")
			view := m.View()
			if view.Cursor == nil || view.Cursor.Shape != tea.CursorBar || !view.Cursor.Blink {
				t.Fatalf("Shift+A lost insertion cursor: %#v\n%s", view.Cursor, ansi.Strip(view.Content))
			}
			rows := strings.Split(view.Content, "\n")
			pos := view.Cursor.Position
			before := ansi.Strip(ansi.Cut(rows[pos.Y], 0, pos.X))
			if !strings.HasSuffix(before, tc.before) {
				t.Fatalf("caret after %q, want suffix %q", before, tc.before)
			}
			if m.editor.draft.Description != tc.source {
				t.Fatal("append changed source")
			}
			m = nativeKeys(m, "z")
			if m.editor.draft.Description != tc.want {
				t.Fatalf("typed at wrong boundary: %q (err=%s)", m.editor.draft.Description, m.lastErr)
			}
			if m.View().Cursor == nil {
				t.Fatal("typing lost cursor")
			}
		})
	}
}
