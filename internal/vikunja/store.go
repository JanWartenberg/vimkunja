package vikunja

import "context"

// Store abstracts where the data comes from (SQLite file vs REST API).
// MVP primarily uses the local SQLite file.
//
// All pagination parameters are 1-based.
type Store interface {
	ListProjects(ctx context.Context, page, perPage int) ([]Project, int, error)
	ListViews(ctx context.Context, projectID int64, page, perPage int) ([]ProjectView, int, error)
	ListTasksInView(ctx context.Context, projectID, viewID int64, page, perPage int) ([]Task, int, error)

	// ToggleTaskDone updates the 'done' flag (and done_at if possible).
	ToggleTaskDone(ctx context.Context, taskID int64, done bool) error
}
