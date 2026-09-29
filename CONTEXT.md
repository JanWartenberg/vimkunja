# Vimkunja

Vimkunja is a keyboard-first terminal interface for Vikunja projects and tasks. This glossary fixes the user-facing terms used by its TUI and rich-text descriptions.

## TUI regions

**Pane**:
A focusable terminal subwindow that represents a distinct aspect of Vimkunja.
_Avoid_: View, window

**Projects Pane**:
The Pane used to browse and select Projects.
_Avoid_: Project Pane

**Tasks Pane**:
The Pane used to browse Tasks for the active Project and View.
_Avoid_: Task Pane

**Task Detail Pane**:
The lower Pane that presents and edits the selected Task's fields and Description.
_Avoid_: Detail Pane, TaskDetailedPane

**View**:
A representation and navigation model for Tasks within the Tasks Pane; it is not a focusable TUI region.
_Avoid_: Pane, window

**List View**, **Kanban View**, **Table View**, **Gantt View**, **Overview View**:
The initial named Views, each representing Tasks in a different form. Future Views may extend this set.

**Manual Kanban View**:
A Kanban View with Vikunja's `bucket_configuration_mode=manual`. Its persisted Buckets accept direct Task moves. _Avoid_: filter-configured Kanban View, which derives Buckets from filters and is outside the initial Board scope.

**Global Search**:
A cross-data query that returns typed matches from the locally browsable database. It is distinct from a Local Search and can activate a Task, Project, or Label-filtered Overview.

**Local Search**:
A future query constrained to the currently focused Pane or its content. _Avoid_: Global Search.

**Label-filtered Overview**:
An Overview scope limited to Tasks assigned one Label. Its Label constraint composes with the Overview status filter. _Avoid_: Label View.

## Rich-text descriptions

**Vikunja Original App (VOA)**:
The upstream Vikunja application and its native user interface, distinguished from Vimkunja.
_Avoid_: Vimkunja, Vikunja API

**Description**:
A Task's rich-text content stored by Vikunja as HTML.
_Avoid_: Notes, body

**Source HTML**:
The HTML representation of a Description read from and written to Vikunja's existing Task Description field; it is persistence data, not the user-facing terminal projection.
_Avoid_: TUI markup, editor syntax

**Description block**:
A logical unit inside a Description, such as a paragraph, heading, list item, code block, quote, or table.
_Avoid_: Source line, HTML line

**Native node**:
A safely parsed Description structure that Vimkunja understands and can edit through its terminal projection.
_Avoid_: Supported node, editable HTML

**Opaque subtree**:
The outermost safely parsed Description structure that Vimkunja cannot edit internally; it is one atomic unit whose original HTML is preserved during unrelated edits.
_Avoid_: Opaque node, unsupported text

**Blocked Description**:
A Description whose Source HTML is malformed, unsafe, or cannot be preserved reliably, so structured editing and saving remain disabled until explicit recovery.
_Avoid_: Opaque Description, read-only node

**Block quote**:
A Description block backed by an HTML `<blockquote>` element: an extended quotation containing editable child blocks. It is distinct from the inline HTML `<q>` element.
_Avoid_: Inline quote, quote marks

**Checkbox list**:
A checklist embedded in a Description, represented by Vikunja/Tiptap `data-type="taskList"` HTML or by a strict plain-text checkbox block during the transitional reader phase; it is not a list of Vimkunja Task records.
_Avoid_: Task list

**Checklist progress summary**:
An informational `Checklist progress: n/m` line attached to one nonempty Checkbox-list block. It counts that block's valid checkbox entries, including nested entries flattened into the block, and never changes the enclosing Task's Done state.
_Avoid_: Subtask status, Task Done state

**Task Done state**:
The top-level completion state of a Task record, distinct from every Checkbox-list entry in its Description.
_Avoid_: Checkbox state

**Logical paragraph jump**:
Description navigation between paragraph blocks using `j` and `k`.
_Avoid_: Line jump

**Visual row jump**:
Description navigation between terminal rows produced by paragraph wrapping using `gj` and `gk`.
_Avoid_: Paragraph jump

**Paragraph-motion unit**:
One navigable Description block traversed by `{` or `}`. Each heading, list or Checkbox-list entry, quote, rule, table, and complete Code block is one unit; a Code block's internal and empty lines do not split it.
_Avoid_: Source line, visual row

## Task history

**Persisted Task Action**:
A successful change to an existing Task's persisted state, treated as one user-level unit for reversal. It excludes Task Comments and unsaved editor drafts.
_Avoid_: keystroke, field change

**Undo Entry**:
A reversible record of one Persisted Task Action, identified with the affected Task and its before-and-after states.
_Avoid_: audit log, change log

**Undo History**:
The session-local sequence of recent Undo Entries available for reversal; it is distinct from the local editing history of a Description.
_Avoid_: status log, database history

**Redo History**:
The session-local sequence of Undo Entries that were reversed and can be reapplied before a new Task change branches the history.
_Avoid_: replay log

## Task comments

**Task Comment**:
A timestamped rich-text contribution associated with one persisted Task and one author; it is a separate record from the Task's Description.
_Avoid_: Ticket comment, Description note

**Comment thread**:
The chronological sequence of a Task's Comments, ordered from oldest to newest.
_Avoid_: Comment page, conversation page

**Comment draft**:
An unsaved new Task Comment that belongs to the current Task edit session until that session is saved or discarded.
_Avoid_: Temporary Description, pending Ticket comment
