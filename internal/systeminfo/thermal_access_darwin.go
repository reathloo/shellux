//go:build darwin

package systeminfo

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
)

// EnableThermalAccess permits this local user to query only powermetrics'
// thermal sampler without a password. It never grants a general sudo right.
func EnableThermalAccess() (bool, error) {
	if os.Geteuid() != 0 {
		executable, err := os.Executable()
		if err != nil {
			return false, fmt.Errorf("find Shellux executable: %w", err)
		}
		command := exec.Command("/usr/bin/sudo", executable, "thermal", "enable")
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return false, fmt.Errorf("authorize thermal access: %w", err)
		}
		return true, nil
	}

	username := os.Getenv("SUDO_USER")
	if username == "" {
		return false, fmt.Errorf("run 'shellux thermal enable' as your normal user, not directly as root")
	}
	localUser, err := user.Lookup(username)
	if err != nil {
		return false, fmt.Errorf("look up local user %q: %w", username, err)
	}
	content, err := thermalSudoersContent(localUser.Username)
	if err != nil {
		return false, err
	}

	temporary, err := os.CreateTemp("/etc/sudoers.d", ".shellux-powermetrics-*")
	if err != nil {
		return false, fmt.Errorf("create sudoers file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.WriteString(content); err != nil {
		temporary.Close()
		return false, fmt.Errorf("write sudoers file: %w", err)
	}
	if err := temporary.Chmod(0o440); err != nil {
		temporary.Close()
		return false, fmt.Errorf("set sudoers permissions: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return false, fmt.Errorf("close sudoers file: %w", err)
	}
	if output, err := exec.Command("/usr/sbin/visudo", "-cf", temporaryPath).CombinedOutput(); err != nil {
		return false, fmt.Errorf("validate sudoers file: %w: %s", err, output)
	}
	if err := os.Rename(temporaryPath, thermalSudoersPath); err != nil {
		return false, fmt.Errorf("install sudoers file: %w", err)
	}
	return false, nil
}
