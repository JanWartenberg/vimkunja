package tui

import (
	tea "charm.land/bubbletea/v2"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"os"
	"strings"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/description"
)

func TestNativeCompatibilityCorpusEdits(t *testing.T) {
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
	if err = json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	replacements := map[string][2]string{
		"nested-lists": {"Nested", "Inner"}, "opaque-node": {"after", "later"}, "opaque-nesting": {"End", "Last"}, "attribute-preservation": {"Text", "Words"}, "table-mixed-content": {"After", "Later"}, "unknown-node": {"B", "C"}, "opaque-inline-q": {"after", "later"}, "unsafe-html": {"After", "Later"}, "unsafe-isolated-and-opaque": {"After", "Later"}, "valid-optional-end-tags": {"B", "D"}, "empty-table": {"B", "C"}, "unknown-inner-block-boundaries": {"End", "Last"}, "embedded-media-neighbors": {"B", "C"}, "opaque-identical-atom-identity": {"B", "middle"}, "mention-label": {"Hallo", "Guten Tag"}, "mention-id-fallback": {"Hallo", "Guten Tag"},
	}
	for _, f := range corpus.Fixtures {
		indices := []int{}
		if _, ok := replacements[f.ID]; ok {
			indices = []int{0}
		}
		switch f.ID {
		case "hard-breaks", "image-empty-alt":
			indices = []int{0}
		case "malformed-html":
			indices = []int{0, 1}
		case "document-wrappers":
			indices = []int{0}
		case "layout-and-palette":
			indices = []int{0, 1}
		case "checkbox-middle-split":
			indices = []int{1}
		case "list-editing", "ordered-list-editing":
			for i := range f.Edits {
				indices = append(indices, i)
			}
		case "basic-blocks", "inline-boundaries", "uniform-toggle", "image-fallbacks", "table-irregular", "empty-table":
			indices = make([]int, len(f.Edits))
			for i := range indices {
				indices[i] = i
			}
		case "inline-marks", "entities-and-whitespace", "serialization-normalization", "root-inline-fragment", "empty-description", "code-and-quotes", "root-block-grouping":
			indices = []int{0}
		case "empty-structures", "all-mark-families-off":
			indices = []int{0, 1, 2}
		case "paragraph-creation-and-split":
			indices = []int{0, 1, 2, 3, 4, 5, 6}
		case "native-selection-and-history":
			indices = []int{0, 1, 2, 3, 4}
		case "opaque-edit-neighbors":
			indices = []int{0, 1, 2, 3}
		case "graphemes-and-delimiter-selection":
			indices = []int{0, 1, 2}
		case "opaque-identical-atom-identity":
			indices = []int{0, 1, 3}
		case "unsafe-attributes-and-urls", "unsafe-url-only":
			indices = []int{0, 1}
		case "duplicate-native-attributes", "unsafe-html", "unsafe-isolated-and-opaque":
			indices = []int{0, 1}
		case "mention-label", "mention-id-fallback":
			indices = []int{0, 1, 2, 3, 4, 5, 6}
		}
		for _, i := range indices {
			if f.Edits[i].HTML == nil {
				t.Fatalf("case %s/%d needs assertions", f.ID, i)
			}
			t.Run(fmt.Sprintf("%s/%d", f.ID, i), func(t *testing.T) {
				m := nativeEditingModel(f.Source)
				m.editor.descriptionView()
				at := func(match func(description.Node) bool, offset int) description.Node {
					n := nativeFindNode(t, m, match)
					if err := m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID, Offset: offset}); err != nil {
						t.Fatal(err)
					}
					m.editor.nativeDraft.UpdateView(m.editor.nativeDescription)
					return n
				}
				text := func(value string, offset int) description.Node {
					return at(func(n description.Node) bool { return n.Kind == description.Text && strings.Contains(n.Text, value) }, offset)
				}
				atom := func(index int) description.Node {
					seen := 0
					return at(func(n description.Node) bool {
						if n.Kind != description.Opaque {
							return false
						}
						seen++
						return seen == index
					}, 0)
				}
				replace := func(from, to string) {
					n := nativeFindNode(t, m, func(n description.Node) bool { return n.Kind == description.Text && strings.Contains(n.Text, from) })
					start := strings.Index(n.Text, from)
					m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID, Offset: start})
					m.editor.nativeDraft.Select(false, false)
					m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID, Offset: start + len(from) - len(string([]rune(from)[len([]rune(from))-1]))})
					m.editor.nativeDraft.UpdateView(m.editor.nativeDescription)
					nativeKeys(m, "c", to, "esc")
				}
				whole := func() { nativeKeys(m, "g", "g", "v", "G", "$") }
				switch f.ID {
				case "hard-breaks":
					seen := 0
					at(func(n description.Node) bool {
						if n.Tag == "br" {
							seen++
							return seen == 4
						}
						return false
					}, 0)
					nativeKeys(m, "x")
				case "malformed-html", "document-wrappers":
					command := "descriptionrecover"
					if i == 1 {
						command = "descriptionclear"
					}
					nativeKeys(m, ":", command, "enter", "y")
				case "image-empty-alt":
					atom(1)
					nativeKeys(m, "y")
					if m.editor.nativeRegister != "[Image: Map]" {
						t.Fatal(m.editor.nativeRegister)
					}
				case "layout-and-palette":
					text("bravo charlie", 6)
					if i == 0 {
						nativeKeys(m, "j", "k", "g", "j", "g", "k")
					} else {
						nativeKeys(m, "space", "esc")
					}
				case "checkbox-middle-split":
					text("AlphaBeta", 5)
					nativeKeys(m, "i", "enter", "esc")
					nativeKeys(m, ":", "e!", "enter")
				case "list-editing", "ordered-list-editing":
					if i == 0 {
						at(func(n description.Node) bool { return n.Tag == "li" && len(n.Children) == 0 }, 0)
						m = nativeKeys(m, "i")
						m = nativeKeys(m, "enter")
						m = nativeKeys(m, "esc")
					} else if i == 1 {
						text("First", len("First"))
						m = nativeKeys(m, "a")
						m = nativeKeys(m, "enter")
						m = nativeKeys(m, "esc")
					} else {
						text("First", 0)
						m = nativeKeys(m, "V", "j", "d")
					}
				case "basic-blocks":
					at(func(n description.Node) bool { return n.Tag == "hr" }, 0)
					if i == 0 {
						m.editor.nativeDraft.SetTarget(description.Target{Anchor: m.editor.nativeDraft.Cursor(), Syntax: 1})
						nativeKeys(m, "x")
					} else {
						nativeKeys(m, "d", "d")
					}
				case "inline-marks":
					whole()
					nativeKeys(m, "space", "0")
				case "inline-boundaries":
					if i == 0 {
						replace("β", "γ")
					} else {
						n := text("deep", 0)
						m.editor.nativeDraft.Select(false, false)
						m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID, Offset: 3})
						nativeKeys(m, "space", "b")
					}
				case "uniform-toggle":
					whole()
					nativeKeys(m, "space", "b")
					if i == 1 {
						nativeKeys(m, "space", "b")
					}
				case "entities-and-whitespace":
					n := nativeFindNode(t, m, func(n description.Node) bool { return n.Kind == description.Text && strings.Contains(n.Text, "é") })
					at(func(x description.Node) bool { return x.ID == n.ID }, strings.Index(n.Text, "é"))
					nativeKeys(m, "a", "!", "esc")
				case "serialization-normalization":
					nativeKeys(m, "space", "2")
				case "root-inline-fragment":
					replace("!", "?")
				case "empty-description":
					m.editor.nativeDraft = description.NewDraft(description.Parse(f.Source))
					m.editor.nativeDraft.Apply(description.Action{Kind: description.Insert, Text: "Hello"}, false)
				case "empty-structures":
					tag := []string{"li", "pre", "blockquote"}[i]
					at(func(n description.Node) bool { return n.Tag == tag }, 0)
					nativeKeys(m, "d", "d")
				case "code-and-quotes":
					n := nativeFindNode(t, m, func(n description.Node) bool { return n.Kind == description.Text && strings.Contains(n.Text, "\n\n") })
					at(func(x description.Node) bool { return x.ID == n.ID }, strings.Index(n.Text, "\n\n")+1)
					nativeKeys(m, "d", "d")
				case "paragraph-creation-and-split":
					switch i {
					case 0:
						nativeKeys(m, "o", "esc")
					case 1:
						nativeKeys(m, "O", "esc")
					case 2:
						text("AlphaBeta", 5)
						nativeKeys(m, "i", "enter", "esc")
					case 3:
						text("AlphaBeta", 5)
						nativeKeys(m, "i", "shift+enter", "esc")
					case 4:
						text("Next", 0)
						nativeKeys(m, "i")
						m = pressKey(m, tea.Key{Code: tea.KeyBackspace})
						nativeKeys(m, "esc")
					case 5:
						text("AlphaBeta", 8)
						nativeKeys(m, "a", "delete", "esc")
					case 6:
						nativeKeys(m, "space", "2", "space", "2")
					}
				case "native-selection-and-history":
					switch i {
					case 0:
						text("Alpha beta", 6)
						nativeKeys(m, "v", "i", "w", "space", "b")
					case 1:
						nativeKeys(m, "V", "d")
					case 2:
						text("Alpha beta", 2)
						nativeKeys(m, "v", "l", "l", "d")
					case 3:
						nativeKeys(m, "d", "d", "u")
					case 4:
						nativeKeys(m, "d", "d", "u", "ctrl+r")
					}
				case "image-fallbacks":
					atom(2)
					nativeKeys(m, "x", "y")
					if i == 1 {
						nativeKeys(m, "u")
					}
				case "table-irregular":
					atom(1)
					nativeKeys(m, "d", "d", "n")
				case "empty-table":
					if i == 0 {
						replace("B", "C")
					} else {
						atom(1)
						nativeKeys(m, "d", "d", "n")
					}
				case "all-mark-families-off":
					value := []string{"I", "U", "S"}[i]
					n := text(value, 0)
					m.editor.nativeDraft.Select(false, false)
					m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID, Offset: 0})
					nativeKeys(m, "space", []string{"i", "u", "s"}[i])
				case "opaque-edit-neighbors":
					if i == 3 {
						replace("left", "L")
						replace("right", "R")
					} else {
						nativeKeys(m, "d", "d")
						if i == 1 {
							nativeKeys(m, "n")
						} else {
							nativeKeys(m, "y")
							if i == 2 {
								nativeKeys(m, "u")
							}
						}
					}
				case "opaque-identical-atom-identity":
					switch i {
					case 0:
						replace("B", "middle")
					case 1:
						atom(2)
						nativeKeys(m, "x", "y", "u")
					case 3:
						nativeKeys(m, "space", "2", "y")
					}
				case "graphemes-and-delimiter-selection":
					switch i {
					case 0:
						text("é", 0)
						nativeKeys(m, "x")
					case 1:
						text("👩‍💻", len("é "))
						nativeKeys(m, "x")
					case 2:
						at(func(n description.Node) bool { return n.Tag == "strong" }, 0)
						m.editor.nativeDraft.SetTarget(description.Target{Anchor: m.editor.nativeDraft.Cursor(), Syntax: 1})
						m.editor.nativeDraft.UpdateView(m.editor.nativeDescription)
						nativeKeys(m, "v", "i", "w", "space", "i")
					}
				case "unsafe-attributes-and-urls", "unsafe-url-only":
					n := atom(1)
					if i == 0 {
						m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID, Offset: 0})
						m.editor.nativeDraft.UpdateView(m.editor.nativeDescription)
						nativeKeys(m, "a", " after", "esc")
					} else {
						if err := m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID, Offset: 2}); err == nil {
							t.Fatal("internal Opaque anchor accepted")
						}
					}
				case "duplicate-native-attributes", "unsafe-html", "unsafe-isolated-and-opaque":
					if i == 1 {
						n := atom(1)
						if err := m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID, Offset: 2}); err == nil {
							t.Fatal("internal Opaque anchor accepted")
						}
					} else if pair, ok := replacements[f.ID]; ok {
						replace(pair[0], pair[1])
					}
				case "mention-label", "mention-id-fallback":
					switch i {
					case 0:
						replace("Hallo", "Guten Tag")
					case 1:
						whole()
						nativeKeys(m, "space", "b")
					case 2:
						n := atom(1)
						if err := m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID, Offset: 2}); err == nil {
							t.Fatal("internal Mention anchor accepted")
						}
					case 3:
						atom(1)
						nativeKeys(m, "y")
						n := nativeFindNode(t, m, func(n description.Node) bool { return n.Kind == description.Text && n.Text == "!" })
						at(func(x description.Node) bool { return x.ID == n.ID }, 0)
						nativeKeys(m, "a")
						m.insertEditorText(m.editor.nativeRegister)
						nativeKeys(m, "esc")
					case 4:
						atom(1)
						nativeKeys(m, "x", "y")
					case 5:
						atom(1)
						nativeKeys(m, "x", "n")
					case 6:
						atom(1)
						nativeKeys(m, "x", "y", "u")
					}
				case "root-block-grouping":
				default:
					pair := replacements[f.ID]
					replace(pair[0], pair[1])
				}
				if m.editor.nativeDraft == nil {
					m.editor.descriptionView()
				}
				candidate, err := m.editor.nativeDraft.PrepareSave()
				if err != nil {
					t.Fatal(err)
				}
				if got := candidate.HTML(); got != *f.Edits[i].HTML {
					t.Fatalf("got %q\nwant %q\nerror %s", got, *f.Edits[i].HTML, m.lastErr)
				}
				if m.editor.draft.Done {
					t.Fatal("Description action changed Task Done")
				}
			})
		}
	}
}

