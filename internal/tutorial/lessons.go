package tutorial

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/JanWartenberg/vimkunja/internal/tui"
	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

type lesson struct {
	topic, task, context, hint string
	project, view              int64
	setup                      []string
	source                     string
	goals                      []func(tui.SessionSnapshot) bool
}

func lessonKey(value string) tea.KeyPressMsg {
	codes := map[string]rune{"enter": tea.KeyEnter, "esc": tea.KeyEscape, "tab": tea.KeyTab, "space": ' ', "home": tea.KeyHome, "end": tea.KeyEnd, "backspace": tea.KeyBackspace, "delete": tea.KeyDelete}
	if c, ok := codes[value]; ok {
		text := ""
		if value == "space" {
			text = " "
		}
		return tea.KeyPressMsg(tea.Key{Code: c, Text: text})
	}
	if strings.HasPrefix(value, "ctrl+") {
		return tea.KeyPressMsg(tea.Key{Code: []rune(strings.TrimPrefix(value, "ctrl+"))[0], Mod: tea.ModCtrl})
	}
	return tea.KeyPressMsg(tea.Key{Code: []rune(value)[0], Text: value})
}

func lessonCommand(value string) []string {
	result := []string{":"}
	for _, r := range value {
		result = append(result, string(r))
	}
	return append(result, "enter")
}

