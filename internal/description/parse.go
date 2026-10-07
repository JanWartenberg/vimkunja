package description

import (
	"fmt"
	stdhtml "html"
	"io"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

type sourceNode struct {
	id       ID
	tag      string
	text     string
	attrs    []Attribute
	children []*sourceNode
	start    int
	end      int
	opaque   bool
	active   bool
	comment  bool
}

var nativeTags = map[string]bool{
	"p": true, "h1": true, "h2": true, "h3": true, "strong": true,
	"b": true, "em": true, "i": true, "u": true, "s": true,
	"strike": true, "code": true, "pre": true, "br": true, "hr": true,
	"ul": true, "ol": true, "li": true, "blockquote": true, "a": true,
}

var voidTags = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

func isBlock(tag string) bool {
	switch tag {
	case "p", "h1", "h2", "h3", "pre", "blockquote", "ul", "ol", "hr", "table", "div":
		return true
	}
	return false
}

func optionalEnd(tag string) bool { return tag == "p" || tag == "li" }

// Parse always returns a document, including a Blocked one with original source
// and safe recovery text. It never uses a repairing HTML tree as its source of
// Native boundaries. Token.Raw is copied before Token mutates its buffer.
func Parse(source string) *Document {
	d := &Document{source: source, class: Native, byID: map[ID]int{}}
	root := &sourceNode{}
	stack := []*sourceNode{root}
	z := html.NewTokenizer(strings.NewReader(source))
	offset := 0
	next := ID(1)
	var problem error
	if !utf8.ValidString(source) {
		problem = fmt.Errorf("%w: invalid UTF-8", ErrBlocked)
	}
	fail := func(reason string) {
		if problem == nil {
			problem = fmt.Errorf("%w: %s", ErrBlocked, reason)
		}
	}
	for {
		kind := z.Next()
		raw := string(z.Raw())
		start := offset
		offset += len(raw)
		if kind == html.ErrorToken {
			if z.Err() != io.EOF {
				fail("tokenizer error")
			}
			if raw != "" {
				fail("incomplete markup")
			}
			break
		}
		token := z.Token()
		parent := stack[len(stack)-1]
		switch kind {
		case html.TextToken:
			// Token.Data loses CRLF. Decode the immutable Raw copy instead.
			text := stdhtml.UnescapeString(raw)
			if !rawTextElement(parent.tag) && looksIncompleteTag(raw) {
				fail("incomplete tag")
			}
			parent.children = append(parent.children, &sourceNode{id: next, text: text, start: start, end: offset})
			next++
		case html.CommentToken:
			if !strings.HasSuffix(raw, "-->") {
				fail("incomplete comment")
			}
			parent.children = append(parent.children, &sourceNode{id: next, tag: "#comment", comment: true, opaque: true, start: start, end: offset})
			next++
		case html.DoctypeToken:
			fail("document wrapper in Description fragment")
		case html.StartTagToken, html.SelfClosingTagToken:
			if token.Data == "html" || token.Data == "head" || token.Data == "body" {
				fail("document wrapper in Description fragment")
			}
			// HTML optional ends are accepted only at unambiguous boundaries;
			// unclosed marks and other repair are not silently popped.
			if parent.tag == "p" && (isBlock(token.Data) || token.Data == "li") {
				parent.end = start
				stack = stack[:len(stack)-1]
				parent = stack[len(stack)-1]
			}
			if token.Data == "li" && parent.tag == "li" {
				parent.end = start
				stack = stack[:len(stack)-1]
				parent = stack[len(stack)-1]
			}
			n := &sourceNode{id: next, tag: token.Data, start: start, end: offset}
			next++
			// Token.Attr has already discarded duplicates in x/net/html.
			// Read source spelling before deciding whether Native is safe.
			for _, a := range sourceAttributes(raw) {
				n.attrs = append(n.attrs, a)
				if a.Name == "data-task-id" && a.Value != "" {
					d.reserved = append(d.reserved, a.Value)
				}
			}
			n.active = token.Data == "script" || activeAttributes(n.attrs)
			n.opaque = !nativeTags[n.tag] || unsafeAttributes(n.attrs) || len(stack) >= 512
			parent.children = append(parent.children, n)
			if !voidTags[n.tag] {
				stack = append(stack, n)
			}
		case html.EndTagToken:
			for len(stack) > 1 && stack[len(stack)-1].tag != token.Data && optionalEnd(stack[len(stack)-1].tag) {
				n := stack[len(stack)-1]
				if n.tag == "li" && token.Data != "ul" && token.Data != "ol" {
					break
				}
				n.end = start
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 1 || stack[len(stack)-1].tag != token.Data {
				fail("mismatched or unexpected closing tag")
				// Continue collecting safe text for recovery; no repaired Native
				// document is ever serialized after this failure.
				continue
			}
			stack[len(stack)-1].end = offset
			stack = stack[:len(stack)-1]
		}
	}
	for len(stack) > 1 {
		n := stack[len(stack)-1]
		n.end = len(source)
		if !optionalEnd(n.tag) {
			fail("unclosed element")
		}
		stack = stack[:len(stack)-1]
	}
	d.recovery = visibleSource(root, false)
	if problem != nil {
		d.class, d.problem = Blocked, problem
		return d
	}
	for _, n := range root.children {
		protectUnsupportedStructure(n, "")
	}
	var converted func(*sourceNode) Node
	converted = func(n *sourceNode) Node {
		if n.opaque {
			d.class = Mixed
			a := atom{id: n.id, raw: source[n.start:n.end], preview: visibleSource(n, true), text: visibleSource(n, false)}
			d.byID[a.id] = len(d.atoms)
			d.atoms = append(d.atoms, a)
			return Node{ID: n.id, Kind: Opaque, Preview: a.preview, blockOpaque: !inlineOpaque(n.tag)}
		}
		if n.tag == "" {
			return Node{ID: n.id, Kind: Text, Text: n.text}
		}
		out := Node{ID: n.id, Kind: Element, Tag: n.tag, Attributes: slices.Clone(n.attrs)}
		for _, child := range n.children {
			out.Children = append(out.Children, converted(child))
		}
		return out
	}
	for _, n := range root.children {
		d.nodes = append(d.nodes, converted(n))
	}
	d.nodes = normalizeRoot(d.nodes)
	if err := validateNative(d.nodes, "", false); err != nil {
		d.class, d.problem = Blocked, fmt.Errorf("%w: %v", ErrBlocked, err)
		return d
	}
	var register func([]Node, []int, []ID)
	register = func(nodes []Node, path []int, parents []ID) {
		for i, n := range nodes {
			p := append(slices.Clone(path), i)
			if n.Kind == Opaque {
				d.atoms[d.byID[n.ID]].path = p
				d.atoms[d.byID[n.ID]].parents = slices.Clone(parents)
			} else {
				register(n.Children, p, append(slices.Clone(parents), n.ID))
			}
		}
	}
	register(d.nodes, nil, nil)
	return d
}

func rawTextElement(tag string) bool {
	switch tag {
	case "script", "style", "textarea", "title", "xmp", "iframe", "noembed", "noframes":
		return true
	}
	return false
}

// A reliably bounded but unsupported Native shape becomes one local atom;
// it must not disable the whole Description or lose its safe attributes.
func protectUnsupportedStructure(n *sourceNode, parent string) {
	if n.opaque || n.tag == "" {
		return
	}
	if n.tag == "li" && parent != "ul" && parent != "ol" {
		n.opaque = true
		return
	}
	for _, child := range n.children {
		if n.tag == "ul" || n.tag == "ol" {
			if child.tag != "li" && (child.tag != "" || strings.Trim(child.text, " \t\r\n\f") != "") {
				n.opaque = true
				return
			}
		}
		if n.tag == "pre" {
			if child.tag != "" && child.tag != "code" {
				n.opaque = true
				return
			}
			for _, text := range child.children {
				if text.tag != "" {
					n.opaque = true
					return
				}
			}
		}
		protectUnsupportedStructure(child, n.tag)
		if n.tag == "a" && containsLink(child) {
			n.opaque = true
			return
		}
	}
}

// Nested anchors are invalid even through marks or an Opaque descendant. Keep
// the bounded outer Link as one atom instead of disabling Native neighbors.
func containsLink(n *sourceNode) bool {
	pending := []*sourceNode{n}
	for len(pending) > 0 {
		child := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if child.tag == "a" {
			return true
		}
		pending = append(pending, child.children...)
	}
	return false
}

func sourceAttributes(raw string) []Attribute {
	i := 1
	for i < len(raw) && !htmlSpace(raw[i]) && raw[i] != '>' && raw[i] != '/' {
		i++
	}
	var attrs []Attribute
	for i < len(raw) {
		for i < len(raw) && htmlSpace(raw[i]) {
			i++
		}
		if i >= len(raw) || raw[i] == '>' {
			break
		}
		if raw[i] == '/' {
			if i+1 < len(raw) && raw[i+1] == '>' {
				break
			}
			attrs = append(attrs, Attribute{Name: "/"})
			i++
			continue
		}
		start := i
		for i < len(raw) && !htmlSpace(raw[i]) && raw[i] != '=' && raw[i] != '>' && raw[i] != '/' {
			i++
		}
		name := strings.ToLower(raw[start:i])
		for i < len(raw) && htmlSpace(raw[i]) {
			i++
		}
		value := ""
		if i < len(raw) && raw[i] == '=' {
			i++
			for i < len(raw) && htmlSpace(raw[i]) {
				i++
			}
			encodedStart := i
			if i < len(raw) && (raw[i] == '\'' || raw[i] == '"') {
				quote := raw[i]
				i++
				start = i
				for i < len(raw) && raw[i] != quote {
					i++
				}
				value = raw[start:i]
				if i < len(raw) {
					i++
				}
			} else {
				start = i
				for i < len(raw) && !htmlSpace(raw[i]) && raw[i] != '>' {
					i++
				}
				value = raw[start:i]
			}
			// Decode one value in attribute context. Text unescaping differs
			// for ambiguous references such as &notit;. Separate tokens retain
			// the values of duplicate attributes instead of dropping them.
			z := html.NewTokenizer(strings.NewReader("<i x=" + raw[encodedStart:i] + ">"))
			if z.Next() == html.StartTagToken {
				t := z.Token()
				if len(t.Attr) != 0 {
					value = t.Attr[0].Val
				}
			}
		}
		attrs = append(attrs, Attribute{name, value})
	}
	return attrs
}

func looksIncompleteTag(text string) bool {
	index := strings.LastIndexByte(text, '<')
	if index < 0 || strings.Contains(text[index:], ">") || index+1 == len(text) {
		return false
	}
	c := text[index+1]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '/' || c == '!'
}

func inlineOpaque(tag string) bool {
	switch tag {
	case "img", "span", "q", "mention-user", "a", "#comment":
		return true
	}
	return false
}

func activeAttributes(attrs []Attribute) bool {
	for _, a := range attrs {
		name := strings.ToLower(a.Name)
		if strings.HasPrefix(name, "on") || name == "srcdoc" {
			return true
		}
		if name == "href" || name == "src" || name == "action" || name == "formaction" || name == "xlink:href" {
			v := strings.Map(func(r rune) rune {
				if unicode.IsSpace(r) || unicode.IsControl(r) {
					return -1
				}
				return unicode.ToLower(r)
			}, a.Value)
			if strings.HasPrefix(v, "javascript:") || strings.HasPrefix(v, "vbscript:") || strings.HasPrefix(v, "data:") {
				return true
			}
		}
	}
	return false
}

func unsafeAttributes(attrs []Attribute) bool {
	if activeAttributes(attrs) {
		return true
	}
	seen := map[string]bool{}
	for _, a := range attrs {
		name := strings.ToLower(a.Name)
		if name == "style" || name == "" || seen[name] || strings.ContainsAny(name, " \t\r\n\f\x00\"'<>/=") {
			return true
		}
		seen[name] = true
	}
	return false
}
