package tui

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func nativeEditingModel(source string) *model {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	return m
}
func nativeKeys(m *model, keys ...string) *model {
	for _, key := range keys {
		k := tea.Key{Code: []rune(key)[0], Text: key}
		switch key {
		case "esc":
			k = tea.Key{Code: tea.KeyEscape}
		case "enter":
			k = tea.Key{Code: tea.KeyEnter}
		case "shift+enter":
			k = tea.Key{Code: tea.KeyEnter, Mod: tea.ModShift}
		case "space":
			k = tea.Key{Code: ' ', Text: " "}
		case "ctrl+r":
			k = tea.Key{Code: 'r', Mod: tea.ModCtrl}
		}
		m = pressKey(m, k)
	}
	return m
}
func TestNativeDraftEventSplitUndoAndDiscard(t *testing.T) {
	m := nativeEditingModel(`<p>AlphaBeta</p>`)
	m = nativeKeys(m, "5", "l", "i", "enter", "esc")
	if m.editor.draft.Description != `<p>Alpha</p><p>Beta</p>` || !m.editor.dirty() {
		t.Fatalf("%q mode=%v", m.editor.draft.Description, m.editor.mode)
	}
	m = nativeKeys(m, "u")
	if m.editor.draft.Description != `<p>AlphaBeta</p>` {
		t.Fatal(m.editor.draft.Description)
	}
	m = nativeKeys(m, "ctrl+r")
	if m.editor.draft.Description != `<p>Alpha</p><p>Beta</p>` {
		t.Fatal(m.editor.draft.Description)
	}
	forceEditCommand(m, nil)
	if m.editor.dirty() || strings.Contains(m.editor.descriptionView().ContentText(), "\n") {
		t.Fatal("discard did not restore document")
	}
}

