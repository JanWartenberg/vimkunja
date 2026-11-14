package vimkunja

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed starter/vikunja-2.7.0.sqlite.gz
var starterDatabase []byte

// OpenStartupDatabase opens an explicit existing database, or selects the default
// and installs a verified starter when absent. It returns the selected absolute
// path and whether this call created it. Existing files are never replaced.
func OpenStartupDatabase(explicitPath string) (*SQLiteStore, string, bool, error) {
	path := explicitPath
	var err error
	if path == "" {
		path, err = defaultDatabasePath(filepath.Join("internal", "db", "vikunja.db"), userDataDirectory)
	} else {
		path, err = filepath.Abs(path)
	}
	if err != nil {
		return nil, path, false, err
	}
	created := false
	if explicitPath == "" {
		created, err = installStarterDatabase(path)
		if err != nil {
			return nil, path, false, err
		}
	}
	store, err := NewVerifiedSQLiteStore(path)
	return store, path, created, err
}

func defaultDatabasePath(legacy string, dataDirectory func() (string, error)) (string, error) {
	// An inaccessible or invalid legacy path must produce an error, rather than
	// silently selecting a new empty database. Lstat also detects dangling links.
	if exists, err := databaseExists(legacy); err != nil {
		return "", fmt.Errorf("inspect legacy database %q: %w", legacy, err)
	} else if exists {
		return filepath.Abs(legacy)
	}
	dir, err := dataDirectory()
	if err != nil {
		return "", fmt.Errorf("resolve default database directory: %w; select an existing database with --db", err)
	}
	return filepath.Abs(filepath.Join(dir, "vimkunja", "vikunja.db"))
}

func userDataDirectory() (string, error) {
	if runtime.GOOS == "windows" {
		if dir := os.Getenv("LOCALAPPDATA"); filepath.IsAbs(dir) {
			return dir, nil
		}
		return "", errors.New("LOCALAPPDATA must name an absolute directory")
	}
	if runtime.GOOS != "darwin" {
		if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
			if !filepath.IsAbs(dir) {
				return "", errors.New("XDG_DATA_HOME must name an absolute directory")
			}
			return dir, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(home) {
		return "", errors.New("home directory must be absolute")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support"), nil
	}
	return filepath.Join(home, ".local", "share"), nil
}

func installStarterDatabase(path string) (bool, error) {
	if exists, err := databaseExists(path); err != nil {
		return false, err
	} else if exists {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return false, fmt.Errorf("create database directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".vimkunja-starter-*")
	if err != nil {
		return false, fmt.Errorf("create starter database: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	reader, err := gzip.NewReader(bytes.NewReader(starterDatabase))
	if err != nil {
		file.Close()
		return false, fmt.Errorf("read embedded starter: %w", err)
	}
	_, copyErr := io.Copy(file, reader)
	err = errors.Join(copyErr, reader.Close(), file.Sync(), file.Close())
	if err != nil {
		return false, fmt.Errorf("write starter database: %w", err)
	}
	store, err := NewVerifiedSQLiteStore(temporary)
	if err != nil {
		return false, fmt.Errorf("validate starter database: %w", err)
	}
	if err := store.Close(); err != nil {
		return false, fmt.Errorf("close starter database: %w", err)
	}
	// A hard link publishes the complete, closed database atomically, without
	// overwriting a winner from another process (Rename can overwrite on Unix).
	if err := os.Link(temporary, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, fmt.Errorf("publish starter database at %q (requires hard-link support): %w", path, err)
	}
	return true, nil
}

func databaseExists(path string) (bool, error) {
	return inspectDatabasePath(path, os.Lstat)
}

func inspectDatabasePath(path string, stat func(string) (os.FileInfo, error)) (bool, error) {
	if _, err := stat(path); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	// Orphaned WAL files may hold data from a previous database. Neither attach
	// them to a new one nor silently switch away from a damaged legacy database.
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := stat(path + suffix); !errors.Is(err, os.ErrNotExist) {
			if err != nil {
				return false, fmt.Errorf("inspect database sidecar %q: %w", path+suffix, err)
			}
			// Another first start may have published and opened the database
			// since the initial lookup. Its live sidecars are not orphans.
			if _, err := stat(path); err == nil {
				return true, nil
			} else if !errors.Is(err, os.ErrNotExist) {
				return false, err
			}
			return false, fmt.Errorf("database is missing but sidecar %q exists; restore or inspect it before starting", path+suffix)
		}
	}
	return false, nil
}
