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
	toggleViewID         int64
	deleteErr            error
	deleteTaskID         int64
	workflowStatus       vimkunja.WorkflowStatus
	workflowStatusErr    error
	workflowStatusViewID int64
	kanbanBoard          vimkunja.KanbanBoard
	kanbanBoards         map[int64]vimkunja.KanbanBoard
	kanbanBoardErr       error
	kanbanLoadProjectID  int64
	kanbanLoadViewID     int64
	views                []vimkunja.ProjectView
	viewsByProject       map[int64][]vimkunja.ProjectView
	tasksByView          map[int64][]vimkunja.Task
	provisionedBoard     vimkunja.KanbanBoard
	provisionErr         error
	provisionCalls       int
	workflowTaskID       int64
	workflowBucketID     int64
	workflowViewID       int64
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
	updatedTaskID        int64
	updatedTaskTitle     string
	taskComments         map[int64][]vimkunja.TaskComment
	commentErr           error
	savedCommentDraft    vimkunja.TaskCommentDraft
	savedCommentTask     vimkunja.TaskDraft
	deletedCommentTaskID int64
	deletedCommentID     int64
	undoStates           map[int64]vimkunja.TaskUndoState
	restoreUndoErr       error
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

func (f *fakeStore) LoadManualKanbanBoard(_ context.Context, projectID int64, viewID ...int64) (vimkunja.KanbanBoard, error) {
	f.kanbanLoadProjectID = projectID
	f.kanbanLoadViewID = 0
	if len(viewID) > 0 {
		f.kanbanLoadViewID = viewID[0]
	}
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

func (f *fakeStore) ListTaskComments(_ context.Context, taskID int64) ([]vimkunja.TaskComment, error) {
	return append([]vimkunja.TaskComment(nil), f.taskComments[taskID]...), f.commentErr
}

func (f *fakeStore) SaveTaskWithComment(_ context.Context, draft vimkunja.TaskDraft, comment vimkunja.TaskCommentDraft) (vimkunja.Task, vimkunja.TaskComment, error) {
	f.savedCommentTask, f.savedCommentDraft = draft, comment
	if f.commentErr != nil {
		return vimkunja.Task{}, vimkunja.TaskComment{}, f.commentErr
	}
	created := vimkunja.TaskComment{ID: 99, TaskID: draft.ID, HTML: "<p>" + comment.Text + "</p>", Author: "Local User", CanDelete: true, Created: time.Date(2026, 9, 4, 13, 30, 0, 0, time.UTC), Updated: time.Date(2026, 9, 4, 13, 30, 0, 0, time.UTC)}
	f.taskComments[draft.ID] = append(f.taskComments[draft.ID], created)
	return vimkunja.Task{ID: draft.ID, ProjectID: draft.ProjectID, Title: draft.Title, Description: draft.Description, Priority: draft.Priority, DueDate: draft.DueDate}, created, nil
}

func (f *fakeStore) DeleteTaskComment(_ context.Context, taskID, commentID int64) error {
	f.deletedCommentTaskID, f.deletedCommentID = taskID, commentID
	if f.commentErr != nil {
		return f.commentErr
	}
	comments := f.taskComments[taskID]
	for index, comment := range comments {
		if comment.ID != commentID {
			continue
		}
		if !comment.CanDelete {
			return errors.New("only the Comment author may delete it")
		}
		f.taskComments[taskID] = append(comments[:index:index], comments[index+1:]...)
		return nil
	}
	return errors.New("Comment not found")
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
		state := f.undoState(taskID)
		state.LabelIDs = append([]int64(nil), labelIDs...)
		f.setUndoState(state)
	}
	return f.labelErr
}

func (f *fakeStore) GetTaskWorkflowStatus(_ context.Context, _, _ int64, viewID ...int64) (vimkunja.WorkflowStatus, error) {
	f.workflowStatusViewID = 0
	if len(viewID) > 0 {
		f.workflowStatusViewID = viewID[0]
	}
	return f.workflowStatus, f.workflowStatusErr
}

func (f *fakeStore) SetTaskWorkflowStatus(_ context.Context, taskID, bucketID int64, viewID ...int64) error {
	f.workflowTaskID = taskID
	f.workflowBucketID = bucketID
	f.workflowViewID = 0
	if len(viewID) > 0 {
		f.workflowViewID = viewID[0]
	}
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
	state := f.undoState(taskID)
	assignedViewID := f.workflowViewID
	state.BucketAssignments = []vimkunja.TaskBucketAssignment{{ViewID: assignedViewID, BucketID: bucketID}}
	if done, ok := f.workflowDone[bucketID]; ok {
		state.Done = done
	}
	f.setUndoState(state)
	return nil
}

func (f *fakeStore) ToggleTaskDone(_ context.Context, taskID int64, done bool, viewID ...int64) error {
	f.toggleTaskID = taskID
	f.toggleDone = done
	f.toggleViewID = 0
	if len(viewID) > 0 {
		f.toggleViewID = viewID[0]
	}
	if f.toggleErr == nil {
		state := f.undoState(taskID)
		state.Done = done
		f.setUndoState(state)
	}
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
	state := f.undoState(draft.ID)
	state.ProjectID, state.Title, state.Description = draft.ProjectID, draft.Title, draft.Description
	state.Priority, state.DueDate = draft.Priority, draft.DueDate
	if draft.DoneSet {
		state.Done = draft.Done
	}
	if draft.LabelIDsSet {
		state.LabelIDs = append([]int64(nil), draft.LabelIDs...)
	}
	f.setUndoState(state)
	return vimkunja.Task{ID: draft.ID, ProjectID: draft.ProjectID, Title: draft.Title, Done: state.Done}, nil
}

func (f *fakeStore) UpdateTaskTitle(_ context.Context, taskID int64, title string) error {
	f.updatedTaskID, f.updatedTaskTitle = taskID, title
	if f.updateErr == nil {
		state := f.undoState(taskID)
		state.Title = title
		f.setUndoState(state)
	}
	return f.updateErr
}

func (f *fakeStore) TaskUndoState(_ context.Context, taskID int64) (vimkunja.TaskUndoState, error) {
	return f.undoState(taskID), nil
}

func (f *fakeStore) RestoreTaskUndoState(_ context.Context, expected, target vimkunja.TaskUndoState) error {
	if f.restoreUndoErr != nil {
		return f.restoreUndoErr
	}
	if !reflect.DeepEqual(f.undoState(expected.TaskID), expected) {
		return errors.New("Task changed since this action")
	}
	f.setUndoState(target)
	return nil
}

func (f *fakeStore) undoState(taskID int64) vimkunja.TaskUndoState {
	if state, ok := f.undoStates[taskID]; ok {
		return state
	}
	for _, tasks := range f.tasksByView {
		for _, task := range tasks {
			if task.ID == taskID {
				return vimkunja.TaskUndoState{TaskID: task.ID, ProjectID: task.ProjectID, Identifier: task.Identifier, Title: task.Title, Description: task.Description, Priority: task.Priority, DueDate: task.DueDate, Done: task.Done}
			}
		}
	}
	for _, task := range f.overviewTasks {
		if task.ID == taskID {
			return vimkunja.TaskUndoState{TaskID: task.ID, ProjectID: task.ProjectID, Identifier: task.Identifier, Title: task.Title, Description: task.Description, Priority: task.Priority, DueDate: task.DueDate, Done: task.Done}
		}
	}
	return vimkunja.TaskUndoState{TaskID: taskID, ProjectID: 1}
}

func (f *fakeStore) setUndoState(state vimkunja.TaskUndoState) {
	if f.undoStates == nil {
		f.undoStates = make(map[int64]vimkunja.TaskUndoState)
	}
	f.undoStates[state.TaskID] = state
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

func TestProjectsPaneShowsScrollbarOnlyForActualOverflow(t *testing.T) {
	m := testModel()
	m.projects = []vimkunja.Project{
		{ID: 1, Title: "One", Position: 1},
		{ID: 2, Title: "Two", Position: 2},
		{ID: 3, Title: "Three", Position: 3},
		{ID: 4, Title: "Four", Position: 4},
	}

	overflowing := ansi.Strip(renderProjects(m, 18, 4))
	if !strings.Contains(overflowing, "┬") || !strings.Contains(overflowing, "█") {
		t.Fatalf("overflowing Projects Pane lacks bounded scrollbar: %q", overflowing)
	}
	for _, line := range strings.Split(overflowing, "\n") {
		if got := lipgloss.Width(line); got != 18 {
			t.Fatalf("scrollbar line width = %d, want 18: %q", got, line)
		}
	}

	nonOverflowing := ansi.Strip(renderProjects(m, 18, 5))
	if strings.Contains(nonOverflowing, "│") || strings.Contains(nonOverflowing, "█") {
		t.Fatalf("non-overflowing Projects Pane shows scrollbar: %q", nonOverflowing)
	}
}

func TestTasksPaneScrollbarTracksSelectionAndFooterPosition(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.width, m.height = 120, 10
	m.tasks = []vimkunja.Task{
		{ID: 1, Title: "One"},
		{ID: 2, Title: "Two"},
		{ID: 3, Title: "Three"},
		{ID: 4, Title: "Four"},
		{ID: 5, Title: "Five"},
	}

	atTop := ansi.Strip(renderTasks(m, 24, 4))
	if !strings.Contains(atTop, "┬") || !strings.Contains(atTop, "█") {
		t.Fatalf("overflowing Tasks Pane lacks bounded scrollbar: %q", atTop)
	}
	if footer := ansi.Strip(m.renderStatusBar(120)); !strings.Contains(footer, "0% │") {
		t.Fatalf("top footer = %q, want right-aligned 0%%", footer)
	}

	m.taskSelected = len(m.tasks) - 1
	atBottom := ansi.Strip(renderTasks(m, 24, 4))
	if strings.Index(atBottom, "█") <= strings.Index(atTop, "█") {
		t.Fatalf("scrollbar Thumb did not advance: top=%q bottom=%q", atTop, atBottom)
	}
	if footer := ansi.Strip(m.renderStatusBar(120)); !strings.Contains(footer, "100% │") {
		t.Fatalf("bottom footer = %q, want right-aligned 100%%", footer)
	}
}

func TestProjectsPaneUnderlinesHeader(t *testing.T) {
	m := testModel()

	view := renderProjects(m, 24, 5)
	header := " " + lipgloss.NewStyle().Bold(true).Underline(true).Render("Projects") + " "
	if !strings.Contains(view, header) {
		t.Fatalf("Projects Pane header lacks underline: %q", view)
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

func TestInfoPaneScrollbarReservesRailWithoutCreatingOverflow(t *testing.T) {
	pane := &infoPane{title: "HELP", lines: []string{"one", "two", "three"}}

	overflowing := ansi.Strip(renderInfoPane(pane, 16, 4, true))
	if !strings.Contains(overflowing, "┬") || !strings.Contains(overflowing, "█") {
		t.Fatalf("overflowing informational Pane lacks bounded scrollbar: %q", overflowing)
	}
	nonOverflowing := ansi.Strip(renderInfoPane(pane, 16, 5, true))
	if strings.Contains(nonOverflowing, "│") || strings.Contains(nonOverflowing, "█") {
		t.Fatalf("non-overflowing informational Pane shows scrollbar: %q", nonOverflowing)
	}

	// This content would gain a third wrapped row if the rail were reserved,
	// but two full-width rows fit exactly and therefore must not create a rail.
	selfInduced := ansi.Strip(renderInfoPane(&infoPane{title: "H", lines: []string{"123456789"}}, 5, 4, true))
	if strings.Contains(selfInduced, "│") || strings.Contains(selfInduced, "█") {
		t.Fatalf("rail created its own informational Pane overflow: %q", selfInduced)
	}
}

func TestNarrowFooterKeepsOneRowAndFallsBackToHelp(t *testing.T) {
	m := testModel()
	m.focus = paneProjects
	const width = 20
	footer := ansi.Strip(m.renderStatusBar(width))
	lines := strings.Split(footer, "\n")
	if len(lines) != 3 || !strings.Contains(footer, ":h") {
		t.Fatalf("narrow footer = %q, want one bordered row with help fallback", footer)
	}
	for _, line := range lines {
		if got := lipgloss.Width(line); got != width {
			t.Fatalf("narrow footer line width = %d, want %d: %q", got, width, line)
		}
	}
}

func TestFooterHintPriorityKeepsHelpBeforeSecondaryActions(t *testing.T) {
	segments := []string{"primary", "navigate", ":h help", "secondary"}
	footer := renderFooterHints(28, "", segments, false)
	for _, want := range []string{"primary", "navigate", ":h"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("footer = %q, missing priority segment %q", footer, want)
		}
	}
	if strings.Contains(footer, "secondary") {
		t.Fatalf("footer = %q, retained secondary action before help", footer)
	}
}

func TestFooterHintFallbackYieldsLabelAndStopsBelowHelpWidth(t *testing.T) {
	segments := []string{"primary", ":h help", "secondary"}
	if got := renderFooterHints(3, "[Tasks]", segments, false); got != ":h" {
		t.Fatalf("label fallback = %q, want compact help without label", got)
	}
	if got := renderFooterHints(1, "", segments, false); got != "" {
		t.Fatalf("sub-help footer = %q, want empty", got)
	}
}

func TestFooterStaticStatePriorityMatrix(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*model)
		want  []string
	}{
		{
			name:  "projects",
			setup: func(m *model) { m.focus = paneProjects },
			want:  []string{"j/k select", "h/l fold/tree", ":h help"},
		},
		{
			name:  "overview",
			setup: func(m *model) { m.focus, m.taskScope = paneTasks, scopeOverview },
			want:  []string{"j/k select", "Enter detail", ":h help"},
		},
		{
			name:  "kanban",
			setup: func(m *model) { m.focus, m.kanbanActive = paneTasks, true },
			want:  []string{"Shift+H/L move bucket", "m choose bucket", "j/k navigate", ":h help"},
		},
		{
			name:  "detail normal",
			setup: func(m *model) { m.focus, m.editor = paneDetail, &taskEditor{} },
			want:  []string{"i edit", "j/k navigate", ":h help"},
		},
		{
			name:  "help",
			setup: func(m *model) { m.focus = paneHelp },
			want:  []string{"Ctrl+W k tasks", ":q close pane"},
		},
		{
			name:  "info",
			setup: func(m *model) { m.focus = paneInfo },
			want:  []string{"q/esc close pane"},
		},
		{
			name:  "search normal",
			setup: func(m *model) { m.searchOpen, m.searchMode = true, searchNormalMode },
			want:  []string{"SEARCH [NORMAL]  j/k navigate", "i/a edit", ":h help"},
		},
		{
			name:  "leader",
			setup: func(m *model) { m.leaderArmed = true },
			want:  []string{"[LEADER] a Overview/return", "o toggle projects pane", "v views"},
		},
		{
			name:  "views",
			setup: func(m *model) { m.viewSwitchArmed = true },
			want:  []string{"[VIEWS] l list", "k Kanban", "t table", "g Gantt"},
		},
		{
			name:  "sort",
			setup: func(m *model) { m.overviewPrompt = overviewPromptSort },
			want:  []string{"SORT  d Due", "p Priority", "t Title", "Esc cancel"},
		},
		{
			name:  "filter",
			setup: func(m *model) { m.overviewPrompt = overviewPromptFilter },
			want:  []string{"FILTER  o open", "a all", "v overdue", "t due today", "Esc cancel"},
		},
		{
			name:  "ctrl-w",
			setup: func(m *model) { m.ctrlWArmed = true },
			want:  []string{"Ctrl-W", "h/j/k/l change pane", "Ctrl-W/w cycle"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			tc.setup(m)
			footer := ansi.Strip(m.renderStatusBar(140))
			for _, want := range tc.want {
				if !strings.Contains(footer, want) {
					t.Fatalf("footer = %q, missing %q", footer, want)
				}
			}
		})
	}
}

func TestNarrowFooterPreservesInputAndErrorTails(t *testing.T) {
	m := testModel()
	m.commandMode = true
	m.commandInput = "a-very-long-command"
	m.commandCursor = len([]rune(m.commandInput))
	footer := ansi.Strip(m.renderStatusBar(25))
	if !strings.Contains(footer, ":…mmand│") {
		t.Fatalf("command footer = %q, want input tail", footer)
	}

	m.commandMode = false
	m.lastErr = "unknown command: a-very-long-command"
	footer = ansi.Strip(m.renderStatusBar(25))
	if !strings.Contains(footer, "Error: …") || !strings.Contains(footer, "mmand") {
		t.Fatalf("error footer = %q, want prefixed error tail", footer)
	}
}

