package tui

import (
	"testing"
	"time"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

func TestTaskMatchesFilter(t *testing.T) {
	today := time.Date(2026, 9, 20, 0, 0, 0, 0, time.Local)
	cases := []struct {
		name   string
		task   vimkunja.Task
		filter overviewFilter
		want   bool
	}{
		{name: "open includes overdue", task: vimkunja.Task{DueDate: "2026-09-19"}, filter: overviewOpen, want: true},
		{name: "all includes done", task: vimkunja.Task{Done: true}, filter: overviewAll, want: true},
		{name: "overdue excludes done", task: vimkunja.Task{Done: true, DueDate: "2026-09-19"}, filter: overviewOverdue, want: false},
		{name: "overdue uses local day", task: vimkunja.Task{DueDate: "2026-09-19"}, filter: overviewOverdue, want: true},
		{name: "in progress excludes default bucket", task: vimkunja.Task{Workflow: vimkunja.TaskWorkflow{Available: true, CurrentBucketID: 10, DefaultBucketID: 10, DoneBucketID: 12}}, filter: overviewInProgress, want: false},
		{name: "in progress excludes done bucket", task: vimkunja.Task{Workflow: vimkunja.TaskWorkflow{Available: true, CurrentBucketID: 12, DefaultBucketID: 10, DoneBucketID: 12}}, filter: overviewInProgress, want: false},
		{name: "in progress requires unambiguous status", task: vimkunja.Task{Workflow: vimkunja.TaskWorkflow{CurrentBucketID: 11, DefaultBucketID: 10, DoneBucketID: 12}}, filter: overviewInProgress, want: false},
		{name: "in progress matches another bucket", task: vimkunja.Task{Workflow: vimkunja.TaskWorkflow{Available: true, CurrentBucketID: 11, DefaultBucketID: 10, DoneBucketID: 12}}, filter: overviewInProgress, want: true},
		{name: "in progress excludes done task", task: vimkunja.Task{Done: true, Workflow: vimkunja.TaskWorkflow{Available: true, CurrentBucketID: 11, DefaultBucketID: 10, DoneBucketID: 12}}, filter: overviewInProgress, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := taskMatchesFilter(tc.task, tc.filter, today); got != tc.want {
				t.Fatalf("taskMatchesFilter() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestProjectTaskFilterPromptAppliesFilter(t *testing.T) {
	m := &model{
		taskScope:      scopeProject,
		taskFilter:     overviewAll,
		overviewPrompt: overviewPromptFilter,
		projectTasks: []vimkunja.Task{
			{ID: 1, Workflow: vimkunja.TaskWorkflow{Available: true, CurrentBucketID: 11, DefaultBucketID: 10, DoneBucketID: 12}},
			{ID: 2, Workflow: vimkunja.TaskWorkflow{Available: true, CurrentBucketID: 10, DefaultBucketID: 10, DoneBucketID: 12}},
		},
	}

	if !m.updateTaskFilterPrompt("i") {
		t.Fatal("updateTaskFilterPrompt() = false, want true")
	}
	if m.taskFilter != overviewInProgress {
		t.Fatalf("taskFilter = %v, want in-progress", m.taskFilter)
	}
	if len(m.tasks) != 1 || m.tasks[0].ID != 1 {
		t.Fatalf("filtered tasks = %#v, want only task 1", m.tasks)
	}
}
