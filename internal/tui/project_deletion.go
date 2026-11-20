package tui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

type projectDeletionUI struct {
	project         vimkunja.Project
	loading, saving bool
	err             string
}

type projectDeletionCheckedMsg struct {
	ui  *projectDeletionUI
	err error
}

type projectDeletedMsg struct {
	ui         *projectDeletionUI
	projects   []vimkunja.Project
	err        error
	refreshErr error
}

func projectDeleteCommand(m *model, args []string) (tea.Model, tea.Cmd) {
	if len(args) != 0 {
		m.recordError(fmt.Errorf(":projectdelete takes no arguments; select the Project first"))
		return m, nil
	}
	if m.savingEditor != nil || (m.editor != nil && m.editor.dirty()) || m.helpHasDraft() || (m.inlineTitle != nil && (m.inlineTitle.saving || m.inlineTitle.title != m.inlineTitle.task.Title)) {
		m.recordError(fmt.Errorf("save or discard the unsaved Task before deleting a Project"))
		return m, nil
	}
	p, ok := m.selectedProject()
	if !ok || p.ID <= 0 {
		m.recordError(fmt.Errorf("select a Project before using :projectdelete"))
		return m, nil
	}
	m.projectDeletion = &projectDeletionUI{project: p, loading: true}
	m.leaderArmed, m.ctrlWArmed, m.gArmed, m.zArmed, m.viewSwitchArmed = false, false, false, false, false
	return m, m.checkProjectDeletion(m.projectDeletion)
}

func (m *model) checkProjectDeletion(ui *projectDeletionUI) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		return projectDeletionCheckedMsg{ui: ui, err: store.CheckProjectDeletion(ctx, ui.project.ID)}
	}
}

func (m *model) updateProjectDeletion(key string) (tea.Model, tea.Cmd) {
	ui := m.projectDeletion
	if ui.saving {
		return m, nil
	}
	switch key {
	case "n", "esc":
		m.projectDeletion = nil
	case "r":
		if ui.loading {
			return m, nil
		}
		ui.loading, ui.err = true, ""
		return m, m.checkProjectDeletion(ui)
	case "y":
		if ui.loading || ui.err != "" {
			return m, nil
		}
		ui.saving, ui.err = true, ""
		store := m.store
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			msg := projectDeletedMsg{ui: ui, err: store.DeleteEmptyProject(ctx, ui.project.ID)}
			if msg.err == nil {
				msg.projects, msg.refreshErr = store.ListProjects(ctx)
			}
			return msg
		}
	}
	return m, nil
}

func (m *model) projectDeletionResult(msg projectDeletedMsg) tea.Cmd {
	if m.projectDeletion != msg.ui {
		return nil
	}
	if msg.err != nil {
		msg.ui.saving, msg.ui.err = false, msg.err.Error()
		return nil
	}
	m.projectDeletion = nil
	if msg.refreshErr != nil {
		// Deletion has committed. Remove the cached entry even if reload fails;
		// never offer to retry the already completed destructive operation.
		for _, p := range m.projects {
			if p.ID != msg.ui.project.ID {
				msg.projects = append(msg.projects, p)
			}
		}
	}
	m.projectRequest++
	m.taskRequest++
	m.viewRequest++
	m.searchRequest++
	m.commentRequest++
	m.taskScope = scopeProject
	m.projectViewPreference = projectViewList
	m.currentViewID, m.initialViewID = 0, 0
	m.kanbanActive = false
	m.kanbanBoard = vimkunja.KanbanBoard{}
	m.kanbanPicker, m.kanbanViewPicker, m.kanbanProvisionConfirm = false, false, false
	m.searchOpen = false
	m.searchOrigin, m.searchReturn = nil, nil
	m.closeHelp()
	m.infoPane, m.editor, m.inlineTitle, m.comments, m.labelUI = nil, nil, nil, nil, nil
	m.showDetail = false
	m.tasks, m.projectTasks, m.projectViews = nil, nil, nil
	m.taskSelected = 0
	m.taskLoading = false
	m.focus = paneProjects
	m.projectsPaneCollapsed, m.projectsPaneAutoShown = false, false
	_, cmd := m.Update(projectsLoadedMsg{projects: msg.projects, request: m.projectRequest})
	if msg.refreshErr != nil {
		m.recordError(fmt.Errorf("Project deleted, but refreshing failed: %w; reload with r", msg.refreshErr))
	}
	return cmd
}

func (m *model) renderProjectDeletion() string {
	ui := m.projectDeletion
	w, h := m.viewportSize()
	layout := modalBounds(w, h, 72, 12)
	question := fmt.Sprintf("Delete Project #%d %q permanently?", ui.project.ID, sanitizeTerminalText(ui.project.Title))
	status := "This has no Tasks or Subprojects."
	hint := "y delete · n / Esc cancel"
	if ui.loading {
		status, hint = "Checking whether this Project can be deleted…", "n / Esc cancel"
	} else if ui.err != "" {
		status, hint = sanitizeTerminalText(ui.err)+" (Cannot be deleted.)", "r recheck · n / Esc close"
	} else if ui.saving {
		status, hint = "Deleting Project…", "Please wait…"
	}
	statusRows := wrapModalWords(status, layout.contentWidth)
	// Keep the checked status visible even when a long Title consumes the form.
	budget := max(0, layout.contentHeight-4)
	statusCount := min(len(statusRows), max(1, budget))
	questionRows := wrapModalWords(question, layout.contentWidth)
	questionCount := min(len(questionRows), max(0, budget-statusCount-1))
	rows := []string{"DELETE PROJECT", ""}
	rows = append(rows, questionRows[:questionCount]...)
	if questionCount > 0 {
		rows = append(rows, "")
	}
	rows = append(rows, statusRows[:statusCount]...)
	rows = append(rows, "", truncatePlain(hint, layout.contentWidth))
	return overlayModalLayout(fitBlock(nil, w, h), renderModalFrame(rows, layout), w, layout.fallback)
}
