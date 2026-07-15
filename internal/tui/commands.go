package tui

import (
	"fmt"
	"strings"

	apperrors "github.com/vimkunja/vimkunja/internal/errors"

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
	registry.Register("new", nil, newTaskCommand)
	registry.Register("w", []string{"write"}, writeCommand)
	registry.Register("wq", nil, writeQuitCommand)
	registry.Register("h", []string{"help"}, helpCommand)
	registry.Register("err", []string{"errors"}, errorsCommand)
	return registry
}

func quitCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.showDetail && m.focus == paneDetail {
		if m.editor != nil && m.editor.dirty() {
			m.recordError(fmt.Errorf("unsaved changes; use :w or :q!"))
			return m, nil
		}
		m.editor = nil
		m.showDetail = false
		m.focus = paneTasks
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
		m.editor = nil
		m.showDetail = false
		m.focus = paneTasks
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

func quitAllCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func helpCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	m.showDetail = false
	m.infoPane = nil
	m.helpPane = &infoPane{
		title: "HELP",
		lines: []string{
			"Welcome to Vimkunja, a vim-motion based Ticket-tool.",
			"",
			"Help content will be extended in Phase 5.",
			"Use :q to close this pane.",
		},
	}
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
