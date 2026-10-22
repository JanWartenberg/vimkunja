# 02 — GitHub-Veröffentlichung nach dem Richtext-Merge vorbereiten

**What to deliver:** Den zusammengeführten Stand für `JanWartenberg/vimkunja` technisch prüfen, eine Veröffentlichungskopie ohne private E-Mail-/DB-Historie vorbereiten und den konkreten Push prüfbar machen. Tatsächliche Veröffentlichung erst auf ausdrücklichen Auftrag.

**Category:** maintenance

**Priority:** High nach dem Richtext-Merge

**Status:** in-progress

**Startbedingung erfüllt:** Richtext und aufgabenbasierte Tutorials von `feature/tutorial-task-workflow` nach `main` übernommen.

**Reihenfolge:** Vor Abschluss der Veröffentlichung zuerst [URL-Copy-Komfort](../../../.archive/2026-10-20/description-link-copy-shortcut/issues/01-choose-accessible-url-copy-shortcut.md), dann [Subtask-Abgleich](../../../.archive/2026-10-20/release-ticket-cleanup/issues/01-reconcile-open-subtasks-before-publication.md). Andere M3-Features sind keine pauschale Voraussetzung.

**Erfüllte Dokumentationsvoraussetzung:** [Ticket 01 — Lizenz und Dokumentation](../../../.archive/2026-10-07/publish-to-github/issues/01-publish-to-github.md) ist abgeschlossen und archiviert.

**Erfüllte Voraussetzung:** [Vikunja-DB-Kompatibilität ausschließlich 2.7.0](../../../.archive/2026-10-05/vikunja-db-compatibility/issues/01-verify-vikunja-2-3-to-2-7-compatibility.md)

## Startbedingung und Reihenfolge

**Dieses Ticket folgt auf den vereinbarten Richtext-Merge.** Original-Repository und bestehende Worktrees behalten ihre Historie; die spätere Bereinigung findet ausschließlich in der Veröffentlichungskopie statt. Die Startbedingung verlangt den vereinbarten Worktree-Merge, keinen pauschalen Abschluss aller Richtext-Backlogtickets oder des gesamten Meilensteins 3.

1. Den zusammengeführten `main`-Commit festhalten; README/STATUS und Lizenzinventar auf Änderungen aus dem Merge abgleichen. Neue Funktionalität oder Abhängigkeiten erfordern nur die betroffenen Ergänzungen.
2. Tests, beide Builds, Vet, Schwachstellenprüfung und CI-Konfiguration auf diesem Stand prüfen. Vorhandene CI weiterverwenden; einen echten Actions-Lauf prüfen, sobald ein geeigneter GitHub-Prüfstand verfügbar ist. Native Windows-Abnahme getrennt von Cross-Builds ausweisen.
3. Eine separate Veröffentlichungskopie aus `main` und ausdrücklich ausgewählten Tags erstellen. E-Mail-/Historienbereinigung ausschließlich in dieser Kopie; Original-Repository, Worktrees und lokale Backups nicht umschreiben.
4. Author-/Committer-Adressen sowie persönliche E-Mail-Vorkommen in aktuellen und historischen Dateiinhalten bereinigen. Zieladresse: **`5298128+JanWartenberg@users.noreply.github.com`**. Der letzte Readiness-Check fand persönliche Metadaten in fünf Commits und zwei historische Dateistände des Veröffentlichungstickets; den aktuellen Bestand erneut ermitteln. Andere Autoren und Drittanbieter-Copyrightangaben nicht pauschal ersetzen.
5. Zugehörige Tags auf die bereinigte Historie übernehmen; vollständige erreichbare Historie und Tag-/Commit-Inhalte auf verbleibende private Adressen, Secrets und Datenbanken einschließlich Sidecars prüfen. Prüfergebnis und Commit-Zuordnung nachvollziehbar festhalten, ohne private Adressen neu in veröffentlichbaren Berichten zu dokumentieren.
6. GitHub-Ziel und Remote-Konfiguration prüfen; Dry Run mit expliziten Referenzen durchführen. Den tatsächlichen öffentlichen Push erst nach ausdrücklichem Auftrag durchführen und danach Remote-Referenzen kontrollieren.

## VERBINDLICH: Backup-Referenzen niemals veröffentlichen

**Nur `main` und ausdrücklich ausgewählte Tags veröffentlichen. `refs/original/refs/heads/main` und andere Backup-Referenzen dürfen NICHT auf GitHub gelangen. KEIN `git push --mirror`.**

