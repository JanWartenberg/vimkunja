package description

import (
	"reflect"
	"strings"
	"testing"
)

func cursorText(t *testing.T, d *Draft, value string, offset int) {
	t.Helper()
	var find func([]Node) bool
	find = func(ns []Node) bool {
		for _, n := range ns {
			if n.Kind == Text && n.Text == value {
				if err := d.SetCursor(Anchor{n.ID, offset}); err != nil {
					t.Fatal(err)
				}
				return true
			}
			if find(n.Children) {
				return true
			}
		}
		return false
	}
	if !find(d.Snapshot()) {
		t.Fatalf("text %q absent", value)
	}
}
func expectDraftHTML(t *testing.T, d *Draft, want string) {
	t.Helper()
	got, err := d.HTML()
	if err != nil || got != want {
		t.Fatalf("HTML %q, err %v; want %q", got, err, want)
	}
}
func TestListIndentAppendsExistingChildren(t *testing.T) {
	d := NewDraft(Parse(`<ul><li>A<ul><li>old</li></ul></li><li>B</li><li>C</li></ul>`))
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	cursorText(t, d, "C", 0)
	if err := d.Apply(Action{Kind: IndentList}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ul><li>A<ul><li>old</li><li>B</li><li>C</li></ul></li></ul>`)
	d.Undo()
	expectDraftHTML(t, d, `<ul><li>A<ul><li>old</li></ul></li><li>B</li><li>C</li></ul>`)
}

func TestListOutdentPreservesReadingOrder(t *testing.T) {
	d := NewDraft(Parse(`<ul><li>P<ul><li>A</li><li>B<ul><li>child</li></ul></li><li>C</li></ul></li><li>Next</li></ul>`))
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: OutdentList}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ul><li>P<ul><li>A</li></ul></li><li>B<ul><li>child</li><li>C</li></ul></li><li>Next</li></ul>`)
}
func TestListSelectionRejectsFirstAndDifferentParents(t *testing.T) {
	for _, pair := range [][2]string{{"A", "B"}, {"B", "D"}} {
		source := `<ul><li>A</li><li>B</li></ul><ul><li>C</li><li>D</li></ul>`
		d := NewDraft(Parse(source))
		cursorText(t, d, pair[0], 0)
		d.Select(true, false)
		cursorText(t, d, pair[1], 0)
		if err := d.Apply(Action{Kind: IndentList}, false); err == nil {
			t.Fatal("invalid selection succeeded")
		}
		expectDraftHTML(t, d, source)
		if d.Undo() {
			t.Fatal("rejected operation changed history")
		}
	}
}
func TestCheckboxMiddleSplitKeepsChildrenOnBeta(t *testing.T) {
	d := NewDraft(Parse(`<ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="A"><p>AlphaBeta</p><ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="child"><p>Child</p></li></ul></li></ul>`))
	cursorText(t, d, "AlphaBeta", 5)
	if err := d.Apply(Action{Kind: Split}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="A"><p>Alpha</p></li><li data-type="taskItem" data-checked="false"><p>Beta</p><ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="child"><p>Child</p></li></ul></li></ul>`)
	c, err := d.PrepareSave()
	if err != nil {
		t.Fatal(err)
	}
	if err := d.AdoptSave(c); err != nil {
		t.Fatal(err)
	}
	ns := d.Snapshot()[0].Children
	if attributeValue(ns[0].Attributes, "data-task-id") != "A" || attributeValue(ns[1].Attributes, "data-task-id") == "" || attributeValue(ns[1].Children[1].Children[0].Attributes, "data-task-id") != "child" {
		t.Fatal("identity repair changed retained IDs")
	}
}

func TestOrderedMiddleExitKeepsContinuation(t *testing.T) {
	d := NewDraft(Parse(`<ol start="7" reversed><li>A</li><li value="3">B</li><li></li><li>C</li></ol>`))
	empty := d.Snapshot()[0].Children[2]
	_ = d.SetCursor(Anchor{empty.ID, 0})
	if err := d.Apply(Action{Kind: Split}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ol start="7" reversed><li>A</li><li value="3">B</li></ol><p></p><ol start="1" reversed><li>C</li></ol>`)
}
func TestEmptyImportedListFillAndDelete(t *testing.T) {
	for _, source := range []string{`<ul data-note="keep"></ul>`, `<ol></ol>`, `<ul data-type="taskList"></ul>`} {
		d := NewDraft(Parse(source))
		v := d.View()
		if v.Layout(40, true).CursorX < 0 {
			t.Fatal("empty list has no fill cursor")
		}
		if err := d.Apply(Action{Kind: Split}, false); err != nil {
			t.Fatal(err)
		}
		if d.DraftHTML() != source {
			t.Fatal("empty Enter materialized entry")
		}
		if err := d.Apply(Action{Kind: Insert, Text: "Hello"}, false); err != nil {
			t.Fatal(err)
		}
		if _, err := d.HTML(); err != nil {
			t.Fatal(err)
		}
		li := d.Snapshot()[0].Children[0]
		if li.Tag != "li" {
			t.Fatal("filled list not an entry")
		}
		if err := d.Apply(Action{Kind: DeleteBlock}, false); err != nil {
			t.Fatal(err)
		}
		expectDraftHTML(t, d, "")
	}
}

