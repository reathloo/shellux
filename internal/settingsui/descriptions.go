package settingsui

import "github.com/reathloo/shellux/internal/locale"

type helpText struct {
	english string
	german  string
}

var entryHelp = map[string]helpText{
	"platform":        {"Shows the operating system, CPU architecture and computer name.", "Zeigt Betriebssystem, CPU-Architektur und Rechnername."},
	"shell":           {"Shows the active shell and terminal application.", "Zeigt die aktive Shell und Terminal-Anwendung."},
	"uptime":          {"Shows how long the computer has been running since its last restart.", "Zeigt die Laufzeit des Computers seit dem letzten Neustart."},
	"date":            {"Shows or hides the date row. Its date, timezone and UTC choices remain saved.", "Blendet die Datumszeile ein oder aus. Datum, Zeitzone und UTC-Auswahl bleiben gespeichert."},
	"hostname":        {"Shows the computer name as its own row.", "Zeigt den Rechnernamen als eigene Zeile."},
	"network-status":  {"Shows or hides the connection-status row. Its selected network details remain saved.", "Blendet die Verbindungszeile ein oder aus. Gewählte Netzwerkdetails bleiben gespeichert."},
	"network-traffic": {"Shows or hides the traffic row for current transfer rates, latency and history.", "Blendet die Traffic-Zeile für aktuelle Datenraten, Latenz und Verlauf ein oder aus."},
	"ip":              {"Shows the active local IPv4 address as its own row.", "Zeigt die aktive lokale IPv4-Adresse als eigene Zeile."},
	"cpu":             {"Shows or hides CPU load and its selected details.", "Blendet die CPU-Auslastung und ihre gewählten Details ein oder aus."},
	"cores":           {"Shows or hides logical CPU-core information and total utilization.", "Blendet Informationen zu logischen CPU-Kernen und Gesamtauslastung ein oder aus."},
	"ram":             {"Shows or hides physical-memory usage and its selected details.", "Blendet die Arbeitsspeichernutzung und ihre gewählten Details ein oder aus."},
	"volume":          {"Shows or hides usage of the system volume.", "Blendet die Belegung des Systemlaufwerks ein oder aus."},
	"battery":         {"Shows or hides battery information. Battery and thermal state share one dashboard row.", "Blendet Batterieinformationen ein oder aus. Batterie und Thermik teilen eine Dashboard-Zeile."},
	"temperature":     {"Shows the system thermal-pressure state. With battery enabled, both appear on one row.", "Zeigt den thermischen Belastungszustand. Mit aktiver Batterie erscheinen beide in einer Zeile."},
	"directory":       {"Shows the current working directory, shortened to fit the dashboard.", "Zeigt das aktuelle Arbeitsverzeichnis, passend zur Dashboard-Breite gekürzt."},
	"spotify":         {"Shows artist and title from the Spotify desktop app, or that nothing is playing.", "Zeigt Interpret und Titel aus der Spotify-Desktop-App oder dass nichts wiedergegeben wird."},
	"progress":        {"Shows playback state, elapsed time and song progress for Spotify.", "Zeigt Wiedergabestatus, vergangene Zeit und Songfortschritt von Spotify."},
	"time":            {"Shows the current time in the dashboard title. It does not occupy a menu slot.", "Zeigt die aktuelle Uhrzeit im Dashboard-Titel. Sie belegt keinen Menüplatz."},
}

