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

// WorkflowStatusAvailability explains whether a project's manual Kanban View
// supplies one unambiguous Workflow Status for a Task.
type WorkflowStatusAvailability int

const (
	WorkflowStatusAvailable WorkflowStatusAvailability = iota
	WorkflowStatusUnavailable
	WorkflowStatusAmbiguous
)

// WorkflowBucket is a Bucket available as a Workflow Status.
type WorkflowBucket struct {
	ID    int64
	Title string
}

// WorkflowStatus is a Task's Bucket assignment in its single manual Kanban
// View. Buckets preserve that View's configured order.
type WorkflowStatus struct {
	Availability  WorkflowStatusAvailability
	CurrentBucket WorkflowBucket
	Buckets       []WorkflowBucket
}

// Task represents a single task within a project.
type Task struct {
	ID                int64  `json:"id"`
	ProjectID         int64  `json:"project_id"`
	ProjectIdentifier string `json:"project_identifier"`
	ProjectTitle      string `json:"project_title"`
	Title             string `json:"title"`
	Done              bool   `json:"done"`
	DueDate           string `json:"due_date"`
	Priority          int64  `json:"priority"`
	Index             int64  `json:"index"`
	Description       string `json:"description"`
	// Identifier is present on some responses; we still can fall back to "#<index>".
	Identifier string `json:"identifier"`
}

// TaskDraft contains the editable fields of a task. DueDate is either empty
// (stored as NULL) or an ISO date in YYYY-MM-DD form. CreatedByID is optional:
// SQLiteStore falls back to the target project's owner when it is zero.
type TaskDraft struct {
	ID               int64
	ProjectID        int64
	Title            string
	Description      string
	Priority         int64
	DueDate          string
	WorkflowBucketID int64
	CreatedByID      int64
}
