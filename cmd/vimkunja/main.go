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
	leaderTimeoutFlag := flag.String("leader-timeout", "", "Leader key timeout (positive Go duration, e.g. 500ms or 2s)")
	ctrlWTimeoutFlag := flag.String("ctrl-w-timeout", "", "Ctrl-W key timeout (positive Go duration, e.g. 500ms or 2s)")
	flag.Parse()

	overrides := tui.ConfigOverrides{
		LeaderTimeout: *leaderTimeoutFlag,
		CtrlWTimeout:  *ctrlWTimeoutFlag,
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "leader-timeout":
			overrides.LeaderTimeoutSet = true
		case "ctrl-w-timeout":
			overrides.CtrlWTimeoutSet = true
		}
	})
	config, warnings, err := tui.LoadUserConfig(overrides)
	if err != nil {
		return err
	}
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, "Warning:", warning)
	}

	// SQLite-only: always use local database file
	store, err := vimkunja.NewSQLiteStore(*dbPathFlag)
	if err != nil {
		return fmt.Errorf("%w; use --db with an existing Vikunja database, or run vimkunja-tutorial to try Vimkunja without one", err)
	}

	config.Store = store
	config.PerPage = *perPageFlag
	config.InitialProjectID = *projectIDFlag
	config.InitialViewID = *viewIDFlag
	m := tui.NewModel(config)

	p := tea.NewProgram(m)
	_, runErr := p.Run()
	closeErr := store.Close()
	return errors.Join(runErr, closeErr)
}
