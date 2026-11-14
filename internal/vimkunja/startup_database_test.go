package vimkunja

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestStarterDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "vikunja.db")
	created, err := installStarterDatabase(path)
	if err != nil || !created {
		t.Fatalf("install = %v, %v", created, err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	store, selected, created, err := OpenStartupDatabase(path)
	if err != nil || selected != path || created {
		t.Fatalf("open = %q, %v, %v", selected, created, err)
	}
	ctx := context.Background()
	projects, err := store.ListProjects(ctx)
	if err != nil || len(projects) != 1 || projects[0].Title != "Inbox" {
		t.Fatalf("projects = %#v, %v", projects, err)
	}
	for query, want := range map[string]int64{
		"SELECT COUNT(*) FROM users WHERE username='local' AND name='Local User' AND password='' AND email=''": 1,
		"SELECT COUNT(*) FROM tasks":          0,
		"SELECT COUNT(*) FROM project_views":  4,
		"SELECT COUNT(*) FROM buckets":        3,
		"SELECT COUNT(*) FROM sessions":       0,
		"SELECT COUNT(*) FROM license_status": 0,
		"SELECT COUNT(*) FROM project_ancestors WHERE ancestor_id=1 AND project_id=1 AND depth=0": 1,
	} {
		if got := compatInt(t, store, query); got != want {
			t.Errorf("%s = %d, want %d", query, got, want)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if created, err := installStarterDatabase(path); err != nil || created {
		t.Fatalf("repeat install = %v, %v", created, err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("opening and repeating initialization changed database bytes")
	}
	store, err = NewVerifiedSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	task, err := store.CreateTask(ctx, TaskDraft{ProjectID: 1, Title: "My first Task"})
	if err != nil || task.Index != 1 {
		t.Fatalf("first Task = %#v, %v", task, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, _, _, err = OpenStartupDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if got := compatInt(t, store, "SELECT COUNT(*) FROM tasks WHERE title='My first Task'"); got != 1 {
		t.Fatal("reopening lost saved Task")
	}
}

func TestStarterConcurrentInstallation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "vikunja.db")
	var wg sync.WaitGroup
	results := make(chan bool, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, err := installStarterDatabase(path)
			if err != nil {
				t.Error(err)
			}
			results <- created
		}()
	}
	wg.Wait()
	close(results)
	winners := 0
	for created := range results {
		if created {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("created %d databases, want one", winners)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 || entries[0].Name() != "vikunja.db" {
		t.Fatalf("temporary files remain: %v", entries)
	}
}

func TestDatabasePublishedDuringSidecarInspection(t *testing.T) {
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		t.Run(suffix, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "vikunja.db")
			firstLookup := true
			stat := func(name string) (os.FileInfo, error) {
				if name == path && firstLookup {
					firstLookup = false
					// Simulate publication just after the missing-file observation.
					if err := os.WriteFile(path, []byte("concurrent winner"), 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path+suffix, nil, 0600); err != nil {
						t.Fatal(err)
					}
					return nil, &os.PathError{Op: "lstat", Path: name, Err: os.ErrNotExist}
				}
				return os.Lstat(name)
			}
			if exists, err := inspectDatabasePath(path, stat); err != nil || !exists {
				t.Fatalf("concurrent winner mistaken for orphaned sidecar: exists=%v, err=%v", exists, err)
			}
		})
	}
}

func TestStartupPreservesInvalidAndMissingExplicitDatabases(t *testing.T) {
	for _, contents := range [][]byte{nil, []byte("not SQLite")} {
		path := filepath.Join(t.TempDir(), "invalid.db")
		if err := os.WriteFile(path, contents, 0600); err != nil {
			t.Fatal(err)
		}
		if created, err := installStarterDatabase(path); created || err != nil {
			t.Fatalf("existing file installation = %v, %v", created, err)
		}
		if store, _, _, err := OpenStartupDatabase(path); err == nil || store != nil {
			t.Fatalf("invalid DB accepted: %v", err)
		}
		after, _ := os.ReadFile(path)
		if !bytes.Equal(contents, after) {
			t.Fatal("invalid file changed")
		}
	}
	path := filepath.Join(t.TempDir(), "missing.db")
	if _, _, _, err := OpenStartupDatabase(path); err == nil {
		t.Fatal("missing explicit path accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("missing explicit path was created: %v", err)
	}
}

func TestStarterInstallationFailures(t *testing.T) {
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		t.Run(suffix, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "vikunja.db")
			if err := os.WriteFile(path+suffix, []byte("preserve"), 0600); err != nil {
				t.Fatal(err)
			}
			if created, err := installStarterDatabase(path); created || err == nil || !strings.Contains(err.Error(), "sidecar") {
				t.Fatalf("orphaned sidecar = %v, %v", created, err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("database created despite sidecar: %v", err)
			}
		})
	}
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := installStarterDatabase(filepath.Join(path, "db")); err == nil {
		t.Fatal("non-directory parent accepted")
	}
	if runtime.GOOS != "windows" && os.Getuid() != 0 {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0500); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(dir, 0700)
		if _, err := installStarterDatabase(filepath.Join(dir, "db")); err == nil {
			t.Fatal("unwritable directory accepted")
		}
	}
}

