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

	// A task's done state is global. Mirror it in the configured done bucket of
	// every view in its project, using each view's own bucket ID.
	if done {
		rows, err := tx.QueryContext(ctx, `
			SELECT pv.id, pv.done_bucket_id
			FROM project_views pv
			JOIN tasks t ON t.project_id = pv.project_id
			WHERE t.id = ? AND pv.done_bucket_id IS NOT NULL
		`, taskID)
		if err != nil {
			return fmt.Errorf("list done buckets: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var viewID, doneBucketID int64
			if err := rows.Scan(&viewID, &doneBucketID); err != nil {
				return fmt.Errorf("scan done bucket: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `
				UPDATE task_buckets
				SET bucket_id = ?
				WHERE task_id = ? AND project_view_id = ?
			`, doneBucketID, taskID, viewID); err != nil {
				return fmt.Errorf("move task to done bucket: %w", err)
			}
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate done buckets: %w", err)
		}
	}
	// Note: When toggling to done=false, we leave task in current bucket
	// (user can manually move it if needed)

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
