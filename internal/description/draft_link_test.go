package description

import "testing"

func TestDraftLinkURLAndUnlinkPreserveLabelMarks(t *testing.T) {
	d := NewDraft(Parse(`<p><a href="custom://resource" title="Guide" rel="help"><strong>Read</strong><br>more</a></p>`))
	if err := d.Apply(Action{Kind: EditLink, Text: "https://example.test/new"}, false); err != nil {
		t.Fatal(err)
	}
	want := `<p><a href="https://example.test/new" target="_blank" title="Guide" rel="help"><strong>Read</strong><br>more</a></p>`
	if got, err := d.HTML(); err != nil || got != want {
		t.Fatalf("%q %v", got, err)
	}
	if err := d.Apply(Action{Kind: EditLink}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p><strong>Read</strong><br>more</p>` {
		t.Fatalf("%q %v", got, err)
	}
	d.Undo()
	if got, _ := d.HTML(); got != want {
		t.Fatal(got)
	}
}

func TestDraftLinkSelectionAndEdges(t *testing.T) {
	d := NewDraft(Parse(`<p><strong>A</strong></p><p>B</p>`))
	first := d.Cursor()
	last := d.Snapshot()[1].Children[0]
	d.Select(false, false)
	d.SetCursor(Anchor{last.ID, 0})
	if err := d.Apply(Action{Kind: EditLink, Text: "https://example.test/x"}, false); err != nil {
		t.Fatal(err)
	}
	want := `<p><a href="https://example.test/x" target="_blank"><strong>A</strong></a></p><p><a href="https://example.test/x" target="_blank">B</a></p>`
	if got, err := d.HTML(); err != nil || got != want {
		t.Fatalf("%q %v", got, err)
	}
	d.CancelSelection()
	d.SetCursor(first)
	if err := d.Apply(Action{Kind: Insert, Text: "X"}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p>X<a href="https://example.test/x" target="_blank"><strong>A</strong></a></p><p><a href="https://example.test/x" target="_blank">B</a></p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftEmptyLinkFallback(t *testing.T) {
	d := NewDraft(Parse(`<p><a href="https://example.test/old"></a></p>`))
	d.Apply(Action{Kind: EditLink, Text: "https://example.test/new"}, false)
	if got, _ := d.HTML(); got != `<p><a href="https://example.test/new" target="_blank"></a></p>` {
		t.Fatal(got)
	}
	d.SetCursor(Anchor{d.Cursor().Node, 1})
	if err := d.Apply(Action{Kind: Insert, Text: "!"}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p><a href="https://example.test/new" target="_blank">https://example.test/new!</a></p>` {
		t.Fatalf("%q %v", got, err)
	}
	d.Undo()
	d.Apply(Action{Kind: EditLink}, false)
	if got, err := d.HTML(); err != nil || got != `<p>https://example.test/new</p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftLinkURLPolicyAndAtomicRejection(t *testing.T) {
	for _, value := range []string{"http://localhost/x", "https://example.test/x", "ftp://user:pass@example.test/x", "git://example.test/x", "obsidian://open?vault=My%20Vault", "notion://example.test/x", "message://example.test/x", "https://[::1]:443/a"} {
		if err := ValidateLinkURL(value); err != nil {
			t.Errorf("valid %q: %v", value, err)
		}
	}
	for _, value := range []string{"/relative", "example.test/x", "custom://resource", "javascript:alert(1)", "https://", "https://bad_host/x", "https://-bad.test/x", "https://x:65536/", "https://x:abc/", "https://x/%xx", "https://x/?q=%xx", "https://x/a b"} {
		d := NewDraft(Parse(`<p><a href="https://example.test/old">Read</a></p>`))
		before, _ := d.HTML()
		if err := d.Apply(Action{Kind: EditLink, Text: value}, false); err == nil {
			t.Errorf("accepted %q", value)
		}
		if got, _ := d.HTML(); got != before || d.Undo() {
			t.Errorf("rejection mutated draft for %q", value)
		}
	}
}

func TestDraftLinkUniqueMetadataRejectsPartialSplit(t *testing.T) {
	d := NewDraft(Parse(`<p><a href="https://example.test/x" id="unique">ABCDE</a></p>`))
	id := d.Cursor().Node
	d.SetCursor(Anchor{id, 1})
	d.Select(false, false)
	d.SetCursor(Anchor{id, 3})
	before, _ := d.HTML()
	if err := d.Apply(Action{Kind: EditLink, Text: "https://example.test/new"}, false); err == nil {
		t.Fatal("unique-ID split accepted")
	}
	if got, _ := d.HTML(); got != before || d.Undo() {
		t.Fatal("rejection mutated draft")
	}
}

func TestDraftEmptyLinkVisualContentDelete(t *testing.T) {
	d := NewDraft(Parse(`<p><a href="https://example.test/x"></a></p>`))
	id := d.Cursor().Node
	d.SetTarget(Target{Anchor: Anchor{id, 0}, Syntax: -1})
	d.Select(false, false)
	d.SetTarget(Target{Anchor: Anchor{id, 0}, Syntax: -len("https://example.test/x")})
	if err := d.Apply(Action{Kind: DeleteSelection}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p></p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftDeletingLastLinkLabelDoesNotResurrectURL(t *testing.T) {
	d := NewDraft(Parse(`<p><a href="https://example.test/x">A</a></p>`))
	if err := d.Apply(Action{Kind: Delete}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p></p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftCrossParagraphSelectionEndingInEmptyLink(t *testing.T) {
	d := NewDraft(Parse(`<p>A</p><p><a href="https://example.test/old"></a></p>`))
	id := d.Snapshot()[1].Children[0].ID
	d.Select(false, false)
	d.SetTarget(Target{Anchor: Anchor{id, 0}, Syntax: -2})
	if err := d.Apply(Action{Kind: EditLink, Text: "https://example.test/new"}, false); err != nil {
		t.Fatal(err)
	}
	want := `<p><a href="https://example.test/new" target="_blank">A</a></p><p><a href="https://example.test/new" target="_blank"></a></p>`
	if got, err := d.HTML(); err != nil || got != want {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftEmptyLinkUnlinkLeavesLiveCursor(t *testing.T) {
	d := NewDraft(Parse(`<p><a href="https://example.test/x"></a></p>`))
	if err := d.Apply(Action{Kind: EditLink}, false); err != nil {
		t.Fatal(err)
	}
	if err := d.Apply(Action{Kind: Type, Text: "!"}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p>!https://example.test/x</p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftEmptyLinkVisualUnlinkRemapsFallbackTargets(t *testing.T) {
	d := NewDraft(Parse(`<p><a href="https://example.test/x"></a></p>`))
	id := d.Cursor().Node
	d.SetTarget(Target{Anchor: Anchor{id, 0}, Syntax: -2})
	d.Select(false, false)
	d.SetTarget(Target{Anchor: Anchor{id, 0}, Syntax: -5})
	if err := d.Apply(Action{Kind: EditLink}, false); err != nil {
		t.Fatal(err)
	}
	d.CancelSelection()
	if err := d.Apply(Action{Kind: Type, Text: "!"}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p>http!s://example.test/x</p>` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDraftCrossParagraphClearMarksIncludesEmptyFallback(t *testing.T) {
	d := NewDraft(Parse(`<p><strong>A</strong></p><p><a href="https://example.test/x"></a></p>`))
	d.Select(false, false)
	link := d.Snapshot()[1].Children[0]
	d.SetTarget(Target{Anchor: Anchor{link.ID, 0}, Syntax: -len("https://example.test/x")})
	if err := d.Apply(Action{Kind: ClearMarks}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := d.HTML(); err != nil || got != `<p>A</p><p>https://example.test/x</p>` {
		t.Fatalf("%q %v", got, err)
	}
}
