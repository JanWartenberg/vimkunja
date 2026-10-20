package description

import (
	"strconv"
	"strings"
	"unicode"
)

// visibleSource is a safe content projection, not terminal layout. An active
// descendant suppresses the whole enclosing atom's preview. Recovery excludes
// hidden active content and presentation hints.
func visibleSource(root *sourceNode, hints bool) string {
	if hints && root.comment {
		return "[HTML comment]"
	}
	if hints {
		pending := []*sourceNode{root}
		for len(pending) > 0 {
			n := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if n.active {
				return "[Active content]"
			}
			pending = append(pending, n.children...)
		}
	}
	type event struct {
		n   *sourceNode
		end bool
	}
	pending := []event{{n: root}}
	var out strings.Builder
	newline := func() {
		if out.Len() > 0 && !strings.HasSuffix(out.String(), "\n") {
			out.WriteByte('\n')
		}
	}
	for len(pending) > 0 {
		e := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		n := e.n
		if e.end {
			if projectionBlock(n.tag) {
				newline()
			}
			continue
		}
		if n.active || n.tag == "style" || n.tag == "head" || n.comment {
			continue
		}
		if n.tag == "" && len(n.children) == 0 {
			out.WriteString(n.text)
			continue
		}
		if n.tag == "br" {
			out.WriteByte('\n')
			continue
		}
		if projectionBlock(n.tag) {
			newline()
		}
		if n.tag == "td" || n.tag == "th" {
			if out.Len() > 0 && !strings.HasSuffix(out.String(), "\n") {
				out.WriteString(" | ")
			}
		}
		attr := func(name string) string {
			for _, a := range n.attrs {
				if a.Name == name {
					return a.Value
				}
			}
			return ""
		}
		switch n.tag {
		case "img":
			label := attr("alt")
			if label == "" {
				label = attr("title")
			}
			if hints {
				if label == "" {
					label = "[Image]"
				} else {
					label = "[Image: " + label + "]"
				}
			}
			out.WriteString(label)
			continue
		case "mention-user":
			label := attr("data-label")
			if label == "" {
				label = attr("data-id")
			}
			if label != "" {
				out.WriteString("@" + label)
			}
			continue
		case "iframe", "object", "embed", "video", "audio":
			if hints {
				out.WriteString("[Embedded media]")
			}
			continue
		}
		pending = append(pending, event{n: n, end: true})
		for i := len(n.children) - 1; i >= 0; i-- {
			pending = append(pending, event{n: n.children[i]})
		}
	}
	text := terminalSafeText(strings.Trim(out.String(), "\n"))
	if hints && strings.TrimSpace(text) == "" {
		if root.tag == "table" {
			return "[Table]"
		}
		return "[Unsupported content]"
	}
	return text
}

func projectionBlock(tag string) bool {
	return isBlock(tag) || tag == "li" || tag == "tr"
}

// Escape controls as visible text, rather than interpreting terminal commands
// or deleting source characters. Original HTML and decoded Native text remain
// untouched. CRLF becomes one displayed line break; a lone CR stays visible.
func terminalSafeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out strings.Builder
	for _, r := range text {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			quoted := strconv.QuoteRune(r)
			out.WriteString(quoted[1 : len(quoted)-1])
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}
