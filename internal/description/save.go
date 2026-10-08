package description

import (
	"crypto/rand"
	"fmt"
	"regexp"
)

// SaveCandidate is immutable prepared HTML. Persistence must compare Baseline
// atomically and return success before the editor adopts HTML as its new source.
// Its zero value is invalid; only PrepareSave can construct a valid candidate.
type SaveCandidate struct {
	html, baseline string
	valid          bool
}

func (c SaveCandidate) HTML() string     { return c.html }
func (c SaveCandidate) Baseline() string { return c.baseline }
func (c SaveCandidate) Valid() bool      { return c.valid }

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// PrepareSave validates preservation before cloning, reserves all observable
// source and draft IDs, and repairs only Native Checkbox identities. A nil
// generator uses cryptographic UUID v4; injection is for deterministic testing.
func (d *Document) PrepareSave(nodes []Node, intent Preservation, generate func() (string, error)) (SaveCandidate, error) {
	if _, err := d.SerializeSnapshot(nodes, intent); err != nil {
		return SaveCandidate{}, err
	}
	nodes = cloneNodes(nodes)
	reserved := map[string]bool{}
	for _, id := range d.reserved {
		reserved[id] = true
	}
	var reserve func([]Node)
	reserve = func(ns []Node) {
		for _, n := range ns {
			for _, a := range n.Attributes {
				if htmlName(a.Name) == "data-task-id" && a.Value != "" {
					reserved[a.Value] = true
				}
			}
			reserve(n.Children)
		}
	}
	reserve(nodes)
	if generate == nil {
		generate = randomUUID
	}
	seen := map[string]bool{}
	var repair func([]Node) error
	repair = func(ns []Node) error {
		for i := range ns {
			n := &ns[i]
			if n.Kind == Element && n.Tag == "li" && attributeValue(n.Attributes, "data-type") == "taskItem" {
				id := attributeValue(n.Attributes, "data-task-id")
				if id == "" || seen[id] {
					found := false
					for attempt := 0; attempt < 128; attempt++ {
						v, err := generate()
						if err != nil {
							return err
						}
						if !uuidV4.MatchString(v) || reserved[v] {
							continue
						}
						id = v
						reserved[id] = true
						found = true
						break
					}
					if !found {
						return fmt.Errorf("cannot generate a distinct lowercase Checkbox UUID v4")
					}
					replaced := false
					for j := range n.Attributes {
						if htmlName(n.Attributes[j].Name) == "data-task-id" {
							n.Attributes[j].Value = id
							replaced = true
						}
					}
					if !replaced {
						n.Attributes = append(n.Attributes, Attribute{"data-task-id", id})
					}
				}
				seen[id] = true
			}
			if err := repair(n.Children); err != nil {
				return err
			}
		}
		return nil
	}
	if err := repair(nodes); err != nil {
		return SaveCandidate{}, err
	}
	output, err := d.SerializeSnapshot(nodes, intent)
	if err != nil {
		return SaveCandidate{}, err
	}
	return SaveCandidate{html: output, baseline: d.source, valid: true}, nil
}

func attributeValue(attrs []Attribute, name string) string {
	for _, a := range attrs {
		if htmlName(a.Name) == name {
			return a.Value
		}
	}
	return ""
}
func randomUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate Checkbox UUID: %w", err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
