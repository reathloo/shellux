package appupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		candidate, current string
		want               int
	}{
		{"0.2.1", "0.2.0", 1},
		{"v1.0.0", "0.9.9", 1},
		{"1.2.3", "1.2.3", 0},
		{"1.2.2", "1.2.3", -1},
	}
	for _, test := range tests {
		got, err := CompareVersions(test.candidate, test.current)
		if err != nil || got != test.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, %v", test.candidate, test.current, got, err)
		}
	}
	for _, invalid := range []string{"1.2", "1.2.3-beta", "01.2.3", "latest"} {
		if _, err := CompareVersions(invalid, "1.0.0"); err == nil {
			t.Fatalf("accepted invalid version %q", invalid)
		}
	}
}

func TestRunInstallsVerifiedRelease(t *testing.T) {
	archive := testArchive(t, map[string]string{
		"shellux":            "new binary",
		"shell/shellux.zsh":  "new zsh",
		"shell/shellux.bash": "new bash",
		"README.md":          "ignored",
	})
	digest := sha256.Sum256(archive)
	name := fmt.Sprintf("shellux_0.2.1_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	checksums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(digest[:]), name)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checksums.txt":
			fmt.Fprint(w, checksums)
		case "/" + name:
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	directory := t.TempDir()
	client := Client{
		BaseURL:       server.URL,
		HTTPClient:    server.Client(),
		Executable:    filepath.Join(directory, "bin", "shellux"),
		DataDirectory: filepath.Join(directory, "data", "shellux"),
		StatePath:     filepath.Join(directory, "state", "update.json"),
	}
	result, err := client.Run(context.Background(), "0.2.0", true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Updated || result.Latest != "0.2.1" {
		t.Fatalf("unexpected result: %+v", result)
	}
	assertFile(t, client.Executable, "new binary", 0111)
	assertFile(t, filepath.Join(client.DataDirectory, "shell", "shellux.zsh"), "new zsh", 0)
	assertFile(t, filepath.Join(client.DataDirectory, "shell", "shellux.bash"), "new bash", 0)
}

func TestCheckOnlyAndBadChecksumDoNotInstall(t *testing.T) {
	archive := testArchive(t, map[string]string{
		"shellux": "binary", "shell/shellux.zsh": "zsh", "shell/shellux.bash": "bash",
	})
	name := fmt.Sprintf("shellux_0.3.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/checksums.txt" {
			fmt.Fprintf(w, "%064d  %s\n", 0, name)
			return
		}
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	directory := t.TempDir()
	client := Client{BaseURL: server.URL, HTTPClient: server.Client(), Executable: filepath.Join(directory, "shellux"), DataDirectory: filepath.Join(directory, "data"), StatePath: filepath.Join(directory, "state.json")}
	result, err := client.Run(context.Background(), "0.2.0", false)
	if err != nil || result.Latest != "0.3.0" || result.Updated {
		t.Fatalf("check result = %+v, %v", result, err)
	}
	if _, err := os.Stat(client.Executable); !os.IsNotExist(err) {
		t.Fatal("check-only created an executable")
	}
	if _, err := client.Run(context.Background(), "0.2.0", true); err == nil {
		t.Fatal("bad checksum was accepted")
	}
	if _, err := os.Stat(client.Executable); !os.IsNotExist(err) {
		t.Fatal("bad checksum installed an executable")
	}
}

func TestAutoCheckIsThrottled(t *testing.T) {
	archive := testArchive(t, map[string]string{
		"shellux": "binary", "shell/shellux.zsh": "zsh", "shell/shellux.bash": "bash",
	})
	digest := sha256.Sum256(archive)
	name := fmt.Sprintf("shellux_0.2.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	requests := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(digest[:]), name)
	}))
	defer server.Close()
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	directory := t.TempDir()
	client := Client{BaseURL: server.URL, HTTPClient: server.Client(), Executable: filepath.Join(directory, "shellux"), DataDirectory: filepath.Join(directory, "data"), StatePath: filepath.Join(directory, "state.json"), Now: func() time.Time { return now }}
	if _, err := client.Auto(context.Background(), "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Auto(context.Background(), "0.2.0"); err != ErrNotDue {
		t.Fatalf("second auto check error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	now = now.Add(25 * time.Hour)
	if _, err := client.Auto(context.Background(), "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestUnsafeArchiveIsRejected(t *testing.T) {
	archive := testArchive(t, map[string]string{"../shellux": "bad", "shellux": "binary", "shell/shellux.zsh": "zsh", "shell/shellux.bash": "bash"})
	if _, err := releaseFiles(archive); err == nil {
		t.Fatal("unsafe archive path was accepted")
	}
}

func testArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gz)
	for name, content := range files {
		data := []byte(content)
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func assertFile(t *testing.T, path, content string, executable os.FileMode) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil || string(b) != content {
		t.Fatalf("%s = %q, %v", path, b, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if executable != 0 && info.Mode().Perm()&executable == 0 {
		t.Fatalf("%s is not executable: %v", path, info.Mode())
	}
}
