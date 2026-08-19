package vimkunja

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB

	closeOnce sync.Once
	closeErr  error
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

// Close safely checkpoints a WAL database before releasing its connection.
// It never removes WAL files itself: if another SQLite connection prevents a
// complete checkpoint, the WAL remains intact and the returned error reports
// that cleanup was incomplete rather than risking data loss.
func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	s.closeOnce.Do(func() {
		var busy, logFrames, checkpointedFrames int
		checkpointErr := s.db.QueryRow(`PRAGMA wal_checkpoint(TRUNCATE)`).Scan(&busy, &logFrames, &checkpointedFrames)
		if checkpointErr == nil && busy != 0 {
			checkpointErr = fmt.Errorf("checkpoint SQLite WAL: database is busy (%d log frames, %d checkpointed)", logFrames, checkpointedFrames)
		} else if checkpointErr != nil {
			checkpointErr = fmt.Errorf("checkpoint SQLite WAL: %w", checkpointErr)
		}

		s.closeErr = errors.Join(checkpointErr, s.db.Close())
	})
	return s.closeErr
}

const (
	viewKindList   = 0
	viewKindGantt  = 1
	viewKindTable  = 2
	viewKindKanban = 3

	bucketConfigurationNone   = 0
	bucketConfigurationManual = 1
	bucketConfigurationFilter = 2
)

func bucketConfigurationModeToString(v int) string {
	switch v {
	case bucketConfigurationManual:
		return "manual"
	case bucketConfigurationFilter:
		return "filter"
	default:
		return "none"
	}
}

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

