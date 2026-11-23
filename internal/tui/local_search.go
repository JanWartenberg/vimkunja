package tui

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/JanWartenberg/vimkunja/internal/searchpattern"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

type localPosition struct {
	pane         paneID
	id           int64
	field        editorField
	line, offset int
}

type localMatch struct {
	position localPosition
	end      int
	target   description.Target
}

type localRecord struct {
	position localPosition
	text     string
}

type localOrigin struct {
	context                        searchOrigin
	projectTasks                   []vimkunja.Task
	filter                         overviewFilter
	kanbanTaskViewport             int
	kanbanEmpty, kanbanEmptyDetail bool
	navigation                     *taskEditor
	target                         description.Target
}

type localSearchInput struct {
	query        string
	cursor       int
	backward     bool
	origin       localOrigin
	start        localPosition
	historyIndex int
	historyDraft string
	pattern      *searchpattern.Pattern
	err          string
	matches      []localMatch
	selected     int
	wrapped      bool
}

// Local search owns no persisted data and never touches Global Search's snapshot.
type localSearchState struct {
	input         *localSearchInput
	pattern       string
	compiled      *searchpattern.Pattern
	backward      bool
	history       []string
	highlight     bool
	current       localPosition
	currentValid  bool
	currentTarget description.Target
	count         int
	pending       *localProjectPreview
}

type localProjectPreview struct {
	origin      localOrigin
	projectID   int64
	viewRequest uint64
}

func (m *model) localSearchEligible() bool {
	if m.helpPane != nil || m.infoPane != nil || m.searchOpen || m.commandMode || m.comments != nil || m.labelUI != nil || m.inlineTitle != nil || m.deleteConfirm || m.deleteArmed || m.kanbanPicker || m.kanbanViewPicker || m.kanbanProvisionConfirm || m.overviewPrompt != overviewPromptNone || m.ctrlWArmed || m.gArmed || m.zArmed || m.leaderArmed || m.viewSwitchArmed {
		return false
	}
	if m.focus == paneProjects || m.focus == paneTasks {
		return true
	}
	if m.focus != paneDetail || m.editor == nil {
		return false
	}
	e := m.editor
	return e.mode == editorNormal && !e.gArmed && !e.dArmed && e.nativeLink == nil && e.nativePending == nil && e.nativePrefix == "" && e.normal.selection == nil && e.normal.operator == 0 && e.normal.textObject == 0 && e.normal.replaceCount == 0 && (e.nativeDraft == nil || e.nativeDraft.Selection() == nil)
}

func (m *model) localSearchKey(key tea.KeyMsg) (tea.Cmd, bool) {
	if !m.localSearchEligible() {
		return nil, false
	}
	s := key.String()
	state := &m.localSearch
	if s == "/" || s == "?" {
		state.count = 0
		return m.openLocalSearch(s == "?"), true
	}
	if len(s) == 1 && s[0] >= '0' && s[0] <= '9' && (s != "0" || state.count > 0) {
		// Text editors already own Vim counts; leave theirs intact.
		if m.focus == paneDetail {
			return nil, false
		}
		state.count = min(100000, state.count*10+int(s[0]-'0'))
		return nil, true
	}
	if s == "n" || s == "N" {
		count := max(1, state.count)
		state.count = 0
		if m.focus == paneDetail {
			count = max(count, max(m.editor.nativeCount, m.editor.normal.count))
			m.editor.nativeCount, m.editor.normal.count = 0, 0
		}
		return m.repeatLocalSearch(s == "N", max(1, count)), true
	}
	state.count = 0
	return nil, false
}

func (m *model) captureLocalOrigin() localOrigin {
	o := localOrigin{context: m.captureSearchOrigin(), projectTasks: m.projectTasks, filter: m.taskFilter, kanbanTaskViewport: m.kanbanTaskViewportStart, kanbanEmpty: m.kanbanEmpty, kanbanEmptyDetail: m.kanbanEmptyDetailOrigin}
	if e := m.editor; e != nil {
		nav := *e
		if e.nativeDescription != nil {
			nav.nativeDescription = e.nativeDescription.Clone()
		}
		o.navigation = &nav
		if e.nativeDraft != nil {
			o.target = e.nativeDraft.Target()
		}
	}
	return o
}

