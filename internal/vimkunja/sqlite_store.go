package vimkunja

import (
	"context"
	"database/sql"
	"fmt"
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
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &dueNull, &t.Priority, &indexNull, &descNull); err != nil {
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
	if _, err := s.db.ExecContext(ctx, q, boolToInt(done), doneAt, time.Now(), taskID); err != nil {
		return fmt.Errorf("toggle task done: %w", err)
	}

	// Phase 6.2: Move task to done_bucket_id when marking done (if configured)
	if done {
		// Query done_bucket_id for this task's project view
		qDoneBucket := `
			SELECT pv.done_bucket_id
			FROM project_views pv
			JOIN tasks t ON t.project_id = pv.project_id
			WHERE t.id = ?
			LIMIT 1
		`
		var doneBucketID sql.NullInt64
		err := s.db.QueryRowContext(ctx, qDoneBucket, taskID).Scan(&doneBucketID)
		if err != nil && err != sql.ErrNoRows {
			// Real error (not just "no view configured")
			return fmt.Errorf("query done_bucket_id: %w", err)
		}

		// If done_bucket_id is configured, move the task
		if doneBucketID.Valid {
			qMoveBucket := `
				UPDATE task_buckets
				SET bucket_id = ?
				WHERE task_id = ?
			`
			if _, err := s.db.ExecContext(ctx, qMoveBucket, doneBucketID.Int64, taskID); err != nil {
				return fmt.Errorf("move task to done bucket: %w", err)
			}
		}
	}
	// Note: When toggling to done=false, we leave task in current bucket
	// (user can manually move it if needed)

	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
