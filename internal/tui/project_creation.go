package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

type projectCreationUI struct {
	title                           string
	cursor                          int
	parentID                        int64
	parents                         []vimkunja.ProjectParent
	field                           int // Title, Parent.
	picker, searching, pickerGArmed bool
	mode                            editorMode
	normal                          textNormalState
	initialParentID                 int64
	commandMode, discardConfirm     bool
	command                         string
	commandCursor                   int
	query                           string
	queryCursor, selected           int
	loading, saving, requireParent  bool
	err                             string
	helpOpen                        bool
	helpScroll                      int
	helpG                           bool
	discardScroll                   int
	discardG                        bool
}

type projectParentsLoadedMsg struct {
	ui      *projectCreationUI
	parents []vimkunja.ProjectParent
	err     error
}

func projectNewCommand(m *model, args []string) (tea.Model, tea.Cmd) {
	return m, m.openProjectCreation(false, args)
}
func projectNewChildCommand(m *model, args []string) (tea.Model, tea.Cmd) {
	return m, m.openProjectCreation(true, args)
}

func (m *model) openProjectCreation(child bool, args []string) tea.Cmd {
	if len(args) != 0 {
		m.recordError(fmt.Errorf("Project creation commands take no arguments"))
		return nil
	}
	if m.projectCreation != nil {
		return nil
	}
	if m.savingEditor != nil || (m.editor != nil && m.editor.dirty()) || m.helpHasDraft() || (m.inlineTitle != nil && (m.inlineTitle.saving || m.inlineTitle.title != m.inlineTitle.task.Title)) {
		m.recordError(fmt.Errorf("save or discard the unsaved Task before creating a Project"))
		return nil
	}
	ui := &projectCreationUI{loading: true, requireParent: child}
	if child {
		p, ok := m.selectedProject()
		if !ok {
			m.recordError(fmt.Errorf("select a Project before using :projectnewchild"))
			return nil
		}
		ui.parentID = p.ID
	}
	ui.initialParentID = ui.parentID
	m.projectCreation = ui
	m.leaderArmed, m.ctrlWArmed, m.gArmed, m.zArmed, m.viewSwitchArmed = false, false, false, false, false
	return m.loadProjectCreationParents(ui)
}

func (m *model) loadProjectCreationParents(ui *projectCreationUI) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		parents, err := store.ProjectCreationParents(context.Background())
		return projectParentsLoadedMsg{ui, parents, err}
	}
}

func (ui *projectCreationUI) parentLabel() string {
	if ui.parentID == 0 {
		return "No parent Project"
	}
	for _, p := range ui.parents {
		if p.ID == ui.parentID {
			return fmt.Sprintf("%s (#%d)", sanitizeTerminalText(p.Path), p.ID)
		}
	}
	return fmt.Sprintf("Unavailable Parent (#%d)", ui.parentID)
}

func (ui *projectCreationUI) matches() []vimkunja.ProjectParent {
	out := []vimkunja.ProjectParent{{ID: 0, Path: "No parent Project"}}
	for _, p := range ui.parents {
		if ui.query == "" || strings.Contains(strings.ToLower(fmt.Sprintf("%s #%d", sanitizeTerminalText(p.Path), p.ID)), strings.ToLower(ui.query)) {
			out = append(out, p)
		}
	}
	return out
}

