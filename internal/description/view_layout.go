package description

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

func appendSpan(row *Row, g glyph) {
	if g.text == "" {
		return
	}
	if len(row.Spans) > 0 && row.Spans[len(row.Spans)-1].Style == g.style && len(row.Spans[len(row.Spans)-1].Text) < 256 {
		row.Spans[len(row.Spans)-1].Text += g.text
	} else {
		row.Spans = append(row.Spans, Span{g.text, g.style})
	}
}
func (v *View) activeBlock() int {
	for i, b := range v.blocks {
		if b.id != 0 && b.id == v.target.Anchor.Node {
			return i
		}
		for _, g := range b.disclosed {
			if g.nav && g.target == v.target {
				return i
			}
		}
	}
	return 0
}
func rowWidth(row Row) int {
	width := 0
	for _, s := range row.Spans {
		width += ansi.StringWidth(s.Text)
	}
	return width
}

// Slice terminal cells without cutting a grapheme or letting a tab depend on
// the terminal's own tab stops. Padding at a clipped wide glyph keeps alignment.
func sliceRow(row Row, start, width int) Row {
	out := Row{}
	column := 0
	written := 0
	for _, s := range row.Spans {
		it := uniseg.NewGraphemes(s.Text)
		for it.Next() {
			text := it.Str()
			w := ansi.StringWidth(text)
			end := column + w
			if end > start && column < start+width {
				if column >= start && end <= start+width {
					gap := column - start - written
					if gap > 0 {
						appendSpan(&out, glyph{text: strings.Repeat(" ", gap), style: s.Style})
						written += gap
					}
					appendSpan(&out, glyph{text: text, style: s.Style})
					written += w
				}
			}
			column = end
		}
	}
	return out
}
func (v *View) Layout(width int, focused bool) Frame {
	f := Frame{CursorX: -1, CursorY: -1, Hint: "Description is read-only; preserved", horizontalLimits: map[ID]int{}}
	active := v.activeBlock()
	if v.doc.class == Blocked {
		f.Hint = "Error: Malformed HTML"
	}
	for bi, b := range v.blocks {
		gs := b.friendly
		if focused && bi == active && len(b.disclosed) > 0 {
			gs = b.disclosed
		}
		prefix := b.prefix
		if b.code {
			prefix += "│ "
			if v.options.ASCII {
				prefix = b.prefix + "| "
			}
		}
		prefixWidth := ansi.StringWidth(prefix)
		capacity := width - prefixWidth
		horizontal := b.code || b.table
		if capacity < 1 || (b.code && focused && bi == active && capacity < 3) {
			f.TooNarrow = f.TooNarrow || bi == active
			for _, g := range gs {
				if g.nav {
					f.points = append(f.points, point{target: g.target, row: len(f.Rows), col: 0, block: bi, word: g.word})
					break
				}
			}
			f.Rows = append(f.Rows, Row{Spans: []Span{{"Pane too narrow", Protected}}})
			continue
		}
		if b.code && focused && bi == active {
			f.Rows = append(f.Rows, Row{Spans: []Span{{prefix + "```", Code | Protected}}})
		}
		firstRow := len(f.Rows)
		firstPoint := len(f.points)
		firstPaint := len(f.paint)
		row := Row{}
		col := 0
		fresh := true
		contentCol := 0
		logicalLine := 0
		atomContinuation := false
		begin := func(continuation bool) {
			row = Row{}
			p := prefix
			if continuation && !horizontal {
				if b.opaque || atomContinuation {
					p += "│ "
					if v.options.ASCII {
						p = prefix + "| "
					}
				} else {
					if strings.Contains(p, "│  ") {
						p = strings.ReplaceAll(p, "│  ", "│↪ ")
					} else {
						p += "↪ "
					}
					if v.options.ASCII {
						if strings.Contains(prefix, "|  ") {
							p = strings.ReplaceAll(prefix, "|  ", "|> ")
						} else {
							p = prefix + "> "
						}
					}
				}
			}
			style := Protected
			if b.code {
				style |= Code
			}
			appendSpan(&row, glyph{text: p, style: style})
			col = ansi.StringWidth(p)
			contentCol = 0
			fresh = true
		}
		addPoint := func(g glyph) {
			w := 1
			it := uniseg.NewGraphemes(g.text)
			if it.Next() {
				w = max(1, ansi.StringWidth(it.Str()))
			}
			f.points = append(f.points, point{target: g.target, row: len(f.Rows), col: col, sourceCol: contentCol, cellWidth: w, block: bi, word: g.word, line: logicalLine, link: g.link})
		}
		begin(false)
		for gi, g := range gs {
			if g.hard {
				if g.nav && (!b.code || fresh) {
					addPoint(g)
					if !horizontal && fresh && (g.style&OpaqueStyle != 0 || !focused || bi != active) {
						appendSpan(&row, glyph{text: " ", style: 0})
						col++
					}
				}
				if !horizontal && !b.opaque && g.style&OpaqueStyle == 0 && focused && bi == active {
					marker := "↵"
					if v.options.ASCII {
						marker = ">"
					}
					appendSpan(&row, glyph{text: marker, style: Protected})
				}
				f.Rows = append(f.Rows, row)
				if b.code {
					logicalLine++
				}
				atomContinuation = g.style&OpaqueStyle != 0
				begin(b.opaque || atomContinuation)
				continue
			}
			text := g.text
			if text == "\t" {
				cells := v.options.Tabstop - contentCol%v.options.Tabstop
				text = strings.Repeat(" ", cells)
				if b.code {
					text = ">" + strings.Repeat("~", cells-1)
					g.style |= TabStyle
				}
			}
			gw := ansi.StringWidth(text)
			availableWidth := width
			// Keep the active hard-break cue and its cursor target visible even
			// when the preceding text would exactly fill the terminal row.
			if focused && bi == active && !horizontal && !b.opaque && gi+1 < len(gs) && gs[gi+1].hard && gs[gi+1].style&OpaqueStyle == 0 {
				availableWidth--
			}
			if !horizontal && col+gw > availableWidth && !fresh {
				f.Rows = append(f.Rows, row)
				atomContinuation = g.style&OpaqueStyle != 0
				begin(true)
			}
			if !horizontal && col+gw > availableWidth {
				f.TooNarrow = f.TooNarrow || bi == active
				appendSpan(&row, glyph{text: "Pane too narrow", style: Protected})
				break
			}
			if g.nav {
				addPoint(g)
			}
			if g.target.Anchor.Node != 0 {
				f.paint = append(f.paint, point{target: g.target, row: len(f.Rows), col: col, cellWidth: gw})
			}
			g.text = text
			appendSpan(&row, g)
			col += gw
			contentCol += gw
			fresh = false
		}
		f.Rows = append(f.Rows, row)
		if horizontal {
			maximum := 0
			for i := firstRow; i < len(f.Rows); i++ {
				maximum = max(maximum, rowWidth(f.Rows[i])-prefixWidth)
			}
			overflow := maximum > capacity
			offset := min(v.horizontal[b.id], max(0, maximum-capacity+1))
			limit := 0
			if overflow {
				limit = maximum - capacity + 1
			}
			f.horizontalLimits[b.id] = limit
			available := capacity
			if overflow {
				available--
			}
			for i := firstRow; i < len(f.Rows); i++ {
				original := f.Rows[i]
				r := Row{}
				style := Protected
				if b.code {
					style |= Code
				}
				appendSpan(&r, glyph{text: prefix, style: style})
				clipped := sliceRow(original, prefixWidth+offset, max(0, available))
				r.Spans = append(r.Spans, clipped.Spans...)
				if overflow {
					padding := max(0, width-1-rowWidth(r))
					appendSpan(&r, glyph{text: strings.Repeat(" ", padding) + "↔", style: style})
					if v.options.ASCII {
						last := &r.Spans[len(r.Spans)-1]
						last.Text = strings.TrimSuffix(last.Text, "↔") + ">"
					}
				}
				f.Rows[i] = r
			}
			for i := firstPoint; i < len(f.points); i++ {
				f.points[i].col -= offset
				if b.table {
					f.points[i].col = max(prefixWidth, min(f.points[i].col, width-1))
				}
			}
			for i := firstPaint; i < len(f.paint); i++ {
				f.paint[i].col -= offset
			}
			if b.code && overflow {
				track := strings.Repeat("─", max(1, capacity))
				if v.options.ASCII {
					track = strings.Repeat("-", max(1, capacity))
				}
				position := 0
				if maximum > capacity {
					position = offset * max(0, capacity-1) / (maximum - capacity + 1)
				}
				runes := []rune(track)
				thumb := '█'
				if v.options.ASCII {
					thumb = '#'
				}
				runes[min(position, len(runes)-1)] = thumb
				f.Rows = append(f.Rows, Row{Spans: []Span{{prefix + string(runes), Code | Protected}}})
			}
		}
		if b.code && focused && bi == active {
			f.Rows = append(f.Rows, Row{Spans: []Span{{prefix + "```", Code | Protected}}})
		}
	}
	if len(f.Rows) == 0 {
		f.Rows = []Row{{}}
	}
	if index, found := findPoint(f.points, v.target); found {
		p := f.points[index]
		f.CursorX, f.CursorY = p.col, p.row
		if p.link != "" {
			f.Hint = "Link → " + p.link + " · read-only"
		}
		b := v.blocks[p.block]
		if b.opaque {
			f.Hint = "Unsupported content is read-only; preserved"
		}
		if b.table {
			f.Hint = "Table is read-only; preserved"
		}
	}

	if f.TooNarrow {
		f.Hint = "Pane too narrow"
	}
	for i, row := range f.Rows {
		if rowWidth(row) > max(0, width) {
			f.Rows[i] = sliceRow(row, 0, max(0, width))
		}
	}
	f.Top = v.top
	return f
}
