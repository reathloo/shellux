// Package appupdate checks and installs verified Shellux release archives.
package appupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultReleaseBaseURL = "https://github.com/reathloo/shellux/releases/latest/download"
	maxChecksumsBytes     = 1 << 20
	maxArchiveBytes       = 64 << 20
	checkInterval         = 24 * time.Hour
)

var ErrNotDue = errors.New("automatic update check is not due")

type Release struct {
	Version     string
	ArchiveName string
	SHA256      string
}

type Result struct {
	Current string
	Latest  string
	Updated bool
}

type Client struct {
	BaseURL       string
	HTTPClient    *http.Client
	Executable    string
	DataDirectory string
	StatePath     string
	Now           func() time.Time
}

func DefaultClient() (*Client, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("find executable: %w", err)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
		executable = resolved
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("find home directory: %w", err)
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
		if runtime.GOOS == "darwin" {
			dataHome = filepath.Join(home, "Library", "Application Support")
		}
	}
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}
	baseURL := os.Getenv("SHELLUX_RELEASE_BASE_URL")
	if baseURL == "" {
		baseURL = DefaultReleaseBaseURL
	}
	return &Client{
		BaseURL:       baseURL,
		Executable:    executable,
		DataDirectory: filepath.Join(dataHome, "shellux"),
		StatePath:     filepath.Join(stateHome, "shellux", "update.json"),
	}, nil
}

func (c *Client) Run(ctx context.Context, current string, install bool) (Result, error) {
	release, err := c.Check(ctx)
	if err != nil {
		return Result{}, err
	}
	result := Result{Current: current, Latest: release.Version}
	comparison, err := CompareVersions(release.Version, current)
	if err != nil {
		return Result{}, err
	}
	if comparison <= 0 || !install {
		return result, nil
	}
	if err := c.Install(ctx, release); err != nil {
		return Result{}, err
	}
	result.Updated = true
	return result, nil
}

// Auto runs at most once per day. A successful release check is remembered
// even if installation fails, avoiding a network request in every new shell.
func (c *Client) Auto(ctx context.Context, current string) (Result, error) {
	releaseLock, err := c.acquireLock()
	if err != nil {
		return Result{}, err
	}
	defer releaseLock()
	if !c.due() {
		return Result{}, ErrNotDue
	}
	release, err := c.Check(ctx)
	if err != nil {
		return Result{}, err
	}
	if err := c.markChecked(); err != nil {
		return Result{}, err
	}
	result := Result{Current: current, Latest: release.Version}
	comparison, err := CompareVersions(release.Version, current)
	if err != nil {
		return Result{}, err
	}
	if comparison <= 0 {
		return result, nil
	}
	if err := c.Install(ctx, release); err != nil {
		return Result{}, err
	}
	result.Updated = true
	return result, nil
}

func (c *Client) Check(ctx context.Context) (Release, error) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return Release{}, fmt.Errorf("updates are not supported on %s", runtime.GOOS)
	}
	architecture := runtime.GOARCH
	if architecture != "amd64" && architecture != "arm64" {
		return Release{}, fmt.Errorf("updates are not supported on %s", architecture)
	}
	b, err := c.fetch(ctx, strings.TrimRight(c.BaseURL, "/")+"/checksums.txt", maxChecksumsBytes)
	if err != nil {
		return Release{}, fmt.Errorf("download checksums: %w", err)
	}
	return parseChecksums(b, runtime.GOOS, architecture)
}

func parseChecksums(b []byte, operatingSystem, architecture string) (Release, error) {
	suffix := "_" + operatingSystem + "_" + architecture + ".tar.gz"
	var match Release
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || !strings.HasSuffix(fields[1], suffix) {
			continue
		}
		name := fields[1]
		if path.Base(name) != name || !strings.HasPrefix(name, "shellux_") || len(fields[0]) != 64 {
			return Release{}, fmt.Errorf("invalid release checksum entry")
		}
		if _, err := hex.DecodeString(fields[0]); err != nil {
			return Release{}, fmt.Errorf("invalid release checksum entry")
		}
		version := strings.TrimSuffix(strings.TrimPrefix(name, "shellux_"), suffix)
		if _, err := parseVersion(version); err != nil {
			return Release{}, fmt.Errorf("invalid release version in %q", name)
		}
		if match.ArchiveName != "" {
			return Release{}, fmt.Errorf("multiple release archives match %s/%s", operatingSystem, architecture)
		}
		match = Release{Version: version, ArchiveName: name, SHA256: strings.ToLower(fields[0])}
	}
	if match.ArchiveName == "" {
		return Release{}, fmt.Errorf("no release archive matches %s/%s", operatingSystem, architecture)
	}
	return match, nil
}

func CompareVersions(candidate, current string) (int, error) {
	a, err := parseVersion(candidate)
	if err != nil {
		return 0, fmt.Errorf("candidate version: %w", err)
	}
	b, err := parseVersion(current)
	if err != nil {
		return 0, fmt.Errorf("current version: %w", err)
	}
	for i := range a {
		if a[i] > b[i] {
			return 1, nil
		}
		if a[i] < b[i] {
			return -1, nil
		}
	}
	return 0, nil
}