func TestScrollPercentageYieldsLowerPriorityHintsAfterThreeCommands(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.width, m.height = 66, 10
	m.tasks = []vimkunja.Task{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}}

	footer := ansi.Strip(m.renderStatusBar(66))
	if !strings.Contains(footer, "0%") || !strings.Contains(footer, "Ctrl+W cycles") || !strings.Contains(footer, "Enter detail") || strings.Contains(footer, "x toggle") {
		t.Fatalf("scrolling footer = %q, want percentage after three hints", footer)
	}
}

func TestScrollingFooterKeepsHintsWhenThreeActionsDoNotFit(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.width, m.height = 60, 10
	m.tasks = []vimkunja.Task{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}}

	footer := ansi.Strip(m.renderStatusBar(60))
	if strings.Contains(footer, "0%") {
		t.Fatalf("scrolling footer = %q, must yield percentage below three-action threshold", footer)
	}
	if !strings.Contains(footer, "Ctrl+W cycles") || !strings.Contains(footer, "Enter detail") {
		t.Fatalf("scrolling footer = %q, want retained hints", footer)
	}
}

func TestModalFramesNeverExceedTheirViewportWidth(t *testing.T) {
	for _, width := range []int{32, 40, 60, 100} {
		t.Run(fmt.Sprintf("%d columns", width), func(t *testing.T) {
			layout := modalBounds(width, 20, 80, 9)
			frame := ansi.Strip(renderModalFrame([]string{"A modal row"}, layout))
			for _, line := range strings.Split(frame, "\n") {
				if got := lipgloss.Width(line); got > width {
					t.Fatalf("modal line width = %d, want <= %d: %q", got, width, line)
				}
			}
		})
	}
}

func TestNarrowModalFallbacksReplacePaneWithoutOverflow(t *testing.T) {
	const width, height = 20, 6
	background := strings.TrimSuffix(strings.Repeat(strings.Repeat("x", width)+"\n", height), "\n")
	m := testModel()
	m.width, m.height = width, height
	m.projects = []vimkunja.Project{{ID: 1, Title: "A very long Project title"}}
	m.currentProjectID = 1
	m.labelUI = &labelUI{kind: labelUITaskPicker}

	modals := []struct {
		name   string
		render func() string
	}{
		{"task deletion", func() string { return m.renderDeleteConfirmationModal(background, width) }},
		{"kanban provisioning", func() string { return m.renderKanbanProvisionConfirmationModal(background, width) }},
		{"global search", func() string { return m.renderSearchOverlay(background, width) }},
		{"labels", func() string { return m.renderLabelOverlay(background, width) }},
	}
	for _, modal := range modals {
		t.Run(modal.name, func(t *testing.T) {
			lines := strings.Split(ansi.Strip(modal.render()), "\n")
			if len(lines) != height {
				t.Fatalf("line count = %d, want %d: %q", len(lines), height, lines)
			}
			for _, line := range lines {
				if got := lipgloss.Width(line); got != width {
					t.Fatalf("line width = %d, want %d: %q", got, width, line)
				}
			}
		})
	}
}

func TestConfirmationBodyScrollsQuestionAndPinsActions(t *testing.T) {
	layout := modalBounds(32, 9, 50, 9)
	question := "This deliberately long confirmation question must scroll without hiding its answer actions."
	body, limit := confirmationBody("CONFIRM", question, "y accept  •  n / Esc cancel", layout, 1)
	if limit == 0 {
		t.Fatal("confirmation question should overflow its reserved message region")
	}
	if got, want := len(body), layout.contentHeight; got != want {
		t.Fatalf("confirmation rows = %d, want fixed content height %d: %q", got, want, body)
	}
	if got := body[len(body)-1]; !strings.Contains(got, "y") || !strings.Contains(got, "n") {
		t.Fatalf("confirmation actions = %q, want pinned answer actions", got)
	}
}

func TestFooterScrollReservationCountsHelpAsThirdSegment(t *testing.T) {
	segments := []string{"j/k select", "h/l fold/tree", ":h help", "z fold commands"}
	width := lipgloss.Width(composeFooter("[Projects]", segments[:3]))
	if !footerHasRoomForCommands(width, "[Projects]", segments, 3) {
		t.Fatalf("three complete segments including help should fit in %d columns", width)
	}
}

func TestFallbackModalUsesActualBackgroundHeight(t *testing.T) {
	const width = 20
	m := testModel()
	m.width, m.height = width, 6
	m.tasks = []vimkunja.Task{{ID: 1, Title: "A long task title for confirmation"}}
	m.deleteTaskID = 1
	for _, height := range []int{5, 6, 7} {
		t.Run(fmt.Sprintf("%d rows", height), func(t *testing.T) {
			background := strings.TrimSuffix(strings.Repeat(strings.Repeat("x", width)+"\n", height), "\n")
			lines := strings.Split(ansi.Strip(m.renderDeleteConfirmationModal(background, width)), "\n")
			if got := len(lines); got != height {
				t.Fatalf("fallback lines = %d, want background height %d", got, height)
			}
			for _, line := range lines {
				if got := lipgloss.Width(line); got != width {
					t.Fatalf("fallback line width = %d, want %d: %q", got, width, line)
				}
			}
		})
	}
}

func TestSearchAndLabelFallbackUseBackgroundHeight(t *testing.T) {
	const width, backgroundHeight = 20, 6
	background := strings.TrimSuffix(strings.Repeat(strings.Repeat("x", width)+"\n", backgroundHeight), "\n")
	m := testModel()
	m.width, m.height = width, 100 // Deliberately disagrees with the Pane background.
	m.labelUI = &labelUI{kind: labelUITaskPicker}

	for name, render := range map[string]func() string{
		"search": func() string { return m.renderSearchOverlay(background, width) },
		"labels": func() string { return m.renderLabelOverlay(background, width) },
	} {
		t.Run(name, func(t *testing.T) {
			lines := strings.Split(ansi.Strip(render()), "\n")
			if got := len(lines); got != backgroundHeight {
				t.Fatalf("fallback lines = %d, want background height %d", got, backgroundHeight)
			}
		})
	}
}

func TestFooterStatesNeverExceedViewportWidth(t *testing.T) {
	states := []struct {
		name  string
		setup func(*model)
	}{
		{"projects", func(m *model) { m.focus = paneProjects }},
		{"tasks", func(m *model) { m.focus = paneTasks }},
		{"command", func(m *model) { m.commandMode, m.commandInput = true, "a-very-long-command" }},
		{"error", func(m *model) { m.lastErr = "a-very-long-error-message" }},
	}
	for _, state := range states {
		for _, width := range []int{5, 20, 40, 80} {
			t.Run(fmt.Sprintf("%s-%d", state.name, width), func(t *testing.T) {
				m := testModel()
				state.setup(m)
				for _, line := range strings.Split(ansi.Strip(m.renderStatusBar(width)), "\n") {
					if got := lipgloss.Width(line); got > width {
						t.Fatalf("footer line width = %d, want <= %d: %q", got, width, line)
					}
				}
			})
		}
	}
}

func TestNarrowPickerRowsDropChromeAndFit(t *testing.T) {
	rows := renderPickerChoicesFitted([]string{"A very long Project or Workflow Status title"}, 0, tuiTheme.selectionAccent, 20)
	for _, row := range rows {
		if got := lipgloss.Width(ansi.Strip(row)); got > 20 {
			t.Fatalf("picker row width = %d, want <= 20: %q", got, row)
		}
		if strings.Contains(row, "╭") || strings.Contains(row, "│") {
			t.Fatalf("narrow picker retained chrome: %q", row)
		}
	}
}

