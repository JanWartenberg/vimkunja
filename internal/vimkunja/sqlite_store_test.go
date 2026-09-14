package vimkunja

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// setupTestDB creates a temporary SQLite DB with minimal schema + test data
func setupTestDB(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "vimkunja-test-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("open test db: %v", err)
	}

	// Minimal schema (projects, tasks, project_views, task_buckets, task_positions)
	schema := `
		CREATE TABLE projects (
			id INTEGER PRIMARY KEY,
			identifier TEXT,
			title TEXT,
			description TEXT,
			parent_project_id INTEGER,
			position REAL,
			is_archived INTEGER DEFAULT 0,
			updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			owner_id INTEGER DEFAULT 1
		);

		CREATE TABLE project_views (
			id INTEGER PRIMARY KEY,
			project_id INTEGER,
			title TEXT,
			view_kind INTEGER,
			position INTEGER,
			done_bucket_id INTEGER,
			default_bucket_id INTEGER,
			bucket_configuration_mode INTEGER DEFAULT 1,
			created DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE tasks (
			id INTEGER PRIMARY KEY,
			project_id INTEGER,
			title TEXT,
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
			title TEXT,
			project_view_id INTEGER,
			position REAL,
			created DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_by_id INTEGER DEFAULT 1
		);

		CREATE TABLE task_buckets (
			id INTEGER PRIMARY KEY,
			task_id INTEGER,
			bucket_id INTEGER,
			project_view_id INTEGER
		);

		CREATE TABLE task_positions (
			task_id INTEGER,
			project_view_id INTEGER,
			position REAL,
			PRIMARY KEY (task_id, project_view_id)
		);
	`

	if _, err := store.db.Exec(schema); err != nil {
		store.db.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("create schema: %v", err)
	}

	cleanup := func() {
		store.db.Close()
		os.Remove(tmpFile.Name())
	}

	return store, cleanup
}

func sqliteDatabasePath(t *testing.T, db *sql.DB) string {
	t.Helper()

	rows, err := db.Query(`PRAGMA database_list`)
	if err != nil {
		t.Fatalf("list attached databases: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sequence int
		var name, path string
		if err := rows.Scan(&sequence, &name, &path); err != nil {
			t.Fatalf("scan attached database: %v", err)
		}
		if name == "main" {
			return path
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("list attached databases: %v", err)
	}
	t.Fatal("main database is not attached")
	return ""
}

func TestSQLiteStoreListsAndAtomicallyAppendsTaskComments(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	if _, err := store.db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, username TEXT NOT NULL);
		CREATE TABLE task_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT, comment TEXT NOT NULL, author_id INTEGER NOT NULL,
			task_id INTEGER NOT NULL, created DATETIME, updated DATETIME
		);
		INSERT INTO users (id, name, username) VALUES (7, 'Ada Lovelace', 'ada');
		INSERT INTO projects (id, title, owner_id) VALUES (1, 'Project', 7);
		INSERT INTO tasks (id, project_id, title, description, priority, "index") VALUES (11, 1, 'Old', '', 0, 1);
		INSERT INTO task_comments (id, comment, author_id, task_id, created, updated) VALUES
			(2, '<p>Second</p>', 7, 11, '2026-09-04 11:00:00', '2026-09-04 11:30:00'),
			(1, '<p>First</p>', 7, 11, '2026-09-04 10:00:00', '2026-09-04 10:00:00');
	`); err != nil {
		t.Fatal(err)
	}

	comments, err := store.ListTaskComments(context.Background(), 11)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 2 || comments[0].ID != 1 || comments[1].ID != 2 || comments[0].Author != "Ada Lovelace" || !comments[1].Edited() {
		t.Fatalf("Task Comments = %#v, want chronological comments with author and edited state", comments)
	}
	if comments[0].Created.Location() != time.UTC {
		t.Fatalf("Comment timestamp location = %v, want UTC", comments[0].Created.Location())
	}

	draft := TaskDraft{ID: 11, ProjectID: 1, Title: "Renamed"}
	task, comment, err := store.SaveTaskWithComment(context.Background(), draft, TaskCommentDraft{Text: "Hello <world>\nline\n\nNext"})
	if err != nil {
		t.Fatal(err)
	}
	if task.Title != "Renamed" || comment.Author != "Ada Lovelace" || comment.HTML != "<p>Hello &lt;world&gt;<br>line</p><p>Next</p>" {
		t.Fatalf("saved Task/Comment = %#v / %#v", task, comment)
	}
	var storedHTML, created string
	if err := store.db.QueryRow(`SELECT comment, CAST(created AS TEXT) FROM task_comments WHERE id = ?`, comment.ID).Scan(&storedHTML, &created); err != nil {
		t.Fatal(err)
	}
	if storedHTML != comment.HTML {
		t.Fatalf("stored Comment = %q, want %q", storedHTML, comment.HTML)
	}
	if _, err := time.Parse(databaseTimestampLayout, created); err != nil {
		t.Fatalf("Comment created = %q, want canonical UTC", created)
	}
}

func TestSQLiteStoreDeletesOnlyCommentsBySoleLocalUser(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	if _, err := store.db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, username TEXT NOT NULL);
		CREATE TABLE task_comments (id INTEGER PRIMARY KEY, comment TEXT NOT NULL, author_id INTEGER NOT NULL, task_id INTEGER NOT NULL, created DATETIME, updated DATETIME);
		INSERT INTO users (id, username) VALUES (1, 'local');
		INSERT INTO task_comments VALUES
			(1, '<p>Own</p>', 1, 11, '2026-09-04 10:00:00', '2026-09-04 10:00:00'),
			(2, '<p>Other</p>', 2, 11, '2026-09-04 10:00:00', '2026-09-04 10:00:00');
	`); err != nil {
		t.Fatal(err)
	}
	comments, err := store.ListTaskComments(context.Background(), 11)
	if err != nil || !comments[0].CanDelete || comments[1].CanDelete {
		t.Fatalf("Comment deletion capabilities = %#v, err=%v", comments, err)
	}
	if err := store.DeleteTaskComment(context.Background(), 11, 1); err != nil {
		t.Fatalf("delete own Comment: %v", err)
	}
	if err := store.DeleteTaskComment(context.Background(), 11, 2); err == nil || !strings.Contains(err.Error(), "only the Comment author") {
		t.Fatalf("delete foreign Comment = %v, want author error", err)
	}
}

