# Vimkunja – Status

**Stand:** 2026-07-17 | **Branch:** main | **Fundament/MVP:** abgeschlossen und manuell verifiziert

## Aktueller Stand

- Lokale Vikunja-SQLite-Datenbank: Projects, List View, Task-Detail und Task-Editor
- Workflow-Status im Detail sowie `x`-Toggle zwischen Done- und Default-Bucket
- Vim-Navigation, Pane-Fokus, Ex-Commands, Help, Fehlerlog und Leader-Key
- `go test ./...` grün: 18 Store-, 2 Error-Log- und 43 TUI-Tests

## Aktuelle Tickets

Die Cursor-Darstellung im Editor ist abgeschlossen und archiviert. Alle folgenden Tickets sind bewusst noch `needs-triage`; die Reihenfolge ist ein Arbeitsvorschlag, keine feste Roadmap.

1. [Home/End als `gg`/`G`-Aliasse](.scratch/navigation-aliases/issues/01-add-home-end-navigation-aliases.md)
2. [Scroll-Position in langen Panes anzeigen](.scratch/scroll-position-feedback/issues/01-indicate-scroll-position.md)
3. [Vikunja-HTML-Beschreibungen lesbar rendern](.scratch/description-rendering/issues/01-render-vikunja-html-descriptions.md)
4. [Task-Metadaten besser darstellen](.scratch/task-presentation/issues/01-improve-task-date-priority-and-identifier-presentation.md)
5. [Erweiterte Vim-Editor-Motions](.scratch/editor-motion-polish/issues/01-add-advanced-vim-editor-motions.md)
6. [Task-Title-Länge entscheiden und validieren](.scratch/title-validation/issues/01-decide-and-enforce-task-title-length.md)

## Dokumentation

- [README.md](README.md) – Einführung, Bedienung, Vision
- `.scratch/` – einzige Quelle für offene Specs, Bugs und Featurewünsche
- `.archive/` – abgeschlossene Anforderungen, Tickets und historische Berichte
