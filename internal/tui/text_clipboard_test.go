package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

func clipboardTextModel(kind, text string) *model {
	m := nativeEditingModel(text)
	switch kind {
	case "Title":
		m.editor.field = fieldTitle
		m.editor.descriptionActive = false
		m.editor.draft.Title = text
	case "Comment":
		m.comments = &commentsUI{composer: &commentComposer{text: text, mode: editorNormal}}
	}
	return m
}
func clipboardTextValue(m *model, kind string) string {
	if kind == "Title" {
		return m.editor.draft.Title
	}
	if kind == "Comment" {
		return m.comments.composer.text
	}
	return m.editor.draft.Description
}
func clipboardAction(t *testing.T, m *model, key string) tea.Cmd {
	t.Helper()
	nativeKeys(m, "space")
	_, cmd := m.Update(tea.KeyPressMsg{Code: []rune(key)[0], Text: key})
	if cmd == nil {
		t.Fatalf("Space+%s returned no clipboard operation", key)
	}
	return cmd
}
func TestTextClipboardVisualCopyAndReplace(t *testing.T) {
	for _, kind := range []string{"Title", "Description", "Comment", "Native"} {
		t.Run(kind, func(t *testing.T) {
			source, want := "abcdef", "X&*def"
			if kind == "Description" {
				want = "<p>X&amp;*def</p>"
			}
			if kind == "Native" {
				source, want = "<p>abcdef</p>", "<p>X&amp;*def</p>"
			}
			m := clipboardTextModel(kind, source)
			copied := ""
			m.clipboardWrite = func(s string) error { copied = s; return nil }
			m.clipboardRead = func() (string, error) { return "X&*", nil }
			nativeKeys(m, "v", "l", "l")
			if !strings.Contains(ansi.Strip(m.Render()), "VISUAL") {
				t.Fatalf("selection mode missing (plain=%+v)\n%s", m.editor.normal.selection, ansi.Strip(m.Render()))
			}
			m.Update(clipboardAction(t, m, "y")())
			if copied != "abc" || clipboardTextValue(m, kind) != source {
				t.Fatalf("copy=%q, text=%q", copied, clipboardTextValue(m, kind))
			}
			m.Update(clipboardAction(t, m, "p")())
			if got := clipboardTextValue(m, kind); got != want {
				t.Fatalf("paste=%q, want %q (err=%s)", got, want, m.lastErr)
			}
			if m.editor.mode != editorNormal {
				t.Fatal("paste unexpectedly entered Insert")
			}
		})
	}
}
func TestTextClipboardLinewiseAndNoSelection(t *testing.T) {
	for _, kind := range []string{"Description", "Comment", "Native"} {
		t.Run(kind, func(t *testing.T) {
			source := "one\ntwo\nthree"
			want := "one\nthree"
			if kind == "Description" {
				want = "<p>one</p><p>three</p>"
			}
			if kind == "Native" {
				source = "<pre><code>one\ntwo\nthree</code></pre>"
				want = "<pre><code>one\nthree</code></pre>"
			}
			m := clipboardTextModel(kind, source)
			copied := ""
			m.clipboardWrite = func(s string) error { copied = s; return nil }
			nativeKeys(m, "j")
			m.Update(clipboardAction(t, m, "y")())
			if copied != "two" {
				t.Fatalf("current line=%q", copied)
			}
			nativeKeys(m, "V", "d")
			if got := clipboardTextValue(m, kind); got != want {
				t.Fatalf("line deletion=%q, want %q", got, want)
			}
		})
	}
}
func TestTextClipboardErrorsAndStalePaste(t *testing.T) {
	for _, kind := range []string{"Title", "Description", "Comment", "Native"} {
		t.Run(kind, func(t *testing.T) {
			source := "abc"
			if kind == "Native" {
				source = "<p>abc</p>"
			}
			m := clipboardTextModel(kind, source)
			m.clipboardRead = func() (string, error) { return "X", nil }
			cmd := clipboardAction(t, m, "p")
			nativeKeys(m, "l", "h")
			m.Update(cmd())
			if clipboardTextValue(m, kind) != source {
				t.Fatal("stale paste changed text")
			}
			m.clipboardRead = func() (string, error) { return "", fmt.Errorf("clipboard unavailable") }
			m.Update(clipboardAction(t, m, "p")())
			if clipboardTextValue(m, kind) != source || !strings.Contains(m.lastErr, "clipboard unavailable") {
				t.Fatalf("error handling: %s", m.lastErr)
			}
		})
	}
}
func TestPlainVisualSelectionMotionsAndEscape(t *testing.T) {
	for _, kind := range []string{"Title", "Description", "Comment"} {
		t.Run(kind, func(t *testing.T) {
			m := clipboardTextModel(kind, "a界ébc")
			nativeKeys(m, "v", "l", "l")
			if !inverseCell.MatchString(m.Render()) {
				t.Fatal("selection has no highlight")
			}
			copied := ""
			m.clipboardWrite = func(s string) error { copied = s; return nil }
			m.Update(clipboardAction(t, m, "y")())
			if copied != "a界é" {
				t.Fatalf("copy splits grapheme: %q", copied)
			}
			nativeKeys(m, "esc")
			if m.editor == nil || clipboardTextValue(m, kind) != "a界ébc" {
				t.Fatal("Esc closed or changed editor")
			}
			nativeKeys(m, "0", "v", "i", "w", "c", "Z", "esc")
			want := "Z"
			if kind == "Description" {
				want = "<p>Z</p>"
			}
			if got := clipboardTextValue(m, kind); got != want {
				t.Fatalf("visual word replacement=%q", got)
			}
		})
	}
}
func TestTextClipboardPasteWithoutSelectionAndEmptyRead(t *testing.T) {
	for _, kind := range []string{"Title", "Description", "Comment", "Native"} {
		t.Run(kind, func(t *testing.T) {
			source, want := "abc", "aXbc"
			if kind == "Description" {
				want = "<p>aXbc</p>"
			}
			if kind == "Native" {
				source, want = "<p>abc</p>", "<p>aXbc</p>"
			}
			m := clipboardTextModel(kind, source)
			nativeKeys(m, "l")
			m.clipboardRead = func() (string, error) { return "X", nil }
			m.Update(clipboardAction(t, m, "p")())
			if clipboardTextValue(m, kind) != want {
				t.Fatalf("paste=%q", clipboardTextValue(m, kind))
			}
			nativeKeys(m, "v")
			m.clipboardRead = func() (string, error) { return "", nil }
			m.Update(clipboardAction(t, m, "p")())
			if clipboardTextValue(m, kind) != want {
				t.Fatal("empty clipboard deleted selection")
			}
			m.clipboardWrite = func(string) error { return fmt.Errorf("write denied") }
			m.Update(clipboardAction(t, m, "y")())
			if !strings.Contains(m.lastErr, "write denied") || clipboardTextValue(m, kind) != want {
				t.Fatal("failed copy changed draft or hid failure")
			}
		})
	}
}
func TestClipboardNativeCodePasteIsLiteralAndUndoable(t *testing.T) {
	m := nativeEditingModel("<pre><code>abc\nnext</code></pre>")
	m.clipboardRead = func() (string, error) { return "\t**x**\r\n", nil }
	nativeKeys(m, "v", "l")
	m.Update(clipboardAction(t, m, "p")())
	if got := m.editor.draft.Description; got != "<pre><code>\t**x**\r\nc\nnext</code></pre>" {
		t.Fatalf("code paste=%q", got)
	}
	nativeKeys(m, "u")
	if m.editor.draft.Description != "<pre><code>abc\nnext</code></pre>" {
		t.Fatal("paste was not one undo step")
	}
}
func TestDescriptionRichtextConversionSaveAndReopen(t *testing.T) {
	m, db := auditEditorModel(t)
	source := "one & two\n\nlast"
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task',?,0,0,1)`, source); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	m.Render()
	nativeKeys(m, "v", "l", "space", "b", "esc")
	if !m.prepareNativeSave() {
		t.Fatal(m.lastErr)
	}
	result := m.saveEditor(false)().(taskSavedMsg)
	if result.err != nil {
		t.Fatal(result.err)
	}
	m.Update(result)
	var saved string
	var done bool
	if err := db.QueryRow("SELECT description,done FROM tasks WHERE id=1").Scan(&saved, &done); err != nil {
		t.Fatal(err)
	}
	if saved != "<p><strong>on</strong>e &amp; two</p><p></p><p>last</p>" || done {
		t.Fatalf("saved=%q done=%v", saved, done)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: saved})
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	nativeKeys(m, "V", "space", "i")
	if !strings.Contains(m.editor.draft.Description, "<em>") {
		t.Fatal("reopened Richtext not formattable")
	}
}

