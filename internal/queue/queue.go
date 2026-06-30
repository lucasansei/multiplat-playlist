// internal/queue/queue.go
package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName = ".config"
	appDirName    = "multiplat-playlist"
	queueFileName = "queue.json"
)

type Track struct {
	Platform string `json:"platform"`
	ID       string `json:"id"`
	URL      string `json:"url"`
}

type Queue struct {
	tracks []Track
	index  int
	path   string
}

type queueData struct {
	Tracks []Track `json:"tracks"`
	Index  int     `json:"index"`
}

func Load() (*Queue, error) {
	path, err := Path()
	if err != nil {
		return nil, fmt.Errorf("get queue path: %w", err)
	}

	return LoadFromPath(path)
}

func LoadFromPath(path string) (*Queue, error) {
	q := &Queue{
		tracks: make([]Track, 0),
		index:  -1,
		path:   path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return q, nil
		}
		return nil, fmt.Errorf("read queue: %w", err)
	}

	var saved queueData
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("parse queue: %w", err)
	}

	q.tracks = saved.Tracks
	q.index = normalizeIndex(saved.Index, len(saved.Tracks))

	return q, nil
}

func (q *Queue) Save() error {
	configDir := filepath.Dir(q.path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create queue dir: %w", err)
	}

	saved := queueData{
		Tracks: q.tracks,
		Index:  q.index,
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal queue: %w", err)
	}

	if err := os.WriteFile(q.path, data, 0644); err != nil {
		return fmt.Errorf("write queue: %w", err)
	}

	return nil
}

func (q *Queue) Add(track Track) {
	q.tracks = append(q.tracks, track)
}

func (q *Queue) Next() *Track {
	if q.index+1 < len(q.tracks) {
		q.index++
		return &q.tracks[q.index]
	}
	return nil
}

func (q *Queue) Current() *Track {
	if q.index >= 0 && q.index < len(q.tracks) {
		return &q.tracks[q.index]
	}
	return nil
}

func (q *Queue) CurrentIndex() int {
	return q.index
}

func (q *Queue) List() []Track {
	return q.tracks
}

func (q *Queue) Size() int {
	return len(q.tracks)
}

func (q *Queue) Clear() {
	q.tracks = make([]Track, 0)
	q.index = -1
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName, appDirName, queueFileName), nil
}

func normalizeIndex(index int, size int) int {
	if size == 0 || index < -1 || index >= size {
		return -1
	}
	return index
}
