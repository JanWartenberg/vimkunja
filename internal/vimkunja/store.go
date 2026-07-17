package vimkunja

import "context"

// Store abstracts data access to the Vikunja database.
// This TUI uses SQLite-only (no REST API).
//
// All pagination parameters are 1-based.
type Store interface {
	ListProjects(ctx context.Context, page, perPage int) ([]Project, int, error)
	ListViews(ctx context.Context, projectID int64, page, perPage int) ([]ProjectView, int, error)
	ListTasksInView(ctx context.Context, projectID, viewID int64, page, perPage int) ([]Task, int, error)
	GetTaskWorkflowStatus(ctx context.Context, projectID, taskID int64) (WorkflowStatus, error)

	// ToggleTaskDone updates the 'done' flag (and done_at if possible).
	ToggleTaskDone(ctx context.Context, taskID int64, done bool) error
	CreateTask(ctx context.Context, draft TaskDraft) (Task, error)
	UpdateTask(ctx context.Context, draft TaskDraft) (Task, error)
}
