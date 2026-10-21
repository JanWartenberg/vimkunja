package tui

import (
	"context"
	"errors"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

// Commands execute synchronously here; only delivery is reordered, without sleeps.
type navigationStore struct {
	*fakeStore
	loadErr error
}

func (s *navigationStore) ListViews(ctx context.Context, projectID int64, page, perPage int) ([]vimkunja.ProjectView, int, error) {
	views, pages, _ := s.fakeStore.ListViews(ctx, projectID, page, perPage)
	return views, pages, s.loadErr
}
func (s *navigationStore) ListTasksInView(ctx context.Context, projectID, viewID int64, page, perPage int) ([]vimkunja.Task, int, error) {
	tasks, pages, _ := s.fakeStore.ListTasksInView(ctx, projectID, viewID, page, perPage)
	return tasks, pages, s.loadErr
}
func (s *navigationStore) LoadManualKanbanBoard(ctx context.Context, projectID int64, viewID ...int64) (vimkunja.KanbanBoard, error) {
	board, _ := s.fakeStore.LoadManualKanbanBoard(ctx, projectID, viewID...)
	return board, s.loadErr
}
func (s *navigationStore) ListTasksForLabel(ctx context.Context, labelID int64, page, perPage int) ([]vimkunja.Task, int, error) {
	tasks, pages, _ := s.fakeStore.ListTasksForLabel(ctx, labelID, page, perPage)
	return tasks, pages, s.loadErr
}

func navigationModel() (*model, *navigationStore) {
	s := &navigationStore{fakeStore: &fakeStore{
		viewsByProject: map[int64][]vimkunja.ProjectView{
			1: {{ID: 10, ViewKind: "list"}, {ID: 11, ViewKind: "kanban", BucketConfigurationMode: "manual"}},
			2: {{ID: 20, ViewKind: "list"}, {ID: 21, ViewKind: "kanban", BucketConfigurationMode: "manual"}},
		},
		tasksByView: map[int64][]vimkunja.Task{10: {{ID: 101, ProjectID: 1, Title: "A"}}, 20: {{ID: 202, ProjectID: 2, Title: "B"}}},
		labelTasks:  map[int64][]vimkunja.Task{1: {{ID: 101, Title: "A"}}, 2: {{ID: 202, Title: "B"}}},
		kanbanBoards: map[int64]vimkunja.KanbanBoard{
			1: {View: vimkunja.ProjectView{ID: 11}, Buckets: []vimkunja.KanbanBucket{{ID: 1, Tasks: []vimkunja.Task{{ID: 101, ProjectID: 1}}}}},
			2: {View: vimkunja.ProjectView{ID: 21}, Buckets: []vimkunja.KanbanBucket{{ID: 2, Tasks: []vimkunja.Task{{ID: 202, ProjectID: 2}}}}},
		},
	}}
	m := NewModel(Config{Store: s}).(*model)
	m.projects = []vimkunja.Project{{ID: 1, Title: "A"}, {ID: 2, Title: "B"}}
	m.currentProjectID, m.currentViewID = 1, 10
	m.projectViews = s.viewsByProject[1]
	return m, s
}

func startNavigationLoad(m *model, kind string, project int) tea.Cmd {
	if kind == "overview" {
		return m.openLabelOverview(vimkunja.Label{ID: int64(project), Title: "Label"})
	}
	m.projectSelected = project - 1
	views := m.onProjectSelectionChanged()
	if kind == "views" {
		return views
	}
	if kind == "kanban" {
		m.projectViewPreference = projectViewKanban
	}
	_, cmd := m.Update(views())
	return cmd
}

type navigationState struct {
	tasks, projectTasks, overviewTasks                       []vimkunja.Task
	views                                                    []vimkunja.ProjectView
	board                                                    vimkunja.KanbanBoard
	projectID, viewID, pendingTaskID, pendingKanbanTaskID    int64
	taskSelected, bucketSelected, cardSelected               int
	taskLoading, viewLoading, kanbanActive, provisionConfirm bool
	lastErr                                                  string
	editor                                                   *taskEditor
	draft                                                    vimkunja.TaskDraft
}

func navigationSnapshot(m *model) navigationState {
	state := navigationState{
		tasks: m.tasks, projectTasks: m.projectTasks, overviewTasks: m.overviewTasks,
		views: m.projectViews, board: m.kanbanBoard,
		projectID: m.currentProjectID, viewID: m.currentViewID, pendingTaskID: m.pendingTaskID, pendingKanbanTaskID: m.pendingKanbanTaskID,
		taskSelected: m.taskSelected, bucketSelected: m.kanbanBucketSelected, cardSelected: m.kanbanTaskSelected,
		taskLoading: m.taskLoading, viewLoading: m.viewLoading, kanbanActive: m.kanbanActive, provisionConfirm: m.kanbanProvisionConfirm,
		lastErr: m.lastErr, editor: m.editor,
	}
	if m.editor != nil {
		state.draft = cloneTaskDraft(m.editor.draft)
	}
	return state
}

func TestNavigationLoadsIgnoreObsoleteResponses(t *testing.T) {
	for _, kind := range []string{"tasks", "views", "kanban", "overview"} {
		for _, returnToA := range []bool{false, true} {
			for _, oldError := range []bool{false, true} {
				for _, completeCurrent := range []bool{false, true} {
					name := kind
					if returnToA {
						name += "/A-B-A"
					} else {
						name += "/A-B"
					}
					if oldError {
						name += "/old-error"
					} else {
						name += "/old-success"
					}
					if completeCurrent {
						name += "/current-complete"
					} else {
						name += "/current-pending"
					}
					t.Run(name, func(t *testing.T) {
						m, s := navigationModel()
						oldCmd := startNavigationLoad(m, kind, 1)
						if oldError {
							s.loadErr = errors.New("obsolete failure")
						}
						old := oldCmd()
						s.loadErr = nil
						current := startNavigationLoad(m, kind, 2)
						if returnToA {
							current = startNavigationLoad(m, kind, 1)
						}
						if completeCurrent {
							_, next := m.Update(current())
							if kind == "views" && next != nil {
								m.Update(next())
							}
							project := int64(2)
							if returnToA {
								project = 1
							}
							if m.taskLoading || m.viewLoading {
								t.Fatal("current response did not finish loading")
							}
							if kind == "kanban" {
								if !m.kanbanActive || m.kanbanBoard.View.ID != project*10+1 {
									t.Fatal("current Board was not adopted")
								}
							} else if len(m.tasks) != 1 || m.tasks[0].ID != project*101 {
								t.Fatalf("current Tasks were not adopted: %v", m.tasks)
							}
						}
						// Unrelated errors, selection and drafts must survive stale deliveries too.
						m.lastErr = "current error"
						m.pendingTaskID = 999
						m.pendingKanbanTaskID = 998
						m.taskSelected = 3
						m.openExistingEditor(vimkunja.Task{ID: 777, Title: "Original"})
						m.editor.draft.Title = "Unsaved"
						before := navigationSnapshot(m)
						_, next := m.Update(old)
						if next != nil || !reflect.DeepEqual(before, navigationSnapshot(m)) {
							t.Fatalf("obsolete %s response changed current navigation/loading/error/selection/draft (loading %v→%v, viewLoading %v→%v, error %q→%q, tasks %v→%v)", kind, before.taskLoading, m.taskLoading, before.viewLoading, m.viewLoading, before.lastErr, m.lastErr, before.tasks, m.tasks)
						}
					})
				}
			}
		}
	}
}

func TestOverviewLoadCapturesLabelBeforeCommandExecution(t *testing.T) {
	m, _ := navigationModel()
	old := m.openLabelOverview(vimkunja.Label{ID: 1})
	_ = m.openLabelOverview(vimkunja.Label{ID: 2})
	msg := old().(overviewTasksLoadedMsg)
	if len(msg.tasks) != 1 || msg.tasks[0].ID != 101 {
		t.Fatalf("old command loaded a different Label: %+v", msg.tasks)
	}
}

func TestListResponseCannotReplacePendingKanban(t *testing.T) {
	m, _ := navigationModel()
	old := startNavigationLoad(m, "tasks", 1)()
	_, board := kanbanCommand(m, nil)
	before := navigationSnapshot(m)
	m.Update(old)
	if !reflect.DeepEqual(before, navigationSnapshot(m)) {
		t.Fatal("List response changed pending Kanban")
	}
	m.Update(board())
	if !m.kanbanActive || m.currentViewID != 11 {
		t.Fatal("current Kanban did not load")
	}
}

func TestNavigationLoadsAcceptCurrentErrors(t *testing.T) {
	for _, kind := range []string{"tasks", "views", "kanban", "overview"} {
		t.Run(kind, func(t *testing.T) {
			m, s := navigationModel()
			cmd := startNavigationLoad(m, kind, 1)
			s.loadErr = errors.New("current load failed")
			m.Update(cmd())
			if m.lastErr != "current load failed" {
				t.Fatalf("current error lost: %q", m.lastErr)
			}
			if kind == "views" {
				if m.viewLoading {
					t.Fatal("Views still loading")
				}
			} else if m.taskLoading {
				t.Fatal("Tasks still loading")
			}
		})
	}
}

func TestNavigationLoadsInvalidateBeforeNewViewsArrive(t *testing.T) {
	for _, kind := range []string{"tasks", "kanban"} {
		t.Run(kind, func(t *testing.T) {
			m, _ := navigationModel()
			old := startNavigationLoad(m, kind, 1)()
			// Return to the same Project, but keep its replacement Views pending.
			m.projectSelected = 1
			_ = m.onProjectSelectionChanged()
			m.projectSelected = 0
			_ = m.onProjectSelectionChanged()
			before := navigationSnapshot(m)
			m.Update(old)
			if !reflect.DeepEqual(before, navigationSnapshot(m)) {
				t.Fatal("old Tasks changed navigation while Views pending")
			}
		})
	}
}

func TestNavigationLoadsCannotReplaceOtherScopes(t *testing.T) {
	for _, oldKind := range []string{"tasks", "kanban", "overview"} {
		t.Run(oldKind, func(t *testing.T) {
			m, _ := navigationModel()
			old := startNavigationLoad(m, oldKind, 1)()
			var current tea.Cmd
			if oldKind == "overview" {
				current = m.restoreProjectScope()
			} else {
				current = m.openOverview()
			}
			before := navigationSnapshot(m)
			m.Update(old)
			if !reflect.DeepEqual(before, navigationSnapshot(m)) {
				t.Fatal("old load changed another scope")
			}
			m.Update(current())
			if m.taskLoading {
				t.Fatal("current load did not finish")
			}
		})
	}
}

func TestNavigationLoadsCannotRestoreEmptyProjects(t *testing.T) {
	for _, kind := range []string{"tasks", "views", "kanban"} {
		t.Run(kind, func(t *testing.T) {
			m, _ := navigationModel()
			old := startNavigationLoad(m, kind, 1)()
			m.Update(projectsLoadedMsg{})
			before := navigationSnapshot(m)
			m.Update(old)
			if !reflect.DeepEqual(before, navigationSnapshot(m)) {
				t.Fatal("old load restored cleared Project context")
			}
		})
	}
}

func TestNavigationLoadsCannotDismissKanbanViewPicker(t *testing.T) {
	m, _ := navigationModel()
	old := startNavigationLoad(m, "tasks", 1)()
	m.projectViews = append(m.projectViews, vimkunja.ProjectView{ID: 12, ViewKind: "kanban", BucketConfigurationMode: "manual"})
	m.beginKanbanViewPicker()
	before := navigationSnapshot(m)
	m.Update(old)
	if !reflect.DeepEqual(before, navigationSnapshot(m)) || !m.kanbanViewPicker {
		t.Fatal("old load changed Kanban View picker")
	}
}

func TestOverviewLoadCannotReplaceRestoredSearchOrigin(t *testing.T) {
	m, _ := navigationModel()
	m.Update(startNavigationLoad(m, "overview", 1)())
	origin := m.captureSearchOrigin()
	old := m.openLabelOverview(vimkunja.Label{ID: 2})()
	m.searchReturn = &searchReturn{origin: origin}
	m.restoreLastSearch()
	before := navigationSnapshot(m)
	m.Update(old)
	if !reflect.DeepEqual(before, navigationSnapshot(m)) {
		t.Fatal("old overview replaced restored Search origin")
	}
}

func TestPendingViewsCannotChangeNewOverview(t *testing.T) {
	for _, loadError := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "error"}[loadError], func(t *testing.T) {
			m, s := navigationModel()
			old := startNavigationLoad(m, "views", 1)
			if loadError {
				s.loadErr = errors.New("old Views failed")
			}
			msg := old()
			s.loadErr = nil
			overview := m.openOverview()
			m.lastErr = "current error"
			before := navigationSnapshot(m)
			m.Update(msg)
			if !reflect.DeepEqual(before, navigationSnapshot(m)) {
				t.Fatal("pending Views changed new Overview")
			}
			m.Update(overview())
			// Returning must obtain usable Views when the old metadata was discarded.
			views := m.restoreProjectScope()
			_, tasks := m.Update(views())
			if tasks == nil {
				t.Fatal("return from Overview did not reload missing Views")
			}
			m.Update(tasks())
			if m.currentViewID != 10 || len(m.tasks) != 1 || m.tasks[0].ID != 101 {
				t.Fatalf("return lost Project context: view=%d tasks=%v", m.currentViewID, m.tasks)
			}
		})
	}
}