func TestStatusBarOmitsChromeBelowMinimumWidth(t *testing.T) {
	m := testModel()
	for width := 1; width < 5; width++ {
		if got := m.renderStatusBar(width); got != "" {
			t.Fatalf("status width %d = %q, want no overflowing chrome", width, got)
		}
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

func TestEscapeAtRootShowsQuitCommandOnly(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if !m.quitHint || !strings.Contains(ansi.Strip(m.renderStatusBar(100)), ":q quit") {
		t.Fatalf("root Escape = hint:%v footer:%q, want :q quit", m.quitHint, m.renderStatusBar(100))
	}

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.quitHint {
		t.Fatal("a subsequent key retained the transient quit hint")
	}

	m.focus = paneDetail
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.mode = editorProjectPicker
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.quitHint {
		t.Fatal("picker Escape must cancel the picker, not show the root quit hint")
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
		ID: 7, ProjectID: 1, ProjectIdentifier: "HOME", ProjectTitle: "Home", Title: "Buy coffee", Description: "<p>Fresh coffee beans</p>", Done: true,
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
	if len(m.searchResults) != 2 || m.searchResults[0].Kind != vimkunja.SearchTaskResult || m.searchResults[1].Kind != vimkunja.SearchDescriptionResult || !m.searchResults[0].Done || !m.searchResults[1].Done {
		t.Fatalf("search results = %#v, want completed Task and Description hits", m.searchResults)
	}
	view := ansi.Strip(m.Render())
	if !strings.Contains(view, "TASKS") || !strings.Contains(view, "DESCRIPTIONS") || strings.Contains(view, "<p>") {
		t.Fatalf("Global Search render = %q, want typed groups without HTML", view)
	}
}

func TestGlobalSearchCompletedTasksAreMutedAndMarked(t *testing.T) {
	m := testModel()
	m.searchSelected = -1
	m.searchResults = []vimkunja.SearchResult{
		{Kind: vimkunja.SearchDescriptionResult, Primary: "Foo", Snippet: "Completed Description", Done: true},
		{Kind: vimkunja.SearchTaskResult, Primary: "Food"},
	}

	lines := m.renderSearchResults(40)
	if len(lines) != 5 {
		t.Fatalf("search lines = %q, want Description and Task groups", lines)
	}
	if got := ansi.Strip(lines[1]); !strings.HasSuffix(got, "[DONE]") || lipgloss.Width(got) != 40 {
		t.Fatalf("completed Task row = %q, want right-aligned [DONE] at width 40", got)
	}
	if !strings.Contains(lines[1], "38;5;"+tuiTheme.muted) {
		t.Fatalf("completed Task row lacks the shared muted colour: %q", lines[1])
	}
	if !strings.Contains(lines[2], "38;5;"+tuiTheme.muted) {
		t.Fatalf("completed Description preview lacks the shared muted colour: %q", lines[2])
	}
	if got := ansi.Strip(lines[4]); strings.Contains(got, "[DONE]") {
		t.Fatalf("open Task row = %q, must not show a completion marker", got)
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
	if !strings.Contains(lines[1], "38;5;"+tuiTheme.selectionAccent) {
		t.Fatalf("selected search cursor lacks pink selection color: %q", lines[1])
	}
	if !strings.Contains(lines[2], "38;5;"+tuiTheme.inactiveSelection) {
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
	if !strings.Contains(view, "\x1b[38;5;"+tuiTheme.muted+"m") {
		t.Fatalf("Kanban bucket border lacks dim white color: %q", view)
	}
}

func TestKanbanSelectionScrollsItsBucketVertically(t *testing.T) {
	m := testModel()
	m.kanbanActive = true
	m.focus = paneTasks
	m.kanbanBoard = vimkunja.KanbanBoard{Buckets: []vimkunja.KanbanBucket{{
		ID: 1, Title: "To-Do", Tasks: []vimkunja.Task{
			{ID: 1, Title: "One"}, {ID: 2, Title: "Two"}, {ID: 3, Title: "Three"}, {ID: 4, Title: "Four"},
		},
	}}}
	m.kanbanTaskSelected = 3

	view := ansi.Strip(renderKanbanBoard(m, 25, 5))
	if !strings.Contains(view, "Four") {
		t.Fatalf("selected card is outside the short Bucket viewport: %q", view)
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
	if view := ansi.Strip(m.Render()); !strings.Contains(view, "CREATE KANBAN VIEW") || !strings.Contains(view, "Create standard Kanban View for Home?") {
		t.Fatalf("kanban confirmation = %q", view)
	}
	if strings.Contains(m.renderStatusBar(100), "Create standard Kanban") {
		t.Fatalf("kanban confirmation leaked into footer = %q", m.renderStatusBar(100))
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

func TestKanbanCommandsCaptureSelectedView(t *testing.T) {
	store := &fakeStore{workflowStatus: vimkunja.WorkflowStatus{Availability: vimkunja.WorkflowStatusAvailable}}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.currentViewID = 10
	m.kanbanActive = true
	m.projectViews = []vimkunja.ProjectView{
		{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"},
		{ID: 20, ViewKind: "kanban", BucketConfigurationMode: "manual"},
	}

	boardCmd := m.loadKanbanBoard(1)
	statusCmd := m.loadWorkflowStatus(1, 7)
	moveCmd := m.moveKanbanTask(7, 100)
	m.currentViewID = 20
	boardCmd()
	statusCmd()
	moveCmd()

	if store.kanbanLoadViewID != 10 || store.workflowStatusViewID != 10 || store.workflowViewID != 10 {
		t.Fatalf("captured view IDs = board:%d status:%d move:%d, want 10", store.kanbanLoadViewID, store.workflowStatusViewID, store.workflowViewID)
	}
}

func TestKanbanDoneToggleCapturesSelectedView(t *testing.T) {
	store := &fakeStore{}
	m := testModel()
	m.store = store
	m.currentViewID = 10
	m.kanbanActive = true
	m.focus = paneTasks
	m.kanbanBoard = vimkunja.KanbanBoard{Buckets: []vimkunja.KanbanBucket{{Tasks: []vimkunja.Task{{ID: 7}}}}}
	m.kanbanBucketSelected = 0
	m.kanbanTaskSelected = 0

	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
	m = updated.(*model)
	m.currentViewID = 20
	cmd()
	if store.toggleViewID != 10 {
		t.Fatalf("toggle view ID = %d, want captured view 10", store.toggleViewID)
	}
}

func TestTaskMovedFromKanbanLoadsDestinationProjectViewsBeforeBoard(t *testing.T) {
	store := &fakeStore{
		viewsByProject: map[int64][]vimkunja.ProjectView{
			2: {{ID: 20, ViewKind: "kanban", BucketConfigurationMode: "manual"}},
		},
		kanbanBoards: map[int64]vimkunja.KanbanBoard{
			2: {Availability: vimkunja.KanbanBoardAvailable, View: vimkunja.ProjectView{ID: 20, ViewKind: "kanban", BucketConfigurationMode: "manual"}},
		},
	}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.currentViewID = 10
	m.kanbanActive = true
	m.projectViewPreference = projectViewKanban
	m.projectViews = []vimkunja.ProjectView{{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"}}

	m.editor = &taskEditor{draft: vimkunja.TaskDraft{ID: 7, ProjectID: 2, Title: "Moved"}}
	m.savingEditor = m.editor
	updated, cmd := m.Update(taskSavedMsg{task: vimkunja.Task{ID: 7, ProjectID: 2, Title: "Moved"}, editor: m.editor, draft: m.editor.draft})
	m = updated.(*model)
	if cmd == nil || m.kanbanActive || m.currentViewID != 0 {
		t.Fatalf("post-save state = cmd:%v Kanban:%v view:%d", cmd != nil, m.kanbanActive, m.currentViewID)
	}
	updated, cmd = m.Update(cmd())
	m = updated.(*model)
	if cmd == nil {
		t.Fatal("destination views did not trigger a board load")
	}
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if !m.kanbanActive || m.currentViewID != 20 || store.kanbanLoadProjectID != 2 || store.kanbanLoadViewID != 20 {
		t.Fatalf("destination board = active:%v current view:%d loaded project:%d view:%d", m.kanbanActive, m.currentViewID, store.kanbanLoadProjectID, store.kanbanLoadViewID)
	}
}

func TestMultiViewKanbanBoardShowsActiveViewTitle(t *testing.T) {
	m := testModel()
	m.projectViews = []vimkunja.ProjectView{
		{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"},
		{ID: 20, ViewKind: "kanban", BucketConfigurationMode: "manual"},
	}
	m.kanbanBoard = vimkunja.KanbanBoard{
		View:    vimkunja.ProjectView{ID: 20, Title: "Personal"},
		Buckets: []vimkunja.KanbanBucket{{ID: 200, Title: "To-Do"}},
	}
	if rendered := renderKanbanBoard(m, 80, 20); !strings.Contains(rendered, "Kanban: Personal") {
		t.Fatalf("board does not identify active view: %q", rendered)
	}
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

func TestDoneToggleKeepsMovedKanbanTaskSelectedAndInDetail(t *testing.T) {
	store := &fakeStore{kanbanBoard: vimkunja.KanbanBoard{
		Availability: vimkunja.KanbanBoardAvailable,
		View:         vimkunja.ProjectView{ID: 10, ViewKind: "kanban", BucketConfigurationMode: "manual"},
		Buckets: []vimkunja.KanbanBucket{
			{ID: 100, Title: "To-Do", Tasks: []vimkunja.Task{{ID: 2, ProjectID: 1, Title: "Move me"}}},
			{ID: 101, Title: "Done"},
		},
	}}
	m := testModel()
	m.store = store
	m.currentProjectID = 1
	m.setKanbanBoard(store.kanbanBoard)
	m.openExistingEditor(store.kanbanBoard.Buckets[0].Tasks[0])
	m.focus = paneTasks

	updated, toggleCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
	m = updated.(*model)
	if toggleCmd == nil || m.pendingKanbanTaskID != 2 {
		t.Fatalf("Done toggle = command:%v pending:%d, want task 2 reload selection", toggleCmd != nil, m.pendingKanbanTaskID)
	}
	store.kanbanBoard.Buckets = []vimkunja.KanbanBucket{
		{ID: 100, Title: "To-Do"},
		{ID: 101, Title: "Done", Tasks: []vimkunja.Task{{ID: 2, ProjectID: 1, Title: "Move me", Done: true}}},
	}
	updated, reloadCmd := m.Update(toggleCmd())
	m = updated.(*model)
	if reloadCmd == nil {
		t.Fatal("successful Done toggle returned no board reload")
	}
	updated, _ = m.Update(reloadCmd())
	m = updated.(*model)
	if task, ok := m.selectedKanbanTask(); !ok || task.ID != 2 || m.editor == nil || m.editor.id != 2 {
		t.Fatalf("Done toggle state = selected:%#v detail:%#v, want moved task 2", task, m.editor)
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

func TestLocalStartOfDayUsesLocalMidnightOutsideUTC(t *testing.T) {
	zone := time.FixedZone("UTC+2", 2*60*60)
	now := time.Date(2026, time.August, 1, 12, 0, 0, 0, zone)
	want := time.Date(2026, time.August, 1, 0, 0, 0, 0, zone)
	if got := localStartOfDay(now); !got.Equal(want) {
		t.Fatalf("local start of day = %s, want %s", got, want)
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

func TestNarrowTaskListHidesMetadataBeforeViolatingTitleBudget(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Important task", Priority: 3, DueDate: "2026-08-02"}}

	line := strings.Split(ansi.Strip(renderTasks(m, 25, 3)), "\n")[1]
	if !strings.Contains(line, "Important task") || strings.Contains(line, "High") || strings.Contains(line, "tomorrow") {
		t.Fatalf("narrow Task list = %q, want title without metadata", line)
	}
}

func TestNarrowOverviewHidesMetadataBeforeViolatingTitleBudget(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.taskScope = scopeOverview
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Important task", Priority: 3, ProjectIdentifier: "VIM", DueDate: "2026-08-02"}}

	line := strings.Split(ansi.Strip(renderTasks(m, 25, 3)), "\n")[1]
	if !strings.Contains(line, "Important task") || strings.Contains(line, "High") || strings.Contains(line, "VIM") || strings.Contains(line, "tomorrow") {
		t.Fatalf("narrow Overview = %q, want title without metadata", line)
	}
}

func TestTaskMetadataLayoutStaysStableWhileScrollingPastLongTitle(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.now = func() time.Time { return time.Date(2026, time.August, 1, 12, 0, 0, 0, time.Local) }
	m.tasks = []vimkunja.Task{
		{ID: 1, Title: "First", Priority: 3, DueDate: "2026-08-02"},
		{ID: 2, Title: "Second", Priority: 3, DueDate: "2026-08-02"},
		{ID: 3, Title: "A deliberately long Task title that must truncate", Priority: 3, DueDate: "2026-08-02"},
	}

	initial := ansi.Strip(renderTasks(m, 35, 3))
	m.taskSelected = 2
	scrolled := ansi.Strip(renderTasks(m, 35, 3))
	for _, view := range []string{initial, scrolled} {
		if !strings.Contains(view, "High") || strings.Contains(view, "tomorrow") {
			t.Fatalf("metadata layout changed while scrolling: %q", view)
		}
	}
	if !strings.Contains(scrolled, "A deliberately long T…") {
		t.Fatalf("long title = %q, want end ellipsis", scrolled)
	}
}

func TestTruncatedTaskTitleUsesMutedEllipsis(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 1, Title: "A deliberately long Task title that must truncate", Priority: 3}}

	view := renderTasks(m, 30, 3)
	if !strings.Contains(view, "…") || !strings.Contains(view, "38;5;"+tuiTheme.muted) {
		t.Fatalf("truncated title = %q, want muted ellipsis", view)
	}
}

func TestCompletedDetailUsesNeutralMetadata(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, Title: "Completed", Priority: 3, DueDate: "2026-08-02", Done: true})
	m.focus = paneTasks

	view := renderTaskDetail(m, 80, 12)
	if strings.Contains(view, "38;5;"+priorityColor(3)) || strings.Count(view, "38;5;"+tuiTheme.muted) < 3 {
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
	if !strings.Contains(line, "\x1b[1;38;5;"+tuiTheme.selectionAccent+"mtomorrow\x1b[m") {
		t.Fatalf("focused task list due date lacks selection color: %q", line)
	}
}

func TestTaskPaneLabelsProjectTasks(t *testing.T) {
	m := testModel()
	header := strings.Split(renderTasks(m, 80, 3), "\n")[0]
	want := " " + lipgloss.NewStyle().Bold(true).Underline(true).Render("Project Tasks") + " "
	if !strings.Contains(header, want) {
		t.Fatalf("Tasks Pane header = %q, want underlined Project Tasks", header)
	}
}

func TestOverviewPaneUnderlinesHeader(t *testing.T) {
	m := testModel()
	m.taskScope = scopeOverview

	header := strings.Split(renderTasks(m, 80, 3), "\n")[0]
	title := fmt.Sprintf("Overview · %s · %s", m.overviewSortLabel(), m.overviewFilterLabel())
	want := " " + lipgloss.NewStyle().Bold(true).Underline(true).Render(title) + " "
	if !strings.Contains(header, want) {
		t.Fatalf("Overview Pane header = %q, want underlined title", header)
	}
}

func TestTaskDetailUnderlinesUncoloredHeader(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})

	header := strings.Split(renderTaskDetail(m, 80, 20), "\n")[0]
	want := " " + lipgloss.NewStyle().Bold(true).Underline(true).Render("TASK EDITOR") + " "
	if !strings.Contains(header, want) {
		t.Fatalf("Task Detail header = %q, want uncolored underlined title", header)
	}
	if strings.Contains(header, "38;5;"+tuiTheme.selectionAccent) {
		t.Fatalf("Task Detail header retains selection color: %q", header)
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
	if !strings.Contains(line, "\x1b[1;38;5;"+tuiTheme.selectionAccent+"mtomorrow\x1b[m") {
		t.Fatalf("focused overview due date lacks selection color: %q", line)
	}
}

func TestFocusedDoneTaskUsesAccentChevrons(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 1, Title: "Completed", Done: true}}

	line := strings.Split(renderTasks(m, 80, 3), "\n")[1]
	if !strings.Contains(line, "\x1b[38;5;"+tuiTheme.selectionAccent+"m> \x1b[m") || !strings.Contains(line, "\x1b[38;5;"+tuiTheme.selectionAccent+"m <\x1b[m") {
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

func TestTaskDeletionClosesOpenDetailAfterSuccess(t *testing.T) {
	store := &fakeStore{}
	m := testModel()
	m.store = store
	m.focus = paneTasks
	m.currentProjectID = 1
	m.currentViewID = 1
	m.tasks = []vimkunja.Task{{ID: 7, ProjectID: 1, Title: "Delete me"}}
	m.showDetail = true
	m.editor = &taskEditor{id: 7, draft: vimkunja.TaskDraft{ID: 7, ProjectID: 1, Title: "Delete me"}}

	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	updated, deleteCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	m = updated.(*model)
	updated, refreshCmd := m.Update(deleteCmd())
	m = updated.(*model)
	if refreshCmd == nil {
		t.Fatal("successful deletion returned no refresh command")
	}
	updated, _ = m.Update(refreshCmd())
	m = updated.(*model)

	if store.deleteTaskID != 7 {
		t.Fatalf("delete = id:%d, want task 7", store.deleteTaskID)
	}
	if m.showDetail || m.editor != nil || m.focus != paneTasks {
		t.Fatalf("after deletion = detail:%v editor:%v focus:%v, want closed detail and Tasks Pane focus", m.showDetail, m.editor != nil, m.focus)
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

func TestActiveOverlaysUseTheSharedBorderAccent(t *testing.T) {
	background := strings.Repeat(strings.Repeat(" ", 80)+"\n", 24)
	for name, overlay := range map[string]string{
		"search": func() string {
			m := testModel()
			m.searchOpen = true
			return m.renderSearchOverlay(background, 80)
		}(),
		"labels": func() string {
			m := testModel()
			m.labelUI = &labelUI{kind: labelUITaskPicker}
			return m.renderLabelOverlay(background, 80)
		}(),
		"delete": func() string {
			m := testModel()
			return m.renderDeleteConfirmationModal(background, 80)
		}(),
	} {
		if !strings.Contains(overlay, "38;5;"+tuiTheme.selectionAccent) {
			t.Fatalf("%s overlay lacks shared selection-accent border: %q", name, overlay)
		}
	}
}

func TestCommandModeHighlightsStatusBorder(t *testing.T) {
	m := testModel()
	m.commandMode = true
	status := m.renderStatusBar(80)
	if !strings.Contains(status, "38;5;"+tuiTheme.selectionAccent) {
		t.Fatalf("command status border lacks selection accent: %q", status)
	}
}

func TestFocusedHelpUsesTheBottomPaneBorder(t *testing.T) {
	m := testModel()
	m.helpPane = &infoPane{title: "Help"}
	m.focus = paneHelp
	view := m.Render()
	if !strings.Contains(view, "38;5;"+tuiTheme.selectionAccent) {
		t.Fatalf("focused Help lacks selection-accent border: %q", view)
	}
}

func TestKanbanSelectionSyncsAnOpenCleanDetail(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.kanbanActive = true
	m.kanbanBoard = vimkunja.KanbanBoard{Buckets: []vimkunja.KanbanBucket{{
		ID: 1, Title: "To-Do", Tasks: []vimkunja.Task{
			{ID: 7, ProjectID: 1, Title: "First"},
			{ID: 8, ProjectID: 1, Title: "Second"},
		},
	}}}
	m.openExistingEditor(m.kanbanBoard.Buckets[0].Tasks[0])
	m.focus = paneTasks

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.editor == nil || m.editor.id != 8 || m.focus != paneTasks {
		t.Fatalf("Kanban selection sync = editor:%#v focus:%v, want task 8 with Tasks focus", m.editor, m.focus)
	}
}

func TestEmptyKanbanDetailCentersItsMessage(t *testing.T) {
	line := strings.Split(renderEmptyKanbanDetail(30, 5), "\n")[2]
	if !strings.HasPrefix(line, "       No task selected") {
		t.Fatalf("empty Detail message = %q, want horizontally centered", line)
	}
}

func TestKanbanSelectionDoesNotReplaceDirtyDetailAndEmptyBucketShowsEmptyDetail(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.currentProjectID = 1
	m.kanbanActive = true
	m.kanbanBoard = vimkunja.KanbanBoard{Buckets: []vimkunja.KanbanBucket{
		{ID: 1, Title: "To-Do", Tasks: []vimkunja.Task{{ID: 7, ProjectID: 1, Title: "First"}, {ID: 8, ProjectID: 1, Title: "Second"}}},
		{ID: 2, Title: "Empty"},
	}}
	m.openExistingEditor(m.kanbanBoard.Buckets[0].Tasks[0])
	m.editor.draft.Title = "Unsaved"
	m.focus = paneTasks

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.editor.id != 7 {
		t.Fatalf("dirty Detail was replaced by selection: %#v", m.editor)
	}
	m.editor.draft = cloneTaskDraft(m.editor.original)
	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	if m.editor != nil || !strings.Contains(ansi.Strip(m.Render()), "No task selected") {
		t.Fatalf("empty Bucket Detail = editor:%#v render:%q", m.editor, m.Render())
	}
	m.focus = paneDetail
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	if m.editor == nil || m.editor.draft.ProjectID != 1 || m.editor.draft.WorkflowBucketID != 2 {
		t.Fatalf("empty Detail new task = %#v, want project 1 bucket 2", m.editor)
	}
	m.editor.draft.Title = "Discard me"
	updated, _ := forceQuitCommand(m, nil)
	m = updated.(*model)
	if !m.showDetail || m.editor != nil || m.focus != paneDetail {
		t.Fatalf("discarded empty-Bucket task = detail:%v editor:%#v focus:%v", m.showDetail, m.editor, m.focus)
	}
}

func TestListPreviewKeepsTasksFocus(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 7, ProjectID: 1, Title: "Preview"}}

	m = pressKey(m, tea.Key{Code: 'v', Text: "v"})
	if !m.showDetail || m.editor == nil || m.editor.id != 7 || m.focus != paneTasks {
		t.Fatalf("preview = detail:%v editor:%#v focus:%v", m.showDetail, m.editor, m.focus)
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
	if !strings.Contains(view, "38;5;"+tuiTheme.selectionAccent) {
		t.Fatalf("confirmation view = %q, want selection-accent modal border", view)
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
		borderStart := strings.Index(line, "\x1b[38;5;"+tuiTheme.selectionAccent)
		if borderStart < 0 {
			continue
		}
		left := ansi.StringWidth(line[:borderStart])
		if want := (terminalWidth - 50) / 2; left != want {
			t.Fatalf("modal starts at column %d, want %d for a 50-column centered modal", left, want)
		}
		return
	}
	t.Fatal("rendered modal has no selection-accent border")
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

func TestNewTaskLoadsWorkflowStatusesForItsProject(t *testing.T) {
	store := &fakeStore{workflowStatus: vimkunja.WorkflowStatus{
		Availability:  vimkunja.WorkflowStatusAvailable,
		CurrentBucket: vimkunja.WorkflowBucket{ID: 10, Title: "To-Do"},
		Buckets:       []vimkunja.WorkflowBucket{{ID: 10, Title: "To-Do"}, {ID: 11, Title: "Doing"}},
	}}
	m := &model{store: store, currentProjectID: 1}
	cmd := m.openNewEditor()
	if cmd == nil || !m.editor.workflowStatusLoading {
		t.Fatal("new editor did not start workflow status loading")
	}
	updated, _ := m.Update(cmd())
	m = updated.(*model)
	if got := m.editor.workflowStatusLabel(); got != "To-Do" {
		t.Fatalf("new task workflow status = %q, want To-Do", got)
	}
	if m.editor.draft.WorkflowBucketID != 10 {
		t.Fatalf("new task workflow bucket = %d, want 10", m.editor.draft.WorkflowBucketID)
	}
}

func TestStaleWorkflowStatusLoadCannotOverwriteReopenedEditor(t *testing.T) {
	store := &fakeStore{workflowStatus: vimkunja.WorkflowStatus{
		Availability:  vimkunja.WorkflowStatusAvailable,
		CurrentBucket: vimkunja.WorkflowBucket{ID: 11, Title: "Doing"},
		Buckets: []vimkunja.WorkflowBucket{
			{ID: 10, Title: "To-Do"}, {ID: 11, Title: "Doing"},
		},
	}}
	m := &model{store: store, currentProjectID: 1}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	oldEditor := m.editor
	oldLoad := m.loadWorkflowStatus(1, 1)

	m.openExistingEditor(vimkunja.Task{ID: 2, ProjectID: 1, Title: "Other"})
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	newEditor := m.editor

	updated, _ := m.Update(oldLoad())
	m = updated.(*model)
	if m.editor != newEditor || !m.editor.workflowStatusLoading {
		t.Fatalf("stale workflow status changed reopened editor: old=%p current=%p loading=%v", oldEditor, m.editor, m.editor.workflowStatusLoading)
	}

	updated, _ = m.Update(m.loadWorkflowStatus(1, 1)())
	m = updated.(*model)
	if got := m.editor.workflowStatusLabel(); got != "Doing" {
		t.Fatalf("current workflow status = %q, want Doing", got)
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

func TestTaskDetailHighlightsLabelsFieldAndSelectedLabel(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.labels = []vimkunja.Label{{ID: 10, Title: "Errands"}, {ID: 11, Title: "Home"}}
	m.editor.field = fieldLabels
	m.editor.labelSelected = 1

	detail := renderTaskDetail(m, 80, 20)
	fieldPrefix := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Bold(true).Render("▸ Labels:   ")
	if !strings.Contains(detail, fieldPrefix) {
		t.Fatalf("active Labels field lacks its highlight: %q", detail)
	}
	selected := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Bold(true).Render("Home")
	if !strings.Contains(detail, selected) {
		t.Fatalf("selected Label lacks its own highlight: %q", detail)
	}
}

func TestTaskDetailHighlightsEmptyLabelsFieldWithoutHighlightingNone(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.field = fieldLabels

	detail := renderTaskDetail(m, 80, 20)
	selectedLabel := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Bold(true).Render("▸ Labels:   ")
	if !strings.Contains(detail, selectedLabel+"none") {
		t.Fatalf("empty Labels field lacks selected label and neutral none: %q", detail)
	}
}

func TestTaskLabelPickerRefusesQuitWithUnsavedLabelChanges(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.labels = []vimkunja.Label{{ID: 10, Title: "Errands"}}
	m.syncEditorLabelDraft()
	m.editor.original = cloneTaskDraft(m.editor.draft)
	m.openTaskLabelPicker()
	m.labelUI.loading = false
	m.toggleEditorLabel(vimkunja.Label{ID: 11, Title: "Home"})

	for _, key := range []tea.Key{{Code: ':', Text: ":"}, {Code: 'q', Text: "q"}, {Code: tea.KeyEnter}} {
		m = pressKey(m, key)
	}
	if m.labelUI == nil || m.lastErr == "" {
		t.Fatalf("Task Label Picker :q state = picker:%v error:%q, want retained changed picker", m.labelUI != nil, m.lastErr)
	}
}

func TestTaskLabelPickerQuitClosesWhenLabelsAreUnchanged(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.labels = []vimkunja.Label{{ID: 10, Title: "Errands"}}
	m.syncEditorLabelDraft()
	m.openTaskLabelPicker()
	m.labelUI.loading = false

	for _, key := range []tea.Key{{Code: ':', Text: ":"}, {Code: 'q', Text: "q"}, {Code: tea.KeyEnter}} {
		m = pressKey(m, key)
	}
	if m.labelUI != nil {
		t.Fatal("Task Label Picker :q did not close an unchanged picker")
	}
}

func TestTaskLabelPickerForceQuitRestoresOnlyItsLabelChanges(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.labels = []vimkunja.Label{{ID: 10, Title: "Errands"}}
	m.syncEditorLabelDraft()
	m.editor.original = cloneTaskDraft(m.editor.draft)
	m.editor.draft.Title = "Unsaved title"
	m.openTaskLabelPicker()
	m.labelUI.loading = false
	m.toggleEditorLabel(vimkunja.Label{ID: 10, Title: "Errands"})

	for _, key := range []tea.Key{{Code: ':', Text: ":"}, {Code: 'q', Text: "q"}, {Code: '!', Text: "!"}, {Code: tea.KeyEnter}} {
		m = pressKey(m, key)
	}
	if m.labelUI != nil || m.editor.draft.Title != "Unsaved title" || !reflect.DeepEqual(m.editor.draft.LabelIDs, []int64{10}) || !m.editor.dirty() {
		t.Fatalf("Task Label Picker :q! state = picker:%v draft:%#v dirty:%v", m.labelUI != nil, m.editor.draft, m.editor.dirty())
	}
}

func TestTaskLabelPickerWriteSavesOnlyLabelsAndKeepsOtherDraftChanges(t *testing.T) {
	store := &fakeStore{}
	m := NewModel(Config{Store: store}).(*model)
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.labels = []vimkunja.Label{{ID: 10, Title: "Errands"}}
	m.syncEditorLabelDraft()
	m.editor.original = cloneTaskDraft(m.editor.draft)
	m.editor.draft.Title = "Unsaved title"
	m.openTaskLabelPicker()
	m.labelUI.loading = false
	m.toggleEditorLabel(vimkunja.Label{ID: 11, Title: "Home"})

	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Code: 'w', Text: "w"})
	updated, saveCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if saveCmd == nil {
		t.Fatal("Task Label Picker :w returned no label-save command")
	}
	updated, _ = m.Update(saveCmd())
	m = updated.(*model)

	if got := editorLabelIDs(store.taskLabels[1]); !reflect.DeepEqual(got, []int64{10, 11}) || store.updatedDraft.Title != "" {
		t.Fatalf("Task Label Picker :w writes = labels:%#v task:%#v", got, store.updatedDraft)
	}
	if m.labelUI == nil || m.editor.draft.Title != "Unsaved title" || !m.editor.dirty() {
		t.Fatalf("Task Label Picker :w state = picker:%v draft:%#v dirty:%v", m.labelUI != nil, m.editor.draft, m.editor.dirty())
	}
	for _, key := range []tea.Key{{Code: ':', Text: ":"}, {Code: 'q', Text: "q"}, {Code: tea.KeyEnter}} {
		m = pressKey(m, key)
	}
	if m.labelUI != nil || m.editor.draft.Title != "Unsaved title" || !m.editor.dirty() {
		t.Fatalf("Task Label Picker :w then :q state = picker:%v draft:%#v dirty:%v", m.labelUI != nil, m.editor.draft, m.editor.dirty())
	}
}

func TestTaskLabelPickerWriteQuitSavesLabelsAndClosesPicker(t *testing.T) {
	store := &fakeStore{}
	m := NewModel(Config{Store: store}).(*model)
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.labels = []vimkunja.Label{{ID: 10, Title: "Errands"}}
	m.syncEditorLabelDraft()
	m.editor.original = cloneTaskDraft(m.editor.draft)
	m.openTaskLabelPicker()
	m.labelUI.loading = false
	m.toggleEditorLabel(vimkunja.Label{ID: 11, Title: "Home"})

	for _, key := range []tea.Key{{Code: ':', Text: ":"}, {Code: 'w', Text: "w"}, {Code: 'q', Text: "q"}} {
		m = pressKey(m, key)
	}
	updated, saveCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if saveCmd == nil {
		t.Fatal("Task Label Picker :wq returned no label-save command")
	}
	updated, _ = m.Update(saveCmd())
	m = updated.(*model)

	if m.labelUI != nil || !reflect.DeepEqual(editorLabelIDs(store.taskLabels[1]), []int64{10, 11}) {
		t.Fatalf("Task Label Picker :wq state = picker:%v labels:%#v", m.labelUI != nil, store.taskLabels[1])
	}
}

func TestTaskLabelPickerDelegatesWriteQuitAll(t *testing.T) {
	store := &fakeStore{}
	m := NewModel(Config{Store: store}).(*model)
	m.projects = []vimkunja.Project{{ID: 1, Title: "Project"}}
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "Saved before quit"
	m.labelUI = &labelUI{kind: labelUITaskPicker}

	for _, key := range []tea.Key{
		{Code: ':', Text: ":"},
		{Code: 'w', Text: "w"},
		{Code: 'q', Text: "q"},
		{Code: 'a', Text: "a"},
		{Code: tea.KeyEnter},
	} {
		m = pressKey(m, key)
	}
	if m.lastErr != "" {
		t.Fatalf("Task Label Picker :wqa error = %q", m.lastErr)
	}
	if !m.quitAfterSave {
		t.Fatal("Task Label Picker :wqa did not arm quit-after-save")
	}
}

func TestDetailFooterExplainsSelectedLabelRemoval(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.labels = []vimkunja.Label{{ID: 10, Title: "Errands"}}
	m.editor.field = fieldLabels

	footer := ansi.Strip(m.renderStatusBar(120))
	for _, hint := range []string{"h/l select Label", "x remove Label", "i pick/edit"} {
		if !strings.Contains(footer, hint) {
			t.Fatalf("Labels footer = %q, want %q", footer, hint)
		}
	}
}

func TestLabelsFieldOmitsTheNativeTextCursor(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.editor.labels = []vimkunja.Label{{ID: 10, Title: "Errands"}}
	m.editor.field = fieldLabels

	if cursor := m.View().Cursor; cursor != nil {
		t.Fatalf("Labels field has an unrelated native cursor: %#v", cursor)
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

func TestTaskLabelPickerPrefersTheSelectedMatchOverCreatingTheQuery(t *testing.T) {
	store := &fakeStore{labels: []vimkunja.Label{{ID: 11, Title: "AZB"}}}
	m := NewModel(Config{Store: store}).(*model)
	m.focus = paneDetail
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.labelUI = &labelUI{kind: labelUITaskPicker, labels: store.labels}

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: 'A', Text: "A"})
	m = pressKey(m, tea.Key{Code: 'Z', Text: "Z"})

	background := strings.Repeat(strings.Repeat(" ", 100)+"\n", 29) + strings.Repeat(" ", 100)
	overlay := ansi.Strip(m.renderLabelOverlay(background, 100))
	for _, want := range []string{`+ Create label "AZ"`, "─────────────────────", "AZB"} {
		if !strings.Contains(overlay, want) {
			t.Fatalf("filtered picker = %q, want %q", overlay, want)
		}
	}

	m = pressKey(m, tea.Key{Code: tea.KeyDown})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if cmd != nil {
		t.Fatal("Enter on selected matching Label returned a create command")
	}
	if got := m.editor.draft.LabelIDs; !reflect.DeepEqual(got, []int64{11}) {
		t.Fatalf("selected Label assignment = %#v, want AZB", got)
	}
	if store.createdLabelTitle != "" {
		t.Fatalf("created Label = %q, want no new Label", store.createdLabelTitle)
	}
}

func TestTaskLabelPickerEscEntersBrowseModeAndOPrefersCreate(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m.labelUI = &labelUI{kind: labelUITaskPicker, labels: []vimkunja.Label{{ID: 11, Title: "AZB"}}}

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: 'A', Text: "A"})
	m = pressKey(m, tea.Key{Code: 'Z', Text: "Z"})
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	if m.labelUI == nil || m.labelUI.inputMode || m.labelUI.input != "AZ" {
		t.Fatalf("first Esc = picker:%v inputMode:%v input:%q, want open browse mode with filter", m.labelUI != nil, m.labelUI.inputMode, m.labelUI.input)
	}
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.labelUI.createSelected || m.labelUI.selected != 0 {
		t.Fatalf("j selected = create:%v label:%d, want first matching Label", m.labelUI.createSelected, m.labelUI.selected)
	}
	m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
	if !m.labelUI.createSelected {
		t.Fatal("k did not return to the Create Label action")
	}
	m = pressKey(m, tea.Key{Code: 'o', Text: "o"})
	if !m.labelUI.createSelected {
		t.Fatal("o did not select the Create Label action")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	if m.labelUI != nil {
		t.Fatal("second Esc did not close the picker")
	}
}

func TestTaskLabelPickerHighlightsOnlyTheSelectedCreateAction(t *testing.T) {
	m := testModel()
	m.labelUI = &labelUI{
		kind:           labelUITaskPicker,
		labels:         []vimkunja.Label{{ID: 11, Title: "AZB"}},
		input:          "AZ",
		inputAction:    labelInputTaskPicker,
		selected:       0,
		createSelected: false,
	}
	accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render(`+ Create label "AZ"`)
	if line := m.renderLabelCreateOption(50)[0]; strings.Contains(line, accent) {
		t.Fatalf("unselected Create action has selection styling: %q", line)
	}
	m.labelUI.createSelected = true
	if line := m.renderLabelCreateOption(50)[0]; !strings.Contains(line, accent) {
		t.Fatalf("selected Create action lacks selection styling: %q", line)
	}
}

func TestLabelOverlayUsesHeadingStyle(t *testing.T) {
	for _, test := range []struct {
		name, title string
		kind        labelUIKind
	}{
		{name: "task picker", title: "LABEL PICKER", kind: labelUITaskPicker},
		{name: "global Labels", title: "LABELS", kind: labelUIGlobalOverview},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := testModel()
			m.width, m.height = 100, 30
			m.labelUI = &labelUI{kind: test.kind}

			background := strings.Repeat(strings.Repeat(" ", 100)+"\n", 29) + strings.Repeat(" ", 100)
			overlay := m.renderLabelOverlay(background, 100)
			want := lipgloss.NewStyle().Bold(true).Underline(true).Render(test.title)
			if !strings.Contains(overlay, want) {
				t.Fatalf("Label overlay title = %q, want shared heading style", overlay)
			}
		})
	}
}

func TestLabelPickerShowsScrollbarForOverflowingMatches(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	labels := make([]vimkunja.Label, 55)
	for i := range labels {
		labels[i] = vimkunja.Label{ID: int64(i + 1), Title: fmt.Sprintf("Label %02d", i+1)}
	}
	m.labelUI = &labelUI{kind: labelUIGlobalOverview, labels: labels}
	background := strings.Repeat(strings.Repeat(" ", 100)+"\n", 29) + strings.Repeat(" ", 100)

	overlay := ansi.Strip(m.renderLabelOverlay(background, 100))
	for _, glyph := range []string{"│", "█", "┬", "┴"} {
		if !strings.Contains(overlay, glyph) {
			t.Fatalf("overflowing Labels overlay lacks scrollbar glyph %q: %q", glyph, overlay)
		}
	}
	for _, line := range strings.Split(overlay, "\n") {
		if !strings.Contains(line, "LABELS") && !strings.Contains(line, "o create") {
			continue
		}
		if strings.ContainsAny(line, "│█┬┴") {
			t.Fatalf("Labels header/instructions have scrollbar glyph: %q", line)
		}
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

func TestCommandPromptHistoryRecordsSubmissionsAndRestoresDraft(t *testing.T) {
	m := testModel()
	m.openCommandPrompt()
	m.commandInput = "  MiSsInG  argument  "
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if got, want := m.commandHistory, []string{"MiSsInG  argument"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("history = %#v, want %#v", got, want)
	}

	m.openCommandPrompt()
	m.commandInput = "draft"
	m = pressKey(m, tea.Key{Code: tea.KeyUp})
	if m.commandInput != "MiSsInG  argument" {
		t.Fatalf("Up recalled %q, want newest history", m.commandInput)
	}
	m = pressKey(m, tea.Key{Code: 'X', Text: "X"})
	m = pressKey(m, tea.Key{Code: tea.KeyUp})
	if m.commandInput != "MiSsInG  argument" {
		t.Fatalf("Up after recalled edit = %q, want clamped oldest entry", m.commandInput)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyDown})
	if m.commandInput != "draft" || m.commandHistoryIndex != -1 {
		t.Fatalf("Down past newest = input:%q index:%d, want original draft and no navigation", m.commandInput, m.commandHistoryIndex)
	}
}

func TestCommandPromptHistoryKeepsOneHundredMostRecentEntries(t *testing.T) {
	m := testModel()
	for i := range commandHistoryLimit + 1 {
		m.recordCommandHistory(fmt.Sprintf("command-%d", i))
	}
	if len(m.commandHistory) != commandHistoryLimit || m.commandHistory[0] != "command-1" || m.commandHistory[99] != "command-100" {
		t.Fatalf("history bounds = %#v", m.commandHistory)
	}
}

func TestCommandPromptCompletionUsesRegistryAndFooterFeedback(t *testing.T) {
	m := testModel()
	m.openCommandPrompt()
	m.commandInput = "k"
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	if m.commandInput != "kanban" || m.commandCompletionFeedback != "" {
		t.Fatalf("unique completion = input:%q feedback:%q", m.commandInput, m.commandCompletionFeedback)
	}

	m.commandInput = "w"
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	const matches = "matches: write, writequit, writequitall"
	if m.commandInput != "w" || m.commandCompletionFeedback != matches {
		t.Fatalf("ambiguous completion = input:%q feedback:%q", m.commandInput, m.commandCompletionFeedback)
	}
	if footer := ansi.Strip(m.renderStatusBar(120)); !strings.Contains(footer, matches) {
		t.Fatalf("footer = %q, missing completion matches", footer)
	}

	m = pressKey(m, tea.Key{Code: 'x', Text: "x"})
	if m.commandCompletionFeedback != "" {
		t.Fatalf("editing left completion feedback %q", m.commandCompletionFeedback)
	}
	m.commandInput = "write argument"
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	if m.commandCompletionFeedback != "Argument completion is not available." {
		t.Fatalf("argument completion feedback = %q", m.commandCompletionFeedback)
	}
}

func TestCommandPromptTabExplainsEmptyInputWithoutRecordingError(t *testing.T) {
	m := testModel()
	m.openCommandPrompt()
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	const want = "Tab completes command names  •  :help lists commands"
	if m.commandCompletionFeedback != want || m.lastErr != "" {
		t.Fatalf("empty Tab = feedback:%q error:%q", m.commandCompletionFeedback, m.lastErr)
	}
}

func TestCommandForceEditDiscardsWholeTaskDraftAndKeepsDetailOpen(t *testing.T) {
	for _, input := range []string{"e!", "edit!"} {
		t.Run(input, func(t *testing.T) {
			m := testModel()
			m.openExistingEditor(vimkunja.Task{
				ID: 7, ProjectID: 1, Title: "Saved title", Description: "saved description",
				Priority: 2, DueDate: "2026-09-20", Done: true,
			})
			m.store.(*fakeStore).taskLabels = map[int64][]vimkunja.Label{7: {
				{ID: 3, Title: "home"}, {ID: 5, Title: "next"},
			}}
			updated, _ := m.Update(m.loadEditorLabels(7)())
			m = updated.(*model)
			m.editor.original.WorkflowBucketID = 11
			m.editor.draft = cloneTaskDraft(m.editor.original)
			m.editor.draft.ProjectID = 2
			m.editor.draft.Title = "Changed title"
			m.editor.draft.Description = "changed description"
			m.editor.draft.Priority = 4
			m.editor.draft.DueDate = "2026-10-01"
			m.editor.draft.Done = false
			m.editor.draft.WorkflowBucketID = 12
			m.editor.draft.LabelIDs = []int64{8}
			m.editor.labels = []vimkunja.Label{{ID: 8, Title: "staged"}}
			m.editor.field = fieldDescription
			m.editor.cursor = len([]rune(m.editor.draft.Description))

			m, cmd := typeCommand(t, m, input)

			if cmd != nil {
				t.Fatalf(":%s returned an unexpected command", input)
			}
			if !m.showDetail || m.focus != paneDetail || m.editor == nil {
				t.Fatalf(":%s closed Task Detail", input)
			}
			if m.editor.mode != editorNormal || m.editor.dirty() {
				t.Fatalf(":%s left mode:%v dirty:%v, want clean Normal mode", input, m.editor.mode, m.editor.dirty())
			}
			if got := m.editor.draft; got.ProjectID != 1 || got.Title != "Saved title" || got.Description != "saved description" ||
				got.Priority != 2 || got.DueDate != "2026-09-20" || !got.Done || got.WorkflowBucketID != 11 ||
				!sameLabelIDs(got.LabelIDs, []int64{3, 5}) {
				t.Fatalf(":%s draft = %#v, want complete saved Task state", input, got)
			}
			if len(m.editor.labels) != 2 || m.editor.labels[0].ID != 3 || m.editor.labels[1].ID != 5 {
				t.Fatalf(":%s labels = %#v, want saved labels", input, m.editor.labels)
			}
		})
	}
}

func TestCommandForceEditRejectsUnsavedNewTaskWithoutChangingDraft(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "Unsaved new Task"

	m, cmd := typeCommand(t, m, "e!")

	if cmd != nil || m.editor == nil || m.editor.draft.Title != "Unsaved new Task" || !m.editor.dirty() {
		t.Fatalf(":e! changed new Task draft: command:%v editor:%#v", cmd != nil, m.editor)
	}
	if !strings.Contains(m.lastErr, "cannot reload an unsaved new Task") || !strings.Contains(m.lastErr, "use :q!") {
		t.Fatalf(":e! error = %q, want new-Task guidance", m.lastErr)
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

func TestCommandQuitAllRefusesUnsavedDetailChanges(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "Unsaved"

	m, cmd := typeCommand(t, m, "qa")
	if cmd != nil {
		t.Fatal(":qa returned a quit command despite unsaved Detail changes")
	}
	if !m.showDetail || m.editor == nil || m.lastErr == "" {
		t.Fatalf(":qa state = detail:%v editor:%v error:%q, want retained dirty Detail", m.showDetail, m.editor != nil, m.lastErr)
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

func TestDescriptionNormalModeOpensLinesAndAppends(t *testing.T) {
	for _, tc := range []struct {
		name        string
		key         tea.Key
		description string
		cursor      int
		wantText    string
		wantCursor  int
	}{
		{"open below", tea.Key{Code: 'o', Text: "o"}, "first\nsecond", 2, "first\n\nsecond", 6},
		{"open above", tea.Key{Code: 'O', Text: "O"}, "first\nsecond", 8, "first\n\nsecond", 6},
		{"append", tea.Key{Code: 'a', Text: "a"}, "first", 2, "first", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.currentProjectID = 1
			m.openNewEditor()
			m.editor.field = fieldDescription
			m.editor.draft.Description = tc.description
			m.editor.cursor = tc.cursor

			m = pressKey(m, tc.key)
			if m.editor.mode != editorInsert {
				t.Fatalf("mode = %v, want Insert", m.editor.mode)
			}
			if got := m.editor.draft.Description; got != tc.wantText {
				t.Fatalf("description = %q, want %q", got, tc.wantText)
			}
			if got := m.editor.cursor; got != tc.wantCursor {
				t.Fatalf("cursor = %d, want %d", got, tc.wantCursor)
			}
		})
	}
}

func TestEditorInsertHomeEndStayInInsertMode(t *testing.T) {
	for _, tc := range []struct {
		name     string
		field    editorField
		value    string
		cursor   int
		wantHome int
		wantEnd  int
	}{
		{name: "title", field: fieldTitle, value: "Möwen bilder", cursor: 5, wantHome: 0, wantEnd: 12},
		{name: "description current line", field: fieldDescription, value: "first\nMöwen bilder\nlast", cursor: 9, wantHome: 6, wantEnd: 18},
		{name: "legacy due date insert mode", field: fieldDueDate, value: "2026-09-04", cursor: 5, wantHome: 0, wantEnd: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.currentProjectID = 1
			m.openNewEditor()
			m.editor.field = tc.field
			m.editor.setFieldValue(tc.value)
			m.editor.cursor = tc.cursor
			m.editor.mode = editorInsert

			m = pressKey(m, tea.Key{Code: tea.KeyHome})
			if m.editor.mode != editorInsert || m.editor.cursor != tc.wantHome {
				t.Fatalf("Home = mode:%v cursor:%d, want Insert/%d", m.editor.mode, m.editor.cursor, tc.wantHome)
			}
			m = pressKey(m, tea.Key{Code: tea.KeyEnd})
			if m.editor.mode != editorInsert || m.editor.cursor != tc.wantEnd {
				t.Fatalf("End = mode:%v cursor:%d, want Insert/%d", m.editor.mode, m.editor.cursor, tc.wantEnd)
			}
		})
	}
}

// Normal date entry uses the picker, not the legacy Insert Mode exercised
// above. Home/End must not navigate picker rows; h/l move within the date mask.
func TestDueDatePickerHomeEndDoNotNavigateRows(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldDueDate
	m.editor.draft.DueDate = "2026-09-04"
	m.editor.cursor = 5

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	if m.editor.mode != editorDueDatePicker || m.editor.cursor != 0 {
		t.Fatalf("i = mode:%v cursor:%d, want Due Date picker/0", m.editor.mode, m.editor.cursor)
	}
	// Row navigation preserves a cursor moved within this picker session,
	// not one inherited from the field edited before opening it.
	for range 5 {
		m = pressKey(m, tea.Key{Code: tea.KeyRight})
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEnd})
	m = pressKey(m, tea.Key{Code: tea.KeyHome})
	if m.editor.mode != editorDueDatePicker || m.editor.dueDatePickerSelected != 0 || m.editor.cursor != 5 {
		t.Fatalf("Home/End = mode:%v selected:%d cursor:%d, want picker/0/5", m.editor.mode, m.editor.dueDatePickerSelected, m.editor.cursor)
	}
	m = pressKey(m, tea.Key{Code: 'h', Text: "h"})
	if m.editor.cursor != 4 {
		t.Fatalf("h cursor = %d, want 4", m.editor.cursor)
	}
	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	if m.editor.cursor != 5 {
		t.Fatalf("l cursor = %d, want 5", m.editor.cursor)
	}
	if m.editor.draft.DueDate != "2026-09-04" {
		t.Fatalf("picker navigation changed date: %q", m.editor.draft.DueDate)
	}
}

func TestCommentComposerNormalModeRendersBlockCursor(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		cursor     int
		wantPlain  string
	}{
		{name: "text", text: "one two", cursor: 4, wantPlain: "one two"},
		{name: "empty draft", wantPlain: " "},
		{name: "empty line", text: "one\n\ntwo", cursor: 4, wantPlain: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.comments = &commentsUI{composer: &commentComposer{text: tc.text, cursor: tc.cursor, mode: editorNormal}}

			rows := m.renderCommentComposer(40, 8)
			if len(rows) < 3 || !strings.Contains(strings.Join(rows[2:], "\n"), "\x1b[") {
				t.Fatalf("Comment Normal-mode composer lacks visible block cursor: %q", rows)
			}
			if tc.name != "empty line" && ansi.Strip(rows[2]) != tc.wantPlain {
				t.Fatalf("Comment row = %q, want %q", ansi.Strip(rows[2]), tc.wantPlain)
			}
		})
	}
}

func TestCommentComposerInsertHomeEndUseCurrentLine(t *testing.T) {
	m := testModel()
	m.comments = &commentsUI{composer: &commentComposer{
		text:   "first\nMöwen bilder\nlast",
		cursor: 9,
		mode:   editorInsert,
	}}

	m = pressKey(m, tea.Key{Code: tea.KeyHome})
	if m.comments.composer.mode != editorInsert || m.comments.composer.cursor != 6 {
		t.Fatalf("Comment Home = mode:%v cursor:%d, want Insert/6", m.comments.composer.mode, m.comments.composer.cursor)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEnd})
	if m.comments.composer.mode != editorInsert || m.comments.composer.cursor != 18 {
		t.Fatalf("Comment End = mode:%v cursor:%d, want Insert/18", m.comments.composer.mode, m.comments.composer.cursor)
	}
}

func TestEditorInsertPastesSystemClipboardAtCursor(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "before after"
	m.editor.cursor = len([]rune("before "))
	m.editor.mode = editorInsert
	m.clipboardRead = func() (string, error) { return "clipboard", nil }

	updated, pasteCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'v', Mod: tea.ModCtrl}))
	m = updated.(*model)
	if pasteCmd == nil {
		t.Fatal("Ctrl+V returned no clipboard command")
	}
	updated, _ = m.Update(pasteCmd())
	m = updated.(*model)

	if got := m.editor.draft.Title; got != "before clipboardafter" {
		t.Fatalf("title after paste = %q", got)
	}
	if got := m.editor.cursor; got != len([]rune("before clipboard")) {
		t.Fatalf("cursor after paste = %d", got)
	}
}

func TestDescriptionInsertAcceptsTerminalPaste(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldDescription
	m.editor.draft.Description = "before after"
	m.editor.cursor = len([]rune("before "))
	m.editor.mode = editorInsert

	updated, _ := m.Update(tea.PasteMsg{Content: "https://example.test"})
	m = updated.(*model)

	if got := m.editor.draft.Description; got != "before https://example.testafter" {
		t.Fatalf("description after terminal paste = %q", got)
	}
}

func TestDescriptionInsertNormalizesBrowserPasteLineEndings(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldDescription
	m.editor.mode = editorInsert

	const pasted = "one two three\r\nalpha, beta\r\n  δelta epsilon\r\nlast word"
	updated, _ := m.Update(tea.PasteMsg{Content: pasted})
	m = updated.(*model)

	const want = "one two three\nalpha, beta\n  δelta epsilon\nlast word"
	if got := m.editor.draft.Description; got != want {
		t.Fatalf("Description after browser paste = %q, want %q", got, want)
	}
}

func TestEditorInsertPastesClipboardIntoDueDateMask(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldDueDate
	m.clipboardRead = func() (string, error) { return "2026-08-08", nil }

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	updated, pasteCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'v', Mod: tea.ModCtrl}))
	m = updated.(*model)
	updated, _ = m.Update(pasteCmd())
	m = updated.(*model)

	if got := m.editor.draft.DueDate; got != "2026-08-08" {
		t.Fatalf("due date after paste = %q", got)
	}
}

func TestEditorInsertClipboardFailuresKeepTheDraft(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "unchanged"
	m.editor.mode = editorInsert
	m.clipboardRead = func() (string, error) { return "", errors.New("clipboard unavailable") }

	updated, pasteCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'v', Mod: tea.ModCtrl}))
	m = updated.(*model)
	updated, _ = m.Update(pasteCmd())
	m = updated.(*model)

	if got := m.editor.draft.Title; got != "unchanged" {
		t.Fatalf("title after failed paste = %q", got)
	}
	if !strings.Contains(m.lastErr, "clipboard unavailable") {
		t.Fatalf("paste error = %q", m.lastErr)
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

func TestProjectPickerKeepsSelectedProjectVisibleInNormalDetailLayout(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	for i := 0; i < 12; i++ {
		m.projects = append(m.projects, vimkunja.Project{ID: int64(i + 1), Title: fmt.Sprintf("Project %02d", i)})
	}
	m.editor = &taskEditor{field: fieldProject, mode: editorProjectPicker}
	for range 10 {
		m.updateProjectPicker(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	}

	view := ansi.Strip(renderTaskDetail(m, 80, 14))
	if !strings.Contains(view, "Project 10") {
		t.Fatalf("selected project scrolled out of the normal Detail layout: %q", view)
	}
}

func TestDetailPickersPageAndJumpUsingTheirVisibleChoiceRows(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(*model)
		selected func(*model) int
		last     int
		visible  string
	}{
		{
			name: "project",
			setup: func(m *model) {
				for i := 0; i < 10; i++ {
					m.projects = append(m.projects, vimkunja.Project{ID: int64(i + 1), Title: fmt.Sprintf("Project %d", i+1)})
				}
				m.editor = &taskEditor{field: fieldProject, mode: editorProjectPicker}
			},
			selected: func(m *model) int { return m.editor.pickerIndex },
			last:     9,
			visible:  "Project 9",
		},
		{
			name: "priority",
			setup: func(m *model) {
				m.editor = &taskEditor{field: fieldPriority, mode: editorPriorityPicker}
			},
			selected: func(m *model) int { return m.editor.priorityIndex },
			last:     len(tuiTheme.priority) - 1,
			visible:  "Do now",
		},
		{
			name: "workflow status",
			setup: func(m *model) {
				buckets := make([]vimkunja.WorkflowBucket, 10)
				for i := range buckets {
					buckets[i] = vimkunja.WorkflowBucket{ID: int64(i + 1), Title: fmt.Sprintf("Status %d", i+1)}
				}
				m.editor = &taskEditor{field: fieldWorkflowStatus, mode: editorWorkflowStatusPicker, workflowStatus: vimkunja.WorkflowStatus{Buckets: buckets}}
			},
			selected: func(m *model) int { return m.editor.workflowStatusIndex },
			last:     9,
			visible:  "Status 9",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.width, m.height = 80, 19 // Four visible picker choice rows.
			m.focus, m.showDetail = paneDetail, true
			tc.setup(m)

			m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
			if got := tc.selected(m); got != 4 {
				t.Fatalf("Ctrl-D selected %d, want one four-row picker page", got)
			}
			m = pressKey(m, tea.Key{Code: 'u', Mod: tea.ModCtrl})
			if got := tc.selected(m); got != 0 {
				t.Fatalf("Ctrl-U selected %d, want first choice", got)
			}
			m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
			m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
			if got := tc.selected(m); got != min(8, tc.last) || !strings.Contains(ansi.Strip(m.Render()), tc.visible) {
				t.Fatalf("paged picker = selected:%d view:%q, want visible %q", got, ansi.Strip(m.Render()), tc.visible)
			}
			m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
			if got := tc.selected(m); got != tc.last {
				t.Fatalf("G selected %d, want last choice %d", got, tc.last)
			}
			m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
			m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
			if got := tc.selected(m); got != 0 {
				t.Fatalf("gg selected %d, want first choice", got)
			}
		})
	}
}

func TestDetailPickerOverlaysShortDetailWithoutClippingItsFrame(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 19
	m.focus = paneDetail
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee", Description: "Description remains behind the picker."})
	m.editor.field = fieldPriority
	m.editor.mode = editorPriorityPicker
	m.editor.priorityIndex = 3

	view := ansi.Strip(renderTaskDetail(m, 80, 8))
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╰") || !strings.Contains(view, "▸ High") {
		t.Fatalf("short Detail picker frame or selected choice was clipped: %q", view)
	}
}

func TestDetailPickerWidthComesFromAllChoicesNotTheCurrentViewport(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.projects = []vimkunja.Project{{ID: 1, Title: "A project title wide enough to set the picker width"}}
	for id := int64(2); id <= 12; id++ {
		m.projects = append(m.projects, vimkunja.Project{ID: id, Title: "short"})
	}
	m.editor = &taskEditor{field: fieldProject, mode: editorProjectPicker}

	first := pickerFrameWidth(renderTaskDetail(m, 80, 14))
	m.editor.pickerIndex = 11
	last := pickerFrameWidth(renderTaskDetail(m, 80, 14))
	if first == 0 || last != first {
		t.Fatalf("picker width changed with its selected viewport: first=%d last=%d", first, last)
	}
}

func pickerFrameWidth(view string) int {
	for _, line := range strings.Split(ansi.Strip(view), "\n") {
		left, right := strings.Index(line, "╭"), strings.Index(line, "╮")
		if left >= 0 && right > left {
			return lipgloss.Width(line[left : right+len("╮")])
		}
	}
	return 0
}

func TestProjectPickerIgnoresPrintableKeys(t *testing.T) {
	m := testModel()
	m.projects = []vimkunja.Project{{ID: 1, Title: "Alpha"}, {ID: 2, Title: "Zulu"}}
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldProject

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'i', Text: "i"}))
	m = updated.(*model)
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'z', Text: "z"}))
	m = updated.(*model)
	view := ansi.Strip(renderTaskDetail(m, 80, 20))
	if m.editor.mode != editorProjectPicker || m.editor.projectFilter != "" || !strings.Contains(view, "Alpha") || !strings.Contains(view, "Zulu") {
		t.Fatalf("project picker accepted printable input: filter=%q view=%q", m.editor.projectFilter, view)
	}
}

