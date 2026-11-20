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

func TestProjectCreationCommandsAndPaneShortcut(t *testing.T) {
	for _, command := range []string{"projectnew", "projectnewchild"} {
		m := testModel()
		m.width, m.height = 100, 30
		m.projects = []vimkunja.Project{{ID: 1, Title: "Parent"}}
		m.projectTree = newProjectTree(m.projects, nil)
		m.store.(*fakeStore).projects = m.projects
		m, cmd := typeCommand(t, m, command)
		if cmd != nil {
			m.Update(cmd())
		}
		rendered := ansi.Strip(m.Render())
		if !strings.Contains(rendered, "NEW PROJECT") || !strings.Contains(rendered, "Title") {
			t.Fatalf("dialog missing for %s: %q", command, rendered)
		}
		if m.View().Cursor != nil {
			t.Fatal("new Project must start in Normal mode")
		}
		m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
		if m.View().Cursor == nil || m.View().Cursor.Shape != tea.CursorBar {
			t.Fatal("Title must use shared Insert cursor")
		}
		m = pressKey(m, tea.Key{Code: tea.KeyEscape})
		m, _ = typeCommand(t, m, "q!")
		if strings.Contains(ansi.Strip(m.Render()), "NEW PROJECT") {
			t.Fatal(":q! must discard dialog")
		}
	}
	m := testModel()
	m.width, m.height = 100, 30
	m.focus = paneProjects
	m = pressKey(m, tea.Key{Code: 'O', Text: "O"})
	if !strings.Contains(ansi.Strip(m.Render()), "NEW PROJECT") {
		t.Fatal("Projects-Pane O did not open dialog")
	}
	m, _ = typeCommand(t, m, "q!")
	m.focus = paneTasks
	m = pressKey(m, tea.Key{Code: 'O', Text: "O"})
	if strings.Contains(ansi.Strip(m.Render()), "NEW PROJECT") {
		t.Fatal("O outside Projects Pane opened dialog")
	}
}

type projectDialogStore struct {
	*fakeStore
	createErr error
}

