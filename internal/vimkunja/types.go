package vimkunja

import "time"

// Project represents a Vikunja project (formerly called "list" in older versions).
type Project struct {
	ID              int64   `json:"id"`
	Identifier      string  `json:"identifier"`
	Title           string  `json:"name"`
	Description     string  `json:"description"`
	ParentProjectID int64   `json:"parent_project_id"`
	Position        float64 `json:"position"`
}

// Label is a Vikunja label. TaskCount is populated only when listing Labels
// for the global Label Overview.
type Label struct {
	ID        int64
	Title     string
	TaskCount int64
}

// SearchData is the complete locally browsable data set used by Global Search.
// It deliberately excludes archived Projects and their Tasks.
type SearchData struct {
	Projects []Project
	Tasks    []Task
	Labels   []Label
}

// ProjectMoveDirection identifies one adjacent move within a Project's siblings.
type ProjectMoveDirection int

const (
	ProjectMovePrevious ProjectMoveDirection = -1
	ProjectMoveNext     ProjectMoveDirection = 1
)

// ProjectView represents a view within a project (Kanban, List, Table, Gantt, etc.).
type ProjectView struct {
	ID                      int64  `json:"id"`
	Title                   string `json:"title"`
	ViewKind                string `json:"view_kind"`
	BucketConfigurationMode string `json:"bucket_configuration_mode"`
	DefaultBucketID         int64  `json:"default_bucket_id"`
	DoneBucketID            int64  `json:"done_bucket_id"`
}

// KanbanBoardAvailability explains whether a Project has one unambiguous
// manual Kanban View that Vimkunja can render.
type KanbanBoardAvailability int

const (
	KanbanBoardAvailable KanbanBoardAvailability = iota
	KanbanBoardUnavailable
	KanbanBoardAmbiguous
)

// KanbanBucket is a persisted Bucket and its Tasks in the active Kanban View.
type KanbanBucket struct {
	ID       int64
	Title    string
	Position float64
	Tasks    []Task
}

// KanbanBoard is the Store-owned read model for one manual Kanban View.
type KanbanBoard struct {
	Availability KanbanBoardAvailability
	View         ProjectView
	Buckets      []KanbanBucket
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

// TaskWorkflow contains the unambiguous manual-Kanban status needed by task
// list filters. Available is false when the project has no unique manual
// Kanban View or when its workflow status cannot be resolved.
type TaskWorkflow struct {
	Available       bool
	CurrentBucketID int64
	DefaultBucketID int64
	DoneBucketID    int64
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
	Workflow          TaskWorkflow
	// Identifier is present on some responses; we still can fall back to "#<index>".
	Identifier string `json:"identifier"`
}

// TaskComment is one persisted contribution in a Task's chronological Comment thread.
type TaskComment struct {
	ID        int64
	TaskID    int64
	HTML      string
	Author    string
	CanDelete bool // True only for the sole local User, matching Vikunja's author-only rule.
	Created   time.Time
	Updated   time.Time
}

func (c TaskComment) Edited() bool {
	return c.Updated.After(c.Created)
}

// TaskCommentDraft is the plain-text content of a new Comment before it is
// safely serialized to Vikunja-compatible HTML.
type TaskCommentDraft struct {
	Text string
}

// TaskDraft contains the editable fields of a task. DueDate is either empty
// (stored as NULL) or an ISO date in YYYY-MM-DD form. CreatedByID is optional:
// SQLiteStore falls back to the target project's owner when it is zero.
// Existing updates preserve the full due timestamp when the date is unchanged;
// changing or clearing the date writes midnight UTC or NULL, respectively.
type TaskDraft struct {
	ID               int64
	ProjectID        int64
	Title            string
	Description      string
	Priority         int64
	DueDate          string
	WorkflowBucketID int64 // Zero omits workflow writes; an unchanged existing assignment preserves Done.
	Done             bool
	DoneSet          bool // Done participates in an existing Task update only when changed in the editor.
	LabelIDs         []int64
	LabelIDsSet      bool // Labels participate only after the Detail has loaded or changed them.
	CreatedByID      int64
}

// TaskBucketAssignment is one persisted Bucket assignment for a Task. A Task
// may have one assignment per Project View, so undo snapshots retain all of
// them rather than only the currently displayed Workflow Status.
type TaskBucketAssignment struct {
	ViewID   int64
	BucketID int64
}

// TaskUndoState is the normalized, complete persisted state covered by Task
// undo/redo. It includes the full due timestamp but excludes audit timestamps,
// position, Comments, and editor drafts.
type TaskUndoState struct {
	TaskID            int64
	ProjectID         int64
	Identifier        string
	Title             string
	Description       string
	Priority          int64
	DueDate           string
	DueDateStored     string // Exact SQLite text; when present it must encode DueDate.
	Done              bool
	LabelIDs          []int64
	BucketAssignments []TaskBucketAssignment
}