func (m *model) updateProjectCreation(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	ui := m.projectCreation
	if ui.saving {
		return m, nil
	}
	k := key.String()
	ui.err = ""
	if ui.discardConfirm {
		_, layout, limit := m.projectDiscardBody()
		if next, handled := navigateManagement(k, &ui.discardG, ui.discardScroll, limit, layout.contentHeight-4); handled {
			ui.discardScroll = next
			return m, nil
		}
		switch k {
		case "y":
			w, h := m.viewportSize()
			if confirmationUsable(w, h) {
				m.projectCreation = nil
			}
		case "n", "esc":
			ui.discardConfirm = false
		}
		return m, nil
	}
	if ui.commandMode {
		return m, m.updateProjectCreationCommand(key)
	}
	if ui.helpOpen {
		switch k {
		case "q", "esc":
			ui.helpOpen = false
		case ":":
			ui.commandMode, ui.command, ui.commandCursor = true, "", 0
		default:
			_, rows, visible := m.projectHelpViewport()
			ui.helpScroll, _ = navigateManagement(k, &ui.helpG, ui.helpScroll, max(0, len(rows)-max(1, visible)), visible)
		}
		return m, nil
	}
	if !ui.picker && ui.mode == editorInsert {
		if k == "ctrl+v" {
			cmd, _ := m.textClipboardKey("p")
			return m, cmd
		}
		if k == "esc" || k == "enter" {
			ui.mode = editorNormal
		} else {
			updateSingleLineTextInput(&ui.title, &ui.cursor, key)
		}
		return m, nil
	}
	if k == "esc" {
		ui.pickerGArmed = false
		m.leaderArmed = false
		if ui.picker && ui.searching {
			ui.searching = false
			return m, nil
		}
		if ui.picker {
			ui.picker = false
			return m, nil
		}
		ui.normal.reset()
		return m, nil
	}
	if ui.picker {
		matches := ui.matches()
		if k == "ctrl+d" || k == "ctrl+u" || k == "pgdown" || k == "pagedown" || k == "pgup" || k == "pageup" {
			step := max(1, m.projectParentPickerRows()/2)
			if k == "ctrl+u" || k == "pgup" || k == "pageup" {
				step = -step
			}
			ui.selected = clamp(ui.selected+step, 0, len(matches)-1)
			ui.pickerGArmed = false
			return m, nil
		}
		if ui.searching {
			if k == "enter" {
				ui.searching = false
			} else if _, changed := updateSingleLineTextInput(&ui.query, &ui.queryCursor, key); changed {
				ui.selected = 0
			}
			return m, nil
		}
		if ui.pickerGArmed {
			ui.pickerGArmed = false
			if k == "g" {
				ui.selected = 0
				return m, nil
			}
		}
		switch k {
		case "g":
			ui.pickerGArmed = true
		case "G", "end":
			ui.selected = len(matches) - 1
		case "home":
			ui.selected = 0
		case "enter":
			ui.parentID = matches[clamp(ui.selected, 0, len(matches)-1)].ID
			ui.picker = false
		case "j", "down", "ctrl+n":
			ui.selected = clamp(ui.selected+1, 0, len(matches)-1)
		case "k", "up", "ctrl+p":
			ui.selected = clamp(ui.selected-1, 0, len(matches)-1)
		case "/":
			ui.searching = true
		case ":":
			ui.commandMode, ui.command, ui.commandCursor = true, "", 0
		}
		return m, nil
	}
	if m.leaderArmed {
		m.leaderArmed = false
		cmd, _ := m.textClipboardKey(k)
		return m, cmd
	}
	if isSpaceKey(key) && ui.field == 0 {
		m.leaderArmed = true
		return m, tea.Tick(m.leaderTimeout, func(time.Time) tea.Msg { return leaderTimeoutMsg{} })
	}
	if k == ":" {
		ui.commandMode, ui.command, ui.commandCursor = true, "", 0
		return m, nil
	}
	if ui.field == 0 {
		motion := k
		switch k {
		case "left":
			motion = "h"
		case "right":
			motion = "l"
		case "home":
			motion = "0"
		case "end":
			motion = "$"
		}
		if result := applyTextNormalKey(&ui.normal, ui.title, ui.cursor, motion, false); result.handled {
			ui.title, ui.cursor = result.text, result.cursor
			if result.enterInsert {
				ui.mode = editorInsert
				return m, cursorBlinkCmd()
			}
			return m, nil
		}
	}
	switch k {
	case "tab", "shift+tab":
		ui.field = (ui.field + 1) % 2
		ui.normal.reset()
	case "j", "down":
		ui.field = 1
		ui.normal.reset()
	case "k", "up":
		ui.field = 0
		ui.normal.reset()
	case "i", "a", "I", "A", "enter":
		if ui.field == 1 {
			if !ui.loading {
				ui.picker, ui.searching, ui.pickerGArmed = true, false, false
				ui.query, ui.queryCursor, ui.selected = "", 0, 0
				for i, p := range ui.matches() {
					if p.ID == ui.parentID {
						ui.selected = i
						break
					}
				}
			}
		} else if k != "enter" {
			switch k {
			case "a":
				moveTextCursor(&ui.cursor, ui.title, 1)
			case "I":
				ui.cursor = 0
			case "A":
				ui.cursor = len([]rune(ui.title))
			}
			ui.mode = editorInsert
			return m, cursorBlinkCmd()
		}
	}
	return m, nil
}

