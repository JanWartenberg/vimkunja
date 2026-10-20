package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestDescriptionInsertPreservesGroupedBackticksAndGermanCharacters(t *testing.T) {
	for _, chunks := range [][]string{{"`", "`", "`", "`"}, {"``", "``"}, {"````"}, {"ü", "Ü", "[", "]"}} {
		m := testModel()
		m.currentProjectID = 1
		m.openNewEditor()
		m.editor.field = fieldDescription
		m.editor.mode = editorInsert
		want := ""
		for _, chunk := range chunks {
			want += chunk
			updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: []rune(chunk)[0], Text: chunk}))
			m = updated.(*model)
			if m.editor.nativeDescription.ContentText() != want {
				t.Fatalf("text chunks %q lost characters: got %q, want %q", chunks, m.editor.draft.Description, want)
			}
		}
		if m.editor.mode != editorInsert {
			t.Fatal("text event left Insert Mode")
		}
	}
}
