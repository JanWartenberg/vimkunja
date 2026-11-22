package vimkunja

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// KanbanConfiguration is a configuration-only draft. Bucket order is slice order;
// negative Bucket IDs identify new draft Buckets, never database rows.
type KanbanConfiguration struct {
	ProjectID, ViewID             int64
	Title                         string
	Position                      float64
	DefaultBucketID, DoneBucketID int64
	Buckets                       []KanbanBucketConfiguration
	Revision                      string
}

type KanbanBucketConfiguration struct {
	ID        int64
	Title     string
	Limit     int64
	TaskCount int
}

func validKanbanTitle(title string, board bool) error {
	n := utf8.RuneCountInString(strings.TrimSpace(title))
	if n == 0 || (board && n > 250) {
		return fmt.Errorf("Board name must contain 1–250 characters; Bucket names must not be empty")
	}
	return nil
}

func (s *SQLiteStore) CreateKanbanView(ctx context.Context, projectID int64, title string) (KanbanConfiguration, error) {
	title = strings.TrimSpace(title)
	if err := validKanbanTitle(title, true); err != nil {
		return KanbanConfiguration{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return KanbanConfiguration{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE projects SET id=id WHERE 0`); err != nil {
		return KanbanConfiguration{}, err
	}
	if err = writableProject(ctx, tx, projectID); err != nil {
		return KanbanConfiguration{}, err
	}
	owner, err := onlyLocalUser(ctx, tx)
	if err != nil {
		return KanbanConfiguration{}, err
	}
	var position float64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position),0)+100 FROM project_views WHERE project_id=?`, projectID).Scan(&position); err != nil {
		return KanbanConfiguration{}, err
	}
	if math.IsInf(position, 0) || math.IsNaN(position) {
		return KanbanConfiguration{}, fmt.Errorf("invalid View position")
	}
	now := utcTimestampNow()
	res, err := tx.ExecContext(ctx, `INSERT INTO project_views(project_id,title,view_kind,bucket_configuration_mode,position,created,updated) VALUES(?,?,3,1,?,?,?)`, projectID, title, position, now, now)
	if err != nil {
		return KanbanConfiguration{}, err
	}
	viewID, err := res.LastInsertId()
	if err != nil {
		return KanbanConfiguration{}, err
	}
	var ids []int64
	for i, name := range []string{"To-Do", "Doing", "Done"} {
		res, err = tx.ExecContext(ctx, `INSERT INTO buckets(title,project_view_id,position,"limit",created_by_id,created,updated) VALUES(?,?,?,0,?,?,?)`, name, viewID, (i+1)*100, owner, now, now)
		if err != nil {
			return KanbanConfiguration{}, err
		}
		id, e := res.LastInsertId()
		if e != nil {
			return KanbanConfiguration{}, e
		}
		ids = append(ids, id)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE project_views SET default_bucket_id=?,done_bucket_id=? WHERE id=?`, ids[0], ids[2], viewID); err != nil {
		return KanbanConfiguration{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO task_buckets(task_id,bucket_id,project_view_id) SELECT id,?,? FROM tasks WHERE project_id=? AND deleted_at IS NULL`, ids[0], viewID, projectID); err != nil {
		return KanbanConfiguration{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO task_positions(task_id,project_view_id,position) SELECT id,?,ROW_NUMBER() OVER(ORDER BY id)*65536 FROM tasks WHERE project_id=? AND deleted_at IS NULL`, viewID, projectID); err != nil {
		return KanbanConfiguration{}, err
	}
	cfg, err := loadKanbanConfiguration(ctx, tx, projectID, viewID)
	if err != nil {
		return KanbanConfiguration{}, err
	}
	if err = tx.Commit(); err != nil {
		return KanbanConfiguration{}, err
	}
	return cfg, nil
}

func (s *SQLiteStore) LoadKanbanConfiguration(ctx context.Context, projectID, viewID int64) (KanbanConfiguration, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return KanbanConfiguration{}, err
	}
	defer tx.Rollback()
	cfg, err := loadKanbanConfiguration(ctx, tx, projectID, viewID)
	if err != nil {
		return KanbanConfiguration{}, err
	}
	if err = tx.Commit(); err != nil {
		return KanbanConfiguration{}, err
	}
	return cfg, nil
}

func loadKanbanConfiguration(ctx context.Context, q *sql.Tx, projectID, viewID int64) (KanbanConfiguration, error) {
	cfg := KanbanConfiguration{ProjectID: projectID, ViewID: viewID}
	var kind, mode int
	var filter sql.NullString
	err := q.QueryRowContext(ctx, `SELECT title,COALESCE(position,0),view_kind,COALESCE(bucket_configuration_mode,0),COALESCE(default_bucket_id,0),COALESCE(done_bucket_id,0),filter FROM project_views WHERE id=? AND project_id=?`, viewID, projectID).Scan(&cfg.Title, &cfg.Position, &kind, &mode, &cfg.DefaultBucketID, &cfg.DoneBucketID, &filter)
	if err != nil {
		return cfg, fmt.Errorf("load Board configuration: %w", err)
	}
	if kind != viewKindKanban || mode != bucketConfigurationManual || kanbanFilterPresent(filter.String) {
		return cfg, fmt.Errorf("only manual Boards without View filters are supported; manage this View in VOA")
	}
	rows, err := q.QueryContext(ctx, `SELECT b.id,b.title,COALESCE(b."limit",0),COUNT(tb.task_id) FROM buckets b LEFT JOIN task_buckets tb ON tb.bucket_id=b.id AND tb.project_view_id=b.project_view_id WHERE b.project_view_id=? GROUP BY b.id ORDER BY b.position,b.id`, viewID)
	if err != nil {
		return cfg, err
	}
	defer rows.Close()
	for rows.Next() {
		var b KanbanBucketConfiguration
		if err = rows.Scan(&b.ID, &b.Title, &b.Limit, &b.TaskCount); err != nil {
			return cfg, err
		}
		cfg.Buckets = append(cfg.Buckets, b)
	}
	if err = rows.Err(); err != nil {
		return cfg, err
	}
	revision := cfg
	revision.Buckets = append([]KanbanBucketConfiguration(nil), cfg.Buckets...)
	for i := range revision.Buckets {
		revision.Buckets[i].TaskCount = 0
	}
	data, err := json.Marshal(revision)
	if err != nil {
		return cfg, err
	}
	cfg.Revision = fmt.Sprintf("%x", sha256.Sum256(data))
	return cfg, nil
}

// Even default JSON filter objects have empty query strings. Unknown filter data
// is treated conservatively so management never erases an imported feature.
func kanbanFilterPresent(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &fields) != nil {
		return true
	}
	for key, value := range fields {
		switch key {
		case "filter", "s", "search":
			var text string
			if json.Unmarshal(value, &text) != nil || strings.TrimSpace(text) != "" {
				return true
			}
		case "sort_by", "order_by":
			var list []string
			if json.Unmarshal(value, &list) != nil || len(list) > 0 {
				return true
			}
		case "filter_timezone", "filter_include_nulls": // Inert without a query.
		default:
			if string(value) != "null" {
				return true
			}
		}
	}
	return false
}

// SaveKanbanConfiguration compares the configuration revision in the same write
// transaction as all changes. Task counts are advisory; capacity is rechecked.
func (s *SQLiteStore) SaveKanbanConfiguration(ctx context.Context, draft KanbanConfiguration) (KanbanConfiguration, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return draft, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE projects SET id=id WHERE 0`); err != nil {
		return draft, err
	}
	if err = writableProject(ctx, tx, draft.ProjectID); err != nil {
		return draft, err
	}
	current, err := loadKanbanConfiguration(ctx, tx, draft.ProjectID, draft.ViewID)
	if err != nil {
		return draft, err
	}
	if draft.Revision == "" || draft.Revision != current.Revision {
		return draft, fmt.Errorf("Board configuration changed; reload explicitly before saving")
	}
	draft.Title = strings.TrimSpace(draft.Title)
	if err = validKanbanTitle(draft.Title, true); err != nil {
		return draft, err
	}
	if math.IsNaN(draft.Position) || math.IsInf(draft.Position, 0) || draft.Position < 0 {
		return draft, fmt.Errorf("View position must be a finite nonnegative number")
	}
	if len(draft.Buckets) == 0 {
		return draft, fmt.Errorf("cannot remove the last Bucket")
	}
	existing := map[int64]bool{}
	for _, b := range current.Buckets {
		existing[b.ID] = true
	}
	seen := map[int64]bool{}
	for _, b := range draft.Buckets {
		if b.ID == 0 || seen[b.ID] || (b.ID > 0 && !existing[b.ID]) {
			return draft, fmt.Errorf("invalid or duplicate Bucket ID")
		}
		seen[b.ID] = true
		if err = validKanbanTitle(b.Title, false); err != nil {
			return draft, err
		}
		if b.Limit < 0 {
			return draft, fmt.Errorf("Bucket limit must be nonnegative")
		}
	}
	if (draft.DefaultBucketID != 0 && !seen[draft.DefaultBucketID]) || (draft.DoneBucketID != 0 && !seen[draft.DoneBucketID]) {
		return draft, fmt.Errorf("Default and Done must reference surviving Buckets of this Board")
	}
	var owner int64
	for _, b := range draft.Buckets {
		if b.ID < 0 {
			owner, err = onlyLocalUser(ctx, tx)
			if err != nil {
				return draft, err
			}
			break
		}
	}
	now := utcTimestampNow()
	resolved := map[int64]int64{}
	for i, b := range draft.Buckets {
		id := b.ID
		if id < 0 {
			res, e := tx.ExecContext(ctx, `INSERT INTO buckets(title,project_view_id,position,"limit",created_by_id,created,updated) VALUES(?,?,?,?,?,?,?)`, strings.TrimSpace(b.Title), draft.ViewID, (i+1)*100, b.Limit, owner, now, now)
			if e != nil {
				return draft, e
			}
			id, err = res.LastInsertId()
			if err != nil {
				return draft, err
			}
		} else {
			if _, err = tx.ExecContext(ctx, `UPDATE buckets SET title=?,"limit"=?,position=?,updated=? WHERE id=? AND project_view_id=?`, strings.TrimSpace(b.Title), b.Limit, (i+1)*100, now, id, draft.ViewID); err != nil {
				return draft, err
			}
		}
		resolved[b.ID] = id
	}
	targetDefault := draft.DefaultBucketID
	if targetDefault == 0 {
		targetDefault = draft.Buckets[0].ID
	}
	targetDefault = resolved[targetDefault]
	var removed []int64
	for _, b := range current.Buckets {
		if !seen[b.ID] {
			removed = append(removed, b.ID)
		}
	}
	if len(removed) > 0 {
		var count, limit int64
		if err = tx.QueryRowContext(ctx, `SELECT COALESCE("limit",0) FROM buckets WHERE id=?`, targetDefault).Scan(&limit); err != nil {
			return draft, err
		}
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_buckets tb JOIN tasks t ON t.id=tb.task_id WHERE tb.project_view_id=? AND tb.bucket_id=? AND t.deleted_at IS NULL`, draft.ViewID, targetDefault).Scan(&count); err != nil {
			return draft, err
		}
		for _, id := range removed {
			var n int64
			if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_buckets tb JOIN tasks t ON t.id=tb.task_id WHERE tb.project_view_id=? AND tb.bucket_id=? AND t.deleted_at IS NULL`, draft.ViewID, id).Scan(&n); err != nil {
				return draft, err
			}
			count += n
		}
		if limit > 0 && count > limit {
			return draft, fmt.Errorf("deleting Buckets would exceed the Default Bucket limit (%d)", limit)
		}
		for _, id := range removed {
			if _, err = tx.ExecContext(ctx, `UPDATE task_buckets SET bucket_id=? WHERE project_view_id=? AND bucket_id=?`, targetDefault, draft.ViewID, id); err != nil {
				return draft, err
			}
			if _, err = tx.ExecContext(ctx, `DELETE FROM buckets WHERE id=? AND project_view_id=?`, id, draft.ViewID); err != nil {
				return draft, err
			}
		}
	}
	defaultID := int64(0)
	if draft.DefaultBucketID != 0 {
		defaultID = resolved[draft.DefaultBucketID]
	}
	doneID := int64(0)
	if draft.DoneBucketID != 0 {
		doneID = resolved[draft.DoneBucketID]
	}
	if _, err = tx.ExecContext(ctx, `UPDATE project_views SET title=?,position=?,default_bucket_id=?,done_bucket_id=?,updated=? WHERE id=? AND project_id=?`, draft.Title, draft.Position, defaultID, doneID, now, draft.ViewID, draft.ProjectID); err != nil {
		return draft, err
	}
	saved, err := loadKanbanConfiguration(ctx, tx, draft.ProjectID, draft.ViewID)
	if err != nil {
		return draft, err
	}
	if err = tx.Commit(); err != nil {
		return draft, err
	}
	return saved, nil
}

// DeleteKanbanView removes only View-owned presentation data. Project Tasks,
// including Tasks in trash, and every other View remain intact.
func (s *SQLiteStore) DeleteKanbanView(ctx context.Context, expected KanbanConfiguration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE projects SET id=id WHERE 0`); err != nil {
		return err
	}
	if err = writableProject(ctx, tx, expected.ProjectID); err != nil {
		return err
	}
	current, err := loadKanbanConfiguration(ctx, tx, expected.ProjectID, expected.ViewID)
	if err != nil {
		return err
	}
	if expected.Revision == "" || expected.Revision != current.Revision {
		return fmt.Errorf("Board configuration changed; reload before deleting")
	}
	for _, table := range []string{"task_buckets", "task_positions", "buckets"} {
		if _, err = tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE project_view_id=?`, expected.ViewID); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM project_views WHERE id=? AND project_id=?`, expected.ViewID, expected.ProjectID); err != nil {
		return err
	}
	return tx.Commit()
}
