package main

import (
	"context"

	"github.com/lucasansei/multiplat-playlist/internal/app"
)

type commandApp interface {
	PlayURL(context.Context, string) error
	QueueAdd(string) error
	QueuePlay(context.Context) error
	QueueList() error
	QueueClear() error
	Pause() error
	Resume() error
	Next() error
	Stop() error
	Status() error
	AuthSpotify() error
	Close() error
}

type appFactory func() (commandApp, error)

type appFactories struct {
	playback appFactory
	queue    appFactory
	config   appFactory
	control  appFactory
}

func defaultAppFactories() appFactories {
	return appFactories{
		playback: func() (commandApp, error) {
			return app.NewPlayback()
		},
		queue: func() (commandApp, error) {
			return app.NewQueue()
		},
		config: func() (commandApp, error) {
			return app.NewConfig()
		},
		control: func() (commandApp, error) {
			return app.NewControl()
		},
	}
}

func (f appFactories) withDefaults() appFactories {
	defaults := defaultAppFactories()
	if f.playback == nil {
		f.playback = defaults.playback
	}
	if f.queue == nil {
		f.queue = defaults.queue
	}
	if f.config == nil {
		f.config = defaults.config
	}
	if f.control == nil {
		f.control = defaults.control
	}
	return f
}
