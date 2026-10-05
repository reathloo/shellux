package version

import "testing"

func TestString(t *testing.T) {
	if got, want := String(), "shellux v0.1.2"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
