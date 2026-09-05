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

**Description**:
A Task's rich-text content stored by Vikunja as HTML.
_Avoid_: Notes, body

**Description block**:
A logical unit inside a Description, such as a paragraph, heading, list item, code block, quote, or table.
_Avoid_: Source line, HTML line

**Checkbox list**:
A checklist embedded in a Description, represented by Vikunja/Tiptap `data-type="taskList"` HTML; it is not a list of Vimkunja Task records.
_Avoid_: Task list

**Task Done state**:
The top-level completion state of a Task record, distinct from every Checkbox-list entry in its Description.
_Avoid_: Checkbox state

**Logical paragraph jump**:
Description navigation between paragraph blocks using `j` and `k`.
_Avoid_: Line jump

**Visual row jump**:
Description navigation between terminal rows produced by paragraph wrapping using `gj` and `gk`.
_Avoid_: Paragraph jump

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
