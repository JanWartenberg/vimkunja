package tutorial

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/JanWartenberg/vimkunja/internal/tui"
	"github.com/charmbracelet/x/ansi"
)

// TaskModel runs result-based lessons. Ordinary input always reaches the
// real application; only the advertised tutorial function keys are reserved.
type TaskModel struct {
	app                tea.Model
	database           *DemoDatabase
	config             tui.Config
	generation         uint64
	width, height      int
	achieved, finished bool
	modal              string
	lessons            []lesson
	lesson, stage      int
	setup              []string
	markup             bool
	feedback           string
	skipped            map[int]bool
	closeErr           error
}

type taskAppMsg struct {
	generation uint64
	message    tea.Msg
}

// NewTaskModel owns fresh disposable data, including after every reset.
// A configured Store is deliberately ignored.
func NewTaskModel(config tui.Config) (*TaskModel, error) { return NewCourseModel(config, false) }

// NewCourseModel starts a result-based basics or markup course. Every task owns
// an independent starting fixture, so exploration cannot break later lessons.
func NewCourseModel(config tui.Config, markup bool) (*TaskModel, error) {
	if markup && config.LiteralTabAlias == "" {
		config.LiteralTabAlias = "ctrl+q"
	}
	lessons := basicLessons()
	if markup {
		lessons = markupLessons(config.LiteralTabAlias)
	}
	m := &TaskModel{config: config, markup: markup, lessons: lessons, skipped: make(map[int]bool), width: 80, height: 24}
	if err := m.restart(); err != nil {
		return nil, err
	}
	m.modal = "intro"
	return m, nil
}

func (m *TaskModel) restart() error {
	database, err := newLessonDatabase(m.markup, m.lessons[m.lesson])
	if err != nil {
		return err
	}
	old := m.database
	config := m.config
	config.Store, config.PerPage = database.Store, 100
	config.InitialProjectID, config.InitialViewID = m.lessons[m.lesson].project, m.lessons[m.lesson].view
	if m.markup {
		config.Tabstop, config.IndentStep = 4, 4
	}
	m.setup = append([]string(nil), m.lessons[m.lesson].setup...)
	m.stage = 0
	m.app, m.database = tui.NewModel(config), database
	m.generation++
	m.achieved, m.finished = false, false
	m.modal = ""
	m.feedback = ""
	if old != nil {
		m.closeErr = errors.Join(m.closeErr, old.Close())
	}
	return nil
}

func (m *TaskModel) Close() error  { return errors.Join(m.closeErr, m.database.Close()) }
func (m *TaskModel) Init() tea.Cmd { return m.wrap(m.app.Init()) }

// Tag command results so work from a previous disposable session cannot update
// the reset application. Batch children must retain the same generation too.
func (m *TaskModel) wrap(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	generation := m.generation
	var wrap func(tea.Cmd) tea.Cmd
	wrap = func(command tea.Cmd) tea.Cmd {
		return func() tea.Msg {
			msg := command()
			if batch, ok := msg.(tea.BatchMsg); ok {
				children := make([]tea.Cmd, 0, len(batch))
				for _, child := range batch {
					if child != nil {
						children = append(children, wrap(child))
					}
				}
				return tea.BatchMsg(children)
			}
			if msg == nil {
				return nil
			}
			return taskAppMsg{generation, msg}
		}
	}
	return wrap(cmd)
}

