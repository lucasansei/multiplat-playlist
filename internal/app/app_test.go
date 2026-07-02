package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasansei/multiplat-playlist/internal/config"
	"github.com/lucasansei/multiplat-playlist/internal/parser"
	"github.com/lucasansei/multiplat-playlist/internal/player"
	"github.com/lucasansei/multiplat-playlist/internal/queue"
	"github.com/lucasansei/multiplat-playlist/internal/session"
)

func TestPlayURLUsesResolverAndPlayer(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	fp := &fakePlayer{}
	var resolved *parser.ParsedURL
	a := newWithDependencies(&config.Config{}, nil, fp, func(_ context.Context, parsed *parser.ParsedURL) (string, error) {
		resolved = parsed
		return "stream://" + parsed.ID, nil
	})

	err := a.PlayURL(context.Background(), "https://youtu.be/dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("PlayURL() error = %v", err)
	}

	if resolved == nil {
		t.Fatal("resolver was not called")
	}
	if resolved.Platform != parser.PlatformYouTube || resolved.ID != "dQw4w9WgXcQ" {
		t.Fatalf("resolved URL = %#v, want youtube dQw4w9WgXcQ", resolved)
	}
	if len(fp.playedURLs) != 1 || fp.playedURLs[0] != "stream://dQw4w9WgXcQ" {
		t.Fatalf("played URLs = %#v, want stream URL", fp.playedURLs)
	}

	state, err := session.Load()
	if err != nil {
		t.Fatalf("session.Load() error = %v", err)
	}
	if state != nil {
		t.Fatalf("session after PlayURL() = %#v, want nil", state)
	}
}

func TestPlayURLWritesDirectPlaybackSessionMetadata(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var saved session.State
	fp := &fakePlayer{
		afterStart: func() {
			state, err := session.Load()
			if err != nil {
				t.Fatalf("session.Load() error = %v", err)
			}
			if state == nil {
				t.Fatal("session.Load() = nil, want active session")
			}
			saved = *state
		},
	}
	a := newWithDependencies(&config.Config{}, nil, fp, func(_ context.Context, parsed *parser.ParsedURL) (string, error) {
		return "stream://" + parsed.ID, nil
	})

	if err := a.PlayURL(context.Background(), "https://youtu.be/dQw4w9WgXcQ"); err != nil {
		t.Fatalf("PlayURL() error = %v", err)
	}

	if saved.PlaybackKind != session.PlaybackKindDirect {
		t.Fatalf("PlaybackKind = %q, want direct", saved.PlaybackKind)
	}
	if saved.ControllerPID != os.Getpid() {
		t.Fatalf("ControllerPID = %d, want current process %d", saved.ControllerPID, os.Getpid())
	}
	if saved.QueueIndex != -1 || saved.QueueSize != 0 {
		t.Fatalf("queue metadata = %d/%d, want -1/0", saved.QueueIndex, saved.QueueSize)
	}
}

func TestPlayURLReturnsResolverError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	wantErr := errors.New("resolver failed")
	fp := &fakePlayer{}
	a := newWithDependencies(&config.Config{}, nil, fp, func(_ context.Context, _ *parser.ParsedURL) (string, error) {
		return "", wantErr
	})

	err := a.PlayURL(context.Background(), "https://youtu.be/dQw4w9WgXcQ")
	if !errors.Is(err, wantErr) {
		t.Fatalf("PlayURL() error = %v, want %v", err, wantErr)
	}
	if len(fp.playedURLs) != 0 {
		t.Fatalf("played URLs = %#v, want none", fp.playedURLs)
	}
}

func TestQueueAddAddsParsedTrackAndClosePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")
	q, err := queue.LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	a := newWithDependencies(&config.Config{}, q, nil, nil)
	if err := a.QueueAdd("spotify:track:3n3Ppam7vgaVa1iaRUc9Lp"); err != nil {
		t.Fatalf("QueueAdd() error = %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	got, err := queue.LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath() after Close() error = %v", err)
	}
	tracks := got.List()
	if len(tracks) != 1 {
		t.Fatalf("loaded track count = %d, want 1", len(tracks))
	}
	want := queue.Track{
		Platform: "spotify",
		ID:       "3n3Ppam7vgaVa1iaRUc9Lp",
		URL:      "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp",
	}
	if tracks[0] != want {
		t.Fatalf("loaded track = %#v, want %#v", tracks[0], want)
	}
}

