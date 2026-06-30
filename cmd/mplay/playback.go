// cmd/mplay/playback.go
package main

import (
	"github.com/lucasansei/multiplat-playlist/internal/app"
	"github.com/spf13/cobra"
)

func newPauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pause",
		Short: "Pause playback",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.NewControl()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.Pause()
		},
	}
}

func newResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Resume playback",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.NewControl()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.Resume()
		},
	}
}

func newNextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Skip to next song",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.NewControl()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.Next()
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop playback",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.NewControl()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.Stop()
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show playback status",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.NewControl()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.Status()
		},
	}
}
