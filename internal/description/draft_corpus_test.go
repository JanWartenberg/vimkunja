package description

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDraftAcceptedCorpusEdits(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/description-rendering/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Fixtures []struct {
			ID     string `json:"id"`
			Source string `json:"source_html"`
			Edits  []struct {
				HTML *string `json:"expected_html"`
			} `json:"edits"`
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	for _, f := range corpus.Fixtures {
		index := -1
		d := NewDraft(Parse(f.Source))
		var action Action
		confirmed := false
		switch f.ID {
		case "paragraph-creation-and-split":
			n := d.Snapshot()[0].Children[0]
			d.SetCursor(Anchor{n.ID, 5})
			action = Action{Kind: Split}
			index = 2
		case "hard-breaks":
			n := d.Snapshot()[0].Children[5]
			d.SetCursor(Anchor{n.ID, 0})
			action = Action{Kind: Delete}
			index = 0
		case "native-selection-and-history":
			n := d.Snapshot()[0].Children[0]
			d.SetCursor(Anchor{n.ID, 6})
			d.SelectWord(false)
			action = Action{Kind: ToggleMark, Text: "strong"}
			index = 0
		case "uniform-toggle":
			n := d.Snapshot()[0]
			d.SetCursor(d.first(n.Children))
			d.Select(false, false)
			last := n.Children[1]
			d.SetCursor(Anchor{last.ID, len(last.Text) - 1})
			action = Action{Kind: ToggleMark, Text: "strong"}
			index = 0
		case "triple-strong-off":
			n := d.Snapshot()[0].Children[0].Children[0].Children[0].Children[0]
			d.SetCursor(Anchor{n.ID, 0})
			d.Select(false, false)
			d.SetCursor(Anchor{n.ID, len(n.Text) - 1})
			action = Action{Kind: ToggleMark, Text: "strong"}
			index = 0
		case "all-mark-families-off":
			n := d.Snapshot()[0].Children[0].Children[0].Children[0]
			d.SetCursor(Anchor{n.ID, 0})
			d.Select(false, false)
			d.SetCursor(Anchor{n.ID, len(n.Text) - 1})
			action = Action{Kind: ToggleMark, Text: "em"}
			index = 0
		case "opaque-parent-conversion":
			action = Action{Kind: Convert, Text: "h2"}
			confirmed = true
			index = 0
		case "opaque-edit-neighbors":
			n := d.Snapshot()[0].Children[1]
			d.SetCursor(Anchor{n.ID, 0})
			action = Action{Kind: DeleteBlock}
			confirmed = true
			index = 0
		case "malformed-html", "document-wrappers":
			action = Action{Kind: RecoverText}
			confirmed = true
			index = 0
		default:
			continue
		}
		t.Run(f.ID, func(t *testing.T) {
			if err := d.Apply(action, confirmed); err != nil {
				t.Fatal(err)
			}
			got, err := d.HTML()
			if err != nil || f.Edits[index].HTML == nil || got != *f.Edits[index].HTML {
				t.Fatalf("%q %v expected %v", got, err, f.Edits[index].HTML)
			}
		})
	}
}

func FuzzDraftActionsAreAtomic(f *testing.F) {
	for _, source := range []string{`<p>AlphaBeta</p>`, `<p>A<img alt='map'>B</p>`, `<p><strong>deep</strong></p>`, `<p>broken<strong>`} {
		f.Add(source, []byte{0, 1, 2, 3, 4, 5})
	}
	f.Fuzz(func(t *testing.T, source string, actions []byte) {
		if len(source) > 512 || len(actions) > 24 {
			t.Skip()
		}
		origin := Parse(source)
		d := NewDraft(origin)
		for _, value := range actions {
			before := d.DraftHTML()
			kind := []ActionKind{Insert, Split, HardBreak, Delete, Backspace, DeleteBlock, OpenBelow, OpenAbove, Convert, ClearDescription, RecoverText}[int(value)%11]
			text := "x"
			if kind == Convert {
				text = "h2"
			}
			err := d.Apply(Action{Kind: kind, Text: text}, false)
			if err != nil && d.DraftHTML() != before {
				t.Fatal("rejected action mutated draft")
			}
			d.View().Layout(16, true)
		}
		if origin.SourceHTML() != source {
			t.Fatal("origin mutated")
		}
	})
}
