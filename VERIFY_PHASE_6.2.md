# Verifikation: Phase 6.2 - Done-Bucket Move

## 🧪 So testest du, ob es funktioniert:

### Voraussetzung:
Deine Vikunja-Datenbank muss eine **Kanban-View** mit `done_bucket_id` haben.

---

## Test 1: Bucket-ID VOR dem Toggle prüfen

**Schritt 1:** Finde eine Task-ID
```bash
sqlite3 internal/db/vikunja.db "
SELECT id, title, done FROM tasks 
WHERE done = 0 
LIMIT 5;
"
```

Notiere dir eine `task_id`, z.B. `42`

**Schritt 2:** Prüfe in welchem Bucket der Task ist
```bash
sqlite3 internal/db/vikunja.db "
SELECT tb.task_id, tb.bucket_id, b.title as bucket_name
FROM task_buckets tb
JOIN buckets b ON b.id = tb.bucket_id
WHERE tb.task_id = 42;
"
```

Beispiel-Output:
```
42|5|In Progress
```
→ Task 42 ist in Bucket 5 ("In Progress")

**Schritt 3:** Prüfe welcher Bucket der Done-Bucket ist
```bash
sqlite3 internal/db/vikunja.db "
SELECT pv.done_bucket_id, b.title
FROM project_views pv
JOIN buckets b ON b.id = pv.done_bucket_id
JOIN tasks t ON t.project_id = pv.project_id
WHERE t.id = 42;
"
```

Beispiel-Output:
```
7|Done
```
→ Done-Bucket ist ID 7 ("Done")

---

## Test 2: Task im TUI togglen

**Schritt 4:** Starte die App
```bash
go run ./cmd/vikunja-tui --db internal/db/vikunja.db
```

**Schritt 5:** Navigiere zum Task (ID 42 aus Schritt 1)
- Mit `hjkl` zum richtigen Projekt/Task
- Drücke `x` um den Task auf "done" zu togglen
- Task sollte jetzt durchgestrichen/grau sein

---

## Test 3: Bucket-ID NACH dem Toggle prüfen

**Schritt 6:** Prüfe ob Task in Done-Bucket verschoben wurde
```bash
sqlite3 internal/db/vikunja.db "
SELECT tb.task_id, tb.bucket_id, b.title as bucket_name
FROM task_buckets tb
JOIN buckets b ON b.id = tb.bucket_id
WHERE tb.task_id = 42;
"
```

**Erwartetes Ergebnis:**
```
42|7|Done
```
→ ✅ Task ist jetzt in Bucket 7 ("Done")!

**Vorher:** `42|5|In Progress`  
**Nachher:** `42|7|Done`

---

## Test 4: Timestamp wurde gebumpt

```bash
sqlite3 internal/db/vikunja.db "
SELECT id, title, done, datetime(updated) as updated_at
FROM tasks
WHERE id = 42;
"
```

**Erwartetes Ergebnis:**
```
42|My Task Title|1|2026-07-10 14:30:15
```
→ `updated` ist aktuelles Datum/Zeit (frisch gebumpt)

---

## ✅ Success Criteria

Phase 6.2 ist erfolgreich wenn:

1. ✅ Task `done` Flag ist auf `1`
2. ✅ Task `updated` Timestamp ist aktuell
3. ✅ Task `bucket_id` ist jetzt `done_bucket_id` (aus project_views)
4. ✅ Keine Fehler beim Togglen

---

## 🔍 Wenn es NICHT klappt:

### Fall 1: Task bleibt im alten Bucket
**Mögliche Ursachen:**
- Projekt hat keine View mit `done_bucket_id` konfiguriert
- Query findet die View nicht
- `task_buckets` Table hat keinen Eintrag für diese Task

**Debug:**
```bash
# Prüfe ob done_bucket_id NULL ist:
sqlite3 internal/db/vikunja.db "
SELECT pv.id, pv.project_id, pv.done_bucket_id 
FROM project_views pv
JOIN tasks t ON t.project_id = pv.project_id
WHERE t.id = 42;
"
```

Wenn `done_bucket_id` = NULL → Das ist OK, dann wird nicht verschoben!

---

### Fall 2: Fehler beim Toggle
**Check TUI Error Footer:**
- Roter Footer erscheint? → Fehler ist geloggt
- Kein Error-System? → Siehe Console-Output (`go run`)

**Debug:**
```bash
# Prüfe ob task_buckets Eintrag existiert:
sqlite3 internal/db/vikunja.db "
SELECT * FROM task_buckets WHERE task_id = 42;
"
```

Wenn leer → Task hat keinen Bucket-Eintrag (Vikunja-DB könnte inkonsistent sein)

---

## 🎯 Schnelltest (ohne DB-Queries)

**Visuell im TUI:**

1. **Kanban-View öffnen** (falls du eine hast):
   - Task sollte nach Toggle in "Done"-Spalte erscheinen
   - (Kanban-View ist noch nicht implementiert, aber die DB ist vorbereitet)

2. **Nach Restart prüfen:**
   - App schließen (`:qa`)
   - App neu starten
   - Task sollte immer noch "done" sein
   - Bucket-Änderung ist persistent

---

## 📊 Zusammenfassung

**Was die Änderung macht:**
```
VORHER:
1. User drückt 'x'
2. Task.done = 1
3. Task.updated = NOW()
4. ❌ Bucket bleibt "In Progress"

NACHHER:
1. User drückt 'x'
2. Task.done = 1
3. Task.updated = NOW()
4. ✅ Query done_bucket_id (z.B. 7)
5. ✅ UPDATE task_buckets SET bucket_id = 7
6. ✅ Task ist jetzt im "Done" Bucket
```

**Warum wichtig:**
- Kanban-Boards zeigen Tasks im richtigen Bucket
- Vikunja Web-UI sieht konsistenten State
- Server-seitige Automation funktioniert (Webhooks für "moved to Done")

---

**Implementiert:** 2026-07-10  
**Code:** `internal/vikunja/sqlite_store.go:215-263`  
**Spec:** SPEC_FUNDAMENT.md - Decision #2 (Write Semantics)