func TestNativeVisualPaletteRetainsSelectionAndConfirmsOpaque(t *testing.T) {
	m := nativeEditingModel(`<p>Alpha beta</p>`)
	m = nativeKeys(m, "w", "v", "i", "w", "space", "b")
	if m.editor.draft.Description != `<p>Alpha <strong>beta</strong></p>` || m.editor.nativeDraft.Selection() == nil {
		t.Fatal(m.editor.draft.Description)
	}
	m = nativeKeys(m, "space", "b")
	if m.editor.draft.Description != `<p>Alpha beta</p>` {
		t.Fatal(m.editor.draft.Description)
	}
	m = nativeKeys(m, "esc")
	if m.editor.nativeDraft.Selection() != nil {
		t.Fatal("Esc retained selection")
	}
	m = nativeEditingModel(`<p>Read <img alt='map' src='secret'> now</p>`)
	m = nativeKeys(m, "space", "2")
	if m.editor.nativePending == nil || m.editor.dirty() {
		t.Fatal("conversion should await confirmation")
	}
	m = nativeKeys(m, "esc")
	if m.editor.nativePending != nil || m.editor.dirty() {
		t.Fatal("cancel mutated")
	}
	m = nativeKeys(m, "space", "2", "y")
	if m.editor.draft.Description != `<h2>Read <img alt='map' src='secret'> now</h2>` {
		t.Fatal(m.editor.draft.Description)
	}
	m = nativeKeys(m, "u")
	if m.editor.draft.Description != `<p>Read <img alt='map' src='secret'> now</p>` {
		t.Fatal(m.editor.draft.Description)
	}
	m = nativeKeys(m, "ctrl+r")
	if m.editor.nativePending != nil || !m.editor.dirty() {
		t.Fatal("Redo reconfirmed")
	}
}
func TestNativePaletteGeometryTimeoutAndInsertSpace(t *testing.T) {
	for _, h := range []int{16, 50} {
		m := nativeEditingModel(`<p>Alpha</p>`)
		m.width = 100
		m.height = h
		m = nativeKeys(m, "space")
		if m.nativePalette() == "" || m.nativePaletteSecondary() != (h == 50) {
			t.Fatalf("height %d secondary %v", h, m.nativePaletteSecondary())
		}
		m.Render()
		m.Update(leaderTimeoutMsg{})
		if m.leaderArmed || m.nativePalette() != "" {
			t.Fatal("timeout failed")
		}
	}
	m := nativeEditingModel(`<p>Alpha</p>`)
	m = nativeKeys(m, "i", "space")
	if m.leaderArmed || m.editor.draft.Description != `<p> Alpha</p>` {
		t.Fatal(m.editor.draft.Description)
	}
	m = nativeKeys(m, "esc", "space", "o")
	if !m.projectsPaneCollapsed {
		t.Fatal("Space o did not toggle projects")
	}
}
func TestNativeSaveConflictKeepsSelectionAndHistory(t *testing.T) {
	m, db := auditEditorModel(t)
	const source = `<ul data-type="taskList"><li data-type="taskItem" data-checked="false">A</li></ul>`
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task',?,0,0,1)`, source); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	m = nativeKeys(m, "a", "B", "esc", "v")
	draft := m.editor.draft.Description
	if _, err := db.Exec(`UPDATE tasks SET description='<p>Remote</p>' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	cmd := m.saveEditor(true)
	if cmd == nil {
		t.Fatal(m.lastErr)
	}
	m.Update(cmd())
	if m.editor == nil || m.editor.draft.Description != draft || m.editor.nativeDraft.Selection() == nil || !strings.Contains(m.lastErr, "changed externally") {
		t.Fatal("conflict lost draft")
	}
	if strings.Contains(m.editor.draft.Description, "data-task-id") {
		t.Fatal("failed save adopted UUID")
	}
	m = nativeKeys(m, "esc", "u")
	if m.editor.draft.Description != source {
		t.Fatal("failed save lost Undo")
	}
}
func TestNativeRepeatedSaveKeepsCheckboxIdentity(t *testing.T) {
	m, db := auditEditorModel(t)
	const source = `<ul data-type="taskList"><li data-type="taskItem" data-checked="true">A</li></ul>`
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task',?,0,0,1)`, source); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	m.editor.field = fieldDescription
	m.editor.descriptionActive = true
	m = nativeKeys(m, "a", "B", "esc")
	cmd := m.saveEditor(false)
	m.Update(cmd())
	if m.lastErr != "" || m.editor.dirty() {
		t.Fatal(m.lastErr)
	}
	saved := m.editor.draft.Description
	if !strings.Contains(saved, "data-task-id") {
		t.Fatal("no saved UUID")
	}
	m = nativeKeys(m, "i", "C", "esc")
	cmd = m.saveEditor(false)
	m.Update(cmd())
	if m.lastErr != "" || m.editor.dirty() {
		t.Fatal(m.lastErr)
	}
	var persisted string
	db.QueryRow(`SELECT description FROM tasks WHERE id=1`).Scan(&persisted)
	id := strings.Split(strings.Split(saved, `data-task-id="`)[1], `"`)[0]
	if !strings.Contains(persisted, id) {
		t.Fatal("identity regenerated")
	}
	m = nativeKeys(m, "u")
	cmd = m.saveEditor(false)
	m.Update(cmd())
	if m.lastErr != "" || !strings.Contains(m.editor.draft.Description, id) {
		t.Fatal("Undo lost adopted identity")
	}
}
func TestNativeRecoveryFromTitleAndInsertionEndCursor(t *testing.T) {
	m := nativeEditingModel(`<p>safe<strong>broken`)
	m.editor.field = fieldTitle
	recoverDescriptionCommand(m, nil)
	m = nativeKeys(m, "esc")
	if m.editor.nativePending != nil {
		t.Fatal("recovery cannot cancel")
	}
	recoverDescriptionCommand(m, nil)
	m = nativeKeys(m, "y")
	if m.editor.mode != editorInsert || m.editor.field != fieldDescription {
		t.Fatal("recovery did not focus Insert")
	}
	m = nativeEditingModel(`<p>Alpha</p>`)
	m = nativeKeys(m, "A")
	f := m.editor.descriptionView().Layout(80, true)
	if f.CursorX != 5 || f.CursorY < 0 {
		t.Fatalf("cursor %d,%d", f.CursorX, f.CursorY)
	}
	m = nativeKeys(m, "!")
	f = m.editor.descriptionView().Layout(80, true)
	if f.CursorX != 6 {
		t.Fatalf("typed cursor %d", f.CursorX)
	}
}
func TestNativePendingConfirmationFreezesPaste(t *testing.T) {
	for _, clipboard := range []bool{false, true} {
		m := nativeEditingModel(`<p>A<img alt='map'>B</p>`)
		m = nativeKeys(m, "w", "w", "i")
		m = pressKey(m, tea.Key{Code: tea.KeyBackspace})
		if m.editor.nativePending == nil {
			t.Fatal("no confirmation")
		}
		before := m.editor.draft.Description
		if clipboard {
			m.Update(clipboardPastedMsg{editor: m.editor, text: "NEW"})
		} else {
			m.Update(tea.PasteMsg{Content: "NEW"})
		}
		if m.editor.draft.Description != before {
			t.Fatal("paste changed pending action")
		}
		m = nativeKeys(m, "y")
		if m.editor.draft.Description != `<p>AB</p>` {
			t.Fatal(m.editor.draft.Description)
		}
	}
}
func TestNativePersistedRestoreInvalidatesDocumentAndCountDoesNotLeak(t *testing.T) {
	m := nativeEditingModel(`<p>Old</p>`)
	m.editor.descriptionView()
	m.editor.field = fieldTitle
	m.refreshEditorAfterUndo(vimkunja.TaskUndoState{TaskID: 1, ProjectID: 1, Title: "Task", Description: `<p>New</p>`})
	if m.editor.descriptionView().ContentText() != "New" {
		t.Fatal("stale native document")
	}
	m = nativeEditingModel(`<p>ABCDEFG</p>`)
	m = nativeKeys(m, "3", "i", "X", "esc", "l")
	if got := m.editor.descriptionView().Cursor().Anchor.Offset; got != 2 {
		t.Fatalf("count leaked: %d", got)
	}
}
func TestNativeFormattedLineInsertionAndPaletteActualRows(t *testing.T) {
	for _, key := range []string{"I", "A"} {
		m := nativeEditingModel(`<p><strong>Alpha</strong></p>`)
		m = nativeKeys(m, key, "!", "esc")
		want := `<p><strong>!Alpha</strong></p>`
		if key == "A" {
			want = `<p><strong>Alpha!</strong></p>`
		}
		if m.editor.draft.Description != want {
			t.Fatalf("%s: %s", key, m.editor.draft.Description)
		}
	}
	for h := 10; h <= 60; h++ {
		m := nativeEditingModel(`<p>long long long long long long long</p>`)
		m.width = 80
		m.height = h
		m = nativeKeys(m, "space")
		if m.nativePaletteSecondary() {
			_, rows := m.nativeDescriptionGeometry()
			if rows < 3 {
				t.Fatalf("height%d rows%d", h, rows)
			}
		}
		m.Render()
	}
}
func TestNativeClearBlockedSaveAndInsertAcrossDelimiters(t *testing.T) {
	m, db := auditEditorModel(t)
	const source = `<p>safe<strong>broken`
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task',?,0,0,1)`, source); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: source})
	clearDescriptionCommand(m, nil)
	m = nativeKeys(m, "y")
	if !m.editor.dirty() || m.editor.draft.Description != "" {
		t.Fatal("clear not dirty")
	}
	m = nativeKeys(m, "u", "ctrl+r")
	cmd := m.saveEditor(false)
	if cmd == nil {
		t.Fatal(m.lastErr)
	}
	m.Update(cmd())
	if m.lastErr != "" || m.editor.dirty() {
		t.Fatal(m.lastErr)
	}
	var saved string
	db.QueryRow(`SELECT description FROM tasks WHERE id=1`).Scan(&saved)
	if saved != "" {
		t.Fatal(saved)
	}
	m = nativeEditingModel(`<p>A<strong>B</strong>C</p>`)
	m = nativeKeys(m, "i")
	// Insert arrows now visit both disclosed bold delimiters and their boundary.
	for i := 0; i < 5; i++ {
		m = pressKey(m, tea.Key{Code: tea.KeyRight})
	}
	m = nativeKeys(m, "!")
	if m.lastErr != "" || m.editor.draft.Description != `<p>A<strong>!B</strong>C</p>` {
		t.Fatalf("%s %s", m.editor.draft.Description, m.lastErr)
	}
}
func TestNativeHardReloadKeepsLateDraft(t *testing.T) {
	m, db := auditEditorModel(t)
	if _, err := db.Exec(`INSERT INTO tasks(id,project_id,title,description,done,priority,"index") VALUES(1,1,'Task','<p>Remote</p>',0,0,1)`); err != nil {
		t.Fatal(err)
	}
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: `<p>Old</p>`})
	_, cmd := reloadDescriptionCommand(m, nil)
	m.editor.draft.Title = "Late draft"
	m.Update(cmd())
	if m.editor.draft.Title != "Late draft" || !strings.Contains(m.lastErr, "changed while reloading") {
		t.Fatal("reload lost late draft")
	}
	_, cmd = reloadDescriptionCommand(m, nil)
	m.Update(cmd())
	if m.editor.draft.Description != `<p>Remote</p>` || m.editor.dirty() {
		t.Fatal("explicit reload failed")
	}
}
func TestNativeOpaqueInsertionBoundariesHaveVisibleCursor(t *testing.T) {
	for _, key := range []string{"i", "a"} {
		m := nativeEditingModel(`<p>A<img alt='map'>B</p>`)
		m = nativeKeys(m, "w", key)
		f := m.editor.descriptionView().Layout(80, true)
		if f.CursorX < 0 || f.CursorY < 0 {
			t.Fatalf("%s cursor missing", key)
		}
		m = nativeKeys(m, "!")
		want := `<p>A!<img alt='map'>B</p>`
		if key == "a" {
			want = `<p>A<img alt='map'>!B</p>`
		}
		if m.editor.draft.Description != want {
			t.Fatal(m.editor.draft.Description)
		}
	}
}
func TestNativeCountedCodeVisualClampsAndOpaqueEscCursor(t *testing.T) {
	m := nativeEditingModel("<pre>A\nB</pre><p>After</p>")
	m = nativeKeys(m, "V", "9", "j")
	if got := m.editor.nativeDraft.Copy(); got != "A\nB" {
		t.Fatalf("clamp: %q", got)
	}
	m = nativeEditingModel(`<p>A<img alt='map'>B</p>`)
	m = nativeKeys(m, "w", "a", "esc")
	f := m.editor.descriptionView().Layout(80, true)
	if f.CursorX < 0 || f.CursorY < 0 {
		t.Fatal("Normal cursor disappeared after opaque a/Esc")
	}
}
func TestNativeEmptyDraftHasNonpersistedCursor(t *testing.T) {
	m := nativeEditingModel(`<p>A</p>`)
	m = nativeKeys(m, "d", "d", "i")
	f := m.editor.descriptionView().Layout(80, true)
	if f.CursorX < 0 || f.CursorY < 0 {
		t.Fatal("empty insert cursor missing")
	}
	m = nativeKeys(m, "esc")
	if m.editor.draft.Description != "" {
		t.Fatal("empty focus synthesized persisted paragraph")
	}
	m = nativeKeys(m, "i", "X")
	if m.editor.draft.Description != `<p>X</p>` {
		t.Fatal(m.editor.draft.Description)
	}
}

