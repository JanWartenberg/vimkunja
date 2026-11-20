package vimkunja

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

// ProjectDraft contains only the fields supported by the creation workflow.
type ProjectDraft struct {
	Title           string
	ParentProjectID int64
}

// ProjectCreationError identifies a correctable creation field.
type ProjectCreationError struct{ Field, Message string }

func (e *ProjectCreationError) Error() string { return e.Message }

func (s *SQLiteStore) CreateProject(ctx context.Context, draft ProjectDraft) (Project, error) {
	draft.Title = strings.TrimSpace(draft.Title)
	if n := utf8.RuneCountInString(draft.Title); n < 1 || n > 250 {
		return Project{}, &ProjectCreationError{Field: "title", Message: "Title must contain 1–250 characters"}
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return Project{}, err
	}
	defer conn.Close()
	var timeout int
	if err := conn.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		return Project{}, err
	}
	if _, err := conn.ExecContext(ctx, `PRAGMA busy_timeout=5000`); err != nil {
		return Project{}, err
	}
	defer conn.ExecContext(context.Background(), fmt.Sprintf("PRAGMA busy_timeout=%d", timeout))
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return Project{}, err
	}
	defer tx.Rollback()
	// Acquire SQLite's writer reservation before reading validation state. This
	// serializes creation checks without adding a schema constraint VOA lacks.
	if _, err = tx.ExecContext(ctx, `UPDATE projects SET id=id WHERE 0`); err != nil {
		return Project{}, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT title FROM projects`)
	if err != nil {
		return Project{}, err
	}
	duplicate := false
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			rows.Close()
			return Project{}, err
		}
		if strings.EqualFold(strings.TrimSpace(title), draft.Title) {
			duplicate = true
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return Project{}, err
	}
	if duplicate {
		return Project{}, &ProjectCreationError{Field: "title", Message: "A Project with this Title already exists (including archived Projects)"}
	}
	owner, err := onlyLocalUser(ctx, tx)
	if err != nil {
		return Project{}, err
	}
	parents, err := projectCreationParents(ctx, tx)
	if err != nil {
		return Project{}, err
	}
	var ancestry []int64
	if draft.ParentProjectID != 0 {
		found := false
		for _, p := range parents {
			if p.ID == draft.ParentProjectID {
				found = true
				ancestry = p.ancestors
				break
			}
		}
		if !found {
			return Project{}, &ProjectCreationError{Field: "parent", Message: "Parent Project is missing, archived or has an invalid hierarchy"}
		}
	}
	var last float64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position),0) FROM projects WHERE COALESCE(parent_project_id,0)=?`, draft.ParentProjectID).Scan(&last); err != nil {
		return Project{}, err
	}
	position := math.Max(0, last) + 65536
	if position <= last {
		position = math.Nextafter(last, math.Inf(1))
	}
	if math.IsInf(position, 0) || math.IsNaN(position) {
		return Project{}, fmt.Errorf("cannot append Project: invalid sibling position")
	}
	now := utcTimestampNow()
	var parent any
	if draft.ParentProjectID != 0 {
		parent = draft.ParentProjectID
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO projects(title,identifier,description,owner_id,parent_project_id,position,created,updated) VALUES (?,'','',?,?,?,?,?)`, draft.Title, owner, parent, position, now, now)
	if err != nil {
		return Project{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Project{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO project_ancestors(ancestor_id,project_id,depth) VALUES(?,?,0)`, id, id); err != nil {
		return Project{}, err
	}
	for depth, ancestorID := range ancestry {
		if _, err = tx.ExecContext(ctx, `INSERT INTO project_ancestors(ancestor_id,project_id,depth) VALUES(?,?,?)`, ancestorID, id, depth+1); err != nil {
			return Project{}, err
		}
	}

	if _, err = tx.ExecContext(ctx, `INSERT INTO project_task_counters(project_id,last_index) VALUES(?,0)`, id); err != nil {
		return Project{}, err
	}
	if err = createProjectViews(ctx, tx, id, owner, now); err != nil {
		return Project{}, err
	}
	if err = tx.Commit(); err != nil {
		return Project{}, err
	}
	return Project{ID: id, Title: draft.Title, ParentProjectID: draft.ParentProjectID, Position: position}, nil
}