func TestListConversionAndCheckboxToggle(t *testing.T) {
	d := NewDraft(Parse(`<p>Hello</p>`))
	if err := d.Apply(Action{Kind: ConvertList, Text: "task"}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>Hello</p></li></ul>`)
	if err := d.Apply(Action{Kind: ToggleCheckbox}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ul data-type="taskList"><li data-type="taskItem" data-checked="true"><p>Hello</p></li></ul>`)
	if err := d.Apply(Action{Kind: ConvertList, Text: "task"}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<p>Hello</p>`)
}

func TestQuoteLocalSplitSharesSourceAndWholeUnwrapConfirms(t *testing.T) {
	source := `<blockquote data-comment-id="42" cite="https://example.test/source"><p>A</p><p>B</p><p>C</p></blockquote>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "B", 0)
	if err := d.Apply(Action{Kind: ToggleQuote}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<blockquote data-comment-id="42" cite="https://example.test/source"><p>A</p></blockquote><p>B</p><blockquote data-comment-id="42" cite="https://example.test/source"><p>C</p></blockquote>`)
	d.Undo()
	expectDraftHTML(t, d, source)
	cursorText(t, d, "A", 0)
	d.Select(true, false)
	cursorText(t, d, "C", 0)
	if err := d.Apply(Action{Kind: UnwrapQuote}, false); err != ErrQuoteMetadataConfirmation {
		t.Fatalf("expected quote confirmation: %v", err)
	}
	expectDraftHTML(t, d, source)
	if err := d.Apply(Action{Kind: UnwrapQuote}, true); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<p>A</p><p>B</p><p>C</p>`)
}
func TestQuoteSharedWrapAndMixedReductionAtomic(t *testing.T) {
	source := `<p>A</p><p>B</p><blockquote><p>C</p></blockquote>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "A", 0)
	d.Select(true, false)
	cursorText(t, d, "B", 0)
	if err := d.Apply(Action{Kind: WrapQuote}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<blockquote><p>A</p><p>B</p></blockquote><blockquote><p>C</p></blockquote>`)
	d.Undo()
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	cursorText(t, d, "C", 0)
	if err := d.Apply(Action{Kind: UnwrapQuote}, false); err == nil {
		t.Fatal("mixed reduction succeeded")
	}
	expectDraftHTML(t, d, source)
}

