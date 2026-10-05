package animation

import "time"

// ShelluxDefault plays the embedded shelluxdefault artwork.
type ShelluxDefault struct{}

var shelluxDefaultFrames = newFrames("shelluxdefault_frames/*.ans", centerFrame)

func (ShelluxDefault) Frame(index int) string  { return shelluxDefaultFrames.frame(index) }
func (ShelluxDefault) FrameCount() int         { return len(shelluxDefaultFrames.names) }
func (ShelluxDefault) Interval() time.Duration { return 80 * time.Millisecond }
