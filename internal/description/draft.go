package description

import (
	"errors"
	"fmt"
	"github.com/rivo/uniseg"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"
)

// Actions describe document operations, never edits to serialized HTML.
type ActionKind uint8

const (
	Insert ActionKind = iota
	Split
	HardBreak
	Backspace
	Delete
	DeleteBlock
	OpenBelow
	OpenAbove
	Convert
	ToggleMark
	ClearMarks
	DeleteSelection
	ReplaceSelection
	ClearDescription
	RecoverText
	IndentList
	OutdentList
	ConvertList
	ToggleCheckbox
	ToggleQuote
	WrapQuote
	UnwrapQuote
	CreateCode
	EscapeCodeMarker
	CodeBefore
	CodeAfter
	InsertTab
	IndentCode
	OutdentCode
	Retab
	RetabAll
	EditLink
	Type
	ReplaceCharacters
	ToggleCase
)

type Action struct {
	Count           int
	Kind            ActionKind
	Text            string
	inside          bool
	skipEscape      bool
	Tabstop, Indent int
}

var ErrConfirmation = errors.New("Preserved content would change; confirm this operation")
var ErrQuoteMetadataConfirmation = fmt.Errorf("Zitat-Metadaten würden gelöscht werden. Bitte Löschung bestätigen.: %w", ErrConfirmation)

type Selection struct {
	Start, End             Anchor
	Blocks, WholeBlock     bool
	startSyntax, endSyntax int
	exclusive              bool // Internal operator ranges use a half-open end.
	codeBlock              ID   // Code-line origin remains stable when endpoints swap.
}
type draftState struct {
	nodes     []Node
	cursor    Anchor
	selection *Selection
	recovered bool
	syntax    int
	inside    bool
}

type undoGroup struct {
	before   draftState
	recorded bool
}

// Draft owns live identities and local history. Its preservation source remains
// immutable even after a save; the persistence baseline advances independently.
type Draft struct {
	confirmation      string
	confirmationBrief string
	group             *undoGroup
	origin            *Document
	state             draftState
	undo, redo        []draftState
	next              ID
	baseline          string
	savedNodes        []Node
	savedRecovered    bool
}

func NewDraft(doc *Document) *Draft {
	d := &Draft{origin: doc, baseline: doc.SourceHTML(), state: draftState{nodes: doc.Snapshot()}}
	d.assign(d.state.nodes)
	d.savedNodes = cloneNodes(d.state.nodes)
	d.state.cursor = d.first(d.state.nodes)
	if n := findNode(d.state.nodes, d.state.cursor.Node); n != nil && n.Tag == "a" && len(n.Children) == 0 {
		d.state.syntax = -1
	}
	return d
}

// NewTextDraft gives legacy plain descriptions the same paragraph editor while
// retaining their exact persistence baseline. Only a real edit produces HTML;
// undoing back to the initial nodes restores the original bytes.
func NewTextDraft(source string) *Draft {
	if !utf8.ValidString(source) {
		return NewDraft(Parse(source))
	}
	d := NewDraft(Parse(""))
	text := strings.ReplaceAll(strings.ReplaceAll(source, "\r\n", "\n"), "\r", "\n")
	for _, line := range strings.Split(text, "\n") {
		var children []Node
		if line != "" {
			children = []Node{d.newNode(Node{Kind: Text, Text: line})}
		}
		d.state.nodes = append(d.state.nodes, d.newNode(Node{Kind: Element, Tag: "p", Children: children}))
	}
	d.baseline = source
	d.savedNodes = cloneNodes(d.state.nodes)
	d.state.cursor = d.first(d.state.nodes)
	return d
}

