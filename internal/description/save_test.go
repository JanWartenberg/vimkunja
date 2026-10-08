package description

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSaveCandidateRepairsOnlyMissingAndLaterDuplicateIDs(t *testing.T) {
	const fresh = "11111111-1111-4111-8111-111111111111"
	const source = `<ul data-type="taskList"><li data-type="taskItem" data-task-id="A" data-checked="true">Alpha</li><li data-type="taskItem" data-task-id="A" data-checked="false">Beta</li></ul><custom data-task-id="reserved">Keep</custom>`
	doc := Parse(source)
	nodes := doc.Snapshot()
	before := doc.Snapshot()
	values := []string{"reserved", "A", fresh}
	candidate, err := doc.PrepareSave(nodes, Preservation{}, func() (string, error) { v := values[0]; values = values[1:]; return v, nil })
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(source, `data-task-id="A" data-checked="false"`, `data-task-id="`+fresh+`" data-checked="false"`, 1)
	if candidate.HTML() != want || candidate.Baseline() != source {
		t.Fatalf("unexpected candidate: %q", candidate.HTML())
	}
	if !reflect.DeepEqual(nodes, before) || doc.SourceHTML() != source {
		t.Fatal("candidate preparation changed draft")
	}
	saved := Parse(candidate.HTML())
	stable, err := saved.PrepareSave(saved.Snapshot(), Preservation{}, func() (string, error) { t.Fatal("stable IDs regenerated"); return "", nil })
	if err != nil || stable.HTML() != want {
		t.Fatalf("second save: %v %q", err, stable.HTML())
	}
}

func TestSaveCandidateFailureNeverMutatesDraft(t *testing.T) {
	const source = `<ul data-type="taskList"><li data-type="taskItem" data-checked="false">One</li><li data-type="taskItem" data-checked="true">Two</li></ul>`
	doc := Parse(source)
	nodes := doc.Snapshot()
	before := doc.Snapshot()
	calls := 0
	candidate, err := doc.PrepareSave(nodes, Preservation{}, func() (string, error) {
		calls++
		if calls == 1 {
			return "11111111-1111-4111-8111-111111111111", nil
		}
		return "", fmt.Errorf("entropy failure")
	})
	if err == nil || candidate.Valid() || !reflect.DeepEqual(nodes, before) {
		t.Fatalf("partial repair escaped: %#v %v", candidate, err)
	}
	candidate, err = doc.PrepareSave(nodes, Preservation{}, func() (string, error) { return "NOT-A-UUID", nil })
	if err == nil || candidate.Valid() {
		t.Fatal("invalid UUID generator was accepted or unbounded")
	}
}

func TestSaveCandidateReservesAllDraftAndOpaqueIDs(t *testing.T) {
	const first = "11111111-1111-4111-8111-111111111111"
	const second = "22222222-2222-4222-8222-222222222222"
	const third = "33333333-3333-4333-8333-333333333333"
	const source = `<ul data-type="taskList"><li data-type="taskItem" data-checked="false">One</li><li data-type="taskItem" data-checked="true">Two</li><li data-type="taskItem" data-task-id="` + first + `" data-checked="false">Three</li></ul><custom DATA-TASK-ID='opaque' data-task-id="` + second + `">Keep</custom>`
	doc := Parse(source)
	nodes := doc.Snapshot()
	values := []string{first, second, third, third, "44444444-4444-4444-8444-444444444444"}
	candidate, err := doc.PrepareSave(nodes, Preservation{}, func() (string, error) { v := values[0]; values = values[1:]; return v, nil })
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(source, `data-checked="false">One`, `data-checked="false" data-task-id="`+third+`">One`, 1)
	want = strings.Replace(want, `data-checked="true">Two`, `data-checked="true" data-task-id="44444444-4444-4444-8444-444444444444">Two`, 1)
	if candidate.HTML() != want {
		t.Fatalf("reservation result: %q", candidate.HTML())
	}
}

