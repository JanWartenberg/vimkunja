# 01 — Prepare license and documentation for GitHub

**What to deliver:** Prepare license and user documentation for the current Vimkunja state so code and documentation can subsequently be published on GitHub.

**Category:** maintenance

**Priority:** High

**Status:** completed

**Successor:** [02 — Prepare publication after the rich-text merge](../../../../.scratch/publish-to-github/issues/02-prepare-publication-after-richtext-merge.md)

**Prerequisite met:** [Vikunja database compatibility: 2.7.0 only](../../../2026-10-05/vikunja-db-compatibility/issues/01-verify-vikunja-2-3-to-2-7-compatibility.md)

## Required scope — 2026-10-07

This existing ticket now covers only **license and documentation**. It can be completed before the rich-text merge. History cleanup, CI acceptance, publication copy, and push verification belong to Ticket 02 and start only after the agreed rich-text merge.

- Keep MIT for our own code; check `LICENSE`, README license section, and `THIRD_PARTY_NOTICES.txt` for consistency, attribution, and completeness for existing dependencies.
- Preserve technical provenance findings and limitations clearly. Describe our MIT code, third-party license conditions, and outstanding requirements of later binary releases.
- Check README introduction, build/start commands, tutorial, configuration, and controls against current `main`. Do not present planned rich-text features as implemented.
- Consistently document Vikunja 2.7.0, sequential operation, database prerequisites, known workflow differences, and platform limitations; mark outdated statements as historical findings.
- Check local file links, fragment anchors, and external sources; identify local artifacts that are not included.
- Limit changes to license/documentation corrections. No dependency/schema changes, CI changes, Git identity/history changes, or publication.

## Already decided / preliminary state

- Jan selected MIT and target `JanWartenberg/vimkunja`. Code and documentation have highest priority; include tickets, prototypes, and safe history as transparently as possible.
- README already describes startup with a database initialized by Vikunja and the synthetic tutorial. A blank database is not a production starting point. An additional versioned starter template remains an optional product decision; implementation is not an acceptance condition of this documentation ticket.
- Native Windows/CalDAV acceptance and specific license conditions of potential macOS binaries remain explicitly open. Document those boundaries; do not build or approve a binary release.
- **Preliminary state 2026-10-07: no blocking questions for Ticket 01.** License decision, repository target, and current feature/database contract are sufficient. Keep the existing language mix without a requested complete translation; documenting the existing startup path is sufficient for now.
- The preliminary state was reported; Jan then requested implementation of Ticket 01. Acceptance below is complete.

## Acceptance for Ticket 01

- [x] MIT file, README license section, and third-party notices finally checked against current state; concrete corrections implemented and results recorded.
- [x] README describes a reproducible build, safe tutorial introduction, and production startup with an existing 2.7.0 database; configuration and controls match implementation.
- [x] Compatibility, platform, workflow, and additional binary-license limits documented without unsupported approvals.
- [x] Documentation links/examples checked; historical findings separated from current claims.
- [x] Ticket 02 owns all operational publication steps; Ticket 01 can close independently of the rich-text merge. Reconcile affected documentation/license attribution again before publication after later feature/dependency changes.

## Completion and results — 2026-10-07

**Ticket 01 is complete for the current `main` state.** Verified codebase: `e77f00c69fba7f8cb69a56eb8b64732c896c1e17`; documentation corrections belong to this completed ticket. Operational publication remains in Ticket 02 after the rich-text merge.

### Implemented

- README now explicitly describes read-only HTML Description projection, distinguishing it from editable plain text and planned native rich-text editing.
- Configuration precedence matches `LoadUserConfig`: CLI, TOML, defaults. Environment overrides are not implemented. CLI configuration example includes an explicit database path.
- Added startup commands for built programs on Linux and Windows PowerShell; relative paths assume the repository directory. Distinguished Go language minimum from maintained build toolchain; explained VOA and clarified normal tutorial cleanup.
- License section separates our MIT code from third-party conditions, links Vikunja's license, and identifies pending macOS binary acceptance including APSL 1.1/2.0. Existing MIT file is consistent and needs no change.
- Corrected notices inventory header to actual verified target architectures and new review date; clarified APSL 1.1/2.0 attribution. Full third-party license texts and copyright notices preserved.

### Validation

