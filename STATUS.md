# Vimkunja – Status

**Stand:** 2026-10-20 | **Branch:** main | **MVP:** abgeschlossen und archiviert | **Aktueller Meilenstein:** Meilenstein 3

## Aktueller Stand

MVP, **Meilenstein 1** und **Meilenstein 2** sind abgeschlossen: tägliche Aufgabenverwaltung ist ohne Wechsel zur Original-App praktikabel. Die weitere Arbeit konzentriert sich auf die offenen Spezifikationen und Erweiterungen des Meilensteins 3.

- Lokale Vikunja-SQLite-Datenbank: Projects, List View, All-Tasks-Overview, Task-Detail, Task-Editor und Task-Löschung
- Hierarchischer Projects Pane mit Einrückung, session-lokaler Faltung, defensiven Recovery-Einträgen und persistenter Geschwister-Sortierung
- Workflow-Status und Done-Toggle im Detail sowie Done-/Default-Bucket-Synchronisierung
- Einheitliche Due-Date-, Priority- und Identifier-Darstellung in Liste, Overview und Detail; stabiles Metadaten-Layout bei schmalen Pane-Breiten
- [Kompakte Task-Details](.archive/2026-08-14/task-detail-compact-layout/issues/01-preserve-description-access-in-short-terminals.md#regression-correction--2026-10-09): Description-Sichtbarkeit und Scroll-Feedback bei geringer Höhe repariert; Layout von Jan unter Windows akzeptiert.
- List-, Overview- und Kanban-Views mit Vim-Navigation, Pane-Fokus, Ex-Commands, Help, Fehlerlog und Leader-Key; Kanban-Karten- und Task-Detail-Auswahl bleiben synchron
- Separates englisches Tutorial mit flachem Startmenü: Bedienung oder Markup jeweils geführt oder frei üben, plus Quit; temporäre SQLite-Datenbank erst nach Moduswahl
- `go test ./...` grün

- [Aufgabenbasierte Tutorials](.archive/2026-10-20/tutorial-task-workflow/issues/02-convert-all-guided-tutorials.md) – allgemeiner Lernweg und Richtext-Workshop auf `main`: 26 allgemeine und 22 Richtext-Aufgaben mit freier Bedienung, Ergebnisprüfung, F5 für den aktuellen Startzustand und F8 nach Erfolg sowie F12 zum Überspringen; kompakte Anzeige und bestätigbare Kontext-/Hilfemodals. Richtext-Ausgangsstand auf `feature/description-model` bei `10ff5d2` gesichert.

- [Native Rich-Text-Descriptions](.archive/2026-10-19/description-rendering/issues/10-verify-native-rich-text-compatibility-and-documentation.md) – implementiert und manuell abgenommen: **AT-01–10 PASS**, Tutorial abgeschlossen. Auswahl/Formatierung, Checkboxen, Code/Tabs, Systemzwischenablage und Live-Vikunja-Roundtrip einschließlich In-App-Reload geprüft; bekannte Fehler durch Regressionen abgesichert. [Editor-Nacharbeiten](.archive/2026-10-19/description-rendering/issues/11-format-visual-selections-and-convert-touched-paragraphs.md) abgeschlossen.

- [Richtext-Fuzz-Nachprüfung](.archive/2026-10-20/description-rendering/fuzz-audit-2026-10-17/report.md) – abgeschlossen; deterministische Regressionen und kurze CI-Fuzz-Prüfungen sichern die Korrekturen ab.

## Vereinbarte nächste Schritte

1. [URL-Copy-Taste](.scratch/description-link-copy-shortcut/issues/01-choose-accessible-url-copy-shortcut.md) – parallel angestoßen; Umsetzung und Prüfung bleiben in dieser Story.
2. [Offene Subtasks prüfen und aufräumen](.archive/2026-10-20/release-ticket-cleanup/issues/01-reconcile-open-subtasks-before-publication.md) – abgeschlossen und archiviert: technischer Abgleich erledigt; Jan bestätigt **20/20 PASS** für Tutorial 9–26, Comment-`~` und Satzzeichen-Fix. [Prüfcheckliste](.archive/2026-10-20/release-ticket-cleanup/acceptance.html).
3. [GitHub-Initialisierung/Veröffentlichung abschließen](.scratch/publish-to-github/issues/02-prepare-publication-after-richtext-merge.md) – verbleibende Release-Voraussetzungen klären und finalen Veröffentlichungsstand prüfen.

## Aktuelle Tickets

Die All-Tasks-Übersicht, Task-Löschung, Cursor-Darstellung im Editor, Home/End-Aliasse, [Home/End im Insert Mode](.archive/2026-09-11/insert-mode-home-end/issues/01-support-home-end-in-insert-mode.md), Done-Toggle im Detail, Task-Metadaten-Darstellung, Task Labels, Task Comments inklusive Einzel-Löschung und [session-lokales Undo/Redo](.archive/2026-09-16/undo-task-actions/issues/01-undo-recent-task-actions.md) sind abgeschlossen. Die Standard-Kanban-Provisionierung, die einzelne und mehrfache manuelle Kanban Views, die Project-Hierarchie und die Globale Suche sind implementiert und archiviert. Die folgenden offenen Tickets sind nach Meilenstein eingeordnet; `needs-triage` in einem Ticket bedeutet weiterhin, dass dessen konkrete Spezifikation noch fehlt. Die Task-Status-Filter-Shortcuts sind implementiert und archiviert.

### Data integrity audit — 2026-10-04

- [Audit report and ticket index](.archive/2026-10-20/data-integrity-audit/report.md#ticket-index) — eleven findings tracked individually. F01/F02 are repaired, manually accepted and closed. F03 rejects missing target Projects transactionally; F04 preserves full due timestamps and detects same-day undo time conflicts. F05 preserves Done and completion metadata when workflow is unchanged. F06 initializes completed Project moves in destination Done Buckets. F07 resolves zero/NULL defaults to the first valid Bucket and preserves assignments without a Done Bucket. F08 preserves exact completion timestamps whenever Done does not transition, including Undo/Redo. F09 is repaired (unsafe recurring completion rejected). F10 is repaired (polymorphic deletion cleanup). F11 is repaired (stable creation UID). Ordinary SQLite/TUI regressions pass; F03–F11 are archived. All eleven audit findings are repaired and archived. Full race-enabled tests, build and vet pass; no manual acceptance of F03–F11 is claimed.

### Meilenstein 2 — abgeschlossen

Die read-only HTML-Description-Projektion, das Scroll-Feedback, die [weiteren Keyboard-Shortcuts](.archive/2026-08-08/more-key-shortcuts/issues/01-keyboard-shortcuts.md), die [kompakte Task-Detail-Ansicht](.archive/2026-08-14/task-detail-compact-layout/issues/01-preserve-description-access-in-short-terminals.md), die [Ex-Command-History samt Completion](.archive/2026-08-12/command-line-enhancements/issues/01-add-project-goto-command-history-and-completion.md), die [inline Task-Titelbearbeitung in List und Overview](.archive/2026-08-18/inline-task-editing/issues/01-edit-task-titles-inline-in-list-and-overview.md), die [feldlokalen Detail-Picker-Overlays](.archive/2026-09-03/detail-picker-overlay/issues/01-render-detail-pickers-as-field-local-overlays.md), der Due-Date-Picker mit relativen Helfern, kanonische UTC-Zeitstempel, der einzeilige Status-Footer, schmale Modal-Layouts und die [Benutzerkonfiguration](.archive/2026-10-20/user-configuration/issues/01-add-user-facing-tui-configuration.md) sind abgeschlossen und archiviert; Kanban-Title-Editing bleibt bewusst ausgenommen. Der native Rich-Text-Editor ist ebenfalls implementiert und abgenommen (Meilenstein 3).

### Meilenstein 3 — aktuell

12. [Checkbox-Listen-Fortschritt anzeigen](.archive/2026-10-20/checkbox-list-progress-summary/issues/01-show-checkbox-list-progress-summary.md) – abgeschlossen und archiviert; Native Checkbox-Änderungen aktualisieren den Fortschritt live, leere Listen werden entfernt. Importierter Plaintext bleibt im gemeinsamen Editor literal.
13. [Erweiterte Vim-Editor-Motions](.archive/2026-10-20/editor-motion-polish/issues/01-add-advanced-vim-editor-motions.md) – abgeschlossen und archiviert; letzter Comment-`~`-Nachtest von Jan akzeptiert. Alter HTML-Schreibschutz-Test durch die Richtext-Abnahme ersetzt; Originalreport bleibt erhalten.
14. [Project-Löschen semantisch spezifizieren](.scratch/project-deletion/issues/01-specify-project-deletion-semantics.md)
14a. [Project-Erstellung spezifizieren](.scratch/project-creation/issues/01-specify-project-creation-workflow.md) – Entry Point, Hierarchie, initiale View und Vikunja-kompatible Persistenz vor der Implementierung festlegen.
15. [Table View rendern](.scratch/table-view/issues/01-render-table-project-view.md)
16. [Gantt View rendern](.scratch/gantt-view/issues/01-render-gantt-project-view.md)
17. [Unabhängige Linktext-/URL-Bearbeitung evaluieren](.scratch/link-label-and-url-editor/issues/01-evaluate-independent-link-label-url-editing.md)
18. [Manuelle Kanban-Buckets verwalten](.scratch/kanban-bucket-management/issues/01-manage-manual-kanban-buckets.md)
19. [Lokale Suche im fokussierten Pane](.scratch/local-search/issues/01-search-within-focused-pane.md) – `/` sowie `n`/`N` sind vereinbart, die Pane-Semantik bleibt offen.
20. [Erweiterte globale Suchabfragen und Ranking](.scratch/global-search-filter/issues/02-add-advanced-global-search-query-and-ranking.md) – Phrasen, Negation, Fuzzy-Matching und Ranking bewusst nach V1.
21. [Fehlerhafte Project-Hierarchien diagnostizieren und reparieren](.scratch/project-hierarchy/issues/02-diagnose-and-repair-malformed-project-hierarchies.md) – Feedback und sichere Reparatur bewusst nach dem Hierarchie-Reader.
22. [Native Rich-Text-Beschreibungen](.archive/2026-10-19/description-rendering/issues/10-verify-native-rich-text-compatibility-and-documentation.md) – abgeschlossen und archiviert; Implementierung 03–09 sowie Integration/Nacharbeiten 10/11 geprüft. Alle zehn manuellen Abnahmepunkte einschließlich Windows-/AltGr-, Clipboard-, Resize- und Live-Vikunja-Prüfung bestanden.
22a. [Ausgewählten Task in die Zwischenablage kopieren](.scratch/task-clipboard-copy/issues/01-copy-selected-task-to-clipboard.md) – `:copy` für genau einen Task; nach der aktuellen Rich-Text-Arbeit einordnen, Metadatenformat und Umgang mit ungespeicherten Änderungen noch spezifizieren.
22b. [Bequemere URL-Copy-Taste](.scratch/description-link-copy-shortcut/issues/01-choose-accessible-url-copy-shortcut.md) – optionaler M3-Follow-up; freie Kleinschreibung statt `Space+Y` mit Jan wählen, danach konsistent verdrahten und dokumentieren.
23. [Getrennte Historien für kontextspezifische Command-Prompts spezifizieren](.scratch/contextual-command-histories/issues/01-specify-independent-command-prompt-histories.md) – nach der M2-Ex-History; eigene session-lokale Historie je Prompt, ohne die Command-Grammatiken zu vereinheitlichen.
23a. [Workspace-/Datenpräferenzen spezifizieren](.scratch/workspace-configuration/issues/01-specify-workspace-preferences-configuration.md) – Default-Project/View, Startfilter und persistierte Navigationspräferenzen; nach der lokalen TUI-Konfiguration.
23b. [Multiline-Layout im Comment Composer wiederverwenden](.archive/2026-10-20/comment-composer-multiline-layout/issues/01-reuse-multiline-layout-in-comment-composer.md) – technisch abgeschlossen und archiviert; gemeinsames Layout und Insert-Cursor mit Regressionen für Umbruch, Leerzeilen, Scrollen und Unicode.
23c. [Hilfe strukturieren und Richtext separieren](.scratch/help-structure/issues/01-rethink-help-structure-and-richtext-reference.md) – uneinheitliche Einrückung und schlechte Lesbarkeit; Aufteilung, Navigation und Layout brainstormen und kritisch prüfen, mindestens eigene Richtext-Hilfeseite. Konzeption offen, keine Umsetzung beauftragt.

### Meilenstein 4

24. [View- und Global-Search-Navigationsverlauf hinzufügen](.scratch/navigation-history/issues/01-add-view-and-search-navigation-history.md) – `Ctrl+O` zurück und `Ctrl+I` vor über semantische TUI-Kontexte; zunächst den Restore-Contract spezifizieren.
25. [Timeboxed Focus Plans brainstormen](.scratch/timeboxed-focus-plans/issues/01-brainstorm-timeboxed-focus-plans.md) – mögliche additive Vimkunja-Schicht für begrenzte, regelmäßige Aufmerksamkeit auf fortlaufende Projects; erst diskutieren und prototypisieren.
26. [Full Vim Normal mode untersuchen](.scratch/full-vim-normal-mode/issues/01-investigate-full-vim-normal-mode.md) – vollständige Vim-Kompatibilität gegen Aufwand, Konflikte und einen begrenzten Scope bewerten; erst nach den Tier-1-Motions.

## Veröffentlichung

- Lokaler Start repariert (2026-10-06): bisherige Standard-DB hatte nur 116 der 150 Migrationen. Upgrade durch offizielles VOA 2.7.0 zuerst auf separater Kopie geprüft, dann auf Jans ausdrücklichen Auftrag produktive DB nach verifiziertem Backup migriert. Standardpfad weiterhin `internal/db/vikunja.db`; Backup `internal/db/vikunja-pre-2.7.0-20261006.db`. Opt-in-Upgrade-Regression prüft Ablehnung, Upgrade, Daten-Erhaltung und Store-Lesen; Startfehler nennen Pfad, Revisionsdetails und Upgrade-Hinweis.
- Review-Fixes (2026-10-06): permanente Löschung reserviert Task-/Alias-Nummern auch bei fehlenden oder veralteten Zählern atomar. Startprüfung verlangt vollständige Unique-Indizes einschließlich Task/Bucket-Zuordnung und einen einspaltigen Zähler-Primärschlüssel. Regressionen, volle Tests, Vet, Builds und VOA-/Upgrade-Roundtrips grün.

- [Vikunja-DB-Kompatibilität ausschließlich 2.7.0](.archive/2026-10-05/vikunja-db-compatibility/issues/01-verify-vikunja-2-3-to-2-7-compatibility.md) — **abgeschlossen; Kompatibilitätsvoraussetzung der Veröffentlichung erfüllt**. Produktive Revisionsprüfung, Store-Fixes und Regressionen implementiert; [Prüfbericht](.archive/2026-10-05/vikunja-db-compatibility/report.md) belegt Fresh-/Baseline-Roundtrips mit offiziellem VOA 2.7.0. Tests, Vet und beide Builds grün. Nur 2.7.0, sequenzieller Betrieb; permanente Löschung von Tasks mit Time Entries gesperrt. Native Windows-Terminaltests weiterhin offen.
- [01 — Lizenz und Dokumentation für GitHub](.archive/2026-10-07/publish-to-github/issues/01-publish-to-github.md) — **abgeschlossen und archiviert**. Lizenzinventar (30 Module/128 zusätzliche Hinweise), README, Beispiele und Links geprüft; HTML-Schreibschutz, Konfigurationspriorität, Binary-Start und Release-Grenzen präzisiert.
- [02 — Veröffentlichung nach dem Richtext-Merge](.scratch/publish-to-github/issues/02-prepare-publication-after-richtext-merge.md) — **Richtext nach `main` übernommen; Ticket 01 ist abgeschlossen**. Danach CI/Tests, E-Mail-/Historienbereinigung in separater Veröffentlichungskopie, finale Secret-/DB-Prüfung und Push-Dry-Run. **Nur `main` und bestätigte Tags; keine Backup-Referenzen, kein Mirror-Push. Öffentlicher Push nur auf ausdrücklichen Auftrag.**

## Dokumentation

- [README.md](README.md) – Einführung, Bedienung, Vision
- `.scratch/` – einzige Quelle für offene Specs, Bugs und Featurewünsche
- `.archive/` – abgeschlossene Anforderungen, Tickets und historische Berichte