func TestQueuePlayEmpty(t *testing.T) {
	q, err := queue.LoadFromPath(filepath.Join(t.TempDir(), "queue.json"))
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	a := newWithDependencies(&config.Config{}, q, &fakePlayer{}, nil)
	err = a.QueuePlay(context.Background())
	if !errors.Is(err, ErrEmptyQueue) {
		t.Fatalf("QueuePlay() error = %v, want %v", err, ErrEmptyQueue)
	}
}

func TestQueuePlayPlaysTracksInOrder(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	q, err := queue.LoadFromPath(filepath.Join(t.TempDir(), "queue.json"))
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}
	q.Add(queue.Track{Platform: "youtube", ID: "one", URL: "https://youtu.be/one"})
	q.Add(queue.Track{Platform: "youtube", ID: "two", URL: "https://youtu.be/two"})

	fp := &fakePlayer{}
	a := newWithDependencies(&config.Config{}, q, fp, func(_ context.Context, parsed *parser.ParsedURL) (string, error) {
		return "stream://" + parsed.ID, nil
	})

	if err := a.QueuePlay(context.Background()); err != nil {
		t.Fatalf("QueuePlay() error = %v", err)
	}

	want := []string{"stream://one", "stream://two"}
	if len(fp.playedURLs) != len(want) {
		t.Fatalf("played URL count = %d, want %d: %#v", len(fp.playedURLs), len(want), fp.playedURLs)
	}
	for i := range want {
		if fp.playedURLs[i] != want[i] {
			t.Fatalf("playedURLs[%d] = %q, want %q", i, fp.playedURLs[i], want[i])
		}
	}
	if q.CurrentIndex() != 1 {
		t.Fatalf("CurrentIndex() = %d, want 1", q.CurrentIndex())
	}
}

func TestQueuePlayWritesQueuePlaybackSessionMetadata(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	q, err := queue.LoadFromPath(filepath.Join(t.TempDir(), "queue.json"))
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}
	q.Add(queue.Track{Platform: "youtube", ID: "one", URL: "https://youtu.be/one"})

	var saved session.State
	fp := &fakePlayer{
		afterStart: func() {
			state, err := session.Load()
			if err != nil {
				t.Fatalf("session.Load() error = %v", err)
			}
			if state == nil {
				t.Fatal("session.Load() = nil, want active session")
			}
			saved = *state
		},
	}
	a := newWithDependencies(&config.Config{}, q, fp, func(_ context.Context, parsed *parser.ParsedURL) (string, error) {
		return "stream://" + parsed.ID, nil
	})

	if err := a.QueuePlay(context.Background()); err != nil {
		t.Fatalf("QueuePlay() error = %v", err)
	}

	if saved.PlaybackKind != session.PlaybackKindQueue {
		t.Fatalf("PlaybackKind = %q, want queue", saved.PlaybackKind)
	}
	if saved.ControllerPID != os.Getpid() {
		t.Fatalf("ControllerPID = %d, want current process %d", saved.ControllerPID, os.Getpid())
	}
	if saved.QueueIndex != 0 || saved.QueueSize != 1 {
		t.Fatalf("queue metadata = %d/%d, want 0/1", saved.QueueIndex, saved.QueueSize)
	}
}

func TestStatusClearsSessionWhenMPVIPCIsUnreachable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	socketPath := filepath.Join(t.TempDir(), "mpv.sock")
	if err := os.WriteFile(socketPath, nil, 0o600); err != nil {
		t.Fatalf("write socket placeholder: %v", err)
	}
	if err := session.Save(session.State{
		Player:     "mpv",
		PID:        os.Getpid(),
		SocketPath: socketPath,
		Track: session.Track{
			Platform: "youtube",
			ID:       "one",
			URL:      "https://youtu.be/one",
		},
		QueueIndex: 0,
		QueueSize:  2,
	}); err != nil {
		t.Fatalf("session.Save() error = %v", err)
	}

	a := newWithDependencies(&config.Config{}, nil, nil, nil)
	if err := a.Status(); err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	state, err := session.Load()
	if err != nil {
		t.Fatalf("session.Load() error = %v", err)
	}
	if state != nil {
		t.Fatalf("session after Status() = %#v, want nil", state)
	}
}

