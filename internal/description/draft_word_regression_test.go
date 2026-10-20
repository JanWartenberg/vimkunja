package description

import (
	"errors"
	"strings"
	"testing"
)

func TestWordSelectionKeepsNestedEntryText(t *testing.T) {
	for _, around := range []bool{false, true} {
		for _, word := range []string{"Alpha", "Beta", "Gamma"} {
			t.Run(word+map[bool]string{false: "/inner", true: "/around"}[around], func(t *testing.T) {
				const source = `<ul><li>Alpha<ul><li>Beta</li></ul>Gamma</li></ul>`
				d := NewDraft(Parse(source))
				var target Anchor
				var find func([]Node)
				find = func(ns []Node) {
					for _, n := range ns {
						if n.Kind == Text && n.Text == word {
							target = Anchor{n.ID, 1}
						}
						find(n.Children)
					}
				}
				find(d.Snapshot())
				if err := d.SetCursor(target); err != nil {
					t.Fatal(err)
				}
				if err := d.SelectWord(around); err != nil {
					t.Fatal(err)
				}
				if got := d.Copy(); got != word {
					t.Fatalf("selected %q, want only %q", got, word)
				}
				if err := d.Apply(Action{Kind: DeleteSelection}, false); err != nil {
					t.Fatal(err)
				}
				got, err := d.HTML()
				if want := strings.Replace(source, word, "", 1); err != nil || got != want {
					t.Fatalf("delete %q: %q %v; want %q", word, got, err, want)
				}
				if !d.Undo() {
					t.Fatal("missing Undo")
				}
				if got, err := d.HTML(); err != nil || got != source {
					t.Fatalf("Undo: %q %v", got, err)
				}
			})
		}
	}
}

func TestWordSelectionStandaloneOpaqueConfirmationAndHistory(t *testing.T) {
	for _, source := range []string{`<table><tr><td>Alpha</td></tr></table>`, `<iframe src='https://example.com'></iframe>`} {
		for _, around := range []bool{false, true} {
			for _, kind := range []ActionKind{DeleteSelection, ReplaceSelection} {
				d := NewDraft(Parse(source))
				atom := d.Snapshot()[0]
				if err := d.SelectWord(around); err != nil {
					t.Fatalf("select %s: %v", source, err)
				}
				if s := d.Selection(); s == nil || s.Start != (Anchor{atom.ID, 0}) || s.End != s.Start {
					t.Fatalf("expected complete atom selection: %+v", s)
				}
				if err := d.Apply(Action{Kind: kind}, false); !errors.Is(err, ErrConfirmation) {
					t.Fatalf("missing confirmation: %v", err)
				}
				if got, err := d.HTML(); err != nil || got != source {
					t.Fatalf("unconfirmed edit: %q %v", got, err)
				}
				if err := d.Apply(Action{Kind: kind}, true); err != nil {
					t.Fatal(err)
				}
				after, err := d.HTML()
				if err != nil || after == source {
					t.Fatalf("confirmed edit: %q %v", after, err)
				}
				if !d.Undo() {
					t.Fatal("missing Undo")
				}
				if got, err := d.HTML(); err != nil || got != source {
					t.Fatalf("Undo: %q %v", got, err)
				}
				if !d.Redo() {
					t.Fatal("missing Redo")
				}
				if got, err := d.HTML(); err != nil || got != after {
					t.Fatalf("Redo: %q %v", got, err)
				}
			}
		}
	}
}

func TestWordSelectionDirectQuoteText(t *testing.T) {
	d := NewDraft(Parse(`<blockquote>Alpha beta</blockquote>`))
	if err := d.SelectWord(false); err != nil {
		t.Fatal(err)
	}
	if got := d.Copy(); got != "Alpha" {
		t.Fatalf("selected %q, want Alpha", got)
	}
}
