# 01 — Publish Vimkunja to GitHub

**What to deliver:** Prepare a public GitHub repository with usable code and documentation, preserving transparent development history where safe.

**Category:** maintenance

**Priority:** High for code and documentation; tickets, prototypes and Git history are secondary.

**Status:** needs-triage

**Order:** 3 of Jan's agreed next-work sequence: finish [URL-Copy shortcut](../../description-link-copy-shortcut/issues/01-choose-accessible-url-copy-shortcut.md), then [subtask cleanup](../../release-ticket-cleanup/issues/01-reconcile-open-subtasks-before-publication.md), then complete GitHub initialization/publication.

## VERBINDLICHE ENTSCHEIDUNG: Backup-Referenz niemals veröffentlichen

**Nur `main` und ausdrücklich ausgewählte Tags veröffentlichen. `refs/original/refs/heads/main` und andere Backup-Referenzen dürfen NICHT auf GitHub gelangen. KEIN `git push --mirror`.**

- Eine separate Veröffentlichungskopie vorbereiten, die nur `main` und die ausgewählten Tags übernimmt und keine Backup-Referenzen enthält.
- Vor dem Push die vollständige erreichbare Historie dieser Referenzen auf Datenbanken und sensible Inhalte prüfen. Ein sauberer aktueller Dateibaum oder `.gitignore` reicht nicht aus.
- Remote-Konfiguration auf Mirror-Modus und zusätzliche Push-Refspecs prüfen. Vor dem tatsächlichen Push einen Dry Run mit denselben expliziten Referenzen ausführen; die Ausgabe darf nur die ausgewählten Ziele enthalten.
- Die lokale Backup-Referenz darf bestehen bleiben. Ihre Löschung und eine erneute Historienumschreibung sind für die Veröffentlichung nach aktuellem Befund nicht erforderlich.
- Die lokale Nutzerdatenbank und laufende Richtext-Arbeit bleiben unangetastet. Dieses Ticket dokumentiert die Vorbereitung; es ist keine Anweisung zum sofortigen Push oder Löschen von Referenzen.

Beispiel für einen Dry Run, sofern `origin` auf das bestätigte Ziel zeigt und beide Tags veröffentlicht werden sollen:

```bash
git push --dry-run origin main refs/tags/MVP refs/tags/Milestone1
```

Nach Prüfung erfolgt der tatsächliche Push mit denselben Referenzen ohne `--dry-run`.

## Veröffentlichungsumfang

- Code und Nutzerdokumentation haben höchste Priorität.
- Tickets, Specs, Prototypen, Archiv und Git-Historie möglichst mitnehmen; unfertige Ideen und dokumentierte Fehlentscheidungen sind kein Ausschlussgrund.
- Keine Secrets, echten Nutzerdatenbanken, privaten oder unangemessenen persönlichen Inhalte veröffentlichen. Persönliche Inhalte zusätzlich manuell prüfen.
- Leere Samples und synthetische Fixtures/Templates sind zulässig, sofern sie keine echten Nutzerdaten enthalten.
- Rich-Text ist abgeschlossen und abgenommen. Vor Abschluss der Erstveröffentlichung folgen jetzt URL-Copy-Follow-up und Subtask-Cleanup; nicht alle übrigen M3-Features sind Veröffentlichungsvoraussetzungen.

## Prüfbefund vom 2026-10-04

### Datenbankhistorie

- Alter Commit `35b34220934d21c3bbd8e26bfe537098f28da966` (`MVP Vimkunja`, 2026-07-08) enthält `internal/db/vikunja.db` (569344 Byte), `internal/db/vikunja.db-shm` (32768 Byte) und `internal/db/vikunja.db-wal` (0 Byte).
- Alter Folgecommit `42f6029` entfernt diese Dateien, entfernt sie aber nicht aus seiner Vorgeschichte.
- Diese alte Historie ist über `refs/original/refs/heads/main` erreichbar.
- Die vollständigen Historien von `main`, `MVP` und `Milestone1` enthalten keine DB-Dateien. Der aktuelle MVP-Root-Commit ist `23cd558`.
- WAL-Dateien können Daten enthalten; SHM-Dateien gehören ebenfalls nicht in die Veröffentlichung. Die aktuell leere historische WAL-Datei wird nicht als Sample übernommen.

### Secret-Scan

