package description

import (
	"fmt"
	"github.com/rivo/uniseg"
	"slices"
	"strings"
)

// Code operations share Draft's atomic validation and history boundary.
func (d *Draft) editCode(a Action) error {
	if a.Tabstop <= 0 {
		a.Tabstop = 4
	}
	if a.Indent != 2 && a.Indent != 4 {
		a.Indent = 4
	}
	if a.Kind == RetabAll {
		var walk func([]Node, *int)
		walk = func(ns []Node, column *int) {
			for i := range ns {
				n := &ns[i]
				if n.Kind == Opaque {
					*column = 0
					continue
				}
				if n.Kind == Text {
					old := n.Text
					initial := *column
					d.remapCodeAnchors(n.ID, func(off int) int { prefix, _ := expandCodeTabs(old[:off], initial, a.Tabstop); return len(prefix) })
					n.Text, *column = expandCodeTabs(old, initial, a.Tabstop)
				} else if n.Tag == "br" {
					*column = 0
				} else if isBlock(n.Tag) || n.Tag == "li" {
					local := 0
					walk(n.Children, &local)
				} else {
					walk(n.Children, column)
				}
			}
		}
		col := 0
		walk(d.state.nodes, &col)
		return nil
	}
	if a.Kind == CreateCode && len(d.state.nodes) == 0 {
		d.state.nodes = []Node{d.newNode(Node{Kind: Element, Tag: "p"})}
		d.state.cursor = d.first(d.state.nodes)
	}
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return err
	}
	b := findNode(d.state.nodes, loc.Block)
	if b == nil || b.Kind != Element {
		return ErrStructure
	}
	if a.Kind == EscapeCodeMarker {
		if d.state.selection != nil || b.Tag != "p" || len(b.Children) != 1 || b.Children[0].Kind != Text || b.Children[0].Text != "\\" || d.state.cursor.Offset != 1 || !strings.HasPrefix(a.Text, "`") {
			return ErrStructure
		}
		if err := d.edit(Action{Kind: Backspace}); err != nil {
			return err
		}
		return d.edit(Action{Kind: Insert, Text: a.Text})
	}
	if a.Kind == CreateCode {
		if d.state.selection != nil || b.Tag != "p" {
			return ErrStructure
		}
		text := ""
		for _, child := range b.Children {
			if child.Kind != Text {
				return fmt.Errorf("Code creation requires plain paragraph text")
			}
			text += child.Text
		}
		if a.Text == "marker" {
			if text != "```" && text != "````" && !d.CodeMarkerPairAtCursor() {
				return ErrStructure
			}
			text = ""
		}
		b.Tag = "pre"
		b.Children = []Node{d.newNode(Node{Kind: Element, Tag: "code", Children: []Node{{Kind: Text, Text: text}}})}
		d.state.cursor = d.first(b.Children)
		return nil
	}
	if b.Tag != "pre" {
		return fmt.Errorf("This command requires Native Code")
	}
	if a.Kind == CodeBefore || a.Kind == CodeAfter {
		if d.state.selection != nil {
			return ErrStructure
		}
		list, i := nodeList(&d.state.nodes, b.ID)
		if a.Kind == CodeAfter {
			i++
		}
		p := d.newNode(Node{Kind: Element, Tag: "p"})
		*list = slices.Insert(*list, i, p)
		d.state.cursor = Anchor{p.ID, 0}
		return nil
	}
	return d.editCodeSource(b, a)
}

