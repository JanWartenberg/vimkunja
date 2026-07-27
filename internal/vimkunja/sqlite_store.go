package vimkunja

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// Ensure at least one connection can validate the file.
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

const (
	viewKindList   = 0
	viewKindGantt  = 1
	viewKindTable  = 2
	viewKindKanban = 3
)

func viewKindToString(v int) string {
	switch v {
	case viewKindList:
		return "list"
	case viewKindGantt:
		return "gantt"
	case viewKindTable:
		return "table"
	case viewKindKanban:
		return "kanban"
	default:
		return "unknown"
	}
}

func (s *SQLiteStore) ListProjects(ctx context.Context, page, perPage int) ([]Project, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	var total int
	// Exclude archived projects like the API default.
	qTotal := `SELECT COUNT(1) FROM projects WHERE (is_archived = 0 OR is_archived IS NULL)`
	if err := s.db.QueryRowContext(ctx, qTotal).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	q := `SELECT id, identifier, title, description
		FROM projects
		WHERE (is_archived = 0 OR is_archived IS NULL)
		ORDER BY id
		LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, q, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		var identifier, description sql.NullString
		if err := rows.Scan(&p.ID, &identifier, &p.Title, &description); err != nil {
			return nil, 0, fmt.Errorf("scan project: %w", err)
		}
		if identifier.Valid {
			p.Identifier = identifier.String
		}
		if description.Valid {
			p.Description = description.String
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate projects: %w", err)
	}
	return out, totalPages, nil
}

func (s *SQLiteStore) ListViews(ctx context.Context, projectID int64, page, perPage int) ([]ProjectView, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 200
	}
	offset := (page - 1) * perPage

	var total int
	qTotal := `SELECT COUNT(1) FROM project_views WHERE project_id = ?`
	if err := s.db.QueryRowContext(ctx, qTotal, projectID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count views: %w", err)
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	q := `SELECT id, title, view_kind FROM project_views
		WHERE project_id = ?
		ORDER BY position ASC, id ASC
		LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, q, projectID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list views: %w", err)
	}
	defer rows.Close()

	var out []ProjectView
	for rows.Next() {
		var v ProjectView
		var viewKindInt int
		if err := rows.Scan(&v.ID, &v.Title, &viewKindInt); err != nil {
			return nil, 0, fmt.Errorf("scan view: %w", err)
		}
		v.ViewKind = viewKindToString(viewKindInt)
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate views: %w", err)
	}

	return out, totalPages, nil
}

