package themepack

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"sync"

	"github.com/reathloo/shellux/themes"
)

var registry = struct {
	sync.RWMutex
	entries map[string]Entry
}{}
var initialize sync.Once

func initRegistry() {
	initialize.Do(func() {
		registry.entries = map[string]Entry{}
		register := func(e Entry) {
			registry.entries[e.Name] = e
		}
		c, err := DecodeCatalog(themes.Catalog)
		if err != nil {
			panic(err)
		}
		for _, e := range c.Themes {
			register(e)
		}
		if s, err := DefaultStore(); err == nil {
			if c, err := s.CachedCatalog(); err == nil {
				for _, e := range c.Themes {
					register(e)
				}
			}
			paths, _ := filepath.Glob(filepath.Join(s.Directory, "*.installed.json"))
			for _, path := range paths {
				if b, err := readBounded(path, MaxCatalogBytes); err == nil {
					if c, err := DecodeCatalog(b); err == nil {
						for _, e := range c.Themes {
							register(e)
						}
					}
				}
			}
		}
	})
}

func Entries() []Entry {
	initRegistry()
	registry.RLock()
	defer registry.RUnlock()
	result := make([]Entry, 0, len(registry.entries))
	for _, e := range registry.entries {
		result = append(result, e)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func Lookup(name string) (Entry, bool) {
	initRegistry()
	registry.RLock()
	defer registry.RUnlock()
	e, ok := registry.entries[name]
	return e, ok
}

// Receipts pin the installed version independently of future catalog changes.
func (s *Store) Installed(name string) (Entry, error) {
	if !identifier.MatchString(name) || len(name) > 32 {
		return Entry{}, fmt.Errorf("invalid theme name")
	}
	b, err := readBounded(filepath.Join(s.Directory, name+".installed.json"), MaxCatalogBytes)
	if err != nil {
		return Entry{}, err
	}
	c, err := DecodeCatalog(b)
	if err != nil {
		return Entry{}, err
	}
	if len(c.Themes) != 1 || c.Themes[0].Name != name {
		return Entry{}, fmt.Errorf("invalid installed theme receipt")
	}
	return c.Themes[0], nil
}

func (s *Store) Install(ctx context.Context, e Entry) (*Package, error) {
	p, err := s.Ensure(ctx, e)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(Catalog{Format: Format, Themes: []Entry{e}})
	if err := atomicWrite(filepath.Join(s.Directory, e.Name+".installed.json"), b); err != nil {
		return nil, err
	}
	return p, nil
}

// Import installs a local archive through the same checks as a download.
func (s *Store) Import(e Entry, b []byte) error {
	path, err := s.archivePath(e)
	if err != nil {
		return err
	}
	if _, err := check(e, b); err != nil {
		return err
	}
	if err := atomicWrite(path, b); err != nil {
		return err
	}
	receipt, _ := json.Marshal(Catalog{Format: Format, Themes: []Entry{e}})
	return atomicWrite(filepath.Join(s.Directory, e.Name+".installed.json"), receipt)
}

func (s *Store) LoadName(name string) (*Package, error) {
	e, err := s.Installed(name)
	if err != nil {
		return nil, err
	}
	return s.Load(e)
}

func LoadName(name string) (*Package, error) {
	s, err := DefaultStore()
	if err != nil {
		return nil, err
	}
	return s.LoadName(name)
}

func IsInstalled(name string) bool {
	s, err := DefaultStore()
	if err != nil {
		return false
	}
	e, err := s.Installed(name)
	if err != nil {
		return false
	}
	_, err = s.Load(e)
	return err == nil
}

func EnsureNames(ctx context.Context, names []string) error {
	s, err := DefaultStore()
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		if e, err := s.Installed(name); err == nil {
			if _, err := s.Load(e); err == nil {
				continue
			}
		}
		e, ok := Lookup(name)
		if !ok {
			return fmt.Errorf("unknown theme %q", name)
		}
		if _, err := s.Install(ctx, e); err != nil {
			return fmt.Errorf("download %s: %w", name, err)
		}
	}
	return nil
}

// UseCatalog merges validated metadata on the UI event loop after refresh.
func UseCatalog(c Catalog) {
	b, err := json.Marshal(c)
	if err != nil {
		return
	}
	if _, err := DecodeCatalog(b); err != nil {
		return
	}
	initRegistry()
	registry.Lock()
	defer registry.Unlock()
	for _, e := range c.Themes {
		if store, err := DefaultStore(); err == nil {
			if pinned, err := store.Installed(e.Name); err == nil {
				e = pinned
			}
		}
		registry.entries[e.Name] = e
	}
}
