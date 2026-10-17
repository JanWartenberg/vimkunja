package tui

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	apperrors "github.com/JanWartenberg/vimkunja/internal/errors"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type paneID int

type projectViewPreference int

const (
	projectViewList projectViewPreference = iota
	projectViewKanban
)

const commandHistoryLimit = 100

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
	Tabstop, IndentStep int
	LiteralTabAlias     string
	Store               vimkunja.Store

	PerPage int

	InitialProjectID    int64
	InitialViewID       int64
	LeaderTimeout       time.Duration
	CtrlWTimeout        time.Duration
	DisableCursorShapes bool
	StartupWarnings     []string
}

type model struct {
	codeConfig          Config
	store               vimkunja.Store
	perPage             int
	disableCursorShapes bool
	now                 func() time.Time
	clipboardWrite      func(string) error
	clipboardRead       func() (string, error)

	width            int
	height           int
	viewportReceived bool

	focus                    paneID
	ctrlWArmed               bool
	ctrlWTimeout             time.Duration
	leaderArmed              bool
	nativePaletteVisible     bool
	viewSwitchArmed          bool
	deleteArmed              bool
	deleteConfirm            bool
	deleteTaskID             int64
	confirmationScroll       int
	confirmationGArmed       bool
	projectsPaneCollapsed    bool
	projectsPaneAutoShown    bool
	lastProjectPaneCollapsed bool
	leaderTimeout            time.Duration
	gArmed                   bool
	zArmed                   bool
	neighbors                map[paneID]paneNeighbors

	lastErr     string
	lastWarning string
	toast       string
	quitHint    bool

	commandMode               bool
	commandInput              string
	commandCursor             int
	commandCompletionFeedback string
	commandHistory            []string
	commandHistoryIndex       int
	commandHistoryDraft       string
	commands                  *CommandRegistry

	searchOpen           bool
	searchMode           searchInputMode
	searchCommandMode    bool
	searchCommandInput   string
	searchCommandCursor  int
	searchGArmed         bool
	searchInput          string
	searchInputCursor    int
	searchResults        []vimkunja.SearchResult
	searchSelected       int
	searchViewport       int
	searchRequest        uint64
	searchLoading        bool
	searchOrigin         *searchOrigin
	searchReturn         *searchReturn
	labelUI              *labelUI
	comments             *commentsUI
	commentRequest       uint64
	commentDeleteConfirm bool
	commentDeleting      bool
	commentDeleteTaskID  int64
	commentDeleteID      int64

	// Projects pane
	projects             []vimkunja.Project
	projectTree          projectTree
	projectSelected      int
	projectViewportStart int
	projectLoading       bool

	// Views (derived from selected project)
	projectViews                []vimkunja.ProjectView
	viewSelected                int
	viewLoading                 bool
	currentViewID               int64
	projectViewPreference       projectViewPreference
	kanbanActive                bool
	kanbanBoard                 vimkunja.KanbanBoard
	kanbanBucketSelected        int
	kanbanTaskSelected          int
	kanbanTaskViewportStart     int
	kanbanViewportStart         int
	kanbanPicker                bool
	kanbanPickerIndex           int
	kanbanViewPicker            bool
	kanbanViewPickerIndex       int
	kanbanProvisionConfirm      bool
	pendingKanbanTaskID         int64
	pendingKanbanBucketID       int64
	pendingKanbanEmptySelection bool
	kanbanEmptyDetailOrigin     bool

	// Tasks pane
	tasks          []vimkunja.Task
	projectTasks   []vimkunja.Task
	taskFilter     overviewFilter
	taskSelected   int
	taskLoading    bool
	inlineTitle    *inlineTitleEdit
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
	savingEditor              *taskEditor
	cursorVisible             bool
	renderedInputCursor       *tea.Cursor

	initialProjectID int64
	initialViewID    int64

	createdTaskIDs    []int64
	lastDeletedTaskID int64
	undoHistory       []*undoEntry
	redoHistory       []*undoEntry
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
	if cfg.Tabstop <= 0 && cfg.LiteralTabAlias == "" {
		cfg.LiteralTabAlias = "ctrl+q"
	}
	if cfg.Tabstop <= 0 {
		cfg.Tabstop = 4
	}
	if cfg.IndentStep != 2 && cfg.IndentStep != 4 {
		cfg.IndentStep = 4
	}
	// A zero Config uses the default alias; TOML can explicitly disable it.
	m := &model{
		codeConfig:          cfg,
		store:               cfg.Store,
		perPage:             max(1, cfg.PerPage),
		disableCursorShapes: cfg.DisableCursorShapes,
		now:                 time.Now,
		initialProjectID:    cfg.InitialProjectID,
		leaderTimeout:       leaderTimeout,
		ctrlWTimeout:        ctrlWTimeout,
		initialViewID:       cfg.InitialViewID,
		taskFilter:          overviewAll,
		projectSelected:     0,
		currentViewID:       cfg.InitialViewID,
		neighbors: map[paneID]paneNeighbors{
			paneProjects: {left: nil, right: ptr(paneTasks), down: ptr(paneDetail)},
			paneTasks:    {left: ptr(paneProjects), right: nil, down: ptr(paneDetail)},
			paneDetail:   {up: ptr(paneTasks)},
		},
		focus:               paneProjects,
		commands:            newDefaultCommandRegistry(),
		commandHistoryIndex: -1,
		cursorVisible:       true,
		lastWarning:         startupWarningSummary(cfg.StartupWarnings),
	}
	for _, warning := range cfg.StartupWarnings {
		apperrors.Log(warning)
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
		undo     *undoEntry
		err      error
	}
	taskToggledMsg struct {
		undo *undoEntry
		err  error
	}
	taskDeletedMsg struct {
		taskID int64
		err    error
	}
	workflowStatusLoadedMsg struct {
		taskID int64
		editor *taskEditor
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
		undo       *undoEntry
		err        error
	}
	labelMutatedMsg struct {
		mutation labelMutation
		label    vimkunja.Label
		err      error
	}
	taskSavedMsg struct {
		task       vimkunja.Task
		editor     *taskEditor
		draft      vimkunja.TaskDraft
		closeAfter bool
		undo       *undoEntry
		err        error
		historyErr error
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
	viewID := int64(0)
	editor := m.editor
	if m.kanbanActive {
		viewID = m.currentViewID
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		var status vimkunja.WorkflowStatus
		var err error
		if viewID != 0 {
			status, err = m.store.GetTaskWorkflowStatus(ctx, projectID, taskID, viewID)
		} else {
			status, err = m.store.GetTaskWorkflowStatus(ctx, projectID, taskID)
		}
		return workflowStatusLoadedMsg{taskID: taskID, editor: editor, status: status, err: err}
	}
}

func (m *model) loadKanbanBoard(projectID int64) tea.Cmd {
	viewID := m.currentKanbanViewID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		var board vimkunja.KanbanBoard
		var err error
		if viewID != 0 {
			board, err = m.store.LoadManualKanbanBoard(ctx, projectID, viewID)
		} else {
			board, err = m.store.LoadManualKanbanBoard(ctx, projectID)
		}
		return kanbanBoardLoadedMsg{board: board, err: err}
	}
}