func (m *model) updateProjectCreationCommand(key tea.KeyMsg) tea.Cmd {
	ui := m.projectCreation
	switch key.String() {
	case "esc":
		ui.commandMode = false
	case "enter":
		command, err := m.resolveManagementCommand(ui.command)
		ui.commandMode = false
		if err != nil {
			ui.err = err.Error()
			return nil
		}
		if command == "" {
			return nil
		}
		if ui.helpOpen {
			if command == "quit" || command == "quit!" {
				ui.helpOpen = false
			} else if command != "help" {
				ui.err = "Command unavailable in help; :q returns to the draft"
			}
			return nil
		}
		switch command {
		case "help":
			ui.helpOpen, ui.helpScroll, ui.helpG = true, 0, false
			ui.err = ""
		case "write", "writequit":
			return m.saveNewProject()
		case "quit!":
			m.projectCreation = nil
		case "quit":
			if ui.title != "" || ui.parentID != ui.initialParentID {
				ui.discardConfirm = true
				ui.discardScroll, ui.discardG = 0, false
			} else {
				m.projectCreation = nil
			}
		default:
			ui.err = "Command unavailable here; :h explains Project commands"
		}
	default:
		updateSingleLineTextInput(&ui.command, &ui.commandCursor, key)
	}
	return nil
}

type projectCreatedMsg struct {
	ui              *projectCreationUI
	project         vimkunja.Project
	projects        []vimkunja.Project
	err, refreshErr error
}

func (m *model) saveNewProject() tea.Cmd {
	ui := m.projectCreation
	if ui.loading || ui.saving {
		return nil
	}
	ui.saving = true
	ui.err = ""
	draft := vimkunja.ProjectDraft{Title: ui.title, ParentProjectID: ui.parentID}
	store := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		p, err := store.CreateProject(ctx, draft)
		msg := projectCreatedMsg{ui: ui, project: p, err: err}
		if err == nil {
			msg.projects, msg.refreshErr = store.ListProjects(ctx)
		}
		return msg
	}
}

func (m *model) projectCreationResult(msg projectCreatedMsg) tea.Cmd {
	ui := m.projectCreation
	if ui != msg.ui {
		return nil
	}
	ui.saving = false
	if msg.err != nil {
		ui.err = msg.err.Error()
		var field *vimkunja.ProjectCreationError
		if errors.As(msg.err, &field) {
			if field.Field == "title" {
				ui.field = 0
			} else {
				ui.field = 1
			}
		}
		// Rebuild choices after stale-parent failures, without silently changing the
		// draft's Parent or retrying a potentially successful creation.
		if field != nil && field.Field == "parent" {
			ui.loading = true
			return m.loadProjectCreationParents(ui)
		}
		return nil
	}
	m.projectCreation = nil
	if msg.refreshErr != nil {
		m.projects = append(m.projects, msg.project)
		m.recordError(fmt.Errorf("Project created, but refreshing failed: %w; reload with r", msg.refreshErr))
	} else {
		m.projects = msg.projects
	}
	m.projectRequest++
	m.projectLoading = false
	m.taskRequest++
	m.viewRequest++
	m.searchRequest++
	m.commentRequest++
	m.projectTree = newProjectTree(m.projects, m.projectTree.collapsedState())
	m.projectTree.reveal(msg.project.ID)
	m.projectSelected = m.findVisibleProjectIndex(msg.project.ID)
	m.currentProjectID = msg.project.ID
	m.currentViewID = 0
	m.initialViewID = 0
	m.taskScope = scopeProject
	m.projectViewPreference = projectViewList
	m.projectsPaneCollapsed = false
	m.projectsPaneAutoShown = false
	m.kanbanActive = false
	m.kanbanBoard = vimkunja.KanbanBoard{}
	m.kanbanPicker = false
	m.kanbanViewPicker = false
	m.kanbanProvisionConfirm = false
	m.searchOpen = false
	m.searchOrigin = nil
	m.searchReturn = nil
	m.closeHelp()
	m.infoPane = nil
	m.editor = nil
	m.showDetail = false
	m.inlineTitle = nil
	m.comments = nil
	m.labelUI = nil
	m.tasks = nil
	m.projectTasks = nil
	m.projectViews = nil
	m.taskSelected = 0
	m.taskLoading = false
	m.focus = paneTasks
	m.viewLoading = true
	return m.loadViews(msg.project.ID)
}