// Optional evidence export records actual model renders, never a display golden
// and never a claim about a physical terminal's event decoder.
func TestNativeCompatibilityLayoutEvidence(t *testing.T) {
	const source = `<blockquote data-comment-id="42"><p>Quote with <strong>marked text</strong>.</p></blockquote><ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="A"><label><input type="checkbox" checked="checked"><span></span></label><div><p>Alpha</p></div></li></ul><p>Before <mention-user data-id="jan" data-label="Jan"></mention-user> after.</p><pre><code>one
	long_line_for_horizontal_scrolling()</code></pre><table><tr><td>Wide</td><td>Table</td></tr></table>`
	m := nativeEditingModel(source)
	m.editor.descriptionView()
	nativeContentCursor(t, m, "Alpha", 0)
	before := m.editor.nativeDraft.DraftHTML()
	anchor := m.editor.nativeDraft.Cursor()
	var captures []struct {
		Width, Height int
		Display       string
	}
	for _, size := range [][2]int{{120, 40}, {54, 12}, {20, 6}, {120, 40}} {
		updated, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = updated.(*model)
		captures = append(captures, struct {
			Width, Height int
			Display       string
		}{size[0], size[1], ansi.Strip(m.Render())})
		if m.editor.nativeDraft.DraftHTML() != before || m.editor.nativeDraft.Cursor() != anchor {
			t.Fatal("resize mutated draft or anchor")
		}
	}
	if path := os.Getenv("VIMKUNJA_LAYOUT_EVIDENCE"); path != "" {
		data, err := json.MarshalIndent(struct {
			Kind, Source string
			Captures     any
		}{"actual-tui-model-render-synthetic-size-events", source, captures}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCapturedCheckboxRealUIEditing(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/description-rendering/captures/upstream-editor-2026-10-11.json")
	if err != nil {
		t.Fatal(err)
	}
	var captured struct{ Captures []struct{ ID, HTML string } }
	if err := json.Unmarshal(data, &captured); err != nil {
		t.Fatal(err)
	}
	source := ""
	for _, c := range captured.Captures {
		if c.ID == "checkboxes" {
			source = c.HTML
		}
	}
	if source == "" {
		t.Fatal("missing checkbox capture")
	}
	m := nativeEditingModel(source)
	m.editor.descriptionView()
	nativeContentCursor(t, m, "Alpha", 0)
	nativeKeys(m, "x")
	if m.lastErr != "" {
		t.Fatal(m.lastErr)
	}
	nativeContentCursor(t, m, "Alpha", 0)
	nativeKeys(m, "i", "!", "esc")
	m.editor.descriptionView()
	candidate, err := m.editor.nativeDraft.PrepareSave()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(candidate.HTML(), "<p>!Alpha</p>") || strings.Contains(candidate.HTML(), `checked="checked"`) {
		t.Fatal(candidate.HTML())
	}
	if m.editor.draft.Done {
		t.Fatal("checkbox action changed Task Done")
	}
}

func TestNativeCompatibilityStructureCorpusEdits(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/description-rendering/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Fixtures []struct {
			ID     string
			Source string `json:"source_html"`
			Edits  []struct {
				HTML *string `json:"expected_html"`
			}
		}
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	supported := map[string]bool{"quote-shared-source-split": true, "quote-unique-attribute-split": true, "quote-shared-wrap-atomic": true, "list-indent-existing-children": true, "list-outdent-reading-order": true, "list-selection-atomicity": true, "list-kind-preservation": true, "ordered-middle-exit": true, "empty-imported-ul": true, "empty-imported-ol": true, "empty-imported-checkbox": true, "opaque-parent-conversion": true, "triple-strong-off": true}
	for _, f := range corpus.Fixtures {
		if !supported[f.ID] {
			continue
		}
		for i, edit := range f.Edits {
			if edit.HTML == nil {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", f.ID, i), func(t *testing.T) {
				m := nativeEditingModel(f.Source)
				m.editor.descriptionView()
				at := func(text string) { nativeContentCursor(t, m, text, 0) }
				selectTo := func(first, last string) { at(first); nativeKeys(m, "V"); at(last) }
				switch f.ID {
				case "quote-shared-source-split":
					at("B")
					nativeKeys(m, "space", "q")
					if i >= 1 {
						nativeKeys(m, "u")
					}
					if i == 2 {
						nativeKeys(m, "ctrl+r")
					}
				case "quote-unique-attribute-split":
					if i == 0 {
						at("B")
						nativeKeys(m, "space", "q")
						if m.lastErr == "" {
							t.Fatal("unique split did not reject")
						}
					} else {
						selectTo("A", "C")
						nativeKeys(m, "space", "q")
						if i == 1 {
							nativeKeys(m, "y")
						} else {
							nativeKeys(m, "n")
						}
					}
				case "quote-shared-wrap-atomic":
					if i == 0 {
						selectTo("A", "B")
						nativeKeys(m, "space", "w")
					} else {
						selectTo("B", "C")
						nativeKeys(m, "space", "q")
						if m.lastErr == "" {
							t.Fatal("mixed quote operation did not reject")
						}
					}
				case "list-indent-existing-children":
					if i == 0 {
						selectTo("B", "B")
					} else {
						selectTo("B", "C")
					}
					nativeKeys(m, "space", "l")
				case "list-outdent-reading-order":
					selectTo("B", "B")
					nativeKeys(m, "space", "k")
				case "list-selection-atomicity":
					switch i {
					case 0:
						selectTo("A", "B")
					case 1:
						selectTo("B", "child")
					case 2:
						selectTo("old", "C")
					case 3:
						selectTo("A", "A")
					}
					if i == 3 {
						nativeKeys(m, "space", "k")
					} else {
						nativeKeys(m, "space", "l")
					}
					if i != 1 && m.lastErr == "" {
						t.Fatal("incompatible list move did not reject")
					}
				case "list-kind-preservation":
					if i == 0 {
						selectTo("B", "B")
						nativeKeys(m, "space", "l")
					} else {
						at("Child")
						nativeKeys(m, "i")
						m = pressKey(m, tea.Key{Code: tea.KeyBackspace})
						nativeKeys(m, "esc")
					}
				case "ordered-middle-exit":
					n := nativeFindNode(t, m, func(n description.Node) bool { return n.Tag == "li" && len(n.Children) == 0 })
					m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID})
					m.editor.nativeDraft.UpdateView(m.editor.nativeDescription)
					nativeKeys(m, "i", "enter", "esc")
				case "empty-imported-ul", "empty-imported-ol", "empty-imported-checkbox":
					switch i {
					case 0:
						nativeKeys(m, "i", "esc")
					case 1:
						nativeKeys(m, "i", "enter", "esc")
					case 2:
						nativeKeys(m, "i", "Hello", "esc")
					case 3:
						nativeKeys(m, "i", "Hello", "esc", "d", "d")
					}
				case "opaque-parent-conversion":
					nativeKeys(m, "space", "2")
					if i == 0 {
						nativeKeys(m, "y")
					} else {
						nativeKeys(m, "n")
					}
				case "triple-strong-off":
					at("deep")
					nativeKeys(m, "v")
					nativeContentCursor(t, m, "deep", 3)
					nativeKeys(m, "space", "b", "esc")
				}
				if m.editor.nativeDraft == nil {
					m.editor.descriptionView()
				}
				candidate, err := m.editor.nativeDraft.PrepareSave()
				if err != nil {
					t.Fatal(err)
				}
				if candidate.HTML() != *edit.HTML {
					t.Fatalf("got %s\nwant %s\nerror: %s", candidate.HTML(), *edit.HTML, m.lastErr)
				}
				if m.editor.draft.Done {
					t.Fatal("Description action changed Task Done")
				}
			})
		}
	}
}