func (s *SQLiteStore) ListTasksInView(ctx context.Context, projectID, viewID int64, page, perPage int) ([]Task, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	var total int
	// MVP: some schema versions (e.g. depending on export/source) may not
	// have a deleted_at column. For prototype purposes we just rely on the
	// project_id filter.
	qTotal := `SELECT COUNT(1)
		FROM tasks
		WHERE project_id = ?`
	if err := s.db.QueryRowContext(ctx, qTotal, projectID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	// MVP simplification: we ignore the view's filter/search/sort rules.
	// We still use task_positions.position for stable ordering.
	q := `SELECT
			t.id,
			t.project_id,
			t.title,
			t.done,
			t.due_date,
			t.priority,
			t."index",
			t.description
		FROM tasks t
		LEFT JOIN task_positions tp
			ON tp.task_id = t.id AND tp.project_view_id = ?
		WHERE t.project_id = ?
		ORDER BY COALESCE(tp.position, 0) ASC, t.id ASC
		LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, q, viewID, projectID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		var dueNull sql.NullTime
		var indexNull sql.NullInt64
		var descNull sql.NullString
		// Scanner order must match SELECT.
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Done, &dueNull, &t.Priority, &indexNull, &descNull); err != nil {
			return nil, 0, fmt.Errorf("scan task: %w", err)
		}
		t.Identifier = ""
		if dueNull.Valid {
			t.DueDate = dueNull.Time.Format("2006-01-02")
		}
		if indexNull.Valid {
			t.Index = indexNull.Int64
		}
		if descNull.Valid {
			t.Description = descNull.String
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate tasks: %w", err)
	}

	return out, totalPages, nil
}

// ListOverviewTasks returns Tasks from every non-archived Project with enough
// Project metadata for the cross-project Overview. It owns the Overview's
// default ordering so the TUI does not need to reproduce database semantics.
func (s *SQLiteStore) ListOverviewTasks(ctx context.Context, page, perPage int) ([]Task, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 200
	}
	offset := (page - 1) * perPage

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1)
		FROM tasks t JOIN projects p ON p.id = t.project_id
		WHERE (p.is_archived = 0 OR p.is_archived IS NULL)`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count overview tasks: %w", err)
	}
	totalPages := (total + perPage - 1) / perPage

	rows, err := s.db.QueryContext(ctx, `SELECT
		t.id, t.project_id, p.identifier, p.title, t.title, t.done, t.due_date,
		t.priority, t."index", t.description
		FROM tasks t JOIN projects p ON p.id = t.project_id
		WHERE (p.is_archived = 0 OR p.is_archived IS NULL)
		ORDER BY t.due_date IS NULL ASC, t.due_date ASC, t.priority DESC,
			p.identifier COLLATE NOCASE ASC, t.title COLLATE NOCASE ASC, t.id ASC
		LIMIT ? OFFSET ?`, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list overview tasks: %w", err)
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		var projectIdentifier, projectTitle, description sql.NullString
		var dueDate sql.NullTime
		var index sql.NullInt64
		if err := rows.Scan(&t.ID, &t.ProjectID, &projectIdentifier, &projectTitle, &t.Title, &t.Done, &dueDate, &t.Priority, &index, &description); err != nil {
			return nil, 0, fmt.Errorf("scan overview task: %w", err)
		}
		if projectIdentifier.Valid {
			t.ProjectIdentifier = projectIdentifier.String
		}
		if projectTitle.Valid {
			t.ProjectTitle = projectTitle.String
		}
		if dueDate.Valid {
			t.DueDate = dueDate.Time.Format("2006-01-02")
		}
		if index.Valid {
			t.Index = index.Int64
		}
		if description.Valid {
			t.Description = description.String
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate overview tasks: %w", err)
	}
	return out, totalPages, nil
}

// GetTaskWorkflowStatus returns a Task's Bucket assignment and available
// Buckets when the Project has exactly one manual Kanban View. With no manual
// Kanban View the status is unavailable; with several it is ambiguous.
func (s *SQLiteStore) GetTaskWorkflowStatus(ctx context.Context, projectID, taskID int64) (WorkflowStatus, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id
		FROM project_views
		WHERE project_id = ? AND view_kind = ?
		ORDER BY position ASC, id ASC
	`, projectID, viewKindKanban)
	if err != nil {
		return WorkflowStatus{}, fmt.Errorf("list manual kanban views: %w", err)
	}
	defer rows.Close()

	var viewIDs []int64
	for rows.Next() {
		var viewID int64
		if err := rows.Scan(&viewID); err != nil {
			return WorkflowStatus{}, fmt.Errorf("scan manual kanban view: %w", err)
		}
		viewIDs = append(viewIDs, viewID)
	}
	if err := rows.Err(); err != nil {
		return WorkflowStatus{}, fmt.Errorf("iterate manual kanban views: %w", err)
	}
	switch len(viewIDs) {
	case 0:
		return WorkflowStatus{Availability: WorkflowStatusUnavailable}, nil
	case 1:
		// Continue below.
	default:
		return WorkflowStatus{Availability: WorkflowStatusAmbiguous}, nil
	}

	buckets, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.title, tb.task_id IS NOT NULL
		FROM buckets b
		LEFT JOIN task_buckets tb
			ON tb.bucket_id = b.id
			AND tb.project_view_id = b.project_view_id
			AND tb.task_id = ?
		WHERE b.project_view_id = ?
		ORDER BY b.position ASC, b.id ASC
	`, taskID, viewIDs[0])
	if err != nil {
		return WorkflowStatus{}, fmt.Errorf("list workflow buckets: %w", err)
	}
	defer buckets.Close()

	status := WorkflowStatus{Availability: WorkflowStatusAvailable}
	for buckets.Next() {
		var bucket WorkflowBucket
		var current bool
		if err := buckets.Scan(&bucket.ID, &bucket.Title, &current); err != nil {
			return WorkflowStatus{}, fmt.Errorf("scan workflow bucket: %w", err)
		}
		status.Buckets = append(status.Buckets, bucket)
		if current {
			status.CurrentBucket = bucket
		}
	}
	if err := buckets.Err(); err != nil {
		return WorkflowStatus{}, fmt.Errorf("iterate workflow buckets: %w", err)
	}
	return status, nil
}

