package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/vimkunja/vimkunja/internal/tui"
	"github.com/vimkunja/vimkunja/internal/vimkunja"
)

func main() {
	dbPathFlag := flag.String("db", "internal/db/vikunja.db", "Path to Vikunja SQLite database file")
	projectIDFlag := flag.Int64("project-id", 0, "Optional: start with this project")
	viewIDFlag := flag.Int64("view-id", 0, "Optional: start with this view")
	perPageFlag := flag.Int("per-page", 100, "Tasks per page for list view")
	userIDFlag := flag.Int64("user-id", 0, "Vikunja user ID for created tasks (defaults to the target project owner)")
	flag.Parse()

	// SQLite-only: always use local database file
	store, err := vimkunja.NewSQLiteStore(*dbPathFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open SQLite database at %s: %v\n", *dbPathFlag, err)
		fmt.Fprintln(os.Stderr, "Please ensure the database file exists and is accessible.")
		os.Exit(1)
	}

	m := tui.NewModel(tui.Config{
		Store:            store,
		PerPage:          *perPageFlag,
		InitialProjectID: *projectIDFlag,
		InitialViewID:    *viewIDFlag,
		ActorUserID:      *userIDFlag,
	})

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// Ensure go vet / linters don't complain about time import if we expand later.
var _ = time.Second
