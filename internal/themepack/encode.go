package themepack

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Encode deduplicates artwork while keeping timing in the manifest. Identical
// inputs produce identical archives and checksums, independent of build date.
func Encode(m Manifest, frames []string, durations []int) ([]byte, Entry, error) {
	if len(frames) == 0 || len(frames) != len(durations) {
		return nil, Entry{}, fmt.Errorf("frames and durations must match")
	}
	m.Frames = nil
	unique := []string{}
	names := map[string]string{}
	for i, frame := range frames {
		name, ok := names[frame]
		if !ok {
			name = fmt.Sprintf("frames/%04d.ans", len(unique))
			unique = append(unique, frame)
			names[frame] = name
		}
		m.Frames = append(m.Frames, Step{File: name, DurationMS: durations[i]})
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	manifest, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, Entry{}, err
	}
	write := func(name string, b []byte) error {
		f, err := w.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err != nil {
			return err
		}
		_, err = f.Write(b)
		return err
	}
	if err := write("theme.json", manifest); err != nil {
		return nil, Entry{}, err
	}
	for i, frame := range unique {
		if err := write(fmt.Sprintf("frames/%04d.ans", i), []byte(frame)); err != nil {
			return nil, Entry{}, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, Entry{}, err
	}
	b := out.Bytes()
	if _, err := Decode(b); err != nil {
		return nil, Entry{}, err
	}
	hash := sha256.Sum256(b)
	e := Entry{Name: m.Name, Version: m.Version, Size: int64(len(b)), SHA256: hex.EncodeToString(hash[:]), Package: "packages/" + m.Name + "-" + m.Version + ".shellux-theme", Background: m.Background, Style: m.Style}
	return b, e, nil
}
