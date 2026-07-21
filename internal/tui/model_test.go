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
	toggleErr         error
	toggleTaskID      int64
	toggleDone        bool
	workflowStatus    vimkunja.WorkflowStatus
	workflowStatusErr error
	updateErr         error
	listTasksCalls    int
	createdDraft      vimkunja.TaskDraft
	updatedDraft      vimkunja.TaskDraft
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

func (f *fakeStore) GetTaskWorkflowStatus(context.Context, int64, int64) (vimkunja.WorkflowStatus, error) {
	return f.workflowStatus, f.workflowStatusErr
}

func (f *fakeStore) SetTaskWorkflowStatus(context.Context, int64, int64) error {
	return nil
}

func (f *fakeStore) ToggleTaskDone(_ context.Context, taskID int64, done bool) error {
	f.toggleTaskID = taskID
	f.toggleDone = done
	return f.toggleErr
}

func (f *fakeStore) CreateTask(_ context.Context, draft vimkunja.TaskDraft) (vimkunja.Task, error) {
	f.createdDraft = draft
	return vimkunja.Task{ID: 99, ProjectID: draft.ProjectID, Title: draft.Title}, nil
}

func (f *fakeStore) UpdateTask(_ context.Context, draft vimkunja.TaskDraft) (vimkunja.Task, error) {
	f.updatedDraft = draft
	if f.updateErr != nil {
		return vimkunja.Task{}, f.updateErr
	}
	return vimkunja.Task{ID: draft.ID, ProjectID: draft.ProjectID, Title: draft.Title}, nil
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

func TestOpeningTaskDetailLoadsWorkflowStatus(t *testing.T) {
	store := &fakeStore{workflowStatus: vimkunja.WorkflowStatus{
		Availability:  vimkunja.WorkflowStatusAvailable,
		CurrentBucket: vimkunja.WorkflowBucket{ID: 11, Title: "Doing"},
		Buckets: []vimkunja.WorkflowBucket{
			{ID: 10, Title: "To-Do"}, {ID: 11, Title: "Doing"}, {ID: 12, Title: "Done"},
		},
	}}
	m := &model{
		store:            store,
		focus:            paneTasks,
		currentProjectID: 1,
		tasks:            []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "Task"}},
	}

	updated, loadStatus := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if !m.showDetail || loadStatus == nil {
		t.Fatalf("opening detail = visible:%v command:%v, want detail and status load", m.showDetail, loadStatus != nil)
	}
	updated, _ = m.Update(loadStatus())
	m = updated.(*model)

	if m.editor.workflowStatusLoading {
		t.Fatal("workflow status remains loading after Store response")
	}
	if m.editor.workflowStatus.CurrentBucket.Title != "Doing" {
		t.Fatalf("current workflow status = %#v, want Doing", m.editor.workflowStatus)
	}
	if got, want := m.editor.workflowStatus.Buckets[0].Title, "To-Do"; got != want {
		t.Fatalf("first workflow bucket = %q, want %q", got, want)
	}
}

func TestWorkflowStatusPickerSelectsBucketAndMarksEditorDirty(t *testing.T) {
	m := &model{focus: paneDetail, editor: &taskEditor{
		field: fieldWorkflowStatus,
		workflowStatus: vimkunja.WorkflowStatus{
			Availability:  vimkunja.WorkflowStatusAvailable,
			CurrentBucket: vimkunja.WorkflowBucket{ID: 10, Title: "To-Do"},
			Buckets: []vimkunja.WorkflowBucket{
				{ID: 10, Title: "To-Do"}, {ID: 11, Title: "Doing"},
			},
		},
	}}

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'i', Text: "i"}))
	m = updated.(*model)
	if m.editor.mode != editorWorkflowStatusPicker {
		t.Fatalf("editor mode = %v, want workflow status picker", m.editor.mode)
	}
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)

	if m.editor.draft.WorkflowBucketID != 11 || !m.editor.dirty() {
		t.Fatalf("selected workflow status = bucket:%d dirty:%v, want Doing and dirty", m.editor.draft.WorkflowBucketID, m.editor.dirty())
	}
}

