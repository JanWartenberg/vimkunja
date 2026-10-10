package tui

import (
	tea "charm.land/bubbletea/v2"
	"encoding/json"
	"os"
	"testing"
)

func TestNativeCodePasteKeepsExactSource(t *testing.T) {
	for _, clipboard := range []bool{false, true} {
		m := nativeEditingModel("<pre><code></code></pre>")
		m = nativeKeys(m, "i")
		const text = "\r\n\tprint(1)\r\n\r\n"
		if clipboard {
			m.Update(clipboardPastedMsg{editor: m.editor, text: text})
		} else {
			m.Update(tea.PasteMsg{Content: text})
		}
		if got := m.editor.draft.Description; got != "<pre><code>"+text+"</code></pre>" {
			t.Fatalf("clipboard=%v: %q", clipboard, got)
		}
	}
}

func TestNativeCodeTypedTriggersAndExits(t *testing.T) {
	for _, chunks := range [][]string{{"`", "`", "`"}, {"``", "``"}, {"````"}} {
		m := nativeEditingModel("<p></p>")
		m = nativeKeys(m, "i")
		m = nativeKeys(m, chunks...)
		m = nativeKeys(m, "enter")
		if got := m.editor.draft.Description; got != "<pre><code></code></pre>" || m.editor.mode != editorInsert {
			t.Fatalf("chunks=%v got=%q", chunks, got)
		}
		m = nativeKeys(m, "esc", "u")
		if got := m.editor.draft.Description; got != "<p>"+joinChunks(chunks)+"</p>" {
			t.Fatal(got)
		}
	}
	for _, key := range []string{"ü", "Ü", "]", "["} {
		m := nativeEditingModel("<blockquote><pre><code>print(1)</code></pre></blockquote>")
		m = nativeKeys(m, key)
		if key == "]" || key == "[" {
			m = nativeKeys(m, "space")
		}
		want := "<blockquote><pre><code>print(1)</code></pre><p></p></blockquote>"
		if key == "Ü" || key == "[" {
			want = "<blockquote><p></p><pre><code>print(1)</code></pre></blockquote>"
		}
		if got := m.editor.draft.Description; got != want {
			t.Fatalf("key=%s got=%q", key, got)
		}
	}
	m := nativeEditingModel("<p></p>")
	m = nativeKeys(m, "space", "c")
	if m.editor.draft.Description != "<pre><code></code></pre>" || m.editor.mode != editorInsert {
		t.Fatal(m.editor.draft.Description)
	}
}
func joinChunks(chunks []string) string {
	s := ""
	for _, c := range chunks {
		s += c
	}
	return s
}

func codeControl(m *model, code rune) *model {
	return pressKey(m, tea.Key{Code: code, Mod: tea.ModCtrl})
}
func codeTab(m *model) *model { return pressKey(m, tea.Key{Code: tea.KeyTab}) }

func TestNativeCodeTabPrefixesAndConfiguredAlias(t *testing.T) {
	for _, prefix := range []rune{'v', 'q', 'b'} {
		m := nativeEditingModel("<pre><code>A</code></pre>")
		m.codeConfig.LiteralTabAlias = "ctrl+" + string(prefix)
		m = nativeKeys(m, "i")
		m = codeControl(m, prefix)
		m = codeTab(m)
		if got := m.editor.draft.Description; got != "<pre><code>\tA</code></pre>" {
			t.Fatalf("prefix=%c got=%q", prefix, got)
		}
		m = codeControl(m, prefix)
		m = nativeKeys(m, "esc")
		if m.editor.nativeLiteralTab || m.editor.mode != editorNormal {
			t.Fatal("prefix Esc did not cancel")
		}
	}
	m := nativeEditingModel("<pre><code>界A</code></pre>")
	m = nativeKeys(m, "a")
	m = codeTab(m)
	if m.editor.draft.Description != "<pre><code>界  A</code></pre>" {
		t.Fatal(m.editor.draft.Description)
	}
	m = codeControl(m, 'v')
	m = nativeKeys(m, "x")
	if m.lastErr == "" || m.editor.nativeLiteralTab || m.editor.draft.Description != "<pre><code>界  A</code></pre>" {
		t.Fatal("invalid prefix must report error without editing")
	}
}