func (m *model) restoreLocalOrigin(o localOrigin) {
	m.localSearch.pending = nil
	// Pending preview responses no longer own the restored context.
	m.viewRequest++
	m.taskRequest++
	m.pendingTaskID, m.pendingSearchDetailTaskID = 0, 0
	m.viewLoading, m.taskLoading = false, false
	m.restoreSearchOrigin(o.context)
	m.projectTasks, m.taskFilter = o.projectTasks, o.filter
	m.kanbanTaskViewportStart = o.kanbanTaskViewport
	m.kanbanEmpty, m.kanbanEmptyDetailOrigin = o.kanbanEmpty, o.kanbanEmptyDetail
	if e, nav := m.editor, o.navigation; e != nil && nav != nil {
		e.field, e.mode, e.cursor = nav.field, nav.mode, nav.cursor
		e.labelSelected = nav.labelSelected
		e.fieldViewportStart, e.detailScroll = nav.fieldViewportStart, nav.detailScroll
		e.descScroll, e.descViewportScroll = nav.descScroll, nav.descViewportScroll
		e.moreFieldsExpanded, e.descriptionActive, e.descriptionNavigated = nav.moreFieldsExpanded, nav.descriptionActive, nav.descriptionNavigated
		if nav.nativeDescription != nil {
			e.nativeDescription = nav.nativeDescription.Clone()
		}
		if e.nativeDraft != nil {
			_ = e.nativeDraft.SetTarget(o.target)
		}
	}
}

func (m *model) localSearchReady() bool {
	if m.projectLoading || m.viewLoading || m.taskLoading {
		m.toast = "Content is loading; retry local search"
		return false
	}
	if m.focus != paneDetail && m.editor != nil && m.editor.dirty() {
		m.recordError(errUnsavedNavigation)
		return false
	}
	return true
}

func (m *model) openLocalSearch(backward bool) tea.Cmd {
	if !m.localSearchReady() {
		return nil
	}
	if m.focus == paneDetail {
		m.editor.descriptionView()
	}
	m.localSearch.input = &localSearchInput{origin: m.captureLocalOrigin(), start: m.localPosition(), backward: backward, historyIndex: -1, selected: -1}
	return nil
}

func (m *model) updateLocalSearch(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	ui := m.localSearch.input
	switch key.String() {
	case "esc":
		m.restoreLocalOrigin(ui.origin)
		m.localSearch.input = nil
		return m, m.reloadLocalOriginMetadata()
	case "enter":
		if ui.query == "" {
			if m.localSearch.pattern == "" {
				ui.err = "No previous search pattern"
				return m, nil
			}
			ui.query = m.localSearch.pattern
			ui.cursor = utf8.RuneCountInString(ui.query)
			cmd := m.previewLocalSearch()
			if ui.pattern == nil {
				return m, cmd
			}
			m.acceptLocalSearch()
			return m, cmd
		}
		if ui.pattern == nil {
			return m, nil
		}
		m.acceptLocalSearch()
		return m, nil
	case "up", "down":
		m.recallLocalSearch(key.String() == "up")
		return m, m.previewLocalSearch()
	default:
		if _, changed := updateSingleLineTextInput(&ui.query, &ui.cursor, key); changed {
			ui.historyIndex = -1
			return m, m.previewLocalSearch()
		}
	}
	return m, nil
}

func (m *model) acceptLocalSearch() {
	s := &m.localSearch
	ui := s.input
	s.pattern, s.compiled, s.backward, s.highlight = ui.query, ui.pattern, ui.backward, true
	s.currentValid = ui.selected >= 0
	if s.currentValid {
		s.current = ui.matches[ui.selected].position
		s.currentTarget = ui.matches[ui.selected].target
	}
	s.history = append(s.history, ui.query)
	if len(s.history) > commandHistoryLimit {
		s.history = s.history[len(s.history)-commandHistoryLimit:]
	}
	if len(ui.matches) == 0 {
		m.toast = "Pattern not found: " + ui.query
	} else if ui.wrapped {
		m.toast = "Search wrapped"
	}
	s.input = nil
}

func (m *model) recallLocalSearch(older bool) {
	s := &m.localSearch
	ui := s.input
	if len(s.history) == 0 {
		return
	}
	if older {
		if ui.historyIndex < 0 {
			ui.historyDraft = ui.query
			ui.historyIndex = len(s.history) - 1
		} else if ui.historyIndex > 0 {
			ui.historyIndex--
		}
	} else {
		if ui.historyIndex < 0 {
			return
		}
		ui.historyIndex++
		if ui.historyIndex >= len(s.history) {
			ui.historyIndex = -1
			ui.query = ui.historyDraft
			ui.cursor = utf8.RuneCountInString(ui.query)
			return
		}
	}
	ui.query = s.history[ui.historyIndex]
	ui.cursor = utf8.RuneCountInString(ui.query)
}

