package description

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// Style describes meaning, not terminal escape sequences. The TUI maps these
// roles to its palette; plain terminals keep the structural text cues.
type Style uint16

const (
	Bold Style = 1 << iota
	Italic
	Underline
	Strike
	InlineCode
	Link
	Heading
	Code
	Protected
	OpaqueStyle
	Summary
	SelectedStyle
	TabStyle
)

type Span struct {
	Text  string
	Style Style
}
type Row struct{ Spans []Span }

// Target retains a source boundary through wrapping/disclosure. Syntax is a
// local delimiter position; zero addresses content, never raw HTML bytes.
type Target struct {
	Anchor Anchor
	Syntax int
}
type Frame struct {
	Rows             []Row
	CursorX, CursorY int
	Top              int
	TooNarrow        bool
	Hint             string
	points           []point
	paint            []point
	horizontalLimits map[ID]int
}
type ViewOptions struct {
	ASCII                   bool
	Tabstop                 int
	OpaqueOpen, OpaqueClose string
}
type glyph struct {
	text   string
	style  Style
	target Target
	nav    bool
	hard   bool
	word   int
	link   string
}
type block struct {
	id, unit            ID
	prefix              string
	friendly, disclosed []glyph
	code, opaque, table bool
	content             string
}
type point struct {
	target                                            Target
	row, col, sourceCol, cellWidth, block, word, line int
	link                                              string
}

// View owns only reading/navigation state. The imported Document stays immutable
// and is never serialized, repaired or persisted merely by opening this view.
type View struct {
	doc            *Document
	options        ViewOptions
	blocks         []block
	target         Target
	top, screenRow int
	positioned     bool
	manualScroll   bool
	inserting      bool
	goalColumn     int
	goalSet        bool
	goalSource     bool
	width          int
	horizontal     map[ID]int
}

