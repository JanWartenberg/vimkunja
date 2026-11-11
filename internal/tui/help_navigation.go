package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type helpEntry struct{ keys, text string }
type helpSection struct {
	title   string
	entries []helpEntry
}
type helpPage struct {
	title    string
	parent   int
	children []int
	sections []helpSection
}
type helpLocation struct{ page, selected, offset int }
type helpBrowser struct {
	helpLocation
	saved   map[int]helpLocation
	history []helpLocation
	at      int
	g       bool
}

func newHelpBrowser() *helpBrowser {
	loc := helpLocation{page: -1}
	return &helpBrowser{helpLocation: loc, saved: map[int]helpLocation{}, history: []helpLocation{loc}}
}
func (b *helpBrowser) children() []int {
	if b.page == -1 {
		return []int{0, 1, 2, 3, 4, 5}
	}
	return helpPages[b.page].children
}
func (b *helpBrowser) remember() {
	b.saved[b.page] = b.helpLocation
	b.history[b.at] = b.helpLocation
}
func (b *helpBrowser) open(page int) {
	b.remember()
	loc, ok := b.saved[page]
	if !ok {
		loc = helpLocation{page: page}
	}
	b.helpLocation = loc
	b.history = append(b.history[:b.at+1], loc)
	if len(b.history) > 100 {
		b.history = b.history[len(b.history)-100:]
	}
	b.at = len(b.history) - 1
	b.g = false
}
func (b *helpBrowser) travel(delta int) {
	next := b.at + delta
	if next < 0 || next >= len(b.history) {
		return
	}
	b.remember()
	b.at = next
	b.helpLocation = b.history[next]
	b.g = false
}

// rows uses a hanging indent so keys and explanations wrap independently.
func (b *helpBrowser) rows(width int) ([]string, []int) {
	width = max(1, width)
	rows := []string{}
	starts := []int{}
	add := func(text, indent string) {
		for _, row := range strings.Split(ansi.Wrap(text, max(1, width-lipgloss.Width(indent)), ""), "\n") {
			rows = append(rows, indent+row)
		}
	}
	if children := b.children(); len(children) > 0 {
		for i, child := range children {
			starts = append(starts, len(rows))
			marker := "  "
			if i == b.selected {
				marker = "> "
			}
			add(marker+helpPages[child].title, "")
		}
	} else {
		for _, section := range helpPages[b.page].sections {
			add(section.title, "")
			for _, entry := range section.entries {
				if entry.keys != "" {
					add(entry.keys, "")
				}
				add(entry.text, "  ")
				rows = append(rows, "")
			}
		}
	}
	if len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	return rows, starts
}
func (b *helpBrowser) breadcrumb(width int) string {
	title := "Help"
	if b.page >= 0 {
		if helpPages[b.page].parent >= 0 {
			title += " / " + helpPages[helpPages[b.page].parent].title
		}
		title += " / " + helpPages[b.page].title
	}
	if ansi.StringWidth(title) > width {
		return ansi.TruncateLeft(title, ansi.StringWidth(title)-max(1, width)+1, "…")
	}
	return title
}
func (m *model) helpSize() (int, int) {
	w, h := m.viewportSize()
	return max(1, w-4), max(1, max(4, h-3-m.detailPaneTop())-3)
}
func (b *helpBrowser) layout(width, height int) ([]string, int) {
	rows, starts := b.rows(width)
	b.offset = clamp(b.offset, 0, max(0, len(rows)-height))
	if len(starts) > 0 {
		b.selected = clamp(b.selected, 0, len(starts)-1)
		start := starts[b.selected]
		end := len(rows)
		if b.selected+1 < len(starts) {
			end = starts[b.selected+1]
		}
		if start < b.offset {
			b.offset = start
		}
		if end > b.offset+height {
			b.offset = min(start, end-height)
		}
	}
	return rows, max(0, len(rows)-height)
}
func renderHelpPane(pane *infoPane, width, height int) string {
	b := pane.help
	rows, _ := b.layout(width, max(1, height-1))
	lines := []string{lipgloss.NewStyle().Bold(true).Render(b.breadcrumb(width))}
	lines = append(lines, rows[b.offset:]...)
	return fitBlock(lines, width, height)
}
func (m *model) updateHelpBrowser(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	b := m.helpPane.help
	width, height := m.helpSize()
	_, limit := b.layout(width, height)
	children := b.children()
	s := k.String()
	if b.g {
		b.g = false
		if s == "g" {
			s = "home"
		} else {
			return m, nil
		}
	}
	step := max(1, height/2)
	delta := 0
	switch s {
	case "ctrl+o":
		b.travel(-1)
	case "ctrl+i", "tab":
		b.travel(1)
	case "-", "backspace":
		if b.page >= 0 {
			b.open(helpPages[b.page].parent)
		}
	case "enter":
		if len(children) > 0 {
			b.open(children[b.selected])
		}
	case "g":
		b.g = true
	case "j", "down":
		delta = 1
	case "k", "up":
		delta = -1
	case "ctrl+d", "pgdown", "pagedown":
		delta = step
	case "ctrl+u", "pgup", "pageup":
		delta = -step
	case "G", "end":
		delta = limit + len(children)
	case "home":
		delta = -limit - len(children)
	}
	// Page changes restore their own location; only movement uses the old bounds.
	if delta != 0 {
		if len(children) > 0 {
			b.selected = clamp(b.selected+delta, 0, len(children)-1)
		} else {
			b.offset = clamp(b.offset+delta, 0, limit)
		}
	}
	b.layout(width, height)
	m.helpPane.offset = b.offset
	return m, nil
}

func helpHints() string {
	return strings.Join([]string{":q close", "j/k move", "Enter open", "Ctrl+O/I history", "-/Backspace parent", "Ctrl+W panes"}, footerSeparator)
}

// The editor and Comment workspace stay live while help covers the bottom pane.
// Keeping their identities preserves drafts, undo, and in-flight read results.
type helpOrigin struct {
	focus  paneID
	editor *taskEditor
	info   *infoPane
}

func (m *model) closeHelp() {
	origin := m.helpOrigin
	m.helpPane = nil
	m.helpOrigin = nil
	m.focus = paneTasks
	if origin == nil {
		return
	}
	switch origin.focus {
	case paneProjects:
		if !m.projectsPaneCollapsed {
			m.focus = paneProjects
		}
	case paneDetail:
		if m.editor != nil && m.editor == origin.editor && m.showDetail {
			m.focus = paneDetail
		}
	case paneInfo:
		if m.infoPane != nil && m.infoPane == origin.info {
			m.focus = paneInfo
		}
	}
}

// Context navigation must not strand help over an invisible or unrelated draft.
func (m *model) leaveHelpForNavigation() bool {
	if m.helpPane == nil {
		return true
	}
	if m.helpHasDraft() {
		m.recordError(errUnsavedNavigation)
		return false
	}
	m.leaveHelpForContext()
	return true
}
func (m *model) helpHasDraft() bool {
	return (m.editor != nil && m.editor.dirty()) || (m.comments != nil && m.comments.composer != nil)
}

// A clean Comments workspace belongs to its Task; intentional navigation closes
// it. Plain :q on help keeps it so the user can resume that same workspace.
func (m *model) leaveHelpForContext() {
	m.closeHelp()
	if m.comments != nil && m.comments.composer == nil {
		m.closeComments()
	}
}