func (d *Draft) editCodeSource(b *Node, a Action) error {
	source := codeSource(b)
	off := d.codeOffset(b, d.state.cursor)
	start, end := 0, len(source)
	if d.state.selection != nil {
		if !d.codeLineSelection() {
			return fmt.Errorf("Select Code source lines with V")
		}
		lo, hi, ext := d.selectionRange()
		start = lo - ext[b.ID].start
		end = hi - ext[b.ID].start
	} else if a.Kind != Retab {
		start, end = codeLineBounds(source, off)
	}
	if start < 0 || end < start || end > len(source) {
		return ErrStructure
	}
	switch a.Kind {
	case InsertTab:
		if d.state.selection != nil {
			return ErrStructure
		}
		col := codeColumn(b, off, a.Tabstop)
		return d.applyCodeEdits(b, []codeEdit{{off, off, strings.Repeat(" ", a.Tabstop-col%a.Tabstop)}})
	case Split, HardBreak:
		return d.applyCodeEdits(b, []codeEdit{{off, off, codeNewline(source, off)}})
	case OpenAbove, OpenBelow:
		pos := start
		if a.Kind == OpenBelow {
			pos = end
		}
		newline := codeNewline(source, off)
		if err := d.applyCodeEdits(b, []codeEdit{{pos, pos, newline}}); err != nil {
			return err
		}
		if a.Kind == OpenAbove {
			d.state.cursor = d.codeAnchor(b, pos, 0)
		} else {
			d.state.cursor = d.codeAnchor(b, pos+len(newline), 0)
		}
		return nil
	case DeleteBlock:
		if source == "" {
			list, i := nodeList(&d.state.nodes, b.ID)
			*list = slices.Delete(*list, i, i+1)
			d.state.cursor = d.first(d.state.nodes)
			d.state.selection = nil
			return nil
		}
		if start == len(source) && start > 0 {
			start--
			if start > 0 && source[start-1] == '\r' {
				start--
			}
		} else if strings.HasPrefix(source[end:], "\r\n") {
			end += 2
		} else if strings.HasPrefix(source[end:], "\n") {
			end++
		}
		return d.applyCodeEdits(b, []codeEdit{{start, end, ""}})
	case Delete, Backspace:
		start, end = off, off
		if a.Kind == Backspace {
			if off == 0 {
				return ErrStructure
			}
			start = graphemeBefore(source, off)
		} else {
			if off == len(source) {
				return ErrStructure
			}
			end = graphemeAfter(source, off)
		}
		return d.applyCodeEdits(b, []codeEdit{{start, end, ""}})
	}
	var edits []codeEdit
	if a.Kind == Retab {
		column := 0
		atoms := codeAtoms(b)
		g := uniseg.NewGraphemes(source[start:end])
		for g.Next() {
			lo, hi := g.Positions()
			if _, ok := atoms[start+lo]; ok {
				column = 0
				continue
			}
			switch g.Str() {
			case "\t":
				width := a.Tabstop - column%a.Tabstop
				edits = append(edits, codeEdit{start + lo, start + hi, strings.Repeat(" ", width)})
				column += width
			case "\n", "\r\n", "\r":
				column = 0
			default:
				column += g.Width()
			}
		}
	} else {
		for pos := start; pos < end; {
			stop := end
			if i := strings.Index(source[pos:end], "\n"); i >= 0 {
				stop = pos + i
			}
			contentEnd := stop
			if contentEnd > pos && source[contentEnd-1] == '\r' {
				contentEnd--
			}
			if contentEnd > pos {
				if a.Kind == IndentCode {
					edits = append(edits, codeEdit{pos, pos, strings.Repeat(" ", a.Indent)})
				} else {
					col, i := 0, pos
					for i < contentEnd && col < a.Indent {
						if source[i] == ' ' {
							col++
							i++
						} else if source[i] == '\t' {
							col += a.Tabstop - col%a.Tabstop
							i++
						} else {
							break
						}
					}
					if i > pos {
						edits = append(edits, codeEdit{pos, i, strings.Repeat(" ", max(0, col-a.Indent))})
					}
				}
			}
			pos = stop + 1
		}
	}
	return d.applyCodeEdits(b, edits)
}

// One logical source can span several preserved <code> wrappers. Source edits
// retain wrappers and leaf identities instead of flattening their metadata.
type codeEdit struct {
	start, end int
	text       string
}

func codeLeaves(b *Node) []*Node {
	var result []*Node
	var walk func(*Node)
	walk = func(n *Node) {
		if n.Kind == Text || n.Kind == Opaque {
			result = append(result, n)
		} else {
			for i := range n.Children {
				walk(&n.Children[i])
			}
		}
	}
	walk(b)
	return result
}

