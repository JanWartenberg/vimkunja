package tui

import "charm.land/lipgloss/v2"

type textFormat struct {
	bold, underline bool
	foreground      string
}

func (format textFormat) render(text string) string {
	style := lipgloss.NewStyle().Bold(format.bold).Underline(format.underline)
	if format.foreground != "" {
		style = style.Foreground(lipgloss.Color(format.foreground))
	}
	return style.Render(text)
}

// renderHeading is the single rendering seam for Pane headings. Themes may
// change the semantic heading format without each Pane choosing its own style.
func renderHeading(text string) string {
	return tuiTheme.heading.render(text)
}

// tuiTheme is the single palette for terminal rendering. Keeping semantic
// roles here lets a future user-configurable theme replace mappings without
// finding terminal colour codes throughout individual panes.
var tuiTheme = struct {
	selectionAccent, inactiveSelection, muted, danger    string
	dueSoon, dueUpcoming, ctrlWHint, bright, dark        string
	scrollbarRail, scrollbarThumb, scrollbarThumbFocused string
	heading                                              textFormat
	priority                                             [6]string
}{
	selectionAccent:       "205",
	inactiveSelection:     "229",
	muted:                 "245",
	danger:                "196",
	dueSoon:               "208",
	dueUpcoming:           "220",
	ctrlWHint:             "227",
	scrollbarRail:         "240",
	scrollbarThumb:        "245",
	scrollbarThumbFocused: "255",
	heading:               textFormat{bold: true, underline: true},
	bright:                "255",
	dark:                  "0",
	priority:              [6]string{"245", "34", "220", "208", "196", "196"},
}

// SelectionAccentColor exposes the shared accent to TUI-adjacent renderers,
// such as the interactive tutorial.
func SelectionAccentColor() string {
	return tuiTheme.selectionAccent
}

func focusedField(text string, focused bool) string {
	if !focused {
		return text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.selectionAccent)).Bold(true).Render("▸ " + text)
}

// pickerField makes the complete active Picker control visible without adding
// a glyph that could change the form's layout.
func pickerField(text string, active bool) string {
	if !active {
		return text
	}
	return lipgloss.NewStyle().Background(lipgloss.Color(tuiTheme.selectionAccent)).Render(text)
}

// priorityColor maps a Vikunja priority to its semantic terminal colour.
func priorityColor(priority int) string {
	return tuiTheme.priority[clamp(priority, 0, len(tuiTheme.priority)-1)]
}
