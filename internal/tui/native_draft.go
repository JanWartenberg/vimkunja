package tui

import (
	tea "charm.land/bubbletea/v2"
	"errors"
	"fmt"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"strings"
)

func (m *model) syncNativeDraft() {
	e := m.editor
	e.draft.Description = e.nativeDraft.DraftHTML()
	e.draft.DescriptionCandidate = nil
	if e.mode == editorNormal {
		e.nativeDraft.InsertionView(e.nativeDescription, false)
	} else {
		e.nativeDraft.UpdateView(e.nativeDescription)
	}
}
func (m *model) applyNative(a description.Action, confirmed, insert bool) {
	e := m.editor
	skipped := (a.Kind == description.ToggleMark || a.Kind == description.ClearMarks) && e.nativeDraft.SelectionContainsOpaque()
	if err := e.nativeDraft.Apply(a, confirmed); err != nil {
		if errors.Is(err, description.ErrConfirmation) {
			e.nativePending = &a
			e.nativePendingInsert = insert
			m.toast = "Preserved content would change: y confirm · n/Esc cancel"
			if err == description.ErrQuoteMetadataConfirmation {
				m.toast = "Zitat-Metadaten würden gelöscht werden. Bitte Löschung bestätigen. · y / n/Esc"
			}
		} else {
			m.recordError(err)
		}
		return
	}
	if insert {
		e.mode = editorInsert
		e.nativeDraft.InsertionView(e.nativeDescription, true)
	}
	m.syncNativeDraft()
	if skipped {
		m.toast = "Preserved content skipped during formatting"
	}
	e.descriptionActive = true
	e.nativePendingInsert = false
	if insert && (a.Kind == description.Split || a.Kind == description.OpenBelow || a.Kind == description.OpenAbove || a.Kind == description.CodeBefore || a.Kind == description.CodeAfter) {
		e.nativeTrigger = e.nativeDraft.CodeMarkerParagraph(true)
		e.nativeCodeEscape = false
	}
	if insert {
		e.mode = editorInsert
	}
}
func (m *model) nativeEditKey(key string) bool {
	e := m.editor
	d := e.nativeDraft
	sel := d.Selection()
	if sel != nil && (key == "i" || key == "a") {
		e.nativePrefix = "v" + key
		return true
	}
	switch key {
	case "i", "a", "I", "A":
		if d.Classification() == description.Blocked {
			m.recordError(description.ErrBlocked)
			return true
		}
		if sel != nil {
			m.applyNative(description.Action{Kind: description.ReplaceSelection}, false, true)
			return true
		}
		if key == "I" || key == "A" {
			_ = d.SetCursor(d.LineBoundary(key == "A"))
		} else if key == "a" {
			_ = d.SetTarget(d.AfterTarget())
		}

		e.mode = editorInsert
		e.nativeCodeEscape = false
		e.nativeTrigger = d.CodeMarkerParagraph(true)
		e.nativeDraft.InsertionView(e.nativeDescription, true)
		e.descriptionActive = true
		d.UpdateView(e.nativeDescription)
		return true
	case "o", "O":
		if sel != nil && key == "o" {
			d.SwapSelection()
			d.UpdateView(e.nativeDescription)
			return true
		}
		kind := description.OpenBelow
		if key == "O" {
			kind = description.OpenAbove
		}
		m.applyNative(description.Action{Kind: kind}, false, true)
		return true
	case "v", "V":
		if d.Classification() == description.Blocked {
			m.recordError(description.ErrBlocked)
			return true
		}
		d.Select(key == "V", false)
		e.descriptionActive = true
		return true
	case "d":
		if sel != nil {
			m.applyNative(description.Action{Kind: description.DeleteSelection}, false, false)
		} else {
			e.nativePrefix = "d"
		}
		return true
	case "c":
		if sel != nil {
			m.applyNative(description.Action{Kind: description.ReplaceSelection}, false, true)
		} else {
			m.recordError(fmt.Errorf("Select content before replacing it"))
		}
		return true
	case "x", "delete", "backspace":
		if key == "x" && sel == nil && d.OnCheckbox() {
			m.applyNative(description.Action{Kind: description.ToggleCheckbox}, false, false)
			return true
		}
		kind := description.Delete
		if key == "backspace" {
			kind = description.Backspace
		}
		if sel != nil {
			kind = description.DeleteSelection
		}
		m.applyNative(description.Action{Kind: kind}, false, false)
		return true
	case "y":
		e.nativeRegister = d.Copy()
		m.toast = "Description text copied"
		return true
	case "p":
		if e.nativeRegister != "" {
			m.applyNative(description.Action{Kind: description.Insert, Text: e.nativeRegister}, false, false)
		}
		return true
	case "u", "ctrl+r":
		changed := false
		if key == "ctrl+r" {
			changed = d.Redo()
		} else {
			changed = d.Undo()
		}
		if changed {
			m.syncNativeDraft()
		}
		return true
	}
	return false
}
func (m *model) updateNativeInsert(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	e := m.editor
	key := k.String()
	if e.nativeLiteralTab {
		e.nativeLiteralTab = false
		if key == "tab" {
			m.nativeInsertText("\t")
			return m, nil
		}
		if key != "esc" {
			m.recordError(fmt.Errorf("Literal Tab prefix requires Tab; Esc cancels"))
			return m, nil
		}
	}
	if e.nativeDraft.InCode() && (key == "ctrl+v" || key == m.codeConfig.LiteralTabAlias && key != "") {
		e.nativeLiteralTab = true
		m.toast = "LITERAL TAB · Tab insert · Esc cancel"
		return m, nil
	}
	switch key {
	case "esc":
		e.nativeCodeEscape = false
		e.nativeTrigger = false
		e.mode = editorNormal
		e.nativeDraft.InsertionView(e.nativeDescription, false)
		return m, nil
	case "ctrl+v":
		return m, m.pasteClipboard(e)
	case "enter":
		e.nativeCodeEscape = false
		if e.nativeTrigger && e.nativeDraft.CodeMarkerParagraph(false) {
			e.nativeTrigger = false
			m.applyNative(description.Action{Kind: description.CreateCode, Text: "marker"}, false, true)
			break
		}
		e.nativeTrigger = false
		if e.nativeDraft.EmptyListFill() {
			e.mode = editorNormal
			e.nativeDraft.InsertionView(e.nativeDescription, false)
			break
		}
		m.applyNative(description.Action{Kind: description.Split}, false, true)
	case "tab":
		e.nativeCodeEscape = false
		e.nativeTrigger = false
		if e.nativeDraft.InCode() {
			m.applyNative(description.Action{Kind: description.InsertTab, Tabstop: m.codeConfig.Tabstop}, false, true)
		} else {
			m.nativeInsertText("\t")
		}
	case "shift+enter":
		e.nativeCodeEscape = false
		e.nativeTrigger = false
		m.applyNative(description.Action{Kind: description.HardBreak}, false, true)
	case "backspace":
		e.nativeCodeEscape = false
		m.applyNative(description.Action{Kind: description.Backspace}, false, true)
	case "delete":
		e.nativeCodeEscape = false
		m.applyNative(description.Action{Kind: description.Delete}, false, true)
	case "left", "right", "up", "down", "home", "end":
		e.nativeCodeEscape = false
		e.nativeTrigger = false
		if key == "home" || key == "end" {
			_ = e.nativeDraft.SetCursor(e.nativeDraft.LineBoundary(key == "end"))
			e.nativeDraft.UpdateView(e.nativeDescription)
			break
		}
		motion := map[string]string{"left": "h", "right": "l", "up": "k", "down": "j"}[key]
		w, h := m.nativeDescriptionGeometry()
		e.nativeDescription.Move(motion, 1, w, h)
		_ = e.nativeDraft.SetTarget(e.nativeDescription.Cursor())
		e.nativeDraft.UpdateView(e.nativeDescription)

	default:
		if text := k.Key().Text; text != "" {
			m.nativeInsertText(text)
		}
	}
	return m, nil
}
func (m *model) nativeInsertText(text string) {
	m.applyNative(description.Action{Kind: description.Type, Text: text}, false, true)

}
func (m *model) prepareNativeSave() bool {
	e := m.editor
	if e == nil || e.nativeDraft == nil {
		return true
	}
	if e.nativePending != nil {
		m.recordError(fmt.Errorf("Confirm or cancel the pending Description action first"))
		return false
	}
	if e.draft.Description == e.original.Description {
		e.draft.DescriptionCandidate = nil
		return true
	}
	c, err := e.nativeDraft.PrepareSave()
	if err != nil {
		m.recordError(err)
		return false
	}
	e.draft.DescriptionCandidate = &c
	return true
}
func (m *model) nativePaletteKey(key string) bool {
	e := m.editor
	if m.focus == paneDetail && e != nil && e.field == fieldDescription && e.mode == editorNormal && key == "c" && e.draft.Description == "" {
		if e.nativeDraft == nil {
			e.nativeDraft = description.NewDraft(description.Parse(""))
			e.nativeDescription = e.nativeDraft.View(description.ViewOptions{ASCII: tuiTheme.nativeASCII, Tabstop: e.nativeTabstop, OpaqueOpen: tuiTheme.opaqueOpen, OpaqueClose: tuiTheme.opaqueClose})
		}
		m.applyNative(description.Action{Kind: description.CreateCode}, false, true)
		return true
	}
	if m.focus != paneDetail || e == nil || e.field != fieldDescription || !e.descriptionActive || e.descriptionView() == nil {
		return false
	}
	if key == "o" {
		return false
	}
	if key == "esc" {
		return true
	}
	sel := e.nativeDraft.Selection()
	if key == "l" && (sel == nil || !sel.Blocks) {
		m.openNativeLink()
		return true
	}
	if sel != nil && !sel.Blocks {
		tags := map[string]string{"b": "strong", "i": "em", "u": "u", "s": "s"}
		if tag := tags[key]; tag != "" {
			m.applyNative(description.Action{Kind: description.ToggleMark, Text: tag}, false, false)
		} else if key == "0" {
			m.applyNative(description.Action{Kind: description.ClearMarks}, false, false)
		} else {
			m.recordError(fmt.Errorf("Description action is not available yet"))
		}
		return true
	}
	if sel != nil && sel.Blocks {
		if e.nativeDraft.CodeSelection() && (key == "x" || key == "c") {
			kind := description.IndentCode
			if key == "x" {
				kind = description.OutdentCode
			}
			m.applyNative(description.Action{Kind: kind, Tabstop: m.codeConfig.Tabstop, Indent: m.codeConfig.IndentStep}, false, false)
			return true
		}
		actions := map[string]description.ActionKind{"k": description.OutdentList, "l": description.IndentList, "q": description.UnwrapQuote, "w": description.WrapQuote}
		if action, ok := actions[key]; ok {
			m.applyNative(description.Action{Kind: action}, false, false)
			return true
		}
	}
	if sel == nil {
		if key == "c" {
			m.applyNative(description.Action{Kind: description.CreateCode}, false, true)
			return true
		}
		if key == "q" {
			m.applyNative(description.Action{Kind: description.ToggleQuote}, false, false)
			return true
		}
		lists := map[string]string{"u": "ul", "n": "ol", "x": "task"}
		if kind := lists[key]; kind != "" {
			m.applyNative(description.Action{Kind: description.ConvertList, Text: kind}, false, false)
			return true
		}
		tags := map[string]string{"1": "h1", "2": "h2", "3": "h3", "0": "p"}
		if tag := tags[key]; tag != "" {
			m.applyNative(description.Action{Kind: description.Convert, Text: tag}, false, false)
			return true
		}
	}
	m.recordError(fmt.Errorf("Description structure action belongs to a later ticket"))
	return true
}
func (m *model) nativePalette() string {
	e := m.editor
	if !m.leaderArmed || m.focus != paneDetail || e == nil || e.field != fieldDescription || !e.descriptionActive || e.descriptionView() == nil {
		return ""
	}
	if s := e.nativeDraft.Selection(); s != nil {
		if !s.Blocks {
			return "FORMAT  b bold · i italic · u underline · s strike · 0 clear · l link · o projects"
		}
		return "STRUCTURE  q/w quote · k/l list · x/c code outdent/indent · o projects"
	}
	return "BLOCK  1/2/3 heading · 0 paragraph · u/n/x list · q quote · c code · l link · y copy URL · o projects"
}
func selectDescriptionBlockCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.focus == paneDetail && m.editor != nil && m.editor.field == fieldDescription && m.editor.descriptionView() != nil {
		m.editor.nativeDraft.Select(true, true)
		m.editor.descriptionActive = true
	} else {
		m.recordError(fmt.Errorf(":selectblock requires a Native Description"))
	}
	return m, nil
}
func clearDescriptionCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	return recoverDescription(m, description.ClearDescription)
}
func recoverDescriptionCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	return recoverDescription(m, description.RecoverText)
}
func recoverDescription(m *model, kind description.ActionKind) (tea.Model, tea.Cmd) {
	if m.editor == nil || m.editor.descriptionView() == nil || m.editor.nativeDraft.Classification() != description.Blocked {
		m.recordError(fmt.Errorf("Recovery requires a Blocked Description"))
		return m, nil
	}
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	m.applyNative(description.Action{Kind: kind}, false, kind == description.RecoverText)
	return m, nil
}