func createProjectViews(ctx context.Context, tx *sql.Tx, projectID, owner int64, now string) error {
	for kind, title := range []string{"List", "Gantt", "Table", "Kanban"} {
		mode := bucketConfigurationNone
		var filter any
		if kind == viewKindList {
			filter = `{"filter":"done = false"}`
		}
		if kind == viewKindKanban {
			mode = bucketConfigurationManual
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO project_views(project_id,title,view_kind,filter,position,bucket_configuration_mode,default_bucket_id,done_bucket_id,created,updated) VALUES(?,?,?,?,?,?,0,0,?,?)`, projectID, title, kind, filter, (kind+1)*100, mode, now, now)
		if err != nil {
			return fmt.Errorf("create Project View: %w", err)
		}
		if kind != viewKindKanban {
			continue
		}
		viewID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		var first, done int64
		for i, name := range []string{"To-Do", "Doing", "Done"} {
			result, err := tx.ExecContext(ctx, `INSERT INTO buckets(title,project_view_id,position,"limit",created_by_id,created,updated) VALUES(?,?,?,0,?,?,?)`, name, viewID, (i+1)*100, owner, now, now)
			if err != nil {
				return err
			}
			bucketID, err := result.LastInsertId()
			if err != nil {
				return err
			}
			if i == 0 {
				first = bucketID
			}
			if i == 2 {
				done = bucketID
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE project_views SET default_bucket_id=?,done_bucket_id=? WHERE id=?`, first, done, viewID); err != nil {
			return err
		}
	}
	return nil
}

// ProjectParent is an eligible Parent with an unambiguous hierarchy path.
type ProjectParent struct {
	ID          int64
	Title, Path string
	ancestors   []int64
}

type projectCreationQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (s *SQLiteStore) ProjectCreationParents(ctx context.Context) ([]ProjectParent, error) {
	return projectCreationParents(ctx, s.db)
}

func projectCreationParents(ctx context.Context, q projectCreationQuerier) ([]ProjectParent, error) {
	rows, err := q.QueryContext(ctx, `SELECT id,title,COALESCE(parent_project_id,0),COALESCE(is_archived,0) FROM projects`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type node struct {
		title    string
		parent   int64
		archived bool
	}
	nodes := map[int64]node{}
	for rows.Next() {
		var id int64
		var n node
		if err := rows.Scan(&id, &n.title, &n.parent, &n.archived); err != nil {
			return nil, err
		}
		nodes[id] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var parents []ProjectParent
	for id, n := range nodes {
		if id <= 0 {
			continue
		}
		current := id
		seen := map[int64]bool{}
		var ancestors []int64
		var titles []string
		valid := true
		for current != 0 {
			ancestor, ok := nodes[current]
			if !ok || current < 0 || ancestor.archived || seen[current] {
				valid = false
				break
			}
			seen[current] = true
			ancestors = append(ancestors, current)
			titles = append(titles, ancestor.title)
			current = ancestor.parent
		}
		if !valid {
			continue
		}
		for i, j := 0, len(titles)-1; i < j; i, j = i+1, j-1 {
			titles[i], titles[j] = titles[j], titles[i]
		}
		parents = append(parents, ProjectParent{ID: id, Title: n.title, Path: strings.Join(titles, " / "), ancestors: ancestors})
	}
	sort.Slice(parents, func(i, j int) bool {
		if parents[i].Path == parents[j].Path {
			return parents[i].ID < parents[j].ID
		}
		return parents[i].Path < parents[j].Path
	})
	return parents, nil
}
