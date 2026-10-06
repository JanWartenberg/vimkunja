package tui

import (
	"reflect"
	"testing"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

func TestProjectTreeOrdersAndIndentsProjects(t *testing.T) {
	projects := []vimkunja.Project{
		{ID: 5, Title: "Second root", Position: 20},
		{ID: 4, Title: "Second child", ParentProjectID: 1, Position: 20},
		{ID: 3, Title: "First child by ID", ParentProjectID: 1, Position: 10},
		{ID: 2, Title: "Second child by ID", ParentProjectID: 1, Position: 10},
		{ID: 1, Title: "First root", Position: 10},
		{ID: 6, Title: "Grandchild", ParentProjectID: 2, Position: 1},
	}

	tree := newProjectTree(projects, nil)
	rows := tree.visibleRows()
	gotIDs := make([]int64, len(rows))
	gotDepths := make([]int, len(rows))
	for i, row := range rows {
		gotIDs[i] = row.Project.ID
		gotDepths[i] = row.Depth
	}
	if want := []int64{1, 2, 6, 3, 4, 5}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("visible Project IDs = %v, want %v", gotIDs, want)
	}
	if want := []int{0, 1, 2, 1, 1, 0}; !reflect.DeepEqual(gotDepths, want) {
		t.Fatalf("visible Project depths = %v, want %v", gotDepths, want)
	}
	if !rows[0].HasChildren || !rows[0].Expanded || rows[5].HasChildren {
		t.Fatalf("branch metadata = first:%#v last:%#v", rows[0], rows[5])
	}
}

func TestProjectTreeReorderedParentCarriesSubtreeAndDescendantFoldState(t *testing.T) {
	projects := []vimkunja.Project{
		{ID: 1, Title: "Parent", Position: 1},
		{ID: 2, Title: "Child branch", ParentProjectID: 1, Position: 1},
		{ID: 4, Title: "Grandchild", ParentProjectID: 2, Position: 1},
		{ID: 3, Title: "Other root", Position: 2},
	}
	tree := newProjectTree(projects, nil)
	tree.collapse(2)
	projects[0].Position, projects[3].Position = projects[3].Position, projects[0].Position

	refreshed := newProjectTree(projects, tree.collapsedState())
	if got, want := visibleProjectIDs(refreshed), []int64{3, 1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("reordered subtree with folded descendant = %v, want %v", got, want)
	}
	refreshed.expand(2)
	if got, want := visibleProjectIDs(refreshed), []int64{3, 1, 2, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expanded reordered descendant = %v, want %v", got, want)
	}
}

func TestProjectTreeFoldsBranchesAndRetainsFoldStateAcrossRefresh(t *testing.T) {
	projects := []vimkunja.Project{
		{ID: 1, Title: "Root"},
		{ID: 2, Title: "Child", ParentProjectID: 1},
		{ID: 3, Title: "Grandchild", ParentProjectID: 2},
	}
	tree := newProjectTree(projects, nil)

	tree.collapse(1)
	if got, want := visibleProjectIDs(tree), []int64{1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("collapsed rows = %v, want %v", got, want)
	}

	refreshed := newProjectTree(append(projects,
		vimkunja.Project{ID: 4, Title: "New branch"},
		vimkunja.Project{ID: 5, Title: "New child", ParentProjectID: 4},
	), tree.collapsedState())
	rows := refreshed.visibleRows()
	if got, want := visibleProjectIDs(refreshed), []int64{1, 4, 5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("refreshed rows = %v, want old branch closed and new branch open as %v", got, want)
	}
	if rows[0].Expanded || !rows[1].Expanded {
		t.Fatalf("refreshed expansion = old:%v new:%v", rows[0].Expanded, rows[1].Expanded)
	}

	refreshed.reveal(3)
	if got, want := visibleProjectIDs(refreshed), []int64{1, 2, 3, 4, 5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("revealed rows = %v, want %v", got, want)
	}
	if parentID, ok := refreshed.parentID(3); !ok || parentID != 2 {
		t.Fatalf("parent of 3 = %d, %v; want 2, true", parentID, ok)
	}
	if childID, ok := refreshed.firstChildID(1); !ok || childID != 2 {
		t.Fatalf("first child of 1 = %d, %v; want 2, true", childID, ok)
	}
}

func visibleProjectIDs(tree projectTree) []int64 {
	rows := tree.visibleRows()
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.Project.ID
	}
	return ids
}

func TestProjectTreeRendersMalformedRelationshipsOnceAsRecoveryRoots(t *testing.T) {
	projects := []vimkunja.Project{
		{ID: 1, Title: "Valid root", Position: 1},
		{ID: 2, Title: "Valid child", ParentProjectID: 1, Position: 1},
		{ID: 3, Title: "Orphan", ParentProjectID: 99, Position: 1},
		{ID: 4, Title: "Orphan descendant", ParentProjectID: 3, Position: 1},
		{ID: 5, Title: "Negative", ParentProjectID: -1, Position: 1},
		{ID: 6, Title: "Self", ParentProjectID: 6, Position: 1},
		{ID: 7, Title: "Cycle A", ParentProjectID: 8, Position: 1},
		{ID: 8, Title: "Cycle B", ParentProjectID: 7, Position: 1},
		{ID: 9, Title: "Cycle descendant", ParentProjectID: 7, Position: 1},
	}

	rows := newProjectTree(projects, nil).visibleRows()
	gotIDs := make([]int64, len(rows))
	for i, row := range rows {
		gotIDs[i] = row.Project.ID
		if i >= 2 && (!row.Recovery || row.Depth != 0 || row.HasChildren) {
			t.Fatalf("malformed Project row = %#v, want leaf recovery root", row)
		}
	}
	if want := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("visible Project IDs = %v, want every Project exactly once as %v", gotIDs, want)
	}
}