func TestListTaskCommentsAcceptsLegacyVimkunjaTimestamp(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	if _, err := store.db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, username TEXT NOT NULL);
		CREATE TABLE task_comments (id INTEGER PRIMARY KEY, comment TEXT NOT NULL, author_id INTEGER NOT NULL, task_id INTEGER NOT NULL, created DATETIME, updated DATETIME);
		INSERT INTO users (id, username) VALUES (1, 'local');
		INSERT INTO task_comments VALUES (1, '<p>Legacy</p>', 1, 11, '2026-09-04 13:00:00.123 +0200 CEST m=+1.25', '2026-09-04 13:00:00.123 +0200 CEST m=+1.25');
	`); err != nil {
		t.Fatal(err)
	}
	comments, err := store.ListTaskComments(context.Background(), 11)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 4, 11, 0, 0, 123000000, time.UTC)
	if len(comments) != 1 || !comments[0].Created.Equal(want) {
		t.Fatalf("legacy Comment timestamp = %#v, want %v", comments, want)
	}
}

func TestSaveTaskWithCommentRollsBackTaskWhenCommentInsertFails(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	if _, err := store.db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, username TEXT NOT NULL);
		CREATE TABLE task_comments (id INTEGER PRIMARY KEY, comment TEXT NOT NULL, author_id INTEGER NOT NULL, task_id INTEGER NOT NULL, created DATETIME, updated DATETIME);
		CREATE TRIGGER reject_comment BEFORE INSERT ON task_comments BEGIN SELECT RAISE(ABORT, 'comment insert failed'); END;
		INSERT INTO users (id, username) VALUES (7, 'local');
		INSERT INTO projects (id, title, owner_id) VALUES (1, 'Project', 7);
		INSERT INTO tasks (id, project_id, title, "index") VALUES (11, 1, 'Original', 1);
	`); err != nil {
		t.Fatal(err)
	}
	_, _, err := store.SaveTaskWithComment(context.Background(), TaskDraft{ID: 11, ProjectID: 1, Title: "Must roll back"}, TaskCommentDraft{Text: "Comment"})
	if err == nil || !strings.Contains(err.Error(), "comment insert failed") {
		t.Fatalf("SaveTaskWithComment error = %v, want Comment-insert error", err)
	}
	var title string
	if err := store.db.QueryRow(`SELECT title FROM tasks WHERE id = 11`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "Original" {
		t.Fatalf("Task title = %q after failed Comment insert, want rollback", title)
	}
}

func TestSQLiteStoreWritesCanonicalUTCTimestamps(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	if _, err := store.db.Exec(`INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task')`); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateTaskTitle(context.Background(), 1, "Renamed"); err != nil {
		t.Fatal(err)
	}
	var updated string
	if err := store.db.QueryRow(`SELECT CAST(updated AS TEXT) FROM tasks WHERE id = 1`).Scan(&updated); err != nil {
		t.Fatal(err)
	}
	if _, err := time.Parse("2006-01-02 15:04:05", updated); err != nil {
		t.Fatalf("persisted timestamp = %q, want canonical UTC YYYY-MM-DD HH:mm:ss", updated)
	}
}

func TestSQLiteStoreClosePreservesWALWhenCheckpointIsBusy(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	dbPath := sqliteDatabasePath(t, store.db)

	var journalMode string
	if err := store.db.QueryRow(`PRAGMA journal_mode = WAL`).Scan(&journalMode); err != nil {
		t.Fatalf("enable WAL: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal mode = %q, want wal", journalMode)
	}
	if _, err := store.db.Exec(`INSERT INTO projects (id, identifier, title) VALUES (1, 'WAL', 'before checkpoint')`); err != nil {
		t.Fatalf("write WAL data: %v", err)
	}

	reader, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open WAL reader: %v", err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	readerTx, err := reader.Begin()
	if err != nil {
		t.Fatalf("begin WAL read transaction: %v", err)
	}
	t.Cleanup(func() { _ = readerTx.Rollback() })
	var projectCount int
	if err := readerTx.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&projectCount); err != nil {
		t.Fatalf("establish WAL read snapshot: %v", err)
	}

	if _, err := store.db.Exec(`INSERT INTO projects (id, identifier, title) VALUES (2, 'KEEP', 'must survive')`); err != nil {
		t.Fatalf("write data behind reader: %v", err)
	}
	if err := store.Close(); err == nil || !strings.Contains(err.Error(), "database is busy") {
		t.Fatalf("Close() error = %v, want busy WAL checkpoint", err)
	}

	if err := readerTx.Rollback(); err != nil {
		t.Fatalf("release WAL reader: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close WAL reader: %v", err)
	}

	reopened, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("reopen WAL database: %v", err)
	}
	defer reopened.Close()
	var title string
	if err := reopened.db.QueryRow(`SELECT title FROM projects WHERE id = 2`).Scan(&title); err != nil {
		t.Fatalf("read data after incomplete checkpoint: %v", err)
	}
	if title != "must survive" {
		t.Fatalf("title = %q, want WAL data", title)
	}
}

