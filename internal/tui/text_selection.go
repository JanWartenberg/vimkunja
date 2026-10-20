package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rivo/uniseg"
)

// Selection offsets are source rune positions, independent of wrapping and ANSI.
type textSelection struct {
	anchor int
	lines  bool
	object rune
}

func (s *textNormalState) selectionRange(text string, cursor int) (int, int) {
	r := []rune(text)
	if s.selection == nil {
		return textLineStart(text, cursor), textLineEnd(text, cursor)
	}
	lo, hi := min(s.selection.anchor, cursor), max(s.selection.anchor, cursor)
	lo, hi = clamp(lo, 0, len(r)), clamp(hi, 0, len(r))
	if s.selection.lines {
		lo = textLineStart(text, lo)
		hi = textLineEnd(text, hi)
		if hi < len(r) {
			hi++
		}
	} else {
		hi = min(hi+1, len(r))
		// A selection always contains complete displayed graphemes.
		it := uniseg.NewGraphemes(text)
		offset := 0
		for it.Next() {
			end := offset + len([]rune(it.Str()))
			if lo > offset && lo < end {
				lo = offset
			}
			if hi > offset && hi < end {
				hi = end
			}
			offset = end
		}
	}
	return lo, hi
}
func (s *textNormalState) selectionLabel() string {
	if s.selection == nil {
		return "NORMAL"
	}
	if s.selection.lines {
		return "VISUAL LINES"
	}
	return "VISUAL"
}
func applyTextSelectionKey(s *textNormalState, text string, cursor int, key string, multiline bool) textNormalResult {
	result := textNormalResult{text: text, cursor: clamp(cursor, 0, len([]rune(text))), handled: true}
	if s.selection == nil {
		s.reset()
		s.selection = &textSelection{anchor: normalCharacterIndex(text, result.cursor), lines: key == "V"}
		result.cursor = s.selection.anchor
		return result
	}
	if key == "esc" {
		s.reset()
		return result
	}
	if s.selection.object != 0 {
		if key == "w" || key == "W" {
			lo, hi := wordObjectRange([]rune(text), result.cursor, key == "W", s.selection.object == 'a', 1)
			s.selection.anchor = lo
			result.cursor = max(lo, hi-1)
			s.selection.lines = false
		}
		s.selection.object = 0
		return result
	}
	if s.operator == 'g' {
		s.operator = 0
		if key == "g" {
			result.cursor = 0
		}
		return result
	}

	switch key {
	case "v", "V":
		lines := key == "V"
		if lines == s.selection.lines {
			s.reset()
		} else {
			s.selection.lines = lines
		}
		return result
	case "o":
		result.cursor, s.selection.anchor = s.selection.anchor, result.cursor
		return result
	case "d", "x", "delete", "backspace", "c":
		lo, hi := s.selectionRange(text, result.cursor)
		// Remove the preceding separator when deleting the final selected line.
		if s.selection.lines && key == "c" {
			hi = textLineEnd(text, max(s.selection.anchor, result.cursor))
		} else if s.selection.lines && hi == len([]rune(text)) && lo > 0 {
			lo--
		}
		result.text, result.cursor = deleteRuneRange(text, lo, hi)
		result.enterInsert = key == "c"
		s.reset()
		return result
	case "i", "a":
		s.selection.object = []rune(key)[0]
		return result
	}

	if key == "g" {
		s.operator = 'g'
		return result
	}
	s.operator = 0
	switch key {
	case "left":
		key = "h"
	case "right":
		key = "l"
	case "up":
		key = "k"
	case "down":
		key = "j"
	case "home":
		key = "0"
	case "end":
		key = "$"
	}
	if len(key) == 1 && (key[0] >= '1' && key[0] <= '9' || key == "0" && s.count > 0) {
		s.count = s.count*10 + int(key[0]-'0')
		return result
	}
	count := max(1, s.count)
	s.count = 0
	switch key {
	case "h", "l", "j", "k", "w", "b", "e", "W", "B", "E", "0", "^", "$":
		result.cursor = moveTextMotion([]rune(text), result.cursor, key, count, multiline)
	case "G":
		result.cursor = len([]rune(text))
	}
	return result
}

func renderSelectionRange(fragment string, start, lo, hi int) string {
	r := []rune(fragment)
	a, b := clamp(lo-start, 0, len(r)), clamp(hi-start, 0, len(r))
	if a >= b {
		if len(r) == 0 && lo <= start && hi > start {
			return lipgloss.NewStyle().Reverse(true).Render(" ")
		}
		return fragment
	}
	return string(r[:a]) + lipgloss.NewStyle().Reverse(true).Render(string(r[a:b])) + string(r[b:])
}
func selectedPlainText(text string, cursor int, state *textNormalState) string {
	lo, hi := state.selectionRange(text, cursor)
	return string([]rune(text)[lo:hi])
}
func pastePlainText(text string, cursor int, state *textNormalState, value string, multiline bool) (string, int) {
	if value == "" {
		return text, cursor
	}
	value = normalizePastedText(value)
	if !multiline {
		value = strings.ReplaceAll(value, "\n", " ")
	}
	if state.selection != nil {
		lo, hi := state.selectionRange(text, cursor)
		text, cursor = deleteRuneRange(text, lo, hi)
	}
	state.reset()
	text = insertTextAtCursor(text, &cursor, value)
	return text, cursor
}