func (m *model) previewLocalSearch() tea.Cmd {
	ui := m.localSearch.input
	m.restoreLocalOrigin(ui.origin)
	ui.err, ui.matches, ui.selected, ui.pattern = "", nil, -1, nil
	ui.wrapped = false
	if ui.query == "" {
		return m.reloadLocalOriginMetadata()
	}
	pattern, err := searchpattern.Compile(ui.query)
	if err != nil {
		ui.err = err.Error()
		return m.reloadLocalOriginMetadata()
	}
	ui.pattern = pattern
	ui.matches = m.localMatches(pattern)
	if len(ui.matches) == 0 {
		ui.err = "Pattern not found"
		return m.reloadLocalOriginMetadata()
	}
	ui.selected, ui.wrapped = chooseLocalMatch(ui.matches, ui.start, ui.backward, 1)
	return m.activateLocalMatch(ui.matches[ui.selected])
}

func (m *model) repeatLocalSearch(reverse bool, count int) tea.Cmd {
	s := &m.localSearch
	if s.compiled == nil {
		m.toast = "No previous search pattern"
		return nil
	}
	if !m.localSearchReady() {
		return nil
	}
	matches := m.localMatches(s.compiled)
	s.highlight = true
	if len(matches) == 0 {
		s.currentValid = false
		m.toast = "Pattern not found: " + s.pattern
		return nil
	}
	start := m.localPosition()
	i, wrapped := chooseLocalMatch(matches, start, s.backward != reverse, count)
	if wrapped {
		m.toast = "Search wrapped"
	}
	s.current, s.currentValid = matches[i].position, true
	s.currentTarget = matches[i].target
	return m.activateLocalMatch(matches[i])
}

func chooseLocalMatch(matches []localMatch, start localPosition, backward bool, count int) (int, bool) {
	// Positions are compared in the same order as the Pane's search records.
	index := -1
	for i, match := range matches {
		if match.position == start {
			index = i
			break
		}
	}
	if index >= 0 {
		delta := count
		if backward {
			delta = -delta
		}
		next := index + delta
		return (next%len(matches) + len(matches)) % len(matches), next < 0 || next >= len(matches)
	}
	// The caller supplies a record ordinal in line for non-Description fields.
	index = len(matches)
	for i, match := range matches {
		p := match.position
		if localPositionAfter(p, start) {
			index = i
			break
		}
	}
	if backward {
		index--
		for index >= 0 && matches[index].position.line == start.line && matches[index].position.field == start.field && matches[index].position.offset >= start.offset {
			index--
		}
		index -= count - 1
	} else {
		index += count - 1
	}
	return (index%len(matches) + len(matches)) % len(matches), index < 0 || index >= len(matches)
}

func localFieldOrder(p localPosition) int {
	if p.pane != paneTasks {
		return int(p.field)
	}
	switch p.field {
	case fieldDone:
		return 0
	case fieldPriority:
		return 1
	case fieldProject:
		return 2
	case fieldTitle:
		return 3
	case fieldDueDate:
		return 4
	}
	return int(p.field)
}

func localPositionAfter(p, start localPosition) bool {
	if p.pane == paneDetail && p.field != start.field {
		return p.field > start.field
	}
	if p.line != start.line {
		return p.line > start.line
	}
	if p.field != start.field {
		return localFieldOrder(p) > localFieldOrder(start)
	}
	return p.offset > start.offset
}

func (m *model) localPosition() localPosition {
	p := localPosition{pane: m.focus, offset: -1}
	s := &m.localSearch
	switch m.focus {
	case paneProjects:
		if project, ok := m.selectedProject(); ok {
			p.id = project.ID
		}
		for i, row := range m.localProjectRows() {
			if row.Project.ID == p.id {
				p.line = i
				break
			}
		}
	case paneTasks:
		p.field = fieldDone
		if task, ok := m.selectedTask(); ok {
			p.id = task.ID
		}
		for i, task := range m.localTasks() {
			if task.ID == p.id {
				p.line = i
				break
			}
		}
	case paneDetail:
		e := m.editor
		p.id, p.field = e.id, e.field
		value := e.fieldValue()
		if e.field != fieldTitle && e.field != fieldDescription {
			for _, record := range m.localRecords() {
				if record.position.field == e.field {
					value = record.text
					break
				}
			}
		}
		p.offset = len(string([]rune(value)[:clamp(e.cursor, 0, utf8.RuneCountInString(value))]))
		if e.field == fieldDescription && e.nativeDraft != nil {
			if s.currentValid && s.current.pane == paneDetail && s.current.id == e.id && s.current.field == fieldDescription && s.currentTarget == e.nativeDraft.Target() {
				return s.current
			}
			p.line, p.offset = e.descriptionView().SearchPosition()
		}
	}
	if s.currentValid && s.current.pane == p.pane && s.current.id == p.id && (p.pane != paneDetail || s.current.field == p.field && s.current.line == p.line && s.current.offset == p.offset) {
		return s.current
	}
	return p
}

