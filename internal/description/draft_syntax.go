package description

import (
	"slices"
	"strings"
)

const editorEscapable = "\\*_~`#-+.[]"

// Disclose protection only where literal text could be mistaken for an inline
// formatting pair. Reuse the editor grammar so lone punctuation has neither a
// synthetic backslash nor an invisible editing target. Typed escapes stay valid.
func disclosedTextEscapes(n Node, inlineCode bool) []bool {
	escapes := make([]bool, len(n.Text))
	if !strings.ContainsAny(n.Text, editorEscapable) {
		return escapes
	}
	probe := n
	probe.typed = []byte(strings.Repeat("\x01", len(n.Text)))
	_, tokens := parseTyped(probe)
	for _, token := range tokens {
		for offset := token.start; offset < token.end; offset++ {
			escapes[offset] = true
		}
	}
	flags := textOrigins(n)
	for offset, r := range n.Text {
		if inlineCode {
			// An interior backtick could be confused with the disclosed closer.
			escapes[offset] = r == '`' || r == '\\'
		} else if r == '\\' && offset+1 < len(n.Text) {
			escapes[offset] = strings.ContainsRune(editorEscapable, rune(n.Text[offset+1]))
		}
		if flags[offset] != 0 {
			escapes[offset] = false
		}
	}
	return escapes
}

func textOrigins(n Node) []byte { out := make([]byte, len(n.Text)); copy(out, n.typed); return out }
func (d *Draft) SetTarget(t Target) error {
	if err := d.SetCursor(t.Anchor); err != nil {
		return err
	}
	d.state.syntax = t.Syntax
	if s := d.state.selection; s != nil {
		s.endSyntax = t.Syntax
	}
	return nil
}
func (d *Draft) Target() Target { return Target{Anchor: d.state.cursor, Syntax: d.state.syntax} }
func (d *Draft) AfterTarget() Target {
	t := d.Target()
	if t.Syntax > 0 {
		t.Syntax++
	} else if t.Syntax < 0 {
		n := findNode(d.state.nodes, t.Anchor.Node)
		if n != nil {
			value := attributeValue(n.Attributes, "href")
			t.Syntax = -graphemeAfter(value, min(len(value), -t.Syntax-1)) - 1
		}
	} else {
		t.Anchor = d.After(t.Anchor)
	}
	return t
}

func (d *Draft) typeText(text string, inside bool) error {
	if d.state.selection != nil {
		inside = true
		if err := d.deleteSelection(Action{Kind: ReplaceSelection}); err != nil {
			return err
		}
	}
	if d.InCode() {
		return d.edit(Action{Kind: Insert, Text: text})
	}
	for _, r := range text {
		// A typed newline outside Code creates a block, not a text byte whose
		// origin can be marked at the resulting cursor's offset.
		if r == '\n' && !d.InCode() {
			if err := d.edit(Action{Kind: Split}); err != nil {
				return err
			}
			continue
		}
		d.continueClosingRun(r)
		n := findNode(d.state.nodes, d.state.cursor.Node)
		escaped := false
		if n != nil && n.Kind == Text && d.state.cursor.Offset > 0 {
			off := d.state.cursor.Offset
			flags := textOrigins(*n)
			if n.Text[off-1] == '\\' && flags[off-1] == 2 {
				if strings.ContainsRune(editorEscapable, r) {
					n.Text = n.Text[:off-1] + n.Text[off:]
					n.typed = slices.Delete(flags, off-1, off)
					d.state.cursor.Offset--
					escaped = true
				} else {
					flags[off-1] = 0
					n.typed = flags
				}
			}
		}
		if err := d.edit(Action{Kind: Insert, Text: string(r), inside: inside}); err != nil {
			return err
		}
		n = findNode(d.state.nodes, d.state.cursor.Node)
		if n == nil || n.Kind != Text {
			continue
		}
		flags := textOrigins(*n)
		off := d.state.cursor.Offset - len(string(r))
		kind := byte(1)
		if escaped {
			kind = 0
		} else if r == '\\' {
			kind = 2
		}
		for i := off; i < d.state.cursor.Offset; i++ {
			flags[i] = kind
		}
		loc, _ := d.live().ResolveAnchor(d.state.cursor)
		for _, id := range loc.Ancestors {
			if p := findNode(d.state.nodes, id); p != nil && p.Tag == "code" && kind != 2 {
				for i := off; i < d.state.cursor.Offset; i++ {
					flags[i] = 0
				}
				break
			}
		}
		n.typed = flags
		if err := d.typedStructure(); err != nil {
			return err
		}
		d.parseTypedAtCursor()
	}
	return nil
}

