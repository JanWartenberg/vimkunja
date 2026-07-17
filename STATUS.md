# Vimkunja – Status

**Stand:** 2026-07-17 | **Branch:** main | **Fundament/MVP:** abgeschlossen und manuell verifiziert

## Aktueller Stand

- Lokale Vikunja-SQLite-Datenbank: Projects, List View, Task-Detail und Task-Editor
- Workflow-Status im Detail sowie `x`-Toggle zwischen Done- und Default-Bucket
- Vim-Navigation, Pane-Fokus, Ex-Commands, Help, Fehlerlog und Leader-Key
- `go test ./...` grün: 18 Store-, 2 Error-Log- und 43 TUI-Tests

## Aktuelle Tickets

Alle folgenden Tickets sind bewusst noch `needs-triage`; die Reihenfolge ist ein Arbeitsvorschlag, keine feste Roadmap.

1. [Cursor-Darstellung im Editor](.scratch/cursor-shapes/issues/01-improve-editor-cursor-shapes.md)
2. [Home/End als `gg`/`G`-Aliasse](.scratch/navigation-aliases/issues/01-add-home-end-navigation-aliases.md)
3. [Scroll-Position in langen Panes anzeigen](.scratch/scroll-position-feedback/issues/01-indicate-scroll-position.md)
4. [Vikunja-HTML-Beschreibungen lesbar rendern](.scratch/description-rendering/issues/01-render-vikunja-html-descriptions.md)
5. [Task-Metadaten besser darstellen](.scratch/task-presentation/issues/01-improve-task-date-priority-and-identifier-presentation.md)
6. [Erweiterte Vim-Editor-Motions](.scratch/editor-motion-polish/issues/01-add-advanced-vim-editor-motions.md)

## Dokumentation

- [README.md](README.md) – Einführung, Bedienung, Vision
- `.scratch/` – einzige Quelle für offene Specs, Bugs und Featurewünsche
- `.archive/` – abgeschlossene Anforderungen, Tickets und historische Berichte
