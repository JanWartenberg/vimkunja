package vimkunja

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestUpdateTaskPreservesDescriptionBytesAndDone(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	const source = "<p>A &amp; 猫</p><custom-node DATA-X='one' data-x=\"two\">\r\nKEEP\t</custom-node><!-- secret --><pre><code>\nA\r\n\tB\r\n</code></pre>"
	if _, err := store.db.Exec(`INSERT INTO projects (id, title) VALUES (1, 'Project'); INSERT INTO tasks (id, project_id, title, description, done, "index") VALUES (1, 1, 'Old', ?, 1, 1)`, source); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{source, strings.Replace(source, "A &amp; 猫", "Edited &amp; 猫", 1)} {
		draft := TaskDraft{ID: 1, ProjectID: 1, Title: "Renamed", Description: text}
		before := draft
		task, err := store.UpdateTask(context.Background(), draft)
		if err != nil {
			t.Fatal(err)
		}
		var persisted string
		var done bool
		if err := store.db.QueryRow(`SELECT description, done FROM tasks WHERE id = 1`).Scan(&persisted, &done); err != nil {
			t.Fatal(err)
		}
		if persisted != text || task.Description != text || !done || !task.Done {
			t.Fatalf("Description bytes or Done changed: persisted=%q, task=%#v, done=%v", persisted, task, done)
		}
		if !reflect.DeepEqual(draft, before) {
			t.Fatal("save mutated caller draft")
		}
	}
}

func TestUpdateTaskFailureRollsBackDescriptionAndRetainsDraft(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	const source = "<p>Original</p><custom-node X='keep'>é\r\n</custom-node>"
	if _, err := store.db.Exec(`INSERT INTO projects (id, title) VALUES (1, 'Project'); INSERT INTO tasks (id, project_id, title, description, done, "index") VALUES (1, 1, 'Old', ?, 1, 1)`, source); err != nil {
		t.Fatal(err)
	}
	// An invalid workflow target fails after the tasks UPDATE, exercising a
	// real transaction rollback rather than pre-write argument validation.
	draft := TaskDraft{ID: 1, ProjectID: 1, Title: "New", Description: "<p>Local</p>", WorkflowBucketID: 999}
	before := draft
	if _, err := store.UpdateTask(context.Background(), draft); err == nil {
		t.Fatal("expected invalid workflow save failure")
	}
	var title, persisted string
	var done bool
	if err := store.db.QueryRow(`SELECT title, description, done FROM tasks WHERE id = 1`).Scan(&title, &persisted, &done); err != nil {
		t.Fatal(err)
	}
	if title != "Old" || persisted != source || !done || !reflect.DeepEqual(draft, before) {
		t.Fatalf("partial save or changed draft: title=%q description=%q done=%v draft=%#v", title, persisted, done, draft)
	}
}
