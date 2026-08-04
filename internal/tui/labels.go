package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type labelUIKind int

const (
	labelUITaskPicker labelUIKind = iota
	labelUIGlobalOverview
)

type labelInputAction int

const (
	labelInputTaskPicker labelInputAction = iota
	labelInputCreate
	labelInputRename
)

type labelUI struct {
	kind          labelUIKind
	labels        []vimkunja.Label
	selected      int
	viewport      int
	input         string
	inputMode     bool
	inputAction   labelInputAction
	inputLabelID  int64
	deleteArmed   bool
	deleteConfirm bool
	commandMode   bool
	commandInput  string
	gArmed        bool
	loading       bool
	origin        *searchOrigin
}

type labelMutation int

const (
	labelCreatedForTask labelMutation = iota
	labelCreatedGlobally
	labelRenamed
	labelDeleted
)

func (m *model) openTaskLabelPicker() tea.Cmd {
	if m.editor == nil {
		return nil
	}
	if m.editor.labelsLoading {
		m.recordError(fmt.Errorf("Labels are still loading"))
		return nil
	}
	m.labelUI = &labelUI{kind: labelUITaskPicker, labels: append([]vimkunja.Label(nil), m.editor.labels...)}
	return m.loadLabels()
}

func (m *model) openGlobalLabelOverview() tea.Cmd {
	origin := m.captureSearchOrigin()
	m.labelUI = &labelUI{kind: labelUIGlobalOverview, origin: &origin}
	return m.loadLabels()
}

func (m *model) closeLabelUI() {
	if m.labelUI != nil && m.labelUI.origin != nil {
		m.restoreSearchOrigin(*m.labelUI.origin)
	}
	m.labelUI = nil
}

func (m *model) loadLabels() tea.Cmd {
	if m.labelUI == nil {
		return nil
	}
	m.labelUI.loading = true
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		labels, err := m.store.ListLabels(ctx)
		return labelsLoadedMsg{labels: labels, err: err}
	}
}

func (m *model) loadEditorLabels(taskID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		labels, err := m.store.ListTaskLabels(ctx, taskID)
		return taskLabelsLoadedMsg{taskID: taskID, labels: labels, err: err}
	}
}

func (m *model) updateLabelUI(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	ui := m.labelUI
	if ui == nil {
		return m, nil
	}
	if ui.loading {
		return m, nil
	}
	if ui.commandMode {
		return m.updateLabelCommand(key)
	}
	if ui.deleteConfirm {
		switch key.String() {
		case "y":
			label, ok := ui.selectedLabel()
			if !ok {
				ui.deleteConfirm = false
				return m, nil
			}
			ui.loading = true
			return m, m.deleteLabel(label.ID)
		case "n", "esc":
			ui.deleteConfirm = false
		}
		return m, nil
	}
	if ui.inputMode {
		return m.updateLabelInput(key)
	}
	return m.updateLabelNormal(key)
}

func (m *model) updateLabelNormal(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	ui := m.labelUI
	s := key.String()
	if s == ":" {
		ui.commandMode, ui.commandInput = true, ""
		return m, nil
	}
	if ui.gArmed {
		ui.gArmed = false
		if s == "g" {
			ui.selected, ui.viewport = 0, 0
		}
		return m, nil
	}
	if ui.deleteArmed {
		ui.deleteArmed = false
		if s == "d" {
			if _, ok := ui.selectedLabel(); ok {
				ui.deleteConfirm = true
			}
		}
		return m, nil
	}
	if s == "esc" || s == "q" {
		m.closeLabelUI()
		return m, nil
	}
	switch s {
	case "g":
		ui.gArmed = true
	case "j", "down":
		m.moveLabelSelection(1)
	case "k", "up":
		m.moveLabelSelection(-1)
	case "G", "end":
		ui.selected = max(0, len(ui.matches())-1)
		m.moveLabelSelection(0)
	case "ctrl+d", "pgdown", "pagedown":
		m.moveLabelSelection(max(1, m.pageStep()/2))
	case "ctrl+u", "pgup", "pageup":
		m.moveLabelSelection(-max(1, m.pageStep()/2))
	case "i", "a":
		if ui.kind == labelUIGlobalOverview {
			if s != "i" {
				return m, nil
			}
			label, ok := ui.selectedLabel()
			if !ok {
				return m, nil
			}
			ui.input, ui.inputAction, ui.inputLabelID, ui.inputMode = label.Title, labelInputRename, label.ID, true
		} else {
			ui.input, ui.inputAction, ui.inputMode = "", labelInputTaskPicker, true
		}
	case "o":
		if ui.kind == labelUIGlobalOverview {
			ui.input, ui.inputAction, ui.inputMode = "", labelInputCreate, true
		}
	case "d":
		if ui.kind == labelUIGlobalOverview {
			ui.deleteArmed = true
		}
	case "enter":
		if ui.kind == labelUITaskPicker {
			if label, ok := ui.selectedLabel(); ok {
				m.toggleEditorLabel(label)
			}
		} else if label, ok := ui.selectedLabel(); ok {
			m.closeLabelUI()
			return m, m.openLabelOverview(label)
		}
	}
	return m, nil
}