- Recomputed dependency closure of both entrypoints for Linux amd64, Windows amd64, and Darwin amd64/arm64 with `CGO_ENABLED=0`: **30 third-party modules**, all versions and root license texts included in `THIRD_PARTY_NOTICES.txt`. **128 additional license comment blocks** from compiled dependency sources also fully present. No dependency change.
- Checked license texts/provenance evidence against local versioned modules and linked MIT/APSL sources. This technical inventory is not complete provenance proof or blanket binary approval; earlier provenance findings and limitations are preserved below.
- Before archiving: **102 publishable Markdown files**, all detected local file links and Markdown fragments without broken targets; **70 distinct external Markdown/module-archive sources** HTTP 200, GitHub line anchors within each raw file. Rechecked archive links afterward.
- Fresh isolated clone of verified code, Go 1.27.1: exact documented build `go build -o bin/ ./cmd/vimkunja ./cmd/vimkunja-tutorial` succeeds. Both help calls succeed; documented timeout flags accepted. Missing database path provides tutorial guidance and creates no database.
- Used README TOML example in a separate temporary configuration directory; started tutorial with CLI override in Linux PTY and checked visible first lesson. `Ctrl+C` exits cleanly and removes the temporary demo database. No full manual tutorial run or native Windows test claimed.
- Existing targeted configuration/HTML write-protection tests (`TestLoadUserConfig*`, `TestHTMLDescriptionRemainsReadOnlyDuringReaderPhase`) pass. Changes affect documentation/tracking only; no new implementation tests needed.

### Remaining limits and handover

- Fresh production startup still uses a database initialized by Vikunja 2.7.0; an additional starter template is an optional later product decision. No user data or real databases opened through the application.
- Native Windows/CalDAV acceptance and license/packaging conditions of later macOS binary releases remain open and identified in README. They do not prevent closing this documentation scope.
- After the rich-text merge, Ticket 02 reconciles only affected feature/dependency changes against README, STATUS, and license inventory before checking the final state.
- No CI, code, schema, Git identity, or history changes; no commit, merge, remote creation, or push with this implementation.

## Historical findings from the former overall ticket

The following dated reports/comments remain as evidence. They document earlier states including resolved findings and CI/privacy work previously tracked here. **They do not expand Ticket 01's current scope; operational follow-up now belongs to Ticket 02.**

## Findings from 2026-10-04

### Database history

- Old commit `35b34220934d21c3bbd8e26bfe537098f28da966` (`MVP Vimkunja`, 2026-07-08) contains `internal/db/vikunja.db` (569344 bytes), `internal/db/vikunja.db-shm` (32768 bytes), and `internal/db/vikunja.db-wal` (0 bytes).
- Old subsequent commit `42f6029` removes those files but not their history.
- That old history is reachable through `refs/original/refs/heads/main`.
- Complete histories of `main`, `MVP`, and `Milestone1` contain no database files. Current MVP root commit is `23cd558`.
- WAL files can contain data; SHM files must not be published either. The currently empty historical WAL file is not included as a sample.

### Secret scan

- Gitleaks 8.30.1: current files including untracked/ignored files; Git history with all refs/reflogs; additionally all locally stored blob, commit, and tag objects including unreachable older states.
- History run: 144 commits, no findings. Additional object run included 1009 blobs, 155 commits, and 2 tag objects.
- Current files: one false positive for the `localStorage` key in `.scratch/editor-motion-polish/acceptance_test_V1.html:101`.
- Object run: four false positives for historical `go.sum` checksums of `modernc.org/token`.
- No real secrets found. Automated detection is not a guarantee; rescan and manually review personal contents before publication.
- Redacted JSON reports temporarily outside the repository at `/tmp/vimkunja-secret-audit-6nnrkfsc/`; these are not permanently available ticket evidence.

## Preliminary check on 2026-10-04: fresh clone and documentation

**Verified state:** committed `main` (`761774b`), fresh `--no-local --single-branch` clone outside the workspace. Ongoing uncommitted changes are excluded. Temporary files at `/tmp/vimkunja-publication-check-jqyy8_9u/`; not permanently available.

### Successfully checked

