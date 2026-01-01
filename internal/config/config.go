package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".config", "multiplat-playlist", "config.json")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return getDefaultConfig(), nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func getDefaultConfig() *Config {
	return &Config{
		Player: PlayerConfig{
			Backend: "mpv",
			Volume:  100,
		},
	}
}

func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "multiplat-playlist")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