func (s *SQLiteStore) ListProjects(ctx context.Context) ([]Project, error) {
	q := `SELECT id, identifier, title, description, parent_project_id, position
		FROM projects
		WHERE (is_archived = 0 OR is_archived IS NULL)
		ORDER BY id`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		var identifier, description sql.NullString
		var parentProjectID sql.NullInt64
		var position sql.NullFloat64
		if err := rows.Scan(&p.ID, &identifier, &p.Title, &description, &parentProjectID, &position); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		if identifier.Valid {
			p.Identifier = identifier.String
		}
		if description.Valid {
			p.Description = description.String
		}
		if parentProjectID.Valid {
			p.ParentProjectID = parentProjectID.Int64
		}
		if position.Valid {
			p.Position = position.Float64
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return out, nil
}

func (s *SQLiteStore) MoveProjectAmongSiblings(ctx context.Context, projectID int64, direction ProjectMoveDirection) error {
	if direction != ProjectMovePrevious && direction != ProjectMoveNext {
		return fmt.Errorf("move Project: invalid direction %d", direction)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("move Project: begin transaction: %w", err)
	}
	defer tx.Rollback()

	var parentProjectID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `
		SELECT parent_project_id
		FROM projects
		WHERE id = ? AND (is_archived = 0 OR is_archived IS NULL)
	`, projectID).Scan(&parentProjectID); err != nil {
		return fmt.Errorf("move Project: load Project: %w", err)
	}
	parentID := int64(0)
	if parentProjectID.Valid {
		parentID = parentProjectID.Int64
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, position
		FROM projects
		WHERE COALESCE(parent_project_id, 0) = ?
			AND (is_archived = 0 OR is_archived IS NULL)
		ORDER BY position ASC, id ASC
	`, parentID)
	if err != nil {
		return fmt.Errorf("move Project: list siblings: %w", err)
	}
	type sibling struct {
		id       int64
		position sql.NullFloat64
	}
	var siblings []sibling
	selected := -1
	for rows.Next() {
		var candidate sibling
		if err := rows.Scan(&candidate.id, &candidate.position); err != nil {
			rows.Close()
			return fmt.Errorf("move Project: scan sibling: %w", err)
		}
		if candidate.id == projectID {
			selected = len(siblings)
		}
		siblings = append(siblings, candidate)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("move Project: close siblings: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("move Project: iterate siblings: %w", err)
	}
	if selected < 0 {
		return fmt.Errorf("move Project: Project %d is not in its sibling group", projectID)
	}
	target := selected + int(direction)
	if target < 0 || target >= len(siblings) {
		return nil
	}

	positions := make([]float64, len(siblings))
	strictPositions := true
	for i, candidate := range siblings {
		if !candidate.position.Valid || (i > 0 && candidate.position.Float64 <= positions[i-1]) {
			strictPositions = false
		}
		positions[i] = candidate.position.Float64
	}
	if !strictPositions {
		for i := range positions {
			positions[i] = float64(i+1) * 65536
		}
	}
	siblings[selected], siblings[target] = siblings[target], siblings[selected]
	movedAt := time.Now()
	for i, candidate := range siblings {
		if _, err := tx.ExecContext(ctx, `UPDATE projects SET position = ?, updated = ? WHERE id = ?`, positions[i], movedAt, candidate.id); err != nil {
			return fmt.Errorf("move Project: persist sibling positions: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("move Project: commit: %w", err)
	}
	return nil
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

	q := `SELECT id, title, view_kind, bucket_configuration_mode FROM project_views
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
		var viewKindInt, bucketConfigurationMode int
		if err := rows.Scan(&v.ID, &v.Title, &viewKindInt, &bucketConfigurationMode); err != nil {
			return nil, 0, fmt.Errorf("scan view: %w", err)
		}
		v.ViewKind = viewKindToString(viewKindInt)
		v.BucketConfigurationMode = bucketConfigurationModeToString(bucketConfigurationMode)
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

// LoadSearchData returns every non-archived Project and its Tasks plus all
// existing Labels. It is intentionally unpaged: Global Search owns a complete
// locally browsable corpus rather than whichever Task page happens to be open.
func (s *SQLiteStore) LoadSearchData(ctx context.Context) (SearchData, error) {
	projects, err := s.ListProjects(ctx)
	if err != nil {
		return SearchData{}, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT
		t.id, t.project_id, p.identifier, p.title, t.title, t.done, t.due_date,
		t.priority, t."index", t.description
		FROM tasks t JOIN projects p ON p.id = t.project_id
		WHERE (p.is_archived = 0 OR p.is_archived IS NULL)
		ORDER BY t.id ASC`)
	if err != nil {
		return SearchData{}, fmt.Errorf("list search tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var projectIdentifier, projectTitle, description sql.NullString
		var dueDate sql.NullTime
		var index sql.NullInt64
		if err := rows.Scan(&task.ID, &task.ProjectID, &projectIdentifier, &projectTitle, &task.Title, &task.Done, &dueDate, &task.Priority, &index, &description); err != nil {
			return SearchData{}, fmt.Errorf("scan search task: %w", err)
		}
		if projectIdentifier.Valid {
			task.ProjectIdentifier = projectIdentifier.String
		}
		if projectTitle.Valid {
			task.ProjectTitle = projectTitle.String
		}
		if dueDate.Valid {
			task.DueDate = dueDate.Time.Format("2006-01-02")
		}
		if index.Valid {
			task.Index = index.Int64
		}
		if description.Valid {
			task.Description = description.String
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return SearchData{}, fmt.Errorf("iterate search tasks: %w", err)
	}

	labelRows, err := s.db.QueryContext(ctx, `SELECT id, title FROM labels ORDER BY id ASC`)
	if err != nil {
		return SearchData{}, fmt.Errorf("list Labels: %w", err)
	}
	defer labelRows.Close()
	var labels []Label
	for labelRows.Next() {
		var label Label
		if err := labelRows.Scan(&label.ID, &label.Title); err != nil {
			return SearchData{}, fmt.Errorf("scan Label: %w", err)
		}
		labels = append(labels, label)
	}
	if err := labelRows.Err(); err != nil {
		return SearchData{}, fmt.Errorf("iterate Labels: %w", err)
	}
	return SearchData{Projects: projects, Tasks: tasks, Labels: labels}, nil
}

// ListTasksForLabel returns non-archived Tasks associated with one existing
// Label, including Project metadata required by the Overview View.
func (s *SQLiteStore) ListTasksForLabel(ctx context.Context, labelID int64, page, perPage int) ([]Task, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 200
	}
	offset := (page - 1) * perPage

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1)
		FROM label_tasks lt
		JOIN tasks t ON t.id = lt.task_id
		JOIN projects p ON p.id = t.project_id
		WHERE lt.label_id = ? AND (p.is_archived = 0 OR p.is_archived IS NULL)`, labelID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count Label Tasks: %w", err)
	}
	totalPages := (total + perPage - 1) / perPage
	rows, err := s.db.QueryContext(ctx, `SELECT
		t.id, t.project_id, p.identifier, p.title, t.title, t.done, t.due_date,
		t.priority, t."index", t.description
		FROM label_tasks lt
		JOIN tasks t ON t.id = lt.task_id
		JOIN projects p ON p.id = t.project_id
		WHERE lt.label_id = ? AND (p.is_archived = 0 OR p.is_archived IS NULL)
		ORDER BY t.due_date IS NULL ASC, t.due_date ASC, t.priority DESC,
			p.identifier COLLATE NOCASE ASC, t.title COLLATE NOCASE ASC, t.id ASC
		LIMIT ? OFFSET ?`, labelID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list Label Tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var projectIdentifier, projectTitle, description sql.NullString
		var dueDate sql.NullTime
		var index sql.NullInt64
		if err := rows.Scan(&task.ID, &task.ProjectID, &projectIdentifier, &projectTitle, &task.Title, &task.Done, &dueDate, &task.Priority, &index, &description); err != nil {
			return nil, 0, fmt.Errorf("scan Label Task: %w", err)
		}
		if projectIdentifier.Valid {
			task.ProjectIdentifier = projectIdentifier.String
		}
		if projectTitle.Valid {
			task.ProjectTitle = projectTitle.String
		}
		if dueDate.Valid {
			task.DueDate = dueDate.Time.Format("2006-01-02")
		}
		if index.Valid {
			task.Index = index.Int64
		}
		if description.Valid {
			task.Description = description.String
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate Label Tasks: %w", err)
	}
	return tasks, totalPages, nil
}

// ListLabels returns every global Label in alphabetical order. TaskCount includes
// associations on archived Tasks because deleting a Label removes all of them.
func (s *SQLiteStore) ListLabels(ctx context.Context) ([]Label, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT l.id, l.title, COUNT(lt.task_id)
		FROM labels l
		LEFT JOIN label_tasks lt ON lt.label_id = l.id
		GROUP BY l.id, l.title
		ORDER BY l.title COLLATE NOCASE ASC, l.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list Labels: %w", err)
	}
	defer rows.Close()

	var labels []Label
	for rows.Next() {
		var label Label
		if err := rows.Scan(&label.ID, &label.Title, &label.TaskCount); err != nil {
			return nil, fmt.Errorf("scan Label: %w", err)
		}
		labels = append(labels, label)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Labels: %w", err)
	}
	return labels, nil
}

// ListTaskLabels returns a Task's assigned Labels in the same stable order
// used by the Task Detail field and picker.
func (s *SQLiteStore) ListTaskLabels(ctx context.Context, taskID int64) ([]Label, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT l.id, l.title
		FROM label_tasks lt JOIN labels l ON l.id = lt.label_id
		WHERE lt.task_id = ?
		ORDER BY l.title COLLATE NOCASE ASC, l.id ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list Task Labels: %w", err)
	}
	defer rows.Close()

	var labels []Label
	for rows.Next() {
		var label Label
		if err := rows.Scan(&label.ID, &label.Title); err != nil {
			return nil, fmt.Errorf("scan Task Label: %w", err)
		}
		labels = append(labels, label)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Task Labels: %w", err)
	}
	return labels, nil
}

// CreateLabel persists one new Label for the active actor. It deliberately
// rejects duplicate normalized titles instead of implicitly merging Labels.
func (s *SQLiteStore) CreateLabel(ctx context.Context, title string) (Label, error) {
	title, err := validLabelTitle(title)
	if err != nil {
		return Label{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Label{}, fmt.Errorf("begin create Label transaction: %w", err)
	}
	defer tx.Rollback()
	createdByID, err := onlyLocalUser(ctx, tx)
	if err != nil {
		return Label{}, err
	}
	if err := ensureUniqueLabelTitle(ctx, tx, 0, title); err != nil {
		return Label{}, err
	}
	now := time.Now()
	result, err := tx.ExecContext(ctx, `INSERT INTO labels (title, created_by_id, created, updated)
		VALUES (?, ?, ?, ?)`, title, createdByID, now, now)
	if err != nil {
		return Label{}, fmt.Errorf("insert Label: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Label{}, fmt.Errorf("get created Label id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Label{}, fmt.Errorf("commit create Label: %w", err)
	}
	return Label{ID: id, Title: title}, nil
}

// RenameLabel changes a Label title without changing its identity or Task
// associations.
func (s *SQLiteStore) RenameLabel(ctx context.Context, labelID int64, title string) error {
	title, err := validLabelTitle(title)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin rename Label transaction: %w", err)
	}
	defer tx.Rollback()
	if err := ensureUniqueLabelTitle(ctx, tx, labelID, title); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE labels SET title = ?, updated = ? WHERE id = ?`, title, time.Now(), labelID)
	if err != nil {
		return fmt.Errorf("rename Label: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("confirm Label rename: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("Label %d not found", labelID)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Label rename: %w", err)
	}
	return nil
}

// DeleteLabel removes one global Label and all of its Task associations, never
// the Tasks themselves.
func (s *SQLiteStore) DeleteLabel(ctx context.Context, labelID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete Label transaction: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM label_tasks WHERE label_id = ?`, labelID); err != nil {
		return fmt.Errorf("delete Label associations: %w", err)
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM labels WHERE id = ?`, labelID)
	if err != nil {
		return fmt.Errorf("delete Label: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("confirm Label deletion: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("Label %d not found", labelID)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Label deletion: %w", err)
	}
	return nil
}

// ReplaceTaskLabels atomically replaces every Label association for one Task.
func (s *SQLiteStore) ReplaceTaskLabels(ctx context.Context, taskID int64, labelIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace Task Labels transaction: %w", err)
	}
	defer tx.Rollback()
	if err := replaceTaskLabels(ctx, tx, taskID, labelIDs); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace Task Labels: %w", err)
	}
	return nil
}

func onlyLocalUser(ctx context.Context, tx *sql.Tx) (int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM users ORDER BY id ASC LIMIT 2`)
	if err != nil {
		return 0, fmt.Errorf("find local Vikunja user: %w", err)
	}
	defer rows.Close()
	var userIDs []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return 0, fmt.Errorf("scan local Vikunja user: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate local Vikunja users: %w", err)
	}
	if len(userIDs) != 1 {
		return 0, fmt.Errorf("create Label: expected exactly one local Vikunja user, found %d", len(userIDs))
	}
	return userIDs[0], nil
}

func validLabelTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("Label title is required")
	}
	return title, nil
}

func normalizedLabelTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

func ensureUniqueLabelTitle(ctx context.Context, tx *sql.Tx, exceptID int64, title string) error {
	var existingID int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM labels WHERE LOWER(TRIM(title)) = ? AND id != ? LIMIT 1`, normalizedLabelTitle(title), exceptID).Scan(&existingID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check duplicate Label title: %w", err)
	}
	return fmt.Errorf("Label title already exists")
}

func replaceTaskLabels(ctx context.Context, tx *sql.Tx, taskID int64, labelIDs []int64) error {
	if taskID == 0 {
		return fmt.Errorf("replace Task Labels: missing task id")
	}
	var taskExists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM tasks WHERE id = ?`, taskID).Scan(&taskExists); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("Task %d not found", taskID)
		}
		return fmt.Errorf("load Task for Label replacement: %w", err)
	}
	seen := make(map[int64]struct{}, len(labelIDs))
	unique := make([]int64, 0, len(labelIDs))
	for _, labelID := range labelIDs {
		if labelID == 0 {
			return fmt.Errorf("replace Task Labels: missing Label id")
		}
		if _, ok := seen[labelID]; ok {
			continue
		}
		seen[labelID] = struct{}{}
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM labels WHERE id = ?`, labelID).Scan(&exists); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("Label %d no longer exists", labelID)
			}
			return fmt.Errorf("validate Task Label: %w", err)
		}
		unique = append(unique, labelID)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM label_tasks WHERE task_id = ?`, taskID); err != nil {
		return fmt.Errorf("remove Task Labels: %w", err)
	}
	now := time.Now()
	for _, labelID := range unique {
		if _, err := tx.ExecContext(ctx, `INSERT INTO label_tasks (task_id, label_id, created) VALUES (?, ?, ?)`, taskID, labelID, now); err != nil {
			return fmt.Errorf("assign Task Label: %w", err)
		}
	}
	return nil
}

