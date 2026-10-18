package description

import (
	"fmt"
	"github.com/rivo/uniseg"
	"slices"
	"strings"
	"unicode"
)

type extent struct {
	start, end int
	block      ID
	node       Node
}

func (d *Draft) extents() (map[ID]extent, []ID) {
	out := map[ID]extent{}
	var blocks []ID
	pos := 0
	var visit func([]Node, ID)
	visit = func(ns []Node, block ID) {
		for _, n := range ns {
			start := pos
			current := block
			if n.Kind == Element && (n.Tag == "p" || strings.HasPrefix(n.Tag, "h") && n.Tag != "html" || n.Tag == "pre" || n.Tag == "li" || (n.Tag == "blockquote" || n.Tag == "ul" || n.Tag == "ol") && len(n.Children) == 0) || n.Kind == Opaque && n.blockOpaque && block == 0 {
				current = n.ID
				blocks = append(blocks, n.ID)
			}
			if n.Kind == Text {
				pos += len(n.Text)
			} else if n.Kind == Opaque || n.Tag == "br" {
				pos++
			} else {
				visit(n.Children, current)
			}
			out[n.ID] = extent{start, pos, current, n}
			if current == n.ID {
				pos++
			}
		}
	}
	visit(d.state.nodes, 0)
	return out, blocks
}
func anchorPosition(a Anchor, ext map[ID]extent) int {
	e := ext[a.Node]
	if e.node.Kind == Text || e.node.Kind == Opaque || e.node.Tag == "br" {
		return e.start + a.Offset
	}
	if a.Offset == 1 {
		return e.end
	}
	return e.start
}
func (d *Draft) selectionRange() (int, int, map[ID]extent) {
	ext, blocks := d.extents()
	s := d.state.selection
	if s == nil {
		p := anchorPosition(d.state.cursor, ext)
		return p, p, ext
	}
	lo, hi := anchorPosition(s.Start, ext), anchorPosition(s.End, ext)
	if lo > hi {
		lo, hi = hi, lo
	}
	if s.Blocks {
		a, b := ext[s.Start.Node], ext[s.End.Node]
		if a.block == b.block && d.codeLineSelection() {
			n := findNode(d.state.nodes, s.codeBlock)
			if n != nil {
				source := codeSource(n)
				base := ext[n.ID].start
				start, stop := codeLineBounds(source, lo-base)
				_, stop = codeLineBounds(source, hi-base)
				lo = base + start
				if strings.HasPrefix(source[stop:], "\r\n") {
					stop += 2
				} else if strings.HasPrefix(source[stop:], "\n") {
					stop++
				}
				hi = base + stop
				if lo == hi && lo > base && strings.HasSuffix(source, "\n") {
					lo--
					if lo > base && source[lo-base-1] == '\r' {
						lo--
					}
				}
				return lo, hi, ext
			}
		}
		if li := d.selectedEntry(a.block); li != nil && ext[a.block].node.Tag != "pre" {
			a.block = li.ID
		}
		if li := d.selectedEntry(b.block); li != nil && ext[b.block].node.Tag != "pre" {
			b.block = li.ID
		}
		lo, hi = min(ext[a.block].start, ext[b.block].start), max(ext[a.block].end, ext[b.block].end)
		_ = blocks
	} else if s.exclusive {
		return lo, hi, ext
	} else {
		end, syntax := s.End, s.endSyntax
		if anchorPosition(s.Start, ext) > anchorPosition(s.End, ext) || s.Start == s.End && s.startSyntax == 0 && s.endSyntax > 0 {
			end, syntax = s.Start, s.startSyntax
		}
		if syntax > 0 && ext[end.Node].node.Kind == Text {
			hi = anchorPosition(end, ext)
		} else {
			hi = anchorPosition(d.After(end), ext)
		}
	}
	return lo, hi, ext
}

