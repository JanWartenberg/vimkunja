# Vikunja TUI - Status

**Aktualisiert:** 2026-07-10  
**Branch:** main (`53efec1`)  
**Phase:** Fundament (2/10 Kriterien ✅)

---

## 🎯 Aktueller Stand

### ✅ Was funktioniert:
- Projekte-Liste anzeigen
- Tasks pro Projekt anzeigen
- Task-Details (Enter auf Task)
- Vim-Navigation (hjkl, gg, G, Ctrl+D/U)
- Pane-Wechsel (Ctrl+W, Ctrl+W h/l/j/k)
- Task done togglen (`x`)
- Farbige Priority-Anzeige (rot=urgent, grün=low)
- SQLite-only (kein REST Client mehr)
- **`updated` timestamp wird gebumpt** ✅
- **Task-Bucket Move bei Toggle** ✅ (Phase 6.2)

### ❌ Was fehlt noch:
- `:q`, `:qa`, `:h` Commands
- Error-Log System
- Help-Pane (`?`)
- Leader-Key (`Space+o` für Pane-Collapse)
- Auto-Refresh nach Writes
- Tests (Store + TUI Model)

---

## 📋 Nächste Schritte

### Phase 7.1 - Store Tests (Empfohlen) (~1-2h)
- Testet: Timestamp bump, Bucket move, Queries
- Sichert Korrektheit ab
- TDD: Erst Tests, dann weitere Features

---

### Option B: User-Facing Features (Schneller Fortschritt)
**Phase 2 - Error Handling** (1-2h)
- Roter Footer bei Fehlern
- Error-Log (`:errors` Command)
- Verhindert: Fehler verschwinden nach 2 Sek

**Phase 3 - Command System** (2-3h)
- `:q` / `:qa` / `:h` implementieren
- Command Registry für Erweiterbarkeit

**Phase 4 - Help-Pane** (1-2h)
- `?` oder `:h` zeigt Keybindings
- Macht UI discoverable

**Phase 5 - Leader-Key** (1-2h)
- `Space+o` collapsed Projects-Pane
- Vorbereitung für mehr Shortcuts

---

## 📊 Fortschritt (SPEC_FUNDAMENT.md)

| Kriterium | Status |
|-----------|--------|
| `client.go` gelöscht | ✅ |
| `updated` timestamp bump | ✅ |
| Commands (`:q`, `:qa`, `:h`) | ⏳ Phase 3 |
| Help-Pane (`?`) | ⏳ Phase 4 |
| Leader-Key (`Space+o`) | ⏳ Phase 5 |
| Done-Bucket Move | ✅ |
| Auto-Refresh | ⏳ Phase 6.3 |
| Error Footer | ⏳ Phase 2 |
| Store Tests | ⏳ Phase 7 |
| TUI Model Tests | ⏳ Phase 7 |

**Progress:** 3/10 (30%)

---

## 📝 TODOs & Offene Fragen

### TODO: Description Rendering (Later)
**Problem:** Task-Descriptions sind HTML (`<p>`, `<strong>`, etc.)  
**Optionen:**
- Strip HTML (einfach, verliert Formatierung)
- HTML→Markdown (Dependency, behält Bold/Lists)
- Custom TUI Renderer (komplex, beste UX)

**Entscheidung:** Für später, erstmal wichtigere Sachen

**Details:** `.archive/TODO_UX.md`

---

### TODO: Weitere UX-Verbesserungen
- Datumsformat (relative Anzeige "in 2 days")
- Lange Titel (Truncate vs. Wrap)
- Empty Description Placeholder
- Task Identifier anzeigen

**Priorität:** P3-P5 (nach Fundament)

---

## 🗂️ Dokumentation

### Aktive Docs:
- **STATUS.md** ← Diese Datei (ALLES WICHTIGE HIER!)
- **SPEC_FUNDAMENT.md** - Komplette Spec (Referenz)

### Archiv:
- `.archive/NEXT_STEPS.md` - Detaillierte Implementierungsanleitung
- `.archive/TODO_PHASE_6.2.md` - Bucket Move Implementierung
- `.archive/TODO_UX.md` - UX-Verbesserungen
- `.archive/TUI_IDEA.md` - Design-Entscheidungen
- `.archive/2026-07-10/` - Fix-Berichte von heute

---

## 🔥 Letzte Änderungen

### Commit `53efec1` (Fix)
- Added `internal/vikunja/types.go` (fehlende Types)
- Fixed: Compilation errors nach REST Client Removal

### Commit `280202c` (Phase 1 + 6.1)
- Removed REST client (`client.go`)
- Fixed `updated` timestamp bump
- SQLite-only

---

## 🚀 Wie weiter?

**Empfehlung:** 
1. **Phase 6.2** (Bucket Move) - 1-2h - Komplettiert Write-Semantik
2. **Phase 7.1** (Store Tests) - 1-2h - Sichert Korrektheit
3. **Phase 2-5** (Commands, Help, etc.) - 6-10h - User-Facing Features

Oder alternativ:
- Direkt zu Phase 2-5 springen für schnellere UX-Verbesserungen
- Phase 6.2 + 7 später nachholen

**Was möchtest du als nächstes angehen?**

---

## 📐 Pattern für dieses Projekt

### Regel #1: Ein zentrales File
**STATUS.md** ist DIE zentrale Datei.  
Alles andere → `.archive/`

### Regel #2: Archivierung
Erledigte Berichte, alte TODOs → `.archive/YYYY-MM-DD/`  
Design-Docs → `.archive/`

### Regel #3: Commits
**Vor jedem Commit:**
1. `go build` testen
2. Diff zeigen
3. **User fragen: "Soll ich committen?"**
4. Auf Bestätigung warten
5. Dann committen

---

**Fragen? → Schau in STATUS.md (diese Datei)**  
**Details? → `.archive/` oder SPEC_FUNDAMENT.md**
