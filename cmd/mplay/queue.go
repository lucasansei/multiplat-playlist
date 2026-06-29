// cmd/mplay/queue.go
package main

import (
	"github.com/lucasansei/multiplat-playlist/internal/app"
	"github.com/spf13/cobra"
)

func newQueueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Manage playback queue",
	}

	cmd.AddCommand(
		newQueueAddCmd(),
		newQueuePlayCmd(),
		newQueueListCmd(),
		newQueueClearCmd(),
	)

	return cmd
}

func newQueueAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [url]",
		Short: "Add a song to the queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.NewQueue()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.QueueAdd(args[0])
		},
	}
}

func newQueuePlayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "play",
		Short: "Start playing the queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.NewPlayback()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.QueuePlay(cmd.Context())
		},
	}
}

func newQueueListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show current queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.NewQueue()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.QueueList()
		},
	}
}

func newQueueClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear the queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.NewQueue()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.QueueClear()
		},
	}
}
