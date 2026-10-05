package vimkunja

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

type undoQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// TaskUndoState returns a normalized snapshot of every persisted Task field
// covered by session-local undo.
func (s *SQLiteStore) TaskUndoState(ctx context.Context, taskID int64) (TaskUndoState, error) {
	return loadTaskUndoState(ctx, s.db, taskID)
}

func loadTaskUndoState(ctx context.Context, q undoQuerier, taskID int64) (TaskUndoState, error) {
	var state TaskUndoState
	var due sql.NullTime
	var description, projectIdentifier sql.NullString
	var index sql.NullInt64
	err := q.QueryRowContext(ctx, `
		SELECT t.id, t.project_id, COALESCE(p.identifier, ''), t."index",
			t.title, t.description, t.priority, t.due_date, t.done,
			COALESCE(CAST(t.due_date AS TEXT), '')
		FROM tasks t JOIN projects p ON p.id = t.project_id
		WHERE t.id = ?
	`, taskID).Scan(&state.TaskID, &state.ProjectID, &projectIdentifier, &index,
		&state.Title, &description, &state.Priority, &due, &state.Done, &state.DueDateStored)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskUndoState{}, fmt.Errorf("load Task undo state: Task %d not found", taskID)
		}
		return TaskUndoState{}, fmt.Errorf("load Task undo state: %w", err)
	}
	state.Description = description.String
	if due.Valid {
		state.DueDate = due.Time.Format("2006-01-02")
	}
	if projectIdentifier.String != "" && index.Valid {
		state.Identifier = fmt.Sprintf("%s-%d", projectIdentifier.String, index.Int64)
	}

	labelRows, err := q.QueryContext(ctx, `SELECT label_id FROM label_tasks WHERE task_id = ? ORDER BY label_id`, taskID)
	if err != nil {
		return TaskUndoState{}, fmt.Errorf("load Task undo labels: %w", err)
	}
	for labelRows.Next() {
		var labelID int64
		if err := labelRows.Scan(&labelID); err != nil {
			labelRows.Close()
			return TaskUndoState{}, fmt.Errorf("scan Task undo label: %w", err)
		}
		state.LabelIDs = append(state.LabelIDs, labelID)
	}
	if err := labelRows.Close(); err != nil {
		return TaskUndoState{}, fmt.Errorf("close Task undo labels: %w", err)
	}
	if err := labelRows.Err(); err != nil {
		return TaskUndoState{}, fmt.Errorf("iterate Task undo labels: %w", err)
	}

	bucketRows, err := q.QueryContext(ctx, `
		SELECT project_view_id, bucket_id FROM task_buckets
		WHERE task_id = ? ORDER BY project_view_id, bucket_id
	`, taskID)
	if err != nil {
		return TaskUndoState{}, fmt.Errorf("load Task undo Buckets: %w", err)
	}
	for bucketRows.Next() {
		var assignment TaskBucketAssignment
		if err := bucketRows.Scan(&assignment.ViewID, &assignment.BucketID); err != nil {
			bucketRows.Close()
			return TaskUndoState{}, fmt.Errorf("scan Task undo Bucket: %w", err)
		}
		state.BucketAssignments = append(state.BucketAssignments, assignment)
	}
	if err := bucketRows.Close(); err != nil {
		return TaskUndoState{}, fmt.Errorf("close Task undo Buckets: %w", err)
	}
	if err := bucketRows.Err(); err != nil {
		return TaskUndoState{}, fmt.Errorf("iterate Task undo Buckets: %w", err)
	}
	return state, nil
}

