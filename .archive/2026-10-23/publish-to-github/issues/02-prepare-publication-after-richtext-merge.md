# 02 — Prepare GitHub publication after the rich-text merge

**What to deliver:** Technically verify the merged state for `JanWartenberg/vimkunja`, prepare a publication copy without private email/database history, and make the exact push reviewable. Actual publication requires an explicit instruction.

**Category:** maintenance

**Priority:** High after the rich-text merge

**Status:** completed

**Completed:** 2026-10-23 — public `main` verified at `4358202d7b72c7c5e9766422dbaa55588e844180`; Windows and Ubuntu Actions pass. The active checkout is now `/workspace/vimkunja`. Earlier paths and pending-work statements below are historical; the final closure record supersedes them.

**Start condition met:** Rich text and task-based tutorials from `feature/tutorial-task-workflow` integrated into `main`.

**Order:** Before finishing publication, complete [URL-copy convenience](../../../../.archive/2026-10-20/description-link-copy-shortcut/issues/01-choose-accessible-url-copy-shortcut.md), then [subtask reconciliation](../../../../.archive/2026-10-20/release-ticket-cleanup/issues/01-reconcile-open-subtasks-before-publication.md). Other M3 features are not blanket prerequisites.

**Documentation prerequisite met:** [Ticket 01 — License and documentation](../../../../.archive/2026-10-07/publish-to-github/issues/01-publish-to-github.md) is completed and archived.

**Prerequisite met:** [Vikunja database compatibility: 2.7.0 only](../../../../.archive/2026-10-05/vikunja-db-compatibility/issues/01-verify-vikunja-2-3-to-2-7-compatibility.md)

## Start condition and order

**This ticket follows the agreed rich-text merge.** The original repository and existing worktrees retain their history; later cleanup happens only in the publication copy. The start condition requires the agreed worktree merge, not completion of every rich-text backlog ticket or all of Milestone 3.

1. Record the merged `main` commit; reconcile README/STATUS and the license inventory with changes from the merge. New functionality or dependencies require only the affected additions.
2. Check tests, both builds, vet, vulnerability scanning, and CI configuration on that state. Reuse existing CI; check a real Actions run once a suitable GitHub test environment is available. Report native Windows acceptance separately from cross-builds.
3. Create a separate publication copy from `main` and explicitly selected tags. Clean email/history only in this copy; do not rewrite the original repository, worktrees, or local backups.
4. Clean author/committer addresses and personal email occurrences in current and historical file contents. Target address: **`5298128+JanWartenberg@users.noreply.github.com`**. The last readiness check found personal metadata in five commits and two historical versions of the publication ticket; determine the current inventory again. Do not blanket-replace other authors or third-party copyrights.
5. Transfer selected tags to the cleaned history; check the complete reachable history and tag/commit contents for remaining private addresses, secrets, and databases including sidecars. Record results and commit mappings without repeating private addresses in publishable reports.
6. Check the GitHub target and remote configuration; perform a dry run with explicit references. Perform the actual public push only after an explicit instruction, then verify remote references.

## REQUIRED: Never publish backup references

**Publish only `main` and explicitly selected tags. `refs/original/refs/heads/main` and other backup references must NOT reach GitHub. NEVER use `git push --mirror`.**

- A clean working tree or `.gitignore` alone does not establish clean history; check every intended reference.
- The local backup reference may remain. Rewriting the original history or deleting its backups is unnecessary; email cleanup happens only in the publication copy.
- Check remote configuration for mirror mode and additional push refspecs. Do not publish feature branches automatically.
- Continue new work on the cleaned history after the transition. Do not merge old worktrees into the new history without control; document the transition.
- Never include local user databases or backups; schema metadata and explicitly verified synthetic fixtures are allowed.

Example dry run if `origin` points to the verified target and both historical tags have been selected for publication:

```bash
git push --dry-run origin main refs/tags/MVP refs/tags/Milestone1
```

After approval, use the same explicit references without `--dry-run` for the actual push.

## Acceptance