func (d *Draft) assign(ns []Node) {
	var scan func([]Node)
	scan = func(ns []Node) {
		for _, n := range ns {
			d.next = max(d.next, n.ID)
			scan(n.Children)
		}
	}
	scan(ns)
	var fill func([]Node)
	fill = func(ns []Node) {
		for i := range ns {
			if ns[i].ID == 0 {
				d.next++
				ns[i].ID = d.next
			}
			fill(ns[i].Children)
		}
	}
	fill(ns)
}
func (d *Draft) Snapshot() []Node { return cloneNodes(d.state.nodes) }
func (d *Draft) Cursor() Anchor   { return d.state.cursor }
func (d *Draft) Baseline() string { return d.baseline }
func (d *Draft) Classification() Classification {
	if d.state.recovered {
		return Native
	}
	return d.origin.class
}
func copyState(s draftState) draftState {
	s.nodes = cloneNodes(s.nodes)
	if s.selection != nil {
		v := *s.selection
		s.selection = &v
	}
	return s
}
func (d *Draft) live() *Document {
	doc := *d.origin
	doc.nodes = d.state.nodes
	if d.state.recovered {
		doc = *Parse("")
		doc.nodes = d.state.nodes
	}
	doc.indexAnchors()
	for id, loc := range doc.locations {
		if loc.Node.Kind == Opaque {
			for i := len(loc.Ancestors) - 1; i >= 0; i-- {
				parent := doc.locations[loc.Ancestors[i]].Node
				if parent.Tag == "p" || parent.Tag == "h1" || parent.Tag == "h2" || parent.Tag == "h3" {
					loc.Block = parent.ID
					doc.locations[id] = loc
					break
				}
			}
		}
	}
	return &doc
}
func (d *Draft) SetCursor(a Anchor) error {
	if _, err := d.live().ResolveAnchor(a); err != nil {
		return err
	}
	d.state.cursor = a
	d.state.syntax = 0
	d.state.inside = false
	if d.state.selection != nil {
		d.state.selection.End = a
		d.state.selection.endSyntax = 0
	}
	return nil
}
func (d *Draft) Selection() *Selection {
	if d.state.selection == nil {
		return nil
	}
	s := *d.state.selection
	return &s
}
func (d *Draft) Select(blocks, whole bool) {
	d.state.selection = &Selection{Start: d.state.cursor, End: d.state.cursor, Blocks: blocks, WholeBlock: whole, startSyntax: d.state.syntax, endSyntax: d.state.syntax}
	if blocks && !whole {
		loc, err := d.live().ResolveAnchor(d.state.cursor)
		if n := findNode(d.state.nodes, loc.Block); err == nil && n != nil && n.Tag == "pre" {
			d.state.selection.codeBlock = n.ID
		}
	}
}
func (d *Draft) CancelSelection() { d.state.selection = nil }
func (d *Draft) SwapSelection() {
	if s := d.state.selection; s != nil {
		s.Start, s.End = s.End, s.Start
		s.startSyntax, s.endSyntax = s.endSyntax, s.startSyntax
		d.state.syntax = s.endSyntax
		d.state.cursor = s.End
	}
}
func (d *Draft) first(ns []Node) Anchor {
	for _, n := range ns {
		if n.Kind == Element && checkboxControlTag(n.Tag) {
			continue
		}
		if n.Kind == Text || n.Kind == Opaque || n.Tag == "br" {
			return Anchor{n.ID, 0}
		}
		if len(n.Children) > 0 {
			return d.first(n.Children)
		}
		return Anchor{n.ID, 0}
	}
	return Anchor{}
}
func (d *Draft) intent(ns []Node) Preservation {
	p := Preservation{Positions: map[ID][]int{}}
	var visit func([]Node, []int)
	visit = func(ns []Node, path []int) {
		for i, n := range ns {
			next := append(slices.Clone(path), i)
			if n.Kind == Opaque {
				p.Positions[n.ID] = next
			}
			visit(n.Children, next)
		}
	}
	visit(normalizeRoot(cloneNodes(ns)), nil)
	for _, a := range d.origin.atoms {
		if _, ok := p.Positions[a.id]; !ok {
			p.Removed = append(p.Removed, a.id)
		}
	}
	return p
}
func (d *Draft) HTML() (string, error) {
	if d.state.recovered {
		return Parse("").SerializeSnapshot(d.state.nodes, Preservation{})
	}
	return d.origin.SerializeSnapshot(d.state.nodes, d.intent(d.state.nodes))
}
func (d *Draft) PrepareSave() (SaveCandidate, error) {
	doc := d.origin
	if d.state.recovered {
		doc = Parse("")
	}
	c, err := doc.PrepareSave(d.state.nodes, d.intentFor(doc), nil)
	if err != nil {
		return c, err
	}
	c.baseline = d.baseline
	c.recovered = d.state.recovered
	return c, nil
}
func (d *Draft) intentFor(doc *Document) Preservation {
	if doc != d.origin {
		return Preservation{}
	}
	return d.intent(d.state.nodes)
}

// AdoptSave uses the exact candidate's node identities, never matching by text
// or position. Histories retain those identities on later undo and save.
func (d *Draft) AdoptSave(c SaveCandidate) error {
	if !c.Valid() || c.baseline != d.baseline || c.nodes == nil {
		return fmt.Errorf("invalid draft save adoption")
	}
	ids := map[ID]string{}
	var collect func([]Node)
	collect = func(ns []Node) {
		for _, n := range ns {
			if n.Tag == "li" && attributeValue(n.Attributes, "data-type") == "taskItem" {
				ids[n.ID] = attributeValue(n.Attributes, "data-task-id")
			}
			collect(n.Children)
		}
	}
	collect(*c.nodes)
	var apply func([]Node)
	apply = func(ns []Node) {
		for i := range ns {
			if id, ok := ids[ns[i].ID]; ok {
				setAttribute(&ns[i], "data-task-id", id)
			}
			apply(ns[i].Children)
		}
	}
	apply(d.state.nodes)
	for i := range d.undo {
		apply(d.undo[i].nodes)
	}
	for i := range d.redo {
		apply(d.redo[i].nodes)
	}
	d.savedNodes = cloneNodes(*c.nodes)
	d.savedRecovered = c.recovered
	d.baseline = c.HTML()
	return nil
}
func setAttribute(n *Node, name, value string) {
	for i := range n.Attributes {
		if n.Attributes[i].Name == name {
			n.Attributes[i].Value = value
			return
		}
	}
	n.Attributes = append(n.Attributes, Attribute{name, value})
}
func (d *Draft) View(options ...ViewOptions) *View {
	v := NewView(d.live(), options...)
	_ = v.SetAnchor(d.state.cursor)
	v.target.Syntax = d.state.syntax
	return v
}

