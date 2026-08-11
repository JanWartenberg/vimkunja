package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/vimkunja/vimkunja/internal/tui"
	"github.com/vimkunja/vimkunja/internal/vimkunja"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	dbPathFlag := flag.String("db", "internal/db/vikunja.db", "Path to Vikunja SQLite database file")
	projectIDFlag := flag.Int64("project-id", 0, "Optional: start with this project")
	viewIDFlag := flag.Int64("view-id", 0, "Optional: start with this view")
	perPageFlag := flag.Int("per-page", 100, "Tasks per page for list view")
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
	})

	p := tea.NewProgram(m)
	_, runErr := p.Run()
	closeErr := store.Close()
	return errors.Join(runErr, closeErr)
}
