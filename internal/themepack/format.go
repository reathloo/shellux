// Package themepack handles data-only theme archives independently of the UI.
package themepack

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

const (
	Format           = 1
	MaxArchiveBytes  = 8 << 20
	MaxExpandedBytes = 32 << 20
	MaxCatalogBytes  = 1 << 20
)

var identifier = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var version = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var digest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var framePath = regexp.MustCompile(`^frames/[0-9]{4}\.ans$`)

func validName(name string) bool {
	if !identifier.MatchString(name) || len(name) > 32 {
		return false
	}
	switch name {
	case "default", "shelluxdefault", "random", "new", "delete", "luxury", "dark-luxury", "neon", "aurora", "rainbow", "black", "dark", "gray", "white", "red", "green", "blue", "navy", "purple":
		return false
	}
	return true
}

// Only reset, default colors and RGB foreground/background sequences are accepted.
var sgr = regexp.MustCompile(`^\x1b\[(?:0|39|49|(?:38|48);2;[0-9]{1,3};[0-9]{1,3};[0-9]{1,3})m`)

type Palette struct {
	Secondary string `json:"secondary"`
	Accent    string `json:"accent"`
	Muted     string `json:"muted"`
}

type Step struct {
	File       string `json:"file"`
	DurationMS int    `json:"duration_ms"`
}

