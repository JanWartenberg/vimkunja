package tui

import "unicode"

type textNormalState struct {
	count         int
	operator      rune
	operatorCount int
	textObject    rune
	replaceCount  int
}

type textNormalResult struct {
	text        string
	cursor      int
	handled     bool
	enterInsert bool
}

func applyTextNormalKey(state *textNormalState, text string, cursor int, key string, multiline bool) textNormalResult {
	result := textNormalResult{text: text, cursor: clamp(cursor, 0, len([]rune(text)))}
	if state.replaceCount > 0 {
		result.handled = true
		count := state.replaceCount
		state.replaceCount = 0
		if key == "esc" || len([]rune(key)) != 1 {
			return result
		}
		r := []rune(text)
		start := normalCharacterIndex(text, result.cursor)
		replacement := []rune(key)[0]
		for i := start; i < len(r) && i < start+count && r[i] != '\n'; i++ {
			r[i] = replacement
		}
		result.text = string(r)
		return result
	}

	if key == "esc" && state.pending() {
		state.reset()
		result.handled = true
		return result
	}
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' || key == "0" && (state.count > 0 || state.operator != 0) {
		digit := int(key[0] - '0')
		state.count = state.count*10 + digit
		result.handled = true
		return result
	}

	if state.textObject != 0 {
		result.handled = true
		if key == "w" {
			count := combinedCount(state)
			start, end := wordObjectRange([]rune(text), result.cursor, false, state.textObject == 'a', count)
			result.text, result.cursor = deleteRuneRange(text, start, end)
			result.enterInsert = state.operator == 'c'
		}
		state.reset()
		return result
	}

	if state.operator != 0 {
		result.handled = true
		if key == "i" || key == "a" {
			state.textObject = []rune(key)[0]
			return result
		}
		count := combinedCount(state)
		op := state.operator
		if key == string(op) && op == 'd' {
			if multiline {
				start, end := lineRange([]rune(text), result.cursor, count)
				result.text, result.cursor = deleteRuneRange(text, start, end)
			}
			state.reset()
			return result
		}
		if key == "w" && op == 'c' || op == 'd' && (key == "w" || key == "b" || key == "e") {
			motion := key
			if op == 'c' { // Vim's cw has ce semantics: preserve following whitespace.
				motion = "e"
			}
			target := moveTextMotion([]rune(text), result.cursor, motion, count, multiline)
			start, end := operatorRange([]rune(text), result.cursor, target, motion)
			result.text, result.cursor = deleteRuneRange(text, start, end)
			result.enterInsert = op == 'c'
		}
		state.reset()
		return result
	}

	if key == "d" || key == "c" {
		state.operator = []rune(key)[0]
		state.operatorCount = max(1, state.count)
		state.count = 0
		result.handled = true
		return result
	}

	hadCount := state.count > 0
	count := max(1, state.count)
	state.count = 0
	switch key {
	case "0", "^", "$", "h", "l", "w", "b", "e", "W", "B", "E":
		result.cursor = moveTextMotion([]rune(text), result.cursor, key, count, multiline)
		result.handled = true
	case "j", "k":
		if multiline {
			result.cursor = moveTextMotion([]rune(text), result.cursor, key, count, true)
			result.handled = true
		}
	case "G":
		if multiline {
			if hadCount {
				result.cursor = nthLineStart([]rune(text), count-1)
			} else {
				result.cursor = len([]rune(text))
			}
			result.handled = true
		}
	case "x":
		start := normalCharacterIndex(text, result.cursor)
		result.text, result.cursor = deleteRuneRange(text, start, min(textLineEnd(text, start), start+count))
		result.handled = true
	case "~":
		result.text, result.cursor = toggleTextCase(text, result.cursor, count)
		result.handled = true
	case "C":
		start := normalCharacterIndex(text, result.cursor)
		result.text, result.cursor = deleteRuneRange(text, start, textLineEnd(text, start))
		result.handled, result.enterInsert = true, true
	case "r":
		state.replaceCount = count
		result.handled = true
	default:
		// A count belongs to one command and must never leak into a later key.
		if count > 1 {
			result.handled = true
		}
	}
	return result
}

