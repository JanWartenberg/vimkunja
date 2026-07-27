// overview_tui is a THROWAWAY interactive TUI prototype.
//
// Question: Does an Overview Pane that is a peer to the project-specific Tasks
// Pane make daily work and later sorting/filtering feel natural, while the Task
// Detail Pane remains absent until a Task is explicitly opened?
//
// Run: go run ./prototype/overview_tui
package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type focusPane int

const (
	focusProjects focusPane = iota
	focusTasks
	focusDetail
)

type prototypeModel struct {
	state  prototypeState
	focus  focusPane
	width  int
	height int
}

var (
	accent = lipgloss.Color("205")
	muted  = lipgloss.Color("245")
	cyan   = lipgloss.Color("81")
	green  = lipgloss.Color("42")
	yellow = lipgloss.Color("220")
	red    = lipgloss.Color("203")
)

func newPrototype() prototypeModel {
	return prototypeModel{
		state: prototypeState{
			surface: surfaceOverview,
			sort:    sortDueAsc,
			filter:  filterOpen,
			message: "Overview is active; no Task Detail Pane is open.",
		},
		focus:  focusTasks,
		width:  112,
		height: 34,
	}
}

func (m prototypeModel) Init() tea.Cmd { return nil }

func (m prototypeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if m.state.sortMenu {
			return m.updateSortMenu(key)
		}
		if m.state.filterMenu {
			return m.updateFilterMenu(key)
		}
		if m.state.detailTask != nil && key == "esc" {
			m.state.detailTask = nil
			m.focus = focusTasks
			m.state.message = "Task Detail Pane closed; it only opens with Enter on a Task."
			return m, nil
		}

		switch key {
		case "ctrl+w", "tab":
			m.cycleFocus()
		case "-":
			m.state.surface = surfaceOverview
			m.state.selectedTask = 0
			m.focus = focusTasks
			m.state.message = "Overview Pane opened: all Projects in one Task list."
		case "j", "down":
			m.moveSelection(1)
		case "k", "up":
			m.moveSelection(-1)
		case "enter":
			m.openSelection()
		case "s":
			m.state.sortMenu = true
			m.state.message = "Sort menu opened (netrw-inspired)."
		case "f":
			m.state.filterMenu = true
			m.state.message = "Filter menu opened."
		case "esc":
			m.state.message = "Nothing to close: the Task Detail Pane is hidden."
		}
		return m, nil
	}
	return m, nil
}

func (m *prototypeModel) cycleFocus() {
	if m.state.detailTask == nil {
		if m.focus == focusProjects {
			m.focus = focusTasks
		} else {
			m.focus = focusProjects
		}
	} else {
		m.focus = (m.focus + 1) % 3
	}
	m.state.message = "Pane focus moved with Ctrl+W / Tab."
}

func (m *prototypeModel) moveSelection(delta int) {
	if m.focus == focusProjects {
		m.state.selectedProject = clamp(m.state.selectedProject+delta, 0, len(prototypeProjects)-1)
		m.state.message = "Project selection changed; Enter opens its Tasks Pane."
		return
	}
	if m.focus == focusTasks {
		items := m.state.visibleTasks()
		m.state.selectedTask = clamp(m.state.selectedTask+delta, 0, max(0, len(items)-1))
		m.state.message = "Task selection changed; Enter explicitly opens the Task Detail Pane."
	}
}

func (m *prototypeModel) openSelection() {
	if m.focus == focusProjects {
		m.state.surface = surfaceProjectTasks
		m.state.selectedTask = 0
		m.focus = focusTasks
		m.state.message = "Project selected: its Tasks Pane replaces the Overview Pane. Press - to return to Overview."
		return
	}
	if m.focus != focusTasks {
		return
	}
	items := m.state.visibleTasks()
	if len(items) == 0 {
		m.state.message = "No Task is available to open."
		return
	}
	task := items[m.state.selectedTask]
	m.state.detailTask = &task
	m.focus = focusDetail
	m.state.message = "Task explicitly opened: Task Detail Pane is now visible."
}

