package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	configDirName   = ".config"
	appDirName      = "multiplat-playlist"
	sessionFileName = "session.json"
)

type Track struct {
	Platform string `json:"platform"`
	ID       string `json:"id"`
	URL      string `json:"url"`
}

type State struct {
	Player     string    `json:"player"`
	PID        int       `json:"pid"`
	SocketPath string    `json:"socket_path"`
	Track      Track     `json:"track"`
	QueueIndex int       `json:"queue_index"`
	QueueSize  int       `json:"queue_size"`
	StartedAt  time.Time `json:"started_at"`
}

func Save(state State) error {
	path, err := Path()
	if err != nil {
		return fmt.Errorf("get session path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write session: %w", err)
	}

	return nil
}

func Load() (*State, error) {
	path, err := Path()
	if err != nil {
		return nil, fmt.Errorf("get session path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read session: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	return &state, nil
}

func Clear() error {
	path, err := Path()
	if err != nil {
		return fmt.Errorf("get session path: %w", err)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove session: %w", err)
	}

	return nil
}

func IsActive(state State) bool {
	if state.PID <= 0 || state.SocketPath == "" {
		return false
	}

	if _, err := os.Stat(state.SocketPath); err != nil {
		return false
	}

	process, err := os.FindProcess(state.PID)
	if err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, configDirName, appDirName, sessionFileName), nil
}
