package vimkunja

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const databaseTimestampLayout = "2006-01-02 15:04:05"

func utcTimestampNow() string {
	return time.Now().UTC().Format(databaseTimestampLayout)
}

func parseDatabaseTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if monotonic := strings.Index(value, " m="); monotonic >= 0 {
		value = value[:monotonic]
	}
	layouts := []string{
		databaseTimestampLayout,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported database timestamp %q", value)
}

type SQLiteStore struct {
	db *sql.DB

	closeOnce sync.Once
	closeErr  error
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("open existing Vikunja database %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("open existing Vikunja database %q: expected a regular file", path)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	// mode=rw also prevents SQLite from creating a file if it disappears after Stat.
	uriPath := filepath.ToSlash(absolutePath)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath // Windows drive paths need file:///C:/... .
	}
	uri := url.URL{Scheme: "file", Path: uriPath, RawQuery: "mode=rw"}
	db, err := sql.Open("sqlite", uri.String())
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
	movedAt := utcTimestampNow()
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
	if err := s.populateTaskWorkflow(ctx, out); err != nil {
		return nil, 0, err
	}

	return out, totalPages, nil
}

// populateTaskWorkflow enriches task list read models with the status from a
// project's only manual Kanban View. Multiple or missing manual Views remain
// unavailable by design, matching GetTaskWorkflowStatus.
func (s *SQLiteStore) populateTaskWorkflow(ctx context.Context, tasks []Task) error {
	if len(tasks) == 0 {
		return nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(tasks)), ",")
	args := make([]any, 0, len(tasks)+2)
	args = append(args, viewKindKanban, bucketConfigurationManual)
	for _, task := range tasks {
		args = append(args, task.ID)
	}
	query := fmt.Sprintf(`
		SELECT t.id, wf.view_id, wf.default_bucket_id, wf.done_bucket_id,
			(SELECT tb.bucket_id
			 FROM task_buckets tb
			 WHERE tb.task_id = t.id AND tb.project_view_id = wf.view_id
			 LIMIT 1)
		FROM tasks t
		LEFT JOIN (
			SELECT project_id, MIN(id) AS view_id,
				MIN(default_bucket_id) AS default_bucket_id,
				MIN(done_bucket_id) AS done_bucket_id
			FROM project_views
			WHERE view_kind = ? AND bucket_configuration_mode = ?
			GROUP BY project_id
			HAVING COUNT(1) = 1
		) wf ON wf.project_id = t.project_id
		WHERE t.id IN (%s)`, placeholders)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("load task workflow status: %w", err)
	}
	defer rows.Close()

	byID := make(map[int64]*Task, len(tasks))
	for i := range tasks {
		byID[tasks[i].ID] = &tasks[i]
	}
	unconfigured := make(map[int64][]*Task)
	for rows.Next() {
		var id int64
		var viewID, defaultBucketID, doneBucketID, currentBucketID sql.NullInt64
		if err := rows.Scan(&id, &viewID, &defaultBucketID, &doneBucketID, &currentBucketID); err != nil {
			return fmt.Errorf("scan task workflow status: %w", err)
		}
		task := byID[id]
		if task == nil || !viewID.Valid {
			continue
		}
		task.Workflow.Available = true
		if defaultBucketID.Valid && defaultBucketID.Int64 > 0 {
			task.Workflow.DefaultBucketID = defaultBucketID.Int64
		} else if viewID.Valid {
			unconfigured[viewID.Int64] = append(unconfigured[viewID.Int64], task)
		}
		if doneBucketID.Valid {
			task.Workflow.DoneBucketID = doneBucketID.Int64
		}
		if currentBucketID.Valid {
			task.Workflow.CurrentBucketID = currentBucketID.Int64
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate task workflow status: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close task workflow status: %w", err)
	}
	for viewID, affected := range unconfigured {
		id, err := resolveDefaultBucket(ctx, s.db, viewID, sql.NullInt64{})
		if err != nil {
			return err
		}
		for _, task := range affected {
			task.Workflow.DefaultBucketID = id
		}
	}
	return nil
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
	if err := s.populateTaskWorkflow(ctx, out); err != nil {
		return nil, 0, err
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
	if err := s.populateTaskWorkflow(ctx, tasks); err != nil {
		return nil, 0, err
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
	now := utcTimestampNow()
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
	result, err := tx.ExecContext(ctx, `UPDATE labels SET title = ?, updated = ? WHERE id = ?`, title, utcTimestampNow(), labelID)
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
		return 0, fmt.Errorf("expected exactly one local Vikunja user, found %d", len(userIDs))
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
	now := utcTimestampNow()
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

	now := utcTimestampNow()
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

// LoadManualKanbanBoard returns a manual Kanban View for a Project,
// including empty Buckets and Tasks in their persisted View order. Without a
// view ID it retains the legacy single-view/ambiguous behavior.
func (s *SQLiteStore) LoadManualKanbanBoard(ctx context.Context, projectID int64, requestedViewID ...int64) (KanbanBoard, error) {
	query := `
		SELECT id, title, default_bucket_id, done_bucket_id
		FROM project_views
		WHERE project_id = ? AND view_kind = ? AND bucket_configuration_mode = ?`
	args := []any{projectID, viewKindKanban, bucketConfigurationManual}
	if len(requestedViewID) > 0 && requestedViewID[0] != 0 {
		query += ` AND id = ?`
		args = append(args, requestedViewID[0])
	}
	query += ` ORDER BY position ASC, id ASC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return KanbanBoard{}, fmt.Errorf("list manual kanban views: %w", err)
	}
	defer rows.Close()

	var views []ProjectView
	for rows.Next() {
		var view ProjectView
		var defaultBucketID, doneBucketID sql.NullInt64
		if err := rows.Scan(&view.ID, &view.Title, &defaultBucketID, &doneBucketID); err != nil {
			return KanbanBoard{}, fmt.Errorf("scan manual kanban view: %w", err)
		}
		view.ViewKind = "kanban"
		view.BucketConfigurationMode = "manual"
		if defaultBucketID.Valid {
			view.DefaultBucketID = defaultBucketID.Int64
		}
		if doneBucketID.Valid {
			view.DoneBucketID = doneBucketID.Int64
		}
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
	if err := rows.Close(); err != nil {
		return KanbanBoard{}, fmt.Errorf("close manual kanban views: %w", err)
	}
	view.DefaultBucketID, err = resolveDefaultBucket(ctx, s.db, view.ID, sql.NullInt64{Int64: view.DefaultBucketID, Valid: view.DefaultBucketID > 0})
	if err != nil {
		return KanbanBoard{}, err
	}
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
// Buckets for the selected manual Kanban View. Without a View ID, Projects
// with no manual View are unavailable and Projects with several are ambiguous.
func (s *SQLiteStore) GetTaskWorkflowStatus(ctx context.Context, projectID, taskID int64, requestedViewID ...int64) (WorkflowStatus, error) {
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
	if len(requestedViewID) > 0 && requestedViewID[0] != 0 {
		found := false
		for _, id := range viewIDs {
			if id == requestedViewID[0] {
				found = true
				break
			}
		}
		if !found {
			return WorkflowStatus{Availability: WorkflowStatusUnavailable}, nil
		}
		viewIDs = []int64{requestedViewID[0]}
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
		SELECT b.id, b.title,
			tb.task_id IS NOT NULL OR (? = 0 AND b.id = CASE WHEN pv.default_bucket_id > 0 THEN pv.default_bucket_id ELSE
				(SELECT fb.id FROM buckets fb WHERE fb.project_view_id = pv.id AND fb.id > 0 ORDER BY fb.position ASC, fb.id ASC LIMIT 1) END)
		FROM buckets b
		JOIN project_views pv ON pv.id = b.project_view_id
		LEFT JOIN task_buckets tb
			ON tb.bucket_id = b.id
			AND tb.project_view_id = b.project_view_id
			AND tb.task_id = ?
		WHERE b.project_view_id = ?
		ORDER BY b.position ASC, b.id ASC
	`, taskID, taskID, viewIDs[0])
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

// SetTaskWorkflowStatus assigns a Task to a Bucket in the selected manual
// Kanban View and atomically synchronizes its global Done state. Without a View
// ID it retains the legacy requirement for exactly one manual View.
func (s *SQLiteStore) SetTaskWorkflowStatus(ctx context.Context, taskID, bucketID int64, requestedViewID ...int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin set workflow status transaction: %w", err)
	}
	defer tx.Rollback()

	var projectID int64
	if err := tx.QueryRowContext(ctx, `SELECT project_id FROM tasks WHERE id = ?`, taskID).Scan(&projectID); err != nil {
		return fmt.Errorf("get task project for workflow status: %w", err)
	}
	if err := setTaskWorkflowStatus(ctx, tx, taskID, projectID, bucketID, requestedViewID...); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit set workflow status: %w", err)
	}
	return nil
}

func setTaskWorkflowStatus(ctx context.Context, tx *sql.Tx, taskID, projectID, bucketID int64, requestedViewID ...int64) error {
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
	if len(requestedViewID) > 0 && requestedViewID[0] != 0 {
		found := false
		for _, candidate := range views {
			if candidate.id == requestedViewID[0] {
				views = []workflowView{candidate}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("set workflow status: view does not belong to the task project")
		}
	} else if len(views) > 1 {
		// A bucket ID is globally unique, so a status selected in a multi-View
		// editor identifies its owning View even when the caller predates the
		// explicit View-scoped API (notably task creation/update drafts).
		var bucketViewID int64
		err := tx.QueryRowContext(ctx, `SELECT project_view_id FROM buckets WHERE id = ?`, bucketID).Scan(&bucketViewID)
		if err == nil {
			for _, candidate := range views {
				if candidate.id == bucketViewID {
					views = []workflowView{candidate}
					break
				}
			}
		}
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
	if err := rejectRecurringCompletion(ctx, tx, taskID, done); err != nil {
		return err
	}
	var doneAt any
	if done {
		doneAt = utcTimestampNow()
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET done = ?, done_at = CASE WHEN COALESCE(done, 0) = ? THEN done_at ELSE ? END, updated = ? WHERE id = ?`, boolToInt(done), boolToInt(done), doneAt, utcTimestampNow(), taskID); err != nil {
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

func syncTaskBucketsForDoneState(ctx context.Context, tx *sql.Tx, taskID int64, done bool, selectedViewID ...int64) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT pv.id,
			CASE WHEN ? THEN pv.done_bucket_id ELSE pv.default_bucket_id END
		FROM project_views pv
		JOIN tasks t ON t.project_id = pv.project_id
		WHERE t.id = ?
			AND pv.view_kind = ?
			AND pv.bucket_configuration_mode = ?
	`, boolToInt(done), taskID, viewKindKanban, bucketConfigurationManual)
	if err != nil {
		return fmt.Errorf("list configured workflow buckets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var viewID int64
		var configured sql.NullInt64
		if err := rows.Scan(&viewID, &configured); err != nil {
			return fmt.Errorf("scan configured workflow bucket: %w", err)
		}
		if len(selectedViewID) > 0 && selectedViewID[0] != 0 && selectedViewID[0] != viewID {
			continue
		}
		bucketID := int64(0)
		if done {
			// No configured Done Bucket means completion does not move the card.
			if !configured.Valid || configured.Int64 <= 0 {
				continue
			}
			bucketID = configured.Int64
		} else {
			var err error
			bucketID, err = resolveDefaultBucket(ctx, tx, viewID, configured)
			if err != nil {
				return err
			}
		}
		if bucketID == 0 {
			continue
		}
		if err := validateViewBucket(ctx, tx, viewID, bucketID); err != nil {
			return err
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

func (s *SQLiteStore) ToggleTaskDone(ctx context.Context, taskID int64, done bool, viewID ...int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin toggle task transaction: %w", err)
	}
	defer tx.Rollback()

	if err := setTaskDoneState(ctx, tx, taskID, done, viewID...); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit toggle task transaction: %w", err)
	}
	return nil
}

// setTaskDoneState changes the global Task Done state and its configured
// workflow buckets within an existing transaction. Completion metadata changes
// only when Done transitions; the SQL preserves its exact stored encoding.
func setTaskDoneState(ctx context.Context, tx *sql.Tx, taskID int64, done bool, viewID ...int64) error {
	if err := rejectRecurringCompletion(ctx, tx, taskID, done); err != nil {
		return err
	}
	if len(viewID) > 0 && viewID[0] != 0 {
		var valid bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM project_views pv
				JOIN tasks t ON t.project_id = pv.project_id
				WHERE t.id = ? AND pv.id = ?
					AND pv.view_kind = ? AND pv.bucket_configuration_mode = ?
			)
		`, taskID, viewID[0], viewKindKanban, bucketConfigurationManual).Scan(&valid); err != nil {
			return fmt.Errorf("validate task workflow view: %w", err)
		}
		if !valid {
			return fmt.Errorf("toggle task: view does not belong to the task project")
		}
	}
	var doneAt any
	if done {
		doneAt = utcTimestampNow()
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET done = ?, done_at = CASE WHEN COALESCE(done, 0) = ? THEN done_at ELSE ? END, updated = ? WHERE id = ?`, boolToInt(done), boolToInt(done), doneAt, utcTimestampNow(), taskID); err != nil {
		return fmt.Errorf("set task done state: %w", err)
	}
	if err := syncTaskBucketsForDoneState(ctx, tx, taskID, done, viewID...); err != nil {
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

	if err := deleteTaskPolymorphicRelations(ctx, tx, taskID); err != nil {
		return err
	}

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

func (s *SQLiteStore) ListTaskComments(ctx context.Context, taskID int64) ([]TaskComment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT tc.id, tc.task_id, tc.comment,
			COALESCE(NULLIF(TRIM(u.name), ''), u.username, 'Unknown user'),
			CASE WHEN (SELECT COUNT(*) FROM users) = 1
				AND tc.author_id = (SELECT id FROM users ORDER BY id LIMIT 1)
				THEN 1 ELSE 0 END,
			CAST(tc.created AS TEXT), COALESCE(CAST(tc.updated AS TEXT), CAST(tc.created AS TEXT))
		FROM task_comments tc
		LEFT JOIN users u ON u.id = tc.author_id
		WHERE tc.task_id = ?
		ORDER BY tc.created ASC, tc.id ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list Task Comments: %w", err)
	}
	defer rows.Close()
	comments := make([]TaskComment, 0)
	for rows.Next() {
		var comment TaskComment
		var created, updated string
		if err := rows.Scan(&comment.ID, &comment.TaskID, &comment.HTML, &comment.Author, &comment.CanDelete, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan Task Comment: %w", err)
		}
		comment.Created, err = parseDatabaseTimestamp(created)
		if err != nil {
			return nil, fmt.Errorf("parse Task Comment created time: %w", err)
		}
		comment.Updated, err = parseDatabaseTimestamp(updated)
		if err != nil {
			return nil, fmt.Errorf("parse Task Comment updated time: %w", err)
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Task Comments: %w", err)
	}
	sort.SliceStable(comments, func(i, j int) bool {
		if comments[i].Created.Equal(comments[j].Created) {
			return comments[i].ID < comments[j].ID
		}
		return comments[i].Created.Before(comments[j].Created)
	})
	return comments, nil
}

func taskCommentHTML(text string) (string, error) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if text == "" {
		return "", fmt.Errorf("Comment text is required")
	}
	paragraphs := strings.Split(text, "\n\n")
	encoded := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		lines := strings.Split(paragraph, "\n")
		for index := range lines {
			lines[index] = html.EscapeString(lines[index])
		}
		encoded = append(encoded, "<p>"+strings.Join(lines, "<br>")+"</p>")
	}
	if len(encoded) == 0 {
		return "", fmt.Errorf("Comment text is required")
	}
	return strings.Join(encoded, ""), nil
}

func (s *SQLiteStore) CreateTask(ctx context.Context, draft TaskDraft) (Task, error) {
	var prepareErr error
	draft, prepareErr = prepareDescriptionDraft(draft)
	if prepareErr != nil {
		return Task{}, prepareErr
	}
	if err := validateDraft(draft); err != nil {
		return Task{}, err
	}

	tx, connection, err := beginTaskSave(ctx, s.db)
	if err != nil {
		return Task{}, fmt.Errorf("begin create task transaction: %w", err)
	}
	defer connection.Close()
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
	now := utcTimestampNow()
	columns := `title, description, done, due_date, project_id, priority, "index", created, updated, created_by_id`
	values := []any{draft.Title, draft.Description, 0, dueDate, draft.ProjectID, draft.Priority, index, now, now, createdByID}
	hasUID, err := taskColumnExists(ctx, tx, "uid")
	if err != nil {
		return Task{}, fmt.Errorf("check Task UID schema: %w", err)
	}
	if hasUID {
		uid, err := newTaskUID()
		if err != nil {
			return Task{}, err
		}
		columns += ", uid"
		values = append(values, uid)
	}
	// Include UID in the initial insert, also supporting NOT NULL/unique schemas.
	query := "INSERT INTO tasks (" + columns + ") VALUES (" + strings.TrimRight(strings.Repeat("?,", len(values)), ",") + ")"
	result, err := tx.ExecContext(ctx, query, values...)
	if err != nil {
		return Task{}, fmt.Errorf("insert task: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Task{}, fmt.Errorf("get created task id: %w", err)
	}
	if err := initializeViewAssignments(ctx, tx, id, draft.ProjectID, false); err != nil {
		return Task{}, err
	}
	if draft.WorkflowBucketID != 0 {
		if err := setTaskWorkflowStatus(ctx, tx, id, draft.ProjectID, draft.WorkflowBucketID); err != nil {
			return Task{}, err
		}
	}
	if draft.LabelIDsSet {
		if err := replaceTaskLabels(ctx, tx, id, draft.LabelIDs); err != nil {
			return Task{}, err
		}
	}
	created := Task{ID: id, ProjectID: draft.ProjectID, Title: draft.Title, Description: draft.Description, Priority: draft.Priority, DueDate: draft.DueDate, Index: index}
	if draft.WorkflowBucketID != 0 {
		if err := tx.QueryRowContext(ctx, `SELECT done FROM tasks WHERE id = ?`, id).Scan(&created.Done); err != nil {
			return Task{}, fmt.Errorf("get created task done state: %w", err)
		}
	}
	if err := commitTaskSave(tx, connection); err != nil {
		return Task{}, fmt.Errorf("commit create task: %w", err)
	}
	return created, nil
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
	result, err := s.db.ExecContext(ctx, `UPDATE tasks SET title = ?, updated = ? WHERE id = ?`, title, utcTimestampNow(), taskID)
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
	var prepareErr error
	draft, prepareErr = prepareDescriptionDraft(draft)
	if prepareErr != nil {
		return Task{}, prepareErr
	}
	if draft.ID == 0 {
		return Task{}, fmt.Errorf("update task: missing task id")
	}
	if err := validateDraft(draft); err != nil {
		return Task{}, err
	}
	tx, connection, err := beginTaskSave(ctx, s.db)
	if err != nil {
		return Task{}, fmt.Errorf("begin update task transaction: %w", err)
	}
	defer connection.Close()
	defer tx.Rollback()
	task, err := updateTaskInTx(ctx, tx, draft)
	if err != nil {
		return Task{}, err
	}
	if err := commitTaskSave(tx, connection); err != nil {
		return Task{}, fmt.Errorf("commit update task: %w", err)
	}
	return task, nil
}

func (s *SQLiteStore) SaveTaskWithComment(ctx context.Context, draft TaskDraft, commentDraft TaskCommentDraft) (Task, TaskComment, error) {
	var prepareErr error
	draft, prepareErr = prepareDescriptionDraft(draft)
	if prepareErr != nil {
		return Task{}, TaskComment{}, prepareErr
	}
	if draft.ID == 0 {
		return Task{}, TaskComment{}, fmt.Errorf("save Task with Comment: Task must be persisted")
	}
	if err := validateDraft(draft); err != nil {
		return Task{}, TaskComment{}, err
	}
	commentHTML, err := taskCommentHTML(commentDraft.Text)
	if err != nil {
		return Task{}, TaskComment{}, err
	}
	tx, connection, err := beginTaskSave(ctx, s.db)
	if err != nil {
		return Task{}, TaskComment{}, fmt.Errorf("begin Task Comment transaction: %w", err)
	}
	defer connection.Close()
	defer tx.Rollback()
	actorID, err := onlyLocalUser(ctx, tx)
	if err != nil {
		return Task{}, TaskComment{}, fmt.Errorf("create Task Comment: %w", err)
	}
	task, err := updateTaskInTx(ctx, tx, draft)
	if err != nil {
		return Task{}, TaskComment{}, err
	}
	now := utcTimestampNow()
	result, err := tx.ExecContext(ctx, `INSERT INTO task_comments (comment, author_id, task_id, created, updated) VALUES (?, ?, ?, ?, ?)`, commentHTML, actorID, draft.ID, now, now)
	if err != nil {
		return Task{}, TaskComment{}, fmt.Errorf("insert Task Comment: %w", err)
	}
	commentID, err := result.LastInsertId()
	if err != nil {
		return Task{}, TaskComment{}, fmt.Errorf("get Task Comment id: %w", err)
	}
	var author string
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(TRIM(name), ''), username) FROM users WHERE id = ?`, actorID).Scan(&author); err != nil {
		return Task{}, TaskComment{}, fmt.Errorf("get Task Comment author: %w", err)
	}
	if err := commitTaskSave(tx, connection); err != nil {
		return Task{}, TaskComment{}, fmt.Errorf("commit Task Comment: %w", err)
	}
	created, _ := parseDatabaseTimestamp(now)
	return task, TaskComment{ID: commentID, TaskID: draft.ID, HTML: commentHTML, Author: author, CanDelete: true, Created: created, Updated: created}, nil
}

// DeleteTaskComment enforces Vikunja's author-only Comment deletion rule in
// the Store so the direct SQLite path cannot bypass the TUI affordance.
func (s *SQLiteStore) DeleteTaskComment(ctx context.Context, taskID, commentID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete Task Comment: begin transaction: %w", err)
	}
	defer tx.Rollback()
	actorID, err := onlyLocalUser(ctx, tx)
	if err != nil {
		return fmt.Errorf("delete Task Comment: %w", err)
	}
	var authorID int64
	if err := tx.QueryRowContext(ctx, `SELECT author_id FROM task_comments WHERE id = ? AND task_id = ?`, commentID, taskID).Scan(&authorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("delete Task Comment: Comment not found")
		}
		return fmt.Errorf("delete Task Comment: load Comment: %w", err)
	}
	if authorID != actorID {
		return fmt.Errorf("delete Task Comment: only the Comment author may delete it")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_comments WHERE id = ? AND task_id = ?`, commentID, taskID); err != nil {
		return fmt.Errorf("delete Task Comment: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete Task Comment: commit: %w", err)
	}
	return nil
}

func updateTaskInTx(ctx context.Context, tx *sql.Tx, draft TaskDraft) (Task, error) {
	var oldProjectID int64
	var done bool
	var currentDue sql.NullTime
	var currentDescription sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT project_id, done, due_date, description FROM tasks WHERE id = ?`, draft.ID).Scan(&oldProjectID, &done, &currentDue, &currentDescription); err != nil {
		return Task{}, fmt.Errorf("get task for update: %w", err)
	}
	if draft.DescriptionBaseline != nil && currentDescription.String != *draft.DescriptionBaseline {
		return Task{}, &DescriptionConflictError{Baseline: *draft.DescriptionBaseline, Current: currentDescription.String}
	}
	// The destination may have disappeared since the editor loaded Projects.
	// Validate it in this transaction before changing the Task or its relations.
	var projectExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id = ?)`, draft.ProjectID).Scan(&projectExists); err != nil {
		return Task{}, fmt.Errorf("validate target project: %w", err)
	}
	if !projectExists {
		return Task{}, fmt.Errorf("update task: project %d not found", draft.ProjectID)
	}
	dueDate, err := dueDateValue(draft.DueDate)
	if err != nil {
		return Task{}, err
	}
	// Date-only drafts cannot represent the existing time. Leave the column
	// untouched when its displayed date is unchanged, including fractional
	// seconds and timezone notation supplied by Vikunja.
	currentDate := ""
	if currentDue.Valid {
		currentDate = currentDue.Time.Format("2006-01-02")
	}
	dueChanged := draft.DueDate != currentDate
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
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET title = ?, description = ?, due_date = CASE WHEN ? THEN ? ELSE due_date END, project_id = ?, priority = ?, "index" = ?, updated = ? WHERE id = ?`, draft.Title, draft.Description, dueChanged, dueDate, draft.ProjectID, draft.Priority, index, utcTimestampNow(), draft.ID); err != nil {
		return Task{}, fmt.Errorf("update task: %w", err)
	}
	if oldProjectID != draft.ProjectID {
		if err := initializeViewAssignments(ctx, tx, draft.ID, draft.ProjectID, done); err != nil {
			return Task{}, err
		}
	} else if draft.WorkflowBucketID != 0 {
		// Reapplying the loaded Bucket is not a workflow transition. In
		// particular, a completed Task may have no configured Done Bucket.
		var unchanged bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM task_buckets tb
				JOIN buckets b ON b.id = tb.bucket_id AND b.project_view_id = tb.project_view_id
				JOIN project_views pv ON pv.id = tb.project_view_id
				WHERE tb.task_id = ? AND tb.bucket_id = ? AND pv.project_id = ?
					AND pv.view_kind = ? AND pv.bucket_configuration_mode = ?
			)
		`, draft.ID, draft.WorkflowBucketID, draft.ProjectID, viewKindKanban, bucketConfigurationManual).Scan(&unchanged); err != nil {
			return Task{}, fmt.Errorf("check current workflow bucket: %w", err)
		}
		if !unchanged {
			if err := setTaskWorkflowStatus(ctx, tx, draft.ID, draft.ProjectID, draft.WorkflowBucketID); err != nil {
				return Task{}, err
			}
		}
		if err := tx.QueryRowContext(ctx, `SELECT done FROM tasks WHERE id = ?`, draft.ID).Scan(&done); err != nil {
			return Task{}, fmt.Errorf("get updated task done state: %w", err)
		}
	}
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

type bucketQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Zero/NULL configuration is absent. Match Vikunja's position ordering and
// make ties deterministic without ever inventing a Bucket-zero assignment.
func resolveDefaultBucket(ctx context.Context, q bucketQuerier, viewID int64, configured sql.NullInt64) (int64, error) {
	if configured.Valid && configured.Int64 > 0 {
		return configured.Int64, nil
	}
	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM buckets WHERE project_view_id = ? AND id > 0 ORDER BY position ASC, id ASC LIMIT 1`, viewID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("resolve default Bucket: %w", err)
	}
	return id, nil
}

func validateViewBucket(ctx context.Context, tx *sql.Tx, viewID, bucketID int64) error {
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM buckets WHERE id = ? AND project_view_id = ?)`, bucketID, viewID).Scan(&exists); err != nil {
		return fmt.Errorf("validate configured Bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("configured Bucket %d does not belong to View %d", bucketID, viewID)
	}
	return nil
}

func initializeViewAssignments(ctx context.Context, tx *sql.Tx, taskID, projectID int64, done bool) error {
	query := `SELECT id, default_bucket_id, NULL FROM project_views WHERE project_id = ?`
	if done {
		query = `SELECT id, default_bucket_id, done_bucket_id FROM project_views WHERE project_id = ?`
	}
	rows, err := tx.QueryContext(ctx, query, projectID)
	if err != nil {
		return fmt.Errorf("list target project views: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var viewID int64
		var bucketID, doneBucketID sql.NullInt64
		if err := rows.Scan(&viewID, &bucketID, &doneBucketID); err != nil {
			return fmt.Errorf("scan target project view: %w", err)
		}
		selectedBucket := int64(0)
		if done && doneBucketID.Valid && doneBucketID.Int64 > 0 {
			selectedBucket = doneBucketID.Int64
		} else {
			var err error
			selectedBucket, err = resolveDefaultBucket(ctx, tx, viewID, bucketID)
			if err != nil {
				return err
			}
		}
		if selectedBucket != 0 {
			if err := validateViewBucket(ctx, tx, viewID, selectedBucket); err != nil {
				return err
			}
		}
		var position float64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), 0) + 1 FROM task_positions WHERE project_view_id = ?`, viewID).Scan(&position); err != nil {
			return fmt.Errorf("next task position: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_positions (task_id, project_view_id, position) VALUES (?, ?, ?)`, taskID, viewID, position); err != nil {
			return fmt.Errorf("insert task position: %w", err)
		}
		if selectedBucket != 0 {
			if _, err := tx.ExecContext(ctx, `INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (?, ?, ?)`, taskID, selectedBucket, viewID); err != nil {
				return fmt.Errorf("insert initial task bucket: %w", err)
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

// Optional historical columns are inspected inside the write transaction.
func taskColumnExists(ctx context.Context, tx *sql.Tx, name string) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pragma_table_info('tasks') WHERE name=? COLLATE NOCASE)`, name).Scan(&exists)
	return exists, err
}

// Recurrence is owned by Vikunja: never freeze its next occurrence through a
// local completion. Other edits and reopening do not advance recurrence.
func rejectRecurringCompletion(ctx context.Context, tx *sql.Tx, taskID int64, done bool) error {
	if !done {
		return nil
	}
	var current bool
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(done,0) FROM tasks WHERE id=?`, taskID).Scan(&current); err != nil {
		return fmt.Errorf("check recurring completion: %w", err)
	}
	if current {
		return nil
	}
	expressions := []string{"0", "0"}
	for i, column := range []string{"repeat_after", "repeat_mode"} {
		exists, err := taskColumnExists(ctx, tx, column)
		if err != nil {
			return fmt.Errorf("check recurrence schema: %w", err)
		}
		if exists {
			expressions[i] = "COALESCE(" + column + ",0)"
		}
	}
	var recurring bool
	if err := tx.QueryRowContext(ctx, "SELECT ("+expressions[0]+" != 0 OR "+expressions[1]+" != 0) FROM tasks WHERE id=?", taskID).Scan(&recurring); err != nil {
		return fmt.Errorf("check recurrence: %w", err)
	}
	if recurring {
		return fmt.Errorf("cannot complete recurring Task in Vimkunja; complete it in Vikunja to advance dates, reminders and checklists")
	}
	return nil
}

