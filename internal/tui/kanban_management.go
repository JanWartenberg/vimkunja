package tui

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

// The manager owns a separate draft and command prompt. Task editors never
// remain attached while this surface is active.
type boardManagementUI struct {
	persisted                   bool
	created                     bool
	confirmScroll               int
	projectID                   int64
	page                        string
	views                       []vimkunja.ProjectView
	selected, field, cursor     int
	original, draft             vimkunja.KanbanConfiguration
	position                    string
	limits                      map[int64]string
	mode                        editorMode
	normal                      textNormalState
	commandMode                 bool
	command                     string
	commandCursor               int
	loading, saving, closeAfter bool
	request                     uint64
	err                         string
	confirm                     string
	pending                     vimkunja.KanbanConfiguration
	deleteIndex                 int
	helpOpen                    bool
	helpScroll                  int
	menuG                       bool
}

type boardManagementMsg struct {
	ui        *boardManagementUI
	request   uint64
	operation string
	views     []vimkunja.ProjectView
	config    vimkunja.KanbanConfiguration
	err       error
}

func (m *model) boardNavigationBlocked() bool {
	if m.savingEditor != nil || (m.editor != nil && m.editor.dirty()) || m.helpHasDraft() || (m.comments != nil && (m.comments.saving || m.comments.composer != nil)) || (m.inlineTitle != nil && (m.inlineTitle.saving || m.inlineTitle.title != m.inlineTitle.task.Title)) {
		m.recordError(fmt.Errorf("save or discard the unsaved Task or Comment before changing Boards"))
		return true
	}
	return false
}

func (m *model) openBoardManagement(create bool) tea.Cmd {
	if m.currentProjectID == 0 {
		m.recordError(fmt.Errorf("Board management requires a Project"))
		return nil
	}
	if m.boardNavigationBlocked() {
		return nil
	}
	if m.boardManagement != nil {
		return nil
	}
	m.leaveHelpForContext()
	m.showDetail = false
	m.editor = nil
	m.comments = nil
	m.inlineTitle = nil
	m.labelUI = nil
	m.infoPane = nil
	m.projectRequest++
	m.viewRequest++
	m.taskRequest++
	m.searchRequest++
	m.commentRequest++
	m.projectLoading = false
	m.viewLoading = false
	m.taskLoading = false
	m.leaderArmed = false
	m.viewSwitchArmed = false
	m.kanbanViewPicker = false
	m.kanbanProvisionConfirm = false
	m.ctrlWArmed = false
	m.commandMode = false
	m.focus = paneTasks
	ui := &boardManagementUI{projectID: m.currentProjectID, page: "manager"}
	m.boardManagement = ui
	if create {
		ui.page = "create"
		ui.draft.ProjectID = ui.projectID
		return nil
	}
	return m.loadBoardManager(ui)
}

