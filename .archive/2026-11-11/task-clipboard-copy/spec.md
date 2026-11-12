# Copy a Task to the system clipboard

## Accepted behavior — 2026-11-11

`:copy` exports exactly one selected Task from List, Overview, Kanban, or Task Detail. Use the current visible state, including unsaved Task Detail changes, without saving, changing selection, moving the cursor, or changing local history. When the selected Task has an open Detail draft, that draft takes precedence even with focus in the Tasks Pane. A different Task selection uses that selected Task, never another Task's draft. New unsaved Task drafts can also be copied without an identifier.

Output uses English field names and this stable order: Title (without a field label), Task identifier, Project, Status, Done, Priority, Due, Labels, then a blank line and the Description. Include all available user-facing metadata; omit empty fields, unset Priority and an empty Description. Done is `Yes` or `No`; missing workflow status is omitted. Prefer the public identifier, otherwise derive the Project identifier and Task index or `#<index>`. Never expose internal IDs. Due dates use the current field value. Labels preserve display order.

Project and workflow metadata must follow the current draft, including an unsaved Project move. Copy captures the draft before asynchronous work. Loading Detail metadata requires retry rather than silently exporting incomplete values. List/Overview/Kanban may read missing labels and workflow through existing Store methods; use the selected Kanban View when applicable. All operations are read-only.

Description projection reuses the Native document model and Opaque safe previews, without parsing through a separate HTML reader. Preserve paragraphs, nested lists, `[ ]`/`[x]` Checkboxes, Code source whitespace, and literal plain text. Links become `label (URL)` unless the label already is the URL; omit purely visual formatting. Blocked content is represented by `[Unreadable description content]`, matching its unavailable display rather than leaking raw HTML.

No selected Task, unsupported context, unsupported arguments, metadata-read errors and clipboard errors use existing error feedback. Validation/read failures do not write the clipboard. Report success only after a successful write. Copy never modifies persisted data or drafts.

## Scope boundary

Comments are excluded from plain `:copy`. A separate command or `:copy -c` is a possible future extension, not part of this implementation. Existing `Space+y/p` text clipboard actions and `Space+e` Link URL copy remain intact. No new shortcut, database schema, dependency, or Store write is needed.

## Verification seams

Test real Ex dispatch through the model to the injected system-clipboard writer, using Store fixtures for external reads. Test whole-Description projection through `description.Draft.PlainText`, including native drafts, lists, Code whitespace, protected content and literal source. Run the project test suite and both builds; two independent reviewers assess bugs/risks and consistency/validation.
