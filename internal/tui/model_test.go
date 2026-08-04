package tui

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	apperrors "github.com/vimkunja/vimkunja/internal/errors"
	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

type fakeStore struct {
	projects             []vimkunja.Project
	projectMoveErr       error
	movedProjectID       int64
	projectMoveDirection vimkunja.ProjectMoveDirection
	toggleErr            error
	toggleTaskID         int64
	toggleDone           bool
	deleteErr            error
	deleteTaskID         int64
	workflowStatus       vimkunja.WorkflowStatus
	workflowStatusErr    error
	kanbanBoard          vimkunja.KanbanBoard
	kanbanBoards         map[int64]vimkunja.KanbanBoard
	kanbanBoardErr       error
	views                []vimkunja.ProjectView
	viewsByProject       map[int64][]vimkunja.ProjectView
	tasksByView          map[int64][]vimkunja.Task
	provisionedBoard     vimkunja.KanbanBoard
	provisionErr         error
	provisionCalls       int
	workflowTaskID       int64
	workflowBucketID     int64
	workflowDone         map[int64]bool
	updateErr            error
	listTasksCalls       int
	listTasksViewID      int64
	overviewTasks        []vimkunja.Task
	overviewTasksCalls   int
	searchData           vimkunja.SearchData
	searchErr            error
	labelTasks           map[int64][]vimkunja.Task
	labelTasksErr        error
	labels               []vimkunja.Label
	taskLabels           map[int64][]vimkunja.Label
	labelErr             error
	createdLabelTitle    string
	renamedLabelID       int64
	renamedLabelTitle    string
	deletedLabelID       int64
	createdDraft         vimkunja.TaskDraft
	updatedDraft         vimkunja.TaskDraft
}

func (f *fakeStore) ListProjects(context.Context) ([]vimkunja.Project, error) {
	return append([]vimkunja.Project(nil), f.projects...), nil
}

func (f *fakeStore) MoveProjectAmongSiblings(_ context.Context, projectID int64, direction vimkunja.ProjectMoveDirection) error {
	f.movedProjectID = projectID
	f.projectMoveDirection = direction
	if f.projectMoveErr != nil {
		return f.projectMoveErr
	}
	selected := -1
	for i, project := range f.projects {
		if project.ID == projectID {
			selected = i
			break
		}
	}
	if selected < 0 {
		return nil
	}
	target := -1
	for i, project := range f.projects {
		if project.ParentProjectID != f.projects[selected].ParentProjectID {
			continue
		}
		if direction == vimkunja.ProjectMovePrevious && project.Position < f.projects[selected].Position && (target < 0 || project.Position > f.projects[target].Position) {
			target = i
		}
		if direction == vimkunja.ProjectMoveNext && project.Position > f.projects[selected].Position && (target < 0 || project.Position < f.projects[target].Position) {
			target = i
		}
	}
	if target >= 0 {
		f.projects[selected].Position, f.projects[target].Position = f.projects[target].Position, f.projects[selected].Position
	}
	return nil
}

func (f *fakeStore) ListViews(_ context.Context, projectID int64, _ int, _ int) ([]vimkunja.ProjectView, int, error) {
	if views, ok := f.viewsByProject[projectID]; ok {
		return views, 0, nil
	}
	return f.views, 0, nil
}

func (f *fakeStore) ListTasksInView(_ context.Context, _ int64, viewID int64, _ int, _ int) ([]vimkunja.Task, int, error) {
	f.listTasksCalls++
	f.listTasksViewID = viewID
	return append([]vimkunja.Task(nil), f.tasksByView[viewID]...), 0, nil
}

func (f *fakeStore) LoadManualKanbanBoard(_ context.Context, projectID int64) (vimkunja.KanbanBoard, error) {
	if board, ok := f.kanbanBoards[projectID]; ok {
		return board, f.kanbanBoardErr
	}
	return f.kanbanBoard, f.kanbanBoardErr
}

func (f *fakeStore) ProvisionStandardManualKanbanView(context.Context, int64) (vimkunja.KanbanBoard, error) {
	f.provisionCalls++
	return f.provisionedBoard, f.provisionErr
}

func (f *fakeStore) ListOverviewTasks(context.Context, int, int) ([]vimkunja.Task, int, error) {
	f.overviewTasksCalls++
	return f.overviewTasks, 0, nil
}

func (f *fakeStore) LoadSearchData(context.Context) (vimkunja.SearchData, error) {
	return f.searchData, f.searchErr
}

func (f *fakeStore) ListTasksForLabel(_ context.Context, labelID int64, _ int, _ int) ([]vimkunja.Task, int, error) {
	return append([]vimkunja.Task(nil), f.labelTasks[labelID]...), 0, f.labelTasksErr
}

func (f *fakeStore) ListLabels(context.Context) ([]vimkunja.Label, error) {
	return append([]vimkunja.Label(nil), f.labels...), f.labelErr
}

func (f *fakeStore) ListTaskLabels(_ context.Context, taskID int64) ([]vimkunja.Label, error) {
	return append([]vimkunja.Label(nil), f.taskLabels[taskID]...), f.labelErr
}

func (f *fakeStore) CreateLabel(_ context.Context, title string) (vimkunja.Label, error) {
	if f.labelErr != nil {
		return vimkunja.Label{}, f.labelErr
	}
	f.createdLabelTitle = title
	label := vimkunja.Label{ID: 99, Title: title}
	f.labels = append(f.labels, label)
	return label, nil
}

func (f *fakeStore) RenameLabel(_ context.Context, labelID int64, title string) error {
	f.renamedLabelID, f.renamedLabelTitle = labelID, title
	if f.labelErr == nil {
		for i := range f.labels {
			if f.labels[i].ID == labelID {
				f.labels[i].Title = title
			}
		}
	}
	return f.labelErr
}

func (f *fakeStore) DeleteLabel(_ context.Context, labelID int64) error {
	f.deletedLabelID = labelID
	if f.labelErr == nil {
		for i, label := range f.labels {
			if label.ID == labelID {
				f.labels = append(f.labels[:i], f.labels[i+1:]...)
				break
			}
		}
	}
	return f.labelErr
}
func (f *fakeStore) ReplaceTaskLabels(_ context.Context, taskID int64, labelIDs []int64) error {
	if f.labelErr == nil {
		labels := make([]vimkunja.Label, 0, len(labelIDs))
		for _, labelID := range labelIDs {
			labels = append(labels, vimkunja.Label{ID: labelID})
		}
		if f.taskLabels == nil {
			f.taskLabels = make(map[int64][]vimkunja.Label)
		}
		f.taskLabels[taskID] = labels
	}
	return f.labelErr
}

func (f *fakeStore) GetTaskWorkflowStatus(context.Context, int64, int64) (vimkunja.WorkflowStatus, error) {
	return f.workflowStatus, f.workflowStatusErr
}

func (f *fakeStore) SetTaskWorkflowStatus(_ context.Context, taskID, bucketID int64) error {
	f.workflowTaskID = taskID
	f.workflowBucketID = bucketID
	var moved vimkunja.Task
	found := false
	for i := range f.kanbanBoard.Buckets {
		for j, task := range f.kanbanBoard.Buckets[i].Tasks {
			if task.ID == taskID {
				moved = task
				f.kanbanBoard.Buckets[i].Tasks = append(f.kanbanBoard.Buckets[i].Tasks[:j], f.kanbanBoard.Buckets[i].Tasks[j+1:]...)
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if found {
		if done, ok := f.workflowDone[bucketID]; ok {
			moved.Done = done
		}
		for i := range f.kanbanBoard.Buckets {
			if f.kanbanBoard.Buckets[i].ID == bucketID {
				f.kanbanBoard.Buckets[i].Tasks = append(f.kanbanBoard.Buckets[i].Tasks, moved)
				break
			}
		}
	}
	return nil
}

func (f *fakeStore) ToggleTaskDone(_ context.Context, taskID int64, done bool) error {
	f.toggleTaskID = taskID
	f.toggleDone = done
	return f.toggleErr
}

func (f *fakeStore) DeleteTask(_ context.Context, taskID int64) error {
	f.deleteTaskID = taskID
	return f.deleteErr
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
	return vimkunja.Task{ID: draft.ID, ProjectID: draft.ProjectID, Title: draft.Title, Done: draft.Done}, nil
}

func TestProjectsPaneRefreshToEmptyClearsStaleProjectContext(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(projectsLoadedMsg{projects: []vimkunja.Project{{ID: 1, Title: "Only Project"}}})
	m = updated.(*model)
	m.tasks = []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "Stale Task"}}
	m.projectViews = []vimkunja.ProjectView{{ID: 10, Title: "List"}}
	m.currentViewID = 10

	updated, _ = m.Update(projectsLoadedMsg{})
	m = updated.(*model)
	if m.currentProjectID != 0 || m.currentViewID != 0 || len(m.tasks) != 0 || len(m.projectViews) != 0 {
		t.Fatalf("empty refresh retained stale context: Project:%d View:%d Tasks:%v Views:%v", m.currentProjectID, m.currentViewID, m.tasks, m.projectViews)
	}
	if !strings.Contains(ansi.Strip(renderProjects(m, 24, 4)), "(no projects)") {
		t.Fatalf("empty refresh did not render Projects empty state: %q", renderProjects(m, 24, 4))
	}
}

func TestProjectsPaneViewportKeepsSelectionVisible(t *testing.T) {
	m := testModel()
	projects := make([]vimkunja.Project, 10)
	for i := range projects {
		projects[i] = vimkunja.Project{ID: int64(i + 1), Title: fmt.Sprintf("Project %d", i+1), Position: float64(i + 1)}
	}
	updated, _ := m.Update(projectsLoadedMsg{projects: projects})
	m = updated.(*model)
	m.projectSelected = 9

	view := ansi.Strip(renderProjects(m, 24, 5))
	if !strings.Contains(view, "> Project 10 <") || strings.Contains(view, "Project 1\n") || m.projectViewportStart != 6 {
		t.Fatalf("Projects viewport = start:%d view:%q, want final four rows with Project 10 selected", m.projectViewportStart, view)
	}
}

func TestProjectsPaneRefreshRetainsSelectionByIDAndUsesFormerPositionFallback(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(projectsLoadedMsg{projects: []vimkunja.Project{
		{ID: 1, Title: "First", Position: 1},
		{ID: 2, Title: "Selected", Position: 2},
		{ID: 3, Title: "Next", Position: 3},
	}})
	m = updated.(*model)
	m.projectSelected = 1

	updated, _ = m.Update(projectsLoadedMsg{projects: []vimkunja.Project{
		{ID: 3, Title: "Next", Position: 1},
		{ID: 2, Title: "Selected", Position: 2},
		{ID: 1, Title: "First", Position: 3},
	}})
	m = updated.(*model)
	if selected, _ := m.selectedProject(); selected.ID != 2 {
		t.Fatalf("refresh selected Project %d, want retained ID 2", selected.ID)
	}

	updated, _ = m.Update(projectsLoadedMsg{projects: []vimkunja.Project{
		{ID: 1, Title: "First", Position: 1},
		{ID: 3, Title: "Next", Position: 2},
	}})
	m = updated.(*model)
	if selected, _ := m.selectedProject(); selected.ID != 1 {
		t.Fatalf("refresh after deletion selected Project %d, want next former row 1", selected.ID)
	}

	m.projectSelected = 1 // Project 3 is now the last visible row.
	updated, _ = m.Update(projectsLoadedMsg{projects: []vimkunja.Project{{ID: 1, Title: "First", Position: 1}}})
	m = updated.(*model)
	if selected, _ := m.selectedProject(); selected.ID != 1 {
		t.Fatalf("refresh after deleting last row selected Project %d, want preceding row 1", selected.ID)
	}
}

func TestProjectsPaneSiblingMovePersistsThenReloadsWithoutOptimisticRedraw(t *testing.T) {
	store := &fakeStore{projects: []vimkunja.Project{
		{ID: 1, Title: "Root", Position: 1},
		{ID: 2, Title: "First", ParentProjectID: 1, Position: 1},
		{ID: 3, Title: "Second", ParentProjectID: 1, Position: 2},
	}}
	m := NewModel(Config{Store: store}).(*model)
	updated, _ := m.Update(projectsLoadedMsg{projects: store.projects})
	m = updated.(*model)
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})

	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'J', Text: "J"}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("Shift+J returned no persistence command")
	}
	if got, want := visibleProjectIDs(m.projectTree), []int64{1, 2, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Project order changed before persistence: %v, want %v", got, want)
	}

	updated, reload := m.Update(cmd())
	m = updated.(*model)
	if store.movedProjectID != 2 || store.projectMoveDirection != vimkunja.ProjectMoveNext || reload == nil {
		t.Fatalf("move call = Project:%d direction:%d reload:%v", store.movedProjectID, store.projectMoveDirection, reload != nil)
	}
	if got := visibleProjectIDs(m.projectTree); !reflect.DeepEqual(got, []int64{1, 2, 3}) {
		t.Fatalf("Project order changed before reload: %v", got)
	}
	updated, _ = m.Update(reload())
	m = updated.(*model)
	selected, _ := m.selectedProject()
	if got, want := visibleProjectIDs(m.projectTree), []int64{1, 3, 2}; !reflect.DeepEqual(got, want) || selected.ID != 2 {
		t.Fatalf("reloaded move = rows:%v selected:%d, want rows:%v selected:2", got, selected.ID, want)
	}
}

