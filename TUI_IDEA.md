# Vikunja TUI (MVP) — Vim-Motions & Pane Layout

## Ziel
Ein Terminal-Interface für Vikunja, das sich **wie vi/vim** bedienen lässt:
- Navigation primär via **Vim-Motions** (`j/k`, `gg/G`, `enter`, …)
- **Multi-Pane Handling** wie Vim-Splits
- MVP bildet erstmal **nur eine Task-Ansicht** ab: `ProjectViewKind = list`

## Daten-/Feature-Scope (MVP)
- **Links:** Projekte auswählen
- **Rechts:** Tasks für die ausgewählte Projekt-View anzeigen
- View-Auswahl: Für ein Projekt wird automatisch die **erste** View mit `view_kind == "list"` genommen.
- Task-CRUD/Comments/Edit sind später — MVP ist “read + navigation”, plus optional “Refresh”.

## Pane-Konzept (Vim-Splits)
Es gibt (zunächst) 2 Panes:
- `projects` (links)
- `tasks` (rechts)

### Fokus-Commands
Wie in Vim:
- `Ctrl+W` (als Leader) + `Ctrl+W`  → Fokus togglen/cycle
- `Ctrl+W` + `h/j/k/l`              → Fokus auf das Nachbar-Pane in diese Richtung

> In unserem MVP existieren nur links/rechts, `j/k` werden aktuell zu “no-op” oder auf die gleiche Zeile gemappt.

### Beispiel-Flow
- `Ctrl+W h` → Fokus auf Projekte
- `j/k` → Projekt wechseln
- `Ctrl+W l` → Fokus auf Tasks
- `j/k` → Tasks scrollen
- `r` → Refresh (projekte+tasks reload)

## Erweiterbarkeit (Design)
Obwohl MVP nur `list`-View rendert, wird die Struktur so gebaut, dass später leicht hinzugefügt werden kann:
- Weitere `ProjectViewKind`-Handler (`kanban`, `table`, …)
- Task-Details-Pane (z.B. “Enter öffnet Detail unter dem aktuellen Pane Layout”)
- Compose von mehreren Panes in einer Art “Pane-Grid” (Nachbarn via Koordinaten oder Graph)

## Technische Entscheidung
- **Sprache:** Go
- **UI Framework:** `bubbletea` + `lipgloss`
- **HTTP/Client:** eigener lightweight Client Wrapper auf Vikunjas REST API (`/api/v2`)

## Rendering (MVP)
- Linke Spalte:
  - Überschrift: `Projects`
  - Liste: `> <identifier or title>`
- Rechte Spalte:
  - Überschrift: `Tasks (list)`
  - Liste: `[%] <priority> <title> (due)`
  - Done: Checkbox / Marker

## Nächste Schritte nach MVP
1. Task-Detail-View (Enter)
2. Kommentar posten (a)
3. Edit Felder (Title/Description/Priority/Due)
4. `kanban` ViewKind: Buckets + MOVE zwischen Buckets (Vim-friendly)
