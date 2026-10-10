package description

import (
	"html"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCodeTabRetabAndPartialOutdent(t *testing.T) {
	d := NewDraft(Parse("<pre><code>\tx\r\n\r\n  y\n\tz</code></pre>"))
	if err := d.Apply(Action{Kind: OutdentCode, Indent: 2, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre><code>  x\r\n\r\n  y\n\tz</code></pre>" {
		t.Fatal(got)
	}
	d.Undo()
	if err := d.Apply(Action{Kind: Retab, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre><code>    x\r\n\r\n  y\n    z</code></pre>" {
		t.Fatal(got)
	}
	d.Undo()
	if err := d.Apply(Action{Kind: InsertTab, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre><code>    \tx\r\n\r\n  y\n\tz</code></pre>" {
		t.Fatal(got)
	}
}

func TestCodeCreationAndNestedExit(t *testing.T) {
	d := NewDraft(Parse("<blockquote data-comment-id=\"42\"><p>````</p></blockquote>"))
	if err := d.Apply(Action{Kind: CreateCode, Text: "marker"}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<blockquote data-comment-id=\"42\"><pre><code></code></pre></blockquote>" {
		t.Fatal(got)
	}
	if err := d.Apply(Action{Kind: CodeAfter}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<blockquote data-comment-id=\"42\"><pre><code></code></pre><p></p></blockquote>" {
		t.Fatal(got)
	}
	d.Undo()
	d.Undo()
	if got, _ := d.HTML(); got != "<blockquote data-comment-id=\"42\"><p>````</p></blockquote>" {
		t.Fatal(got)
	}
}

func TestCodeCRLFOpenAndDeleteEmptyFinalLine(t *testing.T) {
	d := NewDraft(Parse("<pre><code>a\r\nb\r\n</code></pre>"))
	if err := d.Apply(Action{Kind: OpenBelow}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre><code>a\r\n\r\nb\r\n</code></pre>" {
		t.Fatalf("open below %q", got)
	}
	d.Undo()
	n := d.Snapshot()[0].Children[0].Children[0]
	d.SetCursor(Anchor{n.ID, len(n.Text)})
	if err := d.Apply(Action{Kind: DeleteBlock}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre><code>a\r\nb</code></pre>" {
		t.Fatalf("final blank dd %q", got)
	}
}

func TestRetabKeepsCursorAndSelectionOnSourceCharacters(t *testing.T) {
	d := NewDraft(Parse("<pre><code>a\tb\tc\nx\ty</code></pre>"))
	n := d.Snapshot()[0].Children[0].Children[0]
	d.SetCursor(Anchor{n.ID, 2})
	d.Select(true, false)
	if err := d.Apply(Action{Kind: Retab, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if got := d.Cursor(); got.Offset != 4 {
		t.Fatalf("cursor %+v", got)
	}
	if d.Selection().Start.Offset != 4 || d.Selection().End.Offset != 4 {
		t.Fatalf("selection %+v", d.Selection())
	}
	d.Undo()
	d.CancelSelection()
	d.SetCursor(Anchor{n.ID, 4})
	if err := d.Apply(Action{Kind: RetabAll, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if d.Cursor().Node != n.ID || d.Cursor().Offset != 8 {
		t.Fatalf("retaball cursor %+v", d.Cursor())
	}
}

func TestCodeOperationsAcrossPreservedWrappers(t *testing.T) {
	d := NewDraft(Parse("<pre><code>A</code><code class=\"language-go\">B\tC</code></pre>"))
	if err := d.Apply(Action{Kind: Retab, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre><code>A</code><code class=\"language-go\">B  C</code></pre>" {
		t.Fatal(got)
	}
	d.Undo()
	second := d.Snapshot()[0].Children[1].Children[0]
	d.SetCursor(Anchor{second.ID, 0})
	if err := d.Apply(Action{Kind: InsertTab, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre><code>A</code><code class=\"language-go\">   B\tC</code></pre>" {
		t.Fatal(got)
	}
	d = NewDraft(Parse("<pre><code></code><code>A</code></pre>"))
	if err := d.Apply(Action{Kind: DeleteBlock}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre><code></code><code></code></pre>" {
		t.Fatal(got)
	}
	d = NewDraft(Parse("<pre><code>A\n</code><code>B\tC</code></pre>"))
	nodes := d.Snapshot()[0].Children
	d.SetCursor(Anchor{nodes[0].Children[0].ID, 0})
	d.Select(true, false)
	d.SetCursor(Anchor{nodes[1].Children[0].ID, 1})
	if err := d.Apply(Action{Kind: Retab, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre><code>A\n</code><code>B   C</code></pre>" {
		t.Fatal(got)
	}
}

func FuzzCodeRetabSourcePreservation(f *testing.F) {
	for _, source := range []string{"\tA\r\nB\tC\n", "界\tX", "\n\r\n", "a<b\t\"c\"", "👨‍👩‍👧‍👦\tX"} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 4096 || !utf8.ValidString(source) {
			t.Skip()
		}
		d := NewDraft(Parse("<pre><code>" + html.EscapeString(source) + "</code></pre>"))
		if d.Classification() != Native {
			t.Skip()
		}
		before, err := d.HTML()
		if err != nil {
			t.Fatal(err)
		}
		if err := d.Apply(Action{Kind: Retab, Tabstop: 4}, false); err != nil {
			t.Fatal(err)
		}
		snapshot := d.Snapshot()
		got := ""
		if len(snapshot[0].Children[0].Children) > 0 {
			got = snapshot[0].Children[0].Children[0].Text
		}
		strip := func(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "\t", ""), " ", "") }
		if strings.Contains(got, "\t") || strip(got) != strip(source) {
			t.Fatalf("retab changed source characters: %q -> %q", source, got)
		}
		if _, err := d.PrepareSave(); err != nil {
			t.Fatal(err)
		}
		if d.Undo() {
			after, err := d.HTML()
			if err != nil || after != before {
				t.Fatalf("Undo: %q %v", after, err)
			}
		}
	})
}

func TestCodeOpaqueGapsKeepNativeOffsetsAndProtection(t *testing.T) {
	const raw = `<code onclick="alert(1)">A</code>`
	d := NewDraft(Parse("<pre>" + raw + "<code>B\tC</code></pre>"))
	n := d.Snapshot()[0].Children[1].Children[0]
	d.SetCursor(Anchor{n.ID, 1})
	if err := d.Apply(Action{Kind: InsertTab, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre>"+raw+"<code>B   \tC</code></pre>" {
		t.Fatal(got)
	}
	d.Undo()
	if err := d.Apply(Action{Kind: DeleteBlock}, false); err == nil {
		t.Fatal("line deletion crossed atom")
	}
	if got, _ := d.HTML(); got != "<pre>"+raw+"<code>B\tC</code></pre>" {
		t.Fatal(got)
	}
	for _, kind := range []ActionKind{Retab, RetabAll} {
		if err := d.Apply(Action{Kind: kind, Tabstop: 4}, false); err != nil {
			t.Fatal(err)
		}
		if got, _ := d.HTML(); got != "<pre>"+raw+"<code>B   C</code></pre>" {
			t.Fatal(got)
		}
		d.Undo()
	}
	d.Select(true, false)
	if err := d.Apply(Action{Kind: IndentCode, Indent: 2, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre>  "+raw+"<code>B\tC</code></pre>" {
		t.Fatal(got)
	}
	d.Undo()
	d.CancelSelection()
	d.Select(true, true)
	if err := d.Apply(Action{Kind: DeleteSelection}, false); err != ErrConfirmation {
		t.Fatalf("whole block confirmation %v", err)
	}
	if err := d.Apply(Action{Kind: DeleteSelection}, true); err != nil {
		t.Fatal(err)
	}
}

func TestCodeLineDeletionCannotConfirmThroughOpaque(t *testing.T) {
	const source = `<pre><code onclick="alert(1)">A</code><code>B</code></pre>`
	for _, kind := range []ActionKind{DeleteSelection, ReplaceSelection} {
		d := NewDraft(Parse(source))
		n := d.Snapshot()[0].Children[1].Children[0]
		d.SetCursor(Anchor{n.ID, 0})
		d.Select(true, false)
		for _, confirmed := range []bool{false, true} {
			if err := d.Apply(Action{Kind: kind, Text: "X"}, confirmed); err == nil || err == ErrConfirmation {
				t.Fatalf("line operation should reject locally, got %v", err)
			}
			if got, _ := d.HTML(); got != source {
				t.Fatal(got)
			}
			if d.Selection() == nil || d.Undo() {
				t.Fatal("rejected action changed selection/history")
			}
		}
	}
}