func TestDefaultDatabasePath(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "legacy.db")
	dataDir := func() (string, error) { return filepath.Join(root, "data"), nil }
	path, err := defaultDatabasePath(legacy, dataDir)
	if err != nil || path != filepath.Join(root, "data", "vimkunja", "vikunja.db") {
		t.Fatalf("fresh path = %q, %v", path, err)
	}
	// Even an invalid existing legacy file wins; startup must reject it instead
	// of silently switching the user to a new database.
	if err := os.WriteFile(legacy, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	path, err = defaultDatabasePath(legacy, func() (string, error) {
		t.Fatal("legacy selection must not require a user-data directory")
		return "", nil
	})
	if err != nil || path != legacy {
		t.Fatalf("legacy path = %q, %v", path, err)
	}
	if err := os.Remove(legacy); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy+"-wal", []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := defaultDatabasePath(legacy, dataDir); err == nil || !strings.Contains(err.Error(), "sidecar") {
		t.Fatalf("orphaned legacy WAL must block selecting a fresh database: %v", err)
	}
}

func TestStartupDefaultFreshAndLegacy(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("LOCALAPPDATA", filepath.Join(root, "local"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("HOME", root)
	store, selected, created, err := OpenStartupDatabase("")
	if err != nil || !created || !filepath.IsAbs(selected) || !strings.HasPrefix(selected, root+string(filepath.Separator)) {
		t.Fatalf("fresh startup = %q, %v, %v", selected, created, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, repeated, created, err := OpenStartupDatabase("")
	if err != nil || created || repeated != selected {
		t.Fatalf("repeat startup = %q, %v, %v", repeated, created, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(root, "internal", "db", "vikunja.db")
	if _, err := installStarterDatabase(legacy); err != nil {
		t.Fatal(err)
	}
	store, selected, created, err = OpenStartupDatabase("")
	if err != nil || created || selected != legacy {
		t.Fatalf("legacy startup = %q, %v, %v", selected, created, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	// Incompatible legacy data must block startup even when a valid new default
	// is available. The user must explicitly select another database.
	contents := []byte("invalid legacy data")
	if err := os.WriteFile(legacy, contents, 0600); err != nil {
		t.Fatal(err)
	}
	store, selected, created, err = OpenStartupDatabase("")
	if err == nil || store != nil || created || selected != legacy {
		t.Fatalf("invalid legacy startup = %q, %v, %v", selected, created, err)
	}
	after, _ := os.ReadFile(legacy)
	if !bytes.Equal(contents, after) {
		t.Fatal("invalid legacy data changed")
	}
}

func TestUserDataDirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LOCALAPPDATA", root)
	t.Setenv("XDG_DATA_HOME", root)
	t.Setenv("HOME", root)
	want := root
	if runtime.GOOS == "darwin" {
		want = filepath.Join(root, "Library", "Application Support")
	}
	if got, err := userDataDirectory(); err != nil || got != want {
		t.Fatalf("user data directory = %q, %v; want %q", got, err, want)
	}
	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", "relative")
	} else if runtime.GOOS != "darwin" {
		t.Setenv("XDG_DATA_HOME", "relative")
	} else {
		return
	}
	if _, err := userDataDirectory(); err == nil {
		t.Fatal("relative data directory accepted")
	}
}
