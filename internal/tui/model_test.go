package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	apperrors "github.com/vimkunja/vimkunja/internal/errors"
	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

type fakeStore struct {
	toggleErr      error
	listTasksCalls int
	createdDraft   vimkunja.TaskDraft
}

func (f *fakeStore) ListProjects(context.Context, int, int) ([]vimkunja.Project, int, error) {
	return nil, 0, nil
}

func (f *fakeStore) ListViews(context.Context, int64, int, int) ([]vimkunja.ProjectView, int, error) {
	return nil, 0, nil
}

func (f *fakeStore) ListTasksInView(context.Context, int64, int64, int, int) ([]vimkunja.Task, int, error) {
	f.listTasksCalls++
	return nil, 0, nil
}

func (f *fakeStore) ToggleTaskDone(context.Context, int64, bool) error {
	return f.toggleErr
}

func (f *fakeStore) CreateTask(_ context.Context, draft vimkunja.TaskDraft) (vimkunja.Task, error) {
	f.createdDraft = draft
	return vimkunja.Task{ID: 99, ProjectID: draft.ProjectID, Title: draft.Title}, nil
}

func (f *fakeStore) UpdateTask(context.Context, vimkunja.TaskDraft) (vimkunja.Task, error) {
	return vimkunja.Task{}, nil
}

func TestErrorFooterClearsOnNextKey(t *testing.T) {
	m := &model{lastErr: "database is locked", focus: paneProjects}
	key := tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"})

	updated, _ := m.Update(key)
	got := updated.(*model)
	if got.lastErr != "" {
		t.Fatalf("lastErr = %q, want empty after key press", got.lastErr)
	}
}

func testModel() *model {
	return NewModel(Config{Store: &fakeStore{}}).(*model)
}

func typeCommand(t *testing.T, m *model, input string) (*model, tea.Cmd) {
	t.Helper()
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: ':', Text: ":"}))
	m = updated.(*model)
	for _, r := range input {
		updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
		m = updated.(*model)
	}
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	return updated.(*model), cmd
}

func TestToggleFailureDoesNotReloadTasks(t *testing.T) {
	store := &fakeStore{toggleErr: errors.New("write failed")}
	m := &model{
		store:            store,
		perPage:          50,
		focus:            paneTasks,
		currentProjectID: 1,
		currentViewID:    1,
		tasks:            []vimkunja.Task{{ID: 1}},
	}
	key := tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"})

	_, toggleCmd := m.Update(key)
	toggleResult := toggleCmd()
	updated, reloadCmd := m.Update(toggleResult)
	got := updated.(*model)

	if got.lastErr != "write failed" {
		t.Fatalf("lastErr = %q, want write failure", got.lastErr)
	}
	if reloadCmd != nil {
		t.Fatal("failed toggle returned a reload command")
	}
	if store.listTasksCalls != 0 {
		t.Fatalf("ListTasksInView called %d times, want 0", store.listTasksCalls)
	}
}

func TestCommandModeEscapeCancelsInput(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: ':', Text: ":"}))
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m = updated.(*model)

	if m.commandMode || m.commandInput != "" {
		t.Fatalf("command state = mode:%v input:%q, want cancelled", m.commandMode, m.commandInput)
	}
}

func TestCommandQuitClosesDetailPane(t *testing.T) {
	m := testModel()
	m.showDetail = true
	m.focus = paneDetail

	m, cmd := typeCommand(t, m, "q")
	if m.showDetail || m.focus != paneTasks {
		t.Fatalf("detail state = visible:%v focus:%v, want closed task pane", m.showDetail, m.focus)
	}
	if cmd != nil {
		t.Fatal(":q returned quit command while closing detail pane")
	}
}

func TestCommandQuitQuitsWithoutClosablePane(t *testing.T) {
	for _, input := range []string{"q", "quit"} {
		t.Run(input, func(t *testing.T) {
			_, cmd := typeCommand(t, testModel(), input)
			if cmd == nil {
				t.Fatalf(":%s returned no quit command in the main view", input)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf(":%s command returned %T, want tea.QuitMsg", input, cmd())
			}
		})
	}
}

func TestCommandQuitAllQuits(t *testing.T) {
	for _, input := range []string{"qa", "quitall"} {
		t.Run(input, func(t *testing.T) {
			m, cmd := typeCommand(t, testModel(), input)
			if m.commandMode {
				t.Fatal("command mode remained active")
			}
			if cmd == nil {
				t.Fatalf(":%s returned no quit command", input)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf(":%s command returned %T, want tea.QuitMsg", input, cmd())
			}
		})
	}
}

