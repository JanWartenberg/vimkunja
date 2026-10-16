package tutorial

import (
	"fmt"
	"strings"

	"github.com/JanWartenberg/vimkunja/internal/tui"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const tutorialPanelHeight = 6

type expectedKey struct {
	value     string
	label     string
	freeInput bool
	command   string
}

type tutorialStep struct {
	title       string
	instruction string
	note        string
	keys        []expectedKey
	forward     bool
}

// Model wraps the real Vimkunja model. Matching keys are sent to Vimkunja;
// unexpected keys are ignored outside explicitly free input prompts.
type Model struct {
	app tea.Model

	steps    []tutorialStep
	step     int
	key      int
	width    int
	height   int
	feedback string
}

// NewModel adds the tutorial instruction strip to a real Vimkunja model.
func NewModel(app tea.Model) tea.Model {
	return &Model{app: app, steps: implementedTutorialSteps()}
}

func (m *Model) Init() tea.Cmd {
	if m.app == nil {
		return nil
	}
	return m.app.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		return m.updateApp(tea.WindowSizeMsg{Width: size.Width, Height: maxTutorial(10, size.Height-tutorialPanelHeight)})
	}

	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m.updateApp(msg)
	}
	if _, released := key.(tea.KeyReleaseMsg); released {
		return m, nil
	}
	got := normalizeTutorialKey(key.String())
	if got == "ctrl+c" {
		return m.updateApp(msg)
	}
	if m.step >= len(m.steps) {
		return m.updateApp(msg)
	}

	step := m.steps[m.step]
	want := step.keys[m.key]
	if got == "tab" && want.command != "" {
		m.feedback = ""
		_, cmd := m.updateApp(msg)
		prompt, ok := m.app.(interface{ CommandInputMatches(string) bool })
		if ok && prompt.CommandInputMatches(want.command) {
			for step.keys[m.key].value != "enter" {
				m.key++
			}
		}
		return m, cmd
	}
	if want.freeInput {
		m.feedback = ""
		_, cmd := m.updateApp(msg)
		prompt, ok := m.app.(interface{ DescriptionLinkPromptOpen() bool })
		if ok && !prompt.DescriptionLinkPromptOpen() && (got == "enter" || got == "esc") {
			m.key++
		}
		return m, cmd
	}
	if got != want.value {
		m.feedback = fmt.Sprintf("Expected %s; %s was ignored.", want.label, displayTutorialKey(got))
		return m, nil
	}

	m.feedback = ""
	var cmd tea.Cmd
	if step.forward {
		_, cmd = m.updateApp(msg)
	}
	m.key++
	if m.key == len(step.keys) {
		m.step++
		m.key = 0
	}
	return m, cmd
}

func (m *Model) updateApp(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.app == nil {
		return m, nil
	}
	updated, cmd := m.app.Update(msg)
	m.app = updated
	return m, cmd
}

func (m *Model) View() tea.View {
	if m.app == nil {
		return tea.NewView("Vimkunja tutorial could not start.")
	}
	view := m.app.View()
	view.Content = lipgloss.JoinVertical(lipgloss.Left, view.Content, m.renderTutorialPanel())
	view.WindowTitle = "Vimkunja Tutorial"
	return view
}

func (m *Model) renderTutorialPanel() string {
	width := m.width
	if width < 40 {
		width = 80
	}

	title, instruction, note, progress := "TUTORIAL COMPLETE", "You have practiced every implemented feature group.", "Use Ctrl+C to leave.", ""
	if m.step < len(m.steps) {
		step := m.steps[m.step]
		title = fmt.Sprintf("VIMKUNJA TUTORIAL  %d/%d  ·  %s", m.step+1, len(m.steps), step.title)
		instruction = step.instruction
		note = step.note
		progress = fmt.Sprintf("Next: %s", step.keys[m.key].label)
		if len(step.keys) > 1 && !step.keys[m.key].freeInput {
			progress += fmt.Sprintf("  ·  key %d/%d", m.key+1, len(step.keys))
		}
	}
	if m.feedback != "" {
		progress = m.feedback
	}

	innerWidth := maxTutorial(1, width-4)
	lines := []string{
		truncateTutorial(title, innerWidth),
		lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render(truncateTutorial(instruction, innerWidth)),
		truncateTutorial(note, innerWidth),
		truncateTutorial(progress, innerWidth),
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(tui.SelectionAccentColor())).
		Padding(0, 1).
		Width(width).
		Height(tutorialPanelHeight).
		Render(strings.Join(lines, "\n"))
}

