package tutorial

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vimkunja/vimkunja/internal/vimkunja"
)

func TestDemoDatabaseIsFreshMutableAndDisposable(t *testing.T) {
	database, err := NewDemoDatabase()
	if err != nil {
		t.Fatalf("NewDemoDatabase: %v", err)
	}
	path := database.Path
	if filepath.Base(path) == "vikunja.db" || !strings.HasPrefix(filepath.Base(path), "vimkunja-tutorial-") {
		t.Fatalf("tutorial path = %q, want a dedicated temporary database", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("temporary database does not exist: %v", err)
	}

	projects, err := database.Store.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 3 || projects[0].Title != "Learn Vimkunja" {
		t.Fatalf("projects = %#v, want deterministic tutorial fixtures", projects)
	}
	tasks, _, err := database.Store.ListTasksInView(context.Background(), 1, 11, 1, 100)
	if err != nil {
		t.Fatalf("ListTasksInView: %v", err)
	}
	if len(tasks) == 0 || !strings.Contains(tasks[0].Description, "description.\nEvery") || strings.Contains(tasks[0].Description, `\n`) {
		t.Fatalf("tutorial description = %q, want real line breaks", tasks[0].Description)
	}
	board, err := database.Store.LoadManualKanbanBoard(context.Background(), 1)
	if err != nil {
		t.Fatalf("LoadManualKanbanBoard: %v", err)
	}
	if board.Availability != vimkunja.KanbanBoardAvailable || len(board.Buckets) != 3 {
		t.Fatalf("board = %#v, want a three-Bucket manual Board", board)
	}
	if _, err := database.Store.CreateTask(context.Background(), vimkunja.TaskDraft{ProjectID: 1, Title: "Disposable write"}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if err := database.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("database remains after Close: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestEveryDemoDatabaseStartsFromTheSeed(t *testing.T) {
	first, err := NewDemoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Store.CreateTask(context.Background(), vimkunja.TaskDraft{ProjectID: 1, Title: "Only in first run"}); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := NewDemoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	tasks, _, err := second.Store.ListTasksInView(context.Background(), 1, 11, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 8 {
		t.Fatalf("fresh task count = %d, want 8", len(tasks))
	}
}