func TestNextRequiresActivePlaybackSessionAndDoesNotAdvanceQueue(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	q, err := queue.LoadFromPath(filepath.Join(t.TempDir(), "queue.json"))
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}
	q.Add(queue.Track{Platform: "youtube", ID: "one", URL: "https://youtu.be/one"})
	q.Add(queue.Track{Platform: "youtube", ID: "two", URL: "https://youtu.be/two"})
	if got := q.Next(); got == nil {
		t.Fatal("Next() returned nil, want first queue item")
	}

	a := newWithDependencies(&config.Config{}, q, nil, nil)
	err = a.Next()
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("Next() error = %v, want %v", err, ErrNoSession)
	}
	if q.CurrentIndex() != 0 {
		t.Fatalf("CurrentIndex() after app Next() = %d, want 0", q.CurrentIndex())
	}
}

func TestPlayURLPlaysSpotifyPreviewURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	fp := &fakePlayer{}
	sp := &fakeSpotifyPreviewClient{previewURL: "https://preview.example/track.mp3"}
	a := newWithDependencies(&config.Config{
		Spotify: config.SpotifyConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		},
	}, nil, fp, nil)
	a.spotifyClient = sp

	if err := a.PlayURL(context.Background(), "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp"); err != nil {
		t.Fatalf("PlayURL() error = %v", err)
	}

	if sp.trackID != "3n3Ppam7vgaVa1iaRUc9Lp" {
		t.Fatalf("spotify track ID = %q, want 3n3Ppam7vgaVa1iaRUc9Lp", sp.trackID)
	}
	if len(fp.playedURLs) != 1 || fp.playedURLs[0] != "https://preview.example/track.mp3" {
		t.Fatalf("played URLs = %#v, want spotify preview URL", fp.playedURLs)
	}
}

func TestPlayURLReturnsSpotifyPreviewError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	wantErr := errors.New("spotify failed")
	fp := &fakePlayer{}
	a := newWithDependencies(&config.Config{
		Spotify: config.SpotifyConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		},
	}, nil, fp, nil)
	a.spotifyClient = &fakeSpotifyPreviewClient{err: wantErr}

	err := a.PlayURL(context.Background(), "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp")
	if !errors.Is(err, wantErr) {
		t.Fatalf("PlayURL() error = %v, want %v", err, wantErr)
	}
	if len(fp.playedURLs) != 0 {
		t.Fatalf("played URLs = %#v, want none", fp.playedURLs)
	}
}

type fakePlayer struct {
	playedURLs []string
	playErr    error
	socketPath string
	afterStart func()
}

func (f *fakePlayer) Play(_ context.Context, url string, onStart player.StartFunc) error {
	f.playedURLs = append(f.playedURLs, url)
	if onStart != nil {
		socketPath := f.socketPath
		if socketPath == "" {
			socketPath = filepath.Join(os.TempDir(), "mplay-app-test.sock")
		}
		if err := onStart(player.PlaybackSession{
			PID:        os.Getpid(),
			SocketPath: socketPath,
		}); err != nil {
			return err
		}
		if f.afterStart != nil {
			f.afterStart()
		}
	}
	return f.playErr
}

func (f *fakePlayer) Pause() error {
	return nil
}

func (f *fakePlayer) Resume() error {
	return nil
}

func (f *fakePlayer) Stop() error {
	return nil
}

func (f *fakePlayer) Status() string {
	return "playing"
}

func (f *fakePlayer) IsAvailable() bool {
	return true
}

func (f *fakePlayer) Close() error {
	return nil
}

type fakeSpotifyPreviewClient struct {
	trackID    string
	previewURL string
	err        error
}

func (f *fakeSpotifyPreviewClient) PreviewURL(_ context.Context, trackID string) (string, error) {
	f.trackID = trackID
	if f.err != nil {
		return "", f.err
	}
	return f.previewURL, nil
}
