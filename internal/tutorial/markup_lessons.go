package tutorial

import (
	"github.com/JanWartenberg/vimkunja/internal/description"
	"github.com/JanWartenberg/vimkunja/internal/tui"
	"strings"
	"unicode"
)

const workshopSource = `<p>Alpha beta</p><pre><code>original()</code></pre><p><mention-user data-id="jan" data-label="Jan"></mention-user></p>`
const workshopMention = `<p><mention-user data-id="jan" data-label="Jan"></mention-user></p>`

// Recognize meaning rather than a particular serializer's attribute order or
// mark spelling. Opaque previews/source never serve as editable text evidence.
func markedText(source, tag, text, attribute, value string) bool {
	var content func(description.Node) string
	content = func(n description.Node) string {
		if n.Kind == description.Text {
			return n.Text
		}
		if n.Kind == description.Opaque {
			return ""
		}
		var out strings.Builder
		for _, c := range n.Children {
			out.WriteString(content(c))
		}
		return out.String()
	}
	var walk func([]description.Node) bool
	walk = func(nodes []description.Node) bool {
		for _, n := range nodes {
			match := n.Tag == tag || tag == "strong" && n.Tag == "b" || tag == "em" && n.Tag == "i"
			if match && content(n) == text {
				if attribute == "" {
					return true
				}
				for _, a := range n.Attributes {
					if a.Name == attribute && (value == "" && a.Value != "" || a.Value == value) {
						return true
					}
				}
			}
			if walk(n.Children) {
				return true
			}
		}
		return false
	}
	return walk(description.Parse(source).Snapshot())
}
func draftMarked(tag, text, attr, value string) func(tui.SessionSnapshot) bool {
	return func(s tui.SessionSnapshot) bool {
		return draftIs(s, 108) && markedText(s.Draft.Description, tag, text, attr, value)
	}
}

// Keep mark ancestry on the same text run; two separately formatted copies of
// a word do not constitute combined formatting.
func combinedMarks(source, word string) bool {
	var out strings.Builder
	var walk func([]description.Node, bool, bool)
	walk = func(nodes []description.Node, bold, italic bool) {
		for _, n := range nodes {
			if n.Kind == description.Text {
				if bold && italic {
					out.WriteString(n.Text)
				} else {
					out.WriteByte(0)
				}
			} else if n.Kind == description.Opaque {
				out.WriteByte(0)
			} else {
				walk(n.Children, bold || n.Tag == "strong" || n.Tag == "b", italic || n.Tag == "em" || n.Tag == "i")
				switch n.Tag {
				case "p", "blockquote", "li", "pre", "h1", "h2", "h3", "h4", "h5", "h6", "br":
					out.WriteByte(0)
				}
			}
		}
	}
	walk(description.Parse(source).Snapshot(), false, false)
	for _, part := range strings.FieldsFunc(out.String(), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
		if part == word {
			return true
		}
	}
	return false
}

