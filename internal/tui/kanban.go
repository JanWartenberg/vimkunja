package tui

import (
	"context"
	"strings"
	"time"

	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m *model) updateKanbanProvisionConfirmation(key string) (tea.Model, tea.Cmd) {
	m.kanbanProvisionConfirm = false
	switch key {
	case "y":
		m.taskLoading = true
		return m, m.provisionStandardKanbanView(m.currentProjectID)
	case "n", "esc":
		return m, nil
	default:
		return m, nil
	}
}

func (m *model) currentProjectTitle() string {
	for _, project := range m.projects {
		if project.ID == m.currentProjectID {
			return project.Title
		}
	}
	return "Project"
}

func (m *model) setKanbanBoard(board vimkunja.KanbanBoard) {
	m.kanbanActive = true
	m.kanbanBoard = board
	m.currentViewID = board.View.ID
	m.syncOpenDetailDoneState(board)

	bucket := 0
	if m.pendingKanbanBucketID != 0 {
		for i, candidate := range board.Buckets {
			if candidate.ID == m.pendingKanbanBucketID {
				bucket = i
				break
			}
		}
	} else if m.pendingKanbanTaskID != 0 {
		found := false
		for i, candidate := range board.Buckets {
			for _, task := range candidate.Tasks {
				if task.ID == m.pendingKanbanTaskID {
					bucket = i
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	if m.pendingKanbanBucketID == 0 && m.pendingKanbanTaskID == 0 {
		for i, candidate := range board.Buckets {
			if len(candidate.Tasks) > 0 {
				bucket = i
				break
			}
		}
	}
	m.kanbanBucketSelected = clamp(bucket, 0, max(0, len(board.Buckets)-1))
	m.kanbanTaskSelected = -1
	if len(board.Buckets) > 0 {
		for i, task := range board.Buckets[m.kanbanBucketSelected].Tasks {
			if task.ID == m.pendingKanbanTaskID {
				m.kanbanTaskSelected = i
				break
			}
		}
		if m.kanbanTaskSelected < 0 && len(board.Buckets[m.kanbanBucketSelected].Tasks) > 0 {
			m.kanbanTaskSelected = 0
		}
	}
	m.pendingKanbanTaskID = 0
	m.pendingKanbanBucketID = 0
	m.ensureKanbanBucketVisible()
}

// syncOpenDetailDoneState keeps an open Detail Pane consistent with a
// reloaded Board after a workflow move changes the global Done state.
func (m *model) syncOpenDetailDoneState(board vimkunja.KanbanBoard) {
	if m.editor == nil || m.editor.id == 0 {
		return
	}
	for _, bucket := range board.Buckets {
		for _, task := range bucket.Tasks {
			if task.ID != m.editor.id {
				continue
			}
			m.editor.draft.Done = task.Done
			m.editor.original.Done = task.Done
			m.editor.draft.DoneSet = false
			return
		}
	}
}

func (m *model) activeKanbanBucket() (vimkunja.KanbanBucket, bool) {
	if m.kanbanBucketSelected < 0 || m.kanbanBucketSelected >= len(m.kanbanBoard.Buckets) {
		return vimkunja.KanbanBucket{}, false
	}
	return m.kanbanBoard.Buckets[m.kanbanBucketSelected], true
}

func (m *model) selectedKanbanTask() (vimkunja.Task, bool) {
	bucket, ok := m.activeKanbanBucket()
	if !ok || m.kanbanTaskSelected < 0 || m.kanbanTaskSelected >= len(bucket.Tasks) {
		return vimkunja.Task{}, false
	}
	return bucket.Tasks[m.kanbanTaskSelected], true
}

func (m *model) updateKanbanKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := normalizeArrow(key.String())
	if m.kanbanPicker {
		return m.updateKanbanPicker(s)
	}
	if m.deleteArmed {
		m.deleteArmed = false
		if s == "d" {
			m.beginTaskDeletion()
		}
		return m, nil
	}

	switch s {
	case "q":
		return m, tea.Quit
	case "enter":
		if task, ok := m.selectedKanbanTask(); ok {
			m.openExistingEditor(task)
			return m, tea.Batch(m.loadWorkflowStatus(task.ProjectID, task.ID), m.loadEditorLabels(task.ID))
		}
	case "o":
		m.openNewEditor()
	case "r":
		return m, m.refreshActiveTasks()
	case "-":
		return m, m.openOverview()
	case "x":
		if task, ok := m.selectedKanbanTask(); ok {
			m.taskLoading = true
			return m, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				defer cancel()
				return taskToggledMsg{err: m.store.ToggleTaskDone(ctx, task.ID, !task.Done)}
			}
		}
	case "d":
		m.deleteArmed = true
	case "j":
		if bucket, ok := m.activeKanbanBucket(); ok && len(bucket.Tasks) > 0 {
			m.kanbanTaskSelected = clamp(m.kanbanTaskSelected+1, 0, len(bucket.Tasks)-1)
		}
	case "k":
		if bucket, ok := m.activeKanbanBucket(); ok && len(bucket.Tasks) > 0 {
			m.kanbanTaskSelected = clamp(m.kanbanTaskSelected-1, 0, len(bucket.Tasks)-1)
		}
	case "h":
		m.moveKanbanSelection(-1)
	case "l":
		m.moveKanbanSelection(1)
	case "H":
		return m.moveSelectedKanbanTask(-1)
	case "L":
		return m.moveSelectedKanbanTask(1)
	case "m":
		if len(m.kanbanPickerTargets()) > 0 {
			m.kanbanPicker = true
			m.kanbanPickerIndex = 0
		}
	}
	return m, nil
}

func (m *model) moveKanbanSelection(delta int) {
	from := m.kanbanBucketSelected
	to := clamp(from+delta, 0, max(0, len(m.kanbanBoard.Buckets)-1))
	if to == from {
		return
	}
	row := m.kanbanTaskSelected
	m.kanbanBucketSelected = to
	m.kanbanTaskSelected = -1
	if tasks := m.kanbanBoard.Buckets[to].Tasks; len(tasks) > 0 {
		if row >= 0 && row < len(tasks) {
			m.kanbanTaskSelected = row
		} else if delta < 0 {
			m.kanbanTaskSelected = len(tasks) - 1
		} else {
			m.kanbanTaskSelected = 0
		}
	}
	m.ensureKanbanBucketVisible()
}

func (m *model) moveSelectedKanbanTask(delta int) (tea.Model, tea.Cmd) {
	task, ok := m.selectedKanbanTask()
	if !ok {
		return m, nil
	}
	target := m.kanbanBucketSelected + delta
	if target < 0 || target >= len(m.kanbanBoard.Buckets) {
		return m, nil
	}
	m.taskLoading = true
	return m, m.moveKanbanTask(task.ID, m.kanbanBoard.Buckets[target].ID)
}

func (m *model) kanbanPickerTargets() []int {
	targets := make([]int, 0, len(m.kanbanBoard.Buckets)-1)
	for i := range m.kanbanBoard.Buckets {
		if i != m.kanbanBucketSelected {
			targets = append(targets, i)
		}
	}
	return targets
}

func (m *model) updateKanbanPicker(key string) (tea.Model, tea.Cmd) {
	targets := m.kanbanPickerTargets()
	if len(targets) == 0 {
		m.kanbanPicker = false
		return m, nil
	}
	switch key {
	case "esc":
		m.kanbanPicker = false
	case "j":
		m.kanbanPickerIndex = clamp(m.kanbanPickerIndex+1, 0, len(targets)-1)
	case "k":
		m.kanbanPickerIndex = clamp(m.kanbanPickerIndex-1, 0, len(targets)-1)
	case "enter":
		task, ok := m.selectedKanbanTask()
		m.kanbanPicker = false
		if !ok {
			return m, nil
		}
		m.taskLoading = true
		return m, m.moveKanbanTask(task.ID, m.kanbanBoard.Buckets[targets[m.kanbanPickerIndex]].ID)
	}
	return m, nil
}

func (m *model) visibleKanbanColumns() int {
	width, _ := m.viewportSize()
	if !m.projectsPaneCollapsed {
		projectsOuter := min(30, width/3)
		projectsOuter = max(16, projectsOuter)
		width -= projectsOuter
	}
	return max(1, (width-4)/25)
}

func (m *model) ensureKanbanBucketVisible() {
	visible := m.visibleKanbanColumns()
	if m.kanbanBucketSelected < m.kanbanViewportStart {
		m.kanbanViewportStart = m.kanbanBucketSelected
	}
	if m.kanbanBucketSelected >= m.kanbanViewportStart+visible {
		m.kanbanViewportStart = m.kanbanBucketSelected - visible + 1
	}
	m.kanbanViewportStart = clamp(m.kanbanViewportStart, 0, max(0, len(m.kanbanBoard.Buckets)-visible))
}

// renderKanbanBoard is deliberately separate from the list renderer: a Board
// owns a two-dimensional selection and horizontal viewport rather than a flat
// Task slice.
func renderKanbanBoard(m *model, textW, textH int) string {
	if m.taskLoading {
		return fitBlock([]string{" Kanban ", "  loading..."}, textW, textH)
	}
	if m.kanbanPicker {
		return renderKanbanPicker(m, textW, textH)
	}
	visible := max(1, textW/25)
	start := clamp(m.kanbanViewportStart, 0, max(0, len(m.kanbanBoard.Buckets)-visible))
	end := min(len(m.kanbanBoard.Buckets), start+visible)
	columns := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		columnWidth := 25
		if len(m.kanbanBoard.Buckets) <= visible {
			columnCount := end - start
			columnWidth = textW / columnCount
			if index-start < textW%columnCount {
				columnWidth++
			}
		}
		contentWidth := max(1, columnWidth-2)
		bucket := m.kanbanBoard.Buckets[index]
		title := bucket.Title
		if index == start && start > 0 {
			title = "< " + title
		}
		if index == end-1 && end < len(m.kanbanBoard.Buckets) {
			title += " >"
		}
		header := " " + truncatePlain(title, contentWidth) + " "
		if index == m.kanbanBucketSelected {
			header = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render(header)
		} else {
			header = lipgloss.NewStyle().Bold(true).Render(header)
		}
		lines := []string{header, ""}
		for taskIndex, task := range bucket.Tasks {
			lines = append(lines, renderKanbanTaskLine(task, index == m.kanbanBucketSelected && taskIndex == m.kanbanTaskSelected, contentWidth))
		}
		content := fitBlock(lines, contentWidth, max(1, textH-2))
		columns = append(columns, lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(tuiTheme.muted)).
			Render(content))
	}
	if len(columns) == 0 {
		return fitBlock([]string{" Kanban ", "  (no buckets)"}, textW, textH)
	}
	return fitBlock(strings.Split(lipgloss.JoinHorizontal(lipgloss.Top, columns...), "\n"), textW, textH)
}

func renderKanbanTaskLine(task vimkunja.Task, selected bool, contentWidth int) string {
	prefix := " "
	if task.Done {
		prefix = "✓ "
	}
	title := truncatePlain(task.Title, max(1, contentWidth-lipgloss.Width(prefix)))
	if selected {
		title = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render(title)
	}
	return prefix + title
}

func renderKanbanPicker(m *model, textW, textH int) string {
	task, _ := m.selectedKanbanTask()
	lines := []string{` Move "` + task.Title + `" to:`}
	targets := m.kanbanPickerTargets()
	for i, bucketIndex := range targets {
		prefix := "  "
		if i == m.kanbanPickerIndex {
			prefix = "▸ "
		}
		lines = append(lines, prefix+m.kanbanBoard.Buckets[bucketIndex].Title)
	}
	lines = append(lines, "", " j/k choose  •  Enter move  •  Esc cancel")
	return fitBlock(lines, textW, textH)
}
