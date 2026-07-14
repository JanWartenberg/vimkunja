package tui

import (
	"fmt"
	"strings"

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
	registry.Register("q", nil, quitCommand)
	registry.Register("qa", []string{"quitall"}, quitAllCommand)
	return registry
}

func quitCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.showDetail && m.focus == paneDetail {
		m.showDetail = false
		m.focus = paneTasks
		return m, nil
	}
	return m, tea.Quit
}

func quitAllCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func normalizeCommandName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
