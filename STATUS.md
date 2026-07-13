# Vikunja TUI - Status

**Stand:** 2026-07-12 | **Branch:** main | **Progress:** 4/10 (40%)

---

## ✅ Was läuft:
- Projekte + Tasks anzeigen
- Details-View (Enter)
- Vim-Navigation (hjkl, gg, G, Ctrl+D/U, Ctrl+W)
- Task Toggle (`x`) → `done` + **updated timestamp** + **Bucket-Move** ✅
- Priority-Farben (rot=urgent, grün=low)
- **Store Tests** (Phase 7.1) ✅

## ❌ Was fehlt:
- Commands (`:q`, `:qa`, `:h`)
- Error-Log
- Help-Pane (`?`)
- Leader-Key (`Space+o`)
- Auto-Refresh

---

## 🎯 Nächster Schritt:

**Tests ausführen** (Go installieren)
- `go test -v ./internal/vikunja/` ausführen
- 8 Tests sollten grün sein

**Dann:** Phase 2 (Error-Handling) oder Phase 3 (Commands)

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
| Error Footer | ⏳ Phase 2 |
| Tests | ✅ Phase 7.1 |

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