func (m *model) updateLabelCommand(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	ui := m.labelUI
	switch strings.ToLower(key.String()) {
	case "esc":
		ui.commandMode, ui.commandInput = false, ""
	case "backspace":
		runes := []rune(ui.commandInput)
		if len(runes) > 0 {
			ui.commandInput = string(runes[:len(runes)-1])
		}
	case "enter":
		command := strings.ToLower(strings.TrimSpace(ui.commandInput))
		ui.commandMode, ui.commandInput = false, ""
		switch command {
		case "q", "quit":
			m.closeLabelUI()
		case "qa", "quitall", "qa!", "quitall!":
			return m, tea.Quit
		case "":
		default:
			m.recordError(fmt.Errorf("unknown Labels command: %s", command))
		}
	default:
		if text := key.Key().Text; text != "" {
			ui.commandInput += text
		}
	}
	return m, nil
}

func (m *model) updateLabelInput(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	ui := m.labelUI
	s := key.String()
	switch s {
	case "esc":
		ui.inputMode = false
		ui.input = ""
		return m, nil
	case "backspace":
		runes := []rune(ui.input)
		if len(runes) > 0 {
			ui.input = string(runes[:len(runes)-1])
		}
		ui.selected, ui.viewport = 0, 0
		return m, nil
	case "up", "ctrl+p":
		m.moveLabelSelection(-1)
		return m, nil
	case "down", "ctrl+n":
		m.moveLabelSelection(1)
		return m, nil
	case "enter":
		return m, m.confirmLabelInput()
	}
	if text := key.Key().Text; text != "" {
		ui.input += text
		ui.selected, ui.viewport = 0, 0
	}
	return m, nil
}

func (m *model) confirmLabelInput() tea.Cmd {
	ui := m.labelUI
	if ui == nil {
		return nil
	}
	if ui.inputAction == labelInputTaskPicker {
		if label, ok := ui.exactMatch(); ok {
			m.toggleEditorLabel(label)
			ui.input, ui.inputMode = "", false
			return nil
		}
		ui.loading = true
		return m.createLabel(ui.input, labelCreatedForTask)
	}
	ui.loading = true
	if ui.inputAction == labelInputRename {
		if ui.inputLabelID == 0 {
			ui.loading = false
			return nil
		}
		return m.renameLabel(ui.inputLabelID, ui.input)
	}
	return m.createLabel(ui.input, labelCreatedGlobally)
}

func (m *model) createLabel(title string, mutation labelMutation) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		label, err := m.store.CreateLabel(ctx, title)
		return labelMutatedMsg{mutation: mutation, label: label, err: err}
	}
}

func (m *model) renameLabel(labelID int64, title string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		return labelMutatedMsg{mutation: labelRenamed, err: m.store.RenameLabel(ctx, labelID, title)}
	}
}

func (m *model) deleteLabel(labelID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		return labelMutatedMsg{mutation: labelDeleted, err: m.store.DeleteLabel(ctx, labelID)}
	}
}

func (m *model) toggleEditorLabel(label vimkunja.Label) {
	if m.editor == nil {
		return
	}
	for i, assigned := range m.editor.labels {
		if assigned.ID == label.ID {
			m.editor.labels = append(m.editor.labels[:i], m.editor.labels[i+1:]...)
			m.syncEditorLabelDraft()
			return
		}
	}
	m.editor.labels = append(m.editor.labels, label)
	sort.SliceStable(m.editor.labels, func(i, j int) bool {
		return strings.ToLower(m.editor.labels[i].Title) < strings.ToLower(m.editor.labels[j].Title)
	})
	m.syncEditorLabelDraft()
}

func (m *model) removeSelectedEditorLabel() {
	if m.editor == nil || len(m.editor.labels) == 0 {
		return
	}
	index := clamp(m.editor.labelSelected, 0, len(m.editor.labels)-1)
	m.editor.labels = append(m.editor.labels[:index], m.editor.labels[index+1:]...)
	m.editor.labelSelected = clamp(index, 0, max(0, len(m.editor.labels)-1))
	m.syncEditorLabelDraft()
}

func (m *model) syncEditorLabelDraft() {
	e := m.editor
	if e == nil {
		return
	}
	e.draft.LabelIDs = editorLabelIDs(e.labels)
	e.draft.LabelIDsSet = true
}

func editorLabelIDs(labels []vimkunja.Label) []int64 {
	ids := make([]int64, 0, len(labels))
	for _, label := range labels {
		ids = append(ids, label.ID)
	}
	return ids
}

