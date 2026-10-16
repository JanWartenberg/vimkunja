package tutorial_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/tutorial"
)

func menuKey(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Text: text})
}

func TestStartMenuOffersFourModesAndQuit(t *testing.T) {
	menu := tutorial.NewStartMenu()
	screen := menu.View()
	for _, label := range []string{"Learn the basics", "Practice the basics", "Learn markup", "Practice markup", "Quit", "temporary", "j/k"} {
		if !strings.Contains(screen.Content, label) {
			t.Errorf("menu missing %q: %s", label, screen.Content)
		}
	}
	if !screen.AltScreen {
		t.Error("menu should use the alternate screen")
	}
	if _, selected := menu.Selection(); selected {
		t.Fatal("menu selected a mode before Enter")
	}
	for i, want := range []tutorial.Mode{{}, {Practice: true}, {Markup: true}, {Markup: true, Practice: true}} {
		t.Run(string(rune('1'+i)), func(t *testing.T) {
			menu := tutorial.NewStartMenu()
			for j := 0; j < i; j++ {
				menu.Update(menuKey('j', "j"))
			}
			_, cmd := menu.Update(menuKey(tea.KeyEnter, ""))
			got, selected := menu.Selection()
			if !selected || got != want {
				t.Fatalf("selection = %+v, %v; want %+v", got, selected, want)
			}
			if cmd == nil {
				t.Fatal("selection did not finish menu")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatal("selection did not quit menu program")
			}
		})
	}
}

func TestStartMenuNavigationAndCancellation(t *testing.T) {
	for _, code := range []rune{'q', tea.KeyEscape, 'c', tea.KeyEnter} {
		t.Run(string(code), func(t *testing.T) {
			menu := tutorial.NewStartMenu()
			key := menuKey(code, "")
			if code == 'q' {
				key = menuKey('q', "q")
			}
			if code == 'c' {
				key = tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl})
			}
			if code == tea.KeyEnter {
				for i := 0; i < 10; i++ {
					menu.Update(menuKey(tea.KeyDown, ""))
				}
				if !strings.Contains(menu.View().Content, "> Quit") {
					t.Fatal("down should stop at Quit")
				}
			}
			_, cmd := menu.Update(key)
			if _, selected := menu.Selection(); selected {
				t.Fatal("cancellation selected a tutorial")
			}
			if cmd == nil {
				t.Fatal("cancellation did not exit")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatal("cancellation did not quit menu program")
			}
		})
	}
	menu := tutorial.NewStartMenu()
	menu.Update(menuKey(tea.KeyDown, ""))
	menu.Update(menuKey(tea.KeyDown, ""))
	menu.Update(menuKey(tea.KeyUp, ""))
	if !strings.Contains(menu.View().Content, "> Practice the basics") {
		t.Fatal("arrow navigation did not move selection")
	}
	for i := 0; i < 10; i++ {
		menu.Update(menuKey('k', "k"))
	}
	menu.Update(menuKey('x', "x"))
	menu.Update(tea.KeyReleaseMsg(tea.Key{Code: 'j', Text: "j"}))
	if !strings.Contains(menu.View().Content, "> Learn the basics") {
		t.Fatal("up should stop at first option; unrelated/released keys should be ignored")
	}
}
