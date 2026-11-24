package vimkunja

import (
	"context"
	"testing"
)

func TestCreateAdditionalKanbanBoardPreservesSharedTasks(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	if err := s.ToggleTaskDone(ctx, 2, true); err != nil {
		t.Fatal(err)
	}
	before, err := s.TaskUndoState(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := s.CreateKanbanView(ctx, 1, "  Review  ")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "Review" || len(cfg.Buckets) != 3 || cfg.Buckets[0].Title != "To-Do" || cfg.DefaultBucketID != cfg.Buckets[0].ID || cfg.DoneBucketID != cfg.Buckets[2].ID {
		t.Fatalf("configuration = %#v", cfg)
	}
	board, err := s.LoadManualKanbanBoard(ctx, 1, cfg.ViewID)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Buckets[0].Tasks) != 2 || !board.Buckets[0].Tasks[1].Done {
		t.Fatalf("initial cards = %#v", board.Buckets)
	}
	after, err := s.TaskUndoState(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if after.Done != before.Done {
		t.Fatalf("completion changed: %#v -> %#v", before, after)
	}
	old, err := s.LoadManualKanbanBoard(ctx, 1, 11)
	if err != nil || len(old.Buckets[1].Tasks) != 1 {
		t.Fatalf("old Board changed: %#v, %v", old, err)
	}
}

func TestKanbanConfigurationSavesDraftWithoutEditingTasks(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	cfg, err := s.LoadKanbanConfiguration(ctx, 1, 11)
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Title = "Delivery"
	cfg.Position = 77
	cfg.Buckets = append(cfg.Buckets, KanbanBucketConfiguration{ID: -1, Title: "Review", Limit: 3})
	cfg.Buckets[0], cfg.Buckets[2] = cfg.Buckets[2], cfg.Buckets[0]
	saved, err := s.SaveKanbanConfiguration(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Title != "Delivery" || saved.Position != 77 || saved.Buckets[0].Title != "Review" || saved.Buckets[0].ID <= 0 || saved.Buckets[0].Limit != 3 {
		t.Fatalf("saved = %#v", saved)
	}
	got, err := s.LoadKanbanConfiguration(ctx, 1, 11)
	if err != nil || got.Revision != saved.Revision {
		t.Fatalf("reload = %#v, %v", got, err)
	}
	after, err := s.TaskUndoState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if before.Title != after.Title || before.Done != after.Done || before.BucketAssignments[0] != after.BucketAssignments[0] {
		t.Fatalf("Task changed: %#v -> %#v", before, after)
	}
	if _, err = s.SaveKanbanConfiguration(ctx, cfg); err == nil {
		t.Fatal("accepted stale configuration")
	}
}

func TestKanbanBucketDeletionPreservesCompletionAndRollsBackAtCapacity(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	cfg, err := s.LoadKanbanConfiguration(ctx, 1, 11)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Buckets = append(cfg.Buckets, KanbanBucketConfiguration{ID: -1, Title: "Review", Limit: 1})
	cfg, err = s.SaveKanbanConfiguration(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultBucketID = cfg.Buckets[2].ID
	cfg.Buckets = []KanbanBucketConfiguration{cfg.Buckets[1], cfg.Buckets[2]}
	if _, err = s.SaveKanbanConfiguration(ctx, cfg); err == nil {
		t.Fatal("relocation exceeded capacity")
	}
	board, err := s.LoadManualKanbanBoard(ctx, 1, 11)
	if err != nil || len(board.Buckets) != 3 || len(board.Buckets[0].Tasks) != 2 {
		t.Fatalf("failed save changed Board: %#v, %v", board, err)
	}
	cfg.Buckets[1].Limit = 0
	cfg, err = s.SaveKanbanConfiguration(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	board, err = s.LoadManualKanbanBoard(ctx, 1, 11)
	if err != nil || len(board.Buckets) != 2 || len(board.Buckets[1].Tasks) != 2 {
		t.Fatalf("relocation = %#v, %v", board, err)
	}
	for _, task := range board.Buckets[1].Tasks {
		if task.Done {
			t.Fatal("deletion changed Task completion")
		}
	}
}

func TestDeleteKanbanViewKeepsTasksAndOtherViews(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	cfg, err := s.LoadKanbanConfiguration(ctx, 1, 11)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteKanbanView(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	views, _, err := s.ListViews(ctx, 1, 1, 100)
	if err != nil || len(views) != 1 || views[0].ID != 10 {
		t.Fatalf("Views = %#v, %v", views, err)
	}
	tasks, _, err := s.ListTasksInView(ctx, 1, 10, 1, 100)
	if err != nil || len(tasks) != 2 {
		t.Fatalf("Tasks = %#v, %v", tasks, err)
	}
	other, err := s.LoadManualKanbanBoard(ctx, 2, 21)
	if err != nil || len(other.Buckets) != 2 {
		t.Fatalf("other Board = %#v, %v", other, err)
	}
	if err = s.DeleteKanbanView(ctx, cfg); err == nil {
		t.Fatal("deleted stale View twice")
	}
}

func TestKanbanConfigurationRejectsInvalidOrUnsupportedWrites(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	cfg, err := s.LoadKanbanConfiguration(ctx, 1, 11)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*KanbanConfiguration){
		func(c *KanbanConfiguration) { c.DefaultBucketID = 200 },
		func(c *KanbanConfiguration) { c.DoneBucketID = -999 },
		func(c *KanbanConfiguration) { c.Buckets = nil },
		func(c *KanbanConfiguration) { c.Buckets[0].Limit = -1 },
		func(c *KanbanConfiguration) { c.Buckets[0].ID = 200 },
		func(c *KanbanConfiguration) { c.Title = " " },
		func(c *KanbanConfiguration) { c.ProjectID = 2 },
	} {
		draft := cfg
		draft.Buckets = append([]KanbanBucketConfiguration(nil), cfg.Buckets...)
		change(&draft)
		if _, err = s.SaveKanbanConfiguration(ctx, draft); err == nil {
			t.Errorf("invalid draft accepted: %#v", draft)
		}
		got, e := s.LoadKanbanConfiguration(ctx, 1, 11)
		if e != nil || got.Revision != cfg.Revision {
			t.Fatalf("invalid save changed configuration: %#v, %v", got, e)
		}
	}
	compatExec(t, s, `UPDATE project_views SET filter='{"filter":"priority = 3"}' WHERE id=11`)
	if _, err = s.LoadKanbanConfiguration(ctx, 1, 11); err == nil {
		t.Fatal("filtered Board accepted")
	}
	if err = s.DeleteKanbanView(ctx, cfg); err == nil {
		t.Fatal("filtered Board deletion accepted")
	}
	compatExec(t, s, `UPDATE project_views SET filter=NULL WHERE id=11;UPDATE projects SET is_archived=1 WHERE id=1`)
	if _, err = s.SaveKanbanConfiguration(ctx, cfg); err == nil {
		t.Fatal("archived save accepted")
	}
	if err = s.DeleteKanbanView(ctx, cfg); err == nil {
		t.Fatal("archived deletion accepted")
	}
	if _, err = s.CreateKanbanView(ctx, 1, "Blocked"); err == nil {
		t.Fatal("archived creation accepted")
	}
}

func TestKanbanCardCompletionPropagatesAcrossBoards(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	cfg, err := s.CreateKanbanView(ctx, 1, "Review")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetTaskWorkflowStatus(ctx, 1, 101, 11); err != nil {
		t.Fatal(err)
	}
	review, err := s.LoadManualKanbanBoard(ctx, 1, cfg.ViewID)
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Buckets[2].Tasks) != 1 || review.Buckets[2].Tasks[0].ID != 1 {
		t.Fatalf("completion not propagated: %#v", review)
	}
	// A completed card initially in To-Do on a newly created Board stays completed
	// when moved between ordinary Buckets; only leaving its Done Bucket reopens it.
	another, err := s.CreateKanbanView(ctx, 1, "Another")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetTaskWorkflowStatus(ctx, 1, another.Buckets[1].ID, another.ViewID); err != nil {
		t.Fatal(err)
	}
	state, err := s.TaskUndoState(ctx, 1)
	if err != nil || !state.Done {
		t.Fatalf("ordinary move reopened Task: %#v, %v", state, err)
	}
}

func TestFilteredKanbanViewsAreIdentifiedAndCannotBeOpenedAsUnfiltered(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	compatExec(t, s, `UPDATE project_views SET filter='{"filter":"priority = 3"}' WHERE id=11`)
	views, _, err := s.ListViews(ctx, 1, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	var identified bool
	for _, v := range views {
		if v.ID == 11 {
			identified = v.Filtered
		}
	}
	if !identified {
		t.Fatal("filtered View not identified")
	}
	if _, err = s.LoadManualKanbanBoard(ctx, 1, 11); err == nil {
		t.Fatal("filtered Board opened as unfiltered")
	}
}

func TestFilteredKanbanCannotSupplyWorkflowChoices(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	compatExec(t, s, `UPDATE project_views SET filter='{"filter":"priority = 3"}' WHERE id=11`)
	status, err := s.GetTaskWorkflowStatus(ctx, 1, 1, 11)
	if err != nil {
		t.Fatal(err)
	}
	if status.Availability != WorkflowStatusUnavailable {
		t.Fatalf("filtered workflow offered: %#v", status)
	}
	if err = s.SetTaskWorkflowStatus(ctx, 1, 101, 11); err == nil {
		t.Fatal("filtered Board card move accepted")
	}
}

func TestAutomaticKanbanProvisionPreservesExistingFilteredBoards(t *testing.T) {
	for _, setup := range []string{
		`UPDATE project_views SET filter='{"filter":"priority = 3"}' WHERE id=11`,
		`UPDATE project_views SET bucket_configuration_mode=2 WHERE id=11`,
	} {
		t.Run(setup, func(t *testing.T) {
			s := setupVikunja270(t)
			compatExec(t, s, setup)
			if _, err := s.ProvisionStandardManualKanbanView(context.Background(), 1); err == nil {
				t.Fatal("created board alongside existing filtered board")
			}
			var count int
			if err := s.db.QueryRow(`SELECT COUNT(*) FROM project_views WHERE project_id=1 AND view_kind=3`).Scan(&count); err != nil || count != 1 {
				t.Fatalf("kanban views = %d, err = %v", count, err)
			}
		})
	}
}
