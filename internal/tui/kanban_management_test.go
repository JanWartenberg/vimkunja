package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

func TestBoardSwitchKeepsSelectionAndActionsVisible(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 10})
	m.currentProjectID = 1
	for i := int64(1); i <= 20; i++ {
		m.projectViews = append(m.projectViews, vimkunja.ProjectView{ID: i, Title: fmt.Sprintf("Board %02d", i), ViewKind: "kanban", BucketConfigurationMode: "manual"})
	}
	m.currentViewID = 1
	m.beginKanbanViewPicker()
	for i := 0; i < 19; i++ {
		m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	}
	rendered := ansi.Strip(m.Render())
	if !strings.Contains(rendered, "Board 20") || !strings.Contains(rendered, "Esc cancel") {
		t.Fatalf("selection/actions outside viewport: %s", rendered)
	}
}

func TestBoardMenusUseVimNavigation(t *testing.T) {
	for _, page := range []string{"manager", "switch", "help"} {
		t.Run(page, func(t *testing.T) {
			m := testModel()
			m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
			m.currentProjectID = 1
			views := make([]vimkunja.ProjectView, 20)
			for i := range views {
				views[i] = vimkunja.ProjectView{ID: int64(i + 1), Title: fmt.Sprint(i), ViewKind: "kanban", BucketConfigurationMode: "manual"}
			}
			ui := &boardManagementUI{page: "manager", views: views, selected: 10}
			m.boardManagement = ui
			if page == "switch" {
				m.boardManagement = nil
				m.projectViews = views
				m.kanbanViewPicker = true
				m.kanbanViewPickerIndex = 10
			}
			if page == "help" {
				ui.page = "config"
				ui.helpOpen = true
				ui.helpScroll = 10
			}
			position := func() int {
				if page == "switch" {
					return m.kanbanViewPickerIndex
				}
				if page == "help" {
					return ui.helpScroll
				}
				return ui.selected
			}
			m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
			if position() != 10 {
				t.Fatal("single g must wait for gg")
			}
			m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
			if position() != 0 {
				t.Fatal("gg did not select the beginning")
			}
			m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
			if position() == 0 {
				t.Fatal("Ctrl+D did not move half a page")
			}
		})
	}
}

func TestBoardConfirmationSupportsSharedPaging(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 32, Height: 9})
	ui := &boardManagementUI{page: "manager", confirm: "board", pending: vimkunja.KanbanConfiguration{Title: strings.Repeat("Long Board ", 30)}}
	m.boardManagement = ui
	m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
	if ui.confirmScroll == 0 {
		t.Fatal("confirmation ignored Ctrl+D")
	}
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	if ui.confirmScroll != 0 {
		t.Fatal("confirmation ignored gg")
	}
	if !strings.Contains(m.Render(), "y delete") {
		t.Fatal("delete confirmation has no explicit delete action")
	}
}

func TestBoardFieldNavigationClampsAndTabCycles(t *testing.T) {
	m := testModel()
	ui := &boardManagementUI{page: "config"}
	ui.adopt(vimkunja.KanbanConfiguration{Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 1}}})
	m.boardManagement = ui
	m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
	if ui.field != 0 {
		t.Fatal("k wraps instead of staying on the first field")
	}
	ui.field = 3
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if ui.field != 3 {
		t.Fatal("j wraps instead of staying on the last field")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	if ui.field != 0 {
		t.Fatal("Tab must still cycle fields")
	}
}

