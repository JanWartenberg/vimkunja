# 01 — Reconcile open subtasks before GitHub publication

**What to deliver:** A reliable ticket/status index: distinguish genuinely open work from completed subtasks and historical acceptance gaps before finishing GitHub setup/publication.

**Phase:** M3 — release preparation

**Priority:** Medium

**Status:** completed

**Order:** Jan explicitly authorizes this cleanup in parallel with the URL-Copy story on 2026-10-20. URL-Copy remains owned by that parallel work; this ticket supplies the remaining checks before publication.

## Scope and next action

- [x] Review open and implemented Scratch tickets and their unchecked subtasks against implementation, regression evidence and recorded manual acceptance.
- [x] Reconcile the tutorial ticket's historical Rich-text follow-ups with the completed AT-01–10 acceptance. Keep the general tutorial's remaining manual chapters distinct; do not claim unperformed tests passed.
- [x] Check adjacent implemented tickets (Checkbox progress, editor motions, cursor/layout and configuration) for real remaining work versus stale bookkeeping.
- [x] Close/archive only tickets whose scope is demonstrably complete; fix affected links and STATUS entries. Preserve historical evidence and fixture paths used by tests.
- [x] Keep genuine remaining tasks in focused tickets, identify publication blockers, and record what is deferred rather than requiring every M3 feature for release.
- [x] Hand the reconciled open-blocker list to the GitHub publication ticket.

## Boundary

This ticket authorizes status reconciliation and documentation cleanup when picked up. It does not silently implement newly discovered product work, mark unverified acceptance as passed, or delete user data.

## Comments

2026-10-19 — Jan chooses the sequence: (1) URL-Copy shortcut, (2) check remaining subtasks and clean up, (3) finish GitHub initialization/publication. This is a sequencing dependency, not a newly discovered software defect.


## Reconciliation result — 2026-10-20

Technical review, archival and the remaining-work handoff are complete. Jan
accepted all 20 remaining checks on 2026-10-20; no missing implementation
was found in the archived scopes. The reconciliation below records the initial
handoff; its manual items are now closed by the acceptance recorded at the end. Twelve completed tickets and two completed
audit reports are archived under `.archive/2026-10-20/`, with relative links
updated. The Native corpus, captures and supplemental harness remain at their
fixed `.scratch/description-rendering/` paths because tests consume them.

- Checkbox progress: live draft updates and empty-list deletion are implemented;
  the original deferred checkbox is closed. Plain imported checkbox-looking text
  remains literal under the accepted unified-editor contract.
- Comments (create/delete), shared Comment layout, Description cursor offset,
  Task deletion, footer/narrow/modal/Kanban layouts, multi-View compatibility and
  local TUI configuration have complete implementation/test evidence. Their
  technical tickets are archived; no new physical acceptance is fabricated.
- Configuration: document actual CLI > TOML > defaults precedence and existing
  Richtext settings. Environment overrides/template ideas remain outside the
  completed scope; no new product implementation was authorized by cleanup.
- Editor motions: retire obsolete read-only HTML acceptance, preserve original
  report and retain only the physical Comment `~` check.
- General tutorial: retain tasks 9–26 as conservatively unaccepted; Richtext
  tutorial acceptance is already complete.
- The recent literal-punctuation fix is implemented, regression tested and
  uncommitted; it needs a short physical display check, not a repeated full
  Richtext acceptance run. Existing layout feedback “sieht mal gut aus” is
  retained; small-terminal layout is not reopened as a new required test.

### Jan's remaining acceptance

Use [the interactive checklist](../acceptance.html). All controls begin NOT
ASSESSED; automated results are evidence, not preselected human PASS values.

1. General guided tutorial tasks 9–26, one result per task. No repeat of the
   accepted Richtext tutorial or the full old Tier-1 checklist.
2. `~` / `2~` in a Comment draft, including the visible result and saved Comment.
3. The current punctuation fix: ordinary hyphens, underscores and lone markers
   stay literal in Normal/Insert/Visual; no phantom cursor step.

### Genuine remaining work outside this cleanup

- URL-Copy usability is running in parallel and is not changed here.
- Publication ticket 02 still owns final release checks, vulnerability scan,
  actual Actions result, separate publication/history/privacy copy, secret/DB
  history scan, explicit-reference Dry Run and the later authorized public push.
  The current uncommitted fix and cleanup must be committed before a final
  publication base is selected. No publication action was executed here.
- Other needs-triage M3/M4 tickets remain future product work (Projects, Views,
  search/history, Help, Task export, Bucket management and preferences). They
  are neither incomplete Richtext subtasks nor blanket publication prerequisites.
- Live CalDAV remains an explicitly unverified boundary, not a hidden PASS or
  newly required implementation task.

### Validation

`go test ./... ./.scratch/description-rendering/verification` passes in the current
working tree (base main e978e76 plus the literal-punctuation fix). This covers
Description/Checkbox structure/save regressions, Comment motions/layout, config,
Task deletion, footer/modal/narrow/Kanban behavior and both complete tutorial
journeys. Link/LF/index checks cover the documentation moves. No application
source was changed for the cleanup and no user database was opened.

## Closure and manual acceptance — 2026-10-20

Jan explicitly reports all 20 checklist checks PASS and requests closure.
AT-01–18 cover general tutorial tasks 9–26; AT-19 covers Comment `~` / `2~`;
AT-20 covers the literal-punctuation fix. Together with previously recorded
acceptance, no manual subtask remains in this cleanup. The tutorial and editor
motion tickets are completed and archived alongside this ticket.

The HTML summary export failed for Jan, so this explicit chat report is the
acceptance evidence; no exported report or individual comments are invented.
The archived HTML remains an unfilled reproduction of the checklist. Its export
problem does not invalidate the reported product acceptance.

URL-Copy and publication remain separate work. No public push is authorized
by this closure.

Final pre-commit validation: full Go tests including artifact verification,
`go vet ./...`, both entrypoint builds, local Markdown link checks and Git
whitespace checks pass. The accepted punctuation fix is committed as `393a2b0`.
