// internal/app/app.go
package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lucasansei/multiplat-playlist/internal/config"
	"github.com/lucasansei/multiplat-playlist/internal/parser"
	"github.com/lucasansei/multiplat-playlist/internal/player"
	"github.com/lucasansei/multiplat-playlist/internal/queue"
	"github.com/lucasansei/multiplat-playlist/internal/session"
	"github.com/lucasansei/multiplat-playlist/internal/youtube"
)

var (
	ErrEmptyQueue    = errors.New("queue is empty")
	ErrQueueFinished = errors.New("queue finished")
	ErrPlayerMissing = errors.New("player is not configured")
	ErrNoSession     = errors.New("no active playback session")
)

type App struct {
	config        *config.Config
	queue         *queue.Queue
	player        player.Player
	streamResolve streamResolver
}

type streamResolver func(*parser.ParsedURL) (string, error)

func New() (*App, error) {
	return NewPlayback()
}

func NewPlayback() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	q, err := queue.Load()
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}

	p := player.NewMPV()
	if !p.IsAvailable() {
		return nil, fmt.Errorf("player mpv not found in PATH")
	}

	return &App{
		config:        cfg,
		queue:         q,
		player:        p,
		streamResolve: defaultStreamResolver,
	}, nil
}

func NewQueue() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	q, err := queue.Load()
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}

	return &App{
		config:        cfg,
		queue:         q,
		streamResolve: defaultStreamResolver,
	}, nil
}

func NewConfig() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &App{
		config:        cfg,
		streamResolve: defaultStreamResolver,
	}, nil
}

func NewControl() (*App, error) {
	return &App{
		streamResolve: defaultStreamResolver,
	}, nil
}

func newWithDependencies(cfg *config.Config, q *queue.Queue, p player.Player, resolver streamResolver) *App {
	if resolver == nil {
		resolver = defaultStreamResolver
	}
	return &App{
		config:        cfg,
		queue:         q,
		player:        p,
		streamResolve: resolver,
	}
}

func (a *App) Close() error {
	if a.player != nil {
		if err := a.player.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error closing player: %v\n", err)
		}
	}

	if a.queue != nil {
		if err := a.queue.Save(); err != nil {
			return fmt.Errorf("save queue: %w", err)
		}
	}

	return nil
}

func (a *App) PlayURL(ctx context.Context, url string) error {
	if a.player == nil {
		return ErrPlayerMissing
	}

	parsed, err := parser.Parse(url)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	streamURL, err := a.getStreamURL(parsed)
	if err != nil {
		return fmt.Errorf("get stream: %w", err)
	}

	track := queue.Track{
		Platform: string(parsed.Platform),
		ID:       parsed.ID,
		URL:      parsed.Original,
	}

	fmt.Printf("▶ Playing: %s\n", url)
	return a.playStream(ctx, streamURL, track, -1, 0)
}

func (a *App) QueueAdd(url string) error {
	if a.queue == nil {
		return errors.New("queue is not configured")
	}

	parsed, err := parser.Parse(url)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	track := queue.Track{
		Platform: string(parsed.Platform),
		ID:       parsed.ID,
		URL:      parsed.Original,
	}

	a.queue.Add(track)
	fmt.Printf("✓ Added to queue: %s\n", url)
	fmt.Printf("Queue size: %d\n", a.queue.Size())

	return nil
}

func (a *App) QueuePlay(ctx context.Context) error {
	if a.player == nil {
		return ErrPlayerMissing
	}
	if a.queue == nil {
		return errors.New("queue is not configured")
	}

	if a.queue.Size() == 0 {
		return ErrEmptyQueue
	}

	fmt.Printf("Playing queue (%d songs)...\n", a.queue.Size())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		track := a.queue.Next()
		if track == nil {
			fmt.Println("✓ Queue finished")
			return nil
		}

		parsed := &parser.ParsedURL{
			Platform: parser.Platform(track.Platform),
			ID:       track.ID,
			Original: track.URL,
		}

		streamURL, err := a.getStreamURL(parsed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting stream: %v, skipping...\n", err)
			continue
		}

		fmt.Printf("▶ Playing [%d/%d]: %s\n", a.queue.CurrentIndex()+1, a.queue.Size(), track.URL)

		if err := a.playStream(ctx, streamURL, *track, a.queue.CurrentIndex(), a.queue.Size()); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Playback error: %v, skipping...\n", err)
			continue
		}
	}
}

func (a *App) QueueList() error {
	if a.queue == nil {
		return errors.New("queue is not configured")
	}

	if a.queue.Size() == 0 {
		fmt.Println("Queue is empty")
		return nil
	}

	fmt.Printf("Queue (%d songs):\n", a.queue.Size())
	tracks := a.queue.List()
	current := a.queue.CurrentIndex()

	for i, track := range tracks {
		marker := " "
		if i == current {
			marker = "▶"
		}
		fmt.Printf("%s %d. [%s] %s\n", marker, i+1, track.Platform, track.URL)
	}

	return nil
}

func (a *App) QueueClear() error {
	if a.queue == nil {
		return errors.New("queue is not configured")
	}

	a.queue.Clear()
	fmt.Println("✓ Queue cleared")
	return nil
}

func (a *App) Pause() error {
	if err := a.withActiveMPV(func(_ *session.State, client *player.MPVIPCClient) error {
		return client.Pause()
	}); err != nil {
		return err
	}
	fmt.Println("⏸ Paused")
	return nil
}

