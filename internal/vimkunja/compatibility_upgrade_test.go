//go:build integration

package vimkunja

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Opt in with a stopped legacy DB and the official 2.7.0 binary. Only copies
// are opened; neither the test nor Vimkunja migrates the source database.
func TestVikunja270LegacyDatabaseUpgrade(t *testing.T) {
	binary, baseline := os.Getenv("VIMKUNJA_VOA_BINARY"), os.Getenv("VIMKUNJA_VOA_UPGRADE_BASELINE")
	if binary == "" || baseline == "" {
		t.Skip("set VIMKUNJA_VOA_BINARY and VIMKUNJA_VOA_UPGRADE_BASELINE")
	}
	binary, err := filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	path := filepath.Join(root, "vikunja.db")
	for _, suffix := range []string{"", "-wal", "-shm"} {
		data, err := os.ReadFile(baseline + suffix)
		if suffix != "" && os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+suffix, data, 0600); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			current, err := os.ReadFile(baseline + suffix)
			if err != nil || !reflect.DeepEqual(data, current) {
				t.Errorf("source database%s changed", suffix)
			}
		})
	}
	store, err := NewVerifiedSQLiteStore(path)
	if store != nil || err == nil || !strings.Contains(err.Error(), "migration history differs") {
		if store != nil {
			_ = store.Close()
		}
		t.Fatalf("legacy DB must reproduce startup rejection: %v", err)
	}
	legacy, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	before := upgradePreservedData(t, legacy.db)
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "migrate")
	cmd.Dir = root
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "VIKUNJA_") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env,
		"VIKUNJA_DATABASE_TYPE=sqlite", "VIKUNJA_DATABASE_PATH="+path,
		"VIKUNJA_SERVICE_ROOTPATH="+root, "VIKUNJA_SERVICE_PUBLICURL=http://127.0.0.1:3456/",
		"VIKUNJA_MAILER_ENABLED=false",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("official upgrade: %v\n%s", err, output)
	}
	upgraded, err := NewVerifiedSQLiteStore(path)
	if err != nil {
		t.Fatalf("upgraded DB rejected: %v", err)
	}
	t.Cleanup(func() { _ = upgraded.Close() })
	if after := upgradePreservedData(t, upgraded.db); !reflect.DeepEqual(before, after) {
		t.Fatal("upgrade changed preserved business fields")
	}
	var integrity string
	if err := upgraded.db.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("upgraded integrity: %q, %v", integrity, err)
	}
	if _, err := upgraded.ListProjects(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := upgraded.ListOverviewTasks(context.Background(), 1, 100); err != nil {
		t.Fatal(err)
	}
}

func upgradePreservedData(t *testing.T, db *sql.DB) map[string][][]any {
	t.Helper()
	data := make(map[string][][]any)
	for table, columns := range map[string]string{
		"tasks":         `id,title,description,done,done_at,due_date,project_id,priority,"index",uid`,
		"projects":      "id,title,description,identifier",
		"users":         "id,username,password",
		"labels":        "id,title",
		"label_tasks":   "task_id,label_id",
		"task_comments": "id,task_id,author_id,comment",
	} {
		rows, err := db.Query("SELECT " + columns + " FROM " + table + " ORDER BY " + columns)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]any, len(strings.Split(columns, ",")))
			pointers := make([]any, len(values))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			data[table] = append(data[table], values)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	return data
}
