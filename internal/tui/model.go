package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vikunja-tui/vikunja-tui/internal/vikunja"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type paneID int

const (
	paneProjects paneID = iota
	paneTasks
	paneDetail
)

type paneNeighbors struct {
	up, down, left, right *paneID
}

type Config struct {
	Store vikunja.Store

	PerPage int

	InitialProjectID int64
	InitialViewID    int64
}

type model struct {
	store vikunja.Store
	perPage int

	focus paneID
	ctrlWArmed bool
	neighbors map[paneID]paneNeighbors

	lastErr string
	toast string

	// Projects pane
	projects []vikunja.Project
	projectSelected int
	projectLoading bool

	// Views (derived from selected project)
	projectViews []vikunja.ProjectView
	viewSelected int
	viewLoading bool
	currentViewID int64

	// Tasks pane
	tasks []vikunja.Task
	taskSelected int
	taskLoading bool

	// Detail pane
	showDetail bool

	currentProjectID int64

	initialProjectID int64
	initialViewID int64
}

func NewModel(cfg Config) tea.Model {
	m := &model{
		store: cfg.Store,
		perPage: max(1, cfg.PerPage),
		initialProjectID: cfg.InitialProjectID,
		initialViewID: cfg.InitialViewID,
		projectSelected: 0,
		currentViewID: cfg.InitialViewID,
		neighbors: map[paneID]paneNeighbors{
			paneProjects: {left: nil, right: ptr(paneTasks), down: ptr(paneDetail)},
			paneTasks:     {left: ptr(paneProjects), right: nil, down: ptr(paneDetail)},
			paneDetail:    {up: ptr(paneTasks)},
		},
		focus: paneProjects,
	}
	if cfg.InitialProjectID != 0 {
		m.focus = paneTasks
	}
	return m
}

func ptr(p paneID) *paneID { return &p }

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type (
	projectsLoadedMsg struct {
		projects []vikunja.Project
		err error
	}
	viewsLoadedMsg struct {
		views []vikunja.ProjectView
		err error
	}
	tasksLoadedMsg struct {
		tasks []vikunja.Task
		err error
	}
)

func (m *model) Init() tea.Cmd {
	m.projectLoading = true
	return m.loadProjects(1)
}

func (m *model) loadProjects(page int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		items, _, err := m.store.ListProjects(ctx, page, 200)
		return projectsLoadedMsg{projects: items, err: err}
	}
}

func (m *model) loadViews(projectID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		items, _, err := m.store.ListViews(ctx, projectID, 1, 200)
		return viewsLoadedMsg{views: items, err: err}
	}
}

func (m *model) loadTasks(projectID, viewID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		items, _, err := m.store.ListTasksInView(ctx, projectID, viewID, 1, m.perPage)
		return tasksLoadedMsg{tasks: items, err: err}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Vim-split style focus switching
	if key, ok := msg.(tea.KeyMsg); ok {
		k := key.String()
		kLower := strings.ToLower(k)

		// Quit
		if kLower == "ctrl+c" {
			return m, tea.Quit
		}

		// Ctrl+W leader
		if m.ctrlWArmed {
			m.ctrlWArmed = false

			// Ctrl+W + Ctrl+W cycles focus.
			if kLower == "ctrl+w" {
				m.focus = m.nextFocus()
				return m, nil
			}

			switch k {
			case "h":
				m.focus = m.focusNeighborLeftOrStay()
			case "l":
				m.focus = m.focusNeighborRightOrStay()
			case "j":
				m.focus = m.focusNeighborDownOrStay()
			case "k":
				m.focus = m.focusNeighborUpOrStay()
			default:
				// no-op
			}
			return m, nil
		}

		if kLower == "ctrl+w" {
			m.ctrlWArmed = true
			return m, nil
		}

		// Pane-local keys
		switch m.focus {
		case paneProjects:
			return m.updateProjectsKeys(key)
		case paneTasks:
			return m.updateTasksKeys(key)
		case paneDetail:
			return m.updateDetailKeys(key)
		}
		return m, nil
	}

	switch typed := msg.(type) {
	case projectsLoadedMsg:
		m.projectLoading = false
		if typed.err != nil {
			m.lastErr = typed.err.Error()
			return m, nil
		}

		m.projects = typed.projects
		if len(m.projects) == 0 {
			m.lastErr = "No projects returned."
			return m, nil
		}

		if m.initialProjectID != 0 {
			idx := m.findProjectIndex(m.initialProjectID)
			if idx >= 0 {
				m.projectSelected = idx
			}
		}
		m.projectSelected = clamp(m.projectSelected, 0, len(m.projects)-1)
		m.currentProjectID = m.projects[m.projectSelected].ID

		// Initial load: either direct view tasks or derive list view.
		if m.initialViewID != 0 {
			m.currentViewID = m.initialViewID
			m.taskLoading = true
			return m, m.loadTasks(m.currentProjectID, m.currentViewID)
		}

		m.viewLoading = true
		return m, m.loadViews(m.currentProjectID)

	case viewsLoadedMsg:
		m.viewLoading = false
		if typed.err != nil {
			m.lastErr = typed.err.Error()
			return m, nil
		}
		m.projectViews = typed.views
		m.viewSelected = m.pickListViewIndex()
		if m.viewSelected < 0 {
			m.lastErr = "No 'list' views found for the selected project."
			return m, nil
		}
		m.currentViewID = m.projectViews[m.viewSelected].ID

		m.taskLoading = true
		return m, m.loadTasks(m.currentProjectID, m.currentViewID)

	case tasksLoadedMsg:
		m.taskLoading = false
		if typed.err != nil {
			m.lastErr = typed.err.Error()
			return m, nil
		}
		m.tasks = typed.tasks
		m.taskSelected = clamp(m.taskSelected, 0, max(0, len(m.tasks)-1))
		return m, nil
	}

	return m, nil
}

