# 01 — Vikunja-DB-Kompatibilität ausschließlich für 2.7.0 absichern

**What to deliver:** Vimkunjas direkte SQLite-Zugriffe ausschließlich für Vikunja 2.7.0 absichern und die Kompatibilität mit einer echten, synthetisch befüllten Vikunja-2.7.0-Datenbank belegen. Andere Datenbankrevisionen beim produktiven Start verständlich ablehnen.

**Category:** compatibility

**Priority:** High — Voraussetzung für die GitHub-Erstveröffentlichung.

**Status:** completed

**Former blocker:** [Publish to GitHub](../../../2026-10-07/publish-to-github/issues/01-publish-to-github.md) — Kompatibilitätsvoraussetzung erfüllt.

## VERBINDLICH: Veröffentlichung erst nach Abschluss dieses Tickets

**Die öffentliche GitHub-Erstveröffentlichung ist blockiert, bis dieses Ticket abgeschlossen ist und die Kompatibilität mit Vikunja 2.7.0 mit überprüfbarer Evidenz feststeht. Diese Voraussetzung ist seit 2026-10-05 erfüllt. Ein Quellenvergleich oder grüne Tests gegen reduzierte Eigenbau-Schemas allein erfüllen diese Voraussetzung nicht.**

Vorbereitungen an Lizenz, Dokumentation, CI und Veröffentlichungskopie können parallel weiterlaufen. Dieses Ticket beauftragt noch keinen Push.

## Ausgangspunkt

- Vimkunja begann am 2026-07-08. Der lokale Upstream-Checkout wurde laut Reflog am 2026-07-07 geklont und steht auf `82ee780161c6ed250cdb1fc4d2952b72464cb724` (`v2.3.0-963-g82ee78016`). Das ist ein Entwicklungsstand nach 2.3.0, kein identischer Release-Stand.
- Welche Version die damalige Nutzerdatenbank initialisiert hat, ist nicht belegt. Für diese Arbeit ausschließlich disposable Datenbanken verwenden.
- Quellenvergleich der Release-Tags 2.3.0, 2.4.0, 2.5.0, 2.6.0 und 2.7.0 vom 2026-10-05: Kernmodelle weitgehend erhalten, aber neue Persistenzregeln sind für direkte SQLite-Schreibzugriffe relevant. Kein ausgeführter Versions-Roundtrip wurde damit behauptet.
- Jan hat den Umfang auf genau Vikunja 2.7.0 begrenzt. Die frühere Untersuchung von 2.3.0 bis 2.7.0 bleibt historischer Kontext, kein Auftrag für eine Versionsmatrix.

## Verbindlicher Umfang

- Einziges Kompatibilitätsziel: Release-Tag `v2.7.0`; den aufgelösten Commit im Prüfbericht festhalten. Die Abnahme ist durch die dokumentierten Regressionen und echten Server-Roundtrips erfüllt.
- Frühere Releases, spätere Releases einschließlich Patch-Releases und Entwicklungsstände sind nicht unterstützt. Keine Mehrversions-Adapter und keine Upgrade-Matrix bauen.
- Beim produktiven Start Migrationshistorie und relevante Schema-Merkmale gegen den geprüften 2.7.0-Stand prüfen; unbekannte, ältere, neuere oder unvollständige Revisionen vor fachlichen Schreibzugriffen ablehnen. Aus einer SQLite-Datei lässt sich nicht zwingend die genaue Serverversion bestimmen: Releases mit identischer DB-Revision sind dadurch nicht unterscheidbar, erhalten aber keine eigene Freigabe.
- Eine frische, durch Vikunja 2.7.0 initialisierte Datenbank ist der verpflichtende Nachweis. Upgrade-Tests älterer Releases und gezieltes Nachstellen historischer Reparaturmigrationen sind außerhalb dieses Tickets.
- VOA und Vimkunja im Prüflauf nacheinander verwenden. Keine Zusage für parallele Serverbearbeitung. Tutorial-Fixtures bleiben ausschließlich ein synthetischer Demo-Workflow.
- Weitere Versionsprüfungen erfolgen erst bei einem ausdrücklich gewünschten Upgrade. Ziel ist ein begrenzter Wartungsaufwand für Jans Sideproject.

