package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/vimkunja/vimkunja/internal/errors"
	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type paneID int

const (
	paneProjects paneID = iota
	paneTasks
	paneDetail
	paneHelp
	paneInfo
)

type paneNeighbors struct {
	up, down, left, right *paneID
}

type Config struct {
	Store vimkunja.Store

	PerPage int

	InitialProjectID int64
	InitialViewID    int64
	ActorUserID      int64
}

type model struct {
	store   vimkunja.Store
	perPage int

	width  int
	height int

	focus      paneID
	ctrlWArmed bool
	gArmed     bool
	neighbors  map[paneID]paneNeighbors

	lastErr string
	toast   string

	commandMode  bool
	commandInput string
	commands     *CommandRegistry

	// Projects pane
	projects        []vimkunja.Project
	projectSelected int
	projectLoading  bool

	// Views (derived from selected project)
	projectViews  []vimkunja.ProjectView
	viewSelected  int
	viewLoading   bool
	currentViewID int64

	// Tasks pane
	tasks        []vimkunja.Task
	taskSelected int
	taskLoading  bool

	// Detail and auxiliary panes
	showDetail bool
	helpPane   *infoPane
	infoPane   *infoPane

	currentProjectID int64
	actorUserID      int64
	pendingTaskID    int64
	editor           *taskEditor
	cursorVisible    bool

	initialProjectID int64
	initialViewID    int64
}

func NewModel(cfg Config) tea.Model {
	m := &model{
		store:            cfg.Store,
		perPage:          max(1, cfg.PerPage),
		initialProjectID: cfg.InitialProjectID,
		actorUserID:      cfg.ActorUserID,
		initialViewID:    cfg.InitialViewID,
		projectSelected:  0,
		currentViewID:    cfg.InitialViewID,
		neighbors: map[paneID]paneNeighbors{
			paneProjects: {left: nil, right: ptr(paneTasks), down: ptr(paneDetail)},
			paneTasks:    {left: ptr(paneProjects), right: nil, down: ptr(paneDetail)},
			paneDetail:   {up: ptr(paneTasks)},
		},
		focus:         paneProjects,
		commands:      newDefaultCommandRegistry(),
		cursorVisible: true,
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
		projects []vimkunja.Project
		err      error
	}
	viewsLoadedMsg struct {
		views []vimkunja.ProjectView
		err   error
	}
	tasksLoadedMsg struct {
		tasks []vimkunja.Task
		err   error
	}
	taskToggledMsg struct {
		err error
	}
	taskSavedMsg struct {
		task       vimkunja.Task
		closeAfter bool
		err        error
	}
	cursorBlinkMsg struct{}
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
		// A footer error is transient feedback. The full error remains available
		// in the session log for the future :errors pane.
		m.lastErr = ""

		k := key.String()
		kLower := strings.ToLower(k)

		// Quit
		if kLower == "ctrl+c" {
			return m, tea.Quit
		}

		if m.commandMode {
			return m.updateCommandModeKey(key)
		}
		// In editor Insert/Picker modes ':' is field input. In Detail Normal-mode
		// it still opens the shared Ex command line.
		if m.focus == paneDetail && m.editor != nil && (m.editor.mode != editorNormal || k != ":") && kLower != "ctrl+w" {
			return m.updateDetailKeys(key)
		}
		if k == ":" {
			m.commandMode = true
			m.commandInput = ""
			return m, nil
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
			m.gArmed = false
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
		case paneHelp:
			return m.updateHelpKeys(key)
		case paneInfo:
			return m.updateInfoKeys(key)
		}
		return m, nil
	}

	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		return m, nil

	case projectsLoadedMsg:
		m.projectLoading = false
		if typed.err != nil {
			m.recordError(typed.err)
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
			m.recordError(typed.err)
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

	case cursorBlinkMsg:
		if m.focus == paneDetail && m.editor != nil && m.editor.mode == editorInsert {
			m.cursorVisible = !m.cursorVisible
			return m, cursorBlinkCmd()
		}
		return m, nil

	case taskSavedMsg:
		m.taskLoading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.pendingTaskID = typed.task.ID
		if typed.closeAfter {
			m.editor = nil
		} else if m.editor != nil {
			m.editor.id = typed.task.ID
			m.editor.draft.ID = typed.task.ID
			m.editor.original = m.editor.draft
			m.editor.mode = editorNormal
		}
		m.showDetail = !typed.closeAfter
		m.focus = paneDetail
		if typed.closeAfter {
			m.focus = paneTasks
		}
		if idx := m.findProjectIndex(typed.task.ProjectID); idx >= 0 {
			m.projectSelected = idx
		}
		m.currentProjectID = typed.task.ProjectID
		m.currentViewID = 0
		m.viewLoading = true
		return m, m.loadViews(typed.task.ProjectID)

	case taskToggledMsg:
		if typed.err != nil {
			m.taskLoading = false
			m.recordError(typed.err)
			return m, nil
		}
		if m.currentProjectID == 0 || m.currentViewID == 0 {
			m.taskLoading = false
			return m, nil
		}
		return m, m.loadTasks(m.currentProjectID, m.currentViewID)

	case tasksLoadedMsg:
		m.taskLoading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.tasks = typed.tasks
		m.taskSelected = clamp(m.taskSelected, 0, max(0, len(m.tasks)-1))
		if m.pendingTaskID != 0 {
			for i, task := range m.tasks {
				if task.ID == m.pendingTaskID {
					m.taskSelected = i
					break
				}
			}
			m.pendingTaskID = 0
		}
		return m, nil
	}

	return m, nil
}

