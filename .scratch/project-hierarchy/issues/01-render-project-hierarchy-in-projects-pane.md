# 01 — Render Project hierarchy in the Projects Pane

**What to build:** The Projects Pane represents Vikunja's Project parent/child hierarchy through clear keyboard-navigable indentation.

**Blocked by:** Needs confirmation of the available schema fields and specification of collapse, ordering, and selection semantics.

**Category:** enhancement

**Status:** needs-triage

- [ ] Read parent relationships from the existing Vikunja schema without changing it.
- [ ] Define root order, child order, indentation, and handling of missing or cyclic parents.
- [ ] Decide whether branches can collapse and how that interacts with `j/k`, `gg/G`, and filtering.
- [ ] Preserve a stable Project selection when the hierarchy refreshes.

## Further context

Archived source: `SPEC_FUNDAMENT.md`, “Project View should reflect the hierarchy of projects (via indentation)”.