func TestListCompatibleMergeAndRejection(t *testing.T) {
	for _, source := range []string{`<ul><li>A</li><li>B</li></ul>`, `<ul><li><p>A</p></li><li><p>B</p></li></ul>`} {
		d := NewDraft(Parse(source))
		cursorText(t, d, "B", 0)
		if err := d.Apply(Action{Kind: Backspace}, false); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(source, "<p>") {
			expectDraftHTML(t, d, `<ul><li><p>AB</p></li></ul>`)
		} else {
			expectDraftHTML(t, d, `<ul><li>AB</li></ul>`)
		}
	}
	source := `<ul><li data-note="A">A</li><li data-note="B">B</li></ul>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "B", 0)
	if err := d.Apply(Action{Kind: Backspace}, false); err == nil {
		t.Fatal("incompatible merge succeeded")
	}
	expectDraftHTML(t, d, source)
}
func TestListVisualDeleteRemovesSelectedSubtree(t *testing.T) {
	d := NewDraft(Parse(`<ul><li>A<ul><li>child</li></ul></li><li>B</li></ul>`))
	cursorText(t, d, "A", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: DeleteSelection}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ul><li>B</li></ul>`)
}

func TestReversedSplitPreservesBothOrdinalSequences(t *testing.T) {
	d := NewDraft(Parse(`<ol reversed><li>A</li><li>B</li><li></li><li>C</li></ol>`))
	n := d.Snapshot()[0].Children[2]
	_ = d.SetCursor(Anchor{n.ID, 0})
	if err := d.Apply(Action{Kind: Split}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ol reversed start="4"><li>A</li><li>B</li></ol><p></p><ol reversed start="1"><li>C</li></ol>`)
}
func TestOutdentRejectsRemovalOfUnknownContainerMetadata(t *testing.T) {
	source := `<ul><li>P<ul id="unique"><li>A</li></ul></li></ul>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "A", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: OutdentList}, false); err == nil {
		t.Fatal("lost list metadata")
	}
	expectDraftHTML(t, d, source)
}
func TestQuoteLocalUnwrapCannotExpandToWholeList(t *testing.T) {
	source := `<blockquote><ul><li>A</li><li>B</li><li>C</li></ul></blockquote>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "B", 0)
	if err := d.Apply(Action{Kind: ToggleQuote}, false); err == nil {
		t.Fatal("implicitly unwrapped unselected entries")
	}
	expectDraftHTML(t, d, source)
}

func TestListSelectionIncludesNoOutsideBlocks(t *testing.T) {
	source := `<ul><li>A</li><li>B</li></ul><p>C</p>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	cursorText(t, d, "C", 0)
	if err := d.Apply(Action{Kind: IndentList}, false); err == nil {
		t.Fatal("partially moved invalid selection")
	}
	expectDraftHTML(t, d, source)
}
func TestEmptyListEnterPreservesHistory(t *testing.T) {
	d := NewDraft(Parse(`<ul></ul>`))
	if err := d.Apply(Action{Kind: Split}, false); err != nil {
		t.Fatal(err)
	}
	if d.Undo() {
		t.Fatal("no-op fill Enter added Undo")
	}
}
func TestCheckboxParagraphVisualDeleteIncludesChildren(t *testing.T) {
	source := `<ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="A"><p>A</p><ul><li>child</li></ul></li><li data-type="taskItem" data-checked="false" data-task-id="B"><p>B</p></li></ul>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "A", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: DeleteSelection}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="B"><p>B</p></li></ul>`)
}

