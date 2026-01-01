package player

import (
	"os/exec"
)

type Player struct {
	backend string
}

func New(backend string) *Player {
	return &Player{backend: backend}
}

func (p *Player) Play(url string) error {
	cmd := exec.Command(p.backend, "--no-video", url)
	return cmd.Run()
}

func (p *Player) IsAvailable() bool {
	_, err := exec.LookPath(p.backend)
	return err == nil
}
