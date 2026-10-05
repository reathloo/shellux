# Shellux

Ein animierter System-Header für **Zsh und Bash unter macOS und Linux**.
Systeminfos, Musik und dein eigener Look direkt im Terminal – dein gewohnter Prompt bleibt erhalten.

![Shellux-Header mit goldenem Schriftzug, Systeminfos und Spotify-Anzeige](docs/images/shellux-header.png)

## Was Shellux kann

- **System im Blick:** CPU, RAM, Speicherplatz, Batterie, Netzwerk und mehr.
- **Individuelle Anzeige:** Menüpunkte und Details wie Balken, Prozentwerte oder Zeitzone einzeln ein- und ausschalten.
- **Eigener Look:** Animierte Themes mit passenden Farben und Hintergründen. Frei kombinieren und als eigenes Theme speichern.
- **Spotify:** Aktueller Titel und Wiedergabefortschritt unter macOS.
- **Visuelle Einstellungen:** Alles bequem per Tastatur im Terminal anpassen.

## Installation

Das passende Archiv von [GitHub Releases](https://github.com/reathloo/shellux/releases)
laden, entpacken und darin ausführen:

```sh
./install.sh
```

Der Installer kopiert nur die Dateien und zeigt anschließend die zwei Zeilen für
deine Shell-Konfiguration. Für Zsh sehen sie standardmäßig so aus:

```sh
export PATH="$HOME/.local/bin:$PATH"
source "$HOME/.local/share/shellux/shell/shellux.zsh"
```

Diese Zeilen in `~/.zshrc` eintragen. Für **Bash** stattdessen
`shellux.bash` in `~/.bashrc` laden. Danach einen neuen Terminal-Tab öffnen.
Die Datei `checksums.txt` im Release erlaubt die Prüfung des Downloads mit
`shasum -a 256` beziehungsweise `sha256sum`.

## Anpassen

```sh
shellux settings
```

Im Menü findest du **Menu items**, **Themes**, **Appearance** und **General**:
Anzeige zusammenstellen, Themes auswählen, eigene Farben setzen und das Aktualisierungsintervall ändern.

![Shellux-Settings mit Navigation und einzeln schaltbaren Menüpunkten](docs/images/shellux-settings.png)

Mit **Pfeiltasten** navigieren, mit **Tab** den Bereich wechseln und mit **Leertaste / Enter** umschalten oder auswählen.
**Ctrl+S** speichert, **Esc** bricht ab. Weitere Tasten stehen direkt im Menü.

Änderungen werden erst mit **Save** übernommen. Dabei wird der Header neu gestartet und der Scrollback geleert.
**Cancel** verwirft die Änderungen. Zusätzliche Themes werden erst beim Speichern heruntergeladen
und bleiben danach offline verfügbar. Standardtheme, allgemeine Styles und Grundfarben sind eingebaut.

## Im Alltag

| Befehl | Funktion |
| --- | --- |
| `shx` | Header zurückholen und Scrollback leeren |
| `shellux off` / `shellux on` | Header im aktuellen Tab aus- oder einschalten |
| `shellux doctor` | Installation prüfen |
| `shellux help` | Alle Befehle anzeigen |

Während du Commands ausführst, pausiert der Header, damit die Ausgabe vollständig lesbar und scrollbar bleibt.
Mit `shx` holst du ihn zurück (`shellux reload` geht ebenfalls).
Die bisherigen Anpassungs-Commands bleiben als Alternative zum Settings-Menü verfügbar.

## Gut zu wissen

- Bis zu **11 Menüplätze**; leere Kategorien verschwinden automatisch.
- Manche Messwerte hängen von System und Berechtigungen ab. Fehlende Werte erscheinen als `N/A`.
- Spotify benötigt die Desktop-App und gegebenenfalls macOS-Automatisierungsrechte.
- Thermal zeigt den thermischen Zustand, keine Temperatur in °C. Die optionale Einrichtung über `shellux thermal enable` benötigt Administratorrechte.
- Hintergrundfarben funktionieren nur, wenn dein Terminal Farbänderungen unterstützt.

## Entwicklung

Benötigt **Go 1.27.1 oder neuer**. Unter macOS werden außerdem die Xcode Command
Line Tools benötigt. Im geklonten Repository:

```sh
go mod verify
go run ./tools/theme-pack -check
go test ./...
go test -race ./...
go vet ./...
sh shell/shellux_test.sh
sh scripts/install_test.sh
```

Die [CI](.github/workflows/ci.yml) prüft zusätzlich die Shellintegration und das interaktive Settings-Menü unter macOS und Linux.

Eigene Theme-Pakete veröffentlichen: [Anleitung](themes/README.md).

## Lizenz

Shellux steht unter der [PolyForm Noncommercial License 1.0.0](LICENSE). Du darfst
den Code für nichtkommerzielle Zwecke verwenden, verändern und weitergeben.
Kommerzielle Nutzung oder ein Verkauf benötigen eine separate Erlaubnis des
Urhebers. Shellux ist damit **source-available**, aber keine Open-Source-Software
im formalen Sinn. Rechte an Grafiken und anderen Inhalten Dritter bleiben
unberührt.
