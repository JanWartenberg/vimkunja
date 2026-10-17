package description

import (
	"errors"
	"reflect"
	"testing"
)

func TestDraftRejectsInvalidUTF8WithoutChangingStateOrHistory(t *testing.T) {
	for _, kind := range []ActionKind{Insert, Type, ReplaceSelection} {
		t.Run(map[ActionKind]string{Insert: "paste", Type: "type", ReplaceSelection: "replace"}[kind], func(t *testing.T) {
			d := NewDraft(Parse(`<p>Keep</p>`))
			if err := d.Apply(Action{Kind: Insert, Text: "valid"}, false); err != nil {
				t.Fatal(err)
			}
			wantedRedo := copyState(d.state)
			if !d.Undo() {
				t.Fatal("fixture has no redo branch")
			}
			d.Select(false, false)
			if err := d.SetCursor(Anchor{Node: d.Cursor().Node, Offset: 2}); err != nil {
				t.Fatal(err)
			}
			before := copyState(d.state)
			undo, redo, next := len(d.undo), len(d.redo), d.next
			if err := d.Apply(Action{Kind: kind, Text: "prefix\xffsuffix"}, false); !errors.Is(err, ErrStructure) {
				t.Fatalf("invalid UTF-8 was not rejected: %v", err)
			}
			if !reflect.DeepEqual(d.state, before) || len(d.undo) != undo || len(d.redo) != redo || d.next != next {
				t.Fatal("rejected input changed draft or history")
			}
			if _, err := d.PrepareSave(); err != nil {
				t.Fatalf("rejected input left unsavable draft: %v", err)
			}
			if !d.Redo() {
				t.Fatal("rejected input discarded redo branch")
			}
			if !reflect.DeepEqual(d.state, wantedRedo) {
				t.Fatal("rejected input changed redo content")
			}
		})
	}
}