func (m *TaskModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if tagged, ok := msg.(taskAppMsg); ok {
		if tagged.generation != m.generation {
			return m, nil
		}
		msg = tagged.message
		if _, quit := msg.(tea.QuitMsg); quit {
			return m, tea.Quit
		}
	}
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		return m, m.resize()
	}
	// Skip must work before modal/setup handling so repeated presses can jump
	// straight to a later task without confirming intervening introductions.
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "f12" {
		return m, m.advance(!m.achieved)
	}
	if m.modal != "" {
		if key, ok := msg.(tea.KeyPressMsg); ok {
			switch key.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter", "esc":
				m.modal = ""
				return m, nil
			}
			return m, nil
		}
		switch msg.(type) {
		case tea.PasteMsg, tea.KeyReleaseMsg:
			return m, nil
		}
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "f1":
			m.modal = "help"
			return m, m.resize()
		case "f5":
			if err := m.restart(); err != nil {
				m.feedback = fmt.Sprintf("Reset failed: %v", err)
				return m, m.resize()
			}
			delete(m.skipped, m.lesson)
			return m, tea.Batch(m.Init(), m.resize())
		case "f8":
			if m.achieved {
				return m, m.advance(false)
			}
			return m, m.resize()
		}
	}
	// Setup drives the real UI only while preparing a fresh fixture. It is
	// independent of learner input and never contributes to goal recognition.
	if len(m.setup) > 0 {
		switch msg.(type) {
		case tea.KeyPressMsg, tea.PasteMsg, tea.KeyReleaseMsg:
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.app, cmd = m.app.Update(msg)
	commands := []tea.Cmd{m.wrap(cmd)}
	state := m.snapshot()
	for len(m.setup) > 0 && state.Ready {
		key := m.setup[0]
		m.setup = m.setup[1:]
		m.app, cmd = m.app.Update(lessonKey(key))
		commands = append(commands, m.wrap(cmd))
		state = m.snapshot()
	}
	if len(m.setup) == 0 && state.Ready && !m.finished && !m.achieved {
		goals := m.lessons[m.lesson].goals
		if goals[m.stage](state) {
			if m.stage+1 == len(goals) {
				if state.Normal {
					m.achieved = true
				}
			} else {
				m.stage++
			}
		}
	}
	commands = append(commands, m.resize())
	return m, tea.Batch(commands...)
}

// Advance only records a skip once the transition succeeds. Old asynchronous
// results remain isolated by restart's generation tag.
func (m *TaskModel) advance(skip bool) tea.Cmd {
	if m.finished {
		return nil
	}
	previous := m.lesson
	if previous+1 == len(m.lessons) {
		if skip {
			m.skipped[previous] = true
		}
		m.finished = true
		m.modal = ""
		return m.resize()
	}
	m.lesson++
	if err := m.restart(); err != nil {
		m.lesson = previous
		m.feedback = fmt.Sprintf("Next task failed: %v", err)
		return m.resize()
	}
	if skip {
		m.skipped[previous] = true
	}
	m.modal = "intro"
	return tea.Batch(m.Init(), m.resize())
}

func (m *TaskModel) snapshot() tui.SessionSnapshot {
	return m.app.(interface{ SessionSnapshot() tui.SessionSnapshot }).SessionSnapshot()
}

func (m *TaskModel) resize() tea.Cmd {
	var cmd tea.Cmd
	m.app, cmd = m.app.Update(tea.WindowSizeMsg{Width: m.width, Height: max(1, m.height-lipgloss.Height(m.panel()))})
	return m.wrap(cmd)
}

func (m *TaskModel) panel() string {
	task := m.lessons[m.lesson].task
	controls := fmt.Sprintf("%d/%d · F1 help · F5 reset · F12 skip", m.lesson+1, len(m.lessons))
	if m.achieved {
		task = "Goal reached! F8: continue"
	}
	if m.finished {
		task = "TUTORIAL COMPLETE"
		if len(m.skipped) > 0 {
			task = fmt.Sprintf("TUTORIAL ENDED · %d skipped", len(m.skipped))
		}
		controls = "F5 repeat · Ctrl+C exit"
	}
	if m.feedback != "" {
		task = m.feedback
	}
	width := max(1, m.width)
	task = ansi.Wrap(task, max(1, width-2), "")
	// Keep emphasis separate from the text without filling the background.
	task = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("#67E8F9")).Bold(true).PaddingLeft(1).Width(width).Render(task)
	return task + "\n" + ansi.Wrap(controls, width, "")
}

func (m *TaskModel) renderContext() string {
	title := m.lessons[m.lesson].topic
	body := m.lessons[m.lesson].context + " Explore freely; F5 restores this task’s demo data, including saved edits and deletions."
	action := "Enter: start · F12 skip · Ctrl+C exit"
	if m.modal == "help" {
		title += " · Help"
		body = m.lessons[m.lesson].hint
		action = "Enter / Esc: back · F12 skip · Ctrl+C exit"
	}
	width := max(1, min(60, m.width-6))
	progress := fmt.Sprintf("%d / %d", m.lesson+1, len(m.lessons))
	titleWidth := width - ansi.StringWidth(progress) - 1
	header := ansi.Truncate(progress, width, "")
	if titleWidth > 0 {
		label := ansi.Truncate(title, titleWidth, "…")
		header = label + strings.Repeat(" ", width-ansi.StringWidth(label)-ansi.StringWidth(progress)) + progress
	}
	content := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#67E8F9")).Render(header) + "\n\n" + ansi.Wrap(body, width, "") + "\n\n" + ansi.Wrap(action, width, "")
	modal := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#67E8F9")).Padding(1, 2).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m *TaskModel) View() tea.View {
	view := m.app.View()
	view.Content = lipgloss.JoinVertical(lipgloss.Left, view.Content, m.panel())
	if m.modal != "" {
		view.Content = m.renderContext()
		view.Cursor = nil
	}
	view.WindowTitle = "Vimkunja Tutorial"
	return view
}
