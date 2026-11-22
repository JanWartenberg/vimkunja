package tutorial

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/tui"
)

func TestKanbanManagementThroughRealTUIAndSQLite(t *testing.T) {
	db, err := NewDemoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &practiceSession{t: t, db: db, app: tui.NewModel(tui.Config{Store: db.Store, PerPage: 100})}
	s.run(s.app.Init())
	s.message(tea.WindowSizeMsg{Width: 80, Height: 24})
	ctx := context.Background()
	before, _, err := db.Store.ListTasksInView(ctx, 1, 11, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	s.command("kanban new")
	s.keys("i")
	s.text("Experiment")
	s.keys("esc")
	s.command("w")
	if !strings.Contains(s.screen(), "BOARD CONFIGURATION") {
		t.Fatalf("new Board did not open its Bucket editor: %s", s.screen())
	}
	s.command("q")
	views, _, err := db.Store.ListViews(ctx, 1, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	for _, v := range views {
		if v.Title == "Experiment" {
			id = v.ID
		}
	}
	if id == 0 || !strings.Contains(s.screen(), "Experiment") {
		t.Fatalf("new Board not opened: %s", s.screen())
	}
	s.command("kanban switch")
	s.keys("esc")
	if !strings.Contains(s.screen(), "Experiment") {
		t.Fatal("cancel changed Board")
	}
	s.command("kanban manage")
	s.keys("G", "e")
	if !strings.Contains(s.screen(), "BOARD CONFIGURATION") {
		t.Fatalf("configuration missing: %s", s.screen())
	}
	s.keys("A")
	s.text(" edited")
	s.keys("esc", "o", "i")
	s.text("QA")
	s.keys("esc", "space", "d")
	s.command("w")
	cfg, err := db.Store.LoadKanbanConfiguration(ctx, 1, id)
	if err != nil || cfg.Title != "Experiment edited" || len(cfg.Buckets) != 4 || cfg.DefaultBucketID != cfg.Buckets[3].ID {
		t.Fatalf("configuration not saved: %#v %v screen %s", cfg, err, s.screen())
	}
	s.keys("k", "k", "k", "k", "k", "k", "space", "D")
	if !strings.Contains(s.screen(), "CONFIRM BUCKET") {
		t.Fatalf("Bucket deletion missing: %s", s.screen())
	}
	s.keys("y")
	s.command("w")
	cfg, err = db.Store.LoadKanbanConfiguration(ctx, 1, id)
	if err != nil || len(cfg.Buckets) != 3 {
		t.Fatalf("Bucket not deleted: %#v, %v", cfg, err)
	}
	board, err := db.Store.LoadManualKanbanBoard(ctx, 1, id)
	if err != nil || len(board.Buckets[2].Tasks) != len(before) {
		t.Fatalf("Tasks not relocated: %#v, %v", board, err)
	}
	s.command("q")
	if !strings.Contains(s.screen(), "Experiment edited") {
		t.Fatalf("saved configuration not reflected: %s", s.screen())
	}
	s.command("kanban manage")
	s.keys("G", "d", "n")
	if !strings.Contains(s.screen(), "BOARDS") {
		t.Fatal("cancel did not retain manager")
	}
	s.keys("d", "y")
	views, _, err = db.Store.ListViews(ctx, 1, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.ID == id {
			t.Fatal("Board not deleted")
		}
	}
	after, _, err := db.Store.ListTasksInView(ctx, 1, 11, 1, 100)
	if err != nil || len(after) != len(before) {
		t.Fatalf("deleting Board changed Tasks: %#v, %v", after, err)
	}
}
