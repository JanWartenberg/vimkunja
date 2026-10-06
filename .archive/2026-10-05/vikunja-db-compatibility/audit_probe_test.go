// Observational audit probes for Jan's seeded 2.7.0 fixture (Projects 2/3, Tasks 1/2).
// Run through a Go overlay as documented in report.md. Each probe copies the
// baseline and its WAL/SHM sidecars to t.TempDir before opening SQLite.
// A passing probe means execution completed; logged gaps are NOT acceptance passes.
package vimkunja

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCompatibilityAuditProbe(t *testing.T) {
	ctx := context.Background()
	open := func(t *testing.T) *SQLiteStore {
		t.Helper()
		baseline := os.Getenv("VIMKUNJA_AUDIT_BASELINE")
		if baseline == "" {
			baseline = filepath.Join("..", "..", "tmp", "vikunja_clean_db")
		}
		p := filepath.Join(t.TempDir(), "compatibility.db")
		for _, suffix := range []string{"", "-wal", "-shm"} {
			data, err := os.ReadFile(filepath.Join(baseline, "compatibility.db") + suffix)
			if os.IsNotExist(err) && suffix != "" {
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(p+suffix, data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		s, err := NewSQLiteStore(p)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := s.Close(); err != nil {
				t.Error(err)
			}
		})
		return s
	}
	exec := func(t *testing.T, s *SQLiteStore, q string, args ...any) {
		t.Helper()
		if _, err := s.db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	n := func(t *testing.T, s *SQLiteStore, q string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := s.db.QueryRow(q, args...).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	draft := func(t *testing.T, s *SQLiteStore, id int64) TaskDraft {
		t.Helper()
		var d TaskDraft
		d.ID = id
		if err := s.db.QueryRow("SELECT project_id,title,COALESCE(description,''),COALESCE(priority,0) FROM tasks WHERE id=?", id).Scan(&d.ProjectID, &d.Title, &d.Description, &d.Priority); err != nil {
			t.Fatal(err)
		}
		return d
	}
	t.Run("baseline_reads", func(t *testing.T) {
		s := open(t)
		p, e := s.ListProjects(ctx)
		t.Logf("projects=%d err=%v", len(p), e)
		tasks, _, e := s.ListOverviewTasks(ctx, 1, 200)
		t.Logf("overview=%d err=%v", len(tasks), e)
		data, e := s.LoadSearchData(ctx)
		t.Logf("search_tasks=%d labels=%d err=%v", len(data.Tasks), len(data.Labels), e)
		commentTask := n(t, s, "SELECT task_id FROM task_comments LIMIT 1")
		c, e := s.ListTaskComments(ctx, commentTask)
		t.Logf("comments=%d err=%v", len(c), e)
		b, e := s.LoadManualKanbanBoard(ctx, 2)
		t.Logf("kanban_buckets=%d err=%v", len(b.Buckets), e)
		_, _, e = s.ListTasksInView(ctx, 2, 9, 1, 200)
		t.Logf("list_err=%v", e)
	})
	t.Run("create_counter", func(t *testing.T) {
		s := open(t)
		before := n(t, s, "SELECT last_index FROM project_task_counters WHERE project_id=2")
		task, e := s.CreateTask(ctx, TaskDraft{ProjectID: 2, Title: "Audit-created"})
		if e != nil {
			t.Fatal(e)
		}
		after := n(t, s, "SELECT last_index FROM project_task_counters WHERE project_id=2")
		t.Logf("created_index=%d counter_before=%d counter_after=%d", task.Index, before, after)
		exec(t, s, "UPDATE project_task_counters SET last_index=last_index+1 WHERE project_id=2")
		next := n(t, s, "SELECT last_index FROM project_task_counters WHERE project_id=2")
		t.Logf("upstream_next_index=%d existing_tasks_with_index=%d", next, n(t, s, "SELECT count(*) FROM tasks WHERE project_id=2 AND `index`=?", next))
		_, e = s.db.Exec("INSERT INTO tasks(title,project_id,`index`,created,updated,created_by_id) VALUES(?,2,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,1)", "Simulated VOA allocation", next)
		t.Logf("simulated_upstream_insert_err=%v", e)
	})
	t.Run("move_and_undo_aliases", func(t *testing.T) {
		s := open(t)
		before, e := s.TaskUndoState(ctx, 2)
		if e != nil {
			t.Fatal(e)
		}
		d := draft(t, s, 2)
		d.ProjectID = 3
		task, e := s.UpdateTask(ctx, d)
		if e != nil {
			t.Fatal(e)
		}
		after, e := s.TaskUndoState(ctx, 2)
		if e != nil {
			t.Fatal(e)
		}
		t.Logf("moved_index=%d target_counter=%d retired_aliases=%d", task.Index, n(t, s, "SELECT last_index FROM project_task_counters WHERE project_id=3"), n(t, s, "SELECT count(*) FROM task_index_aliases WHERE task_id=2"))
		e = s.RestoreTaskUndoState(ctx, after, before)
		t.Logf("undo_err=%v source_counter=%d aliases=%d", e, n(t, s, "SELECT last_index FROM project_task_counters WHERE project_id=2"), n(t, s, "SELECT count(*) FROM task_index_aliases WHERE task_id=2"))
	})
	t.Run("trashed_task", func(t *testing.T) {
		s := open(t)
		exec(t, s, "UPDATE tasks SET deleted_at='2026-10-05 18:00:00' WHERE id=1")
		exec(t, s, "DELETE FROM task_buckets WHERE task_id=1")
		exec(t, s, "DELETE FROM task_positions WHERE task_id=1")
		tasks, _, e := s.ListOverviewTasks(ctx, 1, 200)
		found := false
		for _, task := range tasks {
			if task.ID == 1 {
				found = true
			}
		}
		t.Logf("trash_visible=%v read_err=%v title_write_err=%v", found, e, s.UpdateTaskTitle(ctx, 1, "Edited trashed task"))
		_, e = s.TaskUndoState(ctx, 1)
		t.Logf("trash_undo_snapshot_err=%v", e)
	})
	t.Run("permanent_delete_relations", func(t *testing.T) {
		s := open(t)
		exec(t, s, "INSERT INTO task_index_aliases(project_id,`index`,task_id) VALUES(3,100,2)")
		exec(t, s, "INSERT INTO time_entries(user_id,task_id,start_time,end_time,created,updated) VALUES(1,2,'2026-10-05 12:00:00','2026-10-05 13:00:00','2026-10-05 12:00:00','2026-10-05 13:00:00')")
		e := s.DeleteTask(ctx, 2)
		t.Logf("delete_err=%v tasks_left=%d aliases_left=%d time_entries_left=%d", e, n(t, s, "SELECT count(*) FROM tasks WHERE id=2"), n(t, s, "SELECT count(*) FROM task_index_aliases WHERE task_id=2"), n(t, s, "SELECT count(*) FROM time_entries WHERE task_id=2"))
	})
	t.Run("archived_target", func(t *testing.T) {
		s := open(t)
		exec(t, s, "UPDATE projects SET is_archived=1 WHERE id=3")
		task, e := s.CreateTask(ctx, TaskDraft{ProjectID: 3, Title: "Created in archived Project"})
		t.Logf("archived_create_id=%d err=%v", task.ID, e)
		d := draft(t, s, 2)
		d.ProjectID = 3
		_, e = s.UpdateTask(ctx, d)
		t.Logf("archived_move_err=%v", e)
	})
	t.Run("bucket_limit", func(t *testing.T) {
		s := open(t)
		exec(t, s, "UPDATE buckets SET `limit`=1 WHERE id=7")
		task, e := s.CreateTask(ctx, TaskDraft{ProjectID: 2, Title: "Over capacity"})
		t.Logf("create_id=%d err=%v bucket_tasks=%d limit=%d", task.ID, e, n(t, s, "SELECT count(*) FROM task_buckets WHERE bucket_id=7"), n(t, s, "SELECT `limit` FROM buckets WHERE id=7"))
	})
	t.Run("rollback", func(t *testing.T) {
		s := open(t)
		count := n(t, s, "SELECT count(*) FROM tasks")
		exec(t, s, "CREATE TRIGGER fail_position BEFORE INSERT ON task_positions BEGIN SELECT RAISE(ABORT,'audit position failure'); END")
		_, e := s.CreateTask(ctx, TaskDraft{ProjectID: 2, Title: "Must roll back"})
		t.Logf("create_err=%v tasks_unchanged=%v", e, n(t, s, "SELECT count(*) FROM tasks") == count)
	})
	t.Run("schema_uniqueness", func(t *testing.T) {
		s := open(t)
		_, e := s.db.Exec("INSERT INTO task_positions(task_id,project_view_id,position) SELECT task_id,project_view_id,position FROM task_positions LIMIT 1")
		t.Logf("duplicate_position_err=%v", e)
	})
	t.Run("sibling_sort", func(t *testing.T) {
		s := open(t)
		before := n(t, s, "SELECT count(*) FROM project_ancestors")
		e := s.MoveProjectAmongSiblings(ctx, 2, ProjectMoveNext)
		t.Logf("sort_err=%v ancestor_rows_unchanged=%v null_roots=%d", e, n(t, s, "SELECT count(*) FROM project_ancestors") == before, n(t, s, "SELECT count(*) FROM projects WHERE parent_project_id IS NULL"))
	})
	t.Run("missing_migration_history", func(t *testing.T) {
		s := open(t)
		exec(t, s, "DROP TABLE migration")
		_, e := s.ListProjects(ctx)
		t.Logf("store_without_history_read_err=%v", e)
	})
}