- Ein sauberer Dateibaum oder `.gitignore` allein belegt keine saubere Historie; alle vorgesehenen Referenzen prüfen.
- Die lokale Backup-Referenz darf bestehen bleiben. Eine Historienumschreibung des Originals oder Löschung seiner Backups ist nicht erforderlich; die E-Mail-Bereinigung erfolgt nur in der Veröffentlichungskopie.
- Remote-Konfiguration auf Mirror-Modus und zusätzliche Push-Refspecs prüfen. Feature-Branches nicht automatisch veröffentlichen.
- Neue Arbeit nach dem Übergang auf Basis der bereinigten Historie fortsetzen. Alte Worktrees nicht unkontrolliert in die neue Historie mergen; der Übergang muss dokumentiert sein.
- Lokale Nutzerdatenbanken und ihre Backups nie übernehmen; Schema-Metadaten und ausdrücklich geprüfte synthetische Fixtures sind zulässig.

Beispiel für den Dry Run, wenn `origin` auf das geprüfte Ziel zeigt und beide historischen Tags zur Veröffentlichung ausgewählt sind:

```bash
git push --dry-run origin main refs/tags/MVP refs/tags/Milestone1
```

Nach Freigabe erfolgt der tatsächliche Push mit denselben expliziten Referenzen ohne `--dry-run`.

## Abnahme

- [ ] Ticket 01 abgeschlossen und vereinbarter Richtext-Merge enthalten; finaler Ausgangscommit dokumentiert.
- [ ] Aktueller Funktionsumfang, README/STATUS und gegebenenfalls neue Dependency-Lizenzhinweise nach dem Merge abgeglichen.
- [ ] Tests, beide Builds, Vet, Schwachstellenprüfung und Workflow-Prüfung erfolgreich; echter Actions-Lauf und Plattformabnahme bzw. verbleibende Grenzen dokumentiert.
- [x] Veröffentlichungskopie enthält nur `main`, keine Tags oder Backup-Referenzen; Original und bestehende Worktrees bleiben erhalten.
- [x] Persönliche Adresse aus Author-/Committer-Metadaten, Commit-/Tag-Inhalten und historischer Dateihistorie entfernt; Zieladresse und künftige lokale Git-Identität des Veröffentlichungsrepositories geprüft.
- [ ] Secret-, Datenschutz- und DB-Historienprüfung der vollständig erreichbaren Veröffentlichungshistorie erfolgreich; keine echten Nutzerdaten oder DB-Sidecars enthalten.
- [ ] Remote-Ziel, Mirror-Einstellungen, Push-Refspecs und expliziter Dry Run geprüft; konkreter Veröffentlichungsstand zur Freigabe bereit.
- [ ] Tatsächlicher öffentlicher Push ausdrücklich beauftragt, durchgeführt und Remote-Referenzen kontrolliert.
- [ ] Übergang auf die bereinigte Historie für weitere Arbeit dokumentiert.

## Grenzen

Dieses Ticket zu aktualisieren erteilt keinen Auftrag, jetzt zu mergen, die Historie umzuschreiben, Remotes anzulegen oder zu veröffentlichen. Automatische Release-Binaries, ein produktives Start-Template und die Migration lokaler Tickets zu GitHub Issues sind keine Voraussetzung für die Quellcode-Erstveröffentlichung. Besondere macOS-Binary-Lizenzbedingungen müssen vor einem entsprechenden späteren Release separat erfüllt werden.

## Bisherige Evidenz