func (m *model) localProjectRows() []projectTreeRow {
	m.projectRows()
	tree := newProjectTree(m.projects, nil)
	return tree.visibleRows()
}

func (m *model) localTasks() []vimkunja.Task {
	if !m.kanbanActive || m.inOverview() {
		return m.tasks
	}
	var tasks []vimkunja.Task
	for _, bucket := range m.kanbanBoard.Buckets {
		tasks = append(tasks, bucket.Tasks...)
	}
	return tasks
}

func (m *model) localRecords() []localRecord {
	var records []localRecord
	add := func(p localPosition, text string) { records = append(records, localRecord{p, ansi.Strip(text)}) }
	switch m.focus {
	case paneProjects:
		for i, row := range m.localProjectRows() {
			add(localPosition{pane: paneProjects, id: row.Project.ID, line: i}, row.Project.Title)
		}
	case paneTasks:
		layout := m.localTaskLayout()
		for i, task := range m.localTasks() {
			p := localPosition{pane: paneTasks, id: task.ID, line: i}
			p.field = fieldTitle
			add(p, task.Title)
			if m.kanbanActive && !m.inOverview() {
				continue
			}
			if layout.showPriority {
				p.field = fieldPriority
				add(p, prioritySummary(task.Priority))
			}
			if layout.showDue {
				p.field = fieldDueDate
				due := "-"
				if task.DueDate != "" {
					due = duePresentation(task, task.DueDate, m.currentTime())
				}
				add(p, due)
			}
			if layout.showProject {
				p.field = fieldProject
				add(p, localTaskProject(task))
			}
			p.field = fieldDone
			if task.Done {
				add(p, "[x]")
			} else {
				add(p, "[ ]")
			}
		}
	case paneDetail:
		e := m.editor
		p := localPosition{pane: paneDetail, id: e.id}
		for _, project := range m.projects {
			if project.ID == e.draft.ProjectID {
				p.field = fieldProject
				add(p, project.Title)
				break
			}
		}
		p.field = fieldTitle
		add(p, e.draft.Title)
		p.field = fieldPriority
		add(p, prioritySummary(e.draft.Priority))
		p.field = fieldDueDate
		due := e.draft.DueDate
		if due == "" {
			due = "-"
		} else {
			due = duePresentation(vimkunja.Task{DueDate: due}, due, m.currentTime())
		}
		add(p, due)
		if !e.labelsLoading {
			var labels []string
			for _, label := range e.labels {
				labels = append(labels, label.Title)
			}
			p.field = fieldLabels
			if len(labels) == 0 {
				add(p, "none")
			} else {
				add(p, strings.Join(labels, ", "))
			}
		}
		if !e.workflowStatusLoading {
			p.field = fieldWorkflowStatus
			add(p, e.workflowStatusLabel())
		}
		if e.id != 0 {
			p.field = fieldDone
			if e.draft.Done {
				add(p, "[x]")
			} else {
				add(p, "[ ]")
			}
		}
	}
	sort.SliceStable(records, func(i, j int) bool { return localPositionAfter(records[j].position, records[i].position) })
	return records
}

func localTaskProject(t vimkunja.Task) string {
	name := t.ProjectIdentifier
	if name == "" {
		name = t.ProjectTitle
	}
	if name == "" {
		name = fmt.Sprint(t.ProjectID)
	}
	return truncatePlain(name, 8)
}