func TestProjectsPaneSiblingMoveFailureRetainsDisplay(t *testing.T) {
	store := &fakeStore{projectMoveErr: errors.New("database is locked")}
	m := testModel()
	m.store = store
	updated, _ := m.Update(projectsLoadedMsg{projects: []vimkunja.Project{
		{ID: 1, Title: "First", Position: 1}, {ID: 2, Title: "Second", Position: 2},
	}})
	m = updated.(*model)
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'K', Text: "K"}))
	m = updated.(*model)
	updated, reload := m.Update(cmd())
	m = updated.(*model)
	if reload != nil || m.lastErr != "database is locked" || !reflect.DeepEqual(visibleProjectIDs(m.projectTree), []int64{1, 2}) {
		t.Fatalf("failed move = reload:%v error:%q rows:%v", reload != nil, m.lastErr, visibleProjectIDs(m.projectTree))
	}
}

func TestProjectsPaneTreeKeysNavigateAndFoldVisibleRows(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(projectsLoadedMsg{projects: []vimkunja.Project{
		{ID: 1, Title: "Root", Position: 1},
		{ID: 2, Title: "Child", ParentProjectID: 1, Position: 1},
		{ID: 3, Title: "Grandchild", ParentProjectID: 2, Position: 1},
		{ID: 4, Title: "Other root", Position: 2},
	}})
	m = updated.(*model)

	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	if selected, _ := m.selectedProject(); selected.ID != 2 {
		t.Fatalf("l on open branch selected Project %d, want first child 2", selected.ID)
	}
	m = pressKey(m, tea.Key{Code: 'h', Text: "h"})
	if selected, _ := m.selectedProject(); selected.ID != 2 || !m.projectTree.collapsed[2] {
		t.Fatalf("first h should close selected branch 2; selected:%d collapsed:%v", selected.ID, m.projectTree.collapsed)
	}
	m = pressKey(m, tea.Key{Code: 'h', Text: "h"})
	if selected, _ := m.selectedProject(); selected.ID != 1 {
		t.Fatalf("second h on closed child selected Project %d, want parent 1", selected.ID)
	}

	m = pressKey(pressKey(m, tea.Key{Code: 'z', Text: "z"}), tea.Key{Code: 'c', Text: "c"})
	if got, want := visibleProjectIDs(m.projectTree), []int64{1, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("zc rows = %v, want %v", got, want)
	}
	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	if selected, _ := m.selectedProject(); selected.ID != 1 {
		t.Fatalf("first l on closed branch selected %d, want branch to stay selected", selected.ID)
	}
	if got, want := visibleProjectIDs(m.projectTree), []int64{1, 2, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first l did not open branch: rows %v, want %v", got, want)
	}
	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	if selected, _ := m.selectedProject(); selected.ID != 2 {
		t.Fatalf("second l selected %d, want child 2", selected.ID)
	}
	m = pressKey(pressKey(m, tea.Key{Code: 'z', Text: "z"}), tea.Key{Code: 'o', Text: "o"})
	if got := visibleProjectIDs(m.projectTree); len(got) != 4 {
		t.Fatalf("zo did not open child branch: %v", got)
	}
	m = pressKey(pressKey(m, tea.Key{Code: 'z', Text: "z"}), tea.Key{Code: 'a', Text: "a"})
	if got, want := visibleProjectIDs(m.projectTree), []int64{1, 2, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("za rows = %v, want %v", got, want)
	}
	m = pressKey(pressKey(m, tea.Key{Code: 'z', Text: "z"}), tea.Key{Code: 'R', Text: "R"})
	if selected, _ := m.selectedProject(); selected.ID != 2 || len(visibleProjectIDs(m.projectTree)) != 4 {
		t.Fatalf("zR did not expand all and retain selection: selected:%d rows:%v", selected.ID, visibleProjectIDs(m.projectTree))
	}
	m = pressKey(pressKey(m, tea.Key{Code: 'z', Text: "z"}), tea.Key{Code: 'M', Text: "M"})
	if selected, _ := m.selectedProject(); selected.ID != 1 || !reflect.DeepEqual(visibleProjectIDs(m.projectTree), []int64{1, 4}) {
		t.Fatalf("zM did not collapse all to selected root: selected:%d rows:%v", selected.ID, visibleProjectIDs(m.projectTree))
	}
}

func TestProjectsPaneMarksMalformedProjectsAsRecoveryRoots(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(projectsLoadedMsg{projects: []vimkunja.Project{
		{ID: 1, Title: "Valid", Position: 1},
		{ID: 2, Title: "Orphan", ParentProjectID: 99, Position: 2},
	}})
	m = updated.(*model)

	view := ansi.Strip(renderProjects(m, 24, 5))
	if !strings.Contains(view, "! Orphan") {
		t.Fatalf("recovery marker missing from Projects Pane: %q", view)
	}
}

func TestProjectsPaneRendersHierarchyAndSelectedProject(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(projectsLoadedMsg{projects: []vimkunja.Project{
		{ID: 2, Identifier: "CHILD", ParentProjectID: 1, Position: 1},
		{ID: 1, Identifier: "ROOT", Position: 1},
	}})
	m = updated.(*model)

	view := ansi.Strip(renderProjects(m, 24, 6))
	if !strings.Contains(view, "▾ > ROOT <") {
		t.Fatalf("selected root presentation missing from Projects Pane: %q", view)
	}
	if !strings.Contains(view, "    CHILD") {
		t.Fatalf("indented child missing from Projects Pane: %q", view)
	}
	if strings.Contains(view, "[X]") {
		t.Fatalf("legacy selection marker remains in Projects Pane: %q", view)
	}
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

func TestGlobalSearchOpensBlankOverlayAndEscapeRestoresOrigin(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	m.focus = paneProjects
	m.currentProjectID = 1

	m = pressKey(m, tea.Key{Code: ' ', Text: " "})
	m = pressKey(m, tea.Key{Code: 'f', Text: "f"})
	if !m.searchOpen || !strings.Contains(ansi.Strip(m.Render()), "GLOBAL SEARCH") || !strings.Contains(ansi.Strip(m.Render()), "Type to search") {
		t.Fatalf("blank Global Search overlay = %q", m.Render())
	}

	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if !m.searchOpen || m.searchMode != searchNormalMode {
		t.Fatalf("first Esc = open:%v mode:%v, want Search NORMAL", m.searchOpen, m.searchMode)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.searchOpen || m.focus != paneProjects || m.currentProjectID != 1 {
		t.Fatalf("second Esc did not restore search origin: open=%v focus=%v project=%d", m.searchOpen, m.focus, m.currentProjectID)
	}
}

func TestGlobalSearchModalKeepsItsBottomBorderWithManyDescriptionResults(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	m.searchOpen = true
	m.searchMode = searchInsertMode
	m.searchInput = "f"
	for i := 0; i < 17; i++ {
		m.searchResults = append(m.searchResults, vimkunja.SearchResult{
			Kind: vimkunja.SearchDescriptionResult, Primary: "Task", Snippet: "A matching Description preview",
		})
	}
	background := strings.Repeat(strings.Repeat(" ", 100)+"\n", 29) + strings.Repeat(" ", 100)
	overlay := ansi.Strip(m.renderSearchOverlay(background, 100))
	if !strings.Contains(overlay, "┛") {
		t.Fatalf("crowded Search overlay has no bottom border: %q", overlay)
	}
}

func TestGlobalSearchUsesAvailableRowsForLaterResultGroups(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	m.searchOpen = true
	m.searchInput = "f"
	for i := 0; i < 7; i++ {
		m.searchResults = append(m.searchResults, vimkunja.SearchResult{Kind: vimkunja.SearchTaskResult, Primary: "Task"})
	}
	m.searchResults = append(m.searchResults,
		vimkunja.SearchResult{Kind: vimkunja.SearchLabelResult, Primary: "Label"},
		vimkunja.SearchResult{Kind: vimkunja.SearchDescriptionResult, Primary: "Described Task", Snippet: "Matching Description"},
	)
	rendered := ansi.Strip(m.renderSearchOverlay(strings.Repeat(strings.Repeat(" ", 100)+"\n", 29)+strings.Repeat(" ", 100), 100))
	if !strings.Contains(rendered, "DESCRIPTIONS") {
		t.Fatalf("Search left room unused instead of rendering later result groups: %q", rendered)
	}
}

func TestGlobalSearchModalHeightDoesNotDependOnResultCount(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	m.searchOpen = true
	m.searchInput = "f"
	background := strings.Repeat(strings.Repeat(" ", 100)+"\n", 29) + strings.Repeat(" ", 100)

	emptyOverlay := ansi.Strip(m.renderSearchOverlay(background, 100))
	emptyTop, emptyBottom := searchModalTopRow(emptyOverlay), searchModalBottomRow(emptyOverlay)
	for i := 0; i < 17; i++ {
		m.searchResults = append(m.searchResults, vimkunja.SearchResult{Kind: vimkunja.SearchDescriptionResult, Primary: "Task", Snippet: "Preview"})
	}
	crowdedOverlay := ansi.Strip(m.renderSearchOverlay(background, 100))
	crowdedTop, crowdedBottom := searchModalTopRow(crowdedOverlay), searchModalBottomRow(crowdedOverlay)
	if emptyTop < 0 || crowdedTop != emptyTop || emptyBottom < 0 || crowdedBottom != emptyBottom {
		t.Fatalf("Search modal bounds = empty:%d-%d crowded:%d-%d, want a stable viewport-derived size", emptyTop, emptyBottom, crowdedTop, crowdedBottom)
	}
}

func searchModalTopRow(rendered string) int {
	for row, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "┏") {
			return row
		}
	}
	return -1
}

