# 02 — GitHub-Veröffentlichung nach dem Richtext-Merge vorbereiten

**What to deliver:** Den zusammengeführten Stand für `JanWartenberg/vimkunja` technisch prüfen, eine Veröffentlichungskopie ohne private E-Mail-/DB-Historie vorbereiten und den konkreten Push prüfbar machen. Tatsächliche Veröffentlichung erst auf ausdrücklichen Auftrag.

**Category:** maintenance

**Priority:** High nach dem Richtext-Merge

**Status:** needs-triage

**Blocked by:** Merge der vereinbarten Richtext-Worktree-Arbeit nach `main` (aktuell Branch `feature/description-model`).

**Erfüllte Dokumentationsvoraussetzung:** [Ticket 01 — Lizenz und Dokumentation](../../../.archive/2026-10-07/publish-to-github/issues/01-publish-to-github.md) ist abgeschlossen und archiviert.

**Erfüllte Voraussetzung:** [Vikunja-DB-Kompatibilität ausschließlich 2.7.0](../../../.archive/2026-10-05/vikunja-db-compatibility/issues/01-verify-vikunja-2-3-to-2-7-compatibility.md)

## Startbedingung und Reihenfolge

**Dieses Ticket beginnt erst nach dem vereinbarten Richtext-Merge.** Laufende Original-Worktrees und ihre Historie bleiben bis zum Zusammenführen unverändert. Der Merge selbst wird hier nicht vorzeitig ausgeführt. Die Startbedingung verlangt den vereinbarten Worktree-Merge, keinen pauschalen Abschluss aller Richtext-Backlogtickets oder des gesamten Meilensteins 3.

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
- [ ] Veröffentlichungskopie enthält nur `main` und bestätigte Tags, keine Backup-Referenzen; Original und bestehende Worktrees bleiben erhalten.
- [ ] Persönliche Adresse aus Author-/Committer-Metadaten, Commit-/Tag-Inhalten und historischer Dateihistorie entfernt; Zieladresse und künftige lokale Git-Identität des Veröffentlichungsrepositories geprüft.
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
