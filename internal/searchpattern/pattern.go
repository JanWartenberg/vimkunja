// Package searchpattern implements an explicit, bounded subset of Vim's
// default-magic search syntax. Unsupported syntax is rejected, never passed
// through to Go's different regular-expression dialect.
package searchpattern

import (
	"fmt"
	"regexp"
	"regexp/syntax"
	"strings"
	"unicode"
)

type Range struct{ Start, End int }

type boundary struct {
	group int
	start bool
}

type Pattern struct {
	re         *regexp.Regexp
	program    *syntax.Prog
	assertions map[uint32]bool
}

func Compile(source string) (*Pattern, error) {
	if len(source) > 4096 {
		return nil, fmt.Errorf("search pattern is too long (maximum 4096 bytes)")
	}
	var out strings.Builder
	var boundaries []boundary
	groups := 0
	upper, force := false, 0
	branchStart, afterStartAnchor := true, false
	runes := []rune(source)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' {
			i++
			if i == len(runes) {
				return nil, fmt.Errorf("unfinished search escape")
			}
			r = runes[i]
			switch r {
			case 'c':
				force = -1
			case 'C':
				force = 1
			case '(':
				out.WriteRune('(')
				groups++
				branchStart, afterStartAnchor = true, false
				continue
			case ')', '|', '+':
				out.WriteRune(r)
				if r == '|' {
					branchStart, afterStartAnchor = true, false
					continue
				}
			case '?', '=':
				out.WriteRune('?')
			case '<', '>':
				groups++
				out.WriteString("()")
				boundaries = append(boundaries, boundary{groups, r == '<'})
			case '{':
				end := i + 1
				for end < len(runes) && runes[end] != '}' {
					end++
				}
				if end == len(runes) {
					return nil, fmt.Errorf("unfinished search repetition")
				}
				body := string(runes[i+1 : end])
				if body == "" || strings.Trim(body, "0123456789,") != "" || strings.Count(body, ",") > 1 {
					return nil, fmt.Errorf("unsupported Vim repetition: \\{%s}", body)
				}
				if strings.HasPrefix(body, ",") {
					body = "0" + body
				}
				out.WriteString("{" + body + "}")
				i = end
			case 'd', 'D', 's', 'S', 'w', 'W':
				out.WriteRune('\\')
				out.WriteRune(r)
			case 't':
				out.WriteString(`\t`)
			case 'n':
				out.WriteString(`\n`)
			case '\\', '.', '*', '[', ']', '^', '$', '/':
				out.WriteString(regexp.QuoteMeta(string(r)))
			default:
				return nil, fmt.Errorf("unsupported Vim search escape: \\%c", r)
			}
			if r != 'c' && r != 'C' {
				branchStart, afterStartAnchor = false, false
			}
			continue
		}
		if r == '[' {
			out.WriteRune('[')
			if i+1 < len(runes) && runes[i+1] == '^' {
				i++
				out.WriteRune('^')
			}
			closed := false
			for i++; i < len(runes); i++ {
				r = runes[i]
				if r == ']' {
					out.WriteRune(']')
					closed = true
					break
				}
				if r == '[' || r == '\\' {
					return nil, fmt.Errorf("unsupported search character class")
				}
				upper = upper || unicode.IsUpper(r)
				out.WriteRune(r)
			}
			if !closed {
				return nil, fmt.Errorf("unfinished search character class")
			}
			branchStart, afterStartAnchor = false, false
			continue
		}
		upper = upper || unicode.IsUpper(r)
		anchorEnd := i+1 == len(runes) || i+2 < len(runes) && runes[i+1] == '\\' && strings.ContainsRune("|)n", runes[i+2])
		if r == '^' && branchStart {
			out.WriteRune(r)
			afterStartAnchor, branchStart = true, false
			continue
		}
		if r == '.' || r == '*' && !branchStart && !afterStartAnchor || r == '$' && anchorEnd {
			out.WriteRune(r)
		} else {
			out.WriteString(regexp.QuoteMeta(string(r)))
		}
		branchStart, afterStartAnchor = false, false
	}
	flags := "(?m)"
	if force < 0 || force == 0 && !upper {
		flags = "(?mi)"
	}
	re, err := regexp.Compile(flags + out.String())
	if err != nil {
		return nil, fmt.Errorf("invalid Vim search pattern: %w", err)
	}
	p := &Pattern{re: re}
	if len(boundaries) > 0 {
		expression, err := syntax.Parse(flags+out.String(), syntax.Perl)
		if err != nil {
			return nil, err
		}
		p.program, err = syntax.Compile(expression.Simplify())
		if err != nil {
			return nil, err
		}
		p.assertions = make(map[uint32]bool, len(boundaries))
		for _, b := range boundaries {
			p.assertions[uint32(b.group*2)] = b.start
		}
	}
	return p, nil
}

func word(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r) || r == '_'
}

// FindAll returns non-overlapping byte ranges, including zero-width matches.
// Word boundaries use Unicode letters/numbers rather than Go's ASCII-only \b.
func (p *Pattern) FindAll(text string) []Range {
	if p.program != nil {
		return p.findAllWithBoundaries(text)
	}
	var result []Range
	for _, indices := range p.re.FindAllStringIndex(text, -1) {
		result = append(result, Range{indices[0], indices[1]})
	}
	return result
}
