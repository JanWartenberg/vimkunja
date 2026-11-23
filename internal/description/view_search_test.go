package description

import (
	"strings"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/searchpattern"
)

func TestSearchReadableTextAcrossMarksWithoutHTMLOrHiddenURLs(t *testing.T) {
	source := `<p>Be<strong>richt</strong> Bericht</p><p><a href="https://hidden.example">sichtbar</a></p><pre><code>first` + "\n" + `Bericht</code></pre>`
	d := NewDraft(Parse(source))
	v := d.View()
	p, _ := searchpattern.Compile("bericht")
	matches := v.Search(p)
	if len(matches) != 3 {
		t.Fatalf("matches: %#v", matches)
	}
	for _, match := range matches {
		if err := d.SetTarget(match.Target); err != nil {
			t.Fatal(err)
		}
		if err := v.SetAnchor(match.Target.Anchor); err != nil {
			t.Fatal(err)
		}
	}
	for _, text := range []string{"strong", "hidden.example", "href"} {
		p, _ := searchpattern.Compile(text)
		if matches := v.Search(p); len(matches) != 0 {
			t.Fatalf("searched hidden content %q: %#v", text, matches)
		}
	}
	v.SearchHighlight(p, -1, -1)
	if v.SourceHTML() != source || d.DraftHTML() != source {
		t.Fatal("search modified source")
	}
}

func TestSearchLogicalAnchorsAndHighlightSurviveWrapping(t *testing.T) {
	v := NewTextDraft("start Bericht end\nBericht").View()
	p, _ := searchpattern.Compile(`^Bericht`)
	matches := v.Search(p)
	if len(matches) != 1 || matches[0].Line != 1 {
		t.Fatal(matches)
	}
	v.SearchHighlight(p, matches[0].Line, matches[0].Offset)
	for _, width := range []int{4, 12, 80} {
		frame := v.Layout(width, false)
		var highlighted strings.Builder
		for _, row := range frame.Rows {
			for _, span := range row.Spans {
				if span.Style&CurrentSearchStyle != 0 {
					highlighted.WriteString(span.Text)
				}
			}
		}
		if highlighted.String() != "Bericht" {
			t.Fatalf("width %d: %q", width, highlighted.String())
		}
	}
}

func TestSearchHighlightsIndividualOpaqueOccurrences(t *testing.T) {
	v := NewDraft(Parse(`<table><tr><td>needle needle</td></tr></table>`)).View()
	p, _ := searchpattern.Compile("needle")
	matches := v.Search(p)
	if len(matches) != 2 {
		t.Fatal(matches)
	}
	v.SearchHighlight(p, matches[1].Line, matches[1].Offset)
	var current strings.Builder
	for _, row := range v.Layout(80, false).Rows {
		for _, span := range row.Spans {
			if span.Style&CurrentSearchStyle != 0 {
				current.WriteString(span.Text)
			}
		}
	}
	if current.String() != "needle" {
		t.Fatalf("current paint leaked across an atomic target: %q", current.String())
	}
}