func TestOverflowingPriorityPickerHasInternalScrollbar(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.editor = &taskEditor{field: fieldPriority, mode: editorPriorityPicker, priorityIndex: 3}

	view := ansi.Strip(renderTaskDetail(m, 80, 5))
	popover := pickerFrame(view)
	if !strings.Contains(popover, "│") || !strings.Contains(popover, "█") {
		t.Fatalf("overflowing Priority picker lacks an internal rail and thumb: %q", popover)
	}
}

func pickerFrame(view string) string {
	lines := strings.Split(view, "\n")
	inside := false
	var frame []string
	for _, line := range lines {
		if strings.Contains(line, "╭") {
			inside = true
		}
		if inside {
			frame = append(frame, line)
		}
		if inside && strings.Contains(line, "╰") {
			break
		}
	}
	return strings.Join(frame, "\n")
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
			model:  &model{focus: paneHelp, helpPane: &infoPane{lines: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}, offset: 1}},
			offset: func(m *model) int { return m.helpPane.offset },
		},
		{
			name:   "errors",
			model:  &model{focus: paneInfo, infoPane: &infoPane{lines: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}, offset: 1}},
			offset: func(m *model) int { return m.infoPane.offset },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pressKey(tc.model, tea.Key{Code: tea.KeyEnd})
			if got := tc.offset(m); got != 3 {
				t.Fatalf("offset after End = %d, want 3", got)
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
	if footer := m.renderStatusBar(120); !strings.Contains(footer, "Ctrl-W") || !strings.Contains(footer, "38;5;"+tuiTheme.ctrlWHint) {
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

func TestCompactTaskDetailCollapsesMetadataBehindMoreFields(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 22
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee", Description: "Fresh beans"})

	compact := ansi.Strip(m.Render())
	for _, want := range []string{"Title:", "More fields", "Description:", "Fresh beans"} {
		if !strings.Contains(compact, want) {
			t.Fatalf("compact Task Detail = %q, want %q", compact, want)
		}
	}
	if strings.Contains(compact, "Priority:") {
		t.Fatalf("compact Task Detail exposes collapsed metadata: %q", compact)
	}

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	expanded := ansi.Strip(m.Render())
	if !strings.Contains(expanded, "Priority:") {
		t.Fatalf("expanded Task Detail = %q, want metadata", expanded)
	}
}

func TestExpandedCompactDetailScrollsToLabels(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 22
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.labelsLoading = false
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	for range 4 { // Project, Priority, Due Date, Labels
		m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	}
	if m.editor.field != fieldLabels || m.editor.detailScroll == 0 {
		t.Fatalf("expanded compact navigation = field:%v scroll:%d, want Labels in a scrolled viewport", m.editor.field, m.editor.detailScroll)
	}
	if view := ansi.Strip(m.Render()); !strings.Contains(view, "Labels:") {
		t.Fatalf("expanded compact Detail did not reveal Labels: %q", view)
	}
}

func TestTaskDetailActivatesDescriptionSeparatelyFromFieldNavigation(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 22
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee", Description: "first\nsecond"})

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.editor.field != fieldDescription {
		t.Fatalf("field after compact navigation = %v, want Description", m.editor.field)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.editor.cursor == 0 {
		t.Fatal("active Description did not receive its Normal-mode navigation")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.editor == nil || m.editor.field != fieldDescription {
		t.Fatalf("Esc closed Description instead of returning to Detail navigation: %#v", m.editor)
	}
}

func TestCompactTaskDetailPagesItsOuterViewport(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 22
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee", Description: "Fresh beans"})

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	before := ansi.Strip(m.Render())
	m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
	after := ansi.Strip(m.Render())
	if after == before {
		t.Fatal("Ctrl-D did not page the Task Detail outer viewport")
	}
	if !strings.Contains(after, "┬") || !strings.Contains(after, "┴") {
		t.Fatalf("outer viewport lacks scrollbar bounds: %q", after)
	}
}

func TestCompactTaskDetailThresholdReservesDescriptionRows(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee", Description: "one\ntwo\nthree"})

	compact := ansi.Strip(renderTaskDetail(m, 80, 8))
	if !strings.Contains(compact, "More fields") || !strings.Contains(compact, "one") || !strings.Contains(compact, "Comments (…)") {
		t.Fatalf("eight-row Task Detail = %q, want compact fields, Description access, and Comments", compact)
	}
	short := ansi.Strip(renderTaskDetail(m, 80, 7))
	if !strings.Contains(short, "one") || !strings.Contains(short, "Comments (…)") {
		t.Fatalf("short Task Detail = %q, want Description access and Comments", short)
	}
	regular := ansi.Strip(renderTaskDetail(m, 80, 12))
	if strings.Contains(regular, "More fields") || !strings.Contains(regular, "Priority:") || !strings.Contains(regular, "Comments (…)") {
		t.Fatalf("twelve-row Task Detail = %q, want regular metadata and pinned Comments", regular)
	}
}

func TestTinyActiveDescriptionUsesWholeDetailPaneAndFollowsCursor(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee", Description: "one\ntwo\nthree\nfour\nfive\nsix"})
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	m.editor.mode = editorInsert
	m.editor.cursor = len([]rune(m.editor.draft.Description))
	m.cursorVisible = true

	view := ansi.Strip(renderTaskDetail(m, 80, 8))
	if !strings.Contains(view, "DESCRIPTION · INSERT") || !strings.Contains(view, "six") {
		t.Fatalf("tiny active Description = %q, want focused editor following cursor", view)
	}
	if strings.Contains(view, "Comments") || strings.Contains(view, "More fields") {
		t.Fatalf("tiny active Description retained Detail chrome: %q", view)
	}

	m.editor.insertText("\nseven")
	view = ansi.Strip(renderTaskDetail(m, 80, 8))
	if !strings.Contains(view, "seven") {
		t.Fatalf("tiny active Description hid inserted line: %q", view)
	}
}

func TestDescriptionNormalModeDeletesCharactersAndLines(t *testing.T) {
	newEditor := func(text string, cursor int) *model {
		m := testModel()
		m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: text})
		m.editor.field, m.editor.descriptionActive, m.editor.cursor = fieldDescription, true, cursor
		return m
	}

	m := newEditor("abc", 1)
	m = pressKey(m, tea.Key{Code: 'x', Text: "x"})
	if m.editor.draft.Description != "ac" {
		t.Fatalf("x Description = %q, want ac", m.editor.draft.Description)
	}
	m = newEditor("abc", 1)
	m = pressKey(m, tea.Key{Code: 'X', Text: "X"})
	if m.editor.draft.Description != "bc" || m.editor.cursor != 0 {
		t.Fatalf("X Description = %q at %d, want bc at 0", m.editor.draft.Description, m.editor.cursor)
	}
	m = newEditor("abc\ndef", 1)
	m = pressKey(m, tea.Key{Code: 'D', Text: "D"})
	if m.editor.draft.Description != "a\ndef" {
		t.Fatalf("D Description = %q, want a\\ndef", m.editor.draft.Description)
	}
	m = newEditor("one\ntwo\nthree", 5)
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	if m.editor.draft.Description != "one\nthree" || m.editor.cursor != 4 {
		t.Fatalf("dd Description = %q at %d, want one\\nthree at 4", m.editor.draft.Description, m.editor.cursor)
	}
}

func TestTinyDescriptionInsertShowsCursorOnEmptyLine(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 10
	m.focus = paneDetail
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "one\ntwo\nthree\n"})
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	m.editor.mode = editorInsert
	m.editor.cursor = len([]rune(m.editor.draft.Description))
	m.cursorVisible = true

	view := ansi.Strip(renderTaskDetail(m, 80, 8))
	if !strings.Contains(view, "│") {
		t.Fatalf("tiny empty Description line lacks insert cursor: %q", view)
	}
}