// BeginUndoGroup joins a change operator and its following Insert session.
// Call before selecting/deleting, so Undo also restores the original cursor.
func (d *Draft) BeginUndoGroup() {
	if d.group == nil {
		d.group = &undoGroup{before: copyState(d.state)}
	}
}

// EndUndoGroup makes subsequent actions independently undoable.
func (d *Draft) EndUndoGroup() { d.group = nil }

func (d *Draft) Undo() bool {
	d.EndUndoGroup()
	if len(d.undo) == 0 {
		return false
	}
	d.redo = append(d.redo, copyState(d.state))
	d.state = d.undo[len(d.undo)-1]
	d.undo = d.undo[:len(d.undo)-1]
	return true
}
func (d *Draft) Redo() bool {
	d.EndUndoGroup()
	if len(d.redo) == 0 {
		return false
	}
	d.undo = append(d.undo, copyState(d.state))
	d.state = d.redo[len(d.redo)-1]
	d.redo = d.redo[:len(d.redo)-1]
	return true
}
func (d *Draft) Apply(a Action, confirmed bool) error {
	d.confirmation, d.confirmationBrief = "", ""
	if !utf8.ValidString(a.Text) {
		return fmt.Errorf("%w: invalid UTF-8 text", ErrStructure)
	}
	if d.Classification() == Blocked && a.Kind != ClearDescription && a.Kind != RecoverText {
		return ErrBlocked
	}
	before := copyState(d.state)
	oldNext := d.next
	err := d.edit(a)
	if err == nil && (a.Kind == Delete || a.Kind == Backspace || a.Kind == DeleteSelection || a.Kind == ReplaceSelection) {
		d.removeDeletedLinkLabels(before, a.Kind)
	}
	if err == nil {
		var stale func([]Node)
		stale = func(ns []Node) {
			for i := range ns {
				n := &ns[i]
				if old := findNode(before.nodes, n.ID); n.syntaxSource != "" && old != nil && !reflect.DeepEqual(n.Children, old.Children) {
					n.syntaxSource = ""
					n.sourceOrigins = nil
				}
				stale(n.Children)
			}
		}
		stale(d.state.nodes)
	}
	if err == nil && (a.Kind == DeleteBlock || a.Kind == DeleteSelection || a.Kind == ReplaceSelection) {
		d.pruneDeletedContainers(before.nodes)
	}
	if err == nil {
		d.assign(d.state.nodes)
		err = validateNative(d.state.nodes, "", false)
	}
	if err == nil && (confirmed && d.needsConfirmation(before, a) || (a.Kind == ToggleMark || a.Kind == ClearMarks) && len(d.origin.atoms) > 0) {
		_, err = d.HTML()
	}
	if err == nil && !confirmed && d.needsConfirmation(before, a) {
		err = ErrConfirmation
		d.confirmation = d.confirmationEffect(before, a)
		if (a.Kind == Split || a.Kind == ToggleQuote || a.Kind == UnwrapQuote) && d.quoteMetadataRemoved(before.nodes) {
			err = ErrQuoteMetadataConfirmation
			d.confirmationBrief = "DELETED: quote metadata"
			d.confirmation = "Quote metadata will be DELETED"
		}
	}
	if err != nil {
		d.state = before
		d.next = oldNext
		return err
	}
	if reflect.DeepEqual(before, d.state) {
		return nil
	}
	if d.group == nil {
		d.undo = append(d.undo, before)
	} else if !d.group.recorded {
		d.undo = append(d.undo, d.group.before)
		d.group.recorded = true
	}
	d.redo = nil
	return nil
}

// ConfirmationMessage describes the actual effect of the last unconfirmed edit.
// It contains no source HTML or hidden attributes.
func (d *Draft) ConfirmationMessage() string { return d.confirmation }

// ConfirmationSummary retains the consequence when the full preview cannot fit.
func (d *Draft) ConfirmationSummary() string { return d.confirmationBrief }

