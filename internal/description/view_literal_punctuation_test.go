package description

import (
	"html"
	"strings"
	"testing"
)

func TestOrdinaryLiteralPunctuationHasNoDisclosureEscapes(t *testing.T) {
	for _, text := range []string{
		"Verstehen, wie TUI-Rendering funktioniert.",
		"'", `"`, "`", "_", "*", "~", "#", "-", "+", "[", "]", `\`,
		"single_under_score", "_word_", "a*b", "a`b", "a~b", "C#", "a+b", "name[index]",
		`C:\temp`, `say "hello" and 'goodbye'`, "[]", "# Heading", "- Milk", "+ Milk",
	} {
		for _, mode := range []string{"html", "plain", "paste"} {
			t.Run(mode+"/"+text, func(t *testing.T) {
				d := NewDraft(Parse("<p>" + html.EscapeString(text) + "</p>"))
				if mode == "plain" {
					d = NewTextDraft(text)
				} else if mode == "paste" {
					d = NewDraft(Parse("<p></p>"))
					if err := d.Apply(Action{Kind: Insert, Text: text}, false); err != nil {
						t.Fatal(err)
					}
				}
				before := d.DraftHTML()
				for _, focused := range []bool{false, true} {
					if got := frameText(d.View().Layout(100, focused)); got != text {
						t.Errorf("focused=%t rendered=%q, want %q", focused, got, text)
					}
				}
				if d.DraftHTML() != before {
					t.Fatal("rendering changed the document")
				}
			})
		}
	}
}

func TestOrdinaryPunctuationMotionAndBackspaceHaveNoHiddenSyntaxStep(t *testing.T) {
	for _, r := range "'\"`_*~#-+[]\\" {
		for _, plain := range []bool{false, true} {
			text := "ä" + string(r) + "b"
			d := NewDraft(Parse("<p>" + html.EscapeString(text) + "</p>"))
			if plain {
				d = NewTextDraft(text)
			}
			before := d.DraftHTML()
			id := d.Cursor().Node
			v := d.View()
			v.Move("l", 1, 100, 10)
			if got := v.Cursor(); got.Syntax != 0 || got.Anchor != (Anchor{id, len("ä")}) {
				t.Errorf("plain=%t character=%q: hidden syntax target %+v", plain, r, got)
			}
			if err := d.SetCursor(Anchor{id, len("ä")}); err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(Action{Kind: Backspace}, false); err != nil {
				t.Fatal(err)
			}
			if got := d.View().ContentText(); got != string(r)+"b" {
				t.Errorf("plain=%t character=%q: Backspace produced %q", plain, r, got)
			}
			d.Undo()
			if d.DraftHTML() != before {
				t.Fatal("Undo lost the original source")
			}
		}
	}
}

func TestLiteralFormattingPairsKeepProtectingEscapes(t *testing.T) {
	for _, text := range []string{"*stars*", "**bold**", "__underline__", "~~strike~~", "`code`"} {
		d := NewDraft(Parse("<p>" + text + "</p>"))
		got := frameText(d.View().Layout(100, true))
		want := strings.NewReplacer("*", "\\*", "_", "\\_", "~", "\\~", "`", "\\`").Replace(text)
		if got != want || d.View().ContentText() != text {
			t.Errorf("literal formatting pair %q lost its disclosure/projection: %q", text, got)
		}
	}
}

func TestSemanticFormattingStillDisclosesRealDelimiters(t *testing.T) {
	for _, tc := range []struct{ tag, marker string }{
		{"em", "*"}, {"strong", "**"}, {"u", "__"}, {"s", "~~"}, {"code", "`"},
	} {
		d := NewDraft(Parse("<p><" + tc.tag + ">text</" + tc.tag + "></p>"))
		want := tc.marker + "text" + tc.marker
		if got := frameText(d.View().Layout(100, true)); got != want {
			t.Errorf("%s disclosed as %q, want %q", tc.tag, got, want)
		}
	}
}
