package description

import (
	"strings"
	"testing"
)

func TestDraftDeliberateSyntaxVersusLiteralInsert(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"***text***", `<p><strong><em>text</em></strong></p>`},
		{"`npm test`", `<p><code>npm test</code></p>`},
		{"`a\\`b`", `<p><code>a` + "`" + `b</code></p>`},
		{"\\*text\\*", `<p>*text*</p>`},
		{"\\\\*text*", `<p>\<em>text</em></p>`},
		{"**bold *italic** tail*", `<p>**bold *italic** tail*</p>`},
		{"**text", `<p>**text</p>`},
		{"X``", `<p>X` + "``" + `</p>`},
	} {
		d := NewDraft(Parse(`<p></p>`))
		for _, r := range tc.input {
			if err := d.Apply(Action{Kind: Type, Text: string(r)}, false); err != nil {
				t.Fatal(err)
			}
		}
		if got, err := d.HTML(); err != nil || got != tc.want {
			t.Errorf("typed %q: %q %v want %q", tc.input, got, err, tc.want)
		}
		d = NewDraft(Parse(`<p></p>`))
		d.Apply(Action{Kind: Insert, Text: tc.input}, false)
		if got, err := d.HTML(); err != nil || got != `<p>`+tc.input+`</p>` {
			t.Errorf("paste %q: %q %v", tc.input, got, err)
		}
	}
}

