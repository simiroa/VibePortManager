package config

import "testing"

func TestRoundTrip(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	want := &Config{
		Version:  configVersion,
		Settings: Settings{AutostartVPM: true, CloseWarningSeen: true},
		Projects: []Project{{
			ID:             "p1",
			Name:           "Demo",
			Path:           `C:\demo`,
			PackageManager: "pnpm",
			Servers:        []Server{{ID: "s1", Name: "dev", Command: "pnpm dev", Port: 3000, Autostart: true}},
		}},
	}

	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got.Projects) != 1 || len(got.Projects[0].Servers) != 1 {
		t.Fatalf("shape mismatch: %+v", got)
	}
	srv := got.Projects[0].Servers[0]
	if srv.Port != 3000 || srv.Name != "dev" || !srv.Autostart {
		t.Errorf("server round-trip wrong: %+v", srv)
	}
	if !got.Settings.AutostartVPM || !got.Settings.CloseWarningSeen {
		t.Errorf("settings round-trip wrong: %+v", got.Settings)
	}
}

func TestLoadDefaultWhenMissing(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir()) // empty dir, no config.json

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Version != configVersion {
		t.Errorf("default version = %d, want %d", got.Version, configVersion)
	}
	if got.Projects == nil {
		t.Errorf("default Projects should be non-nil empty slice")
	}
}

func TestLoadErrorWhenAppdataUnset(t *testing.T) {
	t.Setenv("APPDATA", "")
	if _, err := Load(); err == nil {
		t.Errorf("expected error when APPDATA unset")
	}
}
