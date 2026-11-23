package tui

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/searchpattern"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func localListModel() *model {
	m := testModel()
	m.width, m.height, m.focus = 120, 35, paneTasks
	m.viewportReceived = true
	m.tasks = []vimkunja.Task{{ID: 1, ProjectID: 1, Title: "alpha"}, {ID: 2, ProjectID: 1, Title: "beta beta"}, {ID: 3, ProjectID: 1, Title: "BETA"}}
	m.projectTasks = append([]vimkunja.Task(nil), m.tasks...)
	return m
}

func localQuery(m *model, direction, pattern string) tea.Cmd {
	nativeKeys(m, direction)
	_, cmd := m.Update(tea.PasteMsg{Content: pattern})
	return cmd
}

func TestLocalSearchLivePreviewCountsWrapAndClear(t *testing.T) {
	m := localListModel()
	localQuery(m, "/", "beta")
	if m.taskSelected != 1 || m.editor == nil || m.editor.id != 2 || m.focus != paneTasks || len(m.tasks) != 3 {
		t.Fatal("preview did not select without filtering")
	}
	if ui := m.localSearch.input; ui == nil || len(ui.matches) != 3 || ui.selected != 0 {
		t.Fatalf("input: %#v", ui)
	}
	nativeKeys(m, "enter", "n")
	if m.localSearch.current.offset != 5 || m.editor.id != 2 {
		t.Fatal("did not navigate separate occurrences", m.localSearch.current)
	}
	nativeKeys(m, "2", "n")
	if m.editor.id != 2 || m.localSearch.current.offset != 0 || m.toast != "Search wrapped" {
		t.Fatal("counted wrap", m.localSearch.current, m.toast)
	}
	nativeKeys(m, "N")
	if m.editor.id != 3 {
		t.Fatal("reverse did not wrap")
	}
	nativeKeys(m, "space", "h")
	if m.localSearch.highlight {
		t.Fatal("Space h did not hide highlights")
	}
	nativeKeys(m, "n")
	if !m.localSearch.highlight {
		t.Fatal("n did not restore highlights")
	}
	m, _ = typeCommand(t, m, "nohlsearch")
	if m.localSearch.highlight || m.localSearch.pattern != "beta" {
		t.Fatal("nohlsearch changed pattern")
	}
}

func TestLocalSearchCancelRestoresTaskPreviewAndPreviousPattern(t *testing.T) {
	m := localListModel()
	m.previewSelectedTask()
	original := m.editor
	m.searchReturn = &searchReturn{input: "global"}
	localQuery(m, "/", "beta")
	cmd := ctrlW(m)
	if cmd.focus != paneTasks {
		t.Fatal("changed focus during input")
	}
	nativeKeys(m, "esc")
	if m.taskSelected != 0 || m.editor != original || m.focus != paneTasks || m.localSearch.pattern != "" || m.localSearch.input != nil || m.searchReturn.input != "global" {
		t.Fatal("cancel failed to restore origin")
	}
}

func TestLocalProjectSearchRevealsFoldsAndRejectsLateResponses(t *testing.T) {
	for _, cancel := range []bool{true, false} {
		m := localListModel()
		m.focus = paneProjects
		m.projects = []vimkunja.Project{{ID: 1, Title: "Private"}, {ID: 2, ParentProjectID: 1, Title: "Brief"}}
		m.projectTree = newProjectTree(m.projects, map[int64]bool{1: true})
		m.currentProjectID, m.currentViewID = 1, 10
		localQuery(m, "/", "Brief")
		request := m.viewRequest
		if m.projectTree.collapsed[1] || m.currentProjectID != 2 {
			t.Fatal("folded match not revealed")
		}
		if cancel {
			nativeKeys(m, "esc")
			if !m.projectTree.collapsed[1] || m.currentProjectID != 1 || m.currentViewID != 10 || len(m.tasks) != 3 {
				t.Fatal("origin not restored")
			}
			m.Update(viewsLoadedMsg{request: request, projectID: 2, views: []vimkunja.ProjectView{{ID: 20, ViewKind: "list"}}})
			if m.currentProjectID != 1 || m.currentViewID != 10 {
				t.Fatal("late preview replaced origin")
			}
		} else {
			nativeKeys(m, "enter")
			if m.projectTree.collapsed[1] || m.localSearch.pattern != "Brief" {
				t.Fatal("accepted fold state lost")
			}
			m.Update(viewsLoadedMsg{request: request, projectID: 2, err: errors.New("read failed")})
			if m.currentProjectID != 1 || !strings.Contains(m.lastErr, "preview failed") {
				t.Fatal("accepted pending preview did not restore on failure", m.lastErr)
			}
		}
	}
}

