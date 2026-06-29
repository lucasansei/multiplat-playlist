// internal/app/app.go
package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lucasansei/multiplat-playlist/internal/config"
	"github.com/lucasansei/multiplat-playlist/internal/parser"
	"github.com/lucasansei/multiplat-playlist/internal/player"
	"github.com/lucasansei/multiplat-playlist/internal/queue"
	"github.com/lucasansei/multiplat-playlist/internal/youtube"
)

var (
	ErrEmptyQueue    = errors.New("queue is empty")
	ErrQueueFinished = errors.New("queue finished")
	ErrPlayerMissing = errors.New("player is not configured")
)

type App struct {
	config *config.Config
	queue  *queue.Queue
	player player.Player
}

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
		config: cfg,
		queue:  q,
		player: p,
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
		config: cfg,
		queue:  q,
	}, nil
}

func NewConfig() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &App{
		config: cfg,
	}, nil
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

	fmt.Printf("▶ Playing: %s\n", url)
	return a.player.Play(ctx, streamURL)
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

		if err := a.player.Play(ctx, streamURL); err != nil {
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
	if a.player == nil {
		return ErrPlayerMissing
	}

	if err := a.player.Pause(); err != nil {
		return err
	}
	fmt.Println("⏸ Paused")
	return nil
}

func (a *App) Resume() error {
	if a.player == nil {
		return ErrPlayerMissing
	}

	if err := a.player.Resume(); err != nil {
		return err
	}
	fmt.Println("▶ Resumed")
	return nil
}

func (a *App) Next() error {
	if a.player == nil {
		return ErrPlayerMissing
	}

	if err := a.player.Stop(); err != nil {
		return err
	}
	fmt.Println("⏭ Skipped to next")
	return nil
}

func (a *App) Stop() error {
	if a.player == nil {
		return ErrPlayerMissing
	}

	if err := a.player.Stop(); err != nil {
		return err
	}
	fmt.Println("⏹ Stopped")
	return nil
}

func (a *App) Status() error {
	if a.player == nil {
		return ErrPlayerMissing
	}

	status := a.player.Status()
	var current *queue.Track
	queueSize := 0
	currentIndex := -1
	if a.queue != nil {
		current = a.queue.Current()
		queueSize = a.queue.Size()
		currentIndex = a.queue.CurrentIndex()
	}

	fmt.Printf("Status: %s\n", status)

	if current != nil {
		fmt.Printf("Current: [%s] %s\n", current.Platform, current.URL)
		fmt.Printf("Queue position: %d/%d\n", currentIndex+1, queueSize)
	} else {
		fmt.Println("No track currently playing")
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

func (a *App) getStreamURL(parsed *parser.ParsedURL) (string, error) {
	switch parsed.Platform {
	case parser.PlatformYouTube:
		return youtube.GetStreamURL(parsed.ID)
	case parser.PlatformSpotify:
		return "", errors.New("spotify not yet implemented")
	default:
		return "", fmt.Errorf("unsupported platform: %s", parsed.Platform)
	}
}