func (m prototypeModel) updateSortMenu(key string) (tea.Model, tea.Cmd) {
	m.state.sortMenu = false
	switch key {
	case "d":
		m.state.sort = sortDueDesc
	case "u":
		m.state.sort = sortDueAsc
	case "p":
		m.state.sort = sortPriorityDesc
	case "t":
		m.state.sort = sortTitle
	case "esc":
		m.state.message = "Sort menu cancelled."
		return m, nil
	default:
		m.state.message = "Unknown sort key; sort unchanged."
		return m, nil
	}
	m.state.selectedTask = 0
	m.state.message = "Sort changed to " + sortLabel(m.state.sort) + "."
	return m, nil
}

func (m prototypeModel) updateFilterMenu(key string) (tea.Model, tea.Cmd) {
	m.state.filterMenu = false
	switch key {
	case "o":
		m.state.filter = filterOpen
	case "a":
		m.state.filter = filterAll
	case "v":
		m.state.filter = filterOverdue
	case "t":
		m.state.filter = filterDueToday
	case "esc":
		m.state.message = "Filter menu cancelled."
		return m, nil
	default:
		m.state.message = "Unknown filter key; filter unchanged."
		return m, nil
	}
	m.state.selectedTask = 0
	m.state.message = "Filter changed to " + filterLabel(m.state.filter) + "."
	return m, nil
}

func (m prototypeModel) View() tea.View {
	topHeight := max(12, m.height-9)
	if m.state.detailTask != nil {
		topHeight = max(8, (m.height-11)/2)
	}
	leftWidth := min(26, max(18, m.width/4))
	rightWidth := max(42, m.width-leftWidth-3)

	header := lipgloss.NewStyle().Bold(true).Foreground(accent).Render(" VIMKUNJA · OVERVIEW PANE PROTOTYPE ") +
		lipgloss.NewStyle().Foreground(muted).Render("  synthetic data · no persistence")
	projects := m.renderProjects(leftWidth, topHeight)
	tasks := m.renderTasks(rightWidth, topHeight)
	top := lipgloss.JoinHorizontal(lipgloss.Top, projects, " ", tasks)

	state := m.renderState()
	shortcuts := m.renderShortcuts()
	frame := header + "\n" + top
	if m.state.detailTask != nil {
		frame += "\n" + m.renderDetail(m.width, max(7, m.height-topHeight-8))
	}
	frame += "\n" + state + "\n" + shortcuts
	return tea.NewView(frame)
}

