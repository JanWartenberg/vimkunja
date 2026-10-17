package tutorial

import (
	tea "charm.land/bubbletea/v2"
	"strings"
	"testing"
)

func markupKey(value string) tea.KeyPressMsg {
	code := map[string]rune{"enter": tea.KeyEnter, "esc": tea.KeyEscape, "tab": tea.KeyTab, "space": ' '}
	if c, ok := code[value]; ok {
		text := ""
		if value == "space" {
			text = " "
		}
		return tea.KeyPressMsg(tea.Key{Code: c, Text: text})
	}
	if strings.HasPrefix(value, "ctrl+") {
		return tea.KeyPressMsg(tea.Key{Code: []rune(strings.TrimPrefix(value, "ctrl+"))[0], Mod: tea.ModCtrl})
	}
	return tea.KeyPressMsg(tea.Key{Code: []rune(value)[0], Text: value})
}

// AT-01 must work through the unrestricted app, without pre-opening a draft.
func TestMarkupPracticeChangeInnerWordFromFreshTask(t *testing.T) {
	s := newPracticeSession(t)
	s.keys("c", "i", "w")
	if !strings.Contains(s.screen(), "INSERT") {
		t.Fatal("ciw did not enter Insert")
	}
	s.text("Hallo")
	s.keys("esc", "u")
	if !strings.Contains(s.screen(), "Alpha beta") {
		t.Fatal("one Undo did not restore Alpha beta")
	}
	s.keys("ctrl+r")
	s.wantSaved(`<p>Hallo beta</p>` + practiceTail)
}