// Keep inserted content inside the imported code wrapper, including its metadata.
func (d *Draft) materializeCodeText(b *Node) *Node {
	target := b
	var walk func(*Node)
	walk = func(n *Node) {
		if n.Tag == "code" {
			target = n
			return
		}
		for i := range n.Children {
			walk(&n.Children[i])
		}
	}
	walk(b)
	target.Children = append(target.Children, d.newNode(Node{Kind: Text}))
	return &target.Children[len(target.Children)-1]
}
func codeSource(b *Node) string {
	var text strings.Builder
	for _, n := range codeLeaves(b) {
		if n.Kind == Opaque {
			text.WriteByte(0)
		} else {
			text.WriteString(n.Text)
		}
	}
	return text.String()
}
func (d *Draft) codeOffset(b *Node, a Anchor) int {
	ext, _ := d.extents()
	return anchorPosition(a, ext) - ext[b.ID].start
}
func (d *Draft) codeAnchor(b *Node, pos int, prefer ID) Anchor {
	start := 0
	leaves := codeLeaves(b)
	for _, n := range leaves {
		end := start + codeUnitLength(n)
		if n.ID == prefer && pos >= start && pos <= end {
			return Anchor{n.ID, pos - start}
		}
		start = end
	}
	start = 0
	for _, n := range leaves {
		end := start + codeUnitLength(n)
		if pos < end || pos == end && n == leaves[len(leaves)-1] {
			return Anchor{n.ID, max(0, pos-start)}
		}
		start = end
	}
	return Anchor{b.ID, 0}
}
func (d *Draft) applyCodeEdits(b *Node, edits []codeEdit) error {
	if len(edits) == 0 {
		return nil
	}
	// A stale or cross-block Code selection must not become a source edit.
	sourceLength := len(codeSource(b))
	for _, e := range edits {
		if e.start < 0 || e.end < e.start || e.end > sourceLength {
			return ErrStructure
		}
	}
	cursor := d.state.cursor
	cursorPos := d.codeOffset(b, cursor)
	selection := d.Selection()
	startPos, endPos := 0, 0
	if selection != nil {
		startPos = d.codeOffset(b, selection.Start)
		endPos = d.codeOffset(b, selection.End)
	}
	leaves := codeLeaves(b)
	// An atom's source is private; edits crossing it cannot safely infer Code lines.
	pos := 0
	for _, n := range leaves {
		if n.Kind == Opaque {
			for _, e := range edits {
				if e.start < pos+1 && e.end > pos {
					return fmt.Errorf("Code operation crosses preserved content; select the whole block to remove it")
				}
			}
		}
		pos += codeUnitLength(n)
	}
	if len(leaves) == 0 {
		d.materializeCodeText(b)
		leaves = codeLeaves(b)
	}
	for _, e := range edits {
		pos := 0
		for _, n := range leaves {
			if n.Kind == Opaque && e.start == pos && e.start == e.end && e.text != "" {
				list, i := nodeList(&d.state.nodes, n.ID)
				*list = slices.Insert(*list, i, d.newNode(Node{Kind: Text}))
				leaves = codeLeaves(b)
				break
			}
			pos += codeUnitLength(n)
		}
	}
	if leaves[len(leaves)-1].Kind == Opaque {
		last := leaves[len(leaves)-1]
		for _, e := range edits {
			if e.start == len(codeSource(b)) && e.text != "" {
				list, i := nodeList(&d.state.nodes, last.ID)
				*list = slices.Insert(*list, i+1, d.newNode(Node{Kind: Text}))
				leaves = codeLeaves(b)
				break
			}
		}
	}
	owners := make([]int, len(edits))
	for j, e := range edits {
		pos := 0
		owners[j] = len(leaves) - 1
		for i, n := range leaves {
			pos += codeUnitLength(n)
			if (e.start < pos || len(n.Text) == 0 && e.start == pos) && n.Kind == Text {
				owners[j] = i
				break
			}
		}
	}
	global := 0
	for i, n := range leaves {
		old := n.Text
		if n.Kind == Opaque {
			global++
			continue
		}
		stop := global + len(old)
		local := 0
		var out strings.Builder
		for j, e := range edits {
			if e.end < global || e.start > stop {
				continue
			}
			lo, hi := max(0, e.start-global), min(len(old), e.end-global)
			lo = min(lo, len(old))
			hi = max(hi, lo)
			if hi > local || owners[j] == i {
				out.WriteString(old[local:lo])
				if owners[j] == i {
					out.WriteString(e.text)
				}
				local = hi
			}
		}
		out.WriteString(old[local:])
		n.Text = out.String()
		global = stop
	}
	mapPosition := func(pos int) int {
		delta := 0
		for _, e := range edits {
			if pos < e.start {
				break
			}
			if pos >= e.end {
				delta += len(e.text) - (e.end - e.start)
			} else {
				return e.start + delta + len(e.text)
			}
		}
		return pos + delta
	}
	d.state.cursor = d.codeAnchor(b, mapPosition(cursorPos), cursor.Node)
	if s := d.state.selection; s != nil {
		s.Start = d.codeAnchor(b, mapPosition(startPos), selection.Start.Node)
		s.End = d.codeAnchor(b, mapPosition(endPos), selection.End.Node)
	}
	return nil
}