func TestBoardHintsFollowInputMode(t *testing.T) {
	m := testModel()
	m.width, m.height = 120, 24
	ui := &boardManagementUI{page: "config"}
	ui.adopt(vimkunja.KanbanConfiguration{Title: "Board"})
	m.boardManagement = ui
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	rendered := ansi.Strip(m.Render())
	if !strings.Contains(rendered, "Esc normal") || strings.Contains(rendered, "Space d Default") {
		t.Fatal("Insert advertises inactive Normal commands")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	if !strings.Contains(m.Render(), "COMMAND") {
		t.Fatal("Ex input still shows Normal mode")
	}
}

func TestBoardManagementIdentifiesProjectAndCreatedState(t *testing.T) {
	m := testModel()
	m.width, m.height = 160, 24
	m.currentProjectID = 7
	m.projects = []vimkunja.Project{{ID: 7, Title: "Release planning"}}
	ui := &boardManagementUI{page: "manager", projectID: 7}
	m.boardManagement = ui
	if !strings.Contains(m.Render(), "Release planning (#7)") {
		t.Fatal("manager hides its target Project")
	}
	m.Update(boardManagementMsg{ui: ui, operation: "create", config: vimkunja.KanbanConfiguration{ViewID: 20, Title: "Experiment", Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 1, Title: "To-Do"}}}})
	if !strings.Contains(m.Render(), "Board created") || !strings.Contains(m.Render(), ":q! discards only unsaved configuration") {
		t.Fatal("created Board looks like an unpersisted draft")
	}
}

func TestConfigurationHeaderKeepsBoardIdentityAtOrdinaryWidth(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m.projects = []vimkunja.Project{{ID: 7, Title: "Quarterly customer migration planning"}}
	ui := &boardManagementUI{projectID: 7, page: "config"}
	ui.adopt(vimkunja.KanbanConfiguration{ViewID: 42, Title: "Delivery", Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 1, Title: "To-Do"}, {ID: 2, Title: "Doing"}, {ID: 3, Title: "Done"}}})
	ui.field = 7
	m.boardManagement = ui
	lines := strings.Split(ansi.Strip(m.Render()), "\n")
	header := lines[2]
	for _, want := range []string{"NORMAL", "Delivery", "(#42)", "(#7)"} {
		if !strings.Contains(header, want) {
			t.Fatalf("header hides %q: %s", want, header)
		}
	}
}

func TestBoardCommandsUseEditorAliasesAndRejectUnavailableActions(t *testing.T) {
	for _, command := range []string{"writequit", "quit", "quit!", "edit!"} {
		t.Run(command, func(t *testing.T) {
			m := testModel()
			ui := &boardManagementUI{page: "config", views: []vimkunja.ProjectView{{ID: 1, BucketConfigurationMode: "manual"}}}
			ui.adopt(vimkunja.KanbanConfiguration{ViewID: 1, Title: "Board"})
			m.boardManagement = ui
			m, cmd := typeCommand(t, m, command)
			if ui.err != "" || (command == "writequit" || command == "edit!") && cmd == nil {
				t.Fatalf("editor alias rejected: %s %s", command, ui.err)
			}
		})
	}
	m := testModel()
	ui := &boardManagementUI{page: "manager"}
	m.boardManagement = ui
	m, _ = typeCommand(t, m, "w")
	if ui.err == "" {
		t.Fatal("unavailable command silently ignored")
	}
}

type kanbanManagementStore struct {
	*fakeStore
	cfg     vimkunja.KanbanConfiguration
	saved   vimkunja.KanbanConfiguration
	deleted bool
	err     error
}

func (s *kanbanManagementStore) LoadKanbanConfiguration(context.Context, int64, int64) (vimkunja.KanbanConfiguration, error) {
	return s.cfg, s.err
}
func (s *kanbanManagementStore) SaveKanbanConfiguration(_ context.Context, c vimkunja.KanbanConfiguration) (vimkunja.KanbanConfiguration, error) {
	s.saved = c
	return c, s.err
}
func (s *kanbanManagementStore) CreateKanbanView(_ context.Context, p int64, title string) (vimkunja.KanbanConfiguration, error) {
	s.cfg.Title = title
	return s.cfg, s.err
}
func (s *kanbanManagementStore) DeleteKanbanView(context.Context, vimkunja.KanbanConfiguration) error {
	s.deleted = true
	return s.err
}

type pagedBoardStore struct{ *kanbanManagementStore }

func (s *pagedBoardStore) ListViews(_ context.Context, _ int64, page, perPage int) ([]vimkunja.ProjectView, int, error) {
	start := (page - 1) * perPage
	if start >= len(s.views) {
		return nil, 2, nil
	}
	end := min(start+perPage, len(s.views))
	return s.views[start:end], 2, nil
}

