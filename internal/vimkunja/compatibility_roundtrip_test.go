//go:build integration

package vimkunja

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Run with an official 2.7.0 server binary in VIMKUNJA_VOA_BINARY. Each test
// starts a new real database and alternates server/Store access without overlap.
// Optional VIMKUNJA_VOA_BASELINE names a stopped synthetic DB (sidecars copied);
// VIMKUNJA_VOA_USERNAME/PASSWORD provide its login. The source is never opened.
func TestVikunja270OriginalAppRoundtrip(t *testing.T) {
	binary := os.Getenv("VIMKUNJA_VOA_BINARY")
	if binary == "" {
		t.Skip("set VIMKUNJA_VOA_BINARY to the official Vikunja 2.7.0 server")
	}
	binary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	server := &roundtripServer{t: t, binary: binary, root: root, client: &http.Client{Timeout: 5 * time.Second}}
	server.username, server.password = "roundtrip", "synthetic-roundtrip-password"
	baseline := os.Getenv("VIMKUNJA_VOA_BASELINE")
	if baseline != "" {
		server.username, server.password = os.Getenv("VIMKUNJA_VOA_USERNAME"), os.Getenv("VIMKUNJA_VOA_PASSWORD")
		if server.username == "" || server.password == "" {
			t.Fatal("baseline requires VIMKUNJA_VOA_USERNAME and VIMKUNJA_VOA_PASSWORD")
		}
		for _, suffix := range []string{"", "-wal", "-shm"} {
			data, err := os.ReadFile(baseline + suffix)
			if suffix != "" && os.IsNotExist(err) {
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "vikunja.db")+suffix, data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	server.start()
	if baseline == "" {
		server.request(http.MethodPost, "/register", map[string]any{
			"username": server.username, "email": "roundtrip@example.com", "password": server.password,
		}, http.StatusOK, nil)
	}
	server.login()
	var projectA, projectB struct {
		ID int64 `json:"id"`
	}
	server.request(http.MethodPut, "/projects", map[string]any{"title": "Roundtrip A", "identifier": "RTONE"}, http.StatusCreated, &projectA)
	server.request(http.MethodPut, "/projects", map[string]any{"title": "Roundtrip B", "identifier": "RTTWO"}, http.StatusCreated, &projectB)
	server.stop()

	store, err := NewVerifiedSQLiteStore(filepath.Join(root, "vikunja.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	label, err := store.CreateLabel(ctx, "Roundtrip label")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RenameLabel(ctx, label.ID, "Renamed label"); err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(ctx, TaskDraft{ProjectID: projectA.ID, Title: "Created by Vimkunja"})
	if err != nil {
		t.Fatal(err)
	}
	if task.Index != 1 {
		t.Fatalf("first Task index = %d", task.Index)
	}
	draft := TaskDraft{ID: task.ID, ProjectID: projectA.ID, Title: "Edited by Vimkunja", Description: "<p>Preserved HTML</p>", Priority: 3, DueDate: "2030-01-02", LabelIDsSet: true, LabelIDs: []int64{label.ID}}
	_, comment, err := store.SaveTaskWithComment(ctx, draft, TaskCommentDraft{Text: "Written by Vimkunja"})
	if err != nil {
		t.Fatal(err)
	}
	board, err := store.LoadManualKanbanBoard(ctx, projectA.ID)
	if err != nil || len(board.Buckets) < 3 {
		t.Fatalf("VOA default Board = %#v, %v", board, err)
	}
	if err := store.SetTaskWorkflowStatus(ctx, task.ID, board.Buckets[1].ID); err != nil {
		t.Fatal(err)
	}
	if err := store.ToggleTaskDone(ctx, task.ID, true); err != nil {
		t.Fatal(err)
	}
	before, err := store.TaskUndoState(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	draft.ProjectID = projectB.ID
	if _, err := store.UpdateTask(ctx, draft); err != nil {
		t.Fatal(err)
	}
	after, err := store.TaskUndoState(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreTaskUndoState(ctx, after, before); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreTaskUndoState(ctx, before, after); err != nil {
		t.Fatal(err)
	}
	deleted, err := store.CreateTask(ctx, TaskDraft{ProjectID: projectA.ID, Title: "Permanently deleted by Vimkunja"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteTask(ctx, deleted.ID); err != nil {
		t.Fatal(err)
	}
	// Exercise individual Comment and global Label deletion as well.
	_, discardComment, err := store.SaveTaskWithComment(ctx, TaskDraft{ID: task.ID, ProjectID: projectB.ID, Title: draft.Title, Description: draft.Description, Priority: draft.Priority, DueDate: draft.DueDate}, TaskCommentDraft{Text: "Discarded comment"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteTaskComment(ctx, task.ID, discardComment.ID); err != nil {
		t.Fatal(err)
	}
	discardLabel, err := store.CreateLabel(ctx, "Discarded label")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteLabel(ctx, discardLabel.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	server.start()
	server.login()
	var readTask struct {
		ID          int64  `json:"id"`
		ProjectID   int64  `json:"project_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Done        bool   `json:"done"`
		Index       int64  `json:"index"`
		Labels      []struct {
			ID int64 `json:"id"`
		} `json:"labels"`
	}
	server.request(http.MethodGet, fmt.Sprintf("/tasks/%d", task.ID), nil, http.StatusOK, &readTask)
	if readTask.ID != task.ID || readTask.ProjectID != projectB.ID || readTask.Title != draft.Title || readTask.Description != draft.Description || !readTask.Done || readTask.Index != 2 || len(readTask.Labels) != 1 || readTask.Labels[0].ID != label.ID {
		t.Fatalf("VOA read did not preserve Store changes: %#v", readTask)
	}
	var comments []struct {
		ID      int64  `json:"id"`
		Comment string `json:"comment"`
	}
	server.request(http.MethodGet, fmt.Sprintf("/tasks/%d/comments", task.ID), nil, http.StatusOK, &comments)
	if len(comments) != 1 || comments[0].ID != comment.ID || !strings.Contains(comments[0].Comment, "Written by Vimkunja") {
		t.Fatalf("VOA Comments = %#v", comments)
	}
	server.request(http.MethodGet, fmt.Sprintf("/tasks/%d", deleted.ID), nil, http.StatusNotFound, nil)
	// Every Project transition retired an address, including undo/redo.
	for _, address := range []string{"RTONE/tasks/by-index/1", "RTTWO/tasks/by-index/1", "RTONE/tasks/by-index/2"} {
		var resolved struct {
			ID int64 `json:"id"`
		}
		server.request(http.MethodGet, "/api/v2/projects/"+address, nil, http.StatusOK, &resolved)
		if resolved.ID != task.ID {
			t.Fatalf("old address %s resolves to %d", address, resolved.ID)
		}
	}
	server.request(http.MethodPost, fmt.Sprintf("/tasks/%d", task.ID), map[string]any{
		"title": "Edited again by VOA", "description": readTask.Description, "project_id": projectB.ID,
		"done": true, "priority": 3, "due_date": "2030-01-02T00:00:00Z",
	}, http.StatusOK, nil)
	var nextA, nextB struct {
		Index int64 `json:"index"`
	}
	server.request(http.MethodPut, fmt.Sprintf("/projects/%d/tasks", projectA.ID), map[string]any{"title": "Created afterwards by VOA"}, http.StatusCreated, &nextA)
	server.request(http.MethodPut, fmt.Sprintf("/projects/%d/tasks", projectB.ID), map[string]any{"title": "Created afterwards by VOA"}, http.StatusCreated, &nextB)
	if nextA.Index != 4 || nextB.Index != 3 {
		t.Fatalf("VOA reused reserved numbers: A=%d B=%d", nextA.Index, nextB.Index)
	}
	server.stop()

	store, err = NewVerifiedSQLiteStore(filepath.Join(root, "vikunja.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	data, err := store.LoadSearchData(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, updated := range data.Tasks {
		if updated.ID == task.ID {
			found = updated.Title == "Edited again by VOA" && updated.Done
		}
	}
	if !found {
		t.Fatal("VOA changes could not be read back by Vimkunja")
	}
	t.Log("Real VOA 2.7.0 roundtrip passed: create/edit/labels/comments/buckets/Done/move/delete/undo/redo, retired addresses and subsequent VOA creation")
}

type roundtripServer struct {
	t                            *testing.T
	binary, root, address, token string
	username, password           string
	client                       *http.Client
	cmd                          *exec.Cmd
	done                         chan error
	log                          *os.File
}

func (s *roundtripServer) start() {
	s.t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.t.Fatal(err)
	}
	s.address = listener.Addr().String()
	listener.Close()
	s.log, err = os.Create(filepath.Join(s.root, "server.log"))
	if err != nil {
		s.t.Fatal(err)
	}
	s.cmd = exec.Command(s.binary)
	s.cmd.Dir = s.root
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "VIKUNJA_") {
			s.cmd.Env = append(s.cmd.Env, value)
		}
	}
	s.cmd.Env = append(s.cmd.Env,
		"VIKUNJA_DATABASE_TYPE=sqlite", "VIKUNJA_DATABASE_PATH="+filepath.Join(s.root, "vikunja.db"),
		"VIKUNJA_SERVICE_ROOTPATH="+s.root, "VIKUNJA_SERVICE_INTERFACE="+s.address,
		"VIKUNJA_SERVICE_PUBLICURL=http://"+s.address+"/", "VIKUNJA_MAILER_ENABLED=false",
	)
	s.cmd.Stdout, s.cmd.Stderr = s.log, s.log
	if err := s.cmd.Start(); err != nil {
		s.t.Fatal(err)
	}
	s.done = make(chan error, 1)
	go func() { s.done <- s.cmd.Wait() }()
	s.t.Cleanup(func() {
		if s.cmd != nil {
			s.stop()
		}
	})
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-s.done:
			s.cmd = nil
			data, _ := os.ReadFile(filepath.Join(s.root, "server.log"))
			s.t.Fatalf("VOA exited: %v\n%s", err, data)
		default:
		}
		response, err := s.client.Get("http://" + s.address + "/api/v1/info")
		if err == nil {
			var info struct {
				Version string `json:"version"`
			}
			decodeErr := json.NewDecoder(response.Body).Decode(&info)
			response.Body.Close()
			if response.StatusCode == http.StatusOK && decodeErr == nil {
				if info.Version != "v2.7.0" {
					s.t.Fatalf("expected VOA v2.7.0, got %q", info.Version)
				}
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	s.t.Fatal("VOA did not become ready")
}

func (s *roundtripServer) stop() {
	s.t.Helper()
	if s.cmd == nil {
		return
	}
	if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
		_ = s.cmd.Process.Kill()
	}
	select {
	case <-s.done:
	case <-time.After(10 * time.Second):
		_ = s.cmd.Process.Kill()
		<-s.done
		s.t.Error("VOA shutdown timed out")
	}
	_ = s.log.Close()
	s.cmd = nil
}

func (s *roundtripServer) login() {
	s.t.Helper()
	var login struct {
		Token string `json:"token"`
	}
	s.token = ""
	s.request(http.MethodPost, "/login", map[string]any{"username": s.username, "password": s.password}, http.StatusOK, &login)
	s.token = login.Token
	if s.token == "" {
		s.t.Fatal("VOA returned no login token")
	}
}

func (s *roundtripServer) request(method, path string, body any, wantStatus int, result any) {
	s.t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			s.t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	if !strings.HasPrefix(path, "/api/") {
		path = "/api/v1" + path
	}
	req, err := http.NewRequest(method, "http://"+s.address+path, reader)
	if err != nil {
		s.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	response, err := s.client.Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		s.t.Fatal(err)
	}
	if response.StatusCode != wantStatus {
		s.t.Fatalf("%s %s: status %d, want %d: %s", method, path, response.StatusCode, wantStatus, data)
	}
	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			s.t.Fatal(err)
		}
	}
}
