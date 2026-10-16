package themepack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestInstalledThemesUpdateAndUninstall(t *testing.T) {
	oldBytes, old := fixture(t)
	manifest := Manifest{Format: 1, Name: old.Name, Version: "1.1.0", Width: 51, Height: 26, Background: old.Background, Style: old.Style}
	frame := strings.TrimSuffix(strings.Repeat(strings.Repeat(" ", 51)+"\n", 26), "\n")
	newBytes, latest, err := Encode(manifest, []string{frame}, []int{80})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/themes/"+latest.Package {
			w.Write(newBytes)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	s := Store{Directory: t.TempDir(), CatalogURL: server.URL + "/themes/catalog.json", Client: server.Client()}
	if err := s.Import(old, oldBytes); err != nil {
		t.Fatal(err)
	}
	count, err := s.UpdateInstalled(context.Background(), Catalog{Format: Format, Themes: []Entry{latest}})
	if err != nil || count != 1 {
		t.Fatalf("update = %d, %v", count, err)
	}
	installed, err := s.Installed(old.Name)
	if err != nil || installed.Version != latest.Version {
		t.Fatalf("installed = %+v, %v", installed, err)
	}
	if count, err := s.UpdateInstalled(context.Background(), Catalog{Format: Format, Themes: []Entry{old}}); err != nil || count != 0 {
		t.Fatalf("older catalog must not downgrade: %d, %v", count, err)
	}
	if err := s.Uninstall(old.Name); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Installed(old.Name); !os.IsNotExist(err) {
		t.Fatalf("receipt remains: %v", err)
	}
	for _, entry := range []Entry{old, latest} {
		path, _ := s.archivePath(entry)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("archive remains: %s", path)
		}
	}
	if count, err := s.UpdateInstalled(context.Background(), Catalog{Format: Format, Themes: []Entry{latest}}); err != nil || count != 0 {
		t.Fatalf("uninstalled theme was reinstalled: %d, %v", count, err)
	}
}

func TestFailedAutomaticUpdateKeepsInstalledVersion(t *testing.T) {
	b, old := fixture(t)
	s := Store{Directory: t.TempDir(), CatalogURL: "https://example.invalid/themes/catalog.json"}
	if err := s.Import(old, b); err != nil {
		t.Fatal(err)
	}
	latest := old
	latest.Version = "2.0.0"
	latest.Package = "packages/example-2.0.0.shellux-theme"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.UpdateInstalled(ctx, Catalog{Format: Format, Themes: []Entry{latest}}); err == nil {
		t.Fatal("cancelled update succeeded")
	}
	installed, err := s.Installed(old.Name)
	if err != nil || installed.Version != old.Version {
		t.Fatalf("installed version changed after failure: %+v, %v", installed, err)
	}
}

func TestDownloadInstallAndOfflineReuse(t *testing.T) {
	b, e := fixture(t)
	catalog, _ := json.Marshal(Catalog{Format: 1, Themes: []Entry{e}})
	var downloads atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/themes/catalog.json":
			w.Write(catalog)
		case "/themes/" + e.Package:
			downloads.Add(1)
			w.Write(b)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	s := Store{Directory: t.TempDir(), CatalogURL: server.URL + "/themes/catalog.json", Client: server.Client()}
	if _, err := s.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if downloads.Load() != 0 {
		t.Fatal("catalog fetched artwork")
	}
	if _, err := s.Ensure(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	server.Close()
	if _, err := s.Ensure(context.Background(), e); err != nil {
		t.Fatal("offline reuse:", err)
	}
	if _, err := s.CachedCatalog(); err != nil {
		t.Fatal(err)
	}
	if downloads.Load() != 1 {
		t.Fatal("theme downloaded more than once")
	}
}

func TestFailedInstallLeavesExistingData(t *testing.T) {
	b, e := fixture(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("broken")) }))
	defer server.Close()
	s := Store{Directory: t.TempDir(), CatalogURL: server.URL + "/catalog.json", Client: server.Client()}
	path, _ := s.archivePath(e)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	bad := e
	bad.Version = "2.0.0"
	bad.Package = "packages/example-2.0.0.shellux-theme"
	if _, err := s.Ensure(context.Background(), bad); err == nil {
		t.Fatal("bad download accepted")
	}
	if _, err := s.Load(e); err != nil {
		t.Fatal("existing install damaged:", err)
	}
	badPath, _ := s.archivePath(bad)
	if _, err := os.Stat(badPath); !os.IsNotExist(err) {
		t.Fatal("failed download installed")
	}
	temps, _ := filepath.Glob(filepath.Join(s.Directory, ".theme-*"))
	if len(temps) != 0 {
		t.Fatal("temporary files leaked")
	}
}

func TestCancellationAndWriteFailure(t *testing.T) {
	b, e := fixture(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(b) }))
	defer server.Close()
	s := Store{Directory: t.TempDir(), CatalogURL: server.URL + "/catalog.json", Client: server.Client()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Ensure(ctx, e); err == nil {
		t.Fatal("cancelled download installed")
	}
	s.Directory = filepath.Join(s.Directory, "file")
	if err := os.WriteFile(s.Directory, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Ensure(context.Background(), e); err == nil {
		t.Fatal("write error ignored")
	}
}

func TestInvalidRefreshPreservesCachedCatalog(t *testing.T) {
	_, e := fixture(t)
	catalog, _ := json.Marshal(Catalog{Format: 1, Themes: []Entry{e}})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("invalid")) }))
	defer server.Close()
	s := Store{Directory: t.TempDir(), CatalogURL: server.URL + "/catalog.json", Client: server.Client()}
	if err := os.WriteFile(filepath.Join(s.Directory, "catalog.json"), catalog, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Refresh(context.Background()); err == nil {
		t.Fatal("invalid catalog accepted")
	}
	if _, err := s.CachedCatalog(); err != nil {
		t.Fatal("cache damaged:", err)
	}
}