func (m *model) loadBoardManager(ui *boardManagementUI) tea.Cmd {
	ui.loading = true
	ui.request++
	request := ui.request
	store := m.store
	projectID := ui.projectID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		views, err := loadAllProjectViews(ctx, store, projectID)
		if err != nil {
			return boardManagementMsg{ui: ui, request: request, operation: "list", err: err}
		}
		return boardManagementMsg{ui: ui, request: request, operation: "list", views: views}
	}
}
func (m *model) loadBoardConfiguration(ui *boardManagementUI, operation string) tea.Cmd {
	if len(ui.views) == 0 {
		return nil
	}
	view := ui.views[clamp(ui.selected, 0, len(ui.views)-1)]
	if view.Filtered || view.BucketConfigurationMode != "manual" {
		ui.err = "Filtered Boards are preserved; manage them in VOA"
		return nil
	}
	ui.loading = true
	ui.request++
	request := ui.request
	store := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		cfg, err := store.LoadKanbanConfiguration(ctx, ui.projectID, view.ID)
		return boardManagementMsg{ui: ui, request: request, operation: operation, config: cfg, err: err}
	}
}
func cloneKanbanConfiguration(cfg vimkunja.KanbanConfiguration) vimkunja.KanbanConfiguration {
	cfg.Buckets = append([]vimkunja.KanbanBucketConfiguration(nil), cfg.Buckets...)
	return cfg
}
func (ui *boardManagementUI) adopt(cfg vimkunja.KanbanConfiguration) {
	ui.original = cloneKanbanConfiguration(cfg)
	ui.draft = cloneKanbanConfiguration(cfg)
	ui.position = strconv.FormatFloat(cfg.Position, 'f', -1, 64)
	ui.limits = map[int64]string{}
	for _, b := range cfg.Buckets {
		ui.limits[b.ID] = strconv.FormatInt(b.Limit, 10)
	}
	ui.field = clamp(ui.field, 0, 1+len(cfg.Buckets)*2)
	ui.cursor = 0
	ui.normal.reset()
	ui.mode = editorNormal
	ui.menuG = false
}
func (ui *boardManagementUI) fieldValue() string {
	if ui.field == 0 {
		return ui.draft.Title
	}
	if ui.field == 1 {
		return ui.position
	}
	i := (ui.field - 2) / 2
	if i < 0 || i >= len(ui.draft.Buckets) {
		return ""
	}
	b := ui.draft.Buckets[i]
	if ui.field%2 == 0 {
		return b.Title
	}
	return ui.limits[b.ID]
}
func (ui *boardManagementUI) setFieldValue(value string) {
	switch ui.field {
	case 0:
		ui.draft.Title = value
	case 1:
		ui.position = value
	default:
		i := (ui.field - 2) / 2
		if i >= 0 && i < len(ui.draft.Buckets) {
			if ui.field%2 == 0 {
				ui.draft.Buckets[i].Title = value
			} else {
				ui.limits[ui.draft.Buckets[i].ID] = value
			}
		}
	}
}
func (ui *boardManagementUI) dirty() bool {
	if ui.page == "create" {
		return ui.draft.Title != ""
	}
	if ui.page != "config" {
		return false
	}
	if !reflect.DeepEqual(ui.draft, ui.original) || ui.position != strconv.FormatFloat(ui.original.Position, 'f', -1, 64) {
		return true
	}
	for _, b := range ui.original.Buckets {
		if ui.limits[b.ID] != strconv.FormatInt(b.Limit, 10) {
			return true
		}
	}
	return false
}
func (m *model) applyBoardManagement(msg boardManagementMsg) tea.Cmd {
	ui := m.boardManagement
	if ui != msg.ui || ui.request != msg.request {
		return nil
	}
	ui.loading = false
	ui.saving = false
	if msg.err != nil {
		ui.err = msg.err.Error()
		return nil
	}
	ui.err = ""
	switch msg.operation {
	case "list":
		ui.views = nil
		for _, v := range msg.views {
			if v.ViewKind == "kanban" {
				ui.views = append(ui.views, v)
			}
		}
		ui.selected = clamp(ui.selected, 0, max(0, len(ui.views)-1))
		m.projectViews = msg.views
	case "edit":
		ui.page = "config"
		ui.adopt(msg.config)
	case "preview":
		ui.pending = msg.config
		ui.confirm = "board"
		ui.confirmScroll = 0
		ui.menuG = false
	case "save":
		ui.adopt(msg.config)
		ui.persisted = true
		m.undoHistory = nil
		m.redoHistory = nil
		if ui.closeAfter {
			return m.closeBoardManagement(true, msg.config.ViewID)
		}
	case "create":
		ui.created = true
		m.undoHistory = nil
		m.redoHistory = nil
		ui.page = "config"
		ui.adopt(msg.config)
		ui.persisted = true
		if len(ui.draft.Buckets) > 0 {
			ui.field = 2
		}
		ui.views = append(ui.views, vimkunja.ProjectView{ID: msg.config.ViewID, Title: msg.config.Title, ViewKind: "kanban", BucketConfigurationMode: "manual"})
		if m.lastKanbanViews == nil {
			m.lastKanbanViews = map[int64]int64{}
		}
		m.lastKanbanViews[ui.projectID] = msg.config.ViewID
		return nil
	case "delete":
		if m.lastKanbanViews[ui.projectID] == ui.pending.ViewID {
			delete(m.lastKanbanViews, ui.projectID)
		}
		m.undoHistory = nil
		m.redoHistory = nil
		return m.closeBoardManagement(true, 0)
	}
	return nil
}
func (m *model) closeBoardManagement(refresh bool, viewID int64) tea.Cmd {
	ui := m.boardManagement
	if ui == nil {
		return nil
	}
	m.boardManagement = nil
	m.showDetail = false
	m.focus = paneTasks
	if refresh || ui.persisted {
		m.taskScope = scopeProject
		m.overviewPrompt = overviewPromptNone
		m.currentProjectID = ui.projectID
		if viewID != 0 {
			if m.lastKanbanViews == nil {
				m.lastKanbanViews = map[int64]int64{}
			}
			m.lastKanbanViews[ui.projectID] = viewID
		}
		m.kanbanActive = false
		m.projectViewPreference = projectViewKanban
		m.currentViewID = 0
		return m.loadViews(ui.projectID)
	}
	return nil
}
func (m *model) saveBoardConfiguration(ui *boardManagementUI, closeAfter bool) tea.Cmd {
	draft := cloneKanbanConfiguration(ui.draft)
	if ui.page == "config" {
		position, err := strconv.ParseFloat(strings.TrimSpace(ui.position), 64)
		if err != nil {
			ui.err = "View position must be a nonnegative number"
			return nil
		}
		draft.Position = position
		for i, b := range draft.Buckets {
			limit, err := strconv.ParseInt(strings.TrimSpace(ui.limits[b.ID]), 10, 64)
			if err != nil || limit < 0 {
				ui.err = "Bucket limit must be a nonnegative integer (0 = unlimited)"
				return nil
			}
			draft.Buckets[i].Limit = limit
		}
	}
	ui.saving = true
	ui.closeAfter = closeAfter
	ui.request++
	request := ui.request
	store := m.store
	create := ui.page == "create"
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		var cfg vimkunja.KanbanConfiguration
		var err error
		op := "save"
		if create {
			op = "create"
			cfg, err = store.CreateKanbanView(ctx, ui.projectID, draft.Title)
		} else {
			cfg, err = store.SaveKanbanConfiguration(ctx, draft)
		}
		return boardManagementMsg{ui: ui, request: request, operation: op, config: cfg, err: err}
	}
}
func (m *model) requestBucketDeletion(ui *boardManagementUI) {
	if ui.field < 2 {
		return
	}
	i := (ui.field - 2) / 2
	if len(ui.draft.Buckets) <= 1 {
		ui.err = "Cannot delete the last Bucket"
		return
	}
	b := ui.draft.Buckets[i]
	if b.ID == ui.draft.DefaultBucketID || b.ID == ui.draft.DoneBucketID {
		ui.err = "Reassign the Default/Done role before deleting this Bucket"
		return
	}
	ui.deleteIndex = i
	ui.confirm = "bucket"
	ui.err = ""
	ui.confirmScroll = 0
	ui.menuG = false
}
func (m *model) updateBoardManagement(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	ui := m.boardManagement
	k := key.String()
	if ui.saving {
		return m, nil
	}
	ui.err = ""
	if ui.helpOpen {
		if ui.commandMode {
			switch k {
			case "esc":
				ui.commandMode = false
			case "enter":
				ui.commandMode = false
				command, err := m.resolveManagementCommand(ui.command)
				if err != nil {
					ui.err = err.Error()
				} else {
					switch command {
					case "quit", "quit!":
						ui.helpOpen = false
						ui.menuG = false
					case "help", "":
					default:
						ui.err = "Command unavailable in help; :q returns to the draft"
					}
				}
			default:
				updateSingleLineTextInput(&ui.command, &ui.commandCursor, key)
			}
			return m, nil
		}
		switch k {
		case "esc", "q":
			ui.helpOpen = false
			ui.menuG = false
		case ":":
			ui.commandMode, ui.command, ui.commandCursor = true, "", 0
		default:
			_, h, _ := m.boardManagementSize()
			ui.helpScroll, _ = navigateManagement(k, &ui.menuG, ui.helpScroll, max(0, len(m.boardManagementHelpRows())-max(1, h-2)), h-2)
		}
		return m, nil
	}
	if ui.confirm != "" {
		if k == "esc" || k == "n" {
			ui.confirm = ""
			ui.menuG = false
			return m, nil
		}
		_, layout, limit := m.boardConfirmationBody()
		if next, handled := navigateManagement(k, &ui.menuG, ui.confirmScroll, limit, layout.contentHeight-4); handled {
			ui.confirmScroll = next
			return m, nil
		}
		if k != "y" {
			return m, nil
		}
		if w, h := m.viewportSize(); !confirmationUsable(w, h) {
			ui.err = "Resize before confirming deletion"
			return m, nil
		}
		kind := ui.confirm
		ui.confirm = ""
		switch kind {
		case "discard":
			return m, m.closeBoardManagement(false, 0)
		case "bucket":
			i := ui.deleteIndex
			ui.draft.Buckets = append(ui.draft.Buckets[:i], ui.draft.Buckets[i+1:]...)
			ui.field = clamp(ui.field, 0, 1+len(ui.draft.Buckets)*2)
			ui.normal.reset()
		case "board":
			ui.saving = true
			ui.request++
			request := ui.request
			store := m.store
			cfg := ui.pending
			return m, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				defer cancel()
				return boardManagementMsg{ui: ui, request: request, operation: "delete", err: store.DeleteKanbanView(ctx, cfg)}
			}
		}
		return m, nil
	}
	if ui.commandMode {
		switch k {
		case "esc":
			ui.commandMode = false
		case "enter":
			command, err := m.resolveManagementCommand(ui.command)
			ui.commandMode = false
			if err != nil {
				ui.err = err.Error()
				return m, nil
			}
			switch command {
			case "":
			case "help":
				ui.helpOpen, ui.helpScroll, ui.menuG = true, 0, false
				ui.err = ""
			case "write":
				if !ui.loading && (ui.page == "config" || ui.page == "create") {
					return m, m.saveBoardConfiguration(ui, false)
				}
				ui.err = "Save is unavailable in the Board list; e edits a Board"
			case "writequit", "x":
				if !ui.loading && (ui.page == "config" || ui.page == "create") {
					return m, m.saveBoardConfiguration(ui, true)
				}
				ui.err = "Save is unavailable in the Board list; e edits a Board"
			case "quit!":
				return m, m.closeBoardManagement(false, 0)
			case "quit":
				if ui.dirty() {
					ui.confirm = "discard"
				} else {
					return m, m.closeBoardManagement(false, 0)
				}
			case "edit!":
				if ui.page != "config" {
					ui.err = "Reload requires Board configuration"
				}
				if ui.page == "config" {
					for i, v := range ui.views {
						if v.ID == ui.draft.ViewID {
							ui.selected = i
							return m, m.loadBoardConfiguration(ui, "edit")
						}
					}
				}
			case "bucketdelete":
				if ui.page == "config" {
					m.requestBucketDeletion(ui)
				} else {
					ui.err = "Bucket deletion requires Board configuration"
				}
			default:
				ui.err = "Unknown Board command · :h explains available commands"
			}
		default:
			updateSingleLineTextInput(&ui.command, &ui.commandCursor, key)
		}
		return m, nil
	}
	if ui.mode == editorInsert {
		if k == "esc" || k == "enter" {
			ui.mode = editorNormal
			return m, nil
		}
		if k == "ctrl+v" {
			cmd, _ := m.textClipboardKey("p")
			return m, cmd
		}
		value := ui.fieldValue()
		updateSingleLineTextInput(&value, &ui.cursor, key)
		ui.setFieldValue(value)
		return m, nil
	}
	// On a Bucket row, dd removes the Bucket; other d motions still edit text.
	if !ui.loading && ui.page == "config" && ui.field >= 2 && k == "d" && ui.normal.operator == 'd' && ui.normal.count == 0 && ui.normal.operatorCount == 1 && ui.normal.textObject == 0 && ui.normal.selection == nil {
		ui.normal.reset()
		m.requestBucketDeletion(ui)
		return m, nil
	}
	if !ui.loading && ui.page != "manager" && ui.normal.pending() {
		if ui.applyNormalKey(k) {
			if ui.mode == editorInsert {
				return m, cursorBlinkCmd()
			}
			return m, nil
		}
	}
	if k == ":" {
		ui.commandMode = true
		ui.command = ""
		ui.commandCursor = 0
		return m, nil
	}
	if ui.loading {
		if k == "esc" {
			return m, m.closeBoardManagement(false, 0)
		}
		return m, nil
	}
	if ui.page == "manager" {
		switch k {
		case "q", "esc":
			return m, m.closeBoardManagement(false, 0)

		case "n", "o":
			ui.page = "create"
			ui.field = 0
			ui.draft = vimkunja.KanbanConfiguration{ProjectID: ui.projectID}
			ui.err = ""
		case "e":
			return m, m.loadBoardConfiguration(ui, "edit")
		case "d":
			return m, m.loadBoardConfiguration(ui, "preview")
		case "r":
			return m, m.loadBoardManager(ui)
		case "enter":
			if len(ui.views) > 0 {
				v := ui.views[ui.selected]
				if v.Filtered || v.BucketConfigurationMode != "manual" {
					ui.err = "Filtered Boards are preserved; open them in VOA"
				} else {
					return m, m.closeBoardManagement(true, v.ID)
				}
			}
		default:
			_, h, _ := m.boardManagementSize()
			ui.selected, _ = navigateManagement(k, &ui.menuG, ui.selected, len(ui.views)-1, h-2)
		}
		return m, nil
	}
	if k == "esc" {
		ui.normal.reset()
		m.leaderArmed = false
		return m, nil
	}
	if m.leaderArmed {
		m.leaderArmed = false
		if k == "y" || k == "p" {
			cmd, _ := m.textClipboardKey(k)
			return m, cmd
		}
		if ui.page == "config" && ui.field >= 2 {
			id := ui.draft.Buckets[(ui.field-2)/2].ID
			switch k {
			case "d":
				if ui.draft.DefaultBucketID == id {
					ui.draft.DefaultBucketID = 0
				} else {
					ui.draft.DefaultBucketID = id
				}
			case "x":
				if ui.draft.DoneBucketID == id {
					ui.draft.DoneBucketID = 0
				} else {
					ui.draft.DoneBucketID = id
				}
			case "D":
				m.requestBucketDeletion(ui)
			}
		}
		return m, nil
	}
	if isSpaceKey(key) {
		m.leaderArmed = true
		return m, tea.Tick(m.leaderTimeout, func(time.Time) tea.Msg { return leaderTimeoutMsg{} })
	}
	if ui.applyNormalKey(k) {
		if ui.mode == editorInsert {
			return m, cursorBlinkCmd()
		}
		return m, nil
	}
	if ui.page == "config" {
		if k == "o" || k == "O" {
			id := int64(-1)
			for _, b := range ui.draft.Buckets {
				if b.ID <= id {
					id = b.ID - 1
				}
			}
			index := len(ui.draft.Buckets)
			if k == "O" {
				index = 0
			}
			if ui.field >= 2 {
				index = (ui.field - 2) / 2
				if k == "o" {
					index++
				}
			}
			ui.draft.Buckets = slices.Insert(ui.draft.Buckets, index, vimkunja.KanbanBucketConfiguration{ID: id})
			ui.limits[id] = "0"
			ui.field = 2 + index*2
			ui.cursor = 0
			ui.normal.reset()
			return m, nil
		}
		if (k == "J" || k == "K") && ui.field >= 2 {
			i := (ui.field - 2) / 2
			j := i + 1
			if k == "K" {
				j = i - 1
			}
			if j >= 0 && j < len(ui.draft.Buckets) {
				ui.draft.Buckets[i], ui.draft.Buckets[j] = ui.draft.Buckets[j], ui.draft.Buckets[i]
				ui.field = 2 + j*2
			}
			ui.normal.reset()
			return m, nil
		}
	}
	count := 1
	if ui.page == "config" {
		count = 2 + len(ui.draft.Buckets)*2
	}
	switch k {
	case "j", "down", "tab":
		if k == "tab" {
			ui.field = (ui.field + 1) % count
		} else {
			ui.field = min(ui.field+1, count-1)
		}
		ui.cursor = 0
		ui.normal.reset()
		return m, nil
	case "k", "up", "shift+tab":
		if k == "shift+tab" {
			ui.field = (ui.field + count - 1) % count
		} else {
			ui.field = max(0, ui.field-1)
		}
		ui.cursor = 0
		ui.normal.reset()
		return m, nil
	}
	value := ui.fieldValue()
	switch k {
	case "i", "a", "I", "A":
		switch k {
		case "a":
			moveTextCursor(&ui.cursor, value, 1)
		case "I":
			ui.cursor = 0
		case "A":
			ui.cursor = len([]rune(value))
		}
		ui.mode = editorInsert
		return m, cursorBlinkCmd()
	}

	return m, nil
}

