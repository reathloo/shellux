package animation

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

//go:embed shelluxdefault_frames/*.ans
var frameFiles embed.FS

// Each animation prepares its frames once, on first use. Other animations stay
// embedded without allocating copies or doing layout work at startup.
type frames struct {
	names []string
	load  func() []string
}

func newFrames(pattern string, transform func(string) string) frames {
	names, err := fs.Glob(frameFiles, pattern)
	if err != nil || len(names) == 0 {
		panic(fmt.Sprintf("invalid embedded animation %q: %v", pattern, err))
	}
	return frames{names: names, load: sync.OnceValue(func() []string {
		result := make([]string, len(names))
		for i, name := range names {
			data, err := frameFiles.ReadFile(name)
			if err != nil {
				panic(fmt.Sprintf("read embedded frame %q: %v", name, err))
			}
			result[i] = strings.TrimSuffix(string(data), "\n")
			if transform != nil {
				result[i] = transform(result[i])
			}
		}
		return result
	})}
}

func (f frames) frame(index int) string {
	index %= len(f.names)
	if index < 0 {
		index += len(f.names)
	}
	return f.load()[index]
}

// centerFrame pads shorter artwork to the usual 26-row animation canvas.
func centerFrame(frame string) string {
	missing := 26 - (strings.Count(frame, "\n") + 1)
	if missing <= 0 {
		return frame
	}
	padding := strings.Repeat(" ", 51) + "\n"
	return strings.Repeat(padding, missing/2) + frame + "\n" + strings.TrimSuffix(strings.Repeat(padding, missing-missing/2), "\n")
}
