package vimkunja

import (
	stdhtml "html"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SearchResultKind identifies the record type that produced a Global Search
// row. The type determines how the TUI activates the row.
type SearchResultKind int

const (
	SearchTaskResult SearchResultKind = iota
	SearchLabelResult
	SearchDescriptionResult
	SearchProjectResult
)

// SearchHighlight is a half-open rune range within a visible SearchResult
// field. Rune offsets keep terminal rendering correct for non-ASCII text.
type SearchHighlight struct {
	Start int
	End   int
}

// SearchResult is safe for terminal presentation: Description text is already
// plain text and no source HTML is exposed through this type.
type SearchResult struct {
	Kind                SearchResultKind
	TaskID              int64
	ProjectID           int64
	LabelID             int64
	Primary             string
	Secondary           string
	Snippet             string
	Done                bool
	Highlights          []SearchHighlight
	SecondaryHighlights []SearchHighlight
	SnippetHighlights   []SearchHighlight
}

// QuerySearch applies Global Search's V1 deterministic matching and ordering
// to a complete locally browsable SearchData corpus.
func QuerySearch(data SearchData, query string) []SearchResult {
	terms := searchTerms(query)
	if len(terms) == 0 {
		return nil
	}

	var tasks, labels, descriptions, projects []SearchResult
	for _, task := range data.Tasks {
		if highlights, ok := matchingHighlights(task.Title, terms); ok {
			tasks = append(tasks, SearchResult{
				Kind: SearchTaskResult, TaskID: task.ID, ProjectID: task.ProjectID,
				Primary: task.Title, Secondary: projectContext(task), Done: task.Done, Highlights: highlights,
			})
		}

		description := plainSearchText(task.Description)
		if _, ok := matchingHighlights(description, terms); ok {
			snippet := searchSnippet(description, terms)
			snippetHighlights, _ := matchingHighlights(snippet, terms)
			descriptions = append(descriptions, SearchResult{
				Kind: SearchDescriptionResult, TaskID: task.ID, ProjectID: task.ProjectID,
				Primary: task.Title, Secondary: projectContext(task), Snippet: snippet, Done: task.Done,
				SnippetHighlights: snippetHighlights,
			})
		}
	}
	for _, label := range data.Labels {
		if highlights, ok := matchingHighlights(label.Title, terms); ok {
			labels = append(labels, SearchResult{
				Kind: SearchLabelResult, LabelID: label.ID, Primary: label.Title, Highlights: highlights,
			})
		}
	}
	for _, project := range data.Projects {
		if highlights, ok := matchingHighlights(project.Title, terms); ok {
			projects = append(projects, SearchResult{
				Kind: SearchProjectResult, ProjectID: project.ID, Primary: project.Title,
				Secondary: project.Identifier, Highlights: highlights,
			})
			continue
		}
		if highlights, ok := matchingHighlights(project.Identifier, terms); ok {
			projects = append(projects, SearchResult{
				Kind: SearchProjectResult, ProjectID: project.ID, Primary: project.Title,
				Secondary: project.Identifier, SecondaryHighlights: highlights,
			})
		}
	}

	sortSearchResults(tasks)
	sortSearchResults(labels)
	sortSearchResults(descriptions)
	sortSearchResults(projects)
	return append(append(append(tasks, labels...), descriptions...), projects...)
}

func searchTerms(query string) []string {
	return strings.FieldsFunc(strings.ToLower(query), unicode.IsSpace)
}

func matchingHighlights(value string, terms []string) ([]SearchHighlight, bool) {
	lower := strings.ToLower(value)
	seenTerms := make(map[string]bool, len(terms))
	var highlights []SearchHighlight
	for _, term := range terms {
		if seenTerms[term] {
			continue
		}
		seenTerms[term] = true
		found := false
		for offset := 0; offset < len(lower); {
			matchOffset := strings.Index(lower[offset:], term)
			if matchOffset < 0 {
				break
			}
			byteStart := offset + matchOffset
			start := len([]rune(lower[:byteStart]))
			highlights = append(highlights, SearchHighlight{Start: start, End: start + len([]rune(term))})
			found = true
			_, runeSize := utf8.DecodeRuneInString(lower[byteStart:])
			offset = byteStart + runeSize
		}
		if !found {
			return nil, false
		}
	}
	sort.SliceStable(highlights, func(i, j int) bool {
		if highlights[i].Start != highlights[j].Start {
			return highlights[i].Start < highlights[j].Start
		}
		return highlights[i].End < highlights[j].End
	})
	return highlights, true
}

func sortSearchResults(results []SearchResult) {
	sort.SliceStable(results, func(i, j int) bool {
		left, right := strings.ToLower(results[i].Primary), strings.ToLower(results[j].Primary)
		if left != right {
			return left < right
		}
		return searchResultID(results[i]) < searchResultID(results[j])
	})
}

func searchResultID(result SearchResult) int64 {
	if result.TaskID != 0 {
		return result.TaskID
	}
	if result.LabelID != 0 {
		return result.LabelID
	}
	return result.ProjectID
}

func projectContext(task Task) string {
	if task.ProjectIdentifier == "" {
		return task.ProjectTitle
	}
	if task.ProjectTitle == "" {
		return task.ProjectIdentifier
	}
	return task.ProjectIdentifier + " · " + task.ProjectTitle
}

const searchSnippetLimit = 120

func searchSnippet(text string, terms []string) string {
	if len([]rune(text)) <= searchSnippetLimit {
		return text
	}
	lower := strings.ToLower(text)
	first := len([]rune(text))
	for _, term := range terms {
		if index := strings.Index(lower, term); index >= 0 {
			first = min(first, len([]rune(lower[:index])))
		}
	}
	runes := []rune(text)
	start := max(0, first-30)
	end := min(len(runes), start+searchSnippetLimit)
	if end == len(runes) {
		start = max(0, end-searchSnippetLimit)
	}
	prefix, suffix := "", ""
	if start > 0 {
		prefix = "…"
	}
	if end < len(runes) {
		suffix = "…"
	}
	return prefix + string(runes[start:end]) + suffix
}

// plainSearchText removes HTML markup for Search presentation. It preserves
// text boundaries around tags, decodes entities, and drops terminal control
// characters so source markup can never escape into the TUI.
func plainSearchText(source string) string {
	var text strings.Builder
	inTag := false
	quote := rune(0)
	for _, r := range source {
		switch {
		case inTag:
			if quote != 0 {
				if r == quote {
					quote = 0
				}
				continue
			}
			if r == '\'' || r == '"' {
				quote = r
				continue
			}
			if r == '>' {
				inTag = false
				text.WriteRune(' ')
			}
		case r == '<':
			inTag = true
			text.WriteRune(' ')
		case unicode.IsControl(r):
			text.WriteRune(' ')
		default:
			text.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(stdhtml.UnescapeString(text.String())), " ")
}
