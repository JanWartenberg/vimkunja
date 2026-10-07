package description

import (
	"errors"
	"reflect"
	"testing"
)

func TestDraftIdentitiesAreCheckedBeforeNormalization(t *testing.T) {
	d := Parse("<p>A<img alt='atom'></p>")
	for _, tc := range []struct {
		name string
		node Node
		want error
	}{
		{"merged duplicate", Node{ID: d.Snapshot()[0].Children[0].ID, Kind: Text, Text: "B"}, ErrStructure},
		{"discarded duplicate", Node{ID: d.Snapshot()[0].ID, Kind: Element, Tag: "strong"}, ErrStructure},
		{"discarded atom identity", Node{ID: d.Snapshot()[0].Children[1].ID, Kind: Text}, ErrPreservation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes := d.Snapshot()
			nodes[0].Children = append([]Node{tc.node}, nodes[0].Children...)
			if _, err := d.SerializeSnapshot(nodes, Preservation{}); !errors.Is(err, tc.want) {
				t.Fatalf("invalid draft identity accepted: %v", err)
			}
		})
	}
}

func TestImportedIdentitiesAreUniqueAndStable(t *testing.T) {
	d := Parse("A<img alt='one'><p>B</p>C<img alt='two'><blockquote><p></p></blockquote>")
	seen := map[ID]bool{}
	var visit func([]Node)
	visit = func(nodes []Node) {
		for _, n := range nodes {
			if n.ID == 0 || seen[n.ID] {
				t.Fatalf("missing/duplicate imported identity %d", n.ID)
			}
			seen[n.ID] = true
			visit(n.Children)
		}
	}
	first := d.Snapshot()
	visit(first)
	if !reflect.DeepEqual(first, d.Snapshot()) {
		t.Fatal("snapshot changed identities")
	}
	for i := 0; i < 2; i++ {
		if _, err := d.Serialize(); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(first, d.Snapshot()) {
		t.Fatal("serialization changed imported identities")
	}
}

func TestAnchorResolutionUsesDocumentBoundaries(t *testing.T) {
	d := Parse("<blockquote><ul><li><p>é猫<br><img alt='map'></p><pre><code>\nA\r\n\tB\n</code></pre></li></ul></blockquote><p></p><custom-node>X</custom-node>")
	nodes := d.Snapshot()
	entry := nodes[0].Children[0].Children[0]
	paragraph := entry.Children[0]
	text := paragraph.Children[0]
	for _, offset := range []int{0, 2, 5} {
		got, err := d.ResolveAnchor(Anchor{Node: text.ID, Offset: offset})
		if err != nil || got.Offset != offset || got.Node.Text != "é猫" || got.Block != paragraph.ID {
			t.Fatalf("text anchor: %#v, %v", got, err)
		}
		wantParents := []ID{nodes[0].ID, nodes[0].Children[0].ID, entry.ID, paragraph.ID}
		if !reflect.DeepEqual(got.Ancestors, wantParents) {
			t.Fatalf("wrong ancestors: %v", got.Ancestors)
		}
		got.Ancestors[0] = 0
		got.Node.Text = "mutated"
	}
	for _, offset := range []int{-1, 1, 3, 4, 6} {
		if _, err := d.ResolveAnchor(Anchor{text.ID, offset}); !errors.Is(err, ErrAnchor) {
			t.Fatalf("invalid UTF-8 offset %d accepted: %v", offset, err)
		}
	}
	code := entry.Children[1]
	codeText := code.Children[0].Children[0]
	got, err := d.ResolveAnchor(Anchor{codeText.ID, 4})
	if err != nil || got.Block != code.ID || got.Node.Text != "\nA\r\n\tB\n" {
		t.Fatalf("Code anchor: %#v %v", got, err)
	}
	for _, n := range []Node{paragraph.Children[1], paragraph.Children[2], nodes[1], nodes[2]} {
		for _, offset := range []int{0, 1} {
			if _, err := d.ResolveAnchor(Anchor{n.ID, offset}); err != nil {
				t.Fatalf("structural anchor: %v", err)
			}
		}
		if _, err := d.ResolveAnchor(Anchor{n.ID, 2}); !errors.Is(err, ErrAnchor) {
			t.Fatalf("internal structural position accepted: %v", err)
		}
	}
	if got, err := d.ResolveAnchor(Anchor{nodes[2].ID, 0}); err != nil || got.Block != nodes[2].ID {
		t.Fatalf("standalone Opaque block: %#v %v", got, err)
	}
	for _, id := range []ID{0, 9999} {
		if _, err := d.ResolveAnchor(Anchor{id, 0}); !errors.Is(err, ErrAnchor) {
			t.Fatalf("unknown node accepted: %v", err)
		}
	}
	if _, err := Parse("<strong>open").ResolveAnchor(Anchor{1, 0}); !errors.Is(err, ErrBlocked) {
		t.Fatalf("Blocked anchor accepted: %v", err)
	}
	got, err = d.ResolveAnchor(Anchor{paragraph.ID, 0})
	if err != nil {
		t.Fatal(err)
	}
	got.Node.Children[0].Text = "mutated"
	if d.Snapshot()[0].Children[0].Children[0].Children[0].Children[0].Text != "é猫" {
		t.Fatal("resolved node aliases original")
	}
}

func TestDraftNestingAndCyclesAreRejectedBeforeCloning(t *testing.T) {
	d := Parse("")
	n := Node{Kind: Element, Tag: "blockquote"}
	for i := 0; i < 600; i++ {
		n = Node{Kind: Element, Tag: "blockquote", Children: []Node{n}}
	}
	if _, err := d.SerializeSnapshot([]Node{n}, Preservation{}); !errors.Is(err, ErrStructure) {
		t.Fatalf("excessive depth: %v", err)
	}
	cycle := []Node{{Kind: Element, Tag: "blockquote"}}
	cycle[0].Children = cycle
	if _, err := d.SerializeSnapshot(cycle, Preservation{}); !errors.Is(err, ErrStructure) {
		t.Fatalf("cycle accepted: %v", err)
	}
	opaqueCycle := []Node{{Kind: Opaque}}
	opaqueCycle[0].Children = opaqueCycle
	if _, err := d.SerializeSnapshot(opaqueCycle, Preservation{}); !errors.Is(err, ErrPreservation) {
		t.Fatalf("Opaque cycle accepted: %v", err)
	}
}
