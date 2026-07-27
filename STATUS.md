# Vimkunja – Status

**Stand:** 2026-07-27 | **Branch:** main | **Fundament/MVP:** abgeschlossen und manuell verifiziert

## Aktueller Stand

- Lokale Vikunja-SQLite-Datenbank: Projects, List View, Task-Detail und Task-Editor
- Workflow-Status im Detail sowie `x`-Toggle zwischen Done- und Default-Bucket
- Vim-Navigation, Pane-Fokus, Ex-Commands, Help, Fehlerlog und Leader-Key
- `go test ./...` grün: 18 Store-, 2 Error-Log- und 43 TUI-Tests

## Aktuelle Tickets

Die Cursor-Darstellung im Editor sowie die Home/End-Aliasse sind abgeschlossen und archiviert. Alle offenen Tickets bleiben bis zur Spezifikation `needs-triage`. Die Priorität richtet sich jetzt nach dem **Daily-driver-Ziel**: Vimkunja soll zuerst die tägliche Aufgabenverwaltung ohne Wechsel zur Original-App ermöglichen.

### Hoch – Daily driver

1. [All-Tasks-Übersicht über alle Projects](.scratch/all-tasks-overview/issues/01-add-cross-project-all-tasks-view.md) – zentraler Einstieg in alle offenen Tasks.
2. [Kanban Views rendern und navigieren](.scratch/kanban-board/issues/01-render-and-navigate-kanban-views.md) – zunächst als minimale Board-Ansicht für eine aktive Kanban View; Mehr-View-Semantik folgt separat.
3. [Task mit Bestätigung löschen](.scratch/task-deletion/issues/01-delete-task-with-explicit-confirmation.md) – notwendige Grundoperation für den täglichen Einsatz.

### Mittel – nach dem Daily-driver-Kern

4. [Task-Metadaten besser darstellen](.scratch/task-presentation/issues/01-improve-task-date-priority-and-identifier-presentation.md)
5. [Task Done in der Task Detail Pane togglen](.scratch/task-detail-done-toggle/issues/01-toggle-task-done-from-detail-pane.md)
6. [Globale Suche und Filter spezifizieren](.scratch/global-search-filter/issues/01-specify-global-search-and-filter.md)
7. [Project-Hierarchie im Projects Pane](.scratch/project-hierarchy/issues/01-render-project-hierarchy-in-projects-pane.md)
8. [Task Labels anzeigen und setzen](.scratch/task-labels/issues/01-show-and-edit-existing-task-labels.md)
9. [Task-Titel inline in Listen/Boards bearbeiten](.scratch/inline-task-editing/issues/01-edit-task-titles-inline-in-list-and-kanban.md)
10. [Mehrere manuelle Kanban Views spezifizieren](.scratch/multi-kanban-view-semantics/issues/01-specify-workflow-for-projects-with-multiple-kanban-views.md)
11. [Ex-Commands, History und Completion erweitern](.scratch/command-line-enhancements/issues/01-add-project-goto-command-history-and-completion.md)
12. [Due Date- und Priority-Eingabe spezifizieren](.scratch/task-editor-special-inputs/issues/01-specify-date-picker-and-priority-input.md)
13. [Letzte Task-Aktionen rückgängig machen](.scratch/undo-task-actions/issues/01-undo-recent-task-actions.md)

### Niedrig – bewusst zurückgestellt

14. [Scroll-Position in langen Panes anzeigen](.scratch/scroll-position-feedback/issues/01-indicate-scroll-position.md)
15. [Vikunja-HTML-Beschreibungen lesbar rendern](.scratch/description-rendering/issues/01-render-vikunja-html-descriptions.md) – umfangreiche Rich-Text-Bearbeitung ist nicht Teil des Daily-driver-Kerns; ein kleiner reiner Leseumfang kann später separat geschnitten werden.
16. [Erweiterte Vim-Editor-Motions](.scratch/editor-motion-polish/issues/01-add-advanced-vim-editor-motions.md)
17. [Task-Title-Länge entscheiden und validieren](.scratch/title-validation/issues/01-decide-and-enforce-task-title-length.md)
18. [Project-Löschen semantisch spezifizieren](.scratch/project-deletion/issues/01-specify-project-deletion-semantics.md)
19. [Task Comments anzeigen und anlegen](.scratch/task-comments/issues/01-show-and-create-task-comments.md)
20. [Table View rendern](.scratch/table-view/issues/01-render-table-project-view.md)
21. [Gantt View rendern](.scratch/gantt-view/issues/01-render-gantt-project-view.md)
22. [Benutzerkonfiguration hinzufügen](.scratch/user-configuration/issues/01-add-user-facing-tui-configuration.md)
23. [Checkbox-Listen-Fortschritt anzeigen](.scratch/checkbox-list-progress-summary/issues/01-show-checkbox-list-progress-summary.md)
24. [Unabhängige Linktext-/URL-Bearbeitung evaluieren](.scratch/link-label-and-url-editor/issues/01-evaluate-independent-link-label-url-editing.md)

## Dokumentation

- [README.md](README.md) – Einführung, Bedienung, Vision
- `.scratch/` – einzige Quelle für offene Specs, Bugs und Featurewünsche
- `.archive/` – abgeschlossene Anforderungen, Tickets und historische Berichte