func TestNativeNormalDeleteKeepsVisibleCursor(t *testing.T) {
	m := nativeKeys(nativeEditingModel(`<p>Alpha</p>`), "$", "x")
	frame := m.editor.nativeDescription.Viewport(80, 20, true)
	if frame.CursorX < 0 || frame.CursorY < 0 {
		t.Fatal("Normal delete hid cursor")
	}
	m = nativeKeys(m, "i", "!")
	if m.editor.draft.Description != `<p>Alp!h</p>` {
		t.Fatal(m.editor.draft.Description)
	}
}

func TestNativeFormattingReportsOnlySelectedOpaque(t *testing.T) {
	m := nativeKeys(nativeEditingModel(`<p>A<img alt='map'>B</p>`), "v", "G", "$", "space", "b")
	if !strings.Contains(m.toast, "skipped") {
		t.Fatal("missing Opaque skip notice:", m.toast)
	}
	m = nativeKeys(nativeEditingModel(`<p>A<img alt='map'>B</p>`), "v", "space", "b")
	if strings.Contains(m.toast, "skipped") {
		t.Fatal("unselected Opaque reported skipped")
	}
}

func TestNativeWordOperators(t *testing.T) {
	for _, tc := range []struct {
		name   string
		keys   []string
		want   string
		insert bool
	}{
		{"ciw", []string{"c", "i", "w"}, `<p> beta</p>`, true},
		{"caw", []string{"c", "a", "w"}, `<p>beta</p>`, true},
		{"diw", []string{"d", "i", "w"}, `<p> beta</p>`, false},
		{"daw", []string{"d", "a", "w"}, `<p>beta</p>`, false},
		{"cancel c", []string{"c", "esc"}, `<p>Alpha beta</p>`, false},
		{"cancel ci", []string{"c", "i", "esc"}, `<p>Alpha beta</p>`, false},
		{"invalid object", []string{"c", "i", "q"}, `<p>Alpha beta</p>`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := nativeKeys(nativeEditingModel(`<p>Alpha beta</p>`), tc.keys...)
			if m.editor.draft.Description != tc.want || (m.editor.mode == editorInsert) != tc.insert {
				t.Fatalf("got %s mode %v error %s", m.editor.draft.Description, m.editor.mode, m.lastErr)
			}
		})
	}
}

