package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

func (e *taskEditor) descriptionView() *description.View {
	if e.field == fieldDescription && e.mode == editorInsert {
		return nil
	}
	source := e.draft.Description
	if !descriptionHasHTML(source) && !descriptionHasUnterminatedMarkup(source) && !strings.Contains(source, "<!--") {
		e.nativeDescription = nil
		return nil
	}
	if e.nativeDescription == nil || e.nativeDescription.SourceHTML() != source {
		e.nativeDescription = description.NewView(description.Parse(source), description.ViewOptions{ASCII: tuiTheme.nativeASCII, Tabstop: 4, OpaqueOpen: tuiTheme.opaqueOpen, OpaqueClose: tuiTheme.opaqueClose})
	}
	return e.nativeDescription
}
func nativeSpanStyle(role description.Style) lipgloss.Style {
	style := lipgloss.NewStyle().Bold(role&(description.Bold|description.Heading) != 0).Italic(role&description.Italic != 0).Underline(role&(description.Underline|description.Link) != 0).Strikethrough(role&description.Strike != 0)
	if role&description.Code != 0 {
		style = style.Background(lipgloss.Color(tuiTheme.codeBackground))
	}
	if role&description.InlineCode != 0 {
		style = style.Underline(true)
	}
	if role&description.OpaqueStyle != 0 {
		style = style.Foreground(lipgloss.Color(tuiTheme.opaqueForeground)).Background(lipgloss.Color(tuiTheme.opaqueBackground))
	}
	if role&description.Summary != 0 {
		format := tuiTheme.checkboxListSummary
		style = style.Bold(format.bold).Underline(format.underline)
		if format.foreground != "" {
			style = style.Foreground(lipgloss.Color(format.foreground))
		}
	}
	return style
}
func renderNativeDescriptionFrame(e *taskEditor, f description.Frame, width int) []descriptionLine {
	lines := make([]descriptionLine, len(f.Rows))
	for i, row := range f.Rows {
		var text strings.Builder
		column := 0
		code := false
		cursorPainted := false
		for _, span := range row.Spans {
			style := nativeSpanStyle(span.Style)
			code = code || span.Style&description.Code != 0
			if !e.nativeFocused || !e.descriptionActive || i != f.CursorY || f.CursorX < column || f.CursorX >= column+ansi.StringWidth(span.Text) {
				text.WriteString(style.Render(span.Text))
				column += ansi.StringWidth(span.Text)
				continue
			}
			it := uniseg.NewGraphemes(span.Text)
			for it.Next() {
				g := it.Str()
				w := ansi.StringWidth(g)
				cellStyle := style
				if e.nativeFocused && e.descriptionActive && i == f.CursorY && column == f.CursorX {
					cellStyle = cellStyle.Reverse(true)
					cursorPainted = true
				}
				text.WriteString(cellStyle.Render(g))
				column += w
			}
		}
		if e.nativeFocused && e.descriptionActive && i == f.CursorY && f.CursorX == column && !cursorPainted && column < width {
			text.WriteString(lipgloss.NewStyle().Reverse(true).Render(" "))
			column++
		}
		if code && column < width {
			text.WriteString(nativeSpanStyle(description.Code).Render(strings.Repeat(" ", width-column)))
		}
		lines[i] = descriptionLine{text: text.String(), start: -1, end: -1}
	}
	return lines
}
func (e *taskEditor) nativeDescriptionLayout(width, height int) multilineTextLayout {
	view := e.descriptionView()
	focused := e.field == fieldDescription && e.descriptionActive
	frame := view.Viewport(width, max(1, height), focused)
	if focused || e.descViewportScroll {
		e.descScroll = frame.Top
	}
	return multilineTextLayout{lines: renderNativeDescriptionFrame(e, frame, width), start: frame.Top, cursorRow: frame.CursorY, cursorColumn: frame.CursorX}
}
func (m *model) nativeDescriptionGeometry() (int, int) {
	width, _ := m.viewportSize()
	width -= 4
	if !m.compactDetail() {
		layout := m.taskDetailDescriptionLayout(width, m.detailTextHeight())
		if layout.scrollable {
			width -= 2
		}
		return max(1, width), max(1, layout.rows)
	}
	rows := m.compactDescriptionRows()
	if m.editor.descriptionActive && m.detailTextHeight() < 12 {
		rows = max(1, m.detailTextHeight()-2)
	} else {
		width -= 2
	}
	if len(m.editor.descriptionView().Layout(max(1, width), true).Rows) > rows {
		width -= 2
	}
	return max(1, width), max(1, rows)
}
func (m *model) nativeDescriptionKey(key string) bool {
	e := m.editor
	if e.field != fieldDescription || e.descriptionView() == nil {
		return false
	}
	if key == "esc" {
		if e.nativePrefix != "" || e.nativeCount > 0 {
			e.nativePrefix = ""
			e.nativeCount = 0
			return true
		}
		return false
	}
	if key == "enter" {
		e.descriptionActive = true
		e.moreFieldsExpanded = false
		m.ensureDetailFieldVisible()
		return true
	}
	if !e.descriptionActive && e.descriptionNavigated {
		if key == "j" || key == "down" || key == "k" || key == "up" {
			return false
		}
	}

	if e.nativePrefix != "" {
		prefix := e.nativePrefix
		e.nativePrefix = ""
		if prefix == "g" && (key == "g" || key == "j" || key == "k" || key == "e" || key == "E") {
			key = prefix + key
		} else if prefix == "z" && (key == "h" || key == "l") {
			key = prefix + key
		} else if prefix == "g" {
			e.gArmed = true
			return false
		} else {
			return true
		}
	}
	if key == "i" || key == "a" || key == "A" || key == "I" || key == "o" || key == "O" || key == "x" || key == "X" || key == "d" || key == "D" || key == "c" || key == "C" || key == "s" || key == "r" || key == "R" || key == "v" || key == "V" || key == "u" || key == "ctrl+r" {
		e.nativePrefix = ""
		e.nativeCount = 0
		m.recordError(fmt.Errorf("Description is read-only; editing is not available yet."))
		return true
	}
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' && (key != "0" || e.nativeCount > 0) {
		e.nativeCount = min(100000, e.nativeCount*10+int(key[0]-'0'))
		return true
	}
	if key == "g" || key == "z" {
		e.nativePrefix = key
		return true
	}
	width, height := m.nativeDescriptionGeometry()
	if key == "ctrl+e" || key == "ctrl+y" {
		delta := max(1, e.nativeCount)
		if key == "ctrl+y" {
			delta = -delta
		}
		frame := e.nativeDescription.Scroll(delta, width, height)
		e.descScroll = frame.Top
		e.descViewportScroll = true
		e.nativeCount = 0
		return true
	}
	if key == "G" && e.nativeCount > 0 {
		key = "gg"
	}
	if e.nativeDescription.Move(key, max(1, e.nativeCount), width, height) {
		e.descriptionActive = true
		e.descriptionNavigated = true
		e.descViewportScroll = false
		e.nativeCount = 0
		e.descScroll = e.nativeDescription.Viewport(width, height, true).Top
		return true
	}
	e.nativeCount = 0
	return true
}