func (ui *boardManagementUI) applyNormalKey(key string) bool {
	switch key {
	case "left":
		key = "h"
	case "right":
		key = "l"
	case "home":
		key = "0"
	case "end":
		key = "$"
	}
	result := applyTextNormalKey(&ui.normal, ui.fieldValue(), ui.cursor, key, false)
	if !result.handled {
		return false
	}
	ui.setFieldValue(result.text)
	ui.cursor = result.cursor
	if result.enterInsert {
		ui.mode = editorInsert
	}
	return true
}

func (m *model) pasteBoardManagement(text string) {
	ui := m.boardManagement
	if ui.loading || ui.saving || ui.confirm != "" {
		return
	}
	text = sanitizeTerminalText(strings.NewReplacer("\r", " ", "\n", " ").Replace(text))
	if ui.commandMode {
		ui.command = insertTextAtCursor(ui.command, &ui.commandCursor, text)
	} else if !ui.helpOpen && ui.mode == editorInsert {
		value := ui.fieldValue()
		value = insertTextAtCursor(value, &ui.cursor, text)
		ui.setFieldValue(value)
	}
}

// Keep help local so returning restores the exact configuration draft and field.
func (m *model) boardManagementHelpRows() []string {
	entries := []helpEntry{{":h / :help", "Explain commands for this Board management page."}}
	if m.boardManagement.page == "manager" {
		entries = append(entries, []helpEntry{
			{"j/k / Up/Down", "Select a Board."},
			{"gg/G / Home/End", "Select the first/last Board."},
			{"Enter", "Open the selected Board and return to its Tasks."},
			{"n / o", "Create a standard Board after entering its name."},
			{"e", "Edit the selected Board's configuration and Buckets."},
			{"d", "Preview Board deletion; y confirms, n/Esc cancels. Tasks remain."},
			{"r", "Reload the Board list."},
			{"q / Esc / :q / :q!", "Close management and return to the previous view."},
		}...)
	} else {
		entries = append(entries, []helpEntry{
			{"j/k / Tab/Shift+Tab", "Select a configuration field in Normal mode."},
			{"i / a / I / A", "Edit the field at the cursor / after it / at the start / at the end."},
			{"Enter / Esc", "Finish text input and return to Normal mode."},
			{"v / V", "Select text with the shared Normal-mode editor."},
			{"Space y / Space p", "Copy selected text / paste from the clipboard."},
		}...)
		if m.boardManagement.page == "config" {
			entries = append(entries, []helpEntry{
				{"o / O", "Insert a Bucket after / before the selected Bucket in this draft."},
				{"J / K", "Move the selected Bucket right / left in the Board."},
				{"Space d", "Toggle the selected Bucket's Default role for new Tasks."},
				{"Space x", "Toggle the selected Bucket's Done role for completed Tasks."},
				{"dd / Space D / :bucketdelete", "Preview Bucket removal; y stages it until :w. Tasks move to Default."},
				{"View position", "A nonnegative number controls this Board's order within the Project."},
				{"Bucket limit", "Maximum active Tasks; 0 means unlimited."},
				{":w / :write", "Save Board configuration atomically and keep it open; Tasks are not edited."},
				{":e!", "Discard the draft and reload the current persisted configuration."},
			}...)
		} else {
			entries = append(entries, helpEntry{":w / :write", "Create a standard Board, then configure its To-Do, Doing and Done Buckets."})
		}
		entries = append(entries, []helpEntry{
			{":wq / :x", "Save and close configuration; a new Board first opens its Bucket editor."},
			{":q", "Close; ask before discarding an unsaved draft."},
			{":q!", "Discard unsaved configuration and close."},
		}...)
	}
	w, _, _ := m.boardManagementSize()
	return managementReferenceRows(entries, w)
}

