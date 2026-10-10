package description

import (
	"fmt"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// The creation policy is separate from safe import: foreign protocols can be
// retained without becoming newly creatable through the URL prompt.
var linkSchemes = map[string]bool{"http": true, "https": true, "ftp": true, "git": true, "obsidian": true, "notion": true, "message": true}

func ValidateLinkURL(value string) error {
	fail := func() error {
		return fmt.Errorf("Invalid Link URL: use an absolute http, https, ftp, git, obsidian, notion or message :// URL")
	}
	if strings.IndexFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 || strings.ContainsAny(value, `\<>"`) {
		return fail()
	}
	u, err := url.Parse(value)
	if err != nil || !linkSchemes[strings.ToLower(u.Scheme)] || !strings.HasPrefix(strings.ToLower(value), strings.ToLower(u.Scheme)+"://") || u.Host == "" {
		return fail()
	}
	if _, err := url.PathUnescape(u.RawQuery); err != nil {
		return fail()
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "http" || scheme == "https" || scheme == "ftp" || scheme == "git" {
		host := u.Hostname()
		if host == "" {
			return fail()
		}
		if strings.Contains(host, ":") {
			if net.ParseIP(host) == nil {
				return fail()
			}
		} else {
			for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
				if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
					return fail()
				}
				for _, r := range label {
					if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
						return fail()
					}
				}
			}
		}
		if port := u.Port(); port != "" {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 {
				return fail()
			}
		}
		if strings.HasSuffix(u.Host, ":") {
			return fail()
		}
	}
	return nil
}

type LinkInfo struct {
	URL               string
	Edge, Unsupported bool
}

func (d *Draft) currentLink() *Node {
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return nil
	}
	if loc.Node.Tag == "a" {
		return findNode(d.state.nodes, loc.Node.ID)
	}
	for i := len(loc.Ancestors) - 1; i >= 0; i-- {
		n := findNode(d.state.nodes, loc.Ancestors[i])
		if n != nil && n.Tag == "a" {
			return n
		}
	}
	return nil
}
func (d *Draft) LinkAtCursor() (LinkInfo, bool) {
	n := d.currentLink()
	if n == nil {
		return LinkInfo{}, false
	}
	ext, _ := d.extents()
	pos := anchorPosition(d.state.cursor, ext)
	e := ext[n.ID]
	value := attributeValue(n.Attributes, "href")
	u, _ := url.Parse(value)
	unsupported := u == nil || !linkSchemes[strings.ToLower(u.Scheme)]
	return LinkInfo{URL: value, Edge: len(n.Children) > 0 && (pos == e.start || pos == e.end), Unsupported: unsupported}, true
}
func (d *Draft) editLink(value string) error {
	if value != "" {
		if err := ValidateLinkURL(value); err != nil {
			return err
		}
	}
	if d.state.selection == nil || d.currentLink() != nil && len(d.currentLink().Children) == 0 && d.state.selection.Start.Node == d.currentLink().ID && d.state.selection.End.Node == d.currentLink().ID {
		n := d.currentLink()
		if n == nil {
			return fmt.Errorf("Select text or place the cursor in a Link")
		}
		if value != "" {
			setAttribute(n, "href", value)
			setAttribute(n, "target", "_blank")
			return nil
		}
		list, i := nodeList(&d.state.nodes, n.ID)
		children := n.Children
		if len(children) == 0 {
			children = []Node{d.newNode(Node{Kind: Text, Text: attributeValue(n.Attributes, "href")})}
		}
		// Replacing the slice invalidates n; remap virtual fallback targets first.
		removed := n.ID
		remap := func(anchor *Anchor, syntax *int) {
			if anchor.Node != removed {
				return
			}
			offset := anchor.Offset
			*anchor = d.first(children)
			if *syntax < 0 {
				anchor.Offset = min(-*syntax-1, len(children[0].Text))
			} else if offset == 1 {
				*anchor = d.lastAnchor(children)
			}
			*syntax = 0
		}
		remap(&d.state.cursor, &d.state.syntax)
		if s := d.state.selection; s != nil {
			remap(&s.Start, &s.startSyntax)
			remap(&s.End, &s.endSyntax)
		}
		*list = slices.Replace(*list, i, i+1, children...)
		return nil
	}
	lo, hi, ext := d.selectionRange()
	for _, e := range ext {
		if e.node.Tag == "pre" && e.start < hi && e.end > lo {
			return fmt.Errorf("Links cannot include Code blocks")
		}
	}
	for _, e := range ext {
		if e.node.Tag == "a" && attributeValue(e.node.Attributes, "id") != "" && e.start < hi && e.end > lo && (e.start < lo || e.end > hi || containsOpaque(e.node)) {
			return fmt.Errorf("Cannot split a Link with a unique HTML ID")
		}
	}
	start, end := anchorPosition(d.state.selection.Start, ext), anchorPosition(d.state.selection.End, ext)
	var rewrite func([]Node) []Node
	rewrite = func(ns []Node) []Node {
		var out, inline []Node
		flush := func() {
			if len(inline) > 0 {
				out = append(out, d.formatInline(inline, Action{Kind: EditLink, Text: value}, lo, hi, ext, false)...)
				inline = nil
			}
		}
		for _, n := range ns {
			if n.Kind == Element && (isBlock(n.Tag) || n.Tag == "li") {
				flush()
				if n.Tag != "pre" {
					n.Children = rewrite(n.Children)
				}
				out = append(out, n)
			} else {
				inline = append(inline, n)
			}
		}
		flush()
		return out
	}
	d.state.nodes = rewrite(d.state.nodes)
	d.uniqueNativeIDs()
	d.assign(d.state.nodes)
	d.restoreSelection(start, end)
	return nil
}