func (s *textNormalState) pending() bool {
	return s.count > 0 || s.operator != 0 || s.textObject != 0 || s.replaceCount > 0
}

func (s *textNormalState) reset() { *s = textNormalState{} }

func combinedCount(state *textNormalState) int {
	return max(1, state.operatorCount) * max(1, state.count)
}

func moveTextMotion(r []rune, cursor int, motion string, count int, multiline bool) int {
	pos := clamp(cursor, 0, len(r))
	count = max(1, count)
	if (motion == "j" || motion == "k") && multiline {
		if motion == "k" {
			count = -count
		}
		return verticalTextMotion(r, pos, count)
	}
	if motion == "$" {
		for i := 0; i < count; i++ {
			pos = textLineEnd(string(r), pos)
			if i+1 < count && pos < len(r) {
				pos++
			}
		}
		return pos
	}
	for range count {
		switch motion {
		case "h":
			if pos > textLineStart(string(r), pos) {
				pos--
			}
		case "l":
			if pos < textLineEnd(string(r), pos) {
				pos++
			}
		case "0":
			pos = textLineStart(string(r), pos)
		case "^":
			pos = textLineStart(string(r), pos)
			for pos < len(r) && (r[pos] == ' ' || r[pos] == '\t') {
				pos++
			}
		case "w", "W":
			pos = nextWordStart(r, pos, motion == "W", multiline)
		case "b", "B":
			pos = previousWordStart(r, pos, motion == "B", multiline)
		case "e", "E":
			pos = nextWordEnd(r, pos, motion == "E", multiline)
		}
	}
	return pos
}

func wordClass(r rune, big bool) int {
	if unicode.IsSpace(r) {
		return 0
	}
	if big || unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return 1
	}
	return 2
}

func nextWordStart(r []rune, pos int, big, multiline bool) int {
	if pos >= len(r) {
		return len(r)
	}
	class := wordClass(r[pos], big)
	for pos < len(r) && wordClass(r[pos], big) == class {
		if !multiline && r[pos] == '\n' {
			return pos
		}
		pos++
	}
	for pos < len(r) && wordClass(r[pos], big) == 0 {
		if !multiline && r[pos] == '\n' {
			return pos
		}
		pos++
	}
	return pos
}

func previousWordStart(r []rune, pos int, big, multiline bool) int {
	if pos <= 0 {
		return 0
	}
	pos--
	for pos > 0 && wordClass(r[pos], big) == 0 {
		if !multiline && r[pos] == '\n' {
			return pos + 1
		}
		pos--
	}
	class := wordClass(r[pos], big)
	for pos > 0 && wordClass(r[pos-1], big) == class {
		if !multiline && r[pos-1] == '\n' {
			break
		}
		pos--
	}
	return pos
}

func nextWordEnd(r []rune, pos int, big, multiline bool) int {
	if len(r) == 0 {
		return 0
	}
	pos = min(pos, len(r)-1)
	class := wordClass(r[pos], big)
	if class != 0 && pos+1 < len(r) && wordClass(r[pos+1], big) == class {
		for pos+1 < len(r) && wordClass(r[pos+1], big) == class {
			pos++
		}
		return pos
	}
	if class != 0 {
		pos++
	}
	for pos < len(r) && wordClass(r[pos], big) == 0 {
		if !multiline && r[pos] == '\n' {
			return pos
		}
		pos++
	}
	if pos >= len(r) {
		return len(r)
	}
	class = wordClass(r[pos], big)
	for pos+1 < len(r) && wordClass(r[pos+1], big) == class {
		pos++
	}
	return pos
}