func TestActivatingDescriptionCollapsesExpandedCompactFields(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 22
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee", Description: "first\nsecond\nthird"})

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	for m.editor.field != fieldDescription {
		m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	}
	if m.editor.detailScroll == 0 {
		t.Fatal("expanded metadata did not require outer scrolling before Description")
	}

	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	view := ansi.Strip(m.Render())
	if m.editor.moreFieldsExpanded || !strings.Contains(view, "first") || !strings.Contains(view, "second") {
		t.Fatalf("active Description = expanded:%v view:%q, want collapsed metadata and visible Description", m.editor.moreFieldsExpanded, view)
	}
}

func TestActiveCompactDescriptionPagesItsInnerViewport(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 22
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee", Description: strings.Repeat("line\n", 12)})

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	m = pressKey(m, tea.Key{Code: 'd', Mod: tea.ModCtrl})
	if m.editor.descScroll == 0 {
		t.Fatal("Ctrl-D did not page the active Description viewport")
	}
}

func TestCompactTaskDetailRetainsPickerEditing(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 22
	for id := int64(1); id <= 8; id++ {
		m.projects = append(m.projects, vimkunja.Project{ID: id, Title: fmt.Sprintf("Project %d", id)})
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})

	if m.editor.mode != editorProjectPicker || !strings.Contains(ansi.Strip(m.Render()), "Project 1") {
		t.Fatalf("compact Project picker = mode:%v view:%q", m.editor.mode, ansi.Strip(m.Render()))
	}
	for range 6 {
		m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	}
	if view := ansi.Strip(m.Render()); !strings.Contains(view, "Project 7") {
		t.Fatalf("compact Project picker did not keep its selected choice visible: %q", view)
	}
}

