package tutorial

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"github.com/JanWartenberg/vimkunja/internal/tui"
	"strings"
	"testing"
	"time"
)

func TestMarkupTutorialRunsRealUIAndSavesDisposableData(t *testing.T) {
	for _, alias := range []string{"ctrl+q", "ctrl+b"} {
		t.Run(alias, func(t *testing.T) { runMarkupTutorial(t, alias) })
	}
}
func runMarkupTutorial(t *testing.T, alias string) {
	db, err := NewMarkupDemoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m := NewMarkupModel(tui.NewModel(tui.Config{Store: db.Store, PerPage: 100, LeaderTimeout: time.Millisecond, CtrlWTimeout: time.Millisecond, LiteralTabAlias: alias}), alias).(*Model)
	s := &practiceSession{t: t, db: db, app: m}
	s.run(m.Init())
	s.message(tea.WindowSizeMsg{Width: 120, Height: 40})
	for _, step := range m.steps {
		var values []string
		for _, key := range step.keys {
			values = append(values, key.value)
		}
		if step.title == "Link destination" {
			s.keys("v", "i", "w", "space", "l")
			s.text("https://my.example/other")
			s.keys("enter", "esc")
		} else if step.title == "Recover readable text" {
			s.keys(":")
			s.text("descriptionrec")
			s.keys("tab", "enter", "y")
		} else {
			s.keys(values...)
		}
		// Expected keys alone do not prove the application performed the action.
		if step.title == "Discard recovery" && !strings.Contains(s.screen(), "Malformed HTML") {
			t.Fatal("recovery discard did not restore the blocked Description")
		}
		if step.title == "Clear blocked draft" && strings.Contains(s.screen(), "Malformed HTML") {
			t.Fatal("clear command was not applied")
		}
	}
	if m.step != len(m.steps) || !s.quit {
		t.Fatalf("lesson did not finish and quit: step %d feedback %s", m.step, m.feedback)
	}
	tasks, _, err := db.Store.ListTasksInView(context.Background(), 1, 11, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	task := tasks[len(tasks)-1]
	for _, want := range []string{`<strong>Alpha</strong>`, `href="https://my.example/other"`, `<code>print(1)`, `<mention-user data-id="jan" data-label="Jan"></mention-user>`} {
		if !strings.Contains(task.Description, want) {
			t.Errorf("missing %q in %s", want, task.Description)
		}
	}
	if tasks[len(tasks)-2].Description != `<p>Broken<strong>` {
		t.Fatal("discarded recovery changed original blocked source")
	}
	if task.Done {
		t.Fatal("Description lesson changed Task Done")
	}
	if strings.Contains(task.Description, "oops") {
		t.Fatal("discarded text persisted")
	}
}

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

func TestMarkupLinkInputAllowsCorrectionsPasteAndInvalidRetry(t *testing.T) {
	s := newPracticeSession(t)
	m := NewMarkupModel(s.app, "ctrl+b").(*Model)
	m.step = 5
	s.app = m
	s.keys("v", "i", "w", "space", "l")
	s.text("javascript:bad")
	s.keys("enter")
	if m.step != 5 || !m.app.(interface{ DescriptionLinkPromptOpen() bool }).DescriptionLinkPromptOpen() {
		t.Fatalf("invalid URL must keep the lesson and prompt open: step=%d key=%d feedback=%s screen=%s", m.step, m.key, m.feedback, s.screen())
	}
	for range len("javascript:bad") {
		s.message(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	}
	s.message(tea.PasteMsg{Content: "https://custom.example/pasted"})
	s.keys("enter", "esc")
	if m.step != 6 {
		t.Fatalf("free URL did not finish Link step: step=%d feedback=%s", m.step, m.feedback)
	}
	// Disable guidance only after completing the step to inspect a real save.
	s.app = m.app
	s.wantSaved(`<p><a href="https://custom.example/pasted" target="_blank">Alpha</a> beta</p>` + practiceTail)
}

func TestMarkupLinkInputEmptyAndCancel(t *testing.T) {
	for _, finish := range []string{"enter", "esc"} {
		t.Run(finish, func(t *testing.T) {
			s := newPracticeSession(t)
			m := NewMarkupModel(s.app, "ctrl+b").(*Model)
			m.step = 5
			s.app = m
			s.keys("v", "i", "w", "space", "l", finish, "esc")
			if m.step != 6 {
				t.Fatalf("link prompt did not finish: step %d", m.step)
			}
			s.app = m.app
			s.wantSaved(practiceSource)
		})
	}
}

func TestTutorialCommandCompletionTracksRealPrompt(t *testing.T) {
	s := newPracticeSession(t)
	m := &Model{app: s.app, steps: []tutorialStep{actionStep("Recover", "", "", commandKeys("descriptionrecover")...)}}
	s.app = m
	s.keys(":")
	s.text("descriptionre")
	before := m.key
	s.keys("tab")
	if m.key != before {
		t.Fatal("ambiguous completion advanced lesson")
	}
	s.keys("c", "tab")
	if m.key != len(m.steps[0].keys)-1 {
		t.Fatalf("completion did not reach Enter: key=%d feedback=%s", m.key, m.feedback)
	}
	if !m.app.(interface{ CommandInputMatches(string) bool }).CommandInputMatches("descriptionrecover") {
		t.Fatal("real command prompt was not completed")
	}
}

func TestTutorialCommandCompletionAcceptsExpandedAlias(t *testing.T) {
	s := newPracticeSession(t)
	m := &Model{app: s.app, steps: []tutorialStep{actionStep("Help", "", "", commandKeys("h")...)}}
	s.app = m
	s.keys(":", "h", "tab", "enter")
	if m.step != 1 || !strings.Contains(s.screen(), "HELP") {
		t.Fatalf("expanded help alias failed: step=%d feedback=%s", m.step, m.feedback)
	}
}
