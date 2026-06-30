// internal/player/player.go
package player

import (
	"context"
)

type PlaybackSession struct {
	PID        int
	SocketPath string
}

type StartFunc func(PlaybackSession) error

// Player defines the interface for audio playback control
type Player interface {
	Play(ctx context.Context, url string, onStart StartFunc) error
	Pause() error
	Resume() error
	Stop() error
	Status() string
	IsAvailable() bool
	Close() error
}
