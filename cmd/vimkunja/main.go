package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/tui"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run() error {
	dbPathFlag := flag.String("db", "", "Existing Vikunja 2.7.0 SQLite file (default: legacy internal/db/vikunja.db if present, otherwise create in the platform user-data directory)")
	projectIDFlag := flag.Int64("project-id", 0, "Optional: start with this project")
	viewIDFlag := flag.Int64("view-id", 0, "Optional: start with this view")
	perPageFlag := flag.Int("per-page", 100, "Tasks per page for list view")
	leaderTimeoutFlag := flag.String("leader-timeout", "", "Leader key timeout (positive Go duration, e.g. 500ms or 2s)")
	ctrlWTimeoutFlag := flag.String("ctrl-w-timeout", "", "Ctrl-W key timeout (positive Go duration, e.g. 500ms or 2s)")
	flag.Parse()
	var explicitDB bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "db" {
			explicitDB = true
		}
	})
	if explicitDB && *dbPathFlag == "" {
		return errors.New("--db must name an existing Vikunja 2.7.0 database")
	}

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

	store, dbPath, created, err := vimkunja.OpenStartupDatabase(*dbPathFlag)
	if err != nil {
		return fmt.Errorf("database %q: %w; use --db with an existing verified Vikunja 2.7.0 database, or run vimkunja-tutorial to try Vimkunja", dbPath, err)
	}
	fmt.Fprintln(os.Stderr, "Database:", dbPath)
	if created {
		fmt.Fprintln(os.Stderr, "Created local database with Local User and an empty Inbox Project.")
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
