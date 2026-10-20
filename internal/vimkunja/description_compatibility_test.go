package vimkunja

import (
	"context"
	"strings"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/description"
)

func TestNativeDescriptionSaveRetainsVikunja270WriteGuards(t *testing.T) {
	for _, tc := range []struct {
		name, mutation, want string
		create, comment      bool
		id, project          int64
	}{
		{name: "trash", mutation: `UPDATE tasks SET deleted_at=CURRENT_TIMESTAMP WHERE id=1`, want: "trash", id: 1, project: 1},
		{name: "trash with comment", mutation: `UPDATE tasks SET deleted_at=CURRENT_TIMESTAMP WHERE id=1`, want: "trash", comment: true, id: 1, project: 1},
		{name: "archived source", mutation: `UPDATE projects SET is_archived=1 WHERE id=1`, want: "archived", id: 1, project: 2},
		{name: "archived destination", mutation: `UPDATE projects SET is_archived=1 WHERE id=1`, want: "archived", comment: true, id: 3, project: 1},
		{name: "archived creation", mutation: `UPDATE projects SET is_archived=1 WHERE id=1`, want: "archived", create: true, id: 1, project: 1},
		{name: "full destination", mutation: `UPDATE buckets SET "limit"=2 WHERE id=100`, want: "limit", comment: true, id: 3, project: 1},
		{name: "full creation", mutation: `UPDATE buckets SET "limit"=2 WHERE id=100`, want: "limit", create: true, id: 1, project: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := setupVikunja270(t)
			const source = "<p>Original</p>"
			compatExec(t, s, `UPDATE tasks SET description=?`, source)
			doc := description.Parse(source)
			nodes := doc.Snapshot()
			nodes[0].Children[0].Text = "Changed"
			candidate, err := doc.PrepareSave(nodes, description.Preservation{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			compatExec(t, s, tc.mutation)
			draft := TaskDraft{ID: tc.id, ProjectID: tc.project, Title: "Changed", DescriptionCandidate: &candidate}
			ctx := context.Background()
			switch {
			case tc.create:
				draft.ID = 0
				_, err = s.CreateTask(ctx, draft)
			case tc.comment:
				_, _, err = s.SaveTaskWithComment(ctx, draft, TaskCommentDraft{Text: "Rejected comment"})
			default:
				_, err = s.UpdateTask(ctx, draft)
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("save = %v, want %s rejection", err, tc.want)
			}
			if compatInt(t, s, `SELECT COUNT(*) FROM tasks WHERE description=? AND title <> 'Changed'`, source) != 3 ||
				compatInt(t, s, `SELECT COUNT(*) FROM task_comments`) != 1 ||
				compatInt(t, s, `SELECT last_index FROM project_task_counters WHERE project_id=1`) != 2 ||
				compatInt(t, s, `SELECT COUNT(*) FROM task_index_aliases`) != 0 {
				t.Fatal("rejected native save changed Tasks, Comments, counters or aliases")
			}
		})
	}
}
