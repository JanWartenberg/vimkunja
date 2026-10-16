package description

import (
	"fmt"
	"strings"
)

func normalizeRoot(nodes []Node) []Node {
	nodes = normalizeNodes(nodes, false)
	var out, inline []Node
	flush := func() {
		if len(inline) != 0 {
			p := Node{Kind: Element, Tag: "p", Children: inline}
			normalizeFlow(p.Children)
			p.Children = removeEmptyText(p.Children)
			if len(p.Children) != 0 {
				out = append(out, p)
			}
			inline = nil
		}
	}
	for _, n := range nodes {
		// Opaque block/inline status is determined from source at import; its
		// safe Preview alone must not determine structural placement.
		if n.Kind == Opaque && n.blockOpaque {
			flush()
			out = append(out, n)
		} else if n.Kind == Element && isBlock(n.Tag) {
			flush()
			out = append(out, n)
		} else {
			inline = append(inline, n)
		}
	}
	flush()
	return mergeText(out)
}

func normalizeNodes(nodes []Node, code bool) []Node {
	var out []Node
	for _, n := range nodes {
		if n.Kind == Element {
			for i := range n.Attributes {
				n.Attributes[i].Name = htmlName(n.Attributes[i].Name)
			}
			if n.Tag == "strike" {
				n.Tag = "s"
			}
			if n.Tag == "a" {
				found := false
				for i := range n.Attributes {
					if n.Attributes[i].Name == "target" {
						n.Attributes[i].Value = "_blank"
						found = true
					}
				}
				if !found {
					pos := len(n.Attributes)
					for i, a := range n.Attributes {
						if a.Name == "href" {
							pos = i + 1
							break
						}
					}
					n.Attributes = append(n.Attributes, Attribute{})
					copy(n.Attributes[pos+1:], n.Attributes[pos:])
					n.Attributes[pos] = Attribute{"target", "_blank"}
				}
			}
			n.Children = normalizeNodes(n.Children, code || n.Tag == "pre")
			syncCheckboxControl(&n)
			if n.Tag == "a" && n.replacement && len(n.Children) == 0 {
				continue
			}
			if !code && (n.Tag == "p" || n.Tag == "h1" || n.Tag == "h2" || n.Tag == "h3" || n.Tag == "li") {
				normalizeFlow(n.Children)
				n.Children = removeEmptyText(n.Children)
			}
			if !code && isInlineMark(n.Tag) && len(n.Children) == 0 {
				continue
			}
		}
		if n.Kind == Text && n.Text == "" {
			continue
		}
		out = append(out, n)
	}
	return mergeText(out)
}

func mergeText(nodes []Node) []Node {
	var out []Node
	for i := 0; i < len(nodes); {
		n := nodes[i]
		i++
		if n.Kind == Text {
			parts := []string{n.Text}
			for i < len(nodes) && nodes[i].Kind == Text {
				parts = append(parts, nodes[i].Text)
				i++
			}
			n.Text = strings.Join(parts, "")
		}
		out = append(out, n)
	}
	return out
}

func isInlineMark(tag string) bool {
	switch tag {
	case "strong", "b", "em", "i", "u", "s", "strike", "code":
		return true
	}
	return false
}

func removeEmptyText(nodes []Node) []Node {
	var out []Node
	for _, n := range nodes {
		if n.Kind == Element && n.Tag != "pre" {
			n.Children = removeEmptyText(n.Children)
			if isInlineMark(n.Tag) && len(n.Children) == 0 {
				continue
			}
		}
		if n.Kind != Text || n.Text != "" {
			out = append(out, n)
		}
	}
	return mergeText(out)
}

func htmlSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func collapseSpace(s string) string {
	var out strings.Builder
	space := false
	for i := 0; i < len(s); i++ {
		if htmlSpace(s[i]) {
			if !space {
				out.WriteByte(' ')
			}
			space = true
		} else {
			out.WriteByte(s[i])
			space = false
		}
	}
	return out.String()
}

func normalizeFlow(nodes []Node) {
	var flow []*Node
	var collect func([]Node)
	collect = func(nodes []Node) {
		for i := range nodes {
			n := &nodes[i]
			if n.Kind == Text || n.Kind == Opaque || n.Tag == "br" || isBlock(n.Tag) || n.Tag == "a" && len(n.Children) == 0 {
				flow = append(flow, n)
			} else {
				collect(n.Children)
			}
		}
	}
	collect(nodes)
	started, lastSpace := false, false
	for i, n := range flow {
		if n.Kind != Text {
			started, lastSpace = true, false
			continue
		}
		beforeBR := i+1 < len(flow) && flow[i+1].Tag == "br"
		afterBR := i > 0 && flow[i-1].Tag == "br"
		s := n.Text
		lo, hi := 0, len(s)
		if afterBR {
			for lo < hi && htmlSpace(s[lo]) {
				lo++
			}
		}
		if beforeBR {
			for hi > lo && htmlSpace(s[hi-1]) {
				hi--
			}
		}
		mid := collapseSpace(s[lo:hi])
		if !afterBR && (!started || lastSpace) {
			mid = strings.TrimPrefix(mid, " ")
		}
		if i == len(flow)-1 && !beforeBR {
			mid = strings.TrimSuffix(mid, " ")
		}
		n.Text = s[:lo] + mid + s[hi:]
		if n.Text != "" {
			started = true
			lastSpace = htmlSpace(n.Text[len(n.Text)-1])
		}
	}
}

func validateNative(nodes []Node, parent string, linked bool) error {
	return validateAtDepth(nodes, parent, linked, 0)
}

func validateAtDepth(nodes []Node, parent string, linked bool, depth int) error {
	if len(nodes) != 0 && depth >= 512 {
		return fmt.Errorf("%w: Native nesting exceeds import limit", ErrStructure)
	}
	for _, n := range nodes {
		if n.Kind == Opaque {
			if n.Tag != "" || n.Text != "" || len(n.Attributes) != 0 || len(n.Children) != 0 {
				return fmt.Errorf("%w: atom has editable structure", ErrPreservation)
			}
			continue
		}
		if n.Kind == Text {
			if n.Tag != "" || len(n.Children) != 0 || len(n.Attributes) != 0 {
				return fmt.Errorf("%w: misplaced text", ErrStructure)
			}
			if (parent == "ul" || parent == "ol") && strings.Trim(n.Text, " \t\r\n\f") != "" {
				return fmt.Errorf("%w: text outside list entry", ErrStructure)
			}
			continue
		}
		if n.Kind != Element || (!nativeTags[n.Tag] && !validCheckboxRender(n, parent)) || unsafeAttributes(n.Attributes) || n.Text != "" {
			return fmt.Errorf("%w: unsupported element/attributes", ErrStructure)
		}
		if (parent == "ul" || parent == "ol") && n.Tag != "li" || n.Tag == "li" && parent != "ul" && parent != "ol" {
			return fmt.Errorf("%w: list entry/container mismatch", ErrStructure)
		}
		if (parent == "p" || parent == "h1" || parent == "h2" || parent == "h3" || isInlineMark(parent) || parent == "a") && isBlock(n.Tag) {
			return fmt.Errorf("%w: block inside inline content", ErrStructure)
		}
		if n.Tag == "a" && linked {
			return fmt.Errorf("%w: nested Link", ErrStructure)
		}
		if (n.Tag == "br" || n.Tag == "hr") && len(n.Children) != 0 {
			return fmt.Errorf("%w: void element has children", ErrStructure)
		}
		if err := validateAtDepth(n.Children, n.Tag, linked || n.Tag == "a", depth+1); err != nil {
			return err
		}
	}
	return nil
}
