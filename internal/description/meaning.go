package description

// Compare the canonical intended snapshot with the single save-boundary
// reparse. IDs/source spelling may differ; Native tags, text, depth, block
// boundaries and attributes must not. Opaque identity/bytes/paths are checked
// separately against private source, never inferred from display text.
func sameMeaning(want, got []Node) bool {
	if len(want) != len(got) {
		return false
	}
	for i, a := range want {
		b := got[i]
		if a.Kind != b.Kind {
			return false
		}
		if a.Kind == Opaque {
			if a.blockOpaque != b.blockOpaque {
				return false
			}
			continue
		}
		if a.Tag != b.Tag || a.Text != b.Text || !sameAttributes(a.Tag, a.Attributes, b.Attributes) || !sameMeaning(a.Children, b.Children) {
			return false
		}
	}
	return true
}

func sameAttributes(tag string, want, got []Attribute) bool {
	if len(want) != len(got) {
		return false
	}
	values := make(map[string]string, len(got))
	for _, a := range got {
		values[htmlName(a.Name)] = a.Value
	}
	for _, a := range want {
		name := htmlName(a.Name)
		value, ok := values[name]
		if !ok || !(tag == "ol" && name == "reversed") && value != a.Value {
			return false
		}
	}
	return true
}
