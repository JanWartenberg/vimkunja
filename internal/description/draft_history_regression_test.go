package description

import "testing"

func TestDraftDeleteLinkEndBeforeTrailingSpaceRemainsSavable(t *testing.T) {
	d := NewDraft(Parse(`<p><a href="https://example.org">A Z</a> tail</p>`))
	cursorText(t, d, " tail", 1)
	if err := d.Apply(Action{Kind: Split}, false); err != nil {
		t.Fatal(err)
	}
	cursorText(t, d, "A Z", 2)
	if err := d.Apply(Action{Kind: Delete}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<p><a href="https://example.org" target="_blank">A</a></p><p>tail</p>`)
	if _, err := d.PrepareSave(); err != nil {
		t.Fatal(err)
	}
	if !d.Undo() {
		t.Fatal("missing Undo")
	}
	expectDraftHTML(t, d, `<p><a href="https://example.org" target="_blank">A Z</a></p><p>tail</p>`)
}

func TestDraftTypedNewlineSplitsWithoutPanic(t *testing.T) {
	for _, test := range []struct{ source, input, want string }{
		{`<p>A</p>`, "\n", `<p></p><p>A</p>`},
		{`<p>A</p>`, "x\ny", `<p>x</p><p>yA</p>`},
		{`<p>A</p>`, "\n\n", `<p></p><p></p><p>A</p>`},
		{`<p>A</p>`, "**bold**\ntail", `<p><strong>bold</strong></p><p>tailA</p>`},
		{`<pre><code>A</code></pre>`, "\n", "<pre><code>\nA</code></pre>"},
	} {
		t.Run(test.input+test.source, func(t *testing.T) {
			d := NewDraft(Parse(test.source))
			if err := d.Apply(Action{Kind: Type, Text: test.input}, false); err != nil {
				t.Fatal(err)
			}
			expectDraftHTML(t, d, test.want)
			if _, err := d.PrepareSave(); err != nil {
				t.Fatal(err)
			}
			if !d.Undo() || d.DraftHTML() != test.source {
				t.Fatal("typed chunk Undo did not restore source")
			}
		})
	}
}

func TestNormalizeTrailingHardBreakWhitespaceIsPreserved(t *testing.T) {
	for _, source := range []string{`<p>A<br> </p>`, `<p>A<br>  </p>`, "<p>A<br>\t </p>"} {
		t.Run(source, func(t *testing.T) {
			d := NewDraft(Parse(source))
			if got, err := d.HTML(); err != nil || got != source {
				t.Fatalf("HTML=%q err=%v; want %q", got, err, source)
			}
		})
	}
}

func TestTypedClosingDelimiterAroundCursorKeepsFollowingUnicodeBoundary(t *testing.T) {
	d := NewDraft(Parse(`<p>é👩‍💻x</p>`))
	if err := d.Apply(Action{Kind: Type, Text: "**"}, false); err != nil {
		t.Fatal(err)
	}
	cursorText(t, d, "**é👩‍💻x", 1)
	if err := d.Apply(Action{Kind: Type, Text: "**x**"}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<p><strong><em>x</em></strong>é👩‍💻x</p>`)
	if _, err := d.live().ResolveAnchor(d.Cursor()); err != nil {
		t.Fatalf("typed delimiter left invalid cursor: %+v: %v", d.Cursor(), err)
	}
	// The cursor belongs directly after typed x, before the original text.
	if err := d.Apply(Action{Kind: Insert, Text: "Y"}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<p><strong><em>x</em></strong>Yé👩‍💻x</p>`)
}

func TestFormattingPreservesCursorAfterEmptiedQuote(t *testing.T) {
	d := NewDraft(Parse(`<blockquote>中</blockquote><p>é👩‍💻</p>`))
	cursorText(t, d, "中", 0)
	if err := d.Apply(Action{Kind: Delete}, false); err != nil {
		t.Fatal(err)
	}
	cursorText(t, d, "é👩‍💻", 0)
	d.Select(false, false)
	cursorText(t, d, "é👩‍💻", len("é"))
	before := d.Cursor()
	if err := d.Apply(Action{Kind: ToggleMark, Text: "strong"}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<blockquote></blockquote><p><strong>é👩‍💻</strong></p>`)
	if d.Cursor() != before || d.Selection().End != before {
		t.Fatalf("formatting moved active selection/cursor: cursor=%+v end=%+v want=%+v", d.Cursor(), d.Selection().End, before)
	}
	d.CancelSelection()
	if err := d.Apply(Action{Kind: Insert, Text: "Y"}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<blockquote></blockquote><p><strong>éY👩‍💻</strong></p>`)
}
