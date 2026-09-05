// Package tutorial provides an isolated, guided Vimkunja session.
package tutorial

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/vimkunja/vimkunja/internal/vimkunja"

	_ "modernc.org/sqlite"
)

// DemoDatabase is a disposable SQLite database populated for the tutorial.
// It is never derived from, or connected to, the user's Vikunja database.
type DemoDatabase struct {
	Path  string
	Store *vimkunja.SQLiteStore

	closeOnce sync.Once
	closeErr  error
}

// NewDemoDatabase creates and seeds a fresh database in the operating
// system's temporary directory. Every tutorial run starts from the same data.
func NewDemoDatabase() (*DemoDatabase, error) {
	file, err := os.CreateTemp("", "vimkunja-tutorial-*.db")
	if err != nil {
		return nil, fmt.Errorf("create tutorial database: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("close tutorial database file: %w", err)
	}

	removeOnError := func(err error) (*DemoDatabase, error) {
		_ = os.Remove(path)
		return nil, err
	}
	bootstrap, err := sql.Open("sqlite", path)
	if err != nil {
		return removeOnError(fmt.Errorf("open tutorial database: %w", err))
	}
	if err := createTutorialSchema(bootstrap); err != nil {
		_ = bootstrap.Close()
		return removeOnError(err)
	}
	if err := seedTutorialDatabase(bootstrap); err != nil {
		_ = bootstrap.Close()
		return removeOnError(err)
	}
	if err := bootstrap.Close(); err != nil {
		return removeOnError(fmt.Errorf("close seeded tutorial database: %w", err))
	}

	store, err := vimkunja.NewSQLiteStore(path)
	if err != nil {
		return removeOnError(fmt.Errorf("open seeded tutorial database: %w", err))
	}
	return &DemoDatabase{Path: path, Store: store}, nil
}

// Close closes the Store and deletes the disposable database. It is safe to
// call more than once.
func (d *DemoDatabase) Close() error {
	if d == nil {
		return nil
	}
	d.closeOnce.Do(func() {
		var closeErr error
		if d.Store != nil {
			closeErr = d.Store.Close()
		}
		removeErr := os.Remove(d.Path)
		if errors.Is(removeErr, os.ErrNotExist) {
			removeErr = nil
		}
		d.closeErr = errors.Join(closeErr, removeErr)
	})
	return d.closeErr
}

func createTutorialSchema(db *sql.DB) error {
	const schema = `
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT,
    username TEXT NOT NULL
);
CREATE TABLE projects (
    id INTEGER PRIMARY KEY,
    identifier TEXT,
    title TEXT NOT NULL,
    description TEXT,
    parent_project_id INTEGER,
    position REAL,
    is_archived INTEGER DEFAULT 0,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    owner_id INTEGER DEFAULT 1
);
CREATE TABLE project_views (
    id INTEGER PRIMARY KEY,
    project_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    view_kind INTEGER NOT NULL,
    position REAL,
    done_bucket_id INTEGER,
    default_bucket_id INTEGER,
    bucket_configuration_mode INTEGER DEFAULT 0,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    project_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    done INTEGER DEFAULT 0,
    done_at DATETIME,
    priority INTEGER DEFAULT 0,
    "index" INTEGER,
    due_date DATETIME,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_by_id INTEGER DEFAULT 1
);
CREATE TABLE buckets (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    project_view_id INTEGER NOT NULL,
    position REAL,
    created DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_by_id INTEGER DEFAULT 1
);
CREATE TABLE task_buckets (
    id INTEGER PRIMARY KEY,
    task_id INTEGER NOT NULL,
    bucket_id INTEGER NOT NULL,
    project_view_id INTEGER NOT NULL
);
CREATE TABLE task_positions (
    task_id INTEGER NOT NULL,
    project_view_id INTEGER NOT NULL,
    position REAL,
    PRIMARY KEY (task_id, project_view_id)
);
CREATE TABLE labels (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    hex_color TEXT,
    created_by_id INTEGER NOT NULL,
    created DATETIME NOT NULL,
    updated DATETIME NOT NULL
);
CREATE TABLE label_tasks (
    id INTEGER PRIMARY KEY,
    task_id INTEGER NOT NULL,
    label_id INTEGER NOT NULL,
    created DATETIME NOT NULL
);
CREATE TABLE task_comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    comment TEXT NOT NULL,
    author_id INTEGER NOT NULL,
    task_id INTEGER NOT NULL,
    created DATETIME,
    updated DATETIME
);`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("create tutorial schema: %w", err)
	}
	return nil
}

