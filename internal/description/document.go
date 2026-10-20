// Package description interprets Vikunja Description HTML without coupling it
// to a terminal, database, or editing mode. Opaque source is private and travels
// through serialization by identity, never through an editable HTML buffer.
package description

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type Classification string

const (
	Native  Classification = "Native"
	Mixed   Classification = "mixed Native/Opaque"
	Blocked Classification = "Blocked"
)

type ID uint64

type Kind uint8

const (
	Element Kind = iota
	Text
	Opaque
)

type Attribute struct{ Name, Value string }

// Node is a detached snapshot. Tag preserves semantic variants such as b/strong;
// Text contains decoded characters. Opaque nodes expose only their safe Preview.
// IDs remain stable within a Document; zero denotes a new Native node.
type Node struct {
	ID            ID
	Kind          Kind
	Tag           string
	Text          string
	Attributes    []Attribute
	Children      []Node
	Preview       string
	blockOpaque   bool
	typed         []byte // Draft-only origin flags: imported/pasted bytes are literal.
	syntaxSource  string
	sourceOrigins []byte
	replacement   bool
	dashes        int // Typed rule history; never persisted.
}

// Anchor addresses a decoded-text boundary, not a terminal coordinate. Offset
// is a byte boundary in UTF-8 Text, or 0/1 before/after a structural node;
// grapheme motions belong to the editor.
type Anchor struct {
	Node   ID
	Offset int
}

// Preservation declares the editing operation's intended atom removals/paths.
// Paths address canonical snapshot child indices from the document root. The
// caller owns confirmation; declaring a path is not a substitute for consent.
type Preservation struct {
	Removed   []ID
	Positions map[ID][]int
}

var (
	ErrBlocked      = errors.New("Error: Malformed HTML")
	ErrPreservation = errors.New("Description Opaque preservation failed")
	ErrStructure    = errors.New("invalid Native Description structure")
	ErrAnchor       = errors.New("invalid Description cursor anchor")
	ErrRoundTrip    = errors.New("Description Native meaning changed during serialization")
)

type atom struct {
	id      ID
	raw     string
	path    []int
	parents []ID
	preview string
	text    string
}

type Document struct {
	source    string
	nodes     []Node
	class     Classification
	problem   error
	atoms     []atom
	byID      map[ID]int
	locations map[ID]Location
	// IDs in source, including observable Opaque attributes, are reserved by
	// the save-candidate layer. Parsing never generates/replaces IDs.
	reserved []string
	recovery string
}

func (d *Document) Classification() Classification { return d.class }
func (d *Document) Problem() error                 { return d.problem }
func (d *Document) SourceHTML() string             { return d.source }
func (d *Document) RecoveryText() string           { return d.recovery }
func (d *Document) ReservedIDs() []string          { return slices.Clone(d.reserved) }

func (d *Document) Snapshot() []Node { return cloneNodes(d.nodes) }

func cloneNodes(nodes []Node) []Node {
	if nodes == nil {
		return nil
	}
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		out[i] = n
		out[i].typed = slices.Clone(n.typed)
		out[i].sourceOrigins = slices.Clone(n.sourceOrigins)
		out[i].Attributes = slices.Clone(n.Attributes)
		out[i].Children = cloneNodes(n.Children)
	}
	return out
}

// Serialize writes this document's snapshot. It does not save to a database or
// repair Checkbox IDs; use PrepareSave and Store persistence for those operations.
func (d *Document) Serialize() (string, error) {
	return d.SerializeSnapshot(d.Snapshot(), Preservation{})
}

