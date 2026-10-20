package description

import "testing"

func TestCharacterTransformsPreserveMarksAndProtectedObjects(t *testing.T) {
	source := `<p>a<strong>b</strong><img alt="map" src="x">c</p>`
	d := NewDraft(Parse(source))
	if err := d.Apply(Action{Kind: ToggleCase, Count: 10}, false); err != nil {
		t.Fatal(err)
	}
	if got := d.DraftHTML(); got != `<p>A<strong>B</strong><img alt="map" src="x">c</p>` {
		t.Fatal(got)
	}
	if _, err := d.PrepareSave(); err != nil {
		t.Fatal(err)
	}
	d.Undo()
	if d.DraftHTML() != source {
		t.Fatal("transform did not Undo atomically")
	}
	if err := d.Apply(Action{Kind: ReplaceCharacters, Count: 3, Text: "\n"}, false); err == nil {
		t.Fatal("character replacement accepted a newline")
	}
	if d.DraftHTML() != source {
		t.Fatal("invalid replacement changed source")
	}
}