// recordError makes errors visible in the current session and preserves them
// for the future :errors pane.
func (m *model) recordError(err error) {
	if err == nil {
		return
	}
	m.lastErr = err.Error()
	apperrors.Log(m.lastErr)
	if m.infoPane != nil && m.infoPane.errorLog {
		m.infoPane.lines = errorLogLines()
	}
}

func (m *model) updateCommandModeKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(key.String()) {
	case "esc":
		m.commandMode = false
		m.commandInput = ""
		return m, nil
	case "backspace":
		input := []rune(m.commandInput)
		if len(input) > 0 {
			m.commandInput = string(input[:len(input)-1])
		}
		return m, nil
	case "enter":
		input := m.commandInput
		m.commandMode = false
		m.commandInput = ""

		updated, cmd, err := m.commands.Execute(m, input)
		if err != nil {
			m.recordError(err)
			return m, nil
		}
		return updated, cmd
	}

	if text := key.Key().Text; text != "" {
		m.commandInput += text
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
	if m.focus == paneHelp || m.focus == paneInfo {
		return paneTasks
	}
	n := m.neighbors[m.focus]
	if n.up != nil {
		return *n.up
	}
	return m.focus
}

func (m *model) focusNeighborDownOrStay() paneID {
	if m.focus == paneProjects || m.focus == paneTasks {
		if m.showDetail {
			return paneDetail
		}
		if m.helpPane != nil {
			return paneHelp
		}
		if m.infoPane != nil {
			return paneInfo
		}
	}
	n := m.neighbors[m.focus]
	if n.down != nil {
		return *n.down
	}
	return m.focus
}

