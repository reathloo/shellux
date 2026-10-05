package animation

import "time"

// Animation provides deterministic frames for a terminal animation. Frames may
// include ANSI color sequences; renderers must preserve them.
type Animation interface {
	Frame(index int) string
	FrameCount() int
	Interval() time.Duration
}

// Frame returns the animation frame at index, wrapping cyclically.
func Frame(animation Animation, index int) string {
	count := animation.FrameCount()
	if count == 0 {
		return ""
	}
	index %= count
	if index < 0 {
		index += count
	}
	return animation.Frame(index)
}
