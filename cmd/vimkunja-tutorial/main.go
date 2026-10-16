package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/tui"
	"github.com/JanWartenberg/vimkunja/internal/tutorial"
)

func main() {
	opts, err := parseOptions(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	config, warnings, err := tui.LoadUserConfig(opts.overrides)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, "Warning:", warning)
	}

	if opts.menu {
		menu := tutorial.NewStartMenu()
		if _, err := tea.NewProgram(menu).Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Tutorial menu error:", err)
			os.Exit(1)
		}
		var selected bool
		opts.mode, selected = menu.Selection()
		if !selected {
			return
		}
	}

	newDatabase := tutorial.NewDemoDatabase
	if opts.mode.Markup {
		newDatabase = tutorial.NewMarkupDemoDatabase
	}
	database, err := newDatabase()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create the Vimkunja tutorial database:", err)
		os.Exit(1)
	}

	config.Store = database.Store
	config.PerPage = 100
	if opts.mode.Markup {
		config.InitialProjectID, config.InitialViewID = 0, 0
		config.Tabstop, config.IndentStep, config.LiteralTabAlias = 4, 4, opts.literalTabAlias
	}
	app := tui.NewModel(config)
	var lesson tea.Model = app
	if !opts.mode.Practice {
		if opts.mode.Markup {
			lesson = tutorial.NewMarkupModel(app, config.LiteralTabAlias)
		} else {
			lesson = tutorial.NewModel(app)
		}
	}
	program := tea.NewProgram(lesson)
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