func (m *model) localTaskLayout() taskMetadataLayout {
	w, h := m.viewportSize()
	if !m.projectsPaneCollapsed {
		w -= max(16, min(30, w/3))
	}
	w -= 4
	top := max(4, h-3)
	if m.showDetail || m.helpPane != nil || m.infoPane != nil {
		top = m.detailPaneTop()
	}
	if len(m.tasks) > max(0, top-3) {
		w -= 2
	}
	return taskMetadataLayoutForWidth(m.tasks, w, m.inOverview(), m.currentTime())
}

func (m *model) localMatches(pattern *searchpattern.Pattern) []localMatch {
	var matches []localMatch
	for _, record := range m.localRecords() {
		for _, r := range pattern.FindAll(record.text) {
			p := record.position
			p.offset = r.Start
			matches = append(matches, localMatch{position: p, end: r.End})
		}
	}
	if m.focus == paneDetail && m.editor != nil {
		for _, match := range m.editor.descriptionView().Search(pattern) {
			matches = append(matches, localMatch{position: localPosition{pane: paneDetail, id: m.editor.id, field: fieldDescription, line: match.Line, offset: match.Offset}, end: match.End, target: match.Target})
		}
	}
	return matches
}

func (m *model) activateLocalMatch(match localMatch) tea.Cmd {
	p := match.position
	switch p.pane {
	case paneProjects:
		origin := m.captureLocalOrigin()
		if ui := m.localSearch.input; ui != nil {
			origin = ui.origin
		}
		m.taskRequest++
		m.pendingTaskID, m.pendingSearchDetailTaskID = 0, 0
		m.projectSelected = m.findProjectIndex(p.id)
		m.taskScope = scopeProject
		m.editor, m.showDetail = nil, false
		cmd := m.onProjectSelectionChanged()
		m.localSearch.pending = &localProjectPreview{origin: origin, projectID: p.id, viewRequest: m.viewRequest}
		return cmd
	case paneTasks:
		if m.kanbanActive && !m.inOverview() {
			for b, bucket := range m.kanbanBoard.Buckets {
				for i, task := range bucket.Tasks {
					if task.ID == p.id {
						m.kanbanBucketSelected, m.kanbanTaskSelected = b, i
						m.kanbanViewportStart = b
						focus := m.focus
						m.openExistingEditor(task)
						m.focus = focus
						return tea.Batch(m.loadWorkflowStatus(task.ProjectID, task.ID), m.loadEditorLabels(task.ID), m.loadTaskComments(task.ID))
					}
				}
			}
		}
		for i, task := range m.tasks {
			if task.ID == p.id {
				m.taskSelected = i
				return m.previewSelectedTask()
			}
		}
	case paneDetail:
		e := m.editor
		e.field, e.mode = p.field, editorNormal
		e.descriptionActive = p.field == fieldDescription
		e.moreFieldsExpanded = p.field != fieldDescription
		if p.field == fieldDescription {
			_ = e.nativeDraft.SetTarget(match.target)
			e.nativeDraft.UpdateView(e.nativeDescription)
			e.nativeDescription.FollowCursor()
			e.descViewportScroll = false
		} else {
			for _, r := range m.localRecords() {
				if r.position.field == p.field {
					e.cursor = utf8.RuneCountInString(r.text[:p.offset])
					break
				}
			}
			if p.field == fieldLabels {
				offset := 0
				for i, label := range e.labels {
					if p.offset < offset+len(label.Title) {
						e.labelSelected = i
						break
					}
					offset += len(label.Title) + 2
				}
			}
		}
		m.ensureDetailFieldVisible()
	}
	return nil
}

func (m *model) localPattern() (*searchpattern.Pattern, localPosition, bool) {
	s := &m.localSearch
	if ui := s.input; ui != nil {
		if ui.selected < 0 {
			return nil, localPosition{}, false
		}
		return ui.pattern, ui.matches[ui.selected].position, true
	}
	current, _ := m.activeLocalMatch()
	return s.compiled, current, s.highlight
}

// The saved occurrence still guides n/N, but its active paint belongs only to
// the actual selection/cursor. Ordinary navigation must not leave an old accent.
func (m *model) activeLocalMatch() (localPosition, bool) {
	s := &m.localSearch
	if s.currentValid && m.focus == s.current.pane && (m.focus != paneDetail || m.editor != nil) && m.localPosition() == s.current {
		return s.current, true
	}
	return localPosition{pane: paneID(-1)}, false
}

