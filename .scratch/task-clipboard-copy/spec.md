# Copy a Task to the system clipboard

Let users share one selected Task quickly by copying a readable plain-text snapshot to the system clipboard. The snapshot includes the Description and Task metadata such as Title and Priority, without changing the Task or its saved draft.

The first version applies only when exactly one Task is selected. The initial entry point is the Ex command `:copy`; a keyboard shortcut can be considered after checking the existing keybinding contract. Clipboard write failures must be reported through the existing session error feedback.

The exact metadata fields, ordering, formatting, and behavior when a Task Detail draft has unsaved edits remain to be specified in the issue before implementation.