func (s *projectDialogStore) CreateProject(_ context.Context, d vimkunja.ProjectDraft) (vimkunja.Project, error) {
	if s.createErr != nil {
		return vimkunja.Project{}, s.createErr
	}
	p := vimkunja.Project{ID: 50, Title: strings.TrimSpace(d.Title), ParentProjectID: d.ParentProjectID}
	s.projects = append(s.projects, p)
	s.viewsByProject = map[int64][]vimkunja.ProjectView{50: {{ID: 501, ViewKind: "list"}}}
	return p, nil
}
func deliverProjectCommand(t *testing.T, m *model, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, cmd := range batch {
			deliverProjectCommand(t, m, cmd)
		}
		return
	}
	_, next := m.Update(msg)
	deliverProjectCommand(t, m, next)
}
func TestProjectCreationSavesOnlyExplicitlyAndPreservesFailures(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	store := &projectDialogStore{fakeStore: &fakeStore{projects: []vimkunja.Project{{ID: 1, Title: "Parent"}}}}
	m.store = store
	m.projects = store.projects
	m.projectTree = newProjectTree(m.projects, nil)
	m, cmd := typeCommand(t, m, "projectnew")
	deliverProjectCommand(t, m, cmd)
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	for _, r := range "New Project" {
		m = pressKey(m, tea.Key{Code: r, Text: string(r)})
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if len(store.projects) != 1 {
		t.Fatal("typing persisted a Project")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	store.createErr = &vimkunja.ProjectCreationError{Field: "title", Message: "Duplicate Title"}
	_, cmd = typeCommand(t, m, "w")
	deliverProjectCommand(t, m, cmd)
	text := ansi.Strip(m.Render())
	if !strings.Contains(text, "Duplicate Title") || !strings.Contains(text, "New Project") {
		t.Fatalf("failed creation lost input/error: %q", text)
	}
	store.createErr = nil
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	_, cmd = typeCommand(t, m, "w")
	deliverProjectCommand(t, m, cmd)
	if len(store.projects) != 2 || store.projects[1].ParentProjectID != 0 {
		t.Fatalf("root creation: %#v", store.projects)
	}
	if strings.Contains(ansi.Strip(m.Render()), "NEW PROJECT") || m.focus != paneTasks || m.currentProjectID != 50 || m.currentViewID != 501 {
		t.Fatalf("creation did not activate List context: %q", m.Render())
	}
}

func TestProjectCreationParentPickerAndDraftProtection(t *testing.T) {
	for _, child := range []bool{false, true} {
		m := testModel()
		m.width, m.height = 100, 30
		store := &projectDialogStore{fakeStore: &fakeStore{projects: []vimkunja.Project{{ID: 1, Title: "Alpha"}, {ID: 2, Title: "Beta"}}}}
		m.store = store
		m.projects = store.projects
		m.projectTree = newProjectTree(m.projects, nil)
		command := "projectnew"
		if child {
			command = "projectnewchild"
		}
		m, cmd := typeCommand(t, m, command)
		deliverProjectCommand(t, m, cmd)
		text := ansi.Strip(m.Render())
		want := "No parent Project"
		if child {
			want = "Alpha (#1)"
		}
		if !strings.Contains(text, want) {
			t.Fatalf("Parent default: %q", text)
		}
		m = enterProjectTitle(m, "New")
		m = pressKey(m, tea.Key{Code: tea.KeyTab})
		m = pressKey(m, tea.Key{Code: tea.KeyEnter})
		m = pressKey(m, tea.Key{Code: '/', Text: "/"})
		m = pressKey(m, tea.Key{Code: 'B', Text: "Beta"})
		m = pressKey(m, tea.Key{Code: tea.KeyEnter})
		m = pressKey(m, tea.Key{Code: tea.KeyDown})
		m = pressKey(m, tea.Key{Code: tea.KeyEnter})
		m = pressKey(m, tea.Key{Code: tea.KeyTab})
		_, cmd = typeCommand(t, m, "w")
		deliverProjectCommand(t, m, cmd)
		if len(store.projects) != 3 || store.projects[2].ParentProjectID != 2 {
			t.Fatalf("explicit Parent ignored: %#v", store.projects)
		}
	}
	m := testModel()
	m.editor = &taskEditor{draft: vimkunja.TaskDraft{Title: "Unsaved"}}
	_, cmd := typeCommand(t, m, "projectnew")
	if cmd != nil {
		t.Fatal("dirty Task opened dialog")
	}
	if m.projectCreation != nil || m.editor.draft.Title != "Unsaved" || !strings.Contains(m.lastErr, "unsaved") {
		t.Fatalf("draft protection: %q", m.lastErr)
	}
}

func TestProjectCreationCancellationAndInvalidChild(t *testing.T) {
	m := testModel()
	m.width, m.height = 90, 22
	_, cmd := typeCommand(t, m, "projectnewchild")
	if cmd != nil || m.projectCreation != nil || !strings.Contains(m.lastErr, "select a Project") {
		t.Fatalf("child without selection: %q", m.lastErr)
	}
	m.projects = []vimkunja.Project{{ID: 99, Title: "Archived"}}
	m.projectTree = newProjectTree(m.projects, nil)
	_, cmd = typeCommand(t, m, "projectnewchild")
	deliverProjectCommand(t, m, cmd)
	if m.projectCreation != nil || !strings.Contains(m.lastErr, "cannot be a Parent") {
		t.Fatalf("invalid Parent: %q", m.lastErr)
	}
	_, old := typeCommand(t, m, "projectnew")
	m, _ = typeCommand(t, m, "q!")
	deliverProjectCommand(t, m, old)
	if m.projectCreation != nil {
		t.Fatal("late Parent response reopened cancelled dialog")
	}
}

func TestProjectCreationFitsSmallTerminalsAndKeepsCursor(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {60, 15}, {32, 9}, {20, 6}, {8, 2}, {1, 1}} {
		m := testModel()
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		_, cmd := typeCommand(t, m, "projectnew")
		deliverProjectCommand(t, m, cmd)
		m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
		m = pressKey(m, tea.Key{Code: 'x', Text: strings.Repeat("界", 60)})
		v := m.View()
		text := ansi.Strip(v.Content)
		lines := strings.Split(text, "\n")
		if len(lines) > size[1] {
			t.Fatalf("%v: %d rows", size, len(lines))
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("%v: too wide: %q", size, line)
			}
		}
		if v.Cursor == nil || v.Cursor.X >= size[0] || v.Cursor.Y >= size[1] {
			t.Fatalf("%v: input cursor clipped: %#v", size, v.Cursor)
		}
	}
}

