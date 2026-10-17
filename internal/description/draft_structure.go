package description

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

func isList(n Node) bool { return n.Kind == Element && (n.Tag == "ul" || n.Tag == "ol") }
func listKind(n Node) string {
	if attributeValue(n.Attributes, "data-type") == "taskList" {
		return "task"
	}
	return n.Tag
}
func (d *Draft) parent(id ID) *Node {
	var find func([]Node) *Node
	find = func(ns []Node) *Node {
		for i := range ns {
			for _, c := range ns[i].Children {
				if c.ID == id {
					return &ns[i]
				}
			}
			if n := find(ns[i].Children); n != nil {
				return n
			}
		}
		return nil
	}
	return find(d.state.nodes)
}
func (d *Draft) entryAt(a Anchor) *Node {
	n := findNode(d.state.nodes, a.Node)
	for n != nil {
		if n.Tag == "li" {
			return n
		}
		n = d.parent(n.ID)
	}
	return nil
}

// Only the highest selected subtrees move. Descendants of those entries are
// part of the move even when the final selection anchor precedes their text.
func (d *Draft) listRange() (*Node, int, int, error) {
	if d.state.selection == nil || !d.state.selection.Blocks {
		return nil, 0, 0, fmt.Errorf("Select list entries with V first")
	}
	if d.codeLineSelection() {
		return nil, 0, 0, fmt.Errorf("Code line selections cannot move entire list entries")
	}
	lo, hi, ext := d.selectionRange()
	selected := map[ID]bool{}
	var visit func([]Node)
	visit = func(ns []Node) {
		for _, n := range ns {
			if n.Tag == "li" {
				start := ext[n.ID].start
				end := start
				for _, c := range checkboxContent(n) {
					if isList(c) {
						break
					}
					end = max(end, ext[c.ID].end)
				}
				if start < hi && end > lo || start == end && start >= lo && start <= hi {
					selected[n.ID] = true
				}
			}
			visit(n.Children)
		}
	}
	visit(d.state.nodes)
	for id := range selected {
		for p := d.parent(id); p != nil; p = d.parent(p.ID) {
			if selected[p.ID] {
				delete(selected, id)
				break
			}
		}
	}
	for _, b := range d.View().blocks {
		e, ok := ext[b.id]
		if e.node.Tag == "li" {
			e = projectionExtent(b, ext)
		}
		if !ok || !(e.start < hi && e.end > lo || e.start == e.end && e.start >= lo && e.start <= hi) {
			continue
		}
		li := d.entryAt(Anchor{b.id, 0})
		if li == nil {
			return nil, 0, 0, fmt.Errorf("Every selected block must belong to the moved list subtrees")
		}
		covered := selected[li.ID]
		for p := d.parent(li.ID); p != nil && !covered; p = d.parent(p.ID) {
			covered = selected[p.ID]
		}
		if !covered {
			return nil, 0, 0, ErrStructure
		}
	}
	var container *Node
	first, last := -1, -1
	for id := range selected {
		p := d.parent(id)
		if p == nil || !isList(*p) {
			return nil, 0, 0, ErrStructure
		}
		if container != nil && container.ID != p.ID {
			return nil, 0, 0, fmt.Errorf("List selection must be contiguous siblings")
		}
		container = p
	}
	if container == nil {
		return nil, 0, 0, ErrStructure
	}
	for i, n := range container.Children {
		if selected[n.ID] {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	for i := first; i <= last; i++ {
		if !selected[container.Children[i].ID] {
			return nil, 0, 0, fmt.Errorf("List selection must be contiguous siblings")
		}
	}
	return container, first, last + 1, nil
}
func appendEntries(receiver *Node, source Node, entries []Node) error {
	if len(entries) == 0 {
		return nil
	}
	if len(receiver.Children) >= 2 && receiver.Children[0].Kind == Element && receiver.Children[0].Tag == "label" && receiver.Children[1].Kind == Element && receiver.Children[1].Tag == "div" {
		return appendEntries(&receiver.Children[1], source, entries)
	}
	// Match the final child list only, preserving the relative order of all
	// existing children and of lists of other kinds.
	if len(receiver.Children) > 0 {
		last := &receiver.Children[len(receiver.Children)-1]
		if isList(*last) && listKind(*last) == listKind(source) {
			last.Children = append(last.Children, entries...)
			return nil
		}
	}
	// The original container keeps its metadata. A newly created child list
	// represents only the moved entries and starts destination numbering.
	attrs := []Attribute(nil)
	if listKind(source) == "task" {
		attrs = []Attribute{{"data-type", "taskList"}}
	}
	receiver.Children = append(receiver.Children, Node{Kind: Element, Tag: source.Tag, Attributes: attrs, Children: entries})
	return nil
}
func (d *Draft) moveList(kind ActionKind) error {
	source, lo, hi, err := d.listRange()
	if err != nil {
		return err
	}
	moved := slices.Clone(source.Children[lo:hi])
	if kind == IndentList {
		if lo == 0 || source.Children[lo-1].Tag != "li" {
			return fmt.Errorf("Indent needs an unselected previous sibling")
		}
		if err := appendEntries(&source.Children[lo-1], *source, moved); err != nil {
			return err
		}
		source.Children = slices.Delete(source.Children, lo, hi)
	} else {
		owner := d.parent(source.ID)
		if owner != nil && owner.Kind == Element && owner.Tag == "div" {
			owner = d.parent(owner.ID)
		}
		if owner == nil || owner.Tag != "li" {
			return fmt.Errorf("Cannot outdent a root entry; convert its list instead")
		}
		dest := d.parent(owner.ID)
		if dest == nil || !isList(*dest) {
			return ErrStructure
		}
		content := checkboxContent(*owner)
		if len(content) == 0 || content[len(content)-1].ID != source.ID {
			return fmt.Errorf("Outdent cannot reorder trailing parent content")
		}
		if listKind(*dest) != listKind(*source) {
			if err := d.outdentAcrossKinds(source, owner, dest, lo, hi); err != nil {
				return err
			}
			d.finishListMove(moved)
			return nil
		}
		if err := appendEntries(&moved[len(moved)-1], *source, slices.Clone(source.Children[hi:])); err != nil {
			return err
		}
		if lo == 0 {
			for _, a := range source.Attributes {
				if a.Name != "start" && a.Name != "reversed" && a.Name != "data-type" {
					return fmt.Errorf("Cannot discard source list metadata")
				}
			}
		}
		source.Children = source.Children[:lo]
		// Locate by identity after modifying descendants, before slices can move.
		at := slices.IndexFunc(dest.Children, func(n Node) bool { return n.ID == owner.ID })
		if len(source.Children) == 0 {
			parts, i := nodeList(&d.state.nodes, source.ID)
			*parts = slices.Delete(*parts, i, i+1)
		}
		dest = findNode(d.state.nodes, dest.ID)
		dest.Children = slices.Insert(dest.Children, at+1, moved...)
	}
	d.finishListMove(moved)
	return nil
}

// Retain numbered continuation across a split, honoring explicit li values.
func listOrdinal(list Node, index int) (int, error) {
	value := 1
	step := 1
	if hasAttribute(list.Attributes, "reversed") {
		value = len(list.Children)
		step = -1
	}
	if start := attributeValue(list.Attributes, "start"); start != "" {
		v, err := strconv.Atoi(start)
		if err != nil {
			return 0, ErrStructure
		}
		value = v
	}
	for i, n := range list.Children {
		if raw := attributeValue(n.Attributes, "value"); raw != "" {
			v, err := strconv.Atoi(raw)
			if err != nil {
				return 0, ErrStructure
			}
			value = v
		}
		if i == index {
			return value, nil
		}
		value += step
	}
	return value, nil
}
func hasAttribute(attrs []Attribute, name string) bool {
	return slices.ContainsFunc(attrs, func(a Attribute) bool { return a.Name == name })
}

func newEntry(list Node) Node {
	n := Node{Kind: Element, Tag: "li"}
	if listKind(list) == "task" {
		n.Attributes = []Attribute{{"data-type", "taskItem"}, {"data-checked", "false"}}
		n.Children = []Node{{Kind: Element, Tag: "p"}}
	}
	return n
}
func entryEmpty(n Node) bool {
	for _, c := range n.Children {
		if c.Kind == Opaque || c.Kind == Text && c.Text != "" || isList(c) || c.Tag == "br" || c.Tag == "hr" || !entryEmpty(c) {
			return false
		}
	}
	return true
}
func (d *Draft) splitEntry(li *Node) error {
	if entryEmpty(*li) {
		return d.exitEntry(li)
	}
	for _, a := range li.Attributes {
		switch a.Name {
		case "data-type", "data-task-id", "data-checked", "value":
		default:
			return fmt.Errorf("Cannot allocate entry metadata losslessly")
		}
	}
	left, right, err := splitAt(checkboxContent(*li), d.state.cursor)
	if err != nil {
		return err
	}
	container := d.parent(li.ID)
	beta := newEntry(*container)
	beta.Children = right
	blockContent := attributeValue(li.Attributes, "data-type") == "taskItem" || slices.ContainsFunc(checkboxContent(*li), func(n Node) bool {
		return n.Kind == Element && !isList(n) && isBlock(n.Tag)
	})
	if len(right) == 0 && blockContent || len(right) > 0 && isList(right[0]) {
		beta.Children = append([]Node{{Kind: Element, Tag: "p"}}, right...)
	}
	li.Children = left
	parts, i := nodeList(&d.state.nodes, li.ID)
	beta = d.newNode(beta)
	*parts = slices.Insert(*parts, i+1, beta)
	d.state.cursor = d.first([]Node{(*parts)[i+1]})
	return nil
}
func (d *Draft) openEntry(li *Node, kind ActionKind) error {
	container := d.parent(li.ID)
	parts, i := nodeList(&d.state.nodes, li.ID)
	if kind == OpenBelow {
		i++
	}
	n := d.newNode(newEntry(*container))
	*parts = slices.Insert(*parts, i, n)
	d.state.cursor = d.first([]Node{(*parts)[i]})
	return nil
}
func splitListAttributes(list Node, both bool) error {
	if !both {
		return nil
	}
	for _, a := range list.Attributes {
		if a.Name != "start" && a.Name != "reversed" && a.Name != "data-type" {
			return fmt.Errorf("Cannot allocate list split metadata losslessly")
		}
	}
	return nil
}
func (d *Draft) exitEntry(li *Node) error {
	container := d.parent(li.ID)
	parts, index := nodeList(&d.state.nodes, li.ID)
	if err := splitListAttributes(*container, index > 0 && index+1 < len(*parts)); err != nil {
		return err
	}
	for _, a := range li.Attributes {
		if a.Name != "data-type" && a.Name != "data-checked" && a.Name != "data-task-id" && a.Name != "value" {
			return fmt.Errorf("Cannot discard entry metadata on exit")
		}
	}
	before := slices.Clone((*parts)[:index])
	after := slices.Clone((*parts)[index+1:])
	old := *container
	out := []Node{}
	if len(before) > 0 {
		a := old
		a.Attributes = slices.Clone(old.Attributes)
		if a.Tag == "ol" && hasAttribute(a.Attributes, "reversed") && !hasAttribute(a.Attributes, "start") {
			v, err := listOrdinal(old, 0)
			if err != nil {
				return err
			}
			setAttribute(&a, "start", strconv.Itoa(v))
		}
		a.Children = before
		out = append(out, a)
	}
	flow := flowChildren(checkboxContent(*li))
	d.assign(flow)
	out = append(out, flow...)
	position := d.first(flow)
	if len(after) > 0 {
		a := old
		a.Attributes = slices.Clone(old.Attributes)
		a.Children = after
		if len(before) > 0 {
			a.ID = 0
		}
		if a.Tag == "ol" {
			value, err := listOrdinal(old, index+1)
			if err != nil {
				return err
			}
			setAttribute(&a, "start", strconv.Itoa(value))
		}
		out = append(out, a)
	}
	// An entirely emptied attributed container cannot lose unknown metadata.
	if len(before) == 0 && len(after) == 0 {
		for _, a := range old.Attributes {
			if a.Name != "start" && a.Name != "reversed" && a.Name != "data-type" {
				return ErrStructure
			}
		}
	}
	outer, i := nodeList(&d.state.nodes, container.ID)
	*outer = slices.Replace(*outer, i, i+1, out...)
	d.assign(d.state.nodes)
	d.state.cursor = position
	return nil
}

func (d *Draft) convertList(kind string) error {
	if d.state.selection != nil {
		return d.convertSelectedParagraphs(kind)
	}
	if kind != "ul" && kind != "ol" && kind != "task" {
		return ErrStructure
	}
	if li := d.selectedEntryAtCursor(); li != nil {
		container := d.parent(li.ID)
		for _, a := range li.Attributes {
			if a.Name != "data-type" && a.Name != "data-checked" && a.Name != "data-task-id" && a.Name != "value" {
				return fmt.Errorf("Cannot allocate converted entry metadata")
			}
		}
		var replacement []Node
		if listKind(*container) == kind {
			replacement = flowChildren(checkboxContent(*li))
		} else {
			item := *li
			item.Children = checkboxContent(*li)
			item.Attributes = nil
			if kind == "task" {
				item.Attributes = []Attribute{{"data-type", "taskItem"}, {"data-checked", "false"}}
			}
			list := listForKind(kind)
			list.Children = []Node{item}
			replacement = []Node{list}
		}
		return d.replaceEntry(li, replacement)
	}
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return err
	}
	b := findNode(d.state.nodes, loc.Block)
	if b == nil || b.Kind != Element || (b.Tag != "p" && b.Tag != "h1" && b.Tag != "h2" && b.Tag != "h3") {
		return ErrStructure
	}
	list := listForKind(kind)
	item := newEntry(list)
	item.Children = []Node{*b}
	list.Children = []Node{item}
	parts, i := nodeList(&d.state.nodes, b.ID)
	*parts = slices.Replace(*parts, i, i+1, d.newNode(list))
	return nil
}

// Conversion expands characterwise selections to touched logical paragraphs.
// The outer Apply owns rollback, preservation confirmation and one Undo entry.
func (d *Draft) convertSelectedParagraphs(kind string) error {
	if kind != "ul" && kind != "ol" && kind != "task" {
		return ErrStructure
	}
	if d.codeLineSelection() {
		return fmt.Errorf("Code lines cannot become Checkboxes; select prose paragraphs")
	}
	lo, hi, ext := d.selectionRange()
	var targets []ID
	seen := map[ID]bool{}
	for _, block := range d.View().blocks {
		e := projectionExtent(block, ext)
		if !(e.start < hi && e.end > lo || e.start == e.end && e.start >= lo && e.start <= hi) {
			continue
		}
		id := block.id
		if entry := d.selectedEntry(id); entry != nil {
			id = entry.ID
		}
		if !seen[id] {
			targets = append(targets, id)
			seen[id] = true
		}
	}
	if len(targets) == 0 {
		return ErrStructure
	}
	selection, cursor := d.state.selection, d.state.cursor
	d.state.selection = nil
	defer func() { d.state.selection = selection; d.state.cursor = cursor }()
	for _, id := range targets {
		node := findNode(d.state.nodes, id)
		if node == nil {
			return ErrStructure
		}
		if node.Tag == "li" {
			if parent := d.parent(id); parent != nil && listKind(*parent) == kind {
				continue
			}
		}
		d.state.cursor = d.first([]Node{*node})
		if err := d.convertList(kind); err != nil {
			return err
		}
	}
	// Merge only freshly allocated neighboring containers, leaving imported
	// container metadata and protected-object ancestry intact.
	var merge func([]Node) []Node
	merge = func(nodes []Node) []Node {
		var result []Node
		for _, n := range nodes {
			n.Children = merge(n.Children)
			if len(result) > 0 && isList(n) && findNode(d.origin.nodes, n.ID) == nil {
				previous := &result[len(result)-1]
				if isList(*previous) && findNode(d.origin.nodes, previous.ID) == nil && previous.Tag == n.Tag && slices.Equal(previous.Attributes, n.Attributes) {
					previous.Children = append(previous.Children, n.Children...)
					continue
				}
			}
			result = append(result, n)
		}
		return result
	}
	d.state.nodes = merge(d.state.nodes)
	return nil
}

func listForKind(kind string) Node {
	list := Node{Kind: Element, Tag: kind}
	if kind == "task" {
		list.Tag = "ul"
		list.Attributes = []Attribute{{"data-type", "taskList"}}
	}
	return list
}
func flowChildren(ns []Node) []Node {
	var out, inline []Node
	flush := func() {
		if len(inline) > 0 {
			out = append(out, Node{Kind: Element, Tag: "p", Children: inline})
			inline = nil
		}
	}
	for _, n := range ns {
		if n.Kind == Element && isBlock(n.Tag) || n.Kind == Opaque && n.blockOpaque {
			flush()
			out = append(out, n)
		} else {
			inline = append(inline, n)
		}
	}
	flush()
	if len(out) == 0 {
		out = []Node{{Kind: Element, Tag: "p"}}
	}
	return out
}
func (d *Draft) replaceEntry(li *Node, replacement []Node) error {
	old := cloneNodes([]Node{*d.parent(li.ID)})[0]
	i := slices.IndexFunc(old.Children, func(n Node) bool { return n.ID == li.ID })
	before, after := old.Children[:i], old.Children[i+1:]
	if err := splitListAttributes(old, len(before) > 0 && len(after) > 0); err != nil {
		return err
	}
	if len(before) == 0 && len(after) == 0 {
		for _, a := range old.Attributes {
			if a.Name != "start" && a.Name != "reversed" && a.Name != "data-type" {
				return fmt.Errorf("Cannot discard list metadata")
			}
		}
	}
	out := []Node{}
	if len(before) > 0 {
		list := old
		list.Attributes = slices.Clone(old.Attributes)
		if list.Tag == "ol" && hasAttribute(list.Attributes, "reversed") && !hasAttribute(list.Attributes, "start") {
			v, err := listOrdinal(old, 0)
			if err != nil {
				return err
			}
			setAttribute(&list, "start", strconv.Itoa(v))
		}
		list.Children = before
		out = append(out, list)
	}
	out = append(out, replacement...)
	if len(after) > 0 {
		list := old
		list.Attributes = slices.Clone(old.Attributes)
		list.Children = after
		if len(before) > 0 {
			list.ID = 0
		}
		if list.Tag == "ol" {
			v, err := listOrdinal(old, i+1)
			if err != nil {
				return err
			}
			setAttribute(&list, "start", strconv.Itoa(v))
		}
		out = append(out, list)
	}
	outer, j := nodeList(&d.state.nodes, old.ID)
	*outer = slices.Replace(*outer, j, j+1, out...)
	d.assign(d.state.nodes)
	if findNode(d.state.nodes, d.state.cursor.Node) == nil {
		d.state.cursor = d.first(d.state.nodes)
	}
	return nil
}
func (d *Draft) toggleCheckbox() error {
	li := d.selectedEntryAtCursor()
	if li == nil || attributeValue(li.Attributes, "data-type") != "taskItem" {
		return ErrStructure
	}
	checked := strings.EqualFold(attributeValue(li.Attributes, "data-checked"), "true")
	setAttribute(li, "data-checked", strconv.FormatBool(!checked))
	return nil
}

// Structural selection operates on actual logical projection blocks, then
// resolves document parents; it never rounds a selection to a larger container.
func (d *Draft) quoteTargets() ([]ID, error) {
	if d.state.selection == nil {
		loc, err := d.live().ResolveAnchor(d.state.cursor)
		if err != nil {
			return nil, err
		}
		return []ID{loc.Block}, nil
	}
	if !d.state.selection.Blocks {
		return nil, ErrStructure
	}
	if d.codeLineSelection() {
		return nil, fmt.Errorf("Code line selections cannot change the whole block; use :selectblock")
	}
	lo, hi, ext := d.selectionRange()
	var ids []ID
	for _, b := range d.View().blocks {
		e, ok := ext[b.id]
		if e.node.Tag == "li" {
			e = projectionExtent(b, ext)
		}
		if ok && e.start <= hi && e.end >= lo && (e.start < hi && e.end > lo || e.start == e.end && e.start >= lo && e.start <= hi) {
			ids = append(ids, b.id)
		}
	}
	if len(ids) == 0 {
		return nil, ErrStructure
	}
	return ids, nil
}
func (d *Draft) closestQuote(id ID) *Node {
	n := findNode(d.state.nodes, id)
	for n != nil {
		if n.Tag == "blockquote" {
			return n
		}
		n = d.parent(n.ID)
	}
	return nil
}
func (d *Draft) editQuote(kind ActionKind) error {
	ids, err := d.quoteTargets()
	if err != nil {
		return err
	}
	if kind == ToggleQuote {
		if d.closestQuote(ids[0]) != nil {
			kind = UnwrapQuote
		} else {
			kind = WrapQuote
		}
	}
	if kind == WrapQuote {
		parts, lo := nodeList(&d.state.nodes, ids[0])
		if parts == nil {
			return ErrStructure
		}
		hi := lo + 1
		for _, id := range ids[1:] {
			p, i := nodeList(&d.state.nodes, id)
			if p != parts || i != hi {
				return fmt.Errorf("Quote wrap needs contiguous sibling blocks")
			}
			hi++
		}
		parent := d.parent(ids[0])
		if parent != nil && isList(*parent) {
			return ErrStructure
		}
		q := d.newNode(Node{Kind: Element, Tag: "blockquote", Children: slices.Clone((*parts)[lo:hi])})
		*parts = slices.Replace(*parts, lo, hi, q)
		return nil
	}
	groups := map[ID]map[ID]bool{}
	var order []ID
	var commonParent ID
	for _, id := range ids {
		q := d.closestQuote(id)
		if q == nil {
			return fmt.Errorf("Every selected block must have the quote layer being removed")
		}
		p := d.parent(q.ID)
		pid := ID(0)
		if p != nil {
			pid = p.ID
		}
		if len(order) > 0 && pid != commonParent {
			return fmt.Errorf("Quote reduction needs one structural depth")
		}
		commonParent = pid
		if groups[q.ID] == nil {
			groups[q.ID] = map[ID]bool{}
			order = append(order, q.ID)
		}
		if id == q.ID {
			for _, child := range q.Children {
				groups[q.ID][child.ID] = true
			}
			continue
		}
		child := findNode(d.state.nodes, id)
		if parent := d.parent(child.ID); parent != nil && parent.ID != q.ID {
			return fmt.Errorf("Cannot unwrap only this nested block without changing unselected quote content")
		}
		for d.parent(child.ID) != nil && d.parent(child.ID).ID != q.ID {
			child = d.parent(child.ID)
		}
		groups[q.ID][child.ID] = true
	}
	for _, id := range order {
		q := findNode(d.state.nodes, id)
		selected := groups[id]
		lo, hi := -1, -1
		for i, n := range q.Children {
			if selected[n.ID] {
				if lo < 0 {
					lo = i
				}
				hi = i + 1
			}
		}
		if lo < 0 {
			if len(q.Children) == 0 {
				continue
			}
			return ErrStructure
		}
		for i := lo; i < hi; i++ {
			if !selected[q.Children[i].ID] {
				return fmt.Errorf("Quote unwrap needs contiguous sibling blocks")
			}
		}
		if lo > 0 && hi < len(q.Children) {
			for _, a := range q.Attributes {
				if a.Name != "cite" && a.Name != "data-comment-id" {
					return fmt.Errorf("Cannot split unique or unknown quote attributes losslessly")
				}
			}
		}
	}
	for _, id := range order {
		q := findNode(d.state.nodes, id)
		if len(q.Children) == 0 {
			p := d.newNode(Node{Kind: Element, Tag: "p"})
			outer, i := nodeList(&d.state.nodes, id)
			*outer = slices.Replace(*outer, i, i+1, p)
			if d.state.cursor.Node == id {
				d.state.cursor = Anchor{p.ID, 0}
			}
			if d.state.selection != nil {
				if d.state.selection.Start.Node == id {
					d.state.selection.Start = Anchor{p.ID, 0}
				}
				if d.state.selection.End.Node == id {
					d.state.selection.End = Anchor{p.ID, 0}
				}
			}
			continue
		}
		selected := groups[id]
		lo, hi := -1, -1
		for i, n := range q.Children {
			if selected[n.ID] {
				if lo < 0 {
					lo = i
				}
				hi = i + 1
			}
		}
		out := []Node{}
		if lo > 0 {
			left := *q
			left.Children = slices.Clone(q.Children[:lo])
			out = append(out, left)
		}
		out = append(out, slices.Clone(q.Children[lo:hi])...)
		if hi < len(q.Children) {
			right := *q
			right.Attributes = slices.Clone(q.Attributes)
			right.Children = slices.Clone(q.Children[hi:])
			if lo > 0 {
				right.ID = 0
			}
			out = append(out, right)
		}
		outer, i := nodeList(&d.state.nodes, id)
		*outer = slices.Replace(*outer, i, i+1, out...)
	}
	d.assign(d.state.nodes)
	if findNode(d.state.nodes, d.state.cursor.Node) == nil {
		d.state.cursor = d.first(d.state.nodes)
	}
	return nil
}

func (d *Draft) OnCheckbox() bool {
	li := d.selectedEntryAtCursor()
	return li != nil && attributeValue(li.Attributes, "data-type") == "taskItem"
}

func (d *Draft) mergeEntryBoundary(b *Node, kind ActionKind) (bool, error) {
	li := d.selectedEntryAtCursor()
	if li == nil {
		return false, nil
	}
	// Paragraphs in one entry keep ordinary same-parent paragraph merging.
	if b.ID != li.ID && (len(li.Children) != 1 || li.Children[0].ID != b.ID) {
		return false, nil
	}
	list := d.parent(li.ID)
	i := slices.IndexFunc(list.Children, func(n Node) bool { return n.ID == li.ID })
	j := i + 1
	if kind == Backspace {
		j = i - 1
	}
	if j < 0 || j >= len(list.Children) || list.Children[j].Tag != "li" {
		return true, ErrStructure
	}
	other := list.Children[j]
	if !slices.Equal(li.Attributes, other.Attributes) {
		return true, fmt.Errorf("Cannot merge entries with different attributes or identities")
	}
	left, right := *li, other
	if kind == Backspace {
		left, right = other, *li
	}
	inline := func(n Node) bool {
		for _, c := range n.Children {
			if c.Kind == Element && isBlock(c.Tag) || c.Kind == Opaque && c.blockOpaque {
				return false
			}
		}
		return true
	}
	var boundary Anchor
	if inline(left) && inline(right) {
		boundary = d.lastAnchor(left.Children)
		left.Children = append(left.Children, right.Children...)
	} else if len(left.Children) == 1 && len(right.Children) == 1 && left.Children[0].Tag == "p" && right.Children[0].Tag == "p" && slices.Equal(left.Children[0].Attributes, right.Children[0].Attributes) {
		boundary = d.lastAnchor(left.Children[0].Children)
		left.Children[0].Children = append(left.Children[0].Children, right.Children[0].Children...)
	} else {
		return true, fmt.Errorf("Entry structures cannot merge losslessly")
	}
	if boundary.Node == 0 {
		boundary = d.first(right.Children)
	}
	start := min(i, j)
	list.Children = slices.Replace(list.Children, start, max(i, j)+1, left)
	d.state.cursor = boundary
	return true, nil
}
func (d *Draft) lastAnchor(ns []Node) Anchor {
	for i := len(ns) - 1; i >= 0; i-- {
		n := ns[i]
		if n.Kind == Text {
			return Anchor{n.ID, len(n.Text)}
		}
		if n.Kind == Opaque || n.Tag == "br" {
			return Anchor{n.ID, 1}
		}
		if a := d.lastAnchor(n.Children); a.Node != 0 {
			return a
		}
	}
	return Anchor{}
}

func (d *Draft) EmptyListFill() bool {
	n := findNode(d.state.nodes, d.state.cursor.Node)
	return n != nil && isList(*n) && len(n.Children) == 0
}

func (d *Draft) lastNormalAnchor(ns []Node) Anchor {
	for i := len(ns) - 1; i >= 0; i-- {
		n := ns[i]
		if n.Kind == Text {
			off := len(n.Text)
			if off > 0 && !strings.HasSuffix(n.Text, "\n") {
				off = graphemeBefore(n.Text, off)
			}
			return Anchor{n.ID, off}
		}
		if n.Kind == Opaque || n.Tag == "br" || len(n.Children) == 0 {
			return Anchor{n.ID, 0}
		}
		if a := d.lastNormalAnchor(n.Children); a.Node != 0 {
			return a
		}
	}
	return Anchor{}
}
func projectionExtent(b block, ext map[ID]extent) extent {
	e := ext[b.id]
	first := true
	for _, g := range b.friendly {
		leaf, ok := ext[g.target.Anchor.Node]
		if !ok || leaf.node.Kind != Text && leaf.node.Kind != Opaque && leaf.node.Tag != "br" {
			continue
		}
		if first {
			e.start, e.end = leaf.start, leaf.end
			first = false
		} else {
			e.start = min(e.start, leaf.start)
			e.end = max(e.end, leaf.end)
		}
	}
	return e
}
func (d *Draft) selectedEntry(block ID) *Node {
	n := findNode(d.state.nodes, block)
	for n != nil {
		if n.Tag == "blockquote" || n.Tag == "pre" {
			return nil
		}
		if n.Tag == "li" {
			return n
		}
		n = d.parent(n.ID)
	}
	return nil
}

func (d *Draft) selectedEntryAtCursor() *Node {
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return nil
	}
	return d.selectedEntry(loc.Block)
}