// RestoreTaskUndoState atomically verifies expected and replaces it with
// target. The optimistic comparison prevents undo from overwriting changes
// made outside Vimkunja after the Undo Entry was captured.
func (s *SQLiteStore) RestoreTaskUndoState(ctx context.Context, expected, target TaskUndoState) error {
	if expected.TaskID == 0 || expected.TaskID != target.TaskID {
		return fmt.Errorf("restore Task undo state: mismatched Task")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("restore Task undo state: begin transaction: %w", err)
	}
	defer tx.Rollback()

	current, err := loadTaskUndoState(ctx, tx, expected.TaskID)
	if err != nil {
		return err
	}
	if !sameTaskUndoState(current, expected) {
		return fmt.Errorf("restore Task undo state: Task changed since this action")
	}
	if err := rejectRecurringCompletion(ctx, tx, target.TaskID, target.Done); err != nil {
		return err
	}
	if err := validateUndoTarget(ctx, tx, target); err != nil {
		return err
	}

	dueDate, err := dueDateValue(target.DueDate)
	if err != nil {
		return fmt.Errorf("restore Task undo state: %w", err)
	}
	if target.DueDateStored != "" {
		storedDate, err := storedDueDate(target.DueDateStored)
		if err != nil {
			return fmt.Errorf("restore Task undo state: %w", err)
		}
		if storedDate != target.DueDate {
			return fmt.Errorf("restore Task undo state: stored due timestamp does not match due date")
		}
		// Keep the validated original encoding, including its offset and precision.
		dueDate = target.DueDateStored
	}
	index := int64(0)
	if current.ProjectID == target.ProjectID {
		if err := tx.QueryRowContext(ctx, `SELECT "index" FROM tasks WHERE id = ?`, target.TaskID).Scan(&index); err != nil {
			return fmt.Errorf("restore Task undo state: load Task index: %w", err)
		}
	} else {
		index, err = nextTaskIndex(ctx, tx, target.ProjectID)
		if err != nil {
			return fmt.Errorf("restore Task undo state: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM task_positions WHERE task_id = ?`, target.TaskID); err != nil {
			return fmt.Errorf("restore Task undo state: clear positions: %w", err)
		}
		if err := initializeUndoPositions(ctx, tx, target.TaskID, target.ProjectID); err != nil {
			return err
		}
	}
	// Completion metadata is outside history: preserve it unless Done transitions.
	doneAt := any(nil)
	if target.Done {
		doneAt = utcTimestampNow()
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks SET project_id = ?, title = ?, description = ?, priority = ?, due_date = ?,
			done = ?, done_at = CASE WHEN COALESCE(done, 0) = ? THEN done_at ELSE ? END, "index" = ?, updated = ? WHERE id = ?
	`, target.ProjectID, target.Title, target.Description, target.Priority, dueDate,
		boolToInt(target.Done), boolToInt(target.Done), doneAt, index, utcTimestampNow(), target.TaskID); err != nil {
		return fmt.Errorf("restore Task undo state: update Task: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM label_tasks WHERE task_id = ?`, target.TaskID); err != nil {
		return fmt.Errorf("restore Task undo state: clear Labels: %w", err)
	}
	now := utcTimestampNow()
	for _, labelID := range target.LabelIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO label_tasks (task_id, label_id, created) VALUES (?, ?, ?)`, target.TaskID, labelID, now); err != nil {
			return fmt.Errorf("restore Task undo state: assign Label: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_buckets WHERE task_id = ?`, target.TaskID); err != nil {
		return fmt.Errorf("restore Task undo state: clear Buckets: %w", err)
	}
	for _, assignment := range target.BucketAssignments {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (?, ?, ?)`, target.TaskID, assignment.BucketID, assignment.ViewID); err != nil {
			return fmt.Errorf("restore Task undo state: assign Bucket: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("restore Task undo state: commit: %w", err)
	}
	return nil
}

// storedDueDate accepts the timestamp encodings recognized by our SQLite
// driver. Keep the encoded calendar date, rather than converting it to UTC:
// Task reads use that same date for the editor's date-only presentation.
func storedDueDate(value string) (string, error) {
	goTime := strings.TrimSpace(value)
	if monotonic := strings.Index(goTime, "m="); monotonic > 0 {
		goTime = strings.TrimSpace(goTime[:monotonic])
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", goTime); err == nil {
		return parsed.Format("2006-01-02"), nil
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02T15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04", "2006-01-02T15:04", "2006-01-02",
	} {
		if parsed, err := time.Parse(layout, strings.TrimSuffix(value, "Z")); err == nil {
			return parsed.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("invalid stored due timestamp %q", value)
}

func sameTaskUndoState(left, right TaskUndoState) bool {
	// Identifier is presentation metadata derived from Project identifier and
	// Task index. Project moves may legitimately allocate a new index while
	// restoring all undo-relevant state.
	left.Identifier, right.Identifier = "", ""
	return reflect.DeepEqual(left, right)
}

func validateUndoTarget(ctx context.Context, tx *sql.Tx, target TaskUndoState) error {
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id = ?)`, target.ProjectID).Scan(&exists); err != nil {
		return fmt.Errorf("restore Task undo state: check Project: %w", err)
	}
	if !exists {
		return fmt.Errorf("restore Task undo state: Project %d no longer exists", target.ProjectID)
	}
	labels := append([]int64(nil), target.LabelIDs...)
	sort.Slice(labels, func(i, j int) bool { return labels[i] < labels[j] })
	for _, labelID := range labels {
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM labels WHERE id = ?)`, labelID).Scan(&exists); err != nil {
			return fmt.Errorf("restore Task undo state: check Label: %w", err)
		}
		if !exists {
			return fmt.Errorf("restore Task undo state: Label %d no longer exists", labelID)
		}
	}
	for _, assignment := range target.BucketAssignments {
		if assignment.BucketID == 0 {
			// Vikunja databases may retain an explicit zero assignment for a
			// View with no Bucket. Preserve that sentinel, but still reject an
			// assignment whose View disappeared or moved to another Project.
			if err := tx.QueryRowContext(ctx, `
				SELECT EXISTS(SELECT 1 FROM project_views WHERE id = ? AND project_id = ?)
			`, assignment.ViewID, target.ProjectID).Scan(&exists); err != nil {
				return fmt.Errorf("restore Task undo state: check unassigned View: %w", err)
			}
			if !exists {
				return fmt.Errorf("restore Task undo state: View %d is unavailable", assignment.ViewID)
			}
			continue
		}
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM buckets b JOIN project_views pv ON pv.id = b.project_view_id
				WHERE b.id = ? AND b.project_view_id = ? AND pv.project_id = ?
			)
		`, assignment.BucketID, assignment.ViewID, target.ProjectID).Scan(&exists); err != nil {
			return fmt.Errorf("restore Task undo state: check Bucket: %w", err)
		}
		if !exists {
			return fmt.Errorf("restore Task undo state: Bucket %d is unavailable", assignment.BucketID)
		}
	}
	return nil
}

func initializeUndoPositions(ctx context.Context, tx *sql.Tx, taskID, projectID int64) error {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM project_views WHERE project_id = ?`, projectID)
	if err != nil {
		return fmt.Errorf("restore Task undo state: list target Views: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var viewID int64
		if err := rows.Scan(&viewID); err != nil {
			return fmt.Errorf("restore Task undo state: scan target View: %w", err)
		}
		var position float64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), 0) + 1 FROM task_positions WHERE project_view_id = ?`, viewID).Scan(&position); err != nil {
			return fmt.Errorf("restore Task undo state: next position: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_positions (task_id, project_view_id, position) VALUES (?, ?, ?)`, taskID, viewID, position); err != nil {
			return fmt.Errorf("restore Task undo state: insert position: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("restore Task undo state: iterate target Views: %w", err)
	}
	return nil
}
