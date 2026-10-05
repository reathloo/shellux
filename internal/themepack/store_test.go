package themepack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

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
