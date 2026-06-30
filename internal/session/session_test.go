package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadClear(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	startedAt := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	want := State{
		Player:     "mpv",
		PID:        12345,
		SocketPath: filepath.Join(os.TempDir(), "mpv-socket-test"),
		Track: Track{
			Platform: "youtube",
			ID:       "dQw4w9WgXcQ",
			URL:      "https://youtu.be/dQw4w9WgXcQ",
		},
		QueueIndex: 0,
		QueueSize:  3,
		StartedAt:  startedAt,
	}

	if err := Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got == nil {
		t.Fatal("Load() returned nil state")
	}

	if *got != want {
		t.Fatalf("Load() = %#v, want %#v", *got, want)
	}

	if err := Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	got, err = Load()
	if err != nil {
		t.Fatalf("Load() after Clear() error = %v", err)
	}
	if got != nil {
		t.Fatalf("Load() after Clear() = %#v, want nil", got)
	}
}

func TestLoadMissingSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != nil {
		t.Fatalf("Load() = %#v, want nil", got)
	}
}
