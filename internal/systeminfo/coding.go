package systeminfo

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/reathloo/shellux/internal/config"
)

// Coding contains optional, local development context for the current shell.
type Coding struct {
	Branch, Repository     string
	Ahead, Behind          int
	SyncAvailable          bool
	Go, Node, Python, Java string
	Docker, Kubernetes     string
	CommandSeconds         int
	CommandExit            int
	CommandAvailable       bool
}

var codingCommands = struct {
	sync.Mutex
	values map[string]commandSample
}{values: make(map[string]commandSample)}

var codingCommandTimeout = 2 * time.Second

// codingOutput never starts a shell, asks for credentials, or waits indefinitely.
// Results are cached to keep the live dashboard's one-second refresh cheap.
func codingOutput(directory, name string, args ...string) string {
	key := directory + "\x00" + name + "\x00" + strings.Join(args, "\x00")
	codingCommands.Lock()
	if sample, ok := codingCommands.values[key]; ok && time.Since(sample.at) < 5*time.Second {
		codingCommands.Unlock()
		return sample.output
	}
	codingCommands.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), codingCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = directory
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
	cmd.WaitDelay = 100 * time.Millisecond
	var output cappedOutput
	cmd.Stdout, cmd.Stderr = &output, &output
	err := cmd.Run()
	value := ""
	if err == nil {
		value = strings.TrimSpace(output.String())
	}
	codingCommands.Lock()
	codingCommands.values[key] = commandSample{at: time.Now(), output: value, err: err}
	codingCommands.Unlock()
	return value
}

type cappedOutput struct{ bytes.Buffer }

func (c *cappedOutput) Write(p []byte) (int, error) {
	n := len(p)
	if c.Len() < 4096 {
		_, _ = c.Buffer.Write(p[:min(n, 4096-c.Len())])
	}
	return n, nil
}

// CodingInfo probes only enabled entries and components.
func CodingInfo(visible map[string]bool, directory string) Coding {
	var result Coding
	var probes sync.WaitGroup
	// A small limit avoids flooding slower laptops with toolchain processes.
	probeSlots := make(chan struct{}, 2)
	probe := func(work func()) {
		probes.Add(1)
		go func() {
			defer probes.Done()
			probeSlots <- struct{}{}
			defer func() { <-probeSlots }()
			work()
		}()
	}
	if config.EntryShown(visible, "git-branch") {
		if config.Shown(visible, "git-branch.branch") {
			probe(func() {
				result.Branch = codingOutput(directory, "git", "symbolic-ref", "--quiet", "--short", "HEAD")
				if result.Branch == "" {
					if hash := codingOutput(directory, "git", "rev-parse", "--short", "HEAD"); hash != "" {
						result.Branch = "detached:" + hash
					}
				}
			})
		}
		if config.Shown(visible, "git-branch.repository") {
			probe(func() {
				if root := codingOutput(directory, "git", "rev-parse", "--show-toplevel"); root != "" {
					result.Repository = filepath.Base(root)
				}
			})
		}
	}
	if config.EntryShown(visible, "git-sync") {
		probe(func() {
			if output := codingOutput(directory, "git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); output != "" {
				fields := strings.Fields(output)
				if len(fields) == 2 {
					ahead, aerr := strconv.Atoi(fields[0])
					behind, berr := strconv.Atoi(fields[1])
					if aerr == nil && berr == nil && ahead >= 0 && behind >= 0 {
						result.Ahead, result.Behind, result.SyncAvailable = ahead, behind, true
					}
				}
			}
		})
	}
	if config.EntryShown(visible, "runtimes") {
		if config.Shown(visible, "runtimes.go") {
			probe(func() { result.Go = versionField(codingOutput(directory, "go", "version"), "go") })
		}
		if config.Shown(visible, "runtimes.node") {
			probe(func() { result.Node = versionField(codingOutput(directory, "node", "--version"), "node") })
		}
		if config.Shown(visible, "runtimes.python") {
			probe(func() { result.Python = versionField(codingOutput(directory, "python3", "--version"), "python") })
		}
		if config.Shown(visible, "runtimes.java") {
			probe(func() { result.Java = versionField(codingOutput(directory, "java", "-version"), "java") })
		}
	}
	if config.EntryShown(visible, "dev-context") {
		if config.Shown(visible, "dev-context.docker") {
			probe(func() { result.Docker = codingOutput(directory, "docker", "context", "show") })
		}
		if config.Shown(visible, "dev-context.kubernetes") {
			probe(func() { result.Kubernetes = codingOutput(directory, "kubectl", "config", "current-context") })
		}
	}
	if config.EntryShown(visible, "last-command") {
		seconds, serr := strconv.Atoi(os.Getenv("SHELLUX_LAST_COMMAND_SECONDS"))
		status, cerr := strconv.Atoi(os.Getenv("SHELLUX_LAST_COMMAND_STATUS"))
		if serr == nil && cerr == nil && seconds >= 0 && status >= 0 && status <= 255 {
			result.CommandSeconds, result.CommandExit, result.CommandAvailable = seconds, status, true
		}
	}
	probes.Wait()
	return result
}

func versionField(output, language string) string {
	fields := strings.Fields(strings.SplitN(output, "\n", 2)[0])
	if len(fields) == 0 {
		return ""
	}
	switch language {
	case "go":
		if len(fields) > 2 {
			return strings.TrimPrefix(fields[2], "go")
		}
	case "python":
		if len(fields) > 1 {
			return fields[1]
		}
	case "java":
		for i, field := range fields {
			if field == "version" && i+1 < len(fields) {
				return strings.Trim(fields[i+1], "\"")
			}
		}
		if len(fields) > 1 {
			return fields[1]
		}
	case "node":
		return strings.TrimPrefix(fields[0], "v")
	}
	return ""
}
