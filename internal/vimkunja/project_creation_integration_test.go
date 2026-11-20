//go:build integration

package vimkunja

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
)

func TestProjectCreationOriginalAppRoundtrip(t *testing.T) {
	root := t.TempDir()
	server := starterServer(t, root)
	server.start()
	server.request(http.MethodPost, "/register", map[string]any{"username": server.username, "email": "project@example.invalid", "password": server.password}, http.StatusOK, nil)
	server.stop()
	s, err := NewVerifiedSQLiteStore(filepath.Join(root, "vikunja.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	parent, err := s.CreateProject(ctx, ProjectDraft{Title: "New root"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := s.CreateProject(ctx, ProjectDraft{Title: "New child", ParentProjectID: parent.ID})
	if err != nil {
		t.Fatal(err)
	}
	grandchild, err := s.CreateProject(ctx, ProjectDraft{Title: "Nested", ParentProjectID: child.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	server.start()
	server.login()
	for _, want := range []Project{parent, child, grandchild} {
		var got struct {
			ID     int64  `json:"id"`
			Title  string `json:"title"`
			Parent int64  `json:"parent_project_id"`
			Owner  struct {
				ID int64 `json:"id"`
			} `json:"owner"`
			Views []struct {
				ID      int64  `json:"id"`
				Kind    string `json:"view_kind"`
				Default int64  `json:"default_bucket_id"`
				Done    int64  `json:"done_bucket_id"`
			} `json:"views"`
		}
		server.request(http.MethodGet, fmt.Sprintf("/projects/%d", want.ID), nil, http.StatusOK, &got)
		if got.Title != want.Title || got.Parent != want.ParentProjectID || got.Owner.ID != 1 || len(got.Views) != 4 {
			t.Fatalf("VOA Project = %#v, want %#v", got, want)
		}
		for _, view := range got.Views {
			if view.Kind == "kanban" {
				var buckets []struct {
					Title string `json:"title"`
				}
				server.request(http.MethodGet, fmt.Sprintf("/projects/%d/views/%d/buckets", want.ID, view.ID), nil, http.StatusOK, &buckets)
				if len(buckets) != 3 || buckets[0].Title != "To-Do" || buckets[2].Title != "Done" || view.Default == 0 || view.Done == 0 {
					t.Fatalf("VOA Buckets = %#v, View %#v", buckets, view)
				}
			}
		}
	}
	var imported struct {
		ID int64 `json:"id"`
	}
	server.request(http.MethodPut, "/projects", map[string]any{"title": "New root", "parent_project_id": grandchild.ID}, http.StatusCreated, &imported)
	server.stop()
	s, err = NewVerifiedSQLiteStore(filepath.Join(root, "vikunja.db"))
	if err != nil {
		t.Fatal(err)
	}
	parents, err := s.ProjectCreationParents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range parents {
		if p.ID == imported.ID {
			found = p.Path == "New root / New child / Nested / New root"
		}
	}
	if !found {
		t.Fatalf("VOA-created hierarchy not readable: %#v", parents)
	}
	if _, err := s.CreateProject(ctx, ProjectDraft{Title: "NEW ROOT"}); err == nil {
		t.Fatal("new duplicate Title accepted")
	}
}

func TestProjectDeletionOriginalAppRoundtrip(t *testing.T) {
	root := t.TempDir()
	server := starterServer(t, root)
	server.start()
	server.request(http.MethodPost, "/register", map[string]any{"username": server.username, "email": "delete@example.invalid", "password": server.password}, http.StatusOK, nil)
	server.login()
	var parent, child struct {
		ID int64 `json:"id"`
	}
	server.request(http.MethodPut, "/projects", map[string]any{"title": "Keep root"}, http.StatusCreated, &parent)
	server.request(http.MethodPut, "/projects", map[string]any{"title": "Empty child", "parent_project_id": parent.ID, "is_favorite": true}, http.StatusCreated, &child)
	server.stop()
	s, err := NewVerifiedSQLiteStore(filepath.Join(root, "vikunja.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	if err := s.CheckProjectDeletion(ctx, parent.ID); err == nil {
		t.Fatal("preflight accepted parent with child")
	}
	if err := s.CheckProjectDeletion(ctx, child.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteEmptyProject(ctx, parent.ID); err == nil {
		t.Fatal("accepted parent with child")
	}
	if err := s.DeleteEmptyProject(ctx, child.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	server.start()
	server.login()
	server.request(http.MethodGet, fmt.Sprintf("/projects/%d", child.ID), nil, http.StatusNotFound, nil)
	var got struct {
		ID    int64 `json:"id"`
		Views []struct {
			ID int64 `json:"id"`
		} `json:"views"`
	}
	server.request(http.MethodGet, fmt.Sprintf("/projects/%d", parent.ID), nil, http.StatusOK, &got)
	if got.ID != parent.ID || len(got.Views) != 4 {
		t.Fatalf("surviving Project damaged: %#v", got)
	}
	server.request(http.MethodPut, "/projects", map[string]any{"title": "Empty child", "parent_project_id": parent.ID}, http.StatusCreated, nil)
}