func TestNativeCodeTriggerRejectsPasteImportAndExtraText(t *testing.T) {
	for _, text := range []string{"`````", "````x"} {
		m := nativeEditingModel("<p></p>")
		m = nativeKeys(m, "i", text, "enter")
		if got := m.editor.draft.Description; got != "<p>"+text+"</p><p></p>" {
			t.Fatal(got)
		}
	}
	m := nativeEditingModel("<p></p>")
	m = nativeKeys(m, "i")
	m.Update(tea.PasteMsg{Content: "````"})
	m = nativeKeys(m, "enter")
	if got := m.editor.draft.Description; got != "<p>````</p><p></p>" {
		t.Fatal(got)
	}
	m = nativeEditingModel("<p>```</p>")
	m = nativeKeys(m, "A", "enter")
	if got := m.editor.draft.Description; got != "<p>```</p><p></p>" {
		t.Fatal(got)
	}
	m = nativeEditingModel("<p></p>")
	m = nativeKeys(m, "i", "````")
	m = pressKey(m, tea.Key{Code: tea.KeyBackspace})
	m = nativeKeys(m, "enter")
	if got := m.editor.draft.Description; got != "<pre><code></code></pre>" {
		t.Fatal(got)
	}
}

func TestNativeCodeSelectionIndentRetabAndUndo(t *testing.T) {
	m := nativeEditingModel("<pre><code>\tA\n  B\n\nC</code></pre><p>outside</p>")
	m.codeConfig.IndentStep = 2
	m = nativeKeys(m, "V", "j", "j", "j", "space", "x")
	if got := m.editor.draft.Description; got != "<pre><code>  A\nB\n\nC</code></pre><p>outside</p>" {
		t.Fatal(got)
	}
	m = nativeKeys(m, "u", "space", "c")
	if got := m.editor.draft.Description; got != "<pre><code>  \tA\n    B\n\n  C</code></pre><p>outside</p>" {
		t.Fatal(got)
	}
	m = nativeKeys(m, "u", "esc", "j", "V")
	retabDescriptionCommand(m, nil)
	if got := m.editor.draft.Description; got != "<pre><code>\tA\n  B\n\nC</code></pre><p>outside</p>" {
		t.Fatal(got)
	}
	retabDescriptionCommand(m, []string{"8"})
	if m.lastErr == "" {
		t.Fatal("retab args not rejected")
	}
}

func TestNativeCodeEscapedMarkerStaysLiteral(t *testing.T) {
	for _, chunks := range [][]string{{"\\", "````"}, {"\\````"}} {
		m := nativeEditingModel("<p></p>")
		m = nativeKeys(m, "i")
		m = nativeKeys(m, chunks...)
		m = nativeKeys(m, "enter")
		if got := m.editor.draft.Description; got != "<p>````</p><p></p>" {
			t.Fatalf("%v %q", chunks, got)
		}
	}
}

func TestNativeCodeLastLineDDAndCRLFEvents(t *testing.T) {
	m := nativeEditingModel("<pre><code>print(1)</code></pre>")
	m = nativeKeys(m, "d", "d")
	if got := m.editor.draft.Description; got != "<pre><code></code></pre>" {
		t.Fatal(got)
	}
	m = nativeKeys(m, "d", "d")
	if got := m.editor.draft.Description; got != "" {
		t.Fatal(got)
	}
	m = nativeEditingModel("<pre><code>a\r\nb</code></pre>")
	m = nativeKeys(m, "o", "esc")
	if got := m.editor.draft.Description; got != "<pre><code>a\r\n\r\nb</code></pre>" {
		t.Fatal(got)
	}
	m = nativeKeys(m, "u", "O", "esc")
	if got := m.editor.draft.Description; got != "<pre><code>\r\na\r\nb</code></pre>" {
		t.Fatal(got)
	}
}

