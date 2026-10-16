package systeminfo

import (
	"slices"
	"testing"
)

func TestLoggedUsersDeduplicatesSessions(t *testing.T) {
	got := parseLoggedUsers("alex console Oct 16 12:00\nalex ttys001 Oct 16 12:02\nsam pts/2 Oct 16 12:05\n")
	if !got.Available || !slices.Equal(got.Names, []string{"alex", "sam"}) {
		t.Fatalf("users = %+v", got)
	}
	if empty := parseLoggedUsers(""); !empty.Available || len(empty.Names) != 0 {
		t.Fatalf("empty sessions = %+v", empty)
	}
}
