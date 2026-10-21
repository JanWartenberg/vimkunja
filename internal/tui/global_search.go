package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

type searchInputMode int

const (
	searchInsertMode searchInputMode = iota
	searchNormalMode
)

type searchOrigin struct {
	focus                                    paneID
	projectsPaneCollapsed, projectsPaneAuto  bool
	lastProjectPaneCollapsed                 bool
	projectSelected, projectViewportStart    int
	currentProjectID, currentViewID          int64
	projectViews                             []vimkunja.ProjectView
	viewSelected                             int
	projectViewPreference                    projectViewPreference
	kanbanActive                             bool
	kanbanBoard                              vimkunja.KanbanBoard
	kanbanBucketSelected, kanbanTaskSelected int
	kanbanViewportStart                      int
	tasks                                    []vimkunja.Task
	taskSelected                             int
	taskScope                                taskScope
	overviewTasks                            []vimkunja.Task
	overviewLabel                            vimkunja.Label
	overviewFilter                           overviewFilter
	overviewSort                             overviewSort
	overviewPrompt                           overviewPrompt
	showDetail                               bool
	editor                                   *taskEditor
	projectTreeInitialized                   bool
	projectTreeCollapsed                     map[int64]bool
}

type searchReturn struct {
	origin   searchOrigin
	input    string
	mode     searchInputMode
	results  []vimkunja.SearchResult
	selected int
	viewport int
}

// openSearch leaves the current model state intact until a result is
// activated, so Esc can restore the exact pre-search context without a reload.
func (m *model) openSearch() {
	if m.editor != nil && m.editor.dirty() {
		m.recordError(errUnsavedNavigation)
		return
	}
	origin := m.captureSearchOrigin()
	m.searchOrigin = &origin
	m.searchOpen = true
	m.searchMode = searchInsertMode
	m.searchCommandMode = false
	m.searchCommandInput = ""
	m.searchCommandCursor = 0
	m.searchGArmed = false
	m.searchInput = ""
	m.searchInputCursor = 0
	m.searchResults = nil
	m.searchSelected = 0
	m.searchViewport = 0
	m.searchLoading = false
}

func (m *model) updateSearchKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searchCommandMode {
		return m.updateSearchCommandKeys(key)
	}
	if m.searchMode == searchNormalMode {
		return m.updateSearchNormalKeys(key)
	}
	return m.updateSearchInsertKeys(key)
}

func (m *model) updateSearchInsertKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(key.String()) {
	case "enter":
		return m, m.activateSelectedSearchResult()
	case "esc":
		m.searchMode = searchNormalMode
		m.searchGArmed = false
		return m, nil
	case "backspace", "delete", "left", "right", "home", "end":
		_, changed := updateSingleLineTextInput(&m.searchInput, &m.searchInputCursor, key)
		if changed {
			return m, m.beginSearchQuery()
		}
		return m, nil
	case "up", "ctrl+p":
		m.moveSearchSelection(-1)
		return m, nil
	case "down", "ctrl+n":
		m.moveSearchSelection(1)
		return m, nil
	case "ctrl+d", "pgdown", "pagedown":
		m.moveSearchSelection(m.searchVisibleResultRows() / 2)
		return m, nil
	case "ctrl+u", "pgup", "pageup":
		m.moveSearchSelection(-m.searchVisibleResultRows() / 2)
		return m, nil
	}
	if _, changed := updateSingleLineTextInput(&m.searchInput, &m.searchInputCursor, key); changed {
		return m, m.beginSearchQuery()
	}
	return m, nil
}

func (m *model) updateSearchNormalKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	value := key.String()
	if m.searchGArmed {
		m.searchGArmed = false
		if value == "g" {
			m.searchSelected, m.searchViewport = 0, 0
		}
		return m, nil
	}
	switch value {
	case "esc":
		m.closeSearch()
	case "i", "a":
		m.searchMode = searchInsertMode
	case ":":
		m.searchCommandMode = true
		m.searchCommandInput, m.searchCommandCursor = "", 0
	case "j", "down", "ctrl+n":
		m.moveSearchSelection(1)
	case "k", "up", "ctrl+p":
		m.moveSearchSelection(-1)
	case "g":
		m.searchGArmed = true
	case "G", "end":
		m.searchSelected = max(0, len(m.searchResults)-1)
		m.moveSearchSelection(0)
	case "ctrl+d", "pgdown", "pagedown":
		m.moveSearchSelection(m.searchVisibleResultRows() / 2)
	case "ctrl+u", "pgup", "pageup":
		m.moveSearchSelection(-m.searchVisibleResultRows() / 2)
	case "enter":
		return m, m.activateSelectedSearchResult()
	}
	return m, nil
}