func TestSaveCandidateRejectsInvalidPreservationBeforeGenerating(t *testing.T) {
	for _, source := range []string{`<p>Broken</unexpected>`, `<p>Native</p><custom>Opaque</custom>`} {
		doc := Parse(source)
		nodes := doc.Snapshot()
		if doc.Classification() != Blocked {
			nodes = nodes[:1]
		}
		candidate, err := doc.PrepareSave(nodes, Preservation{}, func() (string, error) { t.Fatal("generated before preservation validation"); return "", nil })
		if err == nil || candidate.Valid() {
			t.Fatal("invalid preservation accepted")
		}
	}
}

func TestCheckboxSaveCorpusCandidates(t *testing.T) {
	data, err := os.ReadFile("../../.scratch/description-rendering/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Fixtures []struct {
			ID     string `json:"id"`
			Source string `json:"source_html"`
		}
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	const first = "22222222-2222-4222-8222-222222222222"
	const second = "33333333-3333-4333-8333-333333333333"
	expected := map[string]string{
		"checkbox-lists":                    `<ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="legacy-id"><p>Open</p></li><li data-type="taskItem" data-checked="true" data-task-id="` + first + `"><p>Done</p></li></ul>`,
		"checkbox-duplicate-ids":            `<ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="same"><p>A</p></li><li data-type="taskItem" data-checked="false" data-task-id="` + first + `"><p>B</p></li></ul>`,
		"checkbox-id-reservation-atomicity": `<ul data-type="taskList"><li data-type="taskItem" data-task-id="same" data-checked="true"><p>A</p></li><li data-type="taskItem" data-task-id="` + first + `" data-checked="false"><p>B</p></li><li data-type="taskItem" data-checked="false" data-task-id="` + second + `"><p>C</p></li></ul><custom-node data-task-id='11111111-1111-4111-8111-111111111111'>KEEP</custom-node>`,
		"checkbox-middle-split":             `<ul data-type="taskList"><li data-type="taskItem" data-checked="true" data-task-id="A"><p>Alpha</p></li><li data-type="taskItem" data-checked="false" data-task-id="` + first + `"><p>Beta</p><ul data-type="taskList"><li data-type="taskItem" data-checked="false" data-task-id="child"><p>Child</p></li></ul></li></ul>`,
	}
	count := 0
	for _, f := range corpus.Fixtures {
		want, ok := expected[f.ID]
		if !ok {
			continue
		}
		count++
		t.Run(f.ID, func(t *testing.T) {
			doc := Parse(f.Source)
			nodes := doc.Snapshot()
			if f.ID == "checkbox-middle-split" {
				alpha := &nodes[0].Children[0]
				childList := alpha.Children[1]
				alpha.Children = alpha.Children[:1]
				alpha.Children[0].Children[0].Text = "Alpha"
				beta := Node{Kind: Element, Tag: "li", Attributes: []Attribute{{"data-type", "taskItem"}, {"data-checked", "false"}}, Children: []Node{{Kind: Element, Tag: "p", Children: []Node{{Kind: Text, Text: "Beta"}}}, childList}}
				nodes[0].Children = append(nodes[0].Children, beta)
			}
			values := []string{first, second}
			if f.ID == "checkbox-id-reservation-atomicity" {
				values = append([]string{"11111111-1111-4111-8111-111111111111"}, values...)
			}
			// The fixture's reserved Opaque ID remains unchanged and is retried.
			candidate, err := doc.PrepareSave(nodes, Preservation{}, func() (string, error) { value := values[0]; values = values[1:]; return value, nil })
			if err != nil {
				t.Fatal(err)
			}
			if candidate.HTML() != want {
				t.Fatalf("corpus save:\n got %s\nwant %s", candidate.HTML(), want)
			}
			saved := Parse(candidate.HTML())
			stable, err := saved.PrepareSave(saved.Snapshot(), Preservation{}, nil)
			if err != nil || stable.HTML() != want {
				t.Fatal("saved IDs unstable")
			}
		})
	}
	if count != len(expected) {
		t.Fatal("missing corpus save fixture")
	}
}
