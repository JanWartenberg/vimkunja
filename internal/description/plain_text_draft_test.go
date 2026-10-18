package description

import "testing"

func TestPlainTextDraftKeepsBaselineUntilEditAndUndo(t *testing.T) {
	const source = "**literal** &amp; text\n\nlast\t "
	d := NewTextDraft(source)
	if d.Baseline() != source || d.DraftHTML() != source {
		t.Fatal("opening converted plain source")
	}
	v := d.View()
	if v.ContentText() != source {
		t.Fatalf("plain text changed on opening: %q", v.ContentText())
	}
	v.Layout(40, true)
	if d.DraftHTML() != source {
		t.Fatal("layout changed source")
	}
	d.Select(true, false)
	if err := d.Apply(Action{Kind: ToggleMark, Text: "strong"}, false); err != nil {
		t.Fatal(err)
	}
	c, err := d.PrepareSave()
	if err != nil {
		t.Fatal(err)
	}
	if c.Baseline() != source || c.HTML() == source {
		t.Fatalf("candidate=%q baseline=%q", c.HTML(), c.Baseline())
	}
	d.Undo()
	if d.DraftHTML() != source {
		t.Fatalf("Undo failed to restore exact plain source: %q", d.DraftHTML())
	}
	d.Redo()
	if err := d.AdoptSave(c); err != nil {
		t.Fatal(err)
	}
	if d.DraftHTML() != c.HTML() {
		t.Fatal("adopt failed")
	}
}