- [x] Ticket 01 complete and agreed rich-text merge included; verified starting commit `bd340c2` recorded. Subsequent documentation corrections and findings documented separately.
- [x] Current features, README/STATUS, and dependency license notices reconciled after the merge.
- [x] Local tests, both builds, vet, vulnerability scan, and workflow checks pass; remaining platform limitations documented.
- [x] Real GitHub Actions run succeeds for the intended publication state: run `39991127576` on `4358202`, Windows and Ubuntu.
- [x] Only `main` selected for publication; no tags, backup references, or local T3 checkpoint references. Original and existing worktrees preserved.
- [x] Personal address removed from author/committer metadata, commit/tag contents, and historical files; target address and future local Git identity of the publication repository verified.
- [x] Secret, privacy, and database-history checks of intended publication history without concrete findings; no real user data or database sidecars found. Scope and limits documented below.
- [x] HTTPS remote configured, mirror/push settings checked, and explicit transfer of `main` alone rehearsed against a local bare repository.
- [x] Remote target, mirror settings, and push refspecs verified; Jan confirmed the explicit HTTPS dry run before publication.
- [x] Actual publication performed by Jan; public repository and remote heads/tags verified: `main` only at `4358202`.
- [x] Transition to cleaned history documented; see continuation guidance and report below.

## Boundaries

Updating this ticket does not authorize a merge, history rewrite, remote creation, or publication. Automatic release binaries, a production starter template, and migrating local tickets to GitHub Issues are not prerequisites for the initial source publication. Specific macOS binary license conditions must be met separately before a later release of that kind.

## Existing evidence