func TestProjectCreationPickerKeepsDuplicateProjectIDsVisible(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 18})
	m.store = &fakeStore{projects: []vimkunja.Project{{ID: 11, Title: strings.Repeat("Same", 40)}, {ID: 22, Title: strings.Repeat("Same", 40)}}}
	_, cmd := typeCommand(t, m, "projectnew")
	deliverProjectCommand(t, m, cmd)
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	text := ansi.Strip(m.Render())
	if !strings.Contains(text, "#11") || !strings.Contains(text, "#22") {
		t.Fatalf("long duplicate Titles hide IDs: %q", text)
	}
}

func TestProjectCreationErrorsStayVisibleInShortTerminals(t *testing.T) {
	for _, size := range [][2]int{{32, 9}, {20, 6}} {
		for _, failure := range []error{&vimkunja.ProjectCreationError{Field: "title", Message: "Duplicate Title"}, fmt.Errorf("disk full")} {
			m := testModel()
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			m.store = &projectDialogStore{fakeStore: &fakeStore{}, createErr: failure}
			_, cmd := typeCommand(t, m, "projectnew")
			deliverProjectCommand(t, m, cmd)
			m = enterProjectTitle(m, "Example")
			m = pressKey(m, tea.Key{Code: tea.KeyTab})
			m = pressKey(m, tea.Key{Code: tea.KeyTab})
			_, cmd = typeCommand(t, m, "w")
			deliverProjectCommand(t, m, cmd)
			text := ansi.Strip(m.Render())
			if !strings.Contains(text, failure.Error()) {
				t.Fatalf("%v: error hidden: %q", size, text)
			}
		}
	}
}

func TestProjectCreationIgnoresDelayedKanbanMutations(t *testing.T) {
	for _, operation := range []string{"move", "provision"} {
		t.Run(operation, func(t *testing.T) {
			m := testModel()
			m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
			board := vimkunja.KanbanBoard{View: vimkunja.ProjectView{ID: 10, ViewKind: "kanban"}, Buckets: []vimkunja.KanbanBucket{{ID: 1, Title: "Old Bucket"}}}
			store := &projectDialogStore{fakeStore: &fakeStore{projects: []vimkunja.Project{{ID: 1, Title: "Old"}}, kanbanBoard: board, provisionedBoard: board}}
			m.store = store
			m.projects = store.projects
			m.projectTree = newProjectTree(m.projects, nil)
			m.currentProjectID = 1
			m.currentViewID = 10
			m.kanbanActive = true
			var late tea.Msg
			if operation == "move" {
				late = m.moveKanbanTask(99, 1)()
			} else {
				late = m.provisionStandardKanbanView(1)()
			}
			_, cmd := typeCommand(t, m, "projectnew")
			deliverProjectCommand(t, m, cmd)
			m = enterProjectTitle(m, "New")
			m = pressKey(m, tea.Key{Code: tea.KeyTab})
			m = pressKey(m, tea.Key{Code: tea.KeyTab})
			_, cmd = typeCommand(t, m, "w")
			deliverProjectCommand(t, m, cmd)
			_, cmd = m.Update(late)
			deliverProjectCommand(t, m, cmd)
			if m.kanbanActive || m.currentProjectID != 50 || m.currentViewID != 501 {
				t.Fatalf("late %s replaced new List: project=%d view=%d kanban=%v", operation, m.currentProjectID, m.currentViewID, m.kanbanActive)
			}
		})
	}
}