func searchModalBottomRow(rendered string) int {
	for row, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "┛") {
			return row
		}
	}
	return -1
}

func TestGlobalSearchModalBoundsStayFixedAtFullResultLineBudget(t *testing.T) {
	m := testModel()
	m.width, m.height = 240, 65
	m.searchOpen = true
	m.searchInput = "f"
	background := strings.Repeat(strings.Repeat(" ", m.width)+"\n", m.height-1) + strings.Repeat(" ", m.width)
	emptyOverlay := ansi.Strip(m.renderSearchOverlay(background, m.width))
	for i := 0; i < 26; i++ {
		m.searchResults = append(m.searchResults, vimkunja.SearchResult{Kind: vimkunja.SearchTaskResult, Primary: "Task"})
	}
	for i := 0; i < 7; i++ {
		m.searchResults = append(m.searchResults, vimkunja.SearchResult{Kind: vimkunja.SearchDescriptionResult, Primary: "Task", Snippet: "Description"})
	}
	fullOverlay := ansi.Strip(m.renderSearchOverlay(background, m.width))
	if searchModalTopRow(fullOverlay) != searchModalTopRow(emptyOverlay) || searchModalBottomRow(fullOverlay) != searchModalBottomRow(emptyOverlay) {
		t.Fatalf("full Search results changed modal bounds: empty %d-%d, full %d-%d", searchModalTopRow(emptyOverlay), searchModalBottomRow(emptyOverlay), searchModalTopRow(fullOverlay), searchModalBottomRow(fullOverlay))
	}
}

func TestGlobalSearchNormalModeNavigatesAndLocalQuitClosesOnlySearch(t *testing.T) {
	m := testModel()
	m.searchOpen = true
	m.searchMode = searchNormalMode
	m.searchResults = []vimkunja.SearchResult{
		{Kind: vimkunja.SearchTaskResult, Primary: "First"},
		{Kind: vimkunja.SearchTaskResult, Primary: "Second"},
	}
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.searchSelected != 1 {
		t.Fatalf("Normal j selected %d, want second result", m.searchSelected)
	}
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	if m.searchSelected != 0 {
		t.Fatalf("Normal gg selected %d, want first result", m.searchSelected)
	}
	m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
	if m.searchSelected != 1 {
		t.Fatalf("Normal G selected %d, want last result", m.searchSelected)
	}
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	if m.searchMode != searchInsertMode {
		t.Fatalf("Normal i mode = %v, want Insert", m.searchMode)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Code: 'q', Text: "q"})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if cmd != nil || m.searchOpen {
		t.Fatalf("Search :q = open:%v cmd:%v, want only Search closed", m.searchOpen, cmd != nil)
	}
}

func TestGlobalSearchLocalQuitAllQuitsVimkunja(t *testing.T) {
	m := testModel()
	m.searchOpen = true
	m.searchMode = searchNormalMode
	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Code: 'q', Text: "q"})
	m = pressKey(m, tea.Key{Code: 'a', Text: "a"})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("Search :qa returned no quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("Search :qa command returned %T, want tea.QuitMsg", cmd())
	}
}

func TestGlobalSearchQueriesAndRendersTypedResults(t *testing.T) {
	store := &fakeStore{searchData: vimkunja.SearchData{Tasks: []vimkunja.Task{{
		ID: 7, ProjectID: 1, ProjectIdentifier: "HOME", ProjectTitle: "Home", Title: "Buy coffee", Description: "<p>Fresh coffee beans</p>",
	}}}}
	m := NewModel(Config{Store: store}).(*model)
	m.width, m.height = 100, 30

	m = pressKey(m, tea.Key{Code: ' ', Text: " "})
	m = pressKey(m, tea.Key{Code: 'f', Text: "f"})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Text: "coffee"}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("search input returned no query command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if len(m.searchResults) != 2 || m.searchResults[0].Kind != vimkunja.SearchTaskResult || m.searchResults[1].Kind != vimkunja.SearchDescriptionResult {
		t.Fatalf("search results = %#v, want Task and Description hits", m.searchResults)
	}
	view := ansi.Strip(m.Render())
	if !strings.Contains(view, "TASKS") || !strings.Contains(view, "DESCRIPTIONS") || strings.Contains(view, "<p>") {
		t.Fatalf("Global Search render = %q, want typed groups without HTML", view)
	}
}

func TestGlobalSearchLabelActivationOpensComposedOverview(t *testing.T) {
	store := &fakeStore{labelTasks: map[int64][]vimkunja.Task{21: {
		{ID: 1, Title: "Open errand", Done: false},
		{ID: 2, Title: "Done errand", Done: true},
	}}}
	m := NewModel(Config{Store: store}).(*model)
	m.searchOpen = true
	m.searchResults = []vimkunja.SearchResult{{Kind: vimkunja.SearchLabelResult, LabelID: 21, Primary: "Errands"}}

	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if cmd == nil || !m.inOverview() || m.overviewLabel.ID != 21 || m.overviewFilter != overviewOpen {
		t.Fatalf("Label activation = scope:%v label:%#v filter:%v cmd:%v", m.taskScope, m.overviewLabel, m.overviewFilter, cmd != nil)
	}
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if len(m.tasks) != 1 || m.tasks[0].ID != 1 {
		t.Fatalf("open Label Overview tasks = %#v, want only open Label task", m.tasks)
	}

	m = pressKey(m, tea.Key{Code: 'f', Text: "f"})
	m = pressKey(m, tea.Key{Code: 'a', Text: "a"})
	if len(m.tasks) != 2 {
		t.Fatalf("all Label Overview tasks = %#v, want both Label tasks", m.tasks)
	}
	if header := ansi.Strip(renderTasks(m, 90, 10)); !strings.Contains(header, "Errands") || !strings.Contains(header, "all") {
		t.Fatalf("Label Overview header = %q, want Label and status constraints", header)
	}
}

func TestGlobalSearchTaskActivationOpensOwningProjectTaskDetail(t *testing.T) {
	store := &fakeStore{
		viewsByProject: map[int64][]vimkunja.ProjectView{2: {{ID: 5, ViewKind: "list"}}},
		tasksByView:    map[int64][]vimkunja.Task{5: {{ID: 7, ProjectID: 2, Title: "Buy coffee"}}},
	}
	m := NewModel(Config{Store: store}).(*model)
	m.projects = []vimkunja.Project{{ID: 1, Title: "Old"}, {ID: 2, Title: "Home"}}
	m.searchOpen = true
	m.searchResults = []vimkunja.SearchResult{{Kind: vimkunja.SearchTaskResult, TaskID: 7, ProjectID: 2, Primary: "Buy coffee"}}

	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("Task activation returned no Project View load")
	}
	updated, cmd = m.Update(cmd())
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("loaded Project View returned no Task load")
	}
	updated, cmd = m.Update(cmd())
	m = updated.(*model)
	if !m.showDetail || m.focus != paneDetail || m.editor == nil || m.editor.id != 7 || m.currentProjectID != 2 || m.currentViewID != 5 {
		t.Fatalf("Task activation state = detail:%v focus:%v editor:%#v project:%d view:%d", m.showDetail, m.focus, m.editor, m.currentProjectID, m.currentViewID)
	}
}