func implementedTutorialSteps() []tutorialStep {
	return []tutorialStep{
		infoStep("Welcome", "This is the real Vimkunja UI backed by a disposable SQLite database.", "Only the keys requested by a lesson are accepted. Ctrl+C always exits.", "Enter"),
		actionStep("Projects Pane", "Navigate Projects in this order: j, k, G, gg, Ctrl+D, Ctrl+U, End, Home.", "Arrow keys are also supported aliases; this lesson practices Vim motions.", keys("j", "k", "G", "g", "g", "ctrl+d", "ctrl+u", "end", "home")...),
		actionStep("Select a Project", "Press Enter to focus the selected Project's Tasks Pane.", "Project selection drives the active List View.", key("enter", "Enter")),
		actionStep("Tasks Pane", "Navigate Tasks: j, k, G, gg, Ctrl+D, Ctrl+U, End, Home.", "gg/G and Home/End jump to the first/last Task.", keys("j", "k", "G", "g", "g", "ctrl+d", "ctrl+u", "end", "home")...),
		actionStep("Task Detail Pane", "Press Enter to open the selected Task in the Task Detail Pane.", "Detail is a normal Pane, not a modal.", key("enter", "Enter")),
		actionStep("Field Navigation", "Press j to move from Title to Priority.", "j/k (or arrows) move through editable fields without wrapping.", key("j", "j")),
		actionStep("Priority Picker", "Open the Picker, choose the next Priority, and confirm: i, j, Enter.", "Pickers use j/k, Enter to confirm, and Esc to cancel.", keys("i", "j", "enter")...),
		actionStep("Due Date Field", "Press j to focus Due Date.", "The unfocused value uses human-readable presentation.", key("j", "j")),
		actionStep("Due Date Input", "Enter the date 2030-01-15: press i, type 20300115, then Enter.", "Digits fill the YYYY-MM-DD mask; Enter returns to Normal mode.", append([]expectedKey{key("i", "i")}, append(textKeys("20300115"), key("enter", "Enter"))...)...),
		actionStep("Workflow Status", "Press j twice to focus Workflow Status.", "The field order passes through Labels before Status.", keys("j", "j")...),
		actionStep("Status Picker", "Change To-Do to Doing: i, j, Enter.", "Choosing Done would also set the global Task Done state.", keys("i", "j", "enter")...),
		actionStep("Done Field", "Focus Done with j, then toggle it on and back off with x, x.", "The change remains a draft until an Ex save command is used.", keys("j", "x", "x")...),
		actionStep("Description", "Focus Description, then try l, h, j, k, w, b, e, W, B, E, 0, ^, $, 2w, G, gg, Ctrl+D, Ctrl+U.", "Counts compose with motions; WORD motions treat punctuation as part of a non-blank token.", keys("j", "l", "h", "j", "k", "w", "b", "e", "W", "B", "E", "0", "^", "$", "2", "w", "G", "g", "g", "ctrl+d", "ctrl+u")...),
		actionStep("Description Change", "Use ciw, type 'Edited', then Esc.", "Operators combine with motions and word objects; daw includes adjacent whitespace.", append([]expectedKey{key("c", "c"), key("i", "i"), key("w", "w")}, append(textKeys("Edited"), key("esc", "Esc"))...)...),
		actionStep("Description Insert", "Use I, type 'Tutorial: ', then Esc to insert at line start.", "I and A preserve Vim-like line-start and line-end entry points.", append([]expectedKey{key("I", "I")}, append(textKeys("Tutorial: "), key("esc", "Esc"))...)...),
		actionStep("Description Append", "Use A, type ' Practiced in tutorial.', then Esc.", "Insert mode supports arrows, Backspace, Delete, and multiline Enter.", append([]expectedKey{key("A", "A")}, append(textKeys(" Practiced in tutorial."), key("esc", "Esc"))...)...),
		actionStep("Save", "Type :w and Enter to save while keeping Detail open.", ":write is the long alias; dirty state is marked with [+].", commandKeys("w")...),
		actionStep("Pane Focus", "Press Ctrl+W, then k to focus the Tasks Pane above.", "Ctrl+W h/j/k/l moves directionally; Ctrl+W w cycles open Panes.", keys("ctrl+w", "k")...),
		actionStep("Pane Focus", "Press Ctrl+W, then j to return to Task Detail.", "Ctrl+W Ctrl+W is another cycling form.", keys("ctrl+w", "j")...),
		actionStep("Close Detail", "Type :q and Enter to close the clean Task Detail Pane.", ":quit is the long alias; focus returns to Tasks.", commandKeys("q")...),
		actionStep("Toggle Done", "Press x to complete the selected Task immediately.", "Vimkunja also moves it to the configured Done Bucket.", key("x", "x")),
		actionStep("Reload", "Press r to reload the active View from SQLite.", "Successful writes already auto-refresh; r is the manual refresh.", key("r", "r")),
		actionStep("Ex New Task", "Type :new and Enter to open a new Task draft.", "The o shortcut uses the same editor.", commandKeys("new")...),
		actionStep("Insert Mode", "Press i, type 'Discard this draft', then Esc.", "In Insert mode, text is literal and Esc returns to Normal mode.", append([]expectedKey{key("i", "i")}, append(textKeys("Discard this draft"), key("esc", "Esc"))...)...),
		actionStep("Dirty Protection", "Try :q. Vimkunja refuses to close this unsaved draft.", "The error footer points to :w or :q!.", commandKeys("q")...),
		actionStep("Discard", "Type :q! and Enter to discard the draft explicitly.", ":quit! is the long alias.", commandKeys("q!")...),
		actionStep("Create Shortcut", "Press o to create another Task draft.", "From Overview, o starts with the Project Picker instead.", key("o", "o")),
		actionStep("Task Title", "Press i, type 'A tutorial task', then Esc.", "Title and Project are the only required fields.", append([]expectedKey{key("i", "i")}, append(textKeys("A tutorial task"), key("esc", "Esc"))...)...),
		actionStep("Save and Close", "Type :wq and Enter to create the Task and close Detail.", ":wqa saves and quits the whole application.", commandKeys("wq")...),
		actionStep("Delete Confirmation", "Press dd, then n to cancel deletion.", "Esc also cancels; deletion never occurs after dd alone.", keys("d", "d", "n")...),
		actionStep("Delete Task", "Press dd, then y to permanently delete the disposable Task.", "The tutorial database is thrown away after the session.", keys("d", "d", "y")...),
		actionStep("All-Tasks Overview", "Press - to open the cross-Project Overview.", "Overview hides completed Tasks and the Projects Pane by default.", key("-", "-")),
		actionStep("Overview Sort", "Press s, then p to sort by Priority descending.", "s d restores Due sorting; s t sorts by Title.", keys("s", "p")...),
		actionStep("Overview Filter", "Press f, then a to include all Tasks.", "Filters are f o open, f v overdue, and f t due today.", keys("f", "a")...),
		actionStep("Overview Creation", "Press o to start a Task draft from Overview.", "Without a Project context, Vimkunja opens the filterable Project Picker.", key("o", "o")),
		actionStep("Project Picker", "Type 'Home', then Enter to choose the matching Project.", "Backspace edits the filter; j/k select among multiple matches; Esc cancels.", append(textKeys("Home"), key("enter", "Enter"))...),
		actionStep("Discard Overview Draft", "Press j, then type :q! to discard this demonstration draft.", "After choosing Project, j moves from Project to Title.", append([]expectedKey{key("j", "j")}, commandKeys("q!")...)...),
		actionStep("Overview Toggle", "Press Space, then a to return to the remembered Project context.", "Space+a toggles Overview while preserving the selected Task.", keys("space", "a")...),
		actionStep("Projects Pane Toggle", "Press Space, then o to hide the Projects Pane.", "The Leader key waits briefly for its second key.", keys("space", "o")...),
		actionStep("Projects Pane Toggle", "Press Space, then o again to restore the Projects Pane.", "The Tasks Pane receives the reclaimed width while Projects is hidden.", keys("space", "o")...),
		actionStep("Kanban View", "Press Space, v, k to select the manual Kanban View.", "Space+v then l returns to List; :kanban and :list are equivalent commands.", keys("space", "v", "k")...),
		actionStep("Kanban Navigation", "Try j, k, l, h to move between cards and Buckets.", "Empty Buckets can still be active; edges are silent no-ops.", keys("j", "k", "l", "h")...),
		actionStep("Adjacent Kanban Move", "Press Shift+L to move the selected card to the right Bucket.", "Shift+H moves left; Done state follows the destination Bucket.", key("L", "Shift+L")),
		actionStep("Bucket Picker", "Open the target Picker, choose Done, then cancel: m, j, Esc.", "The Picker can reach Buckets outside the horizontal viewport.", keys("m", "j", "esc")...),
		actionStep("Arbitrary Kanban Move", "Now move to Done with m, j, Enter.", "The Board reloads and keeps the moved Task selected.", keys("m", "j", "enter")...),
		actionStep("Reopen from Kanban", "Press x to reopen the selected Task into the Default Bucket.", "x and card moves share the same Workflow Status semantics.", key("x", "x")),
		actionStep("List View", "Press Space, v, l to return to the Project's List View.", "Table and Gantt are intentionally not taught because they are not implemented.", keys("space", "v", "l")...),
		actionStep("Focus Projects", "Press Ctrl+W, then h to focus the Projects Pane.", "Directional focus never enters a hidden Pane.", keys("ctrl+w", "h")...),
		actionStep("Project without Kanban", "Press G to select Plan a Trip, then Enter to focus its Tasks.", "This fixture intentionally has no manual Kanban View.", keys("G", "enter")...),
		actionStep("Kanban Provisioning", "Type :kanban, then press n to cancel the first confirmation.", "Vimkunja never creates a Board without explicit confirmation.", append(commandKeys("kanban"), key("n", "n"))...),
		actionStep("Kanban Provisioning", "Type :kanban again, then press y to create the disposable Board.", "Esc also cancels; creation is one SQLite transaction.", append(commandKeys("kanban"), key("y", "y"))...),
		actionStep("Return to Workshop", "Use Space v l, Ctrl+W h, gg, Enter to return to Learn Vimkunja.", "The last non-Overview View preference is remembered globally.", keys("space", "v", "l", "ctrl+w", "h", "g", "g", "enter")...),
		actionStep("Overview Command", "Type :overview and Enter.", "This is the Ex-command equivalent of -.", commandKeys("overview")...),
		actionStep("List Command", "Type :list and Enter to restore Project List context.", "From Overview, the remembered Project context is restored first.", commandKeys("list")...),
		actionStep("Kanban Command", "Type :kanban and Enter to open the existing Board.", "Exactly one manual Kanban View is required; Vimkunja never guesses among several.", commandKeys("kanban")...),
		actionStep("List Command", "Type :list and Enter to return to List View.", "View commands and Space+v use the same application behavior.", commandKeys("list")...),
		actionStep("Help Pane", "Type :h and Enter to open in-application Help.", ":help is the long alias. Help is a normal Pane.", commandKeys("h")...),
		actionStep("Help Scrolling", "Scroll Help with j, k, Ctrl+D, Ctrl+U, G, gg, End, Home.", "PageDown/PageUp and arrow keys are supported aliases.", keys("j", "k", "ctrl+d", "ctrl+u", "G", "g", "g", "end", "home")...),
		actionStep("Help Pane Focus", "Use Ctrl+W k to focus Tasks, then Ctrl+W j to return to Help.", "Help participates in the normal Pane graph.", keys("ctrl+w", "k", "ctrl+w", "j")...),
		actionStep("Close Help", "Type :q and Enter to close the Help Pane.", "Esc deliberately does not close Help; use Pane focus or :q.", commandKeys("q")...),
		actionStep("Error Feedback", "Type :bogus and Enter to intentionally create an error.", "Errors appear in red and remain in the session error log.", commandKeys("bogus")...),
		actionStep("Error Log", "Type :errors and Enter to open the session error log.", ":err is its short alias.", commandKeys("errors")...),
		actionStep("Information Pane", "Try j, k, G, gg, then press q to close the error log.", "Information Panes also support Esc, Ctrl+D/U, PageUp/Down, Home, and End.", append(keys("j", "k", "G", "g", "g"), key("q", "q"))...),
		actionStep("Finish", "Type :qa and Enter to quit the tutorial.", ":qa! force-quits; q quits from a main Pane; Ctrl+C is the emergency exit.", commandKeys("qa")...),
	}
}

