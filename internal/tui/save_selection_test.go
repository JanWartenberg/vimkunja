package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
	"testing"
)

func TestDescriptionSaveKeepsSelectedTask(t *testing.T) {
	for _, scope := range []string{"list", "overview", "overview after kanban", "overview with retained board"} {
		for _, command := range []string{"w", "wq"} {
			t.Run(scope+"/"+command, func(t *testing.T) {
				edited := vimkunja.Task{ID: 2, ProjectID: 1, Title: "XMG Pro: Dev-Umgebung", Description: "Before", Priority: 4}
				other := vimkunja.Task{ID: 1, ProjectID: 1, Title: "Buch: Philosophy of SW Design"}
				list := vimkunja.ProjectView{ID: 10, ViewKind: "list"}
				boardView := vimkunja.ProjectView{ID: 11, ViewKind: "kanban", BucketConfigurationMode: "manual"}
				store := &fakeStore{
					projects:      []vimkunja.Project{{ID: 1, Title: "Sonstige"}},
					views:         []vimkunja.ProjectView{list, boardView},
					tasksByView:   map[int64][]vimkunja.Task{10: {other, edited}},
					overviewTasks: []vimkunja.Task{other, edited},
					kanbanBoard:   vimkunja.KanbanBoard{Availability: vimkunja.KanbanBoardAvailable, View: boardView, Buckets: []vimkunja.KanbanBucket{{ID: 100, Tasks: []vimkunja.Task{other, edited}}}},
				}
				m := NewModel(Config{Store: store}).(*model)
				m.currentProjectID, m.currentViewID = 1, 10
				m.tasks = []vimkunja.Task{edited, other}
				if scope != "list" {
					m.taskScope = scopeOverview
					m.overviewTasks = m.tasks
				}
				if scope == "overview after kanban" || scope == "overview with retained board" {
					m.projectViewPreference = projectViewKanban
				}
				if scope == "overview with retained board" {
					m.setKanbanBoard(store.kanbanBoard)
					cmd := m.openOverview()
					m.Update(cmd())
					for i, task := range m.tasks {
						if task.ID == edited.ID {
							m.taskSelected = i
						}
					}
				}
				m.openExistingEditor(edited)
				m = pressKey(m, tea.Key{Code: 'g', Text: "g"})
				m = pressKey(m, tea.Key{Code: 'n', Text: "n"})
				m = pressKey(m, tea.Key{Code: 'i', Text: "i"})
				m = pressKey(m, tea.Key{Text: "Changed "})
				m = pressKey(m, tea.Key{Code: tea.KeyEscape})
				pending := []tea.Cmd{enterModalCommand(m, command)}
				for steps := 0; len(pending) > 0; steps++ {
					if steps > 30 {
						t.Fatal("reload did not settle")
					}
					cmd := pending[0]
					pending = pending[1:]
					if cmd == nil {
						continue
					}
					msg := cmd()
					if batch, ok := msg.(tea.BatchMsg); ok {
						pending = append(pending, batch...)
						continue
					}
					_, next := m.Update(msg)
					pending = append(pending, next)
				}
				if store.updatedDraft.Description != "Changed Before" {
					t.Fatalf("Description not saved: %q", store.updatedDraft.Description)
				}
				if len(m.tasks) == 0 || m.tasks[m.taskSelected].ID != edited.ID {
					t.Fatalf("selected another Task after save: index=%d tasks=%#v", m.taskSelected, m.tasks)
				}
				if command == "w" && (m.editor == nil || m.editor.id != edited.ID) {
					t.Fatalf("Detail switched to another Task: %#v", m.editor)
				}
				if command == "w" && m.editor.draft.Description != "Changed Before" {
					t.Fatalf("reload replaced the saved Description: %q", m.editor.draft.Description)
				}
				if scope != "list" && !m.inOverview() {
					t.Fatal("save left Overview")
				}
			})
		}
	}
}
