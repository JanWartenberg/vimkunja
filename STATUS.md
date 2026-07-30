# Vimkunja – Status

**Stand:** 2026-07-30 | **Branch:** main | **MVP:** abgeschlossen und archiviert | **Aktueller Meilenstein:** M2 – Bald danach

## Aktueller Stand

MVP und **M1 – Daily Driver** sind abgeschlossen: tägliche Aufgabenverwaltung ist ohne Wechsel zur Original-App praktikabel. Die weitere Arbeit beginnt mit M2-Erweiterungen.

- Lokale Vikunja-SQLite-Datenbank: Projects, List View, All-Tasks-Overview, Task-Detail, Task-Editor und Task-Löschung
- Workflow-Status und Done-Toggle im Detail sowie Done-/Default-Bucket-Synchronisierung
- Einheitliche Due-Date-, Priority- und Identifier-Darstellung in Liste, Overview und Detail
- List-, Overview- und Kanban-Views mit Vim-Navigation, Pane-Fokus, Ex-Commands, Help, Fehlerlog und Leader-Key
- `go test ./...` grün

## Aktuelle Tickets

Die All-Tasks-Übersicht, Task-Löschung, Cursor-Darstellung im Editor, Home/End-Aliasse, Done-Toggle im Detail und Task-Metadaten-Darstellung sind abgeschlossen und archiviert. Die Standard-Kanban-Provisionierung und die einzelne manuelle Kanban View sind implementiert und archiviert. Die folgenden offenen Tickets sind nach Meilenstein eingeordnet; `needs-triage` in einem Ticket bedeutet weiterhin, dass dessen konkrete Spezifikation noch fehlt.

### M2 – Bald danach

1. [Globale Suche und Filter spezifizieren](.scratch/global-search-filter/issues/01-specify-global-search-and-filter.md)
2. [Project-Hierarchie im Projects Pane](.scratch/project-hierarchy/issues/01-render-project-hierarchy-in-projects-pane.md)
3. [Task Labels anzeigen und setzen](.scratch/task-labels/issues/01-show-and-edit-existing-task-labels.md)
4. [Task-Titel inline in Listen/Boards bearbeiten](.scratch/inline-task-editing/issues/01-edit-task-titles-inline-in-list-and-kanban.md)
5. [Ex-Commands, History und Completion erweitern](.scratch/command-line-enhancements/issues/01-add-project-goto-command-history-and-completion.md)
6. [Due Date- und Priority-Eingabe spezifizieren](.scratch/task-editor-special-inputs/issues/01-specify-date-picker-and-priority-input.md)
7. [Scroll-Position in langen Panes anzeigen](.scratch/scroll-position-feedback/issues/01-indicate-scroll-position.md)
8. [Vikunja-HTML-Beschreibungen lesbar rendern](.scratch/description-rendering/issues/01-render-vikunja-html-descriptions.md) – begrenzter Read-only-Schnitt; der vollständige native Rich-Text-Editor bleibt M3.
9. [Task Comments anzeigen und anlegen](.scratch/task-comments/issues/01-show-and-create-task-comments.md)
10. [Benutzerkonfiguration hinzufügen](.scratch/user-configuration/issues/01-add-user-facing-tui-configuration.md) – zunächst Konfigurationsrahmen und nur tatsächlich benötigte Einstellungen.

### M3 – Bewusst zurückgestellt

11. [Mehrere manuelle Kanban Views spezifizieren](.scratch/multi-kanban-view-semantics/issues/01-specify-workflow-for-projects-with-multiple-kanban-views.md)
12. [Letzte Task-Aktionen rückgängig machen](.scratch/undo-task-actions/issues/01-undo-recent-task-actions.md)
13. [Erweiterte Vim-Editor-Motions](.scratch/editor-motion-polish/issues/01-add-advanced-vim-editor-motions.md)
14. [Project-Löschen semantisch spezifizieren](.scratch/project-deletion/issues/01-specify-project-deletion-semantics.md)
15. [Table View rendern](.scratch/table-view/issues/01-render-table-project-view.md)
16. [Gantt View rendern](.scratch/gantt-view/issues/01-render-gantt-project-view.md)
17. [Checkbox-Listen-Fortschritt anzeigen](.scratch/checkbox-list-progress-summary/issues/01-show-checkbox-list-progress-summary.md)
18. [Unabhängige Linktext-/URL-Bearbeitung evaluieren](.scratch/link-label-and-url-editor/issues/01-evaluate-independent-link-label-url-editing.md)
19. [Manuelle Kanban-Buckets verwalten](.scratch/kanban-bucket-management/issues/01-manage-manual-kanban-buckets.md)

## Dokumentation

- [README.md](README.md) – Einführung, Bedienung, Vision
- `.scratch/` – einzige Quelle für offene Specs, Bugs und Featurewünsche
- `.archive/` – abgeschlossene Anforderungen, Tickets und historische Berichte