func (m prototypeModel) renderProjects(width, height int) string {
	lines := []string{paneHeading(" Projects Pane ", m.focus == focusProjects), ""}
	for i, project := range prototypeProjects {
		marker := "  "
		if i == m.state.selectedProject {
			marker = "▸ "
		}
		line := marker + project
		if i == m.state.selectedProject && m.focus == focusProjects {
			line = lipgloss.NewStyle().Bold(true).Foreground(accent).Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(muted).Render("Enter → Tasks Pane"))
	return paneBox(strings.Join(lines, "\n"), width, height, m.focus == focusProjects)
}

func (m prototypeModel) renderTasks(width, height int) string {
	items := m.state.visibleTasks()
	title := " " + m.state.paneName() + " "
	lines := []string{paneHeading(title, m.focus == focusTasks), lipgloss.NewStyle().Foreground(cyan).Render(" " + sortLabel(m.state.sort) + " · " + filterLabel(m.state.filter))}
	if m.state.surface == surfaceOverview {
		lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render(" Project is context; Overview contains Tasks from all Projects."))
	}
	if len(items) == 0 {
		lines = append(lines, "", "  (no Tasks match this filter)")
	}
	for i, task := range items {
		marker := "  "
		if i == m.state.selectedTask {
			marker = "▸ "
		}
		project := ""
		if m.state.surface == surfaceOverview {
			project = fmt.Sprintf(" %-5s", task.Project)
		}
		due := "—"
		if !task.Due.IsZero() {
			due = task.Due.Format("02 Jan")
		}
		line := fmt.Sprintf("%s%s P%d  %-34s  %s", marker, project, task.Priority, truncate(task.Title, 34), due)
		if i == m.state.selectedTask && m.focus == focusTasks {
			line = lipgloss.NewStyle().Bold(true).Foreground(accent).Render(line)
		} else if task.Done {
			line = lipgloss.NewStyle().Faint(true).Foreground(muted).Render(line)
		} else {
			line = stylePriority(line, task.Priority)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(muted).Render("Enter → Task Detail Pane · - → Overview Pane"))
	return paneBox(strings.Join(lines, "\n"), width, height, m.focus == focusTasks)
}

func (m prototypeModel) renderDetail(width, height int) string {
	task := *m.state.detailTask
	due := "no Due Date"
	if !task.Due.IsZero() {
		due = task.Due.Format("2006-01-02")
	}
	content := strings.Join([]string{
		paneHeading(" Task Detail Pane · only after explicit Enter ", m.focus == focusDetail),
		"",
		"  " + task.Title,
		fmt.Sprintf("  Project: %s   Workflow: %s   Priority: P%d   Due: %s", task.Project, task.Bucket, task.Priority, due),
		lipgloss.NewStyle().Foreground(muted).Render("  Esc closes this Pane; selecting a Project or Overview never opens it implicitly."),
	}, "\n")
	return paneBox(content, width, height, m.focus == focusDetail)
}

func (m prototypeModel) renderState() string {
	detail := "hidden"
	if m.state.detailTask != nil {
		detail = "open for #" + fmt.Sprint(m.state.detailTask.ID)
	}
	stateLine := fmt.Sprintf(" STATE  surface=%s · sort=%s · filter=%s · detail=%s", m.state.paneName(), sortLabel(m.state.sort), filterLabel(m.state.filter), detail)
	message := " " + m.state.message
	if m.state.sortMenu {
		message = " SORT MENU (netrw-inspired): [d] Due ↓  [u] Due ↑  [p] Priority ↓  [t] Title  [Esc] cancel"
	}
	if m.state.filterMenu {
		message = " FILTER MENU: [o] open  [a] all  [v] overdue  [t] due today  [Esc] cancel"
	}
	return lipgloss.NewStyle().Foreground(cyan).Render(stateLine) + "\n" + lipgloss.NewStyle().Foreground(muted).Render(message)
}

func (m prototypeModel) renderShortcuts() string {
	return lipgloss.NewStyle().Foreground(muted).Render(" SHORTCUTS  ") +
		"[j/k] select  [Ctrl+W/Tab] Pane focus  [Enter] Project → Tasks / Task → Detail  [-] Overview  [s] sort  [f] filter  [Esc] close Detail  [q] quit"
}

func paneHeading(value string, focused bool) string {
	style := lipgloss.NewStyle().Bold(true)
	if focused {
		style = style.Foreground(accent)
	} else {
		style = style.Foreground(cyan)
	}
	return style.Render(value)
}

func paneBox(content string, width, height int, focused bool) string {
	border := lipgloss.Color("60")
	if focused {
		border = accent
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(max(8, width-2)).
		Height(max(4, height-2)).
		Padding(0, 1).
		Render(content)
}

func stylePriority(value string, priority int) string {
	color := green
	if priority == 2 {
		color = yellow
	}
	if priority >= 3 {
		color = red
	}
	return lipgloss.NewStyle().Foreground(color).Render(value)
}

func truncate(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:max(0, width-1)]) + "…"
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
	program := tea.NewProgram(newPrototype())
	if _, err := program.Run(); err != nil {
		fmt.Println("prototype error:", err)
	}
}
