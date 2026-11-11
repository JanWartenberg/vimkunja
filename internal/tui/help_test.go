package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

func TestHelpTopicNavigationAndHistory(t *testing.T) {
	m := testModel()
	m.width, m.height = 120, 40
	m, _ = typeCommand(t, m, "h")
	if got := ansi.Strip(m.Render()); !strings.Contains(got, "> Navigation & Views") || !strings.Contains(got, "Rich text") {
		t.Fatalf("help overview missing selectable topics:\n%s", got)
	}
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter})
	if got := ansi.Strip(m.Render()); !strings.Contains(got, "Help / Tasks & Detail") || !strings.Contains(got, "Create a Task") {
		t.Fatalf("selected topic did not open:\n%s", got)
	}
	m = pressKey(m, tea.Key{Code: 'o', Mod: tea.ModCtrl})
	if got := ansi.Strip(m.Render()); !strings.Contains(got, "> Tasks & Detail") {
		t.Fatalf("back did not restore overview selection:\n%s", got)
	}
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	if got := ansi.Strip(m.Render()); !strings.Contains(got, "Help / Tasks & Detail") {
		t.Fatalf("Tab did not navigate forward:\n%s", got)
	}
	m = pressKey(m, tea.Key{Code: '-', Text: "-"})
	if got := ansi.Strip(m.Render()); !strings.Contains(got, "> Tasks & Detail") {
		t.Fatalf("parent did not restore overview selection:\n%s", got)
	}
}

func TestHelpClosePreservesDescriptionDraftAndUndo(t *testing.T) {
	m := nativeEditingModel("<p>Original</p>")
	m.width, m.height = 100, 30
	m = nativeKeys(m, "A", "!", "esc")
	before := m.SessionSnapshot()
	m, _ = typeCommand(t, m, "h")
	if !strings.Contains(ansi.Strip(m.Render()), "> Navigation & Views") {
		t.Fatal("help not visible over Description")
	}
	m, _ = typeCommand(t, m, "h")
	m, cmd := typeCommand(t, m, "q")
	after := m.SessionSnapshot()
	if cmd != nil || m.helpPane != nil || !m.showDetail || after.Focus != "Detail" || !after.DescriptionActive || !after.Dirty || after.Draft.Description != before.Draft.Description {
		t.Fatalf("help did not restore dirty Description: before=%+v after=%+v", before, after)
	}
	m = nativeKeys(m, "u")
	if m.editor.draft.Description != "<p>Original</p>" {
		t.Fatalf("local undo lost: %s", m.editor.draft.Description)
	}
}

func TestHelpFromCommentComposerPreservesDraft(t *testing.T) {
	m := nativeEditingModel("<p>Original</p>")
	m.width, m.height = 100, 30
	m.openComments()
	m.startCommentComposer()
	m = nativeKeys(m, "Comment draft", "esc")
	composer := m.comments.composer
	m, _ = typeCommand(t, m, "h")
	if got := ansi.Strip(m.Render()); !strings.Contains(got, "> Navigation & Views") || strings.Contains(got, "COMMENTS COMMAND") {
		t.Fatalf("help not visible over Comment: %s", got)
	}
	m = nativeKeys(m, "j", "enter")
	m, _ = typeCommand(t, m, "q")
	if m.helpPane != nil || m.focus != paneDetail || m.comments == nil || m.comments.composer != composer || composer.text != "Comment draft" || !strings.Contains(ansi.Strip(m.Render()), "Comment draft") {
		t.Fatal("Comment draft/focus not restored")
	}
}

func TestHelpHistoryRestoresReadingPositionAndTruncatesForward(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 24
	m, _ = typeCommand(t, m, "h")
	m = nativeKeys(m, "enter", "j", "j", "j")
	reading := ansi.Strip(m.Render())
	offset := m.SessionSnapshot().HelpOffset
	if offset == 0 {
		t.Fatal("reference did not scroll")
	}
	m = pressKey(m, tea.Key{Code: 'o', Mod: tea.ModCtrl})
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	if got := ansi.Strip(m.Render()); got != reading || m.SessionSnapshot().HelpOffset != offset {
		t.Fatal("reading position not restored")
	}
	m = pressKey(m, tea.Key{Code: 'o', Mod: tea.ModCtrl})
	m = nativeKeys(m, "j", "enter")
	reading = ansi.Strip(m.Render())
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	if ansi.Strip(m.Render()) != reading {
		t.Fatal("new visit retained forward history")
	}
	m, _ = typeCommand(t, m, "h")
	m = pressKey(m, tea.Key{Code: 'o', Mod: tea.ModCtrl})
	if !strings.Contains(ansi.Strip(m.Render()), "> Navigation & Views") {
		t.Fatal("explicit entry did not reset history")
	}
}