func linkLayers(path []Node, value string) []Node {
	attrs := []Attribute{{"href", value}, {"target", "_blank"}}
	for _, p := range path {
		if p.Tag == "a" {
			for _, a := range p.Attributes {
				if a.Name != "href" && a.Name != "target" {
					attrs = append(attrs, a)
				}
			}
			break
		}
	}
	path = slices.DeleteFunc(path, func(p Node) bool { return p.Tag == "a" })
	if value != "" {
		path = append([]Node{{Kind: Element, Tag: "a", Attributes: attrs}}, path...)
	}
	return path
}

// At the exact boundary of an inline Code or Link, insertion belongs to the
// surrounding text. Other marks around that span remain intact.
func (d *Draft) insertInlineEdge(a Action) (bool, error) {
	if a.inside {
		return false, nil
	}
	n := findNode(d.state.nodes, d.state.cursor.Node)
	if n == nil {
		return false, nil
	}
	if n.Tag == "a" && len(n.Children) == 0 {
		value := attributeValue(n.Attributes, "href")
		text := d.newNode(Node{Kind: Text, Text: value})
		n.Children = []Node{text}
		d.state.cursor = Anchor{text.ID, len(value)}
		return false, nil
	}
	if (n.Tag == "a" || isInlineMark(n.Tag)) && len(n.Children) > 0 {
		list, i := nodeList(&d.state.nodes, n.ID)
		if d.state.cursor.Offset == 1 {
			i++
		}
		text := d.newNode(Node{Kind: Text, Text: a.Text})
		*list = slices.Insert(*list, i, text)
		d.state.cursor = Anchor{text.ID, len(a.Text)}
		return true, nil
	}
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return false, err
	}
	ext, _ := d.extents()
	pos := anchorPosition(d.state.cursor, ext)
	for _, id := range loc.Ancestors {
		p := findNode(d.state.nodes, id)
		if p.Tag != "a" {
			continue
		}
		e := ext[id]
		if pos != e.start && pos != e.end {
			continue
		}
		list, i := nodeList(&d.state.nodes, id)
		if pos == e.end {
			i++
		}
		text := d.newNode(Node{Kind: Text, Text: a.Text})
		*list = slices.Insert(*list, i, text)
		d.state.cursor = Anchor{text.ID, len(a.Text)}
		return true, nil
	}
	return false, nil
}

func (d *Draft) removeDeletedLinkLabels(before draftState, kind ActionKind) {
	var clean func([]Node) []Node
	clean = func(ns []Node) []Node {
		var out []Node
		for _, n := range ns {
			n.Children = clean(n.Children)
			old := findNode(before.nodes, n.ID)
			if n.Tag == "a" && contentNodes(n.Children) != "" {
				n.replacement = false
			}
			if n.Tag == "a" && old != nil && contentNodes(n.Children) == "" && !containsOpaque(n) && (len(old.Children) > 0 || before.syntax < 0 && before.cursor.Node == n.ID || before.selection != nil && before.selection.Start.Node == n.ID && before.selection.End.Node == n.ID) {
				if kind == ReplaceSelection {
					n.replacement = true
				} else {
					continue
				}
			}
			out = append(out, n)
		}
		return out
	}
	d.state.nodes = clean(d.state.nodes)
	if findNode(d.state.nodes, d.state.cursor.Node) == nil {
		d.state.cursor = d.first(d.state.nodes)
		d.state.syntax = 0
	}
}

// A pending c-replacement retains its insertion style while its empty label
// already persists as deleted. Leaving Insert releases that temporary context.
func (d *Draft) finalizeReplacementLinks() {
	var clean func([]Node) []Node
	clean = func(ns []Node) []Node {
		var out []Node
		for _, n := range ns {
			n.Children = clean(n.Children)
			if n.Tag == "a" && n.replacement {
				if contentNodes(n.Children) == "" && !containsOpaque(n) {
					out = append(out, n.Children...)
					continue
				}
				n.replacement = false
			}
			out = append(out, n)
		}
		return out
	}
	d.state.nodes = clean(d.state.nodes)
}
