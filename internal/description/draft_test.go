package description

import (
	"strings"
	"testing"
)

func TestDraftParagraphSplitAndHistory(t *testing.T) {
	d := NewDraft(Parse("<p>AlphaBeta</p>"))
	text := d.Snapshot()[0].Children[0]
	if err := d.SetCursor(Anchor{text.ID, 5}); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(Action{Kind: Split}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != "<p>Alpha</p><p>Beta</p>" {
		t.Fatalf("split: %q %v", got, err)
	}
	d.Undo()
	if got, _ := d.HTML(); got != "<p>AlphaBeta</p>" {
		t.Fatal(got)
	}
	d.Redo()
	if err := d.Apply(Action{Kind: Insert, Text: "<x>"}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<p>Alpha</p><p>&lt;x&gt;Beta</p>" {
		t.Fatal(got)
	}
}

func TestDraftSelectionMarksAndOpaqueConfirmation(t *testing.T) {
	const source = `<p><strong><strong><strong>Alpha</strong></strong></strong><widget x='1'>secret</widget>Beta</p>`
	d := NewDraft(Parse(source))
	n := d.Snapshot()[0]
	a := n.Children[0].Children[0].Children[0].Children[0]
	b := n.Children[2]
	d.SetCursor(Anchor{a.ID, 0})
	d.Select(false, false)
	d.SetCursor(Anchor{b.ID, 3})
	if err := d.Apply(Action{Kind: ToggleMark, Text: "strong"}, false); err != nil {
		t.Fatal(err)
	}
	got, err := d.HTML()
	if err != nil {
		t.Fatal(err)
	}
	// Mixed toggle adds bold to Beta; toggling again removes ALL bold layers.
	if err := d.Apply(Action{Kind: ToggleMark, Text: "strong"}, false); err != nil {
		t.Fatal(err)
	}
	got, err = d.HTML()
	if err != nil || got != `<p>Alpha<widget x='1'>secret</widget>Beta</p>` {
		t.Fatalf("%q %v", got, err)
	}
	before := got
	if err := d.Apply(Action{Kind: DeleteSelection}, false); err != ErrConfirmation {
		t.Fatalf("want confirm: %v", err)
	}
	if got, _ := d.HTML(); got != before {
		t.Fatal("cancellation mutated draft")
	}
	if err := d.Apply(Action{Kind: DeleteSelection}, true); err != nil {
		t.Fatal(err)
	}
	d.Undo()
	if got, _ := d.HTML(); got != before {
		t.Fatal(got)
	}
	d.Redo()
	if got, _ := d.HTML(); got != `<p></p>` {
		t.Fatal(got)
	}
}

func TestDraftBreaksGraphemesAndCodeLineSelection(t *testing.T) {
	d := NewDraft(Parse(`<p>A👨‍👩‍👧‍👦B</p>`))
	n := d.Snapshot()[0].Children[0]
	d.SetCursor(Anchor{n.ID, 1})
	if err := d.Apply(Action{Kind: Delete}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != `<p>AB</p>` {
		t.Fatal(got)
	}
	if err := d.Apply(Action{Kind: HardBreak}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != `<p>A<br>B</p>` {
		t.Fatal(got)
	}
	if err := d.Apply(Action{Kind: Backspace}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != `<p>AB</p>` {
		t.Fatal(got)
	}
	d = NewDraft(Parse("<pre>one\ntwo\nthree</pre><p>outside</p>"))
	n = d.Snapshot()[0].Children[0]
	d.SetCursor(Anchor{n.ID, 4})
	d.Select(true, false)
	if err := d.Apply(Action{Kind: DeleteSelection}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre>one\nthree</pre><p>outside</p>" {
		t.Fatal(got)
	}
}

func TestDraftSaveIdentityAdoptionAndRecovery(t *testing.T) {
	d := NewDraft(Parse(`<ul data-type="taskList"><li data-type="taskItem" data-checked="true">A</li></ul>`))
	c, err := d.PrepareSave()
	if err != nil {
		t.Fatal(err)
	}
	if attributeValue(d.Snapshot()[0].Children[0].Attributes, "data-task-id") != "" {
		t.Fatal("premature adoption")
	}
	if err := d.AdoptSave(c); err != nil {
		t.Fatal(err)
	}
	id := attributeValue(d.Snapshot()[0].Children[0].Attributes, "data-task-id")
	if !uuidV4.MatchString(id) {
		t.Fatal(id)
	}
	n := d.Snapshot()[0].Children[0].Children[0]
	d.SetCursor(Anchor{n.ID, 1})
	d.Apply(Action{Kind: Insert, Text: "B"}, false)
	next, err := d.PrepareSave()
	if err != nil || next.Baseline() != c.HTML() {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(next.HTML(), id) {
		t.Fatal("regenerated UUID")
	}
	d = NewDraft(Parse(`<p>safe<script>hidden()</script><strong>broken`))
	if err := d.Apply(Action{Kind: RecoverText}, false); err != ErrConfirmation {
		t.Fatal(err)
	}
	if err := d.Apply(Action{Kind: RecoverText}, true); err != nil {
		t.Fatal(err)
	}
	got, err := d.HTML()
	if err != nil || strings.Contains(got, "hidden") || strings.Contains(got, "Active content") {
		t.Fatalf("%q %v", got, err)
	}
	d.Undo()
	if d.Classification() != Blocked {
		t.Fatal("Undo must restore Blocked source")
	}
}

func TestDraftUniformAndPartialFormatGoldens(t *testing.T) {
	d := NewDraft(Parse(`<p><strong>bold</strong> plain</p>`))
	ns := d.Snapshot()[0].Children
	d.SetCursor(Anchor{ns[0].Children[0].ID, 0})
	d.Select(false, false)
	d.SetCursor(Anchor{ns[1].ID, 5})
	d.Apply(Action{Kind: ToggleMark, Text: "strong"}, false)
	if got, _ := d.HTML(); got != `<p><strong>bold plain</strong></p>` {
		t.Fatalf("mixed on: %s", got)
	}
	d.Apply(Action{Kind: ToggleMark, Text: "strong"}, false)
	if got, _ := d.HTML(); got != `<p>bold plain</p>` {
		t.Fatal(got)
	}
	d = NewDraft(Parse(`<p><strong>Alpha</strong> Beta</p>`))
	n := d.Snapshot()[0].Children[0].Children[0]
	d.SetCursor(Anchor{n.ID, 1})
	d.Select(false, false)
	d.SetCursor(Anchor{n.ID, 3})
	d.Apply(Action{Kind: ToggleMark, Text: "strong"}, false)
	if got, err := d.HTML(); err != nil || got != `<p><strong>A</strong>lph<strong>a</strong> Beta</p>` {
		t.Fatalf("%q %v", got, err)
	}
}
func TestDraftPasteMergeAndRawUndo(t *testing.T) {
	d := NewDraft(Parse(`<P title='yes'>Alpha</P><p>Beta</p>`))
	n := d.Snapshot()[0].Children[0]
	d.SetCursor(Anchor{n.ID, 5})
	d.Apply(Action{Kind: Insert, Text: "\n# literal\n<x>"}, false)
	if got, err := d.HTML(); err != nil || got != `<p title="yes">Alpha</p><p># literal</p><p>&lt;x&gt;</p><p>Beta</p>` {
		t.Fatalf("%q %v", got, err)
	}
	d.Undo()
	if d.DraftHTML() != `<P title='yes'>Alpha</P><p>Beta</p>` {
		t.Fatal(d.DraftHTML())
	}
	d = NewDraft(Parse(`<p>Alpha</p><p>Beta</p>`))
	n = d.Snapshot()[1].Children[0]
	d.SetCursor(Anchor{n.ID, 0})
	if err := d.Apply(Action{Kind: Backspace}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != `<p>AlphaBeta</p>` {
		t.Fatal(got)
	}
	d.Apply(Action{Kind: Insert, Text: "!"}, false)
	if got, _ := d.HTML(); got != `<p>Alpha!Beta</p>` {
		t.Fatal(got)
	}
}
func TestDraftDeleteAcrossHardBreakAndFormatting(t *testing.T) {
	d := NewDraft(Parse(`<p>A<br><strong>B</strong>C</p>`))
	n := d.Snapshot()[0].Children[2].Children[0]
	d.SetCursor(Anchor{n.ID, 0})
	if err := d.Apply(Action{Kind: Backspace}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != `<p>A<strong>B</strong>C</p>` {
		t.Fatal(got)
	}
}
func TestDraftOpaqueNeighborsAndEmptyCodeActions(t *testing.T) {
	d := NewDraft(Parse(`<p>A<widget x='1'>B</widget>C</p>`))
	n := d.Snapshot()[0].Children[1]
	d.SetCursor(Anchor{n.ID, 0})
	if err := d.Apply(Action{Kind: Insert, Text: "before"}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != `<p>Abefore<widget x='1'>B</widget>C</p>` {
		t.Fatal(got)
	}
	d = NewDraft(Parse(`<pre><code></code></pre><p>after</p>`))
	if err := d.Apply(Action{Kind: OpenBelow}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != "<pre><code>\n</code></pre><p>after</p>" {
		t.Fatal(got)
	}
}
func TestDraftPartialNestedMarksSkipOpaqueAndSemanticWord(t *testing.T) {
	d := NewDraft(Parse(`<p><strong><strong><strong>Alpha</strong></strong></strong></p>`))
	n := d.Snapshot()[0].Children[0].Children[0].Children[0].Children[0]
	d.SetCursor(Anchor{n.ID, 1})
	d.Select(false, false)
	d.SetCursor(Anchor{n.ID, 3})
	if err := d.Apply(Action{Kind: ToggleMark, Text: "strong"}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.HTML(); got != `<p><strong><strong><strong>A</strong></strong></strong>lph<strong><strong><strong>a</strong></strong></strong></p>` {
		t.Fatal(got)
	}
	d = NewDraft(Parse(`<p><strong>A<widget>B</widget>C</strong></p>`))
	ns := d.Snapshot()[0].Children[0].Children
	d.SetCursor(Anchor{ns[0].ID, 0})
	d.Select(false, false)
	d.SetCursor(Anchor{ns[2].ID, 0})
	if err := d.Apply(Action{Kind: ToggleMark, Text: "strong"}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p>A<strong><widget>B</widget></strong>C</p>` {
		t.Fatalf("%q %v", got, err)
	}
	for _, index := range []int{0, 1, 2} {
		d = NewDraft(Parse(`<p>Al<strong>ph</strong>a Beta</p>`))
		ns = d.Snapshot()[0].Children
		n = ns[index]
		if n.Kind == Element {
			n = n.Children[0]
		}
		d.SetCursor(Anchor{n.ID, 0})
		d.SelectWord(false)
		if d.Copy() != "Alpha" {
			t.Fatal(d.Copy())
		}
		d.SelectWord(true)
		if d.Copy() != "Alpha " {
			t.Fatal(d.Copy())
		}
	}
}
func TestDraftTrailingBlankCodeDeleteAndOpaquePaint(t *testing.T) {
	d := NewDraft(Parse("<pre>one\n</pre>"))
	n := d.Snapshot()[0].Children[0]
	d.SetCursor(Anchor{n.ID, 4})
	d.Apply(Action{Kind: DeleteBlock}, false)
	if got, _ := d.HTML(); got != `<pre>one</pre>` {
		t.Fatal(got)
	}
	d = NewDraft(Parse(`<p><widget>safe words</widget></p>`))
	d.Select(false, false)
	f := d.Highlight(d.View().Layout(8, true))
	painted := false
	for _, row := range f.Rows {
		for _, s := range row.Spans {
			painted = painted || s.Style&SelectedStyle != 0
		}
	}
	if !painted {
		t.Fatal("selected atom not painted")
	}
}
func TestDraftPartialLinkClearDoesNotDuplicateUniqueMetadata(t *testing.T) {
	const source = `<p><a href="https://example.com" id="unique">ABCDE</a></p>`
	d := NewDraft(Parse(source))
	n := d.Snapshot()[0].Children[0].Children[0]
	d.SetCursor(Anchor{n.ID, 1})
	d.Select(false, false)
	d.SetCursor(Anchor{n.ID, 3})
	if err := d.Apply(Action{Kind: ClearMarks}, false); err == nil {
		t.Fatal("unsafe unique-ID split accepted")
	}
	if d.DraftHTML() != source {
		t.Fatal("rejected split changed draft")
	}
}
func TestDraftVisualTrailingBlankCodeLineDelete(t *testing.T) {
	d := NewDraft(Parse("<pre>one\n</pre>"))
	n := d.Snapshot()[0].Children[0]
	d.SetCursor(Anchor{n.ID, 4})
	d.Select(true, false)
	d.Apply(Action{Kind: DeleteSelection}, false)
	if got, _ := d.HTML(); got != `<pre>one</pre>` {
		t.Fatal(got)
	}
}
func TestDraftWholeOpaqueExtentPaint(t *testing.T) {
	d := NewDraft(Parse(`<p><widget>safe words with more words</widget></p>`))
	d.Select(false, false)
	f := d.Highlight(d.View().Layout(8, true))
	for _, row := range f.Rows {
		for _, s := range row.Spans {
			if s.Style&OpaqueStyle != 0 && s.Style&SelectedStyle == 0 {
				t.Fatalf("unpainted atom piece %q", s.Text)
			}
		}
	}
}
func TestDraftDeletionHonorsAtomicBoundarySide(t *testing.T) {
	for _, tc := range []struct {
		source string
		offset int
		kind   ActionKind
		want   string
	}{{`<p>A<widget>X</widget>B</p>`, 0, Backspace, `<p><widget>X</widget>B</p>`}, {`<p>A<widget>X</widget>B</p>`, 1, Delete, `<p>A<widget>X</widget></p>`}, {`<p>A<br>B</p>`, 0, Backspace, `<p><br>B</p>`}, {`<p>A<br>B</p>`, 1, Delete, `<p>A<br></p>`}} {
		d := NewDraft(Parse(tc.source))
		n := d.Snapshot()[0].Children[1]
		d.SetCursor(Anchor{n.ID, tc.offset})
		if err := d.Apply(Action{Kind: tc.kind}, false); err != nil {
			t.Fatal(err)
		}
		if got, _ := d.HTML(); got != tc.want {
			t.Fatalf("%q", got)
		}
	}
}
func TestDraftEmptyRecoveryChangesDirtyToken(t *testing.T) {
	for _, kind := range []ActionKind{ClearDescription, RecoverText} {
		const source = `<script>hidden()</script><strong>`
		d := NewDraft(Parse(source))
		if err := d.Apply(Action{Kind: kind}, true); err != nil {
			t.Fatal(err)
		}
		if d.DraftHTML() != "" {
			t.Fatal(d.DraftHTML())
		}
		d.Undo()
		if d.DraftHTML() != source {
			t.Fatal("undo lost source")
		}
		d.Redo()
		if d.DraftHTML() != "" {
			t.Fatal("redo lost recovery")
		}
	}
}
func TestDraftFormattingSkipsMultipleOpaqueAtoms(t *testing.T) {
	d := NewDraft(Parse(`<p><strong><img alt='one'>A<img alt='two'></strong></p>`))
	n := d.Snapshot()[0].Children[0].Children[1]
	d.SetCursor(Anchor{n.ID, 0})
	d.Select(false, false)
	if err := d.Apply(Action{Kind: ToggleMark, Text: "strong"}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p><strong><img alt='one'></strong>A<strong><img alt='two'></strong></p>` {
		t.Fatalf("%q %v", got, err)
	}
}
func TestDraftRemovingBoundaryKeepsInsertionPosition(t *testing.T) {
	for _, source := range []string{`<p>A<br>B</p>`, `<p>A<img alt='map'>B</p>`} {
		for _, kind := range []ActionKind{Backspace, Delete} {
			d := NewDraft(Parse(source))
			n := d.Snapshot()[0].Children[1]
			off := 0
			if kind == Backspace {
				off = 1
			}
			d.SetCursor(Anchor{n.ID, off})
			if err := d.Apply(Action{Kind: kind}, true); err != nil {
				t.Fatal(err)
			}
			d.Apply(Action{Kind: Insert, Text: "!"}, false)
			if got, _ := d.HTML(); got != `<p>A!B</p>` {
				t.Fatal(got)
			}
		}
	}
}
func TestDraftSelectionPaintUsesGlobalRowsAfterScroll(t *testing.T) {
	d := NewDraft(Parse(`<p>one</p><p>two</p><p>three</p><p>four</p>`))
	n := d.Snapshot()[3].Children[0]
	d.SetCursor(Anchor{n.ID, 0})
	d.Select(false, false)
	v := d.View()
	f := v.Viewport(40, 2, true)
	if f.Top == 0 {
		t.Fatal("test not scrolled")
	}
	f = d.Highlight(f)
	for row, r := range f.Rows {
		for _, s := range r.Spans {
			if s.Style&SelectedStyle != 0 && (row != 3 || s.Text != "f") {
				t.Fatalf("wrong selected row%d %q", row, s.Text)
			}
		}
	}
	if f.Rows[3].Spans[0].Style&SelectedStyle == 0 {
		t.Fatal("fourth paragraph unselected")
	}
}

func TestDraftEmptyQuoteExitConfirmsMetadataLoss(t *testing.T) {
	const source = `<blockquote data-comment-id="42"><p></p></blockquote>`
	d := NewDraft(Parse(source))
	p := d.Snapshot()[0].Children[0]
	_ = d.SetCursor(Anchor{p.ID, 0})
	if err := d.Apply(Action{Kind: Split}, false); err != ErrQuoteMetadataConfirmation {
		t.Fatalf("expected confirmation, got %v", err)
	}
	if d.DraftHTML() != source {
		t.Fatal("unconfirmed exit changed source")
	}
	if err := d.Apply(Action{Kind: Split}, true); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p></p>` {
		t.Fatalf("%q %v", got, err)
	}
	d.Undo()
	if d.DraftHTML() != source {
		t.Fatal("undo lost metadata")
	}
	d.Redo()
	if got, _ := d.HTML(); got != `<p></p>` {
		t.Fatal(got)
	}
}

func TestDraftFormattingRejectsCodeSelectionAtomically(t *testing.T) {
	const source = `<p><strong>Alpha</strong></p><pre><code>Beta</code></pre>`
	d := NewDraft(Parse(source))
	ns := d.Snapshot()
	_ = d.SetCursor(Anchor{ns[0].Children[0].Children[0].ID, 0})
	d.Select(false, false)
	_ = d.SetCursor(Anchor{ns[1].Children[0].Children[0].ID, 3})
	before := d.Selection()
	for _, a := range []Action{{Kind: ToggleMark, Text: "strong"}, {Kind: ClearMarks}} {
		if err := d.Apply(a, false); err == nil {
			t.Fatal("Code formatting reported success")
		}
		if d.DraftHTML() != source || *d.Selection() != *before {
			t.Fatal("rejected formatting changed draft")
		}
	}
	if d.Undo() {
		t.Fatal("rejected formatting added history")
	}
}
