package description

import (
	"strings"
	"testing"
)

func TestCheckboxSelectionPreservesControlAndSaveUndo(t *testing.T) {
	const source = `<ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="entry"><label><input type="checkbox"><span></span></label><div><blockquote><p>one</p><p>two</p></blockquote></div></li></ul>`
	for _, tc := range []struct {
		name   string
		blocks bool
		want   string
	}{
		{"quoted paragraph", true, strings.Replace(source, `<p>one</p>`, "", 1)},
		{"first character", false, strings.Replace(source, `<p>one</p>`, `<p>ne</p>`, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDraft(Parse(source))
			if d.Classification() != Native {
				t.Fatal("fixture must use editable Native checkbox scaffolding")
			}
			d.Select(tc.blocks, false)
			if err := d.Apply(Action{Kind: DeleteSelection}, false); err != nil {
				t.Fatal(err)
			}
			expectDraftHTML(t, d, tc.want)
			candidate, err := d.PrepareSave()
			if err != nil {
				t.Fatal(err)
			}
			if candidate.HTML() != tc.want {
				t.Fatalf("save changed checkbox scaffolding: %s", candidate.HTML())
			}
			if err := d.AdoptSave(candidate); err != nil {
				t.Fatal(err)
			}
			if !d.Undo() {
				t.Fatal("saved deletion lost local Undo")
			}
			expectDraftHTML(t, d, source)
			if _, err := d.PrepareSave(); err != nil {
				t.Fatal("restored checkbox cannot be saved:", err)
			}
		})
	}
}

func TestCheckboxWholeEntrySelectionRemovesControl(t *testing.T) {
	for _, paragraph := range []string{`<p>one</p>`, `<p></p>`} {
		t.Run(paragraph, func(t *testing.T) {
			source := `<ul data-type="taskList"><li data-type="taskItem" data-checked="false"><label><input type="checkbox"><span></span></label><div>` + paragraph + `</div></li></ul>`
			d := NewDraft(Parse(source))
			d.Select(true, false)
			if err := d.Apply(Action{Kind: DeleteSelection}, false); err != nil {
				t.Fatal(err)
			}
			expectDraftHTML(t, d, "")
			if _, err := d.PrepareSave(); err != nil {
				t.Fatal(err)
			}
			if !d.Undo() {
				t.Fatal("whole-entry deletion lost Undo")
			}
			expectDraftHTML(t, d, source)
		})
	}
}
