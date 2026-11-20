package tutorial

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/tui"
)

func TestPracticeProjectCreationAndDeletionThroughRealTUIAndStore(t *testing.T) {
	demo, err := NewDemoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	defer demo.Close()
	m := tui.NewModel(tui.Config{Store: demo.Store, PerPage: 100})
	var apply func(tea.Cmd)
	apply = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				apply(c)
			}
			return
		}
		var next tea.Cmd
		m, next = m.Update(msg)
		apply(next)
	}
	apply(m.Init())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, r := range ":projectnew" {
		m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	apply(cmd)
	m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'i', Text: "i"}))
	m, _ = m.Update(tea.PasteMsg{Content: "Practice Project"})
	m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	for _, r := range ":w" {
		m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
	m, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	apply(cmd)
	projects, err := demo.Store.ListProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range projects {
		if p.Title == "Practice Project" {
			found = true
		}
	}
	if !found || strings.Contains(m.View().Content, "NEW PROJECT") {
		t.Fatalf("Project not created through real UI: %q", m.View().Content)
	}
	for _, r := range ":projectdelete" {
		m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
	m, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	apply(cmd)
	m, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	apply(cmd)
	projects, err = demo.Store.ListProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range projects {
		if p.Title == "Practice Project" {
			t.Fatal("Project not deleted through real UI and Store")
		}
	}

}