func parseVersion(value string) ([3]uint64, error) {
	var parsed [3]uint64
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return parsed, fmt.Errorf("%q is not a stable semantic version", value)
	}
	for i, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return parsed, fmt.Errorf("%q is not a stable semantic version", value)
		}
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return parsed, fmt.Errorf("%q is not a stable semantic version", value)
		}
		parsed[i] = n
	}
	return parsed, nil
}

func (c *Client) Install(ctx context.Context, release Release) error {
	b, err := c.fetch(ctx, strings.TrimRight(c.BaseURL, "/")+"/"+release.ArchiveName, maxArchiveBytes)
	if err != nil {
		return fmt.Errorf("download release: %w", err)
	}
	digest := sha256.Sum256(b)
	if hex.EncodeToString(digest[:]) != release.SHA256 {
		return fmt.Errorf("release checksum verification failed")
	}
	files, err := releaseFiles(b)
	if err != nil {
		return err
	}
	// Replace the binary first. It remains compatible with the previous shell
	// integration if a later script write fails; the reverse order could leave
	// a new integration invoking commands an old binary does not understand.
	if err := atomicWrite(c.Executable, files["shellux"], 0755); err != nil {
		return fmt.Errorf("install executable: %w", err)
	}
	if err := atomicWrite(filepath.Join(c.DataDirectory, "shell", "shellux.zsh"), files["shell/shellux.zsh"], 0644); err != nil {
		return fmt.Errorf("install Zsh integration: %w", err)
	}
	if err := atomicWrite(filepath.Join(c.DataDirectory, "shell", "shellux.bash"), files["shell/shellux.bash"], 0644); err != nil {
		return fmt.Errorf("install Bash integration: %w", err)
	}
	return nil
}

func releaseFiles(archive []byte) (map[string][]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open release archive: %w", err)
	}
	defer gz.Close()
	wanted := map[string]bool{"shellux": true, "shell/shellux.zsh": true, "shell/shellux.bash": true}
	result := make(map[string][]byte, len(wanted))
	reader := tar.NewReader(gz)
	for count := 0; ; count++ {
		if count > 1000 {
			return nil, fmt.Errorf("release archive contains too many files")
		}
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read release archive: %w", err)
		}
		clean := path.Clean(header.Name)
		if clean != header.Name || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
			return nil, fmt.Errorf("release archive contains unsafe path %q", header.Name)
		}
		if !wanted[clean] {
			continue
		}
		if header.Typeflag != tar.TypeReg || header.Size < 1 || header.Size > maxArchiveBytes {
			return nil, fmt.Errorf("invalid release file %q", clean)
		}
		if _, exists := result[clean]; exists {
			return nil, fmt.Errorf("duplicate release file %q", clean)
		}
		data, err := io.ReadAll(io.LimitReader(reader, maxArchiveBytes+1))
		if err != nil || int64(len(data)) != header.Size {
			return nil, fmt.Errorf("read release file %q", clean)
		}
		result[clean] = data
	}
	for name := range wanted {
		if len(result[name]) == 0 {
			return nil, fmt.Errorf("release archive is missing %q", name)
		}
	}
	return result, nil
}

func (c *Client) fetch(ctx context.Context, address string, limit int64) ([]byte, error) {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return nil, fmt.Errorf("release download requires HTTPS")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	client := http.Client{}
	if c.HTTPClient != nil {
		client = *c.HTTPClient
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 || req.URL.Scheme != "https" || req.URL.User != nil {
			return fmt.Errorf("unsupported release redirect")
		}
		return nil
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release server returned HTTP %d", response.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("release download exceeds size limit")
	}
	return b, nil
}

func atomicWrite(target string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".shellux-update-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	_, writeErr := temporary.Write(data)
	if writeErr == nil {
		writeErr = temporary.Sync()
	}
	closeErr := temporary.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), target)
}

type state struct {
	CheckedAt time.Time `json:"checked_at"`
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) due() bool {
	b, err := os.ReadFile(c.StatePath)
	if err != nil {
		return true
	}
	var saved state
	if json.Unmarshal(b, &saved) != nil || saved.CheckedAt.IsZero() {
		return true
	}
	return c.now().Sub(saved.CheckedAt) >= checkInterval
}

func (c *Client) markChecked() error {
	b, err := json.Marshal(state{CheckedAt: c.now()})
	if err != nil {
		return err
	}
	return atomicWrite(c.StatePath, append(b, '\n'), 0600)
}

func (c *Client) acquireLock() (func(), error) {
	lockPath := c.StatePath + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		if info, statErr := os.Stat(lockPath); statErr == nil && c.now().Sub(info.ModTime()) > 10*time.Minute {
			_ = os.Remove(lockPath)
			file, err = os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		}
	}
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrNotDue
		}
		return nil, err
	}
	_ = file.Close()
	return func() { _ = os.Remove(lockPath) }, nil
}
