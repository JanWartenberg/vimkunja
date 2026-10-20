package description

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
)

func (v *View) Hint() string {
	width := v.width
	if width < 1 {
		width = 80
	}
	return v.Layout(width, true).Hint
}
func findPoint(points []point, target Target) (int, bool) {
	for i, p := range points {
		if p.target == target {
			return i, true
		}
	}
	for i, p := range points {
		if p.target.Anchor == target.Anchor {
			return i, true
		}
	}
	return 0, false
}
func pointIndex(points []point, target Target) int {
	index, _ := findPoint(points, target)
	return index
}

func closest(points []point, indices []int, column int, source bool) int {
	position := func(p point) int {
		if source {
			return p.sourceCol
		}
		return p.col
	}
	best := indices[0]
	distance := abs(position(points[best]) - column)
	for _, i := range indices {
		d := abs(position(points[i]) - column)
		if d < distance {
			best = i
			distance = d
		}
	}
	return best
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func (v *View) Viewport(width, height int, focused bool) Frame {
	f := v.Layout(width, focused)
	height = max(1, height)
	if f.TooNarrow {
		v.width = width
		f.Top = max(0, min(v.top, len(f.Rows)-height))
		return f
	}
	if focused && v.width != width {
		f = v.revealCodeCursor(width, f)
	}
	v.width = width
	if focused && f.CursorY >= 0 && !v.manualScroll {
		if v.positioned {
			v.top = f.CursorY - v.screenRow
		}
		v.top = max(0, min(v.top, len(f.Rows)-height))
		if f.CursorY < v.top {
			v.top = f.CursorY
		}
		if f.CursorY >= v.top+height {
			v.top = f.CursorY - height + 1
		}
		// Keep an adjacent checklist summary visible without giving it a cursor.
		bottom := f.CursorY
		for bottom+1 < len(f.Rows) && bottom-f.CursorY+1 < height {
			summary := false
			for _, span := range f.Rows[bottom+1].Spans {
				summary = summary || span.Style&Summary != 0
			}
			if !summary {
				break
			}
			bottom++
		}
		if bottom >= v.top+height {
			v.top = bottom - height + 1
		}
		v.screenRow = f.CursorY - v.top
		v.positioned = true
	} else {
		v.top = max(0, min(v.top, len(f.Rows)-height))
	}
	f.Top = v.top
	return f
}

// FollowCursor resumes normal cursor tracking after an edit or entering Insert.
func (v *View) FollowCursor() { v.manualScroll = false }

// Scroll changes only the viewport; the document cursor remains anchored.
func (v *View) Scroll(delta, width, height int) Frame {
	f := v.Layout(width, true)
	if !f.TooNarrow {
		v.top = max(0, min(v.top+delta, len(f.Rows)-max(1, height)))
		v.screenRow = f.CursorY - v.top
		v.positioned = false
		v.manualScroll = true
	}
	f.Top = v.top
	return f
}
func (v *View) Move(motion string, count, width, height int) bool {
	count = max(1, min(count, 100000))
	f := v.Layout(width, true)
	if f.TooNarrow {
		v.width = width
		return true
	}
	if len(f.points) == 0 {
		delta := 0
		switch motion {
		case "j", "down":
			delta = count
		case "k", "up":
			delta = -count
		case "ctrl+d", "pgdown", "pagedown":
			delta = count * max(1, height)
		case "ctrl+u", "pgup", "pageup":
			delta = -count * max(1, height)
		}
		if delta != 0 {
			v.Scroll(delta, width, height)
		}
		return true
	}
	v.width = width
	v.manualScroll = false
	points := f.points
	index := pointIndex(points, v.target)

	// Keep logical source-line motion independent of wrapping and protected rows.
	vertical := func(direction int, visual bool) {
		p := points[index]
		sourceGoal := v.blocks[p.block].code && !visual
		if !v.goalSet || v.goalSource != sourceGoal {
			v.goalSource = sourceGoal
			v.goalColumn = p.col
			if sourceGoal {
				v.goalColumn = p.sourceCol
			}
			v.goalSet = true
		}
		groups := [][]int{}
		lastBlock, lastLine := -1, -1
		for i, q := range points {
			line := 0
			if visual {
				line = q.row
			} else if v.blocks[q.block].code {
				line = q.line
			}
			if q.block != lastBlock || line != lastLine {
				groups = append(groups, nil)
				lastBlock, lastLine = q.block, line
			}
			groups[len(groups)-1] = append(groups[len(groups)-1], i)
		}
		current := 0
		for g, indices := range groups {
			for _, i := range indices {
				if i == index {
					current = g
				}
			}
		}
		next := max(0, min(current+direction, len(groups)-1))
		index = closest(points, groups[next], v.goalColumn, sourceGoal && v.blocks[points[groups[next][0]].block].code)
	}
	if motion != "j" && motion != "k" && motion != "up" && motion != "down" && motion != "gj" && motion != "gk" {
		v.goalSet = false
	}
	switch motion {
	case "zh", "zl":
		b := v.blocks[points[index].block]
		if b.code || b.table {
			delta := count
			if motion == "zh" {
				delta = -delta
			}
			limit := f.horizontalLimits[b.id]
			v.horizontal[b.id] = max(0, min(min(v.horizontal[b.id], limit)+delta, limit))
		}
		return true
	case "h", "left", "l", "right":
		direction := 1
		if motion == "h" || motion == "left" {
			direction = -1
		}
		for n := 0; n < count; n++ {
			next := index + direction
			if next < 0 || next >= len(points) || points[next].block != points[index].block || points[next].line != points[index].line {
				break
			}
			index = next
		}
	case "j", "down", "k", "up", "gj", "gk", "ctrl+d", "ctrl+u", "pgdown", "pagedown", "pgup", "pageup":
		direction := 1
		if motion == "k" || motion == "up" || motion == "gk" || motion == "ctrl+u" || motion == "pgup" || motion == "pageup" {
			direction = -1
		}
		if strings.HasPrefix(motion, "ctrl+") || strings.HasPrefix(motion, "pg") || strings.HasPrefix(motion, "page") {
			count = min(count*max(1, height), len(points))
		}
		vertical(direction*count, motion == "gj" || motion == "gk" || strings.HasPrefix(motion, "ctrl+") || strings.HasPrefix(motion, "pg") || strings.HasPrefix(motion, "page"))
	case "}", "ä", "{", "ö":
		direction := 1
		if motion == "{" || motion == "ö" {
			direction = -1
		}
		for n := 0; n < count; n++ {
			unit := v.blocks[points[index].block].unit
			j := index
			for j >= 0 && j < len(points) && v.blocks[points[j].block].unit == unit {
				j += direction
			}
			if j < 0 || j >= len(points) {
				break
			}
			newUnit := v.blocks[points[j].block].unit
			for j > 0 && v.blocks[points[j-1].block].unit == newUnit {
				j--
			}
			index = j
		}
	case "gg":
		index = 0
		if count > 1 {
			vertical(count-1, false)
		}
	case "G":
		index = len(points) - 1
	case "0", "home", "^", "_", "$", "end":
		if motion == "$" || motion == "end" || motion == "_" {
			vertical(count-1, false)
		}
		p := points[index]
		indices := []int{}
		for i, q := range points {
			if q.block == p.block && q.line == p.line {
				indices = append(indices, i)
			}
		}
		if len(indices) > 0 {
			index = indices[0]
			if motion == "$" || motion == "end" {
				index = indices[len(indices)-1]
			} else if motion == "^" || motion == "_" {
				for _, i := range indices {
					if points[i].word != 0 {
						index = i
						break
					}
				}
			}
		}
	case "w", "W", "b", "B", "e", "E", "ge", "gE":
		index = wordMotion(points, index, motion, count)
	default:
		return false
	}
	v.target = points[index].target
	// A move should scroll only at a viewport edge. Disclosure may shift rows;
	// resizing later preserves the resulting screen row through Viewport.
	after := v.Layout(width, true)
	if motion == "G" && len(after.points) > 0 {
		v.target = after.points[len(after.points)-1].target
		after = v.Layout(width, true)
	}
	if motion == "gg" && len(after.points) > 0 {
		for _, p := range after.points {
			if p.block == points[index].block && p.line == points[index].line {
				v.target = p.target
				break
			}
		}
		after = v.Layout(width, true)
	}
	after = v.revealCodeCursor(width, after)
	v.top = max(0, min(v.top, len(after.Rows)-max(1, height)))
	if after.CursorY < v.top {
		v.top = after.CursorY
	}
	if after.CursorY >= v.top+max(1, height) {
		v.top = after.CursorY - max(1, height) + 1
	}
	v.screenRow = after.CursorY - v.top
	v.positioned = true
	return true
}

// Word motion follows semantic text across disclosed marks. Delimiters stay
// reachable by h/l without splitting an imported word at its formatting edge.
func wordMotion(points []point, index int, motion string, count int) int {
	index, _ = wordMotionResult(points, index, motion, count)
	return index
}

// WordOperatorEnd shares semantic word navigation with normal motions. Include
// reports whether the final grapheme belongs to the operator's half-open range.
func (v *View) WordOperatorEnd(motion string, count, width int) (end Target, include bool) {
	f := v.Layout(max(1, width), true)
	if len(f.points) == 0 {
		return v.target, false
	}
	index, after := wordMotionResult(f.points, pointIndex(f.points, v.target), motion, max(1, min(count, 100000)))
	return f.points[index].target, after || motion == "e" || motion == "E"
}

func wordMotionResult(points []point, index int, motion string, count int) (int, bool) {
	after := false
	var indices []int
	current := 0
	for i, p := range points {
		if p.block == points[index].block && p.word >= 0 {
			if i <= index {
				current = len(indices)
			}
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 {
		return index, false
	}
	current = min(current, len(indices)-1)
	big := motion == "W" || motion == "B" || motion == "E" || motion == "gE"
	class := func(i int) int {
		c := points[indices[i]].word
		if big && c > 0 && c != 3 {
			return 1
		}
		return c
	}
	last := len(indices) - 1
	for n := 0; n < min(count, len(indices)); n++ {
		previous := current
		switch motion {
		case "b", "B":
			current = max(0, current-1)
			for current > 0 && class(current) == 0 {
				current--
			}
			c := class(current)
			for current > 0 && c != 3 && class(current-1) == c {
				current--
			}
		case "ge", "gE":
			c := class(current)
			for current > 0 && c != 3 && class(current-1) == c {
				current--
			}
			current = max(0, current-1)
			for current > 0 && class(current) == 0 {
				current--
			}
		case "w", "W":
			c := class(current)
			if c == 3 {
				current = min(last, current+1)
			} else {
				for current < last && class(current) == c {
					current++
				}
			}
			// At the last point there may be no next word start. An operator
			// must consume that final grapheme rather than leave it behind.
			after = current == last && (class(current) == c || class(current) == 0)
			for current < last && class(current) == 0 {
				current++
			}
			after = after || current == last && class(current) == 0
		case "e", "E":
			current = min(last, current+1)
			for current < last && class(current) == 0 {
				current++
			}
			c := class(current)
			for current < last && c != 3 && class(current+1) == c {
				current++
			}
		}
		if current == previous {
			break
		}
	}
	return indices[current], after
}

// Resizing and cursor motion reveal Code using its document column. Explicit
// horizontal scroll at an unchanged width remains independent of the cursor.
func (v *View) revealCodeCursor(width int, f Frame) Frame {
	if len(v.blocks) == 0 || f.TooNarrow || f.CursorY < 0 {
		return f
	}
	b := v.blocks[v.activeBlock()]
	if !b.code {
		return f
	}
	prefix := ansi.StringWidth(b.prefix) + 2
	offset := min(v.horizontal[b.id], f.horizontalLimits[b.id])
	if f.CursorX < prefix {
		offset = max(0, offset-(prefix-f.CursorX))
	}
	right := width
	if f.horizontalLimits[b.id] > 0 {
		right--
	}
	cellWidth := 1
	if i, ok := findPoint(f.points, v.target); ok {
		cellWidth = max(1, f.points[i].cellWidth)
	}
	if f.CursorX+cellWidth > right {
		offset = min(f.horizontalLimits[b.id], offset+f.CursorX+cellWidth-right)
	}
	if offset != v.horizontal[b.id] {
		v.horizontal[b.id] = offset
		return v.Layout(width, true)
	}
	return f
}