func NewView(doc *Document, options ...ViewOptions) *View {
	opts := ViewOptions{Tabstop: 4, OpaqueOpen: "[", OpaqueClose: "]"}
	if len(options) > 0 {
		opts = options[0]
		if opts.Tabstop < 1 {
			opts.Tabstop = 4
		}
		if opts.OpaqueOpen == "" {
			opts.OpaqueOpen = "["
		}
		if opts.OpaqueClose == "" {
			opts.OpaqueClose = "]"
		}
	}
	// Options are presentation text and must not inject terminal controls.
	opts.OpaqueOpen = terminalSafeText(opts.OpaqueOpen)
	opts.OpaqueClose = terminalSafeText(opts.OpaqueClose)
	v := &View{doc: doc, options: opts, horizontal: map[ID]int{}}
	if doc.class == Blocked {
		text := doc.RecoveryText()
		if text == "" {
			text = "[Unreadable Description]"
		}
		v.blocks = []block{{opaque: true, content: text, friendly: literalGlyphs(text, OpaqueStyle)}}
	} else {
		v.compile(doc.Snapshot(), "", 0, 0, false)
	}
	v.ensureEmptyTarget()
	f := v.Layout(80, true)
	if len(f.points) > 0 {
		v.target = f.points[0].target
	}
	return v
}
func (v *View) SourceHTML() string { return v.doc.SourceHTML() }
func (v *View) Cursor() Target     { return v.target }
func (v *View) ContentText() string {
	var parts []string
	for _, b := range v.blocks {
		if b.id == 0 && len(b.friendly) > 0 && b.friendly[0].style&Summary != 0 {
			continue
		}
		parts = append(parts, b.content)
	}
	return strings.Join(parts, "\n")
}
func (v *View) SetAnchor(anchor Anchor) error {
	if _, err := v.doc.ResolveAnchor(anchor); err != nil {
		return err
	}
	f := v.Layout(80, true)
	for _, p := range f.points {
		if p.target.Anchor == anchor && p.target.Syntax == 0 {
			v.target = p.target
			return nil
		}
	}
	for _, p := range f.points {
		if p.target.Anchor == anchor {
			v.target = p.target
			return nil
		}
	}
	return fmt.Errorf("%w: boundary is not a projected grapheme", ErrAnchor)
}
func literalGlyphs(text string, style Style) []glyph {
	var out []glyph
	it := uniseg.NewGraphemes(terminalSafeText(text))
	for it.Next() {
		s := it.Str()
		out = append(out, glyph{text: s, style: style, hard: s == "\n"})
	}
	return out
}
func wordClass(text string) int {
	for _, r := range text {
		if unicode.IsSpace(r) {
			return 0
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' {
			return 1
		}
		return 2
	}
	return 0
}
func (v *View) textGlyphs(text string, id ID, style Style, link string) []glyph {
	var out []glyph
	it := uniseg.NewGraphemes(text)
	for it.Next() {
		start, _ := it.Positions()
		s := it.Str()
		hard := s == "\n" || s == "\r\n"
		out = append(out, glyph{text: terminalSafeText(s), style: style, target: Target{Anchor: Anchor{id, start}}, nav: true, hard: hard, word: wordClass(s), link: link})
	}
	if v.inserting || len(out) == 0 || strings.HasSuffix(text, "\n") {
		out = append(out, glyph{text: "", style: style, target: Target{Anchor: Anchor{id, len(text)}}, nav: true})
	}
	return out
}
func (v *View) delimiter(text string, id ID, end bool) []glyph {
	out := literalGlyphs(text, 0)
	offset := 0
	if end {
		offset = 1
	}
	for i := range out {
		out[i].nav = true
		out[i].target = Target{Anchor: Anchor{id, offset}, Syntax: i + 1}
		out[i].word = -1
	}
	if v.inserting {
		out = append(out, glyph{nav: true, target: Target{Anchor: Anchor{id, offset}, Syntax: len(out) + 1}})
	}
	return out
}
func (v *View) inline(nodes []Node, style Style, disclose bool, link string, layers map[string]int) []glyph {
	var out []glyph
	for _, n := range nodes {
		if n.Kind == Text {
			text := v.textGlyphs(n.Text, n.ID, style, link)
			if disclose && style&Code == 0 {
				var escaped []glyph
				for _, g := range text {
					needs := strings.ContainsAny(g.text, editorDisclosedEscapable)
					if style&InlineCode != 0 {
						needs = g.text == "`" || g.text == "\\"
					}
					if off := g.target.Anchor.Offset; off < len(n.typed) && n.typed[off] != 0 {
						needs = false
					}
					if needs {
						escape := g
						escape.text = "\\"
						escape.target.Syntax = 1
						escape.word = -1
						escaped = append(escaped, escape)
					}
					escaped = append(escaped, g)
				}
				text = escaped
			}
			out = append(out, text...)
			continue
		}
		if n.Kind == Opaque {
			atom := literalGlyphs(v.options.OpaqueOpen+n.Preview+v.options.OpaqueClose, style|OpaqueStyle|Protected)
			for i := range atom {
				atom[i].target = Target{Anchor: Anchor{n.ID, 0}}
				if !v.inserting && i == len(literalGlyphs(v.options.OpaqueOpen, 0)) {
					atom[i].nav = true
					atom[i].word = 3
				}
			}
			if v.inserting {
				out = append(out, glyph{nav: true, target: Target{Anchor: Anchor{n.ID, 0}}})
			}
			out = append(out, atom...)
			if v.inserting {
				out = append(out, glyph{nav: true, target: Target{Anchor: Anchor{n.ID, 1}}})
			}
			continue
		}
		if n.Tag == "br" {
			out = append(out, glyph{hard: true, text: "\n", style: Protected, target: Target{Anchor: Anchor{n.ID, 0}}, nav: true})
			continue
		}
		nextStyle := style
		marker := ""
		family := ""
		switch n.Tag {
		case "strong", "b":
			nextStyle |= Bold
			marker = "**"
			family = "bold"
		case "em", "i":
			nextStyle |= Italic
			marker = "*"
			family = "italic"
		case "u":
			nextStyle |= Underline
			marker = "__"
		case "s":
			nextStyle |= Strike
			marker = "~~"
		case "code":
			if style&Code == 0 {
				nextStyle |= InlineCode
				marker = "`"
			}
		case "a":
			nextStyle |= Link | Underline
			link = terminalSafeText(attributeValue(n.Attributes, "href"))
		}
		if disclose && marker != "" {
			if family != "" && layers[family] > 0 {
				out = append(out, glyph{text: "|" + n.Tag + "|", style: Protected})
			}
			out = append(out, v.delimiter(marker, n.ID, false)...)
		}
		if family != "" {
			layers[family]++
		}
		children := v.inline(n.Children, nextStyle, disclose, link, layers)
		if n.Tag == "a" && len(children) == 0 {
			children = v.textGlyphs(link, n.ID, nextStyle, link)
			for i := range children {
				children[i].target = Target{Anchor: Anchor{n.ID, 0}, Syntax: -children[i].target.Anchor.Offset - 1}
			}
		}
		out = append(out, children...)
		if family != "" {
			layers[family]--
		}
		if disclose && marker != "" {
			out = append(out, v.delimiter(marker, n.ID, true)...)
		}
	}
	if len(out) > 0 && out[len(out)-1].hard {
		last := out[len(out)-1]
		last.hard = false
		last.text = ""
		last.target.Anchor.Offset = 1
		last.nav = true
		out = append(out, last)
	}
	return out
}
func glyphText(gs []glyph) string {
	var b strings.Builder
	for _, g := range gs {
		b.WriteString(g.text)
	}
	return b.String()
}
func (v *View) add(n Node, prefix string, unit ID, code bool) {
	if unit == 0 {
		unit = n.ID
	}
	b := block{id: n.ID, unit: unit, prefix: prefix, code: code}
	if code {
		b.friendly = v.inline(n.Children, Code, false, "", map[string]int{})
		b.disclosed = b.friendly
	} else {
		style := Style(0)
		if strings.HasPrefix(n.Tag, "h") {
			style = Heading | Bold
		}
		b.friendly = v.inline(n.Children, style, false, "", map[string]int{})
		b.disclosed = v.inline(n.Children, style, true, "", map[string]int{})
		if len(b.friendly) == 0 {
			b.friendly = []glyph{{nav: true, target: Target{Anchor: Anchor{n.ID, 0}}}}
			b.disclosed = b.friendly
		}
		if n.Tag == "h1" || n.Tag == "h2" || n.Tag == "h3" {
			count := int(n.Tag[1] - '0')
			b.disclosed = append(v.delimiter(strings.Repeat("#", count)+" ", n.ID, false), b.disclosed...)
			b.friendly = append(literalGlyphs(strings.Repeat("#", count)+" ", Protected|Heading), b.friendly...)
		}
	}
	b.content = contentNodes(n.Children)
	if len(b.friendly) == 0 {
		b.friendly = []glyph{{nav: true, target: Target{Anchor: Anchor{n.ID, 0}}}}
		b.disclosed = b.friendly
	}
	v.blocks = append(v.blocks, b)
}
func (v *View) compile(nodes []Node, prefix string, depth int, unit ID, insideCheckbox bool) {
	for _, n := range nodes {
		if n.Kind == Opaque {
			content := n.Preview
			if !insideCheckbox && v.opaqueTag(n.ID) != "table" && n.Preview != "[Active content]" {
				if preview, ok := v.opaqueChecklistPreview(n.ID); ok {
					n.Preview = preview
				}
			}
			isTable := false
			if n.Preview != "[Active content]" {
				if text, ok := v.tablePreview(n.ID); ok {
					n.Preview = text
					isTable = true
				}
			}
			gs := v.inline([]Node{n}, 0, false, "", map[string]int{})
			u := unit
			if u == 0 {
				u = n.ID
			}
			v.blocks = append(v.blocks, block{id: n.ID, unit: u, prefix: prefix, friendly: gs, disclosed: gs, opaque: true, table: isTable, content: content})
			continue
		}
		switch n.Tag {
		case "blockquote":
			q := "│  "
			if v.options.ASCII {
				q = "|  "
			}
			u := unit
			if u == 0 {
				u = n.ID
			}
			if len(n.Children) == 0 {
				v.add(n, prefix+q, u, false)
			} else {
				v.compile(n.Children, prefix+q, depth, u, insideCheckbox)
			}
		case "ul", "ol":
			if len(n.Children) == 0 {
				v.add(n, prefix, n.ID, false)
				continue
			}
			ordinal := 1
			reversed := false
			for _, a := range n.Attributes {
				if a.Name == "reversed" {
					reversed = true
				}
			}
			if reversed {
				ordinal = 0
				for _, child := range n.Children {
					if child.Tag == "li" || (child.Kind == Opaque && v.opaqueTag(child.ID) == "li") {
						ordinal++
					}
				}
			}
			if value, err := strconv.Atoi(attributeValue(n.Attributes, "start")); err == nil {
				ordinal = value
			}
			for _, li := range n.Children {
				if li.Kind == Opaque {
					childPrefix := prefix
					if v.opaqueTag(li.ID) == "li" {
						attrs := v.opaqueEntryAttributes(li.ID)
						marker := "- "
						if n.Tag == "ol" {
							if value, err := strconv.Atoi(attributeValue(attrs, "value")); err == nil {
								ordinal = value
							}
							marker = strconv.Itoa(ordinal) + ". "
							if reversed {
								ordinal--
							} else {
								ordinal++
							}
						}
						if attributeValue(attrs, "data-type") == "taskItem" {
							marker = "☐ "
							checked := strings.EqualFold(attributeValue(attrs, "data-checked"), "true")
							if checked {
								marker = "☑ "
							}
							if v.options.ASCII {
								marker = "[ ] "
								if checked {
									marker = "[x] "
								}
							}
						}
						childPrefix += strings.Repeat("  ", depth) + marker
					}
					v.compile([]Node{li}, childPrefix, depth, unit, insideCheckbox)
					continue
				}
				marker := "- "
				if n.Tag == "ol" {
					if value, err := strconv.Atoi(attributeValue(li.Attributes, "value")); err == nil {
						ordinal = value
					}
					marker = strconv.Itoa(ordinal) + ". "
					if reversed {
						ordinal--
					} else {
						ordinal++
					}
				}
				if attributeValue(li.Attributes, "data-type") == "taskItem" {
					marker = "☐ "
					if strings.EqualFold(attributeValue(li.Attributes, "data-checked"), "true") {
						marker = "☑ "
					}
					if v.options.ASCII {
						marker = "[ ] "
						if strings.EqualFold(attributeValue(li.Attributes, "data-checked"), "true") {
							marker = "[x] "
						}
					}
				}
				liPrefix := prefix + strings.Repeat("  ", depth) + marker
				u := unit
				if u == 0 {
					u = li.ID
				}
				if len(li.Children) == 0 {
					v.add(li, liPrefix, u, false)
					continue
				}
				// Inline list content is one local paragraph; block children retain their
				// own j/k units while the enclosing entry remains one paragraph unit.
				var inline []Node
				flush := func() {
					if len(inline) > 0 {
						copy := li
						copy.Children = inline
						v.add(copy, liPrefix, u, false)
						inline = nil
					}
				}
				first := true
				for _, child := range checkboxContent(li) {
					if child.Kind != Opaque && !isBlock(child.Tag) {
						inline = append(inline, child)
						continue
					}
					flush()
					childPrefix := liPrefix
					if !first {
						childPrefix = prefix + strings.Repeat("  ", depth) + strings.Repeat(" ", ansi.StringWidth(marker))
					}
					if child.Tag == "ul" || child.Tag == "ol" {
						v.compile([]Node{child}, prefix, depth+1, unit, insideCheckbox || attributeValue(n.Attributes, "data-type") == "taskList")
					} else {
						v.compile([]Node{child}, childPrefix, depth, u, insideCheckbox || attributeValue(n.Attributes, "data-type") == "taskList")
					}
					first = false
				}
				flush()
			}
			if attributeValue(n.Attributes, "data-type") == "taskList" && !insideCheckbox {
				done, total := v.checkCount(n.Children)
				if total > 0 {
					v.blocks = append(v.blocks, block{prefix: prefix, friendly: literalGlyphs(fmt.Sprintf("Checklist progress: %d/%d", done, total), Summary|Protected)})
				}
			}
			if len(n.Children) == 0 {
				v.add(n, prefix, unit, false)
			}
		case "div":
			v.compile(n.Children, prefix, depth, unit, insideCheckbox)
		case "pre":
			v.add(n, prefix, unit, true)
		case "hr":
			u := unit
			if u == 0 {
				u = n.ID
			}
			rule := "───"
			if v.options.ASCII {
				rule = "---"
			}
			v.blocks = append(v.blocks, block{id: n.ID, unit: u, prefix: prefix, friendly: []glyph{{text: rule, style: Protected, target: Target{Anchor: Anchor{n.ID, 0}}, nav: true, word: 3}}, disclosed: v.delimiter(strings.Repeat("-", max(3, n.dashes)), n.ID, false), content: ""})
		default:
			v.add(n, prefix, unit, false)
		}
	}
}
func (v *View) checkCount(nodes []Node) (done, total int) {
	for _, n := range nodes {
		if n.Kind == Opaque {
			d, t := v.opaqueCheckboxCount(n.ID)
			done += d
			total += t
			continue
		}
		if n.Tag == "li" && attributeValue(n.Attributes, "data-type") == "taskItem" {
			total++
			if strings.EqualFold(attributeValue(n.Attributes, "data-checked"), "true") {
				done++
			}
		}
		d, t := v.checkCount(n.Children)
		done += d
		total += t
	}
	return
}

func contentNodes(nodes []Node) string {
	var out strings.Builder
	for _, n := range nodes {
		switch {
		case n.Kind == Text:
			out.WriteString(terminalSafeText(n.Text))
		case n.Kind == Opaque:
			out.WriteString(n.Preview)
		case n.Tag == "br":
			out.WriteByte('\n')
		case n.Tag == "a" && len(n.Children) == 0:
			out.WriteString(terminalSafeText(attributeValue(n.Attributes, "href")))
		default:
			out.WriteString(contentNodes(n.Children))
		}
	}
	return out.String()
}

func (v *View) ensureEmptyTarget() {
	if len(v.blocks) == 0 && v.doc.class != Blocked {
		v.blocks = []block{{friendly: []glyph{{nav: true, target: Target{}}}}}
		v.target = Target{}
	}
}