func TestListOverlapAndKindsPreserveMovedIdentities(t *testing.T) {
	source := `<ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="A"><p>A</p></li><li data-type="taskItem" data-checked="true" data-task-id="B"><p>B</p><ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="child"><p>child</p></li></ul></li></ul>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	cursorText(t, d, "child", 0)
	if err := d.Apply(Action{Kind: IndentList}, false); err != nil {
		t.Fatal(err)
	}
	moved := d.Snapshot()[0].Children[0].Children[1].Children
	if len(moved) != 1 || attributeValue(moved[0].Attributes, "data-task-id") != "B" || attributeValue(moved[0].Attributes, "data-checked") != "true" || attributeValue(moved[0].Children[1].Children[0].Attributes, "data-task-id") != "child" {
		t.Fatal("move duplicated subtree or changed state/IDs")
	}
	d.Undo()
	expectDraftHTML(t, d, source)
	if err := d.Apply(Action{Kind: OutdentList}, false); err == nil {
		t.Fatal("root outdent succeeded")
	}
}
func TestQuoteUnknownSplitRejectsButWholeLayerConfirms(t *testing.T) {
	source := `<blockquote id="unique" data-unknown="value"><p>A</p><p>B</p><p>C</p></blockquote>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "B", 0)
	if err := d.Apply(Action{Kind: ToggleQuote}, false); err == nil {
		t.Fatal("copied unique attributes")
	}
	expectDraftHTML(t, d, source)
	cursorText(t, d, "A", 0)
	d.Select(true, false)
	cursorText(t, d, "C", 0)
	if err := d.Apply(Action{Kind: UnwrapQuote}, false); err != ErrQuoteMetadataConfirmation {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, source)
	if err := d.Apply(Action{Kind: UnwrapQuote}, true); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<p>A</p><p>B</p><p>C</p>`)
	d.Undo()
	expectDraftHTML(t, d, source)
}
func TestStructuralOpaqueMoveConfirmsAndUndoRestores(t *testing.T) {
	source := `<ul><li>A</li><li>B<img alt='map'></li></ul>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: IndentList}, false); err != ErrConfirmation {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, source)
	if err := d.Apply(Action{Kind: IndentList}, true); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ul><li>A<ul><li>B<img alt='map'></li></ul></li></ul>`)
	d.Undo()
	expectDraftHTML(t, d, source)
	d.Redo()
	expectDraftHTML(t, d, `<ul><li>A<ul><li>B<img alt='map'></li></ul></li></ul>`)
}

func FuzzDraftStructureActionsAreAtomic(f *testing.F) {
	for _, source := range []string{`<ul><li>A</li><li>B<ul><li>child</li></ul></li></ul>`, `<ol reversed><li>A</li><li></li><li>C</li></ol>`, `<blockquote data-comment-id="42"><p>A</p><p>B</p><p>C</p></blockquote>`, `<ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="A"><p>AlphaBeta</p></li></ul>`, `<ul data-note="keep"></ul>`, `<p>A<img alt='map'>B</p>`} {
		f.Add(source, []byte{0, 1, 3, 2, 4, 6, 7, 5})
	}
	f.Fuzz(func(t *testing.T, source string, steps []byte) {
		if len(source) > 4096 || len(steps) > 24 {
			return
		}
		d := NewDraft(Parse(source))
		if d.Classification() == Blocked {
			return
		}
		actions := []Action{{Kind: IndentList}, {Kind: OutdentList}, {Kind: Split}, {Kind: ConvertList, Text: "task"}, {Kind: ToggleQuote}, {Kind: WrapQuote}, {Kind: UnwrapQuote}, {Kind: Backspace}}
		for _, step := range steps {
			var anchors []Anchor
			var collect func([]Node)
			collect = func(ns []Node) {
				for _, n := range ns {
					if n.Kind == Text || n.Kind == Opaque || len(n.Children) == 0 {
						anchors = append(anchors, Anchor{n.ID, 0})
					}
					collect(n.Children)
				}
			}
			collect(d.Snapshot())
			if len(anchors) == 0 {
				return
			}
			d.CancelSelection()
			_ = d.SetCursor(anchors[int(step)%len(anchors)])
			if step%2 == 0 {
				d.Select(true, false)
				_ = d.SetCursor(anchors[(int(step)+1)%len(anchors)])
			}
			before := d.Snapshot()
			cursor := d.Cursor()
			selection := d.Selection()
			err := d.Apply(actions[int(step)%len(actions)], step%3 == 0)
			if err != nil {
				if !reflect.DeepEqual(d.Snapshot(), before) || d.Cursor() != cursor || !reflect.DeepEqual(d.Selection(), selection) {
					t.Fatal("failed operation partially mutated draft")
				}
				continue
			}
			if _, err := d.HTML(); err != nil {
				t.Fatalf("accepted action%d cannot serialize: %v", step%8, err)
			}
		}
	})
}

func TestQuoteVisualInsideEntryKeepsLocalScope(t *testing.T) {
	d := NewDraft(Parse(`<ul><li>Outer<blockquote><p>A</p><p>B</p></blockquote></li></ul>`))
	cursorText(t, d, "A", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: UnwrapQuote}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ul><li>Outer<p>A</p><blockquote><p>B</p></blockquote></li></ul>`)
}
func TestEmptyQuoteUnwrapNormalAndVisual(t *testing.T) {
	for _, visual := range []bool{false, true} {
		d := NewDraft(Parse(`<p>A</p><blockquote data-comment-id="42"></blockquote>`))
		q := d.Snapshot()[1]
		_ = d.SetCursor(Anchor{q.ID, 0})
		if visual {
			d.Select(true, false)
		}
		if err := d.Apply(Action{Kind: ToggleQuote}, false); err != ErrQuoteMetadataConfirmation {
			t.Fatal(err)
		}
		if err := d.Apply(Action{Kind: ToggleQuote}, true); err != nil {
			t.Fatal(err)
		}
		expectDraftHTML(t, d, `<p>A</p><p></p>`)
	}
}
func TestQuoteCodeLinesRequireExplicitBlockSelection(t *testing.T) {
	d := NewDraft(Parse("<pre><code>A\nB\nC</code></pre>"))
	cursorText(t, d, "A\nB\nC", 2)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: WrapQuote}, false); err == nil {
		t.Fatal("implicitly wrapped unselected Code lines")
	}
	d.Select(true, true)
	if err := d.Apply(Action{Kind: WrapQuote}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, "<blockquote><pre><code>A\nB\nC</code></pre></blockquote>")
}
func TestListEndSplitEditsNewLabelInsteadOfChild(t *testing.T) {
	for _, source := range []string{`<ul><li>Alpha<ul><li>Child</li></ul></li></ul>`, `<ul data-type="taskList"><li data-type="taskItem" data-checked="true"><p>Alpha</p><ul><li>Child</li></ul></li></ul>`} {
		d := NewDraft(Parse(source))
		cursorText(t, d, "Alpha", 5)
		if err := d.Apply(Action{Kind: Split}, false); err != nil {
			t.Fatal(err)
		}
		if err := d.Apply(Action{Kind: Insert, Text: "Beta"}, false); err != nil {
			t.Fatal(err)
		}
		ns := d.Snapshot()[0].Children
		if len(ns) != 2 || ns[1].Children[0].Tag != "p" || ns[1].Children[0].Children[0].Text != "Beta" || ns[1].Children[1].Children[0].Children[0].Text != "Child" {
			t.Fatal("new label modified old child")
		}
		if _, err := d.HTML(); err != nil {
			t.Fatal(err)
		}
	}
}
func TestMovedSelectionActiveEndAndSafeRetainedMetadata(t *testing.T) {
	d := NewDraft(Parse(`<ul id="list"><li>A</li><li>B</li><li>C</li></ul>`))
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	cursorText(t, d, "C", 0)
	if err := d.Apply(Action{Kind: IndentList}, false); err != nil {
		t.Fatal(err)
	}
	if d.Cursor() != d.Selection().End {
		t.Fatal("active endpoint and cursor differ")
	}
	expectDraftHTML(t, d, `<ul id="list"><li>A<ul><li>B</li><li>C</li></ul></li></ul>`)
}