func TestNativeChangeWordUndoRedoPreservesMarksAndOpaque(t *testing.T) {
	const source = `<p><strong>Alpha</strong> beta <mention-user data-id="jan" data-label="Jan"></mention-user></p>`
	m := nativeKeys(nativeEditingModel(source), "c", "i", "w", "H", "a", "l", "l", "o", "esc")
	const changed = `<p><strong>Hallo</strong> beta <mention-user data-id="jan" data-label="Jan"></mention-user></p>`
	if m.editor.draft.Description != changed {
		t.Fatalf("change: %s, %s", m.editor.draft.Description, m.lastErr)
	}
	m = nativeKeys(m, "u")
	if m.editor.draft.Description != source || m.editor.nativeDraft.Selection() != nil {
		t.Fatalf("undo: %s", m.editor.draft.Description)
	}
	m = nativeKeys(m, "ctrl+r")
	if m.editor.draft.Description != changed {
		t.Fatalf("redo: %s", m.editor.draft.Description)
	}
	if m.editor.nativeDraft.Classification() != description.Mixed {
		t.Fatal("Opaque atom lost")
	}
}

func TestNativeChangeWordGroupsEndOnEscape(t *testing.T) {
	m := nativeKeys(nativeEditingModel(`<p>Alpha beta</p>`), "c", "i", "w", "O", "n", "e", "esc", "w", "c", "i", "w", "T", "w", "o", "esc")
	for _, want := range []string{`<p>One beta</p>`, `<p>Alpha beta</p>`} {
		nativeKeys(m, "u")
		if m.editor.draft.Description != want {
			t.Fatalf("undo got %s want %s", m.editor.draft.Description, want)
		}
	}
	for _, want := range []string{`<p>One beta</p>`, `<p>One Two</p>`} {
		nativeKeys(m, "ctrl+r")
		if m.editor.draft.Description != want {
			t.Fatalf("redo got %s want %s", m.editor.draft.Description, want)
		}
	}
}

