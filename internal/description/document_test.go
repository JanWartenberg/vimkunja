package description

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestCorpusImportAndSerialization(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/description-rendering/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Fixtures []struct {
			ID             string         `json:"id"`
			Classification Classification `json:"classification"`
			Source         string         `json:"source_html"`
			Serialization  struct {
				HTML   *string  `json:"html"`
				Opaque []string `json:"opaque_bytes"`
			} `json:"serialization"`
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Fixtures) != 83 {
		t.Fatalf("fixture inventory changed: %d", len(corpus.Fixtures))
	}
	for _, f := range corpus.Fixtures {
		t.Run(f.ID, func(t *testing.T) {
			d := Parse(f.Source)
			if d.Classification() != f.Classification {
				t.Fatalf("classification: got %s (%v), want %s; nodes %#v", d.Classification(), d.Problem(), f.Classification, d.Snapshot())
			}
			got, err := d.Serialize()
			if f.Classification == Blocked {
				if !errors.Is(err, ErrBlocked) {
					t.Fatalf("ordinary save should fail: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if f.Serialization.HTML != nil && got != *f.Serialization.HTML {
				t.Fatalf("save:\n got %q\nwant %q", got, *f.Serialization.HTML)
			}
			// Missing-ID repair is ticket 04. Import must retain those attributes;
			// this test executes ordinary model serialization, not UUID repair.
			if f.Serialization.HTML == nil && got != f.Source {
				t.Fatalf("model changed Checkbox IDs: %q", got)
			}
			for _, raw := range f.Serialization.Opaque {
				if !strings.Contains(got, raw) {
					t.Fatalf("lost original atom %q", raw)
				}
			}
			again, err := Parse(got).Serialize()
			if err != nil || again != got {
				t.Fatalf("unstable round trip: %q, %v", again, err)
			}
		})
	}
}

func TestOpaqueIdentityAndIntent(t *testing.T) {
	source := "<p>A <img alt='same'> B <img alt='same'> C</p>"
	d := Parse(source)
	nodes := d.Snapshot()
	first, second := nodes[0].Children[1].ID, nodes[0].Children[3].ID
	nodes[0].Children[1], nodes[0].Children[3] = nodes[0].Children[3], nodes[0].Children[1]
	_, err := d.SerializeSnapshot(nodes, Preservation{Positions: map[ID][]int{first: {0, 3}, second: {0, 1}}})
	if !errors.Is(err, ErrPreservation) {
		t.Fatalf("identical atoms swapped: %v", err)
	}

	nodes = d.Snapshot()
	nodes[0].Tag = "h2"
	nodes[0].Children[0].Text = "Changed "
	got, err := d.SerializeSnapshot(nodes, Preservation{})
	if err != nil || got != "<h2>Changed <img alt='same'> B <img alt='same'> C</h2>" {
		t.Fatalf("parent conversion: %q, %v", got, err)
	}
	nodes = d.Snapshot()
	nodes[0].Children = slices.Delete(nodes[0].Children, 1, 2)
	if _, err := d.SerializeSnapshot(nodes, Preservation{}); !errors.Is(err, ErrPreservation) {
		t.Fatalf("silent removal: %v", err)
	}
	_, err = d.SerializeSnapshot(nodes, Preservation{Removed: []ID{first}, Positions: map[ID][]int{second: {0, 1}}})
	if err != nil {
		t.Fatalf("declared deletion/movement: %v", err)
	}
	if original, err := d.Serialize(); err != nil || original != source {
		t.Fatalf("draft leaked into original: %q %v", original, err)
	}
}

func TestSafeProjectionAndRecovery(t *testing.T) {
	source := "<p>Visible</p><custom-node><span onclick='secret()'>Hidden</span></custom-node><script>secret-source</script><style>secret-style</style><p><img alt='Map' src='private-url'></p>"
	d := Parse(source)
	if d.Classification() != Mixed {
		t.Fatal(d.Problem())
	}
	if strings.Contains(d.RecoveryText(), "secret") || strings.Contains(d.RecoveryText(), "Hidden") || strings.Contains(d.RecoveryText(), "private") {
		t.Fatalf("unsafe recovery: %q", d.RecoveryText())
	}
	if d.Snapshot()[1].Preview != "[Active content]" {
		t.Fatalf("active enclosing atom projection: %q", d.Snapshot()[1].Preview)
	}
	for _, source := range []string{"<p>Visible &amp; <strong>safe text", "<!doctype html><html><head><script>secret</script></head><body><p>Body</p></body></html>"} {
		d := Parse(source)
		if d.Classification() != Blocked || strings.Contains(d.RecoveryText(), "secret") || strings.Contains(d.RecoveryText(), "<") {
			t.Fatalf("blocked recovery: %s %q", d.Classification(), d.RecoveryText())
		}
	}
}

func TestPreservationChecksParentIdentity(t *testing.T) {
	d := Parse("<p>Before <img alt='map'> after</p>")
	nodes := d.Snapshot()
	atomID := nodes[0].Children[1].ID
	nodes[0].ID = 0 // A different parent at the same child-index path.
	if _, err := d.SerializeSnapshot(nodes, Preservation{}); !errors.Is(err, ErrPreservation) {
		t.Fatalf("wrong parent accepted: %v", err)
	}
	if _, err := d.SerializeSnapshot(nodes, Preservation{Positions: map[ID][]int{atomID: {0, 1}}}); err != nil {
		t.Fatalf("declared parent replacement: %v", err)
	}
}

func TestOriginalAttributeEvidence(t *testing.T) {
	for _, source := range []string{"<p data-x='one' DATA-X='two'>Safe</p>", "<p /onclick='secret()'>Safe</p>", "<p><a href='java&#x73;cript:secret()'>Safe</a></p>"} {
		d := Parse(source)
		if d.Classification() != Mixed {
			t.Fatalf("unsafe/ambiguous attrs Native: %q %s", source, d.Classification())
		}
		out, err := d.Serialize()
		if err != nil || out != source {
			t.Fatalf("original source changed: %q %v", out, err)
		}
	}
	d := Parse("<p data-note='&notit;'>A</p><custom-node data-task-id='one' DATA-TASK-ID='two'></custom-node>")
	if !slices.Equal(d.ReservedIDs(), []string{"one", "two"}) {
		t.Fatalf("lost reserved IDs: %v", d.ReservedIDs())
	}
	out, err := d.Serialize()
	if err != nil || !strings.Contains(out, "data-note=\"&amp;notit;\"") {
		t.Fatalf("wrong attribute entity decoding: %q %v", out, err)
	}
}

func TestOpaqueSafePreview(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"<img alt='' title='Map'>", "[Image: Map]"},
		{"<mention-user data-label='Jan &amp; Co' data-id='hidden'></mention-user>", "@Jan & Co"},
		{"<mention-user data-id='jan'></mention-user>", "@jan"},
		{"<!--secret-->", "[HTML comment]"},
		{"<custom-node><p>A</p><p>B<br>C</p></custom-node>", "A\nB\nC"},
		{"<table><tr><td>A</td><td>B</td></tr><tr><td>C</td></tr></table>", "A | B\nC"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			d := Parse(tc.source)
			nodes := d.Snapshot()
			if len(nodes) == 0 {
				t.Fatal(d.Problem())
			}
			n := nodes[0]
			if n.Kind != Opaque {
				n = n.Children[0]
			}
			if n.Preview != tc.want {
				t.Fatalf("preview: %q want %q", n.Preview, tc.want)
			}
		})
	}
}