func (m *model) moveKanbanTask(taskID, bucketID int64) tea.Cmd {
	viewID := m.currentViewID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		undo, err := m.trackedTaskMutation(ctx, taskID, func() error {
			return m.store.SetTaskWorkflowStatus(ctx, taskID, bucketID, viewID)
		})
		return kanbanTaskMovedMsg{taskID: taskID, bucketID: bucketID, undo: undo, err: err}
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

// canShowQuitHint limits Escape's quit guidance to the root browsing state.
// Every overlay, prompt, picker, editor, and pending multi-key command owns
// Escape for cancellation instead.
func (m *model) canShowQuitHint() bool {
	return (m.focus == paneProjects || m.focus == paneTasks) &&
		m.editor == nil && m.inlineTitle == nil && m.labelUI == nil && m.comments == nil &&
		!m.searchOpen && !m.commandMode && !m.deleteArmed && !m.deleteConfirm &&
		!m.kanbanPicker && !m.kanbanProvisionConfirm && !m.leaderArmed &&
		!m.viewSwitchArmed && !m.ctrlWArmed && !m.gArmed && !m.zArmed &&
		m.overviewPrompt == overviewPromptNone && m.helpPane == nil && m.infoPane == nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Vim-split style focus switching
	if key, ok := msg.(tea.KeyMsg); ok {
		// Windows input can report a key release after its press. Releases must
		// not consume a pending multi-key command such as Ctrl+W h.
		if _, released := key.(tea.KeyReleaseMsg); released {
			return m, nil
		}
		// Keep the save's editor and close/quit intent stable until completion.
		// Ctrl-C remains the explicit application-wide emergency exit.
		if m.savingEditor != nil && strings.ToLower(key.String()) != "ctrl+c" {
			m.toast = "Saving Task…"
			return m, nil
		}

		// The palette stays readable after timeout until the next key.
		m.nativePaletteVisible = false
		// A footer error is transient feedback. The full error remains available
		// in the session log for the future :errors pane.
		m.lastErr = ""
		m.lastWarning = ""
		m.toast = ""
		m.quitHint = false

		k := key.String()
		kLower := strings.ToLower(k)
		if kLower == "ctrl+w" && m.editor != nil {
			m.editor.nativeLiteralTab = false
			m.editor.nativeTrigger = false
			m.editor.nativeCodeEscape = false
			m.editor.nativePrefix = ""
		}

		if m.focus == paneDetail && m.editor != nil && m.editor.field == fieldDescription && m.editor.mode == editorInsert && m.editor.nativeLiteralTab && kLower != "ctrl+c" && kLower != "ctrl+w" {
			return m.updateNativeInsert(key)
		}
		if m.editor != nil && m.editor.nativeLink != nil && kLower != "ctrl+c" {
			return m.updateNativeLink(key)
		}
		if !m.searchOpen && kLower == "ctrl+o" && m.searchReturn != nil {
			return m, m.restoreLastSearch()
		}

		if m.focus == paneDetail && m.editor != nil && m.editor.nativePending != nil && kLower != "ctrl+c" {
			m.nativeDescriptionKey(k)
			return m, nil
		}

		// Quit
		if kLower == "ctrl+c" {
			return m, tea.Quit
		}

		if kLower == "esc" && m.canShowQuitHint() {
			m.quitHint = true
			return m, nil
		}

		if updated, cmd, handled := m.dispatchModalGlobalCommand(key); handled {
			return updated, cmd
		}
		if m.searchOpen {
			return m.updateSearchKeys(key)
		}
		if m.commentDeleteConfirm {
			return m.updateCommentDeleteConfirmation(k)
		}
		if m.comments != nil {
			return m.updateComments(key)
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
		if m.kanbanViewPicker {
			return m.updateKanbanViewPicker(k)
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
		// Inline title text takes precedence over global leader input. Ctrl+W is
		// the sole exception: it commits the draft before changing Pane focus.
		if m.inlineTitle != nil && !m.ctrlWArmed && kLower != "ctrl+w" {
			return m.updateInlineTitleEdit(key)
		}
		if m.focus == paneDetail && m.editor != nil && m.editor.mode == editorNormal && (m.editor.nativePrefix == "[" || m.editor.nativePrefix == "]") {
			m.nativeDescriptionKey(k)
			return m, nil
		}
		if m.leaderArmed {
			m.leaderArmed = false
			if k == "y" && m.focus == paneDetail && m.editor != nil && m.editor.field == fieldDescription && m.editor.descriptionActive && m.editor.descriptionView() != nil {
				return m, m.copyNativeLink()
			}
			if m.nativePaletteKey(k) {
				return m, nil
			}
			switch k {
			case "f":
				m.openSearch()
			case "l":
				if m.editor != nil && m.editor.field == fieldDescription && m.editor.descriptionActive && m.editor.descriptionView() != nil {
					m.recordError(fmt.Errorf("Description is read-only: Link editing is not supported yet."))
					return m, nil
				}
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
			m.nativePaletteVisible = m.nativePalette() != ""
			return m, tea.Tick(m.leaderTimeout, func(time.Time) tea.Msg { return leaderTimeoutMsg{} })
		}

		if m.undoKeyAllowed() && (k == "u" || kLower == "ctrl+r") {
			return m, m.startUndo(kLower == "ctrl+r")
		}

		// Ctrl+W must be handled before Detail input. Otherwise the direction key
		// is consumed by the editor and cannot move focus out of the Detail pane.
		if m.ctrlWArmed {
			m.ctrlWArmed = false
			if m.inlineTitle != nil {
				return m, m.saveInlineTitleBeforeFocus(kLower, k)
			}

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
			m.openCommandPrompt()
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
		m.viewportReceived = true
		m.normalizeDetailSelection()
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
		// Overview reloads project metadata after a save, but must not activate
		// a project Board: its selection would replace the open Overview Task.
		if !m.inOverview() && m.projectViewPreference == projectViewKanban {
			if len(m.manualKanbanViewIndices()) > 1 {
				return m, m.beginKanbanViewPicker()
			}
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

	case taskCommentsLoadedMsg:
		if m.editor == nil || m.editor.id != typed.taskID || m.editor.commentsRequest != typed.request {
			return m, nil
		}
		m.editor.commentsLoading = false
		m.editor.commentsErr = typed.err
		if typed.err != nil {
			m.recordError(typed.err)
			if m.comments != nil {
				m.comments.loading, m.comments.err = false, typed.err
			}
			return m, nil
		}
		m.editor.comments = append([]vimkunja.TaskComment(nil), typed.comments...)
		if m.comments != nil {
			m.comments.items = append([]vimkunja.TaskComment(nil), typed.comments...)
			m.comments.selected = max(0, len(typed.comments)-1)
			m.comments.loading, m.comments.err = false, nil
		}
		return m, nil

	case taskCommentDeletedMsg:
		m.commentDeleting = false
		m.commentDeleteConfirm = false
		m.commentDeleteTaskID, m.commentDeleteID = 0, 0
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		if m.editor == nil || m.editor.id != typed.taskID {
			return m, nil
		}
		m.editor.comments = removeComment(m.editor.comments, typed.commentID)
		if m.comments != nil {
			m.comments.items = removeComment(m.comments.items, typed.commentID)
			m.comments.selected = clamp(m.comments.selected-1, 0, max(0, len(m.comments.items)-1))
		}
		return m, nil

	case taskCommentSavedMsg:
		quitAfterSave := m.quitAfterSave
		m.quitAfterSave = false
		if m.comments != nil {
			m.comments.loading = false
			m.comments.saving = false
		}
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		if m.editor == nil || m.comments == nil {
			return m, nil
		}
		if typed.taskChanged {
			m.redoHistory = nil
		}
		if typed.editor != nil {
			if typed.editor != m.editor {
				return m, m.refreshActiveTasks()
			}
			m.editor.adoptSavedTask(typed.draft, typed.task)
			if m.editor.dirty() && (typed.closeAfter || quitAfterSave) {
				typed.closeAfter = false
				quitAfterSave = false
				m.recordError(fmt.Errorf("Task draft changed while saving; save again before closing."))
			}
		} else {
			m.editor.id = typed.task.ID
			m.editor.draft.ID = typed.task.ID
			m.editor.draft.Done = typed.task.Done
			m.editor.draft.DoneSet = false
			m.editor.original = cloneTaskDraft(m.editor.draft)
		}
		m.commentRequest++
		m.editor.commentsRequest = m.commentRequest
		m.editor.commentsLoading = false
		m.editor.commentsErr = nil
		m.editor.comments = append(m.editor.comments, typed.comment)
		m.comments.items = append(m.comments.items, typed.comment)
		m.comments.selected = len(m.comments.items) - 1
		m.comments.composer = nil
		m.comments.commandMode, m.comments.commandInput = false, ""
		if typed.closeAfter {
			m.closeComments()
		}
		m.pendingTaskID = typed.task.ID
		if quitAfterSave {
			return m, tea.Quit
		}
		return m, m.refreshActiveTasks()

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
		m.editor.originalLabels = append([]vimkunja.Label(nil), typed.labels...)
		m.editor.draft.LabelIDs = editorLabelIDs(m.editor.labels)
		m.editor.draft.LabelIDsSet = false
		m.editor.original.LabelIDs = append([]int64(nil), m.editor.draft.LabelIDs...)
		m.editor.original.LabelIDsSet = false
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
		m.rememberUndo(typed.undo)
		m.editor.draft.LabelIDs = append([]int64(nil), typed.labelIDs...)
		m.editor.draft.LabelIDsSet = false
		m.editor.original.LabelIDs = append([]int64(nil), typed.labelIDs...)
		m.editor.original.LabelIDsSet = false
		m.editor.originalLabels = append([]vimkunja.Label(nil), m.editor.labels...)
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

	case linkCopiedMsg:
		if typed.err != nil {
			m.recordError(fmt.Errorf("write system clipboard: %w", typed.err))
		} else {
			m.toast = "Link destination copied"
		}
		return m, nil

	case tea.PasteMsg:
		if m.editor != nil && m.editor.nativeLink != nil {
			p := m.editor.nativeLink
			p.input = insertTextAtCursor(p.input, &p.cursor, typed.Content)
			return m, nil
		}
		if m.editor != nil && m.editor.nativePending != nil {
			return m, nil
		}
		text := normalizePastedText(typed.Content)
		if m.comments != nil && m.comments.composer != nil && m.comments.composer.mode == editorInsert {
			composer := m.comments.composer
			composer.text = insertTextAtCursor(composer.text, &composer.cursor, text)
			return m, nil
		}
		if m.focus == paneDetail && m.editor != nil && (m.editor.mode == editorInsert || (m.editor.dueDatePickerActive() && m.editor.dueDatePickerSelected == 0)) {
			m.insertEditorText(m.nativePasteText(typed.Content))
		}
		return m, nil

	case clipboardPastedMsg:
		if typed.link != nil {
			if m.editor == typed.editor && m.editor != nil && m.editor.nativeLink == typed.link {
				if typed.err != nil {
					m.recordError(typed.err)
				} else {
					p := typed.link
					p.input = insertTextAtCursor(p.input, &p.cursor, typed.text)
				}
			}
			return m, nil
		}
		if m.editor != nil && m.editor.nativeLink != nil {
			return m, nil
		}
		if m.editor != nil && m.editor.nativePending != nil {
			return m, nil
		}
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		text := normalizePastedText(typed.text)
		if m.comments != nil && m.comments.composer == typed.composer && typed.composer != nil && typed.composer.mode == editorInsert {
			typed.composer.text = insertTextAtCursor(typed.composer.text, &typed.composer.cursor, text)
			return m, nil
		}
		if m.focus == paneDetail && m.editor == typed.editor && (m.editor.mode == editorInsert || (m.editor.dueDatePickerActive() && m.editor.dueDatePickerSelected == 0)) {
			m.insertEditorText(m.nativePasteText(typed.text))
		}
		return m, nil

	case cursorBlinkMsg:
		if m.focus == paneDetail && m.editor != nil && (m.editor.mode == editorInsert || m.editor.dueDatePickerActive()) {
			m.cursorVisible = !m.cursorVisible
			return m, cursorBlinkCmd()
		}
		return m, nil

	case workflowStatusLoadedMsg:
		if m.editor != nil && ((typed.editor != nil && m.editor == typed.editor) || (typed.editor == nil && m.editor.id == typed.taskID)) {
			m.editor.workflowStatusLoading = false
			if typed.err == nil {
				m.editor.workflowStatus = typed.status
				// A Kanban-created task already has an explicit target bucket;
				// regular new tasks inherit the project's default bucket.
				if m.editor.id != 0 || m.editor.draft.WorkflowBucketID == 0 {
					m.editor.draft.WorkflowBucketID = typed.status.CurrentBucket.ID
					m.editor.original.WorkflowBucketID = typed.status.CurrentBucket.ID
				}
			}
		}
		if typed.err != nil {
			m.recordError(typed.err)
		}
		return m, nil

	case inlineTitleSavedMsg:
		m.applyInlineTitleSave(typed)
		return m, nil

	case undoRestoredMsg:
		return m, m.applyUndoRestore(typed)

	case descriptionReloadedMsg:
		if typed.editor != m.editor {
			return m, nil
		}
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		if !reflect.DeepEqual(typed.submitted, m.editor.draft) {
			m.recordError(fmt.Errorf("Task draft changed while reloading; explicit reload again to discard it"))
			return m, nil
		}
		m.editor.nativeDraft = nil
		m.editor.nativeDescription = nil
		m.editor.nativePending = nil
		m.refreshEditorAfterUndo(typed.state)
		m.editor.mode = editorNormal
		return m, tea.Batch(m.loadEditorLabels(typed.state.TaskID), m.loadUndoWorkflowStatus(typed.state.ProjectID, typed.state.TaskID))
	case taskSavedMsg:
		if typed.editor == nil || typed.editor != m.savingEditor {
			return m, nil
		}
		m.savingEditor = nil
		m.taskLoading = false
		if m.toast == "Saving Task…" {
			m.toast = ""
		}
		quitAfterSave := m.quitAfterSave
		m.quitAfterSave = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		if typed.draft.ID == 0 {
			m.createdTaskIDs = append(m.createdTaskIDs, typed.task.ID)
		}
		if typed.historyErr != nil {
			m.redoHistory = nil
			m.recordError(fmt.Errorf("Task saved; undo history unavailable: %w", typed.historyErr))
		}
		m.rememberUndo(typed.undo)
		if typed.editor != m.editor {
			// A background refresh replaced the Detail while the write ran.
			// Refresh persisted data without attaching the result to another draft.
			return m, m.refreshActiveTasks()
		}
		if m.editor != nil {
			m.editor.adoptSavedTask(typed.draft, typed.task)
			if m.editor.dirty() && (typed.closeAfter || quitAfterSave) {
				typed.closeAfter, quitAfterSave = false, false
				m.recordError(fmt.Errorf("Task draft changed while saving; save again before closing."))
			}
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
			m.editor.mode = editorNormal
			m.showDetail = true
			m.focus = paneDetail
		}
		if m.labelUI != nil && m.labelUI.kind == labelUITaskPicker && m.labelUI.closeAfterSave {
			m.closeLabelUI()
		}
		projectChanged := m.currentProjectID != typed.task.ProjectID
		if idx := m.findProjectIndex(typed.task.ProjectID); idx >= 0 {
			m.projectSelected = idx
		}
		m.currentProjectID = typed.task.ProjectID
		if wasKanban && !projectChanged {
			m.pendingKanbanTaskID = typed.task.ID
			return m, m.loadKanbanBoard(typed.task.ProjectID)
		}
		m.currentViewID = 0
		if projectChanged {
			m.kanbanActive = false
			m.kanbanBoard = vimkunja.KanbanBoard{}
			m.projectViews = nil
		}
		m.pendingTaskID = typed.task.ID
		m.viewLoading = true
		if wasOverview {
			return m, tea.Batch(m.loadViews(typed.task.ProjectID), m.refreshActiveTasks())
		}
		return m, m.loadViews(typed.task.ProjectID)

	case taskDeletedMsg:
		m.deleteConfirm = false
		m.deleteTaskID = 0
		m.confirmationScroll = 0
		m.confirmationGArmed = false
		if typed.err != nil {
			m.pendingKanbanTaskID = 0
			m.pendingKanbanBucketID = 0
			m.pendingKanbanEmptySelection = false
			m.taskLoading = false
			m.recordError(typed.err)
			return m, nil
		}
		m.lastDeletedTaskID = typed.taskID
		if m.editor != nil && m.editor.id == typed.taskID {
			m.closeTaskDetail()
		}
		return m, m.refreshActiveTasks()

	case taskToggledMsg:
		if typed.err != nil {
			m.pendingKanbanTaskID = 0
			m.taskLoading = false
			m.recordError(typed.err)
			return m, nil
		}
		m.rememberUndo(typed.undo)
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
			m.confirmationScroll = 0
			m.confirmationGArmed = false
			return m, nil
		}
		if typed.board.Availability == vimkunja.KanbanBoardAmbiguous {
			m.kanbanActive = false
			m.recordError(fmt.Errorf("multiple manual Kanban Views are not supported"))
			return m, nil
		}
		m.setKanbanBoard(typed.board)
		return m, m.syncOpenKanbanDetail()

	case kanbanProvisionedMsg:
		m.taskLoading = false
		if typed.err != nil {
			m.recordError(typed.err)
			return m, nil
		}
		m.setKanbanBoard(typed.board)
		return m, tea.Batch(m.loadViews(m.currentProjectID), m.syncOpenKanbanDetail())

	case kanbanTaskMovedMsg:
		if typed.err != nil {
			m.taskLoading = false
			m.recordError(typed.err)
			return m, nil
		}
		m.rememberUndo(typed.undo)
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
		m.projectTasks = typed.tasks
		m.applyProjectTaskFilter()
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
					return m, tea.Batch(m.loadWorkflowStatus(task.ProjectID, task.ID), m.loadEditorLabels(task.ID), m.loadTaskComments(task.ID))
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

func (m *model) openCommandPrompt() {
	m.commandMode = true
	m.commandInput = ""
	m.commandCursor = 0
	m.commandCompletionFeedback = ""
	m.commandHistoryIndex = -1
	m.commandHistoryDraft = ""
}

func (m *model) updateCommandModeKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(key.String()) {
	case "esc":
		m.commandMode = false
		m.commandInput = ""
		m.commandCompletionFeedback = ""
		m.commandHistoryIndex = -1
		m.commandHistoryDraft = ""
		return m, nil
	case "backspace", "delete", "left", "right", "home", "end":
		m.commandCompletionFeedback = ""
		updateSingleLineTextInput(&m.commandInput, &m.commandCursor, key)
		return m, nil
	case "up":
		m.recallOlderCommand()
		return m, nil
	case "down":
		m.recallNewerCommand()
		return m, nil
	case "tab":
		m.completeCommandInput()
		return m, nil
	case "enter":
		input := m.commandInput
		m.recordCommandHistory(input)
		m.commandMode = false
		m.commandInput = ""
		m.commandCompletionFeedback = ""
		m.commandHistoryIndex = -1
		m.commandHistoryDraft = ""

		updated, cmd, err := m.commands.Execute(m, input)
		if err != nil {
			m.recordError(err)
			return m, nil
		}
		return updated, cmd
	}

	if handled, _ := updateSingleLineTextInput(&m.commandInput, &m.commandCursor, key); handled {
		m.commandCompletionFeedback = ""
	}
	return m, nil
}

func (m *model) recordCommandHistory(input string) {
	input = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), ":"))
	if input == "" {
		return
	}
	m.commandHistory = append(m.commandHistory, input)
	if len(m.commandHistory) > commandHistoryLimit {
		m.commandHistory = m.commandHistory[len(m.commandHistory)-commandHistoryLimit:]
	}
}

func (m *model) recallOlderCommand() {
	if len(m.commandHistory) == 0 {
		return
	}
	m.commandCompletionFeedback = ""
	if m.commandHistoryIndex < 0 {
		m.commandHistoryDraft = m.commandInput
		m.commandHistoryIndex = len(m.commandHistory) - 1
	} else if m.commandHistoryIndex > 0 {
		m.commandHistoryIndex--
	}
	m.commandInput = m.commandHistory[m.commandHistoryIndex]
	m.commandCursor = len([]rune(m.commandInput))
}

func (m *model) recallNewerCommand() {
	if m.commandHistoryIndex < 0 {
		return
	}
	m.commandCompletionFeedback = ""
	if m.commandHistoryIndex < len(m.commandHistory)-1 {
		m.commandHistoryIndex++
		m.commandInput = m.commandHistory[m.commandHistoryIndex]
		m.commandCursor = len([]rune(m.commandInput))
		return
	}
	m.commandHistoryIndex = -1
	m.commandInput = m.commandHistoryDraft
	m.commandCursor = len([]rune(m.commandInput))
}

func (m *model) completeCommandInput() {
	m.commandCompletionFeedback = ""
	if m.commandInput == "" {
		m.commandCompletionFeedback = "Tab completes command names  •  :help lists commands"
		return
	}
	if strings.ContainsAny(m.commandInput, " \t\r\n") {
		m.commandCompletionFeedback = "Argument completion is not available."
		return
	}

	completion, candidates := m.commands.Complete(m.commandInput)
	if completion != "" {
		m.commandInput = completion
		m.commandCursor = len([]rune(completion))
	}
	if len(candidates) > 0 {
		m.commandCompletionFeedback = "matches: " + strings.Join(candidates, ", ")
	}
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
		return m, m.openNewEditor()
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

func (m *model) previewSelectedTask() tea.Cmd {
	if m.editor != nil && m.editor.dirty() {
		m.recordError(errUnsavedNavigation)
		return nil
	}
	if m.taskSelected < 0 || m.taskSelected >= len(m.tasks) {
		return nil
	}
	task := m.tasks[m.taskSelected]
	if m.inOverview() {
		if idx := m.findProjectIndex(task.ProjectID); idx >= 0 {
			m.projectSelected = idx
		}
		m.currentProjectID = task.ProjectID
	}
	focus := m.focus
	m.openExistingEditor(task)
	m.focus = focus
	return tea.Batch(m.loadWorkflowStatus(task.ProjectID, task.ID), m.loadEditorLabels(task.ID), m.loadTaskComments(task.ID))
}

func (m *model) updateTasksKeys(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.kanbanActive && !m.inOverview() {
		return m.updateKanbanKeys(k)
	}
	s := normalizeArrow(k.String())
	if m.updateTaskFilterPrompt(s) {
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
	case "i":
		m.openInlineTitleEdit(false)
		return m, nil
	case "I":
		m.openInlineTitleEdit(true)
		return m, nil
	case "v":
		return m, m.previewSelectedTask()
	case "enter":
		if m.taskSelected >= 0 && m.taskSelected < len(m.tasks) {
			task := m.tasks[m.taskSelected]
			if m.inOverview() {
				if idx := m.findProjectIndex(task.ProjectID); idx >= 0 {
					m.projectSelected = idx
				}
			}
			m.openExistingEditor(task)
			return m, tea.Batch(m.loadWorkflowStatus(task.ProjectID, task.ID), m.loadEditorLabels(task.ID), m.loadTaskComments(task.ID))
		}
		return m, nil
	case "o":
		if m.inOverview() {
			return m, m.openOverviewNewEditor()
		}
		return m, m.openNewEditor()
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
		m.overviewPrompt = overviewPromptFilter
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
				undo, err := m.trackedTaskMutation(ctx, t.ID, func() error {
					return m.store.ToggleTaskDone(ctx, t.ID, !t.Done)
				})
				return taskToggledMsg{undo: undo, err: err}
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

func (m *model) currentKanbanViewID() int64 {
	for _, view := range m.projectViews {
		if view.ID == m.currentViewID && view.ViewKind == "kanban" && view.BucketConfigurationMode == "manual" {
			return view.ID
		}
	}
	return 0
}

func (m *model) manualKanbanViewIndices() []int {
	indices := make([]int, 0)
	for i, view := range m.projectViews {
		if view.ViewKind == "kanban" && view.BucketConfigurationMode == "manual" {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m *model) pickManualKanbanViewIndex() int {
	indices := m.manualKanbanViewIndices()
	if len(indices) == 0 {
		return -1
	}
	return indices[0]
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

func removeComment(comments []vimkunja.TaskComment, id int64) []vimkunja.TaskComment {
	for index, comment := range comments {
		if comment.ID == id {
			return append(comments[:index:index], comments[index+1:]...)
		}
	}
	return comments
}

func (m *model) nativeEditorCursorActive() bool {
	if m.disableCursorShapes || m.focus != paneDetail || m.editor == nil || m.editor.field == fieldMoreFields {
		return false
	}
	// Relative-date helpers use their inline blinking marker rather than the
	// date row's native cursor coordinates.
	if m.editor.dueDatePickerActive() && m.editor.dueDatePickerSelected != 0 {
		return false
	}
	// The tiny focused Description workspace has a different row geometry from
	// the compact Detail document; its insertion caret follows rendered content.
	return !(m.editor.descriptionActive && m.detailTextHeight() < 12)
}

// Insertion positions come from the shared rendered anchor. Legacy coordinates
// are retained only for Normal-mode and masked overwrite cursors.
func (m *model) editorCursor() *tea.Cursor {
	if m.renderedInputCursor != nil {
		return m.renderedInputCursor
	}
	if m.commandMode || m.searchOpen || m.comments != nil {
		return nil
	}
	if m.editor != nil && m.editor.mode == editorInsert {
		return nil
	}
	if !m.nativeEditorCursorActive() || m.editor.isPickerMode() || m.labelUI != nil || m.editor.field == fieldLabels || (m.editor.field == fieldDescription && m.editor.descriptionView() != nil) {
		return nil
	}
	if m.editor.mode == editorNormal && m.editor.field != fieldDescription && m.editor.field != fieldTitle {
		return nil
	}
	// At the smallest regular Detail height, metadata and the pinned Comments
	// entry consume every Description body row. Do not position a block cursor
	// on the divider below an invisible Description.
	if m.editor.mode == editorNormal && m.editor.field == fieldDescription && m.editor.id != 0 && m.detailTextHeight() == 12 {
		return nil
	}
	if m.editor.mode == editorNormal && m.editor.field == fieldDescription && m.compactDetail() && !m.compactDescriptionCursorRendered() {
		return nil
	}

	if m.editor.field == fieldDescription && !m.compactDetail() {
		w, _ := m.viewportSize()
		layout := m.taskDetailDescriptionLayout(w-4, m.detailTextHeight())
		if layout.cursorRow < layout.start || layout.cursorRow >= layout.start+layout.rows {
			return nil
		}
	}
	x, y := m.editorCursorPosition()
	if m.editor.mode == editorNormal && (m.editor.field == fieldTitle || m.editor.field == fieldDescription) {
		value := m.editor.fieldValue()
		if m.editor.cursor > textLineStart(value, m.editor.cursor) && m.editor.cursor == textLineEnd(value, m.editor.cursor) {
			x--
		}
	}
	if m.editor.dueDatePickerActive() {
		// Masked entry skips separators; indicate the digit typing will replace.
		x += m.editor.dueDateOverwriteCursor() - m.editor.cursor
	}
	cursor := tea.NewCursor(x, y)
	if m.editor.dueDatePickerActive() {
		cursor.Shape = tea.CursorBlock
		cursor.Blink = true
	} else {
		cursor.Shape = tea.CursorBlock
	}
	return cursor
}

// editorCursorPosition locates the focused field in the detail pane. Picker
// fields are visibly highlighted as a whole; the native cursor sits at their
// value position without changing the rendered text.
func (m *model) editorCursorPosition() (int, int) {
	w, h := m.viewportSize()
	topH := m.detailPaneTop()
	if m.compactDetail() {
		return m.compactEditorCursorPosition(w, h, topH)
	}
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
		if m.editor.field == fieldTitle {
			cursorOffset -= m.editor.fieldViewportStart
			if m.editor.fieldViewportStart > 0 {
				cursorOffset++ // leading ellipsis in the horizontally scrolled field
			}
		}
	}
	return clamp(textX+labelWidth+cursorOffset, 0, w-1), clamp(topH+1+row, 0, h-1)
}

func (m *model) compactEditorCursorPosition(w, h, topH int) (int, int) {
	row, ok := m.compactDetailFieldRow(m.editor.field)
	if !ok {
		return 0, 0
	}
	if m.editor.field == fieldDescription {
		return m.compactDescriptionCursorPosition(w, h, topH, row)
	}
	labelWidth := 0
	switch m.editor.field {
	case fieldTitle:
		labelWidth = len([]rune("Title:    "))
	case fieldProject:
		labelWidth = len([]rune("Project:  "))
	case fieldPriority:
		labelWidth = len([]rune("Priority: "))
	case fieldDueDate:
		labelWidth = len([]rune("Due Date: "))
	case fieldLabels:
		labelWidth = len([]rune("Labels:   "))
	case fieldWorkflowStatus:
		labelWidth = len([]rune("Status:   "))
	case fieldDone:
		labelWidth = len([]rune("Done:     "))
	}
	textX := 4 + labelWidth
	return clamp(textX+m.editor.cursor, 0, w-1), clamp(topH+3+row-m.editor.detailScroll, 0, h-1)
}

func (m *model) compactDescriptionCursorPosition(w, h, topH, row int) (int, int) {
	textW := w - 4
	description := wrapTaskDescriptionForEditor(m.editor, m.editor.draft.Description, textW)
	descriptionRows := m.compactDescriptionRows()
	if len(description) > descriptionRows {
		description = wrapTaskDescriptionForEditor(m.editor, m.editor.draft.Description, max(1, textW-2))
	}
	start := clamp(m.editor.descScroll, 0, max(0, len(description)-descriptionRows))
	cursor := clamp(m.editor.cursor, 0, m.editor.fieldRuneLen())
	lineIndex, column := 0, 0
	for i, line := range description {
		if cursor >= line.start && cursor <= line.end {
			lineIndex, column = i, cursor-line.start
			break
		}
	}
	visibleLine := clamp(lineIndex-start, 0, descriptionRows-1)
	return clamp(2+column, 0, w-1), clamp(topH+4+row-m.editor.detailScroll+visibleLine, 0, h-1)
}

func (m *model) compactDescriptionCursorRendered() bool {
	e := m.editor
	if e == nil {
		return false
	}
	w, _ := m.viewportSize()
	descriptionRows := m.compactDescriptionRows()
	description := wrapTaskDescriptionForEditor(e, e.draft.Description, w-4)
	if len(description) > descriptionRows {
		description = wrapTaskDescriptionForEditor(e, e.draft.Description, max(1, w-6))
	}
	cursor := clamp(e.cursor, 0, e.fieldRuneLen())
	lineIndex := 0
	for index, line := range description {
		if cursor >= line.start && cursor <= line.end {
			lineIndex = index
			break
		}
	}
	start := clamp(e.descScroll, 0, max(0, len(description)-descriptionRows))
	if e.descViewportScroll && (lineIndex < start || lineIndex >= start+descriptionRows) {
		return false
	}
	descriptionRow := 0
	for _, field := range m.detailFields() {
		if field == fieldComments {
			continue
		}
		if field == fieldDescription {
			break
		}
		descriptionRow++
	}
	bodyRows := max(1, m.detailTextHeight()-2-compactPinnedCommentRows(m.detailTextHeight(), e.id != 0))
	cursorRow := descriptionRow + 1 + lineIndex - start
	return cursorRow >= e.detailScroll && cursorRow < e.detailScroll+bodyRows
}

type taskDetailDescriptionLayout struct {
	multilineTextLayout
	rows, descriptionRow int
	scrollable           bool
}

func (m *model) taskDetailDescriptionLayout(textW, textH int) taskDetailDescriptionLayout {
	e := m.editor
	constrainedRegular := e.id != 0 && textH < 14
	descriptionRow := 1 // Detail header.
	if !constrainedRegular {
		descriptionRow++ // Header separator.
	}
	descriptionRow += 4 // Project, Title, Priority, Due Date.
	descriptionRow += 2 // Labels and Workflow Status.
	if e.id != 0 {
		descriptionRow++ // Done.
		if textH >= 12 && !e.isPickerMode() {
			descriptionRow++ // Database identifier.
		}
	}
	if !constrainedRegular {
		descriptionRow++ // Description separator.
	}
	descriptionRow++ // Description label.
	rows := availableDetailDescriptionRows(textH, descriptionRow, e.id != 0)
	text := e.draft.Description
	scrollable := len(wrapTaskDescriptionForEditor(e, text, max(1, textW))) > rows
	width := textW
	if scrollable {
		width = max(1, textW-2)
	}
	scroll := 0
	if e.field == fieldDescription {
		scroll = e.descScroll
	}
	layout := layoutTaskDescription(text, width, rows, scroll, e.cursor, e)
	if e.field == fieldDescription && e.descriptionView() == nil && rows > 0 && !e.descViewportScroll {
		layout.start = descriptionViewportStart(layout.lines, rows, scroll, layout.cursorRow, e.mode == editorNormal && e.cursor == e.fieldRuneLen())
	}
	return taskDetailDescriptionLayout{
		multilineTextLayout: layout,
		rows:                rows,
		descriptionRow:      descriptionRow,
		scrollable:          scrollable,
	}
}

func (m *model) descriptionCursorPosition(w, h, topH int) (int, int) {
	textW := w - 4
	contentH := max(4, h-3)
	detailH := max(4, contentH-topH)
	textH := detailH - 2
	layout := m.taskDetailDescriptionLayout(textW, textH)

	visibleLine := clamp(layout.cursorRow-layout.start, 0, layout.rows-1)
	bodyRow := layout.descriptionRow + visibleLine
	return clamp(2+layout.cursorColumn, 0, w-1), clamp(topH+1+bodyRow-m.editor.detailScroll, 0, h-1)
}

// tasksPaneTextX returns the screen column at which renderTasks's text begins.
func (m *model) tasksPaneTextX() int {
	w, _ := m.viewportSize()
	if m.projectsPaneCollapsed {
		return 2 // Tasks border plus horizontal padding.
	}
	projectsOuter := min(30, w/3)
	projectsOuter = max(16, projectsOuter)
	return projectsOuter + 2 // Projects outer width, then Tasks chrome.
}

func (m *model) viewportSize() (int, int) {
	if m.viewportReceived {
		return max(1, m.width), max(1, m.height-m.nativePaletteExtraRows())
	}
	w, h := m.width, m.height
	if w < 40 {
		w = 80
	}
	if h < 10 {
		h = 24
	}
	return w, max(1, h-m.nativePaletteExtraRows())
}

func (m *model) detailPaneTop() int {
	_, h := m.viewportSize()
	return max(4, (h-3)/2)
}

// modalContentHeight matches the Pane area built by Render, excluding the
// status footer. Confirmation key handling uses this same geometry as modal
// rendering, rather than a separate approximation based on m.height.
func (m *model) modalContentHeight() int {
	_, h := m.viewportSize()
	contentH := max(4, h-3)
	if m.showDetail || m.helpPane != nil || m.infoPane != nil {
		topH := m.detailPaneTop()
		return topH + max(4, contentH-topH)
	}
	return contentH
}

func (m *model) Render() string {
	content, cursor := resolveInputCursor(m.renderContent(), m.disableCursorShapes, m.cursorVisible)
	m.renderedInputCursor = cursor
	return content
}

func (m *model) renderContent() string {
	w, h := m.viewportSize()
	if w < modalFallbackWidth || h < modalFallbackHeight {
		return m.renderTextOnlyFallback(w, h)
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
		BorderForeground(lipgloss.Color(paneBorderColor(!m.commandMode && m.focus == paneTasks))).
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
			BorderForeground(lipgloss.Color(paneBorderColor(!m.commandMode && m.focus == paneProjects))).
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
		bottomFocused := !m.commandMode && (m.focus == paneDetail || m.focus == paneHelp || m.focus == paneInfo)
		bottomStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(paneBorderColor(bottomFocused))).
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
	if m.kanbanProvisionConfirm {
		content = m.renderKanbanProvisionConfirmationModal(content, w)
	}
	if m.searchOpen {
		content = m.renderSearchOverlay(content, w)
	}
	if m.labelUI != nil {
		content = m.renderLabelOverlay(content, w)
	}
	if m.comments != nil {
		content = m.renderCommentsOverlay(content, w)
	}
	if m.commentDeleteConfirm {
		content = m.renderCommentDeleteConfirmationModal(content, w)
	}
	status := m.renderStatusBar(w)
	return lipgloss.JoinVertical(lipgloss.Top, content, status)
}

// renderTextOnlyFallback replaces the multi-Pane layout below the minimum
// usable terminal size. It intentionally has no borders: a partial Pane frame
// conveys less than a single current-context line and footer does.
func (m *model) renderTextOnlyFallback(w, h int) string {
	info, hint := "[Tasks]", "j/k navigate  •  :h help"
	switch m.focus {
	case paneProjects:
		info, hint = "[Projects]", "j/k select  •  :h help"
	case paneDetail:
		info, hint = "[Detail]", "i edit  •  j/k navigate  •  :h help"
		if m.editor != nil {
			switch m.editor.mode {
			case editorProjectPicker, editorPriorityPicker, editorWorkflowStatusPicker:
				hint = "j/k select  •  Enter choose  •  Esc cancel"
			}
		}
	case paneHelp:
		info, hint = "[Help]", "q/esc close"
	case paneInfo:
		info, hint = "[Info]", "q/esc close"
	}
	footer := renderFooterHints(w, info, strings.Split(hint, "  •  "), false)
	lines := []string{"Vimkunja"}
	for len(lines) < max(0, h-1) {
		lines = append(lines, "")
	}
	if h > 0 {
		lines = append(lines[:max(0, h-1)], footer)
	}
	return fitBlock(lines, w, h)
}

// focusedScrollPosition reports the focused Pane's visual viewport position.
// Fixed Pane chrome (headers and editor fields) is excluded: only rows that
// actually move under a scrolling command contribute to the percentage.
func (m *model) focusedScrollPosition() (int, int, bool) {
	if m.searchOpen && len(m.searchResults) > 1 && m.searchResultEnd(0) < len(m.searchResults) {
		return m.searchSelected, len(m.searchResults) - 1, true
	}

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
		if m.editor == nil {
			break
		}
		if m.compactDetail() {
			if m.editor.field == fieldDescription && m.editor.descriptionActive {
				limit := m.readOnlyDescriptionScrollLimit()
				if !descriptionHasHTML(m.editor.draft.Description) {
					w, _ := m.viewportSize()
					rows := m.compactDescriptionRows()
					lines := wrapTaskDescriptionForEditor(m.editor, m.editor.draft.Description, w-4)
					if len(lines) > rows {
						lines = wrapTaskDescriptionForEditor(m.editor, m.editor.draft.Description, max(1, w-6))
					}
					limit = max(0, len(lines)-rows)
				}
				if limit > 0 {
					return clamp(m.editor.descScroll, 0, limit), limit, true
				}
			}
			if limit := m.detailScrollLimit(); limit > 0 {
				return clamp(m.editor.detailScroll, 0, limit), limit, true
			}
			break
		}
		if m.editor.field != fieldDescription {
			break
		}
		detailH := max(4, contentH-topH)
		textH := detailH - 2
		layout := m.taskDetailDescriptionLayout(w-4, textH)
		if limit := len(layout.lines) - layout.rows; limit > 0 {
			return layout.start, limit, true
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
	// Rounded border plus horizontal padding needs five cells. Below that the
	// terminal cannot display a usable status bar, so omit it rather than let
	// lipgloss expand and wrap it beyond the viewport.
	if w < 5 {
		return ""
	}

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

	// Hint order is the narrow-footer priority contract: primary action,
	// navigation, help, then secondary actions. renderFooterHints keeps only
	// complete segments from this ordered budget.
	hint := "Ctrl+W cycles  •  Enter detail  •  j/k navigate  •  :h help  •  q quit  •  x toggle"
	if m.focus == paneProjects {
		hint = "j/k select  •  h/l fold/tree  •  :h help  •  z fold commands  •  Shift+J/K reorder  •  r reload"
	}
	if m.focus == paneTasks && m.kanbanActive && !m.inOverview() {
		hint = "Shift+H/L move bucket  •  m choose bucket  •  j/k navigate  •  :h help  •  x toggle  •  Ctrl+W cycles"
	}
	if m.focus == paneTasks && m.inOverview() {
		hint = "j/k select  •  Enter detail  •  :h help  •  o new task  •  s sort  •  f filter  •  Space+a return"
	}
	if (m.focus == paneProjects || m.focus == paneTasks) && !m.inOverview() && w >= 100 {
		hint += "  •  o new task"
	}
	if m.focus == paneDetail {
		mode := "NORMAL"
		inputActive, pickerActive := false, false
		if m.editor != nil {
			switch m.editor.mode {
			case editorInsert:
				mode = "INSERT"
				inputActive = true
			case editorProjectPicker:
				pickerActive = true
				hint = "[PROJECT PICKER] j/k move  •  Ctrl+D/U page  •  gg/G jump  •  Enter choose  •  Esc cancel"
			case editorPriorityPicker:
				pickerActive = true
				hint = "[PRIORITY PICKER] j/k move  •  Ctrl+D/U page  •  gg/G jump  •  Enter choose  •  Esc cancel"
			case editorWorkflowStatusPicker:
				pickerActive = true
				hint = "[STATUS PICKER] j/k move  •  Ctrl+D/U page  •  gg/G jump  •  Enter choose  •  Esc cancel"
			case editorDueDatePicker:
				pickerActive = true
				hint = "[DUE DATE] j/k move  •  digits enter  •  Enter choose  •  Esc cancel"
			}
			if m.editor.dirty() {
				mode += " [+]"
			}
		}
		if inputActive {
			hint = "[" + mode + "]  Esc normal"
		} else if pickerActive {
			// Picker hints were set above; Esc cancels the picker rather than
			// switching the editor from Insert to Normal mode.
		} else if m.editor != nil && m.editor.field == fieldLabels && !m.editor.labelsLoading && len(m.editor.labels) > 0 {
			hint = "[" + mode + "]  i pick/edit  •  h/l select Label  •  :h help  •  x remove Label  •  :w save  •  :q close"
		} else if m.editor != nil && m.editor.gArmed {
			hint = "[FIELD] t title  •  d due date  •  p priority  •  l labels  •  c comments  •  n description  •  Esc cancel"
		} else if m.editor != nil && m.editor.dArmed {
			hint = "[DELETE FIELD] d clear  •  Esc cancel"
		} else if m.editor != nil && m.editor.optionalFieldClearable() {
			hint = "[" + mode + "]  x/dd clear  •  i edit  •  j/k navigate  •  :h help  •  gt title  •  gd due  •  gp priority  •  gl labels  •  gc comments  •  gn description  •  :w save  •  :q close"
		} else {
			hint = "[" + mode + "]  i edit  •  j/k navigate  •  :h help  •  gt title  •  gd due  •  gp priority  •  gl labels  •  gc comments  •  gn description  •  :w save  •  :q close  •  Ctrl+W switch pane"
		}
	}
	if m.focus == paneDetail && m.editor != nil && m.editor.field == fieldDescription && m.editor.descriptionView() != nil {
		hint = "Enter read · j/k blocks · gj/gk rows · {/} paragraphs · zh/zl scroll · Esc fields · :h help"
		if m.editor.descriptionActive {
			hint = m.nativeModeHint()
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
	if m.comments != nil {
		hint = "[COMMENTS] j/k select  •  gg/G jump  •  a/i/o append  •  Esc close"
		if m.comments.saving {
			hint = "[COMMENTS] Saving Task and Comment…"
		} else if m.comments.composer != nil {
			mode := "NORMAL"
			if m.comments.composer.mode == editorInsert {
				mode = "INSERT"
			}
			hint = "[COMMENT " + mode + "] Esc normal  •  :w save  •  :q! discard"
		}
		if m.comments.commandMode {
			hint = "COMMENTS COMMAND  :" + insertCursorText(m.comments.commandInput, m.comments.commandCursor)
		}
	}
	if m.commandMode {
		hint = ":" + insertCursorText(m.commandInput, m.commandCursor)
		if m.commandCompletionFeedback != "" {
			hint += "  •  " + m.commandCompletionFeedback
		}
	}
	if (m.leaderArmed || m.nativePaletteVisible) && !m.nativePaletteSecondary() {
		hint = "[LEADER] a Overview/return  •  o toggle projects pane  •  v views"
		if palette := m.nativePalette(); palette != "" && !m.nativePaletteSecondary() {
			hint = strings.Join(append([]string{m.nativePaletteCue()}, strings.Split(palette, " · ")...), footerSeparator)
		}
	}
	if m.viewSwitchArmed {
		hint = "[VIEWS] l list  •  k Kanban  •  t table  •  g Gantt"
	}
	if m.kanbanViewPicker {
		hint = "[KANBAN VIEWS] j/k choose  •  Enter open  •  Esc cancel"
	}
	if m.overviewPrompt == overviewPromptSort {
		hint = "SORT  d Due  •  p Priority ↓  •  t Title  •  Esc cancel"
	}
	if m.overviewPrompt == overviewPromptFilter {
		hint = "FILTER  o open  •  a all  •  v overdue  •  t due today  •  i in progress  •  Esc cancel"
	}
	if m.ctrlWArmed {
		ctrlWHint := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.ctrlWHint)).Render("Ctrl-W")
		hint = ctrlWHint + " h/j/k/l change pane  •  Ctrl-W/w cycle"
	}
	statusActive := m.commandMode || (m.comments != nil && m.comments.commandMode) || m.leaderArmed || m.viewSwitchArmed || m.kanbanViewPicker || m.overviewPrompt != overviewPromptNone || m.ctrlWArmed
	statusStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(paneBorderColor(statusActive))).
		Padding(0, 1).
		Width(w)
	if m.lastErr != "" {
		hint = fmt.Sprintf("Error: %s", m.lastErr)
		statusStyle = statusStyle.Foreground(lipgloss.Color(tuiTheme.danger)).BorderForeground(lipgloss.Color(tuiTheme.danger))
	} else if m.lastWarning != "" {
		hint = "Warning: " + m.lastWarning
		statusStyle = statusStyle.Foreground(lipgloss.Color(tuiTheme.danger)).BorderForeground(lipgloss.Color(tuiTheme.danger))
	}

	contentWidth := max(1, w-4) // rounded border plus horizontal padding
	scrollTop, maxScroll, scrolling := m.focusedScrollPosition()
	reserveScrollPercent := scrolling && footerHasRoomForCommands(contentWidth-5, info, strings.Split(hint, footerSeparator), 3)
	if m.searchOpen && scrolling {
		reserveScrollPercent = true
	}
	var status string
	switch {
	case m.lastErr != "":
		status = renderFooterMessage(contentWidth, info, "Error: ", m.lastErr)
	case m.lastWarning != "":
		status = renderFooterMessage(contentWidth, info, "Warning: ", m.lastWarning)
	case m.editor != nil && m.editor.nativePending != nil:
		status = m.renderNativeConfirmation(contentWidth, info)
	case m.toast != "":
		status = renderFooterMessage(contentWidth, info, "", m.toast)
	case m.quitHint:
		status = renderFooterMessage(contentWidth, info, "", ":q quit")
	case m.commandMode:
		input := ":" + insertCursorText(m.commandInput, m.commandCursor)
		status = renderFooterMessage(contentWidth, info, ":", insertCursorText(m.commandInput, m.commandCursor))
		if m.commandCompletionFeedback != "" && footerFits(contentWidth, info, []string{input, m.commandCompletionFeedback}) {
			status = composeFooter(info, []string{input, m.commandCompletionFeedback})
		}
	case m.searchOpen:
		if m.searchMode == searchNormalMode && !m.searchCommandMode {
			width := contentWidth
			if reserveScrollPercent {
				width -= 5
			}
			status = renderFooterHints(width, info, []string{
				"SEARCH [NORMAL]  j/k navigate",
				"i/a edit",
				":h help",
				":q / Esc close",
			}, false)
			break
		}
		prefix := fmt.Sprintf("SEARCH [%s] ", m.searchModeLabel())
		input := m.searchInput
		if m.searchCommandMode {
			prefix, input = "SEARCH COMMAND  :", insertCursorText(m.searchCommandInput, m.searchCommandCursor)
		} else if m.searchMode == searchInsertMode {
			input = insertCursorText(m.searchInput, m.searchInputCursor)
		}

		status = renderFooterMessage(contentWidth, info, prefix, input)
	case strings.Contains(hint, insertCursorAnchor):
		status = renderFooterMessage(contentWidth, info, "", hint)
	default:
		required := (m.focus == paneDetail && m.editor != nil && m.editor.mode != editorNormal) || m.focus == paneHelp || m.focus == paneInfo
		width := contentWidth
		if reserveScrollPercent {
			width -= 5 // separator plus enough space for 100%
		}
		status = renderFooterHints(width, info, strings.Split(hint, footerSeparator), required)
	}
	if scrolling && reserveScrollPercent {
		// Once three complete commands fit, the percentage reserves the final
		// five cells and yields lower-priority hints instead.
		if free := contentWidth - lipgloss.Width(status); free >= 5 {
			percent := int(math.Round(100 * float64(scrollTop) / float64(maxScroll)))
			status += strings.Repeat(" ", free-lipgloss.Width(fmt.Sprintf("%d%%", percent))) + fmt.Sprintf("%d%%", percent)
		}
	}
	if m.nativePaletteSecondary() {
		status += "\n" + m.nativePaletteRow(contentWidth)
	}
	return statusStyle.Render(status)
}

const footerSeparator = "  •  "

// renderFooterHints fits whole hint segments into one status row. Segment
// order is its priority; :h help degrades to :h before lower-priority segments
// are considered. The Pane label yields only when no complete segment fits.
func renderFooterHints(width int, info string, segments []string, requiredFirst bool) string {
	for _, label := range []string{info, ""} {
		parts := make([]string, 0, len(segments))
		for i, raw := range segments {
			segment := strings.TrimSpace(raw)
			if segment == "" {
				continue
			}
			if requiredFirst && i == 0 && !footerFits(width, label, []string{segment}) {
				break
			}
			candidate := append(append([]string(nil), parts...), segment)
			if footerFits(width, label, candidate) {
				parts = candidate
				continue
			}
			if segment == ":h help" && footerFits(width, label, append(append([]string(nil), parts...), ":h")) {
				parts = append(parts, ":h")
				continue
			}
			break
		}
		if len(parts) > 0 {
			return composeFooter(label, parts)
		}
		for _, fallback := range []string{":h help", ":h"} {
			if footerFits(width, label, []string{fallback}) {
				return composeFooter(label, []string{fallback})
			}
		}
	}
	return ""
}

func footerHasRoomForCommands(width int, info string, segments []string, count int) bool {
	parts := make([]string, 0, count)
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		parts = append(parts, segment)
		if len(parts) == count {
			return footerFits(width, info, parts)
		}
	}
	return false
}

func footerFits(width int, info string, parts []string) bool {
	return lipgloss.Width(composeFooter(info, parts)) <= width
}

func composeFooter(info string, parts []string) string {
	body := strings.Join(parts, footerSeparator)
	if info == "" {
		return body
	}
	if body == "" {
		return info
	}
	return info + "   " + body
}

// renderFooterMessage preserves a message prefix and the tail nearest the
// cursor or most recent error detail when the footer is narrow.
func renderFooterMessage(width int, info, prefix, value string) string {
	for _, label := range []string{info, ""} {
		available := width
		if label != "" {
			available -= lipgloss.Width(label + "   ")
		}
		// A Pane label must yield when it would leave room only for an
		// ellipsis: active input and error detail are more useful feedback.
		if label != "" && lipgloss.Width(prefix+value) > available && available <= lipgloss.Width(prefix)+1 {
			continue
		}
		message := fitFooterTail(prefix, value, available)
		if footerFits(width, label, []string{message}) {
			return composeFooter(label, []string{message})
		}
	}
	return fitFooterTail(prefix, value, width)
}

func fitFooterTail(prefix, value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(prefix+value) <= width {
		return prefix + value
	}
	if lipgloss.Width(prefix) >= width {
		return truncatePlain(prefix, width)
	}
	available := width - lipgloss.Width(prefix)
	if strings.Contains(value, insertCursorAnchor) {
		return prefix + fitInputCursor(value, available)
	}
	if available == 1 {
		return prefix + "…"
	}
	return prefix + "…" + tailPlain(value, available-1)
}

func tailPlain(value string, width int) string {
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width {
		runes = runes[1:]
	}
	return string(runes)
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
