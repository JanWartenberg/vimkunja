package tui

import (
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

func motionKey(r rune) tea.Key { return tea.Key{Code: r, Text: string(r)} }

func pressMotionKeys(m *model, keys string) *model {
	for _, key := range keys {
		m = pressKey(m, motionKey(key))
	}
	return m
}

func motionEditor(field editorField, text string, cursor int) *model {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "title", Description: "description"})
	m.editor.field = field
	m.editor.descriptionActive = field == fieldDescription
	m.editor.cursor = cursor
	m.editor.setFieldValue(text)
	m.editor.original = cloneTaskDraft(m.editor.draft)
	if field == fieldDescription {
		m.comments = &commentsUI{composer: &commentComposer{text: text, cursor: cursor, mode: editorNormal}}
	}
	return m
}

// The multiline text engine is exercised through the Comment composer. Native
// Description motions/formatting have their document-level integration tests.
func motionCursor(m *model) int {
	if m.comments != nil {
		return m.comments.composer.cursor
	}
	return m.editor.cursor
}
func motionText(m *model) string {
	if m.comments != nil {
		return m.comments.composer.text
	}
	return m.editor.fieldValue()
}
func motionMode(m *model) editorMode {
	if m.comments != nil {
		return m.comments.composer.mode
	}
	return m.editor.mode
}

func TestTextEditorLineMotions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		field  editorField
		text   string
		cursor int
		keys   string
		want   int
	}{
		{name: "zero description", field: fieldDescription, text: "one\n  two", cursor: 8, keys: "0", want: 4},
		{name: "caret description", field: fieldDescription, text: "one\n  two", cursor: 8, keys: "^", want: 6},
		{name: "dollar description", field: fieldDescription, text: "one\n  two", cursor: 5, keys: "$", want: 9},
		{name: "zero title", field: fieldTitle, text: "  title", cursor: 5, keys: "0", want: 0},
		{name: "caret title", field: fieldTitle, text: "  title", cursor: 5, keys: "^", want: 2},
		{name: "dollar title", field: fieldTitle, text: "  title", cursor: 2, keys: "$", want: 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pressMotionKeys(motionEditor(tc.field, tc.text, tc.cursor), tc.keys)
			if motionCursor(m) != tc.want {
				t.Fatalf("%s cursor = %d, want %d", tc.keys, motionCursor(m), tc.want)
			}
		})
	}
}

func TestTextEditorWordMotionsUseVimWordAndWORDClasses(t *testing.T) {
	const text = "alpha, beta  gamma\nδelta"
	for _, tc := range []struct {
		name   string
		cursor int
		keys   string
		want   int
	}{
		{name: "word crosses punctuation", cursor: 0, keys: "w", want: 5},
		{name: "word reaches next keyword", cursor: 5, keys: "w", want: 7},
		{name: "word backward", cursor: 7, keys: "b", want: 5},
		{name: "word end", cursor: 0, keys: "e", want: 4},
		{name: "word end punctuation", cursor: 4, keys: "e", want: 5},
		{name: "WORD skips punctuation boundary", cursor: 0, keys: "W", want: 7},
		{name: "WORD backward", cursor: 12, keys: "B", want: 7},
		{name: "WORD end", cursor: 0, keys: "E", want: 5},
		{name: "motions cross newline", cursor: 13, keys: "w", want: 19},
		{name: "unicode is one word", cursor: 19, keys: "e", want: 23},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pressMotionKeys(motionEditor(fieldDescription, text, tc.cursor), tc.keys)
			if motionCursor(m) != tc.want {
				t.Fatalf("%s cursor = %d, want %d", tc.keys, motionCursor(m), tc.want)
			}
		})
	}
}

func TestTextEditorMotionCounts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		text   string
		cursor int
		keys   string
		want   int
	}{
		{name: "word", text: "one two three", keys: "2w", want: 8},
		{name: "WORD", text: "one,two three four", keys: "2W", want: 14},
		{name: "end word", text: "one two three", keys: "2e", want: 6},
		{name: "horizontal", text: "abcdef", keys: "3l", want: 3},
		{name: "vertical", text: "a\nbb\nccc\ndddd", keys: "2j", want: 5},
		{name: "vertical preserves desired column", text: "abcd\nx\nwxyz", cursor: 3, keys: "2j", want: 10},
		{name: "counted line end", text: "one\ntwo\nthree", keys: "2$", want: 7},
		{name: "counted G", text: "one\ntwo\nthree", keys: "2G", want: 4},
		{name: "counted gg", text: "one\ntwo\nthree", keys: "2gg", want: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pressMotionKeys(motionEditor(fieldDescription, tc.text, tc.cursor), tc.keys)
			if motionCursor(m) != tc.want {
				t.Fatalf("%s cursor = %d, want %d", tc.keys, motionCursor(m), tc.want)
			}
		})
	}
}