func TestLoadSearchDataAndListTasksForLabelExcludeArchivedProjects(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	if _, err := store.db.Exec(`
		CREATE TABLE labels (
			id INTEGER PRIMARY KEY, title TEXT NOT NULL, description TEXT, hex_color TEXT,
			created_by_id INTEGER NOT NULL, created DATETIME NOT NULL, updated DATETIME NOT NULL
		);
		CREATE TABLE label_tasks (
			id INTEGER PRIMARY KEY, label_id INTEGER NOT NULL, task_id INTEGER NOT NULL, created DATETIME NOT NULL
		);
		INSERT INTO projects (id, identifier, title, is_archived) VALUES
			(1, 'HOME', 'Home', 0),
			(2, 'OLD', 'Archived', 1);
		INSERT INTO tasks (id, project_id, title, description, done) VALUES
			(11, 1, 'Buy coffee', '<p>Beans for breakfast</p>', 0),
			(12, 2, 'Buy coffee later', '<p>Hidden</p>', 0);
		INSERT INTO labels (id, title, created_by_id, created, updated) VALUES
			(3, 'Errands', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			(4, 'Unused', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
		INSERT INTO label_tasks (label_id, task_id, created) VALUES
			(3, 11, CURRENT_TIMESTAMP), (3, 12, CURRENT_TIMESTAMP);
	`); err != nil {
		t.Fatalf("seed search data: %v", err)
	}

	data, err := store.LoadSearchData(context.Background())
	if err != nil {
		t.Fatalf("load search data: %v", err)
	}
	if got, want := data.Projects, []Project{{ID: 1, Identifier: "HOME", Title: "Home"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("search projects = %#v, want %#v", got, want)
	}
	if got, want := data.Tasks, []Task{{ID: 11, ProjectID: 1, ProjectIdentifier: "HOME", ProjectTitle: "Home", Title: "Buy coffee", Description: "<p>Beans for breakfast</p>"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("search tasks = %#v, want %#v", got, want)
	}
	if got, want := data.Labels, []Label{{ID: 3, Title: "Errands"}, {ID: 4, Title: "Unused"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("search labels = %#v, want %#v", got, want)
	}

	tasks, pages, err := store.ListTasksForLabel(context.Background(), 3, 1, 20)
	if err != nil {
		t.Fatalf("list label tasks: %v", err)
	}
	if pages != 1 || len(tasks) != 1 || tasks[0].ID != 11 || tasks[0].ProjectTitle != "Home" {
		t.Fatalf("label tasks = %#v, pages = %d; want only active task 11", tasks, pages)
	}
}

func TestSQLiteStoreManagesLabelsAndTaskAssignments(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	if _, err := store.db.Exec(`
		CREATE TABLE labels (
			id INTEGER PRIMARY KEY, title TEXT NOT NULL, description TEXT, hex_color TEXT,
			created_by_id INTEGER NOT NULL, created DATETIME NOT NULL, updated DATETIME NOT NULL
		);
		CREATE TABLE label_tasks (
			id INTEGER PRIMARY KEY, label_id INTEGER NOT NULL, task_id INTEGER NOT NULL, created DATETIME NOT NULL
		);
		CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT NOT NULL);
		INSERT INTO users (id, username) VALUES (9, 'local-user');
		INSERT INTO projects (id, title, is_archived) VALUES (1, 'Current', 0), (2, 'Archived', 1);
		INSERT INTO tasks (id, project_id, title) VALUES (11, 1, 'Current task'), (12, 2, 'Archived task');
		INSERT INTO labels (id, title, created_by_id, created, updated) VALUES
			(1, 'Errands', 7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			(2, 'Unused', 7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
		INSERT INTO label_tasks (id, label_id, task_id, created) VALUES
			(1, 1, 11, CURRENT_TIMESTAMP), (2, 1, 12, CURRENT_TIMESTAMP);
	`); err != nil {
		t.Fatalf("seed labels: %v", err)
	}

	labels, err := store.ListLabels(context.Background())
	if err != nil {
		t.Fatalf("list Labels: %v", err)
	}
	if got, want := labels, []Label{{ID: 1, Title: "Errands", TaskCount: 2}, {ID: 2, Title: "Unused"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("labels = %#v, want %#v", got, want)
	}

	created, err := store.CreateLabel(context.Background(), "  Home  ")
	if err != nil {
		t.Fatalf("create Label: %v", err)
	}
	if created.Title != "Home" || created.ID == 0 {
		t.Fatalf("created Label = %#v", created)
	}
	var actorID int64
	var createdAt, updatedAt time.Time
	if err := store.db.QueryRow(`SELECT created_by_id, created, updated FROM labels WHERE id = ?`, created.ID).Scan(&actorID, &createdAt, &updatedAt); err != nil {
		t.Fatalf("read created Label: %v", err)
	}
	if actorID != 9 || createdAt.IsZero() || updatedAt.IsZero() {
		t.Fatalf("created Label metadata = actor:%d created:%v updated:%v", actorID, createdAt, updatedAt)
	}
	if _, err := store.CreateLabel(context.Background(), "home"); err == nil {
		t.Fatal("duplicate normalized Label title was accepted")
	}
	if err := store.RenameLabel(context.Background(), created.ID, "Errands"); err == nil {
		t.Fatal("rename accepted duplicate normalized Label title")
	}
	if err := store.RenameLabel(context.Background(), created.ID, "Household"); err != nil {
		t.Fatalf("rename Label: %v", err)
	}

	if err := store.ReplaceTaskLabels(context.Background(), 11, []int64{created.ID}); err != nil {
		t.Fatalf("replace Task Labels: %v", err)
	}
	assigned, err := store.ListTaskLabels(context.Background(), 11)
	if err != nil {
		t.Fatalf("list Task Labels: %v", err)
	}
	if got, want := assigned, []Label{{ID: created.ID, Title: "Household"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("assigned Labels = %#v, want %#v", got, want)
	}
	if err := store.ReplaceTaskLabels(context.Background(), 11, []int64{999}); err == nil {
		t.Fatal("replace accepted a deleted or unknown Label")
	}
	assigned, err = store.ListTaskLabels(context.Background(), 11)
	if err != nil || !reflect.DeepEqual(assigned, []Label{{ID: created.ID, Title: "Household"}}) {
		t.Fatalf("failed replace changed assignments: %#v, %v", assigned, err)
	}

	if err := store.DeleteLabel(context.Background(), 1); err != nil {
		t.Fatalf("delete Label: %v", err)
	}
	var associations, taskCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM label_tasks WHERE label_id = 1`).Scan(&associations); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE id IN (11, 12)`).Scan(&taskCount); err != nil {
		t.Fatal(err)
	}
	if associations != 0 || taskCount != 2 {
		t.Fatalf("delete Label associations/tasks = %d/%d, want 0/2", associations, taskCount)
	}
}

func TestCreateLabelUsesTheOnlyDatabaseUser(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()
	if _, err := store.db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT NOT NULL);
		CREATE TABLE labels (
			id INTEGER PRIMARY KEY, title TEXT NOT NULL, description TEXT, hex_color TEXT,
			created_by_id INTEGER NOT NULL, created DATETIME NOT NULL, updated DATETIME NOT NULL
		);
		INSERT INTO users (id, username) VALUES (7, 'default-user');
	`); err != nil {
		t.Fatalf("seed default actor: %v", err)
	}

	label, err := store.CreateLabel(context.Background(), "Inbox")
	if err != nil {
		t.Fatalf("create Label with the only database user: %v", err)
	}
	var actorID int64
	if err := store.db.QueryRow(`SELECT created_by_id FROM labels WHERE id = ?`, label.ID).Scan(&actorID); err != nil {
		t.Fatal(err)
	}
	if actorID != 7 {
		t.Fatalf("Label actor = %d, want 7", actorID)
	}
	if _, err := store.db.Exec(`INSERT INTO users (id, username) VALUES (8, 'other-user')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateLabel(context.Background(), "Ambiguous"); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("multiple database users error = %v, want exactly-one-user error", err)
	}
}

func TestProvisionStandardManualKanbanView_AssignsTasksByDoneState(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title, owner_id) VALUES (1, 'Project', 7);
		INSERT INTO tasks (id, project_id, title, done) VALUES
			(1, 1, 'Open', 0), (2, 1, 'Finished', 1);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	board, err := store.ProvisionStandardManualKanbanView(context.Background(), 1)
	if err != nil {
		t.Fatalf("provision board: %v", err)
	}
	if board.Availability != KanbanBoardAvailable || board.View.Title != "Kanban" {
		t.Fatalf("board = %#v, want available Kanban board", board)
	}
	if got, want := []string{board.Buckets[0].Title, board.Buckets[1].Title, board.Buckets[2].Title}, []string{"To-Do", "Doing", "Done"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("bucket titles = %#v, want %#v", got, want)
	}
	if got, want := []int64{board.Buckets[0].Tasks[0].ID}, []int64{1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("To-Do tasks = %#v, want %#v", got, want)
	}
	if got, want := []int64{board.Buckets[2].Tasks[0].ID}, []int64{2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Done tasks = %#v, want %#v", got, want)
	}

	var viewPosition, defaultBucketID, doneBucketID int64
	var configurationMode int
	if err := store.db.QueryRow(`SELECT position, default_bucket_id, done_bucket_id, bucket_configuration_mode FROM project_views WHERE id = ?`, board.View.ID).Scan(&viewPosition, &defaultBucketID, &doneBucketID, &configurationMode); err != nil {
		t.Fatalf("read provisioned view: %v", err)
	}
	if viewPosition != 400 || configurationMode != bucketConfigurationManual || defaultBucketID != board.Buckets[0].ID || doneBucketID != board.Buckets[2].ID {
		t.Fatalf("provisioned view = position:%d configuration:%d default:%d done:%d", viewPosition, configurationMode, defaultBucketID, doneBucketID)
	}
	var createdByID int64
	if err := store.db.QueryRow(`SELECT created_by_id FROM buckets WHERE id = ?`, board.Buckets[0].ID).Scan(&createdByID); err != nil {
		t.Fatalf("read provisioned bucket creator: %v", err)
	}
	if createdByID != 7 {
		t.Fatalf("bucket creator = %d, want Project owner 7", createdByID)
	}
	var positions int
	if err := store.db.QueryRow(`SELECT COUNT(1) FROM task_positions WHERE project_view_id = ? AND position > 0`, board.View.ID).Scan(&positions); err != nil {
		t.Fatalf("read task positions: %v", err)
	}
	if positions != 2 {
		t.Fatalf("persisted positions = %d, want 2", positions)
	}
}

func TestProvisionStandardManualKanbanView_RejectsMissingProjectWithoutView(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	if _, err := store.ProvisionStandardManualKanbanView(context.Background(), 99); err == nil {
		t.Fatal("provision missing Project succeeded")
	}
	var views int
	if err := store.db.QueryRow(`SELECT COUNT(1) FROM project_views`).Scan(&views); err != nil {
		t.Fatalf("count views after failed provision: %v", err)
	}
	if views != 0 {
		t.Fatalf("views after failed provision = %d, want none", views)
	}
}

func TestLoadManualKanbanBoard_GroupsTasksAndIgnoresFilteredViews(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO project_views (id, project_id, title, view_kind, position, bucket_configuration_mode) VALUES
			(10, 1, 'Kanban', 3, 400, 1),
			(11, 1, 'Filtered', 3, 500, 2);
		INSERT INTO buckets (id, title, project_view_id, position) VALUES
			(100, 'To-Do', 10, 100), (101, 'Doing', 10, 200), (102, 'Done', 10, 300);
		INSERT INTO tasks (id, project_id, title) VALUES
			(1, 1, 'Later'), (2, 1, 'First');
		INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES
			(1, 101, 10), (2, 101, 10);
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES
			(1, 10, 200), (2, 10, 100);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	board, err := store.LoadManualKanbanBoard(context.Background(), 1)
	if err != nil {
		t.Fatalf("load board: %v", err)
	}
	if board.Availability != KanbanBoardAvailable || board.View.ID != 10 {
		t.Fatalf("board = %#v, want available manual view 10", board)
	}
	if got, want := []string{board.Buckets[0].Title, board.Buckets[1].Title, board.Buckets[2].Title}, []string{"To-Do", "Doing", "Done"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("bucket titles = %#v, want %#v", got, want)
	}
	if got, want := []int64{board.Buckets[1].Tasks[0].ID, board.Buckets[1].Tasks[1].ID}, []int64{2, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Doing task order = %#v, want %#v", got, want)
	}
}

func TestGetTaskWorkflowStatus_ReturnsCurrentBucketAndOrderedChoices(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO project_views (id, project_id, title, view_kind) VALUES (10, 1, 'Board', 3);
		INSERT INTO buckets (id, title, project_view_id, position) VALUES
			(12, 'Done', 10, 3), (10, 'To-Do', 10, 1), (11, 'Doing', 10, 2);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task');
		INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (1, 11, 10);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	status, err := store.GetTaskWorkflowStatus(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("get task workflow status: %v", err)
	}
	if status.Availability != WorkflowStatusAvailable || status.CurrentBucket.ID != 11 || status.CurrentBucket.Title != "Doing" {
		t.Fatalf("status = %#v, want available Doing bucket", status)
	}
	if got, want := status.Buckets, []WorkflowBucket{{ID: 10, Title: "To-Do"}, {ID: 11, Title: "Doing"}, {ID: 12, Title: "Done"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("buckets = %#v, want %#v", got, want)
	}
}

func TestGetTaskWorkflowStatus_ReportsUnavailableAndAmbiguousViews(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	status, err := store.GetTaskWorkflowStatus(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("get unavailable workflow status: %v", err)
	}
	if status.Availability != WorkflowStatusUnavailable {
		t.Fatalf("unavailable status = %#v", status)
	}

	_, err = store.db.Exec(`
		INSERT INTO project_views (id, project_id, title, view_kind) VALUES
			(10, 1, 'Board A', 3), (11, 1, 'Board B', 3);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}
	status, err = store.GetTaskWorkflowStatus(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("get ambiguous workflow status: %v", err)
	}
	if status.Availability != WorkflowStatusAmbiguous {
		t.Fatalf("ambiguous status = %#v", status)
	}
	status, err = store.GetTaskWorkflowStatus(context.Background(), 1, 1, 10)
	if err != nil {
		t.Fatalf("get explicitly selected workflow status: %v", err)
	}
	if status.Availability != WorkflowStatusAvailable {
		t.Fatalf("explicitly selected status = %#v, want available", status)
	}
}

func TestSetTaskWorkflowStatus_UpdatesBucketAndDoneState(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id) VALUES (10, 1, 'Board', 3, 12);
		INSERT INTO buckets (id, title, project_view_id, position) VALUES
			(10, 'To-Do', 10, 1), (11, 'Doing', 10, 2), (12, 'Done', 10, 3);
		INSERT INTO tasks (id, project_id, title, done, done_at, updated) VALUES
			(1, 1, 'Task', 1, CURRENT_TIMESTAMP, '2020-01-01 00:00:00');
		INSERT INTO task_buckets (task_id, bucket_id, project_view_id) VALUES (1, 12, 10);
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (1, 10, 123.5);
	`)
	if err != nil {
		t.Fatalf("seed database: %v", err)
	}

	if err := store.SetTaskWorkflowStatus(context.Background(), 1, 11); err != nil {
		t.Fatalf("set Doing workflow status: %v", err)
	}
	var bucketID int64
	var done bool
	var doneAt sql.NullTime
	var updated time.Time
	if err := store.db.QueryRow(`SELECT tb.bucket_id, t.done, t.done_at, t.updated FROM tasks t JOIN task_buckets tb ON tb.task_id = t.id WHERE t.id = 1 AND tb.project_view_id = 10`).Scan(&bucketID, &done, &doneAt, &updated); err != nil {
		t.Fatalf("read Doing state: %v", err)
	}
	if bucketID != 11 || done || doneAt.Valid || !updated.After(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Doing state = bucket:%d done:%v doneAt:%#v updated:%v", bucketID, done, doneAt, updated)
	}
	var position float64
	if err := store.db.QueryRow(`SELECT position FROM task_positions WHERE task_id = 1 AND project_view_id = 10`).Scan(&position); err != nil {
		t.Fatalf("read task position: %v", err)
	}
	if position != 123.5 {
		t.Fatalf("task position = %v, want unchanged 123.5", position)
	}

	if err := store.SetTaskWorkflowStatus(context.Background(), 1, 12); err != nil {
		t.Fatalf("set Done workflow status: %v", err)
	}
	if err := store.db.QueryRow(`SELECT tb.bucket_id, t.done, t.done_at FROM tasks t JOIN task_buckets tb ON tb.task_id = t.id WHERE t.id = 1 AND tb.project_view_id = 10`).Scan(&bucketID, &done, &doneAt); err != nil {
		t.Fatalf("read Done state: %v", err)
	}
	if bucketID != 12 || !done || !doneAt.Valid {
		t.Fatalf("Done state = bucket:%d done:%v doneAt:%#v", bucketID, done, doneAt)
	}
}

func TestToggleTaskDone_UpdatesTimestamp(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert test task
	insertTask := `INSERT INTO tasks (id, project_id, title, updated) VALUES (1, 1, 'Test Task', '2020-01-01 00:00:00')`
	if _, err := store.db.Exec(insertTask); err != nil {
		t.Fatalf("insert test task: %v", err)
	}

	// Toggle done
	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatalf("toggle task done: %v", err)
	}

	// Check updated timestamp is bumped
	var updated time.Time
	q := `SELECT updated FROM tasks WHERE id = 1`
	if err := store.db.QueryRow(q).Scan(&updated); err != nil {
		t.Fatalf("query updated: %v", err)
	}

	// Should be recent (within last 5 seconds)
	if time.Since(updated) > 5*time.Second {
		t.Errorf("updated timestamp not bumped: got %v", updated)
	}
}

func TestToggleTaskDone_MovesBucketWhenDone(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert project, view with done_bucket_id, task, bucket assignment
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id) VALUES (1, 1, 'Kanban', 3, 99);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task 1');
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES (1, 1, 10, 1);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	// Toggle done
	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatalf("toggle task done: %v", err)
	}

	// Check bucket_id is moved to done_bucket_id (99)
	var bucketID int64
	q := `SELECT bucket_id FROM task_buckets WHERE task_id = 1`
	if err := store.db.QueryRow(q).Scan(&bucketID); err != nil {
		t.Fatalf("query bucket_id: %v", err)
	}

	if bucketID != 99 {
		t.Errorf("task not moved to done bucket: got %d, want 99", bucketID)
	}
}

