package tutorial

import (
	tea "charm.land/bubbletea/v2"
	"strings"
)

// NewMarkupModel teaches Native Description editing through the real UI.
// The separate lesson keeps the original general tutorial's sequence intact.
func NewMarkupModel(app tea.Model, literalTabAlias string) tea.Model {
	return &Model{app: app, steps: markupTutorialSteps(literalTabAlias)}
}

func markupTutorialSteps(literalTabAlias string) []tutorialStep {
	aliasLabel := strings.Replace(strings.ToUpper(literalTabAlias), "CTRL+", "Ctrl+", 1)
	return []tutorialStep{
		infoStep("Rich-text workshop", "Native editing in the real UI; every Task lives in a disposable database.", "Use a wide terminal. Only requested keys advance; Ctrl+C exits.", "Enter"),
		actionStep("Open practice Task", "Enter the Tasks Pane, G to the last Task, then Enter.", "This fixture contains paragraphs, Code and a preserved Mention.", keys("enter", "G", "enter")...),
		actionStep("Native navigation", "Press gn, Enter, then gg to reach Alpha.", "Native text has document anchors; active marks disclose editable delimiters.", keys("g", "n", "enter", "g", "g")...),
		actionStep("Visual formatting", "Select Alpha with viw, Space b applies Bold, Esc ends selection.", "Characterwise Visual formatting preserves unrelated content and Opaque atoms.", keys("v", "i", "w", "space", "b", "esc")...),
		actionStep("Local history", "Undo with u, then redo with Ctrl+R.", "Description history is local and remains available after a successful save.", keys("u", "ctrl+r")...),
		actionStep("Link destination", "Select Alpha with viw, Space l; enter any URL, Enter to apply, then Esc.", "Example: https://example.test/tutorial. Paste/edit freely; invalid URLs stay open; empty unlinks.", append(keys("v", "i", "w", "space", "l"), expectedKey{value: "enter", label: "your URL · Enter apply · Esc cancel", freeInput: true}, key("esc", "Esc"))...),
		actionStep("Typed syntax", "o opens a paragraph; type ***Both*** and Esc.", "Direct typing creates marks. Paste and imported marker-like text remain literal.", append(keys("o"), append(textKeys("***Both***"), keys("esc")...)...)...),
		actionStep("Checkbox in Description", "o, type Check item, Esc, then Space x and x.", "Space x creates a Checkbox; x toggles it. This is independent of Task Done.", append(keys("o"), append(textKeys("Check item"), keys("esc", "space", "x", "x")...)...)...),
		actionStep("Leave checklist", "Space x converts this entry back into ordinary text.", "An explicit structural conversion keeps text and removes the Checkbox wrapper.", keys("space", "x")...),
		actionStep("Quote and unwrap", "Press Space q twice: quote the paragraph, then unwrap it.", "Known source references are retained on both remaining fragments when a quote is split.", keys("space", "q", "space", "q")...),
		actionStep("Create Code", "Open another empty paragraph with o, Esc, then Space c.", "Three/four typed backticks plus Enter is the alternate creation route.", keys("o", "esc", "space", "c")...),
		actionStep("Edit Code", "Type print(1), Enter, then "+aliasLabel+" Tab for a literal Tab, Esc.", "Creation enters Insert. Enter is a source newline; "+aliasLabel+" Tab explicitly inserts a Tab.", append(textKeys("print(1)"), keys("enter", literalTabAlias, "tab", "esc")...)...),
		actionStep("Convert Tabs", "Type :retab and Enter.", "Tabs become Spaces to the next tabstop; :retaball also covers Native prose. One Undo step.", commandKeys("retab")...),
		actionStep("Leave Code", "Press ü, type After code, Esc.", "ü/Ü creates an outside paragraph in Insert; ]Space/[Space is the Normal-mode alternative.", append(keys("ü"), append(textKeys("After code"), keys("esc")...)...)...),
		actionStep("Protected Mention", "G reaches the final Mention. x then n cancels deletion.", "Opaque content is readable; only destructive operations need confirmation.", keys("G", "x", "n")...),
		actionStep("Opaque Undo", "x then y confirms removal; u restores its original bytes.", "Native neighbors remain editable. Task Done is independent.", keys("x", "y", "u")...),
		actionStep("Save", "Type :w and Enter.", "A successful save adopts generated Checkbox IDs; failures retain the draft.", commandKeys("w")...),
		actionStep("Discard later edits", "i, type oops, Esc, then :e! Enter.", "Restores the saved Task baseline. External conflicts require explicit :descriptionreload; no merge.", append(keys("i"), append(textKeys("oops"), append(keys("esc"), commandKeys("e!")...)...)...)...),
		actionStep("Open blocked example", "Close Detail with :q; k selects the blocked example, Enter opens it.", "Recovery is explicit and changes only a draft until saved.", append(commandKeys("q"), keys("k", "enter")...)...),
		actionStep("Recover readable text", "Type :descriptionrec, Tab, Enter, then y (or type the full command).", "Preview keeps Broken as text; saving replaces the broken HTML and its formatting. Tab completes unique commands.", append(commandKeys("descriptionrecover"), keys("y")...)...),
		actionStep("Discard recovery", "Esc leaves Insert; then type :e! and Enter to restore the original source.", "No recovery content has been written to SQLite.", append(keys("esc"), commandKeys("e!")...)...),
		actionStep("Clear blocked draft", "Type :descriptionclear, Enter, then y.", "Clear is also confirmed and unsaved; :q! will discard it.", append(commandKeys("descriptionclear"), keys("y")...)...),
		actionStep("Discard and finish", "Type :q! Enter, then :qa Enter.", "The disposable database is deleted when the tutorial exits.", append(commandKeys("q!"), commandKeys("qa")...)...),
	}
}
