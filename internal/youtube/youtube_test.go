package youtube

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetStreamURLUsesYTDLP(t *testing.T) {
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	installFakeYTDLP(t, `#!/bin/sh
printf '%s\n' "$@" > "$MPLAY_FAKE_YTDLP_ARGS"
printf '  https://stream.example/audio.webm  \n'
`, map[string]string{
		"MPLAY_FAKE_YTDLP_ARGS": argsPath,
	})

	got, err := GetStreamURL(context.Background(), "video-id")
	if err != nil {
		t.Fatalf("GetStreamURL() error = %v", err)
	}
	if got != "https://stream.example/audio.webm" {
		t.Fatalf("GetStreamURL() = %q, want trimmed stream URL", got)
	}

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args log: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(argsData)), "\n")
	want := []string{"-g", "-f", "bestaudio", "https://www.youtube.com/watch?v=video-id"}
	if len(args) != len(want) {
		t.Fatalf("yt-dlp args = %#v, want %#v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("yt-dlp args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestGetStreamURLIncludesStderrOnFailure(t *testing.T) {
	installFakeYTDLP(t, `#!/bin/sh
echo 'video unavailable' >&2
exit 42
`, nil)

	_, err := GetStreamURL(context.Background(), "video-id")
	if err == nil {
		t.Fatal("GetStreamURL() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "yt-dlp: video unavailable") {
		t.Fatalf("GetStreamURL() error = %v, want stderr context", err)
	}
}

func TestGetStreamURLHonorsContextCancellation(t *testing.T) {
	installFakeYTDLP(t, `#!/bin/sh
exec sleep 5
`, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := GetStreamURL(ctx, "video-id")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetStreamURL() error = %v, want context canceled", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("GetStreamURL() took %s after cancellation, want under 1s", elapsed)
	}
}

func installFakeYTDLP(t *testing.T, script string, env map[string]string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "yt-dlp")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake yt-dlp: %v", err)
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for key, value := range env {
		t.Setenv(key, value)
	}
}