func TestToggleTaskDone_LeavesBucketWithoutDoneBucket(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind) VALUES (1, 1, 'Kanban', 3);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task 1');
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES (1, 1, 10, 1);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatalf("toggle task done: %v", err)
	}

	var bucketID int64
	if err := store.db.QueryRow(`SELECT bucket_id FROM task_buckets WHERE task_id = 1`).Scan(&bucketID); err != nil {
		t.Fatalf("query bucket_id: %v", err)
	}
	if bucketID != 10 {
		t.Errorf("bucket changed without a done bucket: got %d, want 10", bucketID)
	}
}

func TestToggleTaskDone_MovesTaskToEachViewsDoneBucket(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id) VALUES
			(1, 1, 'Team', 3, 99),
			(2, 1, 'Personal', 3, 199);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task 1');
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES
			(1, 1, 10, 1),
			(2, 1, 20, 2);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatalf("toggle task done: %v", err)
	}

	for viewID, wantBucketID := range map[int64]int64{1: 99, 2: 199} {
		var gotBucketID int64
		if err := store.db.QueryRow(`
			SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = ?
		`, viewID).Scan(&gotBucketID); err != nil {
			t.Fatalf("query bucket for view %d: %v", viewID, err)
		}
		if gotBucketID != wantBucketID {
			t.Errorf("view %d bucket = %d, want %d", viewID, gotBucketID, wantBucketID)
		}
	}
}

