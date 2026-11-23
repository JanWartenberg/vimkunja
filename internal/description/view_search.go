package description

import (
	"strings"

	"github.com/JanWartenberg/vimkunja/internal/searchpattern"
)

// SearchMatch addresses a readable occurrence without exposing HTML offsets or
// depending on terminal wrapping. Opaque text retains its atomic navigation target.
type SearchMatch struct {
	Target       Target
	Line, Offset int
	End          int
}

type searchLine struct {
	text    string
	targets []Target
	paint   []searchGlyph
}

type searchGlyph struct {
	block   int
	target  Target
	ordinal int
}

func searchableGlyph(g glyph) bool {
	return g.target.Syntax <= 0 && (g.style&Protected == 0 || g.style&OpaqueStyle != 0)
}

func (v *View) searchLines() []searchLine {
	var lines []searchLine
	for blockIndex, b := range v.blocks {
		line := searchLine{}
		ordinals := map[Target]int{}
		var text strings.Builder
		for _, g := range b.friendly {
			key := searchGlyph{blockIndex, g.target, ordinals[g.target]}
			if searchableGlyph(g) {
				ordinals[g.target]++
			}
			if g.hard {
				line.text = text.String()
				line.targets = append(line.targets, g.target)
				endPaint := key
				if len(line.paint) > 0 {
					endPaint = line.paint[len(line.paint)-1]
				}
				line.paint = append(line.paint, endPaint)
				lines = append(lines, line)
				line = searchLine{}
				text = strings.Builder{}
				continue
			}
			// Disclosed syntax and structural markers are not readable content.
			if !searchableGlyph(g) {
				continue
			}
			text.WriteString(g.text)
			for range len(g.text) {
				line.targets = append(line.targets, g.target)
				line.paint = append(line.paint, key)
			}
		}
		end := Target{Anchor: Anchor{b.id, 0}}
		if len(line.targets) > 0 {
			end = line.targets[len(line.targets)-1]
		}
		line.targets = append(line.targets, end)
		endPaint := searchGlyph{blockIndex, end, 0}
		if len(line.paint) > 0 {
			endPaint = line.paint[len(line.paint)-1]
		}
		line.paint = append(line.paint, endPaint)
		line.text = text.String()
		lines = append(lines, line)
	}
	return lines
}

func (v *View) Search(pattern *searchpattern.Pattern) []SearchMatch {
	var matches []SearchMatch
	if pattern == nil {
		return matches
	}
	for i, line := range v.searchLines() {
		for _, r := range pattern.FindAll(line.text) {
			matches = append(matches, SearchMatch{Target: line.targets[r.Start], Line: i, Offset: r.Start, End: r.End})
		}
	}
	return matches
}

// SearchPosition maps the navigation cursor to its logical line and byte offset.
func (v *View) SearchPosition() (int, int) {
	for i, line := range v.searchLines() {
		for offset, target := range line.targets {
			if target == v.target {
				return i, offset
			}
		}
	}
	return 0, -1
}

// SearchHighlight adds semantic paint before wrapping and horizontal clipping.
func (v *View) SearchHighlight(pattern *searchpattern.Pattern, currentLine, currentOffset int) {
	v.searchPaint = nil
	if pattern == nil {
		return
	}
	v.searchPaint = make(map[searchGlyph]Style)
	for i, line := range v.searchLines() {
		for _, r := range pattern.FindAll(line.text) {
			style := SearchStyle
			if i == currentLine && r.Start == currentOffset {
				style |= CurrentSearchStyle
			}
			end := max(r.Start+1, r.End)
			for j := r.Start; j < end && j < len(line.targets); j++ {
				v.searchPaint[line.paint[j]] |= style
			}
		}
	}
}

// Clone preserves navigation/scroll state while sharing immutable document data.
func (v *View) Clone() *View {
	copy := *v
	copy.horizontal = make(map[ID]int, len(v.horizontal))
	for id, x := range v.horizontal {
		copy.horizontal[id] = x
	}
	return &copy
}
