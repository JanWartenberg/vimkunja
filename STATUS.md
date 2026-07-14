# Vimkunja - Status

**Stand:** 2026-07-14 | **Branch:** main | **Progress:** in Arbeit (siehe Matrix; Fundament nicht vollständig)

---

## ✅ Was läuft:
- Projekte + Tasks anzeigen
- Details-View (Enter)
- Vim-Navigation (hjkl, gg, G, Ctrl+D/U, Ctrl+W)
- Task Toggle (`x`) → `done` + **updated timestamp** + Sync in jeden konfigurierten Done-Bucket ✅
- Priority-Farben (rot=urgent, grün=low)
- **Tests:** 11 Store-Tests + 2 Error-Log-Tests + 2 TUI-Tests grün ✅

## ❌ Was fehlt:
- Commands (`:q`, `:qa`, `:h`)
- Error-Log
- Help-Pane (`?`)
- Leader-Key (`Space+o`)
- Auto-Refresh

---

## 🎯 Nächster Schritt:

**Vor Phase 4: Create-Task-Grilling durchführen**
- Phase 3 ist umgesetzt: Command-Registry, Command-Mode, `:q`, `:qa`, `:quitall`, `:h`/`:help` und `:errors`.
- `:h` öffnet bis zur Help-Phase einen Platzhalter als normale Pane (`Ctrl+W j/k`, schließen nur mit `:q`); `:errors` zeigt den bestehenden Session-Error-Log.
- Die zwei internen Schritte von Phase 2 sind umgesetzt: thread-sicherer In-Memory-Error-Log, zentrale Erfassung von Lade- und Schreibfehlern sowie roter Fehler-Footer.
- Footer-Fehler werden bei der nächsten Taste ausgeblendet; der Eintrag bleibt im Session-Log.
- Erfolgreiches Task-Toggle und anschließender Reload sind getrennte Nachrichten; ein fehlgeschlagener Write löst keinen Reload aus.
- Als Nächstes: Zielkontext, Keybinding und `:new`-Semantik für Create Task festlegen.

---

## 📐 Fortschritt (SPEC_FUNDAMENT.md):

| Kriterium | Status |
|-----------|--------|
| REST Client entfernt | ✅ |
| `updated` timestamp bump | ✅ |
| Done-Bucket Move | ✅ |
| Commands | ✅ Phase 3 |
| Create Task | ⏳ Phase 4 (kurzes Grilling vor der Spec) |
| Help-Pane | ⏳ Phase 5 |
| Leader-Key | ⏳ Phase 6 |
| Auto-Refresh | ⏳ Phase 7.3 |
| Error-Log-Infrastruktur | ✅ Phase 2 + Anzeige per `:errors` |
| Error Footer | ✅ Phase-2-Schritt 2 |
| Tests | 🟡 11 Store-Tests + 2 Error-Log-Tests + 15 TUI-Tests; weitere TUI-Tests offen |

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
