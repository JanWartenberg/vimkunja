package tui

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	apperrors "github.com/vimkunja/vimkunja/internal/errors"
	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

// CommandHandler handles one Ex command. Args excludes the command name.
type CommandHandler func(m *model, args []string) (tea.Model, tea.Cmd)

// CommandRegistry owns Ex command names, aliases, and dispatch handlers.
type CommandRegistry struct {
	spellings map[string]*registeredCommand
}

type registeredCommand struct {
	canonical string
	aliases   []string
	handler   CommandHandler
}

// NewCommandRegistry creates an empty command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{spellings: make(map[string]*registeredCommand)}
}

// Register adds a canonical name and aliases for a handler. Command spellings are
// case-insensitive and must not collide with a spelling owned by another command.
func (r *CommandRegistry) Register(name string, aliases []string, handler CommandHandler) error {
	if r.spellings == nil {
		r.spellings = make(map[string]*registeredCommand)
	}

	canonical := normalizeCommandName(name)
	if canonical == "" {
		return fmt.Errorf("command name must not be empty")
	}
	spellings := []string{canonical}
	normalizedAliases := make([]string, 0, len(aliases))
	seen := map[string]bool{canonical: true}
	for _, alias := range aliases {
		spelling := normalizeCommandName(alias)
		if spelling == "" {
			return fmt.Errorf("command alias must not be empty")
		}
		if !seen[spelling] {
			spellings = append(spellings, spelling)
			normalizedAliases = append(normalizedAliases, spelling)
			seen[spelling] = true
		}
	}
	for _, spelling := range spellings {
		if owner, exists := r.spellings[spelling]; exists {
			return fmt.Errorf("command spelling %q is already registered for %q", spelling, owner.canonical)
		}
	}

	command := &registeredCommand{canonical: canonical, aliases: normalizedAliases, handler: handler}
	for _, spelling := range spellings {
		r.spellings[spelling] = command
	}
	return nil
}

// Execute parses input and dispatches it to the exact or uniquely prefixed
// registered command. input may include the leading colon shown by the prompt.
func (r *CommandRegistry) Execute(m *model, input string) (tea.Model, tea.Cmd, error) {
	fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(input), ":"))
	if len(fields) == 0 {
		return m, nil, fmt.Errorf("empty command")
	}

	command, candidates := r.resolve(fields[0])
	if command == nil {
		if len(candidates) > 0 {
			return m, nil, fmt.Errorf("ambiguous command: %s (matches: %s)", fields[0], strings.Join(candidates, ", "))
		}
		return m, nil, fmt.Errorf("unknown command: %s", fields[0])
	}

	updated, cmd := command.handler(m, fields[1:])
	return updated, cmd, nil
}

// dispatchModalGlobalCommand handles application-wide commands before modal
// key handlers. Local commands (q, w, wq, etc.) remain owned by each prompt.
// Resolve through the main registry so aliases and unique prefixes cannot drift.
func (m *model) dispatchModalGlobalCommand(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if key.String() != "enter" {
		return m, nil, false
	}
	var active *bool
	var input *string
	switch {
	case m.searchOpen:
		active, input = &m.searchCommandMode, &m.searchCommandInput
	case m.comments != nil:
		if m.comments.saving || m.commentDeleteConfirm {
			return m, nil, false
		}
		active, input = &m.comments.commandMode, &m.comments.commandInput
	case m.labelUI != nil:
		if m.labelUI.loading {
			return m, nil, false
		}
		active, input = &m.labelUI.commandMode, &m.labelUI.commandInput
	default:
		return m, nil, false
	}
	if !*active {
		return m, nil, false
	}
	fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(*input), ":"))
	if len(fields) == 0 {
		return m, nil, false
	}
	command, _ := m.commands.resolve(fields[0])
	if command == nil {
		return m, nil, false
	}
	switch command.canonical {
	case "quitall", "quitall!", "writequitall":
		*active, *input = false, ""
		updated, cmd := command.handler(m, fields[1:])
		return updated, cmd, true
	default:
		return m, nil, false
	}
}

// Complete returns the canonical command for a unique prefix or all canonical
// candidates for an ambiguous prefix. An exact spelling does not take precedence.
func (r *CommandRegistry) Complete(prefix string) (string, []string) {
	command, candidates := r.resolvePrefix(prefix)
	if command != nil {
		return command.canonical, nil
	}
	return "", candidates
}

func (r *CommandRegistry) resolve(name string) (*registeredCommand, []string) {
	if command := r.spellings[normalizeCommandName(name)]; command != nil {
		return command, nil
	}
	return r.resolvePrefix(name)
}

func (r *CommandRegistry) resolvePrefix(prefix string) (*registeredCommand, []string) {
	prefix = normalizeCommandName(prefix)
	matches := make(map[*registeredCommand]bool)
	for spelling, command := range r.spellings {
		if strings.HasPrefix(spelling, prefix) {
			matches[command] = true
		}
	}
	if len(matches) == 1 {
		for command := range matches {
			return command, nil
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}

	candidates := make([]string, 0, len(matches))
	for command := range matches {
		candidates = append(candidates, command.canonical)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return strings.ToLower(candidates[i]) < strings.ToLower(candidates[j])
	})
	return nil, candidates
}

func newDefaultCommandRegistry() *CommandRegistry {
	registry := NewCommandRegistry()
	registrations := []struct {
		name    string
		aliases []string
		handler CommandHandler
	}{
		{"quit", []string{"q"}, quitCommand},
		{"quit!", []string{"q!"}, forceQuitCommand},
		{"quitall", []string{"qa"}, quitAllCommand},
		{"quitall!", []string{"qa!"}, forceQuitAllCommand},
		{"new", nil, newTaskCommand},
		{"write", []string{"w"}, writeCommand},
		{"writequit", []string{"wq"}, writeQuitCommand},
		{"writequitall", []string{"wqa"}, writeQuitAllCommand},
		{"help", []string{"h"}, helpCommand},
		{"errors", []string{"err"}, errorsCommand},
		{"overview", nil, overviewCommand},
		{"list", nil, listCommand},
		{"kanban", nil, kanbanCommand},
		{"search", nil, searchCommand},
		{"labels", nil, labelsCommand},
		{"table", nil, notYetImplementedCommand},
		{"gantt", nil, notYetImplementedCommand},
	}
	for _, registration := range registrations {
		if err := registry.Register(registration.name, registration.aliases, registration.handler); err != nil {
			panic(fmt.Sprintf("register default command: %v", err))
		}
	}
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
		if m.kanbanEmptyDetailOrigin && m.editor != nil && m.editor.id == 0 {
			m.editor = nil
			return m, nil
		}
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
	if m.comments != nil && m.comments.composer != nil {
		cmd := m.saveTaskWithComment(true)
		if cmd != nil {
			m.quitAfterSave = true
		}
		return m, cmd
	}
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
	if (m.editor != nil && m.editor.dirty()) || (m.comments != nil && m.comments.composer != nil) {
		m.recordError(fmt.Errorf("unsaved changes; use :wqa or :qa!"))
		return m, nil
	}
	return m, tea.Quit
}

func forceQuitAllCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
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

func searchCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	m.openSearch()
	return m, nil
}

func labelsCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	return m, m.openGlobalLabelOverview()
}

func kanbanCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	m.projectViewPreference = projectViewKanban
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
	m.projectViewPreference = projectViewList
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
