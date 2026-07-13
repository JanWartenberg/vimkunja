# Session Notes – Vikunja TUI

**Für:** Nächste Agent-Session  
**Letzte Änderung:** 2026-07-12

---

## 🎯 Phase 7.1 - ERLEDIGT (aber Tests nicht ausgeführt!)

**Was passiert ist:**
- Store-Tests erstellt: `internal/vikunja/sqlite_store_test.go`
- 8 Tests decken ab:
  - ✅ Timestamp-Updates (`updated`)
  - ✅ Bucket-Move bei `done=true`
  - ✅ `done_at` setzen/clearen
  - ✅ Pagination
  - ✅ Position-Ordering
  - ✅ Archivierte Projekte ausblenden

**Problem:** Go ist in dieser Session nicht installiert!

---

## 🚀 Nächste Session: START HERE

### 1. Go installieren (falls nötig)
```bash
# Check ob Go verfügbar
go version

# Falls nicht: Installation anstoßen
```

### 2. Tests ausführen
```bash
cd /workspace/vikunja-tui
go test -v ./internal/vikunja/
```

**Erwartung:** Alle 8 Tests grün ✅

### 3. Falls Tests grün → Nächste Phase wählen

**Option A:** Phase 2 (Error-Handling)
- Error-Footer (rot)
- Error-Log (`:errors`)
- Bessere UX bei Fehlern

**Option B:** Phase 3 (Commands)
- `:q`, `:qa`, `:h`
- Command-Registry

**User fragen:** Welche Phase als nächstes?

---

## 📐 Aktueller Stand

- **Fortschritt:** 4/10 (40%)
- **Branch:** main
- **Letzte Phase:** 7.1 (Store Tests)
- **Code:** Funktioniert, aber Tests ungeprüft

---

## 📚 Wichtige Dateien

- `STATUS.md` - Aktueller Fortschritt (gerade updated)
- `SPEC_FUNDAMENT.md` - Komplette Spec
- `AGENTS.md` - Arbeitsregeln für Agents
- `internal/vikunja/sqlite_store_test.go` - **NEU!**

---

**REMEMBER:** Nur auf explizite Anweisung umsetzen, sonst Vorschläge!