func TestProjectCreationUsesFieldsAndFooterAndVimParentNavigation(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	store := &projectDialogStore{fakeStore: &fakeStore{projects: []vimkunja.Project{{ID: 1, Title: "Work"}, {ID: 2, Title: "junk"}}}}
	m.store = store
	_, cmd := typeCommand(t, m, "projectnew")
	deliverProjectCommand(t, m, cmd)
	text := ansi.Strip(m.Render())
	if strings.Contains(text, "[Create]") || strings.Contains(text, "[Cancel]") || !strings.Contains(text, ":w create") {
		t.Fatalf("action buttons or missing footer: %q", text)
	}
	m = enterProjectTitle(m, "New")
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if m.View().Cursor != nil {
		t.Fatal("navigation must not show search input cursor")
	}
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if m.projectCreation.parentID != 1 {
		t.Fatalf("j/k selection: %d", m.projectCreation.parentID)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	m = pressKey(m, tea.Key{Code: '/', Text: "/"})
	if m.View().Cursor == nil {
		t.Fatal("search must use shared Insert cursor")
	}
	for _, r := range "junk" {
		m = pressKey(m, tea.Key{Code: r, Text: string(r)})
	}
	text = ansi.Strip(m.Render())
	if strings.Contains(text, "#1 Work") || !strings.Contains(text, "#2 junk") {
		t.Fatalf("j/k were not searchable: %q", text)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	m = pressKey(m, tea.Key{Code: tea.KeyTab, Mod: tea.ModShift})
	_, cmd = typeCommand(t, m, "w")
	deliverProjectCommand(t, m, cmd)
	if len(store.projects) != 3 || store.projects[2].ParentProjectID != 2 {
		t.Fatalf(":w did not create: %#v", store.projects)
	}
}

func enterProjectTitle(m *model, title string) *model {
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: 'x', Text: title})
	return pressKey(m, tea.Key{Code: tea.KeyEscape})
}

func TestProjectCreationQuitProtectsDraftAndForceQuitDiscards(t *testing.T) {
	for _, response := range []string{"y", "n", "esc"} {
		m := testModel()
		m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		_, cmd := typeCommand(t, m, "projectnew")
		deliverProjectCommand(t, m, cmd)
		m = enterProjectTitle(m, "Keep this draft")
		_, cmd = typeCommand(t, m, "q")
		deliverProjectCommand(t, m, cmd)
		if !strings.Contains(ansi.Strip(m.Render()), "DISCARD NEW PROJECT") {
			t.Fatal(":q did not ask before discarding")
		}
		key := tea.Key{Code: []rune(response)[0], Text: response}
		if response == "esc" {
			key = tea.Key{Code: tea.KeyEscape}
		}
		m = pressKey(m, key)
		if response == "y" {
			if m.projectCreation != nil {
				t.Fatal("yes did not discard")
			}
		} else {
			if m.projectCreation == nil || m.projectCreation.title != "Keep this draft" || m.projectCreation.discardConfirm {
				t.Fatal("cancel lost draft")
			}
			_, cmd = typeCommand(t, m, "q!")
			deliverProjectCommand(t, m, cmd)
			if m.projectCreation != nil {
				t.Fatal(":q! did not discard directly")
			}
		}
	}
	m := testModel()
	_, cmd := typeCommand(t, m, "projectnew")
	deliverProjectCommand(t, m, cmd)
	_, cmd = typeCommand(t, m, "q")
	deliverProjectCommand(t, m, cmd)
	if m.projectCreation != nil {
		t.Fatal("unchanged draft should close without a question")
	}
}

