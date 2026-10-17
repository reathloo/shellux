package config

import (
	"encoding/json"
	"testing"
)

func TestProfileChoicesRoundTripAndValidation(t *testing.T) {
	c := Default()
	if err := c.SelectProfile("coding"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetProfileSelection("auto"); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := decodeConfig(data)
	if err != nil || loaded.SelectedProfile != "coding" || loaded.ProfileSelection != "auto" {
		t.Fatalf("profile round trip = %+v, %v", loaded, err)
	}
	if err := c.SelectProfile("unknown"); err == nil || c.SelectedProfile != "coding" || c.ProfileSelection != "auto" {
		t.Fatal("invalid selection changed the active profile")
	}
	if err := c.SetProfileSelection("unknown"); err == nil || c.ProfileSelection != "auto" {
		t.Fatal("invalid selection mode was accepted")
	}
	if _, err := decodeConfig([]byte(`{"profile_selection":"unknown"}`)); err == nil {
		t.Fatal("unknown saved selection mode was accepted")
	}
	if err := c.SelectProfile("minimal"); err != nil || c.ProfileSelection != "manual" || c.SelectedProfile != "minimal" {
		t.Fatal("manual profile choice did not switch out of automatic selection")
	}
}
