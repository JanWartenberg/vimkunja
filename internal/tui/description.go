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
	if !descriptionHasHTML(source) {
		if descriptionHasUnterminatedMarkup(source) {
			return "[Nicht lesbarer Beschreibungsinhalt]"
		}
		return plainDescriptionText(source)
	}
	document, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return ""
	}

	var paragraphs []string
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "ul", "ol":
				paragraphs = append(paragraphs, renderDescriptionList(node, 0)...)
				return
			case "table":
				paragraphs = append(paragraphs, renderDescriptionTable(node)...)
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
						paragraphs = append(paragraphs, "[Nicht unterstützter Inhalt]")
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
	text := strings.Join(paragraphs, "\n")
	if text == "" && descriptionHasUnterminatedMarkup(source) {
		return "[Nicht lesbarer Beschreibungsinhalt]"
	}
	return text
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

func renderDescriptionTable(table *html.Node) []string {
	var rows []string
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			var cells []string
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && (child.Data == "td" || child.Data == "th") {
					cells = append(cells, strings.Join(strings.Fields(descriptionNodeText(child)), " "))
				}
			}
			if len(cells) > 0 {
				rows = append(rows, strings.Join(cells, " | "))
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
		if checkboxList {
			prefix = "- [ ] "
			if strings.EqualFold(descriptionAttribute(child, "data-checked"), "true") {
				prefix = "- [x] "
			}
		} else if ordered {
			prefix = strconv.Itoa(ordinal) + ". "
			ordinal++
		}
		if text := descriptionListItemText(child); text != "" {
			lines = append(lines, strings.Repeat("  ", depth)+prefix+text)
		}
		for nested := child.FirstChild; nested != nil; nested = nested.NextSibling {
			if nested.Type == html.ElementNode && (nested.Data == "ul" || nested.Data == "ol") {
				lines = append(lines, renderDescriptionList(nested, depth+1)...)
			}
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
