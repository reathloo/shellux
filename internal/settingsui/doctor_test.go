package settingsui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/doctor"
)

func TestDoctorScanAndConfirmedRepairAreImmediate(t *testing.T) {
	saves, repairs := 0, 0
	m := newModel(config.Default(), func(config.Config) error { saves++; return nil })
	m.section, m.focus = doctorSection, 1
	m.doctorScan = func(context.Context) []doctor.Check {
		state, repairable := "warning", true
		if repairs > 0 {
			state, repairable = "ok", false
		}
		return []doctor.Check{{Key: doctor.Integration, State: state, Detail: "Missing or outdated: shellux.zsh", Repairable: repairable}}
	}
	m.doctorFix = func(_ context.Context, key string) error {
		if key != doctor.Integration {
			t.Fatalf("unexpected repair %s", key)
		}
		repairs++
		return nil
	}
	cmd := m.runDoctorScan()
	if !m.doctorBusy || repairs != 0 {
		t.Fatal("scan did not start read-only")
	}
	m.Update(cmd())
	if m.doctorBusy || m.rows()[1].action != "doctor-repair" {
		t.Fatalf("scan result: %+v", m.rows())
	}
	m.cursors[doctorSection] = 1
	m.activate()
	if m.modal == nil || repairs != 0 {
		t.Fatal("repair skipped confirmation")
	}
	m.modal.selected = 1
	cmd = m.updateDialog(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil || repairs != 0 {
		t.Fatal("repair was not scheduled")
	}
	m.Update(cmd())
	if repairs != 1 || saves != 0 || m.dirty() || m.rows()[1].value != "OK" {
		t.Fatalf("repair result: repairs=%d saves=%d rows=%+v", repairs, saves, m.rows())
	}
	m.draft.Language = "de"
	if got := m.doctorDescription(doctor.Check{State: "warning", Detail: "Missing or outdated: shellux.zsh"}); !strings.Contains(got, "Fehlend oder veraltet") {
		t.Fatalf("German doctor detail: %q", got)
	}
}
