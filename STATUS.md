# Vimkunja – Status

**Stand:** 2026-10-09 | **Branch:** main | **MVP:** abgeschlossen und archiviert | **Aktueller Meilenstein:** Meilenstein 3

## Aktueller Stand

MVP, **Meilenstein 1** und **Meilenstein 2** sind abgeschlossen: tägliche Aufgabenverwaltung ist ohne Wechsel zur Original-App praktikabel. Die weitere Arbeit konzentriert sich auf die offenen Spezifikationen und Erweiterungen des Meilensteins 3.

- Lokale Vikunja-SQLite-Datenbank: Projects, List View, All-Tasks-Overview, Task-Detail, Task-Editor und Task-Löschung
- Hierarchischer Projects Pane mit Einrückung, session-lokaler Faltung, defensiven Recovery-Einträgen und persistenter Geschwister-Sortierung
- Workflow-Status und Done-Toggle im Detail sowie Done-/Default-Bucket-Synchronisierung
- Einheitliche Due-Date-, Priority- und Identifier-Darstellung in Liste, Overview und Detail; stabiles Metadaten-Layout bei schmalen Pane-Breiten
- [Kompakte Task-Details](.archive/2026-08-14/task-detail-compact-layout/issues/01-preserve-description-access-in-short-terminals.md#regression-correction--2026-10-09): Description-Sichtbarkeit und Scroll-Feedback bei geringer Höhe repariert; Layout von Jan unter Windows akzeptiert.
- List-, Overview- und Kanban-Views mit Vim-Navigation, Pane-Fokus, Ex-Commands, Help, Fehlerlog und Leader-Key; Kanban-Karten- und Task-Detail-Auswahl bleiben synchron
- Separates englisches Keystroke-Tutorial mit bei jedem Start frisch erzeugter, temporärer SQLite-Datenbank
- `go test ./...` grün

## Aktuelle Tickets

Die All-Tasks-Übersicht, Task-Löschung, Cursor-Darstellung im Editor, Home/End-Aliasse, [Home/End im Insert Mode](.archive/2026-09-11/insert-mode-home-end/issues/01-support-home-end-in-insert-mode.md), Done-Toggle im Detail, Task-Metadaten-Darstellung, Task Labels, Task Comments inklusive Einzel-Löschung und [session-lokales Undo/Redo](.archive/2026-09-16/undo-task-actions/issues/01-undo-recent-task-actions.md) sind abgeschlossen. Die Standard-Kanban-Provisionierung, die einzelne und mehrfache manuelle Kanban Views, die Project-Hierarchie und die Globale Suche sind implementiert und archiviert. Die folgenden offenen Tickets sind nach Meilenstein eingeordnet; `needs-triage` in einem Ticket bedeutet weiterhin, dass dessen konkrete Spezifikation noch fehlt. Die Task-Status-Filter-Shortcuts sind implementiert und archiviert.

### Data integrity audit — 2026-10-04

- [Audit report and ticket index](.scratch/data-integrity-audit/report.md#ticket-index) — eleven findings tracked individually. F01/F02 are repaired, manually accepted and closed. F03 rejects missing target Projects transactionally; F04 preserves full due timestamps and detects same-day undo time conflicts. F05 preserves Done and completion metadata when workflow is unchanged. F06 initializes completed Project moves in destination Done Buckets. F07 resolves zero/NULL defaults to the first valid Bucket and preserves assignments without a Done Bucket. F08 preserves exact completion timestamps whenever Done does not transition, including Undo/Redo. F09 is repaired (unsafe recurring completion rejected). F10 is repaired (polymorphic deletion cleanup). F11 is repaired (stable creation UID). Ordinary SQLite/TUI regressions pass; F03–F11 are archived. All eleven audit findings are repaired and archived. Full race-enabled tests, build and vet pass; no manual acceptance of F03–F11 is claimed.

### Meilenstein 2 — abgeschlossen

Die read-only HTML-Description-Projektion, das Scroll-Feedback, die [weiteren Keyboard-Shortcuts](.archive/2026-08-08/more-key-shortcuts/issues/01-keyboard-shortcuts.md), die [kompakte Task-Detail-Ansicht](.archive/2026-08-14/task-detail-compact-layout/issues/01-preserve-description-access-in-short-terminals.md), die [Ex-Command-History samt Completion](.archive/2026-08-12/command-line-enhancements/issues/01-add-project-goto-command-history-and-completion.md), die [inline Task-Titelbearbeitung in List und Overview](.archive/2026-08-18/inline-task-editing/issues/01-edit-task-titles-inline-in-list-and-overview.md), die [feldlokalen Detail-Picker-Overlays](.archive/2026-09-03/detail-picker-overlay/issues/01-render-detail-pickers-as-field-local-overlays.md), der Due-Date-Picker mit relativen Helfern, kanonische UTC-Zeitstempel, der einzeilige Status-Footer, schmale Modal-Layouts und die [Benutzerkonfiguration](.scratch/user-configuration/issues/01-add-user-facing-tui-configuration.md) sind implementiert; Kanban-Title-Editing bleibt bewusst ausgenommen, der vollständige native Rich-Text-Editor bleibt Meilenstein 3.

### Meilenstein 3 — aktuell

12. [Checkbox-Listen-Fortschritt anzeigen](.scratch/checkbox-list-progress-summary/issues/01-show-checkbox-list-progress-summary.md) – read-only Summary für HTML und strikte Plain-Text-Checkbox-Blöcke implementiert; native Bearbeitung bleibt Teil der Rich-Text-Arbeit.
13. [Erweiterte Vim-Editor-Motions](.scratch/editor-motion-polish/issues/01-add-advanced-vim-editor-motions.md) – Tier-1-Motions, Counts, Word-Operatoren, Textobjekte und `~` für Title, Plain-Text-Description und Comment Composer implementiert und manuell akzeptiert; Browser-Paste normalisiert Zeilenenden, `:e!` verwirft den Task-weiten Draft ohne den Task zu schließen.
14. [Project-Löschen semantisch spezifizieren](.scratch/project-deletion/issues/01-specify-project-deletion-semantics.md)
14a. [Project-Erstellung spezifizieren](.scratch/project-creation/issues/01-specify-project-creation-workflow.md) – Entry Point, Hierarchie, initiale View und Vikunja-kompatible Persistenz vor der Implementierung festlegen.
15. [Table View rendern](.scratch/table-view/issues/01-render-table-project-view.md)
16. [Gantt View rendern](.scratch/gantt-view/issues/01-render-gantt-project-view.md)
17. [Unabhängige Linktext-/URL-Bearbeitung evaluieren](.scratch/link-label-and-url-editor/issues/01-evaluate-independent-link-label-url-editing.md)
18. [Manuelle Kanban-Buckets verwalten](.scratch/kanban-bucket-management/issues/01-manage-manual-kanban-buckets.md)
19. [Lokale Suche im fokussierten Pane](.scratch/local-search/issues/01-search-within-focused-pane.md) – `/` sowie `n`/`N` sind vereinbart, die Pane-Semantik bleibt offen.
20. [Erweiterte globale Suchabfragen und Ranking](.scratch/global-search-filter/issues/02-add-advanced-global-search-query-and-ranking.md) – Phrasen, Negation, Fuzzy-Matching und Ranking bewusst nach V1.
21. [Fehlerhafte Project-Hierarchien diagnostizieren und reparieren](.scratch/project-hierarchy/issues/02-diagnose-and-repair-malformed-project-hierarchies.md) – Feedback und sichere Reparatur bewusst nach dem Hierarchie-Reader.
22. [Native Rich-Text-Beschreibungen](.scratch/description-rendering/issues/02-specify-native-rich-text-description-rendering-and-editing.md#implementation-tickets--2026-10-05) – Regel-Quercheck akzeptiert; 83 synthetische Beispiele. Acht Umsetzungstickets **03–10** vorbereitet, Start mit [03 — Dokumentmodell/HTML-Erhaltung](.scratch/description-rendering/issues/03-implement-source-preserving-description-model.md); reale VOA-Beispiele können parallel gesammelt werden. Native Umsetzung noch nicht begonnen; technische Abnahme in Ticket 10.
23. [Getrennte Historien für kontextspezifische Command-Prompts spezifizieren](.scratch/contextual-command-histories/issues/01-specify-independent-command-prompt-histories.md) – nach der M2-Ex-History; eigene session-lokale Historie je Prompt, ohne die Command-Grammatiken zu vereinheitlichen.
23a. [Workspace-/Datenpräferenzen spezifizieren](.scratch/workspace-configuration/issues/01-specify-workspace-preferences-configuration.md) – Default-Project/View, Startfilter und persistierte Navigationspräferenzen; nach der lokalen TUI-Konfiguration.
23b. [Multiline-Layout im Comment Composer wiederverwenden](.scratch/comment-composer-multiline-layout/issues/01-reuse-multiline-layout-in-comment-composer.md) – bewusst in eine spätere Session verschoben; zuerst screen-level Regressionstests ergänzen.

### Meilenstein 4

24. [View- und Global-Search-Navigationsverlauf hinzufügen](.scratch/navigation-history/issues/01-add-view-and-search-navigation-history.md) – `Ctrl+O` zurück und `Ctrl+I` vor über semantische TUI-Kontexte; zunächst den Restore-Contract spezifizieren.
25. [Timeboxed Focus Plans brainstormen](.scratch/timeboxed-focus-plans/issues/01-brainstorm-timeboxed-focus-plans.md) – mögliche additive Vimkunja-Schicht für begrenzte, regelmäßige Aufmerksamkeit auf fortlaufende Projects; erst diskutieren und prototypisieren.
26. [Full Vim Normal mode untersuchen](.scratch/full-vim-normal-mode/issues/01-investigate-full-vim-normal-mode.md) – vollständige Vim-Kompatibilität gegen Aufwand, Konflikte und einen begrenzten Scope bewerten; erst nach den Tier-1-Motions.

## Veröffentlichung

- Lokaler Start repariert (2026-10-06): bisherige Standard-DB hatte nur 116 der 150 Migrationen. Upgrade durch offizielles VOA 2.7.0 zuerst auf separater Kopie geprüft, dann auf Jans ausdrücklichen Auftrag produktive DB nach verifiziertem Backup migriert. Standardpfad weiterhin `internal/db/vikunja.db`; Backup `internal/db/vikunja-pre-2.7.0-20261006.db`. Opt-in-Upgrade-Regression prüft Ablehnung, Upgrade, Daten-Erhaltung und Store-Lesen; Startfehler nennen Pfad, Revisionsdetails und Upgrade-Hinweis.
- Review-Fixes (2026-10-06): permanente Löschung reserviert Task-/Alias-Nummern auch bei fehlenden oder veralteten Zählern atomar. Startprüfung verlangt vollständige Unique-Indizes einschließlich Task/Bucket-Zuordnung und einen einspaltigen Zähler-Primärschlüssel. Regressionen, volle Tests, Vet, Builds und VOA-/Upgrade-Roundtrips grün.

- [Vikunja-DB-Kompatibilität ausschließlich 2.7.0](.archive/2026-10-05/vikunja-db-compatibility/issues/01-verify-vikunja-2-3-to-2-7-compatibility.md) — **abgeschlossen; Kompatibilitätsvoraussetzung der Veröffentlichung erfüllt**. Produktive Revisionsprüfung, Store-Fixes und Regressionen implementiert; [Prüfbericht](.archive/2026-10-05/vikunja-db-compatibility/report.md) belegt Fresh-/Baseline-Roundtrips mit offiziellem VOA 2.7.0. Tests, Vet und beide Builds grün. Nur 2.7.0, sequenzieller Betrieb; permanente Löschung von Tasks mit Time Entries gesperrt. Native Windows-Terminaltests weiterhin offen.
- [01 — Lizenz und Dokumentation für GitHub](.archive/2026-10-07/publish-to-github/issues/01-publish-to-github.md) — **abgeschlossen und archiviert**. Lizenzinventar (30 Module/128 zusätzliche Hinweise), README, Beispiele und Links geprüft; HTML-Schreibschutz, Konfigurationspriorität, Binary-Start und Release-Grenzen präzisiert.
- [02 — Veröffentlichung nach dem Richtext-Merge](.scratch/publish-to-github/issues/02-prepare-publication-after-richtext-merge.md) — **wartet auf den vereinbarten Worktree-Merge; Ticket 01 ist abgeschlossen**. Danach CI/Tests, E-Mail-/Historienbereinigung in separater Veröffentlichungskopie, finale Secret-/DB-Prüfung und Push-Dry-Run. **Nur `main` und bestätigte Tags; keine Backup-Referenzen, kein Mirror-Push. Öffentlicher Push nur auf ausdrücklichen Auftrag.**

## Dokumentation

- [README.md](README.md) – Einführung, Bedienung, Vision
- `.scratch/` – einzige Quelle für offene Specs, Bugs und Featurewünsche
- `.archive/` – abgeschlossene Anforderungen, Tickets und historische Berichte
