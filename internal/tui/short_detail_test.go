package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"github.com/charmbracelet/x/ansi"
)

func TestShortTaskDetailKeepsDescriptionVisible(t *testing.T) {
	for _, height := range []int{28, 30, 32, 34, 36, 38, 40} {
		for _, html := range []bool{false, true} {
			t.Run(fmt.Sprintf("height=%d/html=%v", height, html), func(t *testing.T) {
				description := "FIRSTDESCRIPTIONLINE\nSECONDDESCRIPTIONLINE\nTHIRDDESCRIPTIONLINE"
				if html {
					description = "<p>FIRSTDESCRIPTIONLINE</p><p>SECONDDESCRIPTIONLINE</p><p>THIRDDESCRIPTIONLINE</p>"
				}
				m := testModel()
				m.width, m.height = 100, height
				m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: description})
				for steps := 0; m.editor.field != fieldDescription && steps < 12; steps++ {
					m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
				}
				if m.editor.field != fieldDescription {
					t.Fatal("j navigation did not reach Description")
				}
				for _, activate := range []bool{false, true} {
					if activate {
						m = pressKey(m, tea.Key{Code: tea.KeyEnter})
					}
					view := ansi.Strip(m.Render())
					for _, line := range []string{"FIRSTDESCRIPTIONLINE", "SECONDDESCRIPTIONLINE", "THIRDDESCRIPTIONLINE"} {
						if !strings.Contains(view, line) {
							t.Fatalf("activated=%v: missing %s in short Task Detail", activate, line)
						}
					}
					if activate && regexp.MustCompile(`\b\d+%`).MatchString(view) {
						t.Fatal("fully visible Description must not report scrolling")
					}
				}
			})
		}
	}
}

func TestShortTaskDetailScrollPercentageTracksVisibleDescription(t *testing.T) {
	percent := regexp.MustCompile(`\b\d+%`)
	visibleLines := regexp.MustCompile(`DESCRIPTIONLINE\d+`)
	for _, height := range []int{28, 30, 32, 34, 36, 38, 40} {
		t.Run(fmt.Sprint(height), func(t *testing.T) {
			lines := make([]string, 24)
			for i := range lines {
				lines[i] = fmt.Sprintf("DESCRIPTIONLINE%02d", i)
			}
			m := testModel()
			m.width, m.height = 100, height
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: strings.Join(lines, "\n")})
			for steps := 0; m.editor.field != fieldDescription && steps < 12; steps++ {
				m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
			}
			m = pressKey(m, tea.Key{Code: tea.KeyEnter})
			before := ansi.Strip(m.Render())
			if !strings.Contains(before, lines[0]) || !strings.Contains(before, "0%") {
				t.Fatalf("initial Description viewport must show first line and 0%%:\n%s", before)
			}
			for step := 0; step < len(lines); step++ {
				m = pressKey(m, tea.Key{Code: 'e', Mod: tea.ModCtrl})
				after := ansi.Strip(m.Render())
				if percent.FindString(before) != percent.FindString(after) {
					// Ignore the footer and cursor: the Description text must move.
					beforeBody := strings.Join(visibleLines.FindAllString(before, -1), "\n")
					afterBody := strings.Join(visibleLines.FindAllString(after, -1), "\n")
					if beforeBody == afterBody {
						t.Fatalf("scroll percentage changed from %s to %s without visible movement", percent.FindString(before), percent.FindString(after))
					}
				}
				before = after
			}
			if !strings.Contains(before, lines[len(lines)-1]) || !strings.Contains(before, "100%") {
				t.Fatalf("scrolled Description viewport must show last line and 100%%:\n%s", before)
			}
		})
	}
}

func TestShortTaskDetailExpandedMetadataKeepsHTMLDescriptionVisible(t *testing.T) {
	m := testModel()
	m.width, m.height = 100, 30
	m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: "<p>FIRSTLINE</p><p>SECONDLINE</p><p>THIRDLINE</p><p>LASTLINE</p>"})
	m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	m = pressKey(m, tea.Key{Code: tea.KeyEnter}) // Expand More fields.
	for steps := 0; m.editor.field != fieldDescription && steps < 12; steps++ {
		m = pressKey(m, tea.Key{Code: 'j', Text: "j"})
	}
	view := ansi.Strip(m.Render())
	for _, line := range []string{"FIRSTLINE", "SECONDLINE", "THIRDLINE"} {
		if !strings.Contains(view, line) {
			t.Fatalf("expanded metadata hides %s after Description navigation", line)
		}
	}
	m.pageDetail(-1) // Outer viewport starts on metadata, before native navigation resumes.
	if strings.Contains(ansi.Strip(m.Render()), "FIRSTLINE") {
		t.Fatal("outer-page fixture must hide Description before resuming inner scrolling")
	}
	m = pressKey(m, tea.Key{Code: 'e', Mod: tea.ModCtrl})
	view = ansi.Strip(m.Render())
	if !strings.Contains(view, "LASTLINE") || !strings.Contains(view, "100%") || !strings.Contains(view, "Comments") {
		t.Fatalf("expanded Description must show last line, 100%% and pinned Comments:\n%s", view)
	}
}

func TestHTMLDescriptionEndScrollReachesLastVisibleLine(t *testing.T) {
	for _, height := range []int{30, 34, 40, 44} {
		t.Run(fmt.Sprint(height), func(t *testing.T) {
			m := testModel()
			m.width, m.height = 100, height
			m.openExistingEditor(vimkunja.Task{ID: 1, ProjectID: 1, Title: "Task", Description: strings.Repeat("<p>Body</p>", 24) + "<p>LASTLINE</p>"})
			// Use the actual native Description navigation flow.
			nativeKeys(m, "g", "n", "enter")
			m = pressKey(m, tea.Key{Code: 'G', Text: "G"})
			view := ansi.Strip(m.Render())
			if !strings.Contains(view, "LASTLINE") || !strings.Contains(view, "100%") {
				t.Fatalf("HTML end-scroll must show last line and 100%% at height %d", height)
			}
			m = pressKey(m, tea.Key{Code: tea.KeyHome})
			view = ansi.Strip(m.Render())
			if !strings.Contains(view, "Body") || strings.Contains(view, "LASTLINE") || !strings.Contains(view, "0%") {
				t.Fatal("HTML Home must restore first viewport and 0%")
			}
		})
	}
}
