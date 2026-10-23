# AGENTS.md – Vimkunja

**For:** AI coding agents
**Project:** Vimkunja – terminal UI for Vikunja (open-source todo/Kanban system)

---

## 🎯 Vision

Build a TUI for a todo list with Projects, Tasks, and a Kanban Board.
The aim is to bring the existing functionality of the open-source Vikunja project to a terminal UI.

**Audience:** Vim users who prefer keyboard-driven command-line workflows.

---

## ⌨️ UX guidelines

- **Keyboard only:** Behavior should feel familiar to Vim users.
- **Vim navigation:** `hjkl`, `gg/G`, `Ctrl+D/U`, `Ctrl+W` (panes).
- **Commands:** `:q`, `:qa`, `:h` (as in Vim).
- **Leader key:** `Space` + action.
- **Consistent selection/clipboard:** Normal/Visual text editors use the shared selection and clipboard logic (`text_normal.go`, `text_selection.go`, `text_clipboard.go`): `v/V`, `Space+y/p`; rich-text Links copy their URL with `Space+e` (`Space+Y` remains an alias). Insert-only prompts retain their input rules.
- **Consistent input cursor:** Every new text input uses `insertCursorText` from `internal/tui/input_cursor.go`; do not add custom block/underscore cursors or field-specific cursor shapes for Insert. Shared rendering positions the blinking terminal bar cursor after layout. Normal/explicit overwrite cursors remain separate.

---

## 🚦 Agent workflow

### ⚠️ IMPORTANT: Change files only when explicitly instructed!

**Preserve Vikunja compatibility:** Maintain compatibility with the original application's API.
**Do not change the database schema:** The existing SQLite schema must remain unchanged.
**Windows host:** `/workspace` is hosted on Windows. Normalize line endings to LF before commits; do not commit line-ending-only changes as functional changes.
**Markdown hygiene:** Avoid unnecessary Markdown files. `README.md` contains the introduction, controls, and vision; `STATUS.md` is the short ticket index. Open work belongs only in `.scratch/<feature>/`; move completed specs and tickets to `.archive/YYYY-MM-DD/`. Delete temporary notes once their content has been incorporated.
**Default mode:** Propose changes; **do not implement them**.
**Implement only when requested:** For example, “implement this,” “make the change,” or “do it.”

### Response format

- **Compact:** About one screen; avoid long scrolling responses.
- **Structure:** Bullets and short paragraphs.
- **Focus:** What the user needs to know, not every detail.

### Steps

1. Read **README.md** and **STATUS.md**.
2. Read the requested or current `.scratch` ticket and its spec.
3. Implement according to Ousterhout's principles.
4. Test Store logic.
5. Update the ticket and STATUS index if priorities change.

---

## Go, builds, and local data

- Language: write documentation, source comments, docstrings, and user-facing messages in English. Preserve technical Unicode fixtures, keybindings, and authoritative third-party notices.
- Stack: Go 1.25, Bubble Tea v2, and SQLite (`modernc.org/sqlite`). Entrypoints: `cmd/vimkunja` and `cmd/vimkunja-tutorial`; core logic is in `internal/vimkunja`, UI in `internal/tui`.
- After Go changes, run at least the affected tests; before a commit, also run `go test ./...` and `go build ./cmd/vimkunja ./cmd/vimkunja-tutorial`.
- Before deleting or renaming Go files, exported types, or functions, check callers with `rg`. Extract shared types first or update every caller.
- `internal/db/vikunja.db` contains local user data: do not change or delete it. Change `go.mod` and `go.sum` only when dependency changes are explicitly requested.

---

## 🧭 Design philosophy (Ousterhout)

- **Deep modules:** Simple interfaces, complex implementations (e.g. `store`).
- **Information hiding:** Hide HTTP/JSON/state details.
- **Pull complexity downward:** Keep the UI simple; put logic in the Store.
- **Define errors out of existence:** Validate on write, not on read.
- **General purpose:** Reusable components; avoid copy/paste.
- **Consistency:** Uniform keybindings, data structures, and error handling.

---

## 📚 References

- **README.md** – introduction, controls, and vision.
- **STATUS.md** – current ticket index.
- `.scratch/` – open specs and tickets.
- `.archive/` – completed specs, tickets, and historical reports.

---

## Agent skills

### Issue tracker

Issues are tracked as local Markdown under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default local triage roles are configured. See `docs/agents/triage-labels.md`.

### Domain docs

This is a single-context repository. See `docs/agents/domain.md`.

---

**Last updated:** 2026-07-12
