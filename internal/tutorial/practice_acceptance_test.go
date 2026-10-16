package tutorial

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/tui"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

const practiceTail = `<pre><code>original()</code></pre><p><mention-user data-id="jan" data-label="Jan"></mention-user></p>`
const practiceSource = `<p>Alpha beta</p>` + practiceTail

// Exercise the unrestricted application from startup, including loading the
// actual fixture Task. No pre-opened editor or direct draft/cursor mutation.
type practiceSession struct {
	t       *testing.T
	app     tea.Model
	db      *DemoDatabase
	delayed []tea.Cmd
	quit    bool
}

func newPracticeSession(t *testing.T) *practiceSession {
	t.Helper()
	db, err := NewMarkupDemoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	s := &practiceSession{t: t, db: db, app: tui.NewModel(tui.Config{Store: db.Store, PerPage: 100, LiteralTabAlias: "ctrl+b", LeaderTimeout: time.Millisecond, CtrlWTimeout: time.Millisecond})}
	s.run(s.app.Init())
	s.message(tea.WindowSizeMsg{Width: 120, Height: 40})
	s.keys("enter", "G", "enter", "g", "n", "enter", "g", "g")
	if !strings.Contains(s.screen(), "Alpha beta") {
		t.Fatal("practice entry did not reach Alpha beta")
	}
	return s
}
func (s *practiceSession) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if msg != nil && reflect.TypeOf(msg).Name() == "cursorBlinkMsg" {
		return
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, child := range batch {
			s.run(child)
		}
		return
	}
	if _, ok := msg.(tea.QuitMsg); ok {
		s.quit = true
		return
	}
	s.message(msg)
}
func (s *practiceSession) message(msg tea.Msg) {
	var cmd tea.Cmd
	s.app, cmd = s.app.Update(msg)
	s.run(cmd)
}
func (s *practiceSession) keys(keys ...string) {
	for _, key := range keys {
		var cmd tea.Cmd
		s.app, cmd = s.app.Update(markupKey(key))
		if key == "space" || key == "ctrl+w" {
			s.delayed = append(s.delayed, cmd)
			continue
		}
		s.run(cmd)
		pending := s.delayed
		s.delayed = nil
		for _, timer := range pending {
			s.run(timer)
		}
	}
}
func (s *practiceSession) text(value string) {
	for _, r := range value {
		if r == ' ' {
			s.keys("space")
		} else {
			s.keys(string(r))
		}
	}
}
func (s *practiceSession) command(value string) { s.keys(":"); s.text(value); s.keys("enter") }
func (s *practiceSession) screen() string       { return ansi.Strip(s.app.View().Content) }
func (s *practiceSession) task(id int64) vimkunja.Task {
	s.t.Helper()
	tasks, _, err := s.db.Store.ListTasksInView(context.Background(), 1, 11, 1, 100)
	if err != nil {
		s.t.Fatal(err)
	}
	for _, task := range tasks {
		if task.ID == id {
			return task
		}
	}
	s.t.Fatalf("Task %d missing", id)
	return vimkunja.Task{}
}
func (s *practiceSession) saved() string {
	s.command("w")
	task := s.task(108)
	if task.Done {
		s.t.Fatal("Description action changed Task Done")
	}
	return task.Description
}
func (s *practiceSession) wantSaved(want string) {
	s.t.Helper()
	if got := s.saved(); got != want {
		s.t.Fatalf("got %s\nwant %s\n%s", got, want, s.screen())
	}
}
func (s *practiceSession) newCode() { s.keys("g", "g", "o", "esc", "space", "c") }

