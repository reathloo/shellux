package systeminfo

import "testing"

func TestThermalSudoersContent(t *testing.T) {
	content, err := thermalSudoersContent("reathloo")
	if err != nil {
		t.Fatalf("thermalSudoersContent() error = %v", err)
	}
	want := "reathloo ALL=(root) NOPASSWD: /usr/bin/powermetrics -n 1 -i 100 --samplers thermal\n"
	if content != want {
		t.Fatalf("thermalSudoersContent() = %q, want %q", content, want)
	}
}

func TestThermalSudoersContentRejectsInvalidUsername(t *testing.T) {
	if _, err := thermalSudoersContent("name with spaces"); err == nil {
		t.Fatal("thermalSudoersContent() error = nil, want invalid username error")
	}
}