// Columns count terminal cells; Tabs are relative to the source line, not its gutter.
func expandCodeTabs(text string, column, tabstop int) (string, int) {
	var out strings.Builder
	g := uniseg.NewGraphemes(text)
	for g.Next() {
		s := g.Str()
		switch s {
		case "\t":
			width := tabstop - column%tabstop
			out.WriteString(strings.Repeat(" ", width))
			column += width
		case "\n", "\r\n", "\r":
			out.WriteString(s)
			column = 0
		default:
			out.WriteString(s)
			column += g.Width()
		}
	}
	return out.String(), column
}

// The end excludes CRLF's CR so line operations never split its two bytes.
func codeLineBounds(text string, offset int) (int, int) {
	start := strings.LastIndex(text[:offset], "\n") + 1
	end := len(text)
	if i := strings.Index(text[offset:], "\n"); i >= 0 {
		end = offset + i
		if end > start && text[end-1] == '\r' {
			end--
		}
	}
	return start, end
}

// CodeMarkerParagraph is deliberately narrow: no imported formatting is consumed.
func (d *Draft) CodeMarkerParagraph(emptyOnly bool) bool {
	text, _, ok := d.codeMarkerParagraph(emptyOnly)
	return ok && (emptyOnly && text == "" || !emptyOnly && (text == "```" || text == "````"))
}

// CodeMarkerPairAtCursor recognizes Enter between two deliberately typed fences.
// Six markers alone, another split position, and imported/Paste text stay literal.
func (d *Draft) CodeMarkerPairAtCursor() bool {
	if d.state.selection != nil {
		return false
	}
	text, offset, ok := d.codeMarkerParagraph(false)
	return ok && text == "``````" && offset == 3
}

func (d *Draft) codeMarkerParagraph(emptyOnly bool) (string, int, bool) {
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return "", 0, false
	}
	b := findNode(d.state.nodes, loc.Block)
	if b == nil || b.Kind != Element || b.Tag != "p" {
		return "", 0, false
	}
	text, offset := "", -1
	for _, n := range b.Children {
		if n.Kind != Text {
			return "", 0, false
		}
		if n.ID == d.state.cursor.Node {
			offset = len(text) + d.state.cursor.Offset
		}
		if !emptyOnly {
			for _, f := range textOrigins(n) {
				if f != 1 {
					return "", 0, false
				}
			}
		}
		text += n.Text
	}
	return text, offset, true
}

func (d *Draft) remapCodeAnchors(id ID, mapOffset func(int) int) {
	if d.state.cursor.Node == id {
		d.state.cursor.Offset = mapOffset(d.state.cursor.Offset)
	}
	if s := d.state.selection; s != nil {
		if s.Start.Node == id {
			s.Start.Offset = mapOffset(s.Start.Offset)
		}
		if s.End.Node == id {
			s.End.Offset = mapOffset(s.End.Offset)
		}
	}
}
func codeNewline(text string, offset int) string {
	_, end := codeLineBounds(text, offset)
	if strings.HasPrefix(text[end:], "\r\n") {
		return "\r\n"
	}
	if end == len(text) && strings.Contains(text, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

// Opaque occupies one document position and ends the known source-column run.
// Its safe Preview may be a label, so it must never substitute for hidden Code.
func codeUnitLength(n *Node) int {
	if n.Kind == Opaque {
		return 1
	}
	return len(n.Text)
}
func codeAtoms(b *Node) map[int]*Node {
	atoms := map[int]*Node{}
	pos := 0
	for _, n := range codeLeaves(b) {
		if n.Kind == Opaque {
			atoms[pos] = n
		}
		pos += codeUnitLength(n)
	}
	return atoms
}
func codeColumn(b *Node, offset, tabstop int) int {
	col, pos := 0, 0
	for _, n := range codeLeaves(b) {
		if pos >= offset {
			break
		}
		if n.Kind == Opaque {
			col = 0
		} else {
			_, col = expandCodeTabs(n.Text[:min(len(n.Text), offset-pos)], col, tabstop)
		}
		pos += codeUnitLength(n)
	}
	return col
}