func TestDraftImportedLiteralEscapesActivateOnlyChangedRegion(t *testing.T) {
	d := NewDraft(Parse("<p>**Milch** `test`</p>"))
	id := d.Cursor().Node
	for _, off := range []int{0, 1, 7, 8} {
		d.SetTarget(Target{Anchor: Anchor{id, off}, Syntax: 1})
		if err := d.Apply(Action{Kind: Delete}, false); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := d.HTML(); err != nil || got != "<p><strong>Milch</strong> `test`</p>" {
		t.Fatalf("%q %v", got, err)
	}
	d.Undo()
	if got, _ := d.HTML(); got != "<p>**Milch** `test`</p>" {
		t.Fatal(got)
	}
}

func TestDraftMismatchedClosingRunStaysLiteral(t *testing.T) {
	d := NewDraft(Parse(`<p></p>`))
	for _, r := range "*text**" {
		if err := d.Apply(Action{Kind: Type, Text: string(r)}, false); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := d.HTML(); err != nil || got != `<p>*text**</p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftDelimiterEditingAndHeadingPrefix(t *testing.T) {
	d := NewDraft(Parse(`<p><strong>text</strong></p>`))
	mark := d.Snapshot()[0].Children[0]
	d.SetTarget(Target{Anchor: Anchor{mark.ID, 0}, Syntax: 1})
	if err := d.Apply(Action{Kind: Type, Text: "A"}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p>A<strong>text</strong></p>` {
		t.Fatalf("%q %v", got, err)
	}
	d = NewDraft(Parse(`<h3>Heading</h3>`))
	id := d.Snapshot()[0].ID
	d.SetTarget(Target{Anchor: Anchor{id, 0}, Syntax: 1})
	if err := d.Apply(Action{Kind: Delete}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<h2>Heading</h2>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftVisualDelimiterDeleteLeavesContent(t *testing.T) {
	d := NewDraft(Parse(`<p><strong>text</strong></p>`))
	id := d.Snapshot()[0].Children[0].ID
	d.SetTarget(Target{Anchor: Anchor{id, 0}, Syntax: 1})
	d.Select(false, false)
	d.SetTarget(Target{Anchor: Anchor{id, 0}, Syntax: 2})
	if err := d.Apply(Action{Kind: DeleteSelection}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p>text**</p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftChunkedTypingMatchesCharacterEvents(t *testing.T) {
	for _, value := range []string{"**bold** trailing *", "`code` trailing **", "*text**", "***text*** tail", "`**npm**` now"} {
		a, b := NewDraft(Parse(`<p></p>`)), NewDraft(Parse(`<p></p>`))
		a.Apply(Action{Kind: Type, Text: value}, false)
		for _, r := range value {
			b.Apply(Action{Kind: Type, Text: string(r)}, false)
		}
		x, xe := a.HTML()
		y, ye := b.HTML()
		if xe != nil || ye != nil || x != y {
			t.Fatalf("%q chunk=%q char=%q errors=%v/%v", value, x, y, xe, ye)
		}
	}
}

func TestDraftSplitRetainsPastedLiteralOrigin(t *testing.T) {
	d := NewDraft(Parse(`<p></p>`))
	d.Apply(Action{Kind: Type, Text: "A "}, false)
	d.Apply(Action{Kind: Insert, Text: "`b"}, false)
	d.SetCursor(Anchor{d.Cursor().Node, 2})
	d.Apply(Action{Kind: Split}, false)
	id := d.Cursor().Node
	d.SetCursor(Anchor{id, 2})
	if err := d.Apply(Action{Kind: Type, Text: "`"}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != "<p>A</p><p>`b`</p>" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftEveryAgreedEscapeAndUndo(t *testing.T) {
	for _, r := range "\\*_~`#-+.[]" {
		d := NewDraft(Parse(`<p></p>`))
		if err := d.Apply(Action{Kind: Type, Text: "\\" + string(r)}, false); err != nil {
			t.Fatal(err)
		}
		if got, err := d.HTML(); err != nil || got != `<p>`+string(r)+`</p>` {
			t.Errorf("%q => %q %v", r, got, err)
		}
		if !d.Undo() {
			t.Fatal("missing atomic Undo")
		}
		if got, _ := d.HTML(); got != `<p></p>` {
			t.Fatal(got)
		}
	}
}

func TestDraftSplitTypedSpanDoesNotReconstructOldContent(t *testing.T) {
	d := NewDraft(Parse(`<p></p>`))
	d.Apply(Action{Kind: Type, Text: "*abcd*"}, false)
	text := d.Snapshot()[0].Children[0].Children[0]
	d.SetCursor(Anchor{text.ID, 2})
	d.Apply(Action{Kind: Split}, false)
	text = d.Snapshot()[1].Children[0].Children[0]
	d.SetCursor(Anchor{text.ID, 2})
	d.Apply(Action{Kind: Type, Text: "*"}, false)
	if got, err := d.HTML(); err != nil || got != `<p><em>ab</em></p><p><em>cd*</em></p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftVisualEscapeRemovalKeepsLiteralCharacters(t *testing.T) {
	d := NewDraft(Parse(`<p>*abc*</p>`))
	id := d.Cursor().Node
	for _, off := range []int{0, 4} {
		d.SetTarget(Target{Anchor: Anchor{id, off}, Syntax: 1})
		d.Select(false, false)
		if err := d.Apply(Action{Kind: DeleteSelection}, false); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := d.HTML(); err != nil || got != `<p><em>abc</em></p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftInlineCodeUnformatKeepsSourceLiteral(t *testing.T) {
	d := NewDraft(Parse(`<p></p>`))
	d.Apply(Action{Kind: Type, Text: "`*text*`"}, false)
	n := d.Snapshot()[0].Children[0].Children[0]
	d.SetCursor(Anchor{n.ID, 0})
	d.Select(false, false)
	d.SetCursor(Anchor{n.ID, 5})
	if err := d.Apply(Action{Kind: ClearMarks}, false); err != nil {
		t.Fatal(err)
	}
	d.CancelSelection()
	n = d.Snapshot()[0].Children[0]
	d.SetCursor(Anchor{n.ID, len(n.Text)})
	d.Apply(Action{Kind: Type, Text: "!"}, false)
	if got, err := d.HTML(); err != nil || got != `<p>*text*!</p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftVisualEscapeHighlightsOnlyBackslash(t *testing.T) {
	d := NewDraft(Parse(`<p>*abc*</p>`))
	d.SetTarget(Target{Anchor: d.Cursor(), Syntax: 1})
	d.Select(false, false)
	f := d.Highlight(d.View().Layout(80, true))
	selected := ""
	for _, row := range f.Rows {
		for _, span := range row.Spans {
			if span.Style&SelectedStyle != 0 {
				selected += span.Text
			}
		}
	}
	if selected != "\\" {
		t.Fatalf("selected %q", selected)
	}
}

func TestDraftInlineCodeStructuralEdgesRemainOutside(t *testing.T) {
	for _, off := range []int{0, 1} {
		d := NewDraft(Parse(`<p><strong><code>abc</code></strong></p>`))
		code := d.Snapshot()[0].Children[0].Children[0]
		d.SetCursor(Anchor{code.ID, off})
		if err := d.Apply(Action{Kind: Type, Text: "X"}, false); err != nil {
			t.Fatal(err)
		}
		want := `<p><strong>X<code>abc</code></strong></p>`
		if off == 1 {
			want = `<p><strong><code>abc</code>X</strong></p>`
		}
		if got, err := d.HTML(); err != nil || got != want {
			t.Fatalf("%q %v", got, err)
		}
	}
}

func TestDraftBackspaceAfterProtectingEscapeKeepsContent(t *testing.T) {
	d := NewDraft(Parse(`<p>A*B*</p>`))
	id := d.Cursor().Node
	d.SetCursor(Anchor{id, 1})
	if err := d.Apply(Action{Kind: Backspace}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p>A*B*</p>` {
		t.Fatalf("%q %v", got, err)
	}
	d.SetTarget(Target{Anchor: Anchor{id, 1}, Syntax: 1})
	if err := d.Apply(Action{Kind: Backspace}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p>*B*</p>` {
		t.Fatalf("before escape: %q %v", got, err)
	}
}

func TestDraftAdjacentMarkDeletionKeepsParagraphs(t *testing.T) {
	for _, tc := range []struct {
		kind   ActionKind
		offset int
		want   string
	}{
		{Backspace, 0, `<p>before</p><p>*bold**</p><p>after</p>`},
		{Delete, 4, `<p>before</p><p>**bold*</p><p>after</p>`},
	} {
		d := NewDraft(Parse(`<p>before</p><p><strong>bold</strong></p><p>after</p>`))
		n := d.Snapshot()[1].Children[0].Children[0]
		d.SetCursor(Anchor{n.ID, tc.offset})
		if err := d.Apply(Action{Kind: tc.kind}, false); err != nil {
			t.Fatal(err)
		}
		if got, err := d.HTML(); err != nil || got != tc.want {
			t.Fatalf("%q %v", got, err)
		}
		if !d.Undo() {
			t.Fatal("missing Undo")
		}
		if got, _ := d.HTML(); got != `<p>before</p><p><strong>bold</strong></p><p>after</p>` {
			t.Fatal(got)
		}
	}
}

func TestDraftReplacementKeepsPlainStyleBesideMark(t *testing.T) {
	for _, source := range []string{`<p><strong>A</strong>!</p>`, `<p>!<strong>A</strong></p>`} {
		d := NewDraft(Parse(source))
		var id ID
		for _, n := range d.Snapshot()[0].Children {
			if n.Kind == Text && n.Text == "!" {
				id = n.ID
			}
		}
		d.SetCursor(Anchor{id, 0})
		d.Select(false, false)
		if err := d.Apply(Action{Kind: ReplaceSelection, Text: "?"}, false); err != nil {
			t.Fatal(err)
		}
		want := strings.Replace(source, "!", "?", 1)
		if got, err := d.HTML(); err != nil || got != want {
			t.Fatalf("%q %v", got, err)
		}
	}
}
