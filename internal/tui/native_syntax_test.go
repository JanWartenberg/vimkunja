package tui

import (
	tea "charm.land/bubbletea/v2"
	"strings"
	"testing"
)

func TestNativeTypedSyntaxAndPasteEvents(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"# Shopping", "<h1>Shopping</h1>"}, {"## Shopping", "<h2>Shopping</h2>"}, {"### Shopping", "<h3>Shopping</h3>"},
		{"- Milk", "<ul><li><p>Milk</p></li></ul>"}, {"1. Milk", "<ol><li><p>Milk</p></li></ol>"},
		{"`npm test`", "<p><code>npm test</code></p>"}, {"***text***", "<p><strong><em>text</em></strong></p>"},
	} {
		m := nativeEditingModel(`<p></p>`)
		m = nativeKeys(m, "i")
		for _, r := range tc.input {
			m = nativeKeys(m, string(r))
		}
		if got := m.editor.draft.Description; got != tc.want {
			t.Errorf("type %q: %q error=%s", tc.input, got, m.lastErr)
		}
		m = nativeEditingModel(`<p></p>`)
		m = nativeKeys(m, "i")
		m.Update(tea.PasteMsg{Content: tc.input})
		if got := m.editor.draft.Description; got != `<p>`+tc.input+`</p>` {
			t.Errorf("paste %q: %q", tc.input, got)
		}
	}
}

func TestNativeTypedRuleIncrementalHistory(t *testing.T) {
	m := nativeEditingModel(`<p></p>`)
	m = nativeKeys(m, "i", "-", "-", "-", "-", "esc")
	if got := m.editor.draft.Description; got != "<hr>" {
		t.Fatal(got)
	}
	m = nativeKeys(m, "x")
	if got := m.editor.draft.Description; got != "<hr>" {
		t.Fatal(got)
	}
	m = nativeKeys(m, "x")
	if got := m.editor.draft.Description; got != "<p>--</p>" {
		t.Fatal(got)
	}
	m = nativeKeys(m, "u")
	if got := m.editor.draft.Description; got != "<hr>" {
		t.Fatal(got)
	}
	m = nativeKeys(m, "u")
	var text strings.Builder
	for _, row := range m.editor.nativeDescription.Layout(80, true).Rows {
		for _, span := range row.Spans {
			text.WriteString(span.Text)
		}
	}
	if text.String() != "----" {
		t.Fatal(text.String())
	}
}

func TestNativeAdjacentSyntaxDeletionDoesNotJoinParagraphs(t *testing.T) {
	for _, tc := range []struct {
		html, text string
		offset     int
		keys       []string
		want       string
	}{
		{`<p>A*B*</p>`, "A*B*", 1, []string{"i", "backspace"}, `<p>A*B*</p>`},
		{`<p>A*B</p>`, "A*B", 1, []string{"i", "backspace"}, `<p>*B</p>`},
		{`<p>before</p><p><strong>bold</strong></p>`, "bold", 0, []string{"i", "backspace"}, `<p>before</p><p>*bold**</p>`},
		{`<p><strong>bold</strong></p><p>after</p>`, "bold", 3, []string{"a", "delete"}, `<p>**bold*</p><p>after</p>`},
	} {
		m := nativeEditingModel(tc.html)
		m.editor.descriptionView()
		nativeContentCursor(t, m, tc.text, tc.offset)
		m = nativeKeys(m, tc.keys...)
		if got := m.editor.draft.Description; got != tc.want {
			t.Fatalf("%q error=%s", got, m.lastErr)
		}
	}
}

func TestNativeStructureAfterTypedDelimiterClearsSyntaxCursor(t *testing.T) {
	for _, tc := range []struct {
		keys []string
		want string
	}{
		{[]string{"enter", "X"}, `<p><strong>b</strong></p><p>X</p>`},
		{[]string{"shift+enter", "X"}, `<p><strong>b</strong><br>X</p>`},
		{[]string{"esc", "o", "X"}, `<p><strong>b</strong></p><p>X</p>`},
	} {
		m := nativeEditingModel(`<p></p>`)
		m = nativeKeys(m, "i", "*", "*", "b", "*", "*")
		m = nativeKeys(m, tc.keys...)
		if got := m.editor.draft.Description; got != tc.want {
			t.Fatalf("%q error=%s", got, m.lastErr)
		}
	}
}
func TestNativeFormatEmptyLinkFallback(t *testing.T) {
	for _, tc := range []struct{ key, want string }{
		{"0", `<p>https://example.test/x</p>`},
		{"b", `<p><a href="https://example.test/x" target="_blank"><strong>https://example.test/x</strong></a></p>`},
	} {
		m := nativeEditingModel(`<p><a href="https://example.test/x"></a></p>`)
		m = nativeKeys(m, "v", "$", "space", tc.key)
		if got := m.editor.draft.Description; got != tc.want {
			t.Fatalf("%q error=%s", got, m.lastErr)
		}
	}
}
