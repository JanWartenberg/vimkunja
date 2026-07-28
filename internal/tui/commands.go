package tui

import (
	"errors"
	"fmt"
	"strings"

	apperrors "github.com/vimkunja/vimkunja/internal/errors"
	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

// CommandHandler handles one Ex command. Args excludes the command name.
type CommandHandler func(m *model, args []string) (tea.Model, tea.Cmd)

// CommandRegistry maps Ex command names and aliases to their handlers.
type CommandRegistry struct {
	handlers map[string]CommandHandler
}

// NewCommandRegistry creates an empty command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{handlers: make(map[string]CommandHandler)}
}

// Register adds name and aliases for handler. Command names are case-insensitive.
func (r *CommandRegistry) Register(name string, aliases []string, handler CommandHandler) {
	if r.handlers == nil {
		r.handlers = make(map[string]CommandHandler)
	}

	r.handlers[normalizeCommandName(name)] = handler
	for _, alias := range aliases {
		r.handlers[normalizeCommandName(alias)] = handler
	}
}

// Execute parses input and dispatches it to the registered command handler.
// input may include the leading colon shown by the command prompt.
func (r *CommandRegistry) Execute(m *model, input string) (tea.Model, tea.Cmd, error) {
	fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(input), ":"))
	if len(fields) == 0 {
		return m, nil, fmt.Errorf("empty command")
	}

	name := normalizeCommandName(fields[0])
	handler, ok := r.handlers[name]
	if !ok {
		return m, nil, fmt.Errorf("unknown command: %s", fields[0])
	}

	updated, cmd := handler(m, fields[1:])
	return updated, cmd, nil
}

func newDefaultCommandRegistry() *CommandRegistry {
	registry := NewCommandRegistry()
	registry.Register("q", []string{"quit"}, quitCommand)
	registry.Register("q!", []string{"quit!"}, forceQuitCommand)
	registry.Register("qa", []string{"quitall"}, quitAllCommand)
	registry.Register("qa!", []string{"quitall!"}, quitAllCommand)
	registry.Register("new", nil, newTaskCommand)
	registry.Register("w", []string{"write"}, writeCommand)
	registry.Register("wq", nil, writeQuitCommand)
	registry.Register("wqa", nil, writeQuitAllCommand)
	registry.Register("h", []string{"help"}, helpCommand)
	registry.Register("err", []string{"errors"}, errorsCommand)
	registry.Register("overview", nil, overviewCommand)
	registry.Register("list", nil, listCommand)
	registry.Register("kanban", nil, kanbanCommand)
	registry.Register("table", nil, notYetImplementedCommand)
	registry.Register("gantt", nil, notYetImplementedCommand)
	return registry
}

func quitCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.showDetail && m.focus == paneDetail {
		if m.editor != nil && m.editor.dirty() {
			m.recordError(fmt.Errorf("unsaved changes; use :w or :q!"))
			return m, nil
		}
		m.closeTaskDetail()
		return m, nil
	}
	if m.helpPane != nil && m.focus == paneHelp {
		m.helpPane = nil
		m.focus = paneTasks
		return m, nil
	}
	if m.infoPane != nil && m.focus == paneInfo {
		m.infoPane = nil
		m.focus = paneTasks
		return m, nil
	}
	return m, tea.Quit
}

func forceQuitCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.showDetail && m.focus == paneDetail {
		m.closeTaskDetail()
		return m, nil
	}
	return m, tea.Quit
}

func newTaskCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	m.openNewEditor()
	return m, nil
}

func writeCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.showDetail && m.focus == paneDetail && m.editor != nil {
		return m, m.saveEditor(false)
	}
	m.recordError(fmt.Errorf(":write requires a task detail pane"))
	return m, nil
}

func writeQuitCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.showDetail && m.focus == paneDetail && m.editor != nil {
		return m, m.saveEditor(true)
	}
	m.recordError(fmt.Errorf(":wq requires a task detail pane"))
	return m, nil
}

func writeQuitAllCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.showDetail && m.focus == paneDetail && m.editor != nil {
		cmd := m.saveEditor(true)
		if cmd != nil {
			m.quitAfterSave = true
		}
		return m, cmd
	}
	return m, tea.Quit
}

func quitAllCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func helpCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	m.showDetail = false
	m.infoPane = nil
	m.helpPane = &infoPane{title: "HELP", lines: helpLines}
	m.focus = paneHelp
	return m, nil
}

func errorsCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	m.showDetail = false
	m.helpPane = nil
	m.infoPane = &infoPane{title: "ERRORS", lines: errorLogLines(), errorLog: true}
	m.focus = paneInfo
	return m, nil
}

func overviewCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	return m, m.openOverview()
}

func kanbanCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.currentProjectID == 0 {
		m.recordError(fmt.Errorf(":kanban requires a Project context"))
		return m, nil
	}
	if m.inOverview() {
		m.taskScope = scopeProject
		m.overviewPrompt = overviewPromptNone
		m.showDetail = false
		m.editor = nil
		m.projectsPaneCollapsed = m.lastProjectPaneCollapsed
		m.focus = paneTasks
	}
	m.taskLoading = true
	return m, m.loadKanbanBoard(m.currentProjectID)
}

func notYetImplementedCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	m.recordError(errors.New("Not yet implemented"))
	return m, nil
}

func listCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.currentProjectID == 0 {
		m.recordError(fmt.Errorf(":list requires a Project context"))
		return m, nil
	}
	m.viewSelected = m.pickListViewIndex()
	if m.viewSelected < 0 {
		m.recordError(fmt.Errorf("no list View for the current Project"))
		return m, nil
	}
	m.currentViewID = m.projectViews[m.viewSelected].ID
	m.kanbanActive = false
	m.kanbanBoard = vimkunja.KanbanBoard{}
	if m.inOverview() {
		return m, m.restoreProjectScope()
	}
	return m, m.refreshActiveTasks()
}

func errorLogLines() []string {
	lines := apperrors.GetAll()
	if len(lines) == 0 {
		return []string{"No errors recorded in this session."}
	}
	return lines
}

func normalizeCommandName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