func TestBoardBeyondFirstViewPageOpensAfterManagement(t *testing.T) {
	for _, operation := range []string{"open", "create"} {
		t.Run(operation, func(t *testing.T) {
			m := testModel()
			views := make([]vimkunja.ProjectView, 201)
			for i := range views {
				views[i] = vimkunja.ProjectView{ID: int64(i + 1), Title: fmt.Sprintf("Board %d", i+1), ViewKind: "kanban", BucketConfigurationMode: "manual"}
			}
			m.store = &pagedBoardStore{&kanbanManagementStore{fakeStore: &fakeStore{views: views}}}
			m.currentProjectID = 1
			ui := &boardManagementUI{projectID: 1, page: "manager", request: 1}
			m.boardManagement = ui
			var cmd tea.Cmd
			if operation == "create" {
				_, cmd = m.Update(boardManagementMsg{ui: ui, request: 1, operation: "create", config: vimkunja.KanbanConfiguration{ViewID: 201}})
				m, cmd = typeCommand(t, m, "q")
			} else {
				cmd = m.closeBoardManagement(true, 201)
			}
			m.Update(cmd())
			if m.currentViewID != 201 || len(m.projectViews) != 201 {
				t.Fatalf("requested Board lost after %s: selected=%d Views=%d", operation, m.currentViewID, len(m.projectViews))
			}
		})
	}
}

func TestBoardConfigurationErrorKeepsHelpersVisible(t *testing.T) {
	m := testModel()
	m.width, m.height = 120, 24
	ui := &boardManagementUI{page: "config", err: "Reassign the Default/Done role before deleting this Bucket"}
	ui.adopt(vimkunja.KanbanConfiguration{Title: "Board", DefaultBucketID: 1, Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 1, Title: "To-Do"}, {ID: 2, Title: "Doing"}}})
	m.boardManagement = ui
	rendered := ansi.Strip(m.Render())
	for _, want := range []string{"Error:", ":h help", "Space d Default", "Space x Done"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("error hides %q: %s", want, rendered)
		}
	}
}

func TestBoardManagementUsesFramedView(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 24
	m.boardManagement = &boardManagementUI{page: "manager"}
	lines := strings.Split(ansi.Strip(m.Render()), "\n")
	if !strings.HasPrefix(lines[0], "╭") || !strings.Contains(lines[len(lines)-1], "╯") {
		t.Fatalf("management has no normal Pane frame: %s", m.Render())
	}
}

func TestBoardManagementPaddingAndFooterSeparator(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 24
	m.boardManagement = &boardManagementUI{page: "manager"}
	lines := strings.Split(ansi.Strip(m.Render()), "\n")
	if strings.Trim(lines[1], "│ ") != "" || !strings.HasPrefix(lines[2], "│ BOARDS") {
		t.Fatalf("management needs one cell of top/left space: %s", m.Render())
	}
	if !strings.Contains(lines[len(lines)-3], "──") || !strings.Contains(lines[len(lines)-2], ":h help") {
		t.Fatal("helper footer needs a horizontal separator")
	}
	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	lines = strings.Split(ansi.Strip(m.Render()), "\n")
	if !strings.HasPrefix(lines[len(lines)-2], "│ :") {
		t.Fatal("Ex prompt must be just above the bottom border")
	}
}