func TestGlobalSearchCtrlORestoresSnapshotAndEscapeReturnsToOrigin(t *testing.T) {
	store := &fakeStore{
		searchData: vimkunja.SearchData{Labels: []vimkunja.Label{{ID: 21, Title: "Errands"}}},
		labelTasks: map[int64][]vimkunja.Task{21: {{ID: 2, Title: "Errand"}}},
	}
	m := NewModel(Config{Store: store}).(*model)
	m.focus = paneTasks
	m.currentProjectID, m.currentViewID = 1, 5
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Origin"}}

	m = pressKey(m, tea.Key{Code: ' ', Text: " "})
	m = pressKey(m, tea.Key{Code: 'f', Text: "f"})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Text: "errands"}))
	m = updated.(*model)
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if !m.inOverview() {
		t.Fatal("Label activation did not leave the Search origin")
	}

	m = pressKey(m, tea.Key{Code: 'o', Mod: tea.ModCtrl})
	if !m.searchOpen || m.searchInput != "errands" || len(m.searchResults) != 1 || m.searchResults[0].LabelID != 21 {
		t.Fatalf("Ctrl+O Search snapshot = open:%v query:%q results:%#v", m.searchOpen, m.searchInput, m.searchResults)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if !m.searchOpen || m.searchMode != searchNormalMode {
		t.Fatalf("first Esc after Ctrl+O = open:%v mode:%v, want Search NORMAL", m.searchOpen, m.searchMode)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.searchOpen || m.inOverview() || m.currentProjectID != 1 || m.currentViewID != 5 || len(m.tasks) != 1 || m.tasks[0].ID != 1 || m.focus != paneTasks {
		t.Fatalf("second Esc after Ctrl+O origin = open:%v scope:%v project:%d view:%d tasks:%#v focus:%v", m.searchOpen, m.taskScope, m.currentProjectID, m.currentViewID, m.tasks, m.focus)
	}
}

func TestSearchCommandOpensGlobalSearchOverlay(t *testing.T) {
	m, cmd := typeCommand(t, testModel(), "search")
	if !m.searchOpen || cmd != nil {
		t.Fatalf(":search = open:%v cmd:%v, want an open overlay without a query", m.searchOpen, cmd != nil)
	}
}

func TestGlobalSearchModalWidthUsesViewportProportionAndNarrowMargins(t *testing.T) {
	if got := searchModalWidth(60); got != 56 {
		t.Fatalf("narrow modal width = %d, want terminal width minus four", got)
	}
	if got := searchModalWidth(200); got != 188 {
		t.Fatalf("wide modal width = %d, want 94%% of viewport", got)
	}
}

func TestGlobalSearchModalHeightUsesEightyPercentOfViewport(t *testing.T) {
	if got := searchModalHeight(50); got != 40 {
		t.Fatalf("modal height = %d, want 80%% of viewport", got)
	}
}

func TestGlobalSearchUsesPinkCursorAndYellowDescriptionHighlight(t *testing.T) {
	m := testModel()
	m.searchOpen = true
	m.searchInput = "coffee"
	m.searchResults = []vimkunja.SearchResult{{
		Kind: vimkunja.SearchDescriptionResult, Primary: "Description task", Snippet: "Coffee notes", SnippetHighlights: []vimkunja.SearchHighlight{{Start: 0, End: 6}},
	}}
	lines := m.renderSearchResults(70)
	if got := ansi.Strip(lines[1]); !strings.HasPrefix(got, "> Description task") {
		t.Fatalf("selected search row = %q, want a > cursor", got)
	}
	if !strings.Contains(lines[1], "38;5;205") {
		t.Fatalf("selected search cursor lacks pink selection color: %q", lines[1])
	}
	if !strings.Contains(lines[2], "38;5;229") {
		t.Fatalf("Description match lacks inactive-selection yellow: %q", lines[2])
	}
}

func TestGlobalSearchDiscardsStaleResponsesAndRetainsResultsOnError(t *testing.T) {
	m := testModel()
	m.searchOpen = true
	m.searchInput = "new"
	m.searchRequest = 2
	m.searchResults = []vimkunja.SearchResult{{Kind: vimkunja.SearchTaskResult, TaskID: 2, Primary: "Newest"}}

	updated, _ := m.Update(searchLoadedMsg{request: 1, query: "old", results: []vimkunja.SearchResult{{Primary: "Stale"}}})
	m = updated.(*model)
	if len(m.searchResults) != 1 || m.searchResults[0].Primary != "Newest" {
		t.Fatalf("stale response replaced results: %#v", m.searchResults)
	}
	updated, _ = m.Update(searchLoadedMsg{request: 2, query: "new", err: errors.New("database unavailable")})
	m = updated.(*model)
	if len(m.searchResults) != 1 || m.searchResults[0].Primary != "Newest" || m.searchInput != "new" || m.lastErr != "database unavailable" {
		t.Fatalf("error response state = results:%#v query:%q error:%q", m.searchResults, m.searchInput, m.lastErr)
	}
}

func TestKanbanBucketsHaveDimBordersAndBlankHeaderRow(t *testing.T) {
	m := testModel()
	m.kanbanActive = true
	m.kanbanBoard = vimkunja.KanbanBoard{Buckets: []vimkunja.KanbanBucket{{
		ID: 1, Title: "To-Do", Tasks: []vimkunja.Task{{ID: 7, Title: "Open"}},
	}}}

	view := renderKanbanBoard(m, 25, 5)
	lines := strings.Split(ansi.Strip(view), "\n")
	if !strings.Contains(lines[0], "┌") || !strings.Contains(lines[4], "└") {
		t.Fatalf("Kanban bucket lacks border: %q", view)
	}
	if !strings.Contains(lines[1], "To-Do") || strings.Contains(lines[2], "Open") || !strings.Contains(lines[3], "Open") {
		t.Fatalf("Kanban bucket spacing = %q, want a blank row after the header", lines)
	}
	if !strings.Contains(view, "\x1b[38;5;245m") {
		t.Fatalf("Kanban bucket border lacks dim white color: %q", view)
	}
}

func TestKanbanBoardUsesAvailableWidthWhenAllBucketsFit(t *testing.T) {
	m := testModel()
	m.kanbanActive = true
	m.kanbanBoard = vimkunja.KanbanBoard{Buckets: []vimkunja.KanbanBucket{
		{ID: 1, Title: "To-Do"},
		{ID: 2, Title: "Doing"},
	}}

	header := strings.Split(ansi.Strip(renderKanbanBoard(m, 80, 4)), "\n")[1]
	if columnStart := strings.Index(header, "Doing"); columnStart < 40 {
		t.Fatalf("second Bucket starts at column %d, want a wide layout using the 80-column pane: %q", columnStart, header)
	}
}

func TestKanbanCommandOpensAvailableManualBoard(t *testing.T) {
	store := &fakeStore{kanbanBoard: vimkunja.KanbanBoard{
		Availability: vimkunja.KanbanBoardAvailable,
		View:         vimkunja.ProjectView{ID: 10, Title: "Kanban", ViewKind: "kanban", BucketConfigurationMode: "manual"},
		Buckets: []vimkunja.KanbanBucket{
			{ID: 100, Title: "To-Do", Tasks: []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "Open"}}},
			{ID: 101, Title: "Doing"},
		},
	}}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.focus = paneTasks

	m, cmd := typeCommand(t, m, "kanban")
	if cmd == nil {
		t.Fatal(":kanban returned no board load command")
	}
	updated, _ := m.Update(cmd())
	m = updated.(*model)
	if m.currentViewID != 10 || !strings.Contains(m.Render(), "To-Do") || !strings.Contains(m.Render(), "Doing") {
		t.Fatalf("kanban board was not opened: view:%d render:%q", m.currentViewID, m.Render())
	}
}

func TestKanbanCommandProvisionsAfterExplicitConfirmation(t *testing.T) {
	store := &fakeStore{
		kanbanBoard: vimkunja.KanbanBoard{Availability: vimkunja.KanbanBoardUnavailable},
		provisionedBoard: vimkunja.KanbanBoard{
			Availability: vimkunja.KanbanBoardAvailable,
			View:         vimkunja.ProjectView{ID: 10, Title: "Kanban", ViewKind: "kanban", BucketConfigurationMode: "manual"},
			Buckets:      []vimkunja.KanbanBucket{{ID: 100, Title: "To-Do"}},
		},
	}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.projects = []vimkunja.Project{{ID: 1, Title: "Home"}}
	m.focus = paneTasks

	m, cmd := typeCommand(t, m, "kanban")
	updated, _ := m.Update(cmd())
	m = updated.(*model)
	if !strings.Contains(m.renderStatusBar(100), `Create standard Kanban View for "Home"? y/n`) {
		t.Fatalf("kanban prompt = %q", m.renderStatusBar(100))
	}
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("confirmation returned no provisioning command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if m.currentViewID != 10 || !strings.Contains(m.Render(), "To-Do") {
		t.Fatalf("provisioned board was not opened: view:%d render:%q", m.currentViewID, m.Render())
	}
}

func TestKanbanProvisionCanCancelAndAmbiguousBoardIsNotSelected(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		store := &fakeStore{kanbanBoard: vimkunja.KanbanBoard{Availability: vimkunja.KanbanBoardUnavailable}}
		m := testModel()
		m.store = store
		m.currentProjectID = 1
		m.focus = paneTasks
		m, cmd := typeCommand(t, m, "kanban")
		updated, _ := m.Update(cmd())
		m = updated.(*model)
		m = pressKey(m, tea.Key{Code: tea.KeyEsc})
		if store.provisionCalls != 0 || m.kanbanProvisionConfirm {
			t.Fatalf("cancelled provision = calls:%d prompt:%v", store.provisionCalls, m.kanbanProvisionConfirm)
		}
	})
	t.Run("failure", func(t *testing.T) {
		store := &fakeStore{
			kanbanBoard:  vimkunja.KanbanBoard{Availability: vimkunja.KanbanBoardUnavailable},
			provisionErr: errors.New("database is locked"),
		}
		m := testModel()
		m.store = store
		m.currentProjectID = 1
		m.currentViewID = 5
		m.focus = paneTasks
		m, cmd := typeCommand(t, m, "kanban")
		updated, _ := m.Update(cmd())
		m = updated.(*model)
		updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
		m = updated.(*model)
		updated, _ = m.Update(cmd())
		m = updated.(*model)
		if store.provisionCalls != 1 || m.currentViewID != 5 || !strings.Contains(m.lastErr, "database is locked") {
			t.Fatalf("failed provision = calls:%d view:%d err:%q", store.provisionCalls, m.currentViewID, m.lastErr)
		}
	})
	t.Run("ambiguous", func(t *testing.T) {
		store := &fakeStore{kanbanBoard: vimkunja.KanbanBoard{Availability: vimkunja.KanbanBoardAmbiguous}}
		m := testModel()
		m.store = store
		m.currentProjectID = 1
		m.focus = paneTasks
		m, cmd := typeCommand(t, m, "kanban")
		updated, _ := m.Update(cmd())
		m = updated.(*model)
		if m.kanbanActive || m.kanbanProvisionConfirm || !strings.Contains(m.lastErr, "multiple manual Kanban Views") {
			t.Fatalf("ambiguous board = active:%v prompt:%v err:%q", m.kanbanActive, m.kanbanProvisionConfirm, m.lastErr)
		}
	})
}

func TestKanbanPickerMovesSelectedTaskToAnyBucket(t *testing.T) {
	store := &fakeStore{kanbanBoard: vimkunja.KanbanBoard{
		Availability: vimkunja.KanbanBoardAvailable,
		View:         vimkunja.ProjectView{ID: 10, Title: "Kanban", ViewKind: "kanban", BucketConfigurationMode: "manual"},
		Buckets: []vimkunja.KanbanBucket{
			{ID: 100, Title: "To-Do", Tasks: []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "Open"}}},
			{ID: 101, Title: "Doing"},
			{ID: 102, Title: "Done"},
		},
	}}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.focus = paneTasks
	m, cmd := typeCommand(t, m, "kanban")
	updated, _ := m.Update(cmd())
	m = updated.(*model)

	m = pressKey(m, tea.Key{Code: 'm', Text: "m"})
	if !strings.Contains(m.Render(), `Move "Open" to:`) {
		t.Fatalf("picker render = %q", m.Render())
	}
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("picker confirmation returned no move command")
	}
	_, _ = m.Update(cmd())
	if store.workflowTaskID != 1 || store.workflowBucketID != 102 {
		t.Fatalf("move = task:%d bucket:%d, want task 1 to Done", store.workflowTaskID, store.workflowBucketID)
	}
}

func TestProjectSwitchKeepsKanbanWhenTheNextProjectHasOne(t *testing.T) {
	store := &fakeStore{
		views: []vimkunja.ProjectView{
			{ID: 5, ViewKind: "list"},
			{ID: 20, ViewKind: "kanban", BucketConfigurationMode: "manual"},
		},
		kanbanBoards: map[int64]vimkunja.KanbanBoard{
			1: {Availability: vimkunja.KanbanBoardAvailable, View: vimkunja.ProjectView{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"}, Buckets: []vimkunja.KanbanBucket{{ID: 100, Title: "To-Do"}}},
			2: {Availability: vimkunja.KanbanBoardAvailable, View: vimkunja.ProjectView{ID: 20, ViewKind: "kanban", BucketConfigurationMode: "manual"}, Buckets: []vimkunja.KanbanBucket{{ID: 200, Title: "To-Do"}}},
		},
	}
	m := testModel()
	m.store = store
	m.projects = []vimkunja.Project{{ID: 1, Title: "First"}, {ID: 2, Title: "Second"}}
	m.currentProjectID = 1
	m.focus = paneTasks
	m, cmd := typeCommand(t, m, "kanban")
	updated, _ := m.Update(cmd())
	m = updated.(*model)

	m.focus = paneProjects
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	m = updated.(*model)
	updated, cmd = m.Update(cmd())
	m = updated.(*model)
	if cmd != nil {
		updated, _ = m.Update(cmd())
		m = updated.(*model)
	}
	if !m.kanbanActive || m.currentViewID != 20 {
		t.Fatalf("project switch = Kanban:%v view:%d, want next Project Kanban view 20", m.kanbanActive, m.currentViewID)
	}
}

func TestProjectSwitchFallsBackToListThenRestoresGlobalKanbanPreference(t *testing.T) {
	store := &fakeStore{
		viewsByProject: map[int64][]vimkunja.ProjectView{
			1: {{ID: 5, ViewKind: "list"}, {ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"}},
			2: {{ID: 6, ViewKind: "list"}},
		},
		kanbanBoards: map[int64]vimkunja.KanbanBoard{
			1: {Availability: vimkunja.KanbanBoardAvailable, View: vimkunja.ProjectView{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"}, Buckets: []vimkunja.KanbanBucket{{ID: 100, Title: "To-Do"}}},
		},
	}
	m := testModel()
	m.store = store
	m.projects = []vimkunja.Project{{ID: 1, Title: "First"}, {ID: 2, Title: "Second"}}
	m.currentProjectID = 1
	m.focus = paneTasks
	m, cmd := typeCommand(t, m, "kanban")
	updated, _ := m.Update(cmd())
	m = updated.(*model)

	m.focus = paneProjects
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	m = updated.(*model)
	updated, cmd = m.Update(cmd())
	m = updated.(*model)
	if cmd != nil {
		updated, _ = m.Update(cmd())
		m = updated.(*model)
	}
	if m.kanbanActive || m.currentViewID != 6 || m.projectViewPreference != projectViewKanban {
		t.Fatalf("fallback = Kanban:%v view:%d preference:%v, want List 6 with Kanban preference", m.kanbanActive, m.currentViewID, m.projectViewPreference)
	}

	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"}))
	m = updated.(*model)
	updated, cmd = m.Update(cmd())
	m = updated.(*model)
	if cmd != nil {
		updated, _ = m.Update(cmd())
		m = updated.(*model)
	}
	if !m.kanbanActive || m.currentViewID != 10 {
		t.Fatalf("return = Kanban:%v view:%d, want Kanban view 10", m.kanbanActive, m.currentViewID)
	}
}

func TestOverviewReturnPreservesKanbanTaskSelection(t *testing.T) {
	store := &fakeStore{kanbanBoard: vimkunja.KanbanBoard{
		Availability: vimkunja.KanbanBoardAvailable,
		View:         vimkunja.ProjectView{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"},
		Buckets: []vimkunja.KanbanBucket{
			{ID: 100, Title: "To-Do", Tasks: []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "First"}}},
			{ID: 101, Title: "Doing", Tasks: []vimkunja.Task{{ID: 2, ProjectID: 1, Title: "Second"}}},
		},
	}}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.focus = paneTasks
	m, cmd := typeCommand(t, m, "kanban")
	updated, _ := m.Update(cmd())
	m = updated.(*model)
	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: '-', Text: "-"}))
	m = updated.(*model)
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	m = pressKey(m, tea.Key{Code: ' ', Text: " "})
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'a', Text: "a"}))
	m = updated.(*model)
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if m.editor == nil || m.editor.id != 2 {
		t.Fatalf("returned Kanban selection = %#v, want task 2", m.editor)
	}
}