func TestStrictBoundaryRegressions(t *testing.T) {
	for _, source := range []string{"<p>A<div>B</div>C</p>", "<p><b>A</p></b>", "<p>A<!-- open", "<p>A<strong"} {
		d := Parse(source)
		if d.Classification() != Blocked {
			t.Fatalf("repair accepted: %q", source)
		}
	}
	if d := Parse(strings.Repeat("<x>", 2000) + "visible" + strings.Repeat("</x>", 2000)); d.Classification() != Mixed {
		t.Fatal(d.Problem())
	}
}

func TestReliablyBoundedProblemsStayLocal(t *testing.T) {
	for _, atom := range []string{"<ul><p>Unusual</p></ul>", "<li>Orphan</li>", "<pre><strong>Marked code</strong></pre>", "<script>const incompleteTag = '<p';</script>"} {
		source := "<p>Before</p>" + atom + "<p>After</p>"
		d := Parse(source)
		if d.Classification() != Mixed {
			t.Fatalf("bounded problem blocked neighbors: %q: %v", atom, d.Problem())
		}
		nodes := d.Snapshot()
		nodes[0].Children[0].Text = "Changed"
		out, err := d.SerializeSnapshot(nodes, Preservation{})
		if err != nil || out != "<p>Changed</p>"+atom+"<p>After</p>" {
			t.Fatalf("local edit: %q %v", out, err)
		}
	}
}

func FuzzParseAndSerialize(f *testing.F) {
	for _, seed := range []string{"", "<p>A</p>", "<p><strong>open", "<ul><li>A<li>B</ul>", "<pre><code>\nA\r\n\tB\n</code></pre>", "<p><img alt='A'><img alt='A'></p>", "<p onclick='x'>A</p>", `<ul data-type="taskList"><li data-type="taskItem" data-checked="true"><label><input type="checkbox" checked="checked"><span></span></label><div><p>A</p></div></li></ul>`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		d := Parse(source)
		if d.SourceHTML() != source {
			t.Fatal("source changed")
		}
		out, err := d.Serialize()
		if d.Classification() == Blocked {
			if !errors.Is(err, ErrBlocked) {
				t.Fatal(err)
			}
			return
		}
		if err != nil {
			t.Fatalf("imported document cannot serialize: %q: %v", source, err)
		}
		again, err := Parse(out).Serialize()
		if err != nil || out != again {
			t.Fatalf("round trip unstable: %q -> %q, %v", out, again, err)
		}
	})
}
