# Session Notes – Vimkunja

**Für:** Nächste Agent-Session  
**Letzte Änderung:** 2026-07-14

---

## Aktueller Stand

- Phase 3 (Commands) ist umgesetzt: Command-Registry, Command-Mode, `:q`/`:quit`, `:qa`/`:quitall`, `:h`/`:help`, `:errors`.
- Help ist bis Phase 5 ein Platzhalter, aber eine normale Pane: `Ctrl+W j/k` wechselt den Fokus, nur `:q` schließt sie.
- Error-Pane zeigt den Session-Log und aktualisiert sich bei neuen TUI-Fehlern.
- Tests/Vet zuletzt grün: 11 Store-, 2 Error-Log- und 15 TUI-Tests.

## Phase 4 – Create/Edit Task: beschlossene Produktregeln

Vollständige Referenz: `SPEC_FUNDAMENT.md`, Abschnitt 6.

- Detail-Pane ist der gemeinsame Editor für neue und bestehende Tasks.
- `o` (vorläufig) in Project-/Tasks-/später Overview-Pane sowie `:new` ohne Argumente öffnen einen Entwurf; `%` bleibt frei.
- Formular: `Project | Title`, `Priority | Due Date`, danach volle Breite Description; Done ist nur lesbar.
- Pflicht: Projekt und Titel. Optional: Priorität, Due Date, Description.
- Speichern: `:w`/`:write`; `:wq` speichert und schließt. `:q`/`:quit` verweigert bei Dirty-State; `:q!`/`:quit!` verwirft und schließt.
- Projektpicker und Prioritätspicker: `i`, Filter bzw. Auswahl mit `j/k` oder Pfeilen, `Enter` übernehmen, `Esc` unverändert abbrechen.
- Due Date: nur `YYYY-MM-DD`; leer → `NULL`, gesetzt → `00:00`; invalid vor Write abweisen, loggen, roten Footer zeigen und im Editor bleiben.
- Footer zeigt immer `[NORMAL]`, `[INSERT]` oder `[COMMAND]`; Dirty-State `[+]`. Fokusfeld bekommt zentral definierte pinke Akzentfarbe.

## Vor Umsetzung von Phase 4 noch entscheiden

1. `Esc` im Detail-Normal-Mode: Pane schließen (alter Zustand) oder Vim-artiges No-op?
2. Minimaler Texteditor: Backspace/Delete und Cursorplatzierung in ein- und mehrzeiligen Feldern.
3. Projektpicker: Sortierung, Darstellung ohne Treffer, Description-Scrolling im ersten Prototyp.

## Session-Start

```bash
cd /workspace/vimkunja
go test ./...
go vet ./...
```

Danach offene Phase-4-Fragen nacheinander klären. Erst nach expliziter Umsetzungserlaubnis implementieren.
