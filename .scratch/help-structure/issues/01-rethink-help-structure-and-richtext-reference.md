# 01 — Hilfe neu strukturieren und Richtext-Referenz separieren

**Phase:** M3 — Bedienbarkeit der integrierten Hilfe

**Category:** discovery / UX

**Priority:** Medium

**Status:** needs-triage

**Erfasst:** 2026-10-19

## Auftrag und Problem

Jan meldet anhand eines Screenshots: Die Hilfe ist nicht bündig eingerückt und schwer zu lesen. Die Hilfe soll strukturiert werden; mindestens Richtext Editing soll eine eigene Hilfeseite erhalten. Die neue Aufteilung muss vorher gründlich brainstormt und kritisch hinterfragt (»gegrillt«) werden. Dieses Ticket beauftragt die Konzeption, noch keine Produktimplementierung.

Der Screenshot zeigt im Pane `HELP` dicht aufeinanderfolgende Richtext-Zeilen: Textobjekte, Absatz-/Code-Verhalten, Visual Modes, Links, lokale Undo-/Redo-Regeln, Recovery-Commands, Listen, Quotes und Inline-Syntax. Beschreibungen beginnen uneinheitlich; mehrere Regeln und Ausnahmen stehen in einer Zeile. Teilweise fehlt der Kontext, in welchem Modus eine Taste gilt. Ein saubereres Spaltenraster allein löst diese Informationsdichte nicht.

## Befund auf `main` vor Richtext-Integration

- `internal/tui/help.go` enthält eine einzige lange Liste manuell mit Leerzeichen ausgerichteter Strings. Schon vorhandene Einträge wie `count+motion`, `diw / daw` und benachbarte Zeilen haben unterschiedliche Beschreibungsanfänge.
- `helpCommand` in `internal/tui/commands.go` öffnet stets dieselbe Liste. `infoPaneRows` in `internal/tui/panes.go` reicht sie an den allgemeinen Textumbruch weiter; es gibt dort kein semantisches Layout für Taste, Modus und Erklärung.
- Die zusätzlichen Richtext-Zeilen des Screenshots stehen in diesem Checkout noch nicht in `help.go`. Ihre konkrete Implementierungsquelle und der Stand nach Integration der Richtext-Arbeit müssen vor Umsetzung geprüft werden. Keine vollständige Fehlerursache des Screenshots aus diesem Checkout ableiten.
- `docs/keybindings.md` trennt implementierte und vereinbarte Bindings; automatische Ableitung der Hilfe ist bisher zurückgestellt. Die [Richtext-Spec](../../description-rendering/spec.md) verlangt außerdem dauerhaft dokumentierte Markup-Regeln und einen Tutorial-Abschnitt. Hilfe, vollständige Referenz und Tutorial haben unterschiedliche Zwecke.

## Vorgaben und offene Entscheidungen

**Fest:** Bündige, gut lesbare Darstellung und mindestens eine separate Richtext-Hilfeseite. Reine Verschiebung derselben Textwand reicht nicht. Die Hilfe bleibt per Tastatur bedienbar und dokumentiert das Verhalten des jeweiligen Builds.

**Offen:** Weitere Seiten, Einstieg, Kontextbezug, Seitennavigation, genaue Gruppierung und Layout. Die folgenden Varianten und Seitennamen sind Diskussionsmaterial, keine beschlossenen Commands oder Architektur.

## Brainstorming: mögliche Aufteilungen

1. **Kleiner Einstieg plus Richtext-Seite:** Allgemeine Hilfe behält die übrigen Themen, erhält klare Gruppen und einen sichtbaren Richtext-Verweis. Geringster Navigationsaufwand; die übrige Seite könnte trotzdem zu lang bleiben.
2. **Themenindex mit wenigen Seiten:** Einstieg mit Grundlagen; weitere Themen für Navigation/Views, Tasks/Detail/Comments und Richtext. Kürzere Suchräume, aber mehr Wechsel und schwierige Zuordnung gemeinsam verwendeter Tasten.
3. **Kontextbezogener Einstieg mit Gesamtindex:** Aus dem Editor direkt zur passenden Hilfe, von überall zum Index. Schnell bei akuten Fragen; kann überraschend sein und darf globale Commands oder andere Themen nicht verstecken.

Als Ausgangspunkt für die separate Richtext-Seite prüfen: Orientierung und Moduswechsel; Navigation/Textauswahl; Textbearbeitung; Formatierung und Links; Listen/Quotes/Code; Speichern, lokale Undo-/Redo-Regeln und sichere Recovery. Kurze Beispiele statt langer, mit Semikola verketteter Regelsätze. Ob das Abschnitte einer Seite oder weitere Unterseiten werden, bleibt offen.

Gemeinsame Vim-Motions für Title, Plain-Text-Description und Comment Composer nicht ungeprüft als Richtext-exklusiv auslagern. Ebenso globale Task-Aktionen von lokalen Draft-Aktionen unterscheiden.

## Grilling: Entscheidungen an konkreten Fragen prüfen

