package vimkunja

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSQLiteStoreDoesNotCreateMissingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	store, err := NewSQLiteStore(path)
	if store != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("open missing database = %v, %v", store, err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing database was created: %v", err)
	}
}

func TestSQLiteStoreRejectsDirectory(t *testing.T) {
	store, err := NewSQLiteStore(t.TempDir())
	if store != nil || err == nil {
		t.Fatalf("open directory = %v, %v", store, err)
	}
}

func TestSQLiteStoreOpensLiteralPaths(t *testing.T) {
	// URI punctuation must remain part of the filename, not SQLite options.
	for _, name := range []string{"space name.db", "hash#name.db", "percent%name.db"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`CREATE TABLE marker (value TEXT); INSERT INTO marker VALUES ('original')`); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			store, err := NewSQLiteStore(path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })
			var value string
			if err := store.db.QueryRowContext(context.Background(), `SELECT value FROM marker`).Scan(&value); err != nil || value != "original" {
				t.Fatalf("literal file value = %q, %v", value, err)
			}
		})
	}
}