func TestTierOneDeleteAndChangeCombinations(t *testing.T) {
	for _, tc := range []struct {
		name       string
		text       string
		cursor     int
		keys       string
		wantText   string
		wantCursor int
		wantMode   editorMode
	}{
		{name: "delete word", text: "one two three", keys: "dw", wantText: "two three"},
		{name: "delete backward word", text: "one two three", cursor: 4, keys: "db", wantText: "two three"},
		{name: "delete through word end", text: "one two three", keys: "de", wantText: " two three"},
		{name: "delete inner word", text: "one two three", cursor: 5, keys: "diw", wantText: "one  three", wantCursor: 4},
		{name: "delete a word", text: "one two three", cursor: 5, keys: "daw", wantText: "one three", wantCursor: 4},
		{name: "delete final a word", text: "one two", cursor: 5, keys: "daw", wantText: "one", wantCursor: 3},
		{name: "change word motion preserves whitespace", text: "one two three", keys: "cw", wantText: " two three", wantMode: editorInsert},
		{name: "change inner word", text: "one two three", cursor: 5, keys: "ciw", wantText: "one  three", wantCursor: 4, wantMode: editorInsert},
		{name: "change a word", text: "one two three", cursor: 5, keys: "caw", wantText: "one three", wantCursor: 4, wantMode: editorInsert},
		{name: "change to line end", text: "one two", cursor: 4, keys: "C", wantText: "one ", wantCursor: 4, wantMode: editorInsert},
		{name: "count after operator", text: "one two three", keys: "d2w", wantText: "three"},
		{name: "count before operator", text: "one two three", keys: "2dw", wantText: "three"},
		{name: "operator counts multiply", text: "one two three four five six seven", keys: "2d3w", wantText: "seven"},
		{name: "counted line delete", text: "one\ntwo\nthree", keys: "2dd", wantText: "three"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pressMotionKeys(motionEditor(fieldDescription, tc.text, tc.cursor), tc.keys)
			if motionText(m) != tc.wantText || motionCursor(m) != tc.wantCursor || motionMode(m) != tc.wantMode {
				t.Fatalf("%s = text:%q cursor:%d mode:%v, want %q/%d/%v", tc.keys, motionText(m), motionCursor(m), motionMode(m), tc.wantText, tc.wantCursor, tc.wantMode)
			}
		})
	}
}

func TestTierOneWordObjectsHandleWhitespacePunctuationAndUnicode(t *testing.T) {
	for _, tc := range []struct {
		name     string
		text     string
		cursor   int
		keys     string
		wantText string
	}{
		{name: "inner whitespace", text: "one   two", cursor: 4, keys: "diw", wantText: "onetwo"},
		{name: "around whitespace takes next word", text: "one   two", cursor: 4, keys: "daw", wantText: "one"},
		{name: "punctuation is a word", text: "one...two", cursor: 4, keys: "diw", wantText: "onetwo"},
		{name: "unicode word", text: "one δelta three", cursor: 5, keys: "diw", wantText: "one  three"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pressMotionKeys(motionEditor(fieldDescription, tc.text, tc.cursor), tc.keys)
			if got := motionText(m); got != tc.wantText {
				t.Fatalf("%s = %q, want %q", tc.keys, got, tc.wantText)
			}
		})
	}
}

func TestTierOneCharacterCommandsUseDisplayedLineEndCharacter(t *testing.T) {
	for _, tc := range []struct {
		keys     string
		wantText string
		wantMode editorMode
	}{
		{keys: "$rX", wantText: "aX\ncd"},
		{keys: "$~", wantText: "aB\ncd"},
		{keys: "$x", wantText: "a\ncd"},
		{keys: "$Crest", wantText: "arest\ncd", wantMode: editorInsert},
	} {
		t.Run(tc.keys, func(t *testing.T) {
			m := pressMotionKeys(motionEditor(fieldDescription, "ab\ncd", 0), tc.keys)
			if got := motionText(m); got != tc.wantText || motionMode(m) != tc.wantMode {
				t.Fatalf("%s = %q in mode %v, want %q in mode %v", tc.keys, got, motionMode(m), tc.wantText, tc.wantMode)
			}
		})
	}
}