type Manifest struct {
	Format     int     `json:"format"`
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	Background string  `json:"background"`
	Style      Palette `json:"style"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Frames     []Step  `json:"frames"`
}

type Entry struct {
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	Size       int64   `json:"size"`
	SHA256     string  `json:"sha256"`
	Package    string  `json:"package"`
	Background string  `json:"background"`
	Style      Palette `json:"style"`
}

type Catalog struct {
	Format int     `json:"format"`
	Themes []Entry `json:"themes"`
}

// Package contains validated artwork. Playback does no disk I/O or allocation.
type Package struct {
	Manifest Manifest
	frames   []string
	tick     time.Duration
}

func (p *Package) FrameCount() int         { return len(p.frames) }
func (p *Package) Interval() time.Duration { return p.tick }
func (p *Package) Frame(i int) string {
	i %= len(p.frames)
	if i < 0 {
		i += len(p.frames)
	}
	return p.frames[i]
}

func colors(background string, p Palette) bool {
	return hexColor.MatchString(background) && hexColor.MatchString(p.Secondary) && hexColor.MatchString(p.Accent) && hexColor.MatchString(p.Muted)
}

func DecodeCatalog(data []byte) (Catalog, error) {
	var c Catalog
	if len(data) > MaxCatalogBytes {
		return c, fmt.Errorf("theme catalog is too large")
	}
	if err := decodeJSON(data, &c); err != nil {
		return c, err
	}
	if c.Format != Format || len(c.Themes) > 1000 {
		return c, fmt.Errorf("unsupported theme catalog")
	}
	seen := map[string]bool{}
	for _, e := range c.Themes {
		if !validName(e.Name) || len(e.Version) > 32 || !version.MatchString(e.Version) || seen[e.Name] || e.Size <= 0 || e.Size > MaxArchiveBytes || !digest.MatchString(e.SHA256) || e.Package != "packages/"+e.Name+"-"+e.Version+".shellux-theme" || !colors(e.Background, e.Style) {
			return c, fmt.Errorf("invalid catalog entry %q", e.Name)
		}
		seen[e.Name] = true
	}
	return c, nil
}

// Decode accepts only a manifest and referenced frames, with bounded expansion.
// Archives are read in place, never extracted onto the filesystem.
func Decode(data []byte) (*Package, error) {
	if len(data) > MaxArchiveBytes {
		return nil, fmt.Errorf("theme archive is too large")
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	if len(r.File) > 1001 {
		return nil, fmt.Errorf("too many theme files")
	}
	files := map[string][]byte{}
	total := int64(0)
	for _, f := range r.File {
		if _, exists := files[f.Name]; exists {
			return nil, fmt.Errorf("duplicate theme file")
		}
		if !f.Mode().IsRegular() || (f.Name != "theme.json" && !framePath.MatchString(f.Name)) {
			return nil, fmt.Errorf("invalid theme file %q", f.Name)
		}
		if f.UncompressedSize64 > MaxExpandedBytes {
			return nil, fmt.Errorf("theme file is too large")
		}
		reader, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, readErr := io.ReadAll(io.LimitReader(reader, MaxExpandedBytes-total+1))
		closeErr := reader.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		total += int64(len(b))
		if total > MaxExpandedBytes {
			return nil, fmt.Errorf("expanded theme is too large")
		}
		files[f.Name] = b
	}
	var m Manifest
	if err := decodeJSON(files["theme.json"], &m); err != nil {
		return nil, err
	}
	if m.Format != Format || !identifier.MatchString(m.Name) || len(m.Name) > 32 || !version.MatchString(m.Version) || !colors(m.Background, m.Style) || m.Width != 51 || m.Height != 26 || len(m.Frames) == 0 || len(m.Frames) > 2000 {
		return nil, fmt.Errorf("invalid theme manifest")
	}
	delete(files, "theme.json")
	used := map[string]bool{}
	art := map[string]string{}
	tick, duration := 0, 0
	for _, step := range m.Frames {
		if step.DurationMS < 10 || step.DurationMS > 5000 {
			return nil, fmt.Errorf("invalid frame duration")
		}
		duration += step.DurationMS
		if duration > 120000 {
			return nil, fmt.Errorf("animation exceeds two minutes")
		}
		if tick == 0 {
			tick = step.DurationMS
		} else {
			tick = gcd(tick, step.DurationMS)
		}
		if used[step.File] {
			continue
		}
		b, ok := files[step.File]
		if !ok {
			return nil, fmt.Errorf("missing frame %q", step.File)
		}
		frame := strings.TrimSuffix(string(b), "\n")
		if err := validateFrame(frame, m.Width, m.Height); err != nil {
			return nil, fmt.Errorf("%s: %w", step.File, err)
		}
		art[step.File], used[step.File] = frame, true
	}
	if len(used) != len(files) {
		return nil, fmt.Errorf("unreferenced files in theme")
	}
	if tick < 10 || duration/tick > 12000 {
		return nil, fmt.Errorf("unsupported frame timing")
	}
	p := &Package{Manifest: m, tick: time.Duration(tick) * time.Millisecond}
	for _, step := range m.Frames {
		for n := 0; n < step.DurationMS/tick; n++ {
			p.frames = append(p.frames, art[step.File])
		}
	}
	return p, nil
}

func decodeJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func validateFrame(frame string, width, height int) error {
	if !utf8.ValidString(frame) {
		return fmt.Errorf("invalid UTF-8")
	}
	for text := frame; len(text) > 0; {
		if text[0] == 27 {
			sequence := sgr.FindString(text)
			if sequence == "" {
				return fmt.Errorf("unsupported terminal escape")
			}
			var mode, r, g, b int
			if strings.Contains(sequence, ";2;") {
				if _, err := fmt.Sscanf(sequence, "\x1b[%d;2;%d;%d;%dm", &mode, &r, &g, &b); err != nil || r > 255 || g > 255 || b > 255 {
					return fmt.Errorf("invalid RGB color")
				}
			}
			text = text[len(sequence):]
			continue
		}
		r, n := utf8.DecodeRuneInString(text)
		if r != '\n' && (unicode.IsControl(r) || unicode.Is(unicode.Cf, r)) {
			return fmt.Errorf("unsupported control character")
		}
		text = text[n:]
	}
	lines := strings.Split(frame, "\n")
	if len(lines) != height {
		return fmt.Errorf("expected %d rows", height)
	}
	for _, line := range lines {
		if ansi.StringWidth(line) != width {
			return fmt.Errorf("expected %d columns", width)
		}
	}
	return nil
}