func TestListFromOverviewRestoresListInsteadOfKanbanTasks(t *testing.T) {
	store := &fakeStore{}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.currentViewID = 10
	m.kanbanActive = true
	m.kanbanBoard = vimkunja.KanbanBoard{View: vimkunja.ProjectView{ID: 10}}
	m.projectViews = []vimkunja.ProjectView{
		{ID: 5, ViewKind: "list"},
		{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"},
	}
	m.taskScope = scopeOverview

	m, cmd := typeCommand(t, m, "list")
	if cmd == nil {
		t.Fatal(":list returned no list load command")
	}
	_, _ = m.Update(cmd())
	if store.listTasksViewID != 5 || m.kanbanActive {
		t.Fatalf(":list from Overview loaded view:%d active:%v, want list view 5", store.listTasksViewID, m.kanbanActive)
	}
}

func TestKanbanInitiallySelectsAnExistingTaskAfterEmptyFirstBucket(t *testing.T) {
	store := &fakeStore{kanbanBoard: vimkunja.KanbanBoard{
		Availability: vimkunja.KanbanBoardAvailable,
		View:         vimkunja.ProjectView{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"},
		Buckets: []vimkunja.KanbanBucket{
			{ID: 100, Title: "To-Do"},
			{ID: 101, Title: "Done", Tasks: []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "Finished", Done: true}}},
		},
	}}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.focus = paneTasks
	m, cmd := typeCommand(t, m, "kanban")
	updated, _ := m.Update(cmd())
	m = updated.(*model)
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if m.editor == nil || m.editor.id != 1 {
		t.Fatalf("initial Kanban selection = %#v, want existing task 1", m.editor)
	}
}

func TestKanbanNavigationActivatesEmptyBucketAndScrollsHorizontally(t *testing.T) {
	store := &fakeStore{kanbanBoard: vimkunja.KanbanBoard{
		Availability: vimkunja.KanbanBoardAvailable,
		View:         vimkunja.ProjectView{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"},
		Buckets: []vimkunja.KanbanBucket{
			{ID: 100, Title: "To-Do", Tasks: []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "First"}}},
			{ID: 101, Title: "Empty"},
			{ID: 102, Title: "Later", Tasks: []vimkunja.Task{{ID: 2, ProjectID: 1, Title: "Second"}}},
			{ID: 103, Title: "Done"},
		},
	}}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.focus = paneTasks
	m, cmd := typeCommand(t, m, "kanban")
	updated, _ := m.Update(cmd())
	m = updated.(*model)

	m = pressKey(m, tea.Key{Code: 'l', Text: "l"}) // Empty Bucket: no selected Task.
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if m.editor != nil {
		t.Fatal("Enter opened a detail pane for an empty Bucket")
	}
	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	if !strings.Contains(m.Render(), "< Empty") || !strings.Contains(m.Render(), "Later >") {
		t.Fatalf("horizontal Board viewport = %q, want hidden-Bucket indicators", m.Render())
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if m.editor == nil || m.editor.id != 2 {
		t.Fatalf("detail after empty Bucket navigation = %#v, want task 2", m.editor)
	}
}

func TestLeaderViewSwitcherOpensKanban(t *testing.T) {
	store := &fakeStore{kanbanBoard: vimkunja.KanbanBoard{
		Availability: vimkunja.KanbanBoardAvailable,
		View:         vimkunja.ProjectView{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"},
		Buckets:      []vimkunja.KanbanBucket{{ID: 100, Title: "To-Do"}},
	}}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.focus = paneTasks

	m = pressKey(m, tea.Key{Code: ' ', Text: " "})
	m = pressKey(m, tea.Key{Code: 'v', Text: "v"})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("Space+v, then k returned no Kanban load command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if m.currentViewID != 10 {
		t.Fatalf("view id = %d, want Kanban view 10", m.currentViewID)
	}
}

func TestKanbanAdjacentMoveKeepsMovedTaskSelected(t *testing.T) {
	store := &fakeStore{kanbanBoard: vimkunja.KanbanBoard{
		Availability: vimkunja.KanbanBoardAvailable,
		View:         vimkunja.ProjectView{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"},
		Buckets: []vimkunja.KanbanBucket{
			{ID: 100, Title: "To-Do", Tasks: []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "Open"}}},
			{ID: 101, Title: "Doing"},
		},
	}}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.focus = paneTasks
	m, cmd := typeCommand(t, m, "kanban")
	updated, _ := m.Update(cmd())
	m = updated.(*model)

	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'L', Text: "L"}))
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("adjacent move returned no command")
	}
	updated, cmd = m.Update(cmd())
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("successful move returned no board reload")
	}
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if m.editor == nil || m.editor.id != 1 {
		t.Fatalf("detail after move = %#v, want moved task 1 selected", m.editor)
	}
}

func TestMovingDoneKanbanTaskUpdatesOpenDetailDoneState(t *testing.T) {
	store := &fakeStore{
		workflowDone: map[int64]bool{101: false},
		kanbanBoard: vimkunja.KanbanBoard{
			Availability: vimkunja.KanbanBoardAvailable,
			View:         vimkunja.ProjectView{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"},
			Buckets: []vimkunja.KanbanBucket{
				{ID: 100, Title: "Done", Tasks: []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "Finished", Done: true}}},
				{ID: 101, Title: "Doing"},
			},
		},
	}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.setKanbanBoard(store.kanbanBoard)
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Finished", Done: true})
	m.focus = paneTasks

	updated, moveCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'L', Text: "L"}))
	m = updated.(*model)
	updated, reloadCmd := m.Update(moveCmd())
	m = updated.(*model)
	updated, _ = m.Update(reloadCmd())
	m = updated.(*model)

	if m.editor.draft.Done || m.editor.original.Done || m.editor.draft.DoneSet {
		t.Fatalf("detail Done state after move = draft:%v original:%v changed:%v, want false, false, false", m.editor.draft.Done, m.editor.original.Done, m.editor.draft.DoneSet)
	}
}

func TestStartFooterAdvertisesHelpInsteadOfRefresh(t *testing.T) {
	footer := testModel().renderStatusBar(120)
	if !strings.Contains(footer, ":h help") {
		t.Fatalf("start footer = %q, want :h help", footer)
	}
	if strings.Contains(footer, "r refresh") {
		t.Fatalf("start footer = %q, must not advertise r refresh", footer)
	}
}

func TestKanbanFooterPrioritizesMoveActions(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.kanbanActive = true

	footer := ansi.Strip(m.renderStatusBar(120))
	for _, want := range []string{"Shift+H/L move bucket", "m choose bucket"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("Kanban footer = %q, want %q", footer, want)
		}
	}
	for _, unwanted := range []string{"Enter detail", "q quit"} {
		if strings.Contains(footer, unwanted) {
			t.Fatalf("Kanban footer = %q, must not prioritize %q", footer, unwanted)
		}
	}
}

func TestTaskDetailFormatsMetadataOutsideDueDateInsertMode(t *testing.T) {
	m := testModel()
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.openExistingEditor(vimkunja.Task{ID: 123, ProjectID: 1, Index: 42, Title: "Launch", Priority: 3, DueDate: "2026-08-02"})

	view := ansi.Strip(renderTaskDetail(m, 80, 12))
	for _, want := range []string{"Priority: High", "Due Date: tomorrow", "(DB-ID: 123, Task-Index: 42)"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail = %q, want %q", view, want)
		}
	}

	m.editor.field = fieldDueDate
	m.editor.mode = editorInsert
	view = ansi.Strip(renderTaskDetail(m, 80, 12))
	if !strings.Contains(view, "Due Date: 2026-08-02") {
		t.Fatalf("due date editor = %q, want canonical input", view)
	}
}

func TestTaskListFormatsDueDatesRelativeToToday(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.tasks = []vimkunja.Task{
		{ID: 1, Title: "Today", DueDate: "2026-08-01"},
		{ID: 2, Title: "Tomorrow", DueDate: "2026-08-02"},
		{ID: 3, Title: "Soon", DueDate: "2026-08-04"},
		{ID: 4, Title: "Late", DueDate: "2026-07-30"},
		{ID: 5, Title: "Later", DueDate: "2026-08-12"},
	}

	view := ansi.Strip(renderTasks(m, 120, 8))
	for _, want := range []string{"today", "tomorrow", "in 3 days", "2 days overdue", "12 Aug"} {
		if !strings.Contains(view, want) {
			t.Fatalf("task list = %q, want %q", view, want)
		}
	}
}

func TestNarrowTaskListDropsDueDateBeforeTruncatingTitle(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Important task", Priority: 3, DueDate: "2026-08-02"}}

	line := strings.Split(ansi.Strip(renderTasks(m, 25, 3)), "\n")[1]
	if !strings.Contains(line, "Important task") || !strings.Contains(line, "High") || strings.Contains(line, "tomorrow") {
		t.Fatalf("narrow Task list = %q, want title and priority without Due Date", line)
	}
}

func TestNarrowOverviewDropsProjectThenDueBeforeTruncatingTitle(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.taskScope = scopeOverview
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Important task", Priority: 3, ProjectIdentifier: "VIM", DueDate: "2026-08-02"}}

	line := strings.Split(ansi.Strip(renderTasks(m, 25, 3)), "\n")[1]
	if !strings.Contains(line, "Important task") || !strings.Contains(line, "High") || strings.Contains(line, "VIM") || strings.Contains(line, "tomorrow") {
		t.Fatalf("narrow Overview = %q, want title and priority without Project or Due Date", line)
	}
}

