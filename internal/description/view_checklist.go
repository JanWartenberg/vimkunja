package description

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
	htmlatom "golang.org/x/net/html/atom"
)

func (v *View) opaqueDOM(id ID) []*html.Node {
	index, ok := v.doc.byID[id]
	if !ok || v.doc.atoms[index].preview == "[Active content]" {
		return nil
	}
	raw := v.doc.atoms[index].raw
	// Do not choose one ambiguous checkbox state from duplicate attributes.
	tokens := html.NewTokenizer(strings.NewReader(raw))
	for {
		kind := tokens.Next()
		if kind == html.ErrorToken {
			break
		}
		if kind == html.StartTagToken || kind == html.SelfClosingTagToken {
			seen := map[string]bool{}
			for _, a := range sourceAttributes(string(tokens.Raw())) {
				if (a.Name == "data-type" || a.Name == "data-checked") && seen[a.Name] {
					return nil
				}
				seen[a.Name] = true
			}
		}
	}
	nodes, err := html.ParseFragment(strings.NewReader(raw), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: htmlatom.Div})
	if err != nil {
		return nil
	}
	return nodes
}
func domAttribute(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}
func domCheckboxCount(nodes []*html.Node) (done, total int) {
	for _, n := range nodes {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			continue
		}
		if n.Type == html.ElementNode && n.Data == "li" && domAttribute(n, "data-type") == "taskItem" {
			total++
			if strings.EqualFold(domAttribute(n, "data-checked"), "true") {
				done++
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			d, t := domCheckboxCount([]*html.Node{ch})
			done += d
			total += t
		}
	}
	return
}
func (v *View) opaqueCheckboxCount(id ID) (int, int) { return domCheckboxCount(v.opaqueDOM(id)) }
func (v *View) opaqueChecklistPreview(id ID) (string, bool) {
	roots := v.opaqueDOM(id)
	if len(roots) == 0 {
		return "", false
	}
	var containsList func(*html.Node) bool
	containsList = func(n *html.Node) bool {
		if n.Data == "script" || n.Data == "style" {
			return false
		}
		if (n.Data == "ul" || n.Data == "ol") && domAttribute(n, "data-type") == "taskList" {
			return true
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if containsList(c) {
				return true
			}
		}
		return false
	}
	found := false
	for _, root := range roots {
		found = found || containsList(root)
	}
	if !found {
		return "", false
	}
	var lines []string
	var entries func(*html.Node, int)
	entries = func(node *html.Node, depth int) {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != html.ElementNode || child.Data != "li" {
				var raw strings.Builder
				html.Render(&raw, child)
				if text := Parse(raw.String()).RecoveryText(); text != "" {
					lines = append(lines, text)
				}
				continue
			}
			if child.Type == html.ElementNode && child.Data == "li" {
				marker := "- "
				if domAttribute(child, "data-type") == "taskItem" {
					marker = "☐ "
					if strings.EqualFold(domAttribute(child, "data-checked"), "true") {
						marker = "☑ "
					}
					if v.options.ASCII {
						marker = "[ ] "
						if strings.EqualFold(domAttribute(child, "data-checked"), "true") {
							marker = "[x] "
						}
					}
				}
				var own strings.Builder
				for c := child.FirstChild; c != nil; c = c.NextSibling {
					if c.Data != "ul" && c.Data != "ol" {
						html.Render(&own, c)
					}
				}
				text := Parse(own.String()).RecoveryText()
				lines = append(lines, strings.Repeat("  ", depth)+marker+text)
				for c := child.FirstChild; c != nil; c = c.NextSibling {
					if c.Data == "ul" || c.Data == "ol" {
						entries(c, depth+1)
					}
				}
			}
		}
	}
	var project func(*html.Node)
	project = func(n *html.Node) {
		if (n.Data == "ul" || n.Data == "ol") && domAttribute(n, "data-type") == "taskList" {
			entries(n, 0)
			done, total := domCheckboxCount([]*html.Node{n})
			if total > 0 {
				lines = append(lines, fmt.Sprintf("Checklist progress: %d/%d", done, total))
			}
			return
		}
		if containsList(n) {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				project(c)
			}
			return
		}
		var raw strings.Builder
		html.Render(&raw, n)
		if text := Parse(raw.String()).RecoveryText(); text != "" {
			lines = append(lines, text)
		}
	}
	for _, root := range roots {
		project(root)
	}
	if len(lines) == 0 {
		return "", false
	}
	return strings.Join(lines, "\n"), true
}

// Only unambiguous known entry metadata contributes presentation. Unknown
// attributes and the original Opaque HTML remain private and untouched.
func (v *View) opaqueEntryAttributes(id ID) []Attribute {
	index, ok := v.doc.byID[id]
	if !ok || v.doc.atoms[index].preview == "[Active content]" {
		return nil
	}
	tokens := html.NewTokenizer(strings.NewReader(v.doc.atoms[index].raw))
	if tokens.Next() != html.StartTagToken || tokens.Token().Data != "li" {
		return nil
	}
	attrs := sourceAttributes(string(tokens.Raw()))
	counts := map[string]int{}
	for _, a := range attrs {
		counts[a.Name]++
	}
	var out []Attribute
	for _, a := range attrs {
		if counts[a.Name] != 1 {
			continue
		}
		if a.Name == "value" || ((a.Name == "data-type" || a.Name == "data-checked") && counts["data-type"] < 2 && counts["data-checked"] < 2) {
			out = append(out, a)
		}
	}
	return out
}
