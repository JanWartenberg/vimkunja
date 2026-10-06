# 01 — Publish Vimkunja to GitHub

**What to deliver:** Prepare a public GitHub repository with usable code and documentation, preserving transparent development history where safe.

**Category:** maintenance

**Priority:** High for code and documentation; tickets, prototypes and Git history are secondary.

**Status:** needs-triage

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
- Vorbereitung kann parallel zum HTML-Richtext-Feature erfolgen. Ob dieses Feature Voraussetzung für die Erstveröffentlichung wird, ist noch nicht entschieden.

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

## Weitere Veröffentlichungsprüfung — 2026-10-05

### Ziel, Modulpfad und direkt behobene Findings

- Jan bestätigt `https://github.com/JanWartenberg/vimkunja` als öffentliches Ziel, analog zu Tagger. `go.mod`, alle bisherigen Go-Imports einschließlich Tests und README-Clone-URL sind angepasst. Dependency-Versionen und `go.sum` bleiben unverändert.
- Der öffentliche GitHub-API-Aufruf für das Ziel liefert derzeit HTTP 404; das beweist nicht, ob das Repository fehlt oder privat ist. Es wurden kein Repository und kein externer Remote angelegt und nichts auf GitHub gepusht.
- `/bin/` ist für die in der README beschriebenen lokalen Builds ignoriert. Globale DB-/WAL-/SHM-/SQLite-Ignore-Regeln schützen zusätzlich vor versehentlichem Staging temporärer Kompatibilitätsdatenbanken außerhalb von `internal/db`; vorhandene lokale DB-Dateien bleiben unverändert. Künftige bewusst veröffentlichte synthetische Fixtures brauchen eine ausdrückliche Ausnahme.

### Lizenzprüfung