func (m *model) updateProjectsKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := normalizeArrow(k.String())

	// `gg` prefix: first g arms, second g jumps to top.
	if m.gArmed {
		m.gArmed = false
		if s == "g" {
			m.projectSelected = 0
			return m, m.onProjectSelectionChanged()
		}
		// otherwise fall through with the resolved key
	}

	last := max(0, len(m.projects)-1)
	switch s {
	case "q":
		return m, tea.Quit
	case "enter":
		// Confirm current project selection and jump to the tasks pane.
		m.focus = paneTasks
		return m, nil
	case "o":
		m.openNewEditor()
		return m, nil
	case "r":
		m.lastErr = ""
		m.projectLoading = true
		return m, m.loadProjects(1)
	case "g":
		m.gArmed = true
		return m, nil
	case "G":
		m.projectSelected = last
		return m, m.onProjectSelectionChanged()
	case "ctrl+d":
		m.projectSelected = clamp(m.projectSelected+m.pageStep(), 0, last)
		return m, m.onProjectSelectionChanged()
	case "ctrl+u":
		m.projectSelected = clamp(m.projectSelected-m.pageStep(), 0, last)
		return m, m.onProjectSelectionChanged()
	case "j":
		m.projectSelected = clamp(m.projectSelected+1, 0, last)
		return m, m.onProjectSelectionChanged()
	case "k":
		m.projectSelected = clamp(m.projectSelected-1, 0, last)
		return m, m.onProjectSelectionChanged()
	}

	return m, nil
}

func (m *model) updateTasksKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := normalizeArrow(k.String())

	// `gg` prefix: first g arms, second g jumps to top.
	if m.gArmed {
		m.gArmed = false
		if s == "g" {
			m.taskSelected = 0
			return m, nil
		}
	}

	last := max(0, len(m.tasks)-1)
	switch s {
	case "q":
		return m, tea.Quit
	case "enter":
		if m.taskSelected >= 0 && m.taskSelected < len(m.tasks) {
			m.openExistingEditor(m.tasks[m.taskSelected])
		}
		return m, nil
	case "o":
		m.openNewEditor()
		return m, nil
	case "r":
		m.lastErr = ""
		if m.currentProjectID != 0 && m.currentViewID != 0 {
			m.taskLoading = true
			return m, m.loadTasks(m.currentProjectID, m.currentViewID)
		}
		return m, nil
	case "g":
		m.gArmed = true
		return m, nil
	case "G":
		m.taskSelected = last
		return m, nil
	case "ctrl+d":
		m.taskSelected = clamp(m.taskSelected+m.pageStep(), 0, last)
		return m, nil
	case "ctrl+u":
		m.taskSelected = clamp(m.taskSelected-m.pageStep(), 0, last)
		return m, nil
	case "j":
		m.taskSelected = clamp(m.taskSelected+1, 0, last)
		return m, nil
	case "k":
		m.taskSelected = clamp(m.taskSelected-1, 0, last)
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
				return taskToggledMsg{err: m.store.ToggleTaskDone(ctx, t.ID, !t.Done)}
			}
		}
	}

	return m, nil
}

// updateHelpKeys intentionally has no close shortcut. Help is a normal pane:
// switch focus with Ctrl+W and close it explicitly with :q.
func (m *model) updateHelpKeys(tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *model) updateInfoKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "esc":
		m.infoPane = nil
		m.focus = paneTasks
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
	w, h := m.width, m.height
	if w < 40 {
		w = 80
	}
	if h < 10 {
		h = 24
	}

	// chrome: border(1 each side) + padding(0,1) -> 4 cols horizontal, 2 rows vertical
	const chromeX = 4
	const chromeY = 2

	statusBarH := 3 // border(2) + 1 content row
	contentH := h - statusBarH
	if contentH < 4 {
		contentH = 4
	}

	// Split top row (panes) and optional detail row.
	topH := contentH
	detailH := 0
	if m.showDetail || m.helpPane != nil || m.infoPane != nil {
		topH = contentH / 2
		if topH < 4 {
			topH = 4
		}
		detailH = contentH - topH
		if detailH < 4 {
			detailH = 4
		}
	}

	// Horizontal split: projects pane fixed-ish, tasks pane fills the rest.
	projectsOuter := 30
	if projectsOuter > w/3 {
		projectsOuter = w / 3
	}
	if projectsOuter < 16 {
		projectsOuter = 16
	}
	tasksOuter := w - projectsOuter
	if tasksOuter < 16 {
		tasksOuter = 16
	}

	leftTextW := projectsOuter - chromeX
	rightTextW := tasksOuter - chromeX
	topTextH := topH - chromeY

	left := renderProjects(m, leftTextW, topTextH)
	right := renderTasks(m, rightTextW, topTextH)

	leftStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Height(topH - 2)
	rightStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Height(topH - 2)

	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(left),
		rightStyle.Render(right),
	)

	content := panes
	if m.showDetail || m.helpPane != nil || m.infoPane != nil {
		var bottom string
		switch {
		case m.showDetail:
			bottom = renderTaskDetail(m, w-chromeX, detailH-chromeY)
		case m.helpPane != nil:
			bottom = renderInfoPane(m.helpPane, w-chromeX, detailH-chromeY)
		default:
			bottom = renderInfoPane(m.infoPane, w-chromeX, detailH-chromeY)
		}
		bottomStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(0, 1).
			Height(detailH - 2)
		content = lipgloss.JoinVertical(lipgloss.Top,
			panes,
			bottomStyle.Render(bottom),
		)
	}

	status := m.renderStatusBar(w)
	return lipgloss.JoinVertical(lipgloss.Top, content, status)
}