func TestNativeEditableTextHint(t *testing.T) {
	m := nativeEditingModel(`<p>Alpha beta</p>`)
	m.editor.descriptionView()
	if strings.Contains(m.nativeModeHint(), "read-only") {
		t.Fatal(m.nativeModeHint())
	}
}

func TestNativeChangeWordOpaqueConfirmation(t *testing.T) {
	const source = `<p>Hi <mention-user data-id="jan" data-label="Jan"></mention-user>!</p>`
	for _, confirm := range []bool{false, true} {
		t.Run(fmt.Sprint(confirm), func(t *testing.T) {
			m := nativeEditingModel(source)
			m.editor.descriptionView()
			n := m.editor.nativeDraft.Snapshot()[0].Children[1]
			m.editor.nativeDraft.SetCursor(description.Anchor{Node: n.ID})
			m.editor.nativeDraft.UpdateView(m.editor.nativeDescription)
			nativeKeys(m, "c", "i", "w")
			if m.editor.nativePending == nil || m.editor.draft.Description != source {
				t.Fatal("Opaque replacement skipped confirmation")
			}
			if !confirm {
				nativeKeys(m, "n", "esc")
				if m.editor.draft.Description != source || m.editor.mode != editorNormal || m.editor.nativePending != nil {
					t.Fatal("cancel changed source or mode")
				}
				return
			}
			nativeKeys(m, "y", "J", "o", "e", "esc")
			if m.editor.draft.Description != `<p>Hi Joe!</p>` {
				t.Fatal(m.editor.draft.Description)
			}
			nativeKeys(m, "u")
			if m.editor.draft.Description != source {
				t.Fatal("Undo lost Opaque bytes")
			}
		})
	}
}