## Relevante Änderungen und Prüfaufgaben

1. **Ab 2.4: Soft-Delete.** `tasks.deleted_at` und Papierkorb-Semantik berücksichtigen. Alle Task-Leseflüsse, Suche, Labels, Kanban, Details, Schreibzugriffe und Undo/Redo prüfen. Vimkunja filtert diese Tasks derzeit nicht generell und löscht endgültig. Festlegen, wie die akzeptierte permanente Löschung mit neueren Versionen vereinbar bleibt; eine Abweichung braucht einen ausdrücklichen Contract.
2. **Ab 2.4: Unique-Indizes.** Eindeutigkeit von `tasks(project_id, index)` und `task_positions(task_id, project_view_id)` in Erstellen, Verschieben und Undo/Redo prüfen. Transaktionen müssen bei Konflikten ohne Teiländerungen abbrechen.
3. **Gültiger 2.7.0-Datenbestand.** Kanban-Bucket-Modus, vorhandene Indizes und vererbte Project-Archivierung gegen die 2.7.0-Regeln prüfen. Historische Reparaturmigrationen aus 2.5/2.6 müssen nicht separat nachgestellt werden.
4. **Ab 2.7: Task-Zähler und Aliase.** `project_task_counters` und `task_index_aliases` prüfen. Vimkunjas bisheriges `MAX(index)+1` pflegt diese Tabellen nicht. Erstellen, Project-Wechsel, Löschung und Undo/Redo müssen Zähler-/Aliasregeln erhalten; danach darf eine Task-Erstellung in Vikunja nicht mit zuvor von Vimkunja vergebenen Nummern kollidieren. Alte Task-Adressen müssen gemäß dem freigegebenen Contract auflösbar bleiben.
5. **Ab 2.7: Project-Hierarchie.** Root-Parents als NULL und `project_ancestors` berücksichtigen. Der Reader unterstützt bereits NULL; aktuelle Geschwister-Sortierung sowie spätere Project-Erstellung, Parent-Wechsel und Löschung gegen die neuen Regeln abgrenzen. Nicht implementierte Project-Features müssen im Rahmen dieses Tickets nicht neu gebaut werden.
6. **Neue Relationen beim Löschen.** Insbesondere `time_entries` und `task_index_aliases` prüfen: Welche Daten erhält oder entfernt die Original-App, und welche Bereinigung fehlt bei Vimkunjas permanenter Löschung? Bestehende Fixes aus dem Data-Integrity-Audit übernehmen, nicht duplizieren.
7. **Archivierung und Bucket-Limits.** Schreibzugriffe auf archivierte Source-/Target-Projects sowie Überschreitungen gesetzter Bucket-Limits verhindern. Erstellen, Project-Wechsel, Workflow-/Done-Wechsel und Undo/Redo einbeziehen.

## Prüfstand — 2026-10-05

[Prüfbericht mit Evidenz](../report.md): produktive Schema-Prüfung und Store-Fixes umgesetzt. Fokussierte Regressionen für Trash, Archivierung, Bucket-Limits, Zähler/Aliase, Rollback und Hierarchie-Erhaltung bestehen. Zwei echte VOA-2.7.0-Roundtrips bestehen: frische Server-Datenbank und disposable Kopie von Jans synthetischer Windows-Baseline. VOA liest/bearbeitet die Store-Änderungen, API v2 löst alte Adressen auf und folgende Task-Erstellungen verwenden neue Nummern. Permanente Löschung bleibt der vorhandene Vimkunja-Contract; Tasks mit Time Entries werden dabei abgelehnt und müssen in VOA gelöscht werden. Alle Tests, Vet und beide Builds erfolgreich; originale Baseline einschließlich WAL/SHM unverändert. Native Windows-Terminaltests werden nicht behauptet.

