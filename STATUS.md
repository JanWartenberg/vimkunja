# Vimkunja - Status

**Stand:** 2026-07-17 | **Branch:** main | **Progress:** Fundament/MVP abgeschlossen

---

## ✅ Was läuft:
- Projekte + Tasks anzeigen
- Details-View (Enter)
- Vim-Navigation (hjkl, gg, G, Ctrl+D/U, Ctrl+W)
- Workflow-Status im Detail: Bucket-Picker (z. B. To-Do, Doing, Done) synchronisiert `done`, `done_at` und `updated` ✅
- Task Toggle (`x`) → Done- bzw. Default-Bucket + **updated timestamp** ✅
- Priority-Farben (rot=urgent, grün=low)
- Create-/Edit-Editor: Entwurf und bestehende Tasks, Projekt-/Prioritäts-/Statuspicker, Date-Maske, Dirty-Schutz sowie `:new`, `:w`, `:wq`, `:wqa`, `:q!`, `:qa!` ✅
- Vollständige, scrollbare Help-Pane via `:h` und `:help` ✅ Phase 5
- Leader-Key (`Space+o`) zum Ein-/Ausblenden der Projects-Pane ✅ Phase 6
- **Tests:** 18 Store-Tests + 2 Error-Log-Tests + 43 TUI-Tests grün ✅
- Automatisierte und manuelle Fundament-/Workflow-Smoke-Tests ✅

---

## 🎯 Nächster Schritt:

- Fundament-Spec und erledigte Tickets archivieren.
- Eine nächste Feature-Spec aus dem priorisierten UX- oder Kanban-Backlog erstellen.

---

## 📐 Fortschritt (SPEC_FUNDAMENT.md):

| Kriterium | Status |
|-----------|--------|
| REST Client entfernt | ✅ |
| `updated` timestamp bump | ✅ |
| Done-Bucket Move | ✅ |
| Commands | ✅ Phase 3 |
| Create/Edit Task | ✅ Phase 4: manuell abgenommen |
| Help-Pane | ✅ Phase 5 |
| Leader-Key | ✅ Phase 6 |
| Auto-Refresh | ✅ nach Toggle sowie Create/Edit |
| Error-Log-Infrastruktur | ✅ Phase 2 + Anzeige per `:errors` |
| Error Footer | ✅ Phase-2-Schritt 2 |
| Tests | ✅ 18 Store-Tests + 2 Error-Log-Tests + 43 TUI-Tests; manueller Smoke-Test bestanden |

---

## 📝 TODO (Later):
- Description Rendering (Original-Vikunja-HTML → TUI-Text/Markdown): Parsing- und Formatierungsstrategie diskutieren
- Erweiterte Vim-Edit-Shortcuts (`o`/`Shift+o`, `i`/`a`, `Shift+i`/`Shift+a`)
- Datumsformat, Truncation, etc.

**Details:** `.archive/TODO_UX.md`

---

## 🗂️ Doku:
- **STATUS.md** ← Diese Datei
- **SPEC_FUNDAMENT.md** - Komplette Spec
- `.archive/` - Erledigte Berichte/TODOs

---

**Arbeitsstand:**
Fundament/MVP einschließlich Workflow-Status ist implementiert und manuell verifiziert. Weitere Arbeit liegt im lokalen `.scratch/`-Backlog.
