package tui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

func TestNativeLinkPromptRejectsAndReplacesURL(t *testing.T) {
	m := nativeEditingModel(`<p><a href="https://example.test/old">Read</a></p>`)
	m = nativeKeys(m, "l", "space", "l")
	if m.editor.nativeLink == nil {
		t.Fatal("URL prompt did not open")
	}
	m.editor.nativeLink.input = "custom://resource"
	m = nativeKeys(m, "enter")
	if m.editor.nativeLink == nil || !strings.Contains(m.lastErr, "Invalid Link URL") || m.editor.dirty() {
		t.Fatalf("invalid URL mutated draft: %s", m.lastErr)
	}
	m.editor.nativeLink.input = "https://example.test/new"
	m = nativeKeys(m, "enter")
	if m.editor.nativeLink != nil || m.editor.draft.Description != `<p><a href="https://example.test/new" target="_blank">Read</a></p>` {
		t.Fatal(m.editor.draft.Description)
	}
}

func TestNativeLinkClipboardReportsActualResult(t *testing.T) {
	for _, key := range []string{"e", "Y"} {
		for _, mode := range []string{"normal", "v", "V"} {
			for _, fail := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/fail=%v", key, mode, fail), func(t *testing.T) {
					m := nativeEditingModel(`<p><a href="https://example.test/x">Read</a></p>`)
					nativeKeys(m, "l")
					if mode != "normal" {
						nativeKeys(m, mode)
					}
					before := m.editor.draft.Description
					cursor := m.editor.nativeDescription.Cursor()
					selection := m.editor.nativeDraft.Selection()
					if selection != nil {
						copied := *selection
						selection = &copied
					}
					called := ""
					m.clipboardWrite = func(value string) error {
						called = value
						if fail {
							return fmt.Errorf("clipboard unavailable")
						}
						return nil
					}
					m.Update(clipboardAction(t, m, key)())
					if called != "https://example.test/x" || m.editor.draft.Description != before || m.editor.dirty() {
						t.Fatal("copy returned wrong URL or changed draft")
					}
					if m.editor.nativeDescription.Cursor() != cursor || !reflect.DeepEqual(m.editor.nativeDraft.Selection(), selection) {
						t.Fatal("copy changed cursor or selection")
					}
					if fail && !strings.Contains(m.lastErr, "clipboard unavailable") {
						t.Fatal(m.lastErr)
					}
					if !fail && m.toast != "Link destination copied" {
						t.Fatal(m.toast)
					}
				})
			}
		}
	}
}

func TestNativeLinkCopyWithoutLinkReportsUnavailable(t *testing.T) {
	for _, key := range []string{"e", "Y"} {
		t.Run(key, func(t *testing.T) {
			m := nativeEditingModel("Plain text")
			m.clipboardWrite = func(string) error { t.Fatal("unexpected clipboard write"); return nil }
			nativeKeys(m, "space", key)
			if !strings.Contains(m.lastErr, "Place the cursor in a Link") || m.editor.dirty() {
				t.Fatalf("unavailable copy changed draft or lost feedback: %s", m.lastErr)
			}
		})
	}
}

func TestNativeLinkCopyHintsShowLowercaseShortcut(t *testing.T) {
	m := nativeEditingModel(`<p><a href="https://example.test/x">Read</a></p>`)
	m.width = 240
	nativeKeys(m, "l")
	hint := ansi.Strip(m.Render())
	if !strings.Contains(hint, "Space+e copy URL") || strings.Contains(hint, "Space+Y") {
		t.Fatalf("Link hint: %s", hint)
	}
	nativeKeys(m, "space")
	palette := ansi.Strip(m.Render())
	if !strings.Contains(palette, "e URL") || strings.Contains(palette, "Y URL") {
		t.Fatalf("palette: %s", palette)
	}
	help := strings.Join(helpLines, "\n")
	if !strings.Contains(help, "Space e") || strings.Contains(help, "Space Y") {
		t.Fatalf("help: %s", help)
	}
}

func TestNativeEmptyLinkFirstAppendMaterializesCurrentFallback(t *testing.T) {
	m := nativeEditingModel(`<p><a href="https://example.test/old"></a></p>`)
	m = nativeKeys(m, "space", "l")
	m.editor.nativeLink.input = "https://example.test/new"
	m = nativeKeys(m, "enter", "A", "!", "esc")
	if got := m.editor.draft.Description; got != `<p><a href="https://example.test/new" target="_blank">https://example.test/new!</a></p>` {
		t.Fatalf("%q error=%s target=%+v", got, m.lastErr, m.editor.nativeDescription.Cursor())
	}
}

func TestNativeLinkReplacementRetainsSpanContext(t *testing.T) {
	for _, keys := range [][]string{{"v", "l", "c", "X"}, {"v", "$", "c", "X"}} {
		m := nativeEditingModel(`<p><a href="https://example.test/x">ABCDE</a></p>`)
		m = nativeKeys(m, keys...)
		want := `<p><a href="https://example.test/x" target="_blank">XCDE</a></p>`
		if len(keys) == 4 && keys[1] == "$" {
			want = `<p><a href="https://example.test/x" target="_blank">X</a></p>`
		}
		if got := m.editor.draft.Description; got != want {
			t.Fatalf("%v: %q", keys, got)
		}
	}
}

func TestNativeLinkContextDoesNotHideConfirmation(t *testing.T) {
	m := nativeEditingModel(`<p><a href="https://example.test/x">Read</a><widget>opaque</widget></p>`)
	m = nativeKeys(m, "d", "d")
	if m.editor.nativePending == nil || !strings.HasPrefix(m.nativeModeHint(), "CONFIRM") {
		t.Fatal(m.nativeModeHint())
	}
}

func TestNativeInlineSyntaxAndLinkSaveAdoption(t *testing.T) {
	m, db := auditEditorModel(t)
	const source = `<p></p>`
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task',?,0,0,1)`, source); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	m.focus = paneDetail
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	m = nativeKeys(m, "i", "***text***", "esc", "0", "v", "$")
	nativeSubmitURL(t, m, "https://example.test/x")
	m = nativeKeys(m, "esc")
	const want = `<p><a href="https://example.test/x" target="_blank"><strong><em>text</em></strong></a></p>`
	save := m.saveEditor(false)
	if save == nil {
		t.Fatal(m.lastErr)
	}
	m.Update(save())
	var saved string
	if err := db.QueryRow(`SELECT description FROM tasks WHERE id=1`).Scan(&saved); err != nil {
		t.Fatal(err)
	}
	if saved != want || m.editor.dirty() || m.editor.original.Description != want {
		t.Fatalf("save=%q baseline=%q err=%s", saved, m.editor.original.Description, m.lastErr)
	}
	m = nativeKeys(m, "u")
	if !m.editor.dirty() || m.editor.draft.Description != `<p><strong><em>text</em></strong></p>` {
		t.Fatal("committed local Undo lost")
	}
}

func TestNativeEmptyLinkUnlinkThenType(t *testing.T) {
	m := nativeEditingModel(`<p><a href="https://example.test/x"></a></p>`)
	m.editor.descriptionView()
	nativeSubmitURL(t, m, "")
	m = nativeKeys(m, "i", "!")
	if got := m.editor.draft.Description; got != `<p>!https://example.test/x</p>` {
		t.Fatalf("%q error=%s", got, m.lastErr)
	}
}
