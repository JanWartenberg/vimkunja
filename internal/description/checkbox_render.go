package description

import "slices"

// TipTap TaskItem serializes its checkbox control and a content div. Recognize
// only this exact, attribute-free wrapper shape; other forms remain Opaque.
func recognizeCheckboxRender(n *sourceNode) {
	if !n.opaque && n.tag == "li" && attributeValue(n.attrs, "data-type") == "taskItem" && len(n.children) >= 2 {
		label, body := n.children[0], n.children[1]
		if label.tag == "label" && body.tag == "div" && len(body.attrs) == 0 {
			body.opaque = false
		}
		if label.tag == "label" && len(label.attrs) == 0 && len(label.children) == 2 && body.tag == "div" && len(body.attrs) == 0 {
			input, span := label.children[0], label.children[1]
			valid := input.tag == "input" && len(input.children) == 0 && attributeValue(input.attrs, "type") == "checkbox" && span.tag == "span" && len(span.attrs) == 0 && len(span.children) == 0
			for _, a := range input.attrs {
				if a.Name != "type" && a.Name != "checked" {
					valid = false
				}
			}
			for _, extra := range n.children[2:] {
				if extra.opaque || (extra.tag != "ul" && extra.tag != "ol") {
					valid = false
				}
			}
			if valid && !unsafeAttributes(input.attrs) {
				label.opaque, body.opaque, input.opaque, span.opaque = false, false, false, false
			}
		}
	}
	for _, child := range n.children {
		recognizeCheckboxRender(child)
	}
}

func checkboxControlTag(tag string) bool { return tag == "label" || tag == "input" || tag == "span" }

func validCheckboxRender(n Node, parent string) bool {
	switch n.Tag {
	case "div":
		return parent == "li" && len(n.Attributes) == 0
	case "label":
		return parent == "li" && len(n.Attributes) == 0 && len(n.Children) == 2 && n.Children[0].Tag == "input" && n.Children[1].Tag == "span"
	case "span":
		return parent == "label" && len(n.Attributes) == 0 && len(n.Children) == 0
	case "input":
		if parent != "label" || len(n.Children) != 0 || attributeValue(n.Attributes, "type") != "checkbox" {
			return false
		}
		for _, a := range n.Attributes {
			if a.Name != "type" && a.Name != "checked" {
				return false
			}
		}
		return true
	}
	return false
}

func syncCheckboxControl(n *Node) {
	if n.Tag != "li" || attributeValue(n.Attributes, "data-type") != "taskItem" {
		return
	}
	for i := range n.Children {
		label := &n.Children[i]
		if label.Kind != Element || label.Tag != "label" || len(label.Children) != 2 || label.Children[0].Tag != "input" {
			continue
		}
		input := &label.Children[0]
		if attributeValue(n.Attributes, "data-checked") == "true" {
			setAttribute(input, "checked", "checked")
		} else {
			input.Attributes = slices.DeleteFunc(input.Attributes, func(a Attribute) bool { return a.Name == "checked" })
		}
	}
}

// The checkbox control is presentation scaffolding. Structural edits operate
// on its content; ordinary serialization still preserves the original wrappers.
func checkboxContent(n Node) []Node {
	if n.Kind != Element || n.Tag != "li" || attributeValue(n.Attributes, "data-type") != "taskItem" || len(n.Children) < 2 || n.Children[0].Kind != Element || n.Children[0].Tag != "label" || n.Children[1].Kind != Element || n.Children[1].Tag != "div" {
		return n.Children
	}
	out := slices.Clone(n.Children[1].Children)
	return append(out, n.Children[2:]...)
}
