// internal/config/config.go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName  = ".config"
	appDirName     = "multiplat-playlist"
	configFileName = "config.json"
)

type Config struct {
	Spotify SpotifyConfig `json:"spotify"`
	Player  PlayerConfig  `json:"player"`
}

type SpotifyConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type PlayerConfig struct {
	Backend string `json:"backend"`
	Volume  int    `json:"volume"`
}

func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, fmt.Errorf("get config path: %w", err)
	}

	return LoadFromPath(path)
}

func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return getDefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return fmt.Errorf("get config path: %w", err)
	}

	return c.SaveToPath(path)
}

func (c *Config) SaveToPath(path string) error {
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName, appDirName, configFileName), nil
}

func getDefaultConfig() *Config {
	return &Config{
		Player: PlayerConfig{
			Backend: "mpv",
			Volume:  100,
		},
	}
}