func TestProjectCreationEnterNeverSavesAndParentOnlyChangeIsDirty(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	s := &projectDialogStore{fakeStore: &fakeStore{projects: []vimkunja.Project{{ID: 1, Title: "Parent"}}}}
	m.store = s
	_, cmd := typeCommand(t, m, "projectnew")
	deliverProjectCommand(t, m, cmd)
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: 'T', Text: "Title :q!"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if len(s.projects) != 1 || m.projectCreation == nil || m.projectCreation.mode != editorNormal || m.projectCreation.title != "Title :q!" {
		t.Fatal("Enter saved or interpreted Insert text as a command")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.projectCreation == nil {
		t.Fatal("Normal Esc discarded draft")
	}
	_, cmd = typeCommand(t, m, "q!")
	deliverProjectCommand(t, m, cmd)
	_, cmd = typeCommand(t, m, "projectnew")
	deliverProjectCommand(t, m, cmd)
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	_, cmd = typeCommand(t, m, "q")
	deliverProjectCommand(t, m, cmd)
	if !m.projectCreation.discardConfirm {
		t.Fatal("Parent-only change was not protected")
	}
}

func TestProjectCreationSharedTextEditingAndClipboard(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	_, cmd := typeCommand(t, m, "projectnew")
	deliverProjectCommand(t, m, cmd)
	m = enterProjectTitle(m, "Alpha Beta")
	for _, key := range []string{"0", "w", "c", "i", "w"} {
		m = pressKey(m, tea.Key{Code: []rune(key)[0], Text: key})
	}
	if m.projectCreation.mode != editorInsert {
		t.Fatal("shared ciw did not enter Insert")
	}
	m = pressKey(m, tea.Key{Code: 'X', Text: "Gamma"})
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.projectCreation.title != "Alpha Gamma" {
		t.Fatalf("shared word edit: %q", m.projectCreation.title)
	}
	copied := ""
	m.clipboardWrite = func(s string) error { copied = s; return nil }
	m = pressKey(m, tea.Key{Code: '0', Text: "0"})
	m = pressKey(m, tea.Key{Code: 'V', Text: "V"})
	m = pressKey(m, tea.Key{Code: ' ', Text: " "})
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	deliverProjectCommand(t, m, cmd)
	if copied != "Alpha Gamma" {
		t.Fatalf("selection copy: %q", copied)
	}
	m.clipboardRead = func() (string, error) { return "Replacement", nil }
	m = pressKey(m, tea.Key{Code: ' ', Text: " "})
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'p', Text: "p"}))
	deliverProjectCommand(t, m, cmd)
	if m.projectCreation.title != "Replacement" || m.projectCreation.mode != editorNormal {
		t.Fatal("selection paste did not use shared Normal behavior")
	}
	m = pressKey(m, tea.Key{Code: 'A', Text: "A"})
	m.clipboardRead = func() (string, error) { return " suffix", nil }
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'v', Mod: tea.ModCtrl}))
	deliverProjectCommand(t, m, cmd)
	if m.projectCreation.title != "Replacement suffix" {
		t.Fatalf("Insert clipboard: %q", m.projectCreation.title)
	}
}

func TestProjectCreationCommandCursorFitsAndCancelledChildIsClean(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {32, 9}, {20, 6}, {8, 2}, {1, 1}} {
		m := testModel()
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		_, cmd := typeCommand(t, m, "projectnew")
		deliverProjectCommand(t, m, cmd)
		m = pressKey(m, tea.Key{Code: ':', Text: ":"})
		m = pressKey(m, tea.Key{Code: 'w', Text: strings.Repeat("w", 60)})
		v := m.View()
		if v.Cursor == nil || v.Cursor.X >= size[0] || v.Cursor.Y >= size[1] {
			t.Fatalf("%v: command cursor clipped: %#v", size, v.Cursor)
		}
	}
	m := testModel()
	m.projects = []vimkunja.Project{{ID: 1, Title: "Parent"}}
	m.projectTree = newProjectTree(m.projects, nil)
	m.store.(*fakeStore).projects = m.projects
	_, cmd := typeCommand(t, m, "projectnewchild")
	deliverProjectCommand(t, m, cmd)
	_, cmd = typeCommand(t, m, "q")
	deliverProjectCommand(t, m, cmd)
	if m.projectCreation != nil {
		t.Fatal("preselected Parent alone made the draft dirty")
	}
}

