# Session Notes – Vikunja TUI

**Für:** Nächste Agent-Session  
**Letzte Änderung:** 2026-07-13

---

## 🎯 Phase 7.1 - ERLEDIGT

**Ergebnis:**
- Go 1.25.0 ist unter `/usr/local/go/bin/go` installiert.
- `go.mod` ist auf Go 1.25.0 ausgerichtet.
- `go test -v ./...` und `go vet ./...` sind grün.
- 7 Store-Tests in `internal/vikunja/sqlite_store_test.go` decken Timestamp-Updates (`updated`), Bucket-Move bei `done=true`, `done_at`, Pagination, Position-Sortierung und archivierte Projekte ab.
- Der Store behandelt nullable `projects.identifier`, `projects.description` und `tasks.index` robust.

---

## 🚀 Nächste Session: START HERE

### 1. Tests ausführen
```bash
cd /workspace/vikunja-tui
go test -v ./...
go vet ./...
```

**Erwartung:** Alle 7 Store-Tests grün ✅

### 2. Nächste Phase wählen

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
- **Code:** Store-Test-Suite und Vet grün

---

## 📚 Wichtige Dateien

- `STATUS.md` - Aktueller Fortschritt (gerade updated)
- `SPEC_FUNDAMENT.md` - Komplette Spec
- `AGENTS.md` - Arbeitsregeln für Agents
- `internal/vikunja/sqlite_store_test.go` - **NEU!**

---

**REMEMBER:** Nur auf explizite Anweisung umsetzen, sonst Vorschläge!