func (d *Draft) confirmationEffect(before draftState, a Action) string {
	switch a.Kind {
	case ClearDescription:
		d.confirmationBrief = "DELETED: entire Description"
		return "Entire Description will be DELETED (draft only until save)"
	case RecoverText:
		d.confirmationBrief = "REPLACED: HTML → plain text"
		text := strings.Join(strings.Fields(d.origin.RecoveryText()), " ")
		if text == "" {
			d.confirmationBrief = "EMPTY: replaces original HTML"
			return "No readable text: recovery creates EMPTY Description; HTML replaced on save"
		}
		preview := ""
		g := uniseg.NewGraphemes(text)
		for count := 0; g.Next(); count++ {
			if count == 24 {
				preview += "…"
				break
			}
			preview += g.Str()
		}
		return fmt.Sprintf("Recover %q; HTML/formatting/objects REPLACED on save", preview)
	}
	removed := 0
	current := map[ID]bool{}
	var collect func([]Node)
	collect = func(ns []Node) {
		for _, n := range ns {
			current[n.ID] = true
			collect(n.Children)
		}
	}
	collect(d.state.nodes)
	var visit func([]Node)
	visit = func(ns []Node) {
		for _, n := range ns {
			if n.Kind == Opaque {
				if !current[n.ID] {
					removed++
				}
				continue
			}
			visit(n.Children)
		}
	}
	visit(before.nodes)
	if removed == 1 {
		d.confirmationBrief = "DELETED: protected object"
		return "Protected object will be DELETED"
	}
	if removed > 1 {
		d.confirmationBrief = fmt.Sprintf("DELETED: %d protected objects", removed)
		return fmt.Sprintf("%d protected objects will be DELETED", removed)
	}
	d.confirmationBrief = "CHANGED: structure; objects kept"
	return "Structure around protected objects will change; objects retained"
}

func (d *Draft) needsConfirmation(before draftState, a Action) bool {
	if a.Kind == ToggleMark || a.Kind == ClearMarks {
		return false
	}
	if a.Kind == ClearDescription || a.Kind == RecoverText {
		return true
	}
	old := map[ID]string{}
	var visit func([]Node, []ID, map[ID]string)
	visit = func(ns []Node, parents []ID, out map[ID]string) {
		for _, n := range ns {
			if n.Kind == Opaque {
				out[n.ID] = fmt.Sprint(parents)
			}
			visit(n.Children, append(slices.Clone(parents), n.ID), out)
		}
	}
	visit(before.nodes, nil, old)
	current := map[ID]string{}
	visit(d.state.nodes, nil, current)
	for id, path := range old {
		if current[id] != path {
			return true
		}
	}
	if (a.Kind == Split || a.Kind == ToggleQuote || a.Kind == UnwrapQuote) && d.quoteMetadataRemoved(before.nodes) {
		return true
	}
	if a.Kind == Convert {
		loc, _ := d.live().ResolveAnchor(d.state.cursor)
		for _, id := range loc.Ancestors {
			if n := findNode(before.nodes, id); n != nil && containsOpaque(*n) {
				return true
			}
		}
		n := findNode(before.nodes, loc.Block)
		return n != nil && containsOpaque(*n)
	}
	return false
}

