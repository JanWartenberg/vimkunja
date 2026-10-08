package description

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestViewHardBreakLogicalMotionAndDisclosure(t *testing.T) {
	d := Parse(`<blockquote><p>A<br>B long long long</p></blockquote><p>End</p>`)
	v := NewView(d)
	if strings.Contains(frameText(v.Layout(10, false)), "↵") {
		t.Fatal("unfocused hard-break marker")
	}
	if !strings.Contains(frameText(v.Layout(10, true)), "│↪ ") {
		t.Fatal("quote continuation prefix")
	}
	v.Move("l", 1, 40, 8)
	v.Move("l", 1, 40, 8)
	if v.Cursor().Anchor.Node != d.Snapshot()[0].Children[0].Children[2].ID {
		t.Fatal("left/right stopped at hard break")
	}
	v.Move("0", 1, 40, 8)
	if v.Cursor().Anchor.Node != d.Snapshot()[0].Children[0].Children[0].ID {
		t.Fatal("0 stopped at hard break")
	}
	v.Move("$", 1, 40, 8)
	if v.Cursor().Anchor.Node != d.Snapshot()[0].Children[0].Children[2].ID {
		t.Fatal("$ stopped at hard break")
	}
}
func TestViewWordBoundariesAndBackwardEnd(t *testing.T) {
	d := Parse(`<p>one two three</p><p>other</p>`)
	v := NewView(d)
	v.Move("w", 2, 40, 8)
	v.Move("ge", 1, 40, 8)
	if v.Cursor().Anchor.Offset != 6 {
		t.Fatalf("ge: %#v", v.Cursor())
	}
	v.Move("w", 20, 40, 8)
	if v.Cursor().Anchor.Node != d.Snapshot()[0].Children[0].ID {
		t.Fatal("word motion crossed block")
	}
	v.Move("b", 20, 40, 8)
	if v.Cursor().Anchor.Offset != 0 {
		t.Fatal("backward word boundary")
	}
	v.Move("_", 1, 40, 8)
	if v.Cursor().Anchor.Offset != 0 {
		t.Fatal("_ unsupported")
	}
	v.Move("ctrl+d", 1, 40, 1)
	if v.Cursor().Anchor.Node != d.Snapshot()[1].Children[0].ID {
		t.Fatal("page motion")
	}
}
func TestViewPageAndCountedLineEnd(t *testing.T) {
	d := Parse("<pre><code>a\nb\nc\nd\ne\nf</code></pre>")
	v := NewView(d)
	v.Move("$", 3, 40, 4)
	if v.Cursor().Anchor.Offset != 4 {
		t.Fatalf("3$: %#v", v.Cursor())
	}
	v.Move("gg", 1, 40, 4)
	v.Move("ctrl+d", 1, 40, 4)
	if v.Cursor().Anchor.Offset != 8 {
		t.Fatalf("Ctrl+D height: %#v", v.Cursor())
	}
}
func TestViewCorpusProjectionSafetyAndFocusedContent(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/description-rendering/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Fixtures []struct {
			ID     string `json:"id"`
			Source string `json:"source_html"`
		}
	}
	if err = json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{
		"layout-and-palette":                "Alpha bravo charlie delta echo\nNext",
		"graphemes-and-delimiter-selection": "é 👩‍💻 Text",
		"code-and-quotes":                   "A quote\nItem\n\t<tag>\n\n  end\n",
		"table-mixed-content":               "Before\nKey | Value\nA | B\nAfter",
		"table-irregular":                   "Before\nMerged\nA | B\nAfter",
		"unknown-inner-block-boundaries":    "A\nB\nC\nEnd",
		"image-empty-alt":                   "[Image: Map]",
		"mention-label":                     "Hallo @Jan & Co!",
		"mention-id-fallback":               "Hallo @jan!",
	}
	for _, fixture := range corpus.Fixtures {
		t.Run(fixture.ID, func(t *testing.T) {
			d := Parse(fixture.Source)
			v := NewView(d)
			if want, ok := expected[fixture.ID]; ok && v.ContentText() != want {
				t.Fatalf("logical content: got %q want %q", v.ContentText(), want)
			}
			for _, width := range []int{1, 4, 12, 40, 80} {
				for _, height := range []int{1, 3, 12} {
					anchor := v.Cursor()
					v.Viewport(width, height, true)
					if v.Cursor() != anchor {
						t.Fatal("resize changed anchor")
					}
					for _, motion := range []string{"gg", "gj", "gk", "j", "k", "}", "{", "w", "e", "b", "G", "zh", "zl"} {
						v.Move(motion, 2, width, height)
					}
					f := v.Viewport(width, height, true)
					for _, row := range f.Rows {
						if rowWidth(row) > width {
							t.Fatal("row exceeds viewport")
						}
						for _, span := range row.Spans {
							for _, r := range span.Text {
								if r == 27 || (r < 32 && r != '\n' && r != '\t') || (r >= 127 && r <= 159) {
									t.Fatalf("unsafe control %U", r)
								}
							}
						}
					}
				}
			}
			if v.SourceHTML() != fixture.Source || d.SourceHTML() != fixture.Source {
				t.Fatal("reading changed source")
			}
		})
	}
}