func TestBucketInsertionFollowsSelectedBucket(t *testing.T) {
	for _, tc := range []struct {
		key          string
		field, index int
	}{{"o", 2, 1}, {"O", 2, 0}, {"o", 5, 2}, {"O", 5, 1}} {
		m := testModel()
		ui := &boardManagementUI{page: "config"}
		ui.adopt(vimkunja.KanbanConfiguration{Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 1, Title: "First"}, {ID: 2, Title: "Second"}, {ID: 3, Title: "Last"}}})
		ui.field = tc.field
		m.boardManagement = ui
		m = pressKey(m, tea.Key{Code: []rune(tc.key)[0], Text: tc.key})
		if len(ui.draft.Buckets) != 4 || ui.draft.Buckets[tc.index].ID >= 0 || ui.field != 2+tc.index*2 {
			t.Fatalf("%s from field %d inserted at wrong location: %#v field %d", tc.key, tc.field, ui.draft.Buckets, ui.field)
		}
		if ui.draft.Buckets[len(ui.draft.Buckets)-1].Title != "Last" {
			t.Fatal("insertion disturbed existing Bucket order")
		}
	}
}

func TestBucketDoubleDeleteUsesCenteredConfirmationAndPreservesText(t *testing.T) {
	m := testModel()
	m.width, m.height = 120, 40
	ui := &boardManagementUI{page: "config"}
	ui.adopt(vimkunja.KanbanConfiguration{Title: "Board", DefaultBucketID: 1, Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 1, Title: "To-Do"}, {ID: 2, Title: "Doing", TaskCount: 3}}})
	ui.field = 4
	m.boardManagement = ui
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	if ui.confirm != "bucket" || ui.draft.Buckets[1].Title != "Doing" || len(ui.draft.Buckets) != 2 {
		t.Fatal("dd must preview Bucket removal without deleting its title or the Bucket")
	}
	lines := strings.Split(ansi.Strip(m.Render()), "\n")
	question, actions := -1, -1
	for i, line := range lines {
		if strings.Contains(line, "CONFIRM BUCKET") {
			question = i
			if strings.Index(line, "CONFIRM BUCKET") < 10 {
				t.Fatal("confirmation is not horizontally centered")
			}
		}
		if strings.Contains(line, "y delete") {
			actions = i
		}
	}
	if question < 5 || actions < question || actions-question > 8 || !strings.Contains(strings.Join(lines, "\n"), "┏") {
		t.Fatalf("confirmation must be a compact centered framed modal: %s", m.Render())
	}
	m.Update(tea.WindowSizeMsg{Width: 20, Height: 4})
	m = pressKey(m, tea.Key{Code: 'y', Text: "y"})
	if ui.confirm != "bucket" || len(ui.draft.Buckets) != 2 || !strings.Contains(m.Render(), "Resize") {
		t.Fatal("a short terminal allowed confirmation without a visible action row")
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = pressKey(m, tea.Key{Code: 'n', Text: "n"})
	if ui.confirm != "" || len(ui.draft.Buckets) != 2 {
		t.Fatal("cancel removed a Bucket")
	}
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'y', Text: "y"})
	if len(ui.draft.Buckets) != 1 || !ui.dirty() {
		t.Fatal("confirmed removal was not staged in the draft")
	}
}

