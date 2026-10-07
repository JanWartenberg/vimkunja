package description

import (
	"fmt"
	"slices"
	"unicode/utf8"
)

// Location resolves an anchor in this immutable document. Node is detached;
// Ancestors run from outermost parent to immediate parent. Block identifies
// the nearest enclosing logical block, independent of terminal coordinates.
type Location struct {
	Node      Node
	Block     ID
	Ancestors []ID
	Offset    int
}

func (d *Document) indexAnchors() {
	d.locations = make(map[ID]Location)
	var visit func([]Node, []ID, ID)
	visit = func(nodes []Node, parents []ID, block ID) {
		for _, n := range nodes {
			current := block
			if n.Kind == Element && (isBlock(n.Tag) || n.Tag == "li") || n.Kind == Opaque && n.blockOpaque {
				current = n.ID
			}
			d.locations[n.ID] = Location{Node: n, Block: current, Ancestors: slices.Clone(parents)}
			visit(n.Children, append(slices.Clone(parents), n.ID), current)
		}
	}
	visit(d.nodes, nil, 0)
}

// ResolveAnchor validates and resolves a document-local cursor target. Text
// offsets are UTF-8 byte boundaries (grapheme motions belong to the editor).
// Other nodes allow 0/1 for before/after, including empty blocks, br and atomic
// Opaque content; their internal source is never a cursor target. Anchors refer
// to this imported document. Draft editing and anchor rebasing are ticket 06.
func (d *Document) ResolveAnchor(anchor Anchor) (Location, error) {
	if d.class == Blocked {
		return Location{}, d.problem
	}
	location, ok := d.locations[anchor.Node]
	if !ok || anchor.Node == 0 {
		return Location{}, fmt.Errorf("%w: unknown node", ErrAnchor)
	}
	if location.Node.Kind == Text {
		text := location.Node.Text
		if anchor.Offset < 0 || anchor.Offset > len(text) || anchor.Offset < len(text) && !utf8.RuneStart(text[anchor.Offset]) {
			return Location{}, fmt.Errorf("%w: not a text boundary", ErrAnchor)
		}
	} else if anchor.Offset != 0 && anchor.Offset != 1 {
		return Location{}, fmt.Errorf("%w: not a structural boundary", ErrAnchor)
	}
	location.Node = cloneNodes([]Node{location.Node})[0]
	location.Ancestors = slices.Clone(location.Ancestors)
	location.Offset = anchor.Offset
	return location, nil
}
