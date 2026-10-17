package tutorial

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/tui"
)

func newTaskSession(t *testing.T) (*TaskModel, *practiceSession) {
	t.Helper()
	m, err := NewTaskModel(tui.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	s := &practiceSession{t: t, app: m}
	s.run(m.Init())
	s.message(tea.WindowSizeMsg{Width: 100, Height: 35})
	s.keys("enter")
	return m, s
}

func TestTaskLessonChecksResultWithFreeInputAndExplicitAdvance(t *testing.T) {
	m, s := newTaskSession(t)
	s.keys("f8")
	if strings.Contains(s.screen(), "TUTORIAL COMPLETE") {
		t.Fatal("advanced without success")
	}
	s.keys("j", "k", "enter", "enter", "0", "C")
	s.message(tea.PasteMsg{Content: "foox"})
	s.keys("backspace")
	if strings.Contains(s.screen(), "Goal reached") {
		t.Fatal("success interrupted Insert mode")
	}
	s.keys("esc")
	if !strings.Contains(s.screen(), "Goal reached") {
		t.Fatalf("result was not recognized: %s", s.screen())
	}
	s.keys("i", "x", "esc")
	if strings.Contains(s.screen(), "TUTORIAL COMPLETE") {
		t.Fatal("advanced automatically")
	}
	s.keys("f8")
	if m.lesson != 1 || !strings.Contains(s.screen(), "Enter: start") {
		t.Fatal("explicit advance did not start next task")
	}
}

func TestTaskLessonResetRestoresSavedAndDeletedData(t *testing.T) {
	m, s := newTaskSession(t)
	oldPath := m.database.Path
	s.keys("enter", "enter", "0", "C")
	s.text("foo")
	s.keys("esc")
	s.command("wq")
	s.keys("d", "d", "y")
	s.keys("f5")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old database retained: %v", err)
	}
	if strings.Contains(s.screen(), "Goal reached") {
		t.Fatal("reset retained completion")
	}
	s.keys("enter", "enter")
	if !strings.Contains(s.screen(), "Practice the Vimkunja tutorial") {
		t.Fatalf("reset did not restore Task: %s", s.screen())
	}
	s.keys("0", "C")
	s.text("foo")
	s.keys("esc")
	if !strings.Contains(s.screen(), "Goal reached") {
		t.Fatal("reset lesson cannot be completed again")
	}
}

func TestTaskLessonResetIgnoresPendingQuitAndCleansDatabase(t *testing.T) {
	m, s := newTaskSession(t)
	s.keys(":", "q", "a")
	_, pendingQuit := m.Update(markupKey("enter"))
	s.keys("f5")
	s.run(pendingQuit)
	if s.quit {
		t.Fatal("old session's Quit terminated reset lesson")
	}
	path := m.database.Path
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("database not removed: %v", err)
	}
}

func TestTaskLessonHintAndResetWorkDuringInsertAtNarrowWidth(t *testing.T) {
	_, s := newTaskSession(t)
	s.message(tea.WindowSizeMsg{Width: 54, Height: 24})
	s.keys("f1")
	if !strings.Contains(s.screen(), "From reset") {
		t.Fatal("help is unavailable")
	}
	s.keys("enter", "enter", "enter", "0", "C")
	s.text("unfinished")
	s.keys("f5", "enter", "enter")
	if strings.Contains(s.screen(), "unfinished") || strings.Contains(s.screen(), "From reset") {
		t.Fatal("reset retained draft/help")
	}
	if !strings.Contains(s.screen(), "Rename Task") || !strings.Contains(s.screen(), "F5 reset") {
		t.Fatalf("task/controls missing: %s", s.screen())
	}
}

func TestTaskLessonDoesNotAcceptMatchingTitleOnAnotherTask(t *testing.T) {
	_, s := newTaskSession(t)
	s.keys("enter", "j", "enter", "0", "C")
	s.text("foo")
	s.keys("esc")
	if strings.Contains(s.screen(), "Goal reached") {
		t.Fatal("wrong Task counted as success")
	}
}

func TestTaskLessonAcceptsInlineTitleEditing(t *testing.T) {
	_, s := newTaskSession(t)
	s.keys("enter", "I", "end")
	for range []rune("Practice the Vimkunja tutorial") {
		s.keys("backspace")
	}
	s.text("foo")
	if strings.Contains(s.screen(), "Goal reached") {
		t.Fatal("unfinished inline edit counted")
	}
	s.keys("enter")
	if !strings.Contains(s.screen(), "Goal reached") {
		t.Fatalf("inline result not accepted: %s", s.screen())
	}
}

