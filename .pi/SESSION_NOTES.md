# Session Notes - Vikunja TUI

**Für Agent:** Hier steht, wo du weitermachen sollst.

---

## 📍 Aktueller Stand (2026-07-10 Ende)

**Commit:** `ad90e1d` - Phase 6.2 complete  
**Branch:** main  
**Progress:** 3/10 Fundament-Kriterien (30%)

### ✅ Was funktioniert:
- SQLite-only (REST client entfernt)
- Task toggle (`x`) mit `updated` timestamp bump
- **Task-Bucket Move** (done → done_bucket_id) ✅ NEU!
- Vim-Navigation, Panes, Details, Priority-Farben

### ⏳ Nächster Schritt:

**Phase 7.1 - Store Tests** (~1-2h)
- Testen: ToggleTaskDone mit Timestamp + Bucket Move
- Testen: ListTasks, ListProjects, ListViews

**Warum jetzt wichtig:**
- Sichert Write-Semantik ab (Phase 6.1 + 6.2)
- Ermöglicht sicheres Refactoring
- Verhindert Regressions

---

## 📝 Implementierungs-Details

### Phase 6.2 Code Location:
- **Datei:** `internal/vikunja/sqlite_store.go:215-263`
- **Logik:** 
  1. Query `done_bucket_id` aus `project_views`
  2. Wenn `done=1` UND `done_bucket_id` existiert → UPDATE `task_buckets`
  3. Wenn `done=0` → Task bleibt im aktuellen Bucket

### Verification:
- User hat Done-Toggle getestet → funktioniert
- Bucket-Move in DB müsste man mit SQL prüfen
- Anleitung: `.archive/2026-07-10/VERIFY_PHASE_6.2.md`

---

## 🗂️ Nächste Schritte (Priorisiert)

### 1. Phase 7.1 - Store Tests (NEXT!)
**Aufwand:** 1-2h  
**Files:**
- `internal/vikunja/sqlite_store_test.go` (NEU)
- Test-Helper: `seedProject()`, `seedTask()`, `seedView()`, `seedBucket()`

**Test Cases:**
```go
TestToggleTaskDone_SetsUpdatedTimestamp
TestToggleTaskDone_MovesToDoneBucket
TestToggleTaskDone_NoDoneBucket_NoMove
TestListTasksInView_OrdersByPosition
TestListProjects_ExcludesArchived
```

**Pattern:** Real SQLite `:memory:` DB, kein Mocking auf Store-Level.

**Reference:** `.archive/NEXT_STEPS.md` Phase 7 (Zeilen 200-280)

---

### 2. Phase 2 - Error Handling
**Aufwand:** 1-2h  
**Was:** 
- `internal/errors/log.go` (Error Logger)
- Red footer bei Fehlern
- `:errors` Command (später in Phase 3)

**Warum:** Aktuell verschwinden Fehler nach 2 Sek, User sieht nichts.

---

### 3. Phase 3-5 - Commands, Help, Leader-Key
**Aufwand:** 5-7h  
**Was:**
- `:q`, `:qa`, `:h` Commands
- Help-Pane mit Keybindings
- `Space+o` für Pane-Collapse

**Warum:** UX-Verbesserungen, Discoverable UI

---

### 4. Phase 6.3 - Auto-Refresh
**Aufwand:** 30 min  
**Was:** Nach erfolgreichem Write → `loadTasks()` aufrufen

---

## 🔥 Offene TODOs (Niedriger Prio)

### Description Rendering (Later)
**Problem:** HTML in Task-Description (`<p>`, `<strong>`)  
**Options:** Strip HTML, HTML→Markdown, Custom Renderer  
**Decision:** Für später (nach Fundament)  
**Details:** `.archive/TODO_UX.md`

### Weitere UX
- Datumsformat (relative "in 2 days")
- Lange Titel (Truncate/Wrap)
- Empty Description Placeholder

---

## 📚 Wichtige Files

### Aktive Docs:
- **STATUS.md** - User's single source of truth (kurz!)
- **SPEC_FUNDAMENT.md** - Komplette Spec (Referenz)
- **.pi/agent.md** - Agent-Regeln (Test vor Commit!)
- **.pi/SESSION_NOTES.md** - Diese Datei (für Agent)

### Archive:
- `.archive/NEXT_STEPS.md` - Detaillierte Implementierungs-Anleitung
- `.archive/TODO_PHASE_6.2.md` - Bucket Move Details
- `.archive/TODO_UX.md` - UX-Verbesserungen
- `.archive/2026-07-10/` - Fix-Berichte von heute

---

## 🎯 Lessons Learned (Diese Session)

### ✅ Was gut lief:
- Strukturierte Phasen (6.1 → 6.2)
- Klare Success Criteria
- Verification-Anleitung erstellt

### ⚠️ Was schief ging:
- Commit ohne Go-Build (Types fehlten)
- Zu viele MD-Files (User verlor Überblick)

### 🔧 Was gefixt wurde:
- Agent-Regel: IMMER User fragen vor Commit
- Dokumentation: Ein File (STATUS.md), Rest archiviert
- Pattern etabliert: Erledigtes → .archive/

---

## 🚀 Nächste Session

**Starte hier:**
1. Lies `STATUS.md` (User-Perspektive)
2. Lies diese Datei (Agent-Perspektive)
3. User fragen: "Phase 7.1 Tests oder Phase 2 Error-Handling?"
4. Implementieren nach `.archive/NEXT_STEPS.md`

**Git Status:**
```
ad90e1d (HEAD -> main) feat: move tasks to done_bucket_id on toggle
53efec1 fix: add missing type definitions (types.go)
280202c refactor: SQLite-only, bump updated timestamp
```

**Fortschritt:** 30% Fundament, dann User-Facing Features.

---

**Erstellt:** 2026-07-10 Ende Session  
**Nächstes Update:** Bei nächster Session
