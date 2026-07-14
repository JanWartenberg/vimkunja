# Vimkunja - Status

**Stand:** 2026-07-13 | **Branch:** main | **Progress:** 4/10 (40%)

---

## ✅ Was läuft:
- Projekte + Tasks anzeigen
- Details-View (Enter)
- Vim-Navigation (hjkl, gg, G, Ctrl+D/U, Ctrl+W)
- Task Toggle (`x`) → `done` + **updated timestamp** + **Bucket-Move** ✅
- Priority-Farben (rot=urgent, grün=low)
- **Tests:** 7 Store-Tests + 2 Error-Log-Tests grün ✅

## ❌ Was fehlt:
- Commands (`:q`, `:qa`, `:h`)
- Error-Log
- Help-Pane (`?`)
- Leader-Key (`Space+o`)
- Auto-Refresh

---

## 🎯 Nächster Schritt:

**Phase 2 fortsetzen – Schritt 2 (sichtbares Error-Feedback)**
- Schritt 1 ist umgesetzt: thread-sicherer In-Memory-Error-Log und zentrale Erfassung von Lade- und Schreibfehlern.
- Erfolgreiches Task-Toggle und anschließender Reload sind nun getrennte Nachrichten; ein fehlgeschlagener Write löst keinen Reload aus.
- Als Nächstes: Fehler-Footer rot rendern und erst bei der nächsten Taste ausblenden.
- `:errors` bleibt bewusst offen für Phase 3 (Command-Infrastruktur).

---

## 📐 Fortschritt (SPEC_FUNDAMENT.md):

| Kriterium | Status |
|-----------|--------|
| REST Client entfernt | ✅ |
| `updated` timestamp bump | ✅ |
| Done-Bucket Move | ✅ |
| Commands | ⏳ Phase 3 |
| Help-Pane | ⏳ Phase 4 |
| Leader-Key | ⏳ Phase 5 |
| Auto-Refresh | ⏳ Phase 6.3 |
| Error-Log-Infrastruktur | 🟡 Phase 2, Schritt 1 erledigt |
| Error Footer | ⏳ Phase 2, Schritt 2 |
| Tests | 🟡 7 Store-Tests + 2 Error-Log-Tests; TUI-Tests offen |

---

## 📝 TODO (Later):
- Description Rendering (HTML → Text/Markdown)
- Datumsformat, Truncation, etc.

**Details:** `.archive/TODO_UX.md`

---

## 🗂️ Doku:
- **STATUS.md** ← Diese Datei
- **SPEC_FUNDAMENT.md** - Komplette Spec
- `.archive/` - Erledigte Berichte/TODOs
- `.pi/SESSION_NOTES.md` - Agent-Notizen (für nächste Session)

---

**Letzter Commit:**  
`ad90e1d` - feat: move tasks to done_bucket_id on toggle (Phase 6.2)