func infoStep(title, instruction, note, next string) tutorialStep {
	return tutorialStep{title: title, instruction: instruction, note: note, keys: []expectedKey{key("enter", next)}}
}

func actionStep(title, instruction, note string, expected ...expectedKey) tutorialStep {
	return tutorialStep{title: title, instruction: instruction, note: note, keys: expected, forward: true}
}

func key(value, label string) expectedKey {
	return expectedKey{value: normalizeTutorialKey(value), label: label}
}

func keys(values ...string) []expectedKey {
	out := make([]expectedKey, 0, len(values))
	for _, value := range values {
		out = append(out, key(value, displayTutorialKey(value)))
	}
	return out
}

func textKeys(value string) []expectedKey {
	out := make([]expectedKey, 0, len([]rune(value)))
	for _, r := range value {
		label := string(r)
		if r == ' ' {
			label = "Space"
		}
		out = append(out, key(string(r), label))
	}
	return out
}

func commandKeys(command string) []expectedKey {
	out := []expectedKey{key(":", ":")}
	out = append(out, textKeys(command)...)
	out = append(out, key("enter", "Enter"))
	for i := 1; i < len(out); i++ {
		out[i].command = command
	}
	return out
}

func normalizeTutorialKey(value string) string {
	switch value {
	case " ", "space":
		return "space"
	case "return":
		return "enter"
	case "pgdown", "pagedown":
		return "pgdown"
	case "pgup", "pageup":
		return "pgup"
	default:
		return value
	}
}

func displayTutorialKey(value string) string {
	switch normalizeTutorialKey(value) {
	case "space":
		return "Space"
	case "enter":
		return "Enter"
	case "esc":
		return "Esc"
	case "ctrl+w":
		return "Ctrl+W"
	case "ctrl+d":
		return "Ctrl+D"
	case "ctrl+u":
		return "Ctrl+U"
	case "ctrl+c":
		return "Ctrl+C"
	case "end":
		return "End"
	case "home":
		return "Home"
	default:
		return value
	}
}

func truncateTutorial(value string, width int) string {
	if lipgloss.Width(value) <= width {
		return value
	}
	runes := []rune(value)
	for len(runes) > 1 && lipgloss.Width(string(runes)+"…") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func maxTutorial(a, b int) int {
	if a > b {
		return a
	}
	return b
}
