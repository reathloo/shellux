# Theme-Pakete

Optionale Theme-Downloads im bestehenden Shellux-Repository.
`catalog.json` enthält Namen, Versionen, Downloadgrößen, SHA-256-Prüfsummen und
Farben. `packages/` enthält die zugehörigen ZIP-Archive mit der Endung
`.shellux-theme`. Nur `shelluxdefault` ist fest eingebaut. Allgemeine Styles (`luxury`, `neon`,
`aurora`, `rainbow`) und die Grundfarben bleiben ebenfalls integriert.

Im Settings-Menü erscheinen alle Katalog-Themes mit Downloadgröße oder als
installiert. Die Auswahl ändert zunächst nur den Entwurf. Save lädt die Pakete
für die aktive Kombination; erst danach wird die Konfiguration gespeichert.
Esc stoppt einen laufenden Download und behält den Entwurf. Bei Downloadfehlern
oder externen Konfigurationsänderungen bleibt das Menü offen.

Der Katalog wird beim ersten Öffnen des Theme-Bereichs im Hintergrund aktualisiert.
Ohne Verbindung bleibt die mitgelieferte beziehungsweise gespeicherte Liste nutzbar.
Der normale Header startet ohne Netzwerkzugriff. Fehlt eine zuvor ausgewählte
Animation, verwendet er vorübergehend das Standardtheme und erhält die Auswahl.

## Neues Theme vorbereiten

Eine Arbeitsmappe außerhalb des Repositorys enthält `theme.json` und ANSI-Frames:

```text
mein-theme/
├── theme.json
└── frames/
    ├── 0000.ans
    └── 0001.ans
```

```json
{
  "format": 1,
  "name": "mein-theme",
  "version": "1.0.0",
  "background": "#101018",
  "style": {
    "secondary": "#a090d0",
    "accent": "#f0d0ff",
    "muted": "#706080"
  },
  "width": 51,
  "height": 26,
  "frames": [
    {"file": "frames/0000.ans", "duration_ms": 80},
    {"file": "frames/0001.ans", "duration_ms": 120}
  ]
}
```

Frames müssen bereits auf 51 Spalten und 26 Zeilen ausgelegt sein. Erlaubt sind
UTF-8-Zeichen, Zeilenumbrüche sowie ANSI-Reset, Standardfarben und RGB-Farben.
Transparente Bereiche verwenden Leerzeichen/Standardhintergrund.
Das Werkzeug prüft das Format und speichert gleiche Frames nur einmal.

Im Repository ausführen:

```sh
go run ./tools/theme-pack -source /absoluter/pfad/mein-theme
go run ./tools/theme-pack -check
go test ./...
```

Das erstellt `themes/packages/mein-theme-1.0.0.shellux-theme` und aktualisiert
den Katalog automatisch. Paket und Katalog werden gemeinsam veröffentlicht.
Für veränderte Inhalte muss die Version erhöht werden; bestehende Paketdateien
werden bei abweichendem Inhalt nicht überschrieben. Ältere Pakete behalten.

Für lokale Entwicklung und Tests lässt sich ein einzelnes Paket ohne Netzwerk
installieren, ohne die aktive Auswahl zu verändern:

```sh
go run ./tools/theme-pack -install slotmaschine
```

## Download-Modul

`internal/themepack` arbeitet unabhängig vom Settings-Menü:

- `Refresh(ctx)` lädt ausschließlich den Katalog und ersetzt den Cache erst
  nach erfolgreicher Prüfung. `CachedCatalog()` lädt ihn offline.
- `Install(ctx, entry)` verwendet ein vorhandenes geprüftes Paket oder lädt es
  herunter. Prüfsumme, Dateigröße, Identität, Farben, Frames und Timing werden
  vor der atomaren Installation geprüft. Abbruch erfolgt über den Kontext.
- `LoadName(name)` lädt die lokal festgehaltene Paketversion. Das Ergebnis erfüllt das bestehende
  Animation-Interface; die Wiedergabe benötigt keine weiteren Dateizugriffe.

Die Pakete werden nicht entpackt. Unterstützt werden Format 1, höchstens 8 MiB
Archivdaten und 32 MiB entpackte Daten, 1000 eindeutige Frames und zwei Minuten
Laufzeit. Fremde Dateien, doppelte Pfade, Links und Terminalsteuerbefehle werden
abgelehnt. HTTPS-Downloads sind zeitlich und in ihrer Größe begrenzt.

Der vorgesehene Katalog liegt unter:
`https://raw.githubusercontent.com/reathloo/shellux/main/themes/catalog.json`.
Pakete werden relativ zu diesem Katalog geladen. Eine öffentliche Bereitstellung
ist für Downloads ohne Benutzerkonto erforderlich. Diese Umstellung
ändert die Sichtbarkeit des Repositorys nicht.

Installationsort: `$XDG_DATA_HOME/shellux/themes`, wenn gesetzt; sonst unter
macOS `~/Library/Application Support/shellux/themes`, unter Linux
`~/.local/share/shellux/themes`.

## Veröffentlichung und Versionen

Katalog und Pakete müssen gemeinsam auf dem konfigurierten Branch veröffentlicht
werden. Solange sie dort fehlen oder das Repository privat ist, schlagen anonyme
Downloads fehl; die UI behält dann den Entwurf und die bisherige Konfiguration.
Ein Push allein ändert die Sichtbarkeit des Repositorys nicht.

Für einen eigenen HTTPS-Katalog kann `SHELLUX_THEME_CATALOG_URL` gesetzt werden.
Relative Paketpfade werden unter derselben Adresse aufgelöst. Dieser Override
wird auch für isolierte Tests genutzt.

Installierte Versionen bleiben lokal festgehalten. Katalogupdates ersetzen sie
nicht automatisch. Eine eigene Oberfläche für Paketupdates und Deinstallation
ist noch nicht Teil dieser Umsetzung. Eigene gespeicherte Theme-Kombinationen
bleiben erhalten; heruntergeladen werden nur Pakete der aktiven Kombination.

Die Tests prüfen Pakete, Offline-Nutzung, Downloadfehler, Abbruch und
Konfigurationskonflikte. Der PTY-Test prüft Zsh/Bash mit einem lokal installierten
Testpaket und ohne Zugriff auf persönliche Theme-Daten.