type syntaxToken struct {
	start, end  int
	marker, tag string
}

func syntaxTokens(n Node) []syntaxToken {
	flags := textOrigins(n)
	var out []syntaxToken
	for i := 0; i < len(n.Text); {
		c := n.Text[i]
		if flags[i] != 1 || !strings.ContainsRune("*_~`", rune(c)) {
			i++
			continue
		}
		j := i + 1
		for j < len(n.Text) && n.Text[j] == c && flags[j] == 1 {
			j++
		}
		marker := n.Text[i:j]
		tag := ""
		switch marker {
		case "*":
			tag = "em"
		case "**":
			tag = "strong"
		case "***":
			tag = "combined"
		case "__":
			tag = "u"
		case "~~":
			tag = "s"
		case "`":
			tag = "code"
		}
		if tag != "" {
			out = append(out, syntaxToken{i, j, marker, tag})
		}
		i = j
	}
	return out
}
func parseTyped(n Node) ([]Node, []syntaxToken) {
	tokens := syntaxTokens(n)
	pairs := map[int]int{}
	var stack []int
	// Crossed or incomplete expressions are kept literal; Code admits only its
	// own closer and treats other delimiters as ordinary source characters.
	for i, t := range tokens {
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			if tokens[top].tag == "code" && t.tag != "code" {
				continue
			}
			if tokens[top].marker == t.marker {
				pairs[top] = i
				stack = stack[:len(stack)-1]
				continue
			}
			for _, j := range stack {
				if tokens[j].marker == t.marker {
					return []Node{n}, nil
				}
			}
		}
		stack = append(stack, i)
	}
	if len(stack) > 0 {
		return []Node{n}, nil
	}
	if len(pairs) == 0 {
		return []Node{n}, nil
	}
	slice := func(lo, hi int) Node {
		v := n
		v.ID = 0
		v.Text = n.Text[lo:hi]
		v.typed = slices.Clone(textOrigins(n)[lo:hi])
		return v
	}
	var build func(int, int, int, int) []Node
	build = func(first, last, lo, hi int) []Node {
		var out []Node
		pos := lo
		for i := first; i < last; i++ {
			j, ok := pairs[i]
			if !ok || j >= last {
				continue
			}
			a, b := tokens[i], tokens[j]
			if a.end == b.start {
				continue
			}
			if pos < a.start {
				out = append(out, slice(pos, a.start))
			}
			var children []Node
			if a.tag == "code" {
				literal := slice(a.end, b.start)
				literal.typed = make([]byte, len(literal.Text))
				children = []Node{literal}
			} else {
				children = build(i+1, j, a.end, b.start)
			}
			tag := a.tag
			if tag == "combined" {
				tag = "strong"
				children = []Node{{Kind: Element, Tag: "em", Children: children}}
			}
			out = append(out, Node{Kind: Element, Tag: tag, Children: children, syntaxSource: n.Text[a.start:b.end], sourceOrigins: slices.Clone(textOrigins(n)[a.start:b.end])})
			pos = b.end
			i = j
		}
		if pos < hi {
			out = append(out, slice(pos, hi))
		}
		return out
	}
	out := build(0, len(tokens), 0, len(n.Text))
	if len(out) == 0 {
		return []Node{n}, nil
	}
	var consumed []syntaxToken
	for i, j := range pairs {
		if tokens[i].end < tokens[j].start {
			consumed = append(consumed, tokens[i], tokens[j])
		}
	}
	return out, consumed
}
func (d *Draft) parseTypedAtCursor() {
	n := findNode(d.state.nodes, d.state.cursor.Node)
	if n == nil || n.Kind != Text {
		return
	}
	loc, _ := d.live().ResolveAnchor(d.state.cursor)
	for _, id := range loc.Ancestors {
		p := findNode(d.state.nodes, id)
		if p.Tag == "pre" || p.Tag == "code" {
			return
		}
	}
	out, consumed := parseTyped(*n)
	if len(out) == 1 && out[0].ID == n.ID {
		return
	}
	// The parser only consumes deliberately active markers in this one leaf.
	// Its semantic cursor is restored by decoded position, excluding delimiters.
	rawLength := len(n.Text)
	before := n.Text[:d.state.cursor.Offset]
	removed := 0
	for _, t := range consumed {
		// A closing run can extend past the cursor into existing active
		// markers. Count its consumed prefix too, keeping the cursor attached
		// to the insertion rather than advancing into following Unicode text.
		removed += max(0, min(t.end, len(before))-t.start)
	}
	list, i := nodeList(&d.state.nodes, n.ID)
	ext, _ := d.extents()
	pos := ext[n.ID].start + max(0, len(before)-removed)
	d.assign(out)
	*list = slices.Replace(*list, i, i+1, out...)
	d.state.cursor = d.anchorAt(pos)
	if len(before) == rawLength && len(out) > 0 {
		last := out[len(out)-1]
		marker := markDelimiter(last.Tag)
		if marker != "" {
			d.state.cursor = Anchor{last.ID, 1}
			d.state.syntax = len(marker) + 1
		}
	}
}