func TestMarkupPracticeAcceptanceCards(t *testing.T) {
	t.Run("AT-02 bold twice", func(t *testing.T) {
		s := newPracticeSession(t)
		s.keys("v", "i", "w", "space", "b", "esc")
		s.wantSaved(`<p><strong>Alpha</strong> beta</p>` + practiceTail)
		s.keys("v", "i", "w", "space", "b", "esc")
		s.wantSaved(practiceSource)
	})
	t.Run("AT-03 link", func(t *testing.T) {
		s := newPracticeSession(t)
		s.keys("v", "i", "w", "space", "l")
		s.text("https://example.test/review")
		s.keys("enter", "esc")
		s.wantSaved(`<p><a href="https://example.test/review" target="_blank">Alpha</a> beta</p>` + practiceTail)
		// System clipboard results have separate injected-writer UI tests; this
		// headless journey must not replace the user's real clipboard.
	})
	t.Run("AT-04 checkbox", func(t *testing.T) {
		s := newPracticeSession(t)
		s.keys("space", "x", "x")
		got := s.saved()
		if !strings.Contains(got, `data-checked="true"`) || !strings.Contains(got, `data-task-id="`) || !strings.Contains(got, `<p>Alpha beta</p>`) {
			t.Fatal(got)
		}
		s.command("q")
		if s.task(108).Done {
			t.Fatal("task Done changed")
		}
	})
	t.Run("AT-05 list indent outdent", func(t *testing.T) {
		s := newPracticeSession(t)
		s.keys("space", "u", "o")
		s.text("Zwei")
		s.keys("esc", "V", "space", "l")
		s.wantSaved(`<ul><li><p>Alpha beta</p><ul><li>Zwei</li></ul></li></ul>` + practiceTail)
		s.keys("space", "k", "esc")
		s.wantSaved(`<ul><li><p>Alpha beta</p></li><li>Zwei</li></ul>` + practiceTail)
	})
	t.Run("AT-06 quote", func(t *testing.T) {
		s := newPracticeSession(t)
		s.keys("space", "q")
		s.wantSaved(`<blockquote><p>Alpha beta</p></blockquote>` + practiceTail)
		s.keys("space", "q")
		s.wantSaved(practiceSource)
	})
	t.Run("AT-07 protected mention", func(t *testing.T) {
		s := newPracticeSession(t)
		s.keys("G", "x", "n")
		s.wantSaved(practiceSource)
		s.keys("x", "y")
		got := s.saved()
		if strings.Contains(got, "mention-user") || !strings.HasPrefix(got, `<p>Alpha beta</p><pre><code>original()</code></pre>`) {
			t.Fatal(got)
		}
		s.keys("u")
		s.wantSaved(practiceSource)
	})
	t.Run("AT-08 direct code", func(t *testing.T) {
		s := newPracticeSession(t)
		s.newCode()
		s.text("print(1)")
		s.keys("esc")
		s.wantSaved(`<p>Alpha beta</p><pre><code>print(1)</code></pre>` + practiceTail)
	})
	for _, marker := range []string{"```", "````"} {
		t.Run("AT-08 marker "+marker, func(t *testing.T) {
			s := newPracticeSession(t)
			s.keys("g", "g", "o")
			s.text(marker)
			if !strings.Contains(s.screen(), marker) {
				t.Fatal("unconfirmed marker not visible")
			}
			s.keys("enter")
			s.text("print(1)")
			s.keys("esc")
			s.wantSaved(`<p>Alpha beta</p><pre><code>print(1)</code></pre>` + practiceTail)
		})
	}
	t.Run("AT-09 code lines", func(t *testing.T) {
		s := newPracticeSession(t)
		s.keys("g", "g", "j", "o")
		s.text("unten")
		s.keys("esc", "O")
		s.text("oben")
		s.keys("esc")
		s.wantSaved("<p>Alpha beta</p><pre><code>original()\noben\nunten</code></pre><p><mention-user data-id=\"jan\" data-label=\"Jan\"></mention-user></p>")
	})
	for _, key := range []string{"ü", "Ü", "]", "["} {
		t.Run("AT-10 exit "+key, func(t *testing.T) {
			s := newPracticeSession(t)
			s.keys("g", "g", "j", key)
			if key == "]" || key == "[" {
				s.keys("space", "i")
			}
			s.text("Absatz")
			s.keys("esc")
			want := `<p>Alpha beta</p><p>Absatz</p>` + practiceTail
			if key == "ü" || key == "]" {
				want = `<p>Alpha beta</p><pre><code>original()</code></pre><p>Absatz</p><p><mention-user data-id="jan" data-label="Jan"></mention-user></p>`
			}
			s.wantSaved(want)
		})
	}
	t.Run("AT-11 tabs retab undo", func(t *testing.T) {
		s := newPracticeSession(t)
		s.newCode()
		s.text("ab")
		s.keys("tab")
		s.text("X")
		s.keys("esc", "o")
		s.text("ab")
		s.keys("ctrl+b", "tab")
		s.text("X")
		s.keys("esc")
		literal := "<p>Alpha beta</p><pre><code>ab  X\nab\tX</code></pre>" + practiceTail
		s.wantSaved(literal)
		s.command("retab")
		s.wantSaved("<p>Alpha beta</p><pre><code>ab  X\nab  X</code></pre>" + practiceTail)
		s.keys("u")
		s.wantSaved(literal)
	})
	t.Run("AT-12 code paste", func(t *testing.T) {
		s := newPracticeSession(t)
		s.newCode()
		s.message(tea.PasteMsg{Content: "print(1)\n\tpass"})
		s.keys("esc")
		s.wantSaved("<p>Alpha beta</p><pre><code>print(1)\n\tpass</code></pre>" + practiceTail)
	})
	for _, paste := range []bool{false, true} {
		name := "typed"
		if paste {
			name = "paste"
		}
		t.Run("AT-12 syntax "+name, func(t *testing.T) {
			s := newPracticeSession(t)
			s.keys("g", "g", "o")
			if paste {
				s.message(tea.PasteMsg{Content: "***Hallo***"})
			} else {
				s.text("***Hallo***")
			}
			s.keys("esc")
			want := `<p>Alpha beta</p><p><strong><em>Hallo</em></strong></p>` + practiceTail
			if paste {
				want = `<p>Alpha beta</p><p>***Hallo***</p>` + practiceTail
			}
			s.wantSaved(want)
		})
	}
	t.Run("AT-13 save reopen", func(t *testing.T) {
		s := newPracticeSession(t)
		s.keys("c", "i", "w")
		s.text("Gespeichert")
		s.keys("esc")
		s.wantSaved(`<p>Gespeichert beta</p>` + practiceTail)
		s.command("q")
		s.keys("enter", "g", "n", "enter")
		if !strings.Contains(s.screen(), "Gespeichert beta") {
			t.Fatal(s.screen())
		}
	})
	t.Run("AT-14 discard", func(t *testing.T) {
		s := newPracticeSession(t)
		s.keys("c", "i", "w")
		s.text("Weg damit")
		s.keys("esc")
		s.command("e!")
		if !strings.Contains(s.screen(), "Alpha beta") || s.task(108).Description != practiceSource {
			t.Fatal(s.screen())
		}
	})
	t.Run("AT-15 recovery exact card", func(t *testing.T) {
		s := newPracticeSession(t)
		s.command("q")
		s.keys("k", "enter")
		s.command("descriptionrecover")
		s.keys("y")
		s.text("!")
		s.keys("esc")
		if !strings.Contains(s.screen(), "!Broken") || strings.Contains(s.screen(), "i!Broken") {
			t.Fatal(s.screen())
		}
		s.command("e!")
		if !strings.Contains(s.screen(), "Malformed HTML") {
			t.Fatal("discard did not restore Blocked Description")
		}
		s.command("descriptionclear")
		s.keys("y")
		if strings.Contains(s.screen(), "Malformed HTML") || strings.Contains(s.screen(), "Broken") {
			t.Fatal("clear did not create an empty editable draft")
		}
		s.command("e!")
		if !strings.Contains(s.screen(), "Malformed HTML") {
			t.Fatal("clear discard did not restore Blocked Description")
		}
		if s.task(107).Description != `<p>Broken<strong>` {
			t.Fatal("recovery wrote before save")
		}
	})
	t.Run("AT-16 resize palette", func(t *testing.T) {
		s := newPracticeSession(t)
		s.keys("i")
		s.text("Test ")
		s.keys("esc")
		for _, size := range [][2]int{{54, 12}, {20, 6}, {120, 40}} {
			s.message(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			s.keys("space", "esc")
		}
		s.wantSaved(`<p>Test Alpha beta</p>` + practiceTail)
	})
}
