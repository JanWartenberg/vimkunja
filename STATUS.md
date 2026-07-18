# Vimkunja – Status

**Stand:** 2026-07-17 | **Branch:** main | **Fundament/MVP:** abgeschlossen und manuell verifiziert

## Aktueller Stand

- Lokale Vikunja-SQLite-Datenbank: Projects, List View, Task-Detail und Task-Editor
- Workflow-Status im Detail sowie `x`-Toggle zwischen Done- und Default-Bucket
- Vim-Navigation, Pane-Fokus, Ex-Commands, Help, Fehlerlog und Leader-Key
- `go test ./...` grün: 18 Store-, 2 Error-Log- und 43 TUI-Tests

## Aktuelle Tickets

Die Cursor-Darstellung im Editor ist abgeschlossen und archiviert. Alle folgenden Tickets sind bewusst noch `needs-triage`; die Reihenfolge ist ein Arbeitsvorschlag, keine feste Roadmap.

### Kurzfristige UX- und Editorarbeit

1. [Home/End als `gg`/`G`-Aliasse](.scratch/navigation-aliases/issues/01-add-home-end-navigation-aliases.md)
2. [Scroll-Position in langen Panes anzeigen](.scratch/scroll-position-feedback/issues/01-indicate-scroll-position.md)
3. [Vikunja-HTML-Beschreibungen lesbar rendern](.scratch/description-rendering/issues/01-render-vikunja-html-descriptions.md)
4. [Task-Metadaten besser darstellen](.scratch/task-presentation/issues/01-improve-task-date-priority-and-identifier-presentation.md)
5. [Erweiterte Vim-Editor-Motions](.scratch/editor-motion-polish/issues/01-add-advanced-vim-editor-motions.md)
6. [Task-Title-Länge entscheiden und validieren](.scratch/title-validation/issues/01-decide-and-enforce-task-title-length.md)
7. [Due Date- und Priority-Eingabe spezifizieren](.scratch/task-editor-special-inputs/issues/01-specify-date-picker-and-priority-input.md)
8. [Task-Titel inline in Listen/Boards bearbeiten](.scratch/inline-task-editing/issues/01-edit-task-titles-inline-in-list-and-kanban.md)

### Wiederhergestellter Feature-Backlog

9. [Kanban Views rendern und navigieren](.scratch/kanban-board/issues/01-render-and-navigate-kanban-views.md)
10. [All-Tasks-Übersicht über alle Projects](.scratch/all-tasks-overview/issues/01-add-cross-project-all-tasks-view.md)
11. [Project-Hierarchie im Projects Pane](.scratch/project-hierarchy/issues/01-render-project-hierarchy-in-projects-pane.md)
12. [Task Labels anzeigen und setzen](.scratch/task-labels/issues/01-show-and-edit-existing-task-labels.md)
13. [Globale Suche und Filter spezifizieren](.scratch/global-search-filter/issues/01-specify-global-search-and-filter.md)
14. [Task mit Bestätigung löschen](.scratch/task-deletion/issues/01-delete-task-with-explicit-confirmation.md)
15. [Project-Löschen semantisch spezifizieren](.scratch/project-deletion/issues/01-specify-project-deletion-semantics.md)
16. [Task Comments anzeigen und anlegen](.scratch/task-comments/issues/01-show-and-create-task-comments.md)
17. [Table View rendern](.scratch/table-view/issues/01-render-table-project-view.md)
18. [Gantt View rendern](.scratch/gantt-view/issues/01-render-gantt-project-view.md)
19. [Ex-Commands, History und Completion erweitern](.scratch/command-line-enhancements/issues/01-add-project-goto-command-history-and-completion.md)
20. [Benutzerkonfiguration hinzufügen](.scratch/user-configuration/issues/01-add-user-facing-tui-configuration.md)
21. [Letzte Task-Aktionen rückgängig machen](.scratch/undo-task-actions/issues/01-undo-recent-task-actions.md)
22. [Mehrere manuelle Kanban Views spezifizieren](.scratch/multi-kanban-view-semantics/issues/01-specify-workflow-for-projects-with-multiple-kanban-views.md)

## Dokumentation

- [README.md](README.md) – Einführung, Bedienung, Vision
- `.scratch/` – einzige Quelle für offene Specs, Bugs und Featurewünsche
- `.archive/` – abgeschlossene Anforderungen, Tickets und historische Berichte
