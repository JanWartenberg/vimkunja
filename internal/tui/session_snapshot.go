package tui

import "github.com/JanWartenberg/vimkunja/internal/vimkunja"

// SessionSnapshot is a detached, read-only description of the active session.
// Embedders can recognize outcomes without interpreting rendered terminal text.
type SessionSnapshot struct {
	// Mutation receipts identify successful writes across Project/View changes.
	CreatedTaskIDs                     []int64
	LastDeletedTaskID                  int64
	Ready, Normal                      bool
	Focus                              string
	ProjectID                          int64
	Task                               vimkunja.Task
	Tasks                              []vimkunja.Task
	Draft, Saved                       *vimkunja.TaskDraft
	Dirty                              bool
	Field                              string
	DescriptionActive, Native, Pending bool
	View, Filter, Sort                 string
	ProjectsHidden                     bool
	Bucket                             string
	Help, Information, Search          bool
	HelpOffset, SearchResults          int
	SearchQuery                        string
	UndoDepth, RedoDepth               int
	LastError                          string
	Comments                           []vimkunja.TaskComment
}

func (m *model) SessionSnapshot() SessionSnapshot {
	s := SessionSnapshot{
		CreatedTaskIDs: append([]int64(nil), m.createdTaskIDs...), LastDeletedTaskID: m.lastDeletedTaskID,
		Ready:     !m.projectLoading && !m.viewLoading && !m.taskLoading,
		Focus:     map[paneID]string{paneProjects: "Projects", paneTasks: "Tasks", paneDetail: "Detail", paneHelp: "Help", paneInfo: "Information"}[m.focus],
		ProjectID: m.currentProjectID,
		View:      "List", ProjectsHidden: m.projectsPaneCollapsed,
		Help: m.helpPane != nil, Information: m.infoPane != nil, Search: m.searchOpen,
		SearchQuery: m.searchInput, SearchResults: len(m.searchResults),
		UndoDepth: len(m.undoHistory), RedoDepth: len(m.redoHistory), LastError: m.lastErr,
	}
	s.Normal = !m.commandMode && m.savingEditor == nil && !m.searchOpen && m.comments == nil && m.labelUI == nil && m.inlineTitle == nil && !m.deleteConfirm && !m.kanbanPicker && !m.kanbanViewPicker && !m.kanbanProvisionConfirm && m.overviewPrompt == overviewPromptNone
	if m.inOverview() {
		s.View = "Overview"
		s.Filter = m.overviewFilterLabel()
		s.Sort = m.overviewSortLabel()
	} else {
		s.Filter = m.projectFilterLabel()
	}
	if m.kanbanActive && !m.inOverview() {
		s.View = "Kanban"
		if bucket, ok := m.activeKanbanBucket(); ok {
			s.Bucket = bucket.Title
		}
		s.Task, _ = m.selectedKanbanTask()
	} else if m.taskSelected >= 0 && m.taskSelected < len(m.tasks) {
		s.Task = m.tasks[m.taskSelected]
	}
	// Preserve knowledge of filtered-out Tasks: absence from the viewport is
	// not evidence of deletion. Visible rows may carry newer inline-edit data.
	source := m.projectTasks
	if m.inOverview() {
		source = m.overviewTasks
	}
	updates := make(map[int64]vimkunja.Task, len(m.tasks))
	for _, task := range m.tasks {
		updates[task.ID] = task
	}
	s.Tasks = make([]vimkunja.Task, 0, len(source))
	for _, task := range source {
		if update, ok := updates[task.ID]; ok {
			task = update
		}
		s.Tasks = append(s.Tasks, task)
	}
	if s.View == "Kanban" {
		s.Tasks = nil
		for _, bucket := range m.kanbanBoard.Buckets {
			s.Tasks = append(s.Tasks, bucket.Tasks...)
		}
	}
	if m.helpPane != nil {
		s.HelpOffset = m.helpPane.offset
	}
	if e := m.editor; e != nil {
		draft, saved := cloneTaskDraft(e.draft), cloneTaskDraft(e.original)
		draft.DescriptionBaseline, draft.DescriptionCandidate = nil, nil
		saved.DescriptionBaseline, saved.DescriptionCandidate = nil, nil
		s.Draft, s.Saved = &draft, &saved
		s.Dirty = e.dirty()
		s.Field = map[editorField]string{fieldTitle: "Title", fieldDescription: "Description", fieldPriority: "Priority", fieldDueDate: "Due Date", fieldWorkflowStatus: "Workflow Status", fieldDone: "Done", fieldLabels: "Labels", fieldComments: "Comments"}[e.field]
		s.DescriptionActive = e.descriptionActive
		s.Native = e.nativeDraft != nil
		s.Pending = e.nativePending != nil
		s.Comments = append([]vimkunja.TaskComment(nil), e.comments...)
		s.Ready = s.Ready && !e.workflowStatusLoading && !e.labelsLoading && !e.commentsLoading
		if m.focus == paneDetail {
			s.Normal = s.Normal && e.mode == editorNormal && e.nativeLink == nil && e.nativePending == nil && (e.nativeDraft == nil || e.nativeDraft.Selection() == nil)
		}
	}
	return s
}