func TestCompletedDetailUsesNeutralMetadata(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, Title: "Completed", Priority: 3, DueDate: "2026-08-02", Done: true})
	m.focus = paneTasks

	view := renderTaskDetail(m, 80, 12)
	if strings.Contains(view, "38;5;"+priorityColor(3)) || strings.Count(view, "38;5;245") < 3 {
		t.Fatalf("completed Detail metadata lacks neutral styling: %q", view)
	}
}

func TestTaskListUsesNamedPriority(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Launch", Priority: 3}}

	view := ansi.Strip(renderTasks(m, 80, 4))
	if !strings.Contains(view, "High") {
		t.Fatalf("task list = %q, want named priority", view)
	}
	if strings.Contains(view, "P3") {
		t.Fatalf("task list = %q, must not show numeric priority", view)
	}
}

func TestOverviewOpensFromProjectsPane(t *testing.T) {
	store := &fakeStore{overviewTasks: []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "Open"}}}
	m := testModel()
	m.store = store
	m.currentProjectID = 1

	m = pressKey(m, tea.Key{Code: '-', Text: "-"})
	if !m.inOverview() || !m.projectsPaneCollapsed || m.focus != paneTasks {
		t.Fatalf("overview from Projects Pane = scope:%v collapsed:%v focus:%v", m.taskScope, m.projectsPaneCollapsed, m.focus)
	}
}

func TestOverviewOpensWithOpenTasksAndHidesProjects(t *testing.T) {
	store := &fakeStore{overviewTasks: []vimkunja.Task{
		{ID: 1, ProjectID: 1, ProjectIdentifier: "VIM", Title: "Open", DueDate: "2026-07-26"},
		{ID: 2, ProjectID: 2, ProjectIdentifier: "HOME", Title: "Done", Done: true},
	}}
	m := testModel()
	m.store = store
	m.focus = paneTasks
	m.currentProjectID = 1
	m.currentViewID = 10
	m.projectsPaneCollapsed = false

	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: '-', Text: "-"}))
	m = updated.(*model)
	if !m.inOverview() || !m.projectsPaneCollapsed || cmd == nil {
		t.Fatalf("overview state = scope:%v collapsed:%v cmd:%v", m.taskScope, m.projectsPaneCollapsed, cmd != nil)
	}
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if store.overviewTasksCalls != 1 || len(m.tasks) != 1 || m.tasks[0].Title != "Open" {
		t.Fatalf("overview tasks = %#v, calls:%d; want only open task", m.tasks, store.overviewTasksCalls)
	}
}

func TestOverviewFilterAndSortPicker(t *testing.T) {
	m := testModel()
	m.taskScope = scopeOverview
	m.overviewTasks = []vimkunja.Task{
		{ID: 1, Title: "Zulu", Priority: 1},
		{ID: 2, Title: "Alpha", Priority: 3},
	}
	m.applyOverview()
	m.focus = paneTasks

	m = pressKey(m, tea.Key{Code: 's', Text: "s"})
	m = pressKey(m, tea.Key{Code: 't', Text: "t"})
	if m.overviewSort != overviewTitle || m.tasks[0].Title != "Alpha" {
		t.Fatalf("sort = %v tasks=%#v, want title order", m.overviewSort, m.tasks)
	}
	m = pressKey(m, tea.Key{Code: 'f', Text: "f"})
	m = pressKey(m, tea.Key{Code: 'a', Text: "a"})
	if m.overviewFilter != overviewAll {
		t.Fatalf("filter = %v, want all", m.overviewFilter)
	}
}

func TestOverviewAlignsPriorityProjectAndTitleColumns(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.taskScope = scopeOverview
	m.tasks = []vimkunja.Task{
		{ID: 1, Title: "First title", Priority: 3, ProjectIdentifier: "VIM"},
		{ID: 2, Title: "Second title", Priority: 5, ProjectIdentifier: "HOME"},
	}

	view := ansi.Strip(renderTasks(m, 100, 4))
	lines := strings.Split(view, "\n")
	first := strings.Index(lines[1], "First title")
	second := strings.Index(lines[2], "Second title")
	if first < 0 || second < 0 || first != second {
		t.Fatalf("Overview columns = %q, title columns at %d and %d", view, first, second)
	}
}

func TestOverviewAlignsDueDatesAfterLongestVisibleTitle(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.taskScope = scopeOverview
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.tasks = []vimkunja.Task{
		{ID: 1, Title: "Short", Priority: 2, ProjectIdentifier: "VIM", DueDate: "2026-08-02"},
		{ID: 2, Title: "A much longer title", Priority: 2, ProjectIdentifier: "VIM", DueDate: "2026-08-12"},
	}

	lines := strings.Split(ansi.Strip(renderTasks(m, 100, 4)), "\n")
	first := strings.Index(lines[1], "tomorrow")
	second := strings.Index(lines[2], "12 Aug")
	if first < 0 || second < 0 || first != second {
		t.Fatalf("Overview Due Date columns = %q, Due Date columns at %d and %d", lines[1:3], first, second)
	}
}

func TestTaskListAlignsDueDatesAfterLongestVisibleTitle(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.tasks = []vimkunja.Task{
		{ID: 1, Title: "Short", Priority: 2, DueDate: "2026-08-02"},
		{ID: 2, Title: "A much longer title", Priority: 5, DueDate: "2026-08-12"},
	}

	lines := strings.Split(ansi.Strip(renderTasks(m, 100, 4)), "\n")
	first := strings.Index(lines[1], "tomorrow")
	second := strings.Index(lines[2], "12 Aug")
	if first < 0 || second < 0 || first != second {
		t.Fatalf("Task list Due Date columns = %q, Due Date columns at %d and %d", lines[1:3], first, second)
	}
}

func TestTaskListAlignsPriorityAndTitleColumns(t *testing.T) {
	m := testModel()
	m.tasks = []vimkunja.Task{
		{ID: 1, Title: "Shared title", Priority: 0},
		{ID: 2, Title: "Shared title", Priority: 3},
	}

	lines := strings.Split(ansi.Strip(renderTasks(m, 100, 4)), "\n")
	first := strings.Index(lines[1], "Shared title")
	second := strings.Index(lines[2], "Shared title")
	if first < 0 || second < 0 || lipgloss.Width(lines[1][:first]) != lipgloss.Width(lines[2][:second]) {
		t.Fatalf("Task list title columns = %q, title columns at %d and %d", lines[1:3], first, second)
	}
}

func TestFocusedTaskListHighlightsDueDateWithoutUnderline(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Launch", DueDate: "2026-08-02"}}

	line := strings.Split(renderTasks(m, 100, 3), "\n")[1]
	if strings.Contains(line, ";4;") || strings.Contains(line, "[4;") {
		t.Fatalf("focused task list task is underlined: %q", line)
	}
	if !strings.Contains(line, "\x1b[1;38;5;205mtomorrow\x1b[m") {
		t.Fatalf("focused task list due date lacks selection color: %q", line)
	}
}

func TestTaskPaneLabelsProjectTasks(t *testing.T) {
	m := testModel()
	if header := ansi.Strip(strings.Split(renderTasks(m, 80, 3), "\n")[0]); !strings.Contains(header, "Project Tasks") {
		t.Fatalf("Tasks Pane header = %q, want Project Tasks", header)
	}
}

func TestOverviewStatusBarHasNoDuplicateOrReturnToOverviewHint(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.taskScope = scopeOverview

	status := ansi.Strip(m.renderStatusBar(140))
	if strings.Count(status, "o new task") != 1 {
		t.Fatalf("Overview status = %q, want one new-task hint", status)
	}
	if strings.Contains(status, "- Overview") {
		t.Fatalf("Overview status = %q, must not offer opening the current Overview", status)
	}
}

func TestFocusedOverviewTaskHighlightsDueDateWithoutUnderline(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.taskScope = scopeOverview
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Launch", ProjectIdentifier: "VIM", DueDate: "2026-08-02"}}

	line := strings.Split(renderTasks(m, 100, 3), "\n")[1]
	if strings.Contains(line, "\x1b[4m") {
		t.Fatalf("focused overview task is underlined: %q", line)
	}
	if !strings.Contains(line, "\x1b[1;38;5;205mtomorrow\x1b[m") {
		t.Fatalf("focused overview due date lacks selection color: %q", line)
	}
}

func TestFocusedDoneTaskUsesAccentChevrons(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Completed", Done: true}}

	line := strings.Split(renderTasks(m, 80, 3), "\n")[1]
	if !strings.Contains(line, "\x1b[38;5;205m> \x1b[m") || !strings.Contains(line, "\x1b[38;5;205m <\x1b[m") {
		t.Fatalf("focused completed task lacks accent chevrons: %q", line)
	}
}

