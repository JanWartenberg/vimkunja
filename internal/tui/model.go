package tui

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	apperrors "github.com/vimkunja/vimkunja/internal/errors"
	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type paneID int

type projectViewPreference int

const (
	projectViewList projectViewPreference = iota
	projectViewKanban
)

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

	InitialProjectID    int64
	InitialViewID       int64
	LeaderTimeout       time.Duration
	CtrlWTimeout        time.Duration
	DisableCursorShapes bool
}

type model struct {
	store               vimkunja.Store
	perPage             int
	disableCursorShapes bool
	now                 func() time.Time
	clipboardRead       func() (string, error)

	width  int
	height int

	focus                    paneID
	ctrlWArmed               bool
	ctrlWTimeout             time.Duration
	leaderArmed              bool
	viewSwitchArmed          bool
	deleteArmed              bool
	deleteConfirm            bool
	deleteTaskID             int64
	projectsPaneCollapsed    bool
	projectsPaneAutoShown    bool
	lastProjectPaneCollapsed bool
	leaderTimeout            time.Duration
	gArmed                   bool
	zArmed                   bool
	neighbors                map[paneID]paneNeighbors

	lastErr string
	toast   string

	commandMode  bool
	commandInput string
	commands     *CommandRegistry

	searchOpen         bool
	searchMode         searchInputMode
	searchCommandMode  bool
	searchCommandInput string
	searchGArmed       bool
	searchInput        string
	searchResults      []vimkunja.SearchResult
	searchSelected     int
	searchViewport     int
	searchRequest      uint64
	searchLoading      bool
	searchOrigin       *searchOrigin
	searchReturn       *searchReturn
	labelUI            *labelUI

	// Projects pane
	projects             []vimkunja.Project
	projectTree          projectTree
	projectSelected      int
	projectViewportStart int
	projectLoading       bool

	// Views (derived from selected project)
	projectViews           []vimkunja.ProjectView
	viewSelected           int
	viewLoading            bool
	currentViewID          int64
	projectViewPreference  projectViewPreference
	kanbanActive           bool
	kanbanBoard            vimkunja.KanbanBoard
	kanbanBucketSelected   int
	kanbanTaskSelected     int
	kanbanViewportStart    int
	kanbanPicker           bool
	kanbanPickerIndex      int
	kanbanProvisionConfirm bool
	pendingKanbanTaskID    int64
	pendingKanbanBucketID  int64

	// Tasks pane
	tasks          []vimkunja.Task
	taskSelected   int
	taskLoading    bool
	taskScope      taskScope
	overviewTasks  []vimkunja.Task
	overviewLabel  vimkunja.Label
	overviewFilter overviewFilter
	overviewSort   overviewSort
	overviewPrompt overviewPrompt

	// Detail and auxiliary panes
	showDetail bool
	helpPane   *infoPane
	infoPane   *infoPane

	currentProjectID          int64
	pendingTaskID             int64
	pendingSearchDetailTaskID int64
	quitAfterSave             bool
	editor                    *taskEditor
	cursorVisible             bool

	initialProjectID int64
	initialViewID    int64
}

func NewModel(cfg Config) tea.Model {
	leaderTimeout := cfg.LeaderTimeout
	if leaderTimeout <= 0 {
		leaderTimeout = DefaultLeaderTimeout
	}
	ctrlWTimeout := cfg.CtrlWTimeout
	if ctrlWTimeout <= 0 {
		ctrlWTimeout = DefaultCtrlWTimeout
	}
	m := &model{
		store:               cfg.Store,
		perPage:             max(1, cfg.PerPage),
		disableCursorShapes: cfg.DisableCursorShapes,
		now:                 time.Now,
		initialProjectID:    cfg.InitialProjectID,
		leaderTimeout:       leaderTimeout,
		ctrlWTimeout:        ctrlWTimeout,
		initialViewID:       cfg.InitialViewID,
		projectSelected:     0,
		currentViewID:       cfg.InitialViewID,
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
	projectMovedMsg struct {
		err error
	}
	viewsLoadedMsg struct {
		views []vimkunja.ProjectView
		err   error
	}
	tasksLoadedMsg struct {
		tasks []vimkunja.Task
		err   error
	}
	kanbanBoardLoadedMsg struct {
		board vimkunja.KanbanBoard
		err   error
	}
	kanbanProvisionedMsg struct {
		board vimkunja.KanbanBoard
		err   error
	}
	kanbanTaskMovedMsg struct {
		taskID   int64
		bucketID int64
		err      error
	}
	taskToggledMsg struct {
		err error
	}
	taskDeletedMsg struct {
		err error
	}
	workflowStatusLoadedMsg struct {
		taskID int64
		status vimkunja.WorkflowStatus
		err    error
	}
	searchLoadedMsg struct {
		request uint64
		query   string
		results []vimkunja.SearchResult
		err     error
	}
	labelsLoadedMsg struct {
		labels []vimkunja.Label
		err    error
	}
	taskLabelsLoadedMsg struct {
		taskID int64
		labels []vimkunja.Label
		err    error
	}
	taskLabelsSavedMsg struct {
		taskID     int64
		labelIDs   []int64
		closeAfter bool
		err        error
	}
	labelMutatedMsg struct {
		mutation labelMutation
		label    vimkunja.Label
		err      error
	}
	taskSavedMsg struct {
		task       vimkunja.Task
		closeAfter bool
		err        error
	}
	cursorBlinkMsg   struct{}
	leaderTimeoutMsg struct{}
	ctrlWTimeoutMsg  struct{}
)

func (m *model) Init() tea.Cmd {
	m.projectLoading = true
	return m.loadProjects()
}

func (m *model) loadProjects() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		items, err := m.store.ListProjects(ctx)
		return projectsLoadedMsg{projects: items, err: err}
	}
}

