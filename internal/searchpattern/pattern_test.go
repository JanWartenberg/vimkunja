package searchpattern

import (
	"reflect"
	"regexp"
	"testing"
)

func TestVimDefaultMagicAndCaseMatching(t *testing.T) {
	for _, tc := range []struct {
		pattern, text string
		want          []string
	}{
		{"bericht", "Bericht bericht BERICHT", []string{"Bericht", "bericht", "BERICHT"}},
		{"Bericht", "Bericht bericht BERICHT", []string{"Bericht"}},
		{`\cBericht`, "bericht BERICHT", []string{"bericht", "BERICHT"}},
		{`\Cbericht`, "Bericht bericht", []string{"bericht"}},
		{`^bericht$`, "Bericht\nother", []string{"Bericht"}},
		{`a+b?`, "a+b? aaab", []string{"a+b?"}},
		{`a^b$c`, "a^b$c", []string{"a^b$c"}},
		{`^*literal`, "*literal", []string{"*literal"}},
		{`\(^rot$\|^blau$\)`, "rot\nblau\nrotblau", []string{"rot", "blau"}},
		{`\(rot\|blau\)\+`, "rotblau grün", []string{"rotblau"}},
		{`ab\?c`, "ac abc", []string{"ac", "abc"}},
		{`ab\=c`, "ac abc", []string{"ac", "abc"}},
		{`[a-c]\{2,3}`, "abc aa d", []string{"abc", "aa"}},
		{`\<über\>`, "über überall drüber ÜBER", []string{"über", "ÜBER"}},
		{`\<é\>`, "é élan", []string{"é"}},
		{`\<a.*\>`, "abc ", []string{"abc"}},
		{`a\>\|ab`, "ab", []string{"ab"}},
		{`\<a.*\>`, "abc def ", []string{"abc def"}},
		{`\<a\>\|ab`, "ab a", []string{"ab", "a"}},
		{`\<\(a\>\|ab\>\)`, "ab a", []string{"ab", "a"}},
		{`\<\|$`, "a b", []string{"", "", ""}},
		{`^\<a\>`, "a a\na", []string{"a", "a"}},
		{`\<\w\+\>`, "über foo bar", []string{"foo", "bar"}},
		{`\D\+`, "123 Äö", []string{" Äö"}},
		{`a.*z`, "a first z last z", []string{"a first z last z"}},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			p, err := Compile(tc.pattern)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, r := range p.FindAll(tc.text) {
				got = append(got, tc.text[r.Start:r.End])
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q; want %q", got, tc.want)
			}
		})
	}
}

func TestWordAssertionExecutionPreservesRegexpPriority(t *testing.T) {
	for _, pair := range [][2]string{
		{`\<a\>`, `\ba\b`},
		{`\<\(a\|ab\)\>`, `\b(a|ab)\b`},
		{`\<\(ab\|a\)\>`, `\b(ab|a)\b`},
		{`\<a\+b\?\>`, `\ba+b?\b`},
		{`\<\(a\|b\)\+\>`, `\b(a|b)+\b`},
		{`\<a\>\|ab`, `\ba\b|ab`},
		{`a\>\|ab`, `a\b|ab`},
	} {
		pattern, err := Compile(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		reference := regexp.MustCompile(pair[1])
		for value := range 729 {
			text, n := "", value
			for range 6 {
				text += string("ab "[n%3])
				n /= 3
			}
			var want []Range
			for _, match := range reference.FindAllStringIndex(text, -1) {
				want = append(want, Range{match[0], match[1]})
			}
			if got := pattern.FindAll(text); !reflect.DeepEqual(got, want) {
				t.Fatalf("%q in %q: got %v; want %v", pair[0], text, got, want)
			}
		}
	}
}

func TestRejectUnsupportedOrInvalidVimPatterns(t *testing.T) {
	for _, pattern := range []string{`\`, `[abc`, `\(`, `\1`, `a\@=`, `\vfoo`, `\{}`, `\{3,2}`, `\{1000000}`, `[[:alpha:]]`, `[\x]`} {
		if _, err := Compile(pattern); err == nil {
			t.Errorf("accepted %q", pattern)
		}
	}
}

func TestZeroWidthMatchesAreBounded(t *testing.T) {
	p, err := Compile(`^\|$`)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.FindAll("äö"); !reflect.DeepEqual(got, []Range{{0, 0}, {4, 4}}) {
		t.Fatal(got)
	}
}