func TestOverviewDueSortPlacesEarliestTasksFirst(t *testing.T) {
	m := testModel()
	m.taskScope = scopeOverview
	m.focus = paneTasks
	m.overviewTasks = []vimkunja.Task{
		{ID: 1, Title: "Soon", DueDate: "2026-08-02"},
		{ID: 2, Title: "No due date"},
		{ID: 3, Title: "Later", DueDate: "2026-08-05"},
	}
	m.applyOverview()

	m = pressKey(m, tea.Key{Code: 's', Text: "s"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	got := []string{m.tasks[0].Title, m.tasks[1].Title, m.tasks[2].Title}
	want := []string{"Soon", "Later", "No due date"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("due sort = %v, want %v", got, want)
		}
	}
}

func TestTasksPaneKeepsSelectedTaskVisibleWithDetailOpen(t *testing.T) {
	for _, scope := range []struct {
		name      string
		taskScope taskScope
	}{
		{name: "project list", taskScope: scopeProject},
		{name: "overview", taskScope: scopeOverview},
	} {
		t.Run(scope.name, func(t *testing.T) {
			m := testModel()
			m.focus = paneTasks
			m.taskScope = scope.taskScope
			m.showDetail = true
			m.editor = &taskEditor{}
			m.tasks = make([]vimkunja.Task, 8)
			for i := range m.tasks {
				m.tasks[i] = vimkunja.Task{ID: int64(i + 1), Title: "Earlier task"}
			}
			m.tasks[len(m.tasks)-1].Title = "Selected task near the bottom"
			m.taskSelected = len(m.tasks) - 1

			if pane := ansi.Strip(renderTasks(m, 80, 8)); !strings.Contains(pane, "Selected task near the bottom") {
				t.Fatalf("Tasks Pane = %q, want selected Task visible", pane)
			}
		})
	}
}

func TestTasksPaneLeavesMarginBelowSelectedTask(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.tasks = make([]vimkunja.Task, 13)
	for i := range m.tasks {
		m.tasks[i] = vimkunja.Task{ID: int64(i + 1), Title: "Task " + string(rune('A'+i))}
	}
	m.taskSelected = 10

	pane := ansi.Strip(renderTasks(m, 80, 8))
	if !strings.Contains(pane, "Task K") || !strings.Contains(pane, "Task L") || !strings.Contains(pane, "Task M") {
		t.Fatalf("Tasks Pane = %q, want selected Task and two rows below it", pane)
	}
}

func TestOverviewNewTaskStartsWithProjectPicker(t *testing.T) {
	m := testModel()
	m.taskScope = scopeOverview
	m.focus = paneTasks
	m.projects = []vimkunja.Project{{ID: 1, Title: "Vimkunja"}}

	m = pressKey(m, tea.Key{Code: 'o', Text: "o"})
	if m.editor == nil || m.editor.mode != editorProjectPicker || m.editor.draft.ProjectID != 0 {
		t.Fatalf("overview new editor = %#v, want an empty Project Picker", m.editor)
	}
}

func TestOpeningTaskFromOverviewKeepsProjectsPaneCollapsed(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.taskScope = scopeOverview
	m.projects = []vimkunja.Project{{ID: 2, Title: "Home"}}
	m.tasks = []vimkunja.Task{{ID: 4, ProjectID: 2, Title: "Task"}}
	m.projectsPaneCollapsed = true

	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if !m.showDetail || !m.projectsPaneCollapsed || m.projectsPaneAutoShown {
		t.Fatalf("detail=%v collapsed=%v auto=%v, want Overview Detail without Projects Pane", m.showDetail, m.projectsPaneCollapsed, m.projectsPaneAutoShown)
	}
}

func TestTaskDeletionRequiresConfirmationAndRefreshesAfterSuccess(t *testing.T) {
	store := &fakeStore{}
	m := testModel()
	m.store = store
	m.focus = paneTasks
	m.currentProjectID = 1
	m.currentViewID = 1
	m.tasks = []vimkunja.Task{{ID: 7, ProjectID: 1, Title: "Delete me"}}

	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	if !m.deleteConfirm || m.deleteTaskID != 7 {
		t.Fatalf("delete confirmation = %v task=%d, want task 7", m.deleteConfirm, m.deleteTaskID)
	}
	updated, deleteCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	m = updated.(*model)
	if deleteCmd == nil {
		t.Fatal("confirmation returned no delete command")
	}
	updated, refreshCmd := m.Update(deleteCmd())
	m = updated.(*model)
	if store.deleteTaskID != 7 || m.deleteConfirm || refreshCmd == nil {
		t.Fatalf("delete = id:%d confirm:%v refresh:%v", store.deleteTaskID, m.deleteConfirm, refreshCmd != nil)
	}
}

func TestKanbanTaskDeletionUsesTheStandardConfirmation(t *testing.T) {
	store := &fakeStore{}
	m := testModel()
	m.store = store
	m.focus = paneTasks
	m.kanbanActive = true
	m.kanbanBoard = vimkunja.KanbanBoard{Buckets: []vimkunja.KanbanBucket{{ID: 1, Title: "To-Do", Tasks: []vimkunja.Task{{ID: 7, Title: "Delete me"}}}}}

	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	if !m.deleteConfirm || m.deleteTaskID != 7 {
		t.Fatalf("Kanban delete confirmation = %v task=%d, want task 7", m.deleteConfirm, m.deleteTaskID)
	}
	if view := m.Render(); !strings.Contains(view, `Delete "Delete me"?`) {
		t.Fatalf("Kanban delete modal = %q, want selected task title", view)
	}
	updated, deleteCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	m = updated.(*model)
	updated, refreshCmd := m.Update(deleteCmd())
	m = updated.(*model)
	if store.deleteTaskID != 7 || m.deleteConfirm || refreshCmd == nil {
		t.Fatalf("Kanban delete = id:%d confirm:%v refresh:%v", store.deleteTaskID, m.deleteConfirm, refreshCmd != nil)
	}
}

func TestKanbanOpenTaskHasNoHighlightedIndentation(t *testing.T) {
	m := testModel()
	m.kanbanActive = true
	m.kanbanBoard = vimkunja.KanbanBoard{Buckets: []vimkunja.KanbanBucket{{ID: 1, Title: "To-Do", Tasks: []vimkunja.Task{{ID: 7, Title: "Open"}}}}}

	line := strings.Split(ansi.Strip(renderKanbanBoard(m, 25, 5)), "\n")[3]
	if !strings.HasPrefix(line, "│ Open") {
		t.Fatalf("selected open Kanban task = %q, want an unhighlighted padding space before the title", line)
	}
}

func TestKanbanOpenTaskPaddingIsNotHighlighted(t *testing.T) {
	line := renderKanbanTaskLine(vimkunja.Task{Title: "Open"}, true, 20)
	if !strings.HasPrefix(line, " ") || strings.Index(line, "\x1b") != 1 {
		t.Fatalf("selected open Kanban task = %q, want a plain padding space before selection styling", line)
	}
}

func TestTaskDeletionConfirmationRendersCenteredModal(t *testing.T) {
	m := testModel()
	m.width, m.height = 120, 30
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 7, Title: "Delete me"}}

	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	view := m.Render()
	if !strings.Contains(view, "DELETE TASK") || !strings.Contains(view, `Delete "Delete me"?`) {
		t.Fatalf("confirmation view = %q, want centered delete modal", view)
	}
	if !strings.Contains(view, "38;5;255") {
		t.Fatalf("confirmation view = %q, want white modal border", view)
	}
	if strings.Contains(m.renderStatusBar(120), "Delete") {
		t.Fatalf("confirmation footer = %q, must remain a normal status footer", m.renderStatusBar(120))
	}
}

func TestTaskDeletionModalFitsNarrowTerminal(t *testing.T) {
	m := testModel()
	m.tasks = []vimkunja.Task{{ID: 7, Title: strings.Repeat("A very long task title ", 8)}}
	m.deleteTaskID = 7
	background := strings.Repeat(strings.Repeat(" ", 40)+"\n", 12)

	for _, line := range strings.Split(m.renderDeleteConfirmationModal(background, 40), "\n") {
		if ansi.StringWidth(line) > 40 {
			t.Fatalf("modal line width = %d, want at most 40: %q", ansi.StringWidth(line), line)
		}
	}
}

func TestTaskDeletionModalIsWidthCappedAndCenteredOnWideTerminal(t *testing.T) {
	const terminalWidth = 120
	m := testModel()
	m.tasks = []vimkunja.Task{{ID: 7, Title: "Delete me"}}
	m.deleteTaskID = 7
	background := strings.Repeat(strings.Repeat(" ", terminalWidth)+"\n", 12)

	for _, line := range strings.Split(m.renderDeleteConfirmationModal(background, terminalWidth), "\n") {
		borderStart := strings.Index(line, "\x1b[38;5;255")
		if borderStart < 0 {
			continue
		}
		left := ansi.StringWidth(line[:borderStart])
		if want := (terminalWidth - 50) / 2; left != want {
			t.Fatalf("modal starts at column %d, want %d for a 50-column centered modal", left, want)
		}
		return
	}
	t.Fatal("rendered modal has no white border")
}

func TestTaskDeletionFailureIsVisibleAndDoesNotRefresh(t *testing.T) {
	store := &fakeStore{deleteErr: errors.New("delete failed")}
	m := testModel()
	m.store = store
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 7, Title: "Keep me"}}

	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	updated, deleteCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	m = updated.(*model)
	updated, refreshCmd := m.Update(deleteCmd())
	m = updated.(*model)
	if refreshCmd != nil || m.lastErr != "delete failed" || store.listTasksCalls != 0 {
		t.Fatalf("failure = refresh:%v error:%q listCalls:%d", refreshCmd != nil, m.lastErr, store.listTasksCalls)
	}
}

func TestTaskDeletionCancellationDoesNotPersist(t *testing.T) {
	store := &fakeStore{}
	m := testModel()
	m.store = store
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 7, Title: "Keep me"}}

	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'n', Text: "n"})
	if m.deleteConfirm || store.deleteTaskID != 0 {
		t.Fatalf("cancellation = confirm:%v task:%d, want no delete", m.deleteConfirm, store.deleteTaskID)
	}
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
	updated, _ = m.Update(workflowStatusLoadedMsg{taskID: 1, status: store.workflowStatus})
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

func TestLabelsFieldAllowsNavigationAndOpensPicker(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.field = fieldLabels

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.editor.field != fieldWorkflowStatus {
		t.Fatalf("j from Labels field = %v, want Workflow Status", m.editor.field)
	}
	m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
	if m.editor.field != fieldLabels {
		t.Fatalf("k from Workflow Status = %v, want Labels", m.editor.field)
	}
	updated, pickerCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'i', Text: "i"}))
	m = updated.(*model)
	if m.labelUI == nil || pickerCmd == nil {
		t.Fatalf("i on Labels field = picker:%v command:%v, want open picker", m.labelUI != nil, pickerCmd != nil)
	}
}

func TestTaskDetailLabelsRenderAndStageRemovalInDraft(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.labels = []vimkunja.Label{{ID: 10, Title: "Errands"}, {ID: 11, Title: "Home"}}
	m.syncEditorLabelDraft()
	m.editor.original = cloneTaskDraft(m.editor.draft)
	m.editor.field = fieldLabels

	detail := ansi.Strip(renderTaskDetail(m, 80, 20))
	if due, labels, status := strings.Index(detail, "Due Date:"), strings.Index(detail, "Labels:   Errands, Home"), strings.Index(detail, "Status:"); due < 0 || labels < due || status < labels {
		t.Fatalf("Detail field order = %q, want Due Date, Labels, Status", detail)
	}
	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	m = pressKey(m, tea.Key{Code: 'x', Text: "x"})
	if got := m.editor.draft.LabelIDs; !reflect.DeepEqual(got, []int64{10}) || !m.editor.dirty() {
		t.Fatalf("staged Labels = %#v dirty:%v, want Errands only and dirty", got, m.editor.dirty())
	}
}

func TestTaskLabelLoadKeepsDetailCleanUntilAnAssignmentChanges(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})

	updated, _ := m.Update(taskLabelsLoadedMsg{taskID: 1, labels: []vimkunja.Label{{ID: 10, Title: "Existing"}}})
	m = updated.(*model)
	if m.editor.dirty() || m.editor.draft.LabelIDsSet || !reflect.DeepEqual(m.editor.draft.LabelIDs, []int64{10}) {
		t.Fatalf("loaded Labels changed Detail draft = %#v dirty:%v", m.editor.draft, m.editor.dirty())
	}
}

func TestTaskLabelLoadDoesNotOverwriteStagedDraft(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labels = []vimkunja.Label{{ID: 10, Title: "New"}}
	m.syncEditorLabelDraft()

	updated, _ := m.Update(taskLabelsLoadedMsg{taskID: 1, labels: []vimkunja.Label{{ID: 11, Title: "Stale"}}})
	m = updated.(*model)
	if got := m.editor.draft.LabelIDs; !reflect.DeepEqual(got, []int64{10}) {
		t.Fatalf("loaded Labels overwrote staged draft: %#v", got)
	}
}

func TestTaskLabelPickerCreatesAndStagesNewLabel(t *testing.T) {
	store := &fakeStore{labels: []vimkunja.Label{{ID: 10, Title: "Errands"}}}
	m := NewModel(Config{Store: store}).(*model)
	m.focus = paneDetail
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.field = fieldLabels

	m = pressKey(m, tea.Key{Code: ' ', Text: " "})
	updated, loadCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	m = updated.(*model)
	updated, _ = m.Update(loadCmd())
	m = updated.(*model)
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	for _, r := range "New" {
		m = pressKey(m, tea.Key{Code: r, Text: string(r)})
	}
	updated, createCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	updated, reloadCmd := m.Update(createCmd())
	m = updated.(*model)
	if reloadCmd != nil {
		updated, _ = m.Update(reloadCmd())
		m = updated.(*model)
	}
	if got := m.editor.draft.LabelIDs; !reflect.DeepEqual(got, []int64{99}) || !m.editor.draft.LabelIDsSet || !m.editor.dirty() {
		t.Fatalf("picker draft = %#v, want staged new Label", m.editor.draft)
	}
}