// Seed only disposable fixtures, before constructing the application. The
// production Store and its schema remain unchanged.
func newLessonDatabase(markup bool, lesson lesson) (*DemoDatabase, error) {
	db, err := newDemoDatabase(markup)
	if err != nil {
		return nil, err
	}
	seed, err := sql.Open("sqlite", db.Path)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	id := int64(101)
	if markup {
		id = 108
	}
	if lesson.source != "" {
		_, err = seed.Exec(`UPDATE tasks SET description=? WHERE id=?`, lesson.source, id)
	}
	if err == nil {
		_, err = seed.Exec(`INSERT INTO labels (id,title,description,hex_color,created_by_id,created,updated) VALUES (1,'Tutorial','','2d9cdb',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	}
	closeErr := seed.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("seed task fixture: %w", err)
	}
	return db, nil
}

func foundTask(s tui.SessionSnapshot, id int64) (vimkunja.Task, bool) {
	for _, task := range s.Tasks {
		if task.ID == id {
			return task, true
		}
	}
	return vimkunja.Task{}, false
}
func draftIs(s tui.SessionSnapshot, id int64) bool { return s.Draft != nil && s.Draft.ID == id }
func sourceIs(value string) func(tui.SessionSnapshot) bool {
	return func(s tui.SessionSnapshot) bool { return draftIs(s, 108) && s.Draft.Description == value }
}

func discardedDescription(s tui.SessionSnapshot, id int64, source string) bool {
	task, ok := foundTask(s, id)
	return ok && task.Description == source && (s.Draft == nil || s.Draft.ID == id && s.Draft.Description == source && !s.Dirty)
}

func basicLessons() []lesson {
	detail := []string{"enter"}
	desc := []string{"enter", "g", "n", "enter"}
	lessons := []lesson{
		{topic: "Text input", task: "Rename Task ‘Practice the Vimkunja tutorial’ to ‘foo’.", context: "The first Task belongs to Project ‘Learn Vimkunja’.", hint: "From reset: Enter, Enter, 0, C; type foo, then Esc. Saving is optional.", goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			if s.Focus == "Detail" {
				return draftIs(s, 101) && s.Draft.Title == "foo"
			}
			return s.Task.ID == 101 && s.Task.Title == "foo"
		}}},
		{topic: "Projects", task: "Open the Tasks Pane of Project ‘Home’.", context: "Project selection determines which Tasks are shown.", hint: "From reset: j selects Home; Enter focuses its Tasks Pane.", goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return s.ProjectID == 2 && s.Focus == "Tasks" }}},
		{topic: "Task navigation", task: "Select Task ‘Inspect task metadata’ in ‘Learn Vimkunja’.", context: "Vim motions and arrow keys are both available.", hint: "G or End selects the last Task; gg or Home returns to the first.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return s.Focus == "Tasks" && s.Task.ID == 108 }}},
		{topic: "Task Detail", task: "Open Task ‘Practice the Vimkunja tutorial’ in Task Detail.", context: "Task Detail is a normal Pane, not a modal.", hint: "Enter opens the selected Task; Ctrl+W changes Pane focus.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return draftIs(s, 101) && s.Focus == "Detail" }}},
		{topic: "Priority", task: "Set Task ‘Practice the Vimkunja tutorial’ to Urgent priority.", context: "Picker changes remain a draft until saved.", hint: "gp opens Priority; j/k choose Urgent, Enter confirms. Alternatively navigate the fields.", project: 1, view: 11, setup: detail, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return draftIs(s, 101) && s.Draft.Priority == 4 }}},
		{topic: "Due date", task: "Set Task ‘Practice the Vimkunja tutorial’ due on 2030-01-15.", context: "Dates are entered with a YYYY-MM-DD mask.", hint: "gd opens Due Date; type 20300115, then Enter.", project: 1, view: 11, setup: detail, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return draftIs(s, 101) && s.Draft.DueDate == "2030-01-15" }}},
		{topic: "Workflow", task: "Set Task ‘Practice the Vimkunja tutorial’ to Doing.", context: "Workflow Status uses this Project’s Kanban Buckets.", hint: "Expand More fields, navigate to Workflow Status, i opens its Picker; choose Doing and Enter.", project: 1, view: 11, setup: detail, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return draftIs(s, 101) && s.Draft.WorkflowBucketID == 122 }}},
		{topic: "Done", task: "Mark Task ‘Practice the Vimkunja tutorial’ as done.", context: "Task Done is separate from Description Checkboxes.", hint: "Close Detail and press x on the Task, or toggle the Done field in Detail.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			if draftIs(s, 101) && s.Draft.Done {
				return true
			}
			t, ok := foundTask(s, 101)
			return ok && t.Done
		}}},
		{topic: "Description editing", task: "Prefix Task ‘Practice the Vimkunja tutorial’ Description with ‘Tutorial: ’.", context: "Description supports Vim motions, word operators and multiline Insert.", hint: "I enters Insert at line start; type ‘Tutorial: ’, then Esc. Try ciw, w/b/e and counts freely.", project: 1, view: 11, setup: desc, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			return draftIs(s, 101) && strings.HasPrefix(s.Draft.Description, "Tutorial: Move through")
		}}},
		{topic: "Save", task: "Save Task ‘Practice the Vimkunja tutorial’ with Description ‘Saved in tutorial.’.", context: "A successful save clears the dirty marker; failed saves keep the draft.", hint: "In Description use 0, C; enter ‘Saved in tutorial.’, Esc, then :w. You may use :write or :wq too.", project: 1, view: 11, setup: desc, source: "Replace me", goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			if s.Saved != nil && s.Saved.ID == 101 && s.Saved.Description == "Saved in tutorial." {
				return true
			}
			t, ok := foundTask(s, 101)
			return ok && t.Description == "Saved in tutorial."
		}}},
		{topic: "Pane focus", task: "Focus the Tasks Pane while keeping Task Detail open.", context: "Ctrl+W moves between Panes without closing them.", hint: "Ctrl+W k focuses the Tasks Pane above Task Detail; Ctrl+W j returns.", project: 1, view: 11, setup: detail, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return s.Focus == "Tasks" && draftIs(s, 101) }}},
		{topic: "Close Detail", task: "Close Task Detail for ‘Practice the Vimkunja tutorial’.", context: "Closing a clean Task returns to the Tasks Pane.", hint: "Use :q or :quit; Ex commands support Tab completion.", project: 1, view: 11, setup: detail, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return s.Draft == nil && s.Focus == "Tasks" }}},
		{topic: "Discard", task: "Discard the unsaved Title change on Task ‘Practice the Vimkunja tutorial’.", context: "This task starts with an unsaved Title ‘Unsaved’.", hint: "Try :q to see dirty protection, then use :e! to restore or :q! to close and discard.", project: 1, view: 11, setup: []string{"enter", "0", "C", "Unsaved", "esc"}, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			t, ok := foundTask(s, 101)
			return ok && t.Title == "Practice the Vimkunja tutorial" && (s.Draft == nil || draftIs(s, 101) && s.Draft.Title == t.Title && !s.Dirty)
		}}},
		{topic: "Create Task", task: "Create and save Task ‘A tutorial task’ in ‘Learn Vimkunja’.", context: "New Tasks need a Title and a Project.", hint: "o or :new opens a draft; i, type the Title, Esc, then :wq saves it. From Overview, choose the Project first.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			for _, t := range s.Tasks {
				if t.Title == "A tutorial task" && t.ProjectID == 1 && slices.Contains(s.CreatedTaskIDs, t.ID) {
					return true
				}
			}
			return false
		}}},
		{topic: "Delete Task", task: "Delete Task ‘Practice the Vimkunja tutorial’.", context: "Deletion is permanent and requires confirmation; these are disposable Tasks.", hint: "dd opens confirmation; n or Esc cancels, y deletes. You can cancel and retry freely.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			return s.LastDeletedTaskID == 101
		}}},
		{topic: "Undo and redo", task: "Complete Task ‘Practice the Vimkunja tutorial’, undo, then redo.", context: "Task history covers persisted edits, not unsaved drafts or deletion.", hint: "x completes the Task, u undoes it, Ctrl+R redoes it. Observe Done and the Bucket together.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{
			func(s tui.SessionSnapshot) bool { t, ok := foundTask(s, 101); return ok && t.Done && s.UndoDepth > 0 },
			func(s tui.SessionSnapshot) bool { t, ok := foundTask(s, 101); return ok && !t.Done && s.RedoDepth > 0 },
			func(s tui.SessionSnapshot) bool { t, ok := foundTask(s, 101); return ok && t.Done && s.RedoDepth == 0 },
		}},
		{topic: "Overview", task: "Open the All-Tasks Overview.", context: "Overview spans Projects and initially shows open Tasks.", hint: "Use - or :overview; Space a toggles the remembered Project context.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return s.View == "Overview" }}},
		{topic: "Overview sort/filter", task: "Show all Tasks in Overview, sorted by Priority.", context: "Sort and status filters are independent.", hint: "Use s p for Priority sorting and f a for all Tasks; experiment with the other choices too.", project: 1, view: 11, setup: []string{"-"}, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			return s.View == "Overview" && s.Sort == "Priority ↓" && s.Filter == "all"
		}}},
		{topic: "Projects Pane", task: "Hide the Projects Pane.", context: "The Tasks Pane reclaims its width.", hint: "Space o toggles the Projects Pane; press it again to restore it.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return s.ProjectsHidden }}},
		{topic: "Kanban", task: "Move Task ‘Practice the Vimkunja tutorial’ to Doing on Kanban.", context: "Card moves synchronize Workflow Status and Task Done.", hint: "Open :kanban or Space v k; Shift+L moves right, or m opens the Bucket Picker. j/k navigate cards; h/l navigate Buckets.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return s.View == "Kanban" && s.Task.ID == 101 && s.Bucket == "Doing" }}},
		{topic: "Provision Kanban", task: "Create a Kanban Board for Project ‘Plan a Trip’.", context: "This Project starts without a manual Kanban View.", hint: "Use :kanban and confirm with y; n cancels without creating anything.", project: 3, view: 31, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return s.ProjectID == 3 && s.View == "Kanban" }}},
		{topic: "Help", task: "Open Help and scroll down.", context: "Help is a normal Pane and participates in Ctrl+W navigation.", hint: "Use :h or :help, then j, Ctrl+D or G to scroll; :q closes Help.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return s.Help && s.Focus == "Help" && s.HelpOffset > 0 }}},
		{topic: "Errors", task: "Open the error log after an unknown command.", context: "The footer shows errors; the session log retains them.", hint: "Try :bogus, then :errors; q closes the information Pane.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return s.LastError != "" }, func(s tui.SessionSnapshot) bool { return s.Information }}},
		{topic: "Task Labels", task: "Assign Label ‘Tutorial’ to Task ‘Practice the Vimkunja tutorial’.", context: "This lesson includes one existing Label.", hint: "gl opens the Task Label Picker; Enter toggles Tutorial; Esc leaves filter input, q closes the Picker. Save with :w.", project: 1, view: 11, setup: detail, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			return draftIs(s, 101) && len(s.Draft.LabelIDs) == 1 && s.Draft.LabelIDs[0] == 1
		}}},
		{topic: "Task Comments", task: "Save Comment ‘Learning works’ on Task ‘Practice the Vimkunja tutorial’.", context: "Comments have their own composer and persist with the Task.", hint: "gc opens Comments; a starts a composer, type ‘Learning works’, Esc, :w saves, :q returns.", project: 1, view: 11, setup: detail, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			for _, c := range s.Comments {
				if c.TaskID == 101 && strings.Contains(c.HTML, "Learning works") {
					return true
				}
			}
			return false
		}}},
		{topic: "Global Search", task: "Find and open Task ‘Buy coffee’ using Global Search.", context: "Global Search can find Tasks across Projects.", hint: "Space f opens Search; type Buy coffee, then Enter opens the selected result.", project: 1, view: 11, goals: []func(tui.SessionSnapshot) bool{
			func(s tui.SessionSnapshot) bool {
				return s.Search && s.SearchResults > 0
			},
			func(s tui.SessionSnapshot) bool { return draftIs(s, 201) && s.Focus == "Detail" },
		}},
	}
	return lessons
}
