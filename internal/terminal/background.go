package terminal

// Background changes the terminal's default background through OSC 11.
// color must be a validated RGB hex value or default, which uses OSC 111.
// Terminal emulators may ignore these sequences.
func Background(color string) string {
	if color == "default" {
		return "\033]111\007"
	}
	return "\033]11;" + color + "\007"
}