func (m *model) moveProject(projectID int64, direction vimkunja.ProjectMoveDirection) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		return projectMovedMsg{err: m.store.MoveProjectAmongSiblings(ctx, projectID, direction)}
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

func (m *model) loadWorkflowStatus(projectID, taskID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		status, err := m.store.GetTaskWorkflowStatus(ctx, projectID, taskID)
		return workflowStatusLoadedMsg{taskID: taskID, status: status, err: err}
	}
}

func (m *model) loadKanbanBoard(projectID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		board, err := m.store.LoadManualKanbanBoard(ctx, projectID)
		return kanbanBoardLoadedMsg{board: board, err: err}
	}
}

func (m *model) moveKanbanTask(taskID, bucketID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		return kanbanTaskMovedMsg{taskID: taskID, bucketID: bucketID, err: m.store.SetTaskWorkflowStatus(ctx, taskID, bucketID)}
	}
}

func (m *model) provisionStandardKanbanView(projectID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		board, err := m.store.ProvisionStandardManualKanbanView(ctx, projectID)
		return kanbanProvisionedMsg{board: board, err: err}
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
		// Windows input can report a key release after its press. Releases must
		// not consume a pending multi-key command such as Ctrl+W h.
		if _, released := key.(tea.KeyReleaseMsg); released {
			return m, nil
		}

		// A footer error is transient feedback. The full error remains available
		// in the session log for the future :errors pane.
		m.lastErr = ""

		k := key.String()
		kLower := strings.ToLower(k)

		if !m.searchOpen && kLower == "ctrl+o" && m.searchReturn != nil {
			return m, m.restoreLastSearch()
		}

		// Quit
		if kLower == "ctrl+c" {
			return m, tea.Quit
		}

		if m.searchOpen {
			return m.updateSearchKeys(key)
		}
		if m.labelUI != nil {
			return m.updateLabelUI(key)
		}
		if m.commandMode {
			return m.updateCommandModeKey(key)
		}
		if m.deleteConfirm {
			return m.updateDeleteConfirmation(k)
		}
		if m.kanbanProvisionConfirm {
			return m.updateKanbanProvisionConfirmation(k)
		}
		if m.viewSwitchArmed {
			m.viewSwitchArmed = false
			switch k {
			case "l":
				return listCommand(m, nil)
			case "k":
				return kanbanCommand(m, nil)
			case "t", "g":
				return notYetImplementedCommand(m, nil)
			}
			return m, nil
		}
		if m.leaderArmed {
			m.leaderArmed = false
			switch k {
			case "f":
				m.openSearch()
			case "l":
				if m.focus == paneDetail && m.editor != nil {
					return m, m.openTaskLabelPicker()
				}
			case "v":
				m.viewSwitchArmed = true
			case "o":
				m.toggleProjectsPane()
			case "a":
				if m.inOverview() {
					return m, m.restoreProjectScope()
				}
				return m, m.openOverview()
			}
			return m, nil
		}
		if m.leaderEligible() && isSpaceKey(key) {
			m.leaderArmed = true
			return m, tea.Tick(m.leaderTimeout, func(time.Time) tea.Msg { return leaderTimeoutMsg{} })
		}

		// Ctrl+W must be handled before Detail input. Otherwise the direction key
		// is consumed by the editor and cannot move focus out of the Detail pane.
		if m.ctrlWArmed {
			m.ctrlWArmed = false

			// Ctrl+W + Ctrl+W cycles focus.
			if kLower == "ctrl+w" || kLower == "w" {
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
			return m, tea.Tick(m.ctrlWTimeout, func(time.Time) tea.Msg { return ctrlWTimeoutMsg{} })
		}

		// In editor Insert/Picker modes ':' is field input. In Detail Normal-mode
		// it still opens the shared Ex command line.
		if m.focus == paneDetail && m.editor != nil && (m.editor.mode != editorNormal || k != ":") {
			return m.updateDetailKeys(key)
		}
		if k == ":" {
			m.commandMode = true
			m.commandInput = ""
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

		oldRows := m.projectTree.visibleRows()
		oldIndex := m.projectSelected
		oldSelectedID := int64(0)
		if oldIndex >= 0 && oldIndex < len(oldRows) {
			oldSelectedID = oldRows[oldIndex].Project.ID
		} else if m.currentProjectID != 0 {
			oldSelectedID = m.currentProjectID
		}
		collapsed := m.projectTree.collapsedState()
		m.projects = typed.projects
		m.projectTree = newProjectTree(m.projects, collapsed)
		if len(m.projects) == 0 {
			m.projectSelected = 0
			m.projectViewportStart = 0
			m.currentProjectID = 0
			m.currentViewID = 0
			m.projectViews = nil
			m.viewLoading = false
			m.tasks = nil
			m.taskLoading = false
			m.kanbanActive = false
			m.kanbanBoard = vimkunja.KanbanBoard{}
			m.lastErr = "No projects returned."
			return m, nil
		}

		targetID := oldSelectedID
		if m.initialProjectID != 0 && len(oldRows) == 0 {
			targetID = m.initialProjectID
		}
		m.restoreProjectSelection(targetID, oldRows, oldIndex)
		selected, _ := m.selectedProject()
		m.currentProjectID = selected.ID

		// Initial load: either direct view tasks or derive list view.
		if m.initialViewID != 0 {
			m.currentViewID = m.initialViewID
			m.taskLoading = true
			return m, m.loadTasks(m.currentProjectID, m.currentViewID)
		}

		m.viewLoading = true
		return m, m.loadViews(m.currentProjectID)

	case projectMovedMsg:
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		return m, m.loadProjects()

	case viewsLoadedMsg:
		m.viewLoading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.projectViews = typed.views
		if m.kanbanActive {
			for i, view := range m.projectViews {
				if view.ID == m.currentViewID {
					m.viewSelected = i
					return m, nil
				}
			}
		}
		if m.projectViewPreference == projectViewKanban {
			if kanbanView := m.pickManualKanbanViewIndex(); kanbanView >= 0 {
				m.viewSelected = kanbanView
				m.currentViewID = m.projectViews[kanbanView].ID
				m.taskLoading = true
				return m, m.loadKanbanBoard(m.currentProjectID)
			}
		}
		m.viewSelected = m.pickListViewIndex()
		if m.viewSelected < 0 {
			m.lastErr = "No 'list' views found for the selected project."
			return m, nil
		}
		m.currentViewID = m.projectViews[m.viewSelected].ID
		if m.inOverview() {
			return m, nil
		}

		m.taskLoading = true
		return m, m.loadTasks(m.currentProjectID, m.currentViewID)

	case leaderTimeoutMsg:
		m.leaderArmed = false
		return m, nil

	case ctrlWTimeoutMsg:
		m.ctrlWArmed = false
		return m, nil

	case labelsLoadedMsg:
		if m.labelUI == nil {
			return m, nil
		}
		m.labelUI.loading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.labelUI.labels = typed.labels
		if m.labelUI.createSelected && !m.labelUI.showsCreateOption() {
			m.labelUI.createSelected = false
		}
		m.labelUI.selected = clamp(m.labelUI.selected, 0, max(0, len(m.labelUI.matches())-1))
		return m, nil

	case taskLabelsLoadedMsg:
		if m.editor == nil || m.editor.id != typed.taskID {
			return m, nil
		}
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.editor.labelsLoading = false
		if m.editor.draft.LabelIDsSet {
			return m, nil
		}
		m.editor.labels = append([]vimkunja.Label(nil), typed.labels...)
		m.editor.draft.LabelIDs = editorLabelIDs(m.editor.labels)
		m.editor.draft.LabelIDsSet = false
		m.editor.original = cloneTaskDraft(m.editor.draft)
		return m, nil

	case taskLabelsSavedMsg:
		if m.labelUI != nil {
			m.labelUI.loading = false
		}
		if m.editor == nil || m.editor.id != typed.taskID {
			return m, nil
		}
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.editor.draft.LabelIDs = append([]int64(nil), typed.labelIDs...)
		m.editor.draft.LabelIDsSet = false
		m.editor.original.LabelIDs = append([]int64(nil), typed.labelIDs...)
		m.editor.original.LabelIDsSet = false
		if m.labelUI != nil && m.labelUI.kind == labelUITaskPicker {
			m.labelUI.labelSnapshot = append([]vimkunja.Label(nil), m.editor.labels...)
			m.labelUI.labelIDsSnapshot = append([]int64(nil), typed.labelIDs...)
			m.labelUI.labelsSetAtOpen = false
			if typed.closeAfter {
				m.closeLabelUI()
			}
		}
		return m, nil

	case labelMutatedMsg:
		if m.labelUI == nil {
			return m, nil
		}
		m.labelUI.loading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		if typed.mutation == labelCreatedForTask {
			m.toggleEditorLabel(typed.label)
			m.clearTaskLabelFilter()
		} else {
			m.labelUI.input, m.labelUI.inputMode, m.labelUI.deleteConfirm = "", false, false
		}
		return m, m.loadLabels()

	case searchLoadedMsg:
		if !m.searchOpen || typed.request != m.searchRequest || typed.query != m.searchInput {
			return m, nil
		}
		m.searchLoading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.searchResults = typed.results
		m.searchSelected = clamp(m.searchSelected, 0, max(0, len(m.searchResults)-1))
		m.searchViewport = clamp(m.searchViewport, 0, max(0, len(m.searchResults)-1))
		return m, nil

	case tea.PasteMsg:
		if m.focus == paneDetail && m.editor != nil && m.editor.mode == editorInsert {
			m.insertEditorText(typed.Content)
		}
		return m, nil

	case clipboardPastedMsg:
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		if m.focus == paneDetail && m.editor == typed.editor && m.editor.mode == editorInsert {
			m.insertEditorText(typed.text)
		}
		return m, nil

	case cursorBlinkMsg:
		if m.focus == paneDetail && m.editor != nil && m.editor.mode == editorInsert {
			m.cursorVisible = !m.cursorVisible
			return m, cursorBlinkCmd()
		}
		return m, nil

	case workflowStatusLoadedMsg:
		if m.editor != nil && m.editor.id == typed.taskID {
			m.editor.workflowStatusLoading = false
			if typed.err == nil {
				m.editor.workflowStatus = typed.status
				m.editor.draft.WorkflowBucketID = typed.status.CurrentBucket.ID
				m.editor.original.WorkflowBucketID = typed.status.CurrentBucket.ID
			}
		}
		if typed.err != nil {
			m.recordError(typed.err)
		}
		return m, nil

	case taskSavedMsg:
		m.taskLoading = false
		quitAfterSave := m.quitAfterSave
		m.quitAfterSave = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		if quitAfterSave {
			return m, tea.Quit
		}
		wasOverview := m.inOverview()
		wasKanban := m.kanbanActive && !wasOverview
		m.pendingTaskID = typed.task.ID
		if typed.closeAfter {
			m.closeTaskDetail()
		} else if m.editor != nil {
			m.editor.id = typed.task.ID
			m.editor.draft.ID = typed.task.ID
			m.editor.draft.Done = typed.task.Done
			m.editor.draft.DoneSet = false
			m.editor.original = cloneTaskDraft(m.editor.draft)
			m.editor.mode = editorNormal
			m.showDetail = true
			m.focus = paneDetail
		}
		if m.labelUI != nil && m.labelUI.kind == labelUITaskPicker && m.labelUI.closeAfterSave {
			m.closeLabelUI()
		}
		if idx := m.findProjectIndex(typed.task.ProjectID); idx >= 0 {
			m.projectSelected = idx
		}
		m.currentProjectID = typed.task.ProjectID
		if wasKanban {
			m.pendingKanbanTaskID = typed.task.ID
			return m, m.loadKanbanBoard(typed.task.ProjectID)
		}
		m.currentViewID = 0
		m.viewLoading = true
		if wasOverview {
			return m, tea.Batch(m.loadViews(typed.task.ProjectID), m.refreshActiveTasks())
		}
		return m, m.loadViews(typed.task.ProjectID)

	case taskDeletedMsg:
		m.deleteConfirm = false
		m.deleteTaskID = 0
		if typed.err != nil {
			m.taskLoading = false
			m.recordError(typed.err)
			return m, nil
		}
		return m, m.refreshActiveTasks()

	case taskToggledMsg:
		if typed.err != nil {
			m.taskLoading = false
			m.recordError(typed.err)
			return m, nil
		}
		return m, m.refreshActiveTasks()

	case kanbanBoardLoadedMsg:
		m.taskLoading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		if typed.board.Availability == vimkunja.KanbanBoardUnavailable {
			m.kanbanActive = false
			m.kanbanProvisionConfirm = true
			return m, nil
		}
		if typed.board.Availability == vimkunja.KanbanBoardAmbiguous {
			m.kanbanActive = false
			m.recordError(fmt.Errorf("multiple manual Kanban Views are not supported"))
			return m, nil
		}
		m.setKanbanBoard(typed.board)
		return m, nil

	case kanbanProvisionedMsg:
		m.taskLoading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.setKanbanBoard(typed.board)
		return m, m.loadViews(m.currentProjectID)

	case kanbanTaskMovedMsg:
		if typed.err != nil {
			m.taskLoading = false
			m.recordError(typed.err)
			return m, nil
		}
		m.pendingKanbanTaskID = typed.taskID
		m.pendingKanbanBucketID = typed.bucketID
		m.taskLoading = true
		return m, m.loadKanbanBoard(m.currentProjectID)

	case overviewTasksLoadedMsg:
		if !m.inOverview() {
			return m, nil
		}
		m.taskLoading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.setOverviewTasks(typed.tasks)
		return m, nil

	case tasksLoadedMsg:
		if m.inOverview() {
			return m, nil
		}
		m.taskLoading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.kanbanActive = false
		m.kanbanBoard = vimkunja.KanbanBoard{}
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
		if m.pendingSearchDetailTaskID != 0 {
			taskID := m.pendingSearchDetailTaskID
			m.pendingSearchDetailTaskID = 0
			for _, task := range m.tasks {
				if task.ID == taskID {
					m.openExistingEditor(task)
					return m, tea.Batch(m.loadWorkflowStatus(task.ProjectID, task.ID), m.loadEditorLabels(task.ID))
				}
			}
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

func isSpaceKey(key tea.KeyMsg) bool {
	return key.String() == " " || key.String() == "space" || key.Key().Text == " "
}

// leaderEligible keeps Space available for field and picker input.
func (m *model) leaderEligible() bool {
	return m.focus != paneDetail || m.editor == nil || m.editor.mode == editorNormal
}

func (m *model) toggleProjectsPane() {
	m.projectsPaneCollapsed = !m.projectsPaneCollapsed
	m.projectsPaneAutoShown = false
	if m.projectsPaneCollapsed && m.focus == paneProjects {
		m.focus = paneTasks
	}
}

func (m *model) nextFocus() paneID {
	open := []paneID{paneTasks}
	if !m.projectsPaneCollapsed {
		open = append([]paneID{paneProjects}, open...)
	}
	if m.showDetail {
		open = append(open, paneDetail)
	} else if m.helpPane != nil {
		open = append(open, paneHelp)
	} else if m.infoPane != nil {
		open = append(open, paneInfo)
	}
	for i, pane := range open {
		if pane == m.focus {
			return open[(i+1)%len(open)]
		}
	}
	return open[0]
}

func (m *model) focusNeighborLeftOrStay() paneID {
	if m.projectsPaneCollapsed {
		return m.focus
	}
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
		return m.focus
	}
	n := m.neighbors[m.focus]
	if n.down != nil {
		return *n.down
	}
	return m.focus
}

func (m *model) updateProjectsKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := normalizeArrow(k.String())

	if m.zArmed {
		m.zArmed = false
		selected, ok := m.selectedProject()
		if !ok {
			return m, nil
		}
		switch s {
		case "o":
			m.projectTree.expand(selected.ID)
		case "c":
			m.projectTree.collapse(selected.ID)
		case "a":
			m.projectTree.toggle(selected.ID)
		case "M":
			rootID := selected.ID
			for {
				parentID, exists := m.projectTree.parentID(rootID)
				if !exists {
					break
				}
				rootID = parentID
			}
			m.projectTree.collapseAll()
			m.projectSelected = m.findVisibleProjectIndex(rootID)
		case "R":
			m.projectTree.expandAll()
			m.projectSelected = m.findVisibleProjectIndex(selected.ID)
		}
		return m, nil
	}

	// `gg` prefix: first g arms, second g jumps to top.
	if m.gArmed {
		m.gArmed = false
		if s == "g" {
			m.projectSelected = 0
			return m, m.onProjectSelectionChanged()
		}
		// otherwise fall through with the resolved key
	}

	rows := m.projectRows()
	last := max(0, len(rows)-1)
	switch s {
	case "q":
		return m, tea.Quit
	case "enter":
		// Confirm current Project and replace Overview with its Tasks Pane.
		m.focus = paneTasks
		if m.inOverview() {
			m.taskScope = scopeProject
			return m, m.onProjectSelectionChanged()
		}
		return m, nil
	case "o":
		m.openNewEditor()
		return m, nil
	case "-":
		return m, m.openOverview()
	case "r":
		m.lastErr = ""
		m.projectLoading = true
		return m, m.loadProjects()
	case "g":
		m.gArmed = true
		return m, nil
	case "z":
		m.zArmed = true
		return m, nil
	case "h":
		if m.projectSelected < 0 || m.projectSelected >= len(rows) {
			return m, nil
		}
		row := rows[m.projectSelected]
		if row.HasChildren && row.Expanded {
			m.projectTree.collapse(row.Project.ID)
			return m, nil
		}
		if parentID, ok := m.projectTree.parentID(row.Project.ID); ok {
			m.projectSelected = m.findVisibleProjectIndex(parentID)
			return m, m.onProjectSelectionChanged()
		}
		return m, nil
	case "l":
		if m.projectSelected < 0 || m.projectSelected >= len(rows) {
			return m, nil
		}
		row := rows[m.projectSelected]
		if row.HasChildren && !row.Expanded {
			m.projectTree.expand(row.Project.ID)
			return m, nil
		}
		if childID, ok := m.projectTree.firstChildID(row.Project.ID); ok {
			m.projectSelected = m.findVisibleProjectIndex(childID)
			return m, m.onProjectSelectionChanged()
		}
		return m, nil
	case "J", "K":
		if m.projectSelected < 0 || m.projectSelected >= len(rows) || rows[m.projectSelected].Recovery {
			return m, nil
		}
		direction := vimkunja.ProjectMoveNext
		if s == "K" {
			direction = vimkunja.ProjectMovePrevious
		}
		return m, m.moveProject(rows[m.projectSelected].Project.ID, direction)
	case "G", "end":
		m.projectSelected = last
		return m, m.onProjectSelectionChanged()
	case "home":
		m.projectSelected = 0
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
	if m.kanbanActive && !m.inOverview() {
		return m.updateKanbanKeys(k)
	}
	s := normalizeArrow(k.String())
	if m.inOverview() && m.updateOverviewPrompt(s) {
		return m, nil
	}
	if m.deleteArmed {
		m.deleteArmed = false
		if s == "d" {
			m.beginTaskDeletion()
		}
		return m, nil
	}

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
			task := m.tasks[m.taskSelected]
			if m.inOverview() {
				if idx := m.findProjectIndex(task.ProjectID); idx >= 0 {
					m.projectSelected = idx
				}
			}
			m.openExistingEditor(task)
			return m, tea.Batch(m.loadWorkflowStatus(task.ProjectID, task.ID), m.loadEditorLabels(task.ID))
		}
		return m, nil
	case "o":
		if m.inOverview() {
			m.openOverviewNewEditor()
		} else {
			m.openNewEditor()
		}
		return m, nil
	case "r":
		m.lastErr = ""
		return m, m.refreshActiveTasks()
	case "-":
		if !m.inOverview() {
			return m, m.openOverview()
		}
		return m, nil
	case "s":
		if m.inOverview() {
			m.overviewPrompt = overviewPromptSort
		}
		return m, nil
	case "f":
		if m.inOverview() {
			m.overviewPrompt = overviewPromptFilter
		}
		return m, nil
	case "g":
		m.gArmed = true
		return m, nil
	case "G", "end":
		m.taskSelected = last
		return m, nil
	case "home":
		m.taskSelected = 0
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
	case "d":
		m.deleteArmed = true
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
func (m *model) updateHelpKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.updateInfoPaneScrollKeys(m.helpPane, k)
}

func (m *model) updateInfoKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "esc":
		m.infoPane = nil
		m.focus = paneTasks
		return m, nil
	}
	return m.updateInfoPaneScrollKeys(m.infoPane, k)
}

