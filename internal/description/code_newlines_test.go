package description

import (
	"strings"
	"testing"
)

func TestCodeOperationsPreserveEverySourceNewline(t *testing.T) {
	for _, tc := range []struct{ name, nl string }{{"LF", "\n"}, {"CRLF", "\r\n"}, {"CR", "\r"}} {
		for _, action := range []struct {
			name      string
			kind      ActionKind
			want      string
			selection bool
			last      bool
		}{
			{"open below", OpenBelow, "a" + tc.nl + tc.nl + "b" + tc.nl, false, false},
			{"delete line", DeleteBlock, "b" + tc.nl, false, false},
			{"delete selected line", DeleteSelection, "b" + tc.nl, true, false},
			{"indent selected line", IndentCode, "    a" + tc.nl + "b" + tc.nl, true, false},
			{"delete final empty line", DeleteBlock, "a" + tc.nl + "b", false, true},
		} {
			t.Run(tc.name+"/"+action.name, func(t *testing.T) {
				source := "<pre><code>a" + tc.nl + "b" + tc.nl + "</code></pre>"
				d := NewDraft(Parse(source))
				if action.last {
					id := d.Cursor().Node
					if err := d.SetCursor(Anchor{id, len("a" + tc.nl + "b" + tc.nl)}); err != nil {
						t.Fatal(err)
					}
				}
				if action.selection {
					d.Select(true, false)
				}
				if err := d.Apply(Action{Kind: action.kind, Indent: 4}, false); err != nil {
					t.Fatal(err)
				}
				if got := d.DraftHTML(); got != "<pre><code>"+action.want+"</code></pre>" {
					t.Fatalf("got %q, want %q", got, action.want)
				}
				if _, err := d.PrepareSave(); err != nil {
					t.Fatal(err)
				}
				d.Undo()
				if d.DraftHTML() != source {
					t.Fatal("Undo changed source newline bytes")
				}
			})
		}
		t.Run(tc.name+"/boundaries and copy", func(t *testing.T) {
			d := NewDraft(Parse("<pre><code>a" + tc.nl + "b" + tc.nl + "</code></pre>"))
			if d.CopyLine() != "a" {
				t.Fatalf("first source line = %q", d.CopyLine())
			}
			if d.LineBoundary(true).Offset != 1 {
				t.Fatalf("line end = %+v", d.LineBoundary(true))
			}
			v := d.View()
			v.Move("j", 1, 80, 10)
			if v.Cursor().Anchor.Offset != 1+len(tc.nl) {
				t.Fatalf("down motion crossed source line: %+v", v.Cursor())
			}
			if strings.Contains(frameText(v.Layout(80, true)), `\r`) {
				t.Fatal("visible CR control escape")
			}
		})
	}
}

func TestCodeMultilineIndentPreservesSourceNewlines(t *testing.T) {
	for _, nl := range []string{"\n", "\r\n", "\r"} {
		source := "<pre><code>a" + nl + "b" + nl + "</code></pre>"
		d := NewDraft(Parse(source))
		id := d.Cursor().Node
		d.Select(true, false)
		if err := d.SetCursor(Anchor{id, 1 + len(nl)}); err != nil {
			t.Fatal(err)
		}
		if err := d.Apply(Action{Kind: IndentCode, Indent: 4}, false); err != nil {
			t.Fatal(err)
		}
		if d.DraftHTML() != "<pre><code>    a"+nl+"    b"+nl+"</code></pre>" {
			t.Fatalf("multiline indent: %q", d.DraftHTML())
		}
		d.Undo()
		if d.DraftHTML() != source {
			t.Fatal("indent Undo changed bytes")
		}
	}
}

func TestCodeCRLFAcrossPreservedWrappersIsOneLineBreak(t *testing.T) {
	for _, source := range []string{"<pre><code>a\r</code><code>\nb</code></pre>", "<pre><code>a\r</code><code>\r\nb</code></pre>"} {
		d := NewDraft(Parse(source))
		v := d.View()
		wantRows := 4
		if strings.Contains(source, "\r\r\n") || strings.Contains(source, "</code><code>\r\n") {
			wantRows = 5
		}
		if got := len(v.Layout(80, true).Rows); got != wantRows {
			t.Fatalf("source=%q: %d rows, want %d", source, got, wantRows)
		}
		if d.DraftHTML() != source {
			t.Fatal("render changed wrappers/source")
		}
	}
}