- Gitleaks 8.30.1: aktueller Dateibestand einschließlich ungetrackter/ignorierter Dateien; Git-Historie mit allen Referenzen und Reflogs; zusätzlich alle lokal gespeicherten Blob-, Commit- und Tag-Objekte einschließlich unerreichbarer Altstände.
- Historienlauf: 144 Commits, keine Treffer. Zusätzlicher Objektlauf umfasste 1009 Blobs, 155 Commits und 2 Tag-Objekte.
- Dateibestand: ein Fehlalarm für den `localStorage`-Schlüssel in `.scratch/editor-motion-polish/acceptance_test_V1.html:101`.
- Objektlauf: vier Fehlalarme für historische `go.sum`-Prüfsummen des Moduls `modernc.org/token`.
- Keine echten Secrets gefunden. Automatische Erkennung ist keine Garantie; vor Veröffentlichung erneut scannen und persönliche Inhalte manuell prüfen.
- Redigierte JSON-Berichte liegen temporär außerhalb des Repositorys unter `/tmp/vimkunja-secret-audit-6nnrkfsc/`; diese Dateien sind keine dauerhaft verfügbare Ticket-Evidenz.

## Vorabprüfung vom 2026-10-04: frischer Clone und Dokumentation

**Geprüfter Stand:** committed `main` (`761774b`), als frischer `--no-local --single-branch`-Clone außerhalb des Workspace. Laufende uncommitted Änderungen sind nicht Teil dieses Ergebnisses. Temporäre Prüfdateien liegen unter `/tmp/vimkunja-publication-check-jqyy8_9u/`; sie sind nicht dauerhaft verfügbar.

### Erfolgreich geprüft

- Clone enthält `main`, die dazugehörigen Remote-Tracking-Referenzen und die Tags `MVP`/`Milestone1`; keine `refs/original/`-Referenz und keine erreichbaren historischen DB-Dateien.
- Go 1.25.0, Linux amd64: `go test ./...`, beide Entrypoint-Builds und `go vet ./...` erfolgreich. Tests liefen mit einem leeren separaten Benutzerkonfigurationsverzeichnis.
- Beide Entrypoints erfolgreich für Windows amd64 mit `CGO_ENABLED=0` cross-kompiliert. Dies belegt keine Windows-Terminal-, Tastatur- oder Clipboard-Funktion.
- Beide `--help`-Aufrufe erfolgreich. Tutorial in einem Linux-PTY gestartet: sichtbare erste Lektion mit synthetischen Tasks; sauberer Exit mit `Ctrl+C`. Kein vollständiger Durchlauf aller Lektionen.
- Nach Tests, Builds und Tutorial ist der Clone-Dateibaum unverändert. Die echte Nutzerdatenbank wurde nicht geöffnet.
- Lokale Lizenzdateien für die in den Linux-Binaries verwendeten Drittmodule vorhanden. Das ist nur eine Bestandsaufnahme, keine abgeschlossene Prüfung der Lizenzbedingungen oder Code-Herkunft. Die eigene Projektlizenz fehlt weiterhin.

### Vor Veröffentlichung zu klären

- **README-Einstieg:** `cd /workspace/vimkunja` ist ein lokaler Entwicklungspfad; Clone-/Installationsanleitung und Go-Voraussetzung fehlen. Der normale Start benötigt eine separat bereitgestellte Vikunja-Datenbank. Das Tutorial ist der funktionierende Einstieg ohne Nutzerdatenbank.
- **Fehlender DB-Pfad:** Ein Start mit `--db` auf eine nicht vorhandene Datei erzeugt eine leere SQLite-Datei. Im nichtinteraktiven Prüfprozess folgt ein TTY-Fehler; ein hilfreicher Fehler für eine fehlende Vikunja-Datenbank ist damit nicht belegt. Das widerspricht der README-Aussage, dass der Pfad existieren muss. Nur mit einem temporären Prüfpfad reproduziert; keine Korrektur vorgenommen.
- **Kompatibilität:** Keine belegte Liste unterstützter Vikunja-Releases gefunden. Store-Tests und Tutorial verwenden selbst angelegte Schemas; vorhandene Recherche verweist auf einen Upstream-Commit, ersetzt aber keinen versionierten Integrationstest mit der Original-App. Ein solcher Roundtrip wurde hier nicht durchgeführt. Die Einschränkung auf genau einen lokalen User bei bestimmten Schreiboperationen muss dokumentiert werden.
- **Vier kaputte relative Markdown-Dateilinks im getrackten Archiv:** `.archive/2026-07-17/fundament-mvp/SPEC_FUNDAMENT.md` verweist auf `.archive/TUI_IDEA.md`; `.archive/2026-07-27/all-tasks-overview/issues/01-add-cross-project-all-tasks-view.md` auf `../../../prototype/overview_tui/`; `.archive/2026-08-02/global-search-filter/issues/01-specify-global-search-and-filter.md` auf `../../local-search/issues/01-search-within-focused-pane.md` und `02-add-advanced-global-search-query-and-ranking.md`. Ziele fehlen relativ zum jeweiligen Dokument. Externe Links und Fragment-Anker sind noch nicht vollständig geprüft.
- **Lokale Pfade:** README sowie Richtext-Spec, Corpus-Dokumentation und Richtext-Ticket enthalten `/workspace`-Verweise. Bei Veröffentlichung durch portable Links ersetzen oder ausdrücklich als historische lokale Artefakte kennzeichnen.
- Persönliche Inhalte/Commit-Metadaten und Herkunft übernommener Bestandteile bleiben manuelle Prüfaufgaben; kein pauschales Freigabeurteil aus diesem technischen Check.