func verticalTextMotion(r []rune, pos, delta int) int {
	text := string(r)
	start := textLineStart(text, pos)
	column := pos - start
	line := 0
	for i := 0; i < start; i++ {
		if r[i] == '\n' {
			line++
		}
	}
	target := max(0, line+delta)
	lineCount := 1
	for _, char := range r {
		if char == '\n' {
			lineCount++
		}
	}
	target = min(target, lineCount-1)
	targetStart := 0
	for i := 0; i < target; i++ {
		targetStart = textLineEnd(text, targetStart) + 1
	}
	return min(targetStart+column, textLineEnd(text, targetStart))
}

func nthLineStart(r []rune, line int) int {
	if line <= 0 {
		return 0
	}
	for pos, char := range r {
		if char == '\n' {
			line--
			if line == 0 {
				return pos + 1
			}
		}
	}
	return len(r)
}

func operatorRange(r []rune, cursor, target int, motion string) (int, int) {
	if target < cursor {
		return target, cursor
	}
	end := target
	if motion == "e" && target < len(r) {
		end++
	}
	return cursor, end
}

func wordObjectRange(r []rune, cursor int, big, around bool, count int) (int, int) {
	if len(r) == 0 {
		return 0, 0
	}
	pos := min(cursor, len(r)-1)
	class := wordClass(r[pos], big)
	start, end := pos, pos+1
	if class == 0 {
		for start > 0 && wordClass(r[start-1], big) == 0 {
			start--
		}
		for end < len(r) && wordClass(r[end], big) == 0 {
			end++
		}
		if !around {
			return start, end
		}
		if end < len(r) {
			nextClass := wordClass(r[end], big)
			for end < len(r) && wordClass(r[end], big) == nextClass {
				end++
			}
			return start, end
		}
		if start > 0 {
			previousClass := wordClass(r[start-1], big)
			for start > 0 && wordClass(r[start-1], big) == previousClass {
				start--
			}
		}
		return start, end
	}
	for start > 0 && wordClass(r[start-1], big) == class {
		start--
	}
	for end < len(r) && wordClass(r[end], big) == class {
		end++
	}
	for i := 1; i < count; i++ {
		end = nextWordStart(r, end, big, true)
		if end >= len(r) {
			break
		}
		nextClass := wordClass(r[end], big)
		for end < len(r) && wordClass(r[end], big) == nextClass {
			end++
		}
	}
	if around {
		if end < len(r) && unicode.IsSpace(r[end]) {
			for end < len(r) && unicode.IsSpace(r[end]) {
				end++
			}
		} else {
			for start > 0 && unicode.IsSpace(r[start-1]) {
				start--
			}
		}
	}
	return start, end
}

func lineRange(r []rune, cursor, count int) (int, int) {
	text := string(r)
	start := textLineStart(text, cursor)
	end := textLineEnd(text, cursor)
	for i := 1; i < count && end < len(r); i++ {
		end = textLineEnd(text, end+1)
	}
	if end < len(r) {
		end++
	} else if start > 0 {
		start--
	}
	return start, end
}

func normalCharacterIndex(text string, cursor int) int {
	cursor = clamp(cursor, 0, len([]rune(text)))
	if cursor > textLineStart(text, cursor) && cursor == textLineEnd(text, cursor) {
		return cursor - 1
	}
	return cursor
}

func toggleTextCase(text string, cursor, count int) (string, int) {
	r := []rune(text)
	start := normalCharacterIndex(text, cursor)
	end := min(textLineEnd(text, start), start+max(1, count))
	for i := start; i < end; i++ {
		switch {
		case unicode.IsUpper(r[i]) || unicode.IsTitle(r[i]):
			r[i] = unicode.ToLower(r[i])
		case unicode.IsLower(r[i]):
			r[i] = unicode.ToUpper(r[i])
		}
	}
	return string(r), end
}

func deleteRuneRange(text string, start, end int) (string, int) {
	r := []rune(text)
	start, end = clamp(start, 0, len(r)), clamp(end, 0, len(r))
	if end < start {
		start, end = end, start
	}
	return string(append(r[:start:start], r[end:]...)), start
}
