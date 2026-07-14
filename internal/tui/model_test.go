package tui

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/vimkunja/vimkunja/internal/errors"
	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

type fakeStore struct {
	toggleErr      error
	listTasksCalls int
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

func TestUnknownCommandRecordsError(t *testing.T) {
	m, cmd := typeCommand(t, testModel(), "missing")
	if cmd != nil {
		t.Fatal("unknown command returned a command")
	}
	if m.lastErr != "unknown command: missing" {
		t.Fatalf("lastErr = %q, want unknown-command error", m.lastErr)
	}
}
