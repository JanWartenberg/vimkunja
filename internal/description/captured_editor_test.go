package description

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestCapturedUpstreamEditorHTMLIsEditableAndPreserved(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/description-rendering/captures/upstream-editor-2026-10-11.json")
	if err != nil {
		t.Fatal(err)
	}
	var captures struct {
		Kind     string                      `json:"kind"`
		Captures []struct{ ID, HTML string } `json:"captures"`
	}
	if err = json.Unmarshal(data, &captures); err != nil {
		t.Fatal(err)
	}
	if captures.Kind != "captured-upstream-editor-component" || len(captures.Captures) != 7 {
		t.Fatal("capture provenance/inventory changed")
	}
	for _, c := range captures.Captures {
		t.Run(c.ID, func(t *testing.T) {
			doc := Parse(c.HTML)
			want := Native
			if c.ID == "mentions" {
				want = Mixed
			}
			if doc.Classification() != want {
				t.Fatalf("got %s want %s: %v", doc.Classification(), want, doc.Problem())
			}
			out, err := doc.Serialize()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(browserSemantics(t, c.HTML), browserSemantics(t, out)) {
				t.Fatalf("capture meaning changed: %s", out)
			}
			if !strings.Contains(out, "<p>") {
				t.Fatal(out)
			}
			if c.ID == "checkboxes" {
				d := NewDraft(doc)
				n := d.Snapshot()[0].Children[0].Children[1].Children[0].Children[0]
				d.SetCursor(Anchor{n.ID, 0})
				d.Select(false, false)
				d.SetCursor(Anchor{n.ID, 4})
				if err := d.Apply(Action{Kind: ReplaceSelection, Text: "Delta"}, false); err != nil {
					t.Fatal(err)
				}
				if err := d.Apply(Action{Kind: ToggleCheckbox}, false); err != nil {
					t.Fatal(err)
				}
				candidate, err := d.PrepareSave()
				if err != nil {
					t.Fatal(err)
				}
				got := candidate.HTML()
				if path := os.Getenv("VIMKUNJA_VOA_ROUNDTRIP_INPUT"); path != "" {
					if err := os.WriteFile(path, []byte(got), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if !strings.Contains(got, `data-task-id="nativeA1"`) || !strings.Contains(got, `data-task-id="nativeB2"`) || !strings.Contains(got, `<p>Delta</p>`) || strings.Contains(got, `checked="checked"`) {
					t.Fatal(got)
				}
			}
			if c.ID == "mentions" && !strings.Contains(out, `<mention-user data-id="jan" data-label="Jan" data-mention-suggestion-char="@"></mention-user>`) {
				t.Fatal(out)
			}
		})
	}
}

func TestCapturedCheckboxProjectionAndStructuralEditing(t *testing.T) {
	const source = `<ul data-type="taskList"><li data-checked="true" data-task-id="nativeA1" data-type="taskItem"><label><input type="checkbox" checked="checked"><span></span></label><div><p>Alpha</p><ul data-type="taskList"><li data-checked="false" data-task-id="nativeB2" data-type="taskItem"><label><input type="checkbox"><span></span></label><div><p>Child</p></div></li></ul></div></li></ul>`
	doc := Parse(source)
	for _, width := range []int{80, 16, 6} {
		got := frameText(NewView(doc).Layout(width, false))
		if width == 80 && (!strings.Contains(got, "☑ Alpha") || !strings.Contains(got, "  ☐ Child")) {
			t.Fatalf("checkbox projection: %q", got)
		}
		if strings.Contains(got, "nativeA1") || strings.Contains(got, "input") {
			t.Fatal(got)
		}
		if doc.SourceHTML() != source {
			t.Fatal("layout changed source")
		}
	}
	d := NewDraft(doc)
	cursorText(t, d, "Alpha", len("Alpha"))
	if err := d.Apply(Action{Kind: Split}, false); err != nil {
		t.Fatal(err)
	}
	candidate, err := d.PrepareSave()
	if err != nil {
		t.Fatal(err)
	}
	got := candidate.HTML()
	if !strings.Contains(got, `data-task-id="nativeA1"`) || !strings.Contains(got, `data-task-id="nativeB2"`) || !strings.Contains(got, `<p></p><ul data-type="taskList">`) {
		t.Fatal(got)
	}
	d.Undo()
	restored, err := d.HTML()
	if err != nil || restored != source {
		t.Fatalf("undo: %s %v", restored, err)
	}
	cursorText(t, d, "Child", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: OutdentList}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := d.PrepareSave(); err != nil {
		t.Fatal(err)
	}
	d.Undo()
	cursorText(t, d, "Child", 0)
	d.Select(true, false)
	if err := d.Apply(Action{Kind: OutdentList}, false); err != nil {
		t.Fatal(err)
	}
	d.Select(true, false)
	if err := d.Apply(Action{Kind: IndentList}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := d.PrepareSave(); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"task", "ul", "ol"} {
		converted := NewDraft(doc)
		cursorText(t, converted, "Alpha", 0)
		if err := converted.Apply(Action{Kind: ConvertList, Text: kind}, false); err != nil {
			t.Fatal(err)
		}
		if _, err := converted.PrepareSave(); err != nil {
			t.Fatalf("convert %s: %v", kind, err)
		}
	}

}

func TestCheckboxRenderRecognitionStaysLocal(t *testing.T) {
	for _, source := range []string{
		`<p>Before</p><label><input type="checkbox"><span></span></label><div>After</div>`,
		`<ul data-type="taskList"><li data-type="taskItem"><label title="keep"><input type="checkbox"><span></span></label><div><p>Text</p></div></li></ul>`,
		`<ul data-type="taskList"><li data-type="taskItem"><label><input type="checkbox" onclick="bad()"><span></span></label><div><p>Text</p></div></li></ul>`,
	} {
		doc := Parse(source)
		if doc.Classification() != Mixed {
			t.Fatalf("unsafe/unrecognized scaffold native: %s", source)
		}
		if strings.Contains(source, "<p>Text</p>") {
			d := NewDraft(doc)
			cursorText(t, d, "Text", 0)
			if err := d.Apply(Action{Kind: Insert, Text: "Editable "}, false); err != nil {
				t.Fatal(err)
			}
			candidate, err := d.PrepareSave()
			if err != nil || !strings.Contains(candidate.HTML(), "<p>Editable Text</p>") {
				t.Fatalf("Native neighbor blocked: %v", err)
			}
		}
		got, err := doc.Serialize()
		if err != nil || got != source {
			t.Fatalf("preservation: %s %v", got, err)
		}
	}
}

func TestCorpusOpaqueCandidateFailuresPreserveOrigin(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/description-rendering/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Fixtures []struct {
			ID     string
			Source string `json:"source_html"`
		}
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	for _, f := range corpus.Fixtures {
		doc := Parse(f.Source)
		if doc.Classification() != Mixed {
			continue
		}
		t.Run(f.ID, func(t *testing.T) {
			origin := doc.Snapshot()
			nodes := doc.Snapshot()
			var invalidate func([]Node) bool
			invalidate = func(ns []Node) bool {
				for i := range ns {
					if ns[i].Kind == Opaque {
						ns[i].ID = 0
						return true
					}
					if invalidate(ns[i].Children) {
						return true
					}
				}
				return false
			}
			if !invalidate(nodes) {
				t.Fatal("mixed document has no preserved atom")
			}
			candidate, err := doc.PrepareSave(nodes, Preservation{}, func() (string, error) { t.Fatal("UUID allocated before preservation validation"); return "", nil })
			if !errors.Is(err, ErrPreservation) || candidate.Valid() {
				t.Fatalf("unsafe candidate escaped: %v", err)
			}
			if doc.SourceHTML() != f.Source || !reflect.DeepEqual(doc.Snapshot(), origin) {
				t.Fatal("failed candidate mutated origin")
			}
			if _, err := doc.Serialize(); err != nil {
				t.Fatalf("origin no longer saves: %v", err)
			}
		})
	}
}
