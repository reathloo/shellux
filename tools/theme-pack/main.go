// theme-pack builds reproducible data-only theme packages and their catalog.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/reathloo/shellux/internal/themepack"
)

func main() {
	source := flag.String("source", "", "directory containing theme.json and frames")
	out := flag.String("out", "themes", "catalog and packages directory")
	check := flag.Bool("check", false, "validate all catalog packages without writing")
	install := flag.String("install", "", "install one local package by name for offline use/testing")
	flag.Parse()
	var err error
	if *install != "" {
		err = installLocal(*out, *install)
	} else if *check {
		err = verify(*out)
	} else if *source != "" {
		err = build(*source, *out)
	} else {
		err = fmt.Errorf("use -source <directory> or -check")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func installLocal(out, name string) error {
	b, err := os.ReadFile(filepath.Join(out, "catalog.json"))
	if err != nil {
		return err
	}
	c, err := themepack.DecodeCatalog(b)
	if err != nil {
		return err
	}
	for _, e := range c.Themes {
		if e.Name == name {
			b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(e.Package)))
			if err != nil {
				return err
			}
			s, err := themepack.DefaultStore()
			if err != nil {
				return err
			}
			if err := s.Import(e, b); err != nil {
				return err
			}
			fmt.Printf("Installed %s locally.\n", name)
			return nil
		}
	}
	return fmt.Errorf("unknown theme %q", name)
}

func build(source, out string) error {
	b, err := os.ReadFile(filepath.Join(source, "theme.json"))
	if err != nil {
		return err
	}
	var m themepack.Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	frames := []string{}
	durations := []int{}
	for _, step := range m.Frames {
		if !filepath.IsLocal(step.File) {
			return fmt.Errorf("frame must be inside source directory")
		}
		b, err := os.ReadFile(filepath.Join(source, step.File))
		if err != nil {
			return err
		}
		frames = append(frames, strings.TrimSuffix(string(b), "\n"))
		durations = append(durations, step.DurationMS)
	}
	b, entry, err := themepack.Encode(m, frames, durations)
	if err != nil {
		return err
	}
	return writePackage(out, b, entry)
}

func writePackage(out string, b []byte, e themepack.Entry) error {
	c := themepack.Catalog{Format: themepack.Format}
	catalogPath := filepath.Join(out, "catalog.json")
	if existing, err := os.ReadFile(catalogPath); err == nil {
		c, err = themepack.DecodeCatalog(existing)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	path := filepath.Join(out, filepath.FromSlash(e.Package))
	if existing, err := os.ReadFile(path); err == nil && string(existing) != string(b) {
		return fmt.Errorf("%s already exists with different content; bump the theme version", path)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return err
	}
	replaced := false
	for i := range c.Themes {
		if c.Themes[i].Name == e.Name {
			c.Themes[i] = e
			replaced = true
		}
	}
	if !replaced {
		c.Themes = append(c.Themes, e)
	}
	sort.Slice(c.Themes, func(i, j int) bool { return c.Themes[i].Name < c.Themes[j].Name })
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if _, err := themepack.DecodeCatalog(b); err != nil {
		return err
	}
	if err := os.WriteFile(catalogPath, append(b, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("%s %s: %d bytes\n", e.Name, e.Version, e.Size)
	return nil
}

func verify(out string) error {
	b, err := os.ReadFile(filepath.Join(out, "catalog.json"))
	if err != nil {
		return err
	}
	c, err := themepack.DecodeCatalog(b)
	if err != nil {
		return err
	}
	s := themepack.Store{Directory: filepath.Join(out, "packages")}
	for _, e := range c.Themes {
		if _, err := s.Load(e); err != nil {
			return fmt.Errorf("%s: %w", e.Name, err)
		}
	}
	fmt.Printf("Validated %d theme packages.\n", len(c.Themes))
	return nil
}
