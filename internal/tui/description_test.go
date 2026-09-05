package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/vimkunja/vimkunja/internal/vimkunja"

	tea "charm.land/bubbletea/v2"
)

func TestDescriptionTerminalTextRendersHTMLParagraphs(t *testing.T) {
	got := descriptionTerminalText("<p>Fresh <strong>coffee</strong> beans</p><p>Ground daily</p>")
	if want := "Fresh coffee beans\nGround daily"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextRendersHeadingsAndBreaks(t *testing.T) {
	got := descriptionTerminalText("<p>First<br>Second</p><h2>Heading</h2><p>After</p>")
	if want := "First\nSecond\nHeading\n\nAfter"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextNormalizesHTMLWhitespace(t *testing.T) {
	got := descriptionTerminalText("<p>First\nSecond\tThird  Fourth</p>")
	if want := "First Second Third Fourth"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextRendersNestedLists(t *testing.T) {
	got := descriptionTerminalText("<ul><li>One</li><li>Two<ol><li>Nested</li></ol></li></ul><ol><li>First</li><li>Second</li></ol>")
	if want := "- One\n- Two\n  1. Nested\n1. First\n2. Second"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextRendersLinksWithoutRepeatingURLs(t *testing.T) {
	got := descriptionTerminalText(`<p><a href="https://example.test/docs">Docs</a> and <a href="https://example.test/same">https://example.test/same</a> <a href="https://example.test/empty"></a></p>`)
	if want := "Docs (https://example.test/docs) and https://example.test/same https://example.test/empty"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextRendersImagesAndOtherMediaSafely(t *testing.T) {
	got := descriptionTerminalText(`<p>Before <img alt="coffee beans"> <img src="https://example.test/image"> <video src="https://example.test/video"></video></p>`)
	if want := "Before [Bild: coffee beans] [Bild] [Medieninhalt]"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextRendersTableRows(t *testing.T) {
	got := descriptionTerminalText(`<table><tr><th>Term</th><th>Meaning</th></tr><tr><td>HTML<br>5</td><td>Markup</td></tr></table>`)
	if want := "Term | Meaning\nHTML 5 | Markup"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextRendersCheckboxListsReadOnly(t *testing.T) {
	got := descriptionTerminalText(`<ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>Open</p></li><li data-type="taskItem" data-checked="true"><p>Done</p></li></ul>`)
	if want := "- [ ] Open\n- [x] Done"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextRendersUnknownElementsOrExplainsEmptyOnes(t *testing.T) {
	got := descriptionTerminalText(`<p>Known</p><custom-node>Readable</custom-node><empty-node></empty-node>`)
	if want := "Known\nReadable\n[Nicht unterstützter Inhalt]"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextRemovesTerminalControls(t *testing.T) {
	got := descriptionTerminalText("<p>Safe\x1b[31m red\u009bZ</p>")
	if want := "Safe[31m redZ"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextToleratesMalformedHTML(t *testing.T) {
	got := descriptionTerminalText("<p>Visible &amp; <strong>safe text")
	if want := "Visible & safe text"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextDoesNotExposeUnreadableMarkup(t *testing.T) {
	got := descriptionTerminalText("<p")
	if want := "[Nicht lesbarer Beschreibungsinhalt]"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextPreservesPlainTextLineBreaks(t *testing.T) {
	got := descriptionTerminalText("First\nSecond\x1b")
	if want := "First\nSecond"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestDescriptionTerminalTextCollapsesConsecutiveEmptyParagraphs(t *testing.T) {
	got := descriptionTerminalText("<p>First</p><p></p><p> </p><p>Second</p>")
	if want := "First\n\nSecond"; got != want {
		t.Fatalf("descriptionTerminalText() = %q, want %q", got, want)
	}
}

func TestTaskDetailRendersHTMLDescriptionWithoutSourceTags(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee", Description: "<p>Fresh <strong>coffee</strong> beans</p>"})

	detail := ansi.Strip(renderTaskDetail(m, 80, 20))
	if strings.Contains(detail, "<p>") || !strings.Contains(detail, "Fresh coffee beans") {
		t.Fatalf("detail = %q, want readable Description without source HTML", detail)
	}
}

func TestHTMLDescriptionRemainsReadOnlyDuringReaderPhase(t *testing.T) {
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Coffee", Description: "<p>Fresh coffee beans</p>"})
	m.editor.field = fieldDescription

	m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
	if m.editor.mode != editorNormal || m.editor.draft.Description != "<p>Fresh coffee beans</p>" {
		t.Fatalf("HTML Description entered raw edit mode: %#v", m.editor)
	}
}

func TestReadOnlyDescriptionScrollbarUsesReservedWidth(t *testing.T) {
	m := testModel()
	m.focus = paneDetail
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Long", Description: strings.Repeat("1234567890", 9)})
	m.editor.field = fieldDescription

	view := ansi.Strip(renderTaskDetail(m, 20, 16))
	if !strings.Contains(view, "┬") || !strings.Contains(view, "█") {
		t.Fatalf("overflowing Description lacks scrollbar: %q", view)
	}
	if !strings.Contains(view, "┬") || !strings.Contains(view, "┴") {
		t.Fatalf("Description scrollbar lacks its bounds: %q", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Description:") {
			break
		}
		if strings.ContainsAny(line, "│█┬┴") {
			t.Fatalf("non-Description field has scrollbar glyph: %q", line)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if got := ansi.StringWidth(line); got != 20 {
			t.Fatalf("scrollbar line width = %d, want 20: %q", got, line)
		}
	}
}

func TestReadOnlyHTMLDescriptionScrolls(t *testing.T) {
	var description strings.Builder
	for line := 1; line <= 18; line++ {
		fmt.Fprintf(&description, "<p>Line %d</p>", line)
	}
	m := testModel()
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Long", Description: description.String()})
	m.editor.field = fieldDescription

	before := ansi.Strip(renderTaskDetail(m, 80, 16))
	if strings.Contains(before, "Line 4") {
		t.Fatalf("fixture does not overflow the Detail Pane: %q", before)
	}
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	after := ansi.Strip(renderTaskDetail(m, 80, 16))
	if m.editor.descScroll == 0 || !strings.Contains(after, "Line 2") {
		t.Fatalf("read-only Description did not scroll: offset=%d detail=%q", m.editor.descScroll, after)
	}
}