func TestClipboardPasteDoesNotBypassOpaqueConfirmation(t *testing.T) {
	source := `<p>A<img alt="map" src="x">B</p>`
	m := nativeEditingModel(source)
	nativeKeys(m, "l", "v")
	m.clipboardRead = func() (string, error) { return "X", nil }
	m.Update(clipboardAction(t, m, "p")())
	if m.editor.nativePending == nil || m.editor.draft.Description != source {
		t.Fatalf("Opaque replacement bypassed confirmation: %s", m.lastErr)
	}
	nativeKeys(m, "esc")
	if m.editor.draft.Description != source {
		t.Fatal("cancel changed Opaque")
	}
}
func TestPlainVisualIncompletePrefixesDoNotDelete(t *testing.T) {
	for _, prefix := range []string{"i", "g"} {
		m := clipboardTextModel("Description", "abc")
		nativeKeys(m, "v", prefix, "d")
		if m.editor.draft.Description != "abc" {
			t.Fatalf("v%s d deleted text", prefix)
		}
	}
}
func TestPlainVisualLineChangeKeepsLineSeparators(t *testing.T) {
	for _, down := range []bool{false, true} {
		m := clipboardTextModel("Description", "one\ntwo")
		if down {
			nativeKeys(m, "j")
		}
		nativeKeys(m, "V", "c", "X", "esc")
		want := "<p>X</p><p>two</p>"
		if down {
			want = "<p>one</p><p>X</p>"
		}
		if m.editor.draft.Description != want {
			t.Fatalf("line change=%q", m.editor.draft.Description)
		}
	}
}
