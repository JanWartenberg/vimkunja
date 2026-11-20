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

type projectDeletionStore struct {
	*fakeStore
	deleted                         []int64
	checkErr, deleteErr, refreshErr error
}

func (s *projectDeletionStore) CheckProjectDeletion(context.Context, int64) error { return s.checkErr }

func (s *projectDeletionStore) DeleteEmptyProject(_ context.Context, id int64) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	for i, p := range s.projects {
		if p.ID == id {
			s.projects = append(s.projects[:i], s.projects[i+1:]...)
			break
		}
	}
	return nil
}
func (s *projectDeletionStore) ListProjects(context.Context) ([]vimkunja.Project, error) {
	return s.projects, s.refreshErr
}

func deletionModel() (*model, *projectDeletionStore) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	s := &projectDeletionStore{fakeStore: &fakeStore{projects: []vimkunja.Project{{ID: 1, Title: "Delete me"}, {ID: 2, Title: "Keep"}}}}
	s.viewsByProject = map[int64][]vimkunja.ProjectView{2: {{ID: 20, ViewKind: "list"}}}
	m.store, m.projects = s, append([]vimkunja.Project(nil), s.projects...)
	m.projectTree = newProjectTree(m.projects, nil)
	m.currentProjectID = 1
	m.focus = paneProjects
	return m, s
}

