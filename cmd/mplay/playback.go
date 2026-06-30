// cmd/mplay/playback.go
package main

import "github.com/spf13/cobra"

func newPauseCmd(newControl appFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "pause",
		Short: "Pause playback",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := newControl()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.Pause()
		},
	}
}

func newResumeCmd(newControl appFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Resume playback",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := newControl()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.Resume()
		},
	}
}

func newNextCmd(newControl appFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Stop current playback so a running queue can advance",
		Long:  "Stop the active MPV playback session. In the current foreground queue model, this advances only when mplay queue play is still running.",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := newControl()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.Next()
		},
	}
}

func newStopCmd(newControl appFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop playback",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := newControl()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.Stop()
		},
	}
}

func newStatusCmd(newControl appFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show playback status",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := newControl()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.Status()
		},
	}
}
