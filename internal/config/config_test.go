package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	d := Default()
	d.Validate()
	if d.FPS != 30 {
		t.Fatalf("FPS=%d want 30", d.FPS)
	}
	if d.Opacity != 1.0 {
		t.Fatalf("opacity=%v want 1.0", d.Opacity)
	}
	if d.Weights["idle"] == 0 {
		t.Fatal("default weights missing idle")
	}
}

func TestValidateClamps(t *testing.T) {
	c := Default()
	c.FPS = 12345
	c.Opacity = 5
	c.Scale = 99
	c.Validate()
	if c.FPS != 30 {
		t.Fatalf("FPS not clamped: %d", c.FPS)
	}
	if c.Opacity != 1 {
		t.Fatalf("opacity not clamped: %v", c.Opacity)
	}
	if c.Scale != 1.0 {
		t.Fatalf("scale not clamped: %v", c.Scale)
	}
}

func TestCorruptConfigFallsBack(t *testing.T) {
	origDir := dirOverride
	defer func() { dirOverride = origDir }()

	tmp := t.TempDir()
	dirOverride = tmp

	os.MkdirAll(filepath.Dir(tmp), 0o755)
	// corrupt file
	if err := os.WriteFile(filepath.Join(tmp, "config.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FPS != 30 {
		t.Fatalf("expected defaults after corrupt file, got FPS=%d", cfg.FPS)
	}
	// backup must exist
	if _, err := os.Stat(filepath.Join(tmp, "config.json.bak.json")); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	origDir := dirOverride
	defer func() { dirOverride = origDir }()

	tmp := t.TempDir()
	dirOverride = tmp

	c := Default()
	c.FPS = 60
	c.Scale = 1.5
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(tmp, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var loaded Config
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("saved config not valid json: %v", err)
	}
	if loaded.FPS != 60 || loaded.Scale != 1.5 {
		t.Fatalf("round trip mismatch: %+v", loaded)
	}
}
