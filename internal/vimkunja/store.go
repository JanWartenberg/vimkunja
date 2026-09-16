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
	LoadManualKanbanBoard(ctx context.Context, projectID int64, viewID ...int64) (KanbanBoard, error)
	ProvisionStandardManualKanbanView(ctx context.Context, projectID int64) (KanbanBoard, error)
	ListOverviewTasks(ctx context.Context, page, perPage int) ([]Task, int, error)
	LoadSearchData(ctx context.Context) (SearchData, error)
	ListTasksForLabel(ctx context.Context, labelID int64, page, perPage int) ([]Task, int, error)
	ListLabels(ctx context.Context) ([]Label, error)
	ListTaskLabels(ctx context.Context, taskID int64) ([]Label, error)
	ListTaskComments(ctx context.Context, taskID int64) ([]TaskComment, error)
	SaveTaskWithComment(ctx context.Context, draft TaskDraft, comment TaskCommentDraft) (Task, TaskComment, error)
	DeleteTaskComment(ctx context.Context, taskID, commentID int64) error
	CreateLabel(ctx context.Context, title string) (Label, error)
	RenameLabel(ctx context.Context, labelID int64, title string) error
	DeleteLabel(ctx context.Context, labelID int64) error
	ReplaceTaskLabels(ctx context.Context, taskID int64, labelIDs []int64) error
	GetTaskWorkflowStatus(ctx context.Context, projectID, taskID int64, viewID ...int64) (WorkflowStatus, error)
	SetTaskWorkflowStatus(ctx context.Context, taskID, bucketID int64, viewID ...int64) error

	// ToggleTaskDone updates the 'done' flag (and done_at if possible).
	ToggleTaskDone(ctx context.Context, taskID int64, done bool, viewID ...int64) error
	DeleteTask(ctx context.Context, taskID int64) error
	CreateTask(ctx context.Context, draft TaskDraft) (Task, error)
	UpdateTask(ctx context.Context, draft TaskDraft) (Task, error)
	UpdateTaskTitle(ctx context.Context, taskID int64, title string) error

	// TaskUndoState and RestoreTaskUndoState form the persistence boundary for
	// session-local undo. Restore is atomic and writes only when the current
	// state still exactly matches expected.
	TaskUndoState(ctx context.Context, taskID int64) (TaskUndoState, error)
	RestoreTaskUndoState(ctx context.Context, expected, target TaskUndoState) error
}
