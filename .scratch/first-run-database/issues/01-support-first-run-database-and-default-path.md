# 01 — Support a fresh database and a default database path

**Phase:** M3 — first-run usability

**What to build:** Let a new installation start without an existing database or an explicit `--db` argument. Create a usable, empty Vikunja-compatible database, either by initializing it or by installing a validated starter template.

**Category:** enhancement

**Status:** needs-triage

**Recorded:** 2026-10-23

## Current behavior

`--db` already defaults to the working-directory-relative path `internal/db/vikunja.db`. This ignored local file is absent from a fresh clone. The Store deliberately opens existing files only (`mode=rw`), and production startup rejects missing or incompatible schemas. The tutorial creates disposable synthetic data; it is not a persistent production starter.

## Requirements

- [ ] Start successfully on a fresh installation without supplying `--db` or preparing a database manually.
- [ ] Provide a documented default database path; decide whether to retain the repository-relative default or use a platform user-data directory that also works outside the repository. Define how existing default databases are handled so an upgrade does not silently switch users to an empty database.
- [ ] When the intended new database is absent, create it through a compatible initialization flow or a validated, distributed starter template. An empty SQLite file alone is insufficient.
- [ ] Preserve Vikunja 2.7.0 schema and migration-history compatibility. Do not invent or modify the upstream schema, silently migrate existing databases, or relax existing compatibility checks.
- [ ] Define the minimum local User, Project, and View setup needed for a usable start. Empty means no personal or demo Tasks; any required starter records must be deliberate and documented. Coordinate with [Project creation](../../project-creation/issues/01-specify-project-creation-workflow.md).
- [ ] Preserve explicit `--db` selection. Specify whether an explicitly supplied missing path is initialized or rejected, and how initialization intent is communicated.
- [ ] Preserve existing databases byte-for-byte unless the user performs ordinary application edits. Reject invalid or incompatible existing files without replacing them.
- [ ] Handle missing parent directories, permissions, partial initialization, and concurrent first starts without overwriting data or leaving an apparently valid partial database. Report the chosen path and actionable failures.
- [ ] If using a template, define its generation, distribution, version validation, and license/provenance requirements. Exclude real user data, machine-specific paths, and demo content.
- [ ] Add focused tests for fresh startup, repeated startup/data preservation, default and explicit paths, incompatible files, and initialization failures. Verify a newly created database with official Vikunja 2.7.0.
- [ ] Update README installation/default-path guidance and CLI help to match the implemented behavior; keep tutorial data disposable and separate.

## Decisions before implementation

Choose initialization versus a starter template, the default-path policy, required starter records, and explicit-path behavior. This ticket records the requested capability; implementation has not been requested yet. Initial GitHub publication is complete; this remains open M3 work.

## Comments

- 2026-10-23 — Jan confirms that `go run ./cmd/vimkunja` starts successfully on Windows from the consolidated `vimkunja` checkout with the existing ignored database. This verifies the current relative default path, not initialization of a missing database. Fresh-installation requirements remain open.
