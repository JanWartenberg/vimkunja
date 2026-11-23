package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

func (m *model) highlightLocalValue(pane paneID, id int64, field editorField, value string) string {
	return m.highlightLocalSlice(pane, id, field, value, ansi.Strip(value), 0, 0)
}

func (m *model) highlightLocalSlice(pane paneID, id int64, field editorField, value, source string, sourceStart, prefixCells int) string {
	pattern, current, visible := m.localPattern()
	if !visible || pattern == nil || m.focus != pane {
		return value
	}
	plain := ansi.Strip(value)
	content := plain
	if prefixCells > 0 {
		content = strings.TrimPrefix(content, "…")
	}
	length := 0
	for length < len(content) && sourceStart+length < len(source) && content[length] == source[sourceStart+length] {
		length++
	}
	ranges := pattern.FindAll(source)
	if len(ranges) == 0 {
		return value
	}
	var out strings.Builder
	column := 0
	for _, match := range ranges {
		lo, hi := max(sourceStart, match.Start), min(sourceStart+length, match.End)
		if match.Start == match.End {
			it := uniseg.NewGraphemes(source)
			for it.Next() {
				start, end := it.Positions()
				if start <= match.Start && (match.Start < end || match.Start == len(source)) {
					lo, hi = max(sourceStart, start), min(sourceStart+length, end)
					break
				}
			}
		}
		if hi <= lo {
			continue
		}
		start := prefixCells + ansi.StringWidth(source[sourceStart:lo])
		end := prefixCells + ansi.StringWidth(source[sourceStart:hi])
		if end <= start {
			continue
		}
		out.WriteString(ansi.Cut(value, column, start))
		style := lipgloss.NewStyle().Reverse(true)
		if current.pane == pane && current.id == id && current.field == field && current.offset == match.Start {
			// Lip Gloss underlining styles each rune, including existing ANSI
			// escapes. Bold/color keep this pre-styled slice intact.
			style = style.Bold(true).Foreground(lipgloss.Color(tuiTheme.selectionAccent))
		}
		out.WriteString(style.Render(ansi.Cut(value, start, end)))
		column = end
	}
	out.WriteString(ansi.Cut(value, column, ansi.StringWidth(value)))
	return out.String()
}

// Search keeps a clipped title's current occurrence visible without changing
// the source or allowing a terminal-cell cut through an ANSI sequence.
func (m *model) localTitle(pane paneID, id int64, field editorField, value string, width int) string {
	width = max(1, width)
	painted := m.highlightLocalValue(pane, id, field, value)
	_, current, visible := m.localPattern()
	plain := ansi.Strip(value)
	start := 0
	if visible && m.focus == pane && current.pane == pane && current.id == id && current.field == field && current.offset <= len(plain) {
		column := ansi.StringWidth(plain[:current.offset])
		if column >= width-2 {
			start = max(0, column-width/3)
		}
	}
	prefix := ""
	if start > 0 {
		prefix = "…"
	}
	available := max(0, width-ansi.StringWidth(prefix))
	end := start + available
	suffix := ""
	if ansi.StringWidth(value) > end {
		suffix = "…"
		end = max(start, end-1)
	}
	return prefix + ansi.Cut(painted, start, end) + suffix
}

func (m *model) highlightLocalDetailField(field editorField, value string) string {
	if m.editor == nil || field == fieldDescription || field == fieldMoreFields || field == fieldComments {
		return value
	}
	if field == fieldTitle {
		source := m.editor.draft.Title
		start, prefix := 0, 0
		if m.editor.field == fieldTitle && m.editor.fieldViewportStart > 0 && strings.HasPrefix(ansi.Strip(value), "…") {
			runes := []rune(source)
			start = len(string(runes[:clamp(m.editor.fieldViewportStart, 0, len(runes))]))
			prefix = 1
		}
		return m.highlightLocalSlice(paneDetail, m.editor.id, field, value, source, start, prefix)
	}
	return m.highlightLocalValue(paneDetail, m.editor.id, field, value)
}
