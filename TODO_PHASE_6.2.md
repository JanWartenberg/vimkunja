# TODO: Phase 6.2 - Done-Bucket Move (DEFERRED)

**Status:** Vorbereitet für spätere Implementierung  
**Abhängigkeit:** Phase 6.1 ✅ abgeschlossen (updated timestamp wird gebumpt)  
**Priorität:** HIGH (notwendig für korrekte Kanban-View)

---

## Problem

Aktuell bumpt `ToggleTaskDone()` nur den `updated` timestamp. Laut SPEC_FUNDAMENT.md (Decision #2) muss beim Toggle zu `done=true` der Task auch in den `done_bucket_id` verschoben werden (falls vorhanden).

**Ohne diesen Fix:**
- Kanban-Boards zeigen done Tasks im falschen Bucket
- DB-State ist inkonsistent mit Vikunja-Server Verhalten

---

## Implementierung

### Datei: `internal/vikunja/sqlite_store.go`

In der Funktion `ToggleTaskDone()`, **NACH** dem `UPDATE tasks`:

```go
func (s *SQLiteStore) ToggleTaskDone(ctx context.Context, taskID int64, done bool) error {
    // ... existing code ...
    
    // CRITICAL: Always bump 'updated' timestamp (Decision #2, SPEC_FUNDAMENT.md)
    q := `UPDATE tasks SET done = ?, done_at = ?, updated = ? WHERE id = ?`
    if _, err := s.db.ExecContext(ctx, q, boolToInt(done), doneAt, time.Now(), taskID); err != nil {
        return fmt.Errorf("toggle task done: %w", err)
    }
    
    // ========== PHASE 6.2: BUCKET MOVE LOGIC (ADD BELOW) ==========
    
    // Step 1: Get the done_bucket_id for this task's project view
    var doneBucketID sql.NullInt64
    qDoneBucket := `
        SELECT pv.done_bucket_id
        FROM project_views pv
        JOIN tasks t ON t.project_id = pv.project_id
        WHERE t.id = ?
        LIMIT 1
    `
    if err := s.db.QueryRowContext(ctx, qDoneBucket, taskID).Scan(&doneBucketID); err != nil {
        // If no view or no done_bucket_id configured, skip bucket move
        if err == sql.ErrNoRows {
            return nil // No view configured, that's OK
        }
        return fmt.Errorf("query done_bucket_id: %w", err)
    }
    
    // Step 2: Move task to done bucket (if toggling to done AND done_bucket_id exists)
    if done && doneBucketID.Valid {
        qMoveBucket := `
            UPDATE task_buckets
            SET bucket_id = ?
            WHERE task_id = ?
        `
        if _, err := s.db.ExecContext(ctx, qMoveBucket, doneBucketID.Int64, taskID); err != nil {
            return fmt.Errorf("move task to done bucket: %w", err)
        }
    }
    
    // TODO (Future): When toggling to done=false, should we move task back to original bucket?
    // Decision: Leave task in current bucket for now (user can manually move).
    
    return nil
}
```

---

## Test Cases (für Phase 7)

### Test 1: Task moves to done_bucket when toggled done
```go
func TestToggleTaskDone_MovesToDoneBucket(t *testing.T) {
    store := setupTestDB(t)
    
    // Seed: Project → View with done_bucket_id → Task in bucket_1
    projectID := seedProject(t, store, "Test Project")
    bucketID := seedBucket(t, store, projectID, "In Progress")
    doneBucketID := seedBucket(t, store, projectID, "Done")
    viewID := seedView(t, store, projectID, "kanban", doneBucketID)
    taskID := seedTask(t, store, projectID, "Test Task", false)
    assignTaskToBucket(t, store, taskID, bucketID, viewID)
    
    // Act: Toggle to done
    err := store.ToggleTaskDone(context.Background(), taskID, true)
    require.NoError(t, err)
    
    // Assert: Task is now in done_bucket_id
    var currentBucketID int64
    err = store.db.QueryRow("SELECT bucket_id FROM task_buckets WHERE task_id = ?", taskID).Scan(&currentBucketID)
    require.NoError(t, err)
    assert.Equal(t, doneBucketID, currentBucketID)
}
```

### Test 2: Task stays in bucket if no done_bucket_id configured
```go
func TestToggleTaskDone_NoDoneBucket_NoMove(t *testing.T) {
    store := setupTestDB(t)
    
    // Seed: Project → View WITHOUT done_bucket_id → Task in bucket_1
    projectID := seedProject(t, store, "Test Project")
    bucketID := seedBucket(t, store, projectID, "Backlog")
    viewID := seedView(t, store, projectID, "kanban", nil) // No done_bucket_id
    taskID := seedTask(t, store, projectID, "Test Task", false)
    assignTaskToBucket(t, store, taskID, bucketID, viewID)
    
    // Act: Toggle to done
    err := store.ToggleTaskDone(context.Background(), taskID, true)
    require.NoError(t, err)
    
    // Assert: Task is STILL in original bucket (no move happened)
    var currentBucketID int64
    err = store.db.QueryRow("SELECT bucket_id FROM task_buckets WHERE task_id = ?", taskID).Scan(&currentBucketID)
    require.NoError(t, err)
    assert.Equal(t, bucketID, currentBucketID)
}
```

### Test 3: Task done flag is set correctly
```go
func TestToggleTaskDone_SetsUpdatedTimestamp(t *testing.T) {
    store := setupTestDB(t)
    
    projectID := seedProject(t, store, "Test Project")
    taskID := seedTask(t, store, projectID, "Test Task", false)
    
    // Get initial updated timestamp
    var initialUpdated time.Time
    err := store.db.QueryRow("SELECT updated FROM tasks WHERE id = ?", taskID).Scan(&initialUpdated)
    require.NoError(t, err)
    
    time.Sleep(10 * time.Millisecond) // Ensure timestamp differs
    
    // Act: Toggle done
    err = store.ToggleTaskDone(context.Background(), taskID, true)
    require.NoError(t, err)
    
    // Assert: Updated timestamp is newer
    var newUpdated time.Time
    err = store.db.QueryRow("SELECT updated FROM tasks WHERE id = ?", taskID).Scan(&newUpdated)
    require.NoError(t, err)
    assert.True(t, newUpdated.After(initialUpdated))
}
```

---

## Migration Path

1. **Aktuell (Phase 6.1):** ✅ `updated` timestamp wird gebumpt
2. **Nächster Schritt (Phase 6.2):** Bucket-Move Logik hinzufügen (siehe Code oben)
3. **Dann (Phase 7.1):** Tests schreiben (siehe Test Cases oben)

---

## Offene Fragen

1. **Was passiert bei `done=false` (Task wird wieder geöffnet)?**
   - Option A: Task bleibt im done_bucket (User kann manuell verschieben)
   - Option B: Task geht zurück in ursprünglichen Bucket (benötigt Tracking)
   - **Empfehlung:** Option A (weniger Komplexität, konsistent mit Vikunja-Server)

2. **Was wenn ein Task in mehreren Views ist?**
   - Aktuell: Query nimmt erste View (`LIMIT 1`)
   - **Annahme:** Tasks haben primäre View (project_views.project_id match)

3. **Was wenn task_buckets Row nicht existiert?**
   - `UPDATE` schlägt fehl → Error zurückgeben
   - **Alternative:** `INSERT OR REPLACE` (robuster?)

---

## Success Criteria

Phase 6.2 ist **DONE** wenn:
- ✅ `ToggleTaskDone(done=true)` verschiebt Task in `done_bucket_id` (falls vorhanden)
- ✅ `ToggleTaskDone(done=true)` macht NICHTS wenn kein `done_bucket_id` konfiguriert
- ✅ `ToggleTaskDone(done=false)` lässt Task im aktuellen Bucket
- ✅ Alle 3 Test Cases (siehe oben) sind grün
- ✅ Manuelle Verifikation: Kanban-View zeigt Tasks im korrekten Bucket

---

**Next:** Siehe NEXT_STEPS.md für Phase 2-5 (Error Handling, Commands, Help, Leader-Key)
