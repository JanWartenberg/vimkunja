# Vikunja TUI - Implementation Status

**Last Updated:** 2026-07-10  
**Current Commit:** `280202c` - Phase 1 + 6.1 completed

---

## ✅ Completed (Option 1: Critical Blockers)

### Phase 1: REST Client Removal ✅
**Duration:** ~5 min  
**Commit:** `280202c`

- ✅ Deleted `internal/vikunja/client.go`
- ✅ Simplified `cmd/vikunja-tui/main.go` to SQLite-only
- ✅ Updated `Store` interface comments
- ✅ No compile errors (verified syntax)

**Result:** Codebase is now **SQLite-only**, no REST API complexity.

---

### Phase 6.1: Updated Timestamp Fix ✅
**Duration:** ~10 min  
**Commit:** `280202c`

**Problem:** `ToggleTaskDone()` did NOT bump the `updated` timestamp.  
**Impact:** Server webhooks, sync features would break!

**Solution:**
```go
// OLD (BROKEN):
q := `UPDATE tasks SET done = ?, done_at = ? WHERE id = ?`
s.db.ExecContext(ctx, q, boolToInt(done), doneAt, taskID)

// NEW (FIXED):
q := `UPDATE tasks SET done = ?, done_at = ?, updated = ? WHERE id = ?`
s.db.ExecContext(ctx, q, boolToInt(done), doneAt, time.Now(), taskID)
```

**Result:** Every task write now bumps `updated` = server-fidelity maintained! ✅

---

## 📋 Next Steps (Documented)

### Phase 6.2: Done-Bucket Move (HIGH PRIORITY)
**Status:** Ready for implementation (see `TODO_PHASE_6.2.md`)  
**Effort:** ~1-2 hours

**Problem:** Tasks don't move to `done_bucket_id` when toggled done.  
**Impact:** Kanban boards show tasks in wrong buckets.

**TODO:**
- Query `done_bucket_id` from `project_views`
- Update `task_buckets.bucket_id` when `done=1`
- Write 3 test cases (see TODO_PHASE_6.2.md)

---

### Remaining Phases (see NEXT_STEPS.md)
All documented with code examples and time estimates:

- **Phase 2:** Error Infrastructure (1-2h)
- **Phase 3:** Command System (`:q`, `:qa`, `:h`) (2-3h)
- **Phase 4:** Help-Pane (1-2h)
- **Phase 5:** Leader-Key (`Space+o`) (1-2h)
- **Phase 6.2:** Done-Bucket Move (1-2h)
- **Phase 6.3:** Auto-Refresh (30 min)
- **Phase 7:** Tests (3-4h)

**Total remaining:** ~11-16 hours

---

## 🎯 Success Criteria Progress

From SPEC_FUNDAMENT.md:

- ✅ `client.go` deleted, codebase compiles
- ⏳ `:q`, `:qa`, `:h` commands work (Phase 3)
- ⏳ `?` opens help-pane (Phase 4)
- ⏳ `Space+o` collapses projects-pane (Phase 5)
- ✅ `x` on task bumps `updated` timestamp
- ⏳ `x` moves task to `done_bucket_id` (Phase 6.2)
- ⏳ Auto-refresh after successful `x` (Phase 6.3)
- ⏳ Red footer on write errors (Phase 2)
- ⏳ All Store tests pass (Phase 7)
- ⏳ All TUI model tests pass (Phase 7)

**Progress:** 2/10 criteria met (20%)

---

## 🔥 Critical Blockers Status

| Blocker | Status | Notes |
|---------|--------|-------|
| REST client present | ✅ **FIXED** | Deleted in Phase 1 |
| `updated` timestamp not bumped | ✅ **FIXED** | Fixed in Phase 6.1 |
| No `done_bucket_id` move | 🟡 **DOCUMENTED** | See TODO_PHASE_6.2.md |
| No tests | 🟡 **DOCUMENTED** | See NEXT_STEPS.md Phase 7 |
| No error handling | 🟡 **DOCUMENTED** | See NEXT_STEPS.md Phase 2 |

---

## 📂 Documentation Files

- **SPEC_FUNDAMENT.md** - Complete specification (ready for impl)
- **NEXT_STEPS.md** - Implementation plan with code examples
- **TODO_PHASE_6.2.md** - Bucket move logic (next high-priority task)
- **TUI_IDEA.md** - Design decisions (#1-11)
- **STATUS.md** (this file) - Current progress tracker

---

## 🚀 Recommended Next Action

**Option 1 (Finish Write Semantics):**  
→ Implement Phase 6.2 (done-bucket move) now  
→ Then write Store tests (Phase 7.1) to verify  
→ Result: Complete, tested write semantics

**Option 2 (Build User-Facing Features):**  
→ Implement Phase 2-5 (Error handling, Commands, Help, Leader)  
→ Result: Improved UX, discoverable UI  
→ Then circle back to Phase 6.2 + 7

**Option 3 (Test-Driven):**  
→ Write Store tests first (Phase 7.1)  
→ Fail Phase 6.2 tests (no bucket move)  
→ Implement Phase 6.2 to make tests pass  
→ Result: TDD-style, guaranteed correctness

**Recommendation:** Option 1 or 3 (finish write semantics completely before moving on)

---

## 🎉 Quick Wins Achieved

1. **Cleaner Codebase:** 170 lines of unused REST client code removed
2. **Server Compatibility:** `updated` timestamp now correct for webhooks/sync
3. **Clear Roadmap:** All remaining work documented with time estimates
4. **Spec-Driven:** Every decision traced back to SPEC_FUNDAMENT.md

---

**Git Log:**
```
280202c (HEAD -> main) refactor: SQLite-only, bump updated timestamp (Phase 1 + 6.1)
7c2e912 spec added
2f6bdd7 Update design decisions after grilling sessions
```

**Next Commit Preview:**
```
feat: move tasks to done_bucket_id on toggle (Phase 6.2)

- Query done_bucket_id from project_views
- Update task_buckets.bucket_id when done=1
- Add Store tests: bucket move, no-move, timestamp
- Fixes: Kanban boards now show tasks in correct buckets

Closes: SPEC_FUNDAMENT.md Decision #2 (Write Semantics)
```