Dated license, CI, secret, and reference checks remain in the [former overall ticket, now Ticket 01](../../../../.archive/2026-10-07/publish-to-github/issues/01-publish-to-github.md#historical-findings-from-the-former-overall-ticket). The last recheck of committed state `e77f00c` found no reachable database files or backup references in a fresh publication clone; secret and vulnerability scans had no findings. These results do not replace checks of the later merged and cleaned state.

## Comments

### 2026-10-07 — Start after the rich-text merge

Jan requests splitting the former publication ticket. Ticket 02 groups email/history cleanup, CI, publication copy, and push checks after the agreed rich-text worktree merge. Jan confirms the GitHub noreply address. Creating this ticket started no operational publication steps.

### 2026-10-07 — Ticket 01 completed

License and documentation for the current state are verified and Ticket 01 is archived. The only remaining start condition is the agreed rich-text merge; then reconcile its specific documentation/dependency changes. No operational publication has started.

### 2026-10-20 — Rich-text integration and inherited follow-up plan

The order recorded in the rich-text worktree on 2026-10-19 remains: URL-copy convenience, subtask reconciliation, then finish publication. AT-01–10 and the live rich-text round trip are accepted; separate general tutorial acceptance remains partly open. Database compatibility with Vikunja 2.7.0 only and the completed license review already established on `main` are not reset by older open checklists in the feature branch. The former overall Ticket 01 still present there must not be recreated as open during the merge; this Ticket 02 tracks remaining work. No public push or original history rewrite is authorized.

### 2026-10-20 — Subtask cleanup: remaining work handed over

Technical ticket reconciliation is done; twelve completed tickets and two audit reports are archived. The [cleanup report](../../../../.archive/2026-10-20/release-ticket-cleanup/issues/01-reconcile-open-subtasks-before-publication.md#reconciliation-result--2026-10-20) separates remaining manual acceptance (general tutorial from task 9, Comment `~`, latest punctuation fix) from already accepted rich text/layout. URL-copy work runs in parallel at Jan's request and remains owned there.

Publication work remains open: commit final working changes, select a starting commit, run release checks/CI, create the publication copy with metadata/history cleanup, scan secrets/databases, and perform an explicit push dry run. `e978e76` establishes the completed merge but does not identify the final publication state because subsequent local changes exist. Public push needs separate approval. This reconciliation does not blanket-check the release boxes above.

### 2026-10-20 — Subtask cleanup completed

Jan confirms all 20 remaining acceptance checks PASS: general tutorial 9–26, Comment `~` / `2~`, and literal punctuation. Cleanup, tutorial, and editor-motion tickets are completed and archived. The previous acceptance handover is resolved. The HTML summary export was broken; Jan's explicit chat response is the acceptance evidence. URL-copy and publication tasks above remain separate.

The accepted punctuation fix is committed as `393a2b0`. The related cleanup completion is versioned separately; a final publication state has not yet been selected.

### 2026-10-21 — URL-Copy integration

`feature/url-copy-key` (`c6d25a0`) is integrated into main: `Space+e` copies a
Link destination; `Space+Y` remains an alias. URL-Copy implementation and
Subtask-Cleanup are complete. Publication checks remain owned by this ticket;
this merge does not perform or authorize a public push.

### 2026-10-22 — Publication focus and email cleanup requested

Jan defers all backlog work not yet started and prioritizes publication. Automated removal of his private email from history is explicitly requested. Perform this in a separate publication copy with the confirmed GitHub noreply address; preserve original history, worktrees, and local backups. Initially `main` only, no tags without explicit selection, and no public push.

#### History cleanup result

- Publication copy: `/workspace/vimkunja-publication`, independent repository without object alternates, remotes, or backup references. Only reference: `refs/heads/main`; no tags selected.
- Source commit: `57c94e46645c6d8639f78e682f7fc365a5d19b03`; cleaned commit: `af4a50fbdce1362066ec1ae982b62ec309b9f80c`.
- Automated with `git-filter-repo` 2.47.0. One confirmed private identity replaced: author/committer fields in five commits and three historical blob versions at both former paths of publication Ticket 01. Commit messages and file contents were also checked and replaced on matches, including case variations and UTF-16. Other identities unchanged.
- All 190 commits and both merges preserved. Every commit compared with the original: parent mapping, messages, metadata, filenames, modes, and all historical contents match except for the requested email replacement and resulting object IDs. No other content changes.
- All 3,324 physically present Git objects and raw files under `.git` checked: no match for the confirmed private address. No database/sidecar filenames or SQLite signatures in reachable blobs. `git fsck --full` passed. This is not a full secret scan or general privacy audit.
- Future local identity of the copy: `Jan Wartenberg <5298128+JanWartenberg@users.noreply.github.com>`. No remote configured; `push.default=nothing` prevents an unintended default push.
- On the cleaned commit using the locally available Go 1.27.1 toolchain: `go test ./... ./.scratch/description-rendering/verification`, both builds, and `go vet ./...` passed. Toolchain version alone does not establish current security scanning.
- Original `main`, local backups, other worktrees, user databases, and the unversioned Task-copy spec preserved. Old history still contains the private address locally and must not be published.

Commit mapping: `/workspace/vimkunja-publication/.git/filter-repo/commit-map`.
Machine-readable results: `/workspace/vimkunja-publication/.git/filter-repo/privacy-verification.json`.
The verified starting commit does not yet include subsequent documentation changes. Record and check the final publication commit once documentation is complete. Other backlog work is deferred.

**Next release steps:** Record the updated documentation state as the basis for final release checks; check current vulnerability/secret scanning, CI configuration, and any platform limitations; verify GitHub target and explicit push dry run. A real Actions run needs a suitable GitHub test environment. Public push still requires an explicit instruction.

**Continuation:** Continue publication work in the cleaned copy. Use old branches/worktrees only as references; merging old history would reintroduce private metadata. Transfer later changes selectively. History-cleanup documentation is already transferred; the unversioned Task-copy spec was not included in the publication copy.

### 2026-10-22 — Independent project identification

Jan requests the README subtitle “An unofficial Vim-style terminal interface for Vikunja.” and disclaimer “Vimkunja is an independent project, not affiliated with or endorsed by Vikunja.” Both appear directly below the project name. This does not document Vikunja's consent to the name. Publication notes reflect the achieved cleanup state; final checks and public push remain open.

### 2026-10-22 — Local publication preparation verified

Jan requests all publication preparation already possible. Verified source starting point: `bd340c216e4f60942bd43341e74e0e76c4669666` (`main`, 191 commits). The following documentation corrections are versioned separately with local findings; nothing was publicly pushed.

- **Documentation and licenses:** 329 publishable files, including 111 Markdown files, checked with 10 local fragment references. Corrected an outdated corpus anchor in archived rich-text Ticket 02; no broken local file/fragment links afterward. All 40 external documentation links respond successfully. All 30 compiled third-party module versions for both entrypoints on Linux amd64, Windows amd64, and Darwin amd64/arm64 match `THIRD_PARTY_NOTICES.txt`; inventory license texts checked against module sources and notices, including Go and both APSL texts. No dependency change needed. README still documents Vikunja 2.7.0 only, sequential database access, and limitations of later binary releases.
- **Provenance recheck:** 916 historical Go blobs on `main` compared with backend Go sources of the local Vikunja upstream (`82ee780161c6ed250cdb1fc4d2952b72464cb724`). No identical sequence of eight nontrivial lines after existing normalization. This is not complete provenance proof and cannot detect short or rewritten copies.
- **Tests and CI:** Go 1.27.1 passes `go test ./... ./.scratch/description-rendering/verification`, both entrypoint builds, and `go vet ./...`. `go mod verify` also passes. Both CI fuzz targets pass with five seconds each. Actionlint successfully checks the existing Linux/Windows workflow; both referenced Action `v7` tags exist upstream. Windows amd64 cross-builds of both programs succeed. Both CLI help commands, missing database path without file creation, and tutorial menu/practice in Linux PTY including Ctrl+C and temporary database cleanup checked. No production user database opened.
- **Vulnerabilities:** The first run with local default Go 1.25.0 reports three reachable standard-library findings (`GO-2025-4010`, `GO-2026-4341`, `GO-2026-4601`). Rescan with available Go 1.27.1: no reachable findings. The scanner additionally lists one vulnerability in imported packages and 13 in required modules without a demonstrated call. Continue building with a current security-patched toolchain; the unpatched system installation was not changed.
- **Secrets and privacy:** Gitleaks 8.30.1 over `main` (188 commits with evaluated changes) and current publishable files without findings. All 3,350 physically present Git objects additionally exported/scanned: four findings, exclusively confirmed `modernc.org/token` checksums in historical `go.sum` blobs. No real secrets found. All object contents/current files checked for the previously cleaned private address including UTF-16: zero matches. No database/sidecar paths in reachable history or SQLite signatures in objects. The personal-path candidate in `help.go` is only the keyboard hint `/Home/End,`. Synthetic test/tutorial fixtures preserved. These technical scans cannot prove the absence of every possible kind of sensitive information.
- **References and push rehearsal:** `origin` points via HTTPS to `https://github.com/JanWartenberg/vimkunja.git`; no mirror mode or additional push refspecs, `push.default=nothing`. The app has since created local `refs/t3/orchestration-v2/checkpoints/...`; these stay local and are not deleted. Explicit dry run then push into a temporary local bare repository produces exactly `refs/heads/main` at `bd340c2`, no tags/other references. `git fsck --full` passes. Nothing published on GitHub.
- **GitHub blocker:** Anonymous repository API responds HTTP 404; this cannot distinguish a missing from a private target. GitHub `ls-remote` and actual HTTPS push dry run fail with `could not read Username for 'https://github.com': No such device or address`. This session has no authenticated GitHub access. Repository state, write permissions, and actual GitHub dry run are therefore unconfirmed. No repository created speculatively.
- **Remaining acceptance:** Real GitHub Actions run requires a suitable accessible GitHub test environment. Windows cross-build does not replace native terminal/clipboard acceptance; those limits and the open live CalDAV test remain documented. Automatic release binaries, macOS binary approval, and a production starter template remain outside initial source publication.

Local raw reports and push rehearsal: `/tmp/vimkunja-readiness-20261022/` (temporary, not published). The documented transition to cleaned history remains mandatory: continue here and do not merge old branches containing private metadata.

**Remaining work:** Jan sets up the GitHub repository and authenticated access; then verify target and actual dry run; publish after an explicit public-push instruction, and check remote references and Actions. This ticket remains `in-progress` until then.

### 2026-10-22 — Commit report; GitHub setup owned by Jan

Jan requests committing the four documentation changes and takes ownership of GitHub setup. Reminder: create `JanWartenberg/vimkunja` as an empty public repository without automatically generated README, license, or `.gitignore` (if not already present), and provide authenticated HTTPS push access. Then check the real dry run; public push still needs a separate instruction.

### 2026-10-23 — English publication documentation

Jan confirms repository creation and a successful HTTPS dry run for `main`. Jan requests translating all repository documentation, comments, and docstrings into English. Translate current and archived prose and remaining German UI messages; preserve keybindings, Unicode test inputs, paths, source links, and authoritative third-party license texts. This does not authorize a public push or another history rewrite.

Translation completed across 26 existing files: README, STATUS, agent instructions, archived/open tickets and specs, corpus notes, HTML prototypes/checklist, prototype sample data, and remaining German confirmation/fallback messages. Source comments and docstrings were already English; the repository language rule now makes this explicit. Multilingual Unicode fixtures, German-layout character bindings, historical filenames, and third-party copyright names remain intact. No license texts, dependency versions, database data, or Git history changed.

Validation: `go test ./... ./.scratch/description-rendering/verification`, both entrypoint builds, and `go vet ./...` pass with Go 1.27.1; focused Description tests also pass after the final error-text adjustment. Gofmt reports no changes. Non-string Go token streams are identical, and HTML JavaScript structure is identical apart from string/template text. Nine English DOM screens initialize successfully; checklist navigation and prototype editor mode controls pass. Native browser preview is unavailable because host AppArmor blocks its sandbox, so no visual-browser acceptance is claimed. All local links and ten fragment references across 111 Markdown files resolve; source URL targets are preserved except the deliberately translated heading anchor. The final language scan has no remaining German prose candidates after excluding documented technical fixtures and false positives. Gitleaks finds no secrets in the translation diff. Jan requests committing the translation and verification record. No push performed.

### 2026-10-23 — Windows configuration-test isolation

The first GitHub Actions run passes on Ubuntu but fails on Windows. Jan also reproduces `TestLoadCodeSettingsAndIndependentFallbacks` locally on Windows and requests the fix. That test set only `XDG_CONFIG_HOME`, while Windows `os.UserConfigDir` uses `AppData`, so it never read the temporary TOML fixture and returned built-in Code defaults. Reuse the existing `setConfigHome(t, dir)` helper, which isolates both variables. Production configuration loading is unchanged.

The existing test reproduces the exact reported defaults failure using a temporary Go overlay that substitutes the Windows config-directory lookup on Linux; it passes after the helper change. This is a deterministic path-selection regression check, not a native Windows execution claim. Full local tests including corpus verification, both builds, and vet pass with Go 1.27.1. Gofmt and whitespace checks pass.

The separate Windows runner timeout remains open: the ten-minute limit applies to the entire Store test package, and the test visible in the timeout stack had run only two seconds. The same test passes locally on Linux in 0.32 seconds; Jan's native Windows Store package passes in 32.331 seconds. The runner stack is inside SQLite `FlushFileBuffers`, but that snapshot alone does not establish why the full package exceeded ten minutes. Do not weaken production SQLite durability or claim this configuration-test fix resolves runner performance. Native Windows/GHA revalidation is still required. Jan requests committing this fix and its verification record; no push performed.

### 2026-10-23 — Publication verified and ticket closed

Jan asks to close publication, reconcile the tickets and STATUS, and commit the documentation. Read-only GitHub API and Git checks establish:

- [JanWartenberg/vimkunja](https://github.com/JanWartenberg/vimkunja) is public, with `main` as its default branch. Remote `main` and local HEAD before this documentation update both identify `4358202d7b72c7c5e9766422dbaa55588e844180`.
- `git ls-remote --heads --tags origin` returns only `refs/heads/main` at that commit. No tags or backup branches were published. The HTTPS remote has its normal fetch refspec, no mirror setting, and no additional push refspec. `main` tracks `origin/main`; the deliberate `push.default=nothing` setting still requires explicit push arguments.
- [Actions run 39991127576](https://github.com/JanWartenberg/vimkunja/actions/runs/39991127576) completed successfully on that commit. Both `check (windows-latest)` and `check (ubuntu-latest)` pass tests, both program builds, and vet; bounded fuzz checks run on Linux. This supplies the missing native Windows CI revalidation. The earlier Store timeout did not recur in this run; its underlying cause is not established.
- Jan consolidated the active checkout as `/workspace/vimkunja` and moved retired folders outside the workspace. Git lists only this main checkout. The production database and sidecars are ignored; previously verified backups were retained during consolidation. No database data is changed by this closure.
- Jan confirms the native Windows startup `go run ./cmd/vimkunja` works in the isolated checkout with the existing database and without `--db`. This does not verify first-run initialization when the database is absent.

All source-publication acceptance points are met. The first-run database/default-path ticket remains open M3 work. Broader native-platform acceptance, live CalDAV, automatic binaries, macOS binary license approval, and migration of tickets to GitHub Issues remain outside this closure. Historical reports retain their original evidence and limitations. This closure makes no additional public push.
