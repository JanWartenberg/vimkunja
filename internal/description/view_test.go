package description

import (
	"strings"
	"testing"
)

func frameText(f Frame) string {
	var lines []string
	for _, row := range f.Rows {
		var b strings.Builder
		for _, span := range row.Spans {
			b.WriteString(span.Text)
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

func TestViewRendersNativeBlocksWithoutChangingSource(t *testing.T) {
	const source = `<h2>Title</h2><p>Hello <strong>bold</strong><br>World <a href="https://hidden.example">label</a></p><blockquote><p>Quote</p></blockquote><ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="secret"><p>Done</p></li></ul><pre><code>print(1)\n</code></pre>`
	doc := Parse(source)
	view := NewView(doc)
	frame := view.Layout(80, false)
	for _, want := range []string{"Title", "Hello bold", "World label", "│  Quote", "☑ Done", "Checklist progress: 1/1", "│ print(1)"} {
		if !strings.Contains(frameText(frame), want) {
			t.Fatalf("missing %q in %q", want, frameText(frame))
		}
	}
	if strings.Contains(frameText(frame), "hidden.example") || strings.Contains(frameText(frame), "secret") || strings.Contains(frameText(frame), "<strong>") {
		t.Fatal("hidden metadata exposed")
	}
	if doc.SourceHTML() != source {
		t.Fatal("projection changed Source HTML")
	}
	bold := false
	for _, row := range frame.Rows {
		for _, span := range row.Spans {
			if span.Style&Bold != 0 && strings.Contains(span.Text, "bold") {
				bold = true
			}
		}
	}
	if !bold {
		t.Fatal("semantic style missing")
	}
}

func TestViewNavigationPreservesGraphemesAndWholeCodeUnits(t *testing.T) {
	doc := Parse("<p>Aé👩‍💻 Z</p><pre><code>a\n\nb\n</code></pre><blockquote><p>Q1</p><p>Q2</p></blockquote><p>End</p>")
	view := NewView(doc)
	text := doc.Snapshot()[0].Children[0]
	if err := view.SetAnchor(Anchor{text.ID, 1}); err != nil {
		t.Fatal(err)
	}
	view.Move("l", 1, 40, 8)
	if view.Cursor().Anchor.Offset != 4 {
		t.Fatalf("split combining grapheme: %#v", view.Cursor())
	}
	view.Move("l", 1, 40, 8)
	if view.Cursor().Anchor.Offset != 15 {
		t.Fatalf("split ZWJ emoji: %#v", view.Cursor())
	}
	view.Move("j", 1, 40, 8)
	if !strings.Contains(frameText(view.Layout(40, true)), "```\n│ a\n│ \n│ b\n│ \n│ ```") {
		t.Fatal("whole Code disclosure/blank lines missing")
	}
	view.Move("j", 1, 40, 8)
	if view.Cursor().Anchor.Offset != 2 {
		t.Fatalf("Code blank line not target: %#v", view.Cursor())
	}
	view.Move("}", 1, 40, 8)
	if !strings.Contains(view.Hint(), "read-only") {
		t.Fatal("missing read hint")
	}
	quoteID := view.Cursor().Anchor.Node
	view.Move("j", 1, 40, 8)
	if view.Cursor().Anchor.Node == quoteID {
		t.Fatal("quote child navigation failed")
	}
	view.Move("}", 1, 40, 8)
	last := doc.Snapshot()[3].Children[0]
	if view.Cursor().Anchor.Node != last.ID {
		t.Fatal("paragraph motion did not leave entire Quote")
	}
	if doc.SourceHTML() != view.SourceHTML() {
		t.Fatal("navigation altered source")
	}
}

func TestViewOpaqueExtentTableAndHorizontalRestoration(t *testing.T) {
	const source = `<p>Before</p><table><tr><td>Name</td><td>Value</td></tr><tr><td>abcdefghij</td><td>42</td></tr></table><custom><p>One</p><p>Two</p></custom><pre><code>abcdefghijk</code></pre><p>After</p>`
	v := NewView(Parse(source))
	f := v.Layout(80, false)
	for _, want := range []string{"Name", "Value", "42", "┌", "One", "Two"} {
		if !strings.Contains(frameText(f), want) {
			t.Fatalf("missing grid/atom %q", want)
		}
	}
	v.Move("j", 1, 12, 6)
	before := frameText(v.Layout(12, true))
	v.Move("zl", 4, 12, 6)
	shifted := frameText(v.Layout(12, true))
	if before == shifted {
		t.Fatal("Table did not scroll horizontally")
	}
	v.Move("j", 1, 12, 6)
	v.Move("k", 1, 12, 6)
	if frameText(v.Layout(12, true)) != shifted {
		t.Fatal("Table horizontal offset lost on focus change")
	}
	v.Move("j", 2, 12, 6)
	if !strings.Contains(frameText(v.Layout(12, true)), "↔") {
		t.Fatal("overwide Code indicator missing")
	}
	anchor := v.Cursor()
	v.Viewport(1, 2, true)
	if v.Cursor() != anchor {
		t.Fatal("narrow Pane changed document anchor")
	}
}

func TestViewWrapResizeScrollAndProtectedSyntax(t *testing.T) {
	doc := Parse(`<p>Literal *stars* plus <strong><b>bold words here</b></strong> end</p><p>Second paragraph</p><p>Third paragraph</p>`)
	v := NewView(doc)
	literal := doc.Snapshot()[0].Children[0]
	v.SetAnchor(Anchor{literal.ID, 8})
	focused := frameText(v.Layout(16, true))
	if !strings.Contains(strings.ReplaceAll(focused, "\n↪ ", ""), `\*stars\*`) {
		t.Fatalf("imported syntax not escaped: %q", focused)
	}
	anchor := v.Cursor()
	v.Viewport(16, 3, true)
	f := v.Viewport(40, 3, true)
	if v.Cursor() != anchor || f.CursorY < f.Top || f.CursorY >= f.Top+3 {
		t.Fatal("resize lost cursor")
	}
	scroll := v.Scroll(2, 16, 2).Top
	if v.Viewport(16, 2, true).Top != scroll || v.Cursor() != anchor {
		t.Fatal("explicit scroll snapped back to cursor")
	}
	if strings.Contains(v.ContentText(), "|b|") || strings.Contains(v.ContentText(), `\*`) {
		t.Fatal("protected syntax copied")
	}
}

func TestViewRulesNumberingAndAtomicWordMotions(t *testing.T) {
	doc := Parse(`<p>Start</p><hr><ol reversed><!-- hidden --><li>One</li><li>Two</li></ol><p>A <mention-user data-label="Jan &amp; Co" data-id="secret"></mention-user>! End</p>`)
	v := NewView(doc)
	v.Move("j", 1, 40, 8)
	if v.Cursor().Anchor.Node != doc.Snapshot()[1].ID {
		t.Fatal("rule skipped by navigation")
	}
	if !strings.Contains(frameText(v.Layout(40, true)), "---") {
		t.Fatal("rule disclosure missing")
	}
	text := frameText(v.Layout(80, false))
	if !strings.Contains(text, "2. One") || !strings.Contains(text, "1. Two") {
		t.Fatalf("reversed list counts comment as entry: %q", text)
	}
	paragraph := doc.Snapshot()[3]
	v.SetAnchor(Anchor{paragraph.Children[0].ID, 0})
	v.Move("w", 1, 40, 8)
	if v.Cursor().Anchor.Node != paragraph.Children[1].ID {
		t.Fatal("Opaque mention is not one word")
	}
	v.Move("W", 1, 40, 8)
	if v.Cursor().Anchor.Node == paragraph.Children[1].ID {
		t.Fatal("WORD motion entered Opaque label")
	}
	if strings.Contains(v.ContentText(), "[@Jan") || strings.Contains(v.ContentText(), "hidden") || strings.Contains(v.ContentText(), "secret") {
		t.Fatal("copy included protected frames or hidden metadata")
	}
}

func TestViewCheckboxSummaryInsideOrdinaryAndOpaqueLists(t *testing.T) {
	for _, source := range []string{`<ul><li>Parent<ul data-type="taskList"><li data-type="taskItem" data-checked="true">Done</li></ul></li></ul>`, `<ul data-type="taskList" style="color:red"><li data-type="taskItem" data-checked="true">Done</li><li data-type="taskItem" data-checked="false">Open</li></ul>`} {
		v := NewView(Parse(source))
		if !strings.Contains(frameText(v.Layout(80, false)), "Checklist progress: 1/") {
			t.Fatalf("read-only summary missing: %q", frameText(v.Layout(80, false)))
		}
	}
}