func TestNativeConfirmationExplainsActualEffect(t *testing.T) {
	const source = `<p><mention-user data-id="jan" data-label="Jan"></mention-user></p>`
	for _, keys := range [][]string{{"x"}, {"c", "i", "w"}} {
		m := nativeEditingModel(source)
		m.editor.descriptionView()
		nativeKeys(m, keys...)
		if m.editor.nativePending == nil || !strings.Contains(m.toast, "object will be DELETED") {
			t.Fatalf("deletion consequence missing: %s", m.toast)
		}
		nativeKeys(m, "n")
		if m.editor.draft.Description != source {
			t.Fatal("cancel changed original")
		}
	}
	m := nativeEditingModel(`<p>Read <img alt='map' src='secret'> now</p>`)
	nativeKeys(m, "space", "2")
	if !strings.Contains(m.toast, "objects retained") || strings.Contains(m.toast, "DELETED") || strings.Contains(m.toast, "secret") {
		t.Fatalf("incorrect structural consequence: %s", m.toast)
	}
	for _, tc := range []struct {
		kind description.ActionKind
		want string
	}{
		{description.ClearDescription, "Entire Description will be DELETED"},
		{description.RecoverText, `Recover "Broken"; HTML/formatting/objects REPLACED on save`},
	} {
		m = nativeEditingModel(`<p>Broken<strong>`)
		m.editor.descriptionView()
		m.applyNative(description.Action{Kind: tc.kind}, false, false)
		if !strings.Contains(m.toast, tc.want) {
			t.Fatalf("missing recovery consequence: %s", m.toast)
		}
	}
}

func TestNativeRecoveryConfirmationEmptyPreviewAndCancel(t *testing.T) {
	m := nativeEditingModel(`<script>hidden-secret</script><p`)
	m.editor.descriptionView()
	m.applyNative(description.Action{Kind: description.RecoverText}, false, true)
	if m.editor.nativePending == nil || !strings.Contains(m.toast, "EMPTY Description") || strings.Contains(m.toast, "hidden-secret") {
		t.Fatalf("misleading empty recovery preview: %s", m.toast)
	}
	nativeKeys(m, "n")
	if m.editor.draft.Description != `<script>hidden-secret</script><p` {
		t.Fatal("cancel changed source")
	}
}

func TestNativeConfirmationConsequencesSurviveNarrowView(t *testing.T) {
	for _, width := range []int{80, 54} {
		for _, tc := range []struct {
			source string
			kind   description.ActionKind
			want   string
		}{
			{`<p><mention-user data-id="jan" data-label="Jan"></mention-user></p>`, description.Delete, "DELETED"},
			{`<p>Broken<strong>`, description.RecoverText, "REPLACED"},
			{`<script>hidden</script><p`, description.RecoverText, "EMPTY"},
		} {
			m := nativeEditingModel(tc.source)
			m.Update(tea.WindowSizeMsg{Width: width, Height: 40})
			m.editor.descriptionView()
			m.applyNative(description.Action{Kind: tc.kind}, false, false)
			nativeKeys(m, "?") // Unrelated keys must not erase a pending warning.
			screen := ansi.Strip(m.View().Content)
			if !strings.Contains(screen, tc.want) || !strings.Contains(screen, "n/Esc") {
				t.Fatalf("width %d missing consequence %s or cancel: %s", width, tc.want, screen)
			}
		}
	}
}