func (m *model) renderStatusBar(w int) string {
	info := ""
	switch m.focus {
	case paneProjects:
		info = "[Projects]"
	case paneTasks:
		info = "[Tasks]"
	case paneDetail:
		info = "[Detail]"
	case paneHelp:
		info = "[Help]"
	case paneInfo:
		info = "[Info]"
	}

	hint := "Ctrl+W cycles  •  Enter detail  •  q quit  •  r refresh  •  j/k navigate  •  x toggle"
	if (m.focus == paneProjects || m.focus == paneTasks) && w >= 100 {
		hint += "  •  o new task"
	}
	if m.focus == paneDetail {
		mode := "NORMAL"
		if m.editor != nil {
			switch m.editor.mode {
			case editorInsert:
				mode = "INSERT"
			case editorProjectPicker, editorPriorityPicker:
				mode = "INSERT"
			}
			if m.editor.dirty() {
				mode += " [+]"
			}
		}
		hint = "[" + mode + "]  i edit  •  :w save  •  :q close  •  Ctrl+W switch pane"
	}
	if m.focus == paneHelp {
		hint = "Ctrl+W k tasks  •  :q close pane"
	}
	if m.focus == paneInfo {
		hint = "q/esc close pane"
	}
	if m.commandMode {
		hint = fmt.Sprintf(":%s_", m.commandInput)
	}
	statusStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(w - 2)
	if m.lastErr != "" {
		hint = fmt.Sprintf("Error: %s", m.lastErr)
		statusStyle = statusStyle.Foreground(lipgloss.Color("196")).BorderForeground(lipgloss.Color("196"))
	}

	return statusStyle.Render(fmt.Sprintf("%s   %s", info, hint))
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

// normalizeArrow maps arrow-key strings to their Vim-style hjkl synonyms so both
// work identically within list panes (without advertising the arrows anywhere).
func normalizeArrow(s string) string {
	switch s {
	case "up":
		return "k"
	case "down":
		return "j"
	case "left":
		return "h"
	case "right":
		return "l"
	}
	return s
}

// pageStep returns roughly half the visible rows of a list pane, used for
// Vim-style Ctrl-D / Ctrl-U half-page jumps.
func (m *model) pageStep() int {
	h := m.height
	if h < 10 {
		h = 24
	}
	contentH := h - 3 // status bar
	if contentH < 4 {
		contentH = 4
	}
	if m.showDetail || m.helpPane != nil || m.infoPane != nil {
		contentH /= 2
	}
	step := (contentH - 2) / 2 // minus border; half a page
	if step < 1 {
		step = 1
	}
	return step
}
