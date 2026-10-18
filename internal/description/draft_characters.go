package description

import (
	"github.com/rivo/uniseg"
	"strings"
	"unicode"
)

// Character transforms preserve inline wrappers and stop at a line, block or
// protected object. Apply owns atomic validation, rollback and the Undo entry.
func (d *Draft) transformCharacters(a Action) error {
	if a.Kind == ReplaceCharacters && (uniseg.GraphemeClusterCount(a.Text) != 1 || strings.ContainsAny(a.Text, "\r\n")) {
		return ErrStructure
	}
	if d.state.syntax != 0 {
		return ErrStructure
	}
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return err
	}
	block := findNode(d.state.nodes, loc.Block)
	if block == nil {
		return ErrStructure
	}
	original := d.state.cursor
	remaining := max(1, min(a.Count, 100000))
	started, stopped := false, false
	var walk func([]Node)
	walk = func(nodes []Node) {
		for i := range nodes {
			n := &nodes[i]
			if stopped || remaining == 0 {
				return
			}
			if n.ID == original.Node {
				started = true
			}
			if n.Kind == Opaque || n.Tag == "br" {
				if started {
					stopped = true
				}
				continue
			}
			if n.Kind != Text {
				if started && isBlock(n.Tag) {
					stopped = true
					return
				}
				walk(n.Children)
				continue
			}
			if !started {
				continue
			}
			start := 0
			if n.ID == original.Node {
				start = original.Offset
			}
			old := n.Text
			flags := textOrigins(*n)
			end := start
			var changed strings.Builder
			it := uniseg.NewGraphemes(old[start:])
			for remaining > 0 && it.Next() {
				g := it.Str()
				if strings.ContainsAny(g, "\r\n") {
					stopped = true
					break
				}
				replacement := a.Text
				if a.Kind == ToggleCase {
					replacement = strings.Map(func(r rune) rune {
						if unicode.IsUpper(r) || unicode.IsTitle(r) {
							return unicode.ToLower(r)
						}
						if unicode.IsLower(r) {
							return unicode.ToUpper(r)
						}
						return r
					}, g)
				}
				changed.WriteString(replacement)
				end += len(g)
				remaining--
			}
			if end > start {
				replacement := changed.String()
				n.Text = old[:start] + replacement + old[end:]
				n.typed = append(append(append([]byte(nil), flags[:start]...), make([]byte, len(replacement))...), flags[end:]...)
				if a.Kind == ToggleCase {
					d.state.cursor = Anchor{n.ID, start + len(replacement)}
				}
			}
		}
	}
	walk(block.Children)
	return nil
}
