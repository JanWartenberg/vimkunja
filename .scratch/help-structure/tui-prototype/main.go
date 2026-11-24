// THROWAWAY UI PROTOTYPE: does a fixed topic overview with always-visible
// explanations work as Vimkunja's in-app help? No database or production wiring.
// Run: go run ./.scratch/help-structure/tui-prototype
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type entry struct{ action, keys, mode, explanation string }
type section struct {
	title   string
	entries []entry
}
type topic struct {
	name, summary string
	sections      []section
}

var topics = []topic{
	{"Navigation & Views", "Projects, pane focus, Overview, and Kanban.", []section{
		{"PANES & PROJECTS", []entry{
			{"Change pane focus", "Ctrl+W h/j/k/l", "Global", "Press Ctrl+W, then a direction within two seconds. Ctrl+W w cycles open panes."},
			{"Fold a Project branch", "zo / zc / za", "Projects", "Open, close, or toggle the selected branch. h closes a branch or selects its parent; l opens it or selects its first child."},
			{"Open Overview", "- / Space+a", "Task context", "- opens Overview. Space+a toggles between Overview and the remembered Project context."},
		}},
		{"KANBAN", []entry{
			{"Open a Kanban View", ":kanban", "Project context", "Select or provision a manual Kanban View for the current Project."},
			{"Select a Task", "h/l, j/k", "Kanban", "h/l selects a Bucket; j/k selects a Task within that Bucket."},
			{"Move a Task", "Shift+H / Shift+L", "Kanban", "Move the selected Task to an adjacent Bucket. To choose any target: m, then j/k and Enter. Esc cancels."},
		}},
	}},
	{"Tasks & Detail", "Create Tasks, edit fields, manage Comments, and save.", []section{
		{"TASKS", []entry{
			{"Create a Task", "o", "Tasks", "Start a new Task in the selected Project."},
			{"Edit a title inline", "i / Shift+I", "List / Overview", "Edit at the end or beginning of the title. Enter saves; Esc discards the inline edit."},
			{"Open Task Detail", "Enter", "Tasks", "Open the selected Task. Use j/k to select fields and i to edit or open a picker."},
			{"Jump to a field", "gt / gd / gp / gl / gn", "Detail", "Edit Title, Due Date, Priority, or Labels; gn focuses Description."},
			{"Open Comments", "gc", "Detail", "Browse the thread with j/k. Use a, i, or o to append a Comment."},
		}},
		{"SAVE & DISCARD", []entry{
			{"Save a Task", ":w / :wq", "Detail", ":w saves and keeps Detail open. :wq saves and closes it. Description edits belong to this Task draft."},
			{"Discard changes", ":q! / :e!", "Detail", ":q! discards and closes. :e! discards the complete draft and reloads an existing Task without closing Detail."},
			{"Delete a Task", "dd, then y", "Tasks", "Permanently delete after confirmation. n or Esc cancels. Tasks with Time Entries cannot be deleted here."},
		}},
	}},
	{"Shared text editing", "Modes, motions, selection, and system clipboard.", []section{
		{"TITLE, DESCRIPTION & COMMENT DRAFTS", []entry{
			{"Start typing", "i / a", "Text Normal", "Insert at or after the cursor. Esc returns to Normal mode."},
			{"Move by word", "w / b / e", "Text Normal", "Move to the next word, previous word, or word end. Counts repeat motions: 2w moves two words."},
			{"Change a word", "ciw / caw", "Text Normal", "Change the inner word or around-word selection and enter Insert. Esc cancels an unfinished prefix."},
			{"Select text", "v / V", "Text Normal", "Select characters or lines. In rich text, V selects logical blocks; Code uses source-line selection. o swaps the active end; Esc cancels."},
			{"Copy text", "Space+y", "Text Normal / Visual", "Copy the selection, or current line / Native paragraph, to the system clipboard. This does not export the complete Task."},
			{"Paste text", "Space+p", "Text Normal / Visual", "Paste at the cursor or replace the selection; stay in Normal mode. Insert mode also supports Ctrl+V."},
		}},
	}},
	{"Rich text", "Formatting, Links, lists, Code, undo, and recovery.", []section{
		{"ORIENTATION", []entry{
			{"Enter the Description", "gn, then Enter", "Detail", "Activate document navigation. Empty, plain-text, and HTML Descriptions use the same editor. Edits stay in the Task draft until :w."},
			{"Navigate the document", "j/k / gj/gk", "Normal", "j/k moves between logical blocks; inside Code it moves between source lines. gj/gk moves between wrapped visible rows."},
		}},
		{"FORMATTING", []entry{
			{"Format selected text", "Space+b / i / u / s", "Visual", "Apply bold, italic, underline, or strike. Space+0 clears marks. Example: v, select text, then Space b."},
			{"Make a heading", "Space+1 / 2 / 3 / 0", "Normal", "Convert the current block to H1, H2, H3, or an ordinary paragraph."},
		}},
		{"LINKS", []entry{
			{"Edit a Link URL", "Space+l", "In a Link", "Open the destination URL. An empty URL removes the Link. Edit its visible text directly in the document."},
			{"Insert a Link", "Space+l", "Outside a Link", "Insert a Link using the current selection. Without a selection, the prompt asks for a URL and then visible text."},
			{"Copy a Link URL", "Space+e", "In a Link", "Copy the destination URL to the system clipboard. Space+Y remains an alias."},
		}},
		{"LISTS & CODE", []entry{
			{"Create a Checkbox list", "Space+x", "Normal", "Toggle Checkbox-list structure. On an existing Checkbox entry, x toggles its checked state."},
			{"Delete a Code line", "dd", "Code Normal", "Delete the current source line. Deleting the final nonempty line leaves an empty Code block; another dd deletes that block. Outside Code, dd deletes a logical block."},
			{"Split or break a paragraph", "Enter / Shift+Enter", "Insert", "Enter splits a paragraph; Shift+Enter inserts a hard break. Inside Code, Enter inserts a source newline."},
		}},
		{"UNDO & RECOVERY", []entry{
			{"Undo a Description edit", "u / Ctrl+R", "Normal", "Undo or redo unsaved Description changes. This local history is separate from undoing persisted Task actions."},
			{"Recover blocked HTML", ":descriptionrecover", "Blocked Description", "Request recovery of safe text. Review confirmation before replacing preserved content. Save the Task to persist the result."},
			{"Clear blocked HTML", ":descriptionclear", "Blocked Description", "Request an empty draft instead of blocked HTML. Confirmation is required; the result remains unsaved until the Task is saved."},
		}},
	}},
	{"Search & filters", "Find Tasks, Projects, and Labels; narrow the Overview.", []section{
		{"GLOBAL SEARCH", []entry{
			{"Open Global Search", "Space+f / :search", "Global", "Find Tasks, Projects, or Labels. Matching is case-insensitive substring search; whitespace-separated terms are combined with AND within one result kind."},
			{"Open a result", "Enter", "Search", "Arrows or Ctrl+N/P selects while typing. Esc enters Search Normal mode; j/k then selects. Enter activates the result."},
			{"Return to Search", "Ctrl+O", "After a Search result", "Return to the latest session-local Search snapshot. This is not a general navigation history."},
		}},
		{"FILTERS", []entry{
			{"Filter Tasks", "f o / a / v / t / i", "List / Overview", "Choose open, all, overdue, due today, or in-progress Tasks. These shortcuts do not filter Kanban."},
		}},
	}},
	{"Commands & quitting", "Command completion, errors, closing panes, and quitting.", []section{
		{"EX COMMANDS", []entry{
			{"Recall / complete a command", "Up/Down / Tab", "Ex prompt", "Recall session history with Up/Down. Tab completes command names."},
			{"Read errors", ":errors", "Global", "Open the session error log."},
			{"Close a pane", ":q", "Normal / Ex", "Close the current pane or quit. Dirty editors require saving or an explicit discard command."},
			{"Quit Vimkunja", ":qa", "Ex", ":qa quits; :qa! explicitly discards unsaved editor changes; :wqa saves and quits."},
		}},
	}},
}

