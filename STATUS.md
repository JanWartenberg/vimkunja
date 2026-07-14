# Vimkunja - Status

**Stand:** 2026-07-13 | **Branch:** main | **Progress:** 4/10 (40%)

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

**Phase 3 fortsetzen – Schritt 3.2 (Command-Mode)**
- Schritt 3.1 ist umgesetzt: Command-Registry mit Alias-Auflösung, einfacher Argument-Parser und Fehlern für leere bzw. unbekannte Commands.
- Die zwei internen Schritte von Phase 2 sind umgesetzt: thread-sicherer In-Memory-Error-Log, zentrale Erfassung von Lade- und Schreibfehlern sowie roter Fehler-Footer.
- Footer-Fehler werden bei der nächsten Taste ausgeblendet; der Eintrag bleibt im Session-Log.
- Erfolgreiches Task-Toggle und anschließender Reload sind getrennte Nachrichten; ein fehlgeschlagener Write löst keinen Reload aus.
- Als Nächstes: Command-Mode und `:q`, `:qa`; anschließend `:h` und `:errors` verdrahten.

---

## 📐 Fortschritt (SPEC_FUNDAMENT.md):

| Kriterium | Status |
|-----------|--------|
| REST Client entfernt | ✅ |
| `updated` timestamp bump | ✅ |
| Done-Bucket Move | ✅ |
| Commands | 🟡 Phase 3, Registry (Schritt 3.1) erledigt |
| Create Task | ⏳ Phase 4 (kurzes Grilling vor der Spec) |
| Help-Pane | ⏳ Phase 5 |
| Leader-Key | ⏳ Phase 6 |
| Auto-Refresh | ⏳ Phase 7.3 |
| Error-Log-Infrastruktur | ✅ intern in Phase 2 (Anzeige per `:errors` folgt in Phase 3) |
| Error Footer | ✅ Phase-2-Schritt 2 |
| Tests | 🟡 11 Store-Tests + 2 Error-Log-Tests + 5 TUI-Tests; weitere TUI-Tests offen |

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
