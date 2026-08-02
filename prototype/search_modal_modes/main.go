// search_modal_modes is a THROWAWAY interactive TUI prototype.
//
// Question: Does a Global Search overlay feel better when its input starts in
// Insert Mode and Esc switches to a Vim-style Normal Mode for result browsing?
//
// Run: go run ./prototype/search_modal_modes
// No database or Vimkunja production state is used.
package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type searchMode int

const (
	insertMode searchMode = iota
	normalMode
	closedMode
)

type result struct {
	kind string
	text string
}

type model struct {
	mode     searchMode
	query    string
	selected int
	scroll   int
	gArmed   bool
	message  string
	width    int
	height   int
}

var corpus = []result{
	{"TASK", "Prepare release notes"},
	{"LABEL", "Release blockers"},
	{"DESCRIPTION", "Read the release notes before launch"},
	{"PROJECT", "Release Engineering"},
	{"TASK", "Buy coffee"},
	{"DESCRIPTION", "Coffee notes from the planning meeting"},
}

func initialModel() model {
	return model{
		mode: insertMode, query: "release", width: 100, height: 30,
		message: "Started in Insert Mode: type to change the query.",
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		return m, nil
	case tea.KeyMsg:
		if _, released := typed.(tea.KeyReleaseMsg); released {
			return m, nil
		}
		key := typed.String()
		if key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}
		if m.mode == closedMode {
			if key == "r" {
				m.mode = insertMode
				m.message = "Overlay reopened in Insert Mode."
			}
			return m, nil
		}
		if m.mode == insertMode {
			return m.updateInsert(typed)
		}
		return m.updateNormal(key)
	}
	return m, nil
}

func (m model) updateInsert(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = normalMode
		m.gArmed = false
		m.message = "Normal Mode: j/k, gg/G, Ctrl+D/U, Enter; i/a returns to input."
	case "backspace":
		runes := []rune(m.query)
		if len(runes) > 0 {
			m.query = string(runes[:len(runes)-1])
		}
		m.resetSelection()
	case "up", "ctrl+p":
		m.move(-1)
	case "down", "ctrl+n":
		m.move(1)
	case "enter":
		m.message = "Would activate the selected result while remaining in the overlay prototype."
	default:
		if text := key.Key().Text; text != "" {
			m.query += text
			m.resetSelection()
		}
	}
	return m, nil
}

func (m model) updateNormal(key string) (tea.Model, tea.Cmd) {
	if m.gArmed {
		m.gArmed = false
		if key == "g" {
			m.selected, m.scroll = 0, 0
			m.message = "gg: first result selected."
			return m, nil
		}
	}
	switch key {
	case "esc":
		m.mode = closedMode
		m.message = "Normal-Mode Esc would close Search and restore its origin. Press r to reopen this prototype."
	case "i", "a":
		m.mode = insertMode
		m.message = "Insert Mode: edit the query again."
	case "j", "down":
		m.move(1)
	case "k", "up":
		m.move(-1)
	case "g":
		m.gArmed = true
		m.message = "g pending: press g for first result."
	case "G", "end":
		m.selected = max(0, len(m.matches())-1)
		m.ensureVisible()
		m.message = "G: last result selected."
	case "ctrl+d", "pgdown":
		m.move(m.visibleRows() / 2)
	case "ctrl+u", "pgup":
		m.move(-m.visibleRows() / 2)
	case "enter":
		m.message = "Would activate the selected result and retain a Ctrl+O snapshot."
	}
	return m, nil
}

func (m *model) resetSelection() {
	m.selected, m.scroll = 0, 0
}

func (m *model) move(delta int) {
	m.selected = clamp(m.selected+delta, 0, max(0, len(m.matches())-1))
	m.ensureVisible()
	m.message = "Result selection changed in " + m.modeLabel() + "."
}

func (m *model) ensureVisible() {
	if m.selected < m.scroll {
		m.scroll = m.selected
	}
	if m.selected >= m.scroll+m.visibleRows() {
		m.scroll = m.selected - m.visibleRows() + 1
	}
}

func (m model) matches() []result {
	terms := strings.Fields(strings.ToLower(m.query))
	if len(terms) == 0 {
		return nil
	}
	var matches []result
	for _, item := range corpus {
		text := strings.ToLower(item.text)
		matchesAll := true
		for _, term := range terms {
			matchesAll = matchesAll && strings.Contains(text, term)
		}
		if matchesAll {
			matches = append(matches, item)
		}
	}
	return matches
}

func (m model) visibleRows() int { return max(3, m.height*80/100-10) }

func (m model) modeLabel() string {
	switch m.mode {
	case insertMode:
		return "INSERT"
	case normalMode:
		return "NORMAL"
	default:
		return "CLOSED"
	}
}

func (m model) View() tea.View {
	if m.mode == closedMode {
		return tea.NewView("SEARCH MODAL CLOSED\n\nState: mode=CLOSED · query=" + fmt.Sprintf("%q", m.query) + "\n" + m.message + "\n\n[r] reopen prototype  [q] quit")
	}
	modalWidth := max(36, m.width*90/100)
	modalHeight := max(12, m.height*80/100)
	matches := m.matches()
	start := clamp(m.scroll, 0, max(0, len(matches)-1))
	end := min(len(matches), start+m.visibleRows())

	modeColor := lipgloss.Color("42")
	if m.mode == normalMode {
		modeColor = lipgloss.Color("205")
	}
	lines := []string{
		lipgloss.NewStyle().Bold(true).Render("GLOBAL SEARCH · MODE PROTOTYPE"),
		lipgloss.NewStyle().Foreground(modeColor).Bold(true).Render("MODE: " + m.modeLabel()),
		"Query: " + m.query + "_",
		"",
	}
	if len(matches) == 0 {
		lines = append(lines, "No results.")
	}
	for index := start; index < end; index++ {
		item := matches[index]
		prefix := "  "
		if index == m.selected {
			prefix = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render(">") + " "
		}
		lines = append(lines, prefix+item.kind+"  "+item.text)
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("STATE: mode="+m.modeLabel()+" · query="+fmt.Sprintf("%q", m.query)+fmt.Sprintf(" · selected=%d · scroll=%d · results=%d · gPending=%v", m.selected, m.scroll, len(matches), m.gArmed)))
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(m.message))
	shortcuts := "INSERT: type, arrows/Ctrl+N/P, Esc → Normal  |  NORMAL: j/k, gg/G, Ctrl+D/U, i/a, Enter, Esc → close  |  q quit"
	lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Render(shortcuts))

	modal := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(modeColor).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight - 4).
		Render(strings.Join(lines, "\n"))
	return tea.NewView(modal)
}

func clamp(value, lower, upper int) int {
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Println("prototype error:", err)
	}
}