// Enter can remove an empty quote layer. Its metadata needs the same explicit
// loss confirmation as other quote unwrap operations.
func (d *Draft) quoteMetadataRemoved(ns []Node) bool {
	for _, n := range ns {
		if n.Tag == "blockquote" && len(n.Attributes) > 0 && findNode(d.state.nodes, n.ID) == nil {
			return true
		}
		if d.quoteMetadataRemoved(n.Children) {
			return true
		}
	}
	return false
}
func containsOpaque(n Node) bool {
	if n.Kind == Opaque {
		return true
	}
	for _, c := range n.Children {
		if containsOpaque(c) {
			return true
		}
	}
	return false
}
func findNode(ns []Node, id ID) *Node {
	for i := range ns {
		if ns[i].ID == id {
			return &ns[i]
		}
		if n := findNode(ns[i].Children, id); n != nil {
			return n
		}
	}
	return nil
}
func nodeList(ns *[]Node, id ID) (*[]Node, int) {
	for i := range *ns {
		if (*ns)[i].ID == id {
			return ns, i
		}
		if p, j := nodeList(&(*ns)[i].Children, id); p != nil {
			return p, j
		}
	}
	return nil, 0
}
func graphemeAfter(s string, offset int) int {
	if offset >= len(s) {
		return len(s)
	}
	g := uniseg.NewGraphemes(s[offset:])
	if g.Next() {
		_, end := g.Positions()
		return offset + end
	}
	return offset
}
func graphemeBefore(s string, offset int) int {
	g := uniseg.NewGraphemes(s)
	prev := 0
	for g.Next() {
		start, end := g.Positions()
		if end >= offset {
			return start
		}
		prev = start
	}
	return prev
}
func (d *Draft) After(a Anchor) Anchor {
	n := findNode(d.state.nodes, a.Node)
	if n != nil && n.Kind == Text {
		a.Offset = graphemeAfter(n.Text, a.Offset)
	} else {
		a.Offset = 1
	}
	return a
}
func (d *Draft) edit(a Action) error {
	if a.Kind == ReplaceCharacters || a.Kind == ToggleCase {
		return d.transformCharacters(a)
	}
	if a.Kind == Delete && a.Count > 1 {
		for range min(a.Count, 100000) {
			if d.state.syntax == 0 && d.Cursor() == d.LineBoundary(true) {
				break
			}
			if err := d.edit(Action{Kind: Delete}); err != nil {
				return err
			}
		}
		return nil
	}
	switch a.Kind {
	case Split, HardBreak, OpenAbove, OpenBelow, DeleteBlock, Convert, ConvertList,
		CreateCode, CodeBefore, CodeAfter, ClearDescription, RecoverText:
		// Structural actions choose a semantic target, not a delimiter gap.
		d.state.syntax = 0
		d.state.inside = false
	}

	if a.Kind == Type && a.Text != "" {
		d.continueClosingRun([]rune(a.Text)[0])
	}
	if d.state.syntax != 0 && (a.Kind == Insert || a.Kind == Type || a.Kind == Delete || a.Kind == Backspace) {
		return d.editSyntax(a)
	}
	if (a.Kind == Backspace || a.Kind == Delete) && !d.InCode() {
		if handled, err := d.editAdjacentSyntax(a); handled || err != nil {
			return err
		}
	}
	if a.Kind == Type {
		return d.typeText(a.Text, a.inside || d.state.inside)
	}
	switch a.Kind {

	case EscapeCodeMarker, CreateCode, CodeBefore, CodeAfter, InsertTab, IndentCode, OutdentCode, Retab, RetabAll:
		return d.editCode(a)
	case EditLink:
		return d.editLink(a.Text)
	case IndentList, OutdentList:
		return d.moveList(a.Kind)
	case ConvertList:
		return d.convertList(a.Text)
	case ToggleCheckbox:
		return d.toggleCheckbox()
	case ToggleQuote, WrapQuote, UnwrapQuote:
		return d.editQuote(a.Kind)
	}
	if a.Kind == ClearDescription || a.Kind == RecoverText {
		d.state.nodes = nil
		d.state.recovered = true
		d.state.selection = nil
		if a.Kind == RecoverText {
			for _, line := range strings.Split(d.origin.RecoveryText(), "\n") {
				if line != "" {
					d.state.nodes = append(d.state.nodes, Node{Kind: Element, Tag: "p", Children: []Node{{Kind: Text, Text: line}}})
				}
			}
		}
		d.assign(d.state.nodes)
		d.state.cursor = d.first(d.state.nodes)
		return nil
	}
	if a.Kind == ToggleMark || a.Kind == ClearMarks {
		return d.format(a)
	}
	if a.Kind == DeleteSelection || a.Kind == ReplaceSelection {
		return d.deleteSelection(a)
	}
	if len(d.state.nodes) == 0 {
		d.state.nodes = []Node{{Kind: Element, Tag: "p"}}
		d.assign(d.state.nodes)
		d.state.cursor = d.first(d.state.nodes)
	}
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return err
	}
	n := findNode(d.state.nodes, d.state.cursor.Node)
	b := findNode(d.state.nodes, loc.Block)
	if b == nil {
		return ErrStructure
	}
	if b.Tag == "pre" && (a.Kind == Split || a.Kind == HardBreak || a.Kind == OpenAbove || a.Kind == OpenBelow || a.Kind == DeleteBlock || (a.Kind == Delete || a.Kind == Backspace) && n.Kind != Opaque) {
		return d.editCodeSource(b, a)
	}
	if isList(*b) && len(b.Children) == 0 {
		if a.Kind == Split {
			return nil
		}
		if a.Kind == Insert && a.Text != "" {
			entry := d.newNode(newEntry(*b))
			b.Children = []Node{entry}
			d.state.cursor = d.first(b.Children)
			return d.edit(a)
		}
	}
	if li := d.selectedEntryAtCursor(); li != nil {
		switch a.Kind {
		case Split:
			return d.splitEntry(li)
		case OpenAbove, OpenBelow:
			return d.openEntry(li, a.Kind)
		case DeleteBlock:
			list, i := nodeList(&d.state.nodes, li.ID)
			*list = slices.Delete(*list, i, i+1)
			d.state.cursor = d.first(d.state.nodes)
			d.state.selection = nil
			return nil
		}
	}
	switch a.Kind {
	case Insert:
		a.inside = a.inside || d.state.inside
		if strings.Contains(a.Text, "\n") && b.Tag != "pre" {
			parts := strings.Split(a.Text, "\n")
			for i, part := range parts {
				if i > 0 {
					if err := d.edit(Action{Kind: Split}); err != nil {
						return err
					}
				}
				if part != "" {
					if err := d.edit(Action{Kind: Insert, Text: part}); err != nil {
						return err
					}
				}
			}
			return nil
		}
		if b.Tag != "pre" {
			if handled, err := d.insertInlineEdge(a); handled || err != nil {
				return err
			}
			n = findNode(d.state.nodes, d.state.cursor.Node)
		}
		if b.Tag == "pre" && len(codeLeaves(b)) == 0 {
			n = d.materializeCodeText(b)
			d.state.cursor = Anchor{Node: n.ID}
		}
		if n.Kind == Opaque {
			list, i := nodeList(&d.state.nodes, n.ID)
			if d.state.cursor.Offset > 0 {
				i++
			}
			created := Node{Kind: Text, Text: a.Text}
			blockAtom := n.blockOpaque
			if len(loc.Ancestors) > 0 {
				parent := findNode(d.state.nodes, loc.Ancestors[len(loc.Ancestors)-1])
				if parent != nil && (parent.Tag == "p" || parent.Tag == "h1" || parent.Tag == "h2" || parent.Tag == "h3" || parent.Tag == "a" || isInlineMark(parent.Tag)) {
					blockAtom = false
				}
			}
			if blockAtom {
				created = Node{Kind: Element, Tag: "p", Children: []Node{created}}
			}
			created = d.newNode(created)
			*list = slices.Insert(*list, i, created)
			d.state.cursor = d.first([]Node{created})
			if t := findNode(d.state.nodes, d.state.cursor.Node); t != nil && t.Kind == Text {
				d.state.cursor.Offset = len(a.Text)
			}
			return nil
		}
		if n.Kind == Text {
			s := n.Text
			off := d.state.cursor.Offset
			flags := textOrigins(*n)
			n.typed = slices.Replace(flags, off, off, make([]byte, len(a.Text))...)
			n.Text = s[:off] + a.Text + s[off:]
			d.state.cursor.Offset += len(a.Text)
			return nil
		}
		if n.Tag == "br" {
			list, i := nodeList(&d.state.nodes, n.ID)
			if d.state.cursor.Offset > 0 {
				i++
			}
			newNode := Node{Kind: Text, Text: a.Text}
			newNode = d.newNode(newNode)
			*list = slices.Insert(*list, i, newNode)
			d.state.cursor = Anchor{newNode.ID, len(a.Text)}
			return nil
		}
		if len(n.Children) == 0 && n.Tag != "hr" {
			n.Children = []Node{{Kind: Text, Text: a.Text}}
			d.assign(n.Children)
			d.state.cursor = Anchor{n.Children[0].ID, len(a.Text)}
			return nil
		}
		return ErrStructure
	case Split, HardBreak:
		if a.Kind == Split && b.Tag == "p" && len(b.Children) == 0 {
			list, i := nodeList(&d.state.nodes, b.ID)
			_ = i
			for _, id := range loc.Ancestors {
				q := findNode(d.state.nodes, id)
				if q != nil && q.Tag == "blockquote" && list == &q.Children {
					outer, j := nodeList(&d.state.nodes, q.ID)
					p := *b
					*list = slices.Delete(*list, i, i+1)
					if len(q.Children) == 0 {
						*outer = slices.Delete(*outer, j, j+1)
					} else {
						j++
					}
					*outer = slices.Insert(*outer, j, p)
					d.state.cursor = Anchor{p.ID, 0}
					return nil
				}
			}
		}
		left, right, err := splitAt(b.Children, d.state.cursor)
		if err != nil {
			if n.ID == b.ID && len(b.Children) == 0 {
				left = nil
				right = nil
			} else {
				return err
			}
		}
		if a.Kind == HardBreak {
			br := Node{Kind: Element, Tag: "br"}
			br = d.newNode(br)
			b.Children = append(left, br)
			b.Children = append(b.Children, right...)
			d.state.cursor = Anchor{br.ID, 1}
			return nil
		}
		if b.Tag != "p" && b.Tag != "h1" && b.Tag != "h2" && b.Tag != "h3" {
			return fmt.Errorf("split for this structure belongs to Ticket 07")
		}
		list, i := nodeList(&d.state.nodes, b.ID)
		b.Children = left
		newBlock := Node{Kind: Element, Tag: b.Tag, Children: right}
		newBlock = d.newNode(newBlock)
		*list = slices.Insert(*list, i+1, newBlock)
		d.state.cursor = d.first([]Node{(*list)[i+1]})
		return nil
	case OpenBelow, OpenAbove:
		list, i := nodeList(&d.state.nodes, b.ID)
		if a.Kind == OpenBelow {
			i++
		}
		p := Node{Kind: Element, Tag: "p"}
		d.next++
		p.ID = d.next
		*list = slices.Insert(*list, i, p)
		d.state.cursor = Anchor{p.ID, 0}
		return nil
	case Convert:
		if b.Kind == Opaque || (b.Tag != "p" && b.Tag != "h1" && b.Tag != "h2" && b.Tag != "h3") {
			return ErrStructure
		}
		if a.Text != "p" && a.Text != "h1" && a.Text != "h2" && a.Text != "h3" {
			return ErrStructure
		}
		if b.Tag == a.Text {
			b.Tag = "p"
		} else {
			b.Tag = a.Text
		}
		return nil
	case DeleteBlock:
		list, i := nodeList(&d.state.nodes, b.ID)
		*list = slices.Delete(*list, i, i+1)
		if len(*list) > 0 {
			d.state.cursor = d.first([]Node{(*list)[min(i, len(*list)-1)]})
		} else {
			d.state.cursor = d.first(d.state.nodes)
		}
		d.state.selection = nil
		return nil
	case Backspace, Delete:
		if n.Kind == Text {
			off := d.state.cursor.Offset
			start, end := off, off
			if a.Kind == Backspace && off > 0 {
				start = graphemeBefore(n.Text, off)
			}
			if a.Kind == Delete && off < len(n.Text) {
				end = graphemeAfter(n.Text, off)
			}
			if start != end {
				n.typed = slices.Delete(textOrigins(*n), start, end)
				n.Text = n.Text[:start] + n.Text[end:]
				d.state.cursor.Offset = start
				return nil
			}
		}
		if (n.Kind == Opaque || n.Tag == "br") && (a.Kind == Backspace && d.state.cursor.Offset == 0 || a.Kind == Delete && d.state.cursor.Offset == 1) {
			return d.deleteBoundary(b, a.Kind)
		}
		if n.Kind == Opaque || n.Tag == "br" {
			ext, _ := d.extents()
			pos := ext[n.ID].start
			list, i := nodeList(&d.state.nodes, n.ID)
			*list = slices.Delete(*list, i, i+1)
			d.state.cursor = d.anchorAt(pos)
			return nil
		}
		return d.deleteBoundary(b, a.Kind)
	}
	return ErrStructure
}
func splitAt(ns []Node, a Anchor) ([]Node, []Node, error) {
	for i, n := range ns {
		if n.ID == a.Node {
			left := cloneNodes(ns[:i])
			right := cloneNodes(ns[i+1:])
			if n.Kind == Text {
				l, r := n, n
				l.Text = n.Text[:a.Offset]
				r.Text = n.Text[a.Offset:]
				l.typed = slices.Clone(textOrigins(n)[:a.Offset])
				r.typed = slices.Clone(textOrigins(n)[a.Offset:])
				r.ID = 0
				if l.Text != "" {
					left = append(left, l)
				}
				if r.Text != "" {
					right = append([]Node{r}, right...)
				}
			} else if a.Offset == 0 {
				right = append([]Node{n}, right...)
			} else {
				left = append(left, n)
			}
			return left, right, nil
		}
		l, r, err := splitAt(n.Children, a)
		if err == nil {
			left := cloneNodes(ns[:i])
			right := cloneNodes(ns[i+1:])
			if len(l) > 0 {
				v := n
				v.Children = l
				v.syntaxSource = ""
				v.sourceOrigins = nil
				left = append(left, v)
			}
			if len(r) > 0 {
				v := n
				v.ID = 0
				v.Children = r
				v.syntaxSource = ""
				v.sourceOrigins = nil
				for _, attr := range v.Attributes {
					if attr.Name == "id" {
						return nil, nil, ErrStructure
					}
				}
				right = append([]Node{v}, right...)
			}
			return left, right, nil
		}
	}
	return nil, nil, ErrAnchor
}