func TestListCommandAfterOverviewReloadsPendingProjectViews(t *testing.T) {
	m, _ := navigationModel()
	// A's metadata is loaded when B is selected; B's metadata is still pending.
	m.projectSelected = 1
	oldViews := m.onProjectSelectionChanged()
	m.Update(m.openOverview()())
	m.Update(oldViews())
	_, cmd := listCommand(m, nil)
	_, next := m.Update(cmd())
	if next != nil {
		m.Update(next())
	}
	if m.currentProjectID != 2 || m.currentViewID != 20 || len(m.tasks) != 1 || m.tasks[0].ProjectID != 2 {
		t.Fatalf("List return used another Project's View: project=%d view=%d tasks=%v", m.currentProjectID, m.currentViewID, m.tasks)
	}
}

func TestProjectMetadataRefreshKeepsPendingOverviewValid(t *testing.T) {
	m, _ := navigationModel()
	overview := m.openLabelOverview(vimkunja.Label{ID: 1})
	_, views := m.Update(projectsLoadedMsg{projects: m.projects})
	m.Update(views())
	m.Update(overview())
	if m.taskLoading || len(m.tasks) != 1 || m.tasks[0].ID != 101 {
		t.Fatalf("metadata refresh cancelled current Overview: loading=%v tasks=%v", m.taskLoading, m.tasks)
	}
}

func TestKanbanAfterOverviewDoesNotUsePreviousProjectViewPicker(t *testing.T) {
	m, _ := navigationModel()
	m.projectViews = append(m.projectViews, vimkunja.ProjectView{ID: 12, ViewKind: "kanban", BucketConfigurationMode: "manual"})
	m.projectSelected = 1
	oldViews := m.onProjectSelectionChanged()
	m.Update(m.openOverview()())
	m.Update(oldViews())
	_, board := kanbanCommand(m, nil)
	if board == nil || m.kanbanViewPicker {
		t.Fatal("Kanban offered the previous Project's View picker")
	}
	_, next := m.Update(board())
	if next != nil {
		m.Update(next())
	}
	if m.currentViewID != 21 {
		t.Fatalf("Kanban return View=%d, want B's View 21", m.currentViewID)
	}
	_, list := listCommand(m, nil)
	if list == nil {
		t.Fatal("Kanban return lost B's List View metadata")
	}
	m.Update(list())
	if m.currentViewID != 20 {
		t.Fatal("List command after Kanban did not use B's View")
	}
}