func (m *model) renderProjectCreation() string {
	ui := m.projectCreation
	w, h := m.viewportSize()
	layout := projectCreationLayout(w, h)
	if ui.discardConfirm {
		if !confirmationUsable(w, h) {
			return fitBlock([]string{"DISCARD NEW PROJECT", "Resize · n/Esc cancel"}, w, h)
		}
		rows, confirmLayout, _ := m.projectDiscardBody()
		layout = confirmLayout
		return overlayModalLayout(fitBlock(nil, w, h), renderModalFrame(rows, layout), w, layout.fallback)
	}
	if ui.helpOpen {
		layout, rows, visible := m.projectHelpViewport()
		ui.helpScroll = clamp(ui.helpScroll, 0, max(0, len(rows)-max(1, visible)))
		helpTitle := "PROJECT HELP"
		if ui.commandMode {
			helpTitle += " · COMMAND"
		}
		body := []string{helpTitle}
		body = append(body, rows[ui.helpScroll:min(len(rows), ui.helpScroll+visible)]...)
		for len(body) < layout.contentHeight-2 {
			body = append(body, "")
		}
		footer := managementHint("q/Esc return · j/k scroll · gg/G jump · Ctrl+D/U page", layout.contentWidth)
		if ui.commandMode {
			footer = ":" + insertCursorText(ui.command, ui.commandCursor) + " "
			if layout.contentWidth == 1 {
				footer = insertCursorText("", 0)
			}
		}
		if layout.contentHeight >= 3 {
			body = append(body, strings.Repeat("─", layout.contentWidth))
		}
		body = append(body, footer)
		if layout.contentHeight == 1 && ui.commandMode {
			body = []string{footer}
		}
		return overlayModalLayout(fitBlock(nil, w, h), renderModalFrame(body, layout), w, layout.fallback)
	}
	// A standalone overlay prevents hidden inputs underneath from supplying a
	// competing terminal cursor, and gives tiny terminals the same modal fallback.
	title := ui.title
	if ui.field == 0 && !ui.saving && !ui.picker && !ui.commandMode {
		if ui.normal.selection != nil {
			lo, hi := ui.normal.selectionRange(title, ui.cursor)
			title = renderSelectionRange(title, 0, lo, hi)
		} else if ui.mode == editorInsert {
			title = insertCursorText(title, ui.cursor)
		} else {
			title = normalCursorText(title, ui.cursor)
		}
	}
	marker := func(field int) string {
		if ui.field == field {
			return "> "
		}
		return "  "
	}
	mode := ui.normal.selectionLabel()
	if ui.mode == editorInsert {
		mode = "INSERT"
	}
	if ui.commandMode {
		mode = "COMMAND"
	}
	hint := ":h help · j/k field · i edit · :w create · :q close"
	rows := []string{"NEW PROJECT · " + mode, marker(0) + "Title: " + title, marker(1) + "Parent: " + ui.parentLabel()}
	if ui.picker {
		query := ui.query
		if ui.searching {
			query = insertCursorText(query, ui.queryCursor)
		}
		pickerTitle := "PARENT PROJECT · NORMAL"
		if ui.searching {
			pickerTitle = "PARENT PROJECT · SEARCH"
		}
		if ui.commandMode {
			pickerTitle = "PARENT PROJECT · COMMAND"
		}
		rows = []string{pickerTitle, "Search: " + query}
		matches := ui.matches()
		ui.selected = clamp(ui.selected, 0, len(matches)-1)
		count := m.projectParentPickerRows()
		start := max(0, ui.selected-count+1)
		for i := start; i < min(len(matches), start+count); i++ {
			prefix := "  "
			if i == ui.selected {
				prefix = "> "
			}
			label := sanitizeTerminalText(matches[i].Path)
			if matches[i].ID != 0 {
				label = fmt.Sprintf("#%d %s", matches[i].ID, label)
			}
			rows = append(rows, prefix+label)
		}
		if ui.searching {
			hint = "Enter finish search · Esc back · Ctrl+U/D page"
		} else {
			hint = "Enter choose · Esc back · j/k select · gg/G jump · Ctrl+U/D page · / search"
		}
	} else {
		if ui.saving {
			rows = append(rows, "Creating Project…")
		} else if ui.loading {
			rows = append(rows, "Loading eligible Parents…")
		}
		hint = ":h help · j/k field · i edit · :w create · :q close"
		if ui.mode == editorInsert {
			hint = "INSERT · Enter / Esc normal"
		}
	}
	command := ""
	if ui.commandMode {
		// Keep one cell after the caret when a long command reaches the edge.
		command = ":" + insertCursorText(ui.command, ui.commandCursor) + " "
		if layout.contentWidth == 1 {
			command = insertCursorText("", 0)
		}
	}
	// Reserve error rows before clipping the form. At short heights, the
	// focused input/action and its failure feedback both take priority over hints.
	var errorRows []string
	if ui.err != "" {
		errorRows = wrapModalWords(sanitizeTerminalText(ui.err), layout.contentWidth)
		limit := max(1, layout.contentHeight/2)
		if len(errorRows) > limit {
			errorRows = errorRows[:limit]
			errorRows[limit-1] = truncatePlain(errorRows[limit-1], max(0, layout.contentWidth-1)) + "…"
		}
	}
	footerRows := min(2, layout.contentHeight)
	if !ui.commandMode && (ui.mode == editorInsert || ui.picker && ui.searching) && layout.contentHeight < 3 {
		footerRows = max(0, layout.contentHeight-1)
	}
	if len(errorRows) > layout.contentHeight-footerRows {
		errorRows = errorRows[:max(0, layout.contentHeight-footerRows)]
	}
	budget := max(0, layout.contentHeight-len(errorRows)-footerRows)
	if len(rows) > budget {
		active := ui.field + 1
		if ui.picker {
			active = 1
		}

		start := max(0, active-budget+1)
		rows = rows[min(start, len(rows)):min(len(rows), start+budget)]
	}
	rows = append(rows, errorRows...)
	for len(rows) < layout.contentHeight-footerRows {
		rows = append(rows, "")
	}
	if footerRows > 1 {
		rows = append(rows, strings.Repeat("─", layout.contentWidth))
	}
	if ui.commandMode {
		rows = append(rows, command)
	} else {
		rows = append(rows, managementHint(hint, layout.contentWidth))
	}

	modal := renderModalFrame(rows, layout)
	return overlayModalLayout(fitBlock(nil, w, h), modal, w, layout.fallback)
}