func (ui *labelUI) matches() []vimkunja.Label {
	query := strings.ToLower(strings.TrimSpace(ui.input))
	matches := make([]vimkunja.Label, 0, len(ui.labels))
	for _, label := range ui.labels {
		if query == "" || strings.Contains(strings.ToLower(label.Title), query) {
			matches = append(matches, label)
		}
	}
	return matches
}

func (ui *labelUI) selectedLabel() (vimkunja.Label, bool) {
	matches := ui.matches()
	if ui.selected < 0 || ui.selected >= len(matches) {
		return vimkunja.Label{}, false
	}
	return matches[ui.selected], true
}

func (ui *labelUI) exactMatch() (vimkunja.Label, bool) {
	query := normalizedLabelTitle(ui.input)
	for _, label := range ui.labels {
		if normalizedLabelTitle(label.Title) == query {
			return label, true
		}
	}
	return vimkunja.Label{}, false
}

func (m *model) labelModalHeight() int {
	viewportHeight := m.height
	if viewportHeight <= 0 {
		viewportHeight = 30
	}
	return searchModalHeight(viewportHeight)
}

func (m *model) labelResultLineBudget() int {
	// The title, prompt/instructions, padding, borders, and one free line keep
	// Lip Gloss' minimum Height from expanding the modal around long lists.
	return max(1, m.labelModalHeight()-12)
}

func (m *model) labelResultEnd(start int) int {
	matches := m.labelUI.matches()
	start = clamp(start, 0, len(matches))
	return min(len(matches), start+m.labelResultLineBudget())
}

func (m *model) moveLabelSelection(delta int) {
	ui := m.labelUI
	if ui == nil || len(ui.matches()) == 0 {
		return
	}
	ui.selected = clamp(ui.selected+delta, 0, len(ui.matches())-1)
	if ui.selected < ui.viewport {
		ui.viewport = ui.selected
	}
	for ui.selected >= m.labelResultEnd(ui.viewport) && ui.viewport < ui.selected {
		ui.viewport++
	}
}

func normalizedLabelTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

func (m *model) renderLabelOverlay(background string, width int) string {
	ui := m.labelUI
	if ui == nil {
		return background
	}
	modalWidth := min(60, max(36, width-8))
	modalHeight := m.labelModalHeight()
	bodyWidth := max(1, modalWidth-6)
	title := "LABEL PICKER"
	if ui.kind == labelUIGlobalOverview {
		title = "LABELS"
	}
	body := []string{lipgloss.NewStyle().Bold(true).Render(title), ""}
	if ui.commandMode {
		body = append(body, ":"+ui.commandInput+"_")
	} else if ui.deleteConfirm {
		label, _ := ui.selectedLabel()
		message := fmt.Sprintf("Delete %q?", label.Title)
		if label.TaskCount > 0 {
			message += fmt.Sprintf(" This removes %d Task associations.", label.TaskCount)
		}
		body = append(body, message, "", "y delete  •  n/Esc cancel")
	} else if ui.inputMode {
		prompt := "Label: "
		if ui.inputAction == labelInputRename {
			prompt = "Rename: "
		}
		body = append(body, prompt+ui.input+"_", "")
		body = append(body, m.renderLabelMatches(bodyWidth)...)
	} else if ui.loading {
		body = append(body, "Loading…")
	} else {
		if ui.kind == labelUITaskPicker {
			body = append(body, "Enter toggles assignment · i/a search or create", "")
		} else {
			body = append(body, "o create · i rename · dd then y delete", "")
		}
		body = append(body, m.renderLabelMatches(bodyWidth)...)
	}
	modal := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("255")).
		Padding(1, 2).
		Width(modalWidth).
		Height(max(1, modalHeight-4)).
		Render(strings.Join(body, "\n"))
	return overlayModal(background, modal, width)
}

func (m *model) renderLabelMatches(width int) []string {
	ui := m.labelUI
	matches := ui.matches()
	if len(matches) == 0 {
		return []string{"No matching Labels."}
	}
	start := clamp(ui.viewport, 0, max(0, len(matches)-1))
	end := m.labelResultEnd(start)
	lines := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		label := matches[index]
		prefix := "  "
		if index == ui.selected {
			prefix = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(selectionAccent)).Render(">") + " "
		}
		line := prefix + truncatePlain(label.Title, max(1, width-lipgloss.Width(prefix)))
		if ui.kind == labelUIGlobalOverview {
			line += fmt.Sprintf("  (%d)", label.TaskCount)
		} else if m.editorHasLabel(label.ID) {
			line += "  [x]"
		}
		lines = append(lines, line)
	}
	return lines
}

func (m *model) editorHasLabel(labelID int64) bool {
	if m.editor == nil {
		return false
	}
	for _, label := range m.editor.labels {
		if label.ID == labelID {
			return true
		}
	}
	return false
}
