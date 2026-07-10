# Vikunja TUI - Nächste Schritte zur Implementierung

**Status:** SPEC_FUNDAMENT.md ist fertig und wartet auf Implementierung  
**Aktueller Stand:** Basis-PoC existiert, aber mehrere kritische Fundament-Features fehlen noch

---

## 📊 Aktueller Implementierungsstand

### ✅ Bereits vorhanden (PoC):
- ✅ SQLite Store mit Basis-Funktionen (ListProjects, ListTasks, ListProjectViews)
- ✅ Bubbletea TUI mit Pane-System (Projects, Tasks, Detail)
- ✅ Vim-Keybindings (hjkl, gg, G, Ctrl+D/U)
- ✅ Pane-Navigation mit Ctrl+W, Ctrl+W h/l/j/k
- ✅ Task Toggle mit `x` (Basis-Implementierung)
- ✅ Detail-Pane als Full-Pane (nicht Modal)

### ❌ Noch fehlend (gemäß SPEC):
- ❌ REST client (`client.go`) noch nicht gelöscht (Decision #1)
- ❌ Command-Line Infrastruktur (`:q`, `:qa`, `:h`) fehlt (Decision #6)
- ❌ Error-Handling System (3-Tier: Log, Footer, Auto-Refresh) fehlt (Decision #9)
- ❌ Help-Pane fehlt (Decision #10)
- ❌ Leader-Key Support (`Space` + 500ms timeout) fehlt (Decision #13)
- ❌ Projects-Pane Collapse (`Space+o`) fehlt (Decision #13)
- ❌ **KRITISCH:** `ToggleTaskDone` bumpt `updated` timestamp NICHT (Decision #2)
- ❌ **KRITISCH:** `ToggleTaskDone` verschiebt Tasks NICHT in `done_bucket_id` (Decision #2)
- ❌ Auto-Refresh nach erfolgreichen Writes fehlt (Decision #3)
- ❌ **Keine Tests vorhanden** (weder Store noch TUI Model)

---

## 🎯 Implementierungsplan (nach Spec Phase 1-7)

### **Phase 1: Cleanup** ⚡ (Priorität: P0, Aufwand: 5 Min)
**Ziel:** REST Client entfernen

- [ ] `internal/vikunja/client.go` löschen
- [ ] Kompilieren prüfen (`go build ./cmd/vikunja-tui`)
- [ ] Commit: `"refactor: remove REST client (Decision #1)"`

**Rationale:** Decision #1 - SQLite-only, 1-User Setup

---

### **Phase 2: Error Infrastructure** 🔴 (Priorität: P0, Aufwand: 1-2h)
**Ziel:** 3-Tier Error-Handling implementieren

#### 2.1 Error Log Package
- [ ] `internal/errors/log.go` erstellen:
  ```go
  package errors
  
  type LogEntry struct {
      Time    time.Time
      Message string
  }
  
  var globalLog []LogEntry
  var mu sync.Mutex
  
  func Log(msg string) {
      mu.Lock()
      defer mu.Unlock()
      globalLog = append(globalLog, LogEntry{Time: time.Now(), Message: msg})
  }
  
  func GetAll() []LogEntry {
      mu.Lock()
      defer mu.Unlock()
      return append([]LogEntry{}, globalLog...)
  }
  
  func Clear() {
      mu.Lock()
      defer mu.Unlock()
      globalLog = nil
  }
  ```

#### 2.2 Model Error State
- [ ] In `model.go` hinzufügen:
  ```go
  type model struct {
      // ... existing fields ...
      lastErr      string
      toast        string
      toastUntil   time.Time
      errorDismissed bool
  }
  ```

#### 2.3 Footer Rendering
- [ ] `renderStatusBar()` erweitern:
  - Rote Hintergrundfarbe bei `lastErr != ""`
  - Grüner Toast für 2-3 Sek bei erfolgreichen Writes
  - Error bleibt bis Tastendruck sichtbar

#### 2.4 Integration in Update()
- [ ] Nach jedem `ToggleTaskDone`:
  - Bei Fehler: `errors.Log()` + `model.lastErr` setzen
  - Bei Erfolg: `model.toast = "✓ Task updated"` + Timer

**Test:** Fehlerhaftes Update durchführen → Roter Footer + Error geloggt

---

### **Phase 3: Command Infrastructure** 💻 (Priorität: P0, Aufwand: 2-3h)
**Ziel:** `:q`, `:qa`, `:h` Befehle funktionieren

#### 3.1 Command Registry
- [ ] `internal/tui/commands.go` erstellen:
  ```go
  package tui
  
  type CommandHandler func(m *model, args []string) (tea.Model, tea.Cmd)
  
  type CommandRegistry struct {
      handlers map[string]CommandHandler
      aliases  map[string]string
  }
  
  func NewCommandRegistry() *CommandRegistry {
      r := &CommandRegistry{
          handlers: make(map[string]CommandHandler),
          aliases:  make(map[string]string),
      }
      
      // Register MVP commands
      r.Register("q", []string{"quit"}, cmdQuit)
      r.Register("qa", []string{"quitall", "qall"}, cmdQuitAll)
      r.Register("h", []string{"help"}, cmdHelp)
      r.Register("errors", nil, cmdErrors)
      
      return r
  }
  
  func (r *CommandRegistry) Execute(m *model, input string) (tea.Model, tea.Cmd, error)
  ```

#### 3.2 Command Handlers
- [ ] `cmdQuit` implementieren: Schließt aktuelles Pane oder quittet App
- [ ] `cmdQuitAll` implementieren: Sofort `tea.Quit`
- [ ] `cmdHelp` implementieren: Öffnet Help-Pane (siehe Phase 4)
- [ ] `cmdErrors` implementieren: Placeholder "Not implemented" (vorerst)

#### 3.3 Command Mode in Model
- [ ] In `model.go` hinzufügen:
  ```go
  type model struct {
      // ... existing fields ...
      commandMode   bool
      commandBuffer string
      commandReg    *CommandRegistry
  }
  ```

#### 3.4 Update() Integration
- [ ] `:` Taste → `commandMode = true`, Footer zeigt `> :_`
- [ ] In Command Mode: Zeichen zu `commandBuffer` hinzufügen
- [ ] `Enter` → `commandReg.Execute()` aufrufen
- [ ] `Esc` → Command Mode verlassen

**Test:** `:qa` → App quittet, `:q` → Pane schließt

---

### **Phase 4: Help-Pane** 📖 (Priorität: P0, Aufwand: 1-2h)
**Ziel:** Help-System verfügbar via `:h` oder `?`

#### 4.1 Help Content
- [ ] `internal/tui/help.go` erstellen:
  ```go
  package tui
  
  const helpText = `
  Vikunja TUI - Quick Reference
  ═════════════════════════════
  
  NAVIGATION
    hjkl          Move cursor (Vim style)
    gg            Jump to top
    G             Jump to bottom
    Ctrl+D/U      Scroll half-page down/up
    Enter         Open selected item
    Esc           Close pane
  
  PANE CONTROL
    Ctrl+W        Cycle pane focus
    Ctrl+W h/l    Move focus left/right
    Ctrl+W j/k    Move focus down/up
    Space+o       Toggle projects pane
  
  TASK ACTIONS
    x             Toggle task done/undone
    r             Refresh current view
  
  COMMANDS
    :q            Close pane (or quit)
    :qa           Quit app
    :h or ?       Show this help
    :errors       Show error log
  
  ... (scrollable content below) ...
  `
  
  type helpPane struct {
      scrollOffset int
      content      []string
  }
  
  func newHelpPane() *helpPane {
      return &helpPane{
          content: strings.Split(helpText, "\n"),
      }
  }
  
  func (h *helpPane) Render(width, height int) string
  func (h *helpPane) Update(msg tea.Msg) tea.Cmd
  ```

#### 4.2 Help-Pane als Pane-Typ
- [ ] In `model.go` neuen `paneID` hinzufügen: `paneHelp`
- [ ] `showHelp bool` State hinzufügen
- [ ] `:h` und `?` öffnen Help-Pane
- [ ] Help-Pane mit `j/k` scrollbar machen
- [ ] `:q` oder `Esc` schließt Help-Pane

**Test:** `?` → Help erscheint, scrollbar mit `j/k`, `Esc` → zurück

---

### **Phase 5: Leader-Key System** 🎹 (Priorität: P0, Aufwand: 1-2h)
**Ziel:** `Space` als Leader-Key mit 500ms Timeout

#### 5.1 Leader State
- [ ] In `model.go` hinzufügen:
  ```go
  type model struct {
      // ... existing fields ...
      leaderArmed    bool
      leaderTimeout  time.Time
  }
  ```

#### 5.2 Leader Logic in Update()
- [ ] `Space` Taste → `leaderArmed = true`, `leaderTimeout = time.Now().Add(500ms)`
- [ ] Nächste Taste innerhalb 500ms → Dispatch Leader-Sequence
- [ ] Nach 500ms ODER nach Dispatch → `leaderArmed = false`
- [ ] Tick-Message alle 100ms prüfen ob Timeout abgelaufen

#### 5.3 Leader-Sequences (MVP)
- [ ] `Space+o` → Toggle `projectsPaneCollapsed`
- [ ] Future: `Space+f` (Filter), `Space+g` (Goto)

#### 5.4 Projects-Pane Collapse
- [ ] `model.projectsPaneCollapsed bool` hinzufügen
- [ ] In `View()` Rendering: Collapsed → Width = 0
- [ ] Auch `Ctrl+W o` als Alternative unterstützen

**Test:** `Space` drücken, dann innerhalb 500ms `o` → Projects-Pane collapsed

---

### **Phase 6: Write Semantics (KRITISCH!)** 🔥 (Priorität: P0, Aufwand: 2-3h)
**Ziel:** `ToggleTaskDone` repliziert Server-Verhalten vollständig

#### 6.1 Updated Timestamp Bump
- [ ] In `sqlite_store.go` `ToggleTaskDone()` erweitern:
  ```go
  func (s *SQLiteStore) ToggleTaskDone(ctx context.Context, taskID int64, done bool) error {
      var doneAt any
      if done {
          doneAt = time.Now()
      } else {
          doneAt = nil
      }
      
      // WICHTIG: updated timestamp immer bumpen!
      q := `UPDATE tasks SET done = ?, done_at = ?, updated = ? WHERE id = ?`
      if _, err := s.db.ExecContext(ctx, q, boolToInt(done), doneAt, time.Now(), taskID); err != nil {
          return fmt.Errorf("toggle task done: %w", err)
      }
      
      // Phase 6.2: Bucket move logic hier einfügen
      
      return nil
  }
  ```

#### 6.2 Done-Bucket Move
- [ ] Query `done_bucket_id` aus `project_views`:
  ```go
  // Nach dem UPDATE tasks, hole done_bucket_id
  var doneBucketID sql.NullInt64
  qView := `SELECT pv.done_bucket_id 
            FROM project_views pv
            JOIN tasks t ON t.project_id = pv.project_id
            WHERE t.id = ? LIMIT 1`
  s.db.QueryRowContext(ctx, qView, taskID).Scan(&doneBucketID)
  
  if done && doneBucketID.Valid {
      // Task in done_bucket verschieben
      qBucket := `UPDATE task_buckets 
                  SET bucket_id = ? 
                  WHERE task_id = ?`
      s.db.ExecContext(ctx, qBucket, doneBucketID.Int64, taskID)
  }
  ```

- [ ] Bei `done=false`: Task wieder in Default-Bucket (bucket_id aus `task_buckets` nehmen)

#### 6.3 Auto-Refresh nach Write
- [ ] In `model.Update()` nach erfolgreichem `ToggleTaskDone`:
  ```go
  case toggleDoneMsg:
      err := m.store.ToggleTaskDone(...)
      if err != nil {
          errors.Log(err.Error())
          m.lastErr = err.Error()
          return m, nil // KEIN Refresh bei Fehler
      }
      
      // Erfolg → Toast + Auto-Refresh
      m.toast = "✓ Task updated"
      m.toastUntil = time.Now().Add(2 * time.Second)
      
      return m, m.loadTasksCmd() // ← AUTO-REFRESH!
  ```

**Test:** Task togglen → `updated` timestamp geändert + Task in done_bucket verschoben

---

### **Phase 7: Tests** 🧪 (Priorität: P0, Aufwand: 3-4h)
**Ziel:** Fundament ist getestet und stabil

#### 7.1 Store Tests
- [ ] `internal/vikunja/sqlite_store_test.go` erstellen
- [ ] Helper: `seedProject()`, `seedTask()`, `seedView()`, `seedBucket()`
- [ ] Tests:
  - `TestToggleTaskDone_SetsUpdatedTimestamp`
  - `TestToggleTaskDone_MovesToDoneBucket`
  - `TestToggleTaskDone_NoDoneBucket_NoMove`
  - `TestListTasksInView_OrdersByPosition`
  - `TestListProjects_ExcludesArchived`

#### 7.2 Fake Store für TUI Tests
- [ ] `internal/tui/fake_store_test.go` erstellen:
  ```go
  type FakeStore struct {
      projects      []vikunja.Project
      tasks         []vikunja.Task
      toggleDoneErr error
  }
  
  func (f *FakeStore) ListProjects(...) ([]vikunja.Project, int, error) {
      return f.projects, len(f.projects), nil
  }
  // ... alle Store-Methoden implementieren
  ```

#### 7.3 TUI Model Tests
- [ ] `internal/tui/model_test.go` erstellen
- [ ] Tests:
  - `TestPaneFocus_CtrlW_Cycles`
  - `TestPaneFocus_CtrlWH_MovesLeft`
  - `TestVimMotion_gg_JumpsTop`
  - `TestVimMotion_G_JumpsBottom`
  - `TestCommandMode_ColonQ_QuitsPaneOrApp`
  - `TestLeaderKey_SpaceO_TogglesProjectsPane`
  - `TestLeaderKey_Timeout_Disarms`
  - `TestErrorHandling_WriteFailure_RedFooter`

**Test:** `go test ./...` → Alle Tests grün

---

## 🎯 Success Criteria (aus Spec)

Die Fundament-Implementierung ist **DONE** wenn:

- ✅ `client.go` gelöscht, Codebase kompiliert
- ✅ `:q`, `:qa`, `:h` Befehle funktionieren
- ✅ `?` öffnet Help-Pane
- ✅ `Space+o` collapsed Projects-Pane
- ✅ `x` auf Task bumpt `updated` + verschiebt in `done_bucket_id`
- ✅ Auto-Refresh nach erfolgreichem `x`
- ✅ Roter Footer bei Write-Errors, Errors geloggt in `:errors`
- ✅ Alle Store Tests bestehen (5+ Test Cases)
- ✅ Alle TUI Model Tests bestehen (8+ Test Cases)
- ✅ Manuelle Smoke-Test Checkliste durchgeführt

---

## ⚡ Empfohlene Reihenfolge

**Quick Wins zuerst:**
1. Phase 1 (Cleanup) - 5 Min
2. Phase 6.1 (Updated Timestamp) - 30 Min - **KRITISCH für Server-Kompatibilität!**
3. Phase 2 (Error Infra) - 1-2h - **Blockiert nichts, aber wichtig für Debugging**

**Dann Fundament:**
4. Phase 3 (Commands) - 2-3h
5. Phase 4 (Help) - 1-2h
6. Phase 5 (Leader) - 1-2h
7. Phase 6.2-6.3 (Bucket Move + Auto-Refresh) - 1-2h

**Absichern:**
8. Phase 7 (Tests) - 3-4h

**Geschätzter Gesamtaufwand:** 12-18 Stunden

---

## 🚨 Kritische Punkte

### ⚠️ BLOCKER:
- **`updated` timestamp nicht gebumpt** → Server-Webhooks, Sync brechen!
- **Keine `done_bucket_id` Verschiebung** → Kanban-Boards zeigen falsche Tasks!

### ⚠️ HIGH PRIORITY:
- **Keine Tests** → Refactoring gefährlich, Bugs schwer zu finden
- **Kein Error-Log** → User sehen Fehler nicht, Debugging unmöglich
- **Keine Help** → Keybindings nicht discoverable

### ℹ️ MEDIUM:
- `client.go` noch da (aber ungenutzt) → Code-Confusion
- Kein Command-System → Keine Erweiterbarkeit

---

## 📝 Notizen

### Nach Fundament (Prio 1):
Dann folgt **SPEC_KANBAN.md** mit:
- Kanban View Rendering (Decisions #12-17)
- 2D Navigation (h/l für Buckets)
- Task-Bewegung zwischen Buckets (Shift+H/L, m+digit)

### Aktuelle Git-Ref:
- Latest Commit: `7c2e912 spec added`
- Branch: Vermutlich `main` oder `master`

### Domain Glossary:
- **Pane:** UI-Sektion (Projects, Tasks, Detail, Help)
- **View:** `project_views` Datenbank-Row (Kanban, List, Table)
- **Focus:** Aktives Pane (empfängt Input)
- **Selection:** Highlightetes Item innerhalb Pane
- **Leader:** Präfix-Taste (`Space`) für Sequenzen

---

**Frage an Entwickler:**
Mit welcher Phase soll ich beginnen? Empfehlung:
1. **Phase 1 + 6.1 (Quick Fix)** - Cleanup + kritische Write-Semantik
2. **Phase 7.1 (Tests first)** - TDD-Approach, Store Tests zuerst
3. **Phase 2-5 (Feature Complete)** - Alles außer Tests implementieren

Was ist deine Präferenz?
