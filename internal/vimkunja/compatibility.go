package vimkunja

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
)

// NewVerifiedSQLiteStore is the productive entry point. NewSQLiteStore remains
// available for the synthetic tutorial and tests that initialize their own DB.
func NewVerifiedSQLiteStore(path string) (*SQLiteStore, error) {
	store, err := NewSQLiteStore(path)
	if err != nil {
		return nil, err
	}
	if err := store.verifyVikunja270(context.Background()); err != nil {
		// Reject without checkpointing or issuing any business writes.
		_ = store.db.Close()
		return nil, fmt.Errorf("unsupported Vikunja database: only the verified 2.7.0 revision is supported: %w", err)
	}
	return store, nil
}

func (s *SQLiteStore) verifyVikunja270(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id FROM migration ORDER BY id`)
	if err != nil {
		return fmt.Errorf("read migration history: %w", err)
	}
	var ids []string
	initCount := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		if id == "SCHEMA_INIT" {
			initCount++
			continue
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	// Timestamp IDs from v2.7.0, commit a16be96aa454671fdf213b0fbe411dd38a098418.
	// SCHEMA_INIT is optional on upgraded installations, not a release identifier.
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(ids, "\n")+"\n")))
	if initCount > 1 || digest != "eca7263cba028994b569bcbd047c86337758c16dbf8fd3363f9d392ee6d9df59" {
		latest := "none"
		if len(ids) > 0 {
			latest = ids[len(ids)-1]
		}
		return fmt.Errorf("migration history differs from Vikunja 2.7.0 (found %d migrations, latest %s; expected 150, latest 20260928140648); upgrade an older database copy using official Vikunja 2.7.0 before opening it in Vimkunja", len(ids), latest)
	}
	// Check every surface used by the Store, not just a last-migration marker.
	for table, columns := range vikunja270Columns {
		for _, column := range strings.Fields(columns) {
			var exists bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pragma_table_info(?) WHERE name = ?)`, table, column).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("missing %s.%s", table, column)
			}
		}
	}
	for table, columns := range map[string]string{
		"tasks": "project_id,index", "task_positions": "task_id,project_view_id",
		"task_index_aliases": "project_id,index", "task_buckets": "task_id,project_view_id",
	} {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 FROM pragma_index_list(?) i WHERE i."unique" = 1 AND i.partial = 0
			AND (SELECT group_concat(name, ',') FROM (SELECT name FROM pragma_index_info(i.name) ORDER BY seqno)) = ?
			AND (SELECT COUNT(*) FROM pragma_index_info(i.name)) = ?
		)`, table, columns, len(strings.Split(columns, ","))).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("missing unique constraint on %s(%s)", table, columns)
		}
	}
	var counterKey bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pragma_table_info('project_task_counters') WHERE name = 'project_id' AND pk = 1)
		AND (SELECT COUNT(*) FROM pragma_table_info('project_task_counters') WHERE pk > 0) = 1`).Scan(&counterKey); err != nil {
		return err
	}
	if !counterKey {
		return fmt.Errorf("missing project_task_counters primary key")
	}
	return nil
}

var vikunja270Columns = map[string]string{
	"projects":              "id identifier title description parent_project_id position is_archived owner_id updated",
	"tasks":                 "id project_id title description done done_at due_date priority index uid created updated created_by_id deleted_at repeat_after repeat_mode",
	"project_views":         "id project_id title view_kind position bucket_configuration_mode default_bucket_id done_bucket_id created updated",
	"buckets":               "id title project_view_id position limit created updated created_by_id",
	"task_positions":        "task_id project_view_id position",
	"task_buckets":          "task_id project_view_id bucket_id",
	"project_task_counters": "project_id last_index",
	"task_index_aliases":    "project_id index task_id",
	"project_ancestors":     "ancestor_id project_id depth",
	"users":                 "id name username",
	"labels":                "id title created_by_id created updated",
	"label_tasks":           "task_id label_id created",
	"task_comments":         "id task_id comment author_id created updated",
	"task_assignees":        "task_id",
	"task_reminders":        "task_id",
	"task_relations":        "task_id other_task_id",
	"task_attachments":      "task_id",
	"task_unread_statuses":  "task_id",
	"favorites":             "entity_id kind",
	"subscriptions":         "entity_id entity_type",
	"reactions":             "entity_id entity_kind",
	"time_entries":          "task_id",
}

