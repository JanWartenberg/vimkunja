package tui

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const descriptionHardBreak = '\ue000'

func descriptionTerminalText(source string) string {
	return projectDescription(source, nil)
}

func projectDescription(source string, summaries map[int]bool) string {
	if !descriptionHasHTML(source) {
		if descriptionHasUnterminatedMarkup(source) {
			return "[Unreadable description content]"
		}
		return plainDescriptionText(source)
	}
	document, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return ""
	}

	text := projectDescriptionNode(document, summaries)
	if text == "" && descriptionHasUnterminatedMarkup(source) {
		return "[Unreadable description content]"
	}
	return text
}

func projectDescriptionNode(document *html.Node, summaries map[int]bool) string {
	var paragraphs []string
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "ul", "ol", "table":
				local := map[int]bool{}
				if summaries == nil {
					local = nil
				}
				var lines []string
				if node.Data == "table" {
					lines = renderDescriptionTable(node, local)
				} else {
					lines = renderChecklistLists(node, 0, local, false)
				}
				base := 0
				for _, paragraph := range paragraphs {
					base += strings.Count(paragraph, "\n") + 1
				}
				for index := range local {
					summaries[base+index] = true
				}
				paragraphs = append(paragraphs, lines...)
				return
			case "p":
				if text := descriptionNodeText(node); text != "" {
					paragraphs = append(paragraphs, text)
				} else if len(paragraphs) > 0 && paragraphs[len(paragraphs)-1] != "" {
					paragraphs = append(paragraphs, "")
				}
				return
			case "h1", "h2", "h3", "h4", "h5", "h6":
				if text := descriptionNodeText(node); text != "" {
					paragraphs = append(paragraphs, text, "")
				}
				return
			case "body", "head", "html":
				// Document containers have no presentation of their own.
			default:
				if !descriptionHasBlock(node) {
					if text := descriptionNodeText(node); text != "" {
						paragraphs = append(paragraphs, text)
					} else {
						paragraphs = append(paragraphs, "[Unsupported content]")
					}
					return
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	for len(paragraphs) > 0 && paragraphs[len(paragraphs)-1] == "" {
		paragraphs = paragraphs[:len(paragraphs)-1]
	}
	return strings.Join(paragraphs, "\n")
}

func descriptionHasHTML(source string) bool {
	tokenizer := html.NewTokenizer(strings.NewReader(source))
	for {
		switch tokenizer.Next() {
		case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken:
			return true
		case html.ErrorToken:
			return false
		}
	}
}

func descriptionHasUnterminatedMarkup(source string) bool {
	open := strings.LastIndex(source, "<")
	if open <= strings.LastIndex(source, ">") || open+1 >= len(source) {
		return false
	}
	next, _ := utf8.DecodeRuneInString(source[open+1:])
	return unicode.IsLetter(next) || next == '!' || next == '/' || next == '?'
}

func plainDescriptionText(source string) string {
	if strings.TrimSpace(source) == "" {
		return ""
	}
	return removeDangerousDescriptionControls(source)
}

func descriptionHasBlock(node *html.Node) bool {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			switch child.Data {
			case "h1", "h2", "h3", "h4", "h5", "h6", "ol", "p", "table", "ul":
				return true
			}
		}
		if descriptionHasBlock(child) {
			return true
		}
	}
	return false
}

func renderDescriptionTable(table *html.Node, summaries map[int]bool) []string {
	var rows []string
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			var cells [][]string
			var cellSummaries []map[int]bool
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type != html.ElementNode || (child.Data != "td" && child.Data != "th") {
					continue
				}
				text := strings.Join(strings.Fields(descriptionNodeText(child)), " ")
				local := map[int]bool{}
				if summaries != nil {
					projected := projectDescriptionNode(child, local)
					// Preserve the compact table projection unless a cell needs
					// logical lines for its display-only checklist summaries.
					if len(local) > 0 {
						text = projected
					}
				}
				cells = append(cells, strings.Split(text, "\n"))
				cellSummaries = append(cellSummaries, local)
			}
			positions := make([]int, len(cells))
			for {
				remaining, summaryRow := false, false
				for column, cell := range cells {
					if positions[column] < len(cell) {
						remaining = true
						summaryRow = summaryRow || cellSummaries[column][positions[column]]
					}
				}
				if !remaining {
					break
				}
				parts := make([]string, len(cells))
				for column, cell := range cells {
					position := positions[column]
					// Never style ordinary content in another cell as a summary.
					if position < len(cell) && cellSummaries[column][position] == summaryRow {
						parts[column] = cell[position]
						positions[column]++
					}
				}
				if summaryRow {
					summaries[len(rows)] = true
				}
				rows = append(rows, strings.Join(parts, " | "))
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(table)
	return rows
}