func TestHelpExCompletionDoesNotNavigateAndSearchReturnDoesNotWin(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	m.searchReturn = &searchReturn{input: "previous search"}
	m, _ = typeCommand(t, m, "h")
	m = nativeKeys(m, "enter")
	m = pressKey(m, tea.Key{Code: 'o', Mod: tea.ModCtrl})
	if m.searchOpen || !strings.Contains(ansi.Strip(m.Render()), "> Navigation & Views") {
		t.Fatal("Search stole help history")
	}
	m = nativeKeys(m, ":", "hel")
	m = pressKey(m, tea.Key{Code: tea.KeyTab})
	if m.commandInput != "help" || !m.commandMode {
		t.Fatal("Tab did not complete the Ex command")
	}
	m = pressKey(m, tea.Key{Code: tea.KeyEscape})
	m, _ = typeCommand(t, m, "q")
	m = pressKey(m, tea.Key{Code: 'o', Mod: tea.ModCtrl})
	if !m.searchOpen {
		t.Fatal("Search return no longer works outside help")
	}
}

func TestHelpScrollFeedbackAndResize(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 24
	m, _ = typeCommand(t, m, "h")
	m = nativeKeys(m, "enter", "G")
	if got := ansi.Strip(m.Render()); !strings.Contains(got, "100%") {
		t.Fatalf("end position feedback missing: %s", got)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 80})
	m = updated.(*model)
	if got := m.SessionSnapshot().HelpOffset; got == 0 {
		t.Fatal("reference unexpectedly fits enlarged pane")
	}
	m = nativeKeys(m, "g", "g")
	if m.SessionSnapshot().HelpOffset != 0 {
		t.Fatal("gg did not return to start")
	}
}

func TestHelpLayoutAtSupportedSizes(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}, {44, 24}, {60, 12}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			for _, topic := range []struct {
				path []int
				end  string
			}{
				{[]int{0}, "Select deferred Table / Gantt View"},
				{[]int{1}, "Leave Insert mode or close Comments"},
				{[]int{2}, "Start / end"},
				{[]int{3, 0}, "Esc cancels"},
				{[]int{3, 1}, "Paste literal"},
				{[]int{3, 2}, "URL inside a Link"},
				{[]int{3, 3}, "another dd removes the block"},
				{[]int{3, 4}, "discard complete draft"},
				{[]int{4}, "not Kanban"},
				{[]int{5}, "Quit and discard"},
			} {
				m := testModel()
				m.width, m.height = size[0], size[1]
				m, _ = typeCommand(t, m, "h")
				for _, index := range topic.path {
					for i := 0; i < index; i++ {
						m = nativeKeys(m, "j")
					}
					m = nativeKeys(m, "enter")
				}
				m = nativeKeys(m, "G")
				output := ansi.Strip(m.Render())
				rows := strings.Split(output, "\n")
				if len(rows) > size[1] {
					t.Fatalf("layout has %d rows; want <=%d", len(rows), size[1])
				}
				for _, row := range rows {
					if ansi.StringWidth(row) > size[0] {
						t.Fatalf("row wider than viewport: %q", row)
					}
				}
				// The terminal may split the final explanation across physical rows.
				compact := strings.ReplaceAll(output, " ", "")
				compact = strings.ReplaceAll(compact, "│", "")
				compact = strings.ReplaceAll(compact, "\n", "")
				if !strings.Contains(compact, strings.ReplaceAll(topic.end, " ", "")) {
					t.Fatalf("final explanation unreachable via G (%v):\n%s", topic.path, output)
				}
			}
		})
	}
}

func TestHelpDoesNotHideNewEditorOrReplaceDirtyDraft(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	m.currentProjectID = 1
	m, _ = typeCommand(t, m, "h")
	m, _ = typeCommand(t, m, "new")
	if m.helpPane != nil || !m.showDetail || m.focus != paneDetail || !strings.Contains(ansi.Strip(m.Render()), "NEW TASK") {
		t.Fatal("new editor hidden by help")
	}
	m.editor.draft.Title = "Keep this draft"
	before := m.editor
	m, _ = typeCommand(t, m, "h")
	m, _ = typeCommand(t, m, "new")
	if m.editor != before || m.lastErr == "" {
		t.Fatal("new command replaced dirty draft beneath help")
	}
}

