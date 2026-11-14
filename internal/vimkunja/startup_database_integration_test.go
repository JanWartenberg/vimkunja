//go:build integration

package vimkunja

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func starterServer(t *testing.T, root string) *roundtripServer {
	t.Helper()
	binary := os.Getenv("VIMKUNJA_VOA_BINARY")
	if binary == "" {
		t.Skip("set VIMKUNJA_VOA_BINARY to the official Vikunja 2.7.0 server")
	}
	binary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	return &roundtripServer{t: t, binary: binary, root: root,
		username: "local", password: "starter-roundtrip-" + rand.Text(),
		client: &http.Client{Timeout: 5 * time.Second}}
}

// TestGenerateStarterDatabase is an explicit maintainer tool. It generates a
// template exclusively from a fresh official server, never a user database.
func TestGenerateStarterDatabase(t *testing.T) {
	output := os.Getenv("VIMKUNJA_STARTER_OUTPUT")
	if output == "" {
		t.Skip("set VIMKUNJA_STARTER_OUTPUT to regenerate the compressed starter")
	}
	server := starterServer(t, t.TempDir())
	server.start()
	server.request(http.MethodPost, "/register", map[string]any{
		"username": server.username, "email": "local@example.invalid", "password": server.password,
	}, http.StatusOK, nil)
	server.login()
	server.stop()
	path := filepath.Join(server.root, "vikunja.db")
	store, err := NewVerifiedSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	// These are synthetic starter records and volatile runtime data, not schema
	// changes. VACUUM removes the discarded session and password from free pages.
	compatExec(t, store, `
		UPDATE users SET password='', email='', pending_email='', name='Local User', timezone='UTC',
			email_reminders_enabled=0, overdue_tasks_reminders_enabled=0;
		DELETE FROM sessions;
		DELETE FROM license_status;
		DELETE FROM saved_filters;
		DELETE FROM buckets WHERE project_view_id IN (SELECT id FROM project_views WHERE project_id < 0);
		DELETE FROM project_views WHERE project_id < 0;
	`)
	for _, table := range []string{"users", "projects", "project_views", "buckets"} {
		compatExec(t, store, "UPDATE "+table+" SET created='2026-01-01 00:00:00', updated='2026-01-01 00:00:00'")
	}
	compatExec(t, store, "PRAGMA wal_checkpoint(TRUNCATE)")
	compatExec(t, store, "PRAGMA journal_mode=DELETE")
	compatExec(t, store, "VACUUM")
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, compressed.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("Starter SQLite SHA256: %x; %d bytes (%d compressed)", sha256.Sum256(data), len(data), compressed.Len())
}

func TestStarterDatabaseOriginalAppRoundtrip(t *testing.T) {
	root := t.TempDir()
	server := starterServer(t, root)
	path := filepath.Join(root, "vikunja.db")
	if _, err := installStarterDatabase(path); err != nil {
		t.Fatal(err)
	}
	store, err := NewVerifiedSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	task, err := store.CreateTask(ctx, TaskDraft{ProjectID: 1, Title: "Created from starter"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	// The distributed local User intentionally has no usable password. Set one
	// with the original application's CLI, exactly as documented for users.
	cmd := exec.Command(server.binary, "user", "reset-password", "1", "--direct", "--password", server.password)
	cmd.Dir = root
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "VIKUNJA_") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env,
		"VIKUNJA_DATABASE_TYPE=sqlite", "VIKUNJA_DATABASE_PATH="+path,
		"VIKUNJA_SERVICE_ROOTPATH="+root, "VIKUNJA_MAILER_ENABLED=false",
		"VIKUNJA_SERVICE_PUBLICURL=http://127.0.0.1:3456/")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("set local User password: %v\n%s", err, output)
	}
	server.start()
	server.login()
	var projects []struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
		Views []struct {
			Kind string `json:"view_kind"`
		} `json:"views"`
	}
	server.request(http.MethodGet, "/projects", nil, http.StatusOK, &projects)
	if len(projects) != 1 || projects[0].Title != "Inbox" || len(projects[0].Views) != 4 {
		t.Fatalf("original app starter Projects = %#v", projects)
	}
	wantKinds := map[string]bool{"list": true, "gantt": true, "table": true, "kanban": true}
	for _, view := range projects[0].Views {
		if !wantKinds[view.Kind] {
			t.Fatalf("unexpected or duplicate starter View kind %q", view.Kind)
		}
		delete(wantKinds, view.Kind)
	}
	var readTask struct {
		Title string `json:"title"`
	}
	server.request(http.MethodGet, fmt.Sprintf("/tasks/%d", task.ID), nil, http.StatusOK, &readTask)
	if readTask.Title != task.Title {
		t.Fatalf("original app Task = %#v", readTask)
	}
	server.request(http.MethodPost, fmt.Sprintf("/tasks/%d", task.ID), map[string]any{
		"title": "Edited by original app", "project_id": 1,
	}, http.StatusOK, nil)
	var nextTask struct {
		Index int64 `json:"index"`
	}
	server.request(http.MethodPut, "/projects/1/tasks", map[string]any{"title": "Created by original app"}, http.StatusCreated, &nextTask)
	if nextTask.Index != 2 {
		t.Fatalf("original app next index = %d", nextTask.Index)
	}
	server.stop()
	store, err = NewVerifiedSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tasks, pages, err := store.ListTasksInView(ctx, 1, 1, 1, 100)
	if err != nil || pages != 1 || len(tasks) != 2 {
		t.Fatalf("Store could not read original app Tasks: %#v, pages=%d, err=%v", tasks, pages, err)
	}
	wantTitles := map[string]bool{"Edited by original app": true, "Created by original app": true}
	for _, task := range tasks {
		if !wantTitles[task.Title] {
			t.Fatalf("unexpected or duplicate Task from Store: %#v", task)
		}
		delete(wantTitles, task.Title)
	}
}
