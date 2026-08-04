# Vimkunja – Status

**Stand:** 2026-08-04 | **Branch:** main | **MVP:** abgeschlossen und archiviert | **Aktueller Meilenstein:** Meilenstein 2

## Aktueller Stand

MVP und **Meilenstein 1** sind abgeschlossen: tägliche Aufgabenverwaltung ist ohne Wechsel zur Original-App praktikabel. Die weitere Arbeit beginnt mit Erweiterungen des Meilensteins 2.

- Lokale Vikunja-SQLite-Datenbank: Projects, List View, All-Tasks-Overview, Task-Detail, Task-Editor und Task-Löschung
- Hierarchischer Projects Pane mit Einrückung, session-lokaler Faltung, defensiven Recovery-Einträgen und persistenter Geschwister-Sortierung
- Workflow-Status und Done-Toggle im Detail sowie Done-/Default-Bucket-Synchronisierung
- Einheitliche Due-Date-, Priority- und Identifier-Darstellung in Liste, Overview und Detail
- List-, Overview- und Kanban-Views mit Vim-Navigation, Pane-Fokus, Ex-Commands, Help, Fehlerlog und Leader-Key
- Separates englisches Keystroke-Tutorial mit bei jedem Start frisch erzeugter, temporärer SQLite-Datenbank
- `go test ./...` grün

## Aktuelle Tickets

Die All-Tasks-Übersicht, Task-Löschung, Cursor-Darstellung im Editor, Home/End-Aliasse, Done-Toggle im Detail, Task-Metadaten-Darstellung und Task Labels sind abgeschlossen und archiviert. Die Standard-Kanban-Provisionierung, die einzelne manuelle Kanban View, die Project-Hierarchie und die Globale Suche sind implementiert und archiviert. Die folgenden offenen Tickets sind nach Meilenstein eingeordnet; `needs-triage` in einem Ticket bedeutet weiterhin, dass dessen konkrete Spezifikation noch fehlt.

### Meilenstein 2

1. [Vikunja-HTML-Beschreibungen lesbar rendern](.scratch/description-rendering/issues/01-render-vikunja-html-descriptions.md) – begrenzter Read-only-Schnitt; der vollständige native Rich-Text-Editor bleibt Meilenstein 3.
2. [Scroll-Position in langen Panes anzeigen](.scratch/scroll-position-feedback/issues/01-indicate-scroll-position.md)
3. [Task-Titel inline in Listen/Boards bearbeiten](.scratch/inline-task-editing/issues/01-edit-task-titles-inline-in-list-and-kanban.md)
4. [Ex-Commands, History und Completion erweitern](.scratch/command-line-enhancements/issues/01-add-project-goto-command-history-and-completion.md)
5. [Due Date- und Priority-Eingabe spezifizieren](.scratch/task-editor-special-inputs/issues/01-specify-date-picker-and-priority-input.md)
6. [Task Comments anzeigen und anlegen](.scratch/task-comments/issues/01-show-and-create-task-comments.md)
7. [Benutzerkonfiguration hinzufügen](.scratch/user-configuration/issues/01-add-user-facing-tui-configuration.md) – zunächst Konfigurationsrahmen und nur tatsächlich benötigte Einstellungen.

### Meilenstein 3

10. [Mehrere manuelle Kanban Views spezifizieren](.scratch/multi-kanban-view-semantics/issues/01-specify-workflow-for-projects-with-multiple-kanban-views.md)
11. [Letzte Task-Aktionen rückgängig machen](.scratch/undo-task-actions/issues/01-undo-recent-task-actions.md)
12. [Erweiterte Vim-Editor-Motions](.scratch/editor-motion-polish/issues/01-add-advanced-vim-editor-motions.md)
13. [Project-Löschen semantisch spezifizieren](.scratch/project-deletion/issues/01-specify-project-deletion-semantics.md)
14. [Table View rendern](.scratch/table-view/issues/01-render-table-project-view.md)
15. [Gantt View rendern](.scratch/gantt-view/issues/01-render-gantt-project-view.md)
16. [Checkbox-Listen-Fortschritt anzeigen](.scratch/checkbox-list-progress-summary/issues/01-show-checkbox-list-progress-summary.md)
17. [Unabhängige Linktext-/URL-Bearbeitung evaluieren](.scratch/link-label-and-url-editor/issues/01-evaluate-independent-link-label-url-editing.md)
18. [Manuelle Kanban-Buckets verwalten](.scratch/kanban-bucket-management/issues/01-manage-manual-kanban-buckets.md)
19. [Lokale Suche im fokussierten Pane](.scratch/local-search/issues/01-search-within-focused-pane.md) – `/` sowie `n`/`N` sind vereinbart, die Pane-Semantik bleibt offen.
20. [Erweiterte globale Suchabfragen und Ranking](.scratch/global-search-filter/issues/02-add-advanced-global-search-query-and-ranking.md) – Phrasen, Negation, Fuzzy-Matching und Ranking bewusst nach V1.
21. [Fehlerhafte Project-Hierarchien diagnostizieren und reparieren](.scratch/project-hierarchy/issues/02-diagnose-and-repair-malformed-project-hierarchies.md) – Feedback und sichere Reparatur bewusst nach dem Hierarchie-Reader.
22. [Native Rich-Text-Beschreibungen rendern und bearbeiten spezifizieren](.scratch/description-rendering/issues/02-specify-native-rich-text-description-rendering-and-editing.md) – zweiter Schritt nach dem Plain-Text-Reader aus Meilenstein 2; basiert auf dem bestehenden Rich-Text-Contract.

### Meilenstein 4

23. [View- und Global-Search-Navigationsverlauf hinzufügen](.scratch/navigation-history/issues/01-add-view-and-search-navigation-history.md) – `Ctrl+O` zurück und `Ctrl+I` vor über semantische TUI-Kontexte; zunächst den Restore-Contract spezifizieren.
24. [Timeboxed Focus Plans brainstormen](.scratch/timeboxed-focus-plans/issues/01-brainstorm-timeboxed-focus-plans.md) – mögliche additive Vimkunja-Schicht für begrenzte, regelmäßige Aufmerksamkeit auf fortlaufende Projects; erst diskutieren und prototypisieren.

## Dokumentation

- [README.md](README.md) – Einführung, Bedienung, Vision
- `.scratch/` – einzige Quelle für offene Specs, Bugs und Featurewünsche
- `.archive/` – abgeschlossene Anforderungen, Tickets und historische Berichte
