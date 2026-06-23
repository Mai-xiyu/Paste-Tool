package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if got := cfg.HotkeyString(); got != "Ctrl+Alt+V" {
		t.Fatalf("HotkeyString() = %q", got)
	}
	if cfg.Paste.StartDelayMS != DefaultStartDelayMS {
		t.Fatalf("StartDelayMS = %d", cfg.Paste.StartDelayMS)
	}
}

func TestSetAndSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Default()
	if err := cfg.Set("hotkey", "Ctrl+Shift+F2"); err != nil {
		t.Fatalf("Set hotkey: %v", err)
	}
	if err := cfg.Set("paste.batch_size", "12"); err != nil {
		t.Fatalf("Set batch size: %v", err)
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := loaded.HotkeyString(); got != "Ctrl+Shift+F2" {
		t.Fatalf("loaded hotkey = %q", got)
	}
	if loaded.Paste.BatchSize != 12 {
		t.Fatalf("loaded batch size = %d", loaded.Paste.BatchSize)
	}
}

func TestRejectsInvalidConfigKey(t *testing.T) {
	cfg := Default()
	if err := cfg.Set("paste.start_delay_ms", "-1"); err == nil {
		t.Fatal("expected invalid negative delay")
	}
	if err := cfg.Set("hotkey", "V"); err == nil {
		t.Fatal("expected invalid hotkey without modifier")
	}
	if err := cfg.Set("hotkey", "Ctrl+Alt+Bad"); err == nil {
		t.Fatal("expected invalid hotkey key")
	}
}

func TestAllowsZeroBatchSize(t *testing.T) {
	cfg := Default()
	if err := cfg.Set("paste.batch_size", "0"); err != nil {
		t.Fatalf("Set batch size: %v", err)
	}
	if cfg.Paste.BatchSize != 0 {
		t.Fatalf("batch size = %d", cfg.Paste.BatchSize)
	}
}