## Umsetzung der Vorab-Findings — 2026-10-04

- **MIT beschlossen und ergänzt:** `LICENSE` enthält MIT mit Copyright 2026 Jan Wartenberg; README verlinkt die Lizenz. Die Herkunft übernommener Bestandteile und Drittanbieter-Lizenzbedingungen bleiben separat zu prüfen.
- **README:** portable Clone-/Build-Anleitung mit vorläufigem `<repository-url>`-Platzhalter, Go 1.25+, Tutorial-Einstieg ohne Nutzerdaten, SQLite-Betrieb und derzeitige Plattform-/Versionsgrenzen dokumentiert. Keine unbelegte Vikunja-Versionsfreigabe.
- **Links:** vier Archivlinks korrigiert. `.archive/TUI_IDEA.md` wird als benötigte historische Designquelle ausdrücklich aus der Archive-Ignore-Regel ausgenommen; Gitleaks findet darin keine Secrets. Alle lokalen Markdown-Dateilinks im aktuellen veröffentlichbaren Dateibestand lösen auf; externe Links und Fragment-Anker sind nicht vollständig geprüft.
- **Lokale Pfade:** README verwendet keine Workspace-Adresse mehr. Richtext-Dokumente nennen historische temporäre HTML-Artefakte ausdrücklich als nicht mitgelieferte Dateien; die frühere externe Upstream-Checkout-Adresse ist durch eine portable Beschreibung ersetzt. Inhaltliche Richtext-Entscheidungen bleiben erhalten.
- **Fehlende DB:** Store öffnet ausschließlich vorhandene reguläre Dateien; SQLite `mode=rw` verhindert auch versehentliches Neuanlegen nach dem Dateicheck. CLI liefert einen Hinweis auf `--db` und das Tutorial. Regressionstests decken fehlende Dateien, Verzeichnisse und Dateinamen mit Leerzeichen, `#` und `%` ab.
- **CI vorbereitet:** `.github/workflows/ci.yml` führt Tests, beide Builds und Vet auf Linux und Windows aus, bei Push auf `main`, Pull Requests und manuell. Workflow lokal mit actionlint v1.7.12 geprüft. Ein tatsächlicher GitHub-Actions-Lauf steht vor Veröffentlichung noch aus.
- **Validierung:** aktueller Workspace besteht `go test ./...`, beide Linux-Builds, `go vet ./...` und beide Windows-amd64-Cross-Builds. Die realen Windows-Terminaltests bleiben offen. Keine Nutzerdaten geöffnet, keine Schema-/Dependency-Änderungen, kein Commit oder Push.

### Blanke DB oder Start-Template: geprüft, Produktentscheidung offen

Eine blanke SQLite-Datei hat kein Vikunja-Schema und bietet keinen nutzbaren frischen Start. Der Store bleibt für bestehende Test-/Bootstrap-Flows in der Lage, eine vorhandene leere Datei zu öffnen; die produktive Anwendung initialisiert sie nicht. Eine vollständige Schema-Prüfung mit eigener Fehlermeldung ist damit noch nicht implementiert.