func deleteTaskPolymorphicRelations(ctx context.Context, tx *sql.Tx, taskID int64) error {
	// Vikunja enums: Favorite Task=1, Subscription Task=3;
	// Reaction Task=0, Comment=1. Entity IDs alone are not unique across kinds.
	deletes := []struct{ table, query string }{
		{"favorites", `DELETE FROM favorites WHERE entity_id=? AND kind=1`},
		{"subscriptions", `DELETE FROM subscriptions WHERE entity_id=? AND entity_type=3`},
		{"reactions", `DELETE FROM reactions WHERE entity_id=? AND entity_kind=0`},
	}
	for _, d := range deletes {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)`, d.table).Scan(&exists); err != nil {
			return fmt.Errorf("check polymorphic %s: %w", d.table, err)
		}
		if !exists {
			continue
		}
		if _, err := tx.ExecContext(ctx, d.query, taskID); err != nil {
			return fmt.Errorf("delete Task %s: %w", d.table, err)
		}
		if d.table == "reactions" {
			var comments bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='task_comments')`).Scan(&comments); err != nil {
				return fmt.Errorf("check reaction Comments: %w", err)
			}
			if comments {
				if _, err := tx.ExecContext(ctx, `DELETE FROM reactions WHERE entity_kind=1 AND entity_id IN (SELECT id FROM task_comments WHERE task_id=?)`, taskID); err != nil {
					return fmt.Errorf("delete Task Comment reactions: %w", err)
				}
			}
		}
	}
	return nil
}

// UUID v4 matches Vikunja's stable CalDAV identity without a new dependency.
func newTaskUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate Task UID: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", value[:4], value[4:6], value[6:8], value[8:10], value[10:]), nil
}