- Clone contains `main`, its remote-tracking references, and tags `MVP`/`Milestone1`; no `refs/original/` or reachable historical database files.
- Go 1.25.0, Linux amd64: `go test ./...`, both entrypoint builds, and `go vet ./...` pass. Tests used an empty separate user configuration directory.
- Both entrypoints cross-compiled for Windows amd64 with `CGO_ENABLED=0`. This does not establish Windows terminal, keyboard, or clipboard functionality.
- Both `--help` calls succeed. Tutorial started in Linux PTY: visible first lesson with synthetic Tasks; clean `Ctrl+C` exit. No complete run of every lesson.
- Clone working tree unchanged after tests, builds, and tutorial. Real user database not opened.
- Local license files exist for third-party modules used by Linux binaries. This is only an inventory, not a completed license-terms or code-provenance review. Our project license is still missing at this point.

### Clarify before publication

- **README introduction:** `cd /workspace/vimkunja` is a local development path; clone/install instructions and Go prerequisites missing. Normal startup needs a separately supplied Vikunja database. Tutorial is the working entry without user data.
- **Missing database path:** Starting with `--db` pointing to a nonexistent file creates an empty SQLite file. A TTY error follows in the noninteractive check; no helpful missing-Vikunja-database error is established. This contradicts README's requirement that the path exist. Reproduced only with a temporary path; no correction made.
- **Compatibility:** No evidenced list of supported Vikunja releases found. Store tests/tutorial use custom schemas; research references an upstream commit but does not replace versioned integration testing with the original application. No such round trip performed here. Document the exactly-one-local-User restriction for certain writes.
- **Four broken relative Markdown file links in tracked archive:** `.archive/2026-07-17/fundament-mvp/SPEC_FUNDAMENT.md` links to `.archive/TUI_IDEA.md`; `.archive/2026-07-27/all-tasks-overview/issues/01-add-cross-project-all-tasks-view.md` to `../../../prototype/overview_tui/`; `.archive/2026-08-02/global-search-filter/issues/01-specify-global-search-and-filter.md` to `../../local-search/issues/01-search-within-focused-pane.md` and `02-add-advanced-global-search-query-and-ranking.md`. Targets missing relative to each document. External links/fragment anchors not yet fully checked.
- **Local paths:** README, rich-text spec, corpus documentation, and rich-text ticket contain `/workspace` references. Replace with portable links or explicitly identify historical local artifacts for publication.
- Personal contents/commit metadata and provenance of included components remain manual checks; no blanket approval from this technical check.

## Preliminary findings implemented — 2026-10-04

- **MIT selected and added:** `LICENSE` contains MIT with Copyright 2026 Jan Wartenberg; README links the license. Provenance of included components and third-party conditions remain separate checks.
- **README:** Portable clone/build instructions with preliminary `<repository-url>` placeholder, Go 1.25+, tutorial without user data, SQLite operation, and current platform/version limits documented. No unsupported Vikunja version approval.
- **Links:** Four archive links corrected. `.archive/TUI_IDEA.md` explicitly exempted from archive ignore rules as a required historical design source; Gitleaks finds no secrets there. All local Markdown file links in current publishable files resolve; external links/fragment anchors not fully checked.
- **Local paths:** README no longer uses a workspace address. Rich-text docs explicitly identify historical temporary HTML artifacts as files not included; former external upstream-checkout path replaced with portable wording. Rich-text product decisions preserved.
- **Missing database:** Store opens only existing regular files; SQLite `mode=rw` prevents accidental creation even after file validation. CLI points to `--db` and tutorial. Regressions cover missing files, directories, and filenames with spaces, `#`, and `%`.
- **CI prepared:** `.github/workflows/ci.yml` runs tests, both builds, and vet on Linux/Windows for pushes to `main`, pull requests, and manual dispatch. Workflow checked locally with actionlint v1.7.12. A real GitHub Actions run remains pending before publication.
- **Validation:** Current workspace passes `go test ./...`, both Linux builds, `go vet ./...`, and both Windows amd64 cross-builds. Real Windows terminal tests remain open. No user data opened, schema/dependency changes, commit, or push.

### Blank database or starter template: checked, product decision open

A blank SQLite file has no Vikunja schema and cannot provide a usable fresh start. Store remains able to open an existing empty file for test/bootstrap flows; the production application does not initialize it. Full schema validation with a dedicated error is not yet implemented at this historical point.

For fresh production startup, the selected Vikunja version should initialize schema/migration history; at least the User/Project/View data needed for the workflow must then exist. A template would need to come from exactly that verified version, contain only synthetic data, and be evidenced by a round trip through the original application. Reduced tutorial database is not a production Vikunja template.

**Still to decide:** Initially document Vikunja-initialized startup, or additionally provide a versioned synthetic template. No production template or independent Vimkunja schema creation introduced without that contract.

