package animation

import (
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

var animationCases = []struct {
	name, directory     string
	animation           Animation
	count, milliseconds int
}{{"shelluxdefault", "shelluxdefault", ShelluxDefault{}, 96, 80}}

func TestAnimations(t *testing.T) {
	for _, tc := range animationCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tc.animation
			if a.FrameCount() != tc.count || a.Interval() != time.Duration(tc.milliseconds)*time.Millisecond {
				t.Fatalf("unexpected count or timing: %d, %s", a.FrameCount(), a.Interval())
			}
			names, err := fs.Glob(frameFiles, tc.directory+"_frames/*.ans")
			if err != nil || len(names) != tc.count {
				t.Fatalf("embedded frames: %v, %v", names, err)
			}
			for i, name := range names {
				raw, err := frameFiles.ReadFile(name)
				if err != nil {
					t.Fatal(err)
				}
				expected := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
				blank := strings.Repeat(" ", 51)
				expected = append([]string{blank, blank, blank, blank, blank}, expected...)
				expected = append(expected, blank, blank, blank, blank, blank, blank)
				frame := a.Frame(i)
				if frame != strings.Join(expected, "\n") {
					t.Fatalf("frame %d changed artwork or placement", i)
				}
				if !strings.Contains(frame, "\033[38;2;") || !strings.Contains(frame, "\033[48;2;") {
					t.Fatalf("frame %d lost truecolor", i)
				}
				lines := strings.Split(frame, "\n")
				if len(lines) != 26 {
					t.Fatalf("frame %d height = %d", i, len(lines))
				}
				for row, line := range lines {
					if width := ansi.StringWidth(line); width != 51 {
						t.Fatalf("frame %d row %d width = %d", i, row, width)
					}
				}
			}
			if Frame(a, -1) != a.Frame(tc.count-1) || Frame(a, tc.count) != a.Frame(0) {
				t.Fatal("cyclic frame wrapping changed")
			}
			if a.Frame(0) == a.Frame(tc.count/2) {
				t.Fatal("animation is static")
			}
			if allocations := testing.AllocsPerRun(100, func() { a.Frame(1) }); allocations != 0 {
				t.Fatalf("cached frame allocates %g times", allocations)
			}
		})
	}
}

// Compare the old per-frame read/format path with the cached path.
var benchmarkFrame string

func BenchmarkWordmarkFrame(b *testing.B) {
	b.Run("ReadAndFormat", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data, err := frameFiles.ReadFile(shelluxDefaultFrames.names[i%96])
			if err != nil {
				b.Fatal(err)
			}
			benchmarkFrame = centerFrame(strings.TrimSuffix(string(data), "\n"))
		}
	})
	b.Run("Cached", func(b *testing.B) {
		a := ShelluxDefault{}
		a.Frame(0)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchmarkFrame = a.Frame(i)
		}
	})
}
