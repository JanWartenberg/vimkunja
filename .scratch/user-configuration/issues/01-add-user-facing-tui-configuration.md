# 01 — Add user-facing TUI configuration

**What to build:** Users can configure supported Vimkunja behavior, starting with timing preferences already exposed programmatically.

**Implementation prerequisites:** Needs a configuration source, precedence, and compatibility specification.

**Category:** enhancement

**Priority:** Low

**Status:** needs-triage

- [ ] Decide configuration file format and location, environment/CLI precedence, and error behavior.
- [ ] Define which current programmatic settings become user-facing, including leader and Ctrl-W timeouts.
- [ ] Decide how future preferences, such as cursor-shape disabling, are represented.
- [ ] Keep defaults stable when no user configuration exists.

## Further context

Archived source: `SPEC_FUNDAMENT.md`, user-facing configuration deferred; current `tui.Config` is programmatic only.