func TestResizePreservesCompactDetailFieldSelection(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 22
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	if m.editor.field != fieldPriority {
		t.Fatalf("expanded compact field = %v, want Priority", m.editor.field)
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(*model)
	if m.editor.field != fieldPriority {
		t.Fatalf("resize changed selected field to %v, want Priority", m.editor.field)
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

func TestPickerFooterDescribesPickerEscape(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.editor = &taskEditor{field: fieldProject, mode: editorProjectPicker}

	footer := ansi.Strip(m.renderStatusBar(120))
	if strings.Contains(footer, "[INSERT]") || strings.Contains(footer, "Esc normal") || !strings.Contains(footer, "Esc cancel") {
		t.Fatalf("project picker footer = %q, want picker-specific cancellation hint", footer)
	}
}

func TestLabelsOverlayOmitsTheNativeTextCursor(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.openNewEditor()
	m.labelUI = &labelUI{kind: labelUIGlobalOverview}

	if cursor := m.View().Cursor; cursor != nil {
		t.Fatalf("Labels overlay cursor = %#v, want none", cursor)
	}
}

func TestTextOnlyFallbackFitsAnExplicitTinyTerminal(t *testing.T) {
	for _, height := range []int{8, 0} {
		t.Run(fmt.Sprintf("%d rows", height), func(t *testing.T) {
			m := testModel()
			m.focus = paneDetail
			m.openNewEditor()
			updated, _ := m.Update(tea.WindowSizeMsg{Width: 54, Height: height})
			m = updated.(*model)

			view := ansi.Strip(m.Render())
			lines := strings.Split(view, "\n")
			if len(lines) != max(1, height) || strings.Contains(view, "┌") || strings.Contains(view, "╭") {
				t.Fatalf("tiny terminal did not use text-only fallback: %q", view)
			}
			for _, line := range lines {
				if lipgloss.Width(line) > 54 {
					t.Fatalf("fallback width = %d, want <= 54: %q", lipgloss.Width(line), line)
				}
			}
		})
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
	for _, field := range []editorField{fieldProject, fieldPriority, fieldWorkflowStatus} {
		m.editor.field = field
		switch field {
		case fieldProject:
			m.editor.mode = editorProjectPicker
		case fieldPriority:
			m.editor.mode = editorPriorityPicker
		case fieldWorkflowStatus:
			m.editor.mode = editorWorkflowStatusPicker
		}
		if view = m.View(); view.Cursor != nil {
			t.Fatalf("picker %v cursor = %#v, want none", field, view.Cursor)
		}
	}
}

func TestGlobalSearchOmitsNativeEditorCursor(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.currentProjectID = 1
	m.openNewEditor()
	m.searchOpen = true

	if view := m.View(); view.Cursor != nil {
		t.Fatalf("Global Search cursor = %#v, want none", view.Cursor)
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

func TestPickerKeepsItsFieldAnchorVisibleBehindTheOverlay(t *testing.T) {
	tests := []struct {
		name  string
		field editorField
		mode  editorMode
		label string
	}{
		{"project", fieldProject, editorProjectPicker, "Project:  "},
		{"priority", fieldPriority, editorPriorityPicker, "Priority: "},
		{"workflow status", fieldWorkflowStatus, editorWorkflowStatusPicker, "Status:   "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.currentProjectID = 1
			m.openNewEditor()
			m.editor.field, m.editor.mode = tc.field, tc.mode

			detail := ansi.Strip(renderTaskDetail(m, 80, 20))
			if !strings.Contains(detail, "▸ "+tc.label) {
				t.Fatalf("picker anchor is not visible behind its overlay: %q", detail)
			}
		})
	}
}

func TestPriorityPickerFramesChoicesAndColoursActivePriority(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldPriority
	m.editor.mode = editorPriorityPicker
	m.editor.draft.Priority = 2
	m.editor.priorityIndex = 2

	detail := renderTaskDetail(m, 80, 20)
	plain := ansi.Strip(detail)
	if !strings.Contains(plain, "╭") || !strings.Contains(plain, "╰") || !strings.Contains(plain, "▸ Medium") {
		t.Fatalf("priority choices must be framed as a complete overlay: %q", plain)
	}
	selected := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(priorityColor(2))).Render("▸ Medium")
	if !strings.Contains(detail, selected) {
		t.Fatalf("active priority lacks semantic colour: %q", detail)
	}
	if picker := strings.Join(renderPriorityPicker(m.editor), "\n"); !strings.Contains(picker, "38;5;"+tuiTheme.selectionAccent+"m╭") {
		t.Fatalf("priority picker border lacks the shared selection accent: %q", picker)
	}
}

func TestProjectAndStatusPickerFramesUseSharedSelectionAccent(t *testing.T) {
	for _, picker := range []struct {
		name    string
		choices []string
	}{
		{name: "project", choices: []string{"Inbox", "Work"}},
		{name: "status", choices: []string{"To do", "Doing"}},
	} {
		t.Run(picker.name, func(t *testing.T) {
			rendered := strings.Join(renderPickerChoices(picker.choices, 1, tuiTheme.selectionAccent), "\n")
			plain := ansi.Strip(rendered)
			if !strings.Contains(plain, "╭") || !strings.Contains(plain, "╰") {
				t.Fatalf("picker lacks frame: %q", plain)
			}
			selected := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Render("▸ " + picker.choices[1])
			if !strings.Contains(rendered, selected) {
				t.Fatalf("picker lacks normal selection highlight: %q", rendered)
			}
			if !strings.Contains(rendered, "38;5;"+tuiTheme.selectionAccent+"m╭") {
				t.Fatalf("picker frame lacks the shared selection accent: %q", rendered)
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

func TestTitleNormalModeViewportFollowsCursorInLongText(t *testing.T) {
	m := testModel()
	m.width, m.height = 54, 30
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = strings.Repeat("a", 60) + "TAIL"
	m.editor.cursor = len([]rune(m.editor.draft.Title))
	m.editor.mode = editorNormal

	view := m.View()
	if !strings.Contains(ansi.Strip(view.Content), "TAIL") {
		t.Fatalf("long Title Normal-mode viewport does not show cursor context: %q", ansi.Strip(view.Content))
	}
	if view.Cursor == nil || view.Cursor.Shape != tea.CursorBlock || view.Cursor.Position.X >= m.width-1 {
		t.Fatalf("long Title cursor = %#v, want visible block cursor inside pane", view.Cursor)
	}
}

func TestTitleNormalModeRendersStaticCursorWhenCursorShapesDisabled(t *testing.T) {
	m := NewModel(Config{Store: &fakeStore{}, DisableCursorShapes: true}).(*model)
	m.width, m.height = 100, 40
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "abc"
	m.editor.cursor = 1
	m.editor.mode = editorNormal

	view := m.View()
	if view.Cursor != nil || !strings.Contains(view.Content, normalCursorText("b", 0)) {
		t.Fatalf("Title fallback = cursor:%#v view:%q, want static block cursor", view.Cursor, view.Content)
	}
}

func TestNativeTitleNormalModeShowsBlockCursor(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 40
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "abc"
	m.editor.cursor = 1
	m.editor.mode = editorNormal

	cursor := m.View().Cursor
	if cursor == nil || cursor.Shape != tea.CursorBlock || cursor.Position != (tea.Position{X: 15, Y: 22}) {
		t.Fatalf("Title Normal-mode cursor = %#v, want block cursor at text position (15,22)", cursor)
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

func TestDescriptionCursorStaysWithDescriptionLinesAfterEmptyLines(t *testing.T) {
	description := strings.Join([]string{
		"Try to build",
		"- a Tool in Pi in general [x]",
		"- a small ticket tool [x]",
		"  - create scratch tickets [x]",
		"  - archive scratch tickets [x]",
		"  - update [x]",
		"",
		"",
		"  - query them [x]",
		"",
		"- compare token usage [x]",
		"- bring to Github [x]",
		"",
		"",
		"LLM should not",
		"last line",
	}, "\n")
	m := testModel()
	m.width, m.height = 160, 50
	m.currentProjectID = 1
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: description})
	m.editor.field = fieldDescription
	m.editor.mode = editorInsert
	m.editor.cursor = len([]rune("Try to build\n- a Tool in Pi in general [x]\n- a small ticket tool [x]\n  - create scratch tickets [x]\n  - archive scratch tickets [x]\n  - update [x]\n\n\n  - query them [x]\n\n"))
	m.editor.descScroll = 9
	m = pressKey(m, tea.Key{Code: 'X', Text: "X"})

	cursor := m.View().Cursor
	rows := strings.Split(ansi.Strip(m.Render()), "\n")
	if cursor == nil || cursor.Position.Y >= len(rows) || !strings.Contains(rows[cursor.Position.Y], "X- compare token usage") {
		t.Fatalf("description cursor after insertion = %#v, row=%q, want cursor on inserted compare line", cursor, rows[max(0, min(len(rows)-1, cursor.Position.Y))])
	}
}

func TestDescriptionCursorStaysWithWrappedInsertionLine(t *testing.T) {
	m := testModel()
	m.width, m.height = 60, 40
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.field = fieldDescription
	m.editor.mode = editorInsert
	m.editor.draft.Description = strings.Repeat("a", 54) + "target"
	m.editor.cursor = 54
	m = pressKey(m, tea.Key{Code: 'X', Text: "X"})

	cursor := m.View().Cursor
	rows := strings.Split(ansi.Strip(m.Render()), "\n")
	if cursor == nil || cursor.Position.Y >= len(rows) || !strings.Contains(rows[cursor.Position.Y], "Xt") {
		t.Fatalf("wrapped Description cursor after insertion = %#v, row=%q, want cursor on inserted wrapped segment", cursor, rows[max(0, min(len(rows)-1, cursor.Position.Y))])
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
	if cursor == nil || cursor.Shape != tea.CursorBlock || cursor.Position.Y >= len(lines) || !strings.Contains(lines[cursor.Position.Y], "hello") {
		t.Fatalf("description cursor = %#v, rendered lines = %q, want block cursor on Description", cursor, lines)
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

	if detail := ansi.Strip(renderTaskDetail(m, 80, 20)); !strings.Contains(detail, "he│llo") {
		t.Fatalf("disabled cursor shapes lost vertical rendered fallback: %q", detail)
	}
}

func TestClosingEditorRemovesNativeCursor(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	if m.View().Cursor == nil {
		t.Fatal("insert editor has no native cursor")
	}

	m.editor = nil
	m.showDetail = false
	m.focus = paneTasks
	if m.View().Cursor != nil {
		t.Fatalf("closed editor cursor = %#v, want no cursor", m.View().Cursor)
	}
}

func TestInlineTitleEditSavesAndRetainsSelection(t *testing.T) {
	m := testModel()
	store := m.store.(*fakeStore)
	m.focus = paneTasks
	m.width, m.height = 100, 30
	m.tasks = []vimkunja.Task{{ID: 7, ProjectID: 1, Title: "Buy milk", Priority: 2, Description: "Notes"}}

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	if rendered := ansi.Strip(m.Render()); strings.Contains(rendered, "▏") {
		t.Fatalf("inline title injects a cursor glyph: %q", rendered)
	}
	if cursor := m.View().Cursor; cursor == nil || cursor.Shape != tea.CursorBar {
		t.Fatalf("inline title cursor = %#v, want native bar cursor", cursor)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyBackspace})
	m = pressKey(m, tea.Key{Code: '!', Text: "!"})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if cmd == nil || m.inlineTitle == nil {
		t.Fatal("Enter did not begin inline title save")
	}
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if m.inlineTitle != nil || m.tasks[0].Title != "Buy mil!" || m.taskSelected != 0 {
		t.Fatalf("inline save = edit:%#v task:%#v selected:%d", m.inlineTitle, m.tasks[0], m.taskSelected)
	}
	if store.updatedTaskID != 7 || store.updatedTaskTitle != "Buy mil!" {
		t.Fatalf("inline title update = id:%d title:%q", store.updatedTaskID, store.updatedTaskTitle)
	}
}

func TestInlineTitleEditEscapeDiscardsAndEmptyOrFailedSaveRetainsDraft(t *testing.T) {
	m := testModel()
	store := m.store.(*fakeStore)
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 7, ProjectID: 1, Title: "Buy milk"}}

	m = pressKey(m, tea.Key{Code: 'I', Text: "I"})
	m = pressKey(m, tea.Key{Code: 'X', Text: "X"})
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.inlineTitle != nil || m.tasks[0].Title != "Buy milk" {
		t.Fatalf("Escape did not discard inline title: %#v", m.inlineTitle)
	}

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	for range []rune("Buy milk") {
		m = pressKey(m, tea.Key{Code: tea.KeyBackspace})
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if m.inlineTitle == nil || m.lastErr != "project and title are required" {
		t.Fatalf("empty title save = edit:%#v error:%q", m.inlineTitle, m.lastErr)
	}
	m = pressKey(m, tea.Key{Code: 'X', Text: "X"})
	store.updateErr = errors.New("database unavailable")
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if m.inlineTitle == nil || m.inlineTitle.title != "X" || m.lastErr != "database unavailable" {
		t.Fatalf("failed inline save = edit:%#v error:%q", m.inlineTitle, m.lastErr)
	}
}

func TestInlineTitleEditHomeEndAndOverviewReuse(t *testing.T) {
	m := testModel()
	m.focus = paneTasks
	m.taskScope = scopeOverview
	m.overviewTasks = []vimkunja.Task{{ID: 7, ProjectID: 1, Title: "Möwen bilder"}}
	m.tasks = append([]vimkunja.Task(nil), m.overviewTasks...)

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: tea.KeyHome})
	if m.inlineTitle.cursor != 0 {
		t.Fatalf("Home cursor = %d, want 0", m.inlineTitle.cursor)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEnd})
	if got, want := m.inlineTitle.cursor, len([]rune("Möwen bilder")); got != want {
		t.Fatalf("End cursor = %d, want %d", got, want)
	}
	m = pressKey(m, tea.Key{Code: '!', Text: "!"})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if m.inlineTitle != nil || m.tasks[0].Title != "Möwen bilder!" || m.overviewTasks[0].Title != "Möwen bilder!" || m.taskSelected != 0 {
		t.Fatalf("Overview inline save = edit:%#v tasks:%#v source:%#v selected:%d", m.inlineTitle, m.tasks, m.overviewTasks, m.taskSelected)
	}
}

func TestInlineTitleEditFocusChangeWaitsForSuccessfulSave(t *testing.T) {
	m := testModel()
	store := m.store.(*fakeStore)
	m.focus = paneTasks
	m.tasks = []vimkunja.Task{{ID: 7, ProjectID: 1, Title: "Buy milk"}}
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: '!', Text: "!"})

	m = ctrlW(m)
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'h', Text: "h"}))
	m = updated.(*model)
	if cmd == nil || m.focus != paneTasks {
		t.Fatalf("Ctrl+W h before save = command:%v focus:%v, want pending command in Tasks", cmd != nil, m.focus)
	}
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if m.focus != paneProjects || m.inlineTitle != nil {
		t.Fatalf("successful focus save = focus:%v inline:%#v", m.focus, m.inlineTitle)
	}

	m.focus = paneTasks
	m.tasks[0].Title = "Buy milk"
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	store.updateErr = errors.New("database unavailable")
	m = ctrlW(m)
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'h', Text: "h"}))
	m = updated.(*model)
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if m.focus != paneTasks || m.inlineTitle == nil || m.lastErr != "database unavailable" {
		t.Fatalf("failed focus save = focus:%v inline:%#v error:%q", m.focus, m.inlineTitle, m.lastErr)
	}

	store.updateErr = nil
	m = pressKey(m, tea.Key{Code: tea.KeyBackspace})
	for range []rune("Buy milk") {
		m = pressKey(m, tea.Key{Code: tea.KeyBackspace})
	}
	m = ctrlW(m)
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'h', Text: "h"}))
	m = updated.(*model)
	if cmd != nil || m.inlineTitle.pendingFocus != nil {
		t.Fatalf("invalid title focus save = command:%v pending:%#v", cmd != nil, m.inlineTitle.pendingFocus)
	}
	m = pressKey(m, tea.Key{Code: 'X', Text: "X"})
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if m.focus != paneTasks {
		t.Fatalf("ordinary save after invalid focus attempt changed focus to %v", m.focus)
	}

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = ctrlW(m)
	updated, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	m = updated.(*model)
	if cmd != nil || m.focus != paneTasks || m.inlineTitle == nil {
		t.Fatalf("no-op Ctrl+W l = command:%v focus:%v inline:%#v", cmd != nil, m.focus, m.inlineTitle)
	}
}