func TestToggleTaskDone_ScopesBucketMoveToSelectedView(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, bucket_configuration_mode, done_bucket_id) VALUES
			(1, 1, 'Team', 3, 1, 99),
			(2, 1, 'Personal', 3, 1, 199);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task 1');
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES
			(1, 1, 10, 1),
			(2, 1, 20, 2);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	if err := store.ToggleTaskDone(context.Background(), 1, true, 1); err != nil {
		t.Fatalf("toggle task done in selected view: %v", err)
	}
	for viewID, wantBucketID := range map[int64]int64{1: 99, 2: 20} {
		var gotBucketID int64
		if err := store.db.QueryRow(`SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = ?`, viewID).Scan(&gotBucketID); err != nil {
			t.Fatalf("query bucket for view %d: %v", viewID, err)
		}
		if gotBucketID != wantBucketID {
			t.Errorf("view %d bucket = %d, want %d", viewID, gotBucketID, wantBucketID)
		}
	}
}

func TestToggleTaskDone_RejectsViewOutsideTaskProject(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Task Project'), (2, 'Other Project');
		INSERT INTO project_views (id, project_id, title, view_kind, bucket_configuration_mode, done_bucket_id) VALUES
			(1, 1, 'Task Board', 3, 1, 99),
			(2, 2, 'Other Board', 3, 1, 199);
		INSERT INTO tasks (id, project_id, title, done) VALUES (1, 1, 'Task 1', 0);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	if err := store.ToggleTaskDone(context.Background(), 1, true, 2); err == nil {
		t.Fatal("toggle with another project's view succeeded")
	}
	var done bool
	if err := store.db.QueryRow(`SELECT done FROM tasks WHERE id = 1`).Scan(&done); err != nil {
		t.Fatalf("query task done state: %v", err)
	}
	if done {
		t.Fatal("invalid selected view changed global done state")
	}
}

