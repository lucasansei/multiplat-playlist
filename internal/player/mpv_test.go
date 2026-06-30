package player

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMPVPlayerPlayDoesNotHoldLockWhileWaiting(t *testing.T) {
	installFakeMPV(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &MPVPlayer{
		dial: func(network string, address string, timeout time.Duration) (net.Conn, error) {
			clientConn, serverConn := net.Pipe()
			go handleFakeMPVConnection(serverConn)
			return clientConn, nil
		},
	}
	started := make(chan struct{})
	playDone := make(chan error, 1)

	go func() {
		playDone <- p.Play(ctx, "fake://stream", func(PlaybackSession) error {
			close(started)
			return nil
		})
	}()

	select {
	case <-started:
	case err := <-playDone:
		t.Fatalf("Play() exited before playback start: %v\nfake mpv log:\n%s", err, readFakeMPVLog(t))
	case <-time.After(2 * time.Second):
		t.Fatalf("Play() did not report playback start\nfake mpv log:\n%s", readFakeMPVLog(t))
	}
	assertFakeMPVArgs(t, readFakeMPVLog(t))

	pauseDone := make(chan error, 1)
	go func() {
		pauseDone <- p.Pause()
	}()

	select {
	case err := <-pauseDone:
		if err != nil {
			t.Fatalf("Pause() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Pause() blocked while Play() was waiting for mpv to exit")
	}

	cancel()

	select {
	case err := <-playDone:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("Play() error after cancellation = %v, want context canceled or nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Play() did not exit after context cancellation")
	}
}

func assertFakeMPVArgs(t *testing.T, log string) {
	t.Helper()

	wantParts := []string{
		"--no-video --really-quiet --no-terminal",
		"--input-ipc-server=",
		"--idle=yes fake://stream",
	}
	for _, want := range wantParts {
		if !strings.Contains(log, want) {
			t.Fatalf("fake mpv log = %q, want to contain %q", log, want)
		}
	}
}

func TestMPVPlayerCleanupIsIdempotent(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "mpv.sock")
	p := &MPVPlayer{
		socketPath: socketPath,
	}
	if err := os.WriteFile(socketPath, nil, 0o600); err != nil {
		t.Fatalf("write socket placeholder: %v", err)
	}

	if err := p.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket placeholder still exists or stat failed: %v", err)
	}
}

func installFakeMPV(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	socketParent := "/tmp"
	if _, err := os.Stat("/private/tmp"); err == nil {
		socketParent = "/private/tmp"
	}
	socketDir, err := os.MkdirTemp(socketParent, "mplay-mpv-test-")
	if err != nil {
		socketDir = t.TempDir()
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(socketDir)
	})

	mpvPath := filepath.Join(dir, "mpv")
	logPath := filepath.Join(dir, "fake-mpv.log")
	script := `#!/bin/sh
socket_path=
for arg in "$@"; do
	case "$arg" in
		--input-ipc-server=*) socket_path=${arg#--input-ipc-server=} ;;
	esac
done
{
	echo "args: $*"
	echo "socket_path: $socket_path"
} >> "$MPLAY_FAKE_MPV_LOG"
if [ -z "$socket_path" ]; then
	echo "missing socket path" >> "$MPLAY_FAKE_MPV_LOG"
	exit 2
fi
if ! : > "$socket_path"; then
	echo "touch socket path failed" >> "$MPLAY_FAKE_MPV_LOG"
	exit 2
fi
exec sleep 3600
`
	if err := os.WriteFile(mpvPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake mpv: %v", err)
	}

	t.Setenv("MPLAY_FAKE_MPV_LOG", logPath)
	t.Setenv("TMPDIR", socketDir)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func readFakeMPVLog(t *testing.T) string {
	t.Helper()

	path := os.Getenv("MPLAY_FAKE_MPV_LOG")
	if path == "" {
		return "(no log path)"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

func handleFakeMPVConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)
	for {
		var command mpvCommand
		if err := decoder.Decode(&command); err != nil {
			return
		}

		data := json.RawMessage("null")
		if len(command.Command) >= 2 && command.Command[0] == "get_property" && command.Command[1] == "pause" {
			data = json.RawMessage("false")
		}

		_ = encoder.Encode(mpvResponse{
			RequestID: command.RequestID,
			Error:     "success",
			Data:      data,
		})
	}
}