// DraftHTML is a cheap persistence-shaped dirty token for the surrounding Task
// draft. Only PrepareSave performs the independent serialize/reparse check.
func (d *Draft) DraftHTML() string {
	if d.Classification() == Blocked {
		return d.origin.SourceHTML()
	}
	if d.state.recovered == d.savedRecovered && reflect.DeepEqual(d.state.nodes, d.savedNodes) {
		return d.baseline
	}
	var out strings.Builder
	doc := d.origin
	if d.state.recovered {
		doc = Parse("")
	}
	var retained []atom
	p := d.intentFor(doc)
	removed := map[ID]bool{}
	for _, id := range p.Removed {
		removed[id] = true
	}
	// Use current (not normalized) paths for draft-only output.
	p.Positions = map[ID][]int{}
	var walk func([]Node, []int)
	walk = func(ns []Node, path []int) {
		for i, n := range ns {
			at := append(slices.Clone(path), i)
			if n.Kind == Opaque {
				p.Positions[n.ID] = at
			}
			walk(n.Children, at)
		}
	}
	walk(d.state.nodes, nil)
	if err := doc.writeNodes(&out, d.state.nodes, nil, nil, removed, p.Positions, &retained, map[ID]bool{}); err != nil {
		return d.origin.SourceHTML()
	}
	return out.String()
}
func (d *Draft) UpdateView(v *View) {
	v.doc = d.live()
	v.blocks = nil
	if d.Classification() == Blocked {
		other := NewView(v.doc, v.options)
		v.blocks = other.blocks
	} else {
		v.compile(d.state.nodes, "", 0, 0, false)
	}
	v.ensureEmptyTarget()
	if err := v.SetAnchor(d.state.cursor); err != nil && d.state.cursor.Node != 0 {
		v.target = Target{Anchor: d.state.cursor}
	}
	v.target.Syntax = d.state.syntax
}
func (d *Draft) newNode(n Node) Node { ns := []Node{n}; d.assign(ns); return ns[0] }
func (d *Draft) pruneDeletedContainers(before []Node) {
	var clean func([]Node) []Node
	clean = func(ns []Node) []Node {
		var out []Node
		for _, n := range ns {
			n.Children = clean(n.Children)
			original := findNode(before, n.ID)
			if len(n.Children) == 0 && original != nil && len(original.Children) > 0 && (n.Tag == "blockquote" || n.Tag == "ul" || n.Tag == "ol") {
				continue
			}
			out = append(out, n)
		}
		return out
	}
	d.state.nodes = clean(d.state.nodes)
	if findNode(d.state.nodes, d.state.cursor.Node) == nil {
		d.state.cursor = d.first(d.state.nodes)
	}
}