func TestHelpCommandOpensHelpPane(t *testing.T) {
	m, cmd := typeCommand(t, testModel(), "help")
	if cmd != nil {
		t.Fatal(":help returned an unexpected command")
	}
	if m.helpPane == nil || m.helpPane.title != "HELP" || m.focus != paneHelp {
		t.Fatalf("help pane = %#v, focus = %v; want help pane with help focus", m.helpPane, m.focus)
	}
}

func TestLeaderTimeoutUsesCentralDefaultAndConfigOverride(t *testing.T) {
	if m := testModel(); m.leaderTimeout != DefaultLeaderTimeout {
		t.Fatalf("default leader timeout = %v, want %v", m.leaderTimeout, DefaultLeaderTimeout)
	}
	m := NewModel(Config{Store: &fakeStore{}, LeaderTimeout: time.Second}).(*model)
	if m.leaderTimeout != time.Second {
		t.Fatalf("configured leader timeout = %v, want %v", m.leaderTimeout, time.Second)
	}
}

func TestLeaderSpaceOTogglesProjectsPane(t *testing.T) {
	m := testModel()
	m.focus = paneProjects
	space := tea.KeyPressMsg(tea.Key{Code: ' ', Text: " "})
	updated, timeoutCmd := m.Update(space)
	m = updated.(*model)
	if !m.leaderArmed || timeoutCmd == nil {
		t.Fatal("Space did not arm the leader with a timeout")
	}
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'o', Text: "o"}))
	m = updated.(*model)
	if !m.projectsPaneCollapsed || m.focus != paneTasks || m.leaderArmed {
		t.Fatalf("Space+o state = collapsed:%v focus:%v armed:%v", m.projectsPaneCollapsed, m.focus, m.leaderArmed)
	}
}

func TestLeaderTimeoutAndCollapsedLeftFocusAreNoOps(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.projectsPaneCollapsed = true
	m.leaderArmed = true
	updated, _ := m.Update(leaderTimeoutMsg{})
	m = updated.(*model)
	if m.leaderArmed {
		t.Fatal("leader remained armed after timeout")
	}

	ctrlW := tea.KeyPressMsg(tea.Key{Code: 'w', Mod: tea.ModCtrl})
	updated, _ = m.Update(ctrlW)
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'h', Text: "h"}))
	m = updated.(*model)
	if m.focus != paneTasks {
		t.Fatalf("Ctrl+W h focus = %v, want tasks while collapsed", m.focus)
	}
}

func TestQuestionMarkDoesNotOpenHelp(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: '?', Text: "?"}))
	m = updated.(*model)
	if m.helpPane != nil || m.focus != paneProjects {
		t.Fatalf("? state = pane:%#v focus:%v, want no help pane and project focus", m.helpPane, m.focus)
	}
}

func TestHelpPaneEscDoesNotClose(t *testing.T) {
	m, _ := typeCommand(t, testModel(), "h")
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m = updated.(*model)

	if m.helpPane == nil || m.focus != paneHelp {
		t.Fatalf("help pane = %#v, focus = %v; want unchanged help pane", m.helpPane, m.focus)
	}
}

func TestHelpPaneCtrlWNavigation(t *testing.T) {
	m, _ := typeCommand(t, testModel(), "h")
	ctrlW := tea.KeyPressMsg(tea.Key{Code: 'w', Mod: tea.ModCtrl})

	updated, _ := m.Update(ctrlW)
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"}))
	m = updated.(*model)
	if m.focus != paneTasks {
		t.Fatalf("focus after Ctrl+W k = %v, want tasks", m.focus)
	}

	updated, _ = m.Update(ctrlW)
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	m = updated.(*model)
	if m.focus != paneHelp {
		t.Fatalf("focus after Ctrl+W j = %v, want help", m.focus)
	}
}

