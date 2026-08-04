package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/vimkunja/vimkunja/internal/tui"
	"github.com/vimkunja/vimkunja/internal/tutorial"
)

func main() {
	database, err := tutorial.NewDemoDatabase()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create the Vimkunja tutorial database:", err)
		os.Exit(1)
	}

	app := tui.NewModel(tui.Config{
		Store:   database.Store,
		PerPage: 100,
	})
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
