package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/JanWartenberg/vimkunja/internal/tui"
	"github.com/JanWartenberg/vimkunja/internal/tutorial"
)

type options struct {
	mode            tutorial.Mode
	menu            bool
	literalTabAlias string
	overrides       tui.ConfigOverrides
}

func parseOptions(args []string, output io.Writer) (options, error) {
	var opts options
	flags := flag.NewFlagSet("vimkunja-tutorial", flag.ContinueOnError)
	flags.SetOutput(output)
	guided := flags.Bool("guided", false, "Start a guided lesson directly (combine with --markup for rich-text editing)")
	flags.BoolVar(&opts.mode.Practice, "practice", false, "Explore freely without guidance or key restrictions (combine with --markup for rich-text examples)")
	flags.BoolVar(&opts.mode.Markup, "markup", false, "Start the Native Rich-text Description workshop")
	flags.StringVar(&opts.literalTabAlias, "literal-tab-alias", "ctrl+q", "Rich-text workshop literal Tab prefix: ctrl+q or ctrl+b")
	flags.StringVar(&opts.overrides.LeaderTimeout, "leader-timeout", "", "Leader key timeout (positive Go duration, e.g. 500ms or 2s)")
	flags.StringVar(&opts.overrides.CtrlWTimeout, "ctrl-w-timeout", "", "Ctrl-W key timeout (positive Go duration, e.g. 500ms or 2s)")
	flags.Usage = func() {
		fmt.Fprintln(output, "Usage: vimkunja-tutorial [options]\nWithout mode flags, choose from the tutorial start menu.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return opts, err
	}
	if flags.NArg() != 0 {
		return opts, fmt.Errorf("unexpected argument: %s", flags.Arg(0))
	}
	if *guided && opts.mode.Practice {
		return opts, fmt.Errorf("--guided and --practice cannot be combined")
	}
	if opts.literalTabAlias != "ctrl+q" && opts.literalTabAlias != "ctrl+b" {
		return opts, fmt.Errorf("workshop literal-tab-alias must be ctrl+q or ctrl+b")
	}
	opts.menu = !*guided && !opts.mode.Markup && !opts.mode.Practice
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "leader-timeout":
			opts.overrides.LeaderTimeoutSet = true
		case "ctrl-w-timeout":
			opts.overrides.CtrlWTimeoutSet = true
		}
	})
	return opts, nil
}