func (m *model) boardManagementSize() (int, int, bool) {
	w, h := m.viewportSize()
	return managementSize(w, h)
}

func (m *model) renderBoardManagement() string {
	w, h, framed := m.boardManagementSize()
	content := m.renderBoardManagementContent(w, h)
	content = frameManagement(content, w, framed)
	if m.boardManagement.confirm != "" {
		content = m.renderBoardManagementConfirmation(content)
	}
	return content
}

func (m *model) renderBoardManagementContent(w, h int) string {
	ui := m.boardManagement
	if ui.helpOpen {
		rows := m.boardManagementHelpRows()
		if ui.err != "" {
			rows = append(wrapModalWords("Error: "+sanitizeTerminalText(ui.err), w), rows...)
		}
		visible := max(0, h-2)
		ui.helpScroll = clamp(ui.helpScroll, 0, max(0, len(rows)-max(1, visible)))
		helpTitle := "BOARD HELP · " + strings.ToUpper(ui.page)
		if ui.commandMode {
			helpTitle += " · COMMAND"
		}
		out := []string{helpTitle}
		out = append(out, rows[ui.helpScroll:min(len(rows), ui.helpScroll+visible)]...)
		for len(out) < h-1 {
			out = append(out, "")
		}
		footer := managementHint("j/k scroll · gg/G jump · Ctrl+D/U page · q/Esc/:q return", w)
		if ui.commandMode {
			footer = ":" + insertCursorText(ui.command, ui.commandCursor) + " "
			if w == 1 {
				footer = insertCursorText("", 0)
			}
		}
		out = append(out, footer)
		if h == 1 && ui.commandMode {
			out = []string{footer}
		}
		for i, row := range out {
			if strings.Contains(row, insertCursorAnchor) {
				row = fitInputCursor(row, w)
			}
			out[i] = ansi.Truncate(row, w, "")
		}
		return fitBlock(out, w, h)
	}
	title := "BOARDS"
	if ui.page == "create" {
		title = "NEW BOARD"
	}
	if ui.page == "config" {
		title = "BOARD CONFIGURATION"
	}
	if ui.dirty() {
		title += " [+]"
	}
	mode := ui.normal.selectionLabel()
	if ui.mode == editorInsert {
		mode = "INSERT"
	}
	if ui.commandMode {
		mode = "COMMAND"
	}
	title += " · " + mode
	if ui.page == "config" {
		available := max(0, w-ansi.StringWidth(title)-6)
		boardBudget := available / 2
		board := managementObjectLabel(ui.original.Title, ui.draft.ViewID, boardBudget)
		project := m.managementProjectIdentity(ui.projectID, max(0, available-ansi.StringWidth(board)))
		title += " · " + board + " · " + project
	} else {
		title += " · " + m.managementProjectIdentity(ui.projectID, max(0, w-ansi.StringWidth(title)-3))
	}
	if w < 12 || h < 3 {
		return fitBlock([]string{ansi.Truncate(title, w, ""), ansi.Truncate("Resize · :q!", w, "")}, w, h)
	}
	var rows []string
	selected := ui.field
	hint := ":h help · :w save · :q close · :q! discard · i edit · j/k fields"
	if ui.page == "manager" {
		hint = ":h help · Enter open · n new · e edit · d delete · r reload · Esc close"
		selected = ui.selected
		for _, v := range ui.views {
			label := fmt.Sprintf("%s (#%d)", sanitizeTerminalText(v.Title), v.ID)
			if v.Filtered || v.BucketConfigurationMode != "manual" {
				label += " [VOA filters]"
			}
			rows = append(rows, label)
		}
		if len(rows) == 0 {
			rows = []string{"No Boards · n creates one"}
		}
	} else {
		values := []string{ui.draft.Title}
		labels := []string{"Name: "}
		if ui.page == "config" {
			values = append(values, ui.position)
			labels = append(labels, "View position: ")
			for index, b := range ui.draft.Buckets {
				roles := ""
				if ui.draft.DefaultBucketID == 0 && index == 0 {
					roles += " [Default fallback]"
				}
				if b.ID == ui.draft.DefaultBucketID {
					roles += " [Default]"
				}
				if b.ID == ui.draft.DoneBucketID {
					roles += " [Done]"
				}
				values = append(values, b.Title, ui.limits[b.ID])
				labels = append(labels, "Bucket"+roles+": ", "  Limit (0=none): ")
			}
			hint = ":h help · Space d Default · Space x Done · dd delete · o/O add · J/K order · :w save · :q close"
		}
		for i, value := range values {
			value = sanitizeTerminalText(value)
			if i == ui.field && !ui.commandMode && ui.confirm == "" {
				if ui.mode == editorInsert {
					value = insertCursorText(value, ui.cursor)
				} else if ui.normal.selection != nil {
					lo, hi := ui.normal.selectionRange(value, ui.cursor)
					value = renderSelectionRange(value, 0, lo, hi)
				} else {
					value = normalCursorText(value, ui.cursor)
				}
			}
			rows = append(rows, labels[i]+value)
		}
	}
	if ui.created && ui.page == "config" {
		rows = append([]string{"Board created · :q! discards only unsaved configuration"}, rows...)
		selected++
	}
	if ui.loading {
		rows = []string{"Loading…"}
		selected = 0
	}
	if ui.mode == editorInsert {
		hint = "Esc normal · Enter normal · Ctrl+V paste"
	}
	if ui.saving {
		hint = "Saving Board configuration…"
	}
	var errorRows []string
	if ui.err != "" {
		errorRows = wrapModalWords("Error: "+sanitizeTerminalText(ui.err), max(1, w))
		errorRows = errorRows[:min(len(errorRows), max(0, min(3, h-2)))]
	}
	if ui.commandMode {
		hint = ":" + insertCursorText(ui.command, ui.commandCursor)
	}
	visible := max(0, h-2-len(errorRows))
	start := max(0, selected-visible+1)
	start = min(start, max(0, len(rows)-visible))
	out := []string{ansi.Truncate(title, w, "")}
	for i := start; i < min(len(rows), start+visible); i++ {
		prefix := "  "
		if i == selected && ui.confirm == "" {
			prefix = "> "
		}
		out = append(out, prefix+rows[i])
	}
	for len(out) < h-1-len(errorRows) {
		out = append(out, "")
	}
	out = append(out, errorRows...)
	if ui.commandMode || ui.saving {
		out = append(out, hint)
	} else {
		out = append(out, managementHint(hint, w))
	}
	for i, row := range out {
		if strings.Contains(row, insertCursorAnchor) {
			row = fitInputCursor(row, w)
		}
		out[i] = ansi.Truncate(row, w, "")
	}
	return fitBlock(out, w, h)
}