func renderDescriptionList(list *html.Node, depth int) []string {
	return renderChecklistLists(list, depth, nil, false)
}

func renderChecklistLists(list *html.Node, depth int, summaries map[int]bool, insideCheckbox bool) []string {
	ordered := list.Data == "ol"
	checkboxList := descriptionAttribute(list, "data-type") == "taskList"
	ordinal := 1
	if ordered {
		for _, attribute := range list.Attr {
			if attribute.Key == "start" {
				if value, err := strconv.Atoi(attribute.Val); err == nil && value > 0 {
					ordinal = value
				}
				break
			}
		}
	}

	var lines []string
	for child := list.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || child.Data != "li" {
			continue
		}
		prefix := "- "
		checkboxEntry := (checkboxList || insideCheckbox) && isCheckboxEntry(child)
		if checkboxEntry {
			prefix = "- [ ] "
			if strings.EqualFold(descriptionAttribute(child, "data-checked"), "true") {
				prefix = "- [x] "
			}
		} else if ordered {
			prefix = strconv.Itoa(ordinal) + ". "
			ordinal++
		}
		if text := descriptionListItemText(child); text != "" || checkboxEntry {
			lines = append(lines, strings.Repeat("  ", depth)+prefix+text)
		}
		var visitNested func(*html.Node)
		visitNested = func(n *html.Node) {
			if n.Type == html.ElementNode && (n.Data == "ul" || n.Data == "ol") {
				local := map[int]bool{}
				if summaries == nil {
					local = nil
				}
				nestedLines := renderChecklistLists(n, depth+1, local, insideCheckbox || checkboxList)
				base := 0
				for _, line := range lines {
					base += strings.Count(line, "\n") + 1
				}
				for index := range local {
					summaries[base+index] = true
				}
				lines = append(lines, nestedLines...)
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				visitNested(c)
			}
		}
		for nested := child.FirstChild; nested != nil; nested = nested.NextSibling {
			visitNested(nested)
		}
	}
	if summaries != nil && checkboxList && !insideCheckbox {
		checked, total := countCheckboxBlock(list)
		if total > 0 {
			index := 0
			for _, line := range lines {
				index += strings.Count(line, "\n") + 1
			}
			summaries[index] = true
			lines = append(lines, strings.Repeat("  ", depth)+checklistSummary(checked, total))
		}
	}
	return lines
}

func descriptionListItemText(item *html.Node) string {
	var text strings.Builder
	appendDescriptionNodeText(&text, item, true)
	return normalizeDescriptionText(text.String())
}

func descriptionNodeText(node *html.Node) string {
	var text strings.Builder
	appendDescriptionNodeText(&text, node, false)
	return normalizeDescriptionText(text.String())
}

func appendDescriptionNodeText(text *strings.Builder, node *html.Node, skipNestedLists bool) {
	if node.Type == html.TextNode {
		text.WriteString(node.Data)
		return
	}
	if node.Type == html.ElementNode {
		switch node.Data {
		case "br":
			text.WriteRune(descriptionHardBreak)
			return
		case "a":
			text.WriteString(descriptionLinkText(node))
			return
		case "img":
			alt := descriptionAttribute(node, "alt")
			if alt == "" {
				text.WriteString("[Bild]")
			} else {
				text.WriteString("[Bild: " + alt + "]")
			}
			return
		case "audio", "embed", "iframe", "object", "video":
			text.WriteString("[Medieninhalt]")
			return
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if skipNestedLists && child.Type == html.ElementNode && (child.Data == "ul" || child.Data == "ol") {
			continue
		}
		appendDescriptionNodeText(text, child, skipNestedLists)
	}
}

func descriptionAttribute(node *html.Node, key string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return normalizeDescriptionText(attribute.Val)
		}
	}
	return ""
}

func descriptionLinkText(link *html.Node) string {
	var visible strings.Builder
	for child := link.FirstChild; child != nil; child = child.NextSibling {
		appendDescriptionNodeText(&visible, child, false)
	}
	label := normalizeDescriptionText(visible.String())
	url := descriptionAttribute(link, "href")
	switch {
	case label == "":
		return url
	case url == "" || label == url:
		return label
	default:
		return label + " (" + url + ")"
	}
}

func normalizeDescriptionText(source string) string {
	source = removeDangerousDescriptionControls(source)
	lines := strings.Split(source, string(descriptionHardBreak))
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.Join(lines, "\n")
}

func removeDangerousDescriptionControls(source string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, source)
}