func (m *model) updateSearchCommandKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(key.String()) {
	case "esc":
		m.searchCommandMode = false
		m.searchCommandInput = ""
		return m, nil
	case "backspace", "delete", "left", "right", "home", "end":
		updateSingleLineTextInput(&m.searchCommandInput, &m.searchCommandCursor, key)
		return m, nil
	case "enter":
		command := strings.ToLower(strings.TrimSpace(m.searchCommandInput))
		m.searchCommandMode = false
		m.searchCommandInput = ""
		switch command {
		case "q", "quit":
			m.closeSearch()
		case "":
		default:
			m.recordError(fmt.Errorf("unknown Search command: %s", command))
		}
		return m, nil
	}
	updateSingleLineTextInput(&m.searchCommandInput, &m.searchCommandCursor, key)
	return m, nil
}

func (m *model) activateSelectedSearchResult() tea.Cmd {
	if m.searchSelected < 0 || m.searchSelected >= len(m.searchResults) {
		return nil
	}
	result := m.searchResults[m.searchSelected]
	if m.searchOrigin != nil {
		m.searchReturn = &searchReturn{
			origin:   *m.searchOrigin,
			input:    m.searchInput,
			mode:     m.searchMode,
			results:  append([]vimkunja.SearchResult(nil), m.searchResults...),
			selected: m.searchSelected,
			viewport: m.searchViewport,
		}
	}
	m.searchOrigin = nil
	m.searchOpen = false
	m.searchCommandMode = false
	m.searchGArmed = false
	m.searchLoading = false
	m.searchRequest++
	switch result.Kind {
	case vimkunja.SearchLabelResult:
		return m.openLabelOverview(vimkunja.Label{ID: result.LabelID, Title: result.Primary})
	case vimkunja.SearchTaskResult, vimkunja.SearchDescriptionResult:
		return m.openSearchProject(result.ProjectID, result.TaskID)
	case vimkunja.SearchProjectResult:
		return m.openSearchProject(result.ProjectID, 0)
	default:
		return nil
	}
}

// openSearchProject enters the owning Project's List View, retaining the
// selected Task identity until its list has been loaded.
func (m *model) openSearchProject(projectID, detailTaskID int64) tea.Cmd {
	projectIndex := m.findProjectIndex(projectID)
	if projectIndex < 0 {
		m.recordError(fmt.Errorf("Search result Project %d is no longer available", projectID))
		return nil
	}
	m.projectSelected = projectIndex
	m.currentProjectID = projectID
	m.currentViewID = 0
	m.projectViewPreference = projectViewList
	m.taskScope = scopeProject
	m.overviewLabel = vimkunja.Label{}
	m.overviewPrompt = overviewPromptNone
	m.kanbanActive = false
	m.kanbanBoard = vimkunja.KanbanBoard{}
	m.showDetail = false
	m.editor = nil
	m.focus = paneTasks
	m.tasks = nil
	m.taskSelected = 0
	m.pendingTaskID = detailTaskID
	m.pendingSearchDetailTaskID = detailTaskID
	m.viewLoading = true
	m.taskLoading = true
	return m.loadViews(projectID)
}

func (m *model) closeSearch() {
	if m.searchOrigin != nil {
		m.restoreSearchOrigin(*m.searchOrigin)
	}
	m.searchOrigin = nil
	m.searchOpen = false
	m.searchMode = searchInsertMode
	m.searchCommandMode = false
	m.searchCommandInput = ""
	m.searchGArmed = false
	m.searchInput = ""
	m.searchResults = nil
	m.searchSelected = 0
	m.searchViewport = 0
	m.searchLoading = false
	m.searchRequest++
}

