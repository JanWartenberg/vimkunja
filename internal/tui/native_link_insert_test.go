package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"strings"
	"testing"
)

func TestNativeLinkInsertWithoutSelection(t *testing.T) {
	for _, source := range []string{"", `<p></p>`, `<p>After</p>`, `<p><strong>After</strong></p>`} {
		for _, label := range []string{"https://example.test/x", "Grüße & <lesen>"} {
			t.Run(source+label, func(t *testing.T) {
				m := nativeEditingModel(source)
				m = nativeKeys(m, "space", "l")
				if m.editor.nativeLink == nil {
					t.Fatal("missing URL prompt")
				}
				m.Update(tea.PasteMsg{Content: "https://example.test/x"})
				m = nativeKeys(m, "enter")
				if m.editor.nativeLink == nil || !m.editor.nativeLink.label || m.editor.dirty() {
					t.Fatal("missing text step or premature mutation")
				}
				if m.editor.nativeLink.input != "" || m.editor.nativeLink.cursor != 0 {
					t.Fatal("new link text must start empty")
				}
				if label != "https://example.test/x" {
					m = nativeKeys(m, label)
				}
				m = nativeKeys(m, "enter")
				if m.editor.nativeLink != nil || !m.editor.dirty() {
					t.Fatalf("insert failed: %s", m.lastErr)
				}
				wantLabel := "https://example.test/x"
				if label != wantLabel {
					wantLabel = "Grüße &amp; &lt;lesen&gt;"
				}
				want := `<p><a href="https://example.test/x" target="_blank">` + wantLabel + `</a>`
				if strings.Contains(source, "strong") {
					want = `<p><a href="https://example.test/x" target="_blank"><strong>` + wantLabel + `</strong></a><strong>After</strong></p>`
				} else {
					if strings.Contains(source, "After") {
						want += "After"
					}
					want += "</p>"
				}
				if got := m.editor.draft.Description; got != want {
					t.Fatalf("got %s want %s", got, want)
				}
				m = nativeKeys(m, "u")
				if m.editor.draft.Description != source || m.editor.dirty() || m.editor.nativeDraft.Selection() != nil {
					t.Fatalf("undo: %q", m.editor.draft.Description)
				}
				m = nativeKeys(m, "ctrl+r")
				if m.editor.draft.Description != want {
					t.Fatalf("redo: %q", m.editor.draft.Description)
				}
			})
		}
	}
}

func TestNativeLinkInsertCancelAndValidation(t *testing.T) {
	for _, textStep := range []bool{false, true} {
		m := nativeEditingModel(`<p>Keep</p>`)
		m = nativeKeys(m, "space", "l")
		m.Update(tea.PasteMsg{Content: "not a url"})
		m = nativeKeys(m, "enter")
		if m.editor.nativeLink == nil || m.editor.nativeLink.label || m.editor.dirty() {
			t.Fatal("invalid URL advanced or mutated")
		}
		if textStep {
			m = nativeKeys(m, "ctrl+u")
			m.Update(tea.PasteMsg{Content: "https://example.test/x"})
			m = nativeKeys(m, "enter")
			m.Update(tea.PasteMsg{Content: "line one\nline two"})
			m = nativeKeys(m, "enter")
			if m.editor.nativeLink == nil || !m.editor.nativeLink.label || m.editor.dirty() {
				t.Fatal("multiline label mutated")
			}
		}
		m = nativeKeys(m, "esc")
		if m.editor.nativeLink != nil || m.editor.dirty() || m.editor.draft.Description != `<p>Keep</p>` {
			t.Fatal("cancel mutated draft")
		}
	}
	m := nativeEditingModel(`<pre><code>original()</code></pre>`)
	m = nativeKeys(m, "space", "l")
	if m.editor.nativeLink != nil || m.editor.dirty() {
		t.Fatal("link prompt opened in code")
	}
}

func TestNativeLinkInsertPasteAndSave(t *testing.T) {
	m, db := auditEditorModel(t)
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task','',0,0,1)`); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.focus = paneDetail
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	m = nativeKeys(m, "space", "l")
	oldPrompt := m.editor.nativeLink
	m.Update(tea.PasteMsg{Content: "https://example.test/x"})
	m = nativeKeys(m, "enter")
	m.Update(clipboardPastedMsg{editor: m.editor, link: oldPrompt, text: "late"})
	if m.editor.nativeLink.input != "" {
		t.Fatal("stale clipboard overwrote label")
	}
	m.Update(tea.PasteMsg{Content: "Read more"})
	m = nativeKeys(m, "enter")
	want := `<p><a href="https://example.test/x" target="_blank">Read more</a></p>`
	save := m.saveEditor(false)
	if save == nil {
		t.Fatal(m.lastErr)
	}
	m.Update(save())
	var got string
	if err := db.QueryRow(`SELECT description FROM tasks WHERE id=1`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want || m.editor.dirty() {
		t.Fatalf("save=%q err=%s", got, m.lastErr)
	}
}

func TestNativeLinkInsertInsideParagraph(t *testing.T) {
	m := nativeEditingModel(`<p>Before After</p><p><mention-user data-id="jan" data-label="Jan"></mention-user></p>`)
	m.editor.descriptionView()
	nativeContentCursor(t, m, "Before After", len("Before "))
	m = nativeKeys(m, "space", "l")
	m.Update(tea.PasteMsg{Content: "https://example.test/x"})
	m = nativeKeys(m, "enter", "Read", "enter")
	const want = `<p>Before <a href="https://example.test/x" target="_blank">Read</a>After</p><p><mention-user data-id="jan" data-label="Jan"></mention-user></p>`
	if got := m.editor.draft.Description; got != want {
		t.Fatalf("%s; err=%s", got, m.lastErr)
	}
	if m.editor.nativeDraft.Selection() != nil {
		t.Fatal("internal selection leaked")
	}
}

func TestNativeLinkEmptyLabelUsesURLAfterClearing(t *testing.T) {
	m := nativeEditingModel(`<p>After</p>`)
	nativeKeys(m, "space", "l")
	m.Update(tea.PasteMsg{Content: "https://example.test/x"})
	nativeKeys(m, "enter", "Temporary")
	m = pressKey(m, tea.Key{Code: 'u', Mod: tea.ModCtrl})
	if m.editor.nativeLink.input != "" {
		t.Fatal("Ctrl+U did not clear label")
	}
	nativeKeys(m, "enter")
	if got := m.editor.draft.Description; got != `<p><a href="https://example.test/x" target="_blank">https://example.test/x</a>After</p>` {
		t.Fatal(got, m.lastErr)
	}
}

func TestFormatWithoutSelectionExplainsWhatToSelect(t *testing.T) {
	for _, tc := range []struct{ key, phrase string }{{"b", "bold"}, {"i", "italic"}, {"s", "strike"}} {
		m := nativeEditingModel(`<p>Alpha beta</p>`)
		nativeKeys(m, "space", tc.key)
		if !strings.Contains(m.lastErr, "Select text") || !strings.Contains(m.lastErr, tc.phrase) {
			t.Fatalf("%s: %s", tc.key, m.lastErr)
		}
		if m.editor.dirty() {
			t.Fatal("hint mutated source")
		}
	}
}