// SelectRange selects the half-open document range used by Vim operators.
func (d *Draft) SelectRange(start, end Target) error {
	if err := d.SetTarget(start); err != nil {
		return err
	}
	d.Select(false, false)
	if err := d.SetTarget(end); err != nil {
		d.CancelSelection()
		return err
	}
	d.state.selection.exclusive = true
	return nil
}

func (d *Draft) Selected(a Anchor) bool {
	if d.state.selection == nil {
		return false
	}
	lo, hi, ext := d.selectionRange()
	p := anchorPosition(a, ext)
	return p >= lo && p < hi
}

// CopyLine returns selected safe text, or the current paragraph/Code source line.
func (d *Draft) CopyLine() string {
	if d.state.selection != nil {
		return d.Copy()
	}
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err == nil {
		if b := findNode(d.state.nodes, loc.Block); b != nil && b.Tag == "pre" {
			source := codeSource(b)
			lo, hi := codeLineBounds(source, d.codeOffset(b, d.state.cursor))
			return source[lo:hi]
		}
	}
	return d.Copy()
}

func (d *Draft) Copy() string {
	lo, hi, ext := d.selectionRange()
	if d.state.selection == nil {
		loc, _ := d.live().ResolveAnchor(d.state.cursor)
		e := ext[loc.Block]
		if loc.Node.Kind == Opaque {
			e = ext[loc.Node.ID]
		}
		lo, hi = e.start, e.end
	}
	var out strings.Builder
	var visit func([]Node)
	visit = func(ns []Node) {
		for _, n := range ns {
			e := ext[n.ID]
			if e.end < lo || e.start > hi {
				continue
			}
			if n.Kind == Text {
				a, b := max(0, lo-e.start), min(len(n.Text), hi-e.start)
				if a < b {
					out.WriteString(n.Text[a:b])
				}
			} else if n.Kind == Opaque {
				if e.start >= lo && e.end <= hi && n.Preview != "[Active content]" && n.Preview != "[Unsupported content]" && n.Preview != "[HTML comment]" {
					out.WriteString(n.Preview)
				}
			} else if n.Tag == "br" {
				if e.start >= lo && e.end <= hi {
					out.WriteByte('\n')
				}
			} else {
				visit(n.Children)
				if e.block == n.ID && e.end < hi && out.Len() > 0 {
					out.WriteByte('\n')
				}
			}
		}
	}
	visit(d.state.nodes)
	return out.String()
}
func markFamily(tag, family string) bool {
	switch family {
	case "strong":
		return tag == "strong" || tag == "b"
	case "em":
		return tag == "em" || tag == "i"
	case "u":
		return tag == "u"
	case "s":
		return tag == "s" || tag == "strike"
	}
	return false
}
func (d *Draft) format(a Action) error {
	if d.state.selection == nil {
		return ErrStructure
	}
	if d.codeLineSelection() {
		return fmt.Errorf("Code source cannot carry inline formatting")
	}
	if a.Kind == ToggleMark && a.Text != "strong" && a.Text != "em" && a.Text != "u" && a.Text != "s" {
		return ErrStructure
	}
	d.materializeSelectedFallback()
	lo, hi, ext := d.selectionRange()
	// Formatting is a real content edit: selected empty labels acquire their
	// visible fallback before marks are applied or removed.
	for _, e := range ext {
		if e.node.Tag != "a" || len(e.node.Children) != 0 || e.start < lo || e.start > hi || e.start == hi && d.state.selection.Start.Node != e.node.ID && d.state.selection.End.Node != e.node.ID {
			continue
		}
		n := findNode(d.state.nodes, e.node.ID)
		text := d.newNode(Node{Kind: Text, Text: attributeValue(n.Attributes, "href")})
		n.Children = []Node{text}
		remap := func(anchor *Anchor, syntax *int) {
			if anchor.Node != n.ID {
				return
			}
			offset := 0
			if *syntax < 0 {
				offset = min(len(text.Text), -*syntax-1)
			} else if anchor.Offset == 1 {
				offset = len(text.Text)
			}
			*anchor = Anchor{text.ID, offset}
			*syntax = 0
		}
		remap(&d.state.cursor, &d.state.syntax)
		remap(&d.state.selection.Start, &d.state.selection.startSyntax)
		remap(&d.state.selection.End, &d.state.selection.endSyntax)
	}
	lo, hi, ext = d.selectionRange()
	for _, e := range ext {
		if e.node.Tag == "pre" && e.start < hi && e.end > lo {
			return fmt.Errorf("Inline formatting cannot include Code blocks")
		}
	}
	all, found := true, false
	var check func([]Node, bool)
	check = func(ns []Node, marked bool) {
		for _, n := range ns {
			e := ext[n.ID]
			next := marked || markFamily(n.Tag, a.Text)
			if n.Kind == Text && e.start < hi && e.end > lo {
				found = true
				all = all && next
			}
			check(n.Children, next)
		}
	}
	check(d.state.nodes, false)
	if !found {
		return nil
	}
	for _, e := range ext {
		n := e.node
		remove := a.Kind == ClearMarks && (isInlineMark(n.Tag) || n.Tag == "a") || a.Kind == ToggleMark && all && markFamily(n.Tag, a.Text)
		if remove && e.start < hi && e.end > lo {
			for _, attr := range n.Attributes {
				if n.Tag != "a" || attr.Name == "id" {
					return ErrStructure
				}
			}
		}
	}

	startPos, endPos := anchorPosition(d.state.selection.Start, ext), anchorPosition(d.state.selection.End, ext)
	var rewrite func([]Node) []Node
	rewrite = func(ns []Node) []Node {
		var out []Node
		var inline []Node
		flush := func() {
			if len(inline) > 0 {
				out = append(out, d.formatInline(inline, a, lo, hi, ext, all)...)
				inline = nil
			}
		}
		for _, n := range ns {
			if n.Kind == Element && (isBlock(n.Tag) || n.Tag == "li") {
				flush()
				if n.Tag != "pre" {
					n.Children = rewrite(n.Children)
				}
				out = append(out, n)
			} else {
				inline = append(inline, n)
			}
		}
		flush()
		return out
	}
	d.state.nodes = rewrite(d.state.nodes)
	d.uniqueNativeIDs()
	d.assign(d.state.nodes)
	d.restoreSelection(startPos, endPos)
	return nil
}