func TestNativeCodeRegressionCommentPasteIsolation(t *testing.T) {
	m := nativeEditingModel("<pre><code></code></pre>")
	m = nativeKeys(m, "i")
	m.comments = &commentsUI{composer: &commentComposer{mode: editorInsert}}
	m.Update(tea.PasteMsg{Content: "a\r\nb"})
	if got := m.comments.composer.text; got != "a\nb" {
		t.Fatalf("comment paste=%q", got)
	}
}
func TestNativeCodeRegressionLiteralPrefixFocusCancellation(t *testing.T) {
	m := nativeEditingModel("<pre><code>A</code></pre>")
	m = nativeKeys(m, "i")
	m = codeControl(m, 'v')
	m = codeControl(m, 'w')
	m = nativeKeys(m, "h")
	m.focus = paneDetail
	m = codeTab(m)
	if got := m.editor.draft.Description; got != "<pre><code>    A</code></pre>" {
		t.Fatalf("after focus switch=%q", got)
	}
}

func TestNativeCodeRegressionEmptyCreation(t *testing.T) {
	m := nativeEditingModel("")
	m = nativeKeys(m, "space", "c")
	if got := m.editor.draft.Description; got != "<pre><code></code></pre>" {
		t.Fatalf("empty direct creation %q", got)
	}
}
func TestNativeCodeRegressionBracketCancellation(t *testing.T) {
	m := nativeEditingModel("<pre><code>A</code></pre>")
	m = nativeKeys(m, "]", "ü", "esc", "space")
	if !m.leaderArmed || m.lastErr != "" || m.editor.nativePrefix != "" {
		t.Fatal("stale bracket consumed leader")
	}
	m = nativeKeys(m, "c")
	if m.editor.draft.Description != "<pre><code>A</code></pre><pre><code></code></pre>" {
		t.Fatal(m.editor.draft.Description)
	}
}

func TestNativeCodeRegressionCRLFAppend(t *testing.T) {
	m := nativeEditingModel("<pre><code>ab\r\ncd</code></pre>")
	m = nativeKeys(m, "A", "X", "esc")
	if got := m.editor.draft.Description; got != "<pre><code>abX\r\ncd</code></pre>" {
		t.Fatalf("append at CRLF line end %q", got)
	}
}