func TestToggleTaskDone_MovesToDefaultBucketWhenUndone(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Setup: task in its Done Bucket, then reopen it.
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id, default_bucket_id) VALUES (1, 1, 'Kanban', 3, 99, 10);
		INSERT INTO tasks (id, project_id, title, done) VALUES (1, 1, 'Task 1', 1);
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES (1, 1, 99, 1);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	// Toggle undone
	if err := store.ToggleTaskDone(ctx, 1, false); err != nil {
		t.Fatalf("toggle task undone: %v", err)
	}

	// Reopening must move to the configured Default Bucket.
	var bucketID int64
	q := `SELECT bucket_id FROM task_buckets WHERE task_id = 1`
	if err := store.db.QueryRow(q).Scan(&bucketID); err != nil {
		t.Fatalf("query bucket_id: %v", err)
	}

	if bucketID != 10 {
		t.Errorf("bucket = %d, want default bucket 10", bucketID)
	}
}

func TestToggleTaskDone_MovesTaskToEachViewsDefaultBucketWhenUndone(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, done_bucket_id, default_bucket_id) VALUES
			(1, 1, 'Team', 3, 99, 10),
			(2, 1, 'Personal', 3, 199, 20);
		INSERT INTO tasks (id, project_id, title, done) VALUES (1, 1, 'Task 1', 1);
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES
			(1, 1, 99, 1),
			(2, 1, 199, 2);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	if err := store.ToggleTaskDone(context.Background(), 1, false); err != nil {
		t.Fatalf("reopen task: %v", err)
	}
	for viewID, wantBucketID := range map[int64]int64{1: 10, 2: 20} {
		var gotBucketID int64
		if err := store.db.QueryRow(`SELECT bucket_id FROM task_buckets WHERE task_id = 1 AND project_view_id = ?`, viewID).Scan(&gotBucketID); err != nil {
			t.Fatalf("query bucket for view %d: %v", viewID, err)
		}
		if gotBucketID != wantBucketID {
			t.Errorf("view %d bucket = %d, want default bucket %d", viewID, gotBucketID, wantBucketID)
		}
	}
}