## Abnahme

- [x] Ausschließlich Vikunja 2.7.0 als geprüftes Ziel dokumentieren; Befunde beheben oder eine begründete Workflow-Einschränkung mit Jan vereinbaren.
- [x] Eine reproduzierbare Testdatenbank durch Vikunja 2.7.0 initialisieren, mit ausschließlich synthetischen Usern, Projects, Views und Tasks. Schema und Migrationshistorie stammen von Vikunja; keine Nutzerdaten kopieren.
- [x] Release-Tag/Commit, angewendete Migrationen, Schema-Merkmale, Testschritte und tatsächliche Ergebnisse dokumentieren. Kein Nachweis für andere Releases erforderlich.
- [x] Erstellen, Bearbeiten, Labels/Comments, Done/Buckets, Project-Wechsel, Löschung und unterstütztes Undo/Redo über Vimkunja prüfen; Ergebnisse anschließend durch Vikunja 2.7.0 lesen und bearbeiten.
- [x] Gezielte Regressionen für Soft-Delete, Constraints, Zähler/Aliase, Project-Hierarchie und Löschrelationen ergänzen. Fehler, Rollback und unveränderte fremde Daten prüfen.
- [x] Produktiver Start lehnt ungetestete DB-Revisionen verständlich und ohne fachliche Schreibzugriffe ab; Regressionen für ältere, neuere und unvollständige Schemas ergänzen. Nicht unterstützte Workflows und Grenzen der Versionserkennung dokumentieren; keine Behauptung sicherer paralleler Serverbearbeitung.
- [x] Store-Implementierung, betroffene Kommentare und README mit dem belegten Contract synchronisieren; HTTP-/JSON-/Schema-Details bleiben im Store.
- [x] Tests und beide Builds erfolgreich; Prüfbericht im Feature-Verzeichnis festhalten und dieses Ticket erst danach abschließen.
- [x] Veröffentlichungsticket und STATUS aktualisieren: diese Voraussetzung darf erst nach erfüllter Abnahme als erledigt markiert werden.

## Umsetzungsschritte

1. `v2.7.0` auf einen Commit pinnen, disposable DB durch diese Version initialisieren und synthetische Daten über VOA anlegen. Reproduktion, Migrationen und Schema festhalten.
2. 2.7.0-Persistenzregeln gegen Store, Suche und Undo/Redo abgleichen; permanente Löschung und Task-Adressen als Contract festlegen.
3. Produktive Schema-Prüfung implementieren und Ablehnung ungetesteter Revisionen testen; Tutorial-/Test-Flows klar abgrenzen.
4. Soft-Delete, Constraints, Zähler/Aliase und Löschrelationen im Store korrigieren; Hierarchie-/Archivierungsregeln prüfen. Fokussierte Regressionen einschließlich Rollback und unveränderter fremder Daten ergänzen.
5. Alle Abnahmeoperationen über Vimkunja ausführen, anschließend durch VOA 2.7.0 lesen und bearbeiten; insbesondere weitere VOA-Task-Erstellung und alte Task-Adressen prüfen.
6. Tests und beide Builds ausführen, Prüfbericht und README synchronisieren. Ticket und Veröffentlichungsblocker erst nach belegter Abnahme abschließen.

## Grenzen

`internal/db/vikunja.db` nicht öffnen, ändern oder löschen. Kein produktives Schema ändern, keine eigenen Vikunja-Migrationen und keine Dependency-Änderungen ohne separaten Auftrag. Ein dauerhaftes Start-Template ist eine nachgelagerte Entscheidung auf Basis der hier geprüften Version, kein Ersatz für Kompatibilitätsnachweise.

## Quellen