func (d *Draft) typedStructure() error {
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return nil
	}
	b := findNode(d.state.nodes, loc.Block)
	if b == nil || b.Tag != "p" || len(b.Children) == 0 || b.Children[0].Kind != Text {
		return nil
	}
	n := &b.Children[0]
	flags := textOrigins(*n)
	if len(b.Children) == 1 && strings.Trim(n.Text, "-") == "" && len(n.Text) >= 3 {
		for _, f := range flags {
			if f != 1 {
				return nil
			}
		}
		count := len(n.Text)
		b.Tag = "hr"
		b.Children = nil
		b.dashes = count
		d.state.cursor = Anchor{b.ID, 0}
		d.state.syntax = count + 1
		return nil
	}
	end := strings.IndexByte(n.Text, ' ') + 1
	if end <= 0 {
		return nil
	}
	prefixes := map[string]string{"# ": "h1", "## ": "h2", "### ": "h3", "- ": "ul", "+ ": "ul", "* ": "ul", "1. ": "ol"}
	tag := prefixes[n.Text[:end]]
	if tag == "" {
		return nil
	}
	for _, f := range flags[:end] {
		if f != 1 {
			return nil
		}
	}
	n.Text = n.Text[end:]
	n.typed = slices.Clone(flags[end:])
	if d.state.cursor.Node == n.ID {
		d.state.cursor.Offset = max(0, d.state.cursor.Offset-end)
	}
	if tag == "ul" || tag == "ol" {
		return d.convertList(tag)
	}
	b.Tag = tag
	return nil
}

func markDelimiter(tag string) string {
	switch tag {
	case "strong", "b":
		return "**"
	case "em", "i":
		return "*"
	case "u":
		return "__"
	case "s", "strike":
		return "~~"
	case "code":
		return "`"
	}
	return ""
}

