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
	"media-activity":  {"Shows microphone and camera activity reported by macOS; unsupported devices show N/A.", "Zeigt von macOS gemeldete Mikrofon- und Kameraaktivität; nicht unterstützte Geräte erscheinen als N/A."},
	"network-status":  {"Shows or hides the connection-status row. Its selected network details remain saved.", "Blendet die Verbindungszeile ein oder aus. Gewählte Netzwerkdetails bleiben gespeichert."},
	"network-traffic": {"Shows or hides the traffic row for current transfer rates, latency and history.", "Blendet die Traffic-Zeile für aktuelle Datenraten, Latenz und Verlauf ein oder aus."},
	"ip":              {"Shows the active local IPv4 address as its own row.", "Zeigt die aktive lokale IPv4-Adresse als eigene Zeile."},
	"packet-loss":     {"Shows failed ICMP probes to the default gateway over the latest twelve checks; blocked ping can look like loss.", "Zeigt fehlgeschlagene ICMP-Proben zum Standard-Gateway über die letzten zwölf Prüfungen; blockierte Pings können wie Verlust wirken."},
	"cpu":             {"Shows or hides CPU load and its selected details.", "Blendet die CPU-Auslastung und ihre gewählten Details ein oder aus."},
	"cores":           {"Shows or hides logical CPU-core information and total utilization.", "Blendet Informationen zu logischen CPU-Kernen und Gesamtauslastung ein oder aus."},
	"ram":             {"Shows or hides physical-memory usage and its selected details.", "Blendet die Arbeitsspeichernutzung und ihre gewählten Details ein oder aus."},
	"load-average":    {"Shows system load averages over one, five and fifteen minutes; these are runnable-task counts, not CPU percentages.", "Zeigt Systemlast-Mittelwerte über eine, fünf und fünfzehn Minuten; das sind Anzahlen ausführbarer Aufgaben, keine CPU-Prozentwerte."},
	"swap":            {"Shows used, total and free swap space independently from physical RAM.", "Zeigt belegten, gesamten und freien Auslagerungsspeicher unabhängig vom Arbeitsspeicher."},
	"gpu":             {"Shows GPU model and hardware core count when macOS reports them; unsupported systems show N/A.", "Zeigt GPU-Modell und Anzahl der Hardware-Kerne, sofern macOS sie meldet; sonst erscheint N/A."},
	"volume":          {"Shows or hides usage of the system volume.", "Blendet die Belegung des Systemlaufwerks ein oder aus."},
	"disk-io":         {"Shows physical block-device read and write throughput since the previous sample.", "Zeigt Lese- und Schreibdurchsatz physischer Blockgeräte seit der letzten Messung."},
	"battery":         {"Shows or hides battery information. Battery and thermal state share one dashboard row.", "Blendet Batterieinformationen ein oder aus. Batterie und Thermik teilen eine Dashboard-Zeile."},
	"temperature":     {"Shows the system thermal-pressure state. With battery enabled, both appear on one row.", "Zeigt den thermischen Belastungszustand. Mit aktiver Batterie erscheinen beide in einer Zeile."},
	"git-branch":      {"Shows the current Git branch and repository in this shell's directory.", "Zeigt den aktuellen Git-Branch und das Repository im Verzeichnis dieser Shell."},
	"git-sync":        {"Shows commits ahead of and behind the configured upstream branch; no network request is made.", "Zeigt Commits vor und hinter dem konfigurierten Upstream-Branch; es erfolgt keine Netzwerkanfrage."},
	"runtimes":        {"Shows installed Go, Node, Python and Java versions reported by their commands.", "Zeigt die von den Befehlen gemeldeten Go-, Node-, Python- und Java-Versionen."},
	"dev-context":     {"Shows the selected local Docker and Kubernetes contexts without contacting a server.", "Zeigt die lokal gewählten Docker- und Kubernetes-Kontexte ohne Serverkontakt."},
	"last-command":    {"Shows duration and exit code of the last non-Shellux command from the sourced shell.", "Zeigt Dauer und Exit-Code des letzten Befehls außerhalb von Shellux aus der eingebundenen Shell."},
	"remote-identity": {"Shows the account and host of this shell.", "Zeigt Benutzerkonto und Rechner dieser Shell."},
	"ssh-link":        {"Shows the client and server endpoints reported by SSH, without network requests.", "Zeigt die von SSH gemeldeten Client- und Server-Endpunkte ohne Netzwerkanfragen."},
	"remote-terminal": {"Shows the SSH terminal device and terminal type.", "Zeigt das SSH-Terminalgerät und den Terminaltyp."},
	"directory":       {"Shows the current working directory, shortened to fit the dashboard.", "Zeigt das aktuelle Arbeitsverzeichnis, passend zur Dashboard-Breite gekürzt."},
	"spotify":         {"Shows artist and title in a scrolling field; the optional album stays fixed beside it.", "Zeigt Interpret und Titel als Laufschrift; das optionale Album steht fest daneben."},
	"progress":        {"Shows playback time and a seekable bar; previous and next buttons share one switch.", "Zeigt Wiedergabezeit und anklickbaren Balken; Vorheriger und Nächster teilen einen Schalter."},
	"logged-users":    {"Shows distinct users with local login sessions, not all running processes.", "Zeigt unterschiedliche Benutzer mit lokalen Anmeldesitzungen, nicht alle laufenden Prozesse."},
	"time":            {"Shows the current time in the dashboard title. It does not occupy a menu slot.", "Zeigt die aktuelle Uhrzeit im Dashboard-Titel. Sie belegt keinen Menüplatz."},
}