func TestLocalSearchInvalidNoMatchHistoryAndEmptyRepeat(t *testing.T) {
	m := localListModel()
	localQuery(m, "/", "[")
	nativeKeys(m, "enter")
	if m.localSearch.input == nil || m.localSearch.input.err == "" || m.taskSelected != 0 {
		t.Fatal("invalid pattern accepted")
	}
	nativeKeys(m, "esc")
	localQuery(m, "/", "absent")
	nativeKeys(m, "enter")
	if m.localSearch.input != nil || m.localSearch.pattern != "absent" || !strings.Contains(m.toast, "not found") {
		t.Fatal("valid no-match search not accepted")
	}
	localQuery(m, "/", "beta")
	nativeKeys(m, "enter")
	localQuery(m, "?", "draft")
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if m.localSearch.input.query != "beta" {
		t.Fatal("search history not recalled")
	}
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if m.localSearch.input.query != "draft" {
		t.Fatal("history draft not restored")
	}
	nativeKeys(m, "esc", "/", "enter")
	if m.localSearch.input != nil || m.localSearch.pattern != "beta" || len(m.commandHistory) != 0 {
		t.Fatal("empty repeat or independent history failed")
	}
}

func TestLocalSearchDetailTargetsDraftTextAndRestoresCursor(t *testing.T) {
	m := nativeEditingModel(`<p>before <strong>needle</strong> needle</p><p><a href="https://hidden.example">link</a></p>`)
	m.width, m.height = 100, 35
	e := m.editor
	e.descriptionView()
	originalTarget := e.nativeDraft.Target()
	originalSource := e.draft.Description
	e.descScroll, e.detailScroll = 2, 1
	localQuery(m, "/", "needle")
	if m.localSearch.input == nil || len(m.localSearch.input.matches) != 2 || e.nativeDraft.Target() == originalTarget || e.field != fieldDescription {
		t.Fatal("Description match did not move cursor")
	}
	nativeKeys(m, "esc")
	if e.nativeDraft.Target() != originalTarget || e.descScroll != 2 || e.detailScroll != 1 || e.draft.Description != originalSource || e.dirty() {
		t.Fatal("Description cancellation mutated draft or navigation")
	}
	e.draft.Title = "unsaved needle"
	e.field = fieldTitle
	localQuery(m, "/", "unsaved")
	if m.localSearch.input == nil || len(m.localSearch.input.matches) != 1 {
		t.Fatal("dirty draft not searched")
	}
	nativeKeys(m, "esc")
	m.focus = paneTasks
	nativeKeys(m, "/")
	if m.localSearch.input != nil || m.lastErr != errUnsavedNavigation.Error() || e.draft.Title != "unsaved needle" {
		t.Fatal("dirty draft navigation unprotected")
	}
}

func TestLocalSearchInputAndModeRouting(t *testing.T) {
	m := nativeEditingModel("body")
	m.editor.field, m.editor.mode = fieldTitle, editorInsert
	nativeKeys(m, "/", "?", "n", "N")
	if m.localSearch.input != nil || !strings.Contains(m.editor.draft.Title, "/?nN") {
		t.Fatal("Insert search keys intercepted")
	}
	m.editor.mode = editorNormal
	nativeKeys(m, "g", "n", "enter")
	if m.editor.field != fieldDescription || !m.editor.descriptionActive {
		t.Fatal("gn navigation intercepted")
	}
	nativeKeys(m, "v", "/")
	if m.localSearch.input != nil {
		t.Fatal("Visual search activated")
	}
	for _, pane := range []paneID{paneHelp, paneInfo} {
		m := localListModel()
		m.focus = pane
		nativeKeys(m, "/", "n")
		if m.localSearch.input != nil || m.taskSelected != 0 {
			t.Fatal("unsupported pane searched background")
		}
	}
}

