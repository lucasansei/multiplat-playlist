// internal/player/player.go
package player

import (
	"context"
)

// Player defines the interface for audio playback control
type Player interface {
	Play(ctx context.Context, url string) error
	Pause() error
	Resume() error
	Stop() error
	Status() string
	IsAvailable() bool
	Close() error
}
