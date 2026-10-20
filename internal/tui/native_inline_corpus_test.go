package tui

import (
	tea "charm.land/bubbletea/v2"
	"encoding/json"
	"fmt"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"os"
	"testing"
)

func nativeFindNode(t *testing.T, m *model, match func(description.Node) bool) description.Node {
	t.Helper()
	var found description.Node
	var visit func([]description.Node)
	visit = func(ns []description.Node) {
		for _, n := range ns {
			if found.ID == 0 && match(n) {
				found = n
			}
			visit(n.Children)
		}
	}
	visit(m.editor.nativeDraft.Snapshot())
	if found.ID == 0 {
		t.Fatal("content cursor not found")
	}
	return found
}
func nativeContentCursor(t *testing.T, m *model, text string, offset int) {
	t.Helper()
	n := nativeFindNode(t, m, func(n description.Node) bool { return n.Kind == description.Text && n.Text == text })
	if err := m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID, Offset: offset}); err != nil {
		t.Fatal(err)
	}
	m.editor.nativeDraft.UpdateView(m.editor.nativeDescription)
}
func nativeSubmitURL(t *testing.T, m *model, value string) {
	t.Helper()
	nativeKeys(m, "space", "l")
	if m.editor.nativeLink == nil {
		t.Fatalf("no URL prompt: %s", m.lastErr)
	}
	m.editor.nativeLink.input = ""
	m.editor.nativeLink.cursor = 0
	m.Update(tea.PasteMsg{Content: value})
	nativeKeys(m, "enter")
}

func TestNativeInlineAcceptedCorpus(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/description-rendering/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Fixtures []struct {
			ID     string `json:"id"`
			Source string `json:"source_html"`
			Edits  []struct {
				HTML *string `json:"expected_html"`
			} `json:"edits"`
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{"editor-delimiters": true, "typed-versus-pasted-markers": true, "literal-import-edit": true, "syntax-escape-parity": true, "native-inline-code": true, "inline-code-typed-literal": true, "typed-rule-dash-history": true, "links": true, "link-import-edges": true, "empty-link-label-edit": true, "cross-paragraph-links": true}
	for _, f := range corpus.Fixtures {
		if !ids[f.ID] {
			continue
		}
		for i, e := range f.Edits {
			if e.HTML == nil {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", f.ID, i), func(t *testing.T) {
				m := nativeEditingModel(f.Source)
				m.editor.descriptionView()
				typeText := func(s string) {
					nativeKeys(m, "i")
					for _, r := range s {
						nativeKeys(m, string(r))
					}
				}
				switch f.ID {
				case "editor-delimiters":
					typeText([]string{"***text***", "\\*text\\*", `C:\temp`, `\\`, "**text", `text\`}[i])
				case "typed-versus-pasted-markers":
					switch i {
					case 0:
						nativeKeys(m, "i")
						m.Update(tea.PasteMsg{Content: "# Shopping\n- Milk"})
					case 1, 2, 3, 4:
						typeText([]string{"", "# Shopping", "## Shopping", "### Shopping", `\# Shopping`}[i])
					case 5:
						nativeKeys(m, "i")
						m.Update(tea.PasteMsg{Content: "<strong>text</strong>"})
					}
				case "syntax-escape-parity":
					typeText(map[int]string{1: `\\*text*`, 2: `\\\*text\*`, 3: "**bold *italic** tail*"}[i])
				case "inline-code-typed-literal":
					values := []string{"`npm test`", "`npm test`", "`a\\`b`", "`text", "X``"}
					if i == 1 {
						nativeKeys(m, "i")
						m.Update(tea.PasteMsg{Content: values[i]})
					} else {
						typeText(values[i])
					}
				case "literal-import-edit":
					if i == 0 {
						nativeContentCursor(t, m, "**Milch** `test`", 2)
						d := m.editor.nativeDraft
						d.Select(false, false)
						d.SetCursor(description.Anchor{Node: d.Cursor().Node, Offset: 6})
						nativeKeys(m, "c", "Hafermilch")
					} else {
						id := m.editor.nativeDraft.Cursor().Node
						for _, off := range []int{0, 1, 7, 8} {
							m.editor.nativeDraft.SetTarget(description.Target{Anchor: description.Anchor{Node: id, Offset: off}, Syntax: 1})
							m.editor.nativeDraft.UpdateView(m.editor.nativeDescription)
							nativeKeys(m, "x")
						}
					}
				case "native-inline-code":
					switch i {
					case 0:
						nativeContentCursor(t, m, "npm test", 4)
						d := m.editor.nativeDraft
						d.Select(false, false)
						d.SetCursor(description.Anchor{Node: d.Cursor().Node, Offset: 7})
						nativeKeys(m, "c", "run")
					case 1:
						n := nativeFindNode(t, m, func(n description.Node) bool { return n.Tag == "code" })
						m.editor.nativeDraft.SetTarget(description.Target{Anchor: description.Anchor{Node: n.ID}, Syntax: 1})
						m.editor.nativeDraft.UpdateView(m.editor.nativeDescription)
						nativeKeys(m, "x")
						nativeContentCursor(t, m, "npm test`", 8)
						nativeKeys(m, "x")
					case 2:
						nativeContentCursor(t, m, "npm test", 0)
						nativeKeys(m, "i", "**")
					}
				case "typed-rule-dash-history":
					typeText("----")
					nativeKeys(m, "esc", "x")
					if i > 0 {
						nativeKeys(m, "x")
					}
					if i == 2 {
						nativeKeys(m, "u")
					}
				case "links":
					switch i {
					case 0:
						nativeSubmitURL(t, m, "")
					case 1:
						d := m.editor.nativeDraft
						d.Select(false, false)
						n := m.editor.nativeDraft.Snapshot()[0].Children[1]
						d.SetCursor(description.Anchor{Node: n.ID, Offset: len(n.Text) - 1})
						nativeKeys(m, "space", "0")
					case 2:
						nativeContentCursor(t, m, "Read", 2)
						nativeKeys(m, "i", "X")
					case 3:
						nativeContentCursor(t, m, "Read", 0)
						nativeKeys(m, "i", "X")
					case 4:
						nativeContentCursor(t, m, "more", 3)
						nativeKeys(m, "a", "X")
					}
				case "link-import-edges":
					nativeContentCursor(t, m, "Custom", 1)
					nativeSubmitURL(t, m, "https://example.test/new")
				case "empty-link-label-edit":
					switch i {
					case 0:
						nativeKeys(m, "i", "esc")
					case 1:
						nativeSubmitURL(t, m, "https://example.test/new")
					case 2:
						nativeKeys(m, "A", "!")
					case 3:
						nativeSubmitURL(t, m, "")
					case 4:
						nativeKeys(m, "v", "$")
						nativeKeys(m, "d")
					}
				case "cross-paragraph-links":
					nativeKeys(m, "v", "j")
					nativeSubmitURL(t, m, "https://example.test/x")
				}
				c, err := m.editor.nativeDraft.PrepareSave()
				if err != nil || c.HTML() != *e.HTML {
					t.Fatalf("got %q err=%v want %q footer=%s", c.HTML(), err, *e.HTML, m.lastErr)
				}
				if m.editor.draft.Done != m.editor.original.Done {
					t.Fatal("Description changed Task Done")
				}
			})
		}
	}
}
