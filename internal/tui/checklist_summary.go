package tui

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

func checklistSummary(checked, total int) string {
	return fmt.Sprintf("Checklist progress: %d/%d", checked, total)
}

func countCheckboxBlock(node *html.Node) (checked, total int) {
	if node.Type != html.ElementNode || node.Data != "ul" || descriptionAttribute(node, "data-type") != "taskList" {
		return 0, 0
	}
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if isCheckboxEntry(n) {
			total++
			if strings.EqualFold(descriptionAttribute(n, "data-checked"), "true") {
				checked++
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return
}

func isCheckboxEntry(node *html.Node) bool {
	return node.Type == html.ElementNode && node.Data == "li" && descriptionAttribute(node, "data-type") == "taskItem"
}

// wrapTaskDescription adds display-only summaries after wrapping. Source offsets
// remain valid for plain-text editing; synthetic rows have no cursor position.
// Comments and other multiline inputs deliberately do not use this projection.
func wrapTaskDescription(source string, width int) []descriptionLine {
	var out []descriptionLine
	appendLine := func(text string, offset int, summary bool) {
		for _, line := range wrapDescription(text, width) {
			if summary {
				line.start, line.end = -1, -1
				line.text = tuiTheme.checkboxListSummary.render(line.text)
			} else {
				line.start += offset
				line.end += offset
			}
			out = append(out, line)
		}
	}
	if descriptionHasHTML(source) {
		summaries := map[int]bool{}
		text := projectDescription(source, summaries)
		offset := 0
		for index, line := range strings.Split(text, "\n") {
			appendLine(line, offset, summaries[index])
			offset += len([]rune(line)) + 1
		}
		return out
	}
	if descriptionHasUnterminatedMarkup(source) {
		return wrapDescription(descriptionTerminalText(source), width)
	}
	checked, total, offset := 0, 0, 0
	flush := func() {
		if total > 0 {
			appendLine(checklistSummary(checked, total), 0, true)
		}
		checked, total = 0, 0
	}
	for _, line := range strings.Split(removeDangerousDescriptionControls(source), "\n") {
		marker := strings.TrimLeftFunc(line, unicode.IsSpace)
		isOpen, isDone := strings.HasPrefix(marker, "[ ]") || strings.HasPrefix(marker, "[]"), strings.HasPrefix(marker, "[x]")
		if !isOpen && !isDone {
			flush()
		} else {
			total++
			if isDone {
				checked++
			}
		}
		appendLine(line, offset, false)
		offset += len([]rune(line)) + 1
	}
	flush()
	return out
}

// descriptionViewportStart keeps the source cursor and its following summary
// together when they fit. Synthetic rows remain display-only, not cursor targets.
func descriptionViewportStart(lines []descriptionLine, rows, scroll, cursorRow int, revealEnd bool) int {
	start := clamp(scroll, 0, max(0, len(lines)-max(0, rows)))
	if rows <= 0 {
		return start
	}
	// At the end in Normal Mode, allow reading the trailing summary even
	// when there is not enough room to keep the source cursor on screen.
	if revealEnd && len(lines) > 0 && lines[len(lines)-1].start == -1 {
		return max(0, len(lines)-rows)
	}
	last := cursorRow
	for last+1 < len(lines) && lines[last+1].start == -1 && last-cursorRow+1 < rows {
		last++
	}
	if cursorRow < start {
		start = cursorRow
	}
	if last >= start+rows {
		start = last - rows + 1
	}
	return start
}

// scrollDescriptionViewport reads projected rows without selecting synthetic
// content. The next editing or navigation action resumes source-cursor tracking.
func (m *model) scrollDescriptionViewport(delta int) {
	e := m.editor
	w, _ := m.viewportSize()
	var lines []descriptionLine
	var rows, start int
	if !m.compactDetail() {
		layout := m.taskDetailDescriptionLayout(w-4, m.detailTextHeight())
		lines, rows, start = layout.lines, layout.rows, layout.start
	} else {
		width := w - 4
		rows = m.compactDescriptionRows()
		if e.descriptionActive && m.detailTextHeight() < 12 {
			rows = max(1, m.detailTextHeight()-2)
		} else {
			width -= 2
		}
		lines = wrapTaskDescriptionForEditor(e, e.draft.Description, width)
		if len(lines) > rows {
			lines = wrapTaskDescriptionForEditor(e, e.draft.Description, max(1, width-2))
		}
		start = e.descScroll
	}
	e.descScroll = clamp(start+delta, 0, max(0, len(lines)-rows))
	e.descViewportScroll = true
}

func wrapTaskDescriptionForEditor(e *taskEditor, source string, width int) []descriptionLine {
	if e != nil && e.descriptionView() != nil {
		return renderNativeDescriptionFrame(e, e.nativeDescription.Layout(width, e.field == fieldDescription && e.descriptionActive), width)
	}
	if e != nil && e.field == fieldDescription && e.mode == editorInsert {
		return wrapDescription(source, width)
	}
	return wrapTaskDescription(source, width)
}

func layoutTaskDescription(source string, width, rows, scroll, cursor int, editor ...*taskEditor) multilineTextLayout {
	var e *taskEditor
	if len(editor) > 0 {
		e = editor[0]
	}
	if e != nil && e.descriptionView() != nil {
		return e.nativeDescriptionLayout(width, rows)
	}
	lines := wrapTaskDescriptionForEditor(e, source, width)
	layout := multilineTextLayout{lines: lines, start: clamp(scroll, 0, max(0, len(lines)-max(0, rows)))}
	for row, line := range lines {
		if cursor >= line.start && cursor <= line.end {
			layout.cursorRow, layout.cursorColumn = row, cursor-line.start
			break
		}
	}
	return layout
}
