# Fundament/MVP – abgeschlossen

**Abgeschlossen:** 2026-07-17

Vimkunjas SQLite-basiertes Fundament wurde automatisiert und manuell verifiziert. Es liefert Projects und List View, einen vollständigen Task-Editor, Workflow-Status über Vikunja-Buckets, konsistente Done-/Default-Bucket-Semantik, Vim-artige Pane-Navigation, Ex-Commands, Help, Fehlerlog und Leader-Key-Unterstützung.

## Verifikation

- `go test ./...` grün
- Fundament-Smoke-Test bestanden
- Workflow-Status gegen die Original-App geprüft

## Archivinhalt

- `fundament-mvp/SPEC_FUNDAMENT.md` – abgeschlossene Fundament-Spec
- `fundament-mvp/tickets/` – abgeschlossene Fundament-Tickets
- `fundament-mvp/workflow-status/` – abgeschlossene Workflow-Status-Spec und Tickets

Offene Arbeit gehört ausschließlich nach `.scratch/`; diese Archivdateien sind keine aktive Roadmap.