// InsertionView includes insertion boundaries without adding Normal-mode
// motion targets. Returning to Normal chooses the preceding complete grapheme.
func (d *Draft) InsertionView(v *View, on bool) {
	v.inserting = on
	if !on {
		d.finalizeReplacementLinks()
		d.state.inside = false
	}
	if !on && d.state.syntax > 0 {
		n := findNode(d.state.nodes, d.state.cursor.Node)
		if n != nil {
			marker := markDelimiter(n.Tag)
			if n.Tag == "hr" {
				marker = strings.Repeat("-", max(3, n.dashes))
			}
			if marker != "" {
				d.state.syntax = min(d.state.syntax, len(marker))
			}
		}
	}
	if !on && d.state.syntax == 0 {
		n := findNode(d.state.nodes, d.state.cursor.Node)
		if n != nil && n.Kind == Text && d.state.cursor.Offset == len(n.Text) && n.Text != "" && !(strings.HasSuffix(n.Text, "\n") || d.InCode() && strings.HasSuffix(n.Text, "\r")) {
			d.state.cursor.Offset = graphemeBefore(n.Text, d.state.cursor.Offset)
		}
	}
	d.UpdateView(v)
	if !on {
		if _, found := findPoint(v.Layout(80, true).points, v.target); !found {
			n := findNode(d.state.nodes, d.state.cursor.Node)
			if n != nil && (n.Kind == Opaque || n.Tag == "br") {
				d.state.cursor.Offset = 0
				d.UpdateView(v)
			}
		}
	}
}