func TestTierOneReplaceAndCountedCharacterDelete(t *testing.T) {
	for _, tc := range []struct {
		name       string
		text       string
		cursor     int
		keys       string
		wantText   string
		wantCursor int
	}{
		{name: "replace character", text: "one", cursor: 1, keys: "rX", wantText: "oXe", wantCursor: 1},
		{name: "replace count", text: "abcde", cursor: 1, keys: "3rX", wantText: "aXXXe", wantCursor: 1},
		{name: "replace stops at newline", text: "ab\ncd", cursor: 1, keys: "3rX", wantText: "aX\ncd", wantCursor: 1},
		{name: "delete character count", text: "abcdef", cursor: 1, keys: "3x", wantText: "aef", wantCursor: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pressMotionKeys(motionEditor(fieldDescription, tc.text, tc.cursor), tc.keys)
			if motionText(m) != tc.wantText || motionCursor(m) != tc.wantCursor {
				t.Fatalf("%s = %q at %d, want %q at %d", tc.keys, motionText(m), motionCursor(m), tc.wantText, tc.wantCursor)
			}
		})
	}
}

func TestNormalModeTildeTogglesCaseWithCountsAndStopsAtLineEnd(t *testing.T) {
	for _, tc := range []struct {
		name       string
		field      editorField
		text       string
		cursor     int
		keys       string
		wantText   string
		wantCursor int
	}{
		{name: "description count", field: fieldDescription, text: "aBδ\nNext", keys: "3~", wantText: "AbΔ\nNext", wantCursor: 3},
		{name: "description line boundary", field: fieldDescription, text: "aB\ncd", cursor: 1, keys: "3~", wantText: "ab\ncd", wantCursor: 2},
		{name: "title", field: fieldTitle, text: "Task", keys: "~", wantText: "task", wantCursor: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pressMotionKeys(motionEditor(tc.field, tc.text, tc.cursor), tc.keys)
			if got := motionText(m); got != tc.wantText || motionCursor(m) != tc.wantCursor {
				t.Fatalf("%s = %q at %d, want %q at %d", tc.keys, got, motionCursor(m), tc.wantText, tc.wantCursor)
			}
		})
	}

	m := testModel()
	m.comments = &commentsUI{composer: &commentComposer{text: "aBδ", mode: editorNormal}}
	m = pressMotionKeys(m, "2~")
	if got := m.comments.composer; got.text != "Abδ" || got.cursor != 2 {
		t.Fatalf("Comment 2~ = %q at %d, want Abδ at 2", got.text, got.cursor)
	}
}

func TestTierOneMotionsAreSharedByTitleAndCommentComposer(t *testing.T) {
	m := pressMotionKeys(motionEditor(fieldTitle, "one two three", 0), "2dw")
	if m.editor.draft.Title != "three" || m.editor.cursor != 0 {
		t.Fatalf("title 2dw = %q at %d, want three at 0", m.editor.draft.Title, m.editor.cursor)
	}

	m = testModel()
	m.comments = &commentsUI{composer: &commentComposer{text: "one two three", mode: editorNormal}}
	m = pressMotionKeys(m, "2dw")
	if m.comments.composer.text != "three" || m.comments.composer.cursor != 0 {
		t.Fatalf("Comment 2dw = %q at %d, want three at 0", m.comments.composer.text, m.comments.composer.cursor)
	}
	m = pressMotionKeys(m, "ciw")
	if m.comments.composer.mode != editorInsert || m.comments.composer.text != "" {
		t.Fatalf("Comment ciw = text:%q mode:%v, want empty Insert", m.comments.composer.text, m.comments.composer.mode)
	}
}

func TestTierOnePrefixesDoNotLeakOrChangeInsertText(t *testing.T) {
	m := motionEditor(fieldDescription, "one two", 0)
	pressMotionKeys(m, "dQw")
	if motionText(m) != "one two" || motionCursor(m) != 4 {
		t.Fatal("invalid operator prefix leaked")
	}
	m = motionEditor(fieldTitle, "one two", 0)
	pressMotionKeys(m, "d")
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	if m.editor == nil || m.editor.normal.pending() {
		t.Fatal("Esc did not cancel pending operator")
	}
	m = motionEditor(fieldDescription, "", 0)
	m.comments.composer.mode = editorInsert
	pressMotionKeys(m, "dw2e")
	if motionText(m) != "dw2e" {
		t.Fatal("Insert interpreted motions")
	}
}
func TestTierOneEditPreservesDirtyAndDiscardContract(t *testing.T) {
	m := nativeEditingModel("one two")
	pressMotionKeys(m, "dw")
	if !m.editor.dirty() {
		t.Fatal("dw did not mark Description dirty")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	m = pressKey(m, tea.Key{Code: tea.KeyEsc})
	if m.editor == nil || m.lastErr == "" {
		t.Fatal("Esc closed a dirty Task")
	}
}