func (m *model) captureSearchOrigin() searchOrigin {
	return searchOrigin{
		focus: m.focus, projectsPaneCollapsed: m.projectsPaneCollapsed, projectsPaneAuto: m.projectsPaneAutoShown,
		lastProjectPaneCollapsed: m.lastProjectPaneCollapsed, projectSelected: m.projectSelected, projectViewportStart: m.projectViewportStart,
		currentProjectID: m.currentProjectID, currentViewID: m.currentViewID, projectViews: append([]vimkunja.ProjectView(nil), m.projectViews...),
		viewSelected: m.viewSelected, projectViewPreference: m.projectViewPreference, kanbanActive: m.kanbanActive, kanbanBoard: m.kanbanBoard,
		kanbanBucketSelected: m.kanbanBucketSelected, kanbanTaskSelected: m.kanbanTaskSelected, kanbanViewportStart: m.kanbanViewportStart,
		tasks: append([]vimkunja.Task(nil), m.tasks...), taskSelected: m.taskSelected, taskScope: m.taskScope,
		overviewTasks: append([]vimkunja.Task(nil), m.overviewTasks...), overviewLabel: m.overviewLabel, overviewFilter: m.overviewFilter,
		overviewSort: m.overviewSort, overviewPrompt: m.overviewPrompt, showDetail: m.showDetail, editor: m.editor,
		projectTreeInitialized: m.projectTree.initialized(), projectTreeCollapsed: m.projectTree.collapsedState(),
	}
}

func (m *model) restoreSearchOrigin(origin searchOrigin) {
	m.focus = origin.focus
	m.projectsPaneCollapsed, m.projectsPaneAutoShown = origin.projectsPaneCollapsed, origin.projectsPaneAuto
	m.lastProjectPaneCollapsed = origin.lastProjectPaneCollapsed
	m.projectSelected, m.projectViewportStart = origin.projectSelected, origin.projectViewportStart
	m.currentProjectID, m.currentViewID = origin.currentProjectID, origin.currentViewID
	m.projectViews, m.viewSelected, m.projectViewPreference = append([]vimkunja.ProjectView(nil), origin.projectViews...), origin.viewSelected, origin.projectViewPreference
	m.kanbanActive, m.kanbanBoard = origin.kanbanActive, origin.kanbanBoard
	m.kanbanBucketSelected, m.kanbanTaskSelected, m.kanbanViewportStart = origin.kanbanBucketSelected, origin.kanbanTaskSelected, origin.kanbanViewportStart
	m.tasks, m.taskSelected, m.taskScope = append([]vimkunja.Task(nil), origin.tasks...), origin.taskSelected, origin.taskScope
	m.overviewTasks, m.overviewLabel = append([]vimkunja.Task(nil), origin.overviewTasks...), origin.overviewLabel
	m.overviewFilter, m.overviewSort, m.overviewPrompt = origin.overviewFilter, origin.overviewSort, origin.overviewPrompt
	m.showDetail, m.editor = origin.showDetail, origin.editor
	if origin.projectTreeInitialized {
		m.projectTree = newProjectTree(m.projects, origin.projectTreeCollapsed)
	} else {
		m.projectTree = projectTree{}
	}
}

func (m *model) restoreLastSearch() tea.Cmd {
	if m.editor != nil && m.editor.dirty() {
		m.recordError(errUnsavedNavigation)
		return nil
	}
	snapshot := m.searchReturn
	// The restored snapshot owns the pane; pending results belong to the
	// context we are leaving, even when its Project or Label IDs match.
	m.viewRequest++
	m.taskRequest++
	m.viewLoading, m.taskLoading = false, false
	m.restoreSearchOrigin(snapshot.origin)
	origin := snapshot.origin
	m.searchOrigin = &origin
	m.searchOpen, m.searchInput = true, snapshot.input
	m.searchInputCursor = len([]rune(m.searchInput))
	m.searchMode = snapshot.mode
	m.searchCommandMode = false
	m.searchCommandInput = ""
	m.searchGArmed = false
	m.searchResults = append([]vimkunja.SearchResult(nil), snapshot.results...)
	m.searchSelected, m.searchViewport = snapshot.selected, snapshot.viewport
	m.searchLoading = false
	return nil
}

func (m *model) beginSearchQuery() tea.Cmd {
	m.searchRequest++
	request, query := m.searchRequest, m.searchInput
	m.searchSelected = 0
	m.searchViewport = 0
	if len(strings.TrimSpace(query)) == 0 {
		m.searchResults = nil
		m.searchLoading = false
		return nil
	}
	m.searchLoading = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		data, err := m.store.LoadSearchData(ctx)
		if err != nil {
			return searchLoadedMsg{request: request, query: query, err: err}
		}
		return searchLoadedMsg{request: request, query: query, results: vimkunja.QuerySearch(data, query)}
	}
}