func TestHelpRestoresOriginAcrossPaneSwitchAndErrors(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	m, _ = typeCommand(t, m, "errors")
	m.infoPane.offset = 1
	origin := m.infoPane
	m, _ = typeCommand(t, m, "h")
	m = pressKey(m, tea.Key{Code: 'w', Mod: tea.ModCtrl})
	m = nativeKeys(m, "k")
	if m.focus != paneTasks {
		t.Fatal("cannot leave help with Ctrl+W k")
	}
	m = pressKey(m, tea.Key{Code: 'w', Mod: tea.ModCtrl})
	m = nativeKeys(m, "j")
	if m.focus != paneHelp {
		t.Fatal("cannot refocus help")
	}
	m, _ = typeCommand(t, m, "q")
	if m.focus != paneInfo || m.infoPane != origin || m.infoPane.offset != 1 {
		t.Fatal("error log origin lost")
	}
	m, _ = typeCommand(t, m, "h")
	m.infoPane = nil
	m, _ = typeCommand(t, m, "q")
	if m.focus != paneTasks {
		t.Fatal("missing origin did not fall back safely")
	}
}

func TestHelpSuspendsCommentInputAndProtectsDraftFromContextChanges(t *testing.T) {
	for _, command := range []string{"new", "overview", "search", "errors", "kanban", "labels"} {
		t.Run(command, func(t *testing.T) {
			m := nativeEditingModel("<p>Original</p>")
			m.width, m.height = 100, 30
			m.openComments()
			m.startCommentComposer()
			m = nativeKeys(m, "Keep comment", "esc")
			c := m.comments.composer
			m, _ = typeCommand(t, m, "h")
			updated, _ := m.Update(tea.PasteMsg{Content: "unwanted paste"})
			m = updated.(*model)
			m = nativeKeys(m, "space", "p")
			m, _ = typeCommand(t, m, command)
			if m.editor == nil || m.comments == nil || m.comments.composer != c || c.text != "Keep comment" || m.lastErr == "" {
				t.Fatalf("%s changed hidden Comment state", command)
			}
			m, _ = typeCommand(t, m, "q")
			if m.focus != paneDetail || m.comments.composer != c {
				t.Fatal("cannot resume Comment after rejected navigation")
			}
		})
	}
}

func TestHelpWrapsWideUnicodeAndLongKeys(t *testing.T) {
	original := helpPages[0]
	t.Cleanup(func() { helpPages[0] = original })
	helpPages[0].sections = []helpSection{{title: "UNICODE", entries: []helpEntry{{strings.Repeat("Ctrl+界", 14), strings.Repeat("説明文 ", 30) + "終わり"}}}}
	m := testModel()
	m.width, m.height = 44, 24
	m, _ = typeCommand(t, m, "h")
	m = nativeKeys(m, "enter")
	for i := 0; i < 25; i++ {
		output := ansi.Strip(m.Render())
		for _, row := range strings.Split(output, "\n") {
			if ansi.StringWidth(row) > 44 {
				t.Fatalf("Unicode exceeded pane: %q", row)
			}
		}
		if strings.Contains(output, "終わり") {
			return
		}
		m = nativeKeys(m, "j")
	}
	t.Fatal("last Unicode explanation cannot be reached")
}

func TestHelpKeepsApplicationSaveAndQuitContracts(t *testing.T) {
	store := &fakeStore{}
	m := NewModel(Config{Store: store}).(*model)
	m.currentProjectID = 1
	m.openNewEditor()
	m.editor.draft.Title = "Save from help"
	m, _ = typeCommand(t, m, "h")
	m, cmd := typeCommand(t, m, "qa")
	if cmd != nil || m.lastErr == "" || !m.SessionSnapshot().Dirty {
		t.Fatal(":qa did not protect hidden draft")
	}
	m, cmd = typeCommand(t, m, "wqa")
	if cmd == nil {
		t.Fatal(":wqa did not start saving from help")
	}
	updated, quit := m.Update(cmd())
	m = updated.(*model)
	if store.createdDraft.Title != "Save from help" || quit == nil {
		t.Fatal(":wqa did not save before quit")
	}
	if _, ok := quit().(tea.QuitMsg); !ok {
		t.Fatal(":wqa failed to quit after successful save")
	}
}

func TestHelpLeavingCleanCommentsOpensRequestedContext(t *testing.T) {
	for _, command := range []string{"new", "overview", "errors"} {
		t.Run(command, func(t *testing.T) {
			m := nativeEditingModel("<p>Original</p>")
			m.width, m.height = 100, 30
			m.openComments()
			m, _ = typeCommand(t, m, "h")
			m, _ = typeCommand(t, m, command)
			if m.comments != nil || m.helpPane != nil {
				t.Fatalf("%s kept obsolete Comments/help over requested context", command)
			}
			if command == "new" && (m.editor == nil || m.editor.id != 0 || !strings.Contains(ansi.Strip(m.Render()), "NEW TASK")) {
				t.Fatal("new editor not visible")
			}
		})
	}
}