func TestViewWordsAcrossMarksAndLocalNarrowPane(t *testing.T) {
	d := Parse(`<p>hel<strong>lo</strong> world</p><blockquote><blockquote><blockquote><p>deep</p></blockquote></blockquote></blockquote>`)
	v := NewView(d)
	v.Move("w", 1, 40, 8)
	if v.Cursor().Anchor != (Anchor{d.Snapshot()[0].Children[2].ID, 1}) {
		t.Fatalf("split semantic word: %#v", v.Cursor())
	}
	v.Move("gg", 1, 40, 8)
	if v.Layout(8, true).TooNarrow {
		t.Fatal("inactive nested quote blocks reader")
	}
	v.Move("j", 1, 8, 1)
	if v.Cursor().Anchor.Node == d.Snapshot()[0].Children[0].ID {
		t.Fatal("j blocked by later narrow block")
	}
	anchor := v.Cursor()
	v.Viewport(1, 1, true)
	if v.Cursor() != anchor {
		t.Fatal("narrow changed anchor")
	}
}
func TestViewASCIIAndStyledTableFallback(t *testing.T) {
	v := NewView(Parse(`<p>A<br>B</p><hr><pre><code>abcdefghij</code></pre>`), ViewOptions{ASCII: true})
	text := frameText(v.Layout(8, true))
	if strings.ContainsAny(text, "↵█─") || !strings.Contains(text, "A>") {
		t.Fatalf("ASCII fallback: %q", text)
	}
	v = NewView(Parse(`<table><tr><td><p style="color:red">A</p></td><td>B</td></tr></table>`))
	text = frameText(v.Layout(40, false))
	if strings.Contains(text, "┌") || !strings.Contains(text, "A | B") {
		t.Fatalf("styled descendant grid: %q", text)
	}
}

func TestViewHorizontalClampAndWrappedOpaqueChecklist(t *testing.T) {
	v := NewView(Parse(`<pre><code>abcdefghijklmnopqrstuvwxyz</code></pre>`))
	v.Move("zl", 1000, 10, 5)
	before := frameText(v.Layout(10, true))
	v.Hint()
	v.Move("zh", 1, 10, 5)
	if frameText(v.Layout(10, true)) == before {
		t.Fatal("hidden horizontal scroll debt")
	}
	v = NewView(Parse(`<custom><p>Before</p><ul data-type="taskList"><li data-type="taskItem" data-checked="true">Done</li></ul><p>After</p></custom>`))
	text := frameText(v.Layout(80, false))
	if !strings.Contains(text, "☑ Done") || !strings.Contains(text, "Checklist progress: 1/1") {
		t.Fatalf("Opaque checklist regression: %q", text)
	}
}
func TestViewCaptionAndVisualPageDistance(t *testing.T) {
	v := NewView(Parse(`<table><caption>Important caption</caption><tr><td>A</td></tr></table>`))
	if !strings.Contains(frameText(v.Layout(40, false)), "Important caption") {
		t.Fatal("caption hidden")
	}
	d := Parse(`<p>first long paragraph wraps over many rows here</p><p>Next</p>`)
	v = NewView(d)
	v.Move("ctrl+d", 1, 10, 2)
	if v.Cursor().Anchor.Node != d.Snapshot()[0].Children[0].ID {
		t.Fatal("page skipped long paragraph")
	}
	f := v.Layout(10, true)
	if f.CursorY != 2 {
		t.Fatalf("page distance: %d", f.CursorY)
	}
}

