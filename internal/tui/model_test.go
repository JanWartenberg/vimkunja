package tui

import (
	"context"
	"errors"
	"testing"

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