- Die tatsächliche Dependency-Closure beider Entrypoints für Linux, Windows und macOS amd64 umfasst insgesamt **30 Drittmodule**. Die Modul-Root-Lizenzen sind MIT bzw. BSD. Die vertiefte Prüfung tatsächlich kompilierter Dependency-Dateien findet zusätzlich MIT-/BSD-2-/BSD-4-Texte und **APSL-2.0-Hinweise in generierten Darwin-Headerdateien von `modernc.org/libc`**; außerdem gelten Musl-/Netdb-/Unicode-Hinweise. Eine reine Root-LICENSE-Prüfung hätte diese zusätzlichen Bedingungen übersehen. Das SQLite-Original ist [Public Domain](https://www.sqlite.org/copyright.html), die `modernc`-Bindings haben eigene BSD-Bedingungen.
- [THIRD_PARTY_NOTICES.txt](../../../THIRD_PARTY_NOTICES.txt) enthält versionierte Modulzuordnung, vollständige Root-Lizenztexte, zusätzliche Lizenz-Kommentarblöcke aus tatsächlich kompilierten Dependency-Dateien (einschließlich Darwin arm64), ergänzende Musl-/Netdb-/Unicode-Hinweise, APSL-1.1-/2.0-Texte samt Source-URLs und die Go-Lizenz. Die Datei und `LICENSE` müssen später bei Binary-Distributionen mitgeliefert werden; README dokumentiert dies. Unbenutzte Logos und ausschließlich in Dependency-Testdaten enthaltene Komponenten werden nicht als ausgelieferte Bestandteile behandelt.
- [Vikunja 2.7.0 ist AGPL-3.0](https://raw.githubusercontent.com/go-vikunja/vikunja/v2.7.0/LICENSE). Vimkunjas Modulgraph bindet keinen Vikunja-Code ein. In den eigenen Go-Dateien wurden keine Übernahme-/Copyright-Vermerke Dritter gefunden. **561 historische Go-Blobs** der vorgesehenen Referenzen wurden mit den Backend-Go-Quellen des lokalen Upstream-Checkouts (`82ee780161c6ed250cdb1fc4d2952b72464cb724`) verglichen: keine identische Folge von acht nichttrivialen Codezeilen nach Entfernen von Leerraum/Kommentarzeilen.
- **Ergebnis für den eigenen Code und die geplante Quellcode-Veröffentlichung:** kein konkreter Lizenzkonflikt gefunden; MIT bleibt bestehen. **Vor einem macOS-Binary-Release ist die APSL-Prüfung offen:** [APSL 2.0](https://opensource.apple.com/apsl/) Abschnitt 2.3 verlangt einen prominenten Hinweis im ausführbaren Code und in der Dokumentation zur Bezugsquelle des betroffenen Quellcodes. Die Notice-Datei allein belegt diese ausführbare Release-Bedingung nicht. Ein macOS-Binary-Release wird hier nicht freigegeben. Der technische Vergleich beweist keine lückenlose unabhängige Herkunft und erkennt insbesondere umgeschriebene oder kurze Übernahmen nicht. Künftige tatsächliche Upstream-Codeübernahmen brauchen eine erneute Herkunfts-/Lizenzprüfung; eine DB-Schnittstelle allein ist kein Nachweis einer Codeübernahme.

### Links und persönliche Inhalte

- **100 veröffentlichbare Markdown-Dateien im abschließenden Lauf:** lokale Dateilinks und sechs Markdown-Fragmente geprüft; nach der Korrektur kein defekter Link gefunden. Nicht getrackte, ignorierte lokale Artefakte wurden nicht als veröffentlichter Inhalt vorausgesetzt.
- **34 externe Dokumentationslinks einschließlich der während des Audits ergänzten Lizenz-/Security-Quellen:** HTTP 200, einschließlich Prüfung der vorhandenen GitHub-Zeilenanker gegen die referenzierten Rohdateien. Externe Linkziele können sich später ändern.
- Suche in veröffentlichbaren Textdateien nach persönlichen Home-Verzeichnissen und E-Mail-Adressen: keine zusätzlichen Kandidaten außerhalb der Drittanbieter-Lizenzhinweise und des hier dokumentierten Commit-Metadatenbefunds. Keine konkrete unangemessene persönliche Passage identifiziert; subjektive Inhaltsfreigabe bleibt bei Jan.
- **Veröffentlichte Commit-Metadaten würden `Jan Wartenberg <5298128+JanWartenberg@users.noreply.github.com>` sowie die lokalen Agent-Identitäten enthalten.** Die persönliche E-Mail-Adresse ist damit öffentlich sichtbar, auch ohne Angabe in der README. Keine Historienumschreibung und keine Änderung der Git-Identität vorgenommen. Vor dem tatsächlichen Push diese Sichtbarkeit bewusst bestätigen oder eine gesonderte Bereinigung beauftragen.

### Referenzen und erneute Secret-Prüfung

- Frischer `--no-local --single-branch`-Clone von committed `main` (`b6aa06061cc0df1a4c78643c8ab06fb3fac14210`), anschließend Dry Run und tatsächlicher Push **ausschließlich in ein temporäres lokales Bare-Repository** mit expliziten Referenzen. Ziel enthält exakt `refs/heads/main`, `refs/tags/MVP` und `refs/tags/Milestone1`. Kein `refs/original/`, keine erreichbaren `.db`-/WAL-/SHM-/SQLite-/SQL-Dump-Dateien. Uncommitted Vorbereitungen sind nicht Teil dieser Historienprobe.
- Workspace hat aktuell keine konfigurierten Remotes und somit keinen Mirror-Modus oder zusätzlichen Remote-Push-Refspec. Die Zielkonfiguration muss nach dem späteren Anlegen von `origin` erneut geprüft werden.
- Gitleaks 8.30.1 erneut über alle Referenzen/Reflogs (**157 Commits**), den gesamten aktuellen Dateibestand und **alle lokalen Git-Objekte** einschließlich unerreichbarer Altstände (**1139 Blobs, 168 Commits, 2 Tags**). Keine echten Secrets gefunden. Ein `localStorage`-Schlüssel und vier alte `modernc.org/token`-Prüfsummen bleiben bestätigte Fehlalarme. Rohberichte sind temporär; diese Zahlen und Ergebnisse sind die dauerhafte Ticket-Evidenz.

### Toolchain und Plattformen

- Mit Go 1.27.1 zusätzlich beide CLI-Help-Aufrufe geprüft, fehlenden DB-Pfad mit hilfreichem Fehler ohne Dateineuanlage bestätigt und Tutorial in einem Linux-PTY mit sichtbarer Ausgabe und sauberem `Ctrl+C`-Exit gestartet. Separates temporäres Benutzerkonfigurationsverzeichnis; keine produktive DB über die Anwendung geöffnet.
- Mit Go 1.25.0 und Go 1.27.1: `go test ./...`, beide Entrypoint-Builds und `go vet ./...` erfolgreich. `go mod verify` erfolgreich; CI nach Änderung erneut mit actionlint 1.7.12 geprüft.
- Beide Entrypoints mit `CGO_ENABLED=0` für Windows amd64 und macOS arm64 cross-kompiliert. Das ist keine native Terminal-, Tastatur-, Clipboard- oder Betriebssystemabnahme. Native Windows-Tests und ein echter GitHub-Actions-Lauf bleiben offen.
- **Neues konkretes Security-Finding:** govulncheck 1.8.0 erkennt mit lokalem Go 1.25.0 [GO-2026-4341](https://pkg.go.dev/vuln/GO-2026-4341) als erreichbaren `net/url.ParseQuery`-Aufruf über den SQLite-Treiber. Die produktive DSN hat nur den festen Queryparameter `mode=rw`; der Aufrufnachweis beweist keine praktische Ausnutzbarkeit durch Vimkunja. Trotzdem ist die ungepatchte Toolchain kein geeigneter Release-Stand.
- **Behoben für CI/Build-Anleitung:** CI nutzt die aktuelle stabile Go-Version statt des festen, ungepatchten Go 1.25.0 aus `go.mod`. README empfiehlt eine aktuelle sicherheitsgepatchte Toolchain. Die Sprach-/Modul-Mindestversion und Dependency-Versionen werden dadurch nicht angehoben; die lokale Systeminstallation bleibt unverändert.
- **Security-Nachprüfung:** govulncheck 1.8.0, mit Go 1.27.1 gebaut und auf Go 1.27.1 ausgeführt, endet erfolgreich ohne Findings (Go-Vulnerability-Datenbankstand 2026-10-01). Die erste Nachprüfung mit dem unter Go 1.26 gebauten Scanner konnte neue Go-1.27-Syntax nicht laden und war daher kein gültiger Scan; nach Neubau des Scanners erfolgreich wiederholt. Keine Dependency-Upgrades erforderlich.

### Review der Veröffentlichungseinheit — 2026-10-06

- Ignore-Regeln decken nun WAL-/SHM-/Rollback-Journal-Dateien für `.db`, `.sqlite` und `.sqlite3` ab; vorher waren nur `.db`-WAL-/SHM-Dateien geschützt.
- Die erhaltenen Darwin-Header nennen APSL 1.1 und 2.0. Notices enthalten jetzt beide Lizenztexte mit Bezugsquellen; die spätere macOS-Binary-Abnahme bleibt offen.
- Die Veröffentlichungseinheit wurde getrennt von uncommitted DB-/Richtext-Arbeit geprüft. Tests, beide Builds, Vet und Workflow-Lint bestehen auf einer isolierten Kopie des gestagten Codes mit Go 1.27.1. Kein Push oder Release beauftragt.

## Offene Schritte / Abnahme

- [x] Öffentliches Ziel `JanWartenberg/vimkunja` festgelegt; Go-Modulpfad und Imports abgeglichen. Repository-Erstellung/öffentlicher Push bleiben ausstehend.
- [x] MIT-Projektlizenz ergänzen.
- [x] Technische Herkunfts- und Drittanbieter-Lizenzprüfung durchgeführt; Umfang und Grenzen siehe Prüfbericht vom 2026-10-05.
- [x] README um Build-Anleitung, Voraussetzungen, Tutorial und aktuelle Plattform-/Kompatibilitätsgrenzen ergänzen; SQLite-Zugriff und fehlenden REST-Modus klar benennen.
- [x] Finale Clone-URL `https://github.com/JanWartenberg/vimkunja.git` eingetragen.
- [ ] Konkrete Vikunja-Version durch Integrationstest belegen (Kompatibilitätsblocker).
- [ ] Blanke DB / versioniertes Start-Template entscheiden; gegebenenfalls Schema-Prüfung und Initialisierungsworkflow spezifizieren.
- [ ] Persönliche Inhalte und Commit-Metadaten prüfen; Veröffentlichungsumfang bestätigen.
- [ ] Zeitpunkt der Erstveröffentlichung gegenüber dem HTML-Richtext-Feature entscheiden.
- [ ] Aktuelle Änderungen sauber ordnen; reine Zeilenendenänderungen nicht als fachliche Änderungen committen.
- [x] CI für Tests, beide Builds und Vet auf Linux/Windows vorbereiten.
- [ ] Ersten echten GitHub-Actions-Lauf prüfen und native Windows-Terminaltests durchführen.
- [ ] Vor etwaigen macOS-Binary-Releases APSL-Source-/Executable-Notice-Bedingungen für generierte Darwin-Dependency-Dateien erfüllen; aktuell ist nur die technische Cross-Kompilation geprüft.
- [ ] Separate Veröffentlichungskopie ohne Backup-Referenzen prüfen; Secret-Scan und DB-Historienprüfung auf dem finalen Stand wiederholen.
- [ ] Tests und beide Builds auf dem finalen Stand ausführen.
- [ ] Dry Run prüfen und ausschließlich `main` sowie bestätigte Tags veröffentlichen; Remote-Referenzen anschließend kontrollieren.

Automatische Release-Binaries und die Migration lokaler Tickets zu GitHub Issues sind keine Voraussetzung für die Erstveröffentlichung.

## Comments

### 2026-10-04 — Entscheidung mit Jan

Möglichst alles transparent veröffentlichen, mit höchster Priorität für Code und Dokumentation. Die Backup-Referenz lokal behalten und durch explizite Auswahl der Push-Referenzen von der Veröffentlichung ausschließen. Keine erneute Historienumschreibung erforderlich, solange die geprüfte DB-freie Historie veröffentlicht wird. Entscheidung ausdrücklich im Veröffentlichungsticket festhalten.

### 2026-10-04 — MIT und direkte Umsetzung

Jan wählt MIT, beauftragt die direkte Umsetzung der übrigen behebbaren Findings und möchte einen frischen Start mit blanker DB oder Template prüfen. Die oben dokumentierten Vorbereitungen sind umgesetzt; ein produktives Start-Template bleibt eine offene Entscheidung.