func FuzzViewProjectionPreservesSource(f *testing.F) {
	for _, source := range []string{"<p>é👩‍💻<br>Next</p>", "<pre><code>\twide\n\n</code></pre>", "<table><tr><td>A</td><td>B</td></tr></table>", "<custom><p>A</p><p>B</p></custom>", "<script>hidden</script>", "<p><strong>bold</strong></p>"} {
		f.Add(source, uint8(12))
	}
	f.Fuzz(func(t *testing.T, source string, size uint8) {
		if len(source) > 4096 {
			t.Skip()
		}
		d := Parse(source)
		v := NewView(d)
		width := 1 + int(size)%80
		for _, motion := range []string{"j", "gj", "$", "w", "ge", "}", "G", "gg"} {
			v.Move(motion, 3, width, 5)
		}
		frame := v.Viewport(width, 5, true)
		for _, row := range frame.Rows {
			if rowWidth(row) > width {
				t.Fatal("row exceeds width")
			}
		}
		if v.SourceHTML() != source {
			t.Fatal("projection mutated source")
		}
	})
}

func TestViewCodeStickyColumnAcrossHorizontalReveal(t *testing.T) {
	v := NewView(Parse("<pre><code>abcdefghijklmnopqrstuvwxyz\nx\nabcdefghijklmnopqrstuvwxyz</code></pre>"))
	v.Move("l", 20, 10, 5)
	v.Move("j", 1, 10, 5)
	v.Move("j", 1, 10, 5)
	if v.Cursor().Anchor.Offset != 49 {
		t.Fatalf("Code desired column lost: %#v", v.Cursor())
	}
}
func TestViewInlineOpaqueMultilineExtent(t *testing.T) {
	v := NewView(Parse(`<p>Before <span style="color:red">A<br>B<br>C</span> After</p>`))
	text := frameText(v.Layout(80, true))
	if strings.Contains(text, "↵") || !strings.Contains(text, "\n│ B\n│ C]") {
		t.Fatalf("inline atom extent: %q", text)
	}
	v = NewView(Parse(`<p>Before <custom>long words long words</custom> After</p>`))
	text = frameText(v.Layout(12, true))
	if !strings.Contains(text, "\n│ ") {
		t.Fatalf("soft-wrapped inline atom extent: %q", text)
	}
}
func TestViewOpaqueChecklistKeepsNonEntrySiblings(t *testing.T) {
	const source = `<ul data-type="taskList" style="color:red">Before<li data-type="taskItem" data-checked="true">Done</li>After</ul>`
	v := NewView(Parse(source))
	text := frameText(v.Layout(80, false))
	a, b, c := strings.Index(text, "Before"), strings.Index(text, "Done"), strings.Index(text, "After")
	if a < 0 || b <= a || c <= b {
		t.Fatalf("Opaque list drops/reorders text: %q", text)
	}
}

