package terminal

import "testing"

func TestBackgroundSequences(t *testing.T) {
	if got := Background("#112233"); got != "\033]11;#112233\007" {
		t.Fatalf("set background = %q", got)
	}
	if got := Background("default"); got != "\033]111\007" {
		t.Fatalf("reset background = %q", got)
	}
}