func (d *Draft) editSyntax(a Action) error {
	t := d.Target()
	n := findNode(d.state.nodes, t.Anchor.Node)
	if n == nil {
		return ErrAnchor
	}
	if t.Syntax < 0 && n.Tag == "a" && len(n.Children) == 0 {
		value := attributeValue(n.Attributes, "href")
		text := d.newNode(Node{Kind: Text, Text: value})
		n.Children = []Node{text}
		d.state.cursor = Anchor{text.ID, min(len(value), -t.Syntax-1)}
		d.state.syntax = 0
		a.inside = true
		return d.edit(a)
	}
	if n.Kind == Text {
		// Explicitly removing a protecting escape activates only that marker.
		flags := textOrigins(*n)
		off := t.Anchor.Offset
		if off >= len(flags) {
			d.state.syntax = 0
			return d.edit(a)
		}
		if a.Kind == Delete {
			flags[off] = 1
			n.typed = flags
			d.state.syntax = 0
			d.parseTypedAtCursor()
			return nil
		}
		d.state.syntax = 0
		a.skipEscape = true
		return d.edit(a)
	}
	if n.Tag == "hr" {
		count := n.dashes
		if count < 3 {
			count = 3
		}
		text := d.newNode(Node{Kind: Text, Text: strings.Repeat("-", count), typed: []byte(strings.Repeat("\x01", count))})
		n.Tag = "p"
		n.Children = []Node{text}
		n.dashes = 0
		d.state.cursor = Anchor{text.ID, min(count, max(0, t.Syntax-1))}
		d.state.syntax = 0
		if err := d.edit(a); err != nil {
			return err
		}
		return d.typedStructure()
	}
	if n.Tag == "h1" || n.Tag == "h2" || n.Tag == "h3" {
		prefix := strings.Repeat("#", int(n.Tag[1]-'0')) + " "
		text := d.newNode(Node{Kind: Text, Text: prefix, typed: []byte(strings.Repeat("\x01", len(prefix)))})
		n.Tag = "p"
		n.Children = append([]Node{text}, n.Children...)
		d.state.cursor = Anchor{text.ID, min(len(prefix), t.Syntax-1)}
		d.state.syntax = 0
		if err := d.edit(a); err != nil {
			return err
		}
		return d.typedStructure()
	}
	marker := markDelimiter(n.Tag)
	if marker == "" || len(n.Attributes) > 0 {
		return ErrStructure
	}
	if (a.Kind == Type || a.Kind == Insert) && (t.Anchor.Offset == 0 && t.Syntax == 1 || t.Anchor.Offset == 1 && t.Syntax > len(marker)) {
		list, i := nodeList(&d.state.nodes, n.ID)
		if t.Anchor.Offset == 1 {
			i++
		}
		text := d.newNode(Node{Kind: Text})
		*list = slices.Insert(*list, i, text)
		d.state.cursor = Anchor{text.ID, 0}
		d.state.syntax = 0
		return d.edit(a)
	}
	// Unroll just this semantic layer. Nested marks and their metadata survive.
	open := d.newNode(Node{Kind: Text, Text: marker, typed: []byte(strings.Repeat("\x01", len(marker)))})
	close := d.newNode(Node{Kind: Text, Text: marker, typed: []byte(strings.Repeat("\x01", len(marker)))})
	list, i := nodeList(&d.state.nodes, n.ID)
	if n.Tag == "code" {
		demoteCodeSource(n.Children)
	}
	children := append([]Node{open}, n.Children...)
	children = append(children, close)
	*list = slices.Replace(*list, i, i+1, children...)
	id := open.ID
	if t.Anchor.Offset == 1 {
		id = close.ID
	}
	d.state.cursor = Anchor{id, min(len(marker), t.Syntax-1)}
	d.state.syntax = 0
	if err := d.edit(a); err != nil {
		return err
	}
	d.reparseAdjacentText()
	return nil
}

// Extend a just-typed closing run before committing to a delimiter width. This
// makes character events and dead-key/chunk events produce the same grammar.
func (d *Draft) continueClosingRun(r rune) {
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return
	}
	ext, _ := d.extents()
	pos := anchorPosition(d.state.cursor, ext)
	for _, id := range append([]ID{loc.Node.ID}, loc.Ancestors...) {
		n := findNode(d.state.nodes, id)
		if n.syntaxSource == "" || pos != ext[id].end {
			continue
		}
		raw := n.syntaxSource
		if rune(raw[len(raw)-1]) != r {
			return
		}
		text := d.newNode(Node{Kind: Text, Text: raw, typed: slices.Clone(n.sourceOrigins)})
		list, i := nodeList(&d.state.nodes, id)
		(*list)[i] = text
		d.state.cursor = Anchor{text.ID, len(raw)}
		d.state.syntax = 0
		return
	}
}

func (d *Draft) reparseAdjacentText() {
	list, i := nodeList(&d.state.nodes, d.state.cursor.Node)
	if list == nil || (*list)[i].Kind != Text {
		return
	}
	lo, hi := i, i+1
	for lo > 0 && (*list)[lo-1].Kind == Text {
		lo--
	}
	for hi < len(*list) && (*list)[hi].Kind == Text {
		hi++
	}
	if hi-lo <= 1 {
		d.parseTypedAtCursor()
		return
	}
	text := Node{Kind: Text}
	off := 0
	for j := lo; j < hi; j++ {
		n := (*list)[j]
		if j < i {
			off += len(n.Text)
		}
		text.Text += n.Text
		text.typed = append(text.typed, textOrigins(n)...)
	}
	text = d.newNode(text)
	off += d.state.cursor.Offset
	*list = slices.Replace(*list, lo, hi, text)
	d.state.cursor = Anchor{text.ID, off}
	d.parseTypedAtCursor()
}

func (d *Draft) materializeSelectedFallback() {
	s := d.state.selection
	if s == nil || s.Blocks || s.Start.Node != s.End.Node {
		return
	}
	n := findNode(d.state.nodes, s.Start.Node)
	if n == nil || n.Tag != "a" || len(n.Children) > 0 {
		return
	}
	value := attributeValue(n.Attributes, "href")
	text := d.newNode(Node{Kind: Text, Text: value})
	n.Children = []Node{text}
	start, end := 0, len(value)
	if s.startSyntax < 0 {
		start = min(len(value), -s.startSyntax-1)
	}
	if s.endSyntax < 0 {
		end = min(len(value), -s.endSyntax-1)
	}
	s.Start = Anchor{text.ID, start}
	s.End = Anchor{text.ID, end}
	s.startSyntax = 0
	s.endSyntax = 0
	d.state.cursor = s.End
	d.state.syntax = 0
}