Die datierten Lizenz-, CI-, Secret- und Referenzprüfungen bleiben im [bisherigen Gesamtticket, jetzt Ticket 01](../../../.archive/2026-10-07/publish-to-github/issues/01-publish-to-github.md#historische-befunde-des-bisherigen-gesamttickets). Die letzte erneute Prüfung des committed Stands `e77f00c` fand keine erreichbaren DB-Dateien oder Backup-Referenzen im frischen Veröffentlichungsklon; Secret- und Schwachstellenscan waren ohne Findings. Diese Ergebnisse ersetzen die Prüfung des später zusammengeführten und bereinigten Stands nicht.

## Comments

### 2026-10-07 — Nach Richtext-Merge starten

Jan beauftragt die Aufteilung des bisherigen Veröffentlichungstickets. Ticket 02 bündelt E-Mail-/Historienbereinigung, CI, Veröffentlichungskopie und Push-Prüfung nach dem vereinbarten Richtext-Worktree-Merge. Die GitHub-noreply-Adresse ist von Jan bestätigt. Keine operativen Veröffentlichungsschritte mit der Anlage dieses Tickets gestartet.

### 2026-10-07 — Ticket 01 abgeschlossen

Lizenz und Dokumentation für den aktuellen Stand sind geprüft und Ticket 01 ist archiviert. Die allein verbleibende Startbedingung dieses Tickets ist der vereinbarte Richtext-Merge; anschließend dessen konkrete Doku-/Dependency-Änderungen abgleichen. Noch keine operative Veröffentlichung begonnen.

### 2026-10-20 — Richtext-Integration und übernommene Folgeplanung

Die im Richtext-Worktree am 2026-10-19 festgehaltene Reihenfolge bleibt erhalten: URL-Copy-Komfort, Subtask-Abgleich, danach Veröffentlichung abschließen. AT-01–10 und der Richtext-Live-Roundtrip sind abgenommen; die separate allgemeine Tutorial-Abnahme bleibt teilweise offen. Die auf `main` bereits belegte DB-Kompatibilität ausschließlich mit Vikunja 2.7.0 und die abgeschlossene Lizenzprüfung werden durch ältere offene Checklisten im Featurebranch nicht zurückgesetzt. Das dort noch vorhandene Gesamtticket 01 wird beim Merge nicht wieder als offenes Ticket angelegt; dieses Ticket 02 führt die verbleibende Arbeit weiter. Kein öffentlicher Push oder Umschreiben der Originalhistorie ist damit beauftragt.


### 2026-10-20 — Subtask-Cleanup: Restarbeit übergeben

Der technische Ticket-Abgleich ist erledigt; zwölf abgeschlossene Tickets und
zwei Auditberichte sind archiviert. Der [Cleanup-Bericht](../../../.archive/2026-10-20/release-ticket-cleanup/issues/01-reconcile-open-subtasks-before-publication.md#reconciliation-result--2026-10-20)
trennt verbleibende manuelle Abnahmen (allgemeines Tutorial ab Aufgabe 9,
Comment-`~`, jüngster Satzzeichen-Fix) von bereits akzeptiertem Richtext/Layout.
URL-Copy läuft auf Jans Auftrag parallel und bleibt dort zu bearbeiten.

Die Veröffentlichungsarbeit bleibt offen: finale Arbeitsänderungen committen,
Ausgangscommit wählen, releasebezogene Prüfungen/CI, Veröffentlichungskopie mit
Metadaten-/Historienbereinigung, Secret-/DB-Scan und expliziter Push-Dry-Run.
`e978e76` belegt den abgeschlossenen Merge, ist wegen nachfolgender lokaler
Änderungen noch keine Zusage für den finalen Veröffentlichungsstand. Ein
öffentlicher Push bleibt separat freizugeben. Die obigen Release-Checkboxen
werden durch diesen Ticket-Abgleich nicht pauschal als erledigt markiert.

### 2026-10-20 — Subtask-Cleanup abgeschlossen

Jan bestätigt alle 20 Restabnahme-Checks PASS: allgemeines Tutorial 9–26,
Comment-`~` / `2~` und literale Satzzeichen. Cleanup, Tutorial- und
Editor-Motion-Ticket sind abgeschlossen und archiviert. Die vorherige
Restabnahme-Übergabe ist damit erledigt. Der HTML-Zusammenfassungsexport war
defekt; Jans explizite Chat-Rückmeldung ist der Abnahmenachweis. URL-Copy und
die oben aufgeführten Veröffentlichungsaufgaben bleiben separat.

Der akzeptierte Satzzeichen-Fix ist als `393a2b0` committet. Der zugehörige
Cleanup-Abschluss wird separat versioniert; ein finaler Veröffentlichungsstand
ist damit noch nicht ausgewählt.

### 2026-10-21 — URL-Copy integration

`feature/url-copy-key` (`c6d25a0`) is integrated into main: `Space+e` copies a
Link destination; `Space+Y` remains an alias. URL-Copy implementation and
Subtask-Cleanup are complete. Publication checks remain owned by this ticket;
this merge does not perform or authorize a public push.

### 2026-10-22 — Veröffentlichungsfokus und E-Mail-Bereinigung beauftragt

Jan stellt alle noch nicht begonnenen Backlog-Aufgaben zurück und priorisiert die Veröffentlichung. Die automatisierte Entfernung seiner privaten E-Mail aus der Historie ist ausdrücklich beauftragt. Umsetzung in einer separaten Veröffentlichungskopie mit der bestätigten GitHub-noreply-Adresse; Originalhistorie, Worktrees und lokale Backups bleiben erhalten. Zunächst nur `main`, keine Tags ohne ausdrückliche Auswahl und kein öffentlicher Push.

#### Ergebnis der Historienbereinigung

- Veröffentlichungskopie: `/workspace/vimkunja-publication`, eigenständiges Repository ohne Objekt-Alternates, Remotes oder Backup-Referenzen. Einzige Referenz: `refs/heads/main`; Tags weiterhin nicht ausgewählt.
- Ausgangscommit: `57c94e46645c6d8639f78e682f7fc365a5d19b03`; bereinigter Commit: `af4a50fbdce1362066ec1ae982b62ec309b9f80c`.
- Automatisiert mit `git-filter-repo` 2.47.0. Eine bestätigte private Identität ersetzt: Author-/Committer-Felder in fünf Commits und drei historische Blob-Versionen in den beiden früheren Pfaden von Veröffentlichungsticket 01. Commit-Nachrichten und Dateiinhalte wurden ebenfalls geprüft und bei Treffern ersetzt, einschließlich Groß-/Kleinschreibung und UTF-16. Andere Identitäten bleiben unverändert.
- Alle 190 Commits und beide Merges erhalten. Jeder Commit wurde gegen das Original verglichen: Parent-Zuordnung, Nachrichten, Metadaten, Dateinamen, Modi und sämtliche historischen Inhalte stimmen bis auf die beauftragte E-Mail-Ersetzung und daraus folgende Objekt-IDs überein. Keine weiteren Inhaltsänderungen.
- Alle 3.324 physisch vorhandenen Git-Objekte und die Rohdateien unter `.git` geprüft: kein Treffer der bestätigten privaten Adresse. Keine Datenbank-/Sidecar-Dateinamen oder SQLite-Dateisignaturen in erreichbaren Blobs gefunden. `git fsck --full` erfolgreich. Das ist kein vollständiger Secret-Scan oder allgemeiner Datenschutz-Audit.
- Künftige lokale Identität der Kopie: `Jan Wartenberg <5298128+JanWartenberg@users.noreply.github.com>`. Kein Remote eingerichtet; `push.default=nothing` verhindert einen unbeabsichtigten Standard-Push.
- Auf dem bereinigten Commit mit der lokal vorhandenen Go-1.27.1-Toolchain: `go test ./... ./.scratch/description-rendering/verification`, beide Builds und `go vet ./...` erfolgreich. Die Toolchain-Version allein belegt keine aktuelle Sicherheitsprüfung.
- Original-`main`, lokale Backups, andere Worktrees, Nutzerdatenbanken und der unversionierte Task-Copy-Spec bleiben erhalten. Die alte Historie enthält lokal weiterhin die private Adresse und darf nicht veröffentlicht werden.

Commit-Zuordnung: `/workspace/vimkunja-publication/.git/filter-repo/commit-map`.
Maschinenlesbare Prüfergebnisse: `/workspace/vimkunja-publication/.git/filter-repo/privacy-verification.json`.
Der geprüfte Ausgangscommit enthält die nachfolgenden Dokumentationsänderungen noch nicht. Der endgültige Veröffentlichungscommit ist nach Abschluss der Dokumentation festzuhalten und zu prüfen. Der sonstige Backlog ist zurückgestellt.

**Nächste Release-Schritte:** Aktualisierten Dokumentationsstand als Ausgangspunkt der finalen Release-Prüfung festhalten; aktuellen Schwachstellen-/Secret-Scan, CI-Konfiguration und gegebenenfalls Plattformgrenzen prüfen; GitHub-Ziel und expliziten Push-Dry-Run prüfen. Ein echter Actions-Lauf benötigt einen geeigneten GitHub-Prüfstand. Öffentlicher Push weiterhin nur auf ausdrücklichen Auftrag.

**Weiterarbeit:** Veröffentlichungsarbeit in der bereinigten Kopie fortsetzen. Alte Branches und Worktrees nur als Referenz verwenden; ein Merge der alten Historie würde private Metadaten wieder eintragen. Spätere Änderungen gezielt übertragen. Die Dokumentation der Historienbereinigung ist bereits übertragen; der unversionierte Task-Copy-Spec wurde nicht in die Veröffentlichungskopie übernommen.

### 2026-10-22 — Unabhängige Projektkennzeichnung

Jan beauftragt den README-Untertitel „An unofficial Vim-style terminal interface for Vikunja.“ und den Hinweis „Vimkunja is an independent project, not affiliated with or endorsed by Vikunja.“ Beide stehen direkt unter dem Projektnamen. Eine Zustimmung des Vikunja-Projekts zum Namen ist damit nicht dokumentiert. Die Veröffentlichungsnotizen sind auf den erreichten Bereinigungsstand abgeglichen; finale Prüfungen und öffentlicher Push bleiben offen.