type descriptionReloadedMsg struct {
	submitted vimkunja.TaskDraft
	editor    *taskEditor
	state     vimkunja.TaskUndoState
	err       error
}

func reloadDescriptionCommand(m *model, _ []string) (tea.Model, tea.Cmd) {
	if m.editor == nil || m.editor.id == 0 {
		m.recordError(fmt.Errorf("Reload requires an existing Task"))
		return m, nil
	}
	// An explicit hard reload discards the complete Task draft, consistently with :e!.
	e := m.editor
	submitted := cloneTaskDraft(e.draft)
	return m, func() tea.Msg {
		ctx, cancel := contextWithTimeout()
		defer cancel()
		state, err := m.store.TaskUndoState(ctx, e.id)
		return descriptionReloadedMsg{submitted: submitted, editor: e, state: state, err: err}
	}
}
func (m *model) nativeModeHint() string {
	e := m.editor
	if e.nativePending != nil {
		return "CONFIRM  y/Enter · n/Esc cancel"
	}
	if hint := m.nativeLinkHint(); hint != "" {
		return hint
	}
	if s := e.nativeDraft.Selection(); s != nil {
		label := "VISUAL"
		if s.Blocks {
			label = "VISUAL BLOCKS"
			if !s.WholeBlock && e.nativeDraft.CodeSelection() {
				label = "VISUAL CODE LINES"
			}
		}
		return label + " · Space format · d delete · o swap · Esc cancel"
	}
	if e.mode == editorInsert && e.nativeDraft.InCode() {
		return "INSERT CODE · Tab spaces · Ctrl+V Tab literal · Esc normal"
	}
	if e.nativeDraft.InCode() {
		return "CODE · o/O source line · ü/Ü paragraph · ]Space/[Space · V lines"
	}
	if e.mode == editorInsert {
		return "INSERT · Esc normal · Enter paragraph · Shift+Enter break"
	}
	if e.nativeDraft.Classification() == description.Blocked {
		return "Malformed HTML · :descriptionrecover · :descriptionclear · :e! discard"
	}
	return strings.TrimSpace(e.nativeDescription.Hint()) + " · i edit · v/V select · Space format · u undo · :w save"
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func (m *model) nativePaletteSecondary() bool {
	if m.nativePalette() == "" {
		return false
	}
	candidate := *m
	candidate.leaderArmed = false
	width, height := candidate.viewportSize()
	candidate.viewportReceived = true
	candidate.width = width
	candidate.height = max(1, height-1)
	_, rows := candidate.nativeDescriptionGeometry()
	return rows >= 3

}
func (m *model) nativePaletteRow(w int) string {
	return renderFooterHints(max(1, w), "", strings.Split(m.nativePalette(), " · "), true)
}

func (m *model) nativePasteText(text string) string {
	e := m.editor
	if m.focus == paneDetail && e != nil && e.field == fieldDescription && e.mode == editorInsert && e.nativeDraft != nil {
		if e.nativeDraft.InCode() {
			return text
		}
	}
	return normalizePastedText(text)
}

func retabDescriptionCommand(m *model, args []string) (tea.Model, tea.Cmd) {
	return m.retabDescription(args, false)
}
func retabAllDescriptionCommand(m *model, args []string) (tea.Model, tea.Cmd) {
	return m.retabDescription(args, true)
}
func (m *model) retabDescription(args []string, all bool) (tea.Model, tea.Cmd) {
	e := m.editor
	if len(args) != 0 {
		m.recordError(fmt.Errorf("Retab takes no arguments"))
		return m, nil
	}
	if m.focus != paneDetail || e == nil || e.field != fieldDescription || e.descriptionView() == nil || e.nativePending != nil {
		m.recordError(fmt.Errorf("Retab requires a Native Description without a pending action"))
		return m, nil
	}
	kind := description.Retab
	if all {
		kind = description.RetabAll
	}
	m.applyNative(description.Action{Kind: kind, Tabstop: m.codeConfig.Tabstop}, false, false)
	return m, nil
}
