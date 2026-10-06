package tui

import (
	"sort"

	"github.com/JanWartenberg/vimkunja/internal/vimkunja"
)

type projectTreeRow struct {
	Project     vimkunja.Project
	Depth       int
	HasChildren bool
	Expanded    bool
	Recovery    bool
}

type projectTree struct {
	projects  map[int64]vimkunja.Project
	valid     map[int64]bool
	roots     []int64
	children  map[int64][]int64
	collapsed map[int64]bool
	recovery  []int64
}

func newProjectTree(projects []vimkunja.Project, previousCollapsed map[int64]bool) projectTree {
	tree := projectTree{
		projects:  make(map[int64]vimkunja.Project, len(projects)),
		valid:     make(map[int64]bool, len(projects)),
		children:  make(map[int64][]int64),
		collapsed: make(map[int64]bool),
	}
	for _, project := range projects {
		tree.projects[project.ID] = project
	}

	const (
		_ = iota // zero is the unvisited state
		projectVisiting
		projectValid
		projectInvalid
	)
	states := make(map[int64]int, len(projects))
	var hasValidPathToRoot func(int64) bool
	hasValidPathToRoot = func(id int64) bool {
		switch states[id] {
		case projectValid:
			return true
		case projectInvalid, projectVisiting:
			return false
		}
		states[id] = projectVisiting
		project := tree.projects[id]
		parentID := project.ParentProjectID
		valid := parentID == 0
		if parentID > 0 && parentID != id {
			if _, exists := tree.projects[parentID]; exists {
				valid = hasValidPathToRoot(parentID)
			}
		}
		if valid {
			states[id] = projectValid
		} else {
			states[id] = projectInvalid
		}
		return valid
	}

	for _, project := range projects {
		if !hasValidPathToRoot(project.ID) {
			tree.recovery = append(tree.recovery, project.ID)
			continue
		}
		tree.valid[project.ID] = true
		if project.ParentProjectID == 0 {
			tree.roots = append(tree.roots, project.ID)
		} else {
			tree.children[project.ParentProjectID] = append(tree.children[project.ParentProjectID], project.ID)
		}
	}
	tree.sortIDs(tree.roots)
	tree.sortIDs(tree.recovery)
	for parentID, childIDs := range tree.children {
		tree.sortIDs(childIDs)
		tree.children[parentID] = childIDs
		if previousCollapsed[parentID] {
			tree.collapsed[parentID] = true
		}
	}
	return tree
}

func (t projectTree) initialized() bool {
	return t.projects != nil
}

func (t projectTree) contains(id int64) bool {
	_, exists := t.projects[id]
	return exists
}

func (t projectTree) sortIDs(ids []int64) {
	sort.Slice(ids, func(i, j int) bool {
		left, right := t.projects[ids[i]], t.projects[ids[j]]
		if left.Position == right.Position {
			return left.ID < right.ID
		}
		return left.Position < right.Position
	})
}

func (t *projectTree) collapse(id int64) {
	if len(t.children[id]) > 0 {
		t.collapsed[id] = true
	}
}

func (t *projectTree) expand(id int64) {
	delete(t.collapsed, id)
}

func (t *projectTree) toggle(id int64) {
	if len(t.children[id]) == 0 {
		return
	}
	if t.collapsed[id] {
		delete(t.collapsed, id)
	} else {
		t.collapsed[id] = true
	}
}

func (t *projectTree) collapseAll() {
	for id, children := range t.children {
		if len(children) > 0 {
			t.collapsed[id] = true
		}
	}
}

func (t *projectTree) expandAll() {
	clear(t.collapsed)
}

func (t projectTree) collapsedState() map[int64]bool {
	state := make(map[int64]bool, len(t.collapsed))
	for id, collapsed := range t.collapsed {
		state[id] = collapsed
	}
	return state
}

func (t *projectTree) reveal(id int64) {
	seen := make(map[int64]bool)
	for t.valid[id] && !seen[id] {
		seen[id] = true
		parentID := t.projects[id].ParentProjectID
		if parentID == 0 {
			return
		}
		delete(t.collapsed, parentID)
		id = parentID
	}
}

func (t projectTree) parentID(id int64) (int64, bool) {
	project, exists := t.projects[id]
	if !exists || !t.valid[id] || project.ParentProjectID == 0 {
		return 0, false
	}
	return project.ParentProjectID, true
}

func (t projectTree) firstChildID(id int64) (int64, bool) {
	children := t.children[id]
	if len(children) == 0 {
		return 0, false
	}
	return children[0], true
}

func (t projectTree) visibleRows() []projectTreeRow {
	rows := make([]projectTreeRow, 0, len(t.projects))
	var appendBranch func(int64, int)
	appendBranch = func(id int64, depth int) {
		project := t.projects[id]
		children := t.children[id]
		expanded := len(children) > 0 && !t.collapsed[id]
		rows = append(rows, projectTreeRow{
			Project:     project,
			Depth:       depth,
			HasChildren: len(children) > 0,
			Expanded:    expanded,
		})
		if expanded {
			for _, childID := range children {
				appendBranch(childID, depth+1)
			}
		}
	}
	for _, rootID := range t.roots {
		appendBranch(rootID, 0)
	}
	for _, recoveryID := range t.recovery {
		rows = append(rows, projectTreeRow{Project: t.projects[recoveryID], Recovery: true})
	}
	return rows
}