// Rich-text sections become separate pages, so the reference stays short.
func init() {
	rich := topics[3]
	topics[3].sections = nil
	topics[3].summary = "Write and format Task Descriptions."
	for _, index := range []int{2, 3, 4, 1} {
		group := rich.sections[index]
		names := map[int]string{1: "Formatting", 2: "Links", 3: "Lists & Code", 4: "Undo & recovery"}
		topics = append(topics, topic{names[index], "", []section{group}})
	}
	topics = append(topics, topic{"Editing basics", "", rich.sections[:1]})
	summaries := []string{
		"Projects, panes, List, Overview, Kanban.",
		"Create, edit, save, delete. Labels and Comments.",
		"Modes, motions, selection, clipboard.",
		"Formatting, Links, lists, Code, recovery.",
		"Global Search and Task filters.",
		"Ex commands, errors, closing and quitting.",
	}
	for i, summary := range summaries {
		topics[i].summary = summary
	}
	replacements := map[string]string{
		"Press Ctrl+W, then a direction within two seconds. Ctrl+W w cycles open panes.":                                                                                        "Ctrl+W, then a direction within two seconds. Ctrl+W w cycles panes.",
		"Select or provision a manual Kanban View for the current Project.":                                                                                                     "Open the Project's Kanban View; create it if missing.",
		"Start a new Task in the selected Project.":                                                                                                                             "Create in the selected Project.",
		"Open the selected Task. Use j/k to select fields and i to edit or open a picker.":                                                                                      "j/k selects a field; i edits it or opens its picker.",
		"Browse the thread with j/k. Use a, i, or o to append a Comment.":                                                                                                       "j/k selects a Comment; a adds one.",
		":w saves and keeps Detail open. :wq saves and closes it. Description edits belong to this Task draft.":                                                                 ":w saves. :wq saves and closes Detail.",
		"Activate document navigation. Empty, plain-text, and HTML Descriptions use the same editor. Edits stay in the Task draft until :w.":                                    "Enter the Description editor. :w saves changes with the Task.",
		"Open the destination URL. An empty URL removes the Link. Edit its visible text directly in the document.":                                                              "Edit the destination. Empty URL removes the Link. Edit Link text in the document.",
		"Insert a Link using the current selection. Without a selection, the prompt asks for a URL and then visible text.":                                                      "Turn selected text into a Link. Without a selection: enter URL, then Link text.",
		"Copy the destination URL to the system clipboard. Space+Y remains an alias.":                                                                                           "Copy the destination to the clipboard. Alias: Space+Y.",
		"Delete the current source line. Deleting the final nonempty line leaves an empty Code block; another dd deletes that block. Outside Code, dd deletes a logical block.": "Delete one source line. The last line leaves empty Code; another dd removes it. Outside Code: delete the current block.",
		"Undo or redo unsaved Description changes. This local history is separate from undoing persisted Task actions.":                                                         "Undo / redo Description draft changes. Task actions have separate history.",
		"Request recovery of safe text. Review confirmation before replacing preserved content. Save the Task to persist the result.":                                           "Recover safe text after confirmation. :w saves the result.",
		"Request an empty draft instead of blocked HTML. Confirmation is required; the result remains unsaved until the Task is saved.":                                         "Replace blocked HTML with an empty draft after confirmation. :w saves it.",
		"Find Tasks, Projects, or Labels. Matching is case-insensitive substring search; whitespace-separated terms are combined with AND within one result kind.":              "Search Tasks, Projects, and Labels. Case-insensitive; all query words must match within one result kind.",
		"Return to the latest session-local Search snapshot. This is not a general navigation history.":                                                                         "Return to the last Search result list.",
	}
	for i := range topics {
		for j := range topics[i].sections {
			for k := range topics[i].sections[j].entries {
				e := &topics[i].sections[j].entries[k]
				if shorter, ok := replacements[e.explanation]; ok {
					e.explanation = shorter
				}
			}
		}
	}
}

