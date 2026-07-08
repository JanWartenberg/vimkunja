package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/vikunja-tui/vikunja-tui/internal/tui"
	"github.com/vikunja-tui/vikunja-tui/internal/vikunja"
	tea "charm.land/bubbletea/v2"
)

func main() {
	serverFlag := flag.String("server", "", "Vikunja base URL, e.g. https://vikunja.example.com")
	tokenFlag := flag.String("token", "", "Vikunja API token (sent as Authorization: Bearer <token>)")
	dbPathFlag := flag.String("db", "internal/db/vikunja.db", "Path to Vikunja SQLite database file")
	projectIDFlag := flag.Int64("project-id", 0, "Optional: start with this project")
	viewIDFlag := flag.Int64("view-id", 0, "Optional: start with this view")
	perPageFlag := flag.Int("per-page", 100, "Tasks per page for list view")
	flag.Parse()

	// MVP default: use the local SQLite file.
	var store vikunja.Store
	if st, err := os.Stat(*dbPathFlag); err == nil && !st.IsDir() {
		ss, err := vikunja.NewSQLiteStore(*dbPathFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to open sqlite db:", err)
			os.Exit(1)
		}
		store = ss
	} else {
		server := firstNonEmpty(*serverFlag, os.Getenv("VIKUNJA_SERVER"))
		token := firstNonEmpty(*tokenFlag, os.Getenv("VIKUNJA_TOKEN"))

		if server == "" || token == "" {
			fmt.Fprintln(os.Stderr, "Missing credentials. Provide --db (preferred) or --server/--token (REST mode).")
			os.Exit(2)
		}
		store = vikunja.NewClient(server, token)
	}

	m := tui.NewModel(tui.Config{
		Store:            store,
		PerPage:          *perPageFlag,
		InitialProjectID: *projectIDFlag,
		InitialViewID:    *viewIDFlag,
	})

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func firstNonEmpty(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

// Ensure go vet / linters don’t complain about time import if we expand later.
var _ = time.Second
