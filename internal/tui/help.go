package tui

// helpPages is the concise reference for implemented bindings. Deferred Views
// are explicitly unavailable. Keep docs/keybindings.md in sync when bindings change.
var helpPages = []helpPage{
	{title: "Navigation & Views", parent: -1, sections: []helpSection{
		{title: "NAVIGATION", entries: []helpEntry{
			{"h j k l / Arrow keys", "Move selection or cursor"},
			{"gg / G / Home / End", "First / last item"},
			{"Ctrl+D / Ctrl+U", "Half page down / up"},
			{"Ctrl+W h j k l", "Change pane focus (configurable Ctrl+W timeout)"},
			{"Ctrl+W Ctrl+W / w", "Cycle open panes"},
		}},
		{title: "PROJECTS PANE", entries: []helpEntry{
			{"O (Shift+O)", "Create a Project; defaults to no parent. Only in the Projects Pane."},
			{":projectnew / :projectnewchild", "Create a Top-level Project / preselect the marked Project as parent."},
			{"h / l", "Close branch or select parent / open branch or first child"},
			{"zo / zc / za", "Open / close / toggle selected branch"},
			{"zM / zR", "Close / open all branches"},
			{"Shift+J / Shift+K", "Move Project down / up among siblings"},
			{"r", "Reload hierarchy"},
			{"Creation dialog", "Normal: j/k or Tab selects Title/Parent; i edits. Title Enter/Esc returns to Normal. :w creates; :q asks before discarding changes; :q! discards directly. Picker: j/k moves, gg/G first/last, Ctrl+U/D half-page, / searches, Enter chooses, Esc returns."},
			{":projectdelete", "Check the marked Project first, showing eligibility or the blocker. y deletes only after a successful check; r rechecks; n/Esc cancels. Tasks (including trash), Subprojects, archive status, Time Entries and background files block deletion."},
			{"New Project Titles", "1–250 characters after trimming; new duplicates rejected globally, ignoring case, including archived Projects. Existing duplicates stay readable."},
		}},
		{title: "OVERVIEW NAVIGATION", entries: []helpEntry{
			{"-", "Open Overview (never toggles)"},
			{"Space+a", "Toggle Overview / remembered Project context"},
			{"s d/p/t", "Overview sort: Due, Priority ↓, Title"},
			{"f o/a/v/t/i", "Task filter: open, all, overdue, due today, in progress (not Kanban)"},
			{":overview / :list", "Open Overview / restore Project List"},
		}},
		{title: "PROJECT VIEWS", entries: []helpEntry{
			{"Space+v, then l / k", "Select List / Kanban View"},
			{":kanban", "Open the last used Board for this Project in the session, or its first supported Board; create and open a standard Board if none exists"},
			{"Space+v b / :kanban switch", "Choose another Board; Esc keeps the current context"},
			{"Space+v m / :kanban manage", "Standalone Board manager: Enter open, n new, e edit, d preview deletion, r reload"},
			{":kanban new", "Name-first creation, then configuration of standard To-Do, Doing and Done Buckets"},
		}},
		{title: "KANBAN BOARD", entries: []helpEntry{
			{"h / l", "Select previous / next Bucket"},
			{"j / k", "Select previous / next Task in Bucket"},
			{"Shift+H / Shift+L", "Move selected Task to adjacent Bucket"},
			{"m", "Choose any target Bucket"},
			{"Enter", "Open selected Task detail"},
			{"Board management :h / :help", "Explain each command for the current page; q/Esc/:q returns to the unchanged draft."},
			{"Board configuration", "No Task Detail is open. j/k select fields; i edits; Enter/Esc leave Insert. :w saves configuration, :wq saves/closes; :q asks before discard, :q! discards; :e! explicitly reloads."},
			{"Bucket configuration", "o/O inserts a Bucket after/before the selected one; J/K reorders it; Space d toggles Default, Space x toggles Done, dd / Space D / :bucketdelete previews deletion. Set Limit to 0 for unlimited. Save staged changes with :w."},
			{"Board/Bucket deletion", "y confirms; n/Esc cancels; j/k scrolls the preview. Tasks remain. The last Bucket is protected; reassign special roles before deletion. Configuration writes clear Task undo history."},
		}},
		{title: "DEFERRED PROJECT VIEWS", entries: []helpEntry{
			{"Space+v, then t / g", "Select Table / Gantt View (deferred)"},
			{":table / :gantt", "Select deferred Table / Gantt View"},
		}},
	}},
	{title: "Tasks & Detail", parent: -1, sections: []helpSection{
		{title: "TASKS", entries: []helpEntry{
			{"i / Shift+I", "Edit List/Overview title at end / beginning"},
			{"Inline title", "Arrows/Home/End, Backspace/Delete; Enter save; Esc discard"},
			{"Enter", "Open task detail editor"},
			{"o", "Create a Task in the selected Project"},
			{":copy", "Copy Task as plain text, including unsaved edits; no Comments"},
			{"x", "Toggle task done"},
			{"u / Ctrl+R", "Undo / redo latest persisted Task action"},
			{"dd, then y", "Delete selected task; n/Esc cancel"},
			{"r", "Reload current view"},
		}},
		{title: "DETAIL EDITOR", entries: []helpEntry{
			{"i", "Insert mode / open focused picker"},
			{"Esc", "Leave Insert mode; close a clean editor in Normal mode"},
			{":new", "New task draft"},
			{":w / :write", "Save and keep editor open"},
			{":wq / :writequit", "Save and close editor"},
			{":wqa / :writequitall", "Save task and quit Vimkunja"},
			{":e! / :edit!", "Discard changes and reload task"},
			{":q", "Close clean editor"},
			{":q!", "Discard editor changes and close"},
			{"Detail pickers", "j/k or arrows; Ctrl+D/U page; gg/G jump; Enter selects"},
			{"Project picker", "Selection only; typing does not filter"},
			{"Due date", "YYYY-MM-DD mask; j/k select days, weeks, or months"},
			{"Due date / Priority", "x or dd clears the draft value"},
			{"Done field", "x toggles the draft value"},
			{"Insert Ctrl+V", "Paste system clipboard at cursor"},
			{"Labels field", "h/l select; x stages removal; i or Space+l opens picker"},
		}},
		{title: "DETAIL FIELD SHORTCUTS", entries: []helpEntry{
			{"gt", "Edit Title"},
			{"gd", "Edit Due Date"},
			{"gp", "Edit Priority"},
			{"gl", "Edit Labels"},
			{"gc", "Open Task Comments"},
			{"gn", "Focus Description"},
		}},
		{title: "TASK COMMENTS", entries: []helpEntry{
			{"Enter / gc", "Open Comments overlay"},
			{"j/k, arrows, gg/G", "Select Comment / jump oldest or newest"},
			{"Ctrl+D/U", "Move by page"},
			{"a / i / o", "Append a Comment; Normal/Insert editing and :w/:wq save; :q protects drafts"},
			{"Composer", "Esc leaves Insert; :w saves; :wq saves and closes; :q! discards. :h opens help without losing the draft."},
			{":qa / :qa! / :wqa", "Quit app / discard and quit / save Task + Comment and quit"},
			{"Esc", "Leave Insert mode or close Comments"},
		}},
	}},
	{title: "Shared text editing", parent: -1, sections: []helpSection{
		{title: "TEXT SELECTION AND SYSTEM CLIPBOARD", entries: []helpEntry{
			{"", "Title, Description, Comment drafts share these Normal/Visual bindings"},
			{"v / V", "Select characters / lines (Richtext: logical blocks)"},
			{"h/j/k/l, words, o", "Extend selection / swap active end; Esc cancels"},
			{"Space y", "Copy selection or current line/Native paragraph"},
			{"Space p", "Paste at cursor or replace selection; stay Normal"},
			{"Space e", "Copy destination URL inside a Richtext Link"},
			{"Empty/plain text", "Same editor automatically; no Richtext switch needed"},
		}},
		{title: "TITLE AND COMMENT TEXT MOTIONS (NORMAL MODE)", entries: []helpEntry{
			{"Description", "See Rich text / Editing basics for document motions and formatting."},
			{"h l / Arrow keys", "Move cursor left / right"},
			{"j k / Arrow keys", "Move cursor down / up"},
			{"0 / ^ / $", "Line start / first non-blank / line end"},
			{"w / b / e", "Next / previous word / word end"},
			{"W / B / E", "Next / previous WORD / WORD end"},
			{"count+motion", "Repeat, e.g. 2w, 3j, 2$"},
			{"dw / db / de", "Delete by word motion; counts compose"},
			{"diw / daw", "Delete inner / around word"},
			{"ciw / caw / C", "Change word object / to line end"},
			{"r{char} / count+x", "Replace / delete characters"},
			{"~ / count+~", "Toggle character case and advance"},
			{"a / Shift+I / Shift+A", "Insert after cursor / at line start / end"},
			{"o / Shift+O", "Insert blank line below / above"},
			{"Ctrl+D/U, PgUp/PgDn", "Scroll by page"},
			{"gg / G / Home / End", "Start / end"},
		}},
	}},
	{title: "Rich text", parent: -1, children: []int{6, 7, 8, 9, 10}, sections: []helpSection{}},
	{title: "Search & filters", parent: -1, sections: []helpSection{
		{title: "LOCAL SEARCH", entries: []helpEntry{
			{"/ / ?", "Search forward / backward within the focused Projects, Tasks or Task Detail Pane in Normal mode. Live preview; Enter accepts, Esc restores context, cursor, scroll and folds."},
			{"n / N; 3n", "Next / opposite-direction occurrence, with counts and wrap feedback. One shared session pattern; current Pane and filters determine scope."},
			{"Space h / :nohlsearch", "Hide match highlights; n/N shows them again. Current occurrence is emphasized; the footer shows its index and total."},
			{"Search input", "Empty Enter repeats the last pattern; Up/Down recalls a separate 100-entry session search history. Insert/Visual modes, dialogs, Comments, Help and Information are excluded."},
			{"Vim patterns", `Default-magic subset: . * ^ $ [a-z], \< \>, \( \), \|, \+ \? \=, \{n,m}, \d \D \s \S \w \W \t \n. Lowercase ignores case; uppercase requires exact case; \c/\C overrides. Unsupported patterns report an error.`},
		}},
		{title: "GLOBAL SEARCH", entries: []helpEntry{
			{"Space+f / :search", "Open Global Search in Insert mode"},
			{"", "Search Insert: Esc Normal; arrows/Ctrl+N/P select; Enter open"},
			{"", "Search Normal: j/k, gg/G, Ctrl+D/U; i/a edit; Esc/:q close; :qa quit; Ctrl+O return"},
		}},
		{title: "FILTERS", entries: []helpEntry{
			{"f o/a/v/t/i", "List / Overview: open, all, overdue, due today, or in progress; not Kanban."},
		}},
	}},
	{title: "Commands & quitting", parent: -1, sections: []helpSection{
		{title: "COMMANDS", entries: []helpEntry{
			{":h / :help", "Open the help overview. :q closes help and restores previous focus and drafts."},
			{"Ex prompt", "Up/Down history; Tab completes command names"},
			{"Space+o", "Toggle projects pane"},
			{":errors", "Open session error log"},
			{":labels", "Manage global Labels"},
			{":q / :quit", "Close current pane or quit"},
			{":qa / :quitall", "Quit Vimkunja"},
			{":qa! / :quitall!", "Quit and discard unsaved editor changes"},
		}},
	}},
	{title: "Editing basics", parent: 3, sections: []helpSection{
		{title: "DESCRIPTION — ONE EDITOR FOR PLAIN TEXT AND HTML", entries: []helpEntry{
			{"gn, Enter", "Activate native document navigation; Esc leaves"},
			{"j/k, gj/gk", "Logical blocks / visible rows; Code uses source lines"},
			{"{/}, ö/ä", "Paragraph units; whole Code/quote is one unit"},
			{"h/l, 0/^/_/$", "Character / logical-line navigation"},
			{"w/b/e/ge, W/B/E/gE", "Semantic word / WORD motions; stop at blocks"},
			{"gg/G, count+motion", "Document ends / counted motion"},
			{"Ctrl+D/U", "About one visible Pane height"},
			{"zh/zl", "Scroll current Code/table horizontally"},
			{"Ctrl+E/Y", "Scroll viewport without moving document cursor"},
			{"i/a/I/A, o/O", "Insert text / open paragraph or Code line"},
		}},
		{title: "EDITING BASICS", entries: []helpEntry{
			{"ciw/caw, diw/daw", "Change / delete inner or around word; Esc cancels prefix"},
			{"Enter / Shift+Enter", "Split paragraph / insert hard break; Code Enter is newline"},
			{"dd / x", "Delete block/Code line / toggle Checkbox or delete grapheme"},
			{"v/V, viw/vaw", "Select text / blocks / semantic word; o swaps end, Esc cancels"},
		}},
	}},
	{title: "Formatting", parent: 3, sections: []helpSection{
		{title: "FORMATTING", entries: []helpEntry{
			{"Visual v/V Space b/i/u/s", "Bold / italic / underline / strike; 0 clears marks"},
			{"Normal Space 1/2/3/0", "H1/H2/H3/paragraph; palette remains visible after timeout; press Space again to act"},
			{"Inline syntax", "Typed **bold**, *italic*, __underline__, ~~strike~~, `code`; Paste literal"},
		}},
	}},
	{title: "Links", parent: 3, sections: []helpSection{
		{title: "LINKS", entries: []helpEntry{
			{"Space l / Space e", "Link insert/edit / copy destination; empty URL unlinks"},
			{"Space+l (outside a Link)", "Link selected text; without selection, enter URL then Link text. Empty text uses the URL; Esc cancels."},
			{"Space+e (Space+Y alias)", "Copy the destination URL inside a Link."},
		}},
	}},
	{title: "Lists & Code", parent: 3, sections: []helpSection{
		{title: "LISTS & CODE", entries: []helpEntry{
			{":selectblock", "Select complete block; Code V stays within source lines"},
			{":retab / :retaball", "Tabs to spaces: current Code/lines or all Native text"},
			{"Normal Space u/n/x/q", "Toggle unordered/numbered/Checkbox list or local quote"},
			{"Visual Space x", "Checkbox per touched paragraph; Code V Space+x outdents"},
			{"Block V Space k/l", "Outdent/indent selected sibling subtrees"},
			{"Block V Space q/w", "Remove/add one shared quote layer; metadata loss confirms"},
			{"Space+c", "Normal: create Code and enter Insert. Typed three/four backticks plus Enter also create Code; pasted markers stay literal."},
			{"ü / Ü; ]Space / [Space", "Insert / Normal: create a paragraph after / before Code."},
			{"Tab / Ctrl+V Tab", "Code Insert: insert spaces / a literal Tab. Configurable Ctrl+Q Tab alias; Tabs follow tabstop."},
			{"V, Space+x / c", "Code line selection: outdent / indent source lines."},
			{"dd (Code Normal)", "Delete a source line; the last line leaves empty Code, another dd removes the block."},
		}},
	}},
	{title: "Undo & recovery", parent: 3, sections: []helpSection{
		{title: "UNDO & RECOVERY", entries: []helpEntry{
			{"y/p, u/Ctrl+R", "Safe visible local register / paste; local Description draft Undo/Redo, separate from persisted Task actions."},
			{"y/Enter, n/Esc", "Confirm / cancel preserved-content changes"},
			{":descriptionrecover", "Confirm recovery of safe text from Blocked HTML"},
			{":descriptionclear", "Confirm empty draft replacing Blocked HTML"},
			{":descriptionreload", "Explicitly reload current Task; discard complete draft"},
		}},
	}},
}

// helpLines also exposes the complete reference to existing content checks.
var helpLines = func() []string {
	lines := []string{"Vimkunja is a task manager in your terminal facilitating Vim Motions."}
	for _, page := range helpPages {
		for _, section := range page.sections {
			lines = append(lines, section.title)
			for _, entry := range section.entries {
				lines = append(lines, entry.keys+"  "+entry.text)
			}
		}
	}
	return lines
}()
