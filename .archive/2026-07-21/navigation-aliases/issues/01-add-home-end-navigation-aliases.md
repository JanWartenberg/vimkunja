# 01 — Add Home and End navigation aliases

**What to build:** A Vimkunja user can use Home as an alias for `gg` and End as an alias for `G` in scrollable informational Panes such as Help and Errors, so common terminal navigation keys work alongside Vim motions.

**Blocked by:** None — deliberately deferred until after the Fundament/MVP.

**Category:** enhancement

**Status:** resolved

- [x] Home moves the visible position to the beginning wherever `gg` is supported.
- [x] End moves the visible position to the end wherever `G` is supported.
- [x] Existing Vim motions and selection behavior remain unchanged.
- [x] Help documents the aliases where appropriate.

## Further context

Apply primarily to Help and other Panes whose content can exceed one screen; evaluate Task and Project list semantics during specification.