func TestEmptyEntryExitRetainsChildMetadata(t *testing.T) {
	d := NewDraft(Parse(`<ul><li><p id="keep" data-note="value"></p></li></ul>`))
	p := d.Snapshot()[0].Children[0].Children[0]
	_ = d.SetCursor(Anchor{p.ID, 0})
	if err := d.Apply(Action{Kind: Split}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<p id="keep" data-note="value"></p>`)
}
func TestQuotedParagraphInsideListKeepsEditScope(t *testing.T) {
	for _, kind := range []ActionKind{DeleteBlock, Split, OpenAbove, OpenBelow} {
		d := NewDraft(Parse(`<ul><li>Outer<blockquote><p>AB</p><p>C</p></blockquote></li></ul>`))
		cursorText(t, d, "AB", 1)
		if err := d.Apply(Action{Kind: kind}, false); err != nil {
			t.Fatal(err)
		}
		want := map[ActionKind]string{
			DeleteBlock: `<ul><li>Outer<blockquote><p>C</p></blockquote></li></ul>`,
			Split:       `<ul><li>Outer<blockquote><p>A</p><p>B</p><p>C</p></blockquote></li></ul>`,
			OpenAbove:   `<ul><li>Outer<blockquote><p></p><p>AB</p><p>C</p></blockquote></li></ul>`,
			OpenBelow:   `<ul><li>Outer<blockquote><p>AB</p><p></p><p>C</p></blockquote></li></ul>`,
		}
		expectDraftHTML(t, d, want[kind])
	}
}
func TestCodeLineSelectionCannotMoveEnclosingEntry(t *testing.T) {
	source := "<ul><li>A</li><li><pre><code>B\nC</code></pre></li></ul>"
	d := NewDraft(Parse(source))
	cursorText(t, d, "B\nC", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: IndentList}, false); err == nil {
		t.Fatal("moved unselected Code line")
	}
	expectDraftHTML(t, d, source)
}

func TestCrossKindOutdentCreatesMatchingContainerAndContinuation(t *testing.T) {
	d := NewDraft(Parse(`<ol><li>P<ul><li>B</li></ul></li><li>Next</li></ol>`))
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: OutdentList}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<ol><li>P</li></ol><ul><li>B</li></ul><ol start="2"><li>Next</li></ol>`)
}
func TestOutdentCannotReorderOwnerTail(t *testing.T) {
	source := `<ul><li>P<ul><li>A</li><li>B</li></ul><p>Tail</p></li><li>Next</li></ul>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: OutdentList}, false); err == nil {
		t.Fatal("outdent reordered unselected tail")
	}
	expectDraftHTML(t, d, source)
}
func TestEmptyQuoteExitRetainsParagraphMetadata(t *testing.T) {
	d := NewDraft(Parse(`<blockquote><p id="keep" data-note="value"></p></blockquote>`))
	p := d.Snapshot()[0].Children[0]
	_ = d.SetCursor(Anchor{p.ID, 0})
	if err := d.Apply(Action{Kind: Split}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<p id="keep" data-note="value"></p>`)
}
func TestVisualEmptyListQuoteDoesNotIncludeNeighbor(t *testing.T) {
	d := NewDraft(Parse(`<ul></ul><ol></ol><p>A</p>`))
	n := d.Snapshot()[0]
	_ = d.SetCursor(Anchor{n.ID, 0})
	d.Select(true, false)
	if err := d.Apply(Action{Kind: WrapQuote}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<blockquote><ul></ul></blockquote><ol></ol><p>A</p>`)
}

func TestRetainedSubtreeSelectionEndingInCodeCanMoveAgain(t *testing.T) {
	source := "<ul><li>A</li><li>B<pre><code>C\nD</code></pre></li></ul>"
	d := NewDraft(Parse(source))
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: IndentList}, false); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(Action{Kind: OutdentList}, false); err != nil {
		t.Fatal("retained selection rejected:", err)
	}
	expectDraftHTML(t, d, source)
}

func TestSwappedRetainedSubtreeWithCodeTailCanMoveAgain(t *testing.T) {
	source := "<ul><li>A</li><li>B<pre><code>C\nD</code></pre></li></ul>"
	d := NewDraft(Parse(source))
	cursorText(t, d, "B", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: IndentList}, false); err != nil {
		t.Fatal(err)
	}
	d.SwapSelection()
	if d.CodeSelection() {
		t.Fatal("endpoint swap changed selection kind")
	}
	if err := d.Apply(Action{Kind: OutdentList}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, source)
}
func TestCodeLineSelectionOriginSurvivesEndpointSwap(t *testing.T) {
	d := NewDraft(Parse("<pre><code>A\nB\nC</code></pre><p>Outside</p>"))
	cursorText(t, d, "A\nB\nC", 2)
	d.Select(true, false)
	cursorText(t, d, "A\nB\nC", 4)
	d.SwapSelection()
	outside := d.Snapshot()[1].Children[0]
	if !d.CodeSelection() || d.AllowSelectionCursor(Anchor{outside.ID, 0}) {
		t.Fatal("Code line boundary lost after swap")
	}
	if err := d.Apply(Action{Kind: WrapQuote}, false); err == nil {
		t.Fatal("Code lines expanded to whole block")
	}
}

func TestSelectedCheckboxConversionPreservesExistingIdentityAndCheckedState(t *testing.T) {
	const source = `<p>Before</p><ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="keep"><p>Existing</p></li></ul><p>After</p>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "After", 1)
	d.Select(false, false)
	cursorText(t, d, "Before", 1)
	if err := d.Apply(Action{Kind: ConvertList, Text: "task"}, false); err != nil {
		t.Fatal(err)
	}
	want := `<ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>Before</p></li></ul><ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="keep"><p>Existing</p></li></ul><ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>After</p></li></ul>`
	expectDraftHTML(t, d, want)
	if err := d.Apply(Action{Kind: ConvertList, Text: "task"}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, want)
	if !d.Undo() {
		t.Fatal("conversion missing from history")
	}
	expectDraftHTML(t, d, source)
}