func TestKanbanOpensImmediatelyAndPickerCancelKeepsBoard(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.currentViewID = 12
	m.focus = paneTasks
	m.projectViews = []vimkunja.ProjectView{{ID: 11, Title: "Delivery", ViewKind: "kanban", BucketConfigurationMode: "manual"}, {ID: 12, Title: "Review", ViewKind: "kanban", BucketConfigurationMode: "manual"}}
	m.setKanbanBoard(vimkunja.KanbanBoard{View: m.projectViews[1]})
	_, cmd := typeCommand(t, m, "kanban")
	if cmd == nil || m.kanbanViewPicker {
		t.Fatal("kanban did not open directly")
	}
	_, cmd = typeCommand(t, m, "kanban switch")
	if !m.kanbanViewPicker {
		t.Fatal("explicit switch did not open picker")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	if m.currentViewID != 12 || !m.kanbanActive || m.kanbanViewPicker {
		t.Fatal("cancel left original Board")
	}
}

func TestBoardConfigurationIsDistinctAndSavesOnlyConfiguration(t *testing.T) {
	m := testModel()
	s := &kanbanManagementStore{fakeStore: &fakeStore{views: []vimkunja.ProjectView{{ID: 11, Title: "Delivery", ViewKind: "kanban", BucketConfigurationMode: "manual"}}}, cfg: vimkunja.KanbanConfiguration{ProjectID: 1, ViewID: 11, Title: "Delivery", Revision: "one", DefaultBucketID: 100, Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 100, Title: "To-Do"}}}}
	m.store = s
	m.currentProjectID = 1
	m.width, m.height = 80, 24
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task must not appear"})
	m, cmd := typeCommand(t, m, "kanban manage")
	if cmd == nil {
		t.Fatal("manager did not load")
	}
	m.Update(cmd())
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'e', Text: "e"}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("configuration did not load")
	}
	m.Update(cmd())
	rendered := ansi.Strip(m.Render())
	if !strings.Contains(rendered, "BOARD CONFIGURATION") || strings.Contains(rendered, "Task must not appear") || m.showDetail {
		t.Fatalf("ambiguous editing surface: %q", rendered)
	}
	m = pressKey(m, tea.Key{Code: 'A', Text: "A"})
	m.Update(tea.PasteMsg{Content: " changed"})
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	m, cmd = typeCommand(t, m, "w")
	if cmd == nil {
		t.Fatal("configuration save not scheduled")
	}
	m.Update(cmd())
	if s.saved.Title != "Delivery changed" || s.updatedDraft.Title != "" {
		t.Fatalf("wrong save: config %#v Task %#v", s.saved, s.updatedDraft)
	}
}

func TestBoardManagementBlocksUnsavedTaskAndRequiresDeletionConfirmation(t *testing.T) {
	m := testModel()
	s := &kanbanManagementStore{fakeStore: &fakeStore{views: []vimkunja.ProjectView{{ID: 11, Title: "Delivery", ViewKind: "kanban", BucketConfigurationMode: "manual"}}}, cfg: vimkunja.KanbanConfiguration{ProjectID: 1, ViewID: 11, Title: "Delivery", Revision: "one", Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 100, Title: "To-Do", TaskCount: 2}}}}
	m.store = s
	m.currentProjectID = 1
	m.width, m.height = 60, 12
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Saved"})
	m.editor.draft.Title = "Unsaved"
	_, cmd := typeCommand(t, m, "kanban manage")
	if cmd != nil || m.editor.draft.Title != "Unsaved" || !m.showDetail {
		t.Fatal("unsaved Task was not protected")
	}
	m.editor.draft.Title = "Saved"
	m, cmd = typeCommand(t, m, "kanban manage")
	m.Update(cmd())
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))
	m.Update(cmd())
	if s.deleted || !strings.Contains(ansi.Strip(m.Render()), "Delete Board") {
		t.Fatal("missing deletion preview")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	if s.deleted {
		t.Fatal("cancel deleted Board")
	}
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))
	m.Update(cmd())
	m.Update(tea.WindowSizeMsg{Width: 8, Height: 2})
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	if cmd != nil {
		t.Fatal("blind deletion allowed in unreadable terminal")
	}
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	if cmd == nil {
		t.Fatal("confirmed deletion did not run")
	}
	m.Update(cmd())
	if !s.deleted {
		t.Fatal("deletion missing")
	}
}

func TestBoardConfigurationResizeKeepsActiveInputAndBounds(t *testing.T) {
	m := testModel()
	ui := &boardManagementUI{projectID: 1, page: "config"}
	ui.adopt(vimkunja.KanbanConfiguration{ProjectID: 1, ViewID: 11, Title: "Delivery", Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 1, Title: "First"}, {ID: 2, Title: "Second"}, {ID: 3, Title: "Third"}}})
	ui.field = 6
	m.boardManagement = ui
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	for _, size := range [][2]int{{80, 24}, {24, 5}, {12, 3}, {8, 2}, {1, 1}, {80, 24}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		rendered := m.Render()
		lines := strings.Split(rendered, "\n")
		if len(lines) > size[1] {
			t.Fatalf("too many rows at %v: %q", size, rendered)
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("too wide at %v: %q", size, line)
			}
		}
		if size[0] >= 12 && size[1] >= 3 && m.renderedInputCursor == nil {
			t.Fatalf("active input cursor lost at %v", size)
		}
	}
}

