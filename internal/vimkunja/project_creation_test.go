package vimkunja

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestCreateProjectProvidesStandardViewsAndNestedProjects(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	root, err := s.CreateProject(ctx, ProjectDraft{Title: "  Work  "})
	if err != nil {
		t.Fatal(err)
	}
	if root.Title != "Work" || root.ParentProjectID != 0 {
		t.Fatalf("Project = %#v", root)
	}
	child, err := s.CreateProject(ctx, ProjectDraft{Title: "Notes", ParentProjectID: root.ID})
	if err != nil {
		t.Fatal(err)
	}
	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range projects {
		if p.ID == child.ID {
			found = p.ParentProjectID == root.ID && p.Title == "Notes"
		}
	}
	if !found {
		t.Fatalf("Subproject not readable: %#v", projects)
	}
	views, _, err := s.ListViews(ctx, root.ID, 1, 100)
	if err != nil || len(views) != 4 {
		t.Fatalf("Views = %#v, %v", views, err)
	}
	for i, kind := range []string{"list", "gantt", "table", "kanban"} {
		if views[i].ViewKind != kind {
			t.Fatalf("View %d = %#v", i, views[i])
		}
	}
	board, err := s.LoadManualKanbanBoard(ctx, child.ID)
	if err != nil || len(board.Buckets) != 3 || board.Buckets[0].Title != "To-Do" || board.Buckets[2].Title != "Done" {
		t.Fatalf("Board = %#v, %v", board, err)
	}
	task, err := s.CreateTask(ctx, TaskDraft{ProjectID: child.ID, Title: "First"})
	if err != nil || task.Index != 1 {
		t.Fatalf("Task = %#v, %v", task, err)
	}
}

func TestCreateProjectRejectsNewDuplicatesAndInvalidTitles(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	compatExec(t, s, `INSERT INTO projects(id,title,owner_id,is_archived,created,updated) VALUES(90,' Archive ',1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),(91,'One',1,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	before, _ := s.ListProjects(ctx)
	for _, title := range []string{"one", " ONE ", "archive", "\t ", strings.Repeat("界", 251)} {
		if _, err := s.CreateProject(ctx, ProjectDraft{Title: title}); err == nil {
			t.Errorf("accepted %q", title)
		}
	}
	after, _ := s.ListProjects(ctx)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("rejected creation changed Projects: %#v", after)
	}
	if _, err := s.CreateProject(ctx, ProjectDraft{Title: strings.Repeat("界", 250)}); err != nil {
		t.Fatal(err)
	}
}

func TestProjectCreationParentsAndRevalidation(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	compatExec(t, s, `INSERT INTO projects(id,title,parent_project_id,owner_id,is_archived,created,updated) VALUES
 (0,'Invalid ID',NULL,1,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
 (10,'Archived',NULL,1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
 (11,'Hidden child',10,1,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
 (12,'Orphan',999,1,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
 (13,'Self',13,1,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
 (14,'Cycle',15,1,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
 (15,'Cycle',14,1,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
 (16,'Negative',-1,1,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
 (17,'Child',1,1,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);`)
	parents, err := s.ProjectCreationParents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[int64]bool{}
	for _, p := range parents {
		ids[p.ID] = true
		if p.ID == 17 && p.Path != "One / Child" {
			t.Fatalf("path = %q", p.Path)
		}
	}
	if !reflect.DeepEqual(ids, map[int64]bool{1: true, 2: true, 17: true}) {
		t.Fatalf("eligible Parents = %#v", parents)
	}
	for _, id := range []int64{10, 11, 12, 13, 14, 15, 16, 999, -1} {
		if _, err := s.CreateProject(ctx, ProjectDraft{Title: fmt.Sprintf("New %d", id), ParentProjectID: id}); err == nil {
			t.Errorf("accepted Parent %d", id)
		}
	}
	// A Parent shown earlier can be archived by VOA before Create is submitted.
	compatExec(t, s, `UPDATE projects SET is_archived=1 WHERE id=1`)
	if _, err := s.CreateProject(ctx, ProjectDraft{Title: "Stale", ParentProjectID: 17}); err == nil {
		t.Fatal("accepted a Parent whose ancestor was archived")
	}
}

func TestCreateProjectRollsBackAndRequiresOneUser(t *testing.T) {
	ctx := context.Background()
	for _, fault := range []string{"bucket failure", "multiple users", "no users"} {
		t.Run(fault, func(t *testing.T) {
			s := setupVikunja270(t)
			before, _ := s.ListProjects(ctx)
			switch fault {
			case "bucket failure":
				compatExec(t, s, `CREATE TRIGGER reject_new_bucket BEFORE INSERT ON buckets BEGIN SELECT RAISE(ABORT,'injected'); END`)
			case "multiple users":
				compatExec(t, s, `INSERT INTO users(id,username,created,updated) VALUES(2,'other',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
			case "no users":
				compatExec(t, s, `DELETE FROM users`)
			}
			if _, err := s.CreateProject(ctx, ProjectDraft{Title: "Failed"}); err == nil {
				t.Fatal("creation should fail")
			}
			after, _ := s.ListProjects(ctx)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("partial creation: %#v", after)
			}
			if fault == "bucket failure" {
				compatExec(t, s, `DROP TRIGGER reject_new_bucket`)
				p, err := s.CreateProject(ctx, ProjectDraft{Title: "Failed"})
				if err != nil {
					t.Fatal(err)
				}
				views, _, err := s.ListViews(ctx, p.ID, 1, 100)
				if err != nil || len(views) != 4 {
					t.Fatalf("Views after retry: %#v, %v", views, err)
				}
			}
		})
	}
}

func TestCreateProjectAppendsAndSerializesDuplicateValidation(t *testing.T) {
	s := setupVikunja270(t)
	ctx := context.Background()
	compatExec(t, s, `UPDATE projects SET position=1000000 WHERE id=2`)
	p, err := s.CreateProject(ctx, ProjectDraft{Title: "Last"})
	if err != nil || p.Position <= 1000000 {
		t.Fatalf("append: %#v, %v", p, err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.CreateProject(ctx, ProjectDraft{Title: "Concurrent"})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful duplicate creations: %d", successes)
	}
	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, p := range projects {
		if p.Title == "Concurrent" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("persisted duplicates: %d", count)
	}
}
