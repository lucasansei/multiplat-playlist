// cmd/mplay/queue.go
package main

import "github.com/spf13/cobra"

func newQueueCmd(factories appFactories) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Manage playback queue",
	}

	cmd.AddCommand(
		newQueueAddCmd(factories.queue),
		newQueuePlayCmd(factories.playback),
		newQueueListCmd(factories.queue),
		newQueueClearCmd(factories.queue),
	)

	return cmd
}

func newQueueAddCmd(newQueue appFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "add [url]",
		Short: "Add a song to the queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := newQueue()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.QueueAdd(args[0])
		},
	}
}

func newQueuePlayCmd(newPlayback appFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "play",
		Short: "Play the queue in the foreground",
		Long:  "Play queued tracks sequentially and keep this command running. Playback controls use the active MPV session; next advances the queue only while this command is still running.",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := newPlayback()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.QueuePlay(cmd.Context())
		},
	}
}

func newQueueListCmd(newQueue appFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show current queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := newQueue()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.QueueList()
		},
	}
}

func newQueueClearCmd(newQueue appFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear the queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := newQueue()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.QueueClear()
		},
	}
}