func TestErrorsCommandShowsSessionLog(t *testing.T) {
	const message = "test error shown by :errors"
	apperrors.Log(message)

	m, cmd := typeCommand(t, testModel(), "errors")
	if cmd != nil {
		t.Fatal(":errors returned an unexpected command")
	}
	if m.infoPane == nil || m.infoPane.title != "ERRORS" {
		t.Fatalf("info pane = %#v, want errors pane", m.infoPane)
	}
	found := false
	for _, line := range m.infoPane.lines {
		if line == message {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("errors pane lines = %#v, missing %q", m.infoPane.lines, message)
	}

	m, cmd = typeCommand(t, m, "foo")
	if cmd != nil {
		t.Fatal("unknown command returned an unexpected command")
	}
	const wantError = "unknown command: foo"
	if m.lastErr != wantError {
		t.Fatalf("lastErr = %q, want %q", m.lastErr, wantError)
	}
	found = false
	for _, line := range m.infoPane.lines {
		if line == wantError {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("errors pane lines = %#v, missing newly logged error %q", m.infoPane.lines, wantError)
	}
}

func TestCommandQuitClosesHelpPane(t *testing.T) {
	m, _ := typeCommand(t, testModel(), "h")
	m, cmd := typeCommand(t, m, "q")
	if cmd != nil {
		t.Fatal(":q returned quit command while closing help pane")
	}
	if m.helpPane != nil || m.focus != paneTasks {
		t.Fatalf("help pane = %#v, focus = %v; want closed task pane", m.helpPane, m.focus)
	}
}

func TestDetailEditorEscapeIsNoOpAndDirtyQuitRefuses(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "Unsaved"

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m = updated.(*model)
	if !m.showDetail || m.focus != paneDetail || m.editor == nil {
		t.Fatal("Esc closed the detail editor; it must be a Normal-mode no-op")
	}

	m, cmd := typeCommand(t, m, "q")
	if cmd != nil || !m.showDetail || m.lastErr != "unsaved changes; use :w or :q!" {
		t.Fatalf(":q state = visible:%v error:%q, want dirty editor retained", m.showDetail, m.lastErr)
	}
}

func TestWriteKeepsSavedEditorOpenAndReloadsTargetProject(t *testing.T) {
	store := &fakeStore{}
	m := NewModel(Config{Store: store}).(*model)
	m.projects = []vimkunja.Project{{ID: 1, Title: "Project"}}
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "Saved task"

	m, cmd := typeCommand(t, m, "w")
	if cmd == nil {
		t.Fatal(":w returned no save command")
	}
	updated, reloadCmd := m.Update(cmd())
	m = updated.(*model)
	if reloadCmd == nil || !m.showDetail || m.focus != paneDetail || m.editor == nil || m.editor.dirty() {
		t.Fatalf(":w state = visible:%v focus:%v editor:%v dirty:%v, want clean open editor and reload", m.showDetail, m.focus, m.editor != nil, m.editor != nil && m.editor.dirty())
	}
	if store.createdDraft.Title != "Saved task" || m.editor.id != 99 {
		t.Fatalf("saved draft/id = %#v/%d, want title and generated id", store.createdDraft, m.editor.id)
	}
}

func TestDueDateMaskInsertsSeparators(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldDueDate
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'i', Text: "i"}))
	m = updated.(*model)
	for _, digit := range "20260801" {
		updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: digit, Text: string(digit)}))
		m = updated.(*model)
	}
	if m.editor.draft.DueDate != "2026-08-01" {
		t.Fatalf("DueDate = %q, want 2026-08-01", m.editor.draft.DueDate)
	}
}

func TestProjectPickerShowsNoMatch(t *testing.T) {
	m := testModel()
	m.projects = []vimkunja.Project{{ID: 1, Title: "Alpha"}}
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldProject

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'i', Text: "i"}))
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'z', Text: "z"}))
	m = updated.(*model)
	if m.editor.mode != editorProjectPicker || !strings.Contains(renderTaskDetail(m, 80, 20), "No match found") {
		t.Fatal("project picker did not render its no-match state")
	}
}

func TestUnknownCommandRecordsError(t *testing.T) {
	m, cmd := typeCommand(t, testModel(), "missing")
	if cmd != nil {
		t.Fatal("unknown command returned a command")
	}
	if m.lastErr != "unknown command: missing" {
		t.Fatalf("lastErr = %q, want unknown-command error", m.lastErr)
	}
}

func pressKey(m *model, key tea.Key) *model {
	updated, _ := m.Update(tea.KeyPressMsg(key))
	return updated.(*model)
}

func ctrlW(m *model) *model {
	return pressKey(m, tea.Key{Code: 'w', Mod: tea.ModCtrl})
}