func TestLocalSearchScopeOrderAndSharedPattern(t *testing.T) {
	m := localListModel()
	m.kanbanActive = true
	m.kanbanBoard.Buckets = []vimkunja.KanbanBucket{{Tasks: []vimkunja.Task{{ID: 20, Title: "beta"}, {ID: 21, Title: "beta"}}}, {Tasks: []vimkunja.Task{{ID: 30, Title: "beta"}}}}
	localQuery(m, "/", "beta")
	var ids []int64
	for _, match := range m.localSearch.input.matches {
		ids = append(ids, match.position.id)
	}
	if !reflect.DeepEqual(ids, []int64{20, 21, 30}) {
		t.Fatal(ids)
	}
	nativeKeys(m, "enter")
	m.focus, m.kanbanActive = paneProjects, false
	m.projects = []vimkunja.Project{{ID: 10, Title: "beta"}}
	m.projectTree = newProjectTree(m.projects, nil)
	nativeKeys(m, "n")
	if m.currentProjectID != 10 {
		t.Fatal("shared pattern not applied to focused Pane")
	}
	// Filtered-out source Tasks must never become local matches.
	m = localListModel()
	m.tasks = m.tasks[:1]
	p, _ := searchpattern.Compile("beta")
	if matches := m.localMatches(p); len(matches) != 0 {
		t.Fatal("searched filtered-out Tasks", matches)
	}
}

func TestLocalSearchPreservesKanbanCardSelection(t *testing.T) {
	m := localListModel()
	m.kanbanActive = true
	m.kanbanBoard.Buckets = []vimkunja.KanbanBucket{{Title: "Bucket", Tasks: []vimkunja.Task{{ID: 20, Title: "beta suffix"}, {ID: 21, Title: "other"}}}}
	localQuery(m, "/", "beta")
	nativeKeys(m, "enter")
	buffer := uv.NewScreenBuffer(50, 10)
	uv.NewStyledString(renderKanbanBoard(m, 50, 10)).Draw(buffer, uv.Rect(0, 0, 50, 10))
	for _, line := range buffer.Lines {
		text := ansi.Strip(line.String())
		if at := strings.Index(text, "suffix"); at >= 0 {
			style := line[ansi.StringWidth(text[:at])].Style
			if style.Attrs&uv.AttrBold == 0 || style.Underline != uv.UnderlineStyleSingle || style.Fg == nil {
				t.Fatalf("search removed the selected card's style outside its match: %#v", style)
			}
			return
		}
	}
	t.Fatal("selected card missing")
}

func TestLocalSearchNarrowRenderingUsesSharedInputCursor(t *testing.T) {
	for _, width := range []int{1, 4, 20, 80, 120} {
		m := localListModel()
		m.width = width
		localQuery(m, "/", strings.Repeat("beta", 15))
		view := m.View()
		for _, line := range strings.Split(view.Content, "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatalf("width %d overflow: %q", width, line)
			}
		}
		if width >= 4 && view.Cursor == nil {
			t.Fatalf("width %d missing shared cursor", width)
		}
	}
}

func TestLocalSearchBackwardDirectionAndEditorCounts(t *testing.T) {
	m := localListModel()
	m.taskSelected = 2
	localQuery(m, "?", "beta")
	if ui := m.localSearch.input; ui.selected != 1 || ui.matches[ui.selected].position.id != 2 {
		t.Fatalf("backward preview: %#v", ui)
	}
	nativeKeys(m, "enter", "n")
	if m.localSearch.current.offset != 0 {
		t.Fatal("n did not follow backward search")
	}
	nativeKeys(m, "N")
	if m.localSearch.current.offset != 5 {
		t.Fatal("N did not reverse backward search")
	}
	m = nativeEditingModel("before beta beta beta after")
	localQuery(m, "/", "beta")
	nativeKeys(m, "enter", "2", "n")
	if m.localSearch.current.offset != 17 {
		t.Fatal("Description count ignored", m.localSearch.current)
	}
}

func TestLocalSearchHighlightRevealsLongTitleAndPreservesAnchors(t *testing.T) {
	m := localListModel()
	m.tasks[1].Title = strings.Repeat("prefix ", 20) + "needle"
	localQuery(m, "/", "needle")
	nativeKeys(m, "enter")
	rendered := m.Render()
	if !strings.Contains(ansi.Strip(rendered), "needle") {
		t.Fatal("offscreen title match not revealed")
	}
	m.focus = paneDetail
	localQuery(m, "/", "needle")
	nativeKeys(m, "enter")
	rendered = m.Render()
	if !strings.Contains(ansi.Strip(rendered), "needle") {
		t.Fatal("Detail title match not revealed")
	}
	// Clipping must not turn a mid-title word into a false ^ match.
	m.localSearch.compiled, _ = searchpattern.Compile(`^needle`)
	value := m.highlightLocalDetailField(fieldTitle, "…needle")
	if strings.Contains(value, "\x1b[") {
		t.Fatalf("highlight reinterpreted a clipped title as a logical line: %q", value)
	}
}

