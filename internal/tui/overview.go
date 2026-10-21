package tui

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

type taskScope int

const (
	scopeProject taskScope = iota
	scopeOverview
)

type overviewFilter int

const (
	overviewOpen overviewFilter = iota
	overviewAll
	overviewOverdue
	overviewDueToday
	overviewInProgress
)

type overviewSort int

const (
	overviewDue overviewSort = iota
	overviewPriorityDesc
	overviewTitle
)

type overviewPrompt int

const (
	overviewPromptNone overviewPrompt = iota
	overviewPromptSort
	overviewPromptFilter
)

type overviewTasksLoadedMsg struct {
	request uint64
	tasks   []vimkunja.Task
	err     error
}

var errUnsavedNavigation = errors.New("unsaved changes; use :w or :q! before changing task context")

func (m *model) inOverview() bool { return m.taskScope == scopeOverview }

func (m *model) loadOverview() tea.Cmd {
	m.taskRequest++
	request := m.taskRequest
	labelID := m.overviewLabel.ID
	perPage := max(200, m.perPage)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		var all []vimkunja.Task
		for page := 1; ; page++ {
			var items []vimkunja.Task
			var totalPages int
			var err error
			if labelID != 0 {
				items, totalPages, err = m.store.ListTasksForLabel(ctx, labelID, page, perPage)
			} else {
				items, totalPages, err = m.store.ListOverviewTasks(ctx, page, perPage)
			}
			if err != nil {
				return overviewTasksLoadedMsg{request: request, err: err}
			}
			all = append(all, items...)
			if totalPages == 0 || page >= totalPages {
				return overviewTasksLoadedMsg{request: request, tasks: all}
			}
		}
	}
}

func (m *model) openOverview() tea.Cmd {
	return m.openOverviewForLabel(vimkunja.Label{})
}

func (m *model) openLabelOverview(label vimkunja.Label) tea.Cmd {
	return m.openOverviewForLabel(label)
}

func (m *model) openOverviewForLabel(label vimkunja.Label) tea.Cmd {
	if m.editor != nil && m.editor.dirty() {
		m.recordError(errUnsavedNavigation)
		return nil
	}
	if m.kanbanActive {
		if task, ok := m.selectedKanbanTask(); ok {
			m.pendingKanbanTaskID = task.ID
		}
	} else if m.taskScope == scopeProject && m.taskSelected >= 0 && m.taskSelected < len(m.tasks) {
		m.pendingTaskID = m.tasks[m.taskSelected].ID
	}
	// Project metadata requested before this navigation no longer owns
	// the visible loading/error state. Saves may request fresh metadata later.
	m.viewRequest++
	m.viewLoading = false
	m.lastProjectPaneCollapsed = m.projectsPaneCollapsed
	m.taskScope = scopeOverview
	m.overviewLabel = label
	m.overviewFilter = overviewOpen
	m.overviewPrompt = overviewPromptNone
	m.showDetail = false
	m.editor = nil
	m.focus = paneTasks
	m.projectsPaneCollapsed = true
	m.overviewTasks = nil
	m.tasks = nil
	m.taskSelected = 0
	m.taskLoading = true
	return m.loadOverview()
}

func (m *model) closeTaskDetail() {
	m.comments = nil
	m.editor = nil
	m.showDetail = false
	m.kanbanEmptyDetailOrigin = false
	m.focus = paneTasks
	if m.projectsPaneAutoShown {
		m.projectsPaneCollapsed = true
		m.projectsPaneAutoShown = false
	}
}

func (m *model) openOverviewNewEditor() tea.Cmd {
	draft := vimkunja.TaskDraft{}
	m.editor = &taskEditor{
		draft: draft, original: draft, field: fieldProject, nativeTabstop: m.codeConfig.Tabstop,
		mode: editorProjectPicker, pickerIndex: m.projectPickerIndex(0, ""),
	}
	m.showDetail = true
	m.focus = paneDetail
	return nil
}

func (m *model) restoreProjectScope() tea.Cmd {
	wasKanban := m.kanbanActive
	if m.editor != nil && m.editor.dirty() {
		m.recordError(errUnsavedNavigation)
		return nil
	}
	m.taskScope = scopeProject
	m.overviewLabel = vimkunja.Label{}
	m.overviewPrompt = overviewPromptNone
	m.showDetail = false
	m.projectsPaneCollapsed = m.lastProjectPaneCollapsed
	m.editor = nil
	m.focus = paneTasks
	m.taskLoading = true
	if wasKanban {
		return m.loadKanbanBoard(m.currentProjectID)
	}
	if m.currentViewID == 0 {
		m.viewLoading = true
		return m.loadViews(m.currentProjectID)
	}
	return m.loadTasks(m.currentProjectID, m.currentViewID)
}

func (m *model) refreshActiveTasks() tea.Cmd {
	m.taskLoading = true
	if m.kanbanActive && !m.inOverview() {
		return m.loadKanbanBoard(m.currentProjectID)
	}
	if m.inOverview() {
		return m.loadOverview()
	}
	return m.loadTasks(m.currentProjectID, m.currentViewID)
}

func (m *model) setOverviewTasks(tasks []vimkunja.Task) {
	m.overviewTasks = tasks
	m.applyOverview()
}

func (m *model) setActiveTaskFilter(filter overviewFilter) {
	if m.inOverview() {
		m.overviewFilter = filter
		m.applyOverview()
		return
	}
	m.taskFilter = filter
	m.applyProjectTaskFilter()
}

