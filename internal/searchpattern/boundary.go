package searchpattern

import (
	"regexp/syntax"
	"unicode/utf8"
)

type searchThread struct {
	pc    uint32
	start int
}

// RE2 has no Unicode word assertions. Evaluate the compiled program with those
// assertions at capture markers, before choosing a match. Filtering completed
// matches would lose valid alternatives and shorter greedy matches. Ordered
// threads preserve RE2's leftmost-first priority and visit each instruction at
// most once per input position, without backtracking.
func (p *Pattern) findBoundaryMatch(text string, from int) (Range, bool) {
	var current, next []searchThread
	seen := make([]bool, len(p.program.Inst))
	var result Range
	found := false
	var add func(uint32, int, int) bool
	add = func(pc uint32, start, at int) bool {
		if seen[pc] {
			return false
		}
		seen[pc] = true
		inst := &p.program.Inst[pc]
		switch inst.Op {
		case syntax.InstFail:
			return false
		case syntax.InstAlt, syntax.InstAltMatch:
			return add(inst.Out, start, at) || add(inst.Arg, start, at)
		case syntax.InstCapture:
			if beginning, ok := p.assertions[inst.Arg]; ok {
				left, right := surroundingRunes(text, at)
				if beginning && (word(left) || !word(right)) || !beginning && (!word(left) || word(right)) {
					return false
				}
			}
			return add(inst.Out, start, at)
		case syntax.InstNop:
			return add(inst.Out, start, at)
		case syntax.InstEmptyWidth:
			left, right := surroundingRunes(text, at)
			if syntax.EmptyOp(inst.Arg)&^syntax.EmptyOpContext(left, right) != 0 {
				return false
			}
			return add(inst.Out, start, at)
		case syntax.InstMatch:
			result, found = Range{start, at}, true
			return true // Discard lower-priority paths, retaining earlier threads.
		default:
			next = append(next, searchThread{pc, start})
			return false
		}
	}
	add(uint32(p.program.Start), from, from)
	current, next = next, current
	for at := from; at < len(text) && (!found || len(current) > 0); {
		r, size := utf8.DecodeRuneInString(text[at:])
		at += size
		clear(seen)
		cut := false
		for _, thread := range current {
			inst := &p.program.Inst[thread.pc]
			matches := false
			switch inst.Op {
			case syntax.InstRune, syntax.InstRune1:
				matches = inst.MatchRune(r)
			case syntax.InstRuneAny:
				matches = true
			case syntax.InstRuneAnyNotNL:
				matches = r != '\n'
			}
			if matches && add(inst.Out, thread.start, at) {
				cut = true
				break
			}
		}
		if !cut && !found {
			add(uint32(p.program.Start), at, at)
		}
		current, next = next, current[:0]
	}
	return result, found
}

func surroundingRunes(text string, at int) (rune, rune) {
	left, right := rune(-1), rune(-1)
	if at > 0 {
		left, _ = utf8.DecodeLastRuneInString(text[:at])
	}
	if at < len(text) {
		right, _ = utf8.DecodeRuneInString(text[at:])
	}
	return left, right
}

func (p *Pattern) findAllWithBoundaries(text string) []Range {
	var result []Range
	previousEnd := -1
	for from := 0; from <= len(text); {
		match, ok := p.findBoundaryMatch(text, from)
		if !ok {
			break
		}
		if match.Start != match.End || match.Start != previousEnd {
			result = append(result, match)
		}
		previousEnd = match.End
		from = match.End
		if match.Start == match.End {
			if from == len(text) {
				break
			}
			_, size := utf8.DecodeRuneInString(text[from:])
			from += size
		}
	}
	return result
}