// Only errors owned by the active preview can cancel it. Ordinary request and
// editor identity guards prevent late responses from a previous query winning.
func (m *model) localSearchLoadError(msg tea.Msg) (tea.Cmd, bool) {
	ui, pending := m.localSearch.input, m.localSearch.pending
	if ui == nil && pending == nil {
		return nil, false
	}
	var err error
	owned := false
	switch msg := msg.(type) {
	case viewsLoadedMsg:
		owned = msg.request == m.viewRequest && msg.projectID == m.currentProjectID
		err = msg.err
		if owned && err == nil {
			probe := *m
			probe.projectViews = msg.views
			if probe.pickListViewIndex() < 0 && (m.projectViewPreference != projectViewKanban || probe.pickManualKanbanViewIndex() < 0) {
				err = fmt.Errorf("Project has no supported Views")
			}
		}
	case tasksLoadedMsg:
		owned = msg.request == m.taskRequest && msg.projectID == m.currentProjectID && !m.inOverview()
		err = msg.err
		if owned && err == nil {
			m.localSearch.pending = nil
		}
	case kanbanBoardLoadedMsg:
		owned = msg.request == m.taskRequest && msg.projectID == m.currentProjectID && !m.inOverview()
		err = msg.err
		if owned && err == nil {
			if msg.board.Availability != vimkunja.KanbanBoardAvailable {
				err = fmt.Errorf("Kanban Board is no longer available")
			} else {
				m.localSearch.pending = nil
			}
		}
	case workflowStatusLoadedMsg:
		owned = m.editor != nil && (msg.editor == m.editor || msg.editor == nil && msg.taskID == m.editor.id)
		err = msg.err
	case taskLabelsLoadedMsg:
		owned = m.editor != nil && msg.taskID == m.editor.id
		err = msg.err
	case taskCommentsLoadedMsg:
		owned = m.editor != nil && msg.taskID == m.editor.id && msg.request == m.editor.commentsRequest
		err = msg.err
	}
	if !owned || err == nil {
		return nil, false
	}
	if ui != nil {
		m.restoreLocalOrigin(ui.origin)
		m.localSearch.input = nil
	} else if pending != nil && pending.projectID == m.currentProjectID && pending.viewRequest == m.viewRequest {
		m.restoreLocalOrigin(pending.origin)
	} else {
		return nil, false
	}
	m.recordError(fmt.Errorf("local search preview failed: %w", err))
	return m.reloadLocalOriginMetadata(), true
}

// Origin reads may have finished while a different preview editor owned the
// Pane. Reissue only still-missing metadata after restoring that editor.
func (m *model) reloadLocalOriginMetadata() tea.Cmd {
	e := m.editor
	if e == nil {
		return nil
	}
	var commands []tea.Cmd
	if e.workflowStatusLoading {
		commands = append(commands, m.loadWorkflowStatus(e.draft.ProjectID, e.id))
	}
	if e.labelsLoading {
		commands = append(commands, m.loadEditorLabels(e.id))
	}
	if e.commentsLoading {
		commands = append(commands, m.loadTaskComments(e.id))
	}
	return tea.Batch(commands...)
}

func noHighlightCommand(m *model, args []string) (tea.Model, tea.Cmd) {
	if len(args) != 0 {
		m.recordError(fmt.Errorf("usage: :nohlsearch"))
		return m, nil
	}
	m.localSearch.highlight = false
	return m, nil
}

func (m *model) localSearchFooter(width int) string {
	s := &m.localSearch
	if ui := s.input; ui != nil {
		prefix := "/"
		if ui.backward {
			prefix = "?"
		}
		feedback := ui.err
		if feedback == "" && ui.selected >= 0 {
			feedback = fmt.Sprintf("[%d/%d]", ui.selected+1, len(ui.matches))
			if ui.wrapped {
				feedback += " wrapped"
			}
		}
		prompt := prefix + insertCursorText(ui.query, ui.cursor)
		if feedback != "" {
			prompt += "  " + feedback
		}
		return fitInputCursor(prompt, width)
	}
	if !s.highlight || s.compiled == nil || !m.localSearchEligible() {
		return ""
	}
	matches := m.localMatches(s.compiled)
	index := 0
	current, active := m.activeLocalMatch()
	for i, match := range matches {
		if active && match.position == current {
			index = i + 1
			break
		}
	}
	return fmt.Sprintf("Search [%d/%d]  n/N next/previous  •  Space h clear", index, len(matches))
}

func (m *model) refreshLocalSearchMetadata() tea.Cmd {
	if ui := m.localSearch.input; ui != nil && ui.origin.context.focus == paneDetail {
		return m.previewLocalSearch()
	}
	return nil
}