func (m *model) infoPaneScrollLimit(pane *infoPane) int {
	if pane == nil {
		return 0
	}
	w, h := m.viewportSize()
	contentH := max(4, h-3)
	topH := m.detailPaneTop()
	detailH := max(4, contentH-topH)
	viewportRows := max(1, detailH-2-2) // Pane chrome, heading, and blank row.
	rows, overflows := infoPaneScrollRows(pane, w-4, viewportRows)
	if !overflows {
		return 0
	}
	return len(rows) - viewportRows
}

func (m *model) updateInfoPaneScrollKeys(pane *infoPane, k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if pane == nil {
		return m, nil
	}
	limit := m.infoPaneScrollLimit(pane)
	if m.gArmed {
		m.gArmed = false
		if k.String() == "g" {
			pane.offset = 0
		}
		return m, nil
	}
	switch k.String() {
	case "g":
		m.gArmed = true
	case "G", "end":
		pane.offset = limit
	case "home":
		pane.offset = 0
	case "j", "down":
		pane.offset = clamp(pane.offset+1, 0, limit)
	case "k", "up":
		pane.offset = clamp(pane.offset-1, 0, limit)
	case "ctrl+d", "pgdown", "pagedown":
		pane.offset = clamp(pane.offset+m.pageStep(), 0, limit)
	case "ctrl+u", "pgup", "pageup":
		pane.offset = clamp(pane.offset-m.pageStep(), 0, limit)
	}
	return m, nil
}

