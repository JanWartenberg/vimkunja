package vimkunja

import (
	"context"
	"fmt"
)

// DeleteEmptyProject permanently removes a leaf Project and its metadata. Tasks
// in trash still count as content; recursive deletion belongs to a separate flow.
func (s *SQLiteStore) DeleteEmptyProject(ctx context.Context, projectID int64) error {
	if projectID <= 0 {
		return fmt.Errorf("select a valid Project to delete")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Reserve the writer before checking emptiness so another writer cannot add
	// content between validation and deletion.
	if _, err := tx.ExecContext(ctx, `UPDATE projects SET id=id WHERE 0`); err != nil {
		return err
	}
	if err := checkProjectDeletion(ctx, tx, projectID); err != nil {
		return err
	}
	var hasDefault bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pragma_table_info('users') WHERE name='default_project_id')`).Scan(&hasDefault); err != nil {
		return err
	}
	if hasDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE users SET default_project_id=0 WHERE default_project_id=?`, projectID); err != nil {
			return err
		}
	}
	for _, relation := range []struct{ table, query string }{
		{"task_positions", `DELETE FROM task_positions WHERE project_view_id IN (SELECT id FROM project_views WHERE project_id=?)`},
		{"task_buckets", `DELETE FROM task_buckets WHERE project_view_id IN (SELECT id FROM project_views WHERE project_id=?)`},
		{"buckets", `DELETE FROM buckets WHERE project_view_id IN (SELECT id FROM project_views WHERE project_id=?)`},
		{"project_views", `DELETE FROM project_views WHERE project_id=?`},
		{"favorites", `DELETE FROM favorites WHERE entity_id=? AND kind=2`},
		{"subscriptions", `DELETE FROM subscriptions WHERE entity_id=? AND entity_type=2`},
		{"link_shares", `DELETE FROM link_shares WHERE project_id=?`},
		{"users_projects", `DELETE FROM users_projects WHERE project_id=?`},
		{"team_projects", `DELETE FROM team_projects WHERE project_id=?`},
		{"webhooks", `DELETE FROM webhooks WHERE project_id=?`},
		{"notifications", `DELETE FROM notifications WHERE project_id=?`},
		{"project_ancestors", `DELETE FROM project_ancestors WHERE project_id=?`},
		{"task_index_aliases", `DELETE FROM task_index_aliases WHERE project_id=?`},
		{"project_task_counters", `DELETE FROM project_task_counters WHERE project_id=?`},
	} {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)`, relation.table).Scan(&exists); err != nil {
			return err
		}
		if exists {
			if _, err := tx.ExecContext(ctx, relation.query, projectID); err != nil {
				return fmt.Errorf("delete Project metadata from %s: %w", relation.table, err)
			}
		}
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id=?`, projectID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("Project %d was not deleted", projectID)
	}
	return tx.Commit()
}

// CheckProjectDeletion reports the current blocker without changing the Project.
// It is advisory: DeleteEmptyProject reuses these checks under its writer lock.
func (s *SQLiteStore) CheckProjectDeletion(ctx context.Context, projectID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return checkProjectDeletion(ctx, tx, projectID)
}

func checkProjectDeletion(ctx context.Context, q bucketQuerier, projectID int64) error {
	if projectID <= 0 {
		return fmt.Errorf("select a valid Project to delete")
	}
	if err := writableProject(ctx, q, projectID); err != nil {
		return err
	}
	for _, check := range []struct{ query, message string }{
		{`SELECT EXISTS(SELECT 1 FROM tasks WHERE project_id=?)`, "Project contains Tasks (including trash); delete it in Vikunja"},
		{`SELECT EXISTS(SELECT 1 FROM projects WHERE parent_project_id=?)`, "Project contains Subprojects; delete them first"},
		{`SELECT EXISTS(SELECT 1 FROM project_ancestors WHERE ancestor_id=? AND project_id<>ancestor_id)`, "Project has descendant relationships; resolve them in Vikunja first"},
	} {
		var exists bool
		if err := q.QueryRowContext(ctx, check.query, projectID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("%s", check.message)
		}
	}
	var hasTimeEntries bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='time_entries')`).Scan(&hasTimeEntries); err != nil {
		return err
	}
	if hasTimeEntries {
		var exists bool
		if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM time_entries WHERE project_id=?)`, projectID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Project has Time Entries; delete it in Vikunja")
		}
	}
	// The synthetic tutorial omits these upstream columns. Background file
	// deletion needs VOA's filesystem handling and is deliberately not performed.
	var hasBackground bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pragma_table_info('projects') WHERE name='background_file_id')`).Scan(&hasBackground); err != nil {
		return err
	}
	if hasBackground {
		var background int64
		if err := q.QueryRowContext(ctx, `SELECT COALESCE(background_file_id,0) FROM projects WHERE id=?`, projectID).Scan(&background); err != nil {
			return err
		}
		if background != 0 {
			return fmt.Errorf("Project has a background file; delete it in Vikunja")
		}
	}
	return nil
}