func TestViewCodeResizeRevealsAnchoredCursor(t *testing.T) {
	v := NewView(Parse("<pre><code>" + strings.Repeat("x", 60) + "</code></pre>"))
	v.Move("l", 50, 80, 8)
	v.Viewport(80, 8, true)
	anchor := v.Cursor()
	f := v.Viewport(12, 8, true)
	if v.Cursor() != anchor || f.CursorX < 2 || f.CursorX >= 11 {
		t.Fatalf("resize hid Code cursor: %#v", f)
	}
	v.Move("zh", 2, 12, 8)
	before := frameText(v.Layout(12, true))
	if frameText(v.Viewport(12, 8, true)) != before {
		t.Fatal("render undid explicit horizontal scroll")
	}
}
func TestViewEmptyHeadingsAndFirstNonblankDelimiter(t *testing.T) {
	for _, tag := range []string{"h1", "h2", "h3"} {
		d := Parse("<p>A</p><" + tag + "></" + tag + "><p>B</p>")
		v := NewView(d)
		v.Move("j", 1, 40, 8)
		if v.Cursor().Anchor.Node != d.Snapshot()[1].ID {
			t.Fatal("empty heading skipped")
		}
		v.Move("j", 1, 40, 8)
		if v.Cursor().Anchor.Node != d.Snapshot()[2].Children[0].ID {
			t.Fatal("cannot leave empty heading")
		}
	}
	d := Parse(`<p><strong>bold</strong></p>`)
	v := NewView(d)
	v.Move("$", 1, 40, 8)
	v.Move("^", 1, 40, 8)
	if v.Cursor().Anchor.Node != d.Snapshot()[0].Children[0].ID || v.Cursor().Syntax == 0 {
		t.Fatal("first nonblank skipped editable delimiter")
	}
}
func TestViewWideCodeCursorGraphemeIsFullyVisible(t *testing.T) {
	v := NewView(Parse(`<pre><code>a界界</code></pre>`))
	v.Move("l", 2, 6, 5)
	f := v.Viewport(6, 5, true)
	if f.CursorX+2 > 5 {
		t.Fatalf("wide cursor grapheme clipped: x=%d", f.CursorX)
	}
}

func TestViewCountedDocumentStartUsesCodeSourceLine(t *testing.T) {
	v := NewView(Parse("<pre><code>a\nb\nc</code></pre>"))
	v.Move("gg", 3, 40, 8)
	if v.Cursor().Anchor.Offset != 4 {
		t.Fatalf("3gg lost source line: %#v", v.Cursor())
	}
}
func TestViewOpaqueEntryNumberingAndNativeCheckboxState(t *testing.T) {
	v := NewView(Parse(`<ol><li style="color:red" value="7">Alpha</li><li>Beta</li></ol>`))
	text := frameText(v.Layout(40, false))
	if !strings.Contains(text, "7. [Alpha]") || !strings.Contains(text, "8. Beta") {
		t.Fatalf("Opaque entry numbering: %q", text)
	}
	for _, source := range []string{`<ul data-type="taskList"><li data-type="taskItem" data-checked="TRUE"><p>Done</p></li></ul>`, `<ul data-type="taskList"><li style="color:red" data-type="taskItem" data-checked="true">Done</li></ul>`} {
		v = NewView(Parse(source))
		text = frameText(v.Layout(40, false))
		if !strings.Contains(text, "☑ ") || !strings.Contains(text, "Checklist progress: 1/1") {
			t.Fatalf("checkbox mismatch: %q", text)
		}
	}
}
func TestViewNarrowHintUsesActualViewport(t *testing.T) {
	v := NewView(Parse(`<pre><code>A</code></pre>`))
	v.Viewport(1, 3, true)
	if v.Hint() != "Pane too narrow" {
		t.Fatalf("hint does not reflect current width: %s", v.Hint())
	}
}
func TestViewHardBreakCursorAtFullRowRemainsVisible(t *testing.T) {
	d := Parse(`<p>abcdef<br>Z</p>`)
	v := NewView(d)
	if err := v.SetAnchor(Anchor{d.Snapshot()[0].Children[1].ID, 0}); err != nil {
		t.Fatal(err)
	}
	f := v.Viewport(6, 5, true)
	if f.TooNarrow || f.CursorX < 0 || f.CursorX >= 6 || !strings.Contains(frameText(f), "↵") {
		t.Fatalf("full-row hardbreak cursor/marker hidden: x=%d text=%q", f.CursorX, frameText(f))
	}
}
func TestViewHardBreakAfterOpaqueFrameRemainsVisible(t *testing.T) {
	d := Parse(`<p><img alt="X"><br>Z</p>`)
	v := NewView(d)
	width := rowWidth(v.Layout(80, false).Rows[0])
	if err := v.SetAnchor(Anchor{d.Snapshot()[0].Children[1].ID, 0}); err != nil {
		t.Fatal(err)
	}
	f := v.Viewport(width, 5, true)
	if f.CursorX >= width || !strings.Contains(frameText(f), "↵") {
		t.Fatalf("Opaque-adjacent hardbreak hidden: x=%d width=%d", f.CursorX, width)
	}
}
