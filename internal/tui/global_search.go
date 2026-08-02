package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/vimkunja/vimkunja/internal/vimkunja"
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
	m.searchInput = ""
	m.searchResults = nil
	m.searchSelected = 0
	m.searchViewport = 0
	m.searchLoading = false
}

func (m *model) updateSearchKeys(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(key.String()) {
	case "enter":
		return m, m.activateSelectedSearchResult()
	case "esc":
		m.closeSearch()
		return m, nil
	case "backspace":
		runes := []rune(m.searchInput)
		if len(runes) > 0 {
			m.searchInput = string(runes[:len(runes)-1])
		}
		return m, m.beginSearchQuery()
	case "up", "ctrl+p":
		m.moveSearchSelection(-1)
		return m, nil
	case "down", "ctrl+n":
		m.moveSearchSelection(1)
		return m, nil
	}
	if text := key.Key().Text; text != "" {
		m.searchInput += text
		return m, m.beginSearchQuery()
	}
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
			results:  append([]vimkunja.SearchResult(nil), m.searchResults...),
			selected: m.searchSelected,
			viewport: m.searchViewport,
		}
	}
	m.searchOrigin = nil
	m.searchOpen = false
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
	m.restoreSearchOrigin(snapshot.origin)
	origin := snapshot.origin
	m.searchOrigin = &origin
	m.searchOpen, m.searchInput = true, snapshot.input
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
	if m.searchSelected >= m.searchViewport+m.searchVisibleResultRows() {
		m.searchViewport = m.searchSelected - m.searchVisibleResultRows() + 1
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

func (m *model) searchVisibleResultRows() int {
	if m.height <= 0 {
		return 8
	}
	// Header, input, separator, borders, and padding consume roughly seven
	// rows of the modal's outer height.
	return max(8, searchModalHeight(m.height)-7)
}

func (m *model) renderSearchOverlay(background string, width int) string {
	modalWidth := searchModalWidth(width)
	modalHeight := searchModalHeight(len(strings.Split(background, "\n")))
	contentWidth := max(1, modalWidth-6)
	body := []string{
		lipgloss.NewStyle().Bold(true).Render("GLOBAL SEARCH"),
		fmt.Sprintf("Search: %s", m.searchInput),
		"",
	}
	switch {
	case strings.TrimSpace(m.searchInput) == "":
		body = append(body, "Type to search Projects, Tasks, Labels, and Descriptions.")
	case m.searchLoading:
		body = append(body, "Searching…")
	case len(m.searchResults) == 0:
		body = append(body, "No results.")
	default:
		body = append(body, m.renderSearchResults(contentWidth)...)
	}
	modal := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("255")).
		Padding(1, 2).
		Width(modalWidth).
		Height(max(1, modalHeight-4)).
		Render(strings.Join(body, "\n"))
	return overlayModal(background, modal, width)
}

func (m *model) renderSearchResults(width int) []string {
	start := clamp(m.searchViewport, 0, max(0, len(m.searchResults)-1))
	end := min(len(m.searchResults), start+m.searchVisibleResultRows())
	var lines []string
	lastKind := vimkunja.SearchResultKind(-1)
	for index := start; index < end; index++ {
		result := m.searchResults[index]
		if result.Kind != lastKind {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).Render(searchGroupTitle(result.Kind)))
			lastKind = result.Kind
		}
		prefix := "  "
		if index == m.searchSelected {
			prefix = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(selectionAccent)).Render(">") + " "
		}
		primary := truncateSearchHighlights(result.Primary, result.Highlights, max(1, width-lipgloss.Width(prefix)))
		line := prefix + primary
		if result.Secondary != "" {
			available := max(0, width-lipgloss.Width(line)-2)
			if available > 0 {
				line += "  " + truncateSearchHighlights(result.Secondary, result.SecondaryHighlights, available)
			}
		}
		lines = append(lines, line)
		if result.Snippet != "" {
			lines = append(lines, "    "+truncateSearchHighlights(result.Snippet, result.SnippetHighlights, max(1, width-4)))
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
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
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