func seedTutorialDatabase(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tutorial seed: %w", err)
	}
	defer tx.Rollback()

	const fixedSeed = `
INSERT INTO users (id, name, username) VALUES (1, 'Tutorial User', 'tutorial');
INSERT INTO projects (id, identifier, title, description, parent_project_id, position, owner_id) VALUES
    (1, 'LEARN', 'Learn Vimkunja', 'The guided workshop project.', 0, 100, 1),
    (2, 'HOME', 'Home', 'A second project for cross-project examples.', 0, 200, 1),
    (3, 'TRIP', 'Plan a Trip', 'This project intentionally starts without a Kanban View.', 0, 300, 1);

INSERT INTO project_views (id, project_id, title, view_kind, position, default_bucket_id, done_bucket_id, bucket_configuration_mode) VALUES
    (11, 1, 'List',   0, 100, NULL, NULL, 0),
    (12, 1, 'Kanban', 3, 400, 121,  123,  1),
    (21, 2, 'List',   0, 100, NULL, NULL, 0),
    (22, 2, 'Kanban', 3, 400, 221,  223,  1),
    (31, 3, 'List',   0, 100, NULL, NULL, 0);

INSERT INTO buckets (id, title, project_view_id, position) VALUES
    (121, 'To-Do', 12, 100), (122, 'Doing', 12, 200), (123, 'Done', 12, 300),
    (221, 'To-Do', 22, 100), (222, 'Doing', 22, 200), (223, 'Done', 22, 300);

INSERT INTO tasks (id, project_id, title, description, done, done_at, priority, "index") VALUES
    (101, 1, 'Practice the Vimkunja tutorial', 'Move through this description.\nEvery change is disposable.\nThe real database is safe.', 0, NULL, 2, 1),
    (102, 1, 'Review keyboard shortcuts', 'Use the in-application Help Pane when needed.', 0, NULL, 3, 2),
    (103, 1, 'Move a card across the board', 'Kanban moves synchronize Workflow Status and Done.', 0, NULL, 4, 3),
    (104, 1, 'Celebrate a completed task', 'Completed Tasks use the Done Bucket.', 1, CURRENT_TIMESTAMP, 1, 4),
    (105, 1, 'Open the All-Tasks Overview', '', 0, NULL, 0, 5),
    (106, 1, 'Edit a due date', '', 0, NULL, 2, 6),
    (107, 1, 'Try Ex commands', '', 0, NULL, 3, 7),
    (108, 1, 'Inspect task metadata', '', 0, NULL, 1, 8),
    (201, 2, 'Buy coffee', 'This task demonstrates cross-Project Overview rows.', 0, NULL, 2, 1),
    (202, 2, 'Water the plants', '', 1, CURRENT_TIMESTAMP, 1, 2),
    (301, 3, 'Choose a destination', 'Provisioning will place this Task in To-Do.', 0, NULL, 3, 1),
    (302, 3, 'Book the trip', 'Completed before the Board exists.', 1, CURRENT_TIMESTAMP, 4, 2);

INSERT INTO task_positions (task_id, project_view_id, position) VALUES
    (101,11,1),(102,11,2),(103,11,3),(104,11,4),(105,11,5),(106,11,6),(107,11,7),(108,11,8),
    (101,12,1),(102,12,2),(103,12,3),(104,12,4),(105,12,5),(106,12,6),(107,12,7),(108,12,8),
    (201,21,1),(202,21,2),(201,22,1),(202,22,2),
    (301,31,1),(302,31,2);

INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES
    (101,121,12),(102,121,12),(103,122,12),(104,123,12),
    (105,121,12),(106,121,12),(107,122,12),(108,121,12),
    (201,221,22),(202,223,22);`
	if _, err := tx.Exec(fixedSeed); err != nil {
		return fmt.Errorf("seed tutorial fixtures: %w", err)
	}

	now := time.Now().Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dueDates := map[int64]time.Time{
		102: today,
		103: today.AddDate(0, 0, 1),
		105: today.AddDate(0, 0, -2),
		201: today.AddDate(0, 0, 4),
		301: today.AddDate(0, 0, 7),
	}
	for taskID, due := range dueDates {
		if _, err := tx.Exec(`UPDATE tasks SET due_date = ? WHERE id = ?`, due, taskID); err != nil {
			return fmt.Errorf("seed tutorial due date for task %d: %w", taskID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tutorial seed: %w", err)
	}
	return nil
}
