package description

import (
	"fmt"
	"html"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzSaveCandidateIdentityAndPreservation(f *testing.F) {
	f.Add("Alpha é 👩‍💻", []byte{0, 1, 1, 2})
	f.Add("<literal>&\"", []byte{2, 0, 3, 3, 0})
	f.Fuzz(func(t *testing.T, text string, modes []byte) {
		if len(text) > 256 || !utf8.ValidString(text) || len(modes) == 0 || len(modes) > 8 {
			t.Skip()
		}
		const reserved = "00000000-0000-4000-8000-000000000001"
		const opaque = `<custom data-task-id="` + reserved + `">Keep &amp; preserve</custom>`
		var source strings.Builder
		source.WriteString(`<ul data-type="taskList">`)
		for _, mode := range modes {
			source.WriteString(`<li data-type="taskItem" data-checked="false"`)
			if mode%4 != 0 {
				fmt.Fprintf(&source, ` data-task-id="id-%d"`, mode%4)
			}
			source.WriteString(`><p>` + html.EscapeString(text) + `</p></li>`)
		}
		source.WriteString(`</ul>` + opaque)
		doc := Parse(source.String())
		if doc.Classification() == Blocked {
			t.Fatalf("generated valid checklist was blocked: %v", doc.Problem())
		}
		nodes, before := doc.Snapshot(), doc.Snapshot()
		counter := 0
		candidate, err := doc.PrepareSave(nodes, Preservation{}, func() (string, error) {
			counter++
			return fmt.Sprintf("00000000-0000-4000-8000-%012x", counter), nil
		})
		if err != nil || !candidate.Valid() {
			t.Fatalf("valid checklist cannot prepare save: %v", err)
		}
		if candidate.Baseline() != source.String() || doc.SourceHTML() != source.String() ||
			!reflect.DeepEqual(nodes, before) || !reflect.DeepEqual(doc.Snapshot(), before) {
			t.Fatal("preparation mutated its source or input snapshot")
		}
		if strings.Count(candidate.HTML(), opaque) != 1 {
			t.Fatal("save did not preserve the exact protected subtree")
		}
		saved := Parse(candidate.HTML())
		// Compare independently of the serializer's own round-trip check: the
		// only permitted semantic change is repairing Checkbox identities.
		meaning := func(nodes []Node) []Node {
			nodes = cloneNodes(nodes)
			var stripIDs func([]Node)
			stripIDs = func(ns []Node) {
				for i := range ns {
					n := &ns[i]
					n.ID = 0
					if n.Kind == Element && n.Tag == "li" {
						attrs := n.Attributes[:0]
						for _, attr := range n.Attributes {
							if attr.Name != "data-task-id" {
								attrs = append(attrs, attr)
							}
						}
						n.Attributes = attrs
					}
					stripIDs(n.Children)
				}
			}
			stripIDs(nodes)
			return nodes
		}
		if !reflect.DeepEqual(meaning(before), meaning(saved.Snapshot())) {
			t.Fatal("save changed content, Checkbox state or structure beyond ID repair")
		}
		seen, original, count := map[string]bool{}, map[string]bool{}, 0
		var check func([]Node)
		check = func(ns []Node) {
			for _, n := range ns {
				if n.Kind == Element && n.Tag == "li" && attributeValue(n.Attributes, "data-type") == "taskItem" {
					id := attributeValue(n.Attributes, "data-task-id")
					if id == "" || id == reserved || seen[id] {
						t.Fatalf("missing, reserved or repeated Checkbox identity: %q", id)
					}
					if count >= len(modes) {
						t.Fatal("save created an extra Checkbox entry")
					}
					expected := fmt.Sprintf("id-%d", modes[count]%4)
					if modes[count]%4 != 0 && !original[expected] {
						if id != expected {
							t.Fatalf("existing first identity changed: %q -> %q", expected, id)
						}
						original[expected] = true
					} else if !uuidV4.MatchString(id) {
						t.Fatalf("repaired identity is not a UUID v4: %q", id)
					}
					seen[id], count = true, count+1
				}
				check(n.Children)
			}
		}
		check(saved.Snapshot())
		if count != len(modes) {
			t.Fatalf("save lost Checkbox entries: got %d, want %d", count, len(modes))
		}
		stable, err := saved.PrepareSave(saved.Snapshot(), Preservation{}, func() (string, error) {
			t.Fatal("reopened saved checklist requested another identity")
			return "", nil
		})
		if err != nil || stable.HTML() != candidate.HTML() {
			t.Fatalf("reopened save changed HTML: %v", err)
		}
	})
}
