//go:build integration

package vimkunja

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
)

func TestKanbanManagementOriginalAppRoundtrip(t *testing.T) {
	server := starterServer(t, t.TempDir())
	server.start()
	server.request(http.MethodPost, "/register", map[string]any{"username": server.username, "email": "kanban@example.invalid", "password": server.password}, http.StatusOK, nil)
	server.login()
	var p struct {
		ID    int64 `json:"id"`
		Views []struct {
			ID   int64  `json:"id"`
			Kind string `json:"view_kind"`
		} `json:"views"`
	}
	server.request(http.MethodPut, "/projects", map[string]any{"title": "Kanban roundtrip"}, http.StatusCreated, &p)
	var originalViewID int64
	for _, v := range p.Views {
		if v.Kind == "kanban" {
			originalViewID = v.ID
		}
	}
	if originalViewID == 0 {
		t.Fatal("missing original Board")
	}
	var task struct {
		ID int64 `json:"id"`
	}
	server.request(http.MethodPut, fmt.Sprintf("/projects/%d/tasks", p.ID), map[string]any{"title": "Shared card"}, http.StatusCreated, &task)
	server.stop()
	path := filepath.Join(server.root, "vikunja.db")
	s, err := NewVerifiedSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	cfg, err := s.CreateKanbanView(ctx, p.ID, "Review")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Title = "Review renamed"
	cfg.Buckets = append(cfg.Buckets, KanbanBucketConfiguration{ID: -1, Title: "QA", Limit: 2})
	cfg, err = s.SaveKanbanConfiguration(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	server.start()
	server.login()
	var got struct {
		Title   string `json:"title"`
		Default int64  `json:"default_bucket_id"`
		Done    int64  `json:"done_bucket_id"`
	}
	server.request(http.MethodGet, fmt.Sprintf("/projects/%d/views/%d", p.ID, cfg.ViewID), nil, http.StatusOK, &got)
	if got.Title != "Review renamed" || got.Default != cfg.DefaultBucketID || got.Done != cfg.DoneBucketID {
		t.Fatalf("VOA configuration = %#v", got)
	}
	var buckets []struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
		Limit int64  `json:"limit"`
	}
	server.request(http.MethodGet, fmt.Sprintf("/projects/%d/views/%d/buckets", p.ID, cfg.ViewID), nil, http.StatusOK, &buckets)
	if len(buckets) != 4 || buckets[3].Title != "QA" || buckets[3].Limit != 2 {
		t.Fatalf("VOA Buckets = %#v", buckets)
	}
	server.request(http.MethodPost, fmt.Sprintf("/projects/%d/views/%d/buckets/%d/tasks", p.ID, cfg.ViewID, cfg.DoneBucketID), map[string]any{"task_id": task.ID}, http.StatusOK, nil)
	server.stop()
	s, err = NewVerifiedSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	original, err := s.LoadManualKanbanBoard(ctx, p.ID, originalViewID)
	if err != nil {
		t.Fatal(err)
	}
	if len(original.Buckets[2].Tasks) != 1 || !original.Buckets[2].Tasks[0].Done {
		t.Fatalf("VOA did not propagate Done: %#v", original)
	}
	cfg, err = s.LoadKanbanConfiguration(ctx, p.ID, cfg.ViewID)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultBucketID = cfg.Buckets[3].ID
	cfg.Buckets = cfg.Buckets[1:]
	cfg, err = s.SaveKanbanConfiguration(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteKanbanView(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	server.start()
	server.login()
	server.request(http.MethodGet, fmt.Sprintf("/projects/%d/views/%d", p.ID, cfg.ViewID), nil, http.StatusNotFound, nil)
	var retained struct {
		Title string `json:"title"`
		Done  bool   `json:"done"`
	}
	server.request(http.MethodGet, fmt.Sprintf("/tasks/%d", task.ID), nil, http.StatusOK, &retained)
	if retained.Title != "Shared card" || !retained.Done {
		t.Fatalf("Task damaged by deletion: %#v", retained)
	}
	server.request(http.MethodGet, fmt.Sprintf("/projects/%d/views/%d", p.ID, originalViewID), nil, http.StatusOK, nil)
}