func (d *Draft) finishListMove(moved []Node) {
	d.assign(d.state.nodes)
	start := d.first([]Node{*findNode(d.state.nodes, moved[0].ID)})
	end := d.lastNormalAnchor([]Node{*findNode(d.state.nodes, moved[len(moved)-1].ID)})
	d.state.cursor = end
	d.state.selection = &Selection{Start: start, End: end, Blocks: true}
}

// Mixed kinds cannot be siblings inside one HTML list. Split the outer list
// around a matching container; original IDs and numbered continuation survive.
func (d *Draft) outdentAcrossKinds(source, owner, dest *Node, lo, hi int) error {
	oldSource := cloneNodes([]Node{*source})[0]
	oldDest := cloneNodes([]Node{*dest})[0]
	at := slices.IndexFunc(dest.Children, func(n Node) bool { return n.ID == owner.ID })
	if err := splitListAttributes(oldDest, at+1 < len(dest.Children)); err != nil {
		return err
	}
	moved := slices.Clone(source.Children[lo:hi])
	if err := appendEntries(&moved[len(moved)-1], *source, slices.Clone(source.Children[hi:])); err != nil {
		return err
	}
	matching := listForKind(listKind(*source))
	matching.Children = moved
	source.Children = source.Children[:lo]
	if lo == 0 {
		matching.ID = oldSource.ID
		matching.Attributes = slices.Clone(oldSource.Attributes)
		parts, i := nodeList(&d.state.nodes, source.ID)
		*parts = slices.Delete(*parts, i, i+1)
	}
	dest = findNode(d.state.nodes, oldDest.ID)
	prefix := *dest
	prefix.Attributes = slices.Clone(oldDest.Attributes)
	prefix.Children = slices.Clone(dest.Children[:at+1])
	if prefix.Tag == "ol" && hasAttribute(prefix.Attributes, "reversed") && !hasAttribute(prefix.Attributes, "start") {
		v, err := listOrdinal(oldDest, 0)
		if err != nil {
			return err
		}
		setAttribute(&prefix, "start", strconv.Itoa(v))
	}
	out := []Node{prefix, matching}
	if at+1 < len(dest.Children) {
		after := *dest
		after.ID = 0
		after.Attributes = slices.Clone(oldDest.Attributes)
		after.Children = slices.Clone(dest.Children[at+1:])
		if after.Tag == "ol" {
			v, err := listOrdinal(oldDest, at+1)
			if err != nil {
				return err
			}
			setAttribute(&after, "start", strconv.Itoa(v))
		}
		out = append(out, after)
	}
	outer, i := nodeList(&d.state.nodes, oldDest.ID)
	*outer = slices.Replace(*outer, i, i+1, out...)
	return nil
}

func (d *Draft) codeLineSelection() bool {
	s := d.state.selection
	return s != nil && s.Blocks && !s.WholeBlock && s.codeBlock != 0
}
