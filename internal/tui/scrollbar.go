package tui

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// renderVerticalScrollbar adds the shared two-column scrollbar to a Pane after
// its content has been laid out at the reserved width. The caller determines
// overflow at the full width, before reserving the rail, so the rail can never
// create overflow by reducing the available text width.
func renderVerticalScrollbar(lines []string, width, height, scrollTop, contentRows, viewportStart, viewportRows int, focused bool) string {
	if width < 3 || height < 1 {
		return fitBlock(lines, width, height)
	}

	contentWidth := width - 2
	for index, line := range lines {
		lines[index] = ansi.Truncate(line, contentWidth, "")
	}
	lines = strings.Split(fitBlock(lines, contentWidth, height), "\n")
	thumbStart, thumbEnd := scrollbarThumb(scrollTop, contentRows, viewportRows)
	thumb := tuiTheme.scrollbarThumb
	if focused {
		thumb = tuiTheme.scrollbarThumbFocused
	}
	railStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tuiTheme.scrollbarRail))
	rail := railStyle.Render("│")
	topCap := viewportStart - 1
	bottomCap := min(height-1, viewportStart+viewportRows-1)
	for row := range lines {
		glyph := rail
		switch {
		case row == topCap:
			glyph = railStyle.Render("┬")
		case row >= viewportStart && row-viewportStart >= thumbStart && row-viewportStart < thumbEnd:
			glyph = lipgloss.NewStyle().Foreground(lipgloss.Color(thumb)).Render("█")
		case row == bottomCap:
			glyph = railStyle.Render("┴")
		}
		lines[row] += " " + glyph
	}
	return strings.Join(lines, "\n")
}

// scrollbarThumb returns the [start, end) rows of the Thumb inside a
// scrollable viewport. A one-row viewport cannot expose a rail row beside its
// Thumb, so it deliberately has no proportional Thumb.
func scrollbarThumb(scrollTop, contentRows, viewportRows int) (int, int) {
	if contentRows <= viewportRows || viewportRows < 2 {
		return 0, 0
	}
	thumbHeight := clamp(int(math.Floor(float64(viewportRows*viewportRows)/float64(contentRows))), 1, viewportRows-1)
	maxScroll := contentRows - viewportRows
	maxStart := viewportRows - thumbHeight
	start := int(math.Round(float64(clamp(scrollTop, 0, maxScroll)) * float64(maxStart) / float64(maxScroll)))
	return start, start + thumbHeight
}
