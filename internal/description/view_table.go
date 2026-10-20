package description

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/net/html"
	htmlatom "golang.org/x/net/html/atom"
)

// Only this document-owned projection sees Opaque HTML. Callers receive safe
// cells, never attributes/URLs, and no tree repair is written back to source.
func (v *View) tablePreview(id ID) (string, bool) {
	index, ok := v.doc.byID[id]
	if !ok {
		return "", false
	}
	raw := v.doc.atoms[index].raw
	tokenizer := html.NewTokenizer(strings.NewReader(raw))
	if tokenizer.Next() != html.StartTagToken || tokenizer.Token().Data != "table" {
		return "", false
	}
	roots, err := html.ParseFragment(strings.NewReader(raw), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: htmlatom.Div})
	if err != nil {
		return "", false
	}
	irregularPlaceholder := false
	var captions []string
	tables := 0
	var rows [][]string
	var inspect func(*html.Node)
	inspect = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			tables++
			if tables > 1 {
				irregularPlaceholder = true
			}
		}
		if n.Type == html.ElementNode && n.Data == "caption" {
			var raw strings.Builder
			html.Render(&raw, n)
			captions = append(captions, Parse(raw.String()).RecoveryText())
		}
		if n.Type == html.ElementNode && n.Data == "table" && n.Parent != nil && n.Parent.Data != "div" {
			irregularPlaceholder = true
		}
		for _, a := range n.Attr {
			if a.Key == "style" || a.Key == "rowspan" || a.Key == "colspan" {
				irregularPlaceholder = true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			inspect(c)
		}
	}

	for _, root := range roots {
		inspect(root)
	}
	irregular := irregularPlaceholder
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		for _, a := range n.Attr {
			if a.Key == "style" || a.Key == "rowspan" || a.Key == "colspan" {
				irregular = true
			}
		}
		if n.Type == html.ElementNode && n.Data == "tr" {
			var cells []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					for _, a := range c.Attr {
						if a.Key == "style" || a.Key == "rowspan" || a.Key == "colspan" {
							irregular = true
						}
					}
					var inner strings.Builder
					for ch := c.FirstChild; ch != nil; ch = ch.NextSibling {
						html.Render(&inner, ch)
					}
					// Recovery excludes script/style, hidden URLs/IDs and unsafe attributes.
					text := Parse(inner.String()).RecoveryText()
					cells = append(cells, strings.Join(strings.Fields(text), " "))
				} else if c.Type == html.ElementNode {
					irregular = true
				}
			}
			rows = append(rows, cells)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	for _, root := range roots {
		visit(root)
	}
	withCaption := func(text string) string {
		if len(captions) == 0 {
			return text
		}
		return strings.Join(captions, "\n") + "\n" + text
	}
	if len(rows) == 0 {
		return withCaption("[Table]"), true
	}
	columns := len(rows[0])
	if columns == 0 {
		irregular = true
	}
	widths := make([]int, columns)
	for _, row := range rows {
		if len(row) != columns {
			irregular = true
			continue
		}
		for i, text := range row {
			widths[i] = max(widths[i], ansi.StringWidth(text))
		}
	}
	totalWidth := columns + 1
	for _, w := range widths {
		totalWidth += w + 2
	}
	// A padded grid can amplify a single large cell across thousands of rows.
	// Keep the bounded row/cell fallback for such unusually large tables too.
	if totalWidth > 0 && len(rows) > 100000/totalWidth {
		irregular = true
	}
	if irregular {
		lines := make([]string, len(rows))
		for i, row := range rows {
			lines[i] = strings.Join(row, " | ")
		}
		return withCaption(strings.Join(lines, "\n")), true
	}
	left, right, junction, horizontal := "┌", "┐", "┬", "─"
	if v.options.ASCII {
		left, right, junction, horizontal = "+", "+", "+", "-"
	}
	border := func(l, j, r string) string {
		parts := make([]string, len(widths))
		for i, w := range widths {
			parts[i] = strings.Repeat(horizontal, w+2)
		}
		return l + strings.Join(parts, j) + r
	}
	lines := []string{border(left, junction, right)}
	vertical := "│"
	if v.options.ASCII {
		vertical = "|"
	}
	for _, row := range rows {
		parts := make([]string, len(row))
		for i, text := range row {
			parts[i] = " " + text + strings.Repeat(" ", widths[i]-ansi.StringWidth(text)+1)
		}
		lines = append(lines, vertical+strings.Join(parts, vertical)+vertical)
	}
	l, j, r := "└", "┴", "┘"
	if v.options.ASCII {
		l, j, r = "+", "+", "+"
	}
	lines = append(lines, border(l, j, r))
	return withCaption(strings.Join(lines, "\n")), true
}

func (v *View) opaqueTag(id ID) string {
	index, ok := v.doc.byID[id]
	if !ok {
		return ""
	}
	tokenizer := html.NewTokenizer(strings.NewReader(v.doc.atoms[index].raw))
	if tokenizer.Next() == html.StartTagToken {
		return tokenizer.Token().Data
	}
	return ""
}
