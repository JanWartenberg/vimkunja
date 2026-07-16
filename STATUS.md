# Vimkunja - Status

**Stand:** 2026-07-14 | **Branch:** main | **Progress:** in Arbeit (siehe Matrix; Fundament nicht vollständig)

---

## ✅ Was läuft:
- Projekte + Tasks anzeigen
- Details-View (Enter)
- Vim-Navigation (hjkl, gg, G, Ctrl+D/U, Ctrl+W)
- Task Toggle (`x`) → `done` + **updated timestamp** + Sync in jeden konfigurierten Done-Bucket ✅
- Priority-Farben (rot=urgent, grün=low)
- Create-/Edit-Editor: Entwurf und bestehende Tasks, Projekt-/Prioritätspicker, Date-Maske, Dirty-Schutz sowie `:new`, `:w`, `:wq`, `:wqa`, `:q!` ✅ Phase 4
- Vollständige, scrollbare Help-Pane via `:h` und `:help` ✅ Phase 5
- Leader-Key (`Space+o`) zum Ein-/Ausblenden der Projects-Pane ✅ Phase 6
- **Tests:** 13 Store-Tests + 2 Error-Log-Tests + 27 TUI-Tests grün ✅

## ❌ Was fehlt:
- Manueller Smoke-Test und Abschlussprüfung des Fundaments (Ticket 02)

---

## 🎯 Nächster Schritt:

**Phase 8: Fundament-Abschluss**
- Die fehlenden TUI-Modelltests für Navigation sind ergänzt.
- Als Nächstes: automatisierte Tests sowie den manuellen Smoke-Test gegen die Fundament-Abschlusskriterien verifizieren.
- `:errors` zeigt den bestehenden Session-Error-Log.

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
| Tests | 🟡 13 Store-Tests + 2 Error-Log-Tests + 27 TUI-Tests; manueller Smoke-Test offen |

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

**Arbeitsstand:**
Phase 4 bis 6 sind implementiert und committet. Arbeitsbaum ist sauber. Letzter Commit: `fb533a2` - Implement Phase 6 leader key and pane collapse