func (m *model) moveSearchSelection(delta int) {
	if len(m.searchResults) == 0 {
		return
	}
	m.searchSelected = clamp(m.searchSelected+delta, 0, len(m.searchResults)-1)
	if m.searchSelected < m.searchViewport {
		m.searchViewport = m.searchSelected
	}
	// Result heights vary: Description entries consume two physical lines and
	// each newly visible kind needs a group heading. Advance the viewport only
	// until the selected result fits in the actual rendered window.
	for m.searchSelected >= m.searchResultEnd(m.searchViewport) && m.searchViewport < m.searchSelected {
		m.searchViewport++
	}
}

const searchNarrowViewportMax = 80

// searchModalWidth keeps the overlay close to edge-to-edge on narrow
// terminals, then scales with the viewport rather than stopping at a fixed
// character width on wide terminals.
func searchModalWidth(viewportWidth int) int {
	if viewportWidth <= searchNarrowViewportMax {
		return max(1, viewportWidth-4)
	}
	return max(1, viewportWidth*94/100)
}

func searchModalHeight(viewportHeight int) int {
	return max(1, viewportHeight*80/100)
}

func (m *model) searchResultLineBudget() int {
	viewportHeight := m.height
	if viewportHeight <= 0 {
		viewportHeight = 30
	}
	// Header, input, separator, borders, and padding consume eleven rows.
	// lipgloss Height is a minimum: leaving this final row free prevents a
	// full result window from expanding the otherwise fixed-height modal.
	budget := searchModalHeight(viewportHeight) - 11
	if m.searchCommandMode {
		// The local : prompt consumes two additional body rows.
		budget -= 2
	}
	return max(1, budget)
}

func (m *model) searchVisibleResultRows() int {
	// A page movement is approximate; rendering uses searchResultEnd below for
	// the exact physical-line limit.
	return max(1, m.searchResultLineBudget()/2)
}

// searchResultEnd returns the exclusive end of the result window beginning at
// start. It budgets rendered lines, rather than result count: Description
// snippets need a second line and each kind transition needs a heading.
func (m *model) searchResultEnd(start int) int {
	start = clamp(start, 0, len(m.searchResults))
	used := 0
	lastKind := vimkunja.SearchResultKind(-1)
	for index := start; index < len(m.searchResults); index++ {
		result := m.searchResults[index]
		needed := 1
		if result.Kind != lastKind {
			needed++
		}
		if result.Snippet != "" {
			needed++
		}
		if used+needed > m.searchResultLineBudget() {
			break
		}
		used += needed
		lastKind = result.Kind
		start = index + 1
	}
	return start
}

func (m *model) searchModeLabel() string {
	if m.searchMode == searchNormalMode {
		return "NORMAL"
	}
	return "INSERT"
}

func (m *model) renderSearchOverlay(background string, width int) string {
	// Background is the actual Pane area to be overlaid, so it is the single
	// source of truth for both modal geometry and fallback replacement.
	viewportHeight := len(strings.Split(background, "\n"))
	layout := modalBounds(width, viewportHeight, searchModalWidth(width), searchModalHeight(viewportHeight))
	contentWidth := layout.contentWidth
	query := fmt.Sprintf("Search: %s", m.searchInput)
	if m.searchMode == searchInsertMode && !m.searchCommandMode {
		query = "Search: " + insertCursorText(m.searchInput, m.searchInputCursor)
	}
	body := []string{
		lipgloss.NewStyle().Bold(true).Render("GLOBAL SEARCH · " + m.searchModeLabel()),
		query,
		"",
	}
	if m.searchCommandMode {
		body = append(body, ":"+insertCursorText(m.searchCommandInput, m.searchCommandCursor), "")
	}
	switch {
	case strings.TrimSpace(m.searchInput) == "":
		body = append(body, "Type to search Projects, Tasks, Labels, and Descriptions.")
	case m.searchLoading:
		body = append(body, "Searching…")
	case len(m.searchResults) == 0:
		body = append(body, "No results.")
	default:
		resultsWidth := contentWidth
		overflows := m.searchResultEnd(0) < len(m.searchResults)
		if overflows {
			resultsWidth -= 2
		}
		resultStart := len(body)
		body = append(body, m.renderSearchResults(resultsWidth)...)
		if overflows {
			scrollTop, contentRows := m.searchResultScrollbarMetrics()
			body = strings.Split(renderVerticalScrollbar(body, contentWidth, len(body), scrollTop, contentRows, resultStart, len(body)-resultStart, true), "\n")
		}
	}
	modal := renderModalFrame(body, layout)
	return overlayModalLayout(background, modal, width, layout.fallback)
}

