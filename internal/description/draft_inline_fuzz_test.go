package description

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func FuzzDraftInlineEventGrouping(f *testing.F) {
	for _, s := range []string{"***text***", "**bold** tail *", "`a\\`b` now", "**bold *italic** tail*", "----", "\\\\*text*", "*text**", "# Heading"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if !utf8.ValidString(value) {
			t.Skip()
		}
		runes := []rune(value)
		if len(runes) > 80 {
			t.Skip()
		}
		value = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return -1
			}
			return r
		}, value)
		a, b := NewDraft(Parse(`<p></p>`)), NewDraft(Parse(`<p></p>`))
		if err := a.Apply(Action{Kind: Type, Text: value}, false); err != nil {
			t.Fatal(err)
		}
		for _, r := range value {
			if err := b.Apply(Action{Kind: Type, Text: string(r)}, false); err != nil {
				t.Fatal(err)
			}
		}
		x, xe := a.HTML()
		y, ye := b.HTML()
		if xe != nil || ye != nil || x != y {
			t.Fatalf("%q: chunk=%q char=%q errors=%v/%v", value, x, y, xe, ye)
		}
		if _, err := a.PrepareSave(); err != nil {
			t.Fatal(err)
		}
	})
}
