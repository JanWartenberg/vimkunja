package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

func (e *taskEditor) descriptionView() *description.View {
	if e.nativeDraft != nil {
		e.nativeDraft.InsertionView(e.nativeDescription, e.field == fieldDescription && e.mode == editorInsert)
		return e.nativeDescription
	}
	if e.field == fieldDescription && e.mode == editorInsert {
		return nil
	}
	source := e.draft.Description
	if !descriptionHasHTML(source) && !descriptionHasUnterminatedMarkup(source) && !strings.Contains(source, "<!--") {
		e.nativeDescription = nil
		return nil
	}
	if e.nativeDescription == nil || e.nativeDescription.SourceHTML() != source {
		e.nativeDraft = description.NewDraft(description.Parse(source))
		e.nativeDescription = e.nativeDraft.View(description.ViewOptions{ASCII: tuiTheme.nativeASCII, Tabstop: e.nativeTabstop, OpaqueOpen: tuiTheme.opaqueOpen, OpaqueClose: tuiTheme.opaqueClose})
	}
	return e.nativeDescription
}
func nativeSpanStyle(role description.Style) lipgloss.Style {
	style := lipgloss.NewStyle().Reverse(role&description.SelectedStyle != 0).Bold(role&(description.Bold|description.Heading) != 0).Italic(role&description.Italic != 0).Underline(role&(description.Underline|description.Link) != 0).Strikethrough(role&description.Strike != 0)
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
	if e.nativeDraft != nil {
		frame = e.nativeDraft.Highlight(frame)
	}
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
	if e.nativePending != nil {
		if key == "y" || key == "enter" {
			a := *e.nativePending
			e.nativePending = nil
			m.applyNative(a, true, e.nativePendingInsert)
		} else if key == "n" || key == "esc" {
			e.nativePending = nil
			e.nativePendingInsert = false
			e.nativeDraft.EndUndoGroup()
		}
		return true
	}
	if key == "esc" {
		if e.nativeDraft.Selection() != nil {
			e.nativeDraft.CancelSelection()
			return true
		}
		if e.nativePrefix != "" || e.nativeCount > 0 {
			e.nativePrefix = ""
			e.nativeCount = 0
			return true
		}
		return false
	}
	if key == "enter" {
		e.nativePrefix = ""
		e.nativeCount = 0
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

	if (key == "ü" || key == "Ü") && e.nativeDraft.InCode() {
		e.nativePrefix = ""
		e.nativeCount = 0
		kind := description.CodeAfter
		if key == "Ü" {
			kind = description.CodeBefore
		}
		m.applyNative(description.Action{Kind: kind}, false, true)
		return true
	}
	if (key == "[" || key == "]") && e.nativeDraft.InCode() {
		e.nativePrefix = key
		return true
	}
	if e.nativePrefix != "" {
		prefix := e.nativePrefix
		e.nativePrefix = ""
		if prefix == "[" || prefix == "]" {
			if key == "space" || key == " " {
				kind := description.CodeAfter
				if prefix == "[" {
					kind = description.CodeBefore
				}
				m.applyNative(description.Action{Kind: kind}, false, false)
			}
			return true
		}
		if prefix == "g" && (key == "g" || key == "j" || key == "k" || key == "e" || key == "E") {
			key = prefix + key
		} else if prefix == "z" && (key == "h" || key == "l") {
			key = prefix + key
		} else if prefix == "d" && key == "d" {
			m.applyNative(description.Action{Kind: description.DeleteBlock}, false, false)
			e.nativeCount = 0
			return true
		} else if (prefix == "c" || prefix == "d") && (key == "i" || key == "a") {
			e.nativePrefix = prefix + key
			return true
		} else if prefix == "ci" || prefix == "ca" || prefix == "di" || prefix == "da" {
			if key == "w" {
				if prefix[0] == 'c' {
					e.nativeDraft.BeginUndoGroup()
				}
				if err := e.nativeDraft.SelectWord(prefix[1] == 'a'); err != nil {
					e.nativeDraft.EndUndoGroup()
					m.recordError(err)
					return true
				}
				kind := description.DeleteSelection
				if prefix[0] == 'c' {
					kind = description.ReplaceSelection
				}
				m.applyNative(description.Action{Kind: kind}, false, prefix[0] == 'c')
			}
			return true
		} else if prefix == "vi" || prefix == "va" {
			if key == "w" {
				if err := e.nativeDraft.SelectWord(prefix == "va"); err != nil {
					m.recordError(err)
				}
				e.nativeDraft.UpdateView(e.nativeDescription)
			}
			return true
		} else if prefix == "g" {
			e.gArmed = true
			return false
		} else {
			return true
		}
	}
	if m.nativeEditKey(key) {
		e.nativeCount = 0
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
		next := e.nativeDescription.Cursor().Anchor
		if !e.nativeDraft.AllowSelectionCursor(next) {
			bounded := e.nativeDraft.ClampCodeSelection(next)
			_ = e.nativeDraft.SetCursor(bounded)
			_ = e.nativeDescription.SetAnchor(bounded)
			m.toast = "Code selection boundary"
		} else {
			_ = e.nativeDraft.SetTarget(e.nativeDescription.Cursor())
		}
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