func (m *model) nextFocus() paneID {
	if m.focus == paneProjects {
		return paneTasks
	}
	return paneProjects
}

func (m *model) focusNeighborLeftOrStay() paneID {
	n := m.neighbors[m.focus]
	if n.left != nil {
		return *n.left
	}
	return m.focus
}

func (m *model) focusNeighborRightOrStay() paneID {
	n := m.neighbors[m.focus]
	if n.right != nil {
		return *n.right
	}
	return m.focus
}

func (m *model) focusNeighborUpOrStay() paneID {
	n := m.neighbors[m.focus]
	if n.up != nil {
		return *n.up
	}
	return m.focus
}

func (m *model) focusNeighborDownOrStay() paneID {
	n := m.neighbors[m.focus]
	if n.down != nil {
		return *n.down
	}
	return m.focus
}

func (m *model) updateProjectsKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q":
		return m, tea.Quit
	case "r":
		m.lastErr = ""
		m.projectLoading = true
		return m, m.loadProjects(1)
	case "j":
		m.projectSelected = clamp(m.projectSelected+1, 0, max(0, len(m.projects)-1))
		return m, m.onProjectSelectionChanged()
	case "k":
		m.projectSelected = clamp(m.projectSelected-1, 0, max(0, len(m.projects)-1))
		return m, m.onProjectSelectionChanged()
	}

	return m, nil
}

func (m *model) updateTasksKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q":
		return m, tea.Quit
	case "enter":
		if len(m.tasks) > 0 {
			m.showDetail = true
			m.focus = paneDetail
		}
		return m, nil
	case "r":
		m.lastErr = ""
		if m.currentProjectID != 0 && m.currentViewID != 0 {
			m.taskLoading = true
			return m, m.loadTasks(m.currentProjectID, m.currentViewID)
		}
	}

	switch k.String() {
	case "j":
		m.taskSelected = clamp(m.taskSelected+1, 0, max(0, len(m.tasks)-1))
		return m, nil
	case "k":
		m.taskSelected = clamp(m.taskSelected-1, 0, max(0, len(m.tasks)-1))
		return m, nil
	case "x":
		// Toggle done (MVP write-support for SQLite mode).
		if m.taskSelected >= 0 && m.taskSelected < len(m.tasks) {
			t := m.tasks[m.taskSelected]
			m.taskLoading = true
			m.lastErr = ""
			return m, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				defer cancel()
				if err := m.store.ToggleTaskDone(ctx, t.ID, !t.Done); err != nil {
					return tasksLoadedMsg{err: err}
				}
				// Reload current view tasks.
				if m.currentProjectID == 0 || m.currentViewID == 0 {
						return tasksLoadedMsg{tasks: nil, err: nil}
				}
				items, _, err := m.store.ListTasksInView(ctx, m.currentProjectID, m.currentViewID, 1, m.perPage)
				return tasksLoadedMsg{tasks: items, err: err}
			}
		}
	}

	return m, nil
}

func (m *model) updateDetailKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "esc":
		m.showDetail = false
		m.focus = paneTasks
		return m, nil
	}
	return m, nil
}

func (m *model) onProjectSelectionChanged() tea.Cmd {
	if len(m.projects) == 0 {
		return nil
	}
	m.currentProjectID = m.projects[m.projectSelected].ID
	m.currentViewID = 0
	m.tasks = nil
	m.taskSelected = 0
	m.lastErr = ""

	m.viewLoading = true
	return m.loadViews(m.currentProjectID)
}

func (m *model) findProjectIndex(id int64) int {
	for i, p := range m.projects {
		if p.ID == id {
			return i
		}
	}
	return -1
}

func (m *model) pickListViewIndex() int {
	for i, v := range m.projectViews {
		if v.ViewKind == "list" {
			return i
		}
	}
	return -1
}

func (m *model) View() tea.View {
	v := tea.NewView(m.Render())
	v.AltScreen = true
	return v
}

func (m *model) Render() string {
	left := renderProjects(m)
	right := renderTasks(m)

	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Render(left),
		lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Render(right),
	)

	content := panes
	if m.showDetail {
		detail := renderTaskDetail(m)
		content = lipgloss.JoinVertical(lipgloss.Top,
			panes,
			lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Width(lipgloss.Width(panes)-2).Render(detail),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Top, content, m.renderStatusBar())
}

func (m *model) renderStatusBar() string {
	info := ""
	switch m.focus {
	case paneProjects:
		info = "[Projects]"
	case paneTasks:
		info = "[Tasks]"
	case paneDetail:
		info = "[Detail]"
	}

	hint := "Ctrl+W cycles  •  Enter detail  •  q quit  •  r refresh  •  j/k navigate  •  x toggle"
	if m.focus == paneDetail {
		hint = "q/esc close detail  •  Ctrl+W cycle"
	}
	if m.lastErr != "" {
		hint = fmt.Sprintf("Error: %s", m.lastErr)
	}

	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Render(
		fmt.Sprintf("%s   %s", info, hint),
	)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