Für einen produktiven frischen Start sollte die gewählte Vikunja-Version selbst Schema und Migrationshistorie initialisieren; anschließend sind mindestens die für den Workflow nötigen User-/Project-/View-Daten erforderlich. Ein Template müsste aus genau dieser geprüften Version erzeugt werden, ausschließlich synthetische Daten enthalten und durch einen Roundtrip in der Original-App belegt werden. Die reduzierte Tutorial-Datenbank ist kein produktives Vikunja-Template.

**Noch festzulegen:** zunächst den von Vikunja initialisierten Start dokumentieren oder zusätzlich ein versioniertes, synthetisches Template anbieten. Kein produktives Template und keine eigenständige Vimkunja-Schemaerstellung wurden ohne diesen Contract eingeführt.

## Offene Schritte / Abnahme

- [x] Öffentliches Ziel `JanWartenberg/vimkunja` festgelegt; Go-Modulpfad und Imports abgeglichen. Repository-Erstellung/öffentlicher Push bleiben ausstehend.
- [x] MIT-Projektlizenz ergänzen.
- [ ] Herkunft übernommener Bestandteile und Drittanbieter-Lizenzbedingungen abschließend prüfen.
- [x] README um Build-Anleitung, Voraussetzungen, Tutorial und aktuelle Plattform-/Kompatibilitätsgrenzen ergänzen; SQLite-Zugriff und fehlenden REST-Modus klar benennen.
- [x] Finale Clone-URL `https://github.com/JanWartenberg/vimkunja.git` eingetragen.
- [ ] Konkrete Vikunja-Version durch Integrationstest belegen (Kompatibilitätsblocker).
- [ ] Blanke DB / versioniertes Start-Template entscheiden; gegebenenfalls Schema-Prüfung und Initialisierungsworkflow spezifizieren.
- [ ] Persönliche Inhalte und Commit-Metadaten prüfen; Veröffentlichungsumfang bestätigen.
- [x] Reihenfolge festgelegt: nach abgeschlossenem Rich-Text erst URL-Copy-Follow-up, dann Subtask-Cleanup, danach GitHub-Initialisierung/Veröffentlichung abschließen.
- [ ] Aktuelle Änderungen sauber ordnen; reine Zeilenendenänderungen nicht als fachliche Änderungen committen.
- [x] CI für Tests, beide Builds und Vet auf Linux/Windows vorbereiten.
- [ ] Ersten echten GitHub-Actions-Lauf prüfen und native Windows-Terminaltests durchführen.
- [ ] Separate Veröffentlichungskopie ohne Backup-Referenzen prüfen; Secret-Scan und DB-Historienprüfung auf dem finalen Stand wiederholen.
- [ ] Tests und beide Builds auf dem finalen Stand ausführen.
- [ ] Dry Run prüfen und ausschließlich `main` sowie bestätigte Tags veröffentlichen; Remote-Referenzen anschließend kontrollieren.

Automatische Release-Binaries und die Migration lokaler Tickets zu GitHub Issues sind keine Voraussetzung für die Erstveröffentlichung.

## Comments

### 2026-10-04 — Entscheidung mit Jan

Möglichst alles transparent veröffentlichen, mit höchster Priorität für Code und Dokumentation. Die Backup-Referenz lokal behalten und durch explizite Auswahl der Push-Referenzen von der Veröffentlichung ausschließen. Keine erneute Historienumschreibung erforderlich, solange die geprüfte DB-freie Historie veröffentlicht wird. Entscheidung ausdrücklich im Veröffentlichungsticket festhalten.

### 2026-10-04 — MIT und direkte Umsetzung

Jan wählt MIT, beauftragt die direkte Umsetzung der übrigen behebbaren Findings und möchte einen frischen Start mit blanker DB oder Template prüfen. Die oben dokumentierten Vorbereitungen sind umgesetzt; ein produktives Start-Template bleibt eine offene Entscheidung.

### 2026-10-19 — agreed next-work sequence

Jan sets publication third, after URL-Copy usability and reconciliation of remaining subtasks. Rich-text AT-01–10 and the live Vikunja round-trip are accepted. Carry that evidence forward; the exact tested Vikunja release is not yet recorded in this ticket, so do not infer a version guarantee. Existing open publication requirements (startup contract, provenance/personal-content review, final history/secret checks and actual CI) remain to be reconciled and completed. This ticket update performs no GitHub creation, merge or push.