// Navigation and rendering share the visible choice count, including on resize.
func (m *model) projectParentPickerRows() int {
	w, h := m.viewportSize()
	return max(1, modalBounds(w, h, 72, 17).contentHeight-4)
}

func projectCreationHelpRows(width int) []string {
	return managementReferenceRows([]helpEntry{
		{":h / :help", "Explain Project creation without losing the draft."},
		{"j/k / Tab/Shift+Tab", "Select Title or Parent in Normal mode."},
		{"i/a/I/A", "Edit the Title using the shared Normal/Insert editor."},
		{"Enter / Esc", "Finish Insert input; Enter on Parent opens its picker."},
		{"Parent picker", "j/k, gg/G, Ctrl+D/U navigate; / searches; Enter selects; Esc returns."},
		{"Space y / Space p", "Copy selected Title text / paste from the clipboard."},
		{":w / :write", "Create the Project and open it."},
		{":wq / :writequit", "Create the Project and close this form."},
		{":q / :quit", "Close; confirm before discarding an unsaved draft."},
		{":q! / :quit!", "Discard this draft and close."},
	}, width)
}

func (m *model) projectDiscardBody() ([]string, modalLayout, int) {
	w, h := m.viewportSize()
	layout := modalBounds(w, h, 72, 10)
	rows, limit := confirmationBody("DISCARD NEW PROJECT", "Discard this new Project and all unsaved input?", "y discard · n / Esc keep editing", layout, m.projectCreation.discardScroll)
	return rows, layout, limit
}

// Keep help movement and rendering on the same effective modal row budget.
func projectCreationLayout(w, h int) modalLayout {
	layout := modalBounds(w, h, 72, 17)
	if !layout.fallback {
		layout.bottomInput = true
		layout.contentHeight++
	}
	return layout
}

func (m *model) projectHelpViewport() (modalLayout, []string, int) {
	w, h := m.viewportSize()
	layout := projectCreationLayout(w, h)
	rows := projectCreationHelpRows(layout.contentWidth)
	if m.projectCreation.err != "" {
		rows = append(wrapModalWords("Error: "+sanitizeTerminalText(m.projectCreation.err), layout.contentWidth), rows...)
	}
	return layout, rows, max(0, layout.contentHeight-3)
}