func TestMoveProjectAmongSiblingsRollsBackEveryPositionOnFailure(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title, parent_project_id, position) VALUES
			(1, 'First', 0, 100), (2, 'Second', 0, 200);
		CREATE TRIGGER reject_second_project_position
		BEFORE UPDATE OF position ON projects
		WHEN OLD.id = 2
		BEGIN
			SELECT RAISE(ABORT, 'position rejected');
		END;
	`)
	if err != nil {
		t.Fatalf("seed Projects: %v", err)
	}

	if err := store.MoveProjectAmongSiblings(context.Background(), 1, ProjectMoveNext); err == nil {
		t.Fatal("move Project succeeded despite rejected sibling update")
	}
	projects, err := store.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("list Projects: %v", err)
	}
	if projects[0].Position != 100 || projects[1].Position != 200 {
		t.Fatalf("positions after failed move = %v, %v; want original transaction", projects[0].Position, projects[1].Position)
	}
}

func TestMoveProjectAmongSiblingsPersistsOnlySiblingOrder(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title, parent_project_id, position) VALUES
			(1, 'Root', 0, 10),
			(2, 'First child', 1, 100),
			(3, 'Second child', 1, 200),
			(4, 'Other root', 0, 20);
	`)
	if err != nil {
		t.Fatalf("insert Projects: %v", err)
	}

	if err := store.MoveProjectAmongSiblings(context.Background(), 3, ProjectMovePrevious); err != nil {
		t.Fatalf("move Project: %v", err)
	}
	projects, err := store.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("list Projects: %v", err)
	}
	positions := make(map[int64]float64, len(projects))
	for _, project := range projects {
		positions[project.ID] = project.Position
	}
	if positions[3] != 100 || positions[2] != 200 {
		t.Fatalf("child positions = first:%v second:%v, want moved order 3 then 2", positions[2], positions[3])
	}
	if positions[1] != 10 || positions[4] != 20 {
		t.Fatalf("root positions changed: %v", positions)
	}
}

func TestListProjects_ReturnsHierarchyFieldsAndNormalizesNullParent(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title, parent_project_id, position) VALUES
			(1, 'Root', NULL, 20),
			(2, 'Child', 1, 10);
	`)
	if err != nil {
		t.Fatalf("insert Projects: %v", err)
	}

	projects, err := store.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("list Projects: %v", err)
	}
	if got, want := projects, []Project{
		{ID: 1, Title: "Root", ParentProjectID: 0, Position: 20},
		{ID: 2, Title: "Child", ParentProjectID: 1, Position: 10},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Projects = %#v, want %#v", got, want)
	}
}

func TestListProjects_ExcludesArchived(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert 2 projects: 1 active, 1 archived
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title, is_archived) VALUES (1, 'Active', 0);
		INSERT INTO projects (id, title, is_archived) VALUES (2, 'Archived', 1);
	`)
	if err != nil {
		t.Fatalf("insert projects: %v", err)
	}

	projects, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}
	if len(projects) > 0 && projects[0].Title != "Active" {
		t.Errorf("expected 'Active', got '%s'", projects[0].Title)
	}
}

func TestListProjects_ReturnsCompleteActiveSet(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	for i := 1; i <= 205; i++ {
		if _, err := store.db.Exec(`INSERT INTO projects (id, title) VALUES (?, ?)`, i, "Project"); err != nil {
			t.Fatalf("insert Project %d: %v", i, err)
		}
	}

	projects, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatalf("list Projects: %v", err)
	}
	if len(projects) != 205 {
		t.Fatalf("Projects = %d, want complete active set of 205", len(projects))
	}
}

func TestListViews_OrdersAndMapsKinds(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Test Project');
		INSERT INTO project_views (id, project_id, title, view_kind, position) VALUES
			(1, 1, 'Kanban', 3, 30),
			(2, 1, 'List', 0, 10),
			(3, 1, 'Table', 2, 20);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	views, _, err := store.ListViews(context.Background(), 1, 1, 50)
	if err != nil {
		t.Fatalf("list views: %v", err)
	}

	want := []struct {
		id   int64
		kind string
	}{{2, "list"}, {3, "table"}, {1, "kanban"}}
	if len(views) != len(want) {
		t.Fatalf("got %d views, want %d", len(views), len(want))
	}
	for i, expected := range want {
		if views[i].ID != expected.id || views[i].ViewKind != expected.kind {
			t.Errorf("view %d = {ID: %d, ViewKind: %q}, want {ID: %d, ViewKind: %q}", i, views[i].ID, views[i].ViewKind, expected.id, expected.kind)
		}
	}
}