func TestLocalSearchPreservesSQLiteAndReloadsOriginMetadata(t *testing.T) {
	m, db := reloadRegressionModel(t, true)
	e := m.editor
	var before string
	if err := db.QueryRow(`SELECT description FROM tasks WHERE id=1`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	localQuery(m, "/", "Alpha")
	nativeKeys(m, "enter", "n", "N")
	var after string
	if err := db.QueryRow(`SELECT description FROM tasks WHERE id=1`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after || e.dirty() {
		t.Fatal("search changed persisted or draft content")
	}
	// Origin metadata that finished while another Task was previewed is retried.
	m = localListModel()
	m.previewSelectedTask()
	original := m.editor
	localQuery(m, "/", "beta")
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if m.editor != original || cmd == nil {
		t.Fatal("restored editor left with orphaned loading state")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatal("missing metadata batch")
	}
	for _, load := range batch {
		m.Update(load())
	}
	if original.labelsLoading || original.workflowStatusLoading || original.commentsLoading {
		t.Fatal("restored metadata never completed")
	}
}

func TestLocalSearchUpdatesWhenDetailMetadataArrives(t *testing.T) {
	m := nativeEditingModel("body")
	e := m.editor
	localQuery(m, "/", "fresh-label")
	if len(m.localSearch.input.matches) != 0 {
		t.Fatal("unexpected initial metadata")
	}
	m.Update(taskLabelsLoadedMsg{taskID: e.id, labels: []vimkunja.Label{{ID: 7, Title: "fresh-label"}}})
	if len(m.localSearch.input.matches) != 1 || e.field != fieldLabels {
		t.Fatal("new metadata was not incorporated")
	}
	if e.dirty() {
		t.Fatal("metadata/search made draft dirty")
	}
}

func TestLocalSearchNavigatesOpaqueOccurrencesAndZeroWidthEnds(t *testing.T) {
	for _, tc := range []struct {
		source, query string
		want          []int
	}{
		{`<table><tr><td>needle needle needle</td></tr></table>`, "needle", []int{11, 18, 4}},
		{"word", "$", []int{4, 3, 1}},
	} {
		m := nativeEditingModel(tc.source)
		localQuery(m, "/", tc.query)
		nativeKeys(m, "enter")
		var got []int
		for range 3 {
			nativeKeys(m, "n")
			got = append(got, m.localSearch.current.offset)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q: got %v want %v", tc.query, got, tc.want)
		}
	}
}

func TestLocalSearchHighlightKeepsANSIOutOfVisibleTaskTitles(t *testing.T) {
	// Unlike ansi.Strip, strip only complete SGR sequences. A nested styler can
	// split an escape into separately styled runes that a terminal shows as text.
	sgr := regexp.MustCompile(`\x1b\[[0-9;:]*m`)
	for _, prefix := range []string{"\x1b[1m", "\x1b[2;38;5;245m", "\x1b[38;2;240;100;180m"} {
		m := localListModel()
		m.localSearch.compiled, _ = searchpattern.Compile("vim")
		m.localSearch.highlight = true
		m.localSearch.current = localPosition{pane: paneTasks, id: 1, field: fieldTitle, offset: 12}
		m.localSearch.currentValid = true
		value := prefix + "Subprojekt: Vimkunja" + "\x1b[m"
		painted := m.highlightLocalValue(paneTasks, 1, fieldTitle, value)
		if got := sgr.ReplaceAllString(painted, ""); got != "Subprojekt: Vimkunja" {
			t.Fatalf("malformed SGR output: %q (raw %q)", got, painted)
		}
		if got := ansi.Strip(painted); got != "Subprojekt: Vimkunja" {
			t.Fatalf("style %q leaked into title: %q (raw %q)", prefix, got, painted)
		}
	}
	for _, done := range []bool{false, true} {
		for _, width := range []int{70, 95, 120} {
			m := localListModel()
			m.width = width
			m.projectsPaneCollapsed = true
			m.tasks = []vimkunja.Task{
				{ID: 1, ProjectID: 1, Title: "Subprojekt: Vimkunja", Done: done},
				{ID: 2, ProjectID: 1, Title: "Vimkunja Code Review"},
			}
			localQuery(m, "/", "vim")
			nativeKeys(m, "enter")
			for range 2 {
				visible := sgr.ReplaceAllString(m.Render(), "")
				for _, task := range m.tasks {
					if !strings.Contains(visible, task.Title) {
						t.Fatalf("done=%v: ANSI leaked into highlighted title %q:\n%s", done, task.Title, visible)
					}
				}
				nativeKeys(m, "n")
			}
		}
	}
}

func TestLocalSearchHighlightDoesNotStickAfterOrdinaryTaskNavigation(t *testing.T) {
	m := localListModel()
	m.tasks = []vimkunja.Task{
		{ID: 1, Title: "allg. Präsentation: Grundkurs Möwen", Priority: 1},
		{ID: 2, Title: "allg. Präsentation: Vogelstimmen", Priority: 1},
		{ID: 3, Title: "Fotos nachbearbeiten (2025)", Priority: 1},
	}
	localQuery(m, "/", `allg\.`)
	nativeKeys(m, "enter", "n", "N", "j")
	if m.taskSelected != 1 || !m.localSearch.currentValid || m.localSearch.current.id != 1 {
		t.Fatalf("search/navigation setup failed: selection=%d search=%#v", m.taskSelected, m.localSearch)
	}
	terminalLines := func(text string) []uv.Line {
		buffer := uv.NewScreenBuffer(90, 10)
		buffer.Method = ansi.GraphemeWidth
		uv.NewStyledString(text).Draw(buffer, uv.Rect(0, 0, 90, 10))
		return buffer.Lines
	}
	for _, key := range []string{"k", "j", "j", "k"} {
		painted := terminalLines(renderTasks(m, 90, 10))
		m.localSearch.highlight = false
		baseline := terminalLines(renderTasks(m, 90, 10))
		m.localSearch.highlight = true
		found := 0
		for row, line := range baseline {
			text := ansi.Strip(line.String())
			at := strings.Index(text, "Präsentation")
			if at < 0 {
				continue
			}
			found++
			column := ansi.StringWidth(text[:at])
			if !painted[row][column].Style.Equal(&line[column].Style) {
				t.Fatalf("Task %q retained search color outside the match after j/k: got %#v want %#v", text, painted[row][column].Style, line[column].Style)
			}
			selectedTitle := m.tasks[m.taskSelected].Title
			if !strings.Contains(text, selectedTitle) {
				matchColumn := ansi.StringWidth(text[:strings.Index(text, "allg.")])
				got := painted[row][matchColumn].Style
				want := line[matchColumn].Style
				want.Attrs |= uv.AttrReverse
				if !got.Equal(&want) {
					t.Fatalf("active search color stayed on an unselected Task after j/k: got %#v want %#v", got, want)
				}
			}
		}
		if found != 2 {
			t.Fatalf("missing Task rows in render: %q", ansi.Strip(renderTasks(m, 90, 10)))
		}
		nativeKeys(m, key)
	}
}

func TestLocalSearchActiveMatchFollowsPaneSelectionAndCursor(t *testing.T) {
	for _, pane := range []paneID{paneProjects, paneTasks, paneDetail} {
		m := localListModel()
		if pane == paneProjects {
			m.focus = paneProjects
			m.projects = []vimkunja.Project{{ID: 1, Title: "needle one"}, {ID: 2, Title: "needle two"}}
		} else if pane == paneTasks {
			m.tasks[0].Title, m.tasks[1].Title = "needle one", "needle two"
		} else {
			m = nativeEditingModel("needle needle")
		}
		localQuery(m, "/", "needle")
		nativeKeys(m, "enter")
		if _, active := m.activeLocalMatch(); !active {
			t.Fatalf("pane %d: search result not active", pane)
		}
		motion := "j"
		if pane == paneDetail {
			motion = "h"
		}
		nativeKeys(m, motion)
		pattern, current, visible := m.localPattern()
		if pattern == nil || !visible || current.pane != paneID(-1) {
			t.Fatalf("pane %d: ordinary movement retained the old accent or removed all highlights: %#v", pane, current)
		}
		if pane != paneProjects {
			nativeKeys(m, "n")
			if _, active := m.activeLocalMatch(); !active {
				t.Fatalf("pane %d: n did not restore active marking", pane)
			}
		}
	}
}

type localPagedStore struct {
	vimkunja.Store
	calls []int
}

func (s *localPagedStore) ListTasksInView(_ context.Context, _, _ int64, page, _ int) ([]vimkunja.Task, int, error) {
	s.calls = append(s.calls, page)
	return []vimkunja.Task{{ID: int64(page), Title: "page"}}, 3, nil
}

func TestProjectListLoadsEveryPageForOffscreenSearch(t *testing.T) {
	m := localListModel()
	store := &localPagedStore{}
	m.store = store
	cmd := m.loadTasks(1, 10)
	m.Update(cmd())
	if len(m.tasks) != 3 || !reflect.DeepEqual(store.calls, []int{1, 2, 3}) {
		t.Fatal("incomplete search context", store.calls, m.tasks)
	}
}
