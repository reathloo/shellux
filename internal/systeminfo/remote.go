package systeminfo

import (
	"net"
	"net/netip"
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/reathloo/shellux/internal/config"
)

// RemoteSession contains only locally available SSH and terminal context.
type RemoteSession struct {
	User, Client, Server, TTY, Term string
}

// RemoteSessionInfo reads enabled details without contacting an SSH server.
func RemoteSessionInfo(visible map[string]bool) RemoteSession {
	var result RemoteSession
	if config.EntryShown(visible, "remote-identity") && config.Shown(visible, "remote-identity.user") {
		if current, err := user.Current(); err == nil {
			result.User = current.Username
		}
	}
	if config.EntryShown(visible, "ssh-link") {
		result.Client, result.Server = sshEndpoints(os.Getenv("SSH_CONNECTION"), os.Getenv("SSH_CLIENT"))
		if !config.Shown(visible, "ssh-link.client") {
			result.Client = ""
		}
		if !config.Shown(visible, "ssh-link.server") {
			result.Server = ""
		}
	}
	if config.EntryShown(visible, "remote-terminal") {
		if config.Shown(visible, "remote-terminal.tty") {
			result.TTY = os.Getenv("SSH_TTY")
		}
		if config.Shown(visible, "remote-terminal.term") {
			result.Term = os.Getenv("TERM")
		}
	}
	return result
}

func sshEndpoints(connection, client string) (string, string) {
	if fields := strings.Fields(connection); len(fields) == 4 {
		if source := sshEndpoint(fields[0], fields[1]); source != "" {
			if target := sshEndpoint(fields[2], fields[3]); target != "" {
				return source, target
			}
		}
	}
	if fields := strings.Fields(client); len(fields) == 3 {
		if source := sshEndpoint(fields[0], fields[1]); source != "" && validSSHPort(fields[2]) {
			return source, "port " + fields[2]
		}
	}
	return "", ""
}

func sshEndpoint(address, port string) string {
	parsed, err := netip.ParseAddr(address)
	if err != nil || !validSSHPort(port) {
		return ""
	}
	return net.JoinHostPort(parsed.String(), port)
}

func validSSHPort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port > 0 && port <= 65535
}
