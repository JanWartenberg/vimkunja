package description

import (
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/net/html"
	htmlatom "golang.org/x/net/html/atom"
)

type browserMeaning struct {
	Kind                 html.NodeType
	Tag, Namespace, Text string
	Attributes           map[string]string
	Children             []browserMeaning
}

func TestDraftAttributeNamesUseHTMLCaseRules(t *testing.T) {
	d := Parse("<p><a href='https://example.test'>A</a></p>")
	nodes := d.Snapshot()
	nodes[0].Children[0].Attributes = []Attribute{{"HREF", "https://example.test"}, {"TARGET", "_self"}, {"data-Ä", "A"}, {"data-ä", "B"}}
	out, err := d.SerializeSnapshot(nodes, Preservation{})
	want := `<p><a href="https://example.test" target="_blank" data-Ä="A" data-ä="B">A</a></p>`
	if err != nil || out != want {
		t.Fatalf("HTML attribute case normalization: %q %v", out, err)
	}
}

// Independent tree-parser oracle for expected HTML semantics. This deliberately
// does not call Parse, normalizeRoot or sameMeaning. It is NOT a source-boundary
// validator: exact Code bytes and Opaque source use their separate goldens.
func browserSemantics(t *testing.T, source string) []browserMeaning {
	t.Helper()
	context := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: htmlatom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(source), context)
	if err != nil {
		t.Fatal(err)
	}
	var convert func(*html.Node) browserMeaning
	convert = func(n *html.Node) browserMeaning {
		out := browserMeaning{Kind: n.Type, Namespace: n.Namespace}
		if n.Type == html.ElementNode {
			out.Tag = n.Data
		} else {
			out.Text = n.Data
		}
		if len(n.Attr) != 0 {
			out.Attributes = map[string]string{}
			for _, a := range n.Attr {
				value := a.Val
				if n.Data == "ol" && a.Key == "reversed" {
					value = ""
				}
				out.Attributes[a.Namespace+":"+a.Key] = value
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			out.Children = append(out.Children, convert(child))
		}
		return out
	}
	var out []browserMeaning
	for _, n := range nodes {
		out = append(out, convert(n))
	}
	return out
}

