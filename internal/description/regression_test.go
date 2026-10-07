package description

import (
	"strings"
	"testing"
	"unicode"
)

func TestRecoveryRetainsUsefulVisibleText(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"<p>Visible &amp; <strong>safe text", "Visible & safe text"},
		{"<!doctype html><html><head><title>Hidden title</title><script>secret</script></head><body><p>Body</p></body></html>", "Body"},
		{"<p>Before</p><script>secret()</script><style>secret-css</style><p>After<strong>unfinished", "Before\nAfterunfinished"},
		{"<p>Literal &lt;tag&gt;</p><ul><li>Alpha</li><li>Beta</li></ul><strong>unfinished", "Literal <tag>\nAlpha\nBeta\nunfinished"},
		{"<p><img alt='Map' src='hidden-url'><mention-user data-label='Jan' data-id='hidden-id'></mention-user></p><strong>unfinished", "Map@Jan\nunfinished"},
		{"<script>hidden-source", ""},
	} {
		t.Run(tc.want, func(t *testing.T) {
			d := Parse(tc.source)
			if d.Classification() != Blocked {
				t.Fatalf("expected Blocked: %s", d.Classification())
			}
			if got := d.RecoveryText(); got != tc.want {
				t.Fatalf("recovery: got %q, want %q", got, tc.want)
			}
			if d.SourceHTML() != tc.source {
				t.Fatal("recovery changed the original source")
			}
		})
	}
}

func TestCodeNormalizationInsideContainers(t *testing.T) {
	for _, content := range []string{"", "\nA\r\n\tB\r\n"} {
		for _, container := range []struct{ before, after string }{
			{"<ul><li>", "</li></ul>"},
			{"<blockquote><ol><li><ul><li>", "</li></ul></li></ol></blockquote>"},
			{"<ul data-type=\"taskList\"><li data-type=\"taskItem\" data-checked=\"true\" data-task-id=\"A\"><p>Text</p>", "</li></ul>"},
		} {
			source := container.before + "<pre data-note=\"keep\"><code class=\"language-go\" data-x=\"keep\">" + content + "</code></pre>" + container.after
			d := Parse(source)
			if d.Classification() != Native {
				t.Fatal(d.Problem())
			}
			out, err := d.Serialize()
			if err != nil || out != source {
				t.Fatalf("nested Code lost structure/metadata/text: %q, %v; want %q", out, err, source)
			}
			again, err := Parse(out).Serialize()
			if err != nil || again != out {
				t.Fatalf("unstable Code round trip: %q, %v", again, err)
			}
		}
	}
}

func TestBooleanAttributeNormalizationIsElementSpecific(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"<p reversed='keep'>A</p>", "<p reversed=\"keep\">A</p>"},
		{"<h2 reversed='A&amp;B'>A</h2>", "<h2 reversed=\"A&amp;B\">A</h2>"},
		{"<ul reversed='keep'><li>A</li></ul>", "<ul reversed=\"keep\"><li>A</li></ul>"},
		{"<ol reversed='false'><li>A</li></ol>", "<ol reversed><li>A</li></ol>"},
		{"<ol reversed></ol>", "<ol reversed></ol>"},
	} {
		out, err := Parse(tc.source).Serialize()
		if err != nil || out != tc.want {
			t.Fatalf("attribute value changed: %q, %v; want %q", out, err, tc.want)
		}
		again, err := Parse(out).Serialize()
		if err != nil || again != out {
			t.Fatalf("unstable attributes: %q, %v", again, err)
		}
	}
}

func TestNestedLinksProtectOnlyTheirOuterLink(t *testing.T) {
	for _, middle := range []struct{ before, after string }{
		{"", ""}, {"<strong>", "</strong>"}, {"<strong><em>", "</em></strong>"},
		{"<custom-node>", "</custom-node>"},
	} {
		atom := "<a href='https://outer.test' data-note='keep'>" + middle.before + "<a href='https://inner.test'>Label</a>" + middle.after + "</a>"
		source := "<p>Before</p><p>Left " + atom + " right</p><p>After</p>"
		d := Parse(source)
		if d.Classification() != Mixed {
			t.Fatalf("nested Link blocks neighbors: %s, %v", d.Classification(), d.Problem())
		}
		nodes := d.Snapshot()
		if nodes[1].Children[1].Kind != Opaque || nodes[1].Children[1].Preview != "Label" {
			t.Fatalf("outer Link should be one safe atom: %#v", nodes[1])
		}
		nodes[0].Children[0].Text = "Changed"
		nodes[2].Children[0].Text = "Also changed"
		out, err := d.SerializeSnapshot(nodes, Preservation{})
		want := "<p>Changed</p><p>Left " + atom + " right</p><p>Also changed</p>"
		if err != nil || out != want {
			t.Fatalf("local edit lost protected Link: %q, %v", out, err)
		}
	}
}

func TestOpaqueListProjectionSeparatesEntries(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"<custom-node><ul><li>Alpha</li><li>Beta</li></ul></custom-node>", "Alpha\nBeta"},
		{"<custom-node><ol><li><p>Alpha</p></li><li>Beta</li></ol></custom-node>", "Alpha\nBeta"},
		{"<custom-node><ul><li>Parent<ul><li>Child</li></ul></li><li>Sibling</li></ul></custom-node>", "Parent\nChild\nSibling"},
	} {
		d := Parse(tc.source)
		if got := d.Snapshot()[0].Preview; got != tc.want {
			t.Fatalf("projection: %q, want %q", got, tc.want)
		}
		if got := d.RecoveryText(); got != tc.want {
			t.Fatalf("recovery boundaries: %q, want %q", got, tc.want)
		}
		if out, err := d.Serialize(); err != nil || out != tc.source {
			t.Fatalf("projection rewrote Opaque source: %q, %v", out, err)
		}
	}
}

func TestSafeProjectionEscapesTerminalControls(t *testing.T) {
	for _, text := range []string{"\x1b[2Jtext", "\x1b]52;c;clipboard\a", "\u009b31mred", "A\x00B\bC\x7fD", "A\rB", "A\r\n\tB é 👩‍💻"} {
		for _, source := range []string{
			"<custom-node>" + text + "</custom-node>",
			"<img alt='" + text + "'>",
			"<mention-user data-label='" + text + "'></mention-user>",
		} {
			d := Parse(source)
			n := d.Snapshot()[0]
			if n.Kind != Opaque {
				n = n.Children[0]
			}
			for _, visible := range []string{n.Preview, d.RecoveryText()} {
				for _, r := range visible {
					if unicode.IsControl(r) && r != '\n' && r != '\t' {
						t.Fatalf("terminal control %U in safe text %q", r, visible)
					}
				}
			}
			if out, err := d.Serialize(); err != nil || !strings.Contains(out, source) {
				t.Fatalf("safe display changed original atom: %q, %v", out, err)
			}
		}
	}
	d := Parse("<custom-node>\x1b[2JA\rB</custom-node>")
	if got := d.Snapshot()[0].Preview; got != "\\x1b[2JA\\rB" {
		t.Fatalf("controls should remain readable: %q", got)
	}
	d = Parse("<custom-node>A\r\n\tB é 👩‍💻</custom-node>")
	if got := d.Snapshot()[0].Preview; got != "A\n\tB é 👩‍💻" {
		t.Fatalf("damaged useful whitespace/Unicode: %q", got)
	}
}
