package tutorial

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Mode selects the fixtures and whether lesson key restrictions apply.
type Mode struct {
	Markup   bool
	Practice bool
}

var startOptions = []struct {
	title       string
	description string
	mode        Mode
}{
	{"Learn the basics", "Guided navigation, Tasks and Views; follow the requested keys.", Mode{}},
	{"Practice the basics", "Explore freely with demo Tasks; no guidance or key restrictions.", Mode{Practice: true}},
	{"Learn markup", "Guided rich-text Description editing; follow the requested keys.", Mode{Markup: true}},
	{"Practice markup", "Explore rich-text examples freely; no guidance or key restrictions.", Mode{Markup: true, Practice: true}},
	{title: "Quit", description: "Leave without starting a tutorial."},
}

// StartMenu offers all tutorial modes on one screen, before any database is created.
type StartMenu struct {
	cursor   int
	selected bool
}

func NewStartMenu() *StartMenu { return &StartMenu{} }

// Selection returns a mode only after the user starts one with Enter.
func (m *StartMenu) Selection() (Mode, bool) {
	return startOptions[m.cursor].mode, m.selected
}

func (m *StartMenu) Init() tea.Cmd { return nil }

func (m *StartMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "j", "down":
		m.cursor = min(m.cursor+1, len(startOptions)-1)
	case "k", "up":
		m.cursor = max(m.cursor-1, 0)
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.selected = m.cursor < len(startOptions)-1
		return m, tea.Quit
	}
	return m, nil
}

func (m *StartMenu) View() tea.View {
	var out strings.Builder
	out.WriteString("VIMKUNJA TUTORIAL\n\n")
	for i, option := range startOptions {
		title := "  " + option.title
		if i == m.cursor {
			title = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true).Render("> " + option.title)
		}
		fmt.Fprintf(&out, "%s\n    %s\n\n", title, option.description)
	}
	out.WriteString("All modes use temporary data, discarded on normal exit.\n\nj/k or arrows: select  |  Enter: start  |  q: quit\n")
	view := tea.NewView(out.String())
	view.AltScreen = true
	view.WindowTitle = "Vimkunja Tutorial"
	return view
}