type inlineRun struct {
	node   Node
	layers []Node
}

func (d *Draft) formatInline(ns []Node, a Action, lo, hi int, ext map[ID]extent, all bool) []Node {
	var runs []inlineRun
	var walk func([]Node, []Node)
	walk = func(ns []Node, layers []Node) {
		for _, n := range ns {
			e := ext[n.ID]
			if n.Kind == Element && n.Tag != "br" && len(n.Children) > 0 {
				walk(n.Children, append(slices.Clone(layers), n))
				continue
			}
			// Empty text still owns an editor anchor. Dropping it can turn its
			// container into a new empty block and shift later selection extents.
			if n.Kind == Text && n.Text == "" {
				runs = append(runs, inlineRun{n, slices.Clone(layers)})
				continue
			}
			if n.Kind != Text {
				if a.Kind == EditLink && n.Tag == "a" && len(n.Children) == 0 && e.start >= lo && e.end <= hi && (e.start < hi || d.state.selection.Start.Node == n.ID || d.state.selection.End.Node == n.ID) {
					if a.Text == "" {
						n = Node{Kind: Text, Text: attributeValue(n.Attributes, "href")}
					} else {
						setAttribute(&n, "href", a.Text)
						setAttribute(&n, "target", "_blank")
					}
				}
				path := slices.Clone(layers)
				if n.Tag == "br" && e.start >= lo && e.end <= hi {
					if a.Kind == EditLink {
						path = linkLayers(path, a.Text)
					} else if a.Kind == ClearMarks || all {
						path = slices.DeleteFunc(path, func(p Node) bool {
							return a.Kind == ClearMarks && (isInlineMark(p.Tag) || p.Tag == "a") || a.Kind == ToggleMark && all && markFamily(p.Tag, a.Text)
						})
					}
				}
				runs = append(runs, inlineRun{n, path})
				continue
			}
			start, end := max(0, min(len(n.Text), lo-e.start)), max(0, min(len(n.Text), hi-e.start))
			cuts := []int{0, start, end, len(n.Text)}
			for i := 0; i < 3; i++ {
				left, right := cuts[i], cuts[i+1]
				if left >= right {
					continue
				}
				v := n
				v.Text = n.Text[left:right]
				v.typed = slices.Clone(textOrigins(n)[left:right])
				if left > 0 {
					v.ID = 0
				}
				path := slices.Clone(layers)
				if e.start+left >= lo && e.start+right <= hi {
					marked := false
					for _, p := range path {
						marked = marked || markFamily(p.Tag, a.Text)
					}
					if a.Kind == ClearMarks {
						for _, layer := range path {
							if layer.Tag == "code" {
								v.typed = make([]byte, len(v.Text))
								break
							}
						}
					}
					if a.Kind == EditLink {
						path = linkLayers(path, a.Text)
					} else if a.Kind == ClearMarks || all {
						path = slices.DeleteFunc(path, func(p Node) bool {
							return a.Kind == ClearMarks && (isInlineMark(p.Tag) || p.Tag == "a") || a.Kind == ToggleMark && all && markFamily(p.Tag, a.Text)
						})
					} else if !marked {
						path = append(path, Node{Kind: Element, Tag: a.Text})
					}
				}
				runs = append(runs, inlineRun{v, path})
			}
		}
	}
	walk(ns, nil)
	// Adjacent leaves with a shared imported path reuse that wrapper. Splits
	// receive fresh Native identities; wrappers around Opaque keep original IDs.
	var out []Node
	var appendRun func(*[]Node, inlineRun, int)
	appendRun = func(dst *[]Node, r inlineRun, depth int) {
		if depth == len(r.layers) {
			*dst = append(*dst, r.node)
			return
		}
		layer := r.layers[depth]
		var wrapper *Node
		if len(*dst) > 0 {
			last := &(*dst)[len(*dst)-1]
			if last.Kind == Element && last.ID == layer.ID && last.Tag == layer.Tag && slices.Equal(last.Attributes, layer.Attributes) {
				wrapper = last
			}
		}
		if wrapper == nil {
			layer.Children = nil
			*dst = append(*dst, layer)
			wrapper = &(*dst)[len(*dst)-1]
		}
		appendRun(&wrapper.Children, r, depth+1)
	}
	for _, r := range runs {
		appendRun(&out, r, 0)
	}
	return coalesceMarks(out)
}
func (d *Draft) uniqueNativeIDs() {
	protected := map[ID]bool{}
	var retain func([]Node, []ID)
	retain = func(ns []Node, parents []ID) {
		for _, n := range ns {
			if n.Kind == Opaque {
				for _, id := range parents {
					protected[id] = true
				}
			}
			retain(n.Children, append(slices.Clone(parents), n.ID))
		}
	}
	retain(d.state.nodes, nil)
	seen := map[ID]bool{}
	var visit func([]Node)
	visit = func(ns []Node) {
		for i := range ns {
			n := &ns[i]
			if n.Kind != Opaque && n.ID != 0 {
				if seen[n.ID] || protected[n.ID] && !containsOpaque(*n) {
					n.ID = 0
				} else {
					seen[n.ID] = true
				}
			}
			visit(n.Children)
		}
	}
	visit(d.state.nodes)
}
func (d *Draft) anchorAt(pos int) Anchor {
	ext, _ := d.extents()
	best := Anchor{}
	bestDistance := int(^uint(0) >> 1)
	var visit func([]Node)
	visit = func(ns []Node) {
		for _, n := range ns {
			e := ext[n.ID]
			if n.Kind == Text {
				if pos >= e.start && pos <= e.end {
					best = Anchor{n.ID, pos - e.start}
					bestDistance = 0
				} else if dist := max(e.start-pos, pos-e.end); dist < bestDistance {
					best = Anchor{n.ID, max(0, min(len(n.Text), pos-e.start))}
					bestDistance = dist
				}
			} else if n.Kind == Opaque || n.Tag == "br" || len(n.Children) == 0 {
				dist := max(e.start-pos, pos-e.end)
				if dist < bestDistance {
					best = Anchor{n.ID, 0}
					bestDistance = dist
				}
			}
			visit(n.Children)
		}
	}
	visit(d.state.nodes)
	return best
}
func (d *Draft) restoreSelection(start, end int) {
	if d.state.selection != nil {
		d.state.selection.Start = d.anchorAt(start)
		d.state.selection.End = d.anchorAt(end)
		d.state.selection.startSyntax = 0
		d.state.selection.endSyntax = 0
		d.state.syntax = 0
		d.state.cursor = d.state.selection.End
	}
}
func (d *Draft) deleteSelection(a Action) error {
	if d.state.selection == nil {
		return ErrStructure
	}
	if err := d.materializeSelectionSyntax(); err != nil {
		return err
	}
	d.materializeSelectedFallback()
	lo, hi, ext := d.selectionRange()
	for _, t := range []Target{{Anchor: d.state.selection.Start, Syntax: d.state.selection.startSyntax}, {Anchor: d.state.selection.End, Syntax: d.state.selection.endSyntax}} {
		if n := findNode(d.state.nodes, t.Anchor.Node); n != nil && n.Kind == Text && t.Syntax > 0 && t.Anchor.Offset < len(n.Text) {
			flags := textOrigins(*n)
			flags[t.Anchor.Offset] = 1
			n.typed = flags
		}
	}
	startAnchor := d.state.selection.Start
	if anchorPosition(startAnchor, ext) > anchorPosition(d.state.selection.End, ext) {
		startAnchor = d.state.selection.End
	}
	startNode := findNode(d.state.nodes, startAnchor.Node)
	var startParent ID
	if startNode != nil && startNode.Kind == Text {
		if p := d.parent(startNode.ID); p != nil {
			startParent = p.ID
		}
	}
	blocks := d.state.selection.Blocks
	codeLines := d.codeLineSelection()
	if codeLines {
		b := findNode(d.state.nodes, d.state.selection.codeBlock)
		if b == nil {
			return ErrStructure
		}
		text := ""
		if a.Kind == ReplaceSelection {
			text = a.Text
		}
		d.state.selection = nil
		return d.applyCodeEdits(b, []codeEdit{{lo - ext[b.ID].start, hi - ext[b.ID].start, text}})
	}
	var replacement ID
	var remove func([]Node) []Node
	remove = func(ns []Node) []Node {
		var out []Node
		for _, n := range ns {
			// Checkbox controls have zero text extent and belong to their entry,
			// not to a partial selection starting at its first content character.
			if n.Kind == Element && checkboxControlTag(n.Tag) {
				out = append(out, n)
				continue
			}
			e := ext[n.ID]
			if e.start >= lo && e.end <= hi && (n.Kind != Element || blocks && !codeLines) {
				if blocks && a.Kind == ReplaceSelection && replacement == 0 && (n.Tag == "p" || n.Tag == "h1" || n.Tag == "h2" || n.Tag == "h3") {
					leaf := d.newNode(Node{Kind: Text})
					n.Children = []Node{leaf}
					replacement = leaf.ID
					out = append(out, n)
					continue
				}
				if a.Kind == ReplaceSelection && !blocks && n.Kind == Text && n.ID == startAnchor.Node {
					// Keep the insertion context when c consumes a whole leaf.
					// Falling back to its neighbor would inherit unrelated marks.
					n.Text, n.typed = "", nil
					out = append(out, n)
				}
				continue
			}
			if n.Kind == Text && e.start < hi && e.end > lo {
				start, end := max(0, lo-e.start), min(len(n.Text), hi-e.start)
				n.typed = slices.Delete(textOrigins(n), start, end)
				n.Text = n.Text[:start] + n.Text[end:]
			} else if n.Kind == Element {
				n.Children = remove(n.Children)
				if a.Kind != ReplaceSelection && n.Tag == "a" && len(n.Children) == 0 && e.start >= lo && e.end <= hi {
					continue
				}
			}
			out = append(out, n)
		}
		return out
	}
	d.state.nodes = remove(d.state.nodes)
	d.state.selection = nil
	d.state.cursor = d.anchorAt(lo)
	if replacement != 0 {
		d.state.cursor = Anchor{Node: replacement}
	}
	d.state.syntax = 0
	if a.Kind == DeleteSelection {
		d.parseTypedAtCursor()
	}
	if a.Kind == ReplaceSelection {
		if !blocks {
			d.state.inside = true
			if n := findNode(d.state.nodes, startAnchor.Node); n != nil && n.Kind == Text {
				d.state.cursor = Anchor{n.ID, min(startAnchor.Offset, len(n.Text))}
			} else if p := findNode(d.state.nodes, startParent); p != nil && (isInlineMark(p.Tag) || p.Tag == "a") && len(p.Children) == 0 {
				n := d.newNode(Node{Kind: Text})
				p.Children = []Node{n}
				d.state.cursor = Anchor{n.ID, 0}
			}
		}
		return d.edit(Action{Kind: Insert, Text: a.Text, inside: true})
	}
	return nil
}
func (d *Draft) deleteBoundary(b *Node, kind ActionKind) error {
	ext, _ := d.extents()
	pos := anchorPosition(d.state.cursor, ext)
	be := ext[b.ID]
	// Crossing inline wrappers and hard breaks uses document positions, never
	// disclosure glyphs. It cannot pull text across a quote/container boundary.
	if kind == Backspace && pos > be.start || kind == Delete && pos < be.end {
		target := pos
		if kind == Backspace {
			target--
		}
		for id, e := range ext {
			n := e.node
			if (n.Kind == Text || n.Kind == Opaque || n.Tag == "br") && e.start <= target && target < e.end {
				offset := max(0, min(e.end-e.start, pos-e.start))
				d.state.cursor = Anchor{id, offset}
				return d.edit(Action{Kind: kind})
			}
		}
		return ErrStructure
	}
	if handled, err := d.mergeEntryBoundary(b, kind); handled {
		return err
	}
	list, i := nodeList(&d.state.nodes, b.ID)
	j := i + 1
	if kind == Backspace {
		j = i - 1
	}
	if j < 0 || j >= len(*list) {
		return ErrStructure
	}
	other := (*list)[j]
	if other.Tag != b.Tag || b.Kind != Element || (b.Tag != "p" && b.Tag != "h1" && b.Tag != "h2" && b.Tag != "h3") || len(other.Attributes) != 0 || len(b.Attributes) != 0 {
		return ErrStructure
	}
	if kind == Backspace {
		boundary := ext[other.ID].end
		(*list)[j].Children = append((*list)[j].Children, b.Children...)
		*list = slices.Delete(*list, i, i+1)
		d.state.cursor = d.anchorAt(boundary)
	} else {
		boundary := be.end
		b.Children = append(b.Children, other.Children...)
		*list = slices.Delete(*list, j, j+1)
		d.state.cursor = d.anchorAt(boundary)
	}
	return nil
}