func TestCorpusMeaningAgainstIndependentTreeParser(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/description-rendering/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Fixtures []struct {
			ID            string         `json:"id"`
			Class         Classification `json:"classification"`
			Source        string         `json:"source_html"`
			Serialization struct {
				HTML *string `json:"html"`
			} `json:"serialization"`
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range corpus.Fixtures {
		t.Run(fixture.ID, func(t *testing.T) {
			if fixture.Class == Blocked {
				if _, err := Parse(fixture.Source).Serialize(); !errors.Is(err, ErrBlocked) {
					t.Fatalf("Blocked save: %v", err)
				}
				return
			}
			want := fixture.Source // UUID repair remains outside this model seam.
			if fixture.Serialization.HTML != nil {
				want = *fixture.Serialization.HTML
			}
			out, err := Parse(fixture.Source).Serialize()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(browserSemantics(t, want), browserSemantics(t, out)) {
				t.Fatalf("Native HTML meaning differs from independently authored expected HTML:\n%s\n%s", want, out)
			}
		})
	}
}

func TestSemanticComparisonCoversNativeMeaning(t *testing.T) {
	for _, tc := range []struct{ a, b string }{
		{"<p>A</p>", "<h2>A</h2>"},
		{"<p>A</p><p>B</p>", "<p>B</p><p>A</p>"},
		{"<p></p>", ""},
		{"<p><b>A</b></p>", "<p><strong>A</strong></p>"},
		{"<p><i>A</i></p>", "<p><em>A</em></p>"},
		{"<p><strong><strong>A</strong></strong></p>", "<p><strong>A</strong></p>"},
		{"<p>A<br>B</p>", "<p>A\nB</p>"},
		{"<p><a href='https://a.test'>A</a></p>", "<p><a href='https://b.test'>A</a></p>"},
		{"<p data-note='A'>A</p>", "<p data-note='B'>A</p>"},
		{"<blockquote cite='https://a.test' data-comment-id='1'><p>A</p></blockquote>", "<blockquote cite='https://a.test' data-comment-id='2'><p>A</p></blockquote>"},
		{"<ol start='2' reversed><li>A</li></ol>", "<ol start='3' reversed><li>A</li></ol>"},
		{"<ul data-type='taskList'><li data-type='taskItem' data-task-id='A' data-checked='true'>X</li></ul>", "<ul data-type='taskList'><li data-type='taskItem' data-task-id='A' data-checked='false'>X</li></ul>"},
		{"<ul><li data-task-id='A'>X</li></ul>", "<ul><li data-task-id='B'>X</li></ul>"},
		{"<pre><code>\tA\n</code></pre>", "<pre><code> A</code></pre>"},
	} {
		if reflect.DeepEqual(browserSemantics(t, tc.a), browserSemantics(t, tc.b)) {
			t.Fatalf("independent oracle failed to distinguish %q and %q", tc.a, tc.b)
		}
		if sameMeaning(Parse(tc.a).Snapshot(), Parse(tc.b).Snapshot()) {
			t.Fatalf("save comparator missed meaningful change: %q and %q", tc.a, tc.b)
		}
	}
	if sameMeaning(Parse("<pre><code>A\r\n</code></pre>").Snapshot(), Parse("<pre><code>A\n</code></pre>").Snapshot()) {
		t.Fatal("comparator lost exact Code CRLF")
	}
	a := Parse("<p data-a='1' data-b='2'><strike>A</strike></p>").Snapshot()
	b := Parse("<p data-b='2' data-a='1'><s>A</s></p>").Snapshot()
	if !sameMeaning(a, b) {
		t.Fatal("representation-only attribute order/strike spelling rejected")
	}
}

func TestGeneratedNativeRoundTrips(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	marks := []string{"strong", "b", "em", "i", "u", "s", "code"}
	for i := 0; i < 100; i++ {
		text := []string{"猫 &amp; é", "A&nbsp;B", "é 👩‍💻", "plain"}[rng.Intn(4)]
		for depth := rng.Intn(5); depth > 0; depth-- {
			mark := marks[rng.Intn(len(marks))]
			text = "<" + mark + ">" + text + "</" + mark + ">"
		}
		text = "<p><a href=\"https://example.test/A?x=1&amp;y=2\" target=\"_blank\" title=\"Keep\">" + text + "<br>after</a></p>"
		expected := "<blockquote cite=\"https://example.test/source\" data-comment-id=\"42\"><ul><li data-note=\"keep\">" + text + "<ol start=\"7\" reversed><li value=\"3\">Child</li></ol></li><li></li></ul><pre><code class=\"language-go\">\n\tA\r\n&amp;&lt;tag&gt;\n</code></pre><p></p></blockquote>"
		out, err := Parse(expected).Serialize()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(browserSemantics(t, expected), browserSemantics(t, out)) {
			t.Fatalf("generated Native meaning changed: %q", out)
		}
		if !strings.Contains(out, "\n\tA\r\n&amp;&lt;tag&gt;\n") {
			t.Fatal("exact Code content changed")
		}
	}
}

func TestSaveRejectsUnrepresentableNativeMeaning(t *testing.T) {
	d := Parse("<p data-note='keep'>A</p>")
	nodes := d.Snapshot()
	nodes[0].Attributes[0].Value = "A\x00B"
	if _, err := d.SerializeSnapshot(nodes, Preservation{}); !errors.Is(err, ErrRoundTrip) {
		t.Fatalf("silent attribute loss should reject save: %v", err)
	}
	if nodes[0].Attributes[0].Value != "A\x00B" || d.Snapshot()[0].Attributes[0].Value != "keep" {
		t.Fatal("rejected save changed draft/original")
	}
	// A CR from an entity is a valid decoded attribute value. It must be
	// escaped as an entity again instead of undergoing tokenizer LF repair.
	out, err := Parse("<p data-note='A&#13;B'>X</p>").Serialize()
	if err != nil || out != "<p data-note=\"A&#13;B\">X</p>" {
		t.Fatalf("attribute CR meaning: %q, %v", out, err)
	}
}

func TestSafeUnicodeAttributeNamesRetainTheirIdentity(t *testing.T) {
	const source = "<p data-Ä=\"upper\" data-ä=\"lower\" DATA-X=\"ASCII\">X</p>"
	const want = "<p data-Ä=\"upper\" data-ä=\"lower\" data-x=\"ASCII\">X</p>"
	d := Parse(source)
	if d.Classification() != Native {
		t.Fatalf("distinct Unicode attributes collapsed: %s", d.Classification())
	}
	out, err := d.Serialize()
	if err != nil || out != want {
		t.Fatalf("changed metadata names: %q %v", out, err)
	}
	if !reflect.DeepEqual(browserSemantics(t, source), browserSemantics(t, out)) {
		t.Fatal("unknown attribute identity changed")
	}
}