func (d *Draft) materializeSelectionSyntax() error {
	s := d.state.selection
	if s == nil || s.Blocks {
		return nil
	}
	for _, id := range []ID{s.Start.Node, s.End.Node} {
		n := findNode(d.state.nodes, id)
		if n == nil || n.Kind == Text {
			continue
		}
		if s.Start.Node == id && s.startSyntax <= 0 && s.End.Node == id && s.endSyntax <= 0 {
			continue
		}
		marker := markDelimiter(n.Tag)
		if n.Tag == "hr" {
			marker = strings.Repeat("-", max(3, n.dashes))
		}
		if n.Tag == "h1" || n.Tag == "h2" || n.Tag == "h3" {
			marker = strings.Repeat("#", int(n.Tag[1]-'0')) + " "
		}
		if marker == "" {
			continue
		}
		if len(n.Attributes) > 0 {
			return ErrStructure
		}
		open := d.newNode(Node{Kind: Text, Text: marker, typed: []byte(strings.Repeat("\x01", len(marker)))})
		close := d.newNode(Node{Kind: Text, Text: marker, typed: []byte(strings.Repeat("\x01", len(marker)))})
		mapEnd := func(a *Anchor, syntax *int) {
			if a.Node != id {
				return
			}
			text := open
			if a.Offset == 1 && markDelimiter(n.Tag) != "" {
				text = close
			}
			a.Node = text.ID
			a.Offset = min(len(marker), max(0, *syntax-1))
			*syntax = 0
		}
		mapEnd(&s.Start, &s.startSyntax)
		mapEnd(&s.End, &s.endSyntax)
		if n.Tag == "code" {
			demoteCodeSource(n.Children)
		}
		if isInlineMark(n.Tag) {
			list, i := nodeList(&d.state.nodes, id)
			children := append([]Node{open}, n.Children...)
			children = append(children, close)
			*list = slices.Replace(*list, i, i+1, children...)
		} else {
			n.Tag = "p"
			n.dashes = 0
			n.Children = append([]Node{open}, n.Children...)
		}
	}
	d.state.cursor = s.End
	d.state.syntax = 0
	return nil
}

func demoteCodeSource(ns []Node) {
	for i := range ns {
		if ns[i].Kind == Text {
			ns[i].typed = make([]byte, len(ns[i].Text))
		}
		demoteCodeSource(ns[i].Children)
	}
}

// Content boundaries include disclosed escapes/mark delimiters. Deleting the
// neighboring editor character must never fall through to a paragraph join.
func (d *Draft) editAdjacentSyntax(a Action) (bool, error) {
	n := findNode(d.state.nodes, d.state.cursor.Node)
	if n == nil || n.Kind != Text {
		return false, nil
	}
	off := d.state.cursor.Offset
	loc, _ := d.live().ResolveAnchor(d.state.cursor)
	if a.Kind == Backspace && !a.skipEscape && off < len(n.Text) && textOrigins(*n)[off] == 0 {
		inlineCode := false
		for _, id := range loc.Ancestors {
			if p := findNode(d.state.nodes, id); p != nil && p.Tag == "code" {
				inlineCode = true
			}
		}
		if disclosedTextEscapes(*n, inlineCode)[off] {
			flags := textOrigins(*n)
			flags[off] = 1
			n.typed = flags
			d.parseTypedAtCursor()
			return true, nil
		}
	}
	ext, _ := d.extents()
	pos := anchorPosition(d.state.cursor, ext)
	for i := len(loc.Ancestors) - 1; i >= 0; i-- {
		p := findNode(d.state.nodes, loc.Ancestors[i])
		marker := markDelimiter(p.Tag)
		if marker == "" {
			continue
		}
		if a.Kind == Backspace && off == 0 && pos == ext[p.ID].start {
			d.state.cursor = Anchor{p.ID, 0}
			d.state.syntax = len(marker) + 1
			return true, d.editSyntax(a)
		}
		if a.Kind == Delete && off == len(n.Text) && pos == ext[p.ID].end {
			d.state.cursor = Anchor{p.ID, 1}
			d.state.syntax = 1
			return true, d.editSyntax(a)
		}
	}
	return false, nil
}