func TestLabelPickerKeepsBottomBorderWithManyLabels(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	labels := make([]vimkunja.Label, 55)
	for i := range labels {
		labels[i] = vimkunja.Label{ID: int64(i + 1), Title: fmt.Sprintf("Label %02d", i+1)}
	}
	m.labelUI = &labelUI{kind: labelUITaskPicker, labels: labels}
	background := strings.Repeat(strings.Repeat(" ", 100)+"\n", 29) + strings.Repeat(" ", 100)

	overlay := ansi.Strip(m.renderLabelOverlay(background, 100))
	if top, bottom := searchModalTopRow(overlay), searchModalBottomRow(overlay); top < 0 || bottom < 0 || bottom >= 29 {
		t.Fatalf("Label picker bounds = %d-%d, want a closed modal within the viewport", top, bottom)
	}
}

func TestLabelPickerScrollsToKeepTheSelectedLabelVisible(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	labels := make([]vimkunja.Label, 55)
	for i := range labels {
		labels[i] = vimkunja.Label{ID: int64(i + 1), Title: fmt.Sprintf("Label %02d", i+1)}
	}
	m.labelUI = &labelUI{kind: labelUITaskPicker, labels: labels}
	m.moveLabelSelection(54)

	if m.labelUI.viewport == 0 {
		t.Fatal("picker viewport did not advance for the selected Label")
	}
	rendered := ansi.Strip(strings.Join(m.renderLabelMatches(60), "\n"))
	if !strings.Contains(rendered, "> Label 55") {
		t.Fatalf("selected Label is not visible: %q", rendered)
	}
}

func TestLabelsCommandRestoresTheExactPriorContext(t *testing.T) {
	store := &fakeStore{labels: []vimkunja.Label{{ID: 10, Title: "Errands"}}}
	m := NewModel(Config{Store: store}).(*model)
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Keep selection"}}
	m.taskSelected = 0

	updated, loadCmd, err := m.commands.Execute(m, ":labels")
	if err != nil || loadCmd == nil {
		t.Fatalf(":labels = command:%v error:%v", loadCmd != nil, err)
	}
	m = updated.(*model)
	updated, _ = m.Update(loadCmd())
	m = updated.(*model)
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.labelUI != nil || m.focus != paneTasks || len(m.tasks) != 1 || m.tasks[0].Title != "Keep selection" {
		t.Fatalf("restored context = labels:%v focus:%v tasks:%#v", m.labelUI != nil, m.focus, m.tasks)
	}
}

func TestGlobalLabelOverviewCreatesRenamesAndDeletesLabels(t *testing.T) {
	store := &fakeStore{labels: []vimkunja.Label{{ID: 10, Title: "Errands"}}}
	m := NewModel(Config{Store: store}).(*model)
	updated, loadCmd, err := m.commands.Execute(m, ":labels")
	if err != nil {
		t.Fatal(err)
	}
	m = updated.(*model)
	updated, _ = m.Update(loadCmd())
	m = updated.(*model)

	m = pressKey(m, tea.Key{Code: 'o', Text: "o"})
	for _, r := range "New" {
		m = pressKey(m, tea.Key{Code: r, Text: string(r)})
	}
	updated, createCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	updated, reloadCmd := m.Update(createCmd())
	m = updated.(*model)
	updated, _ = m.Update(reloadCmd())
	m = updated.(*model)
	if store.createdLabelTitle != "New" {
		t.Fatalf("created Label = %q, want New", store.createdLabelTitle)
	}

	m.labelUI.selected = 0
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	for range "Errands" {
		m = pressKey(m, tea.Key{Code: tea.KeyBackspace})
	}
	for _, r := range "Home" {
		m = pressKey(m, tea.Key{Code: r, Text: string(r)})
	}
	updated, renameCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	updated, reloadCmd = m.Update(renameCmd())
	m = updated.(*model)
	updated, _ = m.Update(reloadCmd())
	m = updated.(*model)
	if store.renamedLabelID != 10 || store.renamedLabelTitle != "Home" {
		t.Fatalf("rename = id:%d title:%q, want 10/Home", store.renamedLabelID, store.renamedLabelTitle)
	}

	m.labelUI.selected = 1
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	updated, deleteCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	m = updated.(*model)
	updated, reloadCmd = m.Update(deleteCmd())
	m = updated.(*model)
	updated, _ = m.Update(reloadCmd())
	m = updated.(*model)
	if store.deletedLabelID != 99 {
		t.Fatalf("deleted Label = %d, want 99", store.deletedLabelID)
	}
}

func TestDetailDoneFieldTogglesOnlyTheDraft(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.field = fieldWorkflowStatus

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.editor.field != fieldDone {
		t.Fatalf("field after Workflow Status = %v, want Done", m.editor.field)
	}
	if detail := ansi.Strip(renderTaskDetail(m, 80, 20)); !strings.Contains(detail, "▸ Done:     [ ]") {
		t.Fatalf("detail = %q, want focused open Done field", detail)
	}
	m = pressKey(m, tea.Key{Code: 'x', Text: "x"})
	if !m.editor.draft.Done || !m.editor.draft.DoneSet || !m.editor.dirty() {
		t.Fatalf("Done draft = %#v, want dirty done state", m.editor.draft)
	}
}

func TestNewTaskDetailOmitsDoneField(t *testing.T) {
	m := testModel()
	m.openNewEditor()
	m.editor.field = fieldWorkflowStatus

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.editor.field != fieldDescription {
		t.Fatalf("field after Workflow Status = %v, want Description", m.editor.field)
	}
	if detail := ansi.Strip(renderTaskDetail(m, 80, 20)); strings.Contains(detail, "Done:") {
		t.Fatalf("new task detail includes Done field: %q", detail)
	}
}

func TestSavingDetailPersistsDoneWithOtherDraftChanges(t *testing.T) {
	store := &fakeStore{}
	m := NewModel(Config{Store: store}).(*model)
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.field = fieldDone
	m = pressKey(m, tea.Key{Code: 'x', Text: "x"})
	m.editor.draft.Title = "Renamed"

	m, saveCmd := typeCommand(t, m, "w")
	if saveCmd == nil {
		t.Fatal(":w returned no save command")
	}
	updated, _ := m.Update(saveCmd())
	m = updated.(*model)
	if store.updatedDraft.Title != "Renamed" || !store.updatedDraft.Done || !store.updatedDraft.DoneSet {
		t.Fatalf("saved draft = %#v, want renamed done Task", store.updatedDraft)
	}
	if m.editor == nil || m.editor.dirty() || m.editor.draft.DoneSet {
		t.Fatalf("saved Done editor = %#v, want clean persisted Done state", m.editor)
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

func TestHelpUsesTaglineInsteadOfRepeatingPaneTitle(t *testing.T) {
	const want = "Vimkunja is a task manager in your terminal facilitating Vim Motions."
	if len(helpLines) == 0 || helpLines[0] != want {
		t.Fatalf("first help line = %q, want %q", helpLines[0], want)
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

func TestDetailEditorEscapeClosesCleanEditorAndRefusesDirtyEditor(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.showDetail || m.editor != nil {
		t.Fatal("Esc did not close a clean detail editor")
	}

	m.openNewEditor()
	m.editor.draft.Title = "Unsaved"
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if !m.showDetail || m.editor == nil || m.lastErr == "" {
		t.Fatalf("dirty Esc state = visible:%v editor:%v error:%q, want editor retained with error", m.showDetail, m.editor != nil, m.lastErr)
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

func TestAdvertisedUnimplementedCommandsExplainTheirStatus(t *testing.T) {
	for _, command := range []string{"table", "gantt"} {
		t.Run(command, func(t *testing.T) {
			m, cmd := typeCommand(t, testModel(), command)
			if cmd != nil {
				t.Fatalf(":%s returned an unexpected command", command)
			}
			if m.lastErr != "Not yet implemented" {
				t.Fatalf(":%s error = %q, want not-yet-implemented status", command, m.lastErr)
			}
			if footer := m.renderStatusBar(120); !strings.Contains(footer, "Error: Not yet implemented") {
				t.Fatalf(":%s footer = %q, want explicit status", command, footer)
			}
		})
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

func TestHomeEndAliasesNavigateProjectsTasksAndDescription(t *testing.T) {
	for _, tc := range []struct {
		name     string
		model    *model
		position func(*model) int
		last     int
	}{
		{
			name: "projects",
			model: &model{
				focus:           paneProjects,
				projects:        []vimkunja.Project{{ID: 1}, {ID: 2}, {ID: 3}},
				projectSelected: 1,
			},
			position: func(m *model) int { return m.projectSelected },
			last:     2,
		},
		{
			name: "tasks",
			model: &model{
				focus:        paneTasks,
				tasks:        []vimkunja.Task{{ID: 1}, {ID: 2}, {ID: 3}},
				taskSelected: 1,
			},
			position: func(m *model) int { return m.taskSelected },
			last:     2,
		},
		{
			name: "description",
			model: func() *model {
				m := testModel()
				m.currentProjectID = 1
				m.openNewEditor()
				m.editor.field = fieldDescription
				m.editor.draft.Description = "first\nsecond"
				m.editor.cursor = 1
				return m
			}(),
			position: func(m *model) int { return m.editor.cursor },
			last:     len([]rune("first\nsecond")),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pressKey(tc.model, tea.Key{Code: tea.KeyEnd})
			if got := tc.position(m); got != tc.last {
				t.Fatalf("position after End = %d, want %d", got, tc.last)
			}

			m = pressKey(m, tea.Key{Code: tea.KeyHome})
			if got := tc.position(m); got != 0 {
				t.Fatalf("position after Home = %d, want 0", got)
			}
		})
	}
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

func TestHelpDocumentsProjectHierarchyBindings(t *testing.T) {
	help := strings.Join(helpLines, "\n")
	for _, binding := range []string{"zo / zc / za", "zM / zR", "Shift+J / Shift+K"} {
		if !strings.Contains(help, binding) {
			t.Fatalf("help does not document Project hierarchy binding %q", binding)
		}
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

func TestDescriptionCursorTracksNewDoneFieldAfterSavingNewTask(t *testing.T) {
	m := NewModel(Config{Store: &fakeStore{}}).(*model)
	m.width, m.height = 100, 40
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "Task"
	m.editor.field = fieldDescription
	m.editor.draft.Description = "hello"
	m.editor.cursor = 2

	m, saveCmd := typeCommand(t, m, "w")
	updated, _ := m.Update(saveCmd())
	m = updated.(*model)
	cursor := m.View().Cursor
	lines := strings.Split(ansi.Strip(m.Render()), "\n")
	if cursor == nil || cursor.Position.Y >= len(lines) || !strings.Contains(lines[cursor.Position.Y], "hello") {
		t.Fatalf("description cursor = %#v, rendered line = %q, want description text", cursor, lines[cursor.Position.Y])
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
