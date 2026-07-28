package tui

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/vimkunja/vimkunja/internal/vimkunja"

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
)

type overviewSort int

const (
	overviewDueAsc overviewSort = iota
	overviewDueDesc
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
	tasks []vimkunja.Task
	err   error
}

var errUnsavedNavigation = errors.New("unsaved changes; use :w or :q! before changing task context")

func (m *model) inOverview() bool { return m.taskScope == scopeOverview }

func (m *model) loadOverview() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		perPage := max(200, m.perPage)
		var all []vimkunja.Task
		for page := 1; ; page++ {
			items, totalPages, err := m.store.ListOverviewTasks(ctx, page, perPage)
			if err != nil {
				return overviewTasksLoadedMsg{err: err}
			}
			all = append(all, items...)
			if totalPages == 0 || page >= totalPages {
				return overviewTasksLoadedMsg{tasks: all}
			}
		}
	}
}

func (m *model) openOverview() tea.Cmd {
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
	m.lastProjectPaneCollapsed = m.projectsPaneCollapsed
	m.taskScope = scopeOverview
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
	m.editor = nil
	m.showDetail = false
	m.focus = paneTasks
	if m.projectsPaneAutoShown {
		m.projectsPaneCollapsed = true
		m.projectsPaneAutoShown = false
	}
}

func (m *model) openOverviewNewEditor() {
	draft := vimkunja.TaskDraft{CreatedByID: m.actorUserID}
	m.editor = &taskEditor{
		draft: draft, original: draft, field: fieldProject,
		mode: editorProjectPicker, pickerIndex: m.projectPickerIndex(0, ""),
	}
	m.showDetail = true
	m.focus = paneDetail
}

func (m *model) restoreProjectScope() tea.Cmd {
	wasKanban := m.kanbanActive
	if m.editor != nil && m.editor.dirty() {
		m.recordError(errUnsavedNavigation)
		return nil
	}
	m.taskScope = scopeProject
	m.overviewPrompt = overviewPromptNone
	m.showDetail = false
	m.projectsPaneCollapsed = m.lastProjectPaneCollapsed
	m.editor = nil
	m.focus = paneTasks
	m.taskLoading = true
	if wasKanban {
		return m.loadKanbanBoard(m.currentProjectID)
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

func (m *model) applyOverview() {
	selectedID := int64(0)
	if m.taskSelected >= 0 && m.taskSelected < len(m.tasks) {
		selectedID = m.tasks[m.taskSelected].ID
	}
	items := make([]vimkunja.Task, 0, len(m.overviewTasks))
	today := time.Now().Local().Truncate(24 * time.Hour)
	for _, task := range m.overviewTasks {
		if overviewMatches(task, m.overviewFilter, today) {
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

func overviewMatches(task vimkunja.Task, filter overviewFilter, today time.Time) bool {
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
	default:
		return false
	}
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
		return leftHasDue
	}
	if leftHasDue && !leftDue.Equal(rightDue) {
		if mode == overviewDueAsc {
			return leftDue.Before(rightDue)
		}
		return leftDue.After(rightDue)
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

func overviewDueDate(task vimkunja.Task) (time.Time, bool) {
	if task.DueDate == "" {
		return time.Time{}, false
	}
	value, err := time.ParseInLocation("2006-01-02", task.DueDate, time.Local)
	return value, err == nil
}

func (m *model) updateOverviewPrompt(key string) bool {
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
			m.overviewSort = overviewDueDesc
		case "u":
			m.overviewSort = overviewDueAsc
		case "p":
			m.overviewSort = overviewPriorityDesc
		case "t":
			m.overviewSort = overviewTitle
		default:
			return true
		}
	} else {
		switch key {
		case "o":
			m.overviewFilter = overviewOpen
		case "a":
			m.overviewFilter = overviewAll
		case "v":
			m.overviewFilter = overviewOverdue
		case "t":
			m.overviewFilter = overviewDueToday
		default:
			return true
		}
	}
	m.applyOverview()
	return true
}

func (m *model) overviewFilterLabel() string {
	switch m.overviewFilter {
	case overviewAll:
		return "all"
	case overviewOverdue:
		return "overdue"
	case overviewDueToday:
		return "due today"
	default:
		return "open"
	}
}

func (m *model) overviewSortLabel() string {
	switch m.overviewSort {
	case overviewDueAsc:
		return "Due ↑"
	case overviewPriorityDesc:
		return "Priority ↓"
	case overviewTitle:
		return "Title"
	default:
		return "Due ↓"
	}
}