func TestProjectPickerClearsWorkflowBucketFromPreviousProject(t *testing.T) {
	m := testModel()
	m.projects = []vimkunja.Project{{ID: 1, Title: "Misc"}, {ID: 2, Title: "Sonstiges"}}
	m.editor = &taskEditor{
		id:       1,
		field:    fieldProject,
		draft:    vimkunja.TaskDraft{ID: 1, ProjectID: 1, Title: "dummy task", WorkflowBucketID: 10},
		original: vimkunja.TaskDraft{ID: 1, ProjectID: 1, Title: "dummy task", WorkflowBucketID: 10},
	}
	m.focus = paneDetail

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})

	if m.editor.draft.ProjectID != 2 {
		t.Fatalf("selected project = %d, want 2", m.editor.draft.ProjectID)
	}
	if m.editor.draft.WorkflowBucketID != 0 {
		t.Fatalf("workflow bucket after project move = %d, want cleared", m.editor.draft.WorkflowBucketID)
	}
}

func TestWorkflowStatusPickerEscapeKeepsPreviousSelection(t *testing.T) {
	m := &model{focus: paneDetail, editor: &taskEditor{
		field: fieldWorkflowStatus,
		draft: vimkunja.TaskDraft{WorkflowBucketID: 10},
		workflowStatus: vimkunja.WorkflowStatus{
			Availability: vimkunja.WorkflowStatusAvailable,
			Buckets: []vimkunja.WorkflowBucket{
				{ID: 10, Title: "To-Do"}, {ID: 11, Title: "Doing"},
			},
		},
	}}

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'i', Text: "i"}))
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m = updated.(*model)

	if m.editor.mode != editorNormal || m.editor.draft.WorkflowBucketID != 10 {
		t.Fatalf("cancelled picker = mode:%v bucket:%d, want normal To-Do", m.editor.mode, m.editor.draft.WorkflowBucketID)
	}
}

func TestSavingDetailPersistsSelectedWorkflowStatus(t *testing.T) {
	store := &fakeStore{}
	m := NewModel(Config{Store: store}).(*model)
	m.editor = &taskEditor{
		id:       1,
		draft:    vimkunja.TaskDraft{ID: 1, ProjectID: 1, Title: "Task", WorkflowBucketID: 11},
		original: vimkunja.TaskDraft{ID: 1, ProjectID: 1, Title: "Task", WorkflowBucketID: 10},
	}
	m.showDetail = true
	m.focus = paneDetail

	m, saveCmd := typeCommand(t, m, "w")
	if saveCmd == nil {
		t.Fatal(":w returned no save command")
	}
	updated, _ := m.Update(saveCmd())
	m = updated.(*model)

	if store.updatedDraft.WorkflowBucketID != 11 {
		t.Fatalf("saved workflow bucket = %d, want 11", store.updatedDraft.WorkflowBucketID)
	}
	if m.editor == nil || m.editor.dirty() {
		t.Fatal("saved editor did not become clean")
	}
}

func TestWorkflowStatusSaveFailureKeepsEditorDirty(t *testing.T) {
	store := &fakeStore{updateErr: errors.New("write failed")}
	m := NewModel(Config{Store: store}).(*model)
	m.editor = &taskEditor{
		id:       1,
		draft:    vimkunja.TaskDraft{ID: 1, ProjectID: 1, Title: "Task", WorkflowBucketID: 11},
		original: vimkunja.TaskDraft{ID: 1, ProjectID: 1, Title: "Task", WorkflowBucketID: 10},
	}
	m.showDetail = true
	m.focus = paneDetail

	m, saveCmd := typeCommand(t, m, "w")
	updated, reloadCmd := m.Update(saveCmd())
	m = updated.(*model)

	if !m.editor.dirty() || m.lastErr != "write failed" || reloadCmd != nil {
		t.Fatalf("failed status save = dirty:%v error:%q reload:%v", m.editor.dirty(), m.lastErr, reloadCmd != nil)
	}
}