// SelectWord selects semantic text independently of disclosed delimiters.
func (d *Draft) SelectWord(around bool) error {
	ext, _ := d.extents()
	live := d.live()
	loc, err := live.ResolveAnchor(d.state.cursor)
	if err != nil {
		return err
	}
	block := ext[loc.Block]
	if block.node.Kind == Opaque {
		a := Anchor{Node: block.node.ID}
		d.state.selection = &Selection{Start: a, End: a}
		d.state.cursor, d.state.syntax = a, 0
		return nil
	}
	if loc.Node.Kind != Text && loc.Node.Kind != Opaque {
		n := findNode(d.state.nodes, loc.Node.ID)
		if n != nil {
			a := d.first(n.Children)
			if a.Node != 0 {
				d.state.cursor = a
			}
		}
	}
	pos := anchorPosition(d.state.cursor, ext)
	// A list entry may contain child blocks and then resume its own text.
	// Keep each contiguous run separate so a word cannot absorb descendants
	// or jump across their document positions.
	var runs [][]extent
	var collect func([]Node)
	collect = func(ns []Node) {
		for _, n := range ns {
			e := ext[n.ID]
			if n.Kind == Text || n.Kind == Opaque || n.Tag == "br" {
				if live.locations[n.ID].Block != loc.Block || e.start == e.end {
					continue
				}
				if len(runs) == 0 || runs[len(runs)-1][len(runs[len(runs)-1])-1].end != e.start {
					runs = append(runs, nil)
				}
				runs[len(runs)-1] = append(runs[len(runs)-1], e)
			} else {
				collect(n.Children)
			}
		}
	}
	n := findNode(d.state.nodes, loc.Block)
	if n == nil {
		return ErrStructure
	}
	collect(n.Children)
	var text strings.Builder
	base := 0
	for _, run := range runs {
		if pos < run[0].start || pos > run[len(run)-1].end {
			continue
		}
		base = run[0].start
		for _, e := range run {
			switch {
			case e.node.Kind == Text:
				text.WriteString(e.node.Text)
			case e.node.Kind == Opaque:
				text.WriteByte(1)
			case e.node.Tag == "br":
				text.WriteByte(' ')
			}
		}
		break
	}
	value := text.String()
	if value == "" {
		return ErrStructure
	}
	off := max(0, min(len(value), pos-base))
	if off == len(value) {
		off = graphemeBefore(value, off)
	}
	start, end := off, graphemeAfter(value, off)
	class := wordClass(value[start:end])
	atom := value[start:end] == "\x01"
	for !atom && start > 0 {
		p := graphemeBefore(value, start)
		if wordClass(value[p:start]) != class || value[p:start] == "\x01" {
			break
		}
		start = p
	}
	for !atom && end < len(value) {
		p := graphemeAfter(value, end)
		if wordClass(value[end:p]) != class || value[end:p] == "\x01" {
			break
		}
		end = p
	}
	if around {
		originalEnd := end
		for end < len(value) {
			p := graphemeAfter(value, end)
			if !strings.ContainsFunc(value[end:p], unicode.IsSpace) {
				break
			}
			end = p
		}
		if end == originalEnd {
			for start > 0 {
				p := graphemeBefore(value, start)
				if !strings.ContainsFunc(value[p:start], unicode.IsSpace) {
					break
				}
				start = p
			}
		}
	}
	// A text endpoint can share a position with the following atom. Word
	// selection addresses the character itself, not the insertion boundary.
	characterAnchor := func(pos int) Anchor {
		for id, e := range ext {
			if (e.node.Kind == Text || e.node.Kind == Opaque || e.node.Tag == "br") && e.start <= pos && pos < e.end {
				return Anchor{Node: id, Offset: pos - e.start}
			}
		}
		return d.anchorAt(pos)
	}
	d.state.selection = &Selection{Start: characterAnchor(base + start), End: characterAnchor(base + graphemeBefore(value, end))}
	d.state.cursor = d.state.selection.End
	d.state.syntax = 0
	return nil
}