- **Was sucht der Nutzer?** Soll man primär nach Aufgabe (»Link ändern«), Taste (`Space+l`) oder Modus suchen? Welche dieser Suchen muss ohne langes Scrollen funktionieren? Braucht V1 dafür Suche oder reicht ein guter Index?
- **Ist die Trennung sinnvoll?** Welche Information fehlt in der allgemeinen Hilfe nach dem Auslagern? Wie findet ein neuer Nutzer Richtext überhaupt? Welche Hinweise müssen an beiden Stellen kurz erscheinen, ohne Regeln doppelt zu pflegen?
- **Wie viel Navigation ist vertretbar?** Reicht Variante 1 bereits? Für jede zusätzliche Seite einen konkreten Suchvorgang nennen, der dadurch leichter wird. Nicht für jeden Pane automatisch eine Seite bauen.
- **Wie startet Hilfe?** Bleibt `:h` ein stabiler Einstieg oder folgt es dem Fokus? Themenargumente wie `:help richtext`, auswählbare Indexeinträge und Kontextlinks vergleichen, noch keine Syntax festlegen. Completion, unbekannte Themen und Rückweg mitdenken.
- **Wie kehrt man zurück?** Index, vorheriges Thema, erneutes Öffnen, Scrollposition und `:q` festlegen. Das Öffnen/Schließen aus laufenden Task-, Richtext- und Comment-Drafts muss Fokus und ungespeicherte Inhalte bewahren; das bestehende Pane-Verhalten dabei ausdrücklich prüfen.
- **Welche Regeln brauchen Beispiele?** `dd` in Absatz versus Code, `Enter` versus `Shift+Enter`, Normal versus Visual/Block-Auswahl, Inline-Markup versus eingefügter Literaltext, lokales Undo versus persistierte Task-Aktion und Recovery aus geschütztem HTML. Nicht alle Ausnahmen in eine Tabellenzeile quetschen.
- **Was bedeutet bündig bei wenig Platz?** Gemeinsamer Beginn der Beschreibungen, Fortsetzungszeilen mit klarer Zuordnung, lange Commands und breite Unicode-Zeichen berücksichtigen. Zweispaltige Darstellung bei genügend Breite gegen gestapelte Taste/Erklärung bei schmalem Pane vergleichen. Ein überlanges Command darf nicht die gesamte Seite unnötig einrücken.
- **Wie bleibt die Seite lesbar?** Überschriften, Leerraum und klare Moduslabels auch ohne Farbe; mehrere Aliasse und Aktionen gegebenenfalls aufteilen. Bei geringer Höhe darf Orientierung nicht hinter großen Kopfbereichen verschwinden. Kontext und Position nach Scrollen/Resize erkennbar halten.
- **Wie verhindert man Drift?** Zuständigkeit zwischen `help.go`, `docs/keybindings.md`, README und Tutorial festlegen. Gemeinsame strukturierte Einträge gegen manuelle Pflege abwägen; keinen Generator allein für diesen Layoutfehler einführen. Geplante Bindings nicht als verfügbar darstellen.

## Vorgehen und Abschluss der Konzeption

- [ ] Aktuelle Hilfe nach Richtext-Integration inventarisieren: Einträge, Modi, Aliasse, Doppelungen, fehlende und noch nicht implementierte Regeln. Screenshot mit dem tatsächlichen Stand abgleichen.
- [ ] Die drei Aufteilungsvarianten an denselben Suchaufgaben durchspielen: Task speichern, Kanban-Karte verschieben, Link bearbeiten, Code-Zeile löschen, blockiertes HTML behandeln und Hilfe wieder verlassen.
- [ ] Mindestens zwei konkrete Text-/Layoutentwürfe vergleichen, einschließlich separater Richtext-Seite. Breiten/Höhen z. B. 120×40 und 80×24 sowie einen durch Pane-Splitting deutlich schmaleren Hilfebereich prüfen; verfügbare Inhaltsfläche statt nur Terminalgröße beachten.
- [ ] Grilling-Fragen beantworten: Einwände, verworfene Alternativen und verbleibende Unsicherheiten im Ticket festhalten. Die einfachste ausreichende Aufteilung begründen.
- [ ] Gewählte Inhaltszuordnung, Einstieg, Navigation/Rückweg, Umbruchregeln und Pflegeverantwortung festhalten. Offene Produktentscheidungen mit Jan klären, bevor das Ticket als implementierbar gilt.
- [ ] Danach fokussierte Umsetzungstickets mit überprüfbaren Abnahmekriterien ableiten; kein vorschnelles `ready-for-agent`.

## Erwartete Abnahme der späteren Umsetzung

- Richtext besitzt eine eigene, aus der allgemeinen Hilfe klar auffindbare Seite; gemeinsame Editor-Grundlagen bleiben für Title und Comments auffindbar.
- Erklärungen beginnen im gewählten Layout bündig. Umbrüche, lange Tastenkombinationen und schmale/kurze Panes bleiben verständlich und schneiden keine Regeln ab.
- Modus, Aktion und wichtige Einschränkungen lassen sich zuordnen; Richtext ist in erkennbare Gruppen gegliedert statt eine zweite unstrukturierte Befehlsliste zu bilden.
- Repräsentative Suchaufgaben sind ohne vollständiges Durchscrollen der gesamten Hilfe lösbar; konkrete Wege und nötige Schritte im Entwurf dokumentieren und manuell bewerten.
- Öffnen, Themenwechsel, Scrollen, Resize und Schließen funktionieren per Tastatur; Drafts, Fokus-Rückkehr und bestehende Pane-Navigation erhalten gezielte Regressionstests. Layouttests prüfen sichtbares Ergebnis statt nur vorhandene Textfragmente.
- Hilfe und bestehender Keybinding-Vertrag stimmen für implementierte Funktionen überein. Keine Änderungen an Vikunja-API, SQLite-Schema oder Editor-Semantik zur Vereinfachung der Hilfetexte.

## Comments

- 2026-10-19 — Jan: »die Hilfe ist nicht bündig eingerückt«, »schwierig zu lesen«; Scratch-Ticket zum Brainstormen, Grilling und einer neuen Aufteilung gewünscht, mindestens Richtext separieren.