// searchResultScrollbarMetrics measures physical rendered rows, not result
// records: group headings and description snippets each consume a row.
func (m *model) searchResultScrollbarMetrics() (scrollTop, contentRows int) {
	for index, result := range m.searchResults {
		if index == 0 || result.Kind != m.searchResults[index-1].Kind {
			contentRows++
			if index < m.searchViewport {
				scrollTop++
			}
		}
		contentRows++
		if index < m.searchViewport {
			scrollTop++
		}
		if result.Snippet != "" {
			contentRows++
			if index < m.searchViewport {
				scrollTop++
			}
		}
	}
	return scrollTop, contentRows
}

func (m *model) renderSearchResults(width int) []string {
	start := clamp(m.searchViewport, 0, max(0, len(m.searchResults)-1))
	end := m.searchResultEnd(start)
	var lines []string
	lastKind := vimkunja.SearchResultKind(-1)
	for index := start; index < end; index++ {
		result := m.searchResults[index]
		if result.Kind != lastKind {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render(searchGroupTitle(result.Kind)))
			lastKind = result.Kind
		}
		prefix := "  "
		if index == m.searchSelected {
			prefix = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render(">") + " "
		}
		doneMarker := ""
		if result.Done {
			doneMarker = "[DONE]"
		}
		contentWidth := max(1, width-lipgloss.Width(prefix)-lipgloss.Width(doneMarker))
		if doneMarker != "" {
			contentWidth = max(1, contentWidth-1) // reserve a separating space
		}
		primary := truncateSearchHighlights(result.Primary, result.Highlights, contentWidth)
		content := primary
		if result.Secondary != "" {
			available := max(0, contentWidth-lipgloss.Width(content)-2)
			if available > 0 {
				content += "  " + truncateSearchHighlights(result.Secondary, result.SecondaryHighlights, available)
			}
		}
		if doneMarker != "" {
			content += strings.Repeat(" ", max(1, width-lipgloss.Width(prefix)-lipgloss.Width(content)-lipgloss.Width(doneMarker))) + doneMarker
			content = completedTaskStyle().Render(ansi.Strip(content))
		}
		lines = append(lines, prefix+content)
		if result.Snippet != "" {
			snippet := "    " + truncateSearchHighlights(result.Snippet, result.SnippetHighlights, max(1, width-4))
			if result.Done {
				snippet = completedTaskStyle().Render(ansi.Strip(snippet))
			}
			lines = append(lines, snippet)
		}
	}
	return lines
}

func searchGroupTitle(kind vimkunja.SearchResultKind) string {
	switch kind {
	case vimkunja.SearchTaskResult:
		return "TASKS"
	case vimkunja.SearchLabelResult:
		return "LABELS"
	case vimkunja.SearchDescriptionResult:
		return "DESCRIPTIONS"
	default:
		return "PROJECTS"
	}
}

func truncateSearchHighlights(text string, highlights []vimkunja.SearchHighlight, width int) string {
	text = truncatePlain(text, width)
	if len(highlights) == 0 {
		return text
	}
	runes := []rune(text)
	sorted := append([]vimkunja.SearchHighlight(nil), highlights...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })
	var out strings.Builder
	position := 0
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.inactiveSelection))
	for _, highlight := range sorted {
		start := clamp(highlight.Start, 0, len(runes))
		end := clamp(highlight.End, start, len(runes))
		if end <= position {
			continue
		}
		if start < position {
			start = position
		}
		out.WriteString(string(runes[position:start]))
		out.WriteString(style.Render(string(runes[start:end])))
		position = end
	}
	out.WriteString(string(runes[position:]))
	return out.String()
}

func overlayModal(background, modal string, width int) string {
	modalLines := strings.Split(modal, "\n")
	backgroundLines := strings.Split(background, "\n")
	modalWidth := lipgloss.Width(modalLines[0])
	left := max(0, (width-modalWidth)/2)
	top := max(0, (len(backgroundLines)-len(modalLines))/2)
	for i, modalLine := range modalLines {
		row := top + i
		if row >= len(backgroundLines) {
			break
		}
		backgroundLine := backgroundLines[row]
		if padding := width - lipgloss.Width(backgroundLine); padding > 0 {
			backgroundLine += strings.Repeat(" ", padding)
		}
		backgroundLines[row] = ansi.Cut(backgroundLine, 0, left) + modalLine + ansi.Cut(backgroundLine, left+modalWidth, width)
	}
	return strings.Join(backgroundLines, "\n")
}
