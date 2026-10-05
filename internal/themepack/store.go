package themepack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const CatalogURL = "https://raw.githubusercontent.com/reathloo/shellux/main/themes/catalog.json"

// Store does no networking on creation or load. Refresh and Ensure are explicit
// operations for the settings UI/CLI; the header only calls Load.
type Store struct {
	Directory  string
	CatalogURL string
	Client     *http.Client
}

func DefaultStore() (*Store, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		base = filepath.Join(home, ".local", "share")
		if runtime.GOOS == "darwin" {
			base = filepath.Join(home, "Library", "Application Support")
		}
	}
	catalogURL := os.Getenv("SHELLUX_THEME_CATALOG_URL")
	if catalogURL == "" {
		catalogURL = CatalogURL
	}
	return &Store{Directory: filepath.Join(base, "shellux", "themes"), CatalogURL: catalogURL}, nil
}

func (s *Store) fetch(ctx context.Context, address string, limit int64) ([]byte, error) {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return nil, fmt.Errorf("theme download requires HTTPS")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	client := http.Client{}
	if s.Client != nil {
		client = *s.Client
	}
	// A redirect must not downgrade HTTPS or send a request to another host.
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 || req.URL.Scheme != "https" || req.URL.Host != u.Host || req.URL.User != nil {
			return fmt.Errorf("unsupported theme redirect")
		}
		return nil
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("theme server returned HTTP %d", res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("theme download exceeds size limit")
	}
	return b, nil
}

func (s *Store) CachedCatalog() (Catalog, error) {
	b, err := readBounded(filepath.Join(s.Directory, "catalog.json"), MaxCatalogBytes)
	if err != nil {
		return Catalog{}, err
	}
	return DecodeCatalog(b)
}

func (s *Store) Refresh(ctx context.Context) (Catalog, error) {
	b, err := s.fetch(ctx, s.CatalogURL, MaxCatalogBytes)
	if err != nil {
		return Catalog{}, err
	}
	c, err := DecodeCatalog(b)
	if err != nil {
		return Catalog{}, err
	}
	if err := atomicWrite(filepath.Join(s.Directory, "catalog.json"), b); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

func (s *Store) archivePath(e Entry) (string, error) {
	// Validate caller-supplied entries too; names must never become arbitrary paths.
	b, _ := json.Marshal(Catalog{Format: Format, Themes: []Entry{e}})
	if _, err := DecodeCatalog(b); err != nil {
		return "", err
	}
	return filepath.Join(s.Directory, e.Name+"-"+e.Version+".shellux-theme"), nil
}

func check(e Entry, b []byte) (*Package, error) {
	hash := sha256.Sum256(b)
	if int64(len(b)) != e.Size || hex.EncodeToString(hash[:]) != e.SHA256 {
		return nil, fmt.Errorf("theme checksum or size mismatch")
	}
	p, err := Decode(b)
	if err != nil {
		return nil, err
	}
	m := p.Manifest
	if m.Name != e.Name || m.Version != e.Version || m.Style != e.Style || m.Background != e.Background {
		return nil, fmt.Errorf("theme does not match catalog")
	}
	return p, nil
}

func (s *Store) Load(e Entry) (*Package, error) {
	path, err := s.archivePath(e)
	if err != nil {
		return nil, err
	}
	b, err := readBounded(path, MaxArchiveBytes)
	if err != nil {
		return nil, err
	}
	return check(e, b)
}

// Ensure downloads only missing/corrupt packages. It returns after a completely
// validated archive has been installed. Callers may then commit configuration.
func (s *Store) Ensure(ctx context.Context, e Entry) (*Package, error) {
	path, err := s.archivePath(e)
	if err != nil {
		return nil, err
	}
	if p, err := s.Load(e); err == nil {
		return p, nil
	}
	base, err := url.Parse(s.CatalogURL)
	if err != nil {
		return nil, err
	}
	rel, _ := url.Parse(e.Package)
	b, err := s.fetch(ctx, base.ResolveReference(rel).String(), e.Size)
	if err != nil {
		return nil, err
	}
	p, err := check(e, b)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := atomicWrite(path, b); err != nil {
		return nil, err
	}
	return p, nil
}

func readBounded(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("theme file exceeds size limit")
	}
	return b, nil
}

func atomicWrite(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".theme-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}