func TestTaskLessonContextRequiresConfirmationAndKeepsStripCompact(t *testing.T) {
	m, err := NewTaskModel(tui.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	s := &practiceSession{t: t, app: m}
	s.run(m.Init())
	s.message(tea.WindowSizeMsg{Width: 100, Height: 35})
	if !strings.Contains(s.screen(), "Enter: start") {
		t.Fatal("context modal missing")
	}
	s.keys("enter")
	if strings.Contains(s.screen(), "Enter: start") {
		t.Fatal("confirmation did not dismiss context")
	}
	if got := strings.Count(m.panel(), "\n") + 1; got != 2 {
		t.Fatalf("strip has %d lines, want 2", got)
	}
	s.keys("enter", "enter")
	if !strings.Contains(s.screen(), "TASK EDITOR") {
		t.Fatal("confirmation reached the application")
	}
	s.keys("f1")
	s.message(tea.PasteMsg{Content: "unexpected"})
	s.keys("esc")
	if strings.Contains(s.screen(), "unexpected") {
		t.Fatal("modal input reached application")
	}
}

func TestF12SkipsSevenTasksIncludingContextModalsAndIgnoresStaleResults(t *testing.T) {
	for _, markup := range []bool{false, true} {
		t.Run(map[bool]string{false: "basics", true: "markup"}[markup], func(t *testing.T) {
			m, err := NewCourseModel(tui.Config{}, markup)
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			s := &practiceSession{t: t, app: m}
			pending := []tea.Cmd{m.Init()}
			s.message(tea.WindowSizeMsg{Width: 54, Height: 24})
			for range 7 {
				_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyF12}))
				pending = append(pending, cmd)
				m.Update(tea.KeyReleaseMsg(tea.Key{Code: tea.KeyF12}))
			}
			// Results of earlier initializations can arrive after the seventh skip.
			for i := len(pending) - 1; i >= 0; i-- {
				s.run(pending[i])
			}
			s.keys("enter")
			if m.lesson != 7 || m.achieved || !strings.Contains(s.screen(), "8/") {
				t.Fatalf("seven skips did not reach fresh task8: %s", s.screen())
			}
			s.keys("f8")
			if m.lesson != 7 {
				t.Fatal("skipped tasks counted as a passed current task")
			}
		})
	}
}

func TestF12WorksFromHelpAndReportsSkippedTasksAtEnd(t *testing.T) {
	m, s := newTaskSession(t)
	s.keys("f1")
	s.message(tea.KeyPressMsg(tea.Key{Code: tea.KeyF12}))
	if m.lesson != 1 {
		t.Fatal("F12 did not skip from Help modal")
	}
	for range len(m.lessons) - 1 {
		s.message(tea.KeyPressMsg(tea.Key{Code: tea.KeyF12}))
	}
	if !m.finished || !strings.Contains(s.screen(), "26 skipped") || strings.Contains(s.screen(), "TUTORIAL COMPLETE") {
		t.Fatalf("skipped course reported as passed: %s", s.screen())
	}
	s.message(tea.KeyPressMsg(tea.Key{Code: tea.KeyF12}))
	s.keys("f5")
	if m.finished || m.lesson != len(m.lessons)-1 {
		t.Fatal("F5 did not repeat last skipped task")
	}
	s.keys("space", "f")
	s.text("Buy")
	s.keys("enter", "f8")
	if !strings.Contains(s.screen(), "25 skipped") {
		t.Fatalf("passed repeat still counted as skipped: %s", s.screen())
	}
}

func TestF12DiscardsAnUnfinishedDraft(t *testing.T) {
	m, s := newTaskSession(t)
	oldPath := m.database.Path
	s.keys("enter", "enter", "i")
	s.text("unfinished")
	s.message(tea.KeyPressMsg(tea.Key{Code: tea.KeyF12}))
	s.keys("enter")
	if m.lesson != 1 || strings.Contains(s.screen(), "unfinished") || m.snapshot().Dirty {
		t.Fatal("skip retained the previous draft")
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("skipped task's database retained: %v", err)
	}
}
