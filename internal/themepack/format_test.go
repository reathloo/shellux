package themepack

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) ([]byte, Entry) {
	t.Helper()
	m := Manifest{Format: 1, Name: "example", Version: "1.0.0", Width: 51, Height: 26, Background: "#000000", Style: Palette{"#112233", "#aabbcc", "#445566"}}
	frame := strings.TrimSuffix(strings.Repeat(strings.Repeat(" ", 51)+"\n", 26), "\n")
	b, e, err := Encode(m, []string{frame, "\x1b[38;2;1;2;3m" + frame + "\x1b[0m", frame}, []int{40, 120, 40})
	if err != nil {
		t.Fatal(err)
	}
	return b, e
}

func TestRoundTrip(t *testing.T) {
	b, e := fixture(t)
	p, err := check(e, b)
	if err != nil {
		t.Fatal(err)
	}
	if p.Interval() != 40*time.Millisecond || p.FrameCount() != 5 || p.Frame(-1) != p.Frame(0) || p.Frame(1) != p.Frame(3) {
		t.Fatal("timing or wrapping changed")
	}
	other, _ := fixture(t)
	if !bytes.Equal(b, other) {
		t.Fatal("archive is not reproducible")
	}
	if n := testing.AllocsPerRun(100, func() { p.Frame(2) }); n != 0 {
		t.Fatalf("playback allocates: %v", n)
	}
}

func TestRejectUnsafeFrames(t *testing.T) {
	for _, s := range []string{"\x1b[2J", "\x1b]52;c;secret\x07", "\x1b]0;title\x07", "\r", "\t", "\x1b[38;2;999;0;0m", "\u009b", "\u202e", string([]byte{0xff})} {
		if err := validateFrame(s, 51, 26); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
}

func TestRejectArchivePathsAndDuplicates(t *testing.T) {
	for _, names := range [][]string{{"../outside"}, {"/absolute"}, {"frames/0000.ans", "frames/0000.ans"}, {"extra.txt"}} {
		var b bytes.Buffer
		w := zip.NewWriter(&b)
		for _, name := range names {
			f, _ := w.Create(name)
			f.Write([]byte("x"))
		}
		w.Close()
		if _, err := Decode(b.Bytes()); err == nil {
			t.Fatalf("accepted %v", names)
		}
	}
}

func TestCatalogValidation(t *testing.T) {
	_, e := fixture(t)
	for _, mutate := range []func(*Entry){
		func(e *Entry) { e.Name = "../escape" }, func(e *Entry) { e.Package = "https://elsewhere/theme" }, func(e *Entry) { e.Size = MaxArchiveBytes + 1 }, func(e *Entry) { e.SHA256 = "incorrect" }, func(e *Entry) { e.Style.Accent = "\x1b[2J" },
	} {
		x := e
		mutate(&x)
		b, _ := json.Marshal(Catalog{Format: 1, Themes: []Entry{x}})
		if _, err := DecodeCatalog(b); err == nil {
			t.Fatal("invalid entry accepted")
		}
	}
	b, _ := json.Marshal(Catalog{Format: 1, Themes: []Entry{e, e}})
	if _, err := DecodeCatalog(b); err == nil {
		t.Fatal("duplicate entry accepted")
	}
}

func TestRejectUnknownAndTrailingJSON(t *testing.T) {
	_, e := fixture(t)
	catalog, _ := json.Marshal(Catalog{Format: 1, Themes: []Entry{e}})
	for _, data := range [][]byte{
		append(append([]byte(nil), catalog...), []byte(` {}`)...),
		[]byte(`{"format":1,"themes":[],"unexpected":true}`),
	} {
		if _, err := DecodeCatalog(data); err == nil {
			t.Fatalf("accepted catalog %s", data)
		}
	}

	b, _ := fixture(t)
	r, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, file := range r.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if file.Name == "theme.json" {
			content = bytes.Replace(content, []byte(`"format": 1`), []byte(`"format": 1, "unexpected": true`), 1)
		}
		dst, err := w.Create(file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dst.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(out.Bytes()); err == nil {
		t.Fatal("accepted manifest with unknown field")
	}
}