func TestNativeCodeAcceptedCorpus(t *testing.T) {
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
	for _, f := range corpus.Fixtures {
		if f.ID != "standalone-code-editing" && f.ID != "code-trigger-events" && f.ID != "code-exit-boundaries" && f.ID != "code-exit-nested" && f.ID != "code-exit-list-entry" && f.ID != "code-tabs-retab" && f.ID != "retaball-native-scope" && f.ID != "code-indent-outdent" && f.ID != "exact-code-import" {
			continue
		}
		for index, e := range f.Edits {
			t.Run(f.ID+"/"+string(rune('0'+index)), func(t *testing.T) {
				m := nativeEditingModel(f.Source)
				switch f.ID {
				case "standalone-code-editing":
					switch index {
					case 0:
						m = nativeKeys(m, "o")
					case 1:
						m = nativeKeys(m, "O")
					case 2:
						m = nativeKeys(m, "a", "enter")
					case 3:
						m = nativeKeys(m, "i")
						m.Update(tea.PasteMsg{Content: "\t<tag>\n  Z"})
					case 4:
						m = nativeKeys(m, "d", "d")
					case 5:
						m = nativeKeys(m, "i", "esc")
					case 6:
						m = nativeKeys(m, "d", "d", "d", "d")
					case 7:
						m = nativeKeys(m, "d", "d", "d", "d", "d", "d")
					}
				case "code-trigger-events":
					if index == 9 {
						m = nativeKeys(m, "space", "c")
						break
					}
					m = nativeKeys(m, "i")
					switch index {
					case 0:
						m = nativeKeys(m, "```", "enter")
					case 1:
						m = nativeKeys(m, "````", "esc")
					case 2:
						m = nativeKeys(m, "`````", "enter")
					case 3:
						m = nativeKeys(m, "````x", "enter")
					case 4:
						m.Update(tea.PasteMsg{Content: "````"})
						m = nativeKeys(m, "enter")
					case 5:
						m = nativeKeys(m, "\\", "````", "enter")
					case 6:
						m = nativeKeys(m, "````")
						m = pressKey(m, tea.Key{Code: tea.KeyBackspace})
						m = nativeKeys(m, "enter")
					case 7:
						m = nativeKeys(m, "````", "enter", "`")
					case 8:
						m = nativeKeys(m, "```", "enter", "esc", "u")
					}
				case "code-exit-boundaries":
					paths := [][]string{{"ü"}, {"Ü"}, {"]", "space"}, {"[", "space"}, {"ü", "esc"}, {"ü", "esc", "u"}}
					m = nativeKeys(m, paths[index]...)
				case "code-exit-nested":
					if index == 0 {
						m = nativeKeys(m, "ü")
					} else {
						m = nativeKeys(m, "Ü")
					}
				case "code-exit-list-entry":
					m = nativeKeys(m, "ü")
				case "code-tabs-retab":
					switch index {
					case 0:
						retabDescriptionCommand(m, nil)
					case 1:
						m = nativeKeys(m, "j", "V")
						retabDescriptionCommand(m, nil)
					case 2:
						m = nativeKeys(m, "i")
						m = codeTab(m)
					case 3:
						m = nativeKeys(m, "i")
						m = codeControl(m, 'v')
						m = codeTab(m)
					case 4:
						retabDescriptionCommand(m, nil)
						m = nativeKeys(m, "u")
					case 5:
						m.codeConfig.Tabstop = 8
						retabDescriptionCommand(m, nil)
					}
				case "retaball-native-scope":
					retabAllDescriptionCommand(m, nil)
				case "code-indent-outdent":
					m.codeConfig.IndentStep = 2
					m = nativeKeys(m, "V", "j", "j", "j", "space")
					if index == 0 {
						m = nativeKeys(m, "x")
					} else {
						m = nativeKeys(m, "c")
					}
				case "exact-code-import":
					if index == 0 {
						m = nativeKeys(m, "i", "esc")
					} else {
						m = nativeKeys(m, "j", "i")
						m.Update(tea.PasteMsg{Content: "X\r\n\t"})
					}
				}
				candidate, err := m.editor.nativeDraft.PrepareSave()
				if err != nil || e.HTML == nil || candidate.HTML() != *e.HTML {
					t.Fatalf("got %q err=%v; want %v", candidate.HTML(), err, e.HTML)
				}
				if m.editor.draft.Done != m.editor.original.Done {
					t.Fatal("Code changed Task Done")
				}
			})
		}
	}
}

func TestNativeCodeOpenBelowInsertion(t *testing.T) {
	m := nativeEditingModel("<pre><code>abc\ndef</code></pre>")
	m = nativeKeys(m, "o", "X", "esc")
	if got := m.editor.draft.Description; got != "<pre><code>abc\nX\ndef</code></pre>" {
		t.Fatalf("got %q cursor %+v", got, m.editor.nativeDraft.Cursor())
	}
}

func TestNativeCodeLiteralPrefixCtrlO(t *testing.T) {
	m := nativeEditingModel("<pre><code>abc</code></pre>")
	m = nativeKeys(m, "i", "X")
	m.searchReturn = &searchReturn{}
	m = codeControl(m, 'v')
	m = codeControl(m, 'o')
	if m.editor.nativeLiteralTab {
		t.Fatal("Ctrl+O was dispatched globally and left literal-Tab prefix armed")
	}
}

func TestNativeCodeBracketEnterCancellation(t *testing.T) {
	m := nativeEditingModel("<pre><code>A</code></pre>")
	m = nativeKeys(m, "]", "enter", "space")
	if m.editor.draft.Description != "<pre><code>A</code></pre>" || !m.leaderArmed {
		t.Fatal("Enter preserved bracket prefix")
	}
}