var componentHelp = map[string]map[string]helpText{
	"date": {
		"value":    {"Shows the local calendar date.", "Zeigt das lokale Kalenderdatum."},
		"timezone": {"Shows the local timezone name, for example CEST or Europe/Berlin.", "Zeigt den Namen der lokalen Zeitzone, zum Beispiel MESZ oder Europe/Berlin."},
		"utc":      {"Shows the current offset from UTC and follows daylight-saving changes.", "Zeigt den aktuellen Abstand zu UTC und berücksichtigt die Sommerzeit."},
	},
	"network-status": {
		"value":      {"Shows whether an active network connection is available.", "Zeigt, ob eine aktive Netzwerkverbindung verfügbar ist."},
		"interface":  {"Shows the active network-interface name, for example en0 or eth0.", "Zeigt den Namen der aktiven Netzwerkschnittstelle, zum Beispiel en0 oder eth0."},
		"ip":         {"Shows the local IPv4 address on the active interface.", "Zeigt die lokale IPv4-Adresse der aktiven Schnittstelle."},
		"connection": {"Shows the connection type, such as Wi-Fi or Ethernet.", "Zeigt die Verbindungsart, zum Beispiel WLAN oder Ethernet."},
		"name":       {"Shows the Wi-Fi network name when the operating system makes it available.", "Zeigt den WLAN-Namen, wenn das Betriebssystem ihn bereitstellt."},
	},
	"network-traffic": {
		"upload":    {"Shows the current outgoing transfer rate in B/s, KB/s or MB/s.", "Zeigt die aktuelle ausgehende Datenrate in B/s, KB/s oder MB/s."},
		"download":  {"Shows the current incoming transfer rate in B/s, KB/s or MB/s.", "Zeigt die aktuelle eingehende Datenrate in B/s, KB/s oder MB/s."},
		"latency":   {"Shows gateway latency in milliseconds. It is measured about every five seconds and may be unavailable.", "Zeigt die Gateway-Latenz in Millisekunden. Sie wird etwa alle fünf Sekunden gemessen und kann nicht verfügbar sein."},
		"sparkline": {"Shows total upload and download activity over the last minute, scaled to the highest value in that period.", "Zeigt die gesamte Upload- und Download-Aktivität der letzten Minute, skaliert auf den höchsten Wert dieses Zeitraums."},
	},
	"cpu": {
		"bar":       {"Shows CPU load as an eight-segment bar from empty to fully utilized.", "Zeigt die CPU-Last als Balken mit acht Stufen von leer bis voll ausgelastet."},
		"percent":   {"Shows CPU load per logical core as a percentage.", "Zeigt die CPU-Last pro logischem Kern als Prozentwert."},
		"value":     {"Shows the current one-minute CPU load average as a number.", "Zeigt den aktuellen CPU-Lastdurchschnitt der letzten Minute als Zahl."},
		"top":       {"Shows the process with the highest CPU average on a second row, refreshed about every five seconds.", "Zeigt den Prozess mit dem höchsten CPU-Mittelwert in einer zweiten Zeile, etwa alle fünf Sekunden aktualisiert."},
		"sparkline": {"Shows the CPU-load trend over the last minute in eight levels.", "Zeigt den CPU-Lastverlauf der letzten Minute in acht Stufen."},
	},
	"cores": {
		"bar":     {"Shows total measured CPU utilization as an eight-segment bar.", "Zeigt die gemessene CPU-Gesamtauslastung als Balken mit acht Stufen."},
		"percent": {"Shows total measured CPU utilization as a percentage.", "Zeigt die gemessene CPU-Gesamtauslastung als Prozentwert."},
		"value":   {"Shows the number of logical CPU cores.", "Zeigt die Anzahl der logischen CPU-Kerne."},
	},
	"ram": {
		"bar":       {"Shows used physical memory as an eight-segment bar.", "Zeigt den belegten Arbeitsspeicher als Balken mit acht Stufen."},
		"percent":   {"Shows the share of physical memory currently in use.", "Zeigt den aktuell belegten Anteil des Arbeitsspeichers als Prozentwert."},
		"value":     {"Shows the amount of physical memory currently in use in GiB.", "Zeigt den aktuell belegten Arbeitsspeicher in GiB."},
		"pressure":  {"Shows macOS memory pressure or Linux PSI. It displays N/A when the system provides no value.", "Zeigt den macOS-Speicherdruck oder Linux-PSI. Ohne verfügbaren Systemwert erscheint N/A."},
		"sparkline": {"Shows the physical-memory usage trend over the last minute.", "Zeigt den Verlauf der Arbeitsspeichernutzung während der letzten Minute."},
	},
	"volume": {
		"bar":     {"Shows used system-volume capacity as an eight-segment bar.", "Zeigt den belegten Speicher des Systemlaufwerks als Balken mit acht Stufen."},
		"percent": {"Shows the used share of the system volume as a percentage.", "Zeigt den belegten Anteil des Systemlaufwerks als Prozentwert."},
		"value":   {"Shows used and total system-volume capacity in GB.", "Zeigt belegten und gesamten Speicher des Systemlaufwerks in GB."},
	},
	"battery": {
		"bar":       {"Shows the current battery charge as an eight-segment bar.", "Zeigt den aktuellen Akkustand als Balken mit acht Stufen."},
		"percent":   {"Shows the current battery charge as a percentage.", "Zeigt den aktuellen Akkustand als Prozentwert."},
		"value":     {"Shows a lightning symbol while the battery is charging; otherwise it shows the charging state when no charge display is selected.", "Zeigt beim Laden ein Blitzsymbol; ohne gewählte Ladeanzeige wird sonst der Ladezustand als Text ausgegeben."},
		"remaining": {"Shows the estimated remaining battery runtime. It displays N/A while charging or when no estimate is available.", "Zeigt die geschätzte verbleibende Akkulaufzeit. Beim Laden oder ohne Schätzung erscheint N/A."},
		"sparkline": {"Shows the battery-charge trend over the last minute.", "Zeigt den Verlauf des Akkustands während der letzten Minute."},
	},
}

func (m *model) menuHelp(entry, part string) string {
	help := entryHelp[entry]
	if part != "" {
		help = componentHelp[entry][part]
	}
	if m.language() == locale.German {
		return help.german
	}
	return help.english
}