func TestListProjectsAndTasks_HandleNullableFields(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, identifier, title, description) VALUES (1, NULL, 'Test Project', NULL);
		INSERT INTO project_views (id, project_id, title, view_kind) VALUES (1, 1, 'List', 0);
		INSERT INTO tasks (id, project_id, title, due_date, "index", description) VALUES (1, 1, 'Task 1', NULL, NULL, NULL);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	projects, err := store.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("list projects with NULL fields: %v", err)
	}
	if len(projects) != 1 || projects[0].Identifier != "" || projects[0].Description != "" {
		t.Errorf("projects = %#v, want one project with empty nullable fields", projects)
	}

	tasks, _, err := store.ListTasksInView(context.Background(), 1, 1, 1, 50)
	if err != nil {
		t.Fatalf("list tasks with NULL fields: %v", err)
	}
	if len(tasks) != 1 || tasks[0].DueDate != "" || tasks[0].Index != 0 || tasks[0].Description != "" {
		t.Errorf("tasks = %#v, want one task with empty nullable fields", tasks)
	}
}

func TestListOverviewTasksIncludesProjectMetadataAndExcludesArchivedProjects(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, identifier, title, is_archived) VALUES
			(1, 'VIM', 'Vimkunja', 0), (2, 'OLD', 'Archived', 1);
		INSERT INTO tasks (id, project_id, title, priority, due_date) VALUES
			(1, 1, 'Later', 3, '2026-07-30'),
			(2, 1, 'Sooner', 2, '2026-07-26'),
			(3, 1, 'Undated', 5, NULL),
			(4, 2, 'Hidden', 5, '2026-08-01');
	`)
	if err != nil {
		t.Fatalf("insert overview fixtures: %v", err)
	}

	tasks, _, err := store.ListOverviewTasks(context.Background(), 1, 50)
	if err != nil {
		t.Fatalf("list overview tasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("overview tasks = %#v, want three non-archived tasks", tasks)
	}
	if tasks[0].Title != "Sooner" || tasks[1].Title != "Later" || tasks[0].ProjectIdentifier != "VIM" || tasks[0].ProjectTitle != "Vimkunja" {
		t.Fatalf("overview tasks = %#v, want nearest due date first with Project metadata", tasks)
	}
}

func TestListTasksInView_OrdersByPosition(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert project, view, 3 tasks with positions
	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Project A');
		INSERT INTO project_views (id, project_id, title, view_kind) VALUES (1, 1, 'List', 0);
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task C');
		INSERT INTO tasks (id, project_id, title) VALUES (2, 1, 'Task A');
		INSERT INTO tasks (id, project_id, title) VALUES (3, 1, 'Task B');
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (1, 1, 3.0);
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (2, 1, 1.0);
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (3, 1, 2.0);
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	tasks, _, err := store.ListTasksInView(ctx, 1, 1, 1, 50)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// Check order: Task A (pos 1.0), Task B (2.0), Task C (3.0)
	expected := []string{"Task A", "Task B", "Task C"}
	for i, task := range tasks {
		if task.Title != expected[i] {
			t.Errorf("task %d: expected '%s', got '%s'", i, expected[i], task.Title)
		}
	}
}

func TestDeleteTaskRemovesTaskAndRelationsTransactionally(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := store.db.Exec(`
		INSERT INTO projects (id, title) VALUES (1, 'Project');
		INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Delete'), (2, 1, 'Keep');
		INSERT INTO task_positions (task_id, project_view_id, position) VALUES (1, 1, 1.0);
		INSERT INTO task_buckets (id, task_id, bucket_id, project_view_id) VALUES (1, 1, 1, 1);
		CREATE TABLE task_relations (task_id INTEGER, other_task_id INTEGER);
		CREATE TABLE task_comments (id INTEGER PRIMARY KEY, task_id INTEGER NOT NULL);
		INSERT INTO task_relations (task_id, other_task_id) VALUES (1, 2), (2, 1);
		INSERT INTO task_comments (id, task_id) VALUES (1, 1), (2, 2);
	`)
	if err != nil {
		t.Fatalf("insert deletion fixtures: %v", err)
	}

	if err := store.DeleteTask(context.Background(), 1); err != nil {
		t.Fatalf("delete task: %v", err)
	}
	for _, check := range []string{
		"SELECT COUNT(1) FROM tasks WHERE id = 1",
		"SELECT COUNT(1) FROM task_positions WHERE task_id = 1",
		"SELECT COUNT(1) FROM task_buckets WHERE task_id = 1",
		"SELECT COUNT(1) FROM task_relations WHERE task_id = 1 OR other_task_id = 1",
		"SELECT COUNT(1) FROM task_comments WHERE task_id = 1",
	} {
		var count int
		if err := store.db.QueryRow(check).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s = %d, err=%v; want 0", check, count, err)
		}
	}
	var remaining int
	if err := store.db.QueryRow("SELECT COUNT(1) FROM tasks WHERE id = 2").Scan(&remaining); err != nil || remaining != 1 {
		t.Fatalf("remaining task = %d, err=%v; want 1", remaining, err)
	}
}

func TestToggleTaskDone_SetsDoneAt(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert task
	_, err := store.db.Exec(`INSERT INTO tasks (id, project_id, title) VALUES (1, 1, 'Task')`)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	// Toggle done
	if err := store.ToggleTaskDone(ctx, 1, true); err != nil {
		t.Fatalf("toggle done: %v", err)
	}

	// Check done_at is set
	var doneAt sql.NullTime
	q := `SELECT done_at FROM tasks WHERE id = 1`
	if err := store.db.QueryRow(q).Scan(&doneAt); err != nil {
		t.Fatalf("query done_at: %v", err)
	}

	if !doneAt.Valid {
		t.Error("done_at not set")
	}
	if time.Since(doneAt.Time) > 5*time.Second {
		t.Errorf("done_at not recent: %v", doneAt.Time)
	}

	// Toggle undone
	if err := store.ToggleTaskDone(ctx, 1, false); err != nil {
		t.Fatalf("toggle undone: %v", err)
	}

	// Check done_at is cleared
	if err := store.db.QueryRow(q).Scan(&doneAt); err != nil {
		t.Fatalf("query done_at after undone: %v", err)
	}

	if doneAt.Valid {
		t.Error("done_at not cleared when undone")
	}
}