func TestTaskToggleUsesDoneStateToSelectWorkflowDirection(t *testing.T) {
	for _, tc := range []struct {
		name string
		task vimkunja.Task
		want bool
	}{
		{name: "open moves to done", task: vimkunja.Task{ID: 1}, want: true},
		{name: "done moves to default", task: vimkunja.Task{ID: 2, Done: true}, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{}
			m := &model{store: store, focus: paneTasks, tasks: []vimkunja.Task{tc.task}}

			_, toggleCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
			toggleCmd()
			if store.toggleTaskID != tc.task.ID || store.toggleDone != tc.want {
				t.Fatalf("toggle = task:%d done:%v, want task:%d done:%v", store.toggleTaskID, store.toggleDone, tc.task.ID, tc.want)
			}
		})
	}
}

func TestToggleSuccessReloadsTasks(t *testing.T) {
	store := &fakeStore{}
	m := &model{
		store:            store,
		perPage:          50,
		focus:            paneTasks,
		currentProjectID: 1,
		currentViewID:    1,
		tasks:            []vimkunja.Task{{ID: 1}},
	}

	_, toggleCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
	updated, reloadCmd := m.Update(toggleCmd())
	m = updated.(*model)
	if reloadCmd == nil {
		t.Fatal("successful toggle returned no reload command")
	}
	reloadCmd()
	if store.listTasksCalls != 1 {
		t.Fatalf("ListTasksInView called %d times, want 1", store.listTasksCalls)
	}
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

func TestCommandForceQuitAllIgnoresUnsavedDetailChanges(t *testing.T) {
	for _, input := range []string{"qa!", "quitall!"} {
		t.Run(input, func(t *testing.T) {
			m := testModel()
			m.currentProjectID = 1
			m.openNewEditor()
			m.editor.draft.Title = "Unsaved"

			_, cmd := typeCommand(t, m, input)
			if cmd == nil {
				t.Fatalf(":%s returned no quit command", input)
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

func TestCtrlWTimeoutUsesCentralDefaultAndConfigOverride(t *testing.T) {
	if m := testModel(); m.ctrlWTimeout != DefaultCtrlWTimeout {
		t.Fatalf("default Ctrl+W timeout = %v, want %v", m.ctrlWTimeout, DefaultCtrlWTimeout)
	}
	m := NewModel(Config{Store: &fakeStore{}, CtrlWTimeout: time.Second}).(*model)
	if m.ctrlWTimeout != time.Second {
		t.Fatalf("configured Ctrl+W timeout = %v, want %v", m.ctrlWTimeout, time.Second)
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

func TestWriteQuitAllSavesTaskThenQuits(t *testing.T) {
	store := &fakeStore{}
	m := NewModel(Config{Store: store}).(*model)
	m.projects = []vimkunja.Project{{ID: 1, Title: "Project"}}
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "Saved before quit"

	m, saveCmd := typeCommand(t, m, "wqa")
	if saveCmd == nil {
		t.Fatal(":wqa returned no save command")
	}

	updated, quitCmd := m.Update(saveCmd())
	m = updated.(*model)
	if store.createdDraft.Title != "Saved before quit" {
		t.Fatalf("saved draft = %#v, want task saved before quit", store.createdDraft)
	}
	if quitCmd == nil {
		t.Fatal(":wqa did not return a quit command after saving")
	}
	if _, ok := quitCmd().(tea.QuitMsg); !ok {
		t.Fatalf(":wqa command returned %T, want tea.QuitMsg", quitCmd())
	}
	if m.quitAfterSave {
		t.Fatal(":wqa left quit-after-save armed")
	}
}

func TestWriteQuitAllQuitsImmediatelyWithoutTaskDetail(t *testing.T) {
	_, quitCmd := typeCommand(t, testModel(), "wqa")
	if quitCmd == nil {
		t.Fatal(":wqa returned no quit command without a task detail pane")
	}
	if _, ok := quitCmd().(tea.QuitMsg); !ok {
		t.Fatalf(":wqa command returned %T, want tea.QuitMsg", quitCmd())
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

func TestHomeEndAliasesScrollInformationalPanes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		model  *model
		offset func(*model) int
	}{
		{
			name:   "help",
			model:  &model{focus: paneHelp, helpPane: &infoPane{lines: []string{"first", "middle", "last"}, offset: 1}},
			offset: func(m *model) int { return m.helpPane.offset },
		},
		{
			name:   "errors",
			model:  &model{focus: paneInfo, infoPane: &infoPane{lines: []string{"first", "middle", "last"}, offset: 1}},
			offset: func(m *model) int { return m.infoPane.offset },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pressKey(tc.model, tea.Key{Code: tea.KeyEnd})
			if got := tc.offset(m); got != 2 {
				t.Fatalf("offset after End = %d, want 2", got)
			}

			m = pressKey(m, tea.Key{Code: tea.KeyHome})
			if got := tc.offset(m); got != 0 {
				t.Fatalf("offset after Home = %d, want 0", got)
			}
		})
	}
}

func TestHelpDocumentsHomeEndAliases(t *testing.T) {
	if !strings.Contains(strings.Join(helpLines, "\n"), "Home / End") {
		t.Fatal("help text does not document Home / End aliases")
	}
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

func TestPaneFocusCtrlWCyclesAllOpenPanes(t *testing.T) {
	m := testModel()
	m.showDetail = true

	m = ctrlW(m)
	m = ctrlW(m)
	if m.focus != paneTasks {
		t.Fatalf("focus after Ctrl+W Ctrl+W = %v, want tasks", m.focus)
	}

	m = ctrlW(m)
	m = pressKey(m, tea.Key{Code: 'w', Text: "w"})
	if m.focus != paneDetail {
		t.Fatalf("focus after Ctrl+W w = %v, want detail", m.focus)
	}

	m = ctrlW(m)
	m = ctrlW(m)
	if m.focus != paneProjects {
		t.Fatalf("focus after wrapped Ctrl+W Ctrl+W = %v, want projects", m.focus)
	}
}

func TestCtrlWTimeoutDisarmsAndShowsFooterState(t *testing.T) {
	m := testModel()
	m = ctrlW(m)
	if !m.ctrlWArmed {
		t.Fatal("Ctrl+W did not arm pane navigation")
	}
	if footer := m.renderStatusBar(120); !strings.Contains(footer, "Ctrl-W") || !strings.Contains(footer, "38;5;227") {
		t.Fatalf("armed Ctrl+W footer = %q, want bright-yellow Ctrl-W hint", footer)
	}

	updated, _ := m.Update(ctrlWTimeoutMsg{})
	m = updated.(*model)
	if m.ctrlWArmed {
		t.Fatal("Ctrl+W remained armed after timeout")
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

func TestPaneFocusCtrlWKLeavesOpenTaskDetail(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()

	m = ctrlW(m)
	m = pressKey(m, tea.Key{Code: 'k', Text: "k"})

	if m.focus != paneTasks {
		t.Fatalf("focus after Ctrl+W k from detail = %v, want tasks", m.focus)
	}
}

func TestUnfocusedTaskDetailHasNoHighlightedField(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldDescription
	m.focus = paneProjects

	if detail := renderTaskDetail(m, 80, 20); strings.Contains(detail, "▸") {
		t.Fatalf("unfocused detail contains a selected field: %q", detail)
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

func TestEditorViewUsesCursorShapeForCurrentMode(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()

	view := m.View()
	if view.Cursor == nil || view.Cursor.Shape != tea.CursorBlock {
		t.Fatalf("normal editor cursor = %#v, want block cursor", view.Cursor)
	}

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	view = m.View()
	if view.Cursor == nil || view.Cursor.Shape != tea.CursorBar {
		t.Fatalf("insert editor cursor = %#v, want bar cursor", view.Cursor)
	}

	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	m.editor.field = fieldPriority
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	view = m.View()
	if view.Cursor == nil || view.Cursor.Shape != tea.CursorBlock {
		t.Fatalf("picker cursor = %#v, want block cursor", view.Cursor)
	}
}

func TestEditorViewOmitsNativeCursorWhenDisabled(t *testing.T) {
	m := NewModel(Config{Store: &fakeStore{}, DisableCursorShapes: true}).(*model)
	m.currentProjectID = 1
	m.openNewEditor()
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})

	if view := m.View(); view.Cursor != nil {
		t.Fatalf("disabled cursor shapes produced cursor %#v", view.Cursor)
	}
}

func TestPickerHighlightsOnlyTheFieldValue(t *testing.T) {
	tests := []struct {
		name  string
		field editorField
		mode  editorMode
		label string
		value string
	}{
		{"project", fieldProject, editorProjectPicker, "Project:  ", "(none)"},
		{"priority", fieldPriority, editorPriorityPicker, "Priority: ", "None"},
		{"workflow status", fieldWorkflowStatus, editorWorkflowStatusPicker, "Status:   ", "unassigned"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.currentProjectID = 1
			m.openNewEditor()
			m.editor.field, m.editor.mode = tc.field, tc.mode

			detail := renderTaskDetail(m, 80, 20)
			highlightedValue := tc.label + "\x1b[48;5;205m" + tc.value + "\x1b[m"
			if !strings.Contains(detail, highlightedValue) {
				t.Fatalf("picker value lacks highlight: %q", detail)
			}
			if strings.Contains(detail, "\x1b[48;5;205m"+tc.label) {
				t.Fatalf("picker label is highlighted: %q", detail)
			}
		})
	}
}

func TestTitleHorizontalMotionsDoNotConsumeFieldNavigation(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldTitle

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.editor.field != fieldPriority {
		t.Fatalf("field after j from title = %v, want priority", m.editor.field)
	}
	m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
	if m.editor.field != fieldTitle {
		t.Fatalf("field after k from priority = %v, want title", m.editor.field)
	}
}

func TestTitleNormalModeUsesHorizontalCursorPositionForInsert(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "abc"

	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: 'X', Text: "X"})

	if got := m.editor.draft.Title; got != "aXbc" {
		t.Fatalf("title after l then insert = %q, want %q", got, "aXbc")
	}
}

func TestNativeTitleCursorAccountsForFocusedFieldPrefix(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 40
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "abc"
	m.editor.cursor = 1
	m.editor.mode = editorInsert

	cursor := m.View().Cursor
	if cursor == nil || cursor.Position != (tea.Position{X: 15, Y: 22}) {
		t.Fatalf("title cursor = %#v, want text position (15,22)", cursor)
	}
}

func TestNativeDescriptionCursorUsesTextPositionWithoutRenderedDuplicate(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 40
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldDescription
	m.editor.draft.Description = "hello"
	m.editor.cursor = 2
	m.editor.mode = editorInsert
	m.cursorVisible = true

	detail := renderTaskDetail(m, 96, 17)
	if strings.Contains(detail, "\x1b[7m") {
		t.Fatalf("native description cursor still has rendered duplicate: %q", detail)
	}
	cursor := m.View().Cursor
	if cursor == nil || cursor.Position != (tea.Position{X: 4, Y: 29}) {
		t.Fatalf("description cursor = %#v, want text position (4,29)", cursor)
	}
}

func TestDisabledDescriptionCursorRetainsRenderedFallback(t *testing.T) {
	m := NewModel(Config{Store: &fakeStore{}, DisableCursorShapes: true}).(*model)
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldDescription
	m.editor.draft.Description = "hello"
	m.editor.cursor = 2
	m.editor.mode = editorInsert
	m.cursorVisible = true

	if detail := renderTaskDetail(m, 80, 20); !strings.Contains(detail, "\x1b[7m") {
		t.Fatalf("disabled cursor shapes lost rendered fallback: %q", detail)
	}
}

func TestClosingEditorRemovesNativeCursor(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	if m.View().Cursor == nil {
		t.Fatal("open editor has no native cursor")
	}

	m.editor = nil
	m.showDetail = false
	m.focus = paneTasks
	if m.View().Cursor != nil {
		t.Fatalf("closed editor cursor = %#v, want no cursor", m.View().Cursor)
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