// SetTaskWorkflowStatus assigns a Task to a Bucket in its single manual Kanban
// View and atomically synchronizes its global Done state.
func (s *SQLiteStore) SetTaskWorkflowStatus(ctx context.Context, taskID, bucketID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin set workflow status transaction: %w", err)
	}
	defer tx.Rollback()

	var projectID int64
	if err := tx.QueryRowContext(ctx, `SELECT project_id FROM tasks WHERE id = ?`, taskID).Scan(&projectID); err != nil {
		return fmt.Errorf("get task project for workflow status: %w", err)
	}
	if err := setTaskWorkflowStatus(ctx, tx, taskID, projectID, bucketID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit set workflow status: %w", err)
	}
	return nil
}

func setTaskWorkflowStatus(ctx context.Context, tx *sql.Tx, taskID, projectID, bucketID int64) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, done_bucket_id
		FROM project_views
		WHERE project_id = ? AND view_kind = ?
		ORDER BY position ASC, id ASC
	`, projectID, viewKindKanban)
	if err != nil {
		return fmt.Errorf("list manual kanban views for workflow status: %w", err)
	}
	defer rows.Close()

	type workflowView struct {
		id           int64
		doneBucketID sql.NullInt64
	}
	var views []workflowView
	for rows.Next() {
		var view workflowView
		if err := rows.Scan(&view.id, &view.doneBucketID); err != nil {
			return fmt.Errorf("scan manual kanban view for workflow status: %w", err)
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate manual kanban views for workflow status: %w", err)
	}
	if len(views) != 1 {
		return fmt.Errorf("set workflow status: project has %d manual kanban views", len(views))
	}
	view := views[0]

	var bucketExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM buckets WHERE id = ? AND project_view_id = ?)`, bucketID, view.id).Scan(&bucketExists); err != nil {
		return fmt.Errorf("check workflow bucket: %w", err)
	}
	if !bucketExists {
		return fmt.Errorf("set workflow status: bucket does not belong to the manual kanban view")
	}

	done := view.doneBucketID.Valid && view.doneBucketID.Int64 == bucketID
	var doneAt any
	if done {
		doneAt = time.Now()
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET done = ?, done_at = ?, updated = ? WHERE id = ?`, boolToInt(done), doneAt, time.Now(), taskID); err != nil {
		return fmt.Errorf("update task workflow done state: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE task_buckets SET bucket_id = ? WHERE task_id = ? AND project_view_id = ?`, bucketID, taskID, view.id)
	if err != nil {
		return fmt.Errorf("update task workflow bucket: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count updated task workflow bucket: %w", err)
	}
	if changed == 0 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (?, ?, ?)`, taskID, bucketID, view.id); err != nil {
			return fmt.Errorf("insert task workflow bucket: %w", err)
		}
	}
	return nil
}

func syncTaskBucketsForDoneState(ctx context.Context, tx *sql.Tx, taskID int64, done bool) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT pv.id,
			CASE WHEN ? THEN pv.done_bucket_id ELSE pv.default_bucket_id END
		FROM project_views pv
		JOIN tasks t ON t.project_id = pv.project_id
		WHERE t.id = ?
			AND CASE WHEN ? THEN pv.done_bucket_id ELSE pv.default_bucket_id END IS NOT NULL
	`, boolToInt(done), taskID, boolToInt(done))
	if err != nil {
		return fmt.Errorf("list configured workflow buckets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var viewID, bucketID int64
		if err := rows.Scan(&viewID, &bucketID); err != nil {
			return fmt.Errorf("scan configured workflow bucket: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE task_buckets
			SET bucket_id = ?
			WHERE task_id = ? AND project_view_id = ?
		`, bucketID, taskID, viewID); err != nil {
			return fmt.Errorf("move task to configured workflow bucket: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate configured workflow buckets: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ToggleTaskDone(ctx context.Context, taskID int64, done bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin toggle task transaction: %w", err)
	}
	defer tx.Rollback()

	// Update done status, done_at timestamp, and ALWAYS bump 'updated' timestamp.
	// This ensures server-fidelity: webhooks, sync, and external tools see the change.
	var doneAt any
	if done {
		doneAt = time.Now()
	} else {
		doneAt = nil
	}

	// CRITICAL: Always bump 'updated' timestamp (Decision #2, SPEC_FUNDAMENT.md)
	q := `UPDATE tasks SET done = ?, done_at = ?, updated = ? WHERE id = ?`
	if _, err := tx.ExecContext(ctx, q, boolToInt(done), doneAt, time.Now(), taskID); err != nil {
		return fmt.Errorf("toggle task done: %w", err)
	}

	// A Task's Done state is global. Mirror it into every configured Done or
	// Default Bucket so reopening cannot leave a Task assigned to Done.
	if err := syncTaskBucketsForDoneState(ctx, tx, taskID, done); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit toggle task transaction: %w", err)
	}
	return nil
}

// CreateTask persists a new task and creates the default per-view position and
// bucket assignments expected by Vikunja's SQLite schema.
func (s *SQLiteStore) CreateTask(ctx context.Context, draft TaskDraft) (Task, error) {
	if err := validateDraft(draft); err != nil {
		return Task{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, fmt.Errorf("begin create task transaction: %w", err)
	}
	defer tx.Rollback()

	var ownerID int64
	if err := tx.QueryRowContext(ctx, `SELECT owner_id FROM projects WHERE id = ?`, draft.ProjectID).Scan(&ownerID); err != nil {
		return Task{}, fmt.Errorf("get project owner: %w", err)
	}
	createdByID := draft.CreatedByID
	if createdByID == 0 {
		createdByID = ownerID
	}
	index, err := nextTaskIndex(ctx, tx, draft.ProjectID)
	if err != nil {
		return Task{}, err
	}
	dueDate, err := dueDateValue(draft.DueDate)
	if err != nil {
		return Task{}, err
	}
	now := time.Now()
	result, err := tx.ExecContext(ctx, `
		INSERT INTO tasks (title, description, done, due_date, project_id, priority, "index", created, updated, created_by_id)
		VALUES (?, ?, 0, ?, ?, ?, ?, ?, ?, ?)
	`, draft.Title, draft.Description, dueDate, draft.ProjectID, draft.Priority, index, now, now, createdByID)
	if err != nil {
		return Task{}, fmt.Errorf("insert task: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Task{}, fmt.Errorf("get created task id: %w", err)
	}
	if err := initializeViewAssignments(ctx, tx, id, draft.ProjectID); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("commit create task: %w", err)
	}
	return Task{ID: id, ProjectID: draft.ProjectID, Title: draft.Title, Description: draft.Description, Priority: draft.Priority, DueDate: draft.DueDate, Index: index}, nil
}

// UpdateTask changes all editable fields. Moving a task also replaces its
// per-view assignments with the target project's defaults.
func (s *SQLiteStore) UpdateTask(ctx context.Context, draft TaskDraft) (Task, error) {
	if draft.ID == 0 {
		return Task{}, fmt.Errorf("update task: missing task id")
	}
	if err := validateDraft(draft); err != nil {
		return Task{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, fmt.Errorf("begin update task transaction: %w", err)
	}
	defer tx.Rollback()

	var oldProjectID int64
	var done bool
	if err := tx.QueryRowContext(ctx, `SELECT project_id, done FROM tasks WHERE id = ?`, draft.ID).Scan(&oldProjectID, &done); err != nil {
		return Task{}, fmt.Errorf("get task for update: %w", err)
	}
	dueDate, err := dueDateValue(draft.DueDate)
	if err != nil {
		return Task{}, err
	}
	index := int64(0)
	if oldProjectID != draft.ProjectID {
		index, err = nextTaskIndex(ctx, tx, draft.ProjectID)
		if err != nil {
			return Task{}, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM task_positions WHERE task_id = ? AND project_view_id IN (SELECT id FROM project_views WHERE project_id = ?)`, draft.ID, oldProjectID); err != nil {
			return Task{}, fmt.Errorf("remove old task positions: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM task_buckets WHERE task_id = ? AND project_view_id IN (SELECT id FROM project_views WHERE project_id = ?)`, draft.ID, oldProjectID); err != nil {
			return Task{}, fmt.Errorf("remove old task buckets: %w", err)
		}
	} else if err := tx.QueryRowContext(ctx, `SELECT "index" FROM tasks WHERE id = ?`, draft.ID).Scan(&index); err != nil {
		return Task{}, fmt.Errorf("get task index: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET title = ?, description = ?, due_date = ?, project_id = ?, priority = ?, "index" = ?, updated = ? WHERE id = ?`, draft.Title, draft.Description, dueDate, draft.ProjectID, draft.Priority, index, time.Now(), draft.ID); err != nil {
		return Task{}, fmt.Errorf("update task: %w", err)
	}
	if oldProjectID != draft.ProjectID {
		if err := initializeViewAssignments(ctx, tx, draft.ID, draft.ProjectID); err != nil {
			return Task{}, err
		}
	} else if draft.WorkflowBucketID != 0 {
		if err := setTaskWorkflowStatus(ctx, tx, draft.ID, draft.ProjectID, draft.WorkflowBucketID); err != nil {
			return Task{}, err
		}
		if err := tx.QueryRowContext(ctx, `SELECT done FROM tasks WHERE id = ?`, draft.ID).Scan(&done); err != nil {
			return Task{}, fmt.Errorf("get updated task done state: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("commit update task: %w", err)
	}
	return Task{ID: draft.ID, ProjectID: draft.ProjectID, Title: draft.Title, Description: draft.Description, Priority: draft.Priority, DueDate: draft.DueDate, Index: index, Done: done}, nil
}

func validateDraft(draft TaskDraft) error {
	if draft.ProjectID == 0 {
		return fmt.Errorf("project is required")
	}
	if strings.TrimSpace(draft.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if draft.Priority < 0 || draft.Priority > 5 {
		return fmt.Errorf("priority must be between 0 and 5")
	}
	_, err := dueDateValue(draft.DueDate)
	return err
}

func dueDateValue(value string) (any, error) {
	if value == "" {
		return nil, nil
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil || date.Format("2006-01-02") != value {
		return nil, fmt.Errorf("due date must use YYYY-MM-DD")
	}
	return date, nil
}

func nextTaskIndex(ctx context.Context, tx *sql.Tx, projectID int64) (int64, error) {
	var index int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX("index"), 0) + 1 FROM tasks WHERE project_id = ?`, projectID).Scan(&index); err != nil {
		return 0, fmt.Errorf("next task index: %w", err)
	}
	return index, nil
}

func initializeViewAssignments(ctx context.Context, tx *sql.Tx, taskID, projectID int64) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, default_bucket_id FROM project_views WHERE project_id = ?`, projectID)
	if err != nil {
		return fmt.Errorf("list target project views: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var viewID int64
		var defaultBucketID sql.NullInt64
		if err := rows.Scan(&viewID, &defaultBucketID); err != nil {
			return fmt.Errorf("scan target project view: %w", err)
		}
		var position float64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), 0) + 1 FROM task_positions WHERE project_view_id = ?`, viewID).Scan(&position); err != nil {
			return fmt.Errorf("next task position: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_positions (task_id, project_view_id, position) VALUES (?, ?, ?)`, taskID, viewID, position); err != nil {
			return fmt.Errorf("insert task position: %w", err)
		}
		if defaultBucketID.Valid {
			if _, err := tx.ExecContext(ctx, `INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (?, ?, ?)`, taskID, defaultBucketID.Int64, viewID); err != nil {
				return fmt.Errorf("insert default task bucket: %w", err)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate target project views: %w", err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