func TestBoardConfigurationClipboardAndDiscardProtectDraft(t *testing.T) {
	m := testModel()
	ui := &boardManagementUI{projectID: 1, page: "config"}
	ui.adopt(vimkunja.KanbanConfiguration{ProjectID: 1, ViewID: 11, Title: "Delivery", Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 1, Title: "To-Do"}}})
	m.boardManagement = ui
	m.width, m.height = 60, 12
	m.clipboardRead = func() (string, error) { return "New ", nil }
	m = pressKey(m, tea.Key{Code: tea.KeySpace, Text: " "})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'p', Text: "p"}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("clipboard read missing")
	}
	m.Update(cmd())
	if ui.draft.Title != "New Delivery" || ui.mode != editorNormal {
		t.Fatalf("clipboard paste = %q", ui.draft.Title)
	}
	m, _ = typeCommand(t, m, "q")
	if !strings.Contains(m.Render(), "Discard this Board") {
		t.Fatal("dirty close lacks confirmation")
	}
	m = pressKey(m, tea.Key{Code: 'n', Text: "n"})
	if m.boardManagement != ui || ui.draft.Title != "New Delivery" {
		t.Fatal("cancel lost draft")
	}
	m, _ = typeCommand(t, m, "q!")
	if m.boardManagement != nil {
		t.Fatal("explicit discard did not close")
	}
}

func TestBoardConfigurationStaleResultsCannotReopenOrSaveAnotherContext(t *testing.T) {
	m := testModel()
	s := &kanbanManagementStore{fakeStore: &fakeStore{}, cfg: vimkunja.KanbanConfiguration{ProjectID: 1, ViewID: 11, Title: "New"}}
	m.store = s
	m.currentProjectID = 1
	m, cmd := typeCommand(t, m, "kanban manage")
	if cmd == nil {
		t.Fatal("load missing")
	}
	late := cmd()
	m, _ = typeCommand(t, m, "q!")
	m.Update(late)
	if m.boardManagement != nil {
		t.Fatal("late result reopened manager")
	}
	m, cmd = typeCommand(t, m, "kanban new")
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m.Update(tea.PasteMsg{Content: "New"})
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	m, cmd = typeCommand(t, m, "w")
	if cmd == nil {
		t.Fatal("save missing")
	}
	_, again := typeCommand(t, m, "w")
	if again != nil {
		t.Fatal("duplicate save scheduled")
	}
	m.Update(cmd())
}

func (*fakeStore) CreateKanbanView(context.Context, int64, string) (vimkunja.KanbanConfiguration, error) {
	return vimkunja.KanbanConfiguration{}, fmt.Errorf("test Store does not implement Board creation")
}
func (*fakeStore) LoadKanbanConfiguration(context.Context, int64, int64) (vimkunja.KanbanConfiguration, error) {
	return vimkunja.KanbanConfiguration{}, fmt.Errorf("test Store does not implement Board configuration")
}
func (*fakeStore) SaveKanbanConfiguration(context.Context, vimkunja.KanbanConfiguration) (vimkunja.KanbanConfiguration, error) {
	return vimkunja.KanbanConfiguration{}, fmt.Errorf("test Store does not implement Board configuration")
}
func (*fakeStore) DeleteKanbanView(context.Context, vimkunja.KanbanConfiguration) error {
	return fmt.Errorf("test Store does not implement Board deletion")
}

func TestSmallKanbanTerminalShowsBoardAndCommandInput(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.focus = paneTasks
	m.kanbanActive = true
	m.kanbanBoard = vimkunja.KanbanBoard{View: vimkunja.ProjectView{ID: 11, Title: "Delivery"}}
	m.projectViews = []vimkunja.ProjectView{{ID: 11, ViewKind: "kanban"}, {ID: 12, ViewKind: "kanban"}}
	m.Update(tea.WindowSizeMsg{Width: 24, Height: 6})
	if !strings.Contains(m.Render(), "Delivery") {
		t.Fatal("small terminal hides Board identity")
	}
	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
	if !strings.Contains(m.Render(), ":k") || m.renderedInputCursor == nil {
		t.Fatal("small terminal hides Ex input")
	}
}