## Further publication review — 2026-10-05

### Target, module path, and directly corrected findings

- Jan confirms `https://github.com/JanWartenberg/vimkunja` as public target, analogous to Tagger. `go.mod`, all previous Go imports including tests, and README clone URL updated. Dependency versions and `go.sum` unchanged.
- Public GitHub API call for the target currently returns HTTP 404; this does not prove whether the repository is missing or private. No repository/external remote created or GitHub push performed.
- `/bin/` ignored for README local builds. Global database/WAL/SHM/SQLite ignore rules additionally protect against accidental staging of temporary compatibility databases outside `internal/db`; existing local databases unchanged. Future intentionally published synthetic fixtures require an explicit exception.

### License review

- Actual dependency closure of both entrypoints on Linux, Windows, and macOS amd64 totals **30 third-party modules**. Module root licenses are MIT/BSD. Deeper review of compiled dependency files additionally finds MIT/BSD-2/BSD-4 texts and **APSL 2.0 notices in generated Darwin headers from `modernc.org/libc`**; Musl/Netdb/Unicode notices also apply. Root-LICENSE-only review would miss those extra conditions. Original SQLite is [public domain](https://www.sqlite.org/copyright.html); `modernc` bindings have their own BSD terms.
- [THIRD_PARTY_NOTICES.txt](../../../../THIRD_PARTY_NOTICES.txt) contains versioned module attribution, full root license texts, additional license comment blocks from compiled dependency files (including Darwin arm64), supplementary Musl/Netdb/Unicode notices, APSL 1.1/2.0 texts with source URLs, and Go license. Include it and `LICENSE` with later binary distributions; README documents this. Unused logos and components present only in dependency test data are not treated as distributed components.
- [Vikunja 2.7.0 is AGPL-3.0](https://raw.githubusercontent.com/go-vikunja/vikunja/v2.7.0/LICENSE). Vimkunja's module graph includes no Vikunja code. No third-party inclusion/copyright notices found in our Go files. **561 historical Go blobs** of intended references compared with backend Go sources of local upstream checkout (`82ee780161c6ed250cdb1fc4d2952b72464cb724`): no identical sequence of eight nontrivial code lines after removing whitespace/comment lines.
- **Result for our code and planned source publication:** No concrete license conflict found; retain MIT. **APSL review remains open before a macOS binary release:** [APSL 2.0](https://opensource.apple.com/apsl/) section 2.3 requires a prominent notice in executable code and documentation identifying the source-code location. The notices file alone does not establish that executable-release condition. No macOS binary release approved here. Technical comparison does not prove complete independent provenance, especially for rewritten/short copies. Future actual upstream code inclusion requires renewed provenance/license review; a database interface alone is not evidence of code copying.

### Links and personal contents

- **100 publishable Markdown files in final run:** Local file links and six Markdown fragments checked; no broken links after correction. Untracked ignored local artifacts not assumed to be published.
- **34 external documentation links including license/security sources added during audit:** HTTP 200, including verification of GitHub line anchors against raw files. External targets can change later.
- Search of publishable text for personal home paths/email: no additional candidates outside third-party notices and commit metadata findings recorded here. No concrete inappropriate personal passage identified; subjective content approval remains Jan's.
- **Published commit metadata would contain `Jan Wartenberg <5298128+JanWartenberg@users.noreply.github.com>` and local agent identities.** At this historical stage the note warns that a personal email would be public even without README inclusion; its address was subsequently replaced during the separate cleanup, so the displayed noreply address is the sanitized form. No history rewrite or Git identity change performed during this check. Confirm intended visibility or request separate cleanup before actual push.

### References and renewed secret checks

- Fresh `--no-local --single-branch` clone of committed `main` (`b6aa06061cc0df1a4c78643c8ab06fb3fac14210`), then dry run and actual push **only to a temporary local bare repository** with explicit references. Target contains exactly `refs/heads/main`, `refs/tags/MVP`, and `refs/tags/Milestone1`. No `refs/original/` or reachable database/WAL/SHM/SQLite/SQL dump files. Uncommitted preparation excluded from this history rehearsal.
- Workspace currently has no remotes, mirror mode, or extra remote push refspec. Recheck target configuration after later creating `origin`.
- Gitleaks 8.30.1 repeated over all refs/reflogs (**157 commits**), current files, and **all local Git objects** including unreachable older states (**1139 blobs, 168 commits, 2 tags**). No real secrets found. One `localStorage` key and four old `modernc.org/token` checksums remain confirmed false positives. Raw reports temporary; these figures/results are durable ticket evidence.

### Toolchain and platforms

- With Go 1.27.1, additionally checked both CLI help calls, missing database path with helpful error/no file creation, and tutorial in Linux PTY with visible output/clean `Ctrl+C` exit. Separate temporary user configuration directory; no production database opened through the application.
- Go 1.25.0 and Go 1.27.1: `go test ./...`, both entrypoint builds, and `go vet ./...` pass. `go mod verify` passes; CI rechecked after changes with actionlint 1.7.12.
- Both entrypoints cross-compiled with `CGO_ENABLED=0` for Windows amd64 and macOS arm64. This is not native terminal/keyboard/clipboard/OS acceptance. Native Windows tests and real GitHub Actions run remain open.
- **New concrete security finding:** govulncheck 1.8.0 with local Go 1.25.0 identifies [GO-2026-4341](https://pkg.go.dev/vuln/GO-2026-4341) as reachable `net/url.ParseQuery` through SQLite driver. Production DSN has only fixed query parameter `mode=rw`; reachability does not prove practical exploitation through Vimkunja. Nevertheless, the unpatched toolchain is unsuitable for release.
- **Fixed for CI/build guidance:** CI uses current stable Go rather than fixed unpatched Go 1.25.0 from `go.mod`. README recommends a current security-patched toolchain. Language/module minimum and dependency versions not raised; local system installation unchanged.
- **Security recheck:** govulncheck 1.8.0 built/run with Go 1.27.1 succeeds without findings (Go vulnerability database as of 2026-10-01). The first recheck with the scanner built under Go 1.26 could not load new Go 1.27 syntax and was invalid; repeated successfully after rebuilding. No dependency upgrades needed.

### Publication-unit review — 2026-10-06

- Ignore rules now cover WAL/SHM/rollback journals for `.db`, `.sqlite`, and `.sqlite3`; previously only `.db` WAL/SHM protected.
- Preserved Darwin headers name APSL 1.1 and 2.0. Notices now include both license texts/source locations; later macOS binary acceptance remains open.
- Publication unit verified separately from uncommitted database/rich-text work. Tests, both builds, vet, and workflow lint pass on an isolated staged-code copy with Go 1.27.1. No push/release requested.

## Comments

### 2026-10-04 — Decision with Jan

Publish as transparently as possible, prioritizing code/documentation. Keep backup reference local and exclude it using explicit push references. No further history rewrite required while publishing verified database-free history. Record this decision explicitly in the publication ticket.

### 2026-10-04 — MIT and direct implementation

Jan chooses MIT, requests direct implementation of remaining fixable findings, and wants to investigate a fresh start with blank database/template. Preparation above implemented; production starter template remains undecided.

### 2026-10-05 — Compatibility is a publication prerequisite

Jan requests a separate ticket for database changes from Vikunja 2.3.0 to 2.7.0 and makes GitHub publication dependent on completion. Blocker link and acceptance condition recorded above as mandatory.

### 2026-10-05 — Only compatibility target: 2.7.0

Jan limits verification/support to exactly Vikunja 2.7.0. Former multi-version/upgrade matrix removed; evidenced 2.7.0 round trip and rejection of untested database revisions remain publication prerequisites. No compatibility approval or push performed.

### 2026-10-05 — Compatibility prerequisite met

Linked 2.7.0 ticket completed/archived. Production schema validation, Store fixes, regressions, and real server round trips on fresh database/copy of synthetic Windows baseline pass. Tests, vet, and both builds pass; [report](../../../2026-10-05/vikunja-db-compatibility/report.md) documents contract/limits. Database blocker resolved; other publication items remain open. No commit/push requested.

### 2026-10-07 — Publication split into two tickets

Jan separates work: license/documentation in Ticket 01 now; email/history cleanup, CI, and push checks in Ticket 02 after rich-text worktree merge. Existing findings preserved. Actual public push needs an explicit instruction. This ticket change starts neither history cleanup nor further license/documentation implementation.

### 2026-10-07 — License/documentation scope completed

Jan requests implementation. Corrections/checks above meet all Ticket 01 acceptance points. Ticket completed/archived under project workflow; Ticket 02 remains open pending agreed rich-text merge.
