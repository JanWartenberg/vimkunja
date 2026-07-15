# Vimkunja - Status

**Stand:** 2026-07-14 | **Branch:** main | **Progress:** in Arbeit (siehe Matrix; Fundament nicht vollständig)

---

## ✅ Was läuft:
- Projekte + Tasks anzeigen
- Details-View (Enter)
- Vim-Navigation (hjkl, gg, G, Ctrl+D/U, Ctrl+W)
- Task Toggle (`x`) → `done` + **updated timestamp** + Sync in jeden konfigurierten Done-Bucket ✅
- Priority-Farben (rot=urgent, grün=low)
- **Tests:** 11 Store-Tests + 2 Error-Log-Tests + 15 TUI-Tests grün ✅

## ❌ Was fehlt:
- Create-/Edit-Task-Editor (Phase 4)
- Vollständige Help-Pane inklusive `?` (Phase 5)
- Leader-Key (`Space+o`) (Phase 6)
- Auto-Refresh für weitere Writes (nach Create/Edit)
- Weitere TUI-Modelltests

---

## 🎯 Nächster Schritt:

**Phase 4: Create-/Edit-Editor vorbereiten**
- Die Produktentscheidungen für Create/Edit sind in `SPEC_FUNDAMENT.md`, Abschnitt 6, dokumentiert.
- Die Produktentscheidungen sind vollständig: Projektpicker alphabetisch mit grauem `No match found`; Description unterstützt Vim-Scrolling (`j/k`, `Ctrl+D/U`, `PageDown/PageUp`, `gg/G`) im Normal-Mode.
- Phase 3 ist umgesetzt: Command-Registry, Command-Mode, `:q`/`:quit`, `:qa`/`:quitall`, `:h`/`:help` und `:errors`.
- `:h` öffnet bis zur Help-Phase einen Platzhalter als normale Pane (`Ctrl+W j/k`, schließen nur mit `:q`); `:errors` zeigt den bestehenden Session-Error-Log.

---

## 📐 Fortschritt (SPEC_FUNDAMENT.md):

| Kriterium | Status |
|-----------|--------|
| REST Client entfernt | ✅ |
| `updated` timestamp bump | ✅ |
| Done-Bucket Move | ✅ |
| Commands | ✅ Phase 3 |
| Create/Edit Task | 🟡 Phase 4: Entscheidungen dokumentiert, technische Fragen offen |
| Help-Pane | ⏳ Phase 5 |
| Leader-Key | ⏳ Phase 6 |
| Auto-Refresh | ⏳ Phase 7.3 |
| Error-Log-Infrastruktur | ✅ Phase 2 + Anzeige per `:errors` |
| Error Footer | ✅ Phase-2-Schritt 2 |
| Tests | 🟡 11 Store-Tests + 2 Error-Log-Tests + 15 TUI-Tests; weitere TUI-Tests offen |

---

## 📝 TODO (Later):
- Description Rendering (Original-Vikunja-HTML → TUI-Text/Markdown): Parsing- und Formatierungsstrategie diskutieren
- Erweiterte Vim-Edit-Shortcuts (`o`/`Shift+o`, `i`/`a`, `Shift+i`/`Shift+a`)
- Editierbare Workflow-Status im Detail-Pane (z. B. Todo, Doing, Done) statt nur Done-Toggle
- Datumsformat, Truncation, etc.

**Details:** `.archive/TODO_UX.md`

---

## 🗂️ Doku:
- **STATUS.md** ← Diese Datei
- **SPEC_FUNDAMENT.md** - Komplette Spec
- `.archive/` - Erledigte Berichte/TODOs

---

**Letzter Commit:**  
`2514119` - updates after halfway grilling for edit of Task Detail Pane