func TestBoardCreationFromOverviewOpensProjectBoard(t *testing.T) {
	m := testModel()
	s := &kanbanManagementStore{fakeStore: &fakeStore{}, cfg: vimkunja.KanbanConfiguration{ProjectID: 1, ViewID: 12, Title: "New"}}
	m.store = s
	m.currentProjectID = 1
	m.taskScope = scopeOverview
	m, _ = typeCommand(t, m, "kanban new")
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m.Update(tea.PasteMsg{Content: "New"})
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	m, cmd := typeCommand(t, m, "w")
	m.Update(cmd())
	if m.boardManagement == nil || m.boardManagement.page != "config" || m.showDetail {
		t.Fatal("new Board did not open its standalone Bucket editor")
	}
	m, _ = typeCommand(t, m, "q")
	if m.inOverview() {
		t.Fatal("closing configuration did not open the Project Board")
	}
}

func TestBoardConfigurationUsesSharedNormalTextCommands(t *testing.T) {
	for _, tc := range []struct {
		keys, want string
		insert     bool
	}{{"ciw", " beta", true}, {"diw", " beta", false}, {"ri", "ilpha beta", false}, {"ro", "olpha beta", false}} {
		t.Run(tc.keys, func(t *testing.T) {
			m := testModel()
			ui := &boardManagementUI{page: "config"}
			ui.adopt(vimkunja.KanbanConfiguration{Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 1, Title: "alpha beta"}}})
			ui.field = 2
			m.boardManagement = ui
			for _, r := range tc.keys {
				m = pressKey(m, tea.Key{Code: r, Text: string(r)})
			}
			if len(ui.draft.Buckets) != 1 || ui.fieldValue() != tc.want || (ui.mode == editorInsert) != tc.insert {
				t.Fatalf("%s: value=%q mode=%v Buckets=%d", tc.keys, ui.fieldValue(), ui.mode, len(ui.draft.Buckets))
			}
		})
	}
}

func TestBoardManagementContextHelpPreservesDraft(t *testing.T) {
	for _, page := range []string{"manager", "create", "config"} {
		t.Run(page, func(t *testing.T) {
			m := testModel()
			m.width, m.height = 100, 40
			ui := &boardManagementUI{page: page}
			ui.adopt(vimkunja.KanbanConfiguration{Title: "Original", Buckets: []vimkunja.KanbanBucketConfiguration{{ID: 1, Title: "To-Do"}}})
			ui.draft.Title = "Unsaved"
			m.boardManagement = ui
			m, _ = typeCommand(t, m, "h")
			rendered := ansi.Strip(m.Render())
			if !strings.Contains(rendered, "HELP") || !strings.Contains(rendered, ":h / :help") || strings.Contains(rendered, "Error:") {
				t.Fatalf("missing contextual help: %s", rendered)
			}
			want := "Enter"
			if page == "config" {
				want = ":bucketdelete"
			}
			if page == "create" {
				want = "standard Board"
			}
			if !strings.Contains(rendered, want) {
				t.Fatalf("missing %s help: %s", page, rendered)
			}
			m, _ = typeCommand(t, m, "w")
			if !strings.Contains(m.Render(), "HELP") || ui.draft.Title != "Unsaved" {
				t.Fatal("help command modified or closed the draft")
			}
			m = pressKey(m, tea.Key{Code: tea.KeyEsc})
			if m.boardManagement != ui || ui.draft.Title != "Unsaved" || strings.Contains(m.Render(), "HELP") {
				t.Fatal("closing help did not restore configuration draft")
			}
			m, _ = typeCommand(t, m, "help")
			m, _ = typeCommand(t, m, "q")
			if m.boardManagement != ui || strings.Contains(m.Render(), "HELP") {
				t.Fatal(":q did not return from help")
			}
		})
	}
}