func (d *Draft) AllowSelectionCursor(a Anchor) bool {
	if !d.codeLineSelection() {
		return true
	}
	end, err := d.live().ResolveAnchor(a)
	return err == nil && end.Block == d.state.selection.codeBlock
}
func (d *Draft) CodeSelection() bool { return d.codeLineSelection() }

// Highlight derives selection paint from document targets after layout.
func (d *Draft) Highlight(f Frame) Frame {
	if d.state.selection == nil {
		return f
	}
	positions := map[[2]int]bool{}
	lo, hi, ext := d.selectionRange()
	visualLo, visualHi, visual := 0, 0, false
	s := d.state.selection
	if !s.Blocks && (s.startSyntax != 0 || s.endSyntax != 0) {
		a, okA := findPoint(f.paint, Target{Anchor: s.Start, Syntax: s.startSyntax})
		b, okB := findPoint(f.paint, Target{Anchor: s.End, Syntax: s.endSyntax})
		if okA && okB {
			visualLo, visualHi = min(a, b), max(a, b)
			visual = true
		}
	}
	for index, p := range f.paint {
		at := anchorPosition(p.target.Anchor, ext)
		if visual && index >= visualLo && index <= visualHi || !visual && at >= lo && at < hi {
			for col := p.col; col < p.col+max(1, p.cellWidth); col++ {
				positions[[2]int{p.row, col}] = true
			}
		}
	}
	for row := range f.Rows {
		var spans []Span
		col := 0
		for _, s := range f.Rows[row].Spans {
			g := uniseg.NewGraphemes(s.Text)
			for g.Next() {
				text := g.Str()
				style := s.Style
				if positions[[2]int{row, col}] && (style&Protected == 0 || style&OpaqueStyle != 0) {
					style |= SelectedStyle
				}
				spans = append(spans, Span{Text: text, Style: style})
				col += uniseg.StringWidth(text)
			}
		}
		f.Rows[row].Spans = spans
	}
	return f
}
func coalesceMarks(ns []Node) []Node {
	var out []Node
	for _, n := range ns {
		n.Children = coalesceMarks(n.Children)
		if len(out) > 0 {
			last := &out[len(out)-1]
			if isInlineMark(n.Tag) && n.Tag == last.Tag && len(n.Attributes) == 0 && len(last.Attributes) == 0 && !containsOpaque(n) && !containsOpaque(*last) {
				last.Children = append(last.Children, n.Children...)
				continue
			}
		}
		out = append(out, n)
	}
	return out
}