func TestProjectCreationIgnoresLateClipboardAfterNavigation(t *testing.T) {
	m := testModel()
	_, cmd := typeCommand(t, m, "projectnew")
	deliverProjectCommand(t, m, cmd)
	m = enterProjectTitle(m, "Unchanged")
	m.clipboardRead = func() (string, error) { return "Late", nil }
	m = pressKey(m, tea.Key{Code: ' ', Text: " "})
	_, late := m.Update(tea.KeyPressMsg(tea.Key{Code: 'p', Text: "p"}))
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	deliverProjectCommand(t, m, late)
	if m.projectCreation.title != "Unchanged" {
		t.Fatal("late clipboard changed another editing context")
	}
}

func TestProjectCreationParentPickerListMotions(t *testing.T) {
	for _, height := range []int{30, 10} {
		m := testModel()
		m.Update(tea.WindowSizeMsg{Width: 100, Height: height})
		store := &fakeStore{}
		for i := int64(1); i <= 30; i++ {
			store.projects = append(store.projects, vimkunja.Project{ID: i, Title: fmt.Sprintf("Parent %02d", i)})
		}
		m.store = store
		_, cmd := typeCommand(t, m, "projectnew")
		deliverProjectCommand(t, m, cmd)
		m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
		m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
		m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
		if m.projectCreation.selected != 30 || !strings.Contains(ansi.Strip(m.Render()), "> #30 Parent 30") {
			t.Fatal("G did not reveal the last Parent")
		}
		m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
		if m.projectCreation.selected != 30 {
			t.Fatal("a single g moved the selection")
		}
		m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
		if m.projectCreation.selected != 0 || !strings.Contains(ansi.Strip(m.Render()), "> No parent Project") {
			t.Fatal("gg did not reveal the first choice")
		}
		rows := 0
		for _, line := range strings.Split(ansi.Strip(m.Render()), "\n") {
			if strings.Contains(line, "#") || strings.Contains(line, "No parent Project") {
				rows++
			}
		}
		step := max(1, rows/2)
		m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
		if m.projectCreation.selected != step {
			t.Fatalf("height %d: Ctrl+D moved %d, want half of %d visible rows", height, m.projectCreation.selected, rows)
		}
		m = pressKey(m, tea.Key{Code: 'u', Mod: tea.ModCtrl})
		m = pressKey(m, tea.Key{Code: 'u', Mod: tea.ModCtrl})
		if m.projectCreation.selected != 0 {
			t.Fatal("Ctrl+U did not clamp at start")
		}
		m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
		m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
		if m.projectCreation.selected != 30 {
			t.Fatal("Ctrl+D did not clamp at end")
		}
		m = pressKey(m, tea.Key{Code: '/', Text: "/"})
		m = pressKey(m, tea.Key{Code: 'P', Text: "Parent 0"})
		m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
		if m.projectCreation.query != "Parent 0" || m.projectCreation.selected != step {
			t.Fatal("paging while searching changed the query or used wrong step")
		}
		m = pressKey(m, tea.Key{Code: tea.KeyEnter})
		m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
		if m.projectCreation.selected != 9 || !strings.Contains(ansi.Strip(m.Render()), "> #9 Parent 09") {
			t.Fatal("G did not use the filtered choices")
		}
		m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
		m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
		m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
		if m.projectCreation.selected != 8 {
			t.Fatal("interrupted gg leaked its pending g")
		}
	}
}
