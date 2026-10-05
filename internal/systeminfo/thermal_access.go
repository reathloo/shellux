package systeminfo

import (
	"fmt"
	"regexp"
)

const thermalSudoersPath = "/etc/sudoers.d/shellux-powermetrics"

var unixUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func thermalSudoersContent(username string) (string, error) {
	if !unixUsernamePattern.MatchString(username) {
		return "", fmt.Errorf("invalid local username %q", username)
	}
	return fmt.Sprintf("%s ALL=(root) NOPASSWD: /usr/bin/powermetrics -n 1 -i 100 --samplers thermal\n", username), nil
}
