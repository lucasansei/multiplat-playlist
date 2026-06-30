package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromPathMissingReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if cfg.Player.Backend != "mpv" {
		t.Fatalf("Player.Backend = %q, want mpv", cfg.Player.Backend)
	}
	if cfg.Player.Volume != 100 {
		t.Fatalf("Player.Volume = %d, want 100", cfg.Player.Volume)
	}
}

func TestLoadFromPathMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadFromPath(path)
	if err == nil {
		t.Fatal("LoadFromPath() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "parse config") {
		t.Fatalf("LoadFromPath() error = %q, want parse config context", err)
	}
}

func TestSaveToPathLoadFromPathRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	want := &Config{
		Spotify: SpotifyConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		},
		Player: PlayerConfig{
			Backend: "mpv",
			Volume:  75,
		},
	}

	if err := want.SaveToPath(path); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	got, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if *got != *want {
		t.Fatalf("LoadFromPath() = %#v, want %#v", *got, *want)
	}
}
