package settingsui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/config"
)

func TestImmediateConfirmationExplainsPersistence(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.draft.Language = "de"
	m.modal = &dialog{kind: "confirm", title: "Repair now?", action: "doctor-repair"}
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "direkt nach der Bestätigung") {
		t.Fatalf("immediate confirmation lacks accurate notice: %s", view)
	}
	if strings.Contains(view, "erst beim Speichern dauerhaft") || strings.Contains(view, "Datei bleibt bis zum Speichern") {
		t.Fatalf("immediate confirmation claims it waits for save: %s", view)
	}
}

func TestDraftConfirmationStillExplainsSaveBoundary(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.modal = &dialog{kind: "confirm", title: "Delete?", action: "delete"}
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "only when you save") || !strings.Contains(view, "file stays unchanged until you save") {
		t.Fatalf("draft confirmation lost save boundary: %s", view)
	}
}
