package vimkunja

import (
	"strings"
	"testing"
)

func TestQuerySearchGroupsTypedMatchesAndNeverReturnsDescriptionHTML(t *testing.T) {
	data := SearchData{
		Projects: []Project{
			{ID: 1, Identifier: "ALPHA", Title: "Alpha"},
			{ID: 2, Identifier: "REL", Title: "Release Notes"},
		},
		Tasks: []Task{
			{ID: 11, ProjectID: 1, ProjectIdentifier: "ALPHA", ProjectTitle: "Alpha", Title: "Prepare Release Notes"},
			{ID: 12, ProjectID: 1, ProjectIdentifier: "ALPHA", ProjectTitle: "Alpha", Title: "Planning", Description: "<p>Read the <strong>release notes</strong> before launch.</p>"},
		},
		Labels: []Label{{ID: 21, Title: "Release Notes"}},
	}

	results := QuerySearch(data, "RELEASE notes")
	if len(results) != 4 {
		t.Fatalf("search results = %#v, want four typed results", results)
	}
	if got, want := []SearchResultKind{results[0].Kind, results[1].Kind, results[2].Kind, results[3].Kind}, []SearchResultKind{SearchTaskResult, SearchLabelResult, SearchDescriptionResult, SearchProjectResult}; !sameResultKinds(got, want) {
		t.Fatalf("search result kinds = %#v, want %#v", got, want)
	}

	if task := results[0]; task.TaskID != 11 || task.Primary != "Prepare Release Notes" || len(task.Highlights) != 2 {
		t.Fatalf("task result = %#v, want title hit with both visible highlights", task)
	}
	if label := results[1]; label.LabelID != 21 || label.Primary != "Release Notes" {
		t.Fatalf("label result = %#v, want Label 21", label)
	}
	if description := results[2]; description.TaskID != 12 || description.Primary != "Planning" || description.Secondary != "ALPHA · Alpha" || strings.Contains(description.Snippet, "<") || !strings.Contains(strings.ToLower(description.Snippet), "release notes") {
		t.Fatalf("description result = %#v, want safe contextual text", description)
	}
	if project := results[3]; project.ProjectID != 2 || project.Primary != "Release Notes" || project.Secondary != "REL" {
		t.Fatalf("project result = %#v, want Project 2", project)
	}
}

func TestQuerySearchHighlightsEveryVisibleOccurrence(t *testing.T) {
	results := QuerySearch(SearchData{Tasks: []Task{{ID: 1, Title: "Coffee coffee"}}}, "coffee")
	if len(results) != 1 {
		t.Fatalf("results = %#v, want one Task", results)
	}
	if got, want := results[0].Highlights, []SearchHighlight{{Start: 0, End: 6}, {Start: 7, End: 13}}; !sameHighlights(got, want) {
		t.Fatalf("highlights = %#v, want %#v", got, want)
	}
}

func TestQuerySearchRequiresEveryTermWithinOneSearchableValue(t *testing.T) {
	results := QuerySearch(SearchData{
		Projects: []Project{{ID: 1, Title: "Release", Identifier: "NOTES"}},
		Tasks:    []Task{{ID: 2, Title: "Release", Description: "notes"}},
	}, "release notes")

	if len(results) != 0 {
		t.Fatalf("cross-field matches = %#v, want no results", results)
	}
}

func sameHighlights(got, want []SearchHighlight) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func sameResultKinds(got, want []SearchResultKind) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