func (m *model) projectRows() []projectTreeRow {
	if !m.projectTree.initialized() && len(m.projects) > 0 {
		m.projectTree = newProjectTree(m.projects, nil)
	}
	return m.projectTree.visibleRows()
}

func (m *model) selectedProject() (vimkunja.Project, bool) {
	rows := m.projectRows()
	if m.projectSelected < 0 || m.projectSelected >= len(rows) {
		return vimkunja.Project{}, false
	}
	return rows[m.projectSelected].Project, true
}

func (m *model) restoreProjectSelection(targetID int64, oldRows []projectTreeRow, oldIndex int) {
	if targetID != 0 {
		m.projectTree.reveal(targetID)
		if index := m.findVisibleProjectIndex(targetID); index >= 0 {
			m.projectSelected = index
			return
		}
	}
	for i := oldIndex + 1; i < len(oldRows); i++ {
		if index := m.findVisibleProjectIndex(oldRows[i].Project.ID); index >= 0 {
			m.projectSelected = index
			return
		}
	}
	for i := min(oldIndex-1, len(oldRows)-1); i >= 0; i-- {
		if index := m.findVisibleProjectIndex(oldRows[i].Project.ID); index >= 0 {
			m.projectSelected = index
			return
		}
	}
	m.projectSelected = 0
}

