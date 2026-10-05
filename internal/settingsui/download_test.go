package settingsui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/themepack"
)

func downloadModel(t *testing.T) (*model, *int) {
	t.Helper()
	saves := 0
	m := newModel(config.Default(), func(config.Config) error { saves++; return nil })
	if _, err := m.draft.ApplyTheme("slotmaschine"); err != nil {
		t.Fatal(err)
	}
	return m, &saves
}

func TestSaveDownloadsPackageBeforeWritingConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	e, ok := themepack.Lookup("heart")
	if !ok {
		t.Fatal("missing fixture")
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "themes", e.Package))
	if err != nil {
		t.Fatal(err)
	}
	var downloads atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { downloads.Add(1); w.Write(b) }))
	defer server.Close()
	store := themepack.Store{Directory: t.TempDir(), CatalogURL: server.URL + "/catalog.json", Client: server.Client()}
	session, err := config.OpenEditor()
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(session.Config, session.Save)
	m.ensure = func(ctx context.Context, c config.Config) error {
		if names := c.RequiredPackages(); len(names) != 1 || names[0] != "heart" {
			t.Fatalf("dependencies: %v", names)
		}
		_, err := store.Install(ctx, e)
		return err
	}
	if _, err := m.draft.ApplyTheme("heart"); err != nil {
		t.Fatal(err)
	}
	if downloads.Load() != 0 {
		t.Fatal("selection downloaded package")
	}
	cmd := m.saveDraft()
	path, _ := config.DefaultPath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("config written before download")
	}
	m.Update(cmd())
	if downloads.Load() != 1 || m.result != Saved || !m.done {
		t.Fatal("download/save failed")
	}
	if _, err := store.Load(e); err != nil {
		t.Fatal(err)
	}
	c, err := config.LoadDefault()
	if err != nil || c.Animation != "heart" {
		t.Fatalf("saved config: %v %v", c, err)
	}
}

func TestDownloadStartsOnlyOnSave(t *testing.T) {
	m, saves := downloadModel(t)
	downloads := 0
	m.ensure = func(_ context.Context, c config.Config) error {
		downloads++
		if c.Animation != "slotmaschine" {
			t.Fatal("wrong selection")
		}
		return nil
	}
	if downloads != 0 || *saves != 0 {
		t.Fatal("selection caused writes")
	}
	cmd := m.saveDraft()
	if cmd == nil || !m.downloading || *saves != 0 {
		t.Fatal("save was not deferred")
	}
	m.Update(cmd())
	if downloads != 1 || *saves != 1 || !m.done {
		t.Fatal("successful download did not commit exactly once")
	}
}

func TestFailedDownloadPreservesDraft(t *testing.T) {
	m, saves := downloadModel(t)
	m.ensure = func(context.Context, config.Config) error { return errors.New("offline") }
	cmd := m.saveDraft()
	m.Update(cmd())
	if m.done || *saves != 0 || m.downloading || m.draft.Animation != "slotmaschine" {
		t.Fatal("failure lost draft or committed")
	}
	m.ensure = func(context.Context, config.Config) error { return nil }
	cmd = m.saveDraft()
	m.Update(cmd())
	if *saves != 1 || !m.done {
		t.Fatal("retry failed")
	}
}

func TestCancelledDownloadIgnoresLateSuccess(t *testing.T) {
	m, saves := downloadModel(t)
	m.ensure = func(context.Context, config.Config) error { return nil }
	cmd := m.saveDraft()
	press(m, "esc")
	m.Update(cmd())
	if m.done || *saves != 0 || m.downloading {
		t.Fatal("late result committed cancelled download")
	}
}

func TestDownloadStillChecksExternalConfigChange(t *testing.T) {
	m, _ := downloadModel(t)
	m.ensure = func(context.Context, config.Config) error { return nil }
	m.save = func(config.Config) error { return config.ErrChanged }
	cmd := m.saveDraft()
	m.Update(cmd())
	if m.done || m.downloading {
		t.Fatal("conflict closed menu")
	}
}

func TestInterruptAndDuplicateSaveDuringDownload(t *testing.T) {
	m, saves := downloadModel(t)
	m.ensure = func(context.Context, config.Config) error { return nil }
	cmd := m.saveDraft()
	if duplicate := m.saveDraft(); duplicate != nil {
		t.Fatal("duplicate save launched download")
	}
	press(m, "ctrl+c")
	if !m.done || m.result != Interrupted || *saves != 0 {
		t.Fatal("interrupt failed")
	}
	_ = cmd
}