func markupLessons(alias string) []lesson {
	if alias == "" {
		alias = "ctrl+q"
	}
	native := []string{"G", "enter", "g", "n", "enter", "g", "g"}
	codeSource := "<pre><code>print(1)\n\t</code></pre>"
	ls := []lesson{
		{topic: "Native navigation", task: "Open Description of Task ‘Inspect task metadata’ in Native mode.", context: "This Task contains editable text, Code and a protected Mention.", hint: "G selects the last Task; Enter opens it, gn focuses Description, Enter activates it. Explore with gg/G, w/b/e, counts and arrows.", goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			return draftIs(s, 108) && s.Field == "Description" && s.Native && s.DescriptionActive
		}}},
		{topic: "Word editing", task: "Change ‘Alpha’ to ‘Hallo’ in Task ‘Inspect task metadata’ Description.", context: "Native text supports word objects and change/delete operators.", hint: "ciw changes the current word; type Hallo, then Esc. You may use selection, deletion and Paste instead.", goals: []func(tui.SessionSnapshot) bool{draftMarked("p", "Hallo beta", "", "")}},
		{topic: "Visual formatting", task: "Make ‘Alpha’ bold in Task ‘Inspect task metadata’ Description.", context: "Visual formatting preserves neighboring text and protected objects.", hint: "viw selects Alpha; Space b applies Bold, Esc ends selection.", goals: []func(tui.SessionSnapshot) bool{draftMarked("strong", "Alpha", "", "")}},
		{topic: "Local history", task: "Undo bold ‘Alpha’, then redo it in this Description.", context: "This task starts with a reversible Bold edit.", hint: "u undoes Description edits; Ctrl+R redoes them. The history remains local to the Description.", setup: append(append([]string(nil), native...), "v", "i", "w", "space", "b", "esc"), goals: []func(tui.SessionSnapshot) bool{
			func(s tui.SessionSnapshot) bool {
				return draftIs(s, 108) && !markedText(s.Draft.Description, "strong", "Alpha", "", "")
			}, draftMarked("strong", "Alpha", "", ""),
		}},
		{topic: "Links", task: "Link ‘Alpha’ to any valid URL in Task ‘Inspect task metadata’ Description.", context: "Link text and destination are edited separately.", hint: "viw, Space l opens URL input; type or Paste a URL, Enter applies it, Esc ends selection. Invalid URLs stay editable; empty input unlinks.", goals: []func(tui.SessionSnapshot) bool{draftMarked("a", "Alpha", "href", "")}},
		{topic: "Typed syntax", task: "Add bold-italic ‘Both’ to Task ‘Inspect task metadata’ Description.", context: "Directly typed syntax creates marks; pasted markers stay literal.", hint: "o starts a new paragraph; type ***Both***, then Esc.", goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			return draftIs(s, 108) && combinedMarks(s.Draft.Description, "Both")
		}}},
		{topic: "Checkbox", task: "Turn ‘Check item’ into a checked Checkbox in this Description.", context: "Description Checkboxes are independent of Task Done.", hint: "Space x converts the paragraph to a Checkbox; x toggles its checked state.", source: `<p>Check item</p>`, goals: []func(tui.SessionSnapshot) bool{draftMarked("li", "Check item", "data-checked", "true")}},
		{topic: "Unwrap Checkbox", task: "Turn the ‘Check item’ Checkbox back into ordinary text.", context: "Explicit conversion removes the Checkbox wrapper and keeps its text.", hint: "Space x converts the selected Checkbox into a paragraph.", source: `<ul data-type="taskList"><li data-type="taskItem" data-checked="true"><p>Check item</p></li></ul>`, goals: []func(tui.SessionSnapshot) bool{sourceIs(`<p>Check item</p>`)}},
		{topic: "Quote", task: "Quote ‘Alpha beta’ in Task ‘Inspect task metadata’ Description.", context: "Quotes are structured containers, not literal prefix characters.", hint: "Space q wraps the paragraph as a Quote; Space q again unwraps it.", source: `<p>Alpha beta</p>`, goals: []func(tui.SessionSnapshot) bool{draftMarked("blockquote", "Alpha beta", "", "")}},
		{topic: "Unwrap Quote", task: "Unwrap the ‘Alpha beta’ Quote into an ordinary paragraph.", context: "Unwrapping keeps readable text in document order.", hint: "Space q unwraps this Quote.", source: `<blockquote><p>Alpha beta</p></blockquote>`, goals: []func(tui.SessionSnapshot) bool{sourceIs(`<p>Alpha beta</p>`)}},
		{topic: "Create Code", task: "Create an empty Code block in this Task Description.", context: "Code retains exact source whitespace.", hint: "Space c creates Code here; or i, three/four backticks, Enter. With six typed backticks, move between the two groups of three and press Enter; Esc finishes.", source: `<p></p>`, goals: []func(tui.SessionSnapshot) bool{draftMarked("code", "", "", "")}},
		{topic: "Code input", task: "Enter print(1), a newline and a literal Tab in this Code block.", context: "Enter inserts a source newline inside Code.", hint: "i enters Code Insert; type print(1), Enter, then " + alias + " Tab inserts a literal Tab; Esc finishes. Paste is also allowed.", source: `<p></p>`, setup: append(append([]string(nil), native...), "space", "c", "esc"), goals: []func(tui.SessionSnapshot) bool{draftMarked("code", "print(1)\n\t", "", "")}},
		{topic: "Retab", task: "Convert this Code block’s Tab into spaces.", context: "Tab expansion uses the next tabstop, configured here as four columns.", hint: "Use :retab; :retaball also covers Native prose. Undo and Redo remain available.", source: codeSource, goals: []func(tui.SessionSnapshot) bool{draftMarked("code", "print(1)\n    ", "", "")}},
		{topic: "Leave Code", task: "Add paragraph ‘After code’ outside this Code block.", context: "The new paragraph must be outside Code, not another source line.", hint: "ü creates a following paragraph; type After code, then Esc. ]Space from Normal is another route; Ü or [Space creates one before Code.", source: `<pre><code>original()</code></pre>`, goals: []func(tui.SessionSnapshot) bool{draftMarked("p", "After code", "", "")}},
		{topic: "Protected objects", task: "Attempt to delete Mention ‘Jan’, then cancel.", context: "Protected objects remain readable and require confirmation for deletion.", hint: "x opens the deletion warning; n or Esc cancels. The warning explicitly says the object would be DELETED.", source: workshopMention, goals: []func(tui.SessionSnapshot) bool{
			func(s tui.SessionSnapshot) bool { return draftIs(s, 108) && s.Pending }, sourceIs(workshopMention),
		}},
		{topic: "Restore protected object", task: "Delete Mention ‘Jan’, then restore it with Undo.", context: "Undo restores the protected object’s original bytes.", hint: "x, y confirms deletion; u restores the Mention.", source: workshopMention, goals: []func(tui.SessionSnapshot) bool{
			func(s tui.SessionSnapshot) bool {
				return draftIs(s, 108) && !strings.Contains(s.Draft.Description, "mention-user")
			}, sourceIs(workshopMention),
		}},
		{topic: "Save rich text", task: "Save Task ‘Inspect task metadata’ with ‘Alpha’ bold.", context: "Saving adopts generated Checkbox IDs only after a successful write.", hint: "viw, Space b, Esc makes Alpha bold; :w or :write saves. :wq also works.", goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			if s.Saved != nil && s.Saved.ID == 108 && markedText(s.Saved.Description, "strong", "Alpha", "", "") {
				return true
			}
			t, ok := foundTask(s, 108)
			return ok && markedText(t.Description, "strong", "Alpha", "", "")
		}}},
		{topic: "Discard rich text", task: "Discard the unsaved ‘oops’ in this Task Description.", context: "The task starts with an unsaved edit against the original source.", hint: "Use :e! to restore the saved Task baseline; :q! discards and closes. External conflicts use explicit :descriptionreload.", setup: append(append([]string(nil), native...), "i", "oops", "esc"), goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return discardedDescription(s, 108, workshopSource) }}},
		{topic: "Recovery", task: "Recover ‘Broken’ as readable text from Task ‘Try Ex commands’.", context: "Recovery replaces original HTML, formatting and objects only when saved.", hint: "Use :descriptionrec, Tab, Enter, y; Esc returns to Normal. The confirmation previews the recovered text.", setup: []string{"G", "k", "enter", "g", "n", "enter"}, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return draftIs(s, 107) && s.Draft.Description == "<p>Broken</p>" }}},
		{topic: "Discard recovery", task: "Discard recovery and restore Task ‘Try Ex commands’ original source.", context: "This task starts with a recovered, unsaved draft.", hint: "Use :e! or :q!; the blocked source must remain unchanged in SQLite.", setup: append([]string{"G", "k", "enter"}, append(lessonCommand("descriptionrecover"), "y", "esc")...), goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			return discardedDescription(s, 107, `<p>Broken<strong>`)
		}}},
		{topic: "Clear blocked source", task: "Clear Task ‘Try Ex commands’ Description as an unsaved draft.", context: "Clear deletes the entire Description after explicit confirmation.", hint: "Use :descriptionclear, Enter, then y; leave Insert with Esc if needed. Saving is not required.", setup: []string{"G", "k", "enter"}, goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool { return draftIs(s, 107) && s.Draft.Description == "" }}},
		{topic: "Discard and finish", task: "Discard the cleared Description of Task ‘Try Ex commands’.", context: "The original blocked source must stay in the disposable database.", hint: "Use :q! to discard and close; F8 completes the workshop, Ctrl+C exits.", setup: append([]string{"G", "k", "enter"}, append(lessonCommand("descriptionclear"), "y", "esc")...), goals: []func(tui.SessionSnapshot) bool{func(s tui.SessionSnapshot) bool {
			return s.Draft == nil && s.Focus == "Tasks" && discardedDescription(s, 107, `<p>Broken<strong>`)
		}}},
	}
	for i := range ls {
		ls[i].project, ls[i].view = 1, 11
		if i > 0 && ls[i].setup == nil {
			ls[i].setup = append([]string(nil), native...)
		}
	}
	return ls
}
