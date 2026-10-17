package description

import (
	"reflect"
	"testing"

	"github.com/rivo/uniseg"
)

// Generate understood documents rather than mostly malformed HTML, then vary
// real cursor/selection positions. The older atomicity targets neither navigate
// through Unicode text nor check that accepted edits remain reversible.
func FuzzDraftStructuredHistory(f *testing.F) {
	for i := byte(0); i < 8; i++ {
		f.Add(i, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})
	}
	f.Fuzz(func(t *testing.T, shape byte, steps []byte) {
		if len(steps) > 48 {
			t.Skip()
		}
		const text = "Aé é 👩‍💻🇩🇪中 Z"
		sources := []string{
			`<p>` + text + `</p><p>tail</p>`,
			`<p><strong>` + text + `</strong> <em>tail</em></p>`,
			`<ul><li>` + text + `<ul><li>child</li></ul></li><li>tail</li></ul>`,
			`<blockquote data-comment-id="42"><p>` + text + `</p><p>tail</p></blockquote>`,
			`<ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="keep"><p>` + text + `</p></li><li data-type="taskItem" data-task-id="tail"><p>tail</p></li></ul>`,
			`<p>` + text + `<img src='map.png' alt='map' data-note='keep'>tail</p><aside data-note='opaque'>protected</aside>`,
			`<p><a href="https://example.org">` + text + `</a> tail</p><pre><code>x\ty\nz</code></pre>`,
			`<p><a href="https://example.org">` + text + "</a> tail</p><pre><code>x\ty\nz</code></pre>",
		}
		// Keep the original seven-shape mapping for retained fuzz regressions.
		index := int(shape) % 7
		if shape == 7 {
			index = 7
		}
		source := sources[index]
		origin := Parse(source)
		if origin.Classification() == Blocked {
			t.Fatal("generated blocked source")
		}
		d := NewDraft(origin)
		actions := []Action{
			{Kind: Insert, Text: "é👩‍💻"}, {Kind: Type, Text: "**x**"},
			{Kind: Split}, {Kind: HardBreak}, {Kind: Backspace}, {Kind: Delete},
			{Kind: DeleteSelection}, {Kind: ReplaceSelection, Text: "中"},
			{Kind: ToggleMark, Text: "strong"}, {Kind: ClearMarks},
			{Kind: IndentList}, {Kind: OutdentList}, {Kind: ConvertList, Text: "task"},
			{Kind: ToggleCheckbox}, {Kind: WrapQuote}, {Kind: UnwrapQuote},
			{Kind: OpenBelow}, {Kind: OpenAbove}, {Kind: Convert, Text: "h2"},
		}
		for i, step := range steps {
			anchors := structuredHistoryAnchors(d.Snapshot())
			if len(anchors) == 0 {
				return // Deleting all blocks leaves an empty, valid draft.
			}
			d.CancelSelection()
			if err := d.SetCursor(anchors[(int(step)+i)%len(anchors)]); err != nil {
				t.Fatal(err)
			}
			if step&3 != 0 {
				d.Select(step&1 != 0, step&7 == 7)
				if err := d.SetCursor(anchors[(int(step)*7+i)%len(anchors)]); err != nil {
					t.Fatal(err)
				}
			}
			before := copyState(d.state)
			undo, redo, next := len(d.undo), len(d.redo), d.next
			err := d.Apply(actions[int(step)%len(actions)], step&4 != 0)
			if err != nil {
				if !reflect.DeepEqual(before, d.state) || len(d.undo) != undo || len(d.redo) != redo || d.next != next {
					t.Fatalf("step %d (%d): rejected edit changed state/history", i, step)
				}
				continue
			}
			if _, err := d.HTML(); err != nil {
				t.Fatalf("step %d (%d): accepted edit cannot serialize: %v html=%s", i, step, err, d.DraftHTML())
			}
			if _, err := d.live().ResolveAnchor(d.Cursor()); err != nil && (len(d.Snapshot()) != 0 || d.Cursor() != (Anchor{})) {
				t.Fatalf("step %d (%d): accepted edit left invalid cursor: %v cursor=%+v html=%s", i, step, err, d.Cursor(), d.DraftHTML())
			}
			after := copyState(d.state)
			if reflect.DeepEqual(before, after) {
				continue
			}
			if !d.Undo() || !reflect.DeepEqual(before, d.state) {
				t.Fatalf("step %d (%d): Undo did not restore pre-edit state", i, step)
			}
			if !d.Redo() || !reflect.DeepEqual(after, d.state) {
				t.Fatalf("step %d (%d): Redo did not restore accepted state", i, step)
			}
		}
		if origin.SourceHTML() != source {
			t.Fatal("editing changed original source")
		}
	})
}

func structuredHistoryAnchors(nodes []Node) []Anchor {
	var anchors []Anchor
	var collect func([]Node)
	collect = func(nodes []Node) {
		for _, n := range nodes {
			if n.Kind == Text {
				anchors = append(anchors, Anchor{n.ID, 0})
				g := uniseg.NewGraphemes(n.Text)
				for g.Next() {
					_, end := g.Positions()
					anchors = append(anchors, Anchor{n.ID, end})
				}
			} else if n.Kind == Opaque || len(n.Children) == 0 {
				anchors = append(anchors, Anchor{n.ID, 0}, Anchor{n.ID, 1})
			} else {
				collect(n.Children)
			}
		}
	}
	collect(nodes)
	return anchors
}