func TestSelectedCheckboxConversionAcrossQuoteKeepsOrderAndOneUndo(t *testing.T) {
	const source = `<blockquote><p>Alpha</p><p>Beta</p></blockquote><p>Gamma</p><p>Keep</p>`
	d := NewDraft(Parse(source))
	cursorText(t, d, "Gamma", 1)
	d.Select(false, false)
	cursorText(t, d, "Alpha", 2)
	if err := d.Apply(Action{Kind: ConvertList, Text: "task"}, false); err != nil {
		t.Fatal(err)
	}
	expectDraftHTML(t, d, `<blockquote><ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>Alpha</p></li><li data-type="taskItem" data-checked="false"><p>Beta</p></li></ul></blockquote><ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>Gamma</p></li></ul><p>Keep</p>`)
	d.Undo()
	expectDraftHTML(t, d, source)
}

func TestSelectedCheckboxConversionProtectsOpaqueAndRejectsCodeAtomically(t *testing.T) {
	const opaque = `<p>Alpha<mention-user data-id="jan" data-label="Jan"></mention-user></p><p>Beta</p>`
	d := NewDraft(Parse(opaque))
	d.Select(true, false)
	cursorText(t, d, "Beta", 0)
	if err := d.Apply(Action{Kind: ConvertList, Text: "task"}, false); err != ErrConfirmation {
		t.Fatalf("missing preservation confirmation: %v", err)
	}
	expectDraftHTML(t, d, opaque)
	if err := d.Apply(Action{Kind: ConvertList, Text: "task"}, true); err != nil {
		t.Fatal(err)
	}
	html, err := d.HTML()
	if err != nil || !strings.Contains(html, `<mention-user data-id="jan" data-label="Jan"></mention-user>`) {
		t.Fatalf("opaque changed: %s %v", html, err)
	}
	const code = `<p>Alpha</p><pre><code>original()</code></pre>`
	d = NewDraft(Parse(code))
	d.Select(false, false)
	cursorText(t, d, "original()", 2)
	if err := d.Apply(Action{Kind: ConvertList, Text: "task"}, false); err == nil {
		t.Fatal("Code silently converted")
	}
	expectDraftHTML(t, d, code)
}
