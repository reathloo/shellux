package systeminfo

import "testing"

func TestSSHEndpoints(t *testing.T) {
	for _, test := range []struct {
		connection, client, source, target string
	}{
		{"192.0.2.1 54321 192.0.2.10 22", "", "192.0.2.1:54321", "192.0.2.10:22"},
		{"2001:db8::1 54321 2001:db8::2 2222", "", "[2001:db8::1]:54321", "[2001:db8::2]:2222"},
		{"", "192.0.2.1 54321 22", "192.0.2.1:54321", "port 22"},
		{"invalid 54321 192.0.2.10 22", "", "", ""},
		{"192.0.2.1 0 192.0.2.10 22", "", "", ""},
	} {
		source, target := sshEndpoints(test.connection, test.client)
		if source != test.source || target != test.target {
			t.Fatalf("sshEndpoints(%q, %q) = %q, %q", test.connection, test.client, source, target)
		}
	}
}

func TestRemoteSessionInfoHonorsDetailSwitches(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "192.0.2.1 54321 192.0.2.10 22")
	t.Setenv("SSH_TTY", "/dev/pts/3")
	t.Setenv("TERM", "xterm-256color")
	visible := map[string]bool{
		"ssh-link":             true,
		"ssh-link.client":      true,
		"ssh-link.server":      false,
		"remote-terminal":      true,
		"remote-terminal.tty":  false,
		"remote-terminal.term": true,
	}
	info := RemoteSessionInfo(visible)
	if info.Client != "192.0.2.1:54321" || info.Server != "" || info.TTY != "" || info.Term != "xterm-256color" {
		t.Fatalf("disabled remote details were collected: %+v", info)
	}
}
