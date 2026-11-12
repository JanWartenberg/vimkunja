package description

import "testing"

func TestPlainTextShareProjection(t *testing.T) {
	source := `<p><strong>Hello</strong> <a href="https://example.com">site</a></p><ul data-type="taskList"><li data-type="taskItem" data-checked="true"><p>Ready</p></li><li data-type="taskItem" data-checked="false"><p>Next</p></li></ul><pre><code>  one
\ttwo</code></pre><p><img alt="map"></p>`
	want := "Hello site (https://example.com)\n\n[x] Ready\n[ ] Next\n\n  one\n\\ttwo\n\n[Image: map]"
	if got := NewDraft(Parse(source)).PlainText(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPlainTextKeepsNestedListsAndCodeWhitespace(t *testing.T) {
	source := "<ol start=\"3\"><li><p>Parent</p><ul><li><p>Child</p></li></ul><p>More</p></li><li><p>Last</p></li></ol><pre><code>\n  one\n\ttwo\n</code></pre>"
	want := "3. Parent\n  - Child\n  More\n4. Last\n\n\n  one\n\ttwo\n"
	if got := NewDraft(Parse(source)).PlainText(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPlainTextLiteralLinesAndSafeProtectedContent(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		plain              bool
	}{
		{"literal lines", "first\n\n**literal**\n", "first\n\n**literal**\n", true},
		{"no duplicate URL", `<p><a href="https://example.com">https://example.com</a></p>`, "https://example.com", false},
		{"protected active", `<p>Before<script>secret()</script>after</p>`, "Before\n\n[Active content]\n\nafter", false},
		{"blocked", `<p>Visible &amp; <strong>safe text`, "[Unreadable description content]", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDraft(Parse(tc.source))
			if tc.plain {
				d = NewTextDraft(tc.source)
			}
			if got := d.PlainText(); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestPlainTextCodeUsesProtectedPreview(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{`<pre><code style="color:red">  shown
here</code></pre>`, "  shown\nhere"},
		{`<pre><code onclick="secret()">shown</code></pre>`, "[Active content]"},
	} {
		if got := NewDraft(Parse(tc.source)).PlainText(); got != tc.want {
			t.Fatalf("got %q want %q", got, tc.want)
		}
	}
}

func TestPlainTextKeepsExplicitOrderedListNumbers(t *testing.T) {
	source := `<ol start="3"><li><p>First</p></li><li value="7"><p>Restart</p></li><li><p>Next</p></li></ol>`
	want := "3. First\n7. Restart\n8. Next"
	if got := NewDraft(Parse(source)).PlainText(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPlainTextKeepsReversedOrderedListNumbers(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{`<ol reversed><li><p>First</p></li><li><p>Second</p></li></ol>`, "2. First\n1. Second"},
		{`<ol reversed start="9"><li><p>First</p></li><li value="4"><p>Restart</p></li><li><p>Next</p></li></ol>`, "9. First\n4. Restart\n3. Next"},
	} {
		if got := NewDraft(Parse(tc.source)).PlainText(); got != tc.want {
			t.Fatalf("got %q, want %q", got, tc.want)
		}
	}
}
