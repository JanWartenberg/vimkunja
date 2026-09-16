# Session-local undo for persisted Task actions

Vimkunja treats each successful, non-empty save of an existing Task as one reversible action, with `u`/`Ctrl-R` operating on a combined session-local history of up to 20 entries. Task Comments, Task creation/deletion, unsaved drafts, and combined `:wqa` Task-plus-Comment saves remain outside that history; this preserves clear domain boundaries and avoids SQLite schema or status-log complexity. Restores are atomic and abort on state conflicts rather than overwriting external changes.

## Status

Accepted

## Consequences

The history disappears on restart, and a combined Task-plus-Comment save cannot be partially undone. These trade-offs keep the first implementation safe and simple; the existing Comment persistence remains fully successful and atomic.