- [Soft-Delete-Migration, 2.4](https://github.com/go-vikunja/vikunja/blob/v2.4.0/pkg/migration/20260707094311.go)
- [Unique Task-Nummern, 2.4](https://github.com/go-vikunja/vikunja/blob/v2.4.0/pkg/migration/20260411013328.go), [Unique Positionen, 2.4](https://github.com/go-vikunja/vikunja/blob/v2.4.0/pkg/migration/20260617153629.go)
- [Kanban-Reparatur, 2.5](https://github.com/go-vikunja/vikunja/blob/v2.5.0/pkg/migration/20260802162816.go), [Archivierung, 2.6](https://github.com/go-vikunja/vikunja/blob/v2.6.0/pkg/migration/20260819212842.go)
- [Task-Zähler/Aliase und Nummernvergabe, 2.7](https://github.com/go-vikunja/vikunja/blob/v2.7.0/pkg/models/task_index.go)
- [Root-Parents, 2.7](https://github.com/go-vikunja/vikunja/blob/v2.7.0/pkg/migration/20260903104500.go), [Project-Ancestors, 2.7](https://github.com/go-vikunja/vikunja/blob/v2.7.0/pkg/migration/20260908202544.go)

## Comments

### 2026-10-06 — Produktive DB nach Backup auf 2.7.0 aktualisiert

Der gehärtete Startcheck lehnte die bestehende produktive Standard-DB mit 116 statt 150 Migrationen ab. Jan beauftragt nach erfolgreichem Upgrade einer Kopie ausdrücklich das Upgrade der produktiven DB mit vorheriger Backup-Kopie. Offizielles VOA 2.7.0 migriert `internal/db/vikunja.db`; Backup `internal/db/vikunja-pre-2.7.0-20261006.db` samt vorhandener SHM-Datei wurde byteweise verifiziert. Daten-Erhaltung, exakte Revisionshistorie, Integrität, Store-Lesen und realer TUI-Start geprüft. Standardpfad unverändert, Guard weiterhin strikt. Opt-in-Upgrade-Regression und verständlicher Revisions-/Upgrade-Hinweis ergänzt; Einzelheiten im Prüfbericht. Kein Commit oder Push.

### 2026-10-05 — Entscheidung mit Jan

Separates Ticket für die DB-Kompatibilität über die Versionen 2.3.0 bis 2.7.0 anlegen und die GitHub-Veröffentlichung verbindlich von dessen Abschluss abhängig machen. Heute wurden Ticket und Abhängigkeit dokumentiert; keine Kompatibilitätsimplementierung oder Veröffentlichung ausgeführt.

### 2026-10-05 — Umfang auf genau 2.7.0 begrenzt

Jan wählt ausschließlich Vikunja 2.7.0, um den Wartungsaufwand des Sideprojects zu begrenzen. Mehrversions-Support und die Upgrade-Matrix entfallen. Ungetestete DB-Revisionen sollen abgelehnt werden; zukünftige Upgrades benötigen einen separaten Auftrag. Diese Änderung legt den Prüfauftrag fest, behauptet keine bereits belegte Kompatibilität und beauftragt keinen Push.

### 2026-10-05 — Baseline bereitgestellt und erste Prüfung beauftragt

Jan initialisiert und befüllt eine echte VOA-2.7.0-Datenbank, stoppt VOA und legt die saubere Kopie einschließlich WAL/SHM unter `tmp/vikunja_clean_db/` ab. Auf „go ahead“ erfolgt die Schema-/Persistenzprüfung auf disposable Kopien. Befunde und Reproduktionsprobes stehen im verlinkten Prüfbericht; die Baseline bleibt unverändert. Implementierung und vollständige Abnahme stehen aus.

### 2026-10-05 — Implementierung und automatisierte Abnahme abgeschlossen

Jan beauftragt, die weiteren Schritte zu übernehmen. Die dokumentierten Store-Fixes und Startprüfung sind umgesetzt; Fresh-/Baseline-Roundtrip, Regressionen, vollständige Tests, Vet und Builds erfolgreich. Prüfbericht und README enthalten den begrenzten 2.7.0-Contract und dessen Grenzen. Der Kompatibilitätsblocker der Veröffentlichung ist erfüllt. Kein Commit und kein Push ausgeführt.
