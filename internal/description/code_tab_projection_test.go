package description

import (
	"strings"
	"testing"
)

func TestCodeTabsHaveVisibleMarkersAtSourceTabstops(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"\t", "│ >~~~"},
		{"a\tb", "│ a>~~b"},
		{"界\tb", "│ 界>~b"},
		{"\t\t", "│ >~~~>~~~"},
		{"a  b", "│ a  b"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			source := "<pre><code>" + tc.source + "</code></pre>"
			doc := Parse(source)
			frame := NewView(doc).Layout(80, false)
			if got := frameText(frame); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			if doc.SourceHTML() != source {
				t.Fatal("projection changed source")
			}
		})
	}
}

func TestCodeTabMarkersKeepSourceNavigationCopyAndRetab(t *testing.T) {
	d := NewDraft(Parse("<pre><code>a\tb</code></pre>"))
	v := d.View(ViewOptions{Tabstop: 4})
	if got := frameText(v.Layout(40, false)); got != "│ a>~~b" {
		t.Fatal(got)
	}
	v.Move("l", 1, 40, 8)
	if v.Cursor().Anchor.Offset != 1 {
		t.Fatal("Tab cursor target lost")
	}
	v.Move("l", 1, 40, 8)
	if v.Cursor().Anchor.Offset != 2 {
		t.Fatal("Tab marker introduced extra cursor targets")
	}
	if err := d.SetTarget(v.Cursor()); err != nil {
		t.Fatal(err)
	}
	if got := d.Copy(); got != "a\tb" {
		t.Fatalf("copied display markers: %q", got)
	}
	if err := d.Apply(Action{Kind: Retab, Tabstop: 4}, false); err != nil {
		t.Fatal(err)
	}
	if got := d.DraftHTML(); got != "<pre><code>a   b</code></pre>" {
		t.Fatal(got)
	}
	if got := frameText(d.View(ViewOptions{Tabstop: 4}).Layout(40, false)); got != "│ a   b" {
		t.Fatalf("retab kept Tab markers: %q", got)
	}
	if !d.Undo() || d.DraftHTML() != "<pre><code>a\tb</code></pre>" {
		t.Fatal("Undo lost source Tab")
	}
	if got := frameText(d.View(ViewOptions{Tabstop: 8}).Layout(40, false)); got != "│ a>~~~~~~b" {
		t.Fatalf("marker ignored configured tabstop: %q", got)
	}
}

func TestDescriptionTabsHaveVisibleMarkersOutsideCode(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"\t", ">~~~"}, {"x\tcc", "x>~~cc"}, {"界\tcc", "界>~cc"}, {"x   cc", "x   cc"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			d := NewTextDraft(tc.source)
			v := d.View(ViewOptions{Tabstop: 4})
			for _, focused := range []bool{false, true} {
				frame := v.Layout(80, focused)
				if got := frameText(frame); got != tc.want {
					t.Fatalf("focused=%v got %q want %q", focused, got, tc.want)
				}
				for _, row := range frame.Rows {
					for _, span := range row.Spans {
						if strings.Contains(span.Text, ">") && span.Style&TabStyle == 0 {
							t.Fatal("Tab marker lacks muted style")
						}
					}
				}
			}
			if got := d.CopyLine(); got != tc.source {
				t.Fatalf("CopyLine changed Tab: %q", got)
			}
			if got := d.DraftHTML(); got != tc.source {
				t.Fatalf("layout changed baseline: %q", got)
			}
		})
	}
	d := NewDraft(Parse(`<p>xcc</p>`))
	d.SetCursor(d.After(d.Cursor()))
	if err := d.Apply(Action{Kind: Insert, Text: "\t"}, false); err != nil {
		t.Fatal(err)
	}
	if got := frameText(d.View(ViewOptions{Tabstop: 4}).Layout(80, false)); got != "x>~~cc" {
		t.Fatal(got)
	}
	if got := d.DraftHTML(); got != "<p>x\tcc</p>" {
		t.Fatalf("marker leaked to HTML: %q", got)
	}
}
