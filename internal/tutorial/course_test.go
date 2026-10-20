package tutorial

import (
	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/tui"
	"strings"
	"testing"
)

func TestGeneralCourseAdvancesOnlyAfterRealResultAndStartsFreshTask(t *testing.T) {
	m, err := NewCourseModel(tui.Config{}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	s := &practiceSession{t: t, app: m}
	s.run(m.Init())
	s.message(tea.WindowSizeMsg{Width: 100, Height: 35})
	s.keys("enter")
	s.keys("f8")
	if m.lesson != 0 {
		t.Fatal("advanced before result")
	}
	s.keys("enter", "I", "end")
	for range []rune("Practice the Vimkunja tutorial") {
		s.keys("backspace")
	}
	s.text("foo")
	s.keys("enter")
	if !strings.Contains(s.screen(), "Goal reached") {
		t.Fatalf("inline result not recognized: %+v %s", m.snapshot(), s.screen())
	}
	oldPath := m.database.Path
	s.keys("f8")
	if m.lesson != 1 || m.database.Path == oldPath {
		t.Fatal("F8 did not start independent next task")
	}
	s.keys("enter", "f5")
	if m.lesson != 1 {
		t.Fatal("F5 moved away from current task")
	}
}

func runCourse(t *testing.T, markup bool, alias string, actions []func(*practiceSession)) {
	t.Helper()
	m, err := NewCourseModel(tui.Config{LiteralTabAlias: alias}, markup)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	s := &practiceSession{t: t, app: m}
	s.run(m.Init())
	s.message(tea.WindowSizeMsg{Width: 120, Height: 40})
	if len(actions) != len(m.lessons) {
		t.Fatalf("%d journeys for %d tasks", len(actions), len(m.lessons))
	}
	for i, action := range actions {
		t.Logf("task %d: %s", i+1, m.lessons[i].topic)
		if m.lesson != i {
			t.Fatalf("unexpected chapter %d", m.lesson)
		}
		s.keys("enter")
		if len(m.setup) != 0 {
			t.Fatalf("setup stalled: %+v %s", m.snapshot(), s.screen())
		}
		if m.achieved {
			t.Fatalf("task completed from setup alone: %s", m.lessons[i].topic)
		}
		s.keys("f8")
		if m.lesson != i {
			t.Fatal("F8 skipped incomplete task")
		}
		s.keys("f5")
		action(s)
		if !m.achieved {
			if m.snapshot().Draft != nil {
				t.Logf("draft=%q", m.snapshot().Draft.Description)
			}
			t.Fatalf("task %d (%s) not recognized, stage %d: %+v\n%s", i+1, m.lessons[i].topic, m.stage, m.snapshot(), s.screen())
		}
		s.keys("f8")
	}
	if !m.finished || !strings.Contains(s.screen(), "TUTORIAL COMPLETE") {
		t.Fatal("course did not finish explicitly")
	}
	s.keys("ctrl+c")
	if !s.quit {
		t.Fatal("completed course cannot exit")
	}
}

func TestAllBasicsTasksThroughRealApplication(t *testing.T) {
	runCourse(t, false, "ctrl+b", []func(*practiceSession){
		func(s *practiceSession) {
			s.keys("enter", "enter", "0", "C")
			s.message(tea.PasteMsg{Content: "foox"})
			s.keys("backspace", "esc")
		},
		func(s *practiceSession) { s.keys("j", "enter") },
		func(s *practiceSession) { s.keys("end") },
		func(s *practiceSession) { s.keys("enter") },
		func(s *practiceSession) { s.keys("g", "p", "j", "j", "enter") },
		func(s *practiceSession) { s.keys("g", "d"); s.text("20300115"); s.keys("enter") },
		func(s *practiceSession) { s.keys("g", "p", "enter", "j", "j", "j", "i", "j", "enter") },
		func(s *practiceSession) { s.keys("x") },
		func(s *practiceSession) { s.keys("I"); s.text("Tutorial: "); s.keys("esc") },
		func(s *practiceSession) {
			s.keys("0", "C")
			s.text("Saved in tutorial.")
			s.keys("esc")
			s.command("write")
		},
		func(s *practiceSession) { s.keys("ctrl+w", "k") },
		func(s *practiceSession) { s.command("quit") },
		func(s *practiceSession) { s.command("q"); s.command("e!") },
		func(s *practiceSession) {
			s.keys("I", "end")
			for range []rune("Practice the Vimkunja tutorial") {
				s.keys("backspace")
			}
			s.text("A tutorial task")
			s.keys("enter")
			if strings.Contains(s.screen(), "Goal reached") {
				s.t.Fatal("renaming counted as creation")
			}
			s.keys("f5")
			s.command("new")
			s.keys("i")
			s.text("A tutorial task")
			s.keys("esc")
			s.command("wq")
		},
		func(s *practiceSession) {
			s.keys("f", "v")
			if strings.Contains(s.screen(), "Goal reached") {
				s.t.Fatal("filtering counted as deletion")
			}
			s.keys("f", "a", "enter", "g", "p", "enter", "k", "k", "i", "j", "enter")
			s.command("wq")
			s.keys("ctrl+w", "h", "g", "g", "enter")
			if strings.Contains(s.screen(), "Goal reached") {
				s.t.Fatal("moving counted as deletion")
			}
			s.keys("f5", "d", "d", "n", "d", "d", "y")
		},
		func(s *practiceSession) { s.keys("x", "u", "ctrl+r") },
		func(s *practiceSession) { s.command("overview") },
		func(s *practiceSession) { s.keys("s", "p", "f", "a") },
		func(s *practiceSession) { s.keys("space", "o") },
		func(s *practiceSession) { s.command("kanban"); s.keys("L") },
		func(s *practiceSession) { s.command("kanban"); s.keys("n"); s.command("kanban"); s.keys("y") },
		func(s *practiceSession) { s.command("help"); s.keys("ctrl+d") },
		func(s *practiceSession) { s.command("bogus"); s.command("errors") },
		func(s *practiceSession) { s.keys("g", "l", "enter", "esc", "q") },
		func(s *practiceSession) {
			s.keys("g", "c", "a")
			s.text("Learning works")
			s.keys("esc")
			s.command("w")
			s.command("q")
		},
		func(s *practiceSession) { s.keys("space", "f"); s.text("Buy"); s.keys("enter") },
	})
}

func TestAllMarkupTasksThroughRealApplication(t *testing.T) {
	for _, alias := range []string{"ctrl+q", "ctrl+b"} {
		t.Run(alias, func(t *testing.T) {
			runCourse(t, true, alias, []func(*practiceSession){
				func(s *practiceSession) { s.keys("G", "enter", "g", "n", "enter") },
				func(s *practiceSession) { s.keys("c", "i", "w"); s.text("Hallo"); s.keys("esc") },
				func(s *practiceSession) { s.keys("v", "i", "w", "space", "b", "esc") },
				func(s *practiceSession) { s.keys("u", "ctrl+r") },
				func(s *practiceSession) {
					s.keys("v", "i", "w", "space", "l")
					s.text("javascript:bad")
					s.keys("enter")
					for range len("javascript:bad") {
						s.keys("backspace")
					}
					s.message(tea.PasteMsg{Content: "https://custom.example/pasted"})
					s.keys("enter", "esc")
				},
				func(s *practiceSession) {
					s.keys("o")
					s.text("**Both**")
					s.keys("esc", "o")
					s.text("*Both*")
					s.keys("esc")
					if strings.Contains(s.screen(), "Goal reached") {
						s.t.Fatal("separate bold and italic words counted as combined marks")
					}
					s.keys("f5", "o")
					s.text("***Both***")
					s.keys("esc")
				},
				func(s *practiceSession) { s.keys("V", "space", "x", "esc", "x") },
				func(s *practiceSession) { s.keys("space", "x") },
				func(s *practiceSession) { s.keys("space", "q") },
				func(s *practiceSession) { s.keys("space", "q") },
				func(s *practiceSession) {
					s.keys("i")
					s.text("``````")
					for range 3 {
						s.message(tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft}))
					}
					s.keys("enter", "esc")
				},
				func(s *practiceSession) { s.keys("i"); s.text("print(1)"); s.keys("enter", alias, "tab", "esc") },
				func(s *practiceSession) { s.command("retab") },
				func(s *practiceSession) { s.keys("ü"); s.text("After code"); s.keys("esc") },
				func(s *practiceSession) { s.keys("x", "n") },
				func(s *practiceSession) { s.keys("x", "y", "u") },
				func(s *practiceSession) { s.keys("v", "i", "w", "space", "b", "esc"); s.command("w") },
				func(s *practiceSession) { s.command("e!") },
				func(s *practiceSession) { s.keys(":"); s.text("descriptionrec"); s.keys("tab", "enter", "y", "esc") },
				func(s *practiceSession) { s.command("e!") },
				func(s *practiceSession) { s.command("descriptionclear"); s.keys("y", "esc") },
				func(s *practiceSession) { s.command("q!") },
			})
		})
	}
}

func TestMarkupCourseDefaultsLiteralTabPrefix(t *testing.T) {
	m, err := NewCourseModel(tui.Config{}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	s := &practiceSession{t: t, app: m}
	s.run(m.Init())
	s.message(tea.WindowSizeMsg{Width: 120, Height: 40})
	s.keys("enter", "G", "enter", "g", "n", "enter", "g", "g", "o", "esc", "space", "c")
	s.text("a")
	s.keys("enter", "ctrl+q", "tab", "esc")
	if m.snapshot().Draft == nil || !markedText(m.snapshot().Draft.Description, "code", "a\n\t", "", "") {
		t.Fatal("documented default literal Tab prefix is disabled")
	}
}
