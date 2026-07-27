package main

import (
	"sort"
	"time"
)

// This pure state module is intentionally separate from the Bubble Tea shell.
// It answers the prototype question: can Overview be a peer Pane to Tasks while
// default sorting and small Vim/netrw-like view controls remain comprehensible?

type prototypeTask struct {
	ID       int
	Project  string
	Title    string
	Due      time.Time
	Priority int
	Bucket   string
	Done     bool
}

type surface int

const (
	surfaceOverview surface = iota
	surfaceProjectTasks
)

type sortMode int

const (
	sortDueDesc sortMode = iota
	sortDueAsc
	sortPriorityDesc
	sortTitle
)

type filterMode int

const (
	filterOpen filterMode = iota
	filterAll
	filterOverdue
	filterDueToday
)

type prototypeState struct {
	surface         surface
	selectedProject int
	selectedTask    int
	sort            sortMode
	filter          filterMode
	detailTask      *prototypeTask
	sortMenu        bool
	filterMenu      bool
	message         string
}

var prototypeProjects = []string{"HOME", "VIM", "ACME", "IDEAS"}

var prototypeTasks = []prototypeTask{
	{21, "VIM", "All Tasks Overview spezifizieren", due("2026-07-30"), 3, "To Do", false},
	{22, "VIM", "Kanban View als Tasks Pane rendern", due("2026-07-29"), 3, "Doing", false},
	{23, "HOME", "Zahnarzt anrufen", due("2026-07-24"), 4, "Inbox", false},
	{24, "ACME", "Angebot gegenlesen", due("2026-08-01"), 2, "Review", false},
	{25, "HOME", "Wocheneinkauf planen", due("2026-07-25"), 1, "To Do", false},
	{26, "IDEAS", "Suche: Labels als Filterziel", time.Time{}, 1, "Backlog", false},
	{27, "VIM", "Task löschen mit Bestätigung", due("2026-07-28"), 2, "To Do", false},
	{28, "ACME", "Stundenzettel senden", due("2026-07-23"), 2, "Done", true},
	{29, "HOME", "Fahrrad zur Inspektion bringen", time.Time{}, 3, "Someday", false},
}

func due(value string) time.Time {
	result, _ := time.Parse("2006-01-02", value)
	return result
}

func (s prototypeState) paneName() string {
	if s.surface == surfaceOverview {
		return "Overview Pane"
	}
	return "Tasks Pane · " + prototypeProjects[s.selectedProject]
}

func (s prototypeState) visibleTasks() []prototypeTask {
	items := make([]prototypeTask, 0, len(prototypeTasks))
	for _, task := range prototypeTasks {
		if s.surface == surfaceProjectTasks && task.Project != prototypeProjects[s.selectedProject] {
			continue
		}
		if !matchesFilter(task, s.filter) {
			continue
		}
		items = append(items, task)
	}
	sortTasks(items, s.sort)
	return items
}

func matchesFilter(task prototypeTask, filter filterMode) bool {
	switch filter {
	case filterOpen:
		return !task.Done
	case filterAll:
		return true
	case filterOverdue:
		return !task.Done && !task.Due.IsZero() && task.Due.Before(due("2026-07-25"))
	case filterDueToday:
		return !task.Done && task.Due.Equal(due("2026-07-25"))
	default:
		return false
	}
}

func sortTasks(items []prototypeTask, mode sortMode) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		switch mode {
		case sortDueDesc:
			if left.Due.IsZero() != right.Due.IsZero() {
				return !left.Due.IsZero()
			}
			if !left.Due.IsZero() && !left.Due.Equal(right.Due) {
				return left.Due.After(right.Due)
			}
			return priorityThenTitle(left, right)
		case sortDueAsc:
			// Default: dated Tasks closest to today come first; undated Tasks are
			// then ordered by priority.
			if left.Due.IsZero() != right.Due.IsZero() {
				return !left.Due.IsZero()
			}
			if !left.Due.IsZero() && !left.Due.Equal(right.Due) {
				return left.Due.Before(right.Due)
			}
			return priorityThenTitle(left, right)
		case sortPriorityDesc:
			return priorityThenTitle(left, right)
		case sortTitle:
			return left.Title < right.Title
		default:
			return false
		}
	})
}

func priorityThenTitle(left, right prototypeTask) bool {
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	return left.Title < right.Title
}

func sortLabel(mode sortMode) string {
	switch mode {
	case sortDueDesc:
		return "Due date ↓ · undated: Priority ↓"
	case sortDueAsc:
		return "Due date ↑ · undated: Priority ↓"
	case sortPriorityDesc:
		return "Priority ↓"
	case sortTitle:
		return "Title A–Z"
	default:
		return "unknown"
	}
}

func filterLabel(mode filterMode) string {
	switch mode {
	case filterOpen:
		return "open only"
	case filterAll:
		return "all Tasks"
	case filterOverdue:
		return "overdue"
	case filterDueToday:
		return "due today"
	default:
		return "unknown"
	}
}
