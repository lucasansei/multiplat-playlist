package main

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestRootPlayUsesPlaybackFactory(t *testing.T) {
	playback := &fakeCommandApp{}
	factories := appFactories{
		playback: fakeFactory(playback),
	}

	if err := executeTestCommand(factories, "https://youtu.be/dQw4w9WgXcQ"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if playback.called != "PlayURL" {
		t.Fatalf("called method = %q, want PlayURL", playback.called)
	}
	if playback.playURL != "https://youtu.be/dQw4w9WgXcQ" {
		t.Fatalf("PlayURL arg = %q, want URL", playback.playURL)
	}
	if playback.closeCalls != 1 {
		t.Fatalf("Close() calls = %d, want 1", playback.closeCalls)
	}
}

func TestRootHelpDoesNotCreateApp(t *testing.T) {
	called := false
	factories := appFactories{
		playback: func() (commandApp, error) {
			called = true
			return &fakeCommandApp{}, nil
		},
	}

	if err := executeTestCommand(factories); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if called {
		t.Fatal("playback factory was called for root help")
	}
}

func TestQueueCommandsUseExpectedFactories(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantMethod string
		wantURL    string
		wantKind   string
	}{
		{
			name:       "add",
			args:       []string{"queue", "add", "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp"},
			wantMethod: "QueueAdd",
			wantURL:    "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp",
			wantKind:   "queue",
		},
		{
			name:       "play",
			args:       []string{"queue", "play"},
			wantMethod: "QueuePlay",
			wantKind:   "playback",
		},
		{
			name:       "list",
			args:       []string{"queue", "list"},
			wantMethod: "QueueList",
			wantKind:   "queue",
		},
		{
			name:       "clear",
			args:       []string{"queue", "clear"},
			wantMethod: "QueueClear",
			wantKind:   "queue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			playback := &fakeCommandApp{}
			queue := &fakeCommandApp{}
			factories := appFactories{
				playback: fakeFactory(playback),
				queue:    fakeFactory(queue),
			}

			if err := executeTestCommand(factories, tt.args...); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			got := queue
			if tt.wantKind == "playback" {
				got = playback
			}
			if got.called != tt.wantMethod {
				t.Fatalf("called method = %q, want %s", got.called, tt.wantMethod)
			}
			if got.queueURL != tt.wantURL {
				t.Fatalf("queue URL = %q, want %q", got.queueURL, tt.wantURL)
			}
			if got.closeCalls != 1 {
				t.Fatalf("Close() calls = %d, want 1", got.closeCalls)
			}
		})
	}
}

func TestPlaybackControlCommandsUseControlFactory(t *testing.T) {
	tests := []struct {
		args       []string
		wantMethod string
	}{
		{args: []string{"pause"}, wantMethod: "Pause"},
		{args: []string{"resume"}, wantMethod: "Resume"},
		{args: []string{"next"}, wantMethod: "Next"},
		{args: []string{"stop"}, wantMethod: "Stop"},
		{args: []string{"status"}, wantMethod: "Status"},
	}

	for _, tt := range tests {
		t.Run(tt.args[0], func(t *testing.T) {
			control := &fakeCommandApp{}
			factories := appFactories{
				control: fakeFactory(control),
			}

			if err := executeTestCommand(factories, tt.args...); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if control.called != tt.wantMethod {
				t.Fatalf("called method = %q, want %s", control.called, tt.wantMethod)
			}
			if control.closeCalls != 1 {
				t.Fatalf("Close() calls = %d, want 1", control.closeCalls)
			}
		})
	}
}

func TestAuthUsesConfigFactory(t *testing.T) {
	configApp := &fakeCommandApp{}
	factories := appFactories{
		config: fakeFactory(configApp),
	}

	if err := executeTestCommand(factories, "auth"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if configApp.called != "AuthSpotify" {
		t.Fatalf("called method = %q, want AuthSpotify", configApp.called)
	}
	if configApp.closeCalls != 1 {
		t.Fatalf("Close() calls = %d, want 1", configApp.closeCalls)
	}
}

func TestFactoryErrorIsReturned(t *testing.T) {
	wantErr := errors.New("factory failed")
	factories := appFactories{
		playback: func() (commandApp, error) {
			return nil, wantErr
		},
	}

	err := executeTestCommand(factories, "https://youtu.be/dQw4w9WgXcQ")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestCommandErrorStillClosesApp(t *testing.T) {
	wantErr := errors.New("play failed")
	playback := &fakeCommandApp{err: wantErr}
	factories := appFactories{
		playback: fakeFactory(playback),
	}

	err := executeTestCommand(factories, "https://youtu.be/dQw4w9WgXcQ")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if playback.closeCalls != 1 {
		t.Fatalf("Close() calls = %d, want 1", playback.closeCalls)
	}
}

func executeTestCommand(factories appFactories, args ...string) error {
	cmd := newRootCmdWithFactories(factories)
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}

func fakeFactory(application commandApp) appFactory {
	return func() (commandApp, error) {
		return application, nil
	}
}

type fakeCommandApp struct {
	called     string
	playURL    string
	queueURL   string
	closeCalls int
	err        error
}

func (f *fakeCommandApp) PlayURL(_ context.Context, url string) error {
	f.called = "PlayURL"
	f.playURL = url
	return f.err
}

func (f *fakeCommandApp) QueueAdd(url string) error {
	f.called = "QueueAdd"
	f.queueURL = url
	return f.err
}

func (f *fakeCommandApp) QueuePlay(context.Context) error {
	f.called = "QueuePlay"
	return f.err
}

func (f *fakeCommandApp) QueueList() error {
	f.called = "QueueList"
	return f.err
}

func (f *fakeCommandApp) QueueClear() error {
	f.called = "QueueClear"
	return f.err
}

func (f *fakeCommandApp) Pause() error {
	f.called = "Pause"
	return f.err
}

func (f *fakeCommandApp) Resume() error {
	f.called = "Resume"
	return f.err
}

func (f *fakeCommandApp) Next() error {
	f.called = "Next"
	return f.err
}

func (f *fakeCommandApp) Stop() error {
	f.called = "Stop"
	return f.err
}

func (f *fakeCommandApp) Status() error {
	f.called = "Status"
	return f.err
}

func (f *fakeCommandApp) AuthSpotify() error {
	f.called = "AuthSpotify"
	return f.err
}

func (f *fakeCommandApp) Close() error {
	f.closeCalls++
	return nil
}