func activeTaskProject(ctx context.Context, q bucketQuerier, taskID int64) (int64, error) {
	var projectID int64
	if err := q.QueryRowContext(ctx, `SELECT project_id FROM tasks WHERE id = ? AND deleted_at IS NULL`, taskID).Scan(&projectID); err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("Task %d not found or in trash", taskID)
		}
		return 0, fmt.Errorf("load active Task: %w", err)
	}
	return projectID, nil
}

func writableProject(ctx context.Context, q bucketQuerier, projectID int64) error {
	var archived bool
	if err := q.QueryRowContext(ctx, `SELECT COALESCE(is_archived, 0) FROM projects WHERE id = ?`, projectID).Scan(&archived); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("project %d not found", projectID)
		}
		return fmt.Errorf("check writable project: %w", err)
	}
	if archived {
		return fmt.Errorf("project %d is archived and read-only", projectID)
	}
	return nil
}

func writableTask(ctx context.Context, q bucketQuerier, taskID int64) error {
	projectID, err := activeTaskProject(ctx, q, taskID)
	if err != nil {
		return err
	}
	return writableProject(ctx, q, projectID)
}

// Moving back through undo consumes a fresh number too. Every retired address
// stays resolvable; deleting a Task never decreases a Project's high-water mark.
func recordTaskIndexAlias(ctx context.Context, tx *sql.Tx, taskID int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO task_index_aliases (project_id, "index", task_id)
		SELECT project_id, "index", id FROM tasks WHERE id = ?
		ON CONFLICT(project_id, "index") DO UPDATE SET task_id = excluded.task_id`, taskID)
	if err != nil {
		return fmt.Errorf("record retired Task address: %w", err)
	}
	return nil
}

// Preserve every address before deleting the Task and its aliases, including
// Projects whose counter is missing or stale after older direct writes.
func reserveDeletedTaskIndexes(ctx context.Context, tx *sql.Tx, taskID int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO project_task_counters (project_id, last_index)
		SELECT project_id, MAX("index") FROM (
			SELECT project_id, "index" FROM tasks WHERE id = ?
			UNION ALL SELECT project_id, "index" FROM task_index_aliases WHERE task_id = ?
		) WHERE "index" IS NOT NULL GROUP BY project_id
		ON CONFLICT(project_id) DO UPDATE SET last_index = MAX(last_index, excluded.last_index)`, taskID, taskID)
	if err != nil {
		return fmt.Errorf("reserve deleted Task addresses: %w", err)
	}
	return nil
}

func checkBucketCapacity(ctx context.Context, tx *sql.Tx, taskID, viewID, bucketID int64) error {
	if bucketID == 0 {
		return nil
	}
	var limit int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE("limit", 0) FROM buckets WHERE id = ? AND project_view_id = ?`, bucketID, viewID).Scan(&limit); err != nil {
		return fmt.Errorf("check Bucket capacity: %w", err)
	}
	if limit <= 0 {
		return nil
	}
	var alreadyAssigned bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM task_buckets WHERE task_id = ? AND project_view_id = ? AND bucket_id = ?)`, taskID, viewID, bucketID).Scan(&alreadyAssigned); err != nil {
		return err
	}
	if alreadyAssigned {
		return nil // An unchanged assignment must remain writable at capacity.
	}
	var count int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_buckets tb JOIN tasks t ON t.id = tb.task_id
		WHERE tb.bucket_id = ? AND tb.project_view_id = ? AND t.deleted_at IS NULL AND t.id != ?`, bucketID, viewID, taskID).Scan(&count); err != nil {
		return err
	}
	if count >= limit {
		return fmt.Errorf("Bucket %d is full (limit %d)", bucketID, limit)
	}
	return nil
}