var componentHelp = map[string]map[string]helpText{
	"spotify": {
		"track": {"Shows the current artist and song title.", "Zeigt Interpret und Titel des aktuellen Songs."},
		"album": {"Shows the album fixed to the right of the scrolling track text.", "Zeigt das Album fest rechts neben der Laufschrift."},
	},
	"progress": {
		"playback": {"Shows playback time, the seek bar and the play/pause button.", "Zeigt Wiedergabezeit, Spulbalken und Start/Pause-Schaltfläche."},
		"skip":     {"Shows the previous and next buttons together.", "Zeigt Vorheriger und Nächster gemeinsam an."},
	},
	"git-branch": {
		"branch":     {"Shows the current branch, or a short commit ID when HEAD is detached.", "Zeigt den aktuellen Branch oder bei gelöstem HEAD eine kurze Commit-ID."},
		"repository": {"Shows the Git repository folder name.", "Zeigt den Ordnernamen des Git-Repositories."},
	},
	"git-sync": {
		"ahead":  {"Shows local commits not in the configured upstream branch.", "Zeigt lokale Commits, die im konfigurierten Upstream-Branch fehlen."},
		"behind": {"Shows upstream commits missing locally; this uses local refs only.", "Zeigt Upstream-Commits, die lokal fehlen; dafür werden nur lokale Referenzen genutzt."},
	},
	"runtimes": {
		"go":     {"Shows the active Go command version.", "Zeigt die Version des aktiven Go-Befehls."},
		"node":   {"Shows the active Node command version.", "Zeigt die Version des aktiven Node-Befehls."},
		"python": {"Shows the active Python 3 command version.", "Zeigt die Version des aktiven Python-3-Befehls."},
		"java":   {"Shows the active Java command version.", "Zeigt die Version des aktiven Java-Befehls."},
	},
	"dev-context": {
		"docker":     {"Shows the local Docker context selection.", "Zeigt den lokal gewählten Docker-Kontext."},
		"kubernetes": {"Shows the current kubectl context from local configuration.", "Zeigt den aktuellen kubectl-Kontext aus der lokalen Konfiguration."},
	},
	"last-command": {
		"duration": {"Shows elapsed whole seconds; commands under one second appear as <1s.", "Zeigt vergangene volle Sekunden; Befehle unter einer Sekunde erscheinen als <1s."},
		"exit":     {"Shows the exit code of the last command in this shell.", "Zeigt den Exit-Code des letzten Befehls in dieser Shell."},
	},
	"remote-identity": {
		"user": {"Shows the current operating-system account.", "Zeigt das aktuelle Betriebssystemkonto."},
		"host": {"Shows the current computer name.", "Zeigt den Namen des aktuellen Rechners."},
	},
	"ssh-link": {
		"client": {"Shows the SSH client's IP address and port.", "Zeigt IP-Adresse und Port des SSH-Clients."},
		"server": {"Shows the SSH server's IP address and port when available.", "Zeigt IP-Adresse und Port des SSH-Servers, wenn verfügbar."},
	},
	"remote-terminal": {
		"tty":  {"Shows the SSH terminal device, or N/A outside an SSH TTY.", "Zeigt das SSH-Terminalgerät oder N/A außerhalb eines SSH-TTY."},
		"term": {"Shows the TERM type used by the current shell.", "Zeigt den TERM-Typ der aktuellen Shell."},
	},
	"load-average": {
		"one":     {"Shows the one-minute runnable-task average.", "Zeigt den Mittelwert ausführbarer Aufgaben über eine Minute."},
		"five":    {"Shows the five-minute runnable-task average.", "Zeigt den Mittelwert ausführbarer Aufgaben über fünf Minuten."},
		"fifteen": {"Shows the fifteen-minute runnable-task average.", "Zeigt den Mittelwert ausführbarer Aufgaben über fünfzehn Minuten."},
	},
	"swap": {
		"value":     {"Shows used and total swap capacity in GB.", "Zeigt belegten und gesamten Auslagerungsspeicher in GB."},
		"percent":   {"Shows the used share of swap space; N/A if no swap is configured.", "Zeigt den belegten Anteil des Auslagerungsspeichers; ohne Swap erscheint N/A."},
		"remaining": {"Shows free swap capacity in GB.", "Zeigt freien Auslagerungsspeicher in GB."},
	},
	"gpu": {
		"name":  {"Shows the graphics processor model reported by macOS.", "Zeigt das von macOS gemeldete Grafikprozessormodell."},
		"cores": {"Shows the reported number of GPU cores, or N/A when unavailable.", "Zeigt die gemeldete Anzahl der GPU-Kerne oder N/A, falls unbekannt."},
	},
	"media-activity": {
		"microphone": {"Shows whether a macOS audio-input device reports running in any process; N/A means it cannot be determined.", "Zeigt, ob ein macOS-Audioeingabegerät Nutzung durch einen Prozess meldet; N/A bedeutet nicht bestimmbar."},
		"camera":     {"Shows whether a macOS camera device reports running in any process; N/A means it cannot be determined.", "Zeigt, ob ein macOS-Kameragerät Nutzung durch einen Prozess meldet; N/A bedeutet nicht bestimmbar."},
	},
	"packet-loss": {
		"percent": {"Shows the failed share of up to twelve gateway pings, sampled about every five seconds.", "Zeigt den Anteil fehlgeschlagener Gateway-Pings aus bis zu zwölf Proben, etwa alle fünf Sekunden gemessen."},
		"samples": {"Shows failed and total gateway probes so the percentage has visible context.", "Zeigt fehlgeschlagene und gesamte Gateway-Proben als Kontext für den Prozentwert."},
	},
	"disk-io": {
		"read":  {"Shows the host disk-read rate in B/s, KB/s or MB/s; the first sample starts at zero.", "Zeigt die Leserate der Host-Laufwerke in B/s, KB/s oder MB/s; die erste Messung beginnt bei null."},
		"write": {"Shows the host disk-write rate in B/s, KB/s or MB/s; the first sample starts at zero.", "Zeigt die Schreibrate der Host-Laufwerke in B/s, KB/s oder MB/s; die erste Messung beginnt bei null."},
	},
	"logged-users": {
		"count": {"Shows the number of distinct user names reported by local login sessions.", "Zeigt die Anzahl unterschiedlicher Benutzernamen aus lokalen Anmeldesitzungen."},
		"names": {"Shows the distinct local login names, shortened when the dashboard is narrow.", "Zeigt die verschiedenen lokalen Anmeldenamen, bei wenig Platz gekürzt."},
	},
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
		"bar":       {"Shows used system-volume capacity as an eight-segment bar.", "Zeigt den belegten Speicher des Systemlaufwerks als Balken mit acht Stufen."},
		"percent":   {"Shows the used share of the system volume as a percentage.", "Zeigt den belegten Anteil des Systemlaufwerks als Prozentwert."},
		"value":     {"Shows used and total system-volume capacity in GB.", "Zeigt belegten und gesamten Speicher des Systemlaufwerks in GB."},
		"remaining": {"Shows remaining available system-volume capacity in GB.", "Zeigt den verbleibenden verfügbaren Speicher des Systemlaufwerks in GB."},
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