func TestDueDatePickerSetsRelativeDatesAndKeepsMaskedEntry(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.now = func() time.Time { return time.Date(2026, time.January, 15, 15, 0, 0, 0, time.UTC) }
	m.openNewEditor()
	m.editor.field = fieldDueDate

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	if m.editor.mode != editorDueDatePicker || m.editor.draft.DueDate != "YYYY-MM-DD" {
		t.Fatalf("date picker = mode:%v due:%q, want picker and mask", m.editor.mode, m.editor.draft.DueDate)
	}
	if rendered := ansi.Strip(renderTaskDetail(m, 80, 24)); !strings.Contains(rendered, "In days:") || !strings.Contains(rendered, "In months:") || strings.Contains(rendered, "Tomorrow") {
		t.Fatalf("date picker choices incorrect: %q", rendered)
	}

	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: '3', Text: "3"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if got := m.editor.draft.DueDate; got != "2026-01-18" || m.editor.mode != editorNormal {
		t.Fatalf("three days = due:%q mode:%v", got, m.editor.mode)
	}

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: '2', Text: "2"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if got := m.editor.draft.DueDate; got != "2026-01-29" {
		t.Fatalf("two weeks = %q, want 2026-01-29", got)
	}

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	for range 3 {
		m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	}
	m = pressKey(m, tea.Key{Code: '1', Text: "1"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if got := m.editor.draft.DueDate; got != "2026-02-15" {
		t.Fatalf("one month = %q, want 2026-02-15", got)
	}

}

func TestDetailFieldShortcutsOpenTheirEditors(t *testing.T) {
	cases := []struct {
		name      string
		key       rune
		wantField editorField
		wantMode  editorMode
	}{
		{name: "title", key: 't', wantField: fieldTitle, wantMode: editorInsert},
		{name: "due date", key: 'd', wantField: fieldDueDate, wantMode: editorDueDatePicker},
		{name: "priority", key: 'p', wantField: fieldPriority, wantMode: editorPriorityPicker},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.currentProjectID = 1
			m.openNewEditor()
			m.editor.field = fieldDescription

			m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
			m = pressKey(m, tea.Key{Code: tc.key, Text: string(tc.key)})

			if m.editor.field != tc.wantField || m.editor.mode != tc.wantMode {
				t.Fatalf("g%c = field:%v mode:%v, want field:%v mode:%v", tc.key, m.editor.field, m.editor.mode, tc.wantField, tc.wantMode)
			}
		})
	}

	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.labelsLoading = false
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'l', Text: "l"})
	if m.editor.field != fieldLabels || m.labelUI == nil || m.labelUI.kind != labelUITaskPicker {
		t.Fatalf("gl = field:%v label UI:%#v, want Labels picker", m.editor.field, m.labelUI)
	}
}

func TestTaskCommentsOpenNavigateAndAppendFromDetail(t *testing.T) {
	store := &fakeStore{taskComments: map[int64][]vimkunja.TaskComment{1: {
		{ID: 1, TaskID: 1, HTML: "<p>First <strong>comment</strong></p>", Author: "Ada", Created: time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)},
		{ID: 2, TaskID: 1, HTML: "<p>Second</p>", Author: "Grace", Created: time.Date(2026, 9, 4, 11, 0, 0, 0, time.UTC)},
	}}}
	m := NewModel(Config{Store: store}).(*model)
	m.width, m.height = 100, 35
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "Description"})
	updated, cmd := m.Update(taskCommentsLoadedMsg{taskID: 1, comments: store.taskComments[1]})
	m = updated.(*model)
	if cmd != nil {
		t.Fatal("loading Task Comments unexpectedly returned a command")
	}

	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'c', Text: "c"})
	if m.comments == nil || m.comments.selected != 1 {
		t.Fatalf("gc Comments state = %#v, want open with newest selected", m.comments)
	}
	view := ansi.Strip(m.Render())
	for _, want := range []string{"TASK COMMENTS", "Grace", "Second"} {
		if !strings.Contains(view, want) {
			t.Fatalf("Comments overlay = %q, missing %q", view, want)
		}
	}
	m = pressKey(m, tea.Key{Code: 'k', Text: "k"})
	if m.comments.selected != 0 {
		t.Fatalf("k selected Comment %d, want oldest", m.comments.selected)
	}

	m = pressKey(m, tea.Key{Code: 'a', Text: "a"})
	m = pressKey(m, tea.Key{Text: "New comment"})
	m.editor.draft.Title = "Renamed"
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Text: "w"})
	updated, saveCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if saveCmd == nil {
		t.Fatal(":w in Comment composer returned no save command")
	}
	updated, _ = m.Update(saveCmd())
	m = updated.(*model)
	if store.savedCommentDraft.Text != "New comment" || store.savedCommentTask.Title != "Renamed" {
		t.Fatalf("atomic save received Task/Comment = %#v / %#v", store.savedCommentTask, store.savedCommentDraft)
	}
	if m.comments.composer != nil || len(m.comments.items) != 3 || m.comments.selected != 2 {
		t.Fatalf("Comments after save = %#v, want new selected Comment and closed composer", m.comments)
	}
}