func (a *App) Resume() error {
	if err := a.withActiveMPV(func(_ *session.State, client *player.MPVIPCClient) error {
		return client.Resume()
	}); err != nil {
		return err
	}
	fmt.Println("▶ Resumed")
	return nil
}

func (a *App) Next() error {
	// In the current foreground queue model, next only stops active playback.
	// A running QueuePlay loop owns queue index advancement after MPV exits.
	if err := a.stopActiveSession(); err != nil {
		return err
	}
	fmt.Println("⏭ Skipped to next")
	return nil
}

func (a *App) Stop() error {
	if err := a.stopActiveSession(); err != nil {
		return err
	}
	fmt.Println("⏹ Stopped")
	return nil
}

func (a *App) Status() error {
	state, client, err := a.activeMPV()
	if err != nil {
		if errors.Is(err, ErrNoSession) {
			fmt.Println("Status: stopped")
			fmt.Println("No track currently playing")
			return nil
		}
		return err
	}
	defer client.Close()

	status, err := client.Status()
	if err != nil {
		return fmt.Errorf("get player status: %w", err)
	}

	fmt.Printf("Status: %s\n", status)
	if state.Track.URL != "" {
		fmt.Printf("Current: [%s] %s\n", state.Track.Platform, state.Track.URL)
	} else {
		fmt.Println("No track currently playing")
	}
	if state.QueueIndex >= 0 && state.QueueSize > 0 {
		fmt.Printf("Queue position: %d/%d\n", state.QueueIndex+1, state.QueueSize)
	}

	return nil
}

func (a *App) AuthSpotify() error {
	fmt.Println("Spotify authentication setup:")
	fmt.Println("1. Go to https://developer.spotify.com/dashboard")
	fmt.Println("2. Create an app and get your Client ID and Secret")
	fmt.Println("3. Enter them below:")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Spotify Client ID: ")
	clientID, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read client ID: %w", err)
	}
	clientID = strings.TrimSpace(clientID)

	fmt.Print("Enter Spotify Client Secret: ")
	clientSecret, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read client secret: %w", err)
	}
	clientSecret = strings.TrimSpace(clientSecret)

	if clientID == "" || clientSecret == "" {
		return errors.New("client ID and secret cannot be empty")
	}

	a.config.Spotify.ClientID = clientID
	a.config.Spotify.ClientSecret = clientSecret

	if err := a.config.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println("✓ Spotify credentials saved!")
	return nil
}

func (a *App) stopActiveSession() error {
	return a.withActiveMPV(func(_ *session.State, client *player.MPVIPCClient) error {
		if err := client.Stop(); err != nil {
			return err
		}
		if err := session.Clear(); err != nil {
			return fmt.Errorf("clear session: %w", err)
		}
		return nil
	})
}

func (a *App) withActiveMPV(fn func(*session.State, *player.MPVIPCClient) error) error {
	state, client, err := a.activeMPV()
	if err != nil {
		return err
	}
	defer client.Close()

	return fn(state, client)
}

func (a *App) activeMPV() (*session.State, *player.MPVIPCClient, error) {
	state, err := a.activeSession()
	if err != nil {
		return nil, nil, err
	}
	if state.Player != "mpv" {
		return nil, nil, fmt.Errorf("unsupported active player: %s", state.Player)
	}

	client, err := player.DialMPVIPC(state.SocketPath)
	if err != nil {
		if clearErr := session.Clear(); clearErr != nil {
			return nil, nil, fmt.Errorf("clear stale session: %w", clearErr)
		}
		return nil, nil, ErrNoSession
	}

	return state, client, nil
}

func (a *App) activeSession() (*session.State, error) {
	state, err := session.Load()
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	if state == nil {
		return nil, ErrNoSession
	}
	if !session.IsActive(*state) {
		if err := session.Clear(); err != nil {
			return nil, fmt.Errorf("clear stale session: %w", err)
		}
		return nil, ErrNoSession
	}
	return state, nil
}

func (a *App) playStream(ctx context.Context, streamURL string, track queue.Track, queueIndex int, queueSize int) error {
	if err := session.Clear(); err != nil {
		return fmt.Errorf("clear previous session: %w", err)
	}

	sessionSaved := false
	err := a.player.Play(ctx, streamURL, func(playback player.PlaybackSession) error {
		state := session.State{
			Player:     "mpv",
			PID:        playback.PID,
			SocketPath: playback.SocketPath,
			Track: session.Track{
				Platform: track.Platform,
				ID:       track.ID,
				URL:      track.URL,
			},
			QueueIndex: queueIndex,
			QueueSize:  queueSize,
			StartedAt:  time.Now().UTC(),
		}

		if err := session.Save(state); err != nil {
			return err
		}
		sessionSaved = true
		return nil
	})

	if sessionSaved {
		clearErr := session.Clear()
		if err != nil {
			if clearErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: error clearing session: %v\n", clearErr)
			}
			return err
		}
		if clearErr != nil {
			return fmt.Errorf("clear session: %w", clearErr)
		}
	}

	return err
}

func (a *App) getStreamURL(parsed *parser.ParsedURL) (string, error) {
	if a.streamResolve != nil {
		return a.streamResolve(parsed)
	}
	return defaultStreamResolver(parsed)
}

func defaultStreamResolver(parsed *parser.ParsedURL) (string, error) {
	switch parsed.Platform {
	case parser.PlatformYouTube:
		return youtube.GetStreamURL(parsed.ID)
	case parser.PlatformSpotify:
		return "", errors.New("spotify not yet implemented")
	default:
		return "", fmt.Errorf("unsupported platform: %s", parsed.Platform)
	}
}
