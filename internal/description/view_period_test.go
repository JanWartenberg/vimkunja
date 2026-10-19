package description

import (
	"strings"
	"testing"
)

func TestOrdinaryPeriodsHaveNoDisclosureEscape(t *testing.T) {
	for _, text := range []string{"Move through this description.", "Price 1.25 at example.org.", "1. ordinary text"} {
		for _, plain := range []bool{false, true} {
			d := NewDraft(Parse("<p>" + text + "</p>"))
			if plain {
				d = NewTextDraft(text)
			}
			v := d.View()
			for _, focused := range []bool{false, true} {
				got := frameText(v.Layout(100, focused))
				if !strings.Contains(got, text) || strings.Contains(got, `\.`) {
					t.Fatalf("plain=%t focused=%t text=%q rendered=%q", plain, focused, text, got)
				}
			}
		}
	}
}

func TestPeriodMotionAndBackspaceHaveNoHiddenSyntaxStep(t *testing.T) {
	for _, plain := range []bool{false, true} {
		source := "<p>abc.</p>"
		d := NewDraft(Parse(source))
		if plain {
			source = "abc."
			d = NewTextDraft(source)
		}
		id := d.Cursor().Node
		v := d.View()
		v.Move("l", 3, 100, 10)
		if got := v.Cursor(); got.Syntax != 0 || got.Anchor != (Anchor{id, 3}) {
			t.Errorf("motion to ordinary period has hidden syntax: %+v", got)
		}
		if err := d.SetCursor(Anchor{id, 3}); err != nil {
			t.Fatal(err)
		}
		if err := d.Apply(Action{Kind: Backspace}, false); err != nil {
			t.Fatal(err)
		}
		if got := d.View().ContentText(); got != "ab." {
			t.Fatalf("Backspace before period = %q, want ab.", got)
		}
		d.Undo()
		if d.DraftHTML() != source {
			t.Fatal("Undo did not restore source")
		}
	}
}
