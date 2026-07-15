package vimkunja

// Project represents a Vikunja project (formerly called "list" in older versions).
type Project struct {
	ID          int64  `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"name"`
	Description string `json:"description"`
}

// ProjectView represents a view within a project (Kanban, List, Table, Gantt, etc.).
type ProjectView struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	ViewKind string `json:"view_kind"`
}

// Task represents a single task within a project.
type Task struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	Title       string `json:"title"`
	Done        bool   `json:"done"`
	DueDate     string `json:"due_date"`
	Priority    int64  `json:"priority"`
	Index       int64  `json:"index"`
	Description string `json:"description"`
	// Identifier is present on some responses; we still can fall back to "#<index>".
	Identifier string `json:"identifier"`
}

// TaskDraft contains the editable fields of a task. DueDate is either empty
// (stored as NULL) or an ISO date in YYYY-MM-DD form. CreatedByID is optional:
// SQLiteStore falls back to the target project's owner when it is zero.
type TaskDraft struct {
	ID          int64
	ProjectID   int64
	Title       string
	Description string
	Priority    int64
	DueDate     string
	CreatedByID int64
}
