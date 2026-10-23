# Foundation/MVP — completed

**Completed:** 2026-07-17

Vimkunja's SQLite foundation was verified automatically and manually. It provides Projects and List View, a complete Task editor, workflow status through Vikunja Buckets, consistent Done/Default Bucket semantics, Vim-style pane navigation, Ex commands, help, error log, and Leader support.

## Verification

- `go test ./...` passes
- Foundation smoke test passed
- Workflow status checked against the original application

## Archive contents

- `fundament-mvp/SPEC_FUNDAMENT.md` – completed foundation spec
- `fundament-mvp/tickets/` – completed foundation tickets
- `fundament-mvp/workflow-status/` – completed workflow-status spec and tickets

Open work belongs only in `.scratch/`; these archive files are not an active roadmap.