// ProvisionStandardManualKanbanView creates Vimkunja's upstream-compatible
// manual Kanban template. Existing completed Tasks deliberately start in Done;
// open Tasks start in To-Do so the global Done state remains consistent.
func (s *SQLiteStore) ProvisionStandardManualKanbanView(ctx context.Context, projectID int64) (KanbanBoard, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return KanbanBoard{}, fmt.Errorf("begin provision kanban view: %w", err)
	}
	defer tx.Rollback()

	var ownerID int64
	if err := tx.QueryRowContext(ctx, `SELECT owner_id FROM projects WHERE id = ?`, projectID).Scan(&ownerID); err != nil {
		if err == sql.ErrNoRows {
			return KanbanBoard{}, fmt.Errorf("project %d not found", projectID)
		}
		return KanbanBoard{}, fmt.Errorf("get project owner for kanban view: %w", err)
	}
	var manualViewCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM project_views WHERE project_id = ? AND view_kind = ? AND bucket_configuration_mode = ?`, projectID, viewKindKanban, bucketConfigurationManual).Scan(&manualViewCount); err != nil {
		return KanbanBoard{}, fmt.Errorf("count manual kanban views: %w", err)
	}
	if manualViewCount != 0 {
		return KanbanBoard{}, fmt.Errorf("project already has %d manual kanban views", manualViewCount)
	}

	now := time.Now()
	result, err := tx.ExecContext(ctx, `
		INSERT INTO project_views (project_id, title, view_kind, position, bucket_configuration_mode, created, updated)
		VALUES (?, 'Kanban', ?, 400, ?, ?, ?)
	`, projectID, viewKindKanban, bucketConfigurationManual, now, now)
	if err != nil {
		return KanbanBoard{}, fmt.Errorf("create manual kanban view: %w", err)
	}
	viewID, err := result.LastInsertId()
	if err != nil {
		return KanbanBoard{}, fmt.Errorf("get manual kanban view id: %w", err)
	}

	createBucket := func(title string, position int) (int64, error) {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO buckets (title, project_view_id, position, created, updated, created_by_id)
			VALUES (?, ?, ?, ?, ?, ?)
		`, title, viewID, position, now, now, ownerID)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	}
	toDoBucketID, err := createBucket("To-Do", 100)
	if err != nil {
		return KanbanBoard{}, fmt.Errorf("create To-Do bucket: %w", err)
	}
	if _, err := createBucket("Doing", 200); err != nil {
		return KanbanBoard{}, fmt.Errorf("create Doing bucket: %w", err)
	}
	doneBucketID, err := createBucket("Done", 300)
	if err != nil {
		return KanbanBoard{}, fmt.Errorf("create Done bucket: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE project_views SET default_bucket_id = ?, done_bucket_id = ? WHERE id = ?`, toDoBucketID, doneBucketID, viewID); err != nil {
		return KanbanBoard{}, fmt.Errorf("configure manual kanban view: %w", err)
	}

	taskRows, err := tx.QueryContext(ctx, `SELECT id, done FROM tasks WHERE project_id = ? ORDER BY id ASC`, projectID)
	if err != nil {
		return KanbanBoard{}, fmt.Errorf("list tasks for kanban view: %w", err)
	}
	type taskDoneState struct {
		id   int64
		done bool
	}
	var tasks []taskDoneState
	for taskRows.Next() {
		var task taskDoneState
		if err := taskRows.Scan(&task.id, &task.done); err != nil {
			return KanbanBoard{}, fmt.Errorf("scan task for kanban view: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := taskRows.Err(); err != nil {
		_ = taskRows.Close()
		return KanbanBoard{}, fmt.Errorf("iterate tasks for kanban view: %w", err)
	}
	if err := taskRows.Close(); err != nil {
		return KanbanBoard{}, fmt.Errorf("close tasks for kanban view: %w", err)
	}
	for i, task := range tasks {
		bucketID := toDoBucketID
		if task.done {
			bucketID = doneBucketID
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (?, ?, ?)`, task.id, bucketID, viewID); err != nil {
			return KanbanBoard{}, fmt.Errorf("assign task to provisioned bucket: %w", err)
		}
		position := math.Pow(2, 32) / float64(len(tasks)) * float64(i+1)
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_positions (task_id, project_view_id, position) VALUES (?, ?, ?)`, task.id, viewID, position); err != nil {
			return KanbanBoard{}, fmt.Errorf("set provisioned task position: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return KanbanBoard{}, fmt.Errorf("commit provision kanban view: %w", err)
	}
	board, err := s.LoadManualKanbanBoard(ctx, projectID)
	if err != nil {
		return KanbanBoard{}, fmt.Errorf("load provisioned kanban board: %w", err)
	}
	return board, nil
}

// LoadManualKanbanBoard returns the one manual Kanban View for a Project,
// including empty Buckets and Tasks in their persisted View order.
func (s *SQLiteStore) LoadManualKanbanBoard(ctx context.Context, projectID int64) (KanbanBoard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title
		FROM project_views
		WHERE project_id = ? AND view_kind = ? AND bucket_configuration_mode = ?
		ORDER BY position ASC, id ASC
	`, projectID, viewKindKanban, bucketConfigurationManual)
	if err != nil {
		return KanbanBoard{}, fmt.Errorf("list manual kanban views: %w", err)
	}
	defer rows.Close()

	var views []ProjectView
	for rows.Next() {
		var view ProjectView
		if err := rows.Scan(&view.ID, &view.Title); err != nil {
			return KanbanBoard{}, fmt.Errorf("scan manual kanban view: %w", err)
		}
		view.ViewKind = "kanban"
		view.BucketConfigurationMode = "manual"
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return KanbanBoard{}, fmt.Errorf("iterate manual kanban views: %w", err)
	}
	switch len(views) {
	case 0:
		return KanbanBoard{Availability: KanbanBoardUnavailable}, nil
	case 1:
	default:
		return KanbanBoard{Availability: KanbanBoardAmbiguous}, nil
	}

	view := views[0]
	bucketRows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.title, b.position,
			t.id, COALESCE(t.project_id, 0), COALESCE(t.title, ''), COALESCE(t.done, 0), t.due_date, COALESCE(t.priority, 0), t."index", t.description
		FROM buckets b
		LEFT JOIN task_buckets tb ON tb.bucket_id = b.id AND tb.project_view_id = b.project_view_id
		LEFT JOIN tasks t ON t.id = tb.task_id
		LEFT JOIN task_positions tp ON tp.task_id = t.id AND tp.project_view_id = b.project_view_id
		WHERE b.project_view_id = ?
		ORDER BY b.position ASC, b.id ASC, tp.position ASC, t.id ASC
	`, view.ID)
	if err != nil {
		return KanbanBoard{}, fmt.Errorf("list kanban buckets: %w", err)
	}
	defer bucketRows.Close()

	board := KanbanBoard{Availability: KanbanBoardAvailable, View: view}
	bucketIndexes := make(map[int64]int)
	for bucketRows.Next() {
		var bucketID int64
		var bucketTitle string
		var bucketPosition float64
		var taskID sql.NullInt64
		var task Task
		var dueDate sql.NullTime
		var index sql.NullInt64
		var description sql.NullString
		if err := bucketRows.Scan(&bucketID, &bucketTitle, &bucketPosition,
			&taskID, &task.ProjectID, &task.Title, &task.Done, &dueDate, &task.Priority, &index, &description); err != nil {
			return KanbanBoard{}, fmt.Errorf("scan kanban bucket: %w", err)
		}
		bucketIndex, ok := bucketIndexes[bucketID]
		if !ok {
			bucketIndex = len(board.Buckets)
			bucketIndexes[bucketID] = bucketIndex
			board.Buckets = append(board.Buckets, KanbanBucket{ID: bucketID, Title: bucketTitle, Position: bucketPosition})
		}
		if !taskID.Valid {
			continue
		}
		task.ID = taskID.Int64
		if dueDate.Valid {
			task.DueDate = dueDate.Time.Format("2006-01-02")
		}
		if index.Valid {
			task.Index = index.Int64
		}
		if description.Valid {
			task.Description = description.String
		}
		board.Buckets[bucketIndex].Tasks = append(board.Buckets[bucketIndex].Tasks, task)
	}
	if err := bucketRows.Err(); err != nil {
		return KanbanBoard{}, fmt.Errorf("iterate kanban buckets: %w", err)
	}
	return board, nil
}

// GetTaskWorkflowStatus returns a Task's Bucket assignment and available
// Buckets when the Project has exactly one manual Kanban View. With no manual
// Kanban View the status is unavailable; with several it is ambiguous.
func (s *SQLiteStore) GetTaskWorkflowStatus(ctx context.Context, projectID, taskID int64) (WorkflowStatus, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id
		FROM project_views
		WHERE project_id = ? AND view_kind = ? AND bucket_configuration_mode = ?
		ORDER BY position ASC, id ASC
	`, projectID, viewKindKanban, bucketConfigurationManual)
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
		WHERE project_id = ? AND view_kind = ? AND bucket_configuration_mode = ?
		ORDER BY position ASC, id ASC
	`, projectID, viewKindKanban, bucketConfigurationManual)
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
			AND pv.view_kind = ?
			AND pv.bucket_configuration_mode = ?
			AND CASE WHEN ? THEN pv.done_bucket_id ELSE pv.default_bucket_id END IS NOT NULL
	`, boolToInt(done), taskID, viewKindKanban, bucketConfigurationManual, boolToInt(done))
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

	if err := setTaskDoneState(ctx, tx, taskID, done); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit toggle task transaction: %w", err)
	}
	return nil
}

// setTaskDoneState changes the global Task Done state and its configured
// workflow buckets within an existing transaction.
func setTaskDoneState(ctx context.Context, tx *sql.Tx, taskID int64, done bool) error {
	var doneAt any
	if done {
		doneAt = time.Now()
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET done = ?, done_at = ?, updated = ? WHERE id = ?`, boolToInt(done), doneAt, time.Now(), taskID); err != nil {
		return fmt.Errorf("set task done state: %w", err)
	}
	if err := syncTaskBucketsForDoneState(ctx, tx, taskID, done); err != nil {
		return err
	}
	return nil
}

// CreateTask persists a new task and creates the default per-view position and
// bucket assignments expected by Vikunja's SQLite schema.
// DeleteTask permanently removes a Task and its Vikunja-owned relations in one
// transaction. Attachment files themselves are retained: removing their link
// cannot safely establish that no other record references the file.
func (s *SQLiteStore) DeleteTask(ctx context.Context, taskID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete task: %w", err)
	}
	defer tx.Rollback()

	deletes := []struct {
		table string
		query string
	}{
		{"task_positions", "DELETE FROM task_positions WHERE task_id = ?"},
		{"task_buckets", "DELETE FROM task_buckets WHERE task_id = ?"},
		{"task_assignees", "DELETE FROM task_assignees WHERE task_id = ?"},
		{"label_tasks", "DELETE FROM label_tasks WHERE task_id = ?"},
		{"task_reminders", "DELETE FROM task_reminders WHERE task_id = ?"},
		{"task_relations", "DELETE FROM task_relations WHERE task_id = ? OR other_task_id = ?"},
		{"task_attachments", "DELETE FROM task_attachments WHERE task_id = ?"},
		{"task_comments", "DELETE FROM task_comments WHERE task_id = ?"},
		{"task_unread_statuses", "DELETE FROM task_unread_statuses WHERE task_id = ?"},
	}
	for _, delete := range deletes {
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?", delete.table).Scan(&exists); err != nil {
			return fmt.Errorf("check %s for task deletion: %w", delete.table, err)
		}
		if exists == 0 {
			continue
		}
		args := []any{taskID}
		if delete.table == "task_relations" {
			args = append(args, taskID)
		}
		if _, err := tx.ExecContext(ctx, delete.query, args...); err != nil {
			return fmt.Errorf("delete task relation from %s: %w", delete.table, err)
		}
	}

	result, err := tx.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("confirm task deletion: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("task %d not found", taskID)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit task deletion: %w", err)
	}
	return nil
}

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
	if draft.LabelIDsSet {
		if err := replaceTaskLabels(ctx, tx, id, draft.LabelIDs); err != nil {
			return Task{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("commit create task: %w", err)
	}
	return Task{ID: id, ProjectID: draft.ProjectID, Title: draft.Title, Description: draft.Description, Priority: draft.Priority, DueDate: draft.DueDate, Index: index}, nil
}

// UpdateTask changes all editable fields. Moving a task also replaces its
// per-view assignments with the target project's defaults.
// UpdateTaskTitle changes only a Task title, leaving every other Task field
// untouched. It is used by inline list editing, whose read model may be stale.
func (s *SQLiteStore) UpdateTaskTitle(ctx context.Context, taskID int64, title string) error {
	if taskID == 0 {
		return fmt.Errorf("update task title: missing task id")
	}
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title is required")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE tasks SET title = ?, updated = ? WHERE id = ?`, title, time.Now(), taskID)
	if err != nil {
		return fmt.Errorf("update task title: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update task title rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("update task title: task not found")
	}
	return nil
}

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
	// Done is applied last so its configured Done/Default Bucket rule wins
	// over a simultaneously selected Workflow Status.
	if draft.DoneSet {
		if err := setTaskDoneState(ctx, tx, draft.ID, draft.Done); err != nil {
			return Task{}, err
		}
		done = draft.Done
	}
	if draft.LabelIDsSet {
		if err := replaceTaskLabels(ctx, tx, draft.ID, draft.LabelIDs); err != nil {
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