func (d *Draft) ClampCodeSelection(target Anchor) Anchor {
	ext, _ := d.extents()
	towardEnd := anchorPosition(target, ext) > anchorPosition(d.state.cursor, ext)
	s := d.state.selection
	if s == nil {
		return d.state.cursor
	}
	n := findNode(d.state.nodes, s.codeBlock)
	if n == nil || n.Tag != "pre" {
		return d.state.cursor
	}
	first := d.first(n.Children)
	if !towardEnd {
		return first
	}
	var last Anchor
	var walk func([]Node)
	walk = func(ns []Node) {
		for _, n := range ns {
			if n.Kind == Text {
				last = Anchor{n.ID, graphemeBefore(n.Text, len(n.Text))}
				if strings.HasSuffix(n.Text, "\n") {
					last.Offset = len(n.Text)
				}
			}
			walk(n.Children)
		}
	}
	walk(n.Children)
	if last.Node == 0 {
		return first
	}
	return last
}

// SelectionContainsOpaque reports whole preserved atoms within the selection.
func (d *Draft) SelectionContainsOpaque() bool {
	if d.state.selection == nil {
		return false
	}
	lo, hi, ext := d.selectionRange()
	for _, e := range ext {
		if e.node.Kind == Opaque && e.start < hi && e.end > lo {
			return true
		}
	}
	return false
}