func (d *Draft) LineBoundary(end bool) Anchor {
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return d.state.cursor
	}
	b := findNode(d.state.nodes, loc.Block)
	if b == nil {
		return d.state.cursor
	}
	if b.Tag == "pre" {
		source := codeSource(b)
		start, stop := codeLineBounds(source, d.codeOffset(b, d.state.cursor))
		if end {
			return d.codeAnchor(b, stop, d.state.cursor.Node)
		}
		return d.codeAnchor(b, start, d.state.cursor.Node)
	}

	var leaves []Node
	var collect func([]Node)
	collect = func(ns []Node) {
		for _, n := range ns {
			if n.Kind == Text || n.Kind == Opaque || n.Tag == "br" || n.Tag == "a" && len(n.Children) == 0 {
				leaves = append(leaves, n)
			} else {
				collect(n.Children)
			}
		}
	}
	collect(b.Children)
	if len(leaves) == 0 {
		return Anchor{b.ID, 0}
	}
	n := leaves[0]
	off := 0
	if end {
		n = leaves[len(leaves)-1]
		off = 1
		if n.Kind == Text {
			off = len(n.Text)
		}
	}
	return Anchor{n.ID, off}
}

// InCode reports the cursor's logical block, including an empty Code wrapper.
func (d *Draft) InCode() bool {
	loc, err := d.live().ResolveAnchor(d.state.cursor)
	if err != nil {
		return false
	}
	b := findNode(d.state.nodes, loc.Block)
	return b != nil && b.Tag == "pre" && b.Kind == Element
}
