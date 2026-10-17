package description

import "testing"

func TestTypedCodeMarkerPairConversionPreservesQuoteAndUndo(t *testing.T) {
	d := NewDraft(Parse(`<blockquote data-comment-id="42"><p></p></blockquote>`))
	if err := d.Apply(Action{Kind: Type, Text: "``````"}, false); err != nil {
		t.Fatal(err)
	}
	cursorText(t, d, "``````", 3)
	before := d.DraftHTML()
	if !d.CodeMarkerPairAtCursor() {
		t.Fatal("typed fence boundary not recognized")
	}
	if err := d.Apply(Action{Kind: CreateCode, Text: "marker"}, false); err != nil {
		t.Fatal(err)
	}
	want := `<blockquote data-comment-id="42"><pre><code></code></pre></blockquote>`
	html, err := d.HTML()
	if err != nil || html != want {
		t.Fatalf("Code conversion: %s (%v)", html, err)
	}
	if !d.Undo() || d.DraftHTML() != before {
		t.Fatal("Undo did not restore original markers and quote")
	}
	if !d.CodeMarkerPairAtCursor() {
		t.Fatal("Undo lost marker cursor/provenance")
	}
	if !d.Redo() {
		t.Fatal("Redo failed")
	}
	if err := d.Apply(Action{Kind: Type, Text: "print(1)"}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := d.PrepareSave(); err != nil {
		t.Fatal(err)
	}
}