func TestProjectDeletionNeedsOneConfirmationAndRefreshes(t *testing.T) {
	m, s := deletionModel()
	_, cmd := typeCommand(t, m, "projectdelete")
	if len(s.deleted) != 0 {
		t.Fatal("command deleted without confirmation")
	}
	deliverProjectCommand(t, m, cmd)
	text := ansi.Strip(m.Render())
	if !strings.Contains(text, "Delete me") || !strings.Contains(text, "y delete") {
		t.Fatalf("confirmation missing: %q", text)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if len(s.deleted) != 0 {
		t.Fatal("Enter accidentally confirmed deletion")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.projectDeletion != nil || len(s.deleted) != 0 {
		t.Fatal("cancel deleted Project")
	}
	_, cmd = typeCommand(t, m, "projectdelete")
	deliverProjectCommand(t, m, cmd)
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	_, duplicate := m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	if duplicate != nil {
		t.Fatal("repeated confirmation submitted duplicate mutation")
	}
	deliverProjectCommand(t, m, cmd)
	if len(s.deleted) != 1 || s.deleted[0] != 1 || len(m.projects) != 1 || m.currentProjectID != 2 || m.focus != paneProjects || m.projectDeletion != nil {
		t.Fatalf("deletion context: deleted=%v projects=%v current=%d", s.deleted, m.projects, m.currentProjectID)
	}
}

func TestProjectDeletionErrorAndDraftProtection(t *testing.T) {
	m, s := deletionModel()
	s.deleteErr = fmt.Errorf("Project contains Tasks")
	_, cmd := typeCommand(t, m, "projectdelete")
	deliverProjectCommand(t, m, cmd)
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	deliverProjectCommand(t, m, cmd)
	if !strings.Contains(ansi.Strip(m.Render()), "Project contains Tasks") || len(m.projects) != 2 {
		t.Fatalf("failure lost Project or error: %q", m.Render())
	}
	m = pressKey(m, tea.Key{Code: 'n', Text: "n"})
	m.editor = &taskEditor{draft: vimkunja.TaskDraft{Title: "Unsaved"}}
	_, cmd = typeCommand(t, m, "projectdelete")
	if cmd != nil || m.projectDeletion != nil || !strings.Contains(m.lastErr, "unsaved") {
		t.Fatalf("draft not protected: %q", m.lastErr)
	}
}

func TestProjectDeletionHandlesLastProjectAndReloadFailure(t *testing.T) {
	for _, last := range []bool{false, true} {
		m, s := deletionModel()
		if last {
			s.projects = s.projects[:1]
			m.projects = m.projects[:1]
			m.projectTree = newProjectTree(m.projects, nil)
		} else {
			s.refreshErr = fmt.Errorf("reload failed")
		}
		_, cmd := typeCommand(t, m, "projectdelete")
		deliverProjectCommand(t, m, cmd)
		_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
		deliverProjectCommand(t, m, cmd)
		if m.projectDeletion != nil {
			t.Fatal("successful deletion stayed open for retry")
		}
		for _, p := range m.projects {
			if p.ID == 1 {
				t.Fatal("deleted Project remained selectable")
			}
		}
		if last && (len(m.projects) != 0 || m.currentProjectID != 0 || m.currentViewID != 0) {
			t.Fatal("last Project left stale context")
		}
		if !last && !strings.Contains(m.lastErr, "Project deleted") {
			t.Fatalf("reload failure not reported: %q", m.lastErr)
		}
	}
}

func TestProjectDeletionFitsShortTerminals(t *testing.T) {
	for _, size := range [][2]int{{60, 15}, {32, 9}, {20, 6}, {8, 2}, {1, 1}} {
		m, _ := deletionModel()
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		_, cmd := typeCommand(t, m, "projectdelete")
		deliverProjectCommand(t, m, cmd)
		text := ansi.Strip(m.View().Content)
		if len(strings.Split(text, "\n")) > size[1] {
			t.Fatalf("%v: too many rows", size)
		}
		for _, line := range strings.Split(text, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("%v: too wide: %q", size, line)
			}
		}
		if m.View().Cursor != nil {
			t.Fatal("confirmation has a hidden input cursor")
		}
	}
}

func TestProjectDeletionShowsCheckedEligibilityAndBlocksPrematureConfirmation(t *testing.T) {
	for _, blocker := range []error{nil, fmt.Errorf("Project contains Tasks (including trash)"), fmt.Errorf("Project contains Subprojects"), fmt.Errorf("database read failed")} {
		m, s := deletionModel()
		s.checkErr = blocker
		_, check := typeCommand(t, m, "projectdelete")
		text := ansi.Strip(m.Render())
		if !strings.Contains(text, "Checking whether") || strings.Contains(text, "y delete") {
			t.Fatalf("unchecked modal claims eligibility: %q", text)
		}
		_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
		if cmd != nil || len(s.deleted) != 0 {
			t.Fatal("deleted before check completed")
		}
		deliverProjectCommand(t, m, check)
		text = ansi.Strip(m.Render())
		if !strings.Contains(text, "DELETE PROJECT") || !strings.Contains(text, `Delete Project #1 "Delete me" permanently?`) {
			t.Fatalf("target missing after check: %q", text)
		}
		if blocker == nil {
			if !strings.Contains(text, "This has no Tasks or Subprojects.") || !strings.Contains(text, "y delete") {
				t.Fatalf("eligible state unclear: %q", text)
			}
		} else {
			if !strings.Contains(text, blocker.Error()) || !strings.Contains(text, "Cannot be deleted") || strings.Contains(text, "y delete") {
				t.Fatalf("blocker unclear: %q", text)
			}
			_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
			if cmd != nil || len(s.deleted) != 0 {
				t.Fatal("blocked modal submitted deletion")
			}
			s.checkErr = nil
			_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"}))
			deliverProjectCommand(t, m, cmd)
			if !strings.Contains(ansi.Strip(m.Render()), "y delete") {
				t.Fatal("recheck did not reveal newly eligible state")
			}
		}
	}
}

func TestProjectDeletionLateCheckCannotReopenCancelledModal(t *testing.T) {
	m, _ := deletionModel()
	_, late := typeCommand(t, m, "projectdelete")
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	_, current := typeCommand(t, m, "projectdelete")
	deliverProjectCommand(t, m, late)
	if !strings.Contains(ansi.Strip(m.Render()), "Checking whether") {
		t.Fatal("old check changed new modal")
	}
	deliverProjectCommand(t, m, current)
	if !strings.Contains(ansi.Strip(m.Render()), "This has no Tasks or Subprojects.") {
		t.Fatal("current check was lost")
	}
}