// SerializeSnapshot is the model's draft seam: Native snapshots may change,
// while retained Opaque content is resolved from this document's private source.
// Neither the input nor this document is mutated on success or failure.
func (d *Document) SerializeSnapshot(nodes []Node, intent Preservation) (string, error) {
	if d.class == Blocked {
		return "", d.problem
	}
	if err := validateNative(nodes, "", false); err != nil {
		return "", err
	}
	// Check the original draft: normalization may merge or remove nodes and
	// must not hide repeated identities or an atom disguised as Native.
	if err := d.validateIdentities(nodes, make(map[ID]bool)); err != nil {
		return "", err
	}
	nodes = cloneNodes(nodes)
	nodes = normalizeRoot(nodes)
	removed := make(map[ID]bool, len(intent.Removed))
	for _, id := range intent.Removed {
		if _, ok := d.byID[id]; !ok || removed[id] {
			return "", fmt.Errorf("%w: unknown or repeated removal %d", ErrPreservation, id)
		}
		removed[id] = true
	}
	for id := range intent.Positions {
		if _, ok := d.byID[id]; !ok || removed[id] {
			return "", fmt.Errorf("%w: invalid position intent %d", ErrPreservation, id)
		}
	}
	var output strings.Builder
	var retained []atom
	nativeIDs := map[ID]bool{}
	if err := d.writeNodes(&output, nodes, nil, nil, removed, intent.Positions, &retained, nativeIDs); err != nil {
		return "", err
	}
	// Identity order cannot be inferred from bytes when two atoms are identical.
	expected := make([]ID, 0, len(d.atoms))
	for _, a := range d.atoms {
		if !removed[a.id] {
			expected = append(expected, a.id)
		}
	}
	if len(retained) != len(expected) {
		return "", fmt.Errorf("%w: an atom was omitted/duplicated without intent", ErrPreservation)
	}
	for i, a := range retained {
		if a.id != expected[i] {
			return "", fmt.Errorf("%w: atom identity/order changed", ErrPreservation)
		}
	}
	result := output.String()
	// One independent structural reparse per candidate checks emitted boundaries,
	// Opaque bytes and intended nesting/position, never per editing keystroke.
	parsed := Parse(result)
	if parsed.class == Blocked || len(parsed.atoms) != len(retained) {
		return "", fmt.Errorf("%w: serialized structure cannot be reparsed", ErrPreservation)
	}
	for i, a := range retained {
		other := parsed.atoms[i]
		if a.raw != other.raw || !slices.Equal(a.path, other.path) {
			return "", fmt.Errorf("%w: atom bytes or parent position changed", ErrPreservation)
		}
	}
	if !sameMeaning(nodes, parsed.nodes) {
		return "", ErrRoundTrip
	}
	return result, nil
}

func (d *Document) validateIdentities(nodes []Node, seen map[ID]bool) error {
	for _, n := range nodes {
		if n.ID != 0 && n.Kind != Opaque {
			if _, collision := d.byID[n.ID]; collision {
				return fmt.Errorf("%w: atom cannot become Native", ErrPreservation)
			}
			if seen[n.ID] {
				return fmt.Errorf("%w: repeated Native identity", ErrStructure)
			}
			seen[n.ID] = true
		}
		if err := d.validateIdentities(n.Children, seen); err != nil {
			return err
		}
	}
	return nil
}

func (d *Document) writeNodes(out *strings.Builder, nodes []Node, parent []int, parents []ID, removed map[ID]bool, positions map[ID][]int, retained *[]atom, nativeIDs map[ID]bool) error {
	for i, n := range nodes {
		path := append(slices.Clone(parent), i)
		if n.Tag == "a" && n.replacement && contentNodes(n.Children) == "" && !containsOpaque(n) {
			continue
		}
		if n.Kind == Opaque {
			index, ok := d.byID[n.ID]
			if !ok || removed[n.ID] || n.Tag != "" || n.Text != "" || len(n.Children) != 0 || len(n.Attributes) != 0 {
				return fmt.Errorf("%w: edited/unknown/removed atom %d", ErrPreservation, n.ID)
			}
			a := d.atoms[index]
			want := a.path
			if intended, ok := positions[n.ID]; ok {
				want = intended
			} else if !slices.Equal(a.parents, parents) {
				return fmt.Errorf("%w: undeclared atom reparenting %d", ErrPreservation, n.ID)
			}
			if !slices.Equal(want, path) {
				return fmt.Errorf("%w: undeclared atom movement %d", ErrPreservation, n.ID)
			}
			a.path = path
			*retained = append(*retained, a)
			out.WriteString(a.raw)
			continue
		}
		if n.ID != 0 {
			if nativeIDs[n.ID] {
				return fmt.Errorf("%w: repeated Native identity", ErrStructure)
			}
			if _, collision := d.byID[n.ID]; collision {
				return fmt.Errorf("%w: atom cannot become Native", ErrPreservation)
			}
			nativeIDs[n.ID] = true
		}
		if n.Kind == Text {
			out.WriteString(escapeText(n.Text))
			continue
		}
		out.WriteByte('<')
		out.WriteString(n.Tag)
		for _, attr := range n.Attributes {
			out.WriteByte(' ')
			out.WriteString(attr.Name)
			if n.Tag != "ol" || attr.Name != "reversed" {
				out.WriteString("=\"")
				out.WriteString(escapeAttribute(attr.Value))
				out.WriteByte('"')
			}
		}
		out.WriteByte('>')
		if err := d.writeNodes(out, n.Children, path, append(slices.Clone(parents), n.ID), removed, positions, retained, nativeIDs); err != nil {
			return err
		}
		if !voidTags[n.Tag] {
			out.WriteString("</")
			out.WriteString(n.Tag)
			out.WriteByte('>')
		}
	}
	return nil
}

func escapeText(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
}

func escapeAttribute(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "\r", "&#13;").Replace(text)
}
