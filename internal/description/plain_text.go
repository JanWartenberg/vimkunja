package description

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// PlainText projects the whole draft for sharing, independently of selection,
// terminal layout and save validation. Opaque atoms use their safe previews.
func (d *Draft) PlainText() string {
	if d.Classification() == Blocked {
		return "[Unreadable description content]"
	}
	if d.plainBaseline && d.state.recovered == d.savedRecovered && reflect.DeepEqual(d.state.nodes, d.savedNodes) {
		return strings.ReplaceAll(strings.ReplaceAll(d.baseline, "\r\n", "\n"), "\r", "\n")
	}
	return shareNodes(d.state.nodes, 0)
}

func shareBlock(n Node) bool {
	return n.Kind == Opaque && n.blockOpaque || n.Kind == Element && (isBlock(n.Tag) || n.Tag == "div")
}

func shareNodes(nodes []Node, depth int) string {
	var out strings.Builder
	previousBlock := false
	for _, n := range nodes {
		text := shareNode(n, depth)
		block := shareBlock(n)
		if text == "" {
			continue
		}
		if out.Len() > 0 && (previousBlock || block) {
			out.WriteString("\n\n")
		}
		out.WriteString(text)
		previousBlock = block
	}
	return out.String()
}

func shareNode(n Node, depth int) string {
	if n.Kind == Text {
		return n.Text
	}
	if n.Kind == Opaque {
		return n.Preview
	}
	switch n.Tag {
	case "br":
		return "\n"
	case "pre":
		return shareCode(n.Children)
	case "ul", "ol":
		var lines []string
		index := 1
		step := 1
		if n.Tag == "ol" && hasAttribute(n.Attributes, "reversed") {
			index = len(n.Children)
			step = -1
		}
		if start, err := strconv.Atoi(attributeValue(n.Attributes, "start")); err == nil {
			index = start
		}
		for _, li := range n.Children {
			if li.Tag != "li" {
				lines = append(lines, shareNode(li, depth))
				continue
			}
			prefix := "- "
			if n.Tag == "ol" {
				if value, err := strconv.Atoi(attributeValue(li.Attributes, "value")); err == nil {
					index = value
				}
				prefix = fmt.Sprintf("%d. ", index)
				index += step
			}
			if attributeValue(li.Attributes, "data-type") == "taskItem" {
				prefix = "[ ] "
				if attributeValue(li.Attributes, "data-checked") == "true" {
					prefix = "[x] "
				}
			}
			indent := strings.Repeat("  ", depth)
			var content strings.Builder
			previousBlock := false
			for _, child := range checkboxContent(li) {
				text := shareNode(child, depth+1)
				if text == "" {
					continue
				}
				block := shareBlock(child)
				if content.Len() > 0 && (previousBlock || block) {
					content.WriteByte('\n')
					if child.Tag != "ul" && child.Tag != "ol" {
						content.WriteString(indent + "  ")
					}
				}
				content.WriteString(text)
				previousBlock = block
			}
			lines = append(lines, indent+prefix+content.String())
		}
		return strings.Join(lines, "\n")
	case "a":
		label := shareNodes(n.Children, depth)
		url := attributeValue(n.Attributes, "href")
		if label == "" {
			return url
		}
		if url == "" || label == url {
			return label
		}
		return label + " (" + url + ")"
	default:
		return shareNodes(n.Children, depth)
	}
}

// Editor codeSource uses a sentinel for protected leaves; export their safe
// previews instead, without adding paragraph separators to source whitespace.
func shareCode(nodes []Node) string {
	var out strings.Builder
	for _, n := range nodes {
		switch {
		case n.Kind == Text:
			out.WriteString(n.Text)
		case n.Kind == Opaque:
			out.WriteString(n.Preview)
		case n.Tag == "br":
			out.WriteByte('\n')
		default:
			out.WriteString(shareCode(n.Children))
		}
	}
	return out.String()
}
