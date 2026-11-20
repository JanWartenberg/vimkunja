package vimkunja

import (
	"context"
	"strings"
	"testing"
)

func TestDeleteEmptyProjectPreservesContentAndRemovesOnlyLeaf(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	root, err := s.CreateProject(ctx, ProjectDraft{Title: "Delete root"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := s.CreateProject(ctx, ProjectDraft{Title: "Delete child", ParentProjectID: root.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CheckProjectDeletion(ctx, root.ID); err == nil || !strings.Contains(err.Error(), "Subprojects") {
		t.Fatalf("parent preflight: %v", err)
	}
	if err := s.DeleteEmptyProject(ctx, root.ID); err == nil || !strings.Contains(err.Error(), "Subprojects") {
		t.Fatalf("parent deletion: %v", err)
	}
	task, err := s.CreateTask(ctx, TaskDraft{Title: "Keep", ProjectID: child.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"active", "done", "trash"} {
		switch state {
		case "done":
			compatExec(t, s, `UPDATE tasks SET done=1 WHERE id=?`, task.ID)
		case "trash":
			compatExec(t, s, `UPDATE tasks SET deleted_at=CURRENT_TIMESTAMP WHERE id=?`, task.ID)
		}
		if err := s.CheckProjectDeletion(ctx, child.ID); err == nil || !strings.Contains(err.Error(), "Tasks") {
			t.Fatalf("%s Task preflight: %v", state, err)
		}
		if err := s.DeleteEmptyProject(ctx, child.ID); err == nil || !strings.Contains(err.Error(), "Tasks") {
			t.Fatalf("%s Task did not prevent deletion: %v", state, err)
		}
	}
	compatExec(t, s, `UPDATE tasks SET deleted_at=NULL WHERE id=?`, task.ID)
	if err := s.DeleteTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteEmptyProject(ctx, child.ID); err != nil {
		t.Fatal(err)
	}
	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	foundRoot := false
	for _, p := range projects {
		if p.ID == child.ID {
			t.Fatal("deleted child is still readable")
		}
		if p.ID == root.ID {
			foundRoot = true
		}
	}
	if !foundRoot {
		t.Fatal("deleting child removed root")
	}
	if err := s.DeleteEmptyProject(ctx, root.ID); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteEmptyProjectCleansMetadataAtomically(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	p, err := s.CreateProject(ctx, ProjectDraft{Title: "Metadata"})
	if err != nil {
		t.Fatal(err)
	}
	compatExec(t, s, `INSERT INTO favorites(entity_id,user_id,kind) VALUES(?,1,2),(?,1,1)`, p.ID, p.ID)
	compatExec(t, s, `INSERT INTO subscriptions(entity_id,user_id,entity_type,created) VALUES(?,1,2,CURRENT_TIMESTAMP),(?,1,3,CURRENT_TIMESTAMP)`, p.ID, p.ID)
	compatExec(t, s, `UPDATE users SET default_project_id=? WHERE id=1`, p.ID)
	compatExec(t, s, `INSERT INTO notifications(notifiable_id,notification,name,project_id,created) VALUES(1,'{}','test',?,CURRENT_TIMESTAMP)`, p.ID)
	compatExec(t, s, `INSERT INTO webhooks(target_url,events,project_id,created_by_id,created,updated) VALUES('https://example.invalid','[]',?,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, p.ID)
	compatExec(t, s, `CREATE TRIGGER fail_project_delete BEFORE DELETE ON projects BEGIN SELECT RAISE(ABORT,'injected'); END`)
	if err := s.DeleteEmptyProject(ctx, p.ID); err == nil {
		t.Fatal("injected failure succeeded")
	}
	views, _, err := s.ListViews(ctx, p.ID, 1, 100)
	if err != nil || len(views) != 4 {
		t.Fatalf("rollback lost Views: %v %v", views, err)
	}
	board, err := s.LoadManualKanbanBoard(ctx, p.ID)
	if err != nil || len(board.Buckets) != 3 {
		t.Fatalf("rollback lost Buckets: %v %v", board, err)
	}
	var defaultID int64
	if err := s.db.QueryRow(`SELECT default_project_id FROM users WHERE id=1`).Scan(&defaultID); err != nil || defaultID != p.ID {
		t.Fatalf("rollback changed default: %d %v", defaultID, err)
	}
	compatExec(t, s, `DROP TRIGGER fail_project_delete`)
	if err := s.DeleteEmptyProject(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`SELECT COUNT(*) FROM project_views WHERE project_id=?`,
		`SELECT COUNT(*) FROM project_ancestors WHERE project_id=? OR ancestor_id=?`,
		`SELECT COUNT(*) FROM project_task_counters WHERE project_id=?`,
		`SELECT COUNT(*) FROM favorites WHERE entity_id=? AND kind=2`,
		`SELECT COUNT(*) FROM subscriptions WHERE entity_id=? AND entity_type=2`,
		`SELECT COUNT(*) FROM notifications WHERE project_id=?`,
		`SELECT COUNT(*) FROM webhooks WHERE project_id=?`,
	} {
		args := []any{p.ID}
		if strings.Count(query, "?") == 2 {
			args = append(args, p.ID)
		}
		var count int
		if err := s.db.QueryRow(query, args...).Scan(&count); err != nil || count != 0 {
			t.Fatalf("metadata remains: %s: %d %v", query, count, err)
		}
	}
	for _, query := range []string{`SELECT COUNT(*) FROM favorites WHERE entity_id=? AND kind=1`, `SELECT COUNT(*) FROM subscriptions WHERE entity_id=? AND entity_type=3`} {
		var count int
		if err := s.db.QueryRow(query, p.ID).Scan(&count); err != nil || count != 1 {
			t.Fatalf("unrelated entity removed: %s: %d %v", query, count, err)
		}
	}
	if err := s.db.QueryRow(`SELECT default_project_id FROM users WHERE id=1`).Scan(&defaultID); err != nil || defaultID != 0 {
		t.Fatalf("default not cleared: %d %v", defaultID, err)
	}
	if _, err := s.CreateProject(ctx, ProjectDraft{Title: "Metadata"}); err != nil {
		t.Fatalf("cannot reuse deleted Title: %v", err)
	}
}

func TestDeleteEmptyProjectRejectsUnsafeTargets(t *testing.T) {
	for _, condition := range []string{"archived", "time", "background", "ancestry", "missing", "invalid"} {
		t.Run(condition, func(t *testing.T) {
			s := setupVikunja270(t)
			ctx := context.Background()
			p, err := s.CreateProject(ctx, ProjectDraft{Title: "Guarded"})
			if err != nil {
				t.Fatal(err)
			}
			id := p.ID
			switch condition {
			case "archived":
				compatExec(t, s, `UPDATE projects SET is_archived=1 WHERE id=?`, id)
			case "time":
				compatExec(t, s, `INSERT INTO time_entries(user_id,project_id,start_time,end_time,created,updated) VALUES(1,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id)
			case "background":
				compatExec(t, s, `UPDATE projects SET background_file_id=99 WHERE id=?`, id)
			case "ancestry":
				compatExec(t, s, `INSERT INTO project_ancestors(ancestor_id,project_id,depth) VALUES(?,999,1)`, id)
			case "missing":
				id = 999
			case "invalid":
				id = 0
			}
			if err := s.CheckProjectDeletion(ctx, id); err == nil {
				t.Fatal("unsafe preflight accepted")
			}
			if err := s.DeleteEmptyProject(ctx, id); err == nil {
				t.Fatal("unsafe deletion accepted")
			}
			views, _, err := s.ListViews(ctx, p.ID, 1, 100)
			if err != nil || len(views) != 4 {
				t.Fatalf("rejection changed Views: %v %v", views, err)
			}
		})
	}
}

func TestDeleteEmptyProjectRollsBackWhenDeletionIsIgnored(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	p, err := s.CreateProject(ctx, ProjectDraft{Title: "Ignored"})
	if err != nil {
		t.Fatal(err)
	}
	compatExec(t, s, `CREATE TRIGGER ignore_project_delete BEFORE DELETE ON projects BEGIN SELECT RAISE(IGNORE); END`)
	if err := s.DeleteEmptyProject(ctx, p.ID); err == nil {
		t.Fatal("ignored deletion reported success")
	}
	views, _, err := s.ListViews(ctx, p.ID, 1, 100)
	if err != nil || len(views) != 4 {
		t.Fatalf("ignored deletion lost metadata: %v %v", views, err)
	}
}

func TestProjectDeletionPreflightDoesNotWriteAndRevalidatesOnDelete(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	p, err := s.CreateProject(ctx, ProjectDraft{Title: "Check only"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CheckProjectDeletion(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	views, _, err := s.ListViews(ctx, p.ID, 1, 100)
	if err != nil || len(views) != 4 {
		t.Fatalf("preflight changed Views: %v %v", views, err)
	}
	board, err := s.LoadManualKanbanBoard(ctx, p.ID)
	if err != nil || len(board.Buckets) != 3 {
		t.Fatalf("preflight changed Buckets: %v %v", board, err)
	}
	if _, err := s.CreateTask(ctx, TaskDraft{Title: "Added after check", ProjectID: p.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteEmptyProject(ctx, p.ID); err == nil || !strings.Contains(err.Error(), "Tasks") {
		t.Fatalf("stale preflight allowed deletion: %v", err)
	}
}