var heading = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
var muted = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
var keyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
var selectedStyle = lipgloss.NewStyle().Reverse(true)

// History stores reading positions; parent navigation follows document structure.
type location struct{ page, selected, offset int }
type model struct {
	width, height int
	location
	saved    map[int]location
	history  []location
	at       int
	colon, g bool
}

func children(page int) []int {
	if page == -1 {
		return []int{0, 1, 2, 3, 4, 5}
	}
	if page == 3 {
		return []int{10, 9, 6, 7, 8}
	}
	return nil
}
func parent(page int) int {
	if page >= 6 {
		return 3
	}
	return -1
}
func (m model) Init() tea.Cmd { return nil }
func (m *model) remember()    { m.saved[m.page] = m.location; m.history[m.at] = m.location }
func (m *model) open(page int) {
	if page == m.page {
		return
	}
	m.remember()
	next, ok := m.saved[page]
	if !ok {
		next = location{page: page}
	}
	m.location = next
	m.history = append(m.history[:m.at+1], next)
	m.at++
}
func (m *model) travel(delta int) {
	next := m.at + delta
	if next < 0 || next >= len(m.history) {
		return
	}
	m.remember()
	m.at = next
	m.location = m.history[m.at]
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		k := msg.String()
		if k == "ctrl+c" {
			return m, tea.Quit
		}
		if m.colon {
			m.colon = false
			switch k {
			case "q":
				return m, tea.Quit
			case "h":
				m.open(-1)
			}
			return m, nil
		}
		links := children(m.page)
		switch k {
		case ":":
			m.colon = true
		case "q":
			return m, tea.Quit
		case "ctrl+o":
			m.travel(-1)
		// Most terminals encode Ctrl+I as Tab.
		case "ctrl+i", "tab":
			m.travel(1)
		case "-", "backspace":
			if m.page != -1 {
				m.open(parent(m.page))
			}
		case "enter":
			if len(links) > 0 {
				m.open(links[m.selected])
			}
		case "j", "down":
			if len(links) > 0 {
				m.selected = min(m.selected+1, len(links)-1)
			} else {
				m.offset++
			}
		case "k", "up":
			if len(links) > 0 {
				m.selected = max(0, m.selected-1)
			} else {
				m.offset--
			}
		case "ctrl+d", "pgdown":
			if len(links) > 0 {
				m.selected = min(m.selected+3, len(links)-1)
			} else {
				m.offset += max(1, (m.height-6)/2)
			}
		case "ctrl+u", "pgup":
			if len(links) > 0 {
				m.selected = max(0, m.selected-3)
			} else {
				m.offset -= max(1, (m.height-6)/2)
			}
		case "G", "end":
			if len(links) > 0 {
				m.selected = len(links) - 1
			} else {
				m.offset = 1 << 20
			}
		case "home":
			m.selected = 0
			m.offset = 0
		case "g":
			if m.g {
				m.selected = 0
				m.offset = 0
			}
		}
		m.g = k == "g" && !m.g
	}
	m.fit()
	return m, nil
}
func wrapped(s string, width int) []string {
	return strings.Split(ansi.Wrap(s, max(1, width), ""), "\n")
}
func (m model) body(width int) ([]string, int) {
	lines := []string{}
	selectedLine := 0
	if m.page >= 0 && topics[m.page].summary != "" {
		lines = append(lines, wrapped(topics[m.page].summary, width)...)
		lines = append(lines, "")
	}
	for i, id := range children(m.page) {
		prefix := "  "
		if i == m.selected {
			prefix = "> "
			selectedLine = len(lines)
		}
		label := prefix + topics[id].name
		if i == m.selected {
			label = selectedStyle.Render(label)
		}
		lines = append(lines, label)
		if m.page == -1 {
			for _, line := range wrapped(topics[id].summary, max(1, width-2)) {
				lines = append(lines, "  "+line)
			}
		}
		lines = append(lines, "")
	}
	if m.page == -1 {
		return lines, selectedLine
	}
	for _, s := range topics[m.page].sections {
		// A leaf page already names its only section in the breadcrumb.
		if len(topics[m.page].sections) > 1 || m.page < 6 {
			lines = append(lines, heading.Render(s.title), "")
		}
		for _, e := range s.entries {
			if width >= 64 && ansi.StringWidth(e.action) <= 29 && ansi.StringWidth(e.keys) <= width-32 {
				lines = append(lines, fmt.Sprintf("%-31s%s", e.action, keyStyle.Render(e.keys)))
			} else {
				lines = append(lines, wrapped(e.action, width)...)
				lines = append(lines, wrapped(keyStyle.Render(e.keys), width)...)
			}
			// Mode stays next to its explanation, without a separate metadata row.
			note := e.mode + ": " + e.explanation
			for _, line := range wrapped(note, max(1, width-2)) {
				lines = append(lines, "  "+line)
			}
			lines = append(lines, "")
		}
	}
	return lines, selectedLine
}
func (m *model) fit() {
	lines, selectedLine := m.body(max(1, m.width-2))
	visible := max(1, m.height-6)
	m.offset = min(max(0, m.offset), max(0, len(lines)-visible))
	if len(children(m.page)) > 0 {
		if selectedLine < m.offset {
			m.offset = selectedLine
		}
		if selectedLine+1 >= m.offset+visible {
			m.offset = max(0, selectedLine+2-visible)
		}
	}
}
func (m model) View() tea.View {
	m.fit()
	w, h := max(1, m.width), max(7, m.height)
	width := max(1, w-2)
	lines, _ := m.body(width)
	title := "HELP"
	if m.page >= 6 {
		title += " > Rich text > " + topics[m.page].name
	} else if m.page >= 0 {
		title += " > " + topics[m.page].name
	}
	content := []string{heading.Render(title), muted.Render(strings.Repeat("─", width))}
	visible := max(1, h-6)
	for i := 0; i < visible; i++ {
		line := ""
		if m.offset+i < len(lines) {
			line = lines[m.offset+i]
		}
		content = append(content, line)
	}
	content = append(content, muted.Render(strings.Repeat("─", width)))
	controls := "j/k scroll"
	if len(children(m.page)) > 0 {
		controls = "j/k select · Enter open"
	}
	content = append(content, controls+" · - up · :q close")
	back, forward := "Ctrl+O back", "Ctrl+I forward"
	if m.at == 0 {
		back = "[Ctrl+O back]"
	}
	if m.at == len(m.history)-1 {
		forward = "[Ctrl+I forward]"
	}
	content = append(content, back+" · "+forward)
	state := fmt.Sprintf("%d–%d/%d · prototype", m.offset+1, min(m.offset+visible, len(lines)), len(lines))
	if m.colon {
		state = ":h overview · :q close"
	}
	content = append(content, muted.Render(state))
	for i := range content {
		content[i] = ansi.Truncate(content[i], w, "")
	}
	v := tea.NewView(strings.Join(content, "\n"))
	v.AltScreen = true
	return v
}
func main() {
	snapshot := flag.Bool("snapshot", false, "Print a noninteractive frame")
	width := flag.Int("width", 100, "Snapshot terminal width")
	height := flag.Int("height", 30, "Snapshot terminal height")
	page := flag.Int("topic", 0, "Snapshot page: 0 overview, 1–6 topics, 7 Links, 8 Code, 9 recovery, 10 formatting, 11 editing basics")
	flag.Parse()
	loc := location{page: max(-1, min(len(topics)-1, *page-1))}
	m := model{width: *width, height: *height, location: loc, saved: map[int]location{}, history: []location{loc}}
	if *snapshot {
		fmt.Println(m.View().Content)
		return
	}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