func TestHelpStaysOpenWhenKanbanRefreshFinishes(t *testing.T) {
	m := nativeEditingModel("<p>Original</p>")
	m.width, m.height = 100, 30
	m.currentProjectID = 1
	original := m.editor
	m, _ = typeCommand(t, m, "h")
	board := vimkunja.KanbanBoard{Availability: vimkunja.KanbanBoardAvailable, Buckets: []vimkunja.KanbanBucket{{ID: 1, Tasks: []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "Refreshed"}}}}}
	updated, _ := m.Update(kanbanBoardLoadedMsg{request: m.taskRequest, projectID: 1, board: board})
	m = updated.(*model)
	if m.helpPane == nil || m.focus != paneHelp || m.editor != original || !strings.Contains(ansi.Strip(m.Render()), "> Navigation & Views") {
		t.Fatal("background Kanban refresh replaced help or its origin")
	}
	m, _ = typeCommand(t, m, "q")
	if m.focus != paneDetail || m.editor != original {
		t.Fatal("cannot resume Detail after background refresh")
	}
}

func TestHelpProtectsHiddenCommentFromGlobalSearchReturn(t *testing.T) {
	m := nativeEditingModel("<p>Original</p>")
	m.width, m.height = 100, 30
	m.searchReturn = &searchReturn{origin: m.captureSearchOrigin(), input: "old query"}
	m.openComments()
	m.startCommentComposer()
	m = nativeKeys(m, "Keep on this Task", "esc")
	original := m.editor
	m, _ = typeCommand(t, m, "h")
	m = pressKey(m, tea.Key{Code: 'w', Mod: tea.ModCtrl})
	m = nativeKeys(m, "k")
	m = pressKey(m, tea.Key{Code: 'o', Mod: tea.ModCtrl})
	if m.searchOpen || m.editor != original || m.helpPane == nil || m.lastErr == "" || m.comments.composer.text != "Keep on this Task" {
		t.Fatal("global Search return bypassed hidden Comment protection")
	}
}

func TestHelpDirectOverviewNavigationClosesCleanComments(t *testing.T) {
	for _, direction := range []string{"k", "w"} {
		t.Run(direction, func(t *testing.T) {
			m := nativeEditingModel("<p>Original</p>")
			m.width, m.height = 100, 30
			m.openComments()
			m, _ = typeCommand(t, m, "h")
			m = pressKey(m, tea.Key{Code: 'w', Mod: tea.ModCtrl})
			m = nativeKeys(m, direction, "-")
			if !m.inOverview() || m.helpPane != nil || m.comments != nil || m.focus != paneTasks {
				t.Fatal("direct Overview navigation retained help or orphaned Comments")
			}
		})
	}
}

func TestHelpQuitFromWorkingPaneProtectsHiddenDraft(t *testing.T) {
	for _, direction := range []string{"k", "w"} {
		t.Run(direction, func(t *testing.T) {
			m := nativeEditingModel("<p>Original</p>")
			m.width, m.height = 100, 30
			m.openComments()
			m.startCommentComposer()
			m = nativeKeys(m, "Unsaved Comment", "esc")
			m, _ = typeCommand(t, m, "h")
			m = pressKey(m, tea.Key{Code: 'w', Mod: tea.ModCtrl})
			m = nativeKeys(m, direction)
			m, cmd := typeCommand(t, m, "q")
			if cmd != nil || m.lastErr == "" || m.comments == nil || m.comments.composer.text != "Unsaved Comment" {
				t.Fatal(":q quit through hidden unsaved Comment")
			}
		})
	}
}

func TestHelpRestoresOverviewSelectionFromSmallerTopicIndex(t *testing.T) {
	for _, key := range []struct {
		name string
		key  tea.Key
	}{
		{"forward Tab", tea.Key{Code: tea.KeyTab}},
		{"forward Ctrl+I", tea.Key{Code: 'i', Mod: tea.ModCtrl}},
		{"parent minus", tea.Key{Code: '-', Text: "-"}},
		{"parent Backspace", tea.Key{Code: tea.KeyBackspace}},
	} {
		t.Run(key.name, func(t *testing.T) {
			m := testModel()
			m.width, m.height = 120, 40
			m, _ = typeCommand(t, m, "h")
			m = nativeKeys(m, "j", "j", "j", "enter", "-", "j", "j", "enter")
			m = pressKey(m, tea.Key{Code: 'o', Mod: tea.ModCtrl})
			if !strings.Contains(ansi.Strip(m.Render()), "> Commands & quitting") {
				t.Fatal("setup did not restore overview selection")
			}
			m = pressKey(m, tea.Key{Code: 'o', Mod: tea.ModCtrl})
			m = pressKey(m, key.key)
			if got := ansi.Strip(m.Render()); !strings.Contains(got, "> Commands & quitting") {
				t.Fatalf("overview selection changed after leaving Rich text index:\n%s", got)
			}
			m = nativeKeys(m, "enter")
			if got := ansi.Strip(m.Render()); !strings.Contains(got, "Help / Commands & quitting") {
				t.Fatalf("restored selection opened wrong topic:\n%s", got)
			}
		})
	}
}
