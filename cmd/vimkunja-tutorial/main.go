package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/tui"
	"github.com/JanWartenberg/vimkunja/internal/tutorial"
)

func main() {
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
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, "Warning:", warning)
	}

	database, err := tutorial.NewDemoDatabase()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create the Vimkunja tutorial database:", err)
		os.Exit(1)
	}

	config.Store = database.Store
	config.PerPage = 100
	app := tui.NewModel(config)
	program := tea.NewProgram(tutorial.NewModel(app))
	_, runErr := program.Run()
	cleanupErr := database.Close()
	if runErr != nil {
		fmt.Fprintln(os.Stderr, "Tutorial error:", runErr)
	}
	if cleanupErr != nil {
		fmt.Fprintln(os.Stderr, "Failed to remove the temporary tutorial database:", cleanupErr)
	}
	if runErr != nil || cleanupErr != nil {
		os.Exit(1)
	}
}