func (m *model) applyProjectTaskFilter() {
	// Project task loading replaces m.tasks, so filtering must retain the
	// unfiltered source separately just like Overview does.
	items := make([]vimkunja.Task, 0, len(m.projectTasks))
	today := localStartOfDay(m.currentTime())
	for _, task := range m.projectTasks {
		if taskMatchesFilter(task, m.taskFilter, today) {
			items = append(items, task)
		}
	}
	m.tasks = items
	m.taskSelected = clamp(m.taskSelected, 0, max(0, len(m.tasks)-1))
}

func (m *model) applyOverview() {
	selectedID := int64(0)
	if m.taskSelected >= 0 && m.taskSelected < len(m.tasks) {
		selectedID = m.tasks[m.taskSelected].ID
	}
	items := make([]vimkunja.Task, 0, len(m.overviewTasks))
	today := localStartOfDay(m.currentTime())
	for _, task := range m.overviewTasks {
		if taskMatchesFilter(task, m.overviewFilter, today) {
			items = append(items, task)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return overviewLess(items[i], items[j], m.overviewSort) })
	m.tasks = items
	m.taskSelected = 0
	targetID := selectedID
	if m.pendingTaskID != 0 {
		targetID = m.pendingTaskID
	}
	for i, task := range items {
		if task.ID == targetID {
			m.taskSelected = i
			break
		}
	}
	m.pendingTaskID = 0
}

func taskMatchesFilter(task vimkunja.Task, filter overviewFilter, today time.Time) bool {
	if filter == overviewAll {
		return true
	}
	if task.Done {
		return false
	}
	due, ok := overviewDueDate(task)
	switch filter {
	case overviewOpen:
		return true
	case overviewOverdue:
		return ok && due.Before(today)
	case overviewDueToday:
		return ok && due.Equal(today)
	case overviewInProgress:
		return task.Workflow.Available && task.Workflow.CurrentBucketID != 0 &&
			task.Workflow.CurrentBucketID != task.Workflow.DefaultBucketID &&
			task.Workflow.CurrentBucketID != task.Workflow.DoneBucketID
	default:
		return false
	}
}

func overviewMatches(task vimkunja.Task, filter overviewFilter, today time.Time) bool {
	return taskMatchesFilter(task, filter, today)
}

func overviewLess(left, right vimkunja.Task, mode overviewSort) bool {
	leftDue, leftHasDue := overviewDueDate(left)
	rightDue, rightHasDue := overviewDueDate(right)
	if mode == overviewPriorityDesc {
		return overviewPriorityThenTitle(left, right)
	}
	if mode == overviewTitle {
		return strings.ToLower(left.Title) < strings.ToLower(right.Title)
	}
	if leftHasDue != rightHasDue {
		// Tasks without a Due Date remain after the dated block in either
		// direction: they are not time-critical and should not dominate it.
		return leftHasDue
	}
	if leftHasDue && !leftDue.Equal(rightDue) {
		return leftDue.Before(rightDue)
	}
	return overviewPriorityThenTitle(left, right)
}

func overviewPriorityThenTitle(left, right vimkunja.Task) bool {
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.ProjectIdentifier != right.ProjectIdentifier {
		return strings.ToLower(left.ProjectIdentifier) < strings.ToLower(right.ProjectIdentifier)
	}
	if left.Title != right.Title {
		return strings.ToLower(left.Title) < strings.ToLower(right.Title)
	}
	return left.ID < right.ID
}

// localStartOfDay returns midnight in value's local calendar day. Truncate(24h)
// is deliberately avoided: it rounds from UTC midnight and shifts the day in
// non-UTC time zones.
func localStartOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func overviewDueDate(task vimkunja.Task) (time.Time, bool) {
	if task.DueDate == "" {
		return time.Time{}, false
	}
	value, err := time.ParseInLocation("2006-01-02", task.DueDate, time.Local)
	return value, err == nil
}

func (m *model) updateTaskFilterPrompt(key string) bool {
	if m.overviewPrompt == overviewPromptNone {
		return false
	}
	prompt := m.overviewPrompt
	m.overviewPrompt = overviewPromptNone
	if key == "esc" {
		return true
	}
	if prompt == overviewPromptSort {
		switch key {
		case "d":
			m.overviewSort = overviewDue
		case "p":
			m.overviewSort = overviewPriorityDesc
		case "t":
			m.overviewSort = overviewTitle
		default:
			return true
		}
	} else {
		var filter overviewFilter
		switch key {
		case "o":
			filter = overviewOpen
		case "a":
			filter = overviewAll
		case "v":
			filter = overviewOverdue
		case "t":
			filter = overviewDueToday
		case "i":
			filter = overviewInProgress
		default:
			return true
		}
		m.setActiveTaskFilter(filter)
		return true
	}
	if m.inOverview() {
		m.applyOverview()
	}
	return true
}

func (m *model) updateOverviewPrompt(key string) bool {
	return m.updateTaskFilterPrompt(key)
}

func taskFilterLabel(filter overviewFilter) string {
	switch filter {
	case overviewAll:
		return "all"
	case overviewOverdue:
		return "overdue"
	case overviewDueToday:
		return "due today"
	case overviewInProgress:
		return "in progress"
	default:
		return "open"
	}
}

func (m *model) overviewFilterLabel() string {
	return taskFilterLabel(m.overviewFilter)
}

func (m *model) projectFilterLabel() string {
	return taskFilterLabel(m.taskFilter)
}

func (m *model) overviewSortLabel() string {
	switch m.overviewSort {
	case overviewDue:
		return "Due"
	case overviewPriorityDesc:
		return "Priority ↓"
	case overviewTitle:
		return "Title"
	default:
		return "Due"
	}
}