func TestVimMotionGGAndGSelectListBounds(t *testing.T) {
	t.Run("projects", func(t *testing.T) {
		m := testModel()
		m.projects = []vimkunja.Project{{ID: 1}, {ID: 2}, {ID: 3}}
		m.projectSelected = 1

		m = pressKey(pressKey(m, tea.Key{Code: 'g', Text: "g"}), tea.Key{Code: 'g', Text: "g"})
		if m.projectSelected != 0 {
			t.Fatalf("project selection after gg = %d, want first project", m.projectSelected)
		}

		m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
		if m.projectSelected != 2 {
			t.Fatalf("project selection after G = %d, want last project", m.projectSelected)
		}
	})

	t.Run("tasks", func(t *testing.T) {
		m := testModel()
		m.focus = paneTasks
		m.tasks = []vimkunja.Task{{ID: 1}, {ID: 2}, {ID: 3}}
		m.taskSelected = 1

		m = pressKey(pressKey(m, tea.Key{Code: 'g', Text: "g"}), tea.Key{Code: 'g', Text: "g"})
		if m.taskSelected != 0 {
			t.Fatalf("task selection after gg = %d, want first task", m.taskSelected)
		}

		m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
		if m.taskSelected != 2 {
			t.Fatalf("task selection after G = %d, want last task", m.taskSelected)
		}
	})
}

func TestPaneFocusCtrlWCyclesProjectsAndTasks(t *testing.T) {
	m := testModel()

	m = ctrlW(m)
	m = ctrlW(m)
	if m.focus != paneTasks {
		t.Fatalf("focus after Ctrl+W Ctrl+W = %v, want tasks", m.focus)
	}

	m = ctrlW(m)
	m = ctrlW(m)
	if m.focus != paneProjects {
		t.Fatalf("focus after second Ctrl+W Ctrl+W = %v, want projects", m.focus)
	}
}

func TestPaneFocusCtrlWDirectionalNavigation(t *testing.T) {
	m := testModel()

	m = ctrlW(m)
	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	if m.focus != paneTasks {
		t.Fatalf("focus after Ctrl+W l = %v, want tasks", m.focus)
	}

	m = ctrlW(m)
	m = pressKey(m, tea.Key{Code: 'h', Text: "h"})
	if m.focus != paneProjects {
		t.Fatalf("focus after Ctrl+W h = %v, want projects", m.focus)
	}

	m.focus = paneTasks
	m.showDetail = true
	m = ctrlW(m)
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.focus != paneDetail {
		t.Fatalf("focus after Ctrl+W j = %v, want detail", m.focus)
	}

	m = ctrlW(m)
	m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
	if m.focus != paneTasks {
		t.Fatalf("focus after Ctrl+W k = %v, want tasks", m.focus)
	}
}

func TestPaneFocusCtrlWIgnoresLeaderKeyRelease(t *testing.T) {
	m := testModel()

	m = ctrlW(m)
	updated, _ := m.Update(tea.KeyReleaseMsg(tea.Key{Code: 'w', Mod: tea.ModCtrl}))
	m = updated.(*model)
	m = pressKey(m, tea.Key{Code: 'h', Text: "h"})

	if m.focus != paneProjects {
		t.Fatalf("focus after Ctrl+W release then h = %v, want projects", m.focus)
	}
}

func TestInsertFooterShowsOnlyAvailableActions(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})

	footer := m.renderStatusBar(120)
	for _, unavailable := range []string{"i edit", ":w save", ":q close"} {
		if strings.Contains(footer, unavailable) {
			t.Fatalf("Insert footer contains unavailable action %q: %q", unavailable, footer)
		}
	}
	if !strings.Contains(footer, "Esc normal") {
		t.Fatalf("Insert footer = %q, want Esc normal hint", footer)
	}
}

func TestNavigationKeepsValidStateAtUnavailableBoundaries(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.projectsPaneCollapsed = true

	m = ctrlW(m)
	m = pressKey(m, tea.Key{Code: 'h', Text: "h"})
	if m.focus != paneTasks {
		t.Fatalf("focus after Ctrl+W h with collapsed projects = %v, want tasks", m.focus)
	}

	m = ctrlW(m)
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.focus != paneTasks {
		t.Fatalf("focus after Ctrl+W j without a lower pane = %v, want tasks", m.focus)
	}

	m.tasks = nil
	m.taskSelected = 0
	m = pressKey(pressKey(m, tea.Key{Code: 'g', Text: "g"}), tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
	if m.taskSelected != 0 {
		t.Fatalf("empty task selection = %d, want stable zero", m.taskSelected)
	}
}
