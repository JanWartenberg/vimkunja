package vimkunja

import "context"

// Store abstracts data access to the Vikunja database.
// This TUI uses SQLite-only (no REST API).
//
// All pagination parameters are 1-based.
type Store interface {
	ListProjects(ctx context.Context) ([]Project, error)
	MoveProjectAmongSiblings(ctx context.Context, projectID int64, direction ProjectMoveDirection) error
	ListViews(ctx context.Context, projectID int64, page, perPage int) ([]ProjectView, int, error)
	ListTasksInView(ctx context.Context, projectID, viewID int64, page, perPage int) ([]Task, int, error)
	LoadManualKanbanBoard(ctx context.Context, projectID int64) (KanbanBoard, error)
	ProvisionStandardManualKanbanView(ctx context.Context, projectID int64) (KanbanBoard, error)
	ListOverviewTasks(ctx context.Context, page, perPage int) ([]Task, int, error)
	LoadSearchData(ctx context.Context) (SearchData, error)
	ListTasksForLabel(ctx context.Context, labelID int64, page, perPage int) ([]Task, int, error)
	ListLabels(ctx context.Context) ([]Label, error)
	ListTaskLabels(ctx context.Context, taskID int64) ([]Label, error)
	CreateLabel(ctx context.Context, title string, createdByID int64) (Label, error)
	RenameLabel(ctx context.Context, labelID int64, title string) error
	DeleteLabel(ctx context.Context, labelID int64) error
	ReplaceTaskLabels(ctx context.Context, taskID int64, labelIDs []int64) error
	GetTaskWorkflowStatus(ctx context.Context, projectID, taskID int64) (WorkflowStatus, error)
	SetTaskWorkflowStatus(ctx context.Context, taskID, bucketID int64) error

	// ToggleTaskDone updates the 'done' flag (and done_at if possible).
	ToggleTaskDone(ctx context.Context, taskID int64, done bool) error
	DeleteTask(ctx context.Context, taskID int64) error
	CreateTask(ctx context.Context, draft TaskDraft) (Task, error)
	UpdateTask(ctx context.Context, draft TaskDraft) (Task, error)
}