func (m *model) findVisibleProjectIndex(id int64) int {
	for i, row := range m.projectTree.visibleRows() {
		if row.Project.ID == id {
			return i
		}
	}
	return -1
}

func (m *model) onProjectSelectionChanged() tea.Cmd {
	m.kanbanActive = false
	m.kanbanBoard = vimkunja.KanbanBoard{}
	selected, ok := m.selectedProject()
	if !ok || m.inOverview() {
		return nil
	}
	m.currentProjectID = selected.ID
	m.currentViewID = 0
	m.tasks = nil
	m.taskSelected = 0
	m.lastErr = ""

	m.viewLoading = true
	return m.loadViews(m.currentProjectID)
}

func (m *model) findProjectIndex(id int64) int {
	if !m.projectTree.contains(id) && len(m.projects) > 0 {
		m.projectTree = newProjectTree(m.projects, m.projectTree.collapsedState())
	}
	m.projectTree.reveal(id)
	return m.findVisibleProjectIndex(id)
}

func (m *model) pickManualKanbanViewIndex() int {
	for i, view := range m.projectViews {
		if view.ViewKind == "kanban" && view.BucketConfigurationMode == "manual" {
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
	v.Cursor = m.editorCursor()
	return v
}

// editorCursor delegates cursor-shape support to Bubble Tea. Its renderer
// restores the terminal default when a previously visible cursor becomes nil.
func (m *model) nativeEditorCursorActive() bool {
	return !m.disableCursorShapes && m.focus == paneDetail && m.editor != nil
}

func (m *model) editorCursor() *tea.Cursor {
	if !m.nativeEditorCursorActive() || m.editor.field == fieldLabels || (m.editor.field == fieldDescription && descriptionHasHTML(m.editor.draft.Description)) {
		return nil
	}

	x, y := m.editorCursorPosition()
	cursor := tea.NewCursor(x, y)
	cursor.Shape = tea.CursorBlock
	if m.editor.mode == editorInsert {
		cursor.Shape = tea.CursorBar
	}
	return cursor
}

// editorCursorPosition locates the focused field in the detail pane. Picker
// fields are visibly highlighted as a whole; the native cursor sits at their
// value position without changing the rendered text.
func (m *model) editorCursorPosition() (int, int) {
	w, h := m.viewportSize()
	topH := m.detailPaneTop()
	textX := 2 // detail border plus horizontal padding
	// Text fields render the focused-field prefix before their label.
	if !m.editor.isPickerMode() {
		textX += 2
	}
	row := 2 // Project follows the header and its blank line
	labelWidth := 0

	switch m.editor.field {
	case fieldTitle:
		row, labelWidth = 3, len([]rune("Title:    "))
	case fieldPriority:
		row, labelWidth = 4, len([]rune("Priority: "))
	case fieldDueDate:
		row, labelWidth = 5, len([]rune("Due Date: "))
	case fieldLabels:
		row, labelWidth = 6, len([]rune("Labels:   "))
	case fieldWorkflowStatus:
		row, labelWidth = 7, len([]rune("Status:   "))
	case fieldDone:
		row, labelWidth = 8, len([]rune("Done:     "))
	case fieldDescription:
		return m.descriptionCursorPosition(w, h, topH)
	default:
		labelWidth = len([]rune("Project:  "))
	}

	cursorOffset := 0
	if !m.editor.isPickerMode() {
		cursorOffset = m.editor.cursor
	}
	return clamp(textX+labelWidth+cursorOffset, 0, w-1), clamp(topH+1+row, 0, h-1)
}

func (m *model) descriptionCursorPosition(w, h, topH int) (int, int) {
	textW := w - 4
	contentH := max(4, h-3)
	detailH := max(4, contentH-topH)
	textH := detailH - 2
	precedingLines := 11 // header, form fields, and Description label
	if m.editor.id == 0 {
		precedingLines-- // New Tasks omit the Done field.
	} else if textH >= 12 && !m.editor.isPickerMode() {
		precedingLines++ // Existing Tasks include the debugging identifier row.
	}
	available := max(1, textH-precedingLines)
	description := wrapDescription(m.editor.draft.Description, textW)
	if len(description) > available {
		// Match the Detail renderer: overflow is detected at full width, then
		// the rail reserves two columns before visual rows are calculated.
		description = wrapDescription(m.editor.draft.Description, max(1, textW-2))
	}
	start := 0
	if len(description) > available {
		start = clamp(m.editor.descScroll, 0, len(description)-available)
	}

	cursor := clamp(m.editor.cursor, 0, m.editor.fieldRuneLen())
	lineIndex, column := 0, 0
	for i, line := range description {
		if cursor >= line.start && cursor <= line.end {
			lineIndex, column = i, cursor-line.start
			break
		}
	}
	visibleLine := clamp(lineIndex-start, 0, available-1)
	return clamp(2+column, 0, w-1), clamp(topH+1+precedingLines+visibleLine, 0, h-1)
}

func (m *model) viewportSize() (int, int) {
	w, h := m.width, m.height
	if w < 40 {
		w = 80
	}
	if h < 10 {
		h = 24
	}
	return w, h
}

func (m *model) detailPaneTop() int {
	_, h := m.viewportSize()
	return max(4, (h-3)/2)
}

func (m *model) Render() string {
	w, h := m.viewportSize()

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
		topH = m.detailPaneTop()
		detailH = contentH - topH
		if detailH < 4 {
			detailH = 4
		}
	}

	// Horizontal split: collapsed projects leave the tasks pane full width.
	topTextH := topH - chromeY
	rightStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Height(topH - 2)
	var panes string
	if m.projectsPaneCollapsed {
		panes = rightStyle.Render(renderTasks(m, w-chromeX, topTextH))
	} else {
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
		leftStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(0, 1).
			Height(topH - 2)
		panes = lipgloss.JoinHorizontal(lipgloss.Top,
			leftStyle.Render(renderProjects(m, projectsOuter-chromeX, topTextH)),
			rightStyle.Render(renderTasks(m, tasksOuter-chromeX, topTextH)),
		)
	}

	content := panes
	if m.showDetail || m.helpPane != nil || m.infoPane != nil {
		var bottom string
		switch {
		case m.showDetail:
			bottom = renderTaskDetail(m, w-chromeX, detailH-chromeY)
		case m.helpPane != nil:
			bottom = renderInfoPane(m.helpPane, w-chromeX, detailH-chromeY, m.focus == paneHelp)
		default:
			bottom = renderInfoPane(m.infoPane, w-chromeX, detailH-chromeY, m.focus == paneInfo)
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

	if m.deleteConfirm {
		content = m.renderDeleteConfirmationModal(content, w)
	}
	if m.searchOpen {
		content = m.renderSearchOverlay(content, w)
	}
	if m.labelUI != nil {
		content = m.renderLabelOverlay(content, w)
	}
	status := m.renderStatusBar(w)
	return lipgloss.JoinVertical(lipgloss.Top, content, status)
}

// focusedScrollPosition reports the focused Pane's visual viewport position.
// Fixed Pane chrome (headers and editor fields) is excluded: only rows that
// actually move under a scrolling command contribute to the percentage.
func (m *model) focusedScrollPosition() (int, int, bool) {
	w, h := m.viewportSize()
	contentH := max(4, h-3)
	topH := contentH
	if m.showDetail || m.helpPane != nil || m.infoPane != nil {
		topH = m.detailPaneTop()
	}
	topTextH := topH - 2

	switch m.focus {
	case paneProjects:
		rows, viewportRows := m.projectRows(), max(0, topTextH-1)
		if len(rows) > viewportRows {
			limit := len(rows) - viewportRows
			return clamp(m.projectViewportStart, 0, limit), limit, true
		}
	case paneTasks:
		viewportRows := max(0, topTextH-1)
		if !m.kanbanActive && len(m.tasks) > viewportRows {
			selected := clamp(m.taskSelected, 0, len(m.tasks)-1)
			bottomMargin := min(2, max(0, viewportRows-1))
			limit := len(m.tasks) - viewportRows
			return clamp(selected-viewportRows+1+bottomMargin, 0, limit), limit, true
		}
	case paneDetail:
		if m.editor == nil || m.editor.field != fieldDescription {
			break
		}
		detailH := max(4, contentH-topH)
		textH := detailH - 2
		precedingLines := 11
		if m.editor.id == 0 {
			precedingLines--
		} else if textH >= 12 && !m.editor.isPickerMode() {
			precedingLines++
		}
		viewportRows := max(1, textH-precedingLines)
		descriptionWidth := w - 4
		contentRows := len(wrapDescription(descriptionTerminalText(m.editor.draft.Description), descriptionWidth))
		if contentRows > viewportRows {
			contentRows = len(wrapDescription(descriptionTerminalText(m.editor.draft.Description), max(1, descriptionWidth-2)))
			limit := contentRows - viewportRows
			return clamp(m.editor.descScroll, 0, limit), limit, true
		}
	case paneHelp, paneInfo:
		pane := m.helpPane
		if m.focus == paneInfo {
			pane = m.infoPane
		}
		if pane == nil {
			break
		}
		detailH := max(4, contentH-topH)
		viewportRows := max(1, detailH-2-2) // Pane chrome, heading, and blank row.
		rows, overflows := infoPaneScrollRows(pane, w-4, viewportRows)
		if overflows {
			limit := len(rows) - viewportRows
			return clamp(pane.offset, 0, limit), limit, true
		}
	}
	return 0, 0, false
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

	hint := "Ctrl+W cycles  •  Enter detail  •  q quit  •  :h help  •  j/k navigate  •  x toggle"
	if m.focus == paneProjects {
		hint = "j/k select  •  h/l fold/tree  •  z fold commands  •  Shift+J/K reorder  •  r reload  •  :h help"
	}
	if m.focus == paneTasks && m.kanbanActive && !m.inOverview() {
		hint = "Shift+H/L move bucket  •  m choose bucket  •  j/k navigate  •  x toggle  •  Ctrl+W cycles  •  :h help"
	}
	if m.focus == paneTasks && m.inOverview() {
		hint = "Enter detail  •  o new task  •  s sort  •  f filter  •  Space+a return"
	}
	if (m.focus == paneProjects || m.focus == paneTasks) && !m.inOverview() && w >= 100 {
		hint += "  •  o new task"
	}
	if m.focus == paneDetail {
		mode := "NORMAL"
		inputActive := false
		if m.editor != nil {
			switch m.editor.mode {
			case editorInsert, editorProjectPicker, editorPriorityPicker:
				mode = "INSERT"
				inputActive = true
			}
			if m.editor.dirty() {
				mode += " [+]"
			}
		}
		if inputActive {
			hint = "[" + mode + "]  Esc normal"
		} else if m.editor != nil && m.editor.field == fieldLabels && !m.editor.labelsLoading && len(m.editor.labels) > 0 {
			hint = "[" + mode + "]  h/l select Label  •  x remove Label  •  i pick/edit  •  :w save  •  :q close"
		} else {
			hint = "[" + mode + "]  i edit  •  :w save  •  :q close  •  Ctrl+W switch pane"
		}
	}
	if m.focus == paneHelp {
		hint = "Ctrl+W k tasks  •  :q close pane"
	}
	if m.focus == paneInfo {
		hint = "q/esc close pane"
	}
	if m.searchOpen {
		hint = fmt.Sprintf("SEARCH [%s]  %s", m.searchModeLabel(), m.searchInput)
		if m.searchCommandMode {
			hint = fmt.Sprintf("SEARCH COMMAND  :%s_", m.searchCommandInput)
		} else if m.searchMode == searchInsertMode {
			hint += "_  •  Esc NORMAL"
		} else {
			hint += "  •  j/k navigate  •  i/a edit  •  :q / Esc close"
		}
	}
	if m.commandMode {
		hint = fmt.Sprintf(":%s_", m.commandInput)
	}
	if m.kanbanProvisionConfirm {
		hint = fmt.Sprintf(`Create standard Kanban View for %q? y/n`, m.currentProjectTitle())
	}
	if m.leaderArmed {
		hint = "[LEADER] a Overview/return  •  o toggle projects pane  •  v views"
	}
	if m.viewSwitchArmed {
		hint = "[VIEWS] l list  •  k Kanban  •  t table  •  g Gantt"
	}
	if m.overviewPrompt == overviewPromptSort {
		hint = "SORT  d Due  •  p Priority ↓  •  t Title  •  Esc cancel"
	}
	if m.overviewPrompt == overviewPromptFilter {
		hint = "FILTER  o open  •  a all  •  v overdue  •  t due today  •  Esc cancel"
	}
	if m.ctrlWArmed {
		ctrlWHint := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.ctrlWHint)).Render("Ctrl-W")
		hint = ctrlWHint + " h/j/k/l change pane  •  Ctrl-W/w cycle"
	}
	statusStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(w)
	if m.lastErr != "" {
		hint = fmt.Sprintf("Error: %s", m.lastErr)
		statusStyle = statusStyle.Foreground(lipgloss.Color(tuiTheme.danger)).BorderForeground(lipgloss.Color(tuiTheme.danger))
	}

	status := fmt.Sprintf("%s   %s", info, hint)
	if scrollTop, maxScroll, scrolling := m.focusedScrollPosition(); scrolling {
		// The percentage needs its own separator plus enough space for 100%.
		// Existing hints always win when the status row is narrow.
		contentWidth := max(0, w-4) // rounded border plus horizontal padding
		if free := contentWidth - lipgloss.Width(status); free >= 5 {
			percent := int(math.Round(100 * float64(scrollTop) / float64(maxScroll)))
			status += strings.Repeat(" ", free-lipgloss.Width(fmt.Sprintf("%d%%", percent))) + fmt.Sprintf("%d%%", percent)
		}
	}
	return statusStyle.Render(status)
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