func (m *model) renderBoardManagementConfirmation(background string) string {
	ui := m.boardManagement
	w, h := m.viewportSize()
	if w < 12 || h < 5 {
		return fitBlock([]string{ansi.Truncate("CONFIRM "+strings.ToUpper(ui.confirm), w, ""), ansi.Truncate("Resize · n/Esc cancel", w, "")}, w, h)
	}
	rows, layout, limit := m.boardConfirmationBody()
	ui.confirmScroll = clamp(ui.confirmScroll, 0, limit)
	return overlayModalLayout(background, renderModalFrame(rows, layout), w, layout.fallback)
}

func (m *model) boardConfirmationBody() ([]string, modalLayout, int) {
	ui := m.boardManagement
	w, h := m.viewportSize()
	message := "Discard this Board configuration draft?"
	if ui.confirm == "board" {
		count := 0
		for _, b := range ui.pending.Buckets {
			count += b.TaskCount
		}
		message = fmt.Sprintf("Delete Board %s (#%d), %d Buckets and %d card assignments? All Project Tasks and other Boards remain. No deletion Undo.", sanitizeTerminalText(ui.pending.Title), ui.pending.ViewID, len(ui.pending.Buckets), count)
	}
	if ui.confirm == "bucket" {
		b := ui.draft.Buckets[ui.deleteIndex]
		destination := ""
		for _, candidate := range ui.draft.Buckets {
			if candidate.ID != b.ID {
				destination = candidate.Title
				break
			}
		}
		for _, candidate := range ui.draft.Buckets {
			if candidate.ID == ui.draft.DefaultBucketID {
				destination = candidate.Title
			}
		}
		message = fmt.Sprintf("Delete Bucket %s (%d assignments)? Tasks move to %s when :w saves. Task completion remains unchanged.", sanitizeTerminalText(b.Title), b.TaskCount, sanitizeTerminalText(destination))
	}
	layout := modalBounds(w, h, 76, h)
	questionLines := wrapModalWords(message, layout.contentWidth)
	layout = modalBounds(w, h, 76, len(questionLines)+8)
	actions := "y delete  •  n / Esc cancel"
	title := "CONFIRM " + strings.ToUpper(ui.confirm)
	style := lipgloss.NewStyle().Bold(true)
	if ui.confirm == "discard" {
		actions = "y discard  •  n / Esc keep editing"
	} else {
		style = style.Foreground(lipgloss.Color(tuiTheme.danger))
	}
	rows, limit := confirmationBody(style.Render(title), message, actions, layout, ui.confirmScroll)
	return rows, layout, limit
}