func TestBoardManagementHelpScrollAndResize(t *testing.T) {
	m := testModel()
	ui := &boardManagementUI{page: "config"}
	ui.adopt(vimkunja.KanbanConfiguration{Title: "Draft"})
	m.boardManagement = ui
	m, _ = typeCommand(t, m, "h")
	for _, size := range [][2]int{{32, 8}, {12, 3}, {4, 1}, {80, 24}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
		rendered := m.Render()
		lines := strings.Split(rendered, "\n")
		if len(lines) > size[1] {
			t.Fatalf("help exceeds height %d: %d", size[1], len(lines))
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("help exceeds width %d", size[0])
			}
		}
	}
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	if ui.helpScroll != 0 {
		t.Fatal("help cannot return to its first command")
	}
	if ui.draft.Title != "Draft" {
		t.Fatal("help mutated the draft")
	}
}

func TestBoardHelpAlignsExplanations(t *testing.T) {
	m := testModel()
	m.width, m.height = 120, 40
	m.boardManagement = &boardManagementUI{page: "config", helpOpen: true}
	column := -1
	count := 0
	for _, line := range strings.Split(ansi.Strip(m.Render()), "\n") {
		if index := strings.Index(line, " — "); index >= 0 {
			if column >= 0 && index != column {
				t.Fatalf("help explanations do not align: %s", m.Render())
			}
			column = index
			count++
		}
	}
	if count < 10 {
		t.Fatalf("missing command reference: %s", m.Render())
	}
}

func TestSingleBoardOmitsRedundantTitleLine(t *testing.T) {
	m := testModel()
	m.projectViews = []vimkunja.ProjectView{{ID: 1, ViewKind: "list"}, {ID: 2, ViewKind: "kanban"}}
	m.kanbanBoard = vimkunja.KanbanBoard{View: vimkunja.ProjectView{ID: 2, Title: "Kanban"}, Buckets: []vimkunja.KanbanBucket{{ID: 1, Title: "To-Do"}}}
	rendered := ansi.Strip(renderKanbanBoard(m, 80, 20))
	if strings.Contains(rendered, "Kanban: Kanban") || !strings.Contains(strings.Split(rendered, "\n")[0], "─") {
		t.Fatalf("redundant single-Board header: %s", rendered)
	}
	m.projectViews = append(m.projectViews, vimkunja.ProjectView{ID: 3, ViewKind: "kanban"})
	if !strings.Contains(renderKanbanBoard(m, 80, 20), "Kanban: Kanban") {
		t.Fatal("multiple Boards must identify the active Board")
	}
}

func TestDeletingLastBoardWithoutListShowsEmptyCreationState(t *testing.T) {
	m := testModel()
	s := &kanbanManagementStore{fakeStore: &fakeStore{}, cfg: vimkunja.KanbanConfiguration{ProjectID: 1, ViewID: 11, Title: "Only", Revision: "one"}}
	m.store = s
	m.currentProjectID = 1
	m.currentViewID = 11
	m.taskScope = scopeProject
	m.projectViewPreference = projectViewKanban
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Stale card"}}
	m.projectTasks = m.tasks
	m.kanbanBoard = vimkunja.KanbanBoard{View: vimkunja.ProjectView{ID: 11, Title: "Only"}}
	m.kanbanActive = true
	ui := &boardManagementUI{projectID: 1, page: "manager", request: 1}
	m.boardManagement = ui
	_, load := m.Update(boardManagementMsg{ui: ui, request: 1, operation: "delete"})
	if load == nil {
		t.Fatal("post-delete reload missing")
	}
	m.Update(load())
	rendered := ansi.Strip(m.Render())
	if len(m.tasks) != 0 || m.currentViewID != 0 || strings.Contains(rendered, "Stale card") || !strings.Contains(rendered, ":kanban new") {
		t.Fatalf("missing empty creation state: %s", rendered)
	}
	m, _ = typeCommand(t, m, "kanban new")
	if !strings.Contains(m.Render(), "NEW BOARD") {
		t.Fatal("cannot create from empty state")
	}
}
