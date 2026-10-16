// Package locale resolves the user's display language without changing command names.
package locale

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Language string

const (
	English Language = "en"
	German  Language = "de"
)

func Valid(value string) bool {
	return value == "" || value == "auto" || value == "en" || value == "de"
}

// Resolve honors an explicit choice, then POSIX locale precedence. On macOS,
// AppleLanguages is used when no language locale was supplied by the terminal.
func Resolve(value string) Language {
	if value == "de" {
		return German
	}
	if value == "en" {
		return English
	}
	return detect(os.Getenv, systemLanguage)
}
func detect(getenv func(string) string, fallback func() string) Language {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			return parse(value)
		}
	}
	return parse(fallback())
}
func parse(value string) Language {
	value = strings.ToLower(strings.TrimSpace(value))
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '_' || r == '-' || r == '.' || r == '@' })
	if len(parts) > 0 && parts[0] == "de" {
		return German
	}
	return English
}

var systemLanguage = sync.OnceValue(func() string {
	if runtime.GOOS != "darwin" {
		return "en"
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	data, err := exec.CommandContext(ctx, "/usr/bin/defaults", "read", "-g", "AppleLanguages").Output()
	if err != nil {
		return "en"
	}
	fields := strings.FieldsFunc(string(data), func(r rune) bool {
		return r == '(' || r == ')' || r == '"' || r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	if len(fields) > 0 {
		return fields[0]
	}
	return "en"
})

// Text translates only known application text, never user content.
func (l Language) Text(text string) string {
	if l == German {
		if translated, ok := german[text]; ok {
			return translated
		}
	}
	return text
}

var german = map[string]string{
	"Date": "Datum", "Network name": "Netzwerkname", "UTC offset": "UTC-Abstand", "Latency": "Latenz",

	"Preparing selected themes… Esc: stop download":               "Themes werden vorbereitet… Esc: Download stoppen",
	"Theme catalog offline; showing saved choices.":               "Theme-Katalog offline; gespeicherte Auswahl wird angezeigt.",
	"Download cancelled. Your changes are preserved.":             "Download abgebrochen. Änderungen bleiben erhalten.",
	"Catalog themes are protected.":                               "Katalog-Themes sind geschützt.",
	"Could not save: ":                                            "Speichern fehlgeschlagen: ",
	"Could not download: ":                                        "Download fehlgeschlagen: ",
	"Theme update check failed: ":                                 "Theme-Updateprüfung fehlgeschlagen: ",
	"Updated %d installed theme(s).":                              "%d installierte Themes aktualisiert.",
	"Select and save another theme before uninstalling this one.": "Vor dem Deinstallieren ein anderes Theme wählen und speichern.",
	"Uninstall downloaded theme %q now?":                          "Heruntergeladenes Theme %q jetzt deinstallieren?",
	"Could not uninstall theme: ":                                 "Theme konnte nicht deinstalliert werden: ",
	"Theme %q uninstalled.":                                       "Theme %q deinstalliert.",

	"Selected: ":                 "Ausgewählt: ",
	"%d / 11 slots":              "%d / 11 Plätze",
	"  ·  %d/11 slots":           "  ·  %d/11 Plätze",
	"%d–%d / %d · ↑/↓ to scroll": "%d–%d / %d · ↑/↓ zum Scrollen",
	"Shellux · Settings\n\nResize the window to at least 64 × 18.\nYour changes are preserved.\nEsc: cancel · Ctrl+C: discard": "Shellux · Einstellungen\n\nFenster auf mindestens 64 × 18 vergrößern.\nÄnderungen bleiben erhalten.\nEsc: abbrechen · Ctrl+C: verwerfen",
	"Unsaved changes\n\n":                             "Ungespeicherte Änderungen\n\n",
	"\n←/→: select · Enter: confirm · Esc: back":      "\n←/→: Auswahl · Enter: bestätigen · Esc: zurück",
	"Name: a–z, 0–9 and hyphens (1–32 characters).":   "Name: a–z, 0–9 und Bindestriche (1–32 Zeichen).",
	"Name: a–z, 0–9 and hyphens, up to 32 characters": "Name: a–z, 0–9 und Bindestriche, bis zu 32 Zeichen",
	"Not charging":       "Lädt nicht",
	"built-in":           "integriert",
	"custom":             "eigenes Theme",
	"Installed":          "Installiert",
	"Download · %.0f KB": "Download · %.0f KB",
	"Animation: %s · Style: %s · Background: %s":          "Animation: %s · Stil: %s · Hintergrund: %s",
	" · Delete: remove custom theme":                      " · Entf: eigenes Theme löschen",
	" · F: favorite":                                      " · F: favorisieren",
	" · U: uninstall now (keeps theme in catalog)":        " · U: jetzt deinstallieren (bleibt im Katalog)",
	" · Downloaded only when you save.":                   " · Download erst beim Speichern.",
	"Current: %s · %s · %s. Save as a custom theme.":      "Aktuell: %s · %s · %s. Als eigenes Theme speichern.",
	"Random: %s · %s · %s":                                "Zufällig: %s · %s · %s",
	"Overwrite custom theme %q?":                          "Eigenes Theme %q überschreiben?",
	"Delete custom theme %q?":                             "Eigenes Theme %q löschen?",
	"Overwrite menu preset %q?":                           "Menüvorlage %q überschreiben?",
	"Delete menu preset %q?":                              "Menüvorlage %q löschen?",
	"This theme name is protected. Choose a custom name.": "Dieser Theme-Name ist geschützt. Wähle einen eigenen Namen.",
	"Theme names: 1–32 letters (a–z), digits or hyphens; no leading or trailing hyphen.":  "Theme-Namen: 1–32 Buchstaben (a–z), Ziffern oder Bindestriche; kein Bindestrich am Anfang oder Ende.",
	"Layout names: 1–32 letters (a–z), digits or hyphens; no leading or trailing hyphen.": "Vorlagennamen: 1–32 Buchstaben (a–z), Ziffern oder Bindestriche; kein Bindestrich am Anfang oder Ende.",

	"SYSTEM": "SYSTEM", "NETWORK": "NETZWERK", "RESOURCES": "RESSOURCEN", "SESSION": "SITZUNG", "HEADER": "KOPFZEILE",
	"platform": "Plattform", "uptime": "Laufzeit", "date": "Datum", "hostname": "Hostname", "traffic": "Traffic", "cores": "Kerne", "memory": "RAM", "ram": "RAM", "volume": "Speicher", "battery": "Batterie", "thermal": "Thermik", "path": "Pfad", "progress": "Fortschr.", "time": "Uhrzeit",
	"Connected": "Verbunden", "Disconnected": "Nicht verbunden", "No song playing": "Keine Wiedergabe", "Normal": "Normal", "Warning": "Warnung", "Critical": "Kritisch", "Nominal": "Normal", "Fair": "Erhöht", "Serious": "Hoch", "load": "Last",
	"Menu items": "Menüpunkte", "Menu layouts": "Menüvorlagen", "Themes": "Themes", "Appearance": "Aussehen", "General": "Allgemein", "Language": "Sprache", "Automatic (system)": "Automatisch (System)", "Select language": "Sprache wählen",
	"Refresh interval": "Aktualisierung", "Animation": "Animation", "Style": "Stil", "Background": "Hintergrund", "Saved": "Gespeichert", "Unsaved": "Ungespeichert", "Notice": "Hinweis", "Enabled": "Aktiviert", "On": "An",
	"Bar": "Balken", "History": "Verlauf", "Percent": "Prozent", "Value": "Wert", "Top process": "Top-Prozess", "Pressure": "Speicherdruck", "Charging": "Laden", "Remaining": "Restzeit", "Connection": "Verbindung", "Name": "Name", "Timezone": "Zeitzone", "Interface": "Interface",
	"Reset menu items": "Menüpunkte zurücksetzen", "Default menu": "Standardmenü", "Save current menu as a layout…": "Aktuelles Menü als Vorlage speichern…", "Random combination": "Zufällige Kombination", "Save current combination as a theme…": "Kombination als Theme speichern…",
	"Save custom theme": "Eigenes Theme speichern", "Save menu layout": "Menüvorlage speichern", "Custom RGB color": "Eigene RGB-Farbe", "Custom RGB color…": "Eigene RGB-Farbe…", "Profile color": "Profilfarbe",
	"Select animation": "Animation wählen", "Select style": "Stil wählen", "Select background": "Hintergrund wählen",
	"Keep editing": "Weiter bearbeiten", "Discard": "Verwerfen", "Back": "Zurück", "Confirm": "Bestätigen", "Discard unsaved changes?": "Ungespeicherte Änderungen verwerfen?",
	"[ Save ]": "[ Speichern ]", "[ Cancel ]": "[ Abbrechen ]", "[Keep editing]": "[Weiter bearbeiten]", "[Discard]": "[Verwerfen]",
	"SHELLUX  ·  Settings": "SHELLUX  ·  Einstellungen", "Changes are applied only when you save.": "Änderungen werden erst beim Speichern übernommen.",
	"Tab / Shift+Tab: focus · ↑/↓: select · Enter: open":                                                           "Tab / Shift+Tab: Bereich · ↑/↓: Auswahl · Enter: öffnen",
	"←/→: switch · Space: toggle · Tab: focus":                                                                     "←/→: Schalter · Leertaste: umschalten · Tab: Bereich",
	"Enter/Esc: done · ↑/↓: item/section · ←/→: switch":                                                            "Enter/Esc: fertig · ↑/↓: Punkt/Bereich · ←/→: Schalter",
	"M: move section · ↑/↓: select":                                                                                "M: Bereich verschieben · ↑/↓: auswählen",
	"M: move item · →: switches · Space: toggle":                                                                   "M: Punkt verschieben · →: Schalter · Leertaste: an/aus",
	"Space: toggle · ←/→: select · M: move item":                                                                   "Leertaste: an/aus · ←/→: auswählen · M: Punkt verschieben",
	"Shift+←/→: move switch · Space: toggle · M: move item":                                                        "Shift+←/→: Schalter verschieben · Leertaste: an/aus · M: Punkt verschieben",
	"Press M, then use Up/Down to move this complete section.":                                                     "M drücken, dann diesen Bereich mit ↑/↓ verschieben.",
	"↑/↓: theme · Enter: apply · Delete: remove custom theme":                                                      "↑/↓: Theme · Enter: wählen · Entf: eigenes Theme löschen",
	"↑/↓: theme · Enter: apply · F: favorite · U: uninstall":                                                       "↑/↓: Theme · Enter: wählen · F: Favorit · U: deinstallieren",
	"↑/↓: layout · Enter: apply/save · Delete: remove saved layout":                                                "↑/↓: Vorlage · Enter: laden/speichern · Entf: eigene Vorlage löschen",
	"Ctrl+S: save · Esc: cancel · Ctrl+C: discard":                                                                 "Ctrl+S: speichern · Esc: abbrechen · Ctrl+C: verwerfen",
	"Enter: apply · Esc: back":                                                                                     "Enter: übernehmen · Esc: zurück",
	"Enter: apply · Esc: back · Ctrl+C: discard":                                                                   "Enter: übernehmen · Esc: zurück · Ctrl+C: verwerfen",
	"↑/↓: select · Enter: apply · Esc: back":                                                                       "↑/↓: Auswahl · Enter: übernehmen · Esc: zurück",
	"←/→ or Tab: select · Enter: confirm · Esc: back":                                                              "←/→ oder Tab: Auswahl · Enter: bestätigen · Esc: zurück",
	"This change is applied permanently only when you save.":                                                       "Diese Änderung wird erst beim Speichern dauerhaft übernommen.",
	"The file stays unchanged until you save.":                                                                     "Die Datei bleibt bis zum Speichern unverändert.",
	"Choose the dashboard and settings language. Automatic follows your system locale.":                            "Sprache für Header und Einstellungen. Automatisch folgt der Systemsprache.",
	"A positive Go duration, such as 1s, 500ms or 2.5s.":                                                           "Eine positive Go-Zeitdauer, z. B. 1s, 500ms oder 2.5s.",
	"Show or hide an item; its component choices are preserved.":                                                   "Eintrag ein- oder ausblenden; Detailauswahl bleibt erhalten.",
	"Battery and thermal share one slot.":                                                                          "Batterie und Thermik teilen einen Platz.",
	"Shares a slot with thermal. Remaining: estimated battery runtime; N/A while charging.":                        "Teilt einen Platz mit Thermik. Restzeit: geschätzte Akkulaufzeit; N/A beim Laden.",
	"Spotify and progress share one slot.":                                                                         "Spotify und Fortschritt teilen einen Platz.",
	"The clock in the header does not use a menu slot.":                                                            "Die Uhr in der Kopfzeile belegt keinen Menüplatz.",
	"Date, timezone name and UTC offset are independent. The offset follows daylight saving time.":                 "Datum, Zeitzone und UTC-Abstand sind unabhängig. Der Abstand berücksichtigt die Sommerzeit.",
	"CPU: load per core. Top: separate ps CPU average, shown below CPU; refreshed every 5s.":                       "CPU: Last pro Kern. Top: separater CPU-Mittelwert von ps, unter CPU; alle 5s aktualisiert.",
	"Pressure: macOS memory pressure level or Linux 10s PSI; N/A if unavailable.":                                  "Speicherdruck: macOS-Stufe oder Linux-PSI über 10s; N/A falls nicht verfügbar.",
	"Connection type and network name are independent; name may be unavailable.":                                   "Verbindungsart und Netzwerkname sind unabhängig; der Name kann fehlen.",
	"Upload, download and latency share one slot. Gateway ping every 5s; N/A without a response.":                  "Upload, Download und Latenz teilen einen Platz. Gateway-Ping alle 5s; N/A ohne Antwort.",
	"No components enabled – hidden in the header":                                                                 "Keine Details aktiv – im Header ausgeblendet",
	"Reset items and components only; appearance stays unchanged.":                                                 "Nur Einträge und Details zurücksetzen; Aussehen bleibt erhalten.",
	"Reset items, components and their order; appearance stays unchanged.":                                         "Einträge, Details und ihre Reihenfolge zurücksetzen; Aussehen bleibt erhalten.",
	"Apply this menu layout: visible items, detail switches and order. Theme and general settings stay unchanged.": "Diese Vorlage lädt sichtbare Menüpunkte, Detailschalter und Reihenfolge. Theme und allgemeine Einstellungen bleiben erhalten.",
	" Delete removes only this saved layout.":                                                                      " Entf löscht nur diese gespeicherte Vorlage.",
	"Restore the built-in menu layout. Saved layouts stay available.":                                              "Das Standardmenü wiederherstellen. Gespeicherte Vorlagen bleiben erhalten.",
	"Save visible menu items, detail switches and order under a name; theme and general settings stay unchanged.":  "Sichtbare Menüpunkte, Detailschalter und Reihenfolge unter einem Namen speichern; Theme und allgemeine Einstellungen bleiben erhalten.",
	"Choose an animation; style and background stay unchanged.":                                                    "Animation wählen; Stil und Hintergrund bleiben erhalten.",
	"Choose header colors independently of the theme.":                                                             "Header-Farben unabhängig vom Theme wählen.",
	"Named colors, theme backgrounds, profile color or a custom RGB color.":                                        "Farbnamen, Theme-Hintergründe, Profilfarbe oder eigene RGB-Farbe.",
	"Combine an animation with a random style and matching background.":                                            "Animation mit zufälligem Stil und passendem Hintergrund kombinieren.",
	"Menu items and components reset.":                                                                             "Menüpunkte und Details zurückgesetzt.",
	"Menu order changed. Save to apply it.":                                                                        "Menüreihenfolge geändert. Zum Übernehmen speichern.",
	"Component order changed. Save to apply it.":                                                                   "Detailreihenfolge geändert. Zum Übernehmen speichern.",
	"Already at the edge of this section.":                                                                         "Am Rand dieses Bereichs bereits angekommen.",
	"Already at the edge of this item.":                                                                            "Am Rand dieses Punktes bereits angekommen.",
	"The header clock is fixed in the title.":                                                                      "Die Uhr ist fest im Titel verankert.",
	"All 11 slots are taken. Hide an item first.":                                                                  "Alle 11 Plätze belegt. Blende zuerst einen Eintrag aus.",
	"File changed externally. Please cancel and reopen settings.":                                                  "Datei extern geändert. Bitte abbrechen und Einstellungen erneut öffnen.",
	"Enter a positive duration, such as 1s or 500ms.":                                                              "Positive Zeitdauer eingeben, z. B. 1s oder 500ms.",
	"Enter an RGB color in #RRGGBB format.":                                                                        "RGB-Farbe im Format #RRGGBB eingeben.",
}

func init() {
	german["History shows the last minute. CPU: load per core. Top: separate ps CPU average, shown below CPU; refreshed every 5s."] = "Verlauf zeigt die letzte Minute. CPU: Last pro Kern. Top: separater CPU-Mittelwert von ps, unter CPU; alle 5s aktualisiert."
	german["History shows the last minute. Pressure: macOS memory pressure level or Linux 10s PSI; N/A if unavailable."] = "Verlauf zeigt die letzte Minute. Speicherdruck: macOS-Stufe oder Linux-PSI über 10s; N/A falls nicht verfügbar."
	german["History shows the last minute. Shares a slot with thermal. Remaining: estimated battery runtime; N/A while charging."] = "Verlauf zeigt die letzte Minute. Teilt einen Platz mit Thermik. Restzeit: geschätzte Akkulaufzeit; N/A beim Laden."
	german["History shows total traffic over the last minute. Upload, download and latency share one slot."] = "Verlauf zeigt den gesamten Traffic der letzten Minute. Upload, Download und Latenz teilen einen Platz."
}