func TestCommentWriteQuitSavesAndClosesOverlay(t *testing.T) {
	store := &fakeStore{taskComments: map[int64][]vimkunja.TaskComment{1: {}}}
	m := NewModel(Config{Store: store}).(*model)
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.commentsLoading = false

	for _, key := range []tea.Key{
		{Code: 'g', Text: "g"}, {Code: 'c', Text: "c"}, {Code: 'a', Text: "a"},
		{Text: "First comment"}, {Code: tea.KeyEscape}, {Code: ':', Text: ":"},
		{Text: "wq"},
	} {
		m = pressKey(m, key)
	}
	updated, saveCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if saveCmd == nil {
		t.Fatal(":wq returned no save command")
	}
	updated, _ = m.Update(saveCmd())
	m = updated.(*model)
	if store.savedCommentDraft.Text != "First comment" {
		t.Fatalf(":wq saved Comment = %q, want first Comment", store.savedCommentDraft.Text)
	}
	if m.comments != nil {
		t.Fatalf(":wq left Comments overlay open: %#v", m.comments)
	}
}

func TestCommentQuitClosesCleanOverlay(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.openComments()

	for _, key := range []tea.Key{
		{Code: ':', Text: ":"}, {Text: "q"}, {Code: tea.KeyEnter},
	} {
		m = pressKey(m, key)
	}
	if m.comments != nil {
		t.Fatalf(":q left clean Comments overlay open: %#v", m.comments)
	}
}

func TestReopeningSameTaskRejectsPreviousCommentsLoad(t *testing.T) {
	m := testModel()
	task := vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"}
	m.openExistingEditor(task)
	_ = m.loadTaskComments(task.ID)
	oldRequest := m.editor.commentsRequest

	m.openExistingEditor(task)
	_ = m.loadTaskComments(task.ID)
	updated, _ := m.Update(taskCommentsLoadedMsg{taskID: task.ID, request: oldRequest, comments: []vimkunja.TaskComment{{ID: 99, HTML: "stale"}}})
	m = updated.(*model)
	if len(m.editor.comments) != 0 {
		t.Fatalf("reopened Task accepted previous Comments load: %#v", m.editor.comments)
	}
}

func TestStaleInitialCommentsLoadCannotHideNewlySavedComment(t *testing.T) {
	store := &fakeStore{taskComments: map[int64][]vimkunja.TaskComment{1: {}}}
	m := NewModel(Config{Store: store}).(*model)
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	_ = m.loadTaskComments(1) // request 1 remains in flight with an empty snapshot
	m.openComments()
	m.comments.composer = &commentComposer{text: "New", mode: editorNormal}
	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Text: "w"})
	updated, saveCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	updated, _ = m.Update(saveCmd())
	m = updated.(*model)
	if len(m.comments.items) != 1 {
		t.Fatalf("saved Comment count = %d, want 1", len(m.comments.items))
	}

	updated, _ = m.Update(taskCommentsLoadedMsg{taskID: 1, request: 1, comments: nil})
	m = updated.(*model)
	if len(m.comments.items) != 1 || m.comments.items[0].HTML != "<p>New</p>" {
		t.Fatalf("stale load replaced saved Comment: %#v", m.comments.items)
	}
}

func TestCommentComposerPastesSystemClipboardLikeDescription(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.comments = &commentsUI{composer: &commentComposer{mode: editorInsert}}
	m.clipboardRead = func() (string, error) { return "pasted\ntext", nil }
	updated, pasteCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'v', Mod: tea.ModCtrl}))
	m = updated.(*model)
	if pasteCmd == nil {
		t.Fatal("Comment Ctrl+V returned no clipboard command")
	}
	updated, _ = m.Update(pasteCmd())
	m = updated.(*model)
	if got := m.comments.composer.text; got != "pasted\ntext" {
		t.Fatalf("Comment clipboard text = %q", got)
	}
}

func TestCommentSaveBlocksDuplicateSubmissionUntilCompletion(t *testing.T) {
	store := &fakeStore{taskComments: map[int64][]vimkunja.TaskComment{1: {}}}
	m := NewModel(Config{Store: store}).(*model)
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.comments = &commentsUI{composer: &commentComposer{text: "Once", mode: editorNormal}}

	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Text: "w"})
	updated, firstSave := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	if firstSave == nil {
		t.Fatal("first :w returned no save command")
	}
	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Text: "w"})
	_, duplicateSave := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if duplicateSave != nil {
		t.Fatal("second :w started a duplicate save while the first was in flight")
	}
}

func TestFailedCommentSaveKeepsInitialLoadValid(t *testing.T) {
	store := &fakeStore{taskComments: map[int64][]vimkunja.TaskComment{}, commentErr: errors.New("write failed")}
	m := NewModel(Config{Store: store}).(*model)
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	_ = m.loadTaskComments(1)
	request := m.editor.commentsRequest
	m.openComments()
	m.comments.composer = &commentComposer{text: "Draft", mode: editorNormal}

	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Text: "w"})
	updated, saveCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	updated, _ = m.Update(saveCmd())
	m = updated.(*model)
	updated, _ = m.Update(taskCommentsLoadedMsg{taskID: 1, request: request, comments: []vimkunja.TaskComment{{ID: 1, HTML: "<p>Existing</p>"}}})
	m = updated.(*model)
	if len(m.editor.comments) != 1 || m.editor.commentsLoading {
		t.Fatalf("failed save invalidated initial load: comments=%#v loading=%v", m.editor.comments, m.editor.commentsLoading)
	}
}

func TestOpeningCommentsKeepsNewestSelectedCommentBodyVisible(t *testing.T) {
	m := testModel()
	m.width, m.height, m.viewportReceived = 60, 16, true
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.commentsLoading = false
	for id := int64(1); id <= 8; id++ {
		body := fmt.Sprintf("Older body %d", id)
		if id == 8 {
			body = "Newest body must be visible"
		}
		m.editor.comments = append(m.editor.comments, vimkunja.TaskComment{ID: id, HTML: "<p>" + body + "</p>", Author: "Ada", Created: time.Date(2026, 9, 4, int(id), 0, 0, 0, time.UTC)})
	}
	m.openComments()
	if view := ansi.Strip(m.Render()); !strings.Contains(view, "Newest body must be visible") {
		t.Fatalf("newest selected Comment body is outside initial viewport: %q", view)
	}
}

func TestTaskCommentsDeleteOnlyOwnComment(t *testing.T) {
	store := &fakeStore{taskComments: map[int64][]vimkunja.TaskComment{1: {
		{ID: 1, TaskID: 1, Author: "Local", CanDelete: true},
		{ID: 2, TaskID: 1, Author: "Ada"},
	}}}
	m := NewModel(Config{Store: store}).(*model)
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.editor.comments = append([]vimkunja.TaskComment(nil), store.taskComments[1]...)
	m.openComments()
	m.comments.selected = 0

	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	updated, deleteCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'd', Text: "d"}))
	m = updated.(*model)
	if !m.commentDeleteConfirm || deleteCmd != nil {
		t.Fatalf("own Comment dd = confirm:%v command:%v", m.commentDeleteConfirm, deleteCmd != nil)
	}
	updated, deleteCmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	m = updated.(*model)
	if deleteCmd == nil {
		t.Fatal("delete confirmation did not start persistence")
	}
	updated, _ = m.Update(deleteCmd())
	m = updated.(*model)
	if store.deletedCommentID != 1 || len(m.comments.items) != 1 || m.comments.items[0].ID != 2 {
		t.Fatalf("deleted Comment = id:%d items:%#v", store.deletedCommentID, m.comments.items)
	}

	m.comments.selected = 0
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	m = pressKey(m, tea.Key{Code: 'd', Text: "d"})
	if m.commentDeleteConfirm || !strings.Contains(m.lastErr, "only the Comment author") {
		t.Fatalf("foreign Comment dd = confirm:%v error:%q", m.commentDeleteConfirm, m.lastErr)
	}
}

func TestTaskCommentsOverlayStaysWithinNormalAndNarrowViewports(t *testing.T) {
	for _, size := range []struct{ width, height int }{{100, 35}, {25, 8}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			m := testModel()
			m.width, m.height, m.viewportReceived = size.width, size.height, true
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
			m.editor.commentsLoading = false
			m.editor.comments = []vimkunja.TaskComment{{ID: 1, TaskID: 1, HTML: "<p>Long comment content that must remain inside the available terminal viewport.</p>", Author: "Ada", Created: time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)}}
			m.openComments()
			view := ansi.Strip(m.Render())
			lines := strings.Split(view, "\n")
			if len(lines) > size.height {
				t.Fatalf("Comments overlay height = %d, viewport = %d", len(lines), size.height)
			}
			for _, line := range lines {
				if lipgloss.Width(line) > size.width {
					t.Fatalf("Comments overlay row width = %d, viewport = %d: %q", lipgloss.Width(line), size.width, line)
				}
			}
		})
	}
}

func TestFailedTaskCommentSaveKeepsBothDrafts(t *testing.T) {
	store := &fakeStore{taskComments: map[int64][]vimkunja.TaskComment{}, commentErr: errors.New("database busy")}
	m := NewModel(Config{Store: store}).(*model)
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Original"})
	m.editor.draft.Title = "Renamed"
	m.comments = &commentsUI{composer: &commentComposer{text: "Unsaved", mode: editorNormal}}

	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Text: "w"})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(*model)
	updated, _ = m.Update(cmd())
	m = updated.(*model)
	if m.editor.draft.Title != "Renamed" || m.comments.composer == nil || m.comments.composer.text != "Unsaved" {
		t.Fatalf("failed save lost Task or Comment draft: editor=%#v comments=%#v", m.editor, m.comments)
	}
	if !strings.Contains(m.lastErr, "database busy") {
		t.Fatalf("failed save error = %q", m.lastErr)
	}
}

func TestCommentForceQuitDiscardsTaskAndCommentDrafts(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Original"})
	m.editor.draft.Title = "Renamed"
	m.comments = &commentsUI{composer: &commentComposer{text: "Unsaved", mode: editorNormal}}
	m = pressKey(m, tea.Key{Code: ':', Text: ":"})
	m = pressKey(m, tea.Key{Text: "q!"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if m.comments != nil || m.editor.draft.Title != "Original" {
		t.Fatalf(":q! left Comments or Task draft: comments=%#v title=%q", m.comments, m.editor.draft.Title)
	}
}

func TestTaskDetailShowsCommentsEntryAndGNFocusesDescription(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 35
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "Description"})
	m.editor.comments = []vimkunja.TaskComment{{ID: 1}, {ID: 2}}
	m.editor.commentsLoading = false
	for _, height := range []int{20, 12, 8} {
		view := ansi.Strip(renderTaskDetail(m, 80, height))
		if !strings.Contains(view, "Comments (2)") {
			t.Fatalf("Task Detail height %d = %q, missing pinned Comments entry", height, view)
		}
	}
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'n', Text: "n"})
	if m.editor.field != fieldDescription || m.editor.mode != editorNormal {
		t.Fatalf("gn = field:%v mode:%v, want Description Normal Mode", m.editor.field, m.editor.mode)
	}
}

func TestGNThenEnterActivatesDescriptionEditing(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 10
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "Text"})
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'n', Text: "n"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if !m.editor.descriptionActive || m.editor.mode != editorNormal {
		t.Fatalf("gn then Enter = active:%v mode:%v, want active Description Normal Mode", m.editor.descriptionActive, m.editor.mode)
	}
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	if m.editor.mode != editorInsert {
		t.Fatalf("i in active Description = mode:%v, want Insert", m.editor.mode)
	}
}

func TestDescriptionCursorHidesWhenPinnedCommentsLeaveNoTextRow(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 30 // Detail text height is 12 rows.
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "Text"})
	m.editor.field = fieldDescription
	if cursor := m.View().Cursor; cursor != nil {
		t.Fatalf("hidden Description body retained cursor %#v", cursor)
	}
}

func TestHiddenDescriptionInExpandedCompactDetailHasNoCursor(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 22 // Compact Detail with a pinned Comments row.
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "Text"})
	m.editor.moreFieldsExpanded = true
	m.editor.field = fieldDescription
	m.editor.detailScroll = 0 // Metadata fills the visible compact viewport.
	if cursor := m.View().Cursor; cursor != nil {
		t.Fatalf("hidden Description retained cursor %#v", cursor)
	}
}

func TestHTMLDescriptionEditingReportsReadOnlyReason(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "<p>HTML</p>"})
	m.editor.field = fieldDescription
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	if m.editor.mode != editorNormal || !strings.Contains(m.lastErr, "read-only") {
		t.Fatalf("HTML Description edit = mode:%v error:%q, want Normal Mode with read-only reason", m.editor.mode, m.lastErr)
	}
}

func TestGNThenITypesIntoPlainTextDescription(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 10
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "Text"})
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'n', Text: "n"})
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Text: "!"})
	if m.editor.mode != editorInsert || m.editor.draft.Description != "!Text" {
		t.Fatalf("gn then i = mode:%v Description:%q, want Insert !Text", m.editor.mode, m.editor.draft.Description)
	}
}

func TestEmptyCommentEscDiscardsWithoutError(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task"})
	m.openComments()
	m.startCommentComposer()
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	if m.comments == nil || m.comments.composer != nil || m.lastErr != "" {
		t.Fatalf("Esc on empty Comment = comments:%#v error:%q, want discarded composer without error", m.comments, m.lastErr)
	}
}

func TestGCRejectsCommentsForUnsavedTask(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
	m = pressKey(m, tea.Key{Code: 'c', Text: "c"})
	if m.comments != nil || !strings.Contains(m.lastErr, "Save Task") {
		t.Fatalf("gc on new Task = comments:%#v error:%q", m.comments, m.lastErr)
	}
}

func TestDueDateAndPriorityCanBeClearedWithXOrDD(t *testing.T) {
	cases := []struct {
		name  string
		field editorField
		keys  []tea.Key
	}{
		{name: "due date x", field: fieldDueDate, keys: []tea.Key{{Code: 'x', Text: "x"}}},
		{name: "due date dd", field: fieldDueDate, keys: []tea.Key{{Code: 'd', Text: "d"}, {Code: 'd', Text: "d"}}},
		{name: "priority x", field: fieldPriority, keys: []tea.Key{{Code: 'x', Text: "x"}}},
		{name: "priority dd", field: fieldPriority, keys: []tea.Key{{Code: 'd', Text: "d"}, {Code: 'd', Text: "d"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", DueDate: "2026-10-04", Priority: 3})
			m.editor.field = tc.field
			for _, key := range tc.keys {
				m = pressKey(m, key)
			}
			if m.editor.draft.DueDate != "" && tc.field == fieldDueDate {
				t.Fatalf("Due Date = %q, want empty", m.editor.draft.DueDate)
			}
			if m.editor.draft.Priority != 0 && tc.field == fieldPriority {
				t.Fatalf("Priority = %d, want None", m.editor.draft.Priority)
			}
			if !m.editor.dirty() {
				t.Fatal("clearing optional field did not mark draft dirty")
			}
		})
	}
}

func TestDetailFieldShortcutsAppearInWideFooterAndHelp(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.openNewEditor()
	footer := ansi.Strip(m.renderStatusBar(320))
	for _, want := range []string{"gt title", "gd due", "gp priority", "gl labels", "gc comments", "gn description"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("wide Detail footer = %q, missing %q", footer, want)
		}
	}
	m.editor.field = fieldDueDate
	if clearFooter := ansi.Strip(m.renderStatusBar(120)); !strings.Contains(clearFooter, "x/dd clear") {
		t.Fatalf("Due Date footer = %q, missing clear hint", clearFooter)
	}
	help := strings.Join(helpLines, "\n")
	for _, want := range []string{"DETAIL FIELD SHORTCUTS", "gt", "gd", "gp", "gl", "gc", "gn", "x or dd clears"} {
		if !strings.Contains(help, want) {
			t.Fatalf("Help missing %q", want)
		}
	}
}

func TestDueDatePickerClearsRelativeInputsWhenReopened(t *testing.T) {
	m := testModel()
	m.currentProjectID = 1
	m.now = func() time.Time { return time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC) }
	m.openNewEditor()
	m.editor.field = fieldDueDate

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: '3', Text: "3"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})

	if got := m.editor.dueDatePickerInputs; got != [3]string{} {
		t.Fatalf("reopened picker inputs = %#v, want empty", got)
	}
	if rendered := ansi.Strip(renderTaskDetail(m, 80, 24)); strings.Contains(rendered, "In days: 3") {
		t.Fatalf("reopened picker retained prior days input: %q", rendered)
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
